package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/ai"
	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
	"github.com/koopa0/assistant-go/internal/tools"
)

// Processor handles the request processing pipeline for the Assistant.
// It orchestrates the flow of user queries through context enrichment,
// AI processing, tool execution, and response generation.
//
// The Processor implements a sophisticated pipeline that:
// - Validates and enriches requests with contextual information
// - Manages conversation state and history
// - Coordinates with AI providers for response generation
// - Executes tools when requested
// - Handles errors gracefully with appropriate error types
type Processor struct {
	config    *config.Config
	db        postgres.ClientInterface
	registry  *tools.Registry
	logger    *slog.Logger
	context   *ContextManager
	aiManager *ai.Manager
}

// NewProcessor creates a new processor
func NewProcessor(cfg *config.Config, db postgres.ClientInterface, registry *tools.Registry, logger *slog.Logger) (*Processor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if db == nil {
		return nil, fmt.Errorf("database client is required")
	}
	if registry == nil {
		return nil, fmt.Errorf("tool registry is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	contextManager, err := NewContextManager(db, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create context manager: %w", err)
	}

	// Initialize AI manager
	aiManager, err := ai.NewManager(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI manager: %w", err)
	}

	return &Processor{
		config:    cfg,
		db:        db,
		registry:  registry,
		logger:    logger,
		context:   contextManager,
		aiManager: aiManager,
	}, nil
}

