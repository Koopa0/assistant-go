package ai

import (
	"context"
	"log/slog"

	"github.com/koopa0/assistant/internal/ai/claude"
	"github.com/koopa0/assistant/internal/ai/gemini"
)

// claudeAdapter wraps a Claude client to implement the Provider interface
type claudeAdapter struct {
	client *claude.Client
}

// Name returns the provider name
func (a *claudeAdapter) Name() string {
	return a.client.Name()
}

// GenerateResponse generates a response using Claude API
func (a *claudeAdapter) GenerateResponse(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error) {
	// Convert ai.GenerateRequest to claude.GenerateRequest
	claudeReq := &claude.GenerateRequest{
		Messages:     convertMessagesToClaude(request.Messages),
		MaxTokens:    request.MaxTokens,
		Temperature:  request.Temperature,
		Model:        request.Model,
		SystemPrompt: request.SystemPrompt,
		Metadata:     request.Metadata,
	}

	// Call Claude client
	claudeResp, err := a.client.GenerateResponse(ctx, claudeReq)
	if err != nil {
		return nil, err
	}

	// Convert claude.GenerateResponse to ai.GenerateResponse
	return &GenerateResponse{
		Content:      claudeResp.Content,
		Model:        claudeResp.Model,
		Provider:     claudeResp.Provider,
		TokensUsed:   TokenUsage(claudeResp.TokensUsed),
		FinishReason: claudeResp.FinishReason,
		ResponseTime: claudeResp.ResponseTime,
		RequestID:    claudeResp.RequestID,
		Metadata:     claudeResp.Metadata,
	}, nil
}

// GenerateEmbedding generates embeddings using Claude API
func (a *claudeAdapter) GenerateEmbedding(ctx context.Context, text string) (*EmbeddingResponse, error) {
	claudeResp, err := a.client.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}

	return &EmbeddingResponse{
		Embedding:    claudeResp.Embedding,
		Model:        claudeResp.Model,
		Provider:     claudeResp.Provider,
		TokensUsed:   claudeResp.TokensUsed,
		ResponseTime: claudeResp.ResponseTime,
		RequestID:    claudeResp.RequestID,
	}, nil
}

// Health checks if Claude API is accessible
func (a *claudeAdapter) Health(ctx context.Context) error {
	return a.client.Health(ctx)
}

// Close closes the Claude client
func (a *claudeAdapter) Close(ctx context.Context) error {
	return a.client.Close(ctx)
}

// GetUsage returns usage statistics
func (a *claudeAdapter) GetUsage(ctx context.Context) (*UsageStats, error) {
	claudeStats, err := a.client.GetUsage(ctx)
	if err != nil {
		return nil, err
	}

	return &UsageStats{
		TotalRequests:   claudeStats.TotalRequests,
		TotalTokens:     claudeStats.TotalTokens,
		InputTokens:     claudeStats.InputTokens,
		OutputTokens:    claudeStats.OutputTokens,
		TotalCost:       claudeStats.TotalCost,
		AverageLatency:  claudeStats.AverageLatency,
		ErrorRate:       claudeStats.ErrorRate,
		LastRequestTime: claudeStats.LastRequestTime,
		RequestsPerHour: claudeStats.RequestsPerHour,
	}, nil
}

// geminiAdapter wraps a Gemini client to implement the Provider interface
type geminiAdapter struct {
	client *gemini.Client
}

// Name returns the provider name
func (a *geminiAdapter) Name() string {
	return a.client.Name()
}

