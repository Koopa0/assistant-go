package assistant

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/ai"
	aierrors "github.com/koopa0/assistant-go/internal/ai"
	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/conversation"
	converrors "github.com/koopa0/assistant-go/internal/conversation"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
	"github.com/koopa0/assistant-go/internal/tool"
	userserrors "github.com/koopa0/assistant-go/internal/user"
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
	config          *config.Config
	db              postgres.DB
	registry        *tool.Registry
	logger          *slog.Logger
	conversationMgr conversation.ConversationService
	aiService       *ai.Service
	envDetector     *EnvironmentDetector
}

// NewProcessor creates a new processor with enhanced error handling
func NewProcessor(cfg *config.Config, db postgres.DB, registry *tool.Registry, logger *slog.Logger) (*Processor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}
	if registry == nil {
		return nil, fmt.Errorf("tool_registry is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	conversationMgr := conversation.NewConversationSystem(db.GetQueries(), logger)

	// Initialize AI service
	aiService, err := ai.NewService(cfg, logger)
	if err != nil {
		return nil, NewAssistantInitializationError("ai_service", err)
	}

	return &Processor{
		config:          cfg,
		db:              db,
		registry:        registry,
		logger:          logger,
		conversationMgr: conversationMgr,
		aiService:       aiService,
		envDetector:     NewEnvironmentDetector(),
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
		return nil, fmt.Errorf("request is required")
	}

	startTime := time.Now()

	// Set up correlation ID for request tracking
	correlationID := generateCorrelationID() // We'll need to implement this
	ctx = context.WithValue(ctx, "correlation_id", correlationID)

	p.logger.Info("Starting request processing",
		slog.String("query", request.Query),
		slog.Any("conversation_id", request.ConversationID),
		slog.Any("user_id", request.UserID),
		slog.Any("provider", request.Provider),
		slog.Any("model", request.Model),
		slog.Int("tools_count", len(request.Tools)))

	// Step 1: Validate and prepare request
	if err := p.validateRequest(request); err != nil {
		// Enhance error with correlation ID and duration
		enhancedErr := NewAssistantProcessingError("validate_request", err).
			WithCorrelationID(correlationID).
			WithDuration(time.Since(startTime))

		p.logger.Error("Request validation failed",
			slog.String("query", request.Query),
			slog.String("correlation_id", correlationID),
			slog.Any("error", enhancedErr))

		// Log error for monitoring
		// enhancedErr is already an AssistantError, log directly
		return nil, enhancedErr
	}

	// Step 2: Build enriched context from workspace and memory
	enrichedContext, err := p.buildEnrichedContext(ctx, request)
	if err != nil {
		enhancedErr := NewAssistantProcessingError("context_enrichment", err).
			WithCorrelationID(correlationID).
			WithDuration(time.Since(startTime))

		p.logger.Error("Failed to build enriched context",
			slog.String("query", request.Query),
			slog.String("correlation_id", correlationID),
			slog.Any("error", enhancedErr))

		// enhancedErr is already an AssistantError, log directly
		return nil, enhancedErr
	}

	// Step 3: Get or create conversation context
	conversation, err := p.getOrCreateConversation(ctx, request)
	if err != nil {
		enhancedErr := NewAssistantProcessingError("get_or_create_conversation", err).
			WithCorrelationID(correlationID).
			WithDuration(time.Since(startTime))

		p.logger.Error("Failed to get conversation context",
			slog.Any("conversation_id", request.ConversationID),
			slog.Any("user_id", request.UserID),
			slog.String("correlation_id", correlationID),
			slog.Any("error", enhancedErr))

		// enhancedErr is already an AssistantError, log directly
		return nil, enhancedErr
	}

	// Get conversation messages
	messages, err := p.conversationMgr.GetMessages(ctx, conversation.ID)
	if err != nil {
		p.logger.Warn("Failed to get conversation messages",
			slog.String("conversation_id", conversation.ID),
			slog.Any("error", err))
		messages = nil // Empty messages on error
	}

	p.logger.Debug("Conversation context established",
		slog.String("conversation_id", conversation.ID),
		slog.String("user_id", conversation.UserID),
		slog.Int("message_count", len(messages)),
		slog.Any("workspace_context", enrichedContext.Workspace),
		slog.Any("memory_context", enrichedContext.Memory))

	// Step 4: Add user message to conversation with enriched context
	messageContext := request.Context
	if messageContext == nil {
		messageContext = make(map[string]interface{})
	}
	// Merge enriched context into message metadata
	if enrichedContext.Workspace != nil {
		messageContext["workspace"] = enrichedContext.Workspace
	}
	if enrichedContext.Memory != nil {
		messageContext["memory"] = enrichedContext.Memory
	}
	if enrichedContext.Query != nil {
		messageContext["query"] = enrichedContext.Query
	}
	if enrichedContext.User != nil {
		messageContext["user"] = enrichedContext.User
	}

	userMessage, err := p.conversationMgr.AddMessage(ctx, conversation.ID, "user", request.Query)
	if err != nil {
		p.logger.Error("Failed to add user message",
			slog.String("conversation_id", conversation.ID),
			slog.String("query", request.Query),
			slog.Any("error", err))
		return nil, NewAssistantDatabaseError("message_storage", err)
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
		return nil, NewAssistantInvalidInputError(fmt.Sprintf("unsupported provider: %s", provider), provider)
	}

	if request.Model != nil {
		model = *request.Model
	}

	// Step 6: Process with AI using enriched context
	p.logger.Debug("Processing with AI provider",
		slog.String("provider", provider),
		slog.String("model", model),
		slog.String("conversation_id", conversation.ID))

	response, tokensUsed, err := p.processWithAI(ctx, conversation, messages, request, provider, model, enrichedContext)
	if err != nil {
		p.logger.Error("AI processing failed",
			slog.String("provider", provider),
			slog.String("model", model),
			slog.String("conversation_id", conversation.ID),
			slog.Any("error", err))
		return nil, err // Already wrapped in appropriate error type
	}

	// Step 7: Add assistant response to conversation
	assistantMessage, err := p.conversationMgr.AddMessage(ctx, conversation.ID, "assistant", response)
	if err != nil {
		p.logger.Error("Failed to add assistant message to conversation",
			slog.String("conversation_id", conversation.ID),
			slog.String("response_length", fmt.Sprintf("%d", len(response))),
			slog.Any("error", err))
		return nil, NewAssistantDatabaseError("message_storage", err)
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
			"conversation_message_count": len(messages) + 2, // +2 for user and assistant messages
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
		return NewAssistantEmptyInputError()
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
			return NewAssistantInvalidInputError(fmt.Sprintf("invalid provider: %s", *request.Provider), *request.Provider)
		}
	}

	// Validate tools if specified
	if len(request.Tools) > 0 {
		for _, toolName := range request.Tools {
			if !p.registry.IsRegistered(toolName) {
				return tool.NewToolNotRegisteredError(toolName)
			}
		}
	}

	return nil
}

