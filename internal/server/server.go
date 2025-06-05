// Package server provides HTTP API server implementation with middleware support,
// request/response handling, and graceful shutdown capabilities.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/observability"
	"github.com/koopa0/assistant-go/internal/server/analytics"
	"github.com/koopa0/assistant-go/internal/server/auth"
	"github.com/koopa0/assistant-go/internal/server/chat"
	"github.com/koopa0/assistant-go/internal/server/collaboration"
	"github.com/koopa0/assistant-go/internal/server/conversation"
	"github.com/koopa0/assistant-go/internal/server/knowledge"
	langchain "github.com/koopa0/assistant-go/internal/server/langchain"
	"github.com/koopa0/assistant-go/internal/server/learning"
	"github.com/koopa0/assistant-go/internal/server/memory"
	"github.com/koopa0/assistant-go/internal/server/system"
	"github.com/koopa0/assistant-go/internal/server/tools"
	"github.com/koopa0/assistant-go/internal/server/users"
	"github.com/koopa0/assistant-go/internal/server/websocket"
	"github.com/koopa0/assistant-go/internal/storage/postgres/sqlc"
)

// Server represents the HTTP API server
type Server struct {
	assistant *assistant.Assistant
	logger    *slog.Logger
	server    *http.Server
	mux       *http.ServeMux
	config    config.ServerConfig
	metrics   *observability.Metrics
}

