// Package server provides HTTP API server implementation with middleware support,
// request/response handling, and graceful shutdown capabilities.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/analytics"
	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/chat"
	"github.com/koopa0/assistant-go/internal/collaboration"
	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/conversation"
	assterrors "github.com/koopa0/assistant-go/internal/errors"
	"github.com/koopa0/assistant-go/internal/knowledge"
	langchainhttp "github.com/koopa0/assistant-go/internal/langchain/http"
	"github.com/koopa0/assistant-go/internal/learning"
	"github.com/koopa0/assistant-go/internal/memory"
	memoryhttp "github.com/koopa0/assistant-go/internal/memory/http"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/server/middleware"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
	"github.com/koopa0/assistant-go/internal/system"
	toolhttp "github.com/koopa0/assistant-go/internal/tool/http"
	"github.com/koopa0/assistant-go/internal/transport/sse"
	"github.com/koopa0/assistant-go/internal/transport/websocket"
	"github.com/koopa0/assistant-go/internal/user"
)

// Server represents the HTTP API server
type Server struct {
	assistant   *assistant.Assistant
	logger      *slog.Logger
	server      *http.Server
	mux         *http.ServeMux
	config      config.ServerConfig
	metrics     *observability.Metrics
	rateLimiter *middleware.RateLimitMiddleware
	authService user.JWTService
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

	// Create server instance first
	server := &Server{
		config:    cfg,
		assistant: assistant,
		logger:    observability.ServerLogger(logger, "http"),
		mux:       mux,
		metrics:   metrics,
	}

	// Create HTTP server with middleware
	httpServer := &http.Server{
		Addr:         cfg.Address,
		Handler:      server.withMiddleware(mux),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	server.server = httpServer

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
	// Get database queries
	var sqlcQueries *sqlc.Queries
	if s.assistant.GetDB() != nil {
		sqlcQueries = s.assistant.GetDB().GetQueries()
	}

	// JWT secret 必須從配置載入，不允許預設值
	jwtSecret := s.config.Security.JWTSecret
	if jwtSecret == "" {
		s.logger.Error("JWT secret 未配置，請設定 SECURITY_JWT_SECRET 環境變數")
		panic("JWT secret is required for production")
	}

	// === Domain-Driven API Services ===

	// Auth Service
	// Instantiate TokenService first, as it's a dependency for AuthService
	// and also used by the server's authMiddleware via s.authService.
	tokenService := user.NewTokenService(jwtSecret, "assistant-api", s.logger)
	s.authService = tokenService // Assign the concrete TokenService to the JWTService interface field for middleware use.

	// Create AuthService, injecting the TokenService.
	authServiceForHandler := user.NewAuthService(sqlcQueries, s.logger, s.metrics, tokenService)
	authHandler := user.NewAuthHTTPHandler(authServiceForHandler)
	authHandler.RegisterRoutes(s.mux)

	// Users Service
	if sqlcQueries != nil {
		usersService := user.NewUserService(sqlcQueries, s.logger, s.metrics)
		usersHandler := user.NewHTTPHandler(usersService, s.logger)
		usersHandler.RegisterRoutes(s.mux)
	}

	// Memory Service - 記憶系統 API
	if sqlcQueries != nil {
		// Instantiate WorkingMemory, which is an in-memory component for the core memory service.
		// TODO: Consider making working memory capacity configurable if not already.
		defaultWorkingMemoryCapacity := 100
		workingMemoryInstance := memory.NewWorkingMemory(defaultWorkingMemoryCapacity)

		// Create the core memory service, injecting the SQLC queries and the working memory instance.
		coreMemoryService := memory.NewService(sqlcQueries, s.logger, workingMemoryInstance)
		// Create database memory store using the core service.
		// This store adapts the coreMemoryService to the MemoryStore interface required by HTTPMemoryService.
		memoryStore := memory.NewDatabaseMemoryStore(coreMemoryService, s.logger)
		// Create HTTP memory service that implements HTTPMemoryServiceInterface.
		// This service is responsible for handling HTTP API logic for memory operations.
		httpMemoryService := memory.NewMemoryService(memoryStore, s.logger, s.metrics)
		memoryHandler := memoryhttp.NewHandler(httpMemoryService, s.logger)
		memoryHandler.RegisterRoutes(s.mux)
	}

	// Enhanced Conversation Service - 增強對話 API
	if sqlcQueries != nil {
		enhancedConvService := conversation.NewEnhancedConversationService(sqlcQueries, s.logger, s.metrics)
		enhancedConvHandler := conversation.NewEnhancedHTTPHandler(enhancedConvService, s.logger)
		enhancedConvHandler.RegisterEnhancedRoutes(s.mux)
	}

	// Consolidated Tools API - 工具系統 API
	toolsHandler := toolhttp.NewHandler(s.assistant.AsToolInterface(), sqlcQueries, s.logger, s.metrics)
	toolsHandler.RegisterRoutes(s.mux)

	// System Service
	if sqlcQueries != nil {
		systemService := system.NewService(s.assistant, sqlcQueries, s.logger, s.metrics)
		systemHandler := system.NewHandler(systemService, s.logger)
		systemHandler.RegisterRoutes(s.mux)
	}

	// Conversation Service - Commented out because EnhancedHTTPHandler already registers these routes
	// conversationService := conversation.NewConversationService(s.assistant.AsConversationInterface(), sqlcQueries, s.logger, s.metrics)
	// conversationHandler := conversation.NewHTTPHandler(conversationService)
	// conversationHandler.RegisterRoutes(s.mux)

	// WebSocket Service
	wsService := websocket.NewWebSocketService(s.assistant, s.logger, s.metrics)
	// WebSocket 使用相同的 JWT secret（已在上面驗證過）
	// Create another AuthService instance for WebSocket, injecting the same TokenService.
	wsAuthService := user.NewAuthService(sqlcQueries, s.logger, s.metrics, tokenService)
	wsHandler := websocket.NewHTTPHandler(wsService, wsAuthService, s.logger)
	wsHandler.RegisterRoutes(s.mux)
	// Start WebSocket background tasks
	go wsService.Start(context.Background())

	// SSE Streaming Service
	sseHandler := sse.NewHandler(s.assistant, s.logger)
	sseHandler.RegisterRoutes(s.mux)

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
	if sqlcQueries != nil {
		knowledgeService := knowledge.NewKnowledgeService(s.assistant, s.logger, s.metrics, sqlcQueries)
		knowledgeHandler := knowledge.NewHTTPHandler(knowledgeService)
		knowledgeHandler.RegisterRoutes(s.mux)
	}

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
		langchainHandler := langchainhttp.NewHandler(langchainService, s.logger)
		// Register individual routes
		s.mux.HandleFunc("GET /api/langchain/agents", langchainHandler.GetAvailableAgents)
		s.mux.HandleFunc("POST /api/langchain/agents/{type}/execute", langchainHandler.ExecuteAgent)
		s.mux.HandleFunc("POST /api/langchain/execute", langchainHandler.ExecutePrompt)
		s.logger.Info("LangChain API routes registered")
	}

	// 保持向後相容的舊 API 路由
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/status", s.handleStatus)
	s.mux.HandleFunc("POST /api/query", s.handleQuery)
	// Conversation routes are already registered by conversationHandler above
	// s.mux.HandleFunc("GET /api/conversations", s.handleListConversations)
	// s.mux.HandleFunc("GET /api/conversations/{id}", s.handleGetConversation)
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

	// Rate limiting middleware
	handler = s.rateLimitMiddleware(handler)

	// Authentication middleware (innermost, closest to handlers)
	if s.authService != nil {
		handler = s.authMiddleware(s.authService)(handler)
	}

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

	response := StatusResponse{
		Status:    "running",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Stats:     stats,
	}
	s.writeJSONResponse(w, http.StatusOK, response)
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

	// Extract user ID from context (set by auth middleware)
	if userID, ok := ctx.Value("user_id").(string); ok {
		request.UserID = &userID
	}

	response, err := s.assistant.ProcessQueryRequest(ctx, &request)
	if err != nil {
		s.logger.Error("Query processing failed", slog.Any("error", err))

		// Handle different error types
		if assistantErr := assterrors.GetAssistantError(err); assistantErr != nil {
			// Map error categories to HTTP status codes
			var statusCode int
			switch assistantErr.Category {
			case assterrors.CategoryValidation:
				statusCode = http.StatusBadRequest
			case assterrors.CategoryAuthentication, assterrors.CategoryAuthorization:
				statusCode = http.StatusUnauthorized
			case assterrors.CategoryInfrastructure:
				if assistantErr.Retryable {
					statusCode = http.StatusServiceUnavailable
				} else {
					statusCode = http.StatusInternalServerError
				}
			default:
				statusCode = http.StatusInternalServerError
			}
			http.Error(w, assistantErr.UserMessage, statusCode)
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
	conversationObj, err := s.assistant.GetConversation(ctx, conversationID) // Renamed from 'conversation' to avoid conflict
	if err != nil {
		// Check for specific 'not found' error to return HTTP 404.
		if conversation.IsConversationNotFoundError(err) { // Using the helper from conversation.errors
			s.logger.Warn("Conversation not found", slog.String("conversation_id", conversationID), slog.Any("error", err))
			s.writeErrorResponse(w, http.StatusNotFound, "Conversation not found")
		} else {
			// For other errors, log details and return a generic server error.
			// Consider using assterrors.GetAssistantError if applicable for more structured responses.
			s.logger.Error("Failed to get conversation", slog.String("conversation_id", conversationID), slog.Any("error", err))
			s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve conversation")
		}
		return
	}

	s.writeJSONResponse(w, http.StatusOK, conversationObj)
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
		// Check if it's a conversation not found error
		if conversation.IsConversationNotFoundError(err) {
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
		s.logger.Error("Tool execution failed via API",
			slog.String("tool", toolName),
			slog.Any("error", err))

		// Use structured error handling for tool execution failures.
		if assistantErr := assterrors.GetAssistantError(err); assistantErr != nil {
			var statusCode int
			switch assistantErr.Category {
			case assterrors.CategoryValidation: // e.g., bad tool input
				statusCode = http.StatusBadRequest
			case assterrors.CategoryTool: // Specific tool execution error
				// Consider if a more specific 5xx is available based on assistantErr.Code
				statusCode = http.StatusInternalServerError
				if assistantErr.Retryable {
					// Potentially map to 503 if retryable, but 500 is generally safe for tool errors.
				}
			// Add other relevant categories if applicable
			default:
				statusCode = http.StatusInternalServerError
			}
			s.writeErrorResponse(w, statusCode, assistantErr.UserMessage)
		} else {
			// Fallback for non-AssistantError types
			s.writeErrorResponse(w, http.StatusInternalServerError, "Tool execution failed due to an internal error")
		}
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