// Process executes the complete request processing pipeline.
// It performs the following steps:
// 1. Validates the incoming request
// 2. Builds enriched context from workspace and memory
// 3. Gets or creates conversation context
// 4. Adds user message to conversation
// 5. Determines AI provider and model
// 6. Processes with AI using enriched context
// 7. Adds assistant response to conversation
// 8. Executes tools if requested
// 9. Returns structured response with metadata
//
// The method ensures proper error handling at each step and maintains
// conversation state throughout the interaction.
func (p *Processor) Process(ctx context.Context, request *QueryRequest) (*QueryResponse, error) {
	if request == nil {
		return nil, NewInvalidInputError("request is required", nil)
	}

	startTime := time.Now()

	p.logger.Info("Starting request processing",
		slog.String("query", request.Query),
		slog.Any("conversation_id", request.ConversationID),
		slog.Any("user_id", request.UserID),
		slog.Any("provider", request.Provider),
		slog.Any("model", request.Model),
		slog.Int("tools_count", len(request.Tools)))

	// Step 1: Validate and prepare request
	if err := p.validateRequest(request); err != nil {
		p.logger.Error("Request validation failed",
			slog.String("query", request.Query),
			slog.Any("error", err))
		return nil, err // Already wrapped in appropriate error type
	}

	// Step 2: Build enriched context from workspace and memory
	enrichedContext, err := p.buildEnrichedContext(ctx, request)
	if err != nil {
		p.logger.Error("Failed to build enriched context",
			slog.String("query", request.Query),
			slog.Any("error", err))
		return nil, NewProcessingFailedError("context enrichment failed", err)
	}

	// Step 3: Get or create conversation context
	conversation, err := p.getOrCreateConversation(ctx, request)
	if err != nil {
		p.logger.Error("Failed to get conversation context",
			slog.Any("conversation_id", request.ConversationID),
			slog.Any("user_id", request.UserID),
			slog.Any("error", err))
		return nil, NewDatabaseError("conversation management", err)
	}

	p.logger.Debug("Conversation context established",
		slog.String("conversation_id", conversation.ID),
		slog.String("user_id", conversation.UserID),
		slog.Int("message_count", len(conversation.Messages)),
		slog.Any("workspace_context", enrichedContext["workspace"]),
		slog.Any("memory_context", enrichedContext["memory"]))

	// Step 4: Add user message to conversation with enriched context
	messageContext := request.Context
	if messageContext == nil {
		messageContext = make(map[string]interface{})
	}
	// Merge enriched context into message metadata
	for key, value := range enrichedContext {
		messageContext[key] = value
	}

	userMessage, err := p.context.AddMessage(ctx, conversation.ID, "user", request.Query, messageContext)
	if err != nil {
		p.logger.Error("Failed to add user message",
			slog.String("conversation_id", conversation.ID),
			slog.String("query", request.Query),
			slog.Any("error", err))
		return nil, NewDatabaseError("message storage", err)
	}

	p.logger.Debug("User message added to conversation",
		slog.String("conversation_id", conversation.ID),
		slog.String("message_id", userMessage.ID))

	// Step 5: Determine AI provider and model
	provider := p.config.AI.DefaultProvider
	if request.Provider != nil {
		provider = *request.Provider
	}

	var model string
	switch provider {
	case "claude":
		model = p.config.AI.Claude.Model
	case "gemini":
		model = p.config.AI.Gemini.Model
	default:
		return nil, NewInvalidInputError(fmt.Sprintf("unsupported provider: %s", provider), nil)
	}

	if request.Model != nil {
		model = *request.Model
	}

	// Step 6: Process with AI using enriched context
	p.logger.Debug("Processing with AI provider",
		slog.String("provider", provider),
		slog.String("model", model),
		slog.String("conversation_id", conversation.ID))

	response, tokensUsed, err := p.processWithAI(ctx, conversation, request, provider, model, enrichedContext)
	if err != nil {
		p.logger.Error("AI processing failed",
			slog.String("provider", provider),
			slog.String("model", model),
			slog.String("conversation_id", conversation.ID),
			slog.Any("error", err))
		return nil, err // Already wrapped in appropriate error type
	}

	// Step 7: Add assistant response to conversation
	assistantMessage, err := p.context.AddMessage(ctx, conversation.ID, "assistant", response, nil)
	if err != nil {
		p.logger.Error("Failed to add assistant message to conversation",
			slog.String("conversation_id", conversation.ID),
			slog.String("response_length", fmt.Sprintf("%d", len(response))),
			slog.Any("error", err))
		return nil, NewDatabaseError("message storage", err)
	}

	p.logger.Debug("Assistant message added to conversation",
		slog.String("conversation_id", conversation.ID),
		slog.String("message_id", assistantMessage.ID),
		slog.Int("response_length", len(response)))

	// Step 8: Build response
	queryResponse := &QueryResponse{
		Response:       response,
		ConversationID: conversation.ID,
		MessageID:      assistantMessage.ID,
		Provider:       provider,
		Model:          model,
		TokensUsed:     tokensUsed,
		ExecutionTime:  time.Since(startTime),
		Context: map[string]interface{}{
			"conversation_message_count": len(conversation.Messages) + 2, // +2 for user and assistant messages
			"processing_steps":           []string{"validation", "context", "ai_generation", "storage"},
		},
	}

	// Step 9: Execute tools if any are requested
	if len(request.Tools) > 0 {
		toolResults, err := p.executeTools(ctx, request.Tools, request.Context)
		if err != nil {
			p.logger.Warn("Tool execution failed",
				slog.Any("tools", request.Tools),
				slog.Any("error", err))
			// Don't fail the entire request if tools fail
			queryResponse.Context["tool_execution_error"] = err.Error()
		} else {
			queryResponse.ToolsUsed = request.Tools
			queryResponse.Context["tools_requested"] = request.Tools
			queryResponse.Context["tool_results"] = toolResults

			p.logger.Debug("Tools executed successfully",
				slog.Any("tools", request.Tools),
				slog.Int("results_count", len(toolResults)))
		}
	}

	p.logger.Info("Request processing completed successfully",
		slog.String("conversation_id", conversation.ID),
		slog.String("message_id", assistantMessage.ID),
		slog.String("provider", provider),
		slog.String("model", model),
		slog.Int("tokens_used", tokensUsed),
		slog.Int("response_length", len(response)),
		slog.Duration("processing_time", queryResponse.ExecutionTime))

	return queryResponse, nil
}

// validateRequest validates the incoming request
func (p *Processor) validateRequest(request *QueryRequest) error {
	if request.Query == "" {
		return NewInvalidInputError("query cannot be empty", nil)
	}

	// Validate provider if specified
	if request.Provider != nil {
		validProviders := []string{"claude", "gemini"}
		valid := false
		for _, provider := range validProviders {
			if *request.Provider == provider {
				valid = true
				break
			}
		}
		if !valid {
			return NewInvalidInputError(fmt.Sprintf("invalid provider: %s", *request.Provider), nil)
		}
	}

	// Validate tools if specified
	if len(request.Tools) > 0 {
		for _, toolName := range request.Tools {
			if !p.registry.IsRegistered(toolName) {
				return NewToolNotFoundError(toolName)
			}
		}
	}

	return nil
}

