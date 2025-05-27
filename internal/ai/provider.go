package ai

import (
	"context"
	"time"
)

// Provider represents an AI provider interface
type Provider interface {
	// Name returns the provider name
	Name() string

	// GenerateResponse generates a response for the given messages
	GenerateResponse(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error)

	// GenerateEmbedding generates embeddings for the given text
	GenerateEmbedding(ctx context.Context, text string) (*EmbeddingResponse, error)

	// Health checks if the provider is healthy and accessible
	Health(ctx context.Context) error

	// Close closes the provider and cleans up resources
	Close(ctx context.Context) error

	// GetUsage returns usage statistics
	GetUsage(ctx context.Context) (*UsageStats, error)
}

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content string `json:"content"`
}

// GenerateRequest represents a request to generate a response
type GenerateRequest struct {
	Messages     []Message              `json:"messages"`
	MaxTokens    int                    `json:"max_tokens,omitempty"`
	Temperature  float64                `json:"temperature,omitempty"`
	Model        string                 `json:"model,omitempty"`
	SystemPrompt *string                `json:"system_prompt,omitempty"`
	Tools        []Tool                 `json:"tools,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// GenerateResponse represents a response from the AI provider
type GenerateResponse struct {
	Content      string                 `json:"content"`
	Model        string                 `json:"model"`
	Provider     string                 `json:"provider"`
	TokensUsed   TokenUsage             `json:"tokens_used"`
	FinishReason string                 `json:"finish_reason"`
	ResponseTime time.Duration          `json:"response_time"`
	RequestID    string                 `json:"request_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	ToolCalls    []ToolCall             `json:"tool_calls,omitempty"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Embedding    []float64     `json:"embedding"`
	Model        string        `json:"model"`
	Provider     string        `json:"provider"`
	TokensUsed   int           `json:"tokens_used"`
	ResponseTime time.Duration `json:"response_time"`
	RequestID    string        `json:"request_id,omitempty"`
}

// Tool represents a tool that can be called by the AI
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool call made by the AI
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// UsageStats represents usage statistics for a provider
type UsageStats struct {
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

// ProviderConfig represents configuration for an AI provider
type ProviderConfig struct {
	APIKey      string        `json:"api_key"`
	BaseURL     string        `json:"base_url"`
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Timeout     time.Duration `json:"timeout"`
}

// Error types for AI providers
const (
	ErrorTypeAuthentication = "authentication_error"
	ErrorTypeRateLimit      = "rate_limit_error"
	ErrorTypeQuotaExceeded  = "quota_exceeded_error"
	ErrorTypeInvalidRequest = "invalid_request_error"
	ErrorTypeServerError    = "server_error"
	ErrorTypeTimeout        = "timeout_error"
	ErrorTypeNetworkError   = "network_error"
	ErrorTypeUnknown        = "unknown_error"
)

// ProviderError represents an error from an AI provider
type ProviderError struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	Code      string `json:"code,omitempty"`
	Provider  string `json:"provider"`
	RequestID string `json:"request_id,omitempty"`
	Retryable bool   `json:"retryable"`
}

// Error implements the error interface
func (e *ProviderError) Error() string {
	return e.Message
}

// IsRetryable returns whether the error is retryable
func (e *ProviderError) IsRetryable() bool {
	return e.Retryable
}

// NewProviderError creates a new provider error
func NewProviderError(errorType, message, provider string) *ProviderError {
	retryable := false
	switch errorType {
	case ErrorTypeRateLimit, ErrorTypeServerError, ErrorTypeTimeout, ErrorTypeNetworkError:
		retryable = true
	}

	return &ProviderError{
		Type:      errorType,
		Message:   message,
		Provider:  provider,
		Retryable: retryable,
	}
}

// ProviderFactory creates AI provider instances
type ProviderFactory interface {
	CreateProvider(name string, config ProviderConfig) (Provider, error)
	SupportedProviders() []string
}

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow(ctx context.Context, key string) error
	Reset(key string)
}

// TokenCounter interface for counting tokens
type TokenCounter interface {
	CountTokens(text string, model string) (int, error)
	EstimateTokens(text string) int
}
