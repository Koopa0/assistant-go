// Package assistant provides the core assistant functionality for the GoAssistant application.
// It includes request processing, tool orchestration, and AI provider integration.
package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/conversation"
	"github.com/koopa0/assistant-go/internal/langchain"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
	"github.com/koopa0/assistant-go/internal/tool"
	"github.com/koopa0/assistant-go/internal/tool/docker"
	"github.com/koopa0/assistant-go/internal/tool/godev"
	postgrestool "github.com/koopa0/assistant-go/internal/tool/postgres"
)

// Assistant is the core orchestrator of the intelligent development companion.
// It coordinates all major subsystems including:
// - Request processing through the Processor
// - Tool management via the Registry
// - Conversation management through Manager
// - Database operations for persistence
//
// The Assistant provides a unified interface for:
// - Processing user queries with context awareness
// - Managing conversations and message history
// - Executing tools directly when needed
// - Health monitoring of all subsystems
// - Graceful shutdown and resource cleanup
type Assistant struct {
	config           *config.Config                   // Application configuration
	db               postgres.DB                      // Database client for persistence
	logger           *slog.Logger                     // Structured logger
	registry         *tool.Registry                   // Tool registry for available tools
	processor        *Processor                       // Request processing pipeline
	conversationMgr  conversation.ConversationService // Conversation service
	langchainService *langchain.Service               // LangChain integration service
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
	// TODO: Replace with typed RequestContext struct
	Context map[string]any `json:"context,omitempty"`

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
	// TODO: Replace with typed ResponseContext struct
	Context map[string]any `json:"context,omitempty"`

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
func New(ctx context.Context, cfg *config.Config, db postgres.DB, logger *slog.Logger) (*Assistant, error) {
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
	registry := tool.NewRegistry(logger)

	// Initialize conversation service using factory function
	conversationMgr := conversation.NewConversationSystem(db.GetQueries(), logger)

	// Initialize processor
	processor, err := NewProcessor(cfg, db, registry, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize processor: %w", err)
	}

	// Initialize LangChain service if configured (optional)
	var langchainService *langchain.Service
	if cfg.Tools.LangChain.EnableMemory {
		// Try to create a working LangChain service
		// Create a simple LangChain client first
		langchainClient := &langchain.LangChainClient{
			// Will be nil for now - requires LLM setup
		}

		// Note: LangChain service creation might fail due to missing dependencies
		// This is expected in demo mode or when LangChain dependencies are not fully configured
		langchainService = langchain.NewService(langchainClient, logger, db.GetQueries())
		logger.Info("LangChain service initialized (limited functionality without LLM)")
	}

	assistant := &Assistant{
		config:           cfg,
		db:               db,
		logger:           logger,
		registry:         registry,
		processor:        processor,
		conversationMgr:  conversationMgr,
		langchainService: langchainService,
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
		return nil, NewAssistantInvalidInputError("request is required", request)
	}

	if request.Query == "" {
		return nil, NewAssistantEmptyInputError()
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
func (a *Assistant) GetConversation(ctx context.Context, conversationID string) (*conversation.Conversation, error) {
	return a.conversationMgr.GetConversation(ctx, conversationID)
}

// GetConversationMessages retrieves messages for a conversation
func (a *Assistant) GetConversationMessages(ctx context.Context, conversationID string) ([]*conversation.Message, error) {
	return a.conversationMgr.GetMessages(ctx, conversationID)
}

// ListConversations lists conversations for a user
func (a *Assistant) ListConversations(ctx context.Context, userID string, limit, offset int) ([]*conversation.Conversation, error) {
	// Note: Current implementation doesn't support pagination
	// TODO: Add pagination support to conversation manager
	return a.conversationMgr.ListConversations(ctx, userID)
}

// DeleteConversation deletes a conversation
func (a *Assistant) DeleteConversation(ctx context.Context, conversationID string) error {
	return a.conversationMgr.DeleteConversation(ctx, conversationID)
}

// GetAvailableTools returns a list of available tools
func (a *Assistant) GetAvailableTools() []Tool {
	toolInfos := a.registry.ListTools()
	tools := make([]Tool, 0, len(toolInfos))
	for _, info := range toolInfos {
		// Convert tool info to Tool struct
		t := Tool{
			Name:        info.Name,
			Description: info.Description,
			Category:    info.Category,
			Version:     info.Version,
			Author:      info.Author,
			IsEnabled:   info.IsEnabled,
		}

		// Convert Parameters if available
		if info.Parameters != nil {
			// Convert to map[string]interface{} for the API
			params := make(map[string]interface{})
			if info.Parameters.Properties != nil {
				for k, v := range info.Parameters.Properties {
					paramInfo := map[string]interface{}{
						"type":        v.Type,
						"description": v.Description,
					}
					if v.Default != nil {
						paramInfo["default"] = v.Default
					}
					if len(v.Enum) > 0 {
						paramInfo["enum"] = v.Enum
					}
					params[k] = paramInfo
				}
			}
			t.Parameters = params
		}

		tools = append(tools, t)
	}
	return tools
}

// GetToolInfo returns information about a specific tool
func (a *Assistant) GetToolInfo(toolName string) (*tool.ToolInfo, error) {
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
//
// HealthCheck performs a health check and returns status
// This implements the HealthChecker interface
func (a *Assistant) HealthCheck(ctx context.Context) error {
	return a.Health(ctx)
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

// GetDB returns the database client interface
func (a *Assistant) GetDB() postgres.DB {
	return a.db
}

// Close gracefully shuts down the assistant
func (a *Assistant) Close(ctx context.Context) error {
	a.logger.Info("Shutting down assistant...")

	// Close processor
	if err := a.processor.Close(ctx); err != nil {
		a.logger.Error("Failed to close processor", slog.Any("error", err))
	}

	// Note: conversation manager doesn't require explicit close

	// Close tool registry
	if err := a.registry.Close(ctx); err != nil {
		a.logger.Error("Failed to close tool registry", slog.Any("error", err))
	}

	a.logger.Info("Assistant shutdown complete")
	return nil
}

// registerBuiltinTools registers the built-in tools
func (a *Assistant) registerBuiltinTools(ctx context.Context) error {
	a.logger.Debug("Registering built-in tool...")

	// Register Go development tool factory
	godevFactory := func(cfg *tool.ToolConfig, logger *slog.Logger) (tool.Tool, error) {
		return godev.NewGoDevTool(logger), nil
	}
	if err := a.registry.Register("godev", godevFactory); err != nil {
		return fmt.Errorf("failed to register godev tool: %w", err)
	}

	// Register Docker tool factory
	dockerFactory := func(cfg *tool.ToolConfig, logger *slog.Logger) (tool.Tool, error) {
		return docker.NewDockerTool(logger), nil
	}
	if err := a.registry.Register("docker", dockerFactory); err != nil {
		return fmt.Errorf("failed to register docker tool: %w", err)
	}

	// Register PostgreSQL tool factory
	postgresFactory := func(cfg *tool.ToolConfig, logger *slog.Logger) (tool.Tool, error) {
		return postgrestool.NewPostgresTool(logger), nil
	}
	if err := a.registry.Register("postgres", postgresFactory); err != nil {
		return fmt.Errorf("failed to register postgres tool: %w", err)
	}

	a.logger.Debug("Built-in tools registered successfully",
		slog.Int("count", 3))
	return nil
}

// AssistantStats represents comprehensive statistics for the assistant
type AssistantStats struct {
	// Database contains database connection pool statistics
	Database *DatabaseStats `json:"database"`

	// Tools contains tool registry statistics
	Tools *ToolRegistryStats `json:"tools,omitempty"`

	// Processor contains request processor statistics
	Processor *ProcessorStats `json:"processor,omitempty"`
}

// DatabaseStats represents database pool statistics
type DatabaseStats struct {
	// Status indicates the database connection status
	Status string `json:"status"`

	// TotalConns is the total number of connections in the pool
	TotalConns int32 `json:"total_connections"`

	// IdleConns is the number of idle connections
	IdleConns int32 `json:"idle_connections"`

	// AcquiredConns is the number of currently acquired connections
	AcquiredConns int32 `json:"acquired_connections"`

	// ConstructingConns is the number of connections being constructed
	ConstructingConns int32 `json:"constructing_connections"`

	// MaxConns is the maximum number of connections allowed
	MaxConns int32 `json:"max_connections"`

	// NewConnsCount is the total number of new connections created
	NewConnsCount int64 `json:"new_connections_count"`

	// AcquireCount is the total number of successful connection acquisitions
	AcquireCount int64 `json:"acquire_count"`

	// AcquireDurationMs is the total duration spent acquiring connections in milliseconds
	AcquireDurationMs int64 `json:"acquire_duration_ms"`
}

// ToolRegistryStats represents tool registry statistics
type ToolRegistryStats struct {
	// RegisteredTools is the count of registered tools
	RegisteredTools int `json:"registered_tools"`

	// ExecutionCounts maps tool names to their execution counts
	ExecutionCounts map[string]int64 `json:"execution_counts,omitempty"`

	// AverageExecutionTimes maps tool names to their average execution times in milliseconds
	AverageExecutionTimes map[string]int64 `json:"average_execution_times_ms,omitempty"`
}

// GetStats returns current statistics
// This implements the StatsProvider interface
func (a *Assistant) GetStats(ctx context.Context) (*Stats, error) {
	// Get detailed stats first
	detailedStats, err := a.Stats(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to simplified Stats for the interface
	stats := &Stats{
		ConversationCount:   0,     // TODO: Get from conversation manager
		MessageCount:        0,     // TODO: Get from conversation manager
		ToolExecutions:      0,     // TODO: Get from processor stats
		AverageResponseTime: "0ms", // TODO: Calculate from processor stats
		LastActivityTime:    time.Now().Format(time.RFC3339),
		ActiveProviders:     []string{a.config.AI.DefaultProvider},
		ProviderUsage:       make(map[string]interface{}),
	}

	// Add tool execution counts if available
	if detailedStats.Tools != nil && detailedStats.Tools.ExecutionCounts != nil {
		totalExecutions := int64(0)
		for _, count := range detailedStats.Tools.ExecutionCounts {
			totalExecutions += count
		}
		stats.ToolExecutions = int(totalExecutions)
	}

	return stats, nil
}

// Stats returns assistant statistics
func (a *Assistant) Stats(ctx context.Context) (*AssistantStats, error) {
	stats := &AssistantStats{}

	// Database stats
	poolStats := a.db.GetPoolStats()
	if poolStats != nil {
		// Use typed pool statistics
		stats.Database = &DatabaseStats{
			Status:            "connected",
			TotalConns:        poolStats.TotalConns,
			IdleConns:         poolStats.IdleConns,
			AcquiredConns:     poolStats.AcquiredConns,
			ConstructingConns: poolStats.ConstructingConns,
			MaxConns:          poolStats.MaxConns,
			NewConnsCount:     poolStats.NewConnsCount,
			AcquireCount:      poolStats.AcquireCount,
			AcquireDurationMs: poolStats.AcquireDuration.Milliseconds(),
		}
	} else {
		// Demo mode - return mock stats
		stats.Database = &DatabaseStats{
			Status:            "demo_mode",
			TotalConns:        0,
			IdleConns:         0,
			AcquiredConns:     0,
			ConstructingConns: 0,
			MaxConns:          0,
		}
	}

	// Tool registry stats - temporarily keep as map until we update registry
	toolStatsMap, err := a.registry.Stats(ctx)
	if err != nil {
		a.logger.Warn("Failed to get tool registry stats", slog.Any("error", err))
	} else if toolStatsMap != nil {
		// Convert map to typed struct
		stats.Tools = &ToolRegistryStats{
			RegisteredTools: len(a.registry.ListTools()),
		}
		// TODO: Extract execution counts and times from toolStatsMap when registry is updated
	}

	// Processor stats - temporarily keep as map until we update processor
	processorStatsMap, err := a.processor.Stats(ctx)
	if err != nil {
		a.logger.Warn("Failed to get processor stats", slog.Any("error", err))
	} else if processorStatsMap != nil {
		// Convert map to typed struct using comprehensive ProcessorStats from types.go
		stats.Processor = &ProcessorStats{
			Processor: &ProcessorInfo{
				Status:  "healthy",
				Version: "1.0.0",   // TODO: Get from build info
				Uptime:  "unknown", // TODO: Track actual uptime
			},
			Health: &HealthStatus{
				Status:    "healthy",
				LastCheck: time.Now(),
			},
		}
		// TODO: Extract counts and times from processorStatsMap when processor is updated
	}

	return stats, nil
}

// ToolExecutionRequest is now defined in types.go

// ExecuteTool provides direct access to tool execution without going through
// the full query processing pipeline. This is useful for:
// - Testing tools in isolation
// - Building tool-specific interfaces
// - Programmatic tool execution
//
// Parameters:
//   - ctx: Context for tool execution
//   - req: Tool execution request with name, input, and config
//
// Returns:
//   - *tool.ToolResult: Tool execution results
//   - error: Execution error if any
func (a *Assistant) ExecuteTool(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error) {
	if req == nil {
		return nil, NewAssistantInvalidInputError("request is required", req)
	}

	if req.ToolName == "" {
		return nil, NewAssistantInvalidInputError("tool_name is required", req.ToolName)
	}

	a.logger.Info("Executing tool directly",
		slog.String("tool", req.ToolName),
		slog.Any("input", req.Input))

	// Convert legacy input and config to typed structures
	toolInput := &tool.ToolInput{
		Parameters: req.Input,
	}

	toolConfig := &tool.ToolConfig{}
	if req.Config != nil {
		// Convert config map to ToolConfig
		toolConfig = tool.ConvertLegacyConfig(req.Config)
	}

	result, err := a.registry.Execute(ctx, req.ToolName, toolInput, toolConfig)
	if err != nil {
		a.logger.Error("Tool execution failed",
			slog.String("tool", req.ToolName),
			slog.Any("error", err))
		return nil, err
	}

	a.logger.Info("Tool execution completed",
		slog.String("tool", req.ToolName),
		slog.Bool("success", result.Success),
		slog.Duration("execution_time", result.ExecutionTime))

	// Convert tool.ToolResult to ToolExecutionResponse
	executionResp := &ToolExecutionResponse{
		Success:       result.Success,
		Error:         result.Error,
		ExecutionTime: result.ExecutionTime,
		ToolsUsed:     []string{req.ToolName},
	}

	// Convert result data if present
	if result.Data != nil {
		executionResp.Data = &ToolResultData{
			Result: result.Data.Result,
			Output: result.Data.Output,
		}
		// Convert artifacts
		if len(result.Data.Artifacts) > 0 {
			executionResp.Data.Artifacts = make([]ToolArtifact, 0, len(result.Data.Artifacts))
			for _, artifact := range result.Data.Artifacts {
				executionResp.Data.Artifacts = append(executionResp.Data.Artifacts, ToolArtifact{
					Name:        artifact.Name,
					Type:        artifact.Type,
					Content:     artifact.Content,
					Path:        artifact.Path,
					ContentType: artifact.ContentType,
					Size:        artifact.Size,
				})
			}
		}
	}

	// Convert metadata if present
	if result.Metadata != nil {
		executionResp.Metadata = &ToolExecutionMetadata{
			StartTime:  result.Metadata.StartTime,
			EndTime:    result.Metadata.EndTime,
			CPUTime:    result.Metadata.CPUTime,
			MemoryUsed: result.Metadata.MemoryUsed,
			Custom:     result.Metadata.Custom,
			Warnings:   result.Metadata.Warnings,
			Debug:      result.Metadata.Debug,
		}
	}

	// Set result interface{} for compatibility
	if result.Data != nil {
		executionResp.Result = result.Data.Result
	}

	return executionResp, nil
}

// GetLangChainService returns the LangChain service instance if available
func (a *Assistant) GetLangChainService() *langchain.Service {
	return a.langchainService
}

// ProcessQueryStream processes a query and returns a streaming response
func (a *Assistant) ProcessQueryStream(ctx context.Context, query string) (*StreamingResponse, error) {
	request := &QueryRequest{
		Query: query,
	}
	return a.ProcessQueryStreamEnhanced(ctx, request)
}