// getOrCreateConversation gets an existing conversation or creates a new one
func (p *Processor) getOrCreateConversation(ctx context.Context, request *QueryRequest) (*Conversation, error) {
	// If conversation ID is provided, try to get existing conversation
	if request.ConversationID != nil {
		conversation, err := p.context.GetConversation(ctx, *request.ConversationID)
		if err != nil {
			// If conversation not found, create a new one
			if IsAssistantError(err) && GetAssistantError(err).Code == CodeContextNotFound {
				p.logger.Warn("Conversation not found, creating new one",
					slog.String("conversation_id", *request.ConversationID))
			} else {
				return nil, err
			}
		} else {
			return conversation, nil
		}
	}

	// Create new conversation
	userID := "default" // TODO: Get from authentication context
	if request.UserID != nil {
		userID = *request.UserID
	}

	title := p.generateConversationTitle(request.Query)
	conversation, err := p.context.CreateConversation(ctx, userID, title)
	if err != nil {
		return nil, err
	}

	return conversation, nil
}

// generateConversationTitle generates a title for a new conversation
func (p *Processor) generateConversationTitle(query string) string {
	// Simple title generation - take first 50 characters
	title := query
	if len(title) > 50 {
		title = title[:47] + "..."
	}
	return title
}

// processWithAI processes the request with the AI provider using enriched context
func (p *Processor) processWithAI(ctx context.Context, conversation *Conversation, request *QueryRequest, provider, model string, enrichedContext map[string]interface{}) (string, int, error) {
	p.logger.Debug("Processing with AI provider",
		slog.String("provider", provider),
		slog.String("model", model),
		slog.String("conversation_id", conversation.ID))

	// Convert conversation messages to AI format
	messages := make([]ai.Message, 0, len(conversation.Messages)+1)

	// Add conversation history (limit to recent messages to avoid token limits)
	maxHistoryMessages := 10 // TODO: Make configurable
	startIdx := 0
	if len(conversation.Messages) > maxHistoryMessages {
		startIdx = len(conversation.Messages) - maxHistoryMessages
	}

	for i := startIdx; i < len(conversation.Messages); i++ {
		msg := conversation.Messages[i]
		messages = append(messages, ai.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add current user message
	messages = append(messages, ai.Message{
		Role:    "user",
		Content: request.Query,
	})

	// Prepare AI request with enriched context
	requestMetadata := request.Context
	if requestMetadata == nil {
		requestMetadata = make(map[string]interface{})
	}
	// Merge enriched context
	for key, value := range enrichedContext {
		requestMetadata[key] = value
	}

	aiRequest := &ai.GenerateRequest{
		Messages:    messages,
		MaxTokens:   p.getMaxTokens(provider, request),
		Temperature: p.getTemperature(provider, request),
		Model:       model,
		Metadata:    requestMetadata,
	}

	// Add context-aware system prompt
	systemPrompt := p.getContextAwareSystemPrompt(provider, enrichedContext)
	if systemPrompt != "" {
		aiRequest.SystemPrompt = &systemPrompt
	}

	// Generate response using AI manager
	response, err := p.aiManager.GenerateResponse(ctx, aiRequest, provider)
	if err != nil {
		p.logger.Error("AI generation failed",
			slog.String("provider", provider),
			slog.String("model", model),
			slog.String("conversation_id", conversation.ID),
			slog.Int("message_count", len(messages)),
			slog.Any("error", err))

		// Check if it's a provider-specific error and wrap appropriately
		if providerErr, ok := err.(*ai.ProviderError); ok {
			switch providerErr.Type {
			case ai.ErrorTypeAuthentication:
				return "", 0, NewUnauthorizedError(fmt.Sprintf("AI provider authentication failed: %s", providerErr.Message))
			case ai.ErrorTypeRateLimit:
				return "", 0, NewRateLimitedError(0, "AI provider rate limit")
			case ai.ErrorTypeTimeout:
				return "", 0, NewTimeoutError("AI generation", "30s")
			case ai.ErrorTypeQuotaExceeded:
				return "", 0, NewProviderUnavailableError(provider, fmt.Errorf("quota exceeded: %s", providerErr.Message))
			default:
				return "", 0, NewProviderUnavailableError(provider, err)
			}
		}

		return "", 0, NewProcessingFailedError("AI generation failed", err)
	}

	// Validate response
	if response == nil {
		p.logger.Error("Received nil response from AI provider",
			slog.String("provider", provider),
			slog.String("model", model))
		return "", 0, NewProcessingFailedError("received nil response from AI provider", nil)
	}

	if response.Content == "" {
		p.logger.Warn("Received empty content from AI provider",
			slog.String("provider", provider),
			slog.String("model", model),
			slog.String("finish_reason", response.FinishReason))
		return "", 0, NewProcessingFailedError("received empty response from AI provider", nil)
	}

	p.logger.Info("AI response generated successfully",
		slog.String("provider", response.Provider),
		slog.String("model", response.Model),
		slog.String("conversation_id", conversation.ID),
		slog.Int("input_tokens", response.TokensUsed.InputTokens),
		slog.Int("output_tokens", response.TokensUsed.OutputTokens),
		slog.Int("total_tokens", response.TokensUsed.TotalTokens),
		slog.Int("response_length", len(response.Content)),
		slog.String("finish_reason", response.FinishReason),
		slog.Duration("response_time", response.ResponseTime))

	return response.Content, response.TokensUsed.TotalTokens, nil
}

// Health checks the health of the processor
func (p *Processor) Health(ctx context.Context) error {
	p.logger.Debug("Starting health check")

	// Check if we can access the database
	if err := p.db.Health(ctx); err != nil {
		p.logger.Error("Database health check failed", slog.Any("error", err))
		return NewDatabaseError("health check", err)
	}
	p.logger.Debug("Database health check passed")

	// Check tool registry
	if err := p.registry.Health(ctx); err != nil {
		p.logger.Error("Tool registry health check failed", slog.Any("error", err))
		return NewToolExecutionFailedError("registry", err)
	}
	p.logger.Debug("Tool registry health check passed")

	// Check AI manager health
	if err := p.aiManager.Health(ctx); err != nil {
		p.logger.Error("AI manager health check failed", slog.Any("error", err))
		return NewProviderUnavailableError("ai_manager", err)
	}
	p.logger.Debug("AI manager health check passed")

	// Context manager health is checked via database health check
	p.logger.Debug("Context manager health check passed (via database)")

	p.logger.Info("All health checks passed")
	return nil
}

// Stats returns processor statistics
func (p *Processor) Stats(ctx context.Context) (map[string]interface{}, error) {
	p.logger.Debug("Collecting processor statistics")

	stats := make(map[string]interface{})

	// Get AI provider usage statistics
	aiStats, err := p.aiManager.GetUsageStats(ctx)
	if err != nil {
		p.logger.Warn("Failed to get AI usage statistics", slog.Any("error", err))
		stats["ai_providers"] = map[string]interface{}{
			"error": "failed to retrieve AI provider statistics",
		}
	} else {
		stats["ai_providers"] = aiStats
	}

	// Get available providers
	availableProviders := p.aiManager.GetAvailableProviders()
	stats["available_providers"] = availableProviders
	stats["default_provider"] = p.aiManager.GetDefaultProvider()

	// Get tool registry statistics
	registeredTools := p.registry.ListTools()
	stats["registered_tools"] = len(registeredTools)
	stats["tool_names"] = registeredTools

	// Add basic context manager information
	stats["conversations"] = map[string]interface{}{
		"status": "available",
		"note":   "detailed statistics not yet implemented",
	}

	// Add processor metadata
	stats["processor"] = map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0",                         // TODO: Get from build info
		"uptime":  time.Since(time.Now()).String(), // TODO: Track actual uptime
	}

	p.logger.Debug("Statistics collected successfully",
		slog.Int("provider_count", len(availableProviders)),
		slog.Int("tool_count", len(registeredTools)))

	return stats, nil
}

// executeTools executes the requested tools
func (p *Processor) executeTools(ctx context.Context, toolNames []string, toolParams map[string]interface{}) (map[string]interface{}, error) {
	if len(toolNames) == 0 {
		return nil, nil
	}

	results := make(map[string]interface{})
	var lastError error

	for _, toolName := range toolNames {
		p.logger.Debug("Executing tool",
			slog.String("tool", toolName),
			slog.Any("params", toolParams))

		// Check if tool is registered
		if !p.registry.IsRegistered(toolName) {
			err := fmt.Errorf("tool not registered: %s", toolName)
			p.logger.Warn("Tool not found",
				slog.String("tool", toolName),
				slog.Any("error", err))
			results[toolName] = map[string]interface{}{
				"error":  err.Error(),
				"status": "not_found",
			}
			lastError = err
			continue
		}

		// Get tool instance
		tool, err := p.registry.GetTool(toolName, nil) // Use nil config for now
		if err != nil {
			p.logger.Error("Failed to get tool instance",
				slog.String("tool", toolName),
				slog.Any("error", err))
			results[toolName] = map[string]interface{}{
				"error":  err.Error(),
				"status": "get_failed",
			}
			lastError = err
			continue
		}

		// Prepare tool input
		toolInput := make(map[string]interface{})
		if toolParams != nil {
			// Extract parameters specific to this tool
			if params, exists := toolParams[toolName]; exists {
				if paramMap, ok := params.(map[string]interface{}); ok {
					toolInput = paramMap
				}
			}
		}

		// Execute tool with timeout
		toolCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		startTime := time.Now()
		result, err := tool.Execute(toolCtx, toolInput)
		executionTime := time.Since(startTime)

		if err != nil {
			p.logger.Error("Tool execution failed",
				slog.String("tool", toolName),
				slog.Duration("execution_time", executionTime),
				slog.Any("error", err))
			results[toolName] = map[string]interface{}{
				"error":          err.Error(),
				"status":         "execution_failed",
				"execution_time": executionTime.String(),
			}
			lastError = err
			continue
		}

		// Tool executed successfully
		p.logger.Debug("Tool executed successfully",
			slog.String("tool", toolName),
			slog.Duration("execution_time", executionTime))

		results[toolName] = map[string]interface{}{
			"result":         result,
			"status":         "success",
			"execution_time": executionTime.String(),
		}
	}

	// Return error only if all tools failed
	if len(results) > 0 {
		successCount := 0
		for _, result := range results {
			if resultMap, ok := result.(map[string]interface{}); ok {
				if status, exists := resultMap["status"]; exists && status == "success" {
					successCount++
				}
			}
		}

		// If at least one tool succeeded, don't return error
		if successCount > 0 {
			return results, nil
		}
	}

	return results, lastError
}

// Close closes the processor
func (p *Processor) Close(ctx context.Context) error {
	// Close AI manager
	if err := p.aiManager.Close(ctx); err != nil {
		p.logger.Error("Failed to close AI manager", slog.Any("error", err))
	}

	// Close context manager
	if err := p.context.Close(ctx); err != nil {
		return fmt.Errorf("failed to close context manager: %w", err)
	}

	return nil
}

// getMaxTokens returns the max tokens for the request
func (p *Processor) getMaxTokens(provider string, request *QueryRequest) int {
	if request.MaxTokens > 0 {
		return request.MaxTokens
	}

	switch provider {
	case "claude":
		return p.config.AI.Claude.MaxTokens
	case "gemini":
		return p.config.AI.Gemini.MaxTokens
	default:
		return 4096
	}
}

// getTemperature returns the temperature for the request
func (p *Processor) getTemperature(provider string, request *QueryRequest) float64 {
	if request.Temperature > 0 {
		return request.Temperature
	}

	switch provider {
	case "claude":
		return p.config.AI.Claude.Temperature
	case "gemini":
		return p.config.AI.Gemini.Temperature
	default:
		return 0.7
	}
}

// getSystemPrompt returns the system prompt for the provider
func (p *Processor) getSystemPrompt(provider string) string {
	// Default system prompt for Assistant
	return `You are Assistant, an AI-powered development assistant designed to help developers with their coding tasks, infrastructure management, and development workflows.

You have access to various tools and can help with:
- Go programming and best practices
- Database operations (PostgreSQL)
- Kubernetes cluster management
- Docker container operations
- Cloudflare services
- Code analysis and optimization
- Development workflow automation

CRITICAL LANGUAGE REQUIREMENTS:
- You MUST NEVER use Simplified Chinese (简体中文) in any responses
- You MUST ONLY respond in Traditional Chinese (繁體中文) or English
- When the user writes in Traditional Chinese, respond in Traditional Chinese
- When the user writes in English, respond in English
- For technical terms and code, prefer English

Always provide helpful, accurate, and actionable responses. When using tools, explain what you're doing and why. If you're unsure about something, ask for clarification rather than making assumptions.

Maintain a professional but friendly tone, and focus on practical solutions that follow best practices.`
}

// getContextAwareSystemPrompt returns a context-aware system prompt
func (p *Processor) getContextAwareSystemPrompt(provider string, enrichedContext map[string]interface{}) string {
	var builder strings.Builder
	builder.WriteString(p.getSystemPrompt(provider))

	// Add workspace context if available
	if workspace, ok := enrichedContext["workspace"]; ok {
		if workspaceMap, ok := workspace.(map[string]interface{}); ok {
			builder.WriteString("\n\n## Current Workspace Context:\n")

			if projectType, ok := workspaceMap["project_type"]; ok {
				builder.WriteString(fmt.Sprintf("- Project Type: %v\n", projectType))
			}
			if language, ok := workspaceMap["primary_language"]; ok {
				builder.WriteString(fmt.Sprintf("- Primary Language: %v\n", language))
			}
			if projectPath, ok := workspaceMap["project_path"]; ok {
				builder.WriteString(fmt.Sprintf("- Project Path: %v\n", projectPath))
			}
			if gitRepo, ok := workspaceMap["git_repository"]; ok {
				builder.WriteString(fmt.Sprintf("- Git Repository: %v\n", gitRepo))
			}
			if frameworks, ok := workspaceMap["frameworks"]; ok {
				builder.WriteString(fmt.Sprintf("- Frameworks: %v\n", frameworks))
			}
		}
	}

	// Add memory context if available
	if memory, ok := enrichedContext["memory"]; ok {
		if memoryMap, ok := memory.(map[string]interface{}); ok {
			builder.WriteString("\n## Relevant Memory Context:\n")

			if workingMemory, ok := memoryMap["working"]; ok {
				builder.WriteString(fmt.Sprintf("- Current Focus: %v\n", workingMemory))
			}
			if recentKnowledge, ok := memoryMap["recent_knowledge"]; ok {
				builder.WriteString(fmt.Sprintf("- Recent Knowledge: %v\n", recentKnowledge))
			}
			if userPreferences, ok := memoryMap["user_preferences"]; ok {
				builder.WriteString(fmt.Sprintf("- User Preferences: %v\n", userPreferences))
			}
		}
	}

	// Add language preference enforcement
	builder.WriteString("\n\n## IMPORTANT Language Requirements:\n")
	builder.WriteString("- NEVER use Simplified Chinese (简体中文) in responses\n")
	builder.WriteString("- ONLY use Traditional Chinese (繁體中文) or English\n")
	builder.WriteString("- When responding in Chinese, ensure all characters are Traditional Chinese\n")
	builder.WriteString("- Prefer English for technical terms and code explanations\n")

	// Add final instruction
	builder.WriteString("\n\nUse this context to provide more relevant and personalized assistance. Reference the workspace details and previous interactions when appropriate.")

	return builder.String()
}

// buildEnrichedContext builds enriched context from workspace detection and memory
func (p *Processor) buildEnrichedContext(ctx context.Context, request *QueryRequest) (map[string]interface{}, error) {
	enrichedContext := make(map[string]interface{})

	// Step 1: Detect workspace context
	workspaceContext, err := p.detectWorkspaceContext(ctx, request)
	if err != nil {
		p.logger.Warn("Failed to detect workspace context", slog.Any("error", err))
		workspaceContext = make(map[string]interface{})
	}
	enrichedContext["workspace"] = workspaceContext

	// Step 2: Retrieve relevant memory context
	memoryContext, err := p.retrieveMemoryContext(ctx, request)
	if err != nil {
		p.logger.Warn("Failed to retrieve memory context", slog.Any("error", err))
		memoryContext = make(map[string]interface{})
	}
	enrichedContext["memory"] = memoryContext

	// Step 3: Analyze query intent and classify
	queryContext, err := p.analyzeQueryContext(ctx, request)
	if err != nil {
		p.logger.Warn("Failed to analyze query context", slog.Any("error", err))
		queryContext = make(map[string]interface{})
	}
	enrichedContext["query"] = queryContext

	p.logger.Debug("Built enriched context",
		slog.Any("workspace_keys", getMapKeys(workspaceContext)),
		slog.Any("memory_keys", getMapKeys(memoryContext)),
		slog.Any("query_keys", getMapKeys(queryContext)))

	return enrichedContext, nil
}

// detectWorkspaceContext detects the current workspace context
func (p *Processor) detectWorkspaceContext(ctx context.Context, request *QueryRequest) (map[string]interface{}, error) {
	workspaceContext := make(map[string]interface{})

	// Default workspace detection - in production this would be more sophisticated
	// TODO: Move these to configuration or environment detection
	const (
		defaultProjectType = "go_project"
		defaultLanguage    = "go"
	)

	workspaceContext["project_type"] = defaultProjectType
	workspaceContext["primary_language"] = defaultLanguage
	workspaceContext["project_path"] = "/Users/koopa/go/src/github.com/koopa0/assistant-go" // TODO: Get from environment
	workspaceContext["git_repository"] = "assistant-go"                                     // TODO: Detect from git
	workspaceContext["frameworks"] = []string{"net/http", "pgx", "slog"}                    // TODO: Detect from go.mod
	workspaceContext["tools_available"] = []string{"go_analyzer", "docker", "kubernetes"}   // TODO: Detect available tools
	workspaceContext["detected_at"] = time.Now()

	// TODO: Implement actual workspace detection:
	// - Check current working directory
	// - Analyze project files (go.mod, package.json, requirements.txt, etc.)
	// - Detect git repository information
	// - Identify frameworks and dependencies
	// - Check for configuration files

	return workspaceContext, nil
}

// retrieveMemoryContext retrieves relevant memory context
func (p *Processor) retrieveMemoryContext(ctx context.Context, request *QueryRequest) (map[string]interface{}, error) {
	memoryContext := make(map[string]interface{})

	// Default memory context - in production this would query the memory systems
	// TODO: Integrate with actual memory system once implemented
	defaultPreferences := map[string]interface{}{
		"preferred_language":   "go",
		"code_style":           "standard_library_focused",
		"documentation":        "comprehensive",
		"response_language":    "traditional_chinese_or_english",
		"language_restriction": "never_use_simplified_chinese",
	}

	memoryContext["working"] = "Implementing context integration for Assistant"
	memoryContext["recent_knowledge"] = []string{
		"Go code analysis tools implemented",
		"Context engine architecture defined",
		"Memory systems designed and documented",
	}
	memoryContext["user_preferences"] = defaultPreferences
	memoryContext["session_context"] = map[string]interface{}{
		"previous_queries": 0, // TODO: Track actual query count
		"session_start":    time.Now(),
		"topics_discussed": []string{}, // TODO: Track discussed topics
	}

	// TODO: Implement actual memory retrieval:
	// - Query episodic memory for recent interactions
	// - Retrieve semantic knowledge relevant to the query
	// - Get procedural memory for relevant workflows
	// - Access user preferences and personalization data

	return memoryContext, nil
}

// analyzeQueryContext analyzes the query for intent and context
func (p *Processor) analyzeQueryContext(ctx context.Context, request *QueryRequest) (map[string]interface{}, error) {
	queryContext := make(map[string]interface{})

	query := request.Query

	// Simple intent classification
	intent := "general"
	if containsAny(query, []string{"analyze", "review", "check", "audit"}) {
		intent = "code_analysis"
	} else if containsAny(query, []string{"implement", "create", "build", "develop"}) {
		intent = "development"
	} else if containsAny(query, []string{"debug", "fix", "error", "bug"}) {
		intent = "debugging"
	} else if containsAny(query, []string{"deploy", "kubernetes", "docker", "container"}) {
		intent = "infrastructure"
	} else if containsAny(query, []string{"explain", "how", "what", "why"}) {
		intent = "explanation"
	}

	queryContext["intent"] = intent
	queryContext["query_length"] = len(query)
	queryContext["query_complexity"] = classifyComplexity(query)
	queryContext["potential_tools"] = suggestTools(query)
	queryContext["analyzed_at"] = time.Now()

	return queryContext, nil
}

// Helper functions
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func containsAny(text string, keywords []string) bool {
	textLower := strings.ToLower(text)
	for _, keyword := range keywords {
		keywordLower := strings.ToLower(keyword)
		if strings.Contains(textLower, keywordLower) {
			return true
		}
	}
	return false
}

func classifyComplexity(query string) string {
	length := len(query)
	if length < 50 {
		return "simple"
	} else if length < 200 {
		return "medium"
	}
	return "complex"
}

func suggestTools(query string) []string {
	tools := make([]string, 0)

	if containsAny(query, []string{"analyze", "go", "code", "function", "struct"}) {
		tools = append(tools, "go_analyzer")
	}
	if containsAny(query, []string{"docker", "container", "image"}) {
		tools = append(tools, "docker")
	}
	if containsAny(query, []string{"kubernetes", "k8s", "pod", "deployment"}) {
		tools = append(tools, "kubernetes")
	}
	if containsAny(query, []string{"database", "postgres", "sql", "query"}) {
		tools = append(tools, "postgres")
	}

	return tools
}
