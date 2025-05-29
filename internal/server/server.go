package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/koopa0/assistant/internal/assistant"
	"github.com/koopa0/assistant/internal/config"
	"github.com/koopa0/assistant/internal/observability"
)

// Server represents the HTTP API server
type Server struct {
	config    config.ServerConfig
	assistant *assistant.Assistant
	logger    *slog.Logger
	server    *http.Server
	mux       *http.ServeMux
}

// New creates a new HTTP API server
func New(cfg config.ServerConfig, assistant *assistant.Assistant, logger *slog.Logger) (*Server, error) {
	if assistant == nil {
		return nil, fmt.Errorf("assistant is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Create server mux
	mux := http.NewServeMux()

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         cfg.Address,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	server := &Server{
		config:    cfg,
		assistant: assistant,
		logger:    observability.ServerLogger(logger, "http"),
		server:    httpServer,
		mux:       mux,
	}

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// Start starts the HTTP API server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting HTTP API server",
		slog.String("address", s.config.Address),
		slog.Bool("tls_enabled", s.config.EnableTLS))

	// Start server
	if s.config.EnableTLS {
		if s.config.TLSCertFile == "" || s.config.TLSKeyFile == "" {
			return fmt.Errorf("TLS cert and key files are required when TLS is enabled")
		}
		return s.server.ListenAndServeTLS(s.config.TLSCertFile, s.config.TLSKeyFile)
	}

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP API server...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()

	// Shutdown server
	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Info("HTTP API server shutdown complete")
	return nil
}

// setupRoutes sets up the HTTP API routes
func (s *Server) setupRoutes() {
	// Apply middleware
	handler := s.withMiddleware(s.mux)
	s.server.Handler = handler

	// API routes
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/status", s.handleStatus)
	s.mux.HandleFunc("POST /api/query", s.handleQuery)
	s.mux.HandleFunc("GET /api/conversations", s.handleListConversations)
	s.mux.HandleFunc("GET /api/conversations/{id}", s.handleGetConversation)
	s.mux.HandleFunc("DELETE /api/conversations/{id}", s.handleDeleteConversation)
	s.mux.HandleFunc("GET /api/tools", s.handleListTools)
	s.mux.HandleFunc("GET /api/tools/{name}", s.handleGetTool)
	s.mux.HandleFunc("POST /api/tools/{name}/execute", s.handleExecuteTool)

	s.logger.Debug("HTTP API routes configured")
}

// withMiddleware applies middleware to the handler
func (s *Server) withMiddleware(handler http.Handler) http.Handler {
	// Apply middleware in reverse order (last applied is executed first)

	// Recovery middleware (outermost)
	handler = s.recoveryMiddleware(handler)

	// Logging middleware
	handler = s.loggingMiddleware(handler)

	// CORS middleware
	handler = s.corsMiddleware(handler)

	// Request ID middleware
	handler = s.requestIDMiddleware(handler)

	return handler
}

// Health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check assistant health
	if err := s.assistant.Health(ctx); err != nil {
		s.logger.Error("Health check failed", slog.Any("error", err))
		http.Error(w, "Service unhealthy", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
}

// Status endpoint
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := s.assistant.Stats(ctx)
	if err != nil {
		s.logger.Error("Failed to get stats", slog.Any("error", err))
		http.Error(w, "Failed to get status", http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "running",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"stats":     stats,
	})
}

// Query endpoint
func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var request assistant.QueryRequest
	if err := s.parseJSONRequest(r, &request); err != nil {
		s.logger.Warn("Invalid query request", slog.Any("error", err))
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	response, err := s.assistant.ProcessQueryRequest(ctx, &request)
	if err != nil {
		s.logger.Error("Query processing failed", slog.Any("error", err))

		// Handle different error types
		if assistantErr := assistant.GetAssistantError(err); assistantErr != nil {
			switch assistantErr.Code {
			case assistant.CodeInvalidInput:
				http.Error(w, assistantErr.Message, http.StatusBadRequest)
			case assistant.CodeRateLimited:
				http.Error(w, assistantErr.Message, http.StatusTooManyRequests)
			case assistant.CodeUnauthorized:
				http.Error(w, assistantErr.Message, http.StatusUnauthorized)
			case assistant.CodeTimeout:
				http.Error(w, assistantErr.Message, http.StatusRequestTimeout)
			default:
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// List conversations endpoint
func (s *Server) handleListConversations(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement conversation listing
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"conversations": []interface{}{},
		"total":         0,
	})
}

// Get conversation endpoint
func (s *Server) handleGetConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("id")
	if conversationID == "" {
		http.Error(w, "Conversation ID is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	conversation, err := s.assistant.GetConversation(ctx, conversationID)
	if err != nil {
		if assistantErr := assistant.GetAssistantError(err); assistantErr != nil && assistantErr.Code == assistant.CodeContextNotFound {
			http.Error(w, "Conversation not found", http.StatusNotFound)
		} else {
			s.logger.Error("Failed to get conversation", slog.Any("error", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	s.writeJSONResponse(w, http.StatusOK, conversation)
}

// Delete conversation endpoint
func (s *Server) handleDeleteConversation(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("id")
	if conversationID == "" {
		http.Error(w, "Conversation ID is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := s.assistant.DeleteConversation(ctx, conversationID); err != nil {
		if assistantErr := assistant.GetAssistantError(err); assistantErr != nil && assistantErr.Code == assistant.CodeContextNotFound {
			http.Error(w, "Conversation not found", http.StatusNotFound)
		} else {
			s.logger.Error("Failed to delete conversation", slog.Any("error", err))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// List tools endpoint
func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	tools := s.assistant.GetAvailableTools()
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"tools": tools,
		"total": len(tools),
	})
}

// Get tool endpoint
func (s *Server) handleGetTool(w http.ResponseWriter, r *http.Request) {
	toolName := r.PathValue("name")
	if toolName == "" {
		http.Error(w, "Tool name is required", http.StatusBadRequest)
		return
	}

	toolInfo, err := s.assistant.GetToolInfo(toolName)
	if err != nil {
		http.Error(w, "Tool not found", http.StatusNotFound)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, toolInfo)
}

// Execute tool endpoint
func (s *Server) handleExecuteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	toolName := r.PathValue("name")
	if toolName == "" {
		http.Error(w, "Tool name is required", http.StatusBadRequest)
		return
	}

	// Parse request body
	var request struct {
		Input  map[string]interface{} `json:"input"`
		Config map[string]interface{} `json:"config,omitempty"`
	}

	if err := s.parseJSONRequest(r, &request); err != nil {
		s.logger.Warn("Invalid tool execution request", slog.Any("error", err))
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Execute tool through assistant
	result, err := s.assistant.ExecuteTool(ctx, toolName, request.Input, request.Config)
	if err != nil {
		s.logger.Error("Tool execution failed",
			slog.String("tool", toolName),
			slog.Any("error", err))
		http.Error(w, fmt.Sprintf("Tool execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, result)
}
