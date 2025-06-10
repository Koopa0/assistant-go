// Package knowledge provides knowledge graph services for the Assistant API server.
package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
	"github.com/koopa0/assistant-go/internal/user"
)

// KnowledgeService handles knowledge graph logic
type KnowledgeService struct {
	assistant *assistant.Assistant
	logger    *slog.Logger
	metrics   *observability.Metrics
	queries   *sqlc.Queries
}

// NewKnowledgeService creates a new knowledge service
func NewKnowledgeService(assistant *assistant.Assistant, logger *slog.Logger, metrics *observability.Metrics, queries *sqlc.Queries) *KnowledgeService {
	return &KnowledgeService{
		assistant: assistant,
		logger:    observability.ServerLogger(logger, "knowledge_service"),
		metrics:   metrics,
		queries:   queries,
	}
}

// KnowledgeNode represents a knowledge node
type KnowledgeNode struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // concept, technology, pattern, problem, solution
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Tags        []string               `json:"tags"`
	Importance  float64                `json:"importance"` // 0-1
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Position    *NodePosition          `json:"position,omitempty"` // For visualization
}

// NodePosition represents node position for visualization
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z,omitempty"`
}

// KnowledgeEdge represents a knowledge edge (relationship)
type KnowledgeEdge struct {
	ID         string                 `json:"id"`
	Source     string                 `json:"source"` // Source node ID
	Target     string                 `json:"target"` // Target node ID
	Type       string                 `json:"type"`   // related_to, depends_on, implements, solves, conflicts_with
	Weight     float64                `json:"weight"` // Relationship strength 0-1
	Properties map[string]interface{} `json:"properties,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// KnowledgeGraph represents a knowledge graph
type KnowledgeGraph struct {
	Nodes      []KnowledgeNode        `json:"nodes"`
	Edges      []KnowledgeEdge        `json:"edges"`
	Statistics GraphStatistics        `json:"statistics"`
	Layout     string                 `json:"layout"` // force-directed, hierarchical, circular
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// GraphStatistics represents graph statistics
type GraphStatistics struct {
	NodeCount           int            `json:"node_count"`
	EdgeCount           int            `json:"edge_count"`
	NodeTypes           map[string]int `json:"node_types"`
	EdgeTypes           map[string]int `json:"edge_types"`
	Density             float64        `json:"density"`
	AvgDegree           float64        `json:"avg_degree"`
	ConnectedComponents int            `json:"connected_components"`
	Diameter            int            `json:"diameter"`
}

// GetKnowledgeGraph retrieves the knowledge graph with filters
func (s *KnowledgeService) GetKnowledgeGraph(ctx context.Context, nodeType string, depth int, includeRelated bool) (*KnowledgeGraph, error) {
	// Get current user ID from context
	userID := user.GetUserID(ctx)
	if userID == "" {
		return nil, user.ErrNoUserInContext
	}

	userUUID := pgtype.UUID{}
	if err := userUUID.Scan(userID); err != nil {
		s.logger.Error("Invalid user ID", slog.Any("error", err))
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// If queries are not available, fall back to mock data
	if s.queries == nil {
		s.logger.Warn("Database queries not available, returning mock data")
		nodes, edges := s.generateMockGraph(nodeType, depth, includeRelated)
		stats := s.calculateGraphStatistics(nodes, edges)
		return &KnowledgeGraph{
			Nodes:      nodes,
			Edges:      edges,
			Statistics: stats,
			Layout:     "force-directed",
			Metadata: map[string]interface{}{
				"last_updated": time.Now(),
				"version":      "1.0",
			},
		}, nil
	}

	// Get nodes from database
	var nodeTypes []string
	if nodeType != "" {
		nodeTypes = []string{nodeType}
	}

	dbNodes, err := s.queries.GetKnowledgeNodes(ctx, sqlc.GetKnowledgeNodesParams{
		Column1: userUUID,
		Column2: nodeTypes,
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		s.logger.Error("Failed to get knowledge nodes", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get knowledge nodes: %w", err)
	}

	// Get edges from database
	dbEdges, err := s.queries.GetKnowledgeEdges(ctx, sqlc.GetKnowledgeEdgesParams{
		Column1: userUUID,
		Column2: nil, // Get all edge types
		Limit:   200,
		Offset:  0,
	})
	if err != nil {
		s.logger.Error("Failed to get knowledge edges", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get knowledge edges: %w", err)
	}

	// Convert database nodes to API nodes
	nodes := make([]KnowledgeNode, 0, len(dbNodes))
	nodeMap := make(map[string]bool)
	for _, dbNode := range dbNodes {
		node := s.convertDBNodeToAPINode(dbNode)
		nodes = append(nodes, node)
		nodeMap[node.ID] = true
	}

	// Convert database edges to API edges
	edges := make([]KnowledgeEdge, 0, len(dbEdges))
	for _, dbEdge := range dbEdges {
		// Only include edges where both nodes are in our node set
		sourceID := s.uuidToString(dbEdge.SourceNodeID)
		targetID := s.uuidToString(dbEdge.TargetNodeID)
		if nodeMap[sourceID] && nodeMap[targetID] {
			edge := s.convertDBEdgeToAPIEdge(dbEdge)
			edges = append(edges, edge)
		}
	}

	// Assign positions for visualization
	s.assignNodePositions(nodes)

	// Calculate statistics
	stats := s.calculateGraphStatistics(nodes, edges)

	return &KnowledgeGraph{
		Nodes:      nodes,
		Edges:      edges,
		Statistics: stats,
		Layout:     "force-directed",
		Metadata: map[string]interface{}{
			"last_updated": time.Now(),
			"version":      "1.0",
			"user_id":      userID,
		},
	}, nil
}

// GetNodes retrieves nodes with filters
func (s *KnowledgeService) GetNodes(ctx context.Context, nodeType string, tags []string, minImportance float64) ([]KnowledgeNode, error) {
	// Get current user ID from context
	userID := ctx.Value("user_id")
	if userID == nil {
		userID = "a0000000-0000-4000-8000-000000000001" // Default user ID
		s.logger.Warn("No user ID in context, using default", slog.String("default_id", userID.(string)))
	}

	userUUID := pgtype.UUID{}
	if userIDStr, ok := userID.(string); ok {
		if err := userUUID.Scan(userIDStr); err != nil {
			s.logger.Error("Invalid user ID", slog.Any("error", err))
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
	}

	// If queries are not available, fall back to mock data
	if s.queries == nil {
		s.logger.Warn("Database queries not available, returning mock data")
		allNodes := s.generateMockNodes(50)
		filtered := []KnowledgeNode{}
		for _, node := range allNodes {
			if nodeType != "" && node.Type != nodeType {
				continue
			}
			if len(tags) > 0 && !hasAnyTag(node.Tags, tags) {
				continue
			}
			if node.Importance < minImportance {
				continue
			}
			filtered = append(filtered, node)
		}
		return filtered, nil
	}

	// Build query parameters
	var nodeTypes []string
	if nodeType != "" {
		nodeTypes = []string{nodeType}
	}

	// Get nodes from database
	dbNodes, err := s.queries.GetKnowledgeNodes(ctx, sqlc.GetKnowledgeNodesParams{
		Column1: userUUID,
		Column2: nodeTypes,
		Limit:   100,
		Offset:  0,
	})
	if err != nil {
		s.logger.Error("Failed to get knowledge nodes", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get knowledge nodes: %w", err)
	}

	// Convert and filter nodes
	filtered := []KnowledgeNode{}
	for _, dbNode := range dbNodes {
		node := s.convertDBNodeToAPINode(dbNode)

		// Apply tag filter
		if len(tags) > 0 && !hasAnyTag(node.Tags, tags) {
			continue
		}

		// Apply importance filter
		if node.Importance < minImportance {
			continue
		}

		filtered = append(filtered, node)
	}

	return filtered, nil
}

// CreateNode creates a new knowledge node
func (s *KnowledgeService) CreateNode(ctx context.Context, node KnowledgeNode) (*KnowledgeNode, error) {
	// Get current user ID from context
	userID := ctx.Value("user_id")
	if userID == nil {
		userID = "a0000000-0000-4000-8000-000000000001" // Default user ID
		s.logger.Warn("No user ID in context, using default", slog.String("default_id", userID.(string)))
	}

	userUUID := pgtype.UUID{}
	if userIDStr, ok := userID.(string); ok {
		if err := userUUID.Scan(userIDStr); err != nil {
			s.logger.Error("Invalid user ID", slog.Any("error", err))
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
	}

	// If queries are not available, return mock data
	if s.queries == nil {
		s.logger.Warn("Database queries not available, returning mock data")
		node.ID = fmt.Sprintf("node_%d", time.Now().Unix())
		node.CreatedAt = time.Now()
		node.UpdatedAt = time.Now()
		if node.Importance == 0 {
			node.Importance = 0.5
		}
		return &node, nil
	}

	// Set default importance if not provided
	if node.Importance == 0 {
		node.Importance = 0.5
	}

	// Convert properties to JSON
	propertiesJSON, err := json.Marshal(node.Properties)
	if err != nil {
		s.logger.Error("Failed to marshal properties", slog.Any("error", err))
		return nil, fmt.Errorf("failed to marshal properties: %w", err)
	}

	// Store tags in properties if provided
	if len(node.Tags) > 0 {
		if node.Properties == nil {
			node.Properties = make(map[string]interface{})
		}
		node.Properties["tags"] = node.Tags
		var err error
		propertiesJSON, err = json.Marshal(node.Properties)
		if err != nil {
			return nil, fmt.Errorf("marshal node properties: %w", err)
		}
	}

	// Create node in database
	dbNode, err := s.queries.CreateKnowledgeNode(ctx, sqlc.CreateKnowledgeNodeParams{
		Column1:     userUUID,
		NodeType:    node.Type,
		NodeName:    node.Name,
		DisplayName: pgtype.Text{String: node.Name, Valid: true},
		Description: pgtype.Text{String: node.Description, Valid: node.Description != ""},
		Properties:  propertiesJSON,
		Importance:  pgtype.Float8{Float64: node.Importance, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to create knowledge node", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create knowledge node: %w", err)
	}

	// Convert back to API node
	createdNode := s.convertDBNodeToAPINode(dbNode)
	return &createdNode, nil
}

// GetNode retrieves a specific node
func (s *KnowledgeService) GetNode(ctx context.Context, nodeID string) (*KnowledgeNode, error) {
	// If queries are not available, return mock data
	if s.queries == nil {
		s.logger.Warn("Database queries not available, returning mock data")
		return &KnowledgeNode{
			ID:          nodeID,
			Type:        "technology",
			Name:        "Go Programming",
			Description: "Modern programming language designed for building scalable systems",
			Properties: map[string]interface{}{
				"version":   "1.24",
				"paradigm":  "concurrent",
				"use_cases": []string{"backend", "microservices", "cli"},
			},
			Tags:       []string{"golang", "programming", "backend"},
			Importance: 0.95,
			CreatedAt:  time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:  time.Now().Add(-2 * time.Hour),
		}, nil
	}

	// Parse the node ID as UUID
	nodeUUID := pgtype.UUID{}
	if err := nodeUUID.Scan(nodeID); err != nil {
		s.logger.Error("Invalid node ID", slog.Any("error", err))
		return nil, fmt.Errorf("invalid node ID: %w", err)
	}

	// Get node from database
	dbNode, err := s.queries.GetKnowledgeNode(ctx, nodeUUID)
	if err != nil {
		s.logger.Error("Failed to get knowledge node", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get knowledge node: %w", err)
	}

	// Update access frequency
	_, err = s.queries.UpdateKnowledgeNodeAccess(ctx, nodeUUID)
	if err != nil {
		s.logger.Warn("Failed to update node access", slog.Any("error", err))
		// Continue even if access update fails
	}

	// Convert to API node
	node := s.convertDBNodeToAPINode(dbNode)
	return &node, nil
}

// SearchGraph searches the knowledge graph
func (s *KnowledgeService) SearchGraph(ctx context.Context, query string, searchType string, maxResults int) ([]KnowledgeNode, []KnowledgeEdge, error) {
	// Get current user ID from context
	userID := ctx.Value("user_id")
	if userID == nil {
		userID = "a0000000-0000-4000-8000-000000000001" // Default user ID
		s.logger.Warn("No user ID in context, using default", slog.String("default_id", userID.(string)))
	}

	userUUID := pgtype.UUID{}
	if userIDStr, ok := userID.(string); ok {
		if err := userUUID.Scan(userIDStr); err != nil {
			s.logger.Error("Invalid user ID", slog.Any("error", err))
			return nil, nil, fmt.Errorf("invalid user ID: %w", err)
		}
	}

	// If queries are not available, fall back to mock data
	if s.queries == nil {
		s.logger.Warn("Database queries not available, returning mock data")
		nodes := []KnowledgeNode{}
		edges := []KnowledgeEdge{}
		mockNodes := s.generateMockNodes(20)
		for i, node := range mockNodes {
			if i >= maxResults {
				break
			}
			if strings.Contains(strings.ToLower(node.Name), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(node.Description), strings.ToLower(query)) {
				nodes = append(nodes, node)
			}
		}
		return nodes, edges, nil
	}

	// Set default max results
	if maxResults <= 0 {
		maxResults = 20
	}

	// Search nodes by name using full-text search
	dbNodes, err := s.queries.SearchKnowledgeNodesByName(ctx, sqlc.SearchKnowledgeNodesByNameParams{
		Column1:        userUUID,
		PlaintoTsquery: query,
		Limit:          int32(maxResults),
	})
	if err != nil {
		s.logger.Error("Failed to search knowledge nodes", slog.Any("error", err))
		return nil, nil, fmt.Errorf("failed to search knowledge nodes: %w", err)
	}

	// Convert database nodes to API nodes
	nodes := make([]KnowledgeNode, 0, len(dbNodes))
	nodeIDs := make(map[string]bool)
	for _, dbNode := range dbNodes {
		node := s.convertDBNodeToAPINode(dbNode)
		nodes = append(nodes, node)
		nodeIDs[node.ID] = true
	}

	// Get edges connected to found nodes
	edges := []KnowledgeEdge{}
	if len(nodeIDs) > 0 && searchType != "nodes_only" {
		// Get all edges for this user
		dbEdges, err := s.queries.GetKnowledgeEdges(ctx, sqlc.GetKnowledgeEdgesParams{
			Column1: userUUID,
			Column2: nil,
			Limit:   100,
			Offset:  0,
		})
		if err != nil {
			s.logger.Warn("Failed to get edges for search results", slog.Any("error", err))
			// Continue without edges
		} else {
			// Filter edges that connect to our found nodes
			for _, dbEdge := range dbEdges {
				sourceID := s.uuidToString(dbEdge.SourceNodeID)
				targetID := s.uuidToString(dbEdge.TargetNodeID)
				if nodeIDs[sourceID] || nodeIDs[targetID] {
					edge := s.convertDBEdgeToAPIEdge(dbEdge)
					edges = append(edges, edge)
				}
			}
		}
	}

	return nodes, edges, nil
}

// FindPaths finds paths between two nodes
func (s *KnowledgeService) FindPaths(ctx context.Context, sourceID, targetID string, maxLength int) ([][]string, error) {
	// Get current user ID from context
	userID := ctx.Value("user_id")
	if userID == nil {
		userID = "a0000000-0000-4000-8000-000000000001" // Default user ID
		s.logger.Warn("No user ID in context, using default", slog.String("default_id", userID.(string)))
	}

	userUUID := pgtype.UUID{}
	if userIDStr, ok := userID.(string); ok {
		if err := userUUID.Scan(userIDStr); err != nil {
			s.logger.Error("Invalid user ID", slog.Any("error", err))
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
	}

	// If queries are not available, return mock data
	if s.queries == nil {
		s.logger.Warn("Database queries not available, returning mock data")
		return [][]string{
			{sourceID, "node_intermediate_1", targetID},
			{sourceID, "node_intermediate_2", "node_intermediate_3", targetID},
		}, nil
	}

	// Parse source and target UUIDs
	sourceUUID := pgtype.UUID{}
	targetUUID := pgtype.UUID{}
	if err := sourceUUID.Scan(sourceID); err != nil {
		return nil, fmt.Errorf("invalid source ID: %w", err)
	}
	if err := targetUUID.Scan(targetID); err != nil {
		return nil, fmt.Errorf("invalid target ID: %w", err)
	}

	// Set default max length
	if maxLength <= 0 {
		maxLength = 5
	}

	// Simple BFS pathfinding implementation
	// In a production system, this would be done with a graph database or specialized algorithm
	paths := [][]string{}
	visited := make(map[string]bool)
	queue := [][]string{{sourceID}}

	for len(queue) > 0 && len(paths) < 3 {
		currentPath := queue[0]
		queue = queue[1:]

		currentNode := currentPath[len(currentPath)-1]
		if currentNode == targetID {
			paths = append(paths, currentPath)
			continue
		}

		if len(currentPath) >= maxLength {
			continue
		}

		// Get connections for current node
		currentUUID := pgtype.UUID{}
		if err := currentUUID.Scan(currentNode); err != nil {
			continue
		}

		connections, err := s.queries.GetNodeConnections(ctx, sqlc.GetNodeConnectionsParams{
			Column1: userUUID,
			Column2: currentUUID,
		})
		if err != nil {
			s.logger.Warn("Failed to get node connections", slog.Any("error", err))
			continue
		}

		for _, conn := range connections {
			nextNodeID := ""
			if connID, ok := conn.ConnectedNodeID.(pgtype.UUID); ok {
				nextNodeID = s.uuidToString(connID)
			}

			if nextNodeID != "" && !visited[nextNodeID] {
				newPath := append([]string{}, currentPath...)
				newPath = append(newPath, nextNodeID)
				queue = append(queue, newPath)
				visited[nextNodeID] = true
			}
		}
	}

	// If no paths found, check if nodes exist and are connected indirectly
	if len(paths) == 0 {
		// Get connected nodes for source
		sourceConnections, _ := s.queries.GetConnectedNodes(ctx, sqlc.GetConnectedNodesParams{
			Column1: userUUID,
			Column2: sourceUUID,
		})

		// Get connected nodes for target
		targetConnections, _ := s.queries.GetConnectedNodes(ctx, sqlc.GetConnectedNodesParams{
			Column1: userUUID,
			Column2: targetUUID,
		})

		// Find common connections
		for _, sConn := range sourceConnections {
			for _, tConn := range targetConnections {
				if s.uuidToString(sConn.ID) == s.uuidToString(tConn.ID) {
					paths = append(paths, []string{sourceID, s.uuidToString(sConn.ID), targetID})
					if len(paths) >= 3 {
						return paths, nil
					}
				}
			}
		}
	}

	return paths, nil
}

// GetClusters identifies clusters in the knowledge graph
func (s *KnowledgeService) GetClusters(ctx context.Context, algorithm string) ([]map[string]interface{}, error) {
	// Get current user ID from context
	userID := ctx.Value("user_id")
	if userID == nil {
		userID = "a0000000-0000-4000-8000-000000000001" // Default user ID
		s.logger.Warn("No user ID in context, using default", slog.String("default_id", userID.(string)))
	}

	userUUID := pgtype.UUID{}
	if userIDStr, ok := userID.(string); ok {
		if err := userUUID.Scan(userIDStr); err != nil {
			s.logger.Error("Invalid user ID", slog.Any("error", err))
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
	}

	// If queries are not available, return mock data
	if s.queries == nil {
		s.logger.Warn("Database queries not available, returning mock data")
		return []map[string]interface{}{
			{
				"id":         "cluster_1",
				"name":       "Backend Development",
				"size":       15,
				"density":    0.75,
				"core_nodes": []string{"golang", "databases", "api"},
				"color":      "#4CAF50",
			},
			{
				"id":         "cluster_2",
				"name":       "Frontend Technologies",
				"size":       12,
				"density":    0.68,
				"core_nodes": []string{"react", "css", "javascript"},
				"color":      "#2196F3",
			},
			{
				"id":         "cluster_3",
				"name":       "DevOps & Infrastructure",
				"size":       10,
				"density":    0.82,
				"core_nodes": []string{"docker", "kubernetes", "ci-cd"},
				"color":      "#FF9800",
			},
		}, nil
	}

	// Get highly connected nodes (minimum 3 connections)
	highlyConnected, err := s.queries.GetHighlyConnectedNodes(ctx, sqlc.GetHighlyConnectedNodesParams{
		Column1: userUUID,
		ID:      pgtype.UUID{Bytes: [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3}, Valid: true}, // 3 connections minimum
	})
	if err != nil {
		s.logger.Error("Failed to get highly connected nodes", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get highly connected nodes: %w", err)
	}

	// Group nodes by type to form basic clusters
	typeClusters := make(map[string][]string)
	for _, node := range highlyConnected {
		typeClusters[node.NodeType] = append(typeClusters[node.NodeType], node.NodeName)
	}

	// Define cluster names and colors by type
	clusterMeta := map[string]struct {
		name  string
		color string
	}{
		"technology": {"Technologies & Tools", "#4CAF50"},
		"concept":    {"Concepts & Patterns", "#2196F3"},
		"pattern":    {"Design Patterns", "#9C27B0"},
		"problem":    {"Problems & Challenges", "#F44336"},
		"solution":   {"Solutions & Approaches", "#FF9800"},
	}

	// Build cluster results
	clusters := []map[string]interface{}{}
	clusterID := 1

	for nodeType, nodeNames := range typeClusters {
		if len(nodeNames) < 2 {
			continue // Skip clusters with less than 2 nodes
		}

		meta, exists := clusterMeta[nodeType]
		if !exists {
			meta = struct {
				name  string
				color string
			}{nodeType + " Cluster", "#607D8B"}
		}

		// Get the most important nodes as core nodes
		coreNodes := nodeNames
		if len(coreNodes) > 5 {
			coreNodes = coreNodes[:5]
		}

		// Calculate density (simplified: based on connection count)
		totalConnections := 0
		for _, node := range highlyConnected {
			if node.NodeType == nodeType {
				totalConnections += int(node.ConnectionCount)
			}
		}
		maxPossibleConnections := len(nodeNames) * (len(nodeNames) - 1) / 2
		density := 0.0
		if maxPossibleConnections > 0 {
			density = float64(totalConnections) / float64(maxPossibleConnections*2) // Divide by 2 since connections are counted twice
			if density > 1.0 {
				density = 1.0
			}
		}

		cluster := map[string]interface{}{
			"id":         fmt.Sprintf("cluster_%d", clusterID),
			"name":       meta.name,
			"type":       nodeType,
			"size":       len(nodeNames),
			"density":    fmt.Sprintf("%.2f", density),
			"core_nodes": coreNodes,
			"color":      meta.color,
		}
		clusters = append(clusters, cluster)
		clusterID++
	}

	// If no clusters found, provide default response
	if len(clusters) == 0 {
		clusters = append(clusters, map[string]interface{}{
			"id":      "cluster_default",
			"name":    "Knowledge Graph",
			"size":    0,
			"density": 0.0,
			"message": "No significant clusters found. Add more connections between nodes to form clusters.",
		})
	}

	return clusters, nil
}

// GetRecommendations generates knowledge recommendations
func (s *KnowledgeService) GetRecommendations(ctx context.Context, nodeID string, recommendationType string) ([]map[string]interface{}, error) {
	// Get current user ID from context
	userID := ctx.Value("user_id")
	if userID == nil {
		userID = "a0000000-0000-4000-8000-000000000001" // Default user ID
		s.logger.Warn("No user ID in context, using default", slog.String("default_id", userID.(string)))
	}

	userUUID := pgtype.UUID{}
	if userIDStr, ok := userID.(string); ok {
		if err := userUUID.Scan(userIDStr); err != nil {
			s.logger.Error("Invalid user ID", slog.Any("error", err))
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
	}

	// Parse the node ID as UUID
	nodeUUID := pgtype.UUID{}
	if err := nodeUUID.Scan(nodeID); err != nil {
		s.logger.Error("Invalid node ID", slog.Any("error", err))
		return nil, fmt.Errorf("invalid node ID: %w", err)
	}

	// If queries are not available, return mock data
	if s.queries == nil {
		s.logger.Warn("Database queries not available, returning mock data")
		return []map[string]interface{}{
			{
				"node_id":     "rec_node_1",
				"name":        "Advanced Go Patterns",
				"type":        "pattern",
				"reason":      "Based on your interest in Go programming",
				"score":       0.92,
				"connections": 5,
			},
			{
				"node_id":     "rec_node_2",
				"name":        "Microservices Architecture",
				"type":        "concept",
				"reason":      "Related to your backend development focus",
				"score":       0.87,
				"connections": 8,
			},
			{
				"node_id":     "rec_node_3",
				"name":        "gRPC and Protocol Buffers",
				"type":        "technology",
				"reason":      "Complements your Go and API development",
				"score":       0.85,
				"connections": 6,
			},
		}, nil
	}

	// Get the current node details
	currentNode, err := s.queries.GetKnowledgeNode(ctx, nodeUUID)
	if err != nil {
		s.logger.Error("Failed to get current node", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get current node: %w", err)
	}

	// Get connected nodes
	connectedNodes, err := s.queries.GetConnectedNodes(ctx, sqlc.GetConnectedNodesParams{
		Column1: userUUID,
		Column2: nodeUUID,
	})
	if err != nil {
		s.logger.Error("Failed to get connected nodes", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get connected nodes: %w", err)
	}

	// Create a map of already connected node IDs
	connectedMap := make(map[string]bool)
	for _, conn := range connectedNodes {
		connectedMap[s.uuidToString(conn.ID)] = true
	}

	// Get recommendations based on type
	recommendations := []map[string]interface{}{}

	switch recommendationType {
	case "similar":
		// Get nodes of the same type that are not already connected
		similarNodes, err := s.queries.GetKnowledgeNodes(ctx, sqlc.GetKnowledgeNodesParams{
			Column1: userUUID,
			Column2: []string{currentNode.NodeType},
			Limit:   20,
			Offset:  0,
		})
		if err != nil {
			s.logger.Warn("Failed to get similar nodes", slog.Any("error", err))
		} else {
			for _, node := range similarNodes {
				if s.uuidToString(node.ID) != nodeID && !connectedMap[s.uuidToString(node.ID)] {
					rec := map[string]interface{}{
						"node_id":     s.uuidToString(node.ID),
						"name":        node.NodeName,
						"type":        node.NodeType,
						"reason":      fmt.Sprintf("Similar %s to %s", node.NodeType, currentNode.NodeName),
						"score":       node.Importance.Float64,
						"connections": 0, // Would need another query to get actual count
					}
					recommendations = append(recommendations, rec)
					if len(recommendations) >= 5 {
						break
					}
				}
			}
		}

	case "complementary":
		// Get nodes of different types that might complement this one
		complementaryTypes := s.getComplementaryTypes(currentNode.NodeType)
		if len(complementaryTypes) > 0 {
			compNodes, err := s.queries.GetKnowledgeNodes(ctx, sqlc.GetKnowledgeNodesParams{
				Column1: userUUID,
				Column2: complementaryTypes,
				Limit:   20,
				Offset:  0,
			})
			if err != nil {
				s.logger.Warn("Failed to get complementary nodes", slog.Any("error", err))
			} else {
				for _, node := range compNodes {
					if !connectedMap[s.uuidToString(node.ID)] {
						rec := map[string]interface{}{
							"node_id":     s.uuidToString(node.ID),
							"name":        node.NodeName,
							"type":        node.NodeType,
							"reason":      fmt.Sprintf("Complements %s with %s", currentNode.NodeName, node.NodeType),
							"score":       node.Importance.Float64 * 0.9, // Slightly lower score for complementary
							"connections": 0,
						}
						recommendations = append(recommendations, rec)
						if len(recommendations) >= 5 {
							break
						}
					}
				}
			}
		}

	default:
		// Get high-importance nodes that are not connected
		highImportanceNodes, err := s.queries.GetNodesByImportance(ctx, sqlc.GetNodesByImportanceParams{
			Column1:    userUUID,
			Importance: pgtype.Float8{Float64: 0.7, Valid: true},
			Limit:      20,
		})
		if err != nil {
			s.logger.Warn("Failed to get high importance nodes", slog.Any("error", err))
		} else {
			for _, node := range highImportanceNodes {
				if s.uuidToString(node.ID) != nodeID && !connectedMap[s.uuidToString(node.ID)] {
					rec := map[string]interface{}{
						"node_id":     s.uuidToString(node.ID),
						"name":        node.NodeName,
						"type":        node.NodeType,
						"reason":      "High importance node in your knowledge graph",
						"score":       node.Importance.Float64,
						"connections": 0,
					}
					recommendations = append(recommendations, rec)
					if len(recommendations) >= 5 {
						break
					}
				}
			}
		}
	}

	// If no recommendations found, provide helpful message
	if len(recommendations) == 0 {
		recommendations = append(recommendations, map[string]interface{}{
			"message": "No recommendations available. Try creating more nodes or connections to get personalized recommendations.",
		})
	}

	return recommendations, nil
}

// getComplementaryTypes returns node types that complement the given type
func (s *KnowledgeService) getComplementaryTypes(nodeType string) []string {
	complementMap := map[string][]string{
		"technology": {"pattern", "concept", "solution"},
		"concept":    {"technology", "pattern", "solution"},
		"pattern":    {"technology", "problem", "solution"},
		"problem":    {"solution", "pattern"},
		"solution":   {"problem", "technology", "pattern"},
	}

	if complements, exists := complementMap[nodeType]; exists {
		return complements
	}

	// Default: return all other types
	allTypes := []string{"technology", "concept", "pattern", "problem", "solution"}
	result := []string{}
	for _, t := range allTypes {
		if t != nodeType {
			result = append(result, t)
		}
	}
	return result
}

// uuidToString converts a pgtype.UUID to string
func (s *KnowledgeService) uuidToString(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid.Bytes[0:4],
		uuid.Bytes[4:6],
		uuid.Bytes[6:8],
		uuid.Bytes[8:10],
		uuid.Bytes[10:16])
}

// Helper methods

func (s *KnowledgeService) generateMockGraph(nodeType string, depth int, includeRelated bool) ([]KnowledgeNode, []KnowledgeEdge) {
	nodes := s.generateMockNodes(30)
	edges := []KnowledgeEdge{}

	// Create edges between nodes
	for i := 0; i < len(nodes)-1; i++ {
		for j := i + 1; j < len(nodes); j++ {
			if rand.Float64() < 0.3 { // 30% chance of connection
				edge := KnowledgeEdge{
					ID:        fmt.Sprintf("edge_%d_%d", i, j),
					Source:    nodes[i].ID,
					Target:    nodes[j].ID,
					Type:      []string{"related_to", "depends_on", "implements"}[rand.Intn(3)],
					Weight:    0.5 + rand.Float64()*0.5,
					CreatedAt: time.Now().Add(-time.Duration(rand.Intn(30)) * 24 * time.Hour),
				}
				edges = append(edges, edge)
			}
		}
	}

	// Filter by node type if specified
	if nodeType != "" {
		filteredNodes := []KnowledgeNode{}
		for _, node := range nodes {
			if node.Type == nodeType {
				filteredNodes = append(filteredNodes, node)
			}
		}
		nodes = filteredNodes
	}

	// Assign positions for visualization
	for i := range nodes {
		angle := float64(i) * 2 * math.Pi / float64(len(nodes))
		radius := 200 + rand.Float64()*100
		nodes[i].Position = &NodePosition{
			X: radius * math.Cos(angle),
			Y: radius * math.Sin(angle),
		}
	}

	return nodes, edges
}

func (s *KnowledgeService) generateMockNodes(count int) []KnowledgeNode {
	nodeTypes := []string{"concept", "technology", "pattern", "problem", "solution"}
	techNames := []string{
		"Go", "Python", "JavaScript", "React", "Vue", "Angular",
		"Docker", "Kubernetes", "PostgreSQL", "MongoDB", "Redis",
		"GraphQL", "REST API", "gRPC", "WebSocket", "OAuth",
		"Microservices", "Event Sourcing", "CQRS", "DDD", "TDD",
		"CI/CD", "GitOps", "Infrastructure as Code", "Observability",
	}

	nodes := []KnowledgeNode{}
	for i := 0; i < count && i < len(techNames); i++ {
		node := KnowledgeNode{
			ID:          fmt.Sprintf("node_%d", i+1),
			Type:        nodeTypes[rand.Intn(len(nodeTypes))],
			Name:        techNames[i],
			Description: fmt.Sprintf("Knowledge about %s and its applications", techNames[i]),
			Properties: map[string]interface{}{
				"complexity": rand.Intn(5) + 1,
				"popularity": rand.Float64(),
			},
			Tags:       generateTags(techNames[i]),
			Importance: 0.3 + rand.Float64()*0.7,
			CreatedAt:  time.Now().Add(-time.Duration(rand.Intn(365)) * 24 * time.Hour),
			UpdatedAt:  time.Now().Add(-time.Duration(rand.Intn(30)) * 24 * time.Hour),
		}
		nodes = append(nodes, node)
	}

	return nodes
}

func (s *KnowledgeService) calculateGraphStatistics(nodes []KnowledgeNode, edges []KnowledgeEdge) GraphStatistics {
	nodeTypes := make(map[string]int)
	edgeTypes := make(map[string]int)

	for _, node := range nodes {
		nodeTypes[node.Type]++
	}

	for _, edge := range edges {
		edgeTypes[edge.Type]++
	}

	density := 0.0
	if len(nodes) > 1 {
		maxEdges := len(nodes) * (len(nodes) - 1) / 2
		density = float64(len(edges)) / float64(maxEdges)
	}

	avgDegree := 0.0
	if len(nodes) > 0 {
		avgDegree = float64(len(edges)*2) / float64(len(nodes))
	}

	return GraphStatistics{
		NodeCount:           len(nodes),
		EdgeCount:           len(edges),
		NodeTypes:           nodeTypes,
		EdgeTypes:           edgeTypes,
		Density:             math.Round(density*100) / 100,
		AvgDegree:           math.Round(avgDegree*10) / 10,
		ConnectedComponents: 3, // Mock value
		Diameter:            5, // Mock value
	}
}

func generateTags(name string) []string {
	name = strings.ToLower(name)
	tags := []string{name}

	categoryTags := map[string][]string{
		"go":         {"golang", "backend", "programming"},
		"python":     {"programming", "scripting", "ml"},
		"javascript": {"frontend", "programming", "web"},
		"react":      {"frontend", "framework", "ui"},
		"docker":     {"containers", "devops", "infrastructure"},
		"kubernetes": {"orchestration", "devops", "containers"},
		"postgresql": {"database", "sql", "rdbms"},
		"mongodb":    {"database", "nosql", "document"},
	}

	if additionalTags, ok := categoryTags[name]; ok {
		tags = append(tags, additionalTags...)
	}

	return tags
}

func hasAnyTag(nodeTags, searchTags []string) bool {
	for _, searchTag := range searchTags {
		for _, nodeTag := range nodeTags {
			if strings.EqualFold(nodeTag, searchTag) {
				return true
			}
		}
	}
	return false
}

// convertDBNodeToAPINode converts a database node to API node
func (s *KnowledgeService) convertDBNodeToAPINode(dbNode *sqlc.KnowledgeNode) KnowledgeNode {
	// Parse properties
	properties := make(map[string]interface{})
	if dbNode.Properties != nil {
		_ = json.Unmarshal(dbNode.Properties, &properties)
	}

	// Extract tags from properties or generate them
	tags := []string{}
	if tagsList, ok := properties["tags"].([]interface{}); ok {
		for _, tag := range tagsList {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	} else {
		// Generate tags based on node name and type
		tags = generateTags(dbNode.NodeName)
	}

	// Get importance value
	importance := 0.5
	if dbNode.Importance.Valid {
		importance = dbNode.Importance.Float64
	}

	return KnowledgeNode{
		ID:          s.uuidToString(dbNode.ID),
		Type:        dbNode.NodeType,
		Name:        dbNode.NodeName,
		Description: dbNode.Description.String,
		Properties:  properties,
		Tags:        tags,
		Importance:  importance,
		CreatedAt:   dbNode.CreatedAt,
		UpdatedAt:   dbNode.UpdatedAt,
	}
}

// convertDBEdgeToAPIEdge converts a database edge to API edge
func (s *KnowledgeService) convertDBEdgeToAPIEdge(dbEdge *sqlc.GetKnowledgeEdgesRow) KnowledgeEdge {
	// Parse properties
	properties := make(map[string]interface{})
	if dbEdge.Properties != nil {
		_ = json.Unmarshal(dbEdge.Properties, &properties)
	}

	// Get strength value
	strength := 0.5
	if dbEdge.Strength.Valid {
		strength = dbEdge.Strength.Float64
	}

	return KnowledgeEdge{
		ID:         s.uuidToString(dbEdge.ID),
		Source:     s.uuidToString(dbEdge.SourceNodeID),
		Target:     s.uuidToString(dbEdge.TargetNodeID),
		Type:       dbEdge.EdgeType,
		Weight:     strength,
		Properties: properties,
		CreatedAt:  dbEdge.CreatedAt,
	}
}

// assignNodePositions assigns positions to nodes for visualization
func (s *KnowledgeService) assignNodePositions(nodes []KnowledgeNode) {
	if len(nodes) == 0 {
		return
	}

	// Use circular layout for now
	for i := range nodes {
		angle := float64(i) * 2 * math.Pi / float64(len(nodes))
		radius := 200 + rand.Float64()*100
		nodes[i].Position = &NodePosition{
			X: radius * math.Cos(angle),
			Y: radius * math.Sin(angle),
		}
	}
}
