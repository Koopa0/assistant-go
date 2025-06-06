// Package knowledge provides knowledge graph services for the Assistant API server.
package knowledge

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/platform/observability"
)

// KnowledgeService handles knowledge graph logic
type KnowledgeService struct {
	assistant *assistant.Assistant
	logger    *slog.Logger
	metrics   *observability.Metrics
}

// NewKnowledgeService creates a new knowledge service
func NewKnowledgeService(assistant *assistant.Assistant, logger *slog.Logger, metrics *observability.Metrics) *KnowledgeService {
	return &KnowledgeService{
		assistant: assistant,
		logger:    observability.ServerLogger(logger, "knowledge_service"),
		metrics:   metrics,
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
	// TODO: Implement actual graph retrieval from database
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

// GetNodes retrieves nodes with filters
func (s *KnowledgeService) GetNodes(ctx context.Context, nodeType string, tags []string, minImportance float64) ([]KnowledgeNode, error) {
	// TODO: Implement actual node retrieval from database
	allNodes := s.generateMockNodes(50)

	// Apply filters
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

// CreateNode creates a new knowledge node
func (s *KnowledgeService) CreateNode(ctx context.Context, node KnowledgeNode) (*KnowledgeNode, error) {
	// TODO: Implement actual node creation in database
	node.ID = fmt.Sprintf("node_%d", time.Now().Unix())
	node.CreatedAt = time.Now()
	node.UpdatedAt = time.Now()

	if node.Importance == 0 {
		node.Importance = 0.5
	}

	return &node, nil
}

// GetNode retrieves a specific node
func (s *KnowledgeService) GetNode(ctx context.Context, nodeID string) (*KnowledgeNode, error) {
	// TODO: Implement actual node retrieval from database
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

// SearchGraph searches the knowledge graph
func (s *KnowledgeService) SearchGraph(ctx context.Context, query string, searchType string, maxResults int) ([]KnowledgeNode, []KnowledgeEdge, error) {
	// TODO: Implement actual graph search
	nodes := []KnowledgeNode{}
	edges := []KnowledgeEdge{}

	// Mock search results
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

// FindPaths finds paths between two nodes
func (s *KnowledgeService) FindPaths(ctx context.Context, sourceID, targetID string, maxLength int) ([][]string, error) {
	// TODO: Implement actual pathfinding algorithm
	// Mock implementation returns some sample paths
	paths := [][]string{
		{sourceID, "node_intermediate_1", targetID},
		{sourceID, "node_intermediate_2", "node_intermediate_3", targetID},
	}

	return paths, nil
}

// GetClusters identifies clusters in the knowledge graph
func (s *KnowledgeService) GetClusters(ctx context.Context, algorithm string) ([]map[string]interface{}, error) {
	// TODO: Implement actual clustering algorithm
	clusters := []map[string]interface{}{
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
	}

	return clusters, nil
}

// GetRecommendations generates knowledge recommendations
func (s *KnowledgeService) GetRecommendations(ctx context.Context, nodeID string, recommendationType string) ([]map[string]interface{}, error) {
	// TODO: Implement actual recommendation algorithm
	recommendations := []map[string]interface{}{
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
	}

	return recommendations, nil
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
