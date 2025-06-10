package langchain

import (
	"context"
	"time"
)

// AI types defined locally to avoid import cycles

// AIMessage represents a message in the AI package
type AIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIGenerateRequest represents a request to generate a response in the AI package
type AIGenerateRequest struct {
	Messages     []AIMessage            `json:"messages"`
	Model        string                 `json:"model,omitempty"`
	MaxTokens    int                    `json:"max_tokens,omitempty"`
	Temperature  float64                `json:"temperature,omitempty"`
	SystemPrompt *string                `json:"system_prompt,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AITokenUsage represents token usage statistics in the AI package
type AITokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// AIGenerateResponse represents a response from AI generation in the AI package
type AIGenerateResponse struct {
	Content      string                 `json:"content"`
	Model        string                 `json:"model"`
	Provider     string                 `json:"provider"`
	TokensUsed   AITokenUsage           `json:"tokens_used"`
	FinishReason string                 `json:"finish_reason"`
	ResponseTime time.Duration          `json:"response_time"`
	RequestID    string                 `json:"request_id"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AIEmbeddingResponse represents a response from embedding generation in the AI package
type AIEmbeddingResponse struct {
	Embedding    []float64     `json:"embedding"`
	Model        string        `json:"model"`
	Provider     string        `json:"provider"`
	TokensUsed   int           `json:"tokens_used"`
	ResponseTime time.Duration `json:"response_time"`
}

// AIUsageStats represents usage statistics for a provider in the AI package
type AIUsageStats struct {
	TotalRequests   int64         `json:"total_requests"`
	TotalTokens     int64         `json:"total_tokens"`
	InputTokens     int64         `json:"input_tokens"`
	OutputTokens    int64         `json:"output_tokens"`
	TotalCost       float64       `json:"total_cost"`
	AverageLatency  time.Duration `json:"average_latency"`
	ErrorRate       float64       `json:"error_rate"`
	LastRequestTime *time.Time    `json:"last_request_time,omitempty"`
	RequestsPerHour float64       `json:"requests_per_hour"`
}

// LangChainAdapter adapts Client to implement the ai.Provider interface
type LangChainAdapter struct {
	client *Client
}

// NewLangChainAdapter creates a new adapter for Client
func NewLangChainAdapter(client *Client) *LangChainAdapter {
	return &LangChainAdapter{
		client: client,
	}
}

// Name returns the provider name
func (a *LangChainAdapter) Name() string {
	return a.client.Name()
}

// GenerateResponse generates a response using LangChain
func (a *LangChainAdapter) GenerateResponse(ctx context.Context, request *AIGenerateRequest) (*AIGenerateResponse, error) {
	// Convert AIGenerateRequest to langchain.GenerateRequest
	langchainRequest := &GenerateRequest{
		Messages:     convertAIMessagesToLangChain(request.Messages),
		Model:        request.Model,
		MaxTokens:    request.MaxTokens,
		Temperature:  request.Temperature,
		SystemPrompt: request.SystemPrompt,
		Metadata:     request.Metadata,
	}

	// Call the LangChain client
	langchainResponse, err := a.client.GenerateResponse(ctx, langchainRequest)
	if err != nil {
		return nil, err
	}

	// Convert langchain.GenerateResponse to AIGenerateResponse
	return &AIGenerateResponse{
		Content:      langchainResponse.Content,
		Model:        langchainResponse.Model,
		Provider:     langchainResponse.Provider,
		TokensUsed:   convertLangChainTokenUsageToAI(langchainResponse.TokensUsed),
		FinishReason: langchainResponse.FinishReason,
		ResponseTime: langchainResponse.ResponseTime,
		RequestID:    langchainResponse.RequestID,
		Metadata:     langchainResponse.Metadata,
	}, nil
}

// GenerateEmbedding generates embeddings using LangChain
func (a *LangChainAdapter) GenerateEmbedding(ctx context.Context, text string) (*AIEmbeddingResponse, error) {
	// Call the LangChain client
	langchainResponse, err := a.client.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}

	// Convert langchain.EmbeddingResponse to AIEmbeddingResponse
	return &AIEmbeddingResponse{
		Embedding:    langchainResponse.Embedding,
		Model:        langchainResponse.Model,
		Provider:     langchainResponse.Provider,
		TokensUsed:   langchainResponse.TokensUsed,
		ResponseTime: langchainResponse.ResponseTime,
	}, nil
}

// Health checks if the LangChain client is healthy
func (a *LangChainAdapter) Health(ctx context.Context) error {
	return a.client.Health(ctx)
}

// Close closes the LangChain client
func (a *LangChainAdapter) Close(ctx context.Context) error {
	return a.client.Close(ctx)
}

// GetUsage returns usage statistics
func (a *LangChainAdapter) GetUsage(ctx context.Context) (*AIUsageStats, error) {
	langchainStats, err := a.client.GetUsage(ctx)
	if err != nil {
		return nil, err
	}

	// Convert langchain.UsageStats to AIUsageStats
	return &AIUsageStats{
		TotalRequests:   langchainStats.TotalRequests,
		TotalTokens:     langchainStats.TotalTokens,
		InputTokens:     langchainStats.InputTokens,
		OutputTokens:    langchainStats.OutputTokens,
		TotalCost:       langchainStats.TotalCost,
		AverageLatency:  langchainStats.AverageLatency,
		ErrorRate:       langchainStats.ErrorRate,
		LastRequestTime: langchainStats.LastRequestTime,
		RequestsPerHour: langchainStats.RequestsPerHour,
	}, nil
}

// Helper functions for type conversion

func convertAIMessagesToLangChain(messages []AIMessage) []Message {
	langchainMessages := make([]Message, len(messages))
	for i, msg := range messages {
		langchainMessages[i] = Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return langchainMessages
}

func convertLangChainTokenUsageToAI(usage TokenUsage) AITokenUsage {
	return AITokenUsage{
		InputTokens:  usage.InputTokens,
		OutputTokens: usage.OutputTokens,
		TotalTokens:  usage.TotalTokens,
	}
}
