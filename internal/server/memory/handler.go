package memory

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// HTTPHandler 處理記憶系統的 HTTP 請求
type HTTPHandler struct {
	service *MemoryService
	logger  *slog.Logger
}

// NewHTTPHandler 建立新的 HTTP 處理器
func NewHTTPHandler(service *MemoryService, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes 註冊路由
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	// 記憶節點 API
	mux.HandleFunc("GET /memory/nodes", h.handleGetMemoryNodes)
	mux.HandleFunc("PUT /memory/nodes/{id}", h.handleUpdateMemoryNode)

	// 記憶圖譜 API
	mux.HandleFunc("GET /memory/graph", h.handleGetMemoryGraph)

	// API v1 相容性
	mux.HandleFunc("GET /api/v1/memory/nodes", h.handleGetMemoryNodes)
	mux.HandleFunc("PUT /api/v1/memory/nodes/{id}", h.handleUpdateMemoryNode)
	mux.HandleFunc("GET /api/v1/memory/graph", h.handleGetMemoryGraph)
}

// handleGetMemoryNodes 處理取得記憶節點的請求
func (h *HTTPHandler) handleGetMemoryNodes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 從請求中取得 user_id (實際應該從 JWT token 或 session 中取得)
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		// 暫時使用預設的 Koopa 用戶
		userID = "a0000000-0000-4000-8000-000000000001"
	}

	// 解析查詢參數
	filters := h.parseMemoryFilters(r)

	// 取得記憶節點
	nodes, err := h.service.GetMemoryNodes(ctx, userID, filters)
	if err != nil {
		h.logger.Error("Failed to get memory nodes", slog.Any("error", err))
		h.writeErrorResponse(w, "取得記憶節點失敗", http.StatusInternalServerError)
		return
	}

	// 回應
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"nodes": nodes,
			"total": len(nodes),
		},
		"message": "記憶節點取得成功",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// handleGetMemoryGraph 處理取得記憶圖譜的請求
func (h *HTTPHandler) handleGetMemoryGraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 從請求中取得 user_id
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		// 暫時使用預設的 Koopa 用戶
		userID = "a0000000-0000-4000-8000-000000000001"
	}

	// 取得記憶圖譜
	graph, err := h.service.GetMemoryGraph(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get memory graph", slog.Any("error", err))
		h.writeErrorResponse(w, "取得記憶圖譜失敗", http.StatusInternalServerError)
		return
	}

	// 回應
	response := map[string]interface{}{
		"success": true,
		"data":    graph,
		"message": "記憶圖譜取得成功",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// handleUpdateMemoryNode 處理更新記憶節點的請求
func (h *HTTPHandler) handleUpdateMemoryNode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 取得節點 ID
	nodeID := r.PathValue("id")
	if nodeID == "" {
		h.writeErrorResponse(w, "節點 ID 是必需的", http.StatusBadRequest)
		return
	}

	// 解析請求體
	var updates MemoryNodeUpdate
	if err := h.parseJSONRequest(r, &updates); err != nil {
		h.logger.Warn("Invalid update request", slog.Any("error", err))
		h.writeErrorResponse(w, "無效的請求格式", http.StatusBadRequest)
		return
	}

	// 更新記憶節點
	node, err := h.service.UpdateMemoryNode(ctx, nodeID, updates)
	if err != nil {
		h.logger.Error("Failed to update memory node", slog.Any("error", err))
		h.writeErrorResponse(w, "更新記憶節點失敗", http.StatusInternalServerError)
		return
	}

	// 回應
	response := map[string]interface{}{
		"success": true,
		"data":    node,
		"message": "記憶節點更新成功",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// parseMemoryFilters 解析記憶過濾器參數
func (h *HTTPHandler) parseMemoryFilters(r *http.Request) MemoryFilters {
	filters := MemoryFilters{}

	// 類型過濾
	if memoryType := r.URL.Query().Get("type"); memoryType != "" {
		filters.Type = &memoryType
	}

	// 最小重要性過濾
	if minImpStr := r.URL.Query().Get("min_importance"); minImpStr != "" {
		if minImp, err := strconv.ParseFloat(minImpStr, 64); err == nil {
			filters.MinImportance = &minImp
		}
	}

	// 最大年齡過濾
	if maxAgeStr := r.URL.Query().Get("max_age"); maxAgeStr != "" {
		if maxAge, err := time.ParseDuration(maxAgeStr); err == nil {
			filters.MaxAge = &maxAge
		}
	}

	// 搜尋過濾
	if search := r.URL.Query().Get("search"); search != "" {
		filters.Search = &search
	}

	return filters
}

// parseJSONRequest 解析 JSON 請求
func (h *HTTPHandler) parseJSONRequest(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}

// writeJSONResponse 寫入 JSON 回應
func (h *HTTPHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", slog.Any("error", err))
	}
}

// writeErrorResponse 寫入錯誤回應
func (h *HTTPHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"message": message,
			"code":    statusCode,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	h.writeJSONResponse(w, statusCode, response)
}

// Health 檢查處理器健康狀態
func (h *HTTPHandler) Health() error {
	return nil
}
