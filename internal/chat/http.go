package chat

import (
	"net/http"

	"github.com/koopa0/assistant-go/internal/platform/server/handlers"
)

// HTTPHandler handles HTTP requests for chat
type HTTPHandler struct {
	*handlers.Handler
	service *ChatService
}

// NewHTTPHandler creates a new HTTP handler for chat
func NewHTTPHandler(service *ChatService) *HTTPHandler {
	return &HTTPHandler{
		Handler: handlers.NewHandler(service.logger),
		service: service,
	}
}

// RegisterRoutes registers all chat routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	// Chat completion (OpenAI compatible)
	mux.HandleFunc("POST /api/v1/chat/completions", h.HandleChatCompletion)
	mux.HandleFunc("GET /api/v1/chat/conversations", h.HandleListConversations)
	mux.HandleFunc("GET /api/v1/chat/conversations/{id}", h.HandleGetConversation)

	// NOTE: The following routes should be moved to their respective handlers:
	// - /api/v1/memory/* → memory handler
	// - /api/v1/knowledge/* → knowledge handler
	// - /api/v1/tools → tools handler
	// - /api/v1/health → system handler
	// - /api/v1/search → search handler (or keep in chat if search is chat-specific)
}

// HandleChatCompletion handles chat completion requests (OpenAI compatible)
func (h *HTTPHandler) HandleChatCompletion(w http.ResponseWriter, r *http.Request) {
	h.LogRequest(r, "chat.completion")
	ctx := r.Context()

	var req ChatCompletionRequest
	if err := h.DecodeJSON(r, &req); err != nil {
		h.WriteBadRequest(w, "Invalid request format", err.Error())
		return
	}

	// Validate request
	if len(req.Messages) == 0 {
		h.WriteBadRequest(w, "Messages cannot be empty")
		return
	}

	// Process chat completion
	response, err := h.service.ProcessChatCompletion(ctx, &req)
	if err != nil {
		h.LogError(r, "chat.completion", err)
		h.WriteInternalError(w, err)
		return
	}

	h.WriteSuccess(w, response, "Chat completion processed successfully")
}

// HandleListConversations lists conversations
func (h *HTTPHandler) HandleListConversations(w http.ResponseWriter, r *http.Request) {
	h.LogRequest(r, "chat.list_conversations")
	ctx := r.Context()

	// Parse pagination
	page, limit, err := h.ParsePagination(r)
	if err != nil {
		h.WriteBadRequest(w, err.Error())
		return
	}

	// Get user ID from context
	userID, err := h.GetUserID(ctx)
	if err != nil {
		// Use default for now until auth is implemented
		userID = "api_user"
	}

	// Calculate offset from page
	offset := (page - 1) * limit

	conversations, err := h.service.ListConversations(ctx, userID, limit, offset)
	if err != nil {
		h.LogError(r, "chat.list_conversations", err)
		h.WriteInternalError(w, err)
		return
	}

	// TODO: Get total count for proper pagination
	total := len(conversations) // This is not accurate, need total count from DB

	h.WriteSuccessWithPagination(w, conversations, page, limit, total)
}

// HandleGetConversation retrieves a specific conversation
func (h *HTTPHandler) HandleGetConversation(w http.ResponseWriter, r *http.Request) {
	h.LogRequest(r, "chat.get_conversation")
	ctx := r.Context()
	conversationID := r.PathValue("id")

	if conversationID == "" {
		h.WriteBadRequest(w, "Conversation ID is required")
		return
	}

	conversation, err := h.service.GetConversation(ctx, conversationID)
	if err != nil {
		h.LogError(r, "chat.get_conversation", err)
		h.WriteNotFound(w, "conversation")
		return
	}

	h.WriteSuccess(w, conversation, "Conversation retrieved successfully")
}

// TODO: The following handlers should be moved to their respective packages:
// - HandleGetWorkingMemory -> memory package
// - HandleListConcepts -> knowledge package
// - HandleListTools -> tools package
// - HandleHealth -> system package
// - HandleSearch -> search package (or keep if chat-specific)
// - HandleAPIInfo -> system package
