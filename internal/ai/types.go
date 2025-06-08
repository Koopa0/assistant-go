package ai

import (
	"time"
)

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content string `json:"content"`
}

// RequestMetadata contains metadata for AI requests
type RequestMetadata struct {
	RequestID      string            `json:"request_id,omitempty"`
	UserID         string            `json:"user_id,omitempty"`
	SessionID      string            `json:"session_id,omitempty"`
	ConversationID string            `json:"conversation_id,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Features       map[string]string `json:"features,omitempty"` // Feature flags or settings
}

// ResponseMetadata contains metadata for AI responses
type ResponseMetadata struct {
	ProcessingTime time.Duration `json:"processing_time"`
	Provider       string        `json:"provider"`
	Model          string        `json:"model"`
	ModelVersion   string        `json:"model_version,omitempty"`
	Region         string        `json:"region,omitempty"`
	Debug          *DebugInfo    `json:"debug,omitempty"`
}

// DebugInfo contains debugging information for development
type DebugInfo struct {
	PromptTokens     int                `json:"prompt_tokens"`
	CompletionTokens int                `json:"completion_tokens"`
	InternalMetrics  map[string]float64 `json:"internal_metrics,omitempty"`
	Warnings         []string           `json:"warnings,omitempty"`
}

// GenerateRequest represents a request to generate a response
type GenerateRequest struct {
	Messages     []Message        `json:"messages"`
	MaxTokens    int              `json:"max_tokens,omitempty"`
	Temperature  float64          `json:"temperature,omitempty"`
	Model        string           `json:"model,omitempty"`
	SystemPrompt *string          `json:"system_prompt,omitempty"`
	Tools        []Tool           `json:"tools,omitempty"`
	Metadata     *RequestMetadata `json:"metadata,omitempty"`
}

// GenerateResponse represents a response from the AI provider
type GenerateResponse struct {
	Content      string            `json:"content"`
	Model        string            `json:"model"`
	Provider     string            `json:"provider"`
	TokensUsed   TokenUsage        `json:"tokens_used"`
	FinishReason string            `json:"finish_reason"`
	ResponseTime time.Duration     `json:"response_time"`
	RequestID    string            `json:"request_id,omitempty"`
	Metadata     *ResponseMetadata `json:"metadata,omitempty"`
	ToolCalls    []ToolCall        `json:"tool_calls,omitempty"`
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

// ToolParameterSchema represents the JSON Schema for tool parameters
type ToolParameterSchema struct {
	Type        string                       `json:"type"` // "object"
	Properties  map[string]ParameterProperty `json:"properties"`
	Required    []string                     `json:"required,omitempty"`
	Description string                       `json:"description,omitempty"`
}

// ParameterProperty represents a property in the parameter schema
type ParameterProperty struct {
	Type        string      `json:"type"` // "string", "number", "boolean", "array", "object"
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"` // Default value can be any type
	Enum        []string    `json:"enum,omitempty"`
	Format      string      `json:"format,omitempty"` // e.g., "date-time", "email"
	MinLength   *int        `json:"minLength,omitempty"`
	MaxLength   *int        `json:"maxLength,omitempty"`
	Minimum     *float64    `json:"minimum,omitempty"`
	Maximum     *float64    `json:"maximum,omitempty"`
}

// Tool represents a tool that can be called by the AI
type Tool struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Parameters  *ToolParameterSchema `json:"parameters"`
}

// ToolArguments represents structured arguments for a tool call
type ToolArguments struct {
	// Common tool arguments
	Action     string            `json:"action,omitempty"`
	Target     string            `json:"target,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Options    *ToolOptions      `json:"options,omitempty"`
}

// ToolOptions represents optional settings for tool execution
type ToolOptions struct {
	Timeout    time.Duration `json:"timeout,omitempty"`
	MaxRetries int           `json:"max_retries,omitempty"`
	DryRun     bool          `json:"dry_run,omitempty"`
	Verbose    bool          `json:"verbose,omitempty"`
}

// ToolCall represents a tool call made by the AI
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments *ToolArguments `json:"arguments"`
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
