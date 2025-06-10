package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/koopa0/assistant-go/internal/platform/observability"
)

// ProviderConfig represents configuration for an AI provider
type ProviderConfig struct {
	APIKey      string        `json:"api_key"`
	BaseURL     string        `json:"base_url"`
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Timeout     time.Duration `json:"timeout"`
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
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// GenerateResponse represents a response from the AI provider
type GenerateResponse struct {
	Content      string         `json:"content"`
	Model        string         `json:"model"`
	Provider     string         `json:"provider"`
	TokensUsed   TokenUsage     `json:"tokens_used"`
	FinishReason string         `json:"finish_reason"`
	ResponseTime time.Duration  `json:"response_time"`
	RequestID    string         `json:"request_id,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
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

// Client represents a Claude API client
type Client struct {
	config     ProviderConfig
	httpClient *http.Client
	logger     *slog.Logger
	stats      *UsageStats
	statsMutex sync.RWMutex
}

// APIMessage represents a message in Claude format
type APIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// APIRequest represents a request to Claude API
type APIRequest struct {
	Model       string       `json:"model"`
	MaxTokens   int          `json:"max_tokens"`
	Messages    []APIMessage `json:"messages"`
	Temperature *float64     `json:"temperature,omitempty"`
	System      *string      `json:"system,omitempty"`
}

// APIResponse represents a response from Claude API
type APIResponse struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Role         string    `json:"role"`
	Content      []Content `json:"content"`
	Model        string    `json:"model"`
	StopReason   string    `json:"stop_reason"`
	StopSequence *string   `json:"stop_sequence"`
	Usage        Usage     `json:"usage"`
}

// Content represents content in Claude response
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Usage represents usage information from Claude
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// APIError represents an error from Claude API
type APIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// APIErrorResponse represents an error response from Claude API
type APIErrorResponse struct {
	Type  string   `json:"type"`
	Error APIError `json:"error"`
}

// NewClient creates a new Claude client
func NewClient(config ProviderConfig, logger *slog.Logger) (*Client, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Claude API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}

	if config.Model == "" {
		config.Model = "claude-3-sonnet-20240229"
	}

	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
		logger:     observability.AILogger(logger, "claude", config.Model),
		stats: &UsageStats{
			TotalRequests:   0,
			TotalTokens:     0,
			InputTokens:     0,
			OutputTokens:    0,
			TotalCost:       0,
			AverageLatency:  0,
			ErrorRate:       0,
			RequestsPerHour: 0,
		},
	}, nil
}

// Name returns the provider name
func (c *Client) Name() string {
	return "claude"
}

// GenerateResponse generates a response using Claude API
func (c *Client) GenerateResponse(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	c.logger.Debug("Generating response with Claude",
		slog.Int("message_count", len(request.Messages)),
		slog.String("model", c.getModel(request.Model)))

	// Convert messages to Claude format
	apiMessages := make([]APIMessage, 0, len(request.Messages))
	var systemPrompt *string

	for _, msg := range request.Messages {
		if msg.Role == "system" {
			// Claude handles system messages separately
			systemPrompt = &msg.Content
		} else {
			apiMessages = append(apiMessages, APIMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	// Use system prompt from request if provided
	if request.SystemPrompt != nil {
		systemPrompt = request.SystemPrompt
	}

	// Prepare Claude request
	apiReq := APIRequest{
		Model:     c.getModel(request.Model),
		MaxTokens: c.getMaxTokens(request.MaxTokens),
		Messages:  apiMessages,
		System:    systemPrompt,
	}

	if request.Temperature > 0 {
		apiReq.Temperature = &request.Temperature
	}

	// Make API request
	response, err := c.makeRequest(ctx, apiReq)
	if err != nil {
		c.updateErrorStats()
		return nil, err
	}

	// Extract content
	content := ""
	if len(response.Content) > 0 && response.Content[0].Type == "text" {
		content = response.Content[0].Text
	}

	// Calculate response time
	responseTime := time.Since(startTime)

	// Update statistics
	c.updateStats(response.Usage.InputTokens, response.Usage.OutputTokens, responseTime)

	// Build response
	aiResponse := &GenerateResponse{
		Content:  content,
		Model:    response.Model,
		Provider: "claude",
		TokensUsed: TokenUsage{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
			TotalTokens:  response.Usage.InputTokens + response.Usage.OutputTokens,
		},
		FinishReason: response.StopReason,
		ResponseTime: responseTime,
		RequestID:    response.ID,
		Metadata: map[string]interface{}{
			"stop_sequence": response.StopSequence,
		},
	}

	c.logger.Debug("Claude response generated",
		slog.String("request_id", response.ID),
		slog.Int("input_tokens", response.Usage.InputTokens),
		slog.Int("output_tokens", response.Usage.OutputTokens),
		slog.Duration("response_time", responseTime))

	return aiResponse, nil
}

// GenerateEmbedding generates embeddings (Claude doesn't support embeddings directly)
func (c *Client) GenerateEmbedding(ctx context.Context, text string) (*EmbeddingResponse, error) {
	return nil, NewProviderError(ErrorTypeInvalidRequest,
		"Claude does not support embedding generation", "claude")
}

// Health checks if Claude API is accessible
func (c *Client) Health(ctx context.Context) error {
	// Simple health check with a minimal request
	request := &GenerateRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 10,
	}

	_, err := c.GenerateResponse(ctx, request)
	if err != nil {
		return fmt.Errorf("Claude health check failed: %w", err)
	}

	return nil
}

// Close closes the client
func (c *Client) Close(ctx context.Context) error {
	// No cleanup needed for HTTP client
	return nil
}

// GetUsage returns usage statistics
func (c *Client) GetUsage(ctx context.Context) (*UsageStats, error) {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()

	// Return a copy of the stats
	stats := *c.stats
	return &stats, nil
}

// makeRequest makes an HTTP request to Claude API
func (c *Client) makeRequest(ctx context.Context, request APIRequest) (*APIResponse, error) {
	// Marshal request
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, NewProviderError(ErrorTypeInvalidRequest,
			fmt.Sprintf("failed to marshal request: %v", err), "claude")
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1/messages", c.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, NewProviderError(ErrorTypeNetworkError,
			fmt.Sprintf("failed to create request: %v", err), "claude")
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Make request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, NewProviderError(ErrorTypeNetworkError,
			fmt.Sprintf("request failed: %v", err), "claude")
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewProviderError(ErrorTypeNetworkError,
			fmt.Sprintf("failed to read response: %v", err), "claude")
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	// Parse successful response
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, NewProviderError(ErrorTypeServerError,
			fmt.Sprintf("failed to parse response: %v", err), "claude")
	}

	return &apiResp, nil
}

// handleErrorResponse handles error responses from Claude API
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	var errorResp APIErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		return NewProviderError(ErrorTypeServerError,
			fmt.Sprintf("HTTP %d: failed to parse error response", statusCode), "claude")
	}

	errorType := ErrorTypeServerError
	switch statusCode {
	case http.StatusUnauthorized:
		errorType = ErrorTypeAuthentication
	case http.StatusTooManyRequests:
		errorType = ErrorTypeRateLimit
	case http.StatusBadRequest:
		errorType = ErrorTypeInvalidRequest
	case http.StatusPaymentRequired:
		errorType = ErrorTypeQuotaExceeded
	case http.StatusRequestTimeout:
		errorType = ErrorTypeTimeout
	}

	return NewProviderError(errorType, errorResp.Error.Message, "claude")
}

// getModel returns the model to use
func (c *Client) getModel(requestModel string) string {
	if requestModel != "" {
		return requestModel
	}
	return c.config.Model
}

// getMaxTokens returns the max tokens to use
func (c *Client) getMaxTokens(requestMaxTokens int) int {
	if requestMaxTokens > 0 {
		return requestMaxTokens
	}
	return c.config.MaxTokens
}

// updateStats updates usage statistics
func (c *Client) updateStats(inputTokens, outputTokens int, responseTime time.Duration) {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()

	c.stats.TotalRequests++
	c.stats.InputTokens += int64(inputTokens)
	c.stats.OutputTokens += int64(outputTokens)
	c.stats.TotalTokens += int64(inputTokens + outputTokens)

	// Update average latency
	if c.stats.TotalRequests == 1 {
		c.stats.AverageLatency = responseTime
	} else {
		c.stats.AverageLatency = time.Duration(
			(int64(c.stats.AverageLatency)*(c.stats.TotalRequests-1) + int64(responseTime)) / c.stats.TotalRequests,
		)
	}

	now := time.Now()
	c.stats.LastRequestTime = &now
}

// updateErrorStats updates error statistics
func (c *Client) updateErrorStats() {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()

	c.stats.TotalRequests++
	// Error rate calculation would need more sophisticated tracking
}