// GenerateResponse generates a response using Gemini API
func (a *geminiAdapter) GenerateResponse(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error) {
	// Convert ai.GenerateRequest to gemini.GenerateRequest
	geminiReq := &gemini.GenerateRequest{
		Messages:     convertMessagesToGemini(request.Messages),
		MaxTokens:    request.MaxTokens,
		Temperature:  request.Temperature,
		Model:        request.Model,
		SystemPrompt: request.SystemPrompt,
		Metadata:     request.Metadata,
	}

	// Call Gemini client
	geminiResp, err := a.client.GenerateResponse(ctx, geminiReq)
	if err != nil {
		return nil, err
	}

	// Convert gemini.GenerateResponse to ai.GenerateResponse
	return &GenerateResponse{
		Content:      geminiResp.Content,
		Model:        geminiResp.Model,
		Provider:     geminiResp.Provider,
		TokensUsed:   TokenUsage(geminiResp.TokensUsed),
		FinishReason: geminiResp.FinishReason,
		ResponseTime: geminiResp.ResponseTime,
		RequestID:    geminiResp.RequestID,
		Metadata:     geminiResp.Metadata,
	}, nil
}

// GenerateEmbedding generates embeddings using Gemini API
func (a *geminiAdapter) GenerateEmbedding(ctx context.Context, text string) (*EmbeddingResponse, error) {
	geminiResp, err := a.client.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}

	return &EmbeddingResponse{
		Embedding:    geminiResp.Embedding,
		Model:        geminiResp.Model,
		Provider:     geminiResp.Provider,
		TokensUsed:   geminiResp.TokensUsed,
		ResponseTime: geminiResp.ResponseTime,
		RequestID:    geminiResp.RequestID,
	}, nil
}

// Health checks if Gemini API is accessible
func (a *geminiAdapter) Health(ctx context.Context) error {
	return a.client.Health(ctx)
}

// Close closes the Gemini client
func (a *geminiAdapter) Close(ctx context.Context) error {
	return a.client.Close(ctx)
}

// GetUsage returns usage statistics
func (a *geminiAdapter) GetUsage(ctx context.Context) (*UsageStats, error) {
	geminiStats, err := a.client.GetUsage(ctx)
	if err != nil {
		return nil, err
	}

	return &UsageStats{
		TotalRequests:   geminiStats.TotalRequests,
		TotalTokens:     geminiStats.TotalTokens,
		InputTokens:     geminiStats.InputTokens,
		OutputTokens:    geminiStats.OutputTokens,
		TotalCost:       geminiStats.TotalCost,
		AverageLatency:  geminiStats.AverageLatency,
		ErrorRate:       geminiStats.ErrorRate,
		LastRequestTime: geminiStats.LastRequestTime,
		RequestsPerHour: geminiStats.RequestsPerHour,
	}, nil
}

// Helper functions to convert between types
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

// RegisterProviders registers all available AI providers with the factory
func RegisterProviders(factory *Factory) {
	// Register Claude provider
	factory.RegisterProvider("claude", func(config ProviderConfig, logger *slog.Logger) (Provider, error) {
		// Convert ai.ProviderConfig to claude.ProviderConfig
		claudeConfig := claude.ProviderConfig{
			APIKey:      config.APIKey,
			BaseURL:     config.BaseURL,
			Model:       config.Model,
			MaxTokens:   config.MaxTokens,
			Temperature: config.Temperature,
			Timeout:     config.Timeout,
		}

		claudeClient, err := claude.NewClient(claudeConfig, logger)
		if err != nil {
			return nil, err
		}

		return &claudeAdapter{client: claudeClient}, nil
	})

	// Register Gemini provider
	factory.RegisterProvider("gemini", func(config ProviderConfig, logger *slog.Logger) (Provider, error) {
		// Convert ai.ProviderConfig to gemini.ProviderConfig
		geminiConfig := gemini.ProviderConfig{
			APIKey:      config.APIKey,
			BaseURL:     config.BaseURL,
			Model:       config.Model,
			MaxTokens:   config.MaxTokens,
			Temperature: config.Temperature,
			Timeout:     config.Timeout,
		}

		geminiClient, err := gemini.NewClient(geminiConfig, logger)
		if err != nil {
			return nil, err
		}

		return &geminiAdapter{client: geminiClient}, nil
	})
}