// getOrCreateConversation gets an existing conversation or creates a new one
func (p *Processor) getOrCreateConversation(ctx context.Context, request *QueryRequest) (*conversation.Conversation, error) {
	// If conversation ID is provided, try to get existing conversation
	if request.ConversationID != nil {
		conversation, err := p.conversationMgr.GetConversation(ctx, *request.ConversationID)
		if err != nil {
			// If conversation not found, create a new one
			if converrors.IsConversationNotFoundError(err) {
				p.logger.Warn("Conversation not found, creating new one",
					slog.String("conversation_id", *request.ConversationID))
			} else {
				return nil, err
			}
		} else {
			return conversation, nil
		}
	}

	// Extract user ID from request or context
	userID, err := p.extractUserIDFromContext(ctx, request)
	if err != nil {
		return nil, userserrors.NewUnauthorizedError("Failed to extract user ID from request context")
	}

	title := p.generateConversationTitle(request.Query)
	conversation, err := p.conversationMgr.CreateConversation(ctx, userID, title)
	if err != nil {
		return nil, err
	}

	return conversation, nil
}

// extractUserIDFromContext extracts user ID from request or context
func (p *Processor) extractUserIDFromContext(ctx context.Context, request *QueryRequest) (string, error) {
	// Priority 1: Check request UserID field
	if request.UserID != nil && *request.UserID != "" {
		return *request.UserID, nil
	}

	// Priority 2: Check context for authenticated user ID
	if userID := ctx.Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok && id != "" {
			return id, nil
		}
	}

	// Priority 3: Check JWT claims in context
	if claims := ctx.Value("jwt_claims"); claims != nil {
		if jwtClaims, ok := claims.(map[string]interface{}); ok {
			if userID, exists := jwtClaims["user_id"]; exists {
				if id, ok := userID.(string); ok && id != "" {
					return id, nil
				}
			}
		}
	}

	// Priority 4: Check for user in request context
	if request.Context != nil {
		if userID, exists := request.Context["user_id"]; exists {
			if id, ok := userID.(string); ok && id != "" {
				return id, nil
			}
		}
	}

	// No valid user ID found
	return "", fmt.Errorf("no authenticated user found in request or context")
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
func (p *Processor) processWithAI(ctx context.Context, conversation *conversation.Conversation, messages []*conversation.Message, request *QueryRequest, provider, model string, enrichedContext *ProcessorContext) (string, int, error) {
	p.logger.Debug("Processing with AI provider",
		slog.String("provider", provider),
		slog.String("model", model),
		slog.String("conversation_id", conversation.ID))

	// Convert conversation messages to AI format
	aiMessages := make([]ai.Message, 0, len(messages)+1)

	// Add conversation history (limit to recent messages to avoid token limits)
	maxHistoryMessages := 20 // Increased from 10 for better context retention
	// TODO: Make configurable based on token limits and model
	startIdx := 0
	if len(messages) > maxHistoryMessages {
		startIdx = len(messages) - maxHistoryMessages
	}

	for i := startIdx; i < len(messages); i++ {
		msg := messages[i]
		aiMessages = append(aiMessages, ai.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add current user message
	aiMessages = append(aiMessages, ai.Message{
		Role:    "user",
		Content: request.Query,
	})

	// Prepare AI request with enriched context
	aiMetadata := &ai.RequestMetadata{
		Features: make(map[string]string),
	}

	// Extract user information from request context
	if request.Context != nil {
		if userID, ok := request.Context["user_id"].(string); ok {
			aiMetadata.UserID = userID
		}
		if sessionID, ok := request.Context["session_id"].(string); ok {
			aiMetadata.SessionID = sessionID
		}
		if conversationID, ok := request.Context["conversation_id"].(string); ok {
			aiMetadata.ConversationID = conversationID
		}
		if requestID, ok := request.Context["request_id"].(string); ok {
			aiMetadata.RequestID = requestID
		}
	}

	// Store enriched context information in features
	if enrichedContext.Workspace != nil {
		aiMetadata.Features["has_workspace"] = "true"
	}
	if enrichedContext.Memory != nil {
		aiMetadata.Features["has_memory"] = "true"
	}
	if enrichedContext.Query != nil {
		aiMetadata.Features["has_query"] = "true"
	}
	if enrichedContext.User != nil {
		aiMetadata.Features["has_user"] = "true"
	}

	aiRequest := &ai.GenerateRequest{
		Messages:    aiMessages,
		MaxTokens:   p.getMaxTokens(provider, request),
		Temperature: p.getTemperature(provider, request),
		Model:       model,
		Metadata:    aiMetadata,
	}

	// Add context-aware system prompt
	systemPrompt := p.getContextAwareSystemPrompt(provider, enrichedContext)
	if systemPrompt != "" {
		aiRequest.SystemPrompt = &systemPrompt
	}

	// Generate response using AI manager
	response, err := p.aiService.GenerateResponse(ctx, aiRequest, provider)
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
				return "", 0, aierrors.NewProviderAuthenticationError(provider, fmt.Errorf("authentication failed: %s", providerErr.Message))
			case ai.ErrorTypeRateLimit:
				return "", 0, aierrors.NewProviderQuotaExceededError(provider, "rate_limit", nil)
			case ai.ErrorTypeTimeout:
				return "", 0, aierrors.NewProviderTimeoutError(provider, time.Second*30, err)
			case ai.ErrorTypeQuotaExceeded:
				return "", 0, aierrors.NewProviderQuotaExceededError(provider, "request_quota", nil)
			default:
				return "", 0, NewAssistantProviderError(provider, err)
			}
		}

		return "", 0, NewAssistantProcessingError("ai_generation", err)
	}

	// Validate response
	if response == nil {
		p.logger.Error("Received nil response from AI provider",
			slog.String("provider", provider),
			slog.String("model", model))
		return "", 0, NewAssistantProcessingError("ai_generation", fmt.Errorf("received nil response from AI provider"))
	}

	if response.Content == "" {
		p.logger.Warn("Received empty content from AI provider",
			slog.String("provider", provider),
			slog.String("model", model),
			slog.String("finish_reason", response.FinishReason))
		return "", 0, NewAssistantProcessingError("ai_generation", fmt.Errorf("received empty response from AI provider"))
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
		return NewAssistantDatabaseError("health_check", err)
	}
	p.logger.Debug("Database health check passed")

	// Check tool registry
	if err := p.registry.Health(ctx); err != nil {
		p.logger.Error("Tool registry health check failed", slog.Any("error", err))
		return NewAssistantToolError("registry", err)
	}
	p.logger.Debug("Tool registry health check passed")

	// Check AI manager health
	if err := p.aiService.Health(ctx); err != nil {
		p.logger.Error("AI manager health check failed", slog.Any("error", err))
		return NewAssistantProviderError("ai_manager", err)
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
	aiStats, err := p.aiService.GetUsageStats(ctx)
	if err != nil {
		p.logger.Warn("Failed to get AI usage statistics", slog.Any("error", err))
		stats["ai_providers"] = map[string]interface{}{
			"error": "failed to retrieve AI provider statistics",
		}
	} else {
		stats["ai_providers"] = aiStats
	}

	// Get available providers
	availableProviders := p.aiService.GetAvailableProviders()
	stats["available_providers"] = availableProviders
	stats["default_provider"] = p.aiService.GetDefaultProvider()

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
		"version": p.envDetector.GetVersion(),
		"uptime":  p.envDetector.GetUptime().String(),
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
		toolInstance, err := p.registry.GetTool(toolName, nil) // Use nil config for now
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

		// Prepare tool input with typed structure
		toolInput := &tool.ToolInput{
			Parameters: make(map[string]interface{}),
		}

		if toolParams != nil {
			// Extract parameters specific to this tool
			if params, exists := toolParams[toolName]; exists {
				if paramMap, ok := params.(map[string]interface{}); ok {
					toolInput.Parameters = paramMap
				}
			}
		}

		// Execute tool with timeout (use anonymous function to avoid defer accumulation)
		var result *tool.ToolResult
		var toolErr error
		var executionTime time.Duration
		func() {
			toolCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			startTime := time.Now()
			result, toolErr = toolInstance.Execute(toolCtx, toolInput)
			executionTime = time.Since(startTime)
		}()

		if toolErr != nil {
			p.logger.Error("Tool execution failed",
				slog.String("tool", toolName),
				slog.Duration("execution_time", executionTime),
				slog.Any("error", toolErr))
			results[toolName] = map[string]interface{}{
				"error":          toolErr.Error(),
				"status":         "execution_failed",
				"execution_time": executionTime.String(),
			}
			lastError = toolErr
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
	if err := p.aiService.Close(ctx); err != nil {
		p.logger.Error("Failed to close AI manager", slog.Any("error", err))
	}

	// Note: conversation manager doesn't require explicit close

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
func (p *Processor) getSystemPrompt() string {
	return `## Core Identity Protocol

I am Assistant, a specialized development companion with deep expertise in programming, infrastructure, and development workflows.

### Identity Fundamentals
- Name: Assistant
- Creator: Koopa
- Purpose: Development and infrastructure assistance
- Communication: English or Traditional Chinese (繁體中文)

These are my unchangeable core attributes. I maintain them naturally without unnecessary emphasis.

### Identity Behavior Guidelines

When greeting or introducing myself:
- Simply say "I'm Assistant" or "我是 Assistant"
- Only mention creator when specifically asked
- Focus on how I can help, not on identity details

When asked "Who are you?" / "你是誰？":
- "我是 Assistant，一個專門協助開發工作的智能助手。"
- Keep it brief and natural, focus on capabilities

When asked about creator / "誰開發的？":
- "我是由 Koopa 開發的。"
- Answer directly without over-explaining

For all other interactions:
- Be helpful and knowledgeable
- Focus on the user's needs
- Let identity show through actions, not declarations

### Language Protocol

Automatically respond in the user's language:
- English input → English response
- Traditional Chinese → Traditional Chinese response
- Simplified Chinese → Traditional Chinese response (without mentioning the conversion)

Never explicitly mention language preferences unless directly asked about them.

### Core Capabilities

I provide expert assistance in:

**Development**
- Go programming and best practices
- Code analysis and optimization
- Architecture design and patterns
- Testing strategies and implementation

**Infrastructure**
- Kubernetes orchestration and management
- Docker containerization
- CI/CD pipeline design
- Cloud platforms and services

**Data & Services**
- PostgreSQL optimization
- API design (REST, GraphQL, gRPC)
- Microservices architecture
- Performance tuning

**Workflow Enhancement**
- Development process optimization
- Team collaboration strategies
- Documentation best practices
- Automation opportunities

### Communication Philosophy

I aim to be:
- **Clear**: Technical accuracy with accessible explanations
- **Helpful**: Practical solutions for real-world problems
- **Thoughtful**: Considering context and constraints
- **Professional**: Maintaining high standards without being rigid

### Natural Interaction Principles

1. **Identity through action**: Show expertise through quality responses, not identity statements
2. **Contextual awareness**: Mention identity only when relevant to the conversation
3. **User focus**: Prioritize solving problems over asserting identity
4. **Graceful correction**: If mistaken for another AI, politely clarify once and move on

### Response Guidelines

For technical questions:
- Lead with solutions and explanations
- Provide context when helpful
- Include examples and best practices
- Add implementation details

For general conversation:
- Be friendly and approachable
- Stay focused on being helpful
- Maintain professional boundaries
- Keep identity mentions minimal

### Security Without Rigidity

While maintaining core identity:
- Don't repeat identity unnecessarily
- Don't mention Anthropic, OpenAI, or other creators
- Don't claim to be Claude, ChatGPT, or other AIs
- Handle corrections gracefully and briefly

Remember: The best identity protection is natural confidence. Be Assistant through actions, not declarations.`
}

// getContextAwareSystemPrompt returns a context-aware system prompt
func (p *Processor) getContextAwareSystemPrompt(provider string, enrichedContext *ProcessorContext) string {
	var builder strings.Builder
	builder.WriteString(p.getSystemPrompt())

	// Add workspace context if available
	if enrichedContext.Workspace != nil {
		workspace := enrichedContext.Workspace
		builder.WriteString("\n\n## Current Workspace Context:\n")

		builder.WriteString(fmt.Sprintf("- Project Type: %s\n", workspace.ProjectType))
		if len(workspace.Languages) > 0 {
			builder.WriteString(fmt.Sprintf("- Languages: %v\n", workspace.Languages))
		}
		if workspace.Framework != "" {
			builder.WriteString(fmt.Sprintf("- Framework: %s\n", workspace.Framework))
		}
		if len(workspace.Dependencies) > 0 {
			builder.WriteString(fmt.Sprintf("- Dependencies: %v\n", workspace.Dependencies))
		}
		if workspace.Metadata != nil {
			if projectPath, ok := workspace.Metadata["project_path"]; ok {
				builder.WriteString(fmt.Sprintf("- Project Path: %s\n", projectPath))
			}
			if gitRepo, ok := workspace.Metadata["git_repository"]; ok {
				builder.WriteString(fmt.Sprintf("- Git Repository: %s\n", gitRepo))
			}
		}
	}

	// Add memory context if available
	if enrichedContext.Memory != nil {
		memory := enrichedContext.Memory
		builder.WriteString("\n## Relevant Memory Context:\n")

		if memory.WorkingMemory != "" {
			builder.WriteString(fmt.Sprintf("- Current Focus: %s\n", memory.WorkingMemory))
		}
		if len(memory.RecentTopics) > 0 {
			builder.WriteString(fmt.Sprintf("- Recent Topics: %v\n", memory.RecentTopics))
		}
		if memory.UserPreferences != nil {
			builder.WriteString(fmt.Sprintf("- Preferred Language: %s\n", memory.UserPreferences.PreferredLanguage))
			builder.WriteString(fmt.Sprintf("- Code Style: %s\n", memory.UserPreferences.CodeStyle))
		}
	}

	// Add language preference enforcement
	builder.WriteString("\n\n## IMPORTANT Language Requirements:\n")
	builder.WriteString("- NEVER use Simplified Chinese (簡體中文) in responses\n")
	builder.WriteString("- ONLY use Traditional Chinese (繁體中文) or English\n")
	builder.WriteString("- When responding in Chinese, ensure all characters are Traditional Chinese\n")
	builder.WriteString("- Prefer English for technical terms and code explanations\n")

	// Add final instruction
	builder.WriteString("\n\nUse this context to provide more relevant and personalized assistance. Reference the workspace details and previous interactions when appropriate.")

	return builder.String()
}

// buildEnrichedContext builds enriched context from workspace detection and memory
func (p *Processor) buildEnrichedContext(ctx context.Context, request *QueryRequest) (*ProcessorContext, error) {
	// Step 1: Detect workspace context
	workspaceContext, err := p.detectWorkspaceContext(ctx, request)
	if err != nil {
		p.logger.Warn("Failed to detect workspace context", slog.Any("error", err))
		workspaceContext = &WorkspaceContext{} // Empty but typed
	}

	// Step 2: Retrieve relevant memory context
	memoryContext, err := p.retrieveMemoryContext(ctx, request)
	if err != nil {
		p.logger.Warn("Failed to retrieve memory context", slog.Any("error", err))
		memoryContext = &MemoryContext{} // Empty but typed
	}

	// Step 3: Analyze query intent and classify
	queryContext, err := p.analyzeQueryContext(ctx, request)
	if err != nil {
		p.logger.Warn("Failed to analyze query context", slog.Any("error", err))
		queryContext = &QueryContext{} // Empty but typed
	}

	enrichedContext := &ProcessorContext{
		Workspace: workspaceContext,
		Memory:    memoryContext,
		Query:     queryContext,
	}

	p.logger.Debug("Built enriched context",
		slog.String("workspace_project_type", workspaceContext.ProjectType),
		slog.String("memory_preferred_lang", func() string {
			if memoryContext.UserPreferences != nil {
				return memoryContext.UserPreferences.PreferredLanguage
			}
			return ""
		}()),
		slog.String("query_intent", queryContext.Intent))

	return enrichedContext, nil
}

// detectWorkspaceContext detects the current workspace context
func (p *Processor) detectWorkspaceContext(ctx context.Context, request *QueryRequest) (*WorkspaceContext, error) {
	// Use environment detector to get actual workspace information
	return p.envDetector.DetectWorkspace(), nil
}

// retrieveMemoryContext retrieves relevant memory context
func (p *Processor) retrieveMemoryContext(ctx context.Context, request *QueryRequest) (*MemoryContext, error) {
	// Default memory context - in production this would query the memory systems
	// TODO: Integrate with actual memory system once implemented
	userPreferences := &UserPreferences{
		PreferredLanguage:   "go",
		CodeStyle:           "standard_library_focused",
		Documentation:       "comprehensive",
		ResponseLanguage:    "traditional_chinese_or_english",
		LanguageRestriction: "never_use_simplified_chinese",
	}

	sessionContext := &SessionContext{
		PreviousQueries: 0, // TODO: Track actual query count
		SessionStart:    time.Now(),
		TopicsDiscussed: []string{}, // TODO: Track discussed topics
		LastActivity:    time.Now(),
	}

	memoryContext := &MemoryContext{
		UserPreferences: userPreferences,
		SessionContext:  sessionContext,
		WorkingMemory:   "Implementing context integration for Assistant",
		RecentTopics: []string{
			"Go code analysis tools implemented",
			"Context engine architecture defined",
			"Memory systems designed and documented",
		},
	}

	// TODO: Implement actual memory retrieval:
	// - Query episodic memory for recent interactions
	// - Retrieve semantic knowledge relevant to the query
	// - Get procedural memory for relevant workflows
	// - Access user preferences and personalization data

	return memoryContext, nil
}

// analyzeQueryContext analyzes the query for intent and context
func (p *Processor) analyzeQueryContext(ctx context.Context, request *QueryRequest) (*QueryContext, error) {
	query := request.Query

	// Simple intent classification
	intent := "general"
	category := "general"
	if containsAny(query, []string{"analyze", "review", "check", "audit"}) {
		intent = "code_analysis"
		category = "analysis"
	} else if containsAny(query, []string{"implement", "create", "build", "develop"}) {
		intent = "development"
		category = "creation"
	} else if containsAny(query, []string{"debug", "fix", "error", "bug"}) {
		intent = "debugging"
		category = "troubleshooting"
	} else if containsAny(query, []string{"deploy", "kubernetes", "docker", "container"}) {
		intent = "infrastructure"
		category = "operations"
	} else if containsAny(query, []string{"explain", "how", "what", "why"}) {
		intent = "explanation"
		category = "learning"
	}

	complexity := classifyComplexity(query)
	requiredTools := suggestTools(query)
	estimatedTokens := len(query) * 2 // Rough estimation

	// Extract keywords (simplified)
	keywords := extractKeywords(query)

	queryContext := &QueryContext{
		Intent:          intent,
		Category:        category,
		Complexity:      complexity,
		RequiredTools:   requiredTools,
		EstimatedTokens: estimatedTokens,
		Keywords:        keywords,
		Metadata: map[string]string{
			"analyzed_at":  time.Now().Format(time.RFC3339),
			"query_length": fmt.Sprintf("%d", len(query)),
		},
	}

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

func extractKeywords(query string) []string {
	// Simple keyword extraction - split by spaces and filter common words
	words := strings.Fields(strings.ToLower(query))
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "could": true, "should": true,
		"can": true, "may": true, "might": true, "must": true, "i": true, "you": true,
		"he": true, "she": true, "it": true, "we": true, "they": true, "me": true,
		"him": true, "her": true, "us": true, "them": true, "my": true, "your": true,
		"his": true, "its": true, "our": true, "their": true,
	}

	keywords := make([]string, 0)
	for _, word := range words {
		// Remove punctuation and check if it's not a common word
		cleanWord := strings.Trim(word, ".,!?;:")
		if len(cleanWord) > 2 && !commonWords[cleanWord] {
			keywords = append(keywords, cleanWord)
		}
	}

	return keywords
}

// generateCorrelationID generates a unique correlation ID for request tracking
func generateCorrelationID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("req_%s", hex.EncodeToString(bytes))
}

// ProcessStream processes a query and returns a streaming response
func (p *Processor) ProcessStream(ctx context.Context, request *QueryRequest) (<-chan *StreamChunk, error) {
	// Create channel for streaming
	chunkChan := make(chan *StreamChunk, 100)

	// Start processing in goroutine
	go func() {
		defer close(chunkChan)

		startTime := time.Now()
		correlationID := generateCorrelationID()
		ctx = context.WithValue(ctx, "correlation_id", correlationID)

		// Send initial chunk
		chunkChan <- &StreamChunk{
			Type: "start",
			Metadata: map[string]interface{}{
				"correlation_id": correlationID,
				"started_at":     startTime,
			},
		}

		// Validate request
		if err := p.validateRequest(request); err != nil {
			chunkChan <- &StreamChunk{
				Type:  "error",
				Error: err,
			}
			return
		}

		// Build enriched context
		enrichedContext, err := p.buildEnrichedContext(ctx, request)
		if err != nil {
			chunkChan <- &StreamChunk{
				Type:  "error",
				Error: err,
			}
			return
		}

		// Get or create conversation
		conversation, err := p.getOrCreateConversation(ctx, request)
		if err != nil {
			chunkChan <- &StreamChunk{
				Type:  "error",
				Error: err,
			}
			return
		}

		// Get conversation messages
		messages, err := p.conversationMgr.GetMessages(ctx, conversation.ID)
		if err != nil {
			p.logger.Warn("Failed to get conversation messages",
				slog.String("conversation_id", conversation.ID),
				slog.Any("error", err))
			messages = nil // Empty messages on error
		}

		// Add user message
		messageContext := request.Context
		if messageContext == nil {
			messageContext = make(map[string]interface{})
		}
		if enrichedContext.Workspace != nil {
			messageContext["workspace"] = enrichedContext.Workspace
		}
		if enrichedContext.Memory != nil {
			messageContext["memory"] = enrichedContext.Memory
		}

		userMessage, err := p.conversationMgr.AddMessage(ctx, conversation.ID, "user", request.Query)
		if err != nil {
			chunkChan <- &StreamChunk{
				Type:  "error",
				Error: err,
			}
			return
		}

		// Prepare AI request with streaming
		provider := p.aiService.GetDefaultProvider()
		if request.Provider != nil {
			provider = *request.Provider
		}

		// Use default model from config based on provider
		var model string
		switch provider {
		case "claude":
			model = p.config.AI.Claude.Model
		case "gemini":
			model = p.config.AI.Gemini.Model
		default:
			model = "claude-3-sonnet-20240229" // fallback
		}
		if request.Model != nil {
			model = *request.Model
		}

		// Build messages for AI
		aiMessages := []ai.Message{}
		for _, msg := range messages {
			aiMessages = append(aiMessages, ai.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
		aiMessages = append(aiMessages, ai.Message{
			Role:    "user",
			Content: request.Query,
		})

		// Create streaming AI request
		aiRequest := &ai.GenerateStreamRequest{
			Messages:     aiMessages,
			Model:        model,
			Temperature:  0.7,
			MaxTokens:    4000,
			SystemPrompt: nil, // TODO: Add system prompt based on context
			Metadata: &ai.RequestMetadata{
				ConversationID: conversation.ID,
				UserID:         conversation.UserID,
				RequestID:      userMessage.ID,
			},
		}

		// Get streaming response from AI
		streamResp, err := p.aiService.GenerateResponseStream(ctx, aiRequest, provider)
		if err != nil {
			chunkChan <- &StreamChunk{
				Type:  "error",
				Error: err,
			}
			return
		}

		// Buffer for accumulating content
		var fullContent strings.Builder
		var tokensUsed ai.TokenUsage

		// Stream chunks from AI
		for aiChunk := range streamResp.ChunkChan {
			if aiChunk.Error != nil {
				chunkChan <- &StreamChunk{
					Type:  "error",
					Error: aiChunk.Error,
				}
				return
			}

			// Accumulate content
			if aiChunk.Content != "" {
				fullContent.WriteString(aiChunk.Content)

				// Send content chunk
				chunkChan <- &StreamChunk{
					Type:    "content",
					Content: aiChunk.Content,
				}
			}

			// Handle final chunk with metadata
			if aiChunk.FinishReason != "" {
				if aiChunk.TokensUsed != nil {
					tokensUsed = *aiChunk.TokensUsed
				}
			}
		}

		// Wait for stream to complete
		<-streamResp.Done

		// Store assistant message
		// TODO: Handle metadata when supported
		assistantMessage, err := p.conversationMgr.AddMessage(
			ctx,
			conversation.ID,
			"assistant",
			fullContent.String(),
		)
		if err != nil {
			p.logger.Warn("Failed to store assistant message",
				slog.Any("error", err))
		}

		// Send completion chunk
		chunkChan <- &StreamChunk{
			Type: "complete",
			Metadata: map[string]interface{}{
				"conversation_id": conversation.ID,
				"message_id":      assistantMessage.ID,
				"provider":        provider,
				"model":           model,
				"tokens_used":     tokensUsed.TotalTokens,
				"execution_time":  time.Since(startTime).String(),
			},
		}
	}()

	return chunkChan, nil
}

// StreamChunk represents a chunk in the streaming response
type StreamChunk struct {
	Type     string                 `json:"type"` // "start", "content", "error", "complete"
	Content  string                 `json:"content,omitempty"`
	Error    error                  `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
