package conversation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/koopa0/assistant-go/internal/core/conversation"
	converrors "github.com/koopa0/assistant-go/internal/core/conversation"
)

// HTTPHandler handles HTTP requests for conversations
type HTTPHandler struct {
	service *ConversationService
}

// NewHTTPHandler creates a new HTTP handler for conversations
func NewHTTPHandler(service *ConversationService) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

// MessageAttachment represents an attachment to a message
type MessageAttachment struct {
	Type     string `json:"type"`
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// RegisterRoutes registers all conversation routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /conversations", h.HandleListConversations)
	mux.HandleFunc("GET /conversations/{conversationId}", h.HandleGetConversation)
	mux.HandleFunc("POST /conversations", h.HandleCreateConversation)
	mux.HandleFunc("POST /conversations/{conversationId}/messages", h.HandleSendMessage)
	mux.HandleFunc("PUT /conversations/{conversationId}", h.HandleUpdateConversation)
	mux.HandleFunc("DELETE /conversations/{conversationId}", h.HandleDeleteConversation)
	mux.HandleFunc("POST /conversations/{conversationId}/archive", h.HandleArchiveConversation)
	mux.HandleFunc("POST /conversations/{conversationId}/unarchive", h.HandleUnarchiveConversation)
	mux.HandleFunc("GET /conversations/{conversationId}/export", h.HandleExportConversation)
}

// HandleListConversations returns a paginated list of conversations
func (h *HTTPHandler) HandleListConversations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by auth middleware)
	userID, _ := ctx.Value("userID").(string)
	if userID == "" {
		userID = "demo_user" // For demo purposes
	}

	// Parse query parameters
	params := ListConversationsParams{
		UserID:    userID,
		Search:    r.URL.Query().Get("search"),
		Category:  r.URL.Query().Get("category"),
		Status:    r.URL.Query().Get("status"),
		SortBy:    r.URL.Query().Get("sortBy"),
		SortOrder: r.URL.Query().Get("sortOrder"),
		Page:      h.parseInt(r.URL.Query().Get("page"), 1),
		Limit:     h.parseInt(r.URL.Query().Get("limit"), 20),
	}

	// Get conversations
	conversations, total, err := h.service.ListConversations(ctx, params)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得對話列表", http.StatusInternalServerError)
		return
	}

	h.writeSuccessWithPagination(w, conversations, params.Page, params.Limit, total)
}

// HandleGetConversation returns a single conversation with messages
func (h *HTTPHandler) HandleGetConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conversationID := r.PathValue("conversationId")

	if conversationID == "" {
		h.writeError(w, "INVALID_REQUEST", "對話 ID 為必填", http.StatusBadRequest)
		return
	}

	// Get conversation
	conversation, err := h.service.GetConversation(ctx, conversationID)
	if err != nil {
		if converrors.IsConversationNotFoundError(err) {
			h.writeError(w, "NOT_FOUND", "找不到此對話", http.StatusNotFound)
		} else {
			h.writeError(w, "SERVER_ERROR", "無法取得對話", http.StatusInternalServerError)
		}
		return
	}

	h.writeSuccess(w, conversation, "取得對話成功")
}

// HandleCreateConversation creates a new conversation
func (h *HTTPHandler) HandleCreateConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, _ := ctx.Value("userID").(string)
	if userID == "" {
		userID = "demo_user"
	}

	var req struct {
		Title          string                 `json:"title,omitempty"`
		InitialMessage string                 `json:"initialMessage"`
		Metadata       map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	if req.InitialMessage == "" {
		h.writeError(w, "INVALID_REQUEST", "初始訊息為必填", http.StatusBadRequest)
		return
	}

	// Create conversation
	queryResp, err := h.service.CreateConversation(ctx, userID, req.Title, req.InitialMessage, req.Metadata)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法處理初始訊息", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":      queryResp.ConversationID,
		"title":   req.Title,
		"created": time.Now().UTC(),
		"initialResponse": map[string]interface{}{
			"messageId": queryResp.MessageID,
			"content":   queryResp.Response,
			"metadata": map[string]interface{}{
				"provider":   queryResp.Provider,
				"model":      queryResp.Model,
				"tokensUsed": queryResp.TokensUsed,
			},
		},
	}

	h.writeSuccess(w, response, "對話建立成功")
}

