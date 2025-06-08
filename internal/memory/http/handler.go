package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/koopa0/assistant-go/internal/memory"
	"github.com/koopa0/assistant-go/internal/platform/server/handlers"
	"github.com/koopa0/assistant-go/internal/platform/server/middleware"
)

// Handler handles HTTP requests for memory system
type Handler struct {
	*handlers.Handler
	service memory.HTTPMemoryServiceInterface
}

// NewHandler creates a new HTTP handler for memory endpoints
func NewHandler(service memory.HTTPMemoryServiceInterface, logger *slog.Logger) *Handler {
	return &Handler{
		Handler: handlers.NewHandler(logger),
		service: service,
	}
}

// RegisterRoutes registers memory routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Memory node API
	mux.HandleFunc("GET /memory/nodes", h.handleGetMemoryNodes)
	mux.HandleFunc("PUT /memory/nodes/{id}", h.handleUpdateMemoryNode)

	// Memory graph API
	mux.HandleFunc("GET /memory/graph", h.handleGetMemoryGraph)

	// API v1 compatibility
	mux.HandleFunc("GET /api/v1/memory/nodes", h.handleGetMemoryNodes)
	mux.HandleFunc("PUT /api/v1/memory/nodes/{id}", h.handleUpdateMemoryNode)
	mux.HandleFunc("GET /api/v1/memory/graph", h.handleGetMemoryGraph)
}

// handleGetMemoryNodes handles get memory nodes request
func (h *Handler) handleGetMemoryNodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user_id from request (should be from JWT token or session in production)
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		// Temporarily use default Koopa user
		userID = "a0000000-0000-4000-8000-000000000001"
	}

	// Parse query parameters
	filters := h.parseMemoryFilters(r)

	// Get memory nodes
	nodes, err := h.getMemoryNodes(ctx, userID, filters)
	if err != nil {
		h.LogError(r, "memory.get_nodes", err)
		h.WriteInternalError(w, err)
		return
	}

	// Response
	middleware.WriteSuccess(w, map[string]interface{}{
		"nodes": nodes,
		"total": len(nodes),
	}, "Memory nodes retrieved successfully")
}

// handleGetMemoryGraph handles get memory graph request
func (h *Handler) handleGetMemoryGraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user_id from request
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		// Temporarily use default Koopa user
		userID = "a0000000-0000-4000-8000-000000000001"
	}

	// Get memory graph
	graph, err := h.getMemoryGraph(ctx, userID)
	if err != nil {
		h.LogError(r, "memory.get_graph", err)
		h.WriteInternalError(w, err)
		return
	}

	// Response
	middleware.WriteSuccess(w, graph, "Memory graph retrieved successfully")
}

// handleUpdateMemoryNode handles update memory node request
func (h *Handler) handleUpdateMemoryNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get node ID from path
	nodeID := r.PathValue("id")
	if nodeID == "" {
		h.WriteBadRequest(w, "Node ID is required")
		return
	}

	// Get user_id from request
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		// Temporarily use default Koopa user
		userID = "a0000000-0000-4000-8000-000000000001"
	}

	// Parse request body
	var updates memory.MemoryNodeUpdate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.LogError(r, "memory.update_node.parse", err)
		h.WriteBadRequest(w, "Invalid request body")
		return
	}

	// Update memory node
	node, err := h.updateMemoryNode(ctx, userID, nodeID, updates)
	if err != nil {
		h.LogError(r, "memory.update_node", err)
		h.WriteInternalError(w, err)
		return
	}

	// Response
	middleware.WriteSuccess(w, node, "Memory node updated successfully")
}

// parseMemoryFilters parses memory filter parameters
func (h *Handler) parseMemoryFilters(r *http.Request) memory.MemoryFilters {
	filters := memory.MemoryFilters{}

	// Type filter
	if memoryType := r.URL.Query().Get("type"); memoryType != "" {
		filters.Type = &memoryType
	}

	// Minimum importance filter
	if minImpStr := r.URL.Query().Get("min_importance"); minImpStr != "" {
		if minImp, err := strconv.ParseFloat(minImpStr, 64); err == nil {
			filters.MinImportance = &minImp
		}
	}

	// Maximum age filter
	if maxAgeStr := r.URL.Query().Get("max_age"); maxAgeStr != "" {
		if maxAge, err := time.ParseDuration(maxAgeStr); err == nil {
			filters.MaxAge = &maxAge
		}
	}

	// Search filter
	if search := r.URL.Query().Get("search"); search != "" {
		filters.Search = &search
	}

	return filters
}

// getMemoryNodes retrieves memory nodes based on filters
func (h *Handler) getMemoryNodes(ctx context.Context, userID string, filters memory.MemoryFilters) ([]interface{}, error) {
	// Get memory nodes from service
	nodes, err := h.service.GetMemoryNodes(ctx, userID, filters)
	if err != nil {
		return nil, err
	}

	// Convert to response format
	result := make([]interface{}, len(nodes))
	for i, node := range nodes {
		result[i] = map[string]interface{}{
			"id":           node.ID,
			"user_id":      node.UserID,
			"type":         node.Type,
			"content":      node.Content,
			"importance":   node.Importance,
			"access_count": node.AccessCount,
			"last_access":  node.LastAccess,
			"created_at":   node.CreatedAt,
			"expires_at":   node.ExpiresAt,
			"metadata":     node.Metadata,
		}
	}

	return result, nil
}

// getMemoryGraph retrieves the memory graph for a user
func (h *Handler) getMemoryGraph(ctx context.Context, userID string) (interface{}, error) {
	// Get memory graph from service
	graph, err := h.service.GetMemoryGraph(ctx, userID, memory.MemoryFilters{})
	if err != nil {
		return nil, err
	}

	return graph, nil
}

// updateMemoryNode updates a memory node
func (h *Handler) updateMemoryNode(ctx context.Context, userID, nodeID string, updates memory.MemoryNodeUpdate) (interface{}, error) {
	// Update the node
	node, err := h.service.UpdateMemoryNode(ctx, userID, nodeID, updates)
	if err != nil {
		return nil, err
	}

	return node, nil
}
