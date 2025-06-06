package conversation

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// EnhancedHTTPHandler 處理對話系統的 HTTP 請求
type EnhancedHTTPHandler struct {
	service *EnhancedConversationService
	logger  *slog.Logger
}

// NewEnhancedHTTPHandler 建立新的增強 HTTP 處理器
func NewEnhancedHTTPHandler(service *EnhancedConversationService, logger *slog.Logger) *EnhancedHTTPHandler {
	return &EnhancedHTTPHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterEnhancedRoutes 註冊增強對話路由
func (h *EnhancedHTTPHandler) RegisterEnhancedRoutes(mux *http.ServeMux) {
	// 對話系統 API
	mux.HandleFunc("GET /conversations", h.handleGetConversations)
	mux.HandleFunc("POST /conversations", h.handleCreateConversation)
	mux.HandleFunc("GET /conversations/{id}", h.handleGetConversation)
	mux.HandleFunc("POST /conversations/{id}/messages", h.handleSendMessage)

	// API v1 相容性
	mux.HandleFunc("GET /api/v1/conversations", h.handleGetConversations)
	mux.HandleFunc("POST /api/v1/conversations", h.handleCreateConversation)
	mux.HandleFunc("GET /api/v1/conversations/{id}", h.handleGetConversation)
	mux.HandleFunc("POST /api/v1/conversations/{id}/messages", h.handleSendMessage)
}

// handleGetConversations 處理取得對話列表的請求
func (h *EnhancedHTTPHandler) handleGetConversations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 從請求中取得 user_id (實際應該從 JWT token 或 session 中取得)
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		// 暫時使用預設的 Koopa 用戶
		userID = "a0000000-0000-4000-8000-000000000001"
	}

	// 解析查詢參數
	var archived *bool
	if archivedStr := r.URL.Query().Get("archived"); archivedStr != "" {
		if archivedVal, err := strconv.ParseBool(archivedStr); err == nil {
			archived = &archivedVal
		}
	}

	// 分頁參數
	limit := int32(20) // 預設每頁 20 筆
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limitVal, err := strconv.ParseInt(limitStr, 10, 32); err == nil && limitVal > 0 {
			limit = int32(limitVal)
		}
	}

	offset := int32(0)
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offsetVal, err := strconv.ParseInt(offsetStr, 10, 32); err == nil && offsetVal >= 0 {
			offset = int32(offsetVal)
		}
	}

	// 取得對話列表
	conversations, total, err := h.service.GetConversations(ctx, userID, archived, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get conversations", slog.Any("error", err))
		h.writeErrorResponse(w, "取得對話列表失敗", http.StatusInternalServerError)
		return
	}

	// 回應
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"conversations": conversations,
			"pagination": map[string]interface{}{
				"total":    total,
				"limit":    limit,
				"offset":   offset,
				"has_more": offset+limit < total,
			},
		},
		"message": "對話列表取得成功",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// handleCreateConversation 處理建立新對話的請求
func (h *EnhancedHTTPHandler) handleCreateConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 從請求中取得 user_id
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		// 暫時使用預設的 Koopa 用戶
		userID = "a0000000-0000-4000-8000-000000000001"
	}

	// 解析請求體
	var req CreateConversationRequest
	if err := h.parseJSONRequest(r, &req); err != nil {
		h.logger.Warn("Invalid create conversation request", slog.Any("error", err))
		h.writeErrorResponse(w, "無效的請求格式", http.StatusBadRequest)
		return
	}

	// 驗證必填字段
	if req.Title == "" {
		h.writeErrorResponse(w, "對話標題是必需的", http.StatusBadRequest)
		return
	}

	// 建立對話
	conversation, err := h.service.CreateConversation(ctx, userID, req)
	if err != nil {
		h.logger.Error("Failed to create conversation", slog.Any("error", err))
		h.writeErrorResponse(w, "建立對話失敗", http.StatusInternalServerError)
		return
	}

	// 回應
	response := map[string]interface{}{
		"success": true,
		"data":    conversation,
		"message": "對話建立成功",
	}

	h.writeJSONResponse(w, http.StatusCreated, response)
}

// handleGetConversation 處理取得對話詳情的請求
func (h *EnhancedHTTPHandler) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 取得對話 ID
	conversationID := r.PathValue("id")
	if conversationID == "" {
		h.writeErrorResponse(w, "對話 ID 是必需的", http.StatusBadRequest)
		return
	}

	// 取得對話詳情
	conversation, err := h.service.GetConversation(ctx, conversationID)
	if err != nil {
		h.logger.Error("Failed to get conversation", slog.Any("error", err))
		h.writeErrorResponse(w, "取得對話詳情失敗", http.StatusInternalServerError)
		return
	}

	// 回應
	response := map[string]interface{}{
		"success": true,
		"data":    conversation,
		"message": "對話詳情取得成功",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// handleSendMessage 處理發送訊息的請求
func (h *EnhancedHTTPHandler) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 取得對話 ID
	conversationID := r.PathValue("id")
	if conversationID == "" {
		h.writeErrorResponse(w, "對話 ID 是必需的", http.StatusBadRequest)
		return
	}

	// 解析請求體
	var req SendMessageRequest
	if err := h.parseJSONRequest(r, &req); err != nil {
		h.logger.Warn("Invalid send message request", slog.Any("error", err))
		h.writeErrorResponse(w, "無效的請求格式", http.StatusBadRequest)
		return
	}

	// 驗證必填字段
	if req.Role == "" {
		h.writeErrorResponse(w, "訊息角色是必需的", http.StatusBadRequest)
		return
	}
	if req.Content == "" {
		h.writeErrorResponse(w, "訊息內容是必需的", http.StatusBadRequest)
		return
	}

	// 發送訊息
	response, err := h.service.SendMessage(ctx, conversationID, req)
	if err != nil {
		h.logger.Error("Failed to send message", slog.Any("error", err))
		h.writeErrorResponse(w, "發送訊息失敗", http.StatusInternalServerError)
		return
	}

	// 回應
	responseBody := map[string]interface{}{
		"success": true,
		"data":    response,
		"message": "訊息發送成功",
	}

	h.writeJSONResponse(w, http.StatusCreated, responseBody)
}

// parseJSONRequest 解析 JSON 請求
func (h *EnhancedHTTPHandler) parseJSONRequest(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}

// writeJSONResponse 寫入 JSON 回應
func (h *EnhancedHTTPHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", slog.Any("error", err))
	}
}

// writeErrorResponse 寫入錯誤回應
func (h *EnhancedHTTPHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
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
