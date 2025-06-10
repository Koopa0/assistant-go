package conversation

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/koopa0/assistant-go/internal/platform/server/handlers"
	"github.com/koopa0/assistant-go/internal/platform/server/middleware"
	"github.com/koopa0/assistant-go/internal/user"
)

// HTTPHandler handles HTTP requests for conversations
type HTTPHandler struct {
	*handlers.Handler
	service ConversationService
}

// NewHTTPHandler creates a new HTTP handler for conversations
func NewHTTPHandler(service ConversationService) *HTTPHandler {
	// TODO: Accept logger as parameter
	return &HTTPHandler{
		Handler: handlers.NewHandler(nil), // Logger should be passed in
		service: service,
	}
}

// RegisterRoutes registers all conversation routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /conversations", h.HandleListConversations)
	mux.HandleFunc("GET /conversations/{conversationId}", h.HandleGetConversation)
	mux.HandleFunc("POST /conversations", h.HandleCreateConversation)
	mux.HandleFunc("POST /conversations/{conversationId}/messages", h.HandleAddMessage)
	mux.HandleFunc("PUT /conversations/{conversationId}", h.HandleUpdateConversation)
	mux.HandleFunc("DELETE /conversations/{conversationId}", h.HandleDeleteConversation)
	mux.HandleFunc("POST /conversations/{conversationId}/archive", h.HandleArchiveConversation)
	mux.HandleFunc("GET /conversations/{conversationId}/stats", h.HandleGetStats)
}

// Request/Response types

type AddMessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Handlers

// HandleListConversations returns a list of conversations for a user
func (h *HTTPHandler) HandleListConversations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by auth middleware)
	userID := user.GetUserID(ctx)
	if userID == "" {
		h.WriteUnauthorized(w, "Authentication required")
		return
	}

	// Parse query parameters
	page := h.parseInt(r.URL.Query().Get("page"), 1)
	limit := h.parseInt(r.URL.Query().Get("limit"), 20)

	// Get conversations
	conversations, err := h.service.ListConversations(ctx, userID)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "Failed to list conversations", http.StatusInternalServerError)
		return
	}

	// Apply pagination (simple implementation)
	start := (page - 1) * limit
	end := start + limit
	if start >= len(conversations) {
		conversations = []*Conversation{}
	} else if end > len(conversations) {
		conversations = conversations[start:]
	} else {
		conversations = conversations[start:end]
	}

	middleware.WriteSuccess(w, map[string]interface{}{
		"conversations": conversations,
		"total":         len(conversations),
		"page":          page,
		"limit":         limit,
	}, "Conversations retrieved successfully")
}

// HandleGetConversation returns a single conversation
func (h *HTTPHandler) HandleGetConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conversationID := r.PathValue("conversationId")

	if conversationID == "" {
		h.WriteError(w, "INVALID_REQUEST", "Conversation ID is required", http.StatusBadRequest)
		return
	}

	// Get conversation
	conversation, err := h.service.GetConversation(ctx, conversationID)
	if err != nil {
		var notFoundErr ConversationNotFoundError
		if errors.As(err, &notFoundErr) {
			h.WriteError(w, "NOT_FOUND", "Conversation not found", http.StatusNotFound)
		} else {
			h.WriteError(w, "SERVER_ERROR", "Failed to get conversation", http.StatusInternalServerError)
		}
		return
	}

	// Get messages
	messages, err := h.service.GetMessages(ctx, conversationID)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "Failed to get messages", http.StatusInternalServerError)
		return
	}

	// Combine response
	response := map[string]interface{}{
		"conversation": conversation,
		"messages":     messages,
	}

	middleware.WriteSuccess(w, response, "Conversation retrieved successfully")
}

// HandleCreateConversation creates a new conversation
func (h *HTTPHandler) HandleCreateConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, _ := ctx.Value("userID").(string)
	if userID == "" {
		userID = "demo_user"
	}

	var req CreateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteError(w, "INVALID_REQUEST", "Invalid request format", http.StatusBadRequest)
		return
	}

	// Create conversation
	conversation, err := h.service.CreateConversation(ctx, userID, req.Title)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "Failed to create conversation", http.StatusInternalServerError)
		return
	}

	middleware.WriteSuccess(w, conversation, "Conversation created successfully")
}

