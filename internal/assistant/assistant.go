package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
	"github.com/koopa0/assistant-go/internal/tools"
)

// Assistant represents the core assistant interface
type Assistant struct {
	config    *config.Config
	db        *postgres.Client
	logger    *slog.Logger
	registry  *tools.Registry
	processor *Processor
	context   *ContextManager
}

// QueryRequest represents a query request
type QueryRequest struct {
	Query          string                 `json:"query"`
	ConversationID *string                `json:"conversation_id,omitempty"`
	UserID         *string                `json:"user_id,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Tools          []string               `json:"tools,omitempty"`
	Provider       *string                `json:"provider,omitempty"`
	Model          *string                `json:"model,omitempty"`
	MaxTokens      int                    `json:"max_tokens,omitempty"`
	Temperature    float64                `json:"temperature,omitempty"`
	SystemPrompt   *string                `json:"system_prompt,omitempty"`
}

// QueryResponse represents a query response
type QueryResponse struct {
	Response       string                 `json:"response"`
	ConversationID string                 `json:"conversation_id"`
	MessageID      string                 `json:"message_id"`
	Provider       string                 `json:"provider"`
	Model          string                 `json:"model"`
	TokensUsed     int                    `json:"tokens_used"`
	ExecutionTime  time.Duration          `json:"execution_time"`
	ToolsUsed      []string               `json:"tools_used,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Error          *string                `json:"error,omitempty"`
}

// New creates a new Assistant instance
func New(ctx context.Context, cfg *config.Config, db *postgres.Client, logger *slog.Logger) (*Assistant, error) {
	if cfg == nil {
		return nil, NewConfigurationError("config", fmt.Errorf("config is required"))
	}
	if db == nil {
		return nil, NewConfigurationError("database", fmt.Errorf("database client is required"))
	}
	if logger == nil {
		return nil, NewConfigurationError("logger", fmt.Errorf("logger is required"))
	}

	// Initialize tool registry
	registry := tools.NewRegistry(logger)

	// Initialize context manager
	contextManager, err := NewContextManager(db, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize context manager: %w", err)
	}

	// Initialize processor
	processor, err := NewProcessor(cfg, db, registry, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize processor: %w", err)
	}

	assistant := &Assistant{
		config:    cfg,
		db:        db,
		logger:    logger,
		registry:  registry,
		processor: processor,
		context:   contextManager,
	}

	// Register built-in tools
	if err := assistant.registerBuiltinTools(ctx); err != nil {
		return nil, fmt.Errorf("failed to register builtin tools: %w", err)
	}

	logger.Info("Assistant initialized successfully",
		slog.String("mode", cfg.Mode),
		slog.String("default_provider", cfg.AI.DefaultProvider))

	return assistant, nil
}

// ProcessQuery processes a user query and returns a response
func (a *Assistant) ProcessQuery(ctx context.Context, query string) (string, error) {
	request := &QueryRequest{
		Query: query,
	}

	response, err := a.ProcessQueryRequest(ctx, request)
	if err != nil {
		return "", err
	}

	return response.Response, nil
}