// HandleSendMessage sends a message to a conversation
func (h *HTTPHandler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conversationID := r.PathValue("conversationId")

	if conversationID == "" {
		h.writeError(w, "INVALID_REQUEST", "對話 ID 為必填", http.StatusBadRequest)
		return
	}

	var req struct {
		Content     string              `json:"content"`
		Attachments []MessageAttachment `json:"attachments,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		h.writeError(w, "INVALID_REQUEST", "訊息內容為必填", http.StatusBadRequest)
		return
	}

	// Convert attachments to interface slice
	var attachments []interface{}
	for _, att := range req.Attachments {
		attachments = append(attachments, att)
	}

	// Send message
	queryResp, err := h.service.SendMessage(ctx, conversationID, req.Content, attachments)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法處理訊息", http.StatusInternalServerError)
		return
	}

	response := MessageResponse{
		ID:        queryResp.MessageID,
		Role:      "assistant",
		Content:   queryResp.Response,
		Timestamp: time.Now().UTC(),
		Metadata: map[string]interface{}{
			"processingTime": queryResp.ExecutionTime.Milliseconds(),
			"toolsUsed":      queryResp.ToolsUsed,
			"provider":       queryResp.Provider,
			"model":          queryResp.Model,
			"tokensUsed":     queryResp.TokensUsed,
		},
	}

	h.writeSuccess(w, response, "訊息發送成功")
}

// HandleUpdateConversation updates conversation metadata
func (h *HTTPHandler) HandleUpdateConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("conversationId")

	if conversationID == "" {
		h.writeError(w, "INVALID_REQUEST", "對話 ID 為必填", http.StatusBadRequest)
		return
	}

	var req struct {
		Title    string   `json:"title,omitempty"`
		Category string   `json:"category,omitempty"`
		Tags     []string `json:"tags,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	// In a real implementation, update the conversation
	h.service.logger.Info("Updating conversation",
		"conversationId", conversationID,
		"title", req.Title)

	h.writeSuccess(w, nil, "對話更新成功")
}

// HandleDeleteConversation deletes a conversation
func (h *HTTPHandler) HandleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conversationID := r.PathValue("conversationId")

	if conversationID == "" {
		h.writeError(w, "INVALID_REQUEST", "對話 ID 為必填", http.StatusBadRequest)
		return
	}

	err := h.service.DeleteConversation(ctx, conversationID)
	if err != nil {
		if converrors.IsConversationNotFoundError(err) {
			h.writeError(w, "NOT_FOUND", "找不到此對話", http.StatusNotFound)
		} else {
			h.writeError(w, "SERVER_ERROR", "無法刪除對話", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleArchiveConversation archives a conversation
func (h *HTTPHandler) HandleArchiveConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("conversationId")

	if conversationID == "" {
		h.writeError(w, "INVALID_REQUEST", "對話 ID 為必填", http.StatusBadRequest)
		return
	}

	// In a real implementation, archive the conversation
	h.service.logger.Info("Archiving conversation", "conversationId", conversationID)

	h.writeSuccess(w, nil, "對話已封存")
}

// HandleUnarchiveConversation unarchives a conversation
func (h *HTTPHandler) HandleUnarchiveConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("conversationId")

	if conversationID == "" {
		h.writeError(w, "INVALID_REQUEST", "對話 ID 為必填", http.StatusBadRequest)
		return
	}

	// In a real implementation, unarchive the conversation
	h.service.logger.Info("Unarchiving conversation", "conversationId", conversationID)

	h.writeSuccess(w, nil, "對話已取消封存")
}

// HandleExportConversation exports a conversation
func (h *HTTPHandler) HandleExportConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conversationID := r.PathValue("conversationId")
	format := r.URL.Query().Get("format")

	if conversationID == "" {
		h.writeError(w, "INVALID_REQUEST", "對話 ID 為必填", http.StatusBadRequest)
		return
	}

	if format == "" {
		format = "json"
	}

	// Get conversation
	conversation, err := h.service.ExportConversation(ctx, conversationID)
	if err != nil {
		h.writeError(w, "NOT_FOUND", "找不到此對話", http.StatusNotFound)
		return
	}

	// Export based on format
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"conversation_%s.json\"", conversationID))
		json.NewEncoder(w).Encode(conversation)
	case "txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"conversation_%s.txt\"", conversationID))
		h.exportAsText(w, conversation)
	default:
		h.writeError(w, "INVALID_REQUEST", "不支援的匯出格式", http.StatusBadRequest)
	}
}

// Helper methods

func (h *HTTPHandler) exportAsText(w http.ResponseWriter, conv *conversation.Conversation) {
	fmt.Fprintf(w, "對話標題: %s\n", conv.Title)
	fmt.Fprintf(w, "建立時間: %s\n", conv.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "更新時間: %s\n\n", conv.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprint(w, "=== 對話內容 ===\n\n")

	for _, msg := range conv.Messages {
		role := "使用者"
		if msg.Role == "assistant" {
			role = "助理"
		}
		fmt.Fprintf(w, "[%s] %s:\n", role, msg.CreatedAt.Format("15:04:05"))
		fmt.Fprintf(w, "%s\n\n", msg.Content)
	}
}

func (h *HTTPHandler) parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	var value int
	if _, err := fmt.Sscanf(s, "%d", &value); err != nil {
		return defaultValue
	}
	if value <= 0 {
		return defaultValue
	}
	return value
}

// Response helpers

func (h *HTTPHandler) writeSuccess(w http.ResponseWriter, data interface{}, message string) {
	response := map[string]interface{}{
		"success":   true,
		"data":      data,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) writeSuccessWithPagination(w http.ResponseWriter, data interface{}, page, limit, total int) {
	response := map[string]interface{}{
		"success": true,
		"data":    data,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + limit - 1) / limit,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, code, message string, status int) {
	response := map[string]interface{}{
		"success":   false,
		"error":     code,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}
