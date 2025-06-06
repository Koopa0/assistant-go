package knowledge

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTTPHandler handles HTTP requests for knowledge graph
type HTTPHandler struct {
	service *KnowledgeService
}

// NewHTTPHandler creates a new HTTP handler for knowledge
func NewHTTPHandler(service *KnowledgeService) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

// RegisterRoutes registers all knowledge graph routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/knowledge/graph", h.HandleGetKnowledgeGraph)
	mux.HandleFunc("GET /api/v1/knowledge/graph/nodes", h.HandleGetNodes)
	mux.HandleFunc("POST /api/v1/knowledge/graph/nodes", h.HandleCreateNode)
	mux.HandleFunc("GET /api/v1/knowledge/graph/nodes/{id}", h.HandleGetNode)
	mux.HandleFunc("PUT /api/v1/knowledge/graph/nodes/{id}", h.HandleUpdateNode)
	mux.HandleFunc("DELETE /api/v1/knowledge/graph/nodes/{id}", h.HandleDeleteNode)
	mux.HandleFunc("GET /api/v1/knowledge/graph/edges", h.HandleGetEdges)
	mux.HandleFunc("POST /api/v1/knowledge/graph/edges", h.HandleCreateEdge)
	mux.HandleFunc("GET /api/v1/knowledge/graph/search", h.HandleSearchGraph)
	mux.HandleFunc("GET /api/v1/knowledge/graph/paths", h.HandleFindPaths)
	mux.HandleFunc("GET /api/v1/knowledge/graph/clusters", h.HandleGetClusters)
	mux.HandleFunc("GET /api/v1/knowledge/graph/recommendations", h.HandleGetRecommendations)
}

// HandleGetKnowledgeGraph retrieves the knowledge graph
func (h *HTTPHandler) HandleGetKnowledgeGraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	nodeType := r.URL.Query().Get("type")
	depth, _ := strconv.Atoi(r.URL.Query().Get("depth"))
	if depth <= 0 {
		depth = 2
	}
	includeRelated := r.URL.Query().Get("include_related") == "true"

	graph, err := h.service.GetKnowledgeGraph(ctx, nodeType, depth, includeRelated)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得知識圖譜", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, graph)
}

// HandleGetNodes retrieves nodes with filters
func (h *HTTPHandler) HandleGetNodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	nodeType := r.URL.Query().Get("type")
	tags := strings.Split(r.URL.Query().Get("tags"), ",")
	minImportance, _ := strconv.ParseFloat(r.URL.Query().Get("min_importance"), 64)

	// Clean up empty strings
	cleanTags := []string{}
	for _, tag := range tags {
		if tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}

	nodes, err := h.service.GetNodes(ctx, nodeType, cleanTags, minImportance)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得節點", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"nodes": nodes,
		"count": len(nodes),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleCreateNode creates a new knowledge node
func (h *HTTPHandler) HandleCreateNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var node KnowledgeNode
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if node.Name == "" || node.Type == "" {
		h.writeError(w, "INVALID_REQUEST", "名稱和類型為必填欄位", http.StatusBadRequest)
		return
	}

	createdNode, err := h.service.CreateNode(ctx, node)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法建立節點", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusCreated, createdNode)
}

// HandleGetNode retrieves a specific node
func (h *HTTPHandler) HandleGetNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nodeID := r.PathValue("id")

	if nodeID == "" {
		h.writeError(w, "INVALID_REQUEST", "節點 ID 為必填", http.StatusBadRequest)
		return
	}

	node, err := h.service.GetNode(ctx, nodeID)
	if err != nil {
		h.writeError(w, "NOT_FOUND", "找不到此節點", http.StatusNotFound)
		return
	}

	h.writeJSON(w, http.StatusOK, node)
}

// HandleUpdateNode updates a knowledge node
func (h *HTTPHandler) HandleUpdateNode(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")

	if nodeID == "" {
		h.writeError(w, "INVALID_REQUEST", "節點 ID 為必填", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	// TODO: Implement node update logic
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "節點更新成功",
		"node_id": nodeID,
	})
}