// New creates a new HTTP API server
func New(cfg config.ServerConfig, assistant *assistant.Assistant, logger *slog.Logger, metrics *observability.Metrics) (*Server, error) {
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
		metrics:   metrics,
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

	// Get database queries
	var sqlcQueries *sqlc.Queries
	if s.assistant.GetDB() != nil {
		sqlcQueries = s.assistant.GetDB().GetQueries()
	}

	// JWT secret for auth (in production, load from config)
	jwtSecret := "your-secret-key-change-in-production"

	// === Domain-Driven API Services ===

	// Auth Service
	authService := auth.NewAuthService(sqlcQueries, s.logger, s.metrics, jwtSecret)
	authHandler := auth.NewHTTPHandler(authService)
	authHandler.RegisterRoutes(s.mux)

	// Users Service
	if sqlcQueries != nil {
		usersService := users.NewUserService(sqlcQueries, s.logger, s.metrics)
		usersHandler := users.NewHTTPHandler(usersService, s.logger)
		usersHandler.RegisterRoutes(s.mux)
	}

	// Memory Service - 記憶系統 API
	if sqlcQueries != nil {
		memoryService := memory.NewMemoryService(sqlcQueries, s.logger, s.metrics)
		memoryHandler := memory.NewHTTPHandler(memoryService, s.logger)
		memoryHandler.RegisterRoutes(s.mux)
	}

	// Enhanced Conversation Service - 增強對話 API
	if sqlcQueries != nil {
		enhancedConvService := conversation.NewEnhancedConversationService(sqlcQueries, s.logger, s.metrics)
		enhancedConvHandler := conversation.NewEnhancedHTTPHandler(enhancedConvService, s.logger)
		enhancedConvHandler.RegisterEnhancedRoutes(s.mux)
	}

	// Enhanced Tools Service - 增強工具 API
	enhancedToolsService := tools.NewEnhancedToolService(s.assistant, sqlcQueries, s.logger, s.metrics)
	enhancedToolsHandler := tools.NewEnhancedHTTPHandler(enhancedToolsService, s.logger)
	enhancedToolsHandler.RegisterEnhancedRoutes(s.mux)

	// Tools Service
	toolsService := tools.NewToolService(s.assistant, sqlcQueries, s.logger, s.metrics)
	toolsHandler := tools.NewHTTPHandler(toolsService, s.logger)
	toolsHandler.RegisterRoutes(s.mux)

	// System Service
	systemService := system.NewSystemService(s.assistant, sqlcQueries, s.logger, s.metrics)
	systemHandler := system.NewHTTPHandler(systemService)
	systemHandler.RegisterRoutes(s.mux)

	// Conversation Service
	conversationService := conversation.NewConversationService(s.assistant, sqlcQueries, s.logger, s.metrics)
	conversationHandler := conversation.NewHTTPHandler(conversationService)
	conversationHandler.RegisterRoutes(s.mux)

	// WebSocket Service
	wsService := websocket.NewWebSocketService(s.assistant, s.logger, s.metrics)
	wsHandler := websocket.NewHTTPHandler(wsService, s.logger)
	wsHandler.RegisterRoutes(s.mux)
	// Start WebSocket background tasks
	go wsService.Start(context.Background())

	// Chat Service (API v1 compatible)
	chatService := chat.NewChatService(s.assistant, s.logger, s.metrics)
	chatHandler := chat.NewHTTPHandler(chatService)
	chatHandler.RegisterRoutes(s.mux)

	// Analytics Service (includes timeline and insights)
	if sqlcQueries != nil {
		analyticsService := analytics.NewAnalyticsService(sqlcQueries, s.logger, s.metrics)
		analyticsHandler := analytics.NewHTTPHandler(analyticsService)
		analyticsHandler.RegisterRoutes(s.mux)
	}

	// Knowledge Service
	knowledgeService := knowledge.NewKnowledgeService(s.assistant, s.logger, s.metrics)
	knowledgeHandler := knowledge.NewHTTPHandler(knowledgeService)
	knowledgeHandler.RegisterRoutes(s.mux)

	// Learning Service
	if sqlcQueries != nil {
		learningService := learning.NewLearningService(sqlcQueries, s.logger, s.metrics)
		learningHandler := learning.NewHTTPHandler(learningService)
		learningHandler.RegisterRoutes(s.mux)
	}

	// Collaboration Service
	if sqlcQueries != nil {
		collaborationService := collaboration.NewCollaborationService(sqlcQueries, s.logger, s.metrics)
		collaborationHandler := collaboration.NewHTTPHandler(collaborationService)
		collaborationHandler.RegisterRoutes(s.mux)
	}

	// LangChain Service (if available)
	if langchainService := s.assistant.GetLangChainService(); langchainService != nil {
		langchainSvc := langchain.NewLangChainService(langchainService, s.logger)
		langchainHandler := langchain.NewHTTPHandler(langchainSvc, s.logger)
		langchainHandler.RegisterRoutes(s.mux)
		s.logger.Info("LangChain API routes registered")
	}

	// 保持向後相容的舊 API 路由
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/status", s.handleStatus)
	s.mux.HandleFunc("POST /api/query", s.handleQuery)
	s.mux.HandleFunc("GET /api/conversations", s.handleListConversations)
	s.mux.HandleFunc("GET /api/conversations/{id}", s.handleGetConversation)
	s.mux.HandleFunc("DELETE /api/conversations/{id}", s.handleDeleteConversation)
	s.mux.HandleFunc("GET /api/tools", s.handleListTools)
	s.mux.HandleFunc("GET /api/tools/{name}", s.handleGetTool)
	s.mux.HandleFunc("POST /api/tools/{name}/execute", s.handleExecuteTool)

	// 根路由 - 提供 API 資訊
	s.mux.HandleFunc("GET /", s.handleRoot)

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
	toolReq := &assistant.ToolExecutionRequest{
		ToolName: toolName,
		Input:    request.Input,
		Config:   request.Config,
	}
	result, err := s.assistant.ExecuteTool(ctx, toolReq)
	if err != nil {
		s.logger.Error("Tool execution failed",
			slog.String("tool", toolName),
			slog.Any("error", err))
		http.Error(w, fmt.Sprintf("Tool execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, result)
}

// handleRoot provides API information at the root endpoint
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	apiInfo := map[string]interface{}{
		"name":        "Assistant API",
		"version":     "v1.0.0",
		"description": "智慧開發助手 API",
		"endpoints": map[string]interface{}{
			"health": "/api/health",
			"status": "/api/status",
			"v1": map[string]interface{}{
				"base":          "/api/v1",
				"chat":          "/api/v1/chat/completions",
				"conversations": "/api/v1/conversations",
				"memory":        "/api/v1/memory",
				"knowledge":     "/api/v1/knowledge",
				"skills":        "/api/v1/skills",
				"tools":         "/api/v1/tools",
			},
		},
		"documentation": map[string]interface{}{
			"swagger": "/api/docs",
			"openapi": "/api/openapi.json",
		},
		"features": []string{
			"聊天完成（OpenAI 相容）",
			"對話管理",
			"記憶體系統",
			"知識管理",
			"技能執行",
			"工具整合",
			"即時串流",
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	s.writeJSONResponse(w, http.StatusOK, apiInfo)
}
