package server

import (
	"log/slog"
	"net/http"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/web/handlers"
)

// WebServer handles the web interface
type WebServer struct {
	assistant *assistant.Assistant
	handlers  *handlers.Handlers
	logger    *slog.Logger
	mux       *http.ServeMux
}

// NewWebServer creates a new web server instance
func NewWebServer(assistant *assistant.Assistant, logger *slog.Logger) *WebServer {
	webHandlers := handlers.New(assistant, logger)

	return &WebServer{
		assistant: assistant,
		handlers:  webHandlers,
		logger:    logger,
		mux:       http.NewServeMux(),
	}
}

// SetupRoutes configures all web routes
func (ws *WebServer) SetupRoutes() {
	// Static files
	staticDir := "internal/web/static/"
	ws.mux.Handle("GET /static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir(staticDir))))

	// Main pages
	ws.mux.HandleFunc("GET /", ws.handlers.HandleDashboard)
	ws.mux.HandleFunc("GET /dashboard", ws.handlers.HandleDashboard)
	ws.mux.HandleFunc("GET /chat", ws.handlers.HandleChat)
	ws.mux.HandleFunc("GET /chat/{id}", ws.handleChatConversation)
	ws.mux.HandleFunc("GET /tools", ws.handlers.HandleTools)
	ws.mux.HandleFunc("GET /development", ws.handleDevelopment)
	ws.mux.HandleFunc("GET /database", ws.handleDatabase)
	ws.mux.HandleFunc("GET /infrastructure", ws.handleInfrastructure)
	ws.mux.HandleFunc("GET /settings", ws.handleSettings)

	// API endpoints for HTMX
	ws.mux.HandleFunc("GET /api/activities", ws.handlers.HandleAPI)
	ws.mux.HandleFunc("GET /api/stats", ws.handlers.HandleAPI)
	ws.mux.HandleFunc("POST /api/preferences/theme", ws.handlers.HandlePreferences)
	ws.mux.HandleFunc("POST /api/preferences/language", ws.handlers.HandlePreferences)

	// Chat API endpoints
	ws.mux.HandleFunc("POST /api/chat/new", ws.handleChatNew)
	ws.mux.HandleFunc("POST /api/chat/agent", ws.handleChatAgent)
	ws.mux.HandleFunc("GET /api/chat/conversations", ws.handleChatConversations)
	ws.mux.HandleFunc("GET /api/chat/{id}/messages", ws.handleChatMessages)
	ws.mux.HandleFunc("POST /api/chat/{id}/send", ws.handleChatSend)

	// Health check
	ws.mux.HandleFunc("GET /health", ws.handleHealth)

	ws.logger.Info("Web routes configured successfully")
}

// ServeHTTP implements http.Handler
func (ws *WebServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Add security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Add HTMX headers for better UX
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Push-Url", "false") // Prevent URL changes for partial updates
	}

	// Log request
	ws.logger.Debug("Web request",
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("user_agent", r.Header.Get("User-Agent")),
		slog.Bool("htmx", r.Header.Get("HX-Request") == "true"),
	)

	ws.mux.ServeHTTP(w, r)
}

// handleChatConversation handles individual chat conversation pages
func (ws *WebServer) handleChatConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("id")
	ws.logger.Debug("Loading chat conversation", slog.String("id", conversationID))

	// For now, redirect to main chat page
	// TODO: Load specific conversation
	http.Redirect(w, r, "/chat", http.StatusSeeOther)
}

// handleDevelopment renders the development assistant page
func (ws *WebServer) handleDevelopment(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement development page
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
		<div class="p-6">
			<h1 class="text-headline-large text-on-surface mb-4">Development Assistant</h1>
			<p class="text-body-large text-on-surface-variant">Development assistant page coming soon...</p>
		</div>
	`))
}

// handleDatabase renders the database manager page
func (ws *WebServer) handleDatabase(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement database page
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
		<div class="p-6">
			<h1 class="text-headline-large text-on-surface mb-4">Database Manager</h1>
			<p class="text-body-large text-on-surface-variant">Database manager page coming soon...</p>
		</div>
	`))
}

// handleInfrastructure renders the infrastructure monitor page
func (ws *WebServer) handleInfrastructure(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement infrastructure page
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
		<div class="p-6">
			<h1 class="text-headline-large text-on-surface mb-4">Infrastructure Monitor</h1>
			<p class="text-body-large text-on-surface-variant">Infrastructure monitor page coming soon...</p>
		</div>
	`))
}

// handleSettings renders the settings page
func (ws *WebServer) handleSettings(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement settings page
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
		<div class="p-6">
			<h1 class="text-headline-large text-on-surface mb-4">Settings</h1>
			<p class="text-body-large text-on-surface-variant">Settings page coming soon...</p>
		</div>
	`))
}

// Chat API handlers

// handleChatNew creates a new chat conversation
func (ws *WebServer) handleChatNew(w http.ResponseWriter, r *http.Request) {
	// TODO: Create new conversation
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok", "conversation_id": "new-chat"}`))
}

// handleChatAgent switches the active agent
func (ws *WebServer) handleChatAgent(w http.ResponseWriter, r *http.Request) {
	// TODO: Switch agent
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok"}`))
}

// handleChatConversations returns the list of conversations
func (ws *WebServer) handleChatConversations(w http.ResponseWriter, r *http.Request) {
	// TODO: Return conversation list
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<div class="p-4 text-center text-on-surface-variant">No conversations yet</div>`))
}

// handleChatMessages returns messages for a conversation
func (ws *WebServer) handleChatMessages(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("id")
	ws.logger.Debug("Loading messages", slog.String("conversation_id", conversationID))

	// TODO: Load actual messages
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<div class="p-4 text-center text-on-surface-variant">No messages yet</div>`))
}

// handleChatSend sends a new message
func (ws *WebServer) handleChatSend(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	message := r.FormValue("message")
	if message == "" {
		http.Error(w, "Message cannot be empty", http.StatusBadRequest)
		return
	}

	ws.logger.Debug("Sending message",
		slog.String("conversation_id", conversationID),
		slog.String("message", message),
	)

	// TODO: Process message with AI agent
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<div class="p-4 text-center text-on-surface-variant">Message sent</div>`))
}

// handleHealth returns server health status
func (ws *WebServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "healthy", "service": "goassistant-web"}`))
}