// HandleDeleteNode deletes a knowledge node
func (h *HTTPHandler) HandleDeleteNode(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("id")

	if nodeID == "" {
		h.writeError(w, "INVALID_REQUEST", "節點 ID 為必填", http.StatusBadRequest)
		return
	}

	// TODO: Implement node deletion logic
	w.WriteHeader(http.StatusNoContent)
}

// HandleGetEdges retrieves edges with filters
func (h *HTTPHandler) HandleGetEdges(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement edge retrieval logic
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"edges": []interface{}{},
		"count": 0,
	})
}

// HandleCreateEdge creates a new edge
func (h *HTTPHandler) HandleCreateEdge(w http.ResponseWriter, r *http.Request) {
	var edge KnowledgeEdge
	if err := json.NewDecoder(r.Body).Decode(&edge); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if edge.Source == "" || edge.Target == "" || edge.Type == "" {
		h.writeError(w, "INVALID_REQUEST", "來源、目標和類型為必填欄位", http.StatusBadRequest)
		return
	}

	// TODO: Implement edge creation logic
	h.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "關係建立成功",
		"edge":    edge,
	})
}

// HandleSearchGraph searches the knowledge graph
func (h *HTTPHandler) HandleSearchGraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	query := r.URL.Query().Get("q")
	searchType := r.URL.Query().Get("type")
	maxResults, _ := strconv.Atoi(r.URL.Query().Get("max_results"))

	if query == "" {
		h.writeError(w, "INVALID_REQUEST", "搜尋查詢為必填", http.StatusBadRequest)
		return
	}

	if maxResults <= 0 {
		maxResults = 20
	}

	nodes, edges, err := h.service.SearchGraph(ctx, query, searchType, maxResults)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "搜尋失敗", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"query":       query,
		"nodes":       nodes,
		"edges":       edges,
		"node_count":  len(nodes),
		"edge_count":  len(edges),
		"search_time": time.Now().Format(time.RFC3339),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleFindPaths finds paths between two nodes
func (h *HTTPHandler) HandleFindPaths(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	source := r.URL.Query().Get("source")
	target := r.URL.Query().Get("target")
	maxLength, _ := strconv.Atoi(r.URL.Query().Get("max_length"))

	if source == "" || target == "" {
		h.writeError(w, "INVALID_REQUEST", "來源和目標節點為必填", http.StatusBadRequest)
		return
	}

	if maxLength <= 0 {
		maxLength = 5
	}

	paths, err := h.service.FindPaths(ctx, source, target, maxLength)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法找到路徑", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"source":     source,
		"target":     target,
		"paths":      paths,
		"path_count": len(paths),
		"max_length": maxLength,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleGetClusters identifies clusters in the knowledge graph
func (h *HTTPHandler) HandleGetClusters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	algorithm := r.URL.Query().Get("algorithm")
	if algorithm == "" {
		algorithm = "louvain"
	}

	clusters, err := h.service.GetClusters(ctx, algorithm)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法識別群集", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"clusters":      clusters,
		"cluster_count": len(clusters),
		"algorithm":     algorithm,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleGetRecommendations generates knowledge recommendations
func (h *HTTPHandler) HandleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	nodeID := r.URL.Query().Get("node_id")
	recommendationType := r.URL.Query().Get("type")

	if nodeID == "" {
		h.writeError(w, "INVALID_REQUEST", "節點 ID 為必填", http.StatusBadRequest)
		return
	}

	if recommendationType == "" {
		recommendationType = "related"
	}

	recommendations, err := h.service.GetRecommendations(ctx, nodeID, recommendationType)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法生成推薦", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"node_id":         nodeID,
		"type":            recommendationType,
		"recommendations": recommendations,
		"count":           len(recommendations),
		"generated_at":    time.Now().Format(time.RFC3339),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Helper methods

func (h *HTTPHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, code, message string, status int) {
	response := map[string]interface{}{
		"success":   false,
		"error":     code,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	h.writeJSON(w, status, response)
}