// HandleAddMessage adds a message to a conversation
func (h *HTTPHandler) HandleAddMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conversationID := r.PathValue("conversationId")

	if conversationID == "" {
		h.WriteError(w, "INVALID_REQUEST", "Conversation ID is required", http.StatusBadRequest)
		return
	}

	var req AddMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteError(w, "INVALID_REQUEST", "Invalid request format", http.StatusBadRequest)
		return
	}

	// Add message
	message, err := h.service.AddMessage(ctx, conversationID, req.Role, req.Content)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "Failed to add message", http.StatusInternalServerError)
		return
	}

	middleware.WriteSuccess(w, message, "Message added successfully")
}

// HandleUpdateConversation updates conversation metadata
func (h *HTTPHandler) HandleUpdateConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conversationID := r.PathValue("conversationId")

	if conversationID == "" {
		h.WriteError(w, "INVALID_REQUEST", "Conversation ID is required", http.StatusBadRequest)
		return
	}

	// Get existing conversation
	conversation, err := h.service.GetConversation(ctx, conversationID)
	if err != nil {
		h.WriteError(w, "NOT_FOUND", "Conversation not found", http.StatusNotFound)
		return
	}

	// Parse update request
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.WriteError(w, "INVALID_REQUEST", "Invalid request format", http.StatusBadRequest)
		return
	}

	// Update fields
	if title, ok := updates["title"].(string); ok {
		conversation.Title = title
	}

	// Update metadata fields
	if tags, ok := updates["tags"].([]interface{}); ok {
		conversation.Metadata.Tags = make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				conversation.Metadata.Tags[i] = tagStr
			}
		}
	}

	if category, ok := updates["category"].(string); ok {
		conversation.Metadata.Category = category
	}

	// Save updates
	if err := h.service.UpdateConversation(ctx, conversation); err != nil {
		h.WriteError(w, "SERVER_ERROR", "Failed to update conversation", http.StatusInternalServerError)
		return
	}

	middleware.WriteSuccess(w, conversation, "Conversation updated successfully")
}

// HandleDeleteConversation deletes a conversation
func (h *HTTPHandler) HandleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conversationID := r.PathValue("conversationId")

	if conversationID == "" {
		h.WriteError(w, "INVALID_REQUEST", "Conversation ID is required", http.StatusBadRequest)
		return
	}

	err := h.service.DeleteConversation(ctx, conversationID)
	if err != nil {
		var notFoundErr ConversationNotFoundError
		if errors.As(err, &notFoundErr) {
			h.WriteError(w, "NOT_FOUND", "Conversation not found", http.StatusNotFound)
		} else {
			h.WriteError(w, "SERVER_ERROR", "Failed to delete conversation", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleArchiveConversation archives a conversation
func (h *HTTPHandler) HandleArchiveConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conversationID := r.PathValue("conversationId")

	if conversationID == "" {
		h.WriteError(w, "INVALID_REQUEST", "Conversation ID is required", http.StatusBadRequest)
		return
	}

	err := h.service.ArchiveConversation(ctx, conversationID)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "Failed to archive conversation", http.StatusInternalServerError)
		return
	}

	middleware.WriteSuccess(w, map[string]string{"message": "Conversation archived"}, "Conversation archived successfully")
}

// HandleGetStats returns conversation statistics
func (h *HTTPHandler) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	conversationID := r.PathValue("conversationId")

	if conversationID == "" {
		h.WriteError(w, "INVALID_REQUEST", "Conversation ID is required", http.StatusBadRequest)
		return
	}

	stats, err := h.service.GetConversationStats(ctx, conversationID)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "Failed to get stats", http.StatusInternalServerError)
		return
	}

	middleware.WriteSuccess(w, stats, "Statistics retrieved successfully")
}

// Helper methods

func (h *HTTPHandler) parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(s)
	if err != nil || value <= 0 {
		return defaultValue
	}
	return value
}
