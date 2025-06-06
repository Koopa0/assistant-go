package ai

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/koopa0/assistant-go/internal/ai/claude"
	"github.com/koopa0/assistant-go/internal/ai/gemini"
	"github.com/koopa0/assistant-go/internal/ai/prompts"
	"github.com/koopa0/assistant-go/internal/config"
)

// Service provides a unified AI service using direct dependencies
// Uses concrete clients instead of interfaces for simplicity
type Service struct {
	claudeClient    *claude.Client
	geminiClient    *gemini.Client
	promptService   *prompts.PromptService
	defaultProvider string
	logger          *slog.Logger
}

// NewService creates a new AI service with direct dependencies
// Following Go's "accept interfaces, return structs" principle
func NewService(cfg *config.Config, logger *slog.Logger) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	svc := &Service{
		defaultProvider: cfg.AI.DefaultProvider,
		logger:          logger,
		promptService:   prompts.NewPromptService(logger),
	}

	// Initialize Claude if configured
	if cfg.AI.Claude.APIKey != "" {
		claudeConfig := claude.ProviderConfig{
			APIKey:      cfg.AI.Claude.APIKey,
			BaseURL:     cfg.AI.Claude.BaseURL,
			Model:       cfg.AI.Claude.Model,
			MaxTokens:   cfg.AI.Claude.MaxTokens,
			Temperature: cfg.AI.Claude.Temperature,
			Timeout:     30 * time.Second,
		}

		claudeClient, err := claude.NewClient(claudeConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create Claude client: %w", err)
		}
		svc.claudeClient = claudeClient
		logger.Info("Claude client initialized")
	}

	// Initialize Gemini if configured
	if cfg.AI.Gemini.APIKey != "" {
		geminiConfig := gemini.ProviderConfig{
			APIKey:      cfg.AI.Gemini.APIKey,
			BaseURL:     cfg.AI.Gemini.BaseURL,
			Model:       cfg.AI.Gemini.Model,
			MaxTokens:   cfg.AI.Gemini.MaxTokens,
			Temperature: cfg.AI.Gemini.Temperature,
			Timeout:     30 * time.Second,
		}

		geminiClient, err := gemini.NewClient(geminiConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create Gemini client: %w", err)
		}
		svc.geminiClient = geminiClient
		logger.Info("Gemini client initialized")
	}

	// Validate at least one provider is available
	if svc.claudeClient == nil && svc.geminiClient == nil {
		return nil, fmt.Errorf("no AI providers configured")
	}

	// Validate default provider is available
	if !svc.isProviderAvailable(svc.defaultProvider) {
		return nil, fmt.Errorf("default provider %s is not available", svc.defaultProvider)
	}

	logger.Info("AI service initialized",
		slog.String("default_provider", svc.defaultProvider),
		slog.Int("provider_count", svc.getProviderCount()),
		slog.Any("available_providers", svc.GetAvailableProviders()))

	return svc, nil
}

