package chat

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// HTTPHandler handles HTTP requests for chat
type HTTPHandler struct {
	service *ChatService
}

// NewHTTPHandler creates a new HTTP handler for chat
func NewHTTPHandler(service *ChatService) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

// RegisterRoutes registers all chat routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	// Chat completion (OpenAI compatible)
	mux.HandleFunc("POST /api/v1/chat/completions", h.HandleChatCompletion)
	mux.HandleFunc("GET /api/v1/chat/conversations", h.HandleListConversations)
	mux.HandleFunc("GET /api/v1/chat/conversations/{id}", h.HandleGetConversation)

	// Memory management
	mux.HandleFunc("GET /api/v1/memory/working", h.HandleGetWorkingMemory)

	// Knowledge management
	mux.HandleFunc("GET /api/v1/knowledge/concepts", h.HandleListConcepts)

	// Tools management
	mux.HandleFunc("GET /api/v1/tools", h.HandleListTools)

	// System health
	mux.HandleFunc("GET /api/v1/health", h.HandleHealth)

	// Search
	mux.HandleFunc("POST /api/v1/search", h.HandleSearch)

	// API info
	mux.HandleFunc("GET /api/v1", h.HandleAPIInfo)
}

// HandleChatCompletion handles chat completion requests (OpenAI compatible)
func (h *HTTPHandler) HandleChatCompletion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	// Validate request
	if len(req.Messages) == 0 {
		h.writeError(w, "INVALID_REQUEST", "訊息列表不能為空", http.StatusBadRequest)
		return
	}

	// Process chat completion
	response, err := h.service.ProcessChatCompletion(ctx, &req)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "處理聊天完成失敗", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleListConversations lists conversations
func (h *HTTPHandler) HandleListConversations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 20
	}

	// Get user ID from context (TODO: implement auth middleware)
	userID := "api_user"

	conversations, err := h.service.ListConversations(ctx, userID, limit, offset)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得對話列表", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"conversations": conversations,
		"count":         len(conversations),
		"limit":         limit,
		"offset":        offset,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleGetConversation retrieves a specific conversation
func (h *HTTPHandler) HandleGetConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conversationID := r.PathValue("id")

	if conversationID == "" {
		h.writeError(w, "INVALID_REQUEST", "對話 ID 為必填", http.StatusBadRequest)
		return
	}

	conversation, err := h.service.GetConversation(ctx, conversationID)
	if err != nil {
		h.writeError(w, "NOT_FOUND", "找不到此對話", http.StatusNotFound)
		return
	}

	h.writeJSON(w, http.StatusOK, conversation)
}

// HandleGetWorkingMemory retrieves working memory
func (h *HTTPHandler) HandleGetWorkingMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (TODO: implement auth middleware)
	userID := "api_user"

	memory, err := h.service.GetWorkingMemory(ctx, userID)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得工作記憶體", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, memory)
}

// HandleListConcepts lists knowledge concepts
func (h *HTTPHandler) HandleListConcepts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	conceptType := r.URL.Query().Get("type")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if limit <= 0 {
		limit = 50
	}

	concepts, err := h.service.ListConcepts(ctx, conceptType, limit)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得概念列表", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"concepts": concepts,
		"count":    len(concepts),
		"type":     conceptType,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleListTools lists available tools
func (h *HTTPHandler) HandleListTools(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tools, err := h.service.ListTools(ctx)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得工具列表", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"tools": tools,
		"count": len(tools),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleHealth returns system health status
func (h *HTTPHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check assistant health
	if err := h.service.assistant.Health(ctx); err != nil {
		h.writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status":    "unhealthy",
			"error":     err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleSearch handles search requests
func (h *HTTPHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Query string `json:"query"`
		Type  string `json:"type,omitempty"`
		Limit int    `json:"limit,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		h.writeError(w, "INVALID_REQUEST", "查詢內容為必填", http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	results, err := h.service.Search(ctx, req.Query, req.Type, req.Limit)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "搜尋失敗", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"query":   req.Query,
		"results": results,
		"count":   len(results),
		"type":    req.Type,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleAPIInfo returns API information
func (h *HTTPHandler) HandleAPIInfo(w http.ResponseWriter, r *http.Request) {
	apiInfo := map[string]interface{}{
		"name":        "Assistant API v1",
		"version":     "1.0.0",
		"description": "智慧開發助手 API v1",
		"endpoints": map[string]interface{}{
			"chat":          "/api/v1/chat/completions",
			"conversations": "/api/v1/chat/conversations",
			"memory":        "/api/v1/memory/working",
			"knowledge":     "/api/v1/knowledge/concepts",
			"tools":         "/api/v1/tools",
			"health":        "/api/v1/health",
			"search":        "/api/v1/search",
		},
		"features": []string{
			"聊天完成（OpenAI 相容）",
			"對話管理",
			"工作記憶體",
			"知識概念",
			"工具執行",
			"系統健康檢查",
			"語義搜尋",
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	h.writeJSON(w, http.StatusOK, apiInfo)
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
