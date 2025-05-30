// Package assistant provides the core assistant functionality for the GoAssistant application.
// It includes request processing, tool orchestration, and AI provider integration.
package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
	"github.com/koopa0/assistant-go/internal/tools"
	"github.com/koopa0/assistant-go/internal/tools/godev"
)

// Assistant is the core orchestrator of the intelligent development companion.
// It coordinates all major subsystems including:
// - Request processing through the Processor
// - Tool management via the Registry
// - Conversation management through ContextManager
// - Database operations for persistence
//
// The Assistant provides a unified interface for:
// - Processing user queries with context awareness
// - Managing conversations and message history
// - Executing tools directly when needed
// - Health monitoring of all subsystems
// - Graceful shutdown and resource cleanup
type Assistant struct {
	config    *config.Config           // Application configuration
	db        postgres.ClientInterface // Database client for persistence
	logger    *slog.Logger             // Structured logger
	registry  *tools.Registry          // Tool registry for available tools
	processor *Processor               // Request processing pipeline
	context   *ContextManager          // Conversation context manager
}

// QueryRequest represents a comprehensive query request to the Assistant.
// It includes the query text and various optional parameters to control
// the processing behavior.
type QueryRequest struct {
	// Query is the user's input text (required)
	Query string `json:"query"`

	// ConversationID links this query to an existing conversation
	ConversationID *string `json:"conversation_id,omitempty"`

	// UserID identifies the user making the request
	UserID *string `json:"user_id,omitempty"`

	// Context provides additional key-value pairs for request processing
	Context map[string]interface{} `json:"context,omitempty"`

	// Tools specifies which tools should be available for this request
	Tools []string `json:"tools,omitempty"`

	// Provider overrides the default AI provider (claude, gemini)
	Provider *string `json:"provider,omitempty"`

	// Model overrides the default model for the provider
	Model *string `json:"model,omitempty"`

	// MaxTokens limits the response length (provider-specific defaults apply)
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls response randomness (0.0 = deterministic, 1.0 = creative)
	Temperature float64 `json:"temperature,omitempty"`

	// SystemPrompt overrides the default system prompt
	SystemPrompt *string `json:"system_prompt,omitempty"`
}

// QueryResponse represents the complete response from processing a query.
// It includes the generated response text along with comprehensive metadata
// about the processing pipeline.
type QueryResponse struct {
	// Response is the generated text response from the AI
	Response string `json:"response"`

	// ConversationID identifies the conversation this response belongs to
	ConversationID string `json:"conversation_id"`

	// MessageID uniquely identifies this specific message
	MessageID string `json:"message_id"`

	// Provider indicates which AI provider was used (claude, gemini)
	Provider string `json:"provider"`

	// Model specifies the exact model used for generation
	Model string `json:"model"`

	// TokensUsed reports the total token consumption
	TokensUsed int `json:"tokens_used"`

	// ExecutionTime measures the total processing duration
	ExecutionTime time.Duration `json:"execution_time"`

	// ToolsUsed lists any tools that were executed during processing
	ToolsUsed []string `json:"tools_used,omitempty"`

	// Context contains additional metadata about the processing
	Context map[string]interface{} `json:"context,omitempty"`

	// Error contains error message if something went wrong
	Error *string `json:"error,omitempty"`
}

// New creates and initializes a new Assistant instance.
// It sets up all required subsystems including:
// - Tool registry with built-in tools
// - Context manager for conversation handling
// - Processor for request pipeline
//
// Parameters:
//   - ctx: Context for initialization (not stored)
//   - cfg: Application configuration (required)
//   - db: Database client for persistence (required)
//   - logger: Structured logger instance (required)
//
// Returns:
//   - *Assistant: Initialized assistant ready for use
//   - error: Configuration or initialization error
func New(ctx context.Context, cfg *config.Config, db postgres.ClientInterface, logger *slog.Logger) (*Assistant, error) {
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

// ProcessQuery is a simplified interface for processing a plain text query.
// It creates a minimal QueryRequest and returns just the response text.
//
// For more control over the processing, use ProcessQueryRequest instead.
//
// Parameters:
//   - ctx: Context for request processing
//   - query: The user's input text
//
// Returns:
//   - string: The generated response text
//   - error: Processing error if any
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

// ProcessQueryRequest processes a fully structured query request.
// This is the main entry point for query processing and provides
// full control over all processing parameters.
//
// The method:
// 1. Validates the request
// 2. Delegates to the Processor for pipeline execution
// 3. Tracks execution time
// 4. Returns comprehensive response with metadata
//
// Parameters:
//   - ctx: Context for request processing
//   - request: Structured request with query and options
//
// Returns:
//   - *QueryResponse: Complete response with metadata
//   - error: Processing error if any
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

// Health performs comprehensive health checks on all subsystems.
// It verifies:
// - Database connectivity and operations
// - Tool registry functionality
// - Processor pipeline health
//
// This method is suitable for use as a health check endpoint.
//
// Returns:
//   - error: nil if all systems are healthy, error with details otherwise
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
	a.logger.Debug("Registering built-in tools...")

	// Register Go development tools
	if err := a.registerGoTools(); err != nil {
		return fmt.Errorf("failed to register Go tools: %w", err)
	}

	// TODO: Register additional tools as they are implemented
	// PostgreSQL tools, Docker tools, Kubernetes tools, etc.

	a.logger.Debug("Built-in tools registered successfully")
	return nil
}