// ProcessQueryRequest processes a structured query request
func (a *Assistant) ProcessQueryRequest(ctx context.Context, request *QueryRequest) (*QueryResponse, error) {
	if request == nil {
		return nil, NewInvalidInputError("request is required", nil)
	}

	if request.Query == "" {
		return nil, NewInvalidInputError("query is required", nil)
	}

	startTime := time.Now()

	a.logger.Info("Processing query request",
		slog.String("query", request.Query),
		slog.Any("conversation_id", request.ConversationID),
		slog.Any("user_id", request.UserID))

	// Process the request through the processor
	response, err := a.processor.Process(ctx, request)
	if err != nil {
		a.logger.Error("Failed to process query",
			slog.String("query", request.Query),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to process query: %w", err)
	}

	// Calculate execution time
	response.ExecutionTime = time.Since(startTime)

	a.logger.Info("Query processed successfully",
		slog.String("conversation_id", response.ConversationID),
		slog.String("message_id", response.MessageID),
		slog.Duration("execution_time", response.ExecutionTime),
		slog.Int("tokens_used", response.TokensUsed))

	return response, nil
}

// GetConversation retrieves a conversation by ID
func (a *Assistant) GetConversation(ctx context.Context, conversationID string) (*Conversation, error) {
	return a.context.GetConversation(ctx, conversationID)
}

// ListConversations lists conversations for a user
func (a *Assistant) ListConversations(ctx context.Context, userID string, limit, offset int) ([]*Conversation, error) {
	return a.context.ListConversations(ctx, userID, limit, offset)
}

// DeleteConversation deletes a conversation
func (a *Assistant) DeleteConversation(ctx context.Context, conversationID string) error {
	return a.context.DeleteConversation(ctx, conversationID)
}

// GetAvailableTools returns a list of available tools
func (a *Assistant) GetAvailableTools() []tools.ToolInfo {
	return a.registry.ListTools()
}

// GetToolInfo returns information about a specific tool
func (a *Assistant) GetToolInfo(toolName string) (*tools.ToolInfo, error) {
	return a.registry.GetToolInfo(toolName)
}

// Health checks the health of the assistant and its dependencies
func (a *Assistant) Health(ctx context.Context) error {
	// Check database health
	if err := a.db.Health(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Check tool registry health
	if err := a.registry.Health(ctx); err != nil {
		return fmt.Errorf("tool registry health check failed: %w", err)
	}

	// Check processor health
	if err := a.processor.Health(ctx); err != nil {
		return fmt.Errorf("processor health check failed: %w", err)
	}

	return nil
}

// Close gracefully shuts down the assistant
func (a *Assistant) Close(ctx context.Context) error {
	a.logger.Info("Shutting down assistant...")

	// Close processor
	if err := a.processor.Close(ctx); err != nil {
		a.logger.Error("Failed to close processor", slog.Any("error", err))
	}

	// Close context manager
	if err := a.context.Close(ctx); err != nil {
		a.logger.Error("Failed to close context manager", slog.Any("error", err))
	}

	// Close tool registry
	if err := a.registry.Close(ctx); err != nil {
		a.logger.Error("Failed to close tool registry", slog.Any("error", err))
	}

	a.logger.Info("Assistant shutdown complete")
	return nil
}

// registerBuiltinTools registers the built-in tools
func (a *Assistant) registerBuiltinTools(ctx context.Context) error {
	// TODO: Register built-in tools as they are implemented
	// This will be expanded in Phase 3 when we implement the tool system

	a.logger.Debug("Registering built-in tools...")

	// Placeholder for tool registration
	// Example:
	// if err := a.registry.Register("postgres", postgresToolFactory); err != nil {
	//     return fmt.Errorf("failed to register postgres tool: %w", err)
	// }

	a.logger.Debug("Built-in tools registered successfully")
	return nil
}

// Stats returns assistant statistics
func (a *Assistant) Stats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Database stats
	dbStats := a.db.Stats()
	stats["database"] = map[string]interface{}{
		"total_connections":        dbStats.TotalConns(),
		"idle_connections":         dbStats.IdleConns(),
		"acquired_connections":     dbStats.AcquiredConns(),
		"constructing_connections": dbStats.ConstructingConns(),
	}

	// Tool registry stats
	toolStats, err := a.registry.Stats(ctx)
	if err != nil {
		a.logger.Warn("Failed to get tool registry stats", slog.Any("error", err))
	} else {
		stats["tools"] = toolStats
	}

	// Processor stats
	processorStats, err := a.processor.Stats(ctx)
	if err != nil {
		a.logger.Warn("Failed to get processor stats", slog.Any("error", err))
	} else {
		stats["processor"] = processorStats
	}

	return stats, nil
}