// GenerateResponse generates a response using the specified or default provider
func (s *Service) GenerateResponse(ctx context.Context, request *GenerateRequest, providerName ...string) (*GenerateResponse, error) {
	provider := s.defaultProvider
	if len(providerName) > 0 && providerName[0] != "" {
		provider = providerName[0]
	}

	switch provider {
	case "claude":
		if s.claudeClient == nil {
			return nil, fmt.Errorf("Claude provider not available")
		}
		// Convert ai.GenerateRequest to claude.GenerateRequest
		claudeReq := &claude.GenerateRequest{
			Messages:     convertMessagesToClaude(request.Messages),
			MaxTokens:    request.MaxTokens,
			Temperature:  request.Temperature,
			Model:        request.Model,
			SystemPrompt: request.SystemPrompt,
			Metadata:     request.Metadata,
		}
		resp, err := s.claudeClient.GenerateResponse(ctx, claudeReq)
		if err != nil {
			return nil, err
		}
		return convertClaudeResponse(resp), nil
	case "gemini":
		if s.geminiClient == nil {
			return nil, fmt.Errorf("Gemini provider not available")
		}
		// Convert ai.GenerateRequest to gemini.GenerateRequest
		geminiReq := &gemini.GenerateRequest{
			Messages:     convertMessagesToGemini(request.Messages),
			MaxTokens:    request.MaxTokens,
			Temperature:  request.Temperature,
			Model:        request.Model,
			SystemPrompt: request.SystemPrompt,
			Metadata:     request.Metadata,
		}
		resp, err := s.geminiClient.GenerateResponse(ctx, geminiReq)
		if err != nil {
			return nil, err
		}
		return convertGeminiResponse(resp), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

// GenerateEmbedding generates embeddings using the specified or default provider
func (s *Service) GenerateEmbedding(ctx context.Context, text string, providerName ...string) (*EmbeddingResponse, error) {
	provider := s.defaultProvider
	if len(providerName) > 0 && providerName[0] != "" {
		provider = providerName[0]
	}

	switch provider {
	case "claude":
		if s.claudeClient == nil {
			return nil, fmt.Errorf("Claude provider not available")
		}
		resp, err := s.claudeClient.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, err
		}
		return convertClaudeEmbeddingResponse(resp), nil
	case "gemini":
		if s.geminiClient == nil {
			return nil, fmt.Errorf("Gemini provider not available")
		}
		resp, err := s.geminiClient.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, err
		}
		return convertGeminiEmbeddingResponse(resp), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

// GetAvailableProviders returns a list of available providers
func (s *Service) GetAvailableProviders() []string {
	var providers []string
	if s.claudeClient != nil {
		providers = append(providers, "claude")
	}
	if s.geminiClient != nil {
		providers = append(providers, "gemini")
	}
	return providers
}

// GetDefaultProvider returns the default provider name
func (s *Service) GetDefaultProvider() string {
	return s.defaultProvider
}

// SetDefaultProvider sets the default provider
func (s *Service) SetDefaultProvider(name string) error {
	if !s.isProviderAvailable(name) {
		return fmt.Errorf("provider %s is not available", name)
	}

	s.defaultProvider = name
	s.logger.Info("Default provider changed", slog.String("provider", name))
	return nil
}

// Health checks the health of all available providers
func (s *Service) Health(ctx context.Context) error {
	if s.claudeClient != nil {
		if err := s.claudeClient.Health(ctx); err != nil {
			return fmt.Errorf("Claude health check failed: %w", err)
		}
	}
	if s.geminiClient != nil {
		if err := s.geminiClient.Health(ctx); err != nil {
			return fmt.Errorf("Gemini health check failed: %w", err)
		}
	}
	return nil
}

// GetUsageStats returns usage statistics for all providers
func (s *Service) GetUsageStats(ctx context.Context) (map[string]*UsageStats, error) {
	stats := make(map[string]*UsageStats)

	if s.claudeClient != nil {
		providerStats, err := s.claudeClient.GetUsage(ctx)
		if err != nil {
			s.logger.Warn("Failed to get usage stats for Claude",
				slog.Any("error", err))
		} else {
			stats["claude"] = convertClaudeUsageStats(providerStats)
		}
	}

	if s.geminiClient != nil {
		providerStats, err := s.geminiClient.GetUsage(ctx)
		if err != nil {
			s.logger.Warn("Failed to get usage stats for Gemini",
				slog.Any("error", err))
		} else {
			stats["gemini"] = convertGeminiUsageStats(providerStats)
		}
	}

	return stats, nil
}

// Close closes all provider connections
func (s *Service) Close(ctx context.Context) error {
	var lastErr error

	if s.claudeClient != nil {
		if err := s.claudeClient.Close(ctx); err != nil {
			s.logger.Error("Failed to close Claude client",
				slog.Any("error", err))
			lastErr = err
		}
	}

	if s.geminiClient != nil {
		if err := s.geminiClient.Close(ctx); err != nil {
			s.logger.Error("Failed to close Gemini client",
				slog.Any("error", err))
			lastErr = err
		}
	}

	return lastErr
}

// Private helper methods

func (s *Service) isProviderAvailable(provider string) bool {
	switch provider {
	case "claude":
		return s.claudeClient != nil
	case "gemini":
		return s.geminiClient != nil
	default:
		return false
	}
}

func (s *Service) getProviderCount() int {
	count := 0
	if s.claudeClient != nil {
		count++
	}
	if s.geminiClient != nil {
		count++
	}
	return count
}

// Conversion functions

func convertMessagesToClaude(messages []Message) []claude.Message {
	claudeMessages := make([]claude.Message, len(messages))
	for i, msg := range messages {
		claudeMessages[i] = claude.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return claudeMessages
}

func convertMessagesToGemini(messages []Message) []gemini.Message {
	geminiMessages := make([]gemini.Message, len(messages))
	for i, msg := range messages {
		geminiMessages[i] = gemini.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return geminiMessages
}

func convertClaudeResponse(resp *claude.GenerateResponse) *GenerateResponse {
	return &GenerateResponse{
		Content:      resp.Content,
		Model:        resp.Model,
		Provider:     resp.Provider,
		TokensUsed:   TokenUsage(resp.TokensUsed),
		FinishReason: resp.FinishReason,
		ResponseTime: resp.ResponseTime,
		RequestID:    resp.RequestID,
		Metadata:     resp.Metadata,
	}
}

func convertGeminiResponse(resp *gemini.GenerateResponse) *GenerateResponse {
	return &GenerateResponse{
		Content:      resp.Content,
		Model:        resp.Model,
		Provider:     resp.Provider,
		TokensUsed:   TokenUsage(resp.TokensUsed),
		FinishReason: resp.FinishReason,
		ResponseTime: resp.ResponseTime,
		RequestID:    resp.RequestID,
		Metadata:     resp.Metadata,
	}
}

func convertClaudeEmbeddingResponse(resp *claude.EmbeddingResponse) *EmbeddingResponse {
	return &EmbeddingResponse{
		Embedding:    resp.Embedding,
		Model:        resp.Model,
		Provider:     resp.Provider,
		TokensUsed:   resp.TokensUsed,
		ResponseTime: resp.ResponseTime,
		RequestID:    resp.RequestID,
	}
}

func convertGeminiEmbeddingResponse(resp *gemini.EmbeddingResponse) *EmbeddingResponse {
	return &EmbeddingResponse{
		Embedding:    resp.Embedding,
		Model:        resp.Model,
		Provider:     resp.Provider,
		TokensUsed:   resp.TokensUsed,
		ResponseTime: resp.ResponseTime,
		RequestID:    resp.RequestID,
	}
}

func convertClaudeUsageStats(stats *claude.UsageStats) *UsageStats {
	return &UsageStats{
		TotalRequests:   stats.TotalRequests,
		TotalTokens:     stats.TotalTokens,
		InputTokens:     stats.InputTokens,
		OutputTokens:    stats.OutputTokens,
		TotalCost:       stats.TotalCost,
		AverageLatency:  stats.AverageLatency,
		ErrorRate:       stats.ErrorRate,
		LastRequestTime: stats.LastRequestTime,
		RequestsPerHour: stats.RequestsPerHour,
	}
}

func convertGeminiUsageStats(stats *gemini.UsageStats) *UsageStats {
	return &UsageStats{
		TotalRequests:   stats.TotalRequests,
		TotalTokens:     stats.TotalTokens,
		InputTokens:     stats.InputTokens,
		OutputTokens:    stats.OutputTokens,
		TotalCost:       stats.TotalCost,
		AverageLatency:  stats.AverageLatency,
		ErrorRate:       stats.ErrorRate,
		LastRequestTime: stats.LastRequestTime,
		RequestsPerHour: stats.RequestsPerHour,
	}
}

// ProcessEnhancedQuery processes a user query with intelligent prompt enhancement
func (s *Service) ProcessEnhancedQuery(ctx context.Context, userQuery string, promptCtx *prompts.PromptContext, providerName ...string) (*EnhancedQueryResponse, error) {
	// Enhance the query with intelligent prompts
	enhanced, err := s.promptService.EnhanceQuery(ctx, userQuery, promptCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to enhance query: %w", err)
	}

	s.logger.Info("Processing enhanced query",
		"original_query", userQuery,
		"task_type", enhanced.TaskType,
		"confidence", enhanced.Confidence,
		"template", enhanced.PromptTemplate)

	// Create AI request with enhanced prompt
	request := &GenerateRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: enhanced.EnhancedQuery,
			},
		},
		SystemPrompt: &enhanced.SystemPrompt,
		Temperature:  0.1, // Lower temperature for more consistent technical responses
		MaxTokens:    4000,
		Metadata: map[string]interface{}{
			"task_type":       enhanced.TaskType,
			"confidence":      enhanced.Confidence,
			"prompt_template": enhanced.PromptTemplate,
			"module_path":     promptCtx.ModulePath,
			"project_type":    promptCtx.ProjectType,
		},
	}

	// Generate response using AI service
	response, err := s.GenerateResponse(ctx, request, providerName...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AI response: %w", err)
	}

	// Return enhanced response
	return &EnhancedQueryResponse{
		OriginalQuery:  userQuery,
		EnhancedQuery:  enhanced.EnhancedQuery,
		TaskType:       enhanced.TaskType,
		Confidence:     enhanced.Confidence,
		PromptTemplate: enhanced.PromptTemplate,
		AIResponse:     response,
		Context:        enhanced.Context,
	}, nil
}

// GetPromptService returns the prompt service for direct access
func (s *Service) GetPromptService() *prompts.PromptService {
	return s.promptService
}

// EnhancedQueryResponse represents the response from an enhanced query
type EnhancedQueryResponse struct {
	OriginalQuery  string                 `json:"original_query"`
	EnhancedQuery  string                 `json:"enhanced_query"`
	TaskType       string                 `json:"task_type"`
	Confidence     float64                `json:"confidence"`
	PromptTemplate string                 `json:"prompt_template"`
	AIResponse     *GenerateResponse      `json:"ai_response"`
	Context        *prompts.PromptContext `json:"context"`
}