// registerGoTools registers Go development tools
func (a *Assistant) registerGoTools() error {
	// Register Go Analyzer
	if err := a.registry.Register("go_analyzer", func(config map[string]interface{}, logger *slog.Logger) (tools.Tool, error) {
		return godev.NewGoAnalyzer(config, logger)
	}); err != nil {
		return fmt.Errorf("failed to register go_analyzer: %w", err)
	}

	// Register Go Formatter
	if err := a.registry.Register("go_formatter", func(config map[string]interface{}, logger *slog.Logger) (tools.Tool, error) {
		return godev.NewGoFormatter(config, logger)
	}); err != nil {
		return fmt.Errorf("failed to register go_formatter: %w", err)
	}

	// Register Go Tester
	if err := a.registry.Register("go_tester", func(config map[string]interface{}, logger *slog.Logger) (tools.Tool, error) {
		return godev.NewGoTester(config, logger)
	}); err != nil {
		return fmt.Errorf("failed to register go_tester: %w", err)
	}

	// Register Go Builder
	if err := a.registry.Register("go_builder", func(config map[string]interface{}, logger *slog.Logger) (tools.Tool, error) {
		return godev.NewGoBuilder(config, logger)
	}); err != nil {
		return fmt.Errorf("failed to register go_builder: %w", err)
	}

	// Register Go Dependency Analyzer
	if err := a.registry.Register("go_dependency_analyzer", func(config map[string]interface{}, logger *slog.Logger) (tools.Tool, error) {
		return godev.NewGoDependencyAnalyzer(config, logger)
	}); err != nil {
		return fmt.Errorf("failed to register go_dependency_analyzer: %w", err)
	}

	a.logger.Debug("Go development tools registered")
	return nil
}

// Stats returns assistant statistics
func (a *Assistant) Stats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Database stats
	dbStats := a.db.Stats()
	if dbStats != nil {
		// Only access stats if we have a real database connection
		stats["database"] = map[string]interface{}{
			"total_connections":        dbStats.TotalConns(),
			"idle_connections":         dbStats.IdleConns(),
			"acquired_connections":     dbStats.AcquiredConns(),
			"constructing_connections": dbStats.ConstructingConns(),
		}
	} else {
		// Demo mode - return mock stats
		stats["database"] = map[string]interface{}{
			"status":               "demo_mode",
			"total_connections":    0,
			"idle_connections":     0,
			"acquired_connections": 0,
		}
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

// ExecuteTool provides direct access to tool execution without going through
// the full query processing pipeline. This is useful for:
// - Testing tools in isolation
// - Building tool-specific interfaces
// - Programmatic tool execution
//
// Parameters:
//   - ctx: Context for tool execution
//   - toolName: Name of the registered tool to execute
//   - input: Tool-specific input parameters
//   - config: Tool-specific configuration (nil for defaults)
//
// Returns:
//   - *tools.ToolResult: Tool execution results
//   - error: Execution error if any
func (a *Assistant) ExecuteTool(ctx context.Context, toolName string, input map[string]interface{}, config map[string]interface{}) (*tools.ToolResult, error) {
	a.logger.Info("Executing tool directly",
		slog.String("tool", toolName),
		slog.Any("input", input))

	if config == nil {
		config = make(map[string]interface{})
	}

	result, err := a.registry.Execute(ctx, toolName, input, config)
	if err != nil {
		a.logger.Error("Tool execution failed",
			slog.String("tool", toolName),
			slog.Any("error", err))
		return nil, err
	}

	a.logger.Info("Tool execution completed",
		slog.String("tool", toolName),
		slog.Bool("success", result.Success),
		slog.Duration("execution_time", result.ExecutionTime))

	return result, nil
}
