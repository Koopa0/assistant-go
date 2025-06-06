package gemini

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
	Content      string                 `json:"content"`
	Model        string                 `json:"model"`
	Provider     string                 `json:"provider"`
	TokensUsed   TokenUsage             `json:"tokens_used"`
	FinishReason string                 `json:"finish_reason"`
	ResponseTime time.Duration          `json:"response_time"`
	RequestID    string                 `json:"request_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
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

// Client represents a Gemini API client
type Client struct {
	config     ProviderConfig
	httpClient *http.Client
	logger     *slog.Logger
	stats      *UsageStats
	statsMutex sync.RWMutex
}

// Content represents content in Gemini format
type Content struct {
	Parts []Part `json:"parts"`
	Role  string `json:"role,omitempty"`
}

// Part represents a part of content
type Part struct {
	Text string `json:"text"`
}

// APIRequest represents a request to Gemini API
type APIRequest struct {
	Contents          []Content         `json:"contents"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings    []SafetySetting   `json:"safetySettings,omitempty"`
	SystemInstruction *Content          `json:"systemInstruction,omitempty"`
}

// GenerationConfig represents generation configuration
type GenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// SafetySetting represents safety settings
type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// APIResponse represents a response from Gemini API
type APIResponse struct {
	Candidates     []Candidate     `json:"candidates"`
	UsageMetadata  *Usage          `json:"usageMetadata,omitempty"`
	PromptFeedback *PromptFeedback `json:"promptFeedback,omitempty"`
}

// Candidate represents a candidate response
type Candidate struct {
	Content       Content        `json:"content"`
	FinishReason  string         `json:"finishReason"`
	Index         int            `json:"index"`
	SafetyRatings []SafetyRating `json:"safetyRatings,omitempty"`
}

// Usage represents usage information from Gemini
type Usage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// PromptFeedback represents feedback about the prompt
type PromptFeedback struct {
	BlockReason   string         `json:"blockReason,omitempty"`
	SafetyRatings []SafetyRating `json:"safetyRatings,omitempty"`
}

// SafetyRating represents a safety rating
type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

// APIError represents an error from Gemini API
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// APIErrorResponse represents an error response from Gemini API
type APIErrorResponse struct {
	Error APIError `json:"error"`
}

// NewClient creates a new Gemini client
func NewClient(config ProviderConfig, logger *slog.Logger) (*Client, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Gemini API key is required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://generativelanguage.googleapis.com"
	}

	if config.Model == "" {
		config.Model = "gemini-pro"
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
		logger:     observability.AILogger(logger, "gemini", config.Model),
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
	return "gemini"
}

// GenerateResponse generates a response using Gemini API
func (c *Client) GenerateResponse(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	c.logger.Debug("Generating response with Gemini",
		slog.Int("message_count", len(request.Messages)),
		slog.String("model", c.getModel(request.Model)))

	// Convert messages to Gemini format
	contents := make([]Content, 0, len(request.Messages))
	var systemInstruction *Content

	for _, msg := range request.Messages {
		if msg.Role == "system" {
			// Gemini handles system messages as system instruction
			systemInstruction = &Content{
				Parts: []Part{{Text: msg.Content}},
			}
		} else {
			role := msg.Role
			if role == "assistant" {
				role = "model" // Gemini uses "model" instead of "assistant"
			}

			contents = append(contents, Content{
				Parts: []Part{{Text: msg.Content}},
				Role:  role,
			})
		}
	}

	// Use system prompt from request if provided
	if request.SystemPrompt != nil {
		systemInstruction = &Content{
			Parts: []Part{{Text: *request.SystemPrompt}},
		}
	}

	// Prepare generation config
	var genConfig *GenerationConfig
	if request.Temperature > 0 || request.MaxTokens > 0 {
		genConfig = &GenerationConfig{}
		if request.Temperature > 0 {
			genConfig.Temperature = &request.Temperature
		}
		if request.MaxTokens > 0 {
			maxTokens := c.getMaxTokens(request.MaxTokens)
			genConfig.MaxOutputTokens = &maxTokens
		}
	}

	// Prepare Gemini request
	apiReq := APIRequest{
		Contents:          contents,
		GenerationConfig:  genConfig,
		SystemInstruction: systemInstruction,
		SafetySettings:    c.getDefaultSafetySettings(),
	}

	// Make API request
	response, err := c.makeRequest(ctx, apiReq)
	if err != nil {
		c.updateErrorStats()
		return nil, err
	}

	// Extract content
	content := ""
	finishReason := "unknown"
	if len(response.Candidates) > 0 {
		candidate := response.Candidates[0]
		if len(candidate.Content.Parts) > 0 {
			content = candidate.Content.Parts[0].Text
		}
		finishReason = candidate.FinishReason
	}

	// Calculate response time
	responseTime := time.Since(startTime)

	// Extract token usage
	var tokenUsage TokenUsage
	if response.UsageMetadata != nil {
		tokenUsage = TokenUsage{
			InputTokens:  response.UsageMetadata.PromptTokenCount,
			OutputTokens: response.UsageMetadata.CandidatesTokenCount,
			TotalTokens:  response.UsageMetadata.TotalTokenCount,
		}
	}

	// Update statistics
	c.updateStats(tokenUsage.InputTokens, tokenUsage.OutputTokens, responseTime)

	// Build response
	aiResponse := &GenerateResponse{
		Content:      content,
		Model:        c.getModel(request.Model),
		Provider:     "gemini",
		TokensUsed:   tokenUsage,
		FinishReason: finishReason,
		ResponseTime: responseTime,
		Metadata: map[string]interface{}{
			"candidates_count": len(response.Candidates),
		},
	}

	c.logger.Debug("Gemini response generated",
		slog.Int("input_tokens", tokenUsage.InputTokens),
		slog.Int("output_tokens", tokenUsage.OutputTokens),
		slog.Duration("response_time", responseTime))

	return aiResponse, nil
}

// GenerateEmbedding generates embeddings using Gemini API
func (c *Client) GenerateEmbedding(ctx context.Context, text string) (*EmbeddingResponse, error) {
	startTime := time.Now()

	c.logger.Debug("Generating embedding with Gemini", slog.String("text_length", fmt.Sprintf("%d", len(text))))

	// Prepare embedding request
	reqBody := map[string]interface{}{
		"model": "models/embedding-001",
		"content": map[string]interface{}{
			"parts": []map[string]string{
				{"text": text},
			},
		},
	}

	// Make API request
	url := fmt.Sprintf("%s/v1beta/models/embedding-001:embedContent?key=%s", c.config.BaseURL, c.config.APIKey)

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, NewProviderError(ErrorTypeInvalidRequest,
			fmt.Sprintf("failed to marshal request: %v", err), "gemini")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return nil, NewProviderError(ErrorTypeNetworkError,
			fmt.Sprintf("failed to create request: %v", err), "gemini")
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, NewProviderError(ErrorTypeNetworkError,
			fmt.Sprintf("request failed: %v", err), "gemini")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewProviderError(ErrorTypeNetworkError,
			fmt.Sprintf("failed to read response: %v", err), "gemini")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	// Parse embedding response
	var embeddingResp struct {
		Embedding struct {
			Values []float64 `json:"values"`
		} `json:"embedding"`
	}

	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, NewProviderError(ErrorTypeServerError,
			fmt.Sprintf("failed to parse response: %v", err), "gemini")
	}

	responseTime := time.Since(startTime)

	return &EmbeddingResponse{
		Embedding:    embeddingResp.Embedding.Values,
		Model:        "embedding-001",
		Provider:     "gemini",
		TokensUsed:   c.estimateTokens(text),
		ResponseTime: responseTime,
	}, nil
}

// Health checks if Gemini API is accessible
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
		return fmt.Errorf("Gemini health check failed: %w", err)
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

// makeRequest makes an HTTP request to Gemini API
func (c *Client) makeRequest(ctx context.Context, request APIRequest) (*APIResponse, error) {
	// Marshal request
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, NewProviderError(ErrorTypeInvalidRequest,
			fmt.Sprintf("failed to marshal request: %v", err), "gemini")
	}

	// Create HTTP request
	model := c.getModel("")
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", c.config.BaseURL, model, c.config.APIKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, NewProviderError(ErrorTypeNetworkError,
			fmt.Sprintf("failed to create request: %v", err), "gemini")
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, NewProviderError(ErrorTypeNetworkError,
			fmt.Sprintf("request failed: %v", err), "gemini")
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewProviderError(ErrorTypeNetworkError,
			fmt.Sprintf("failed to read response: %v", err), "gemini")
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, body)
	}

	// Parse successful response
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, NewProviderError(ErrorTypeServerError,
			fmt.Sprintf("failed to parse response: %v", err), "gemini")
	}

	return &apiResp, nil
}

// handleErrorResponse handles error responses from Gemini API
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	var errorResp APIErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		return NewProviderError(ErrorTypeServerError,
			fmt.Sprintf("HTTP %d: failed to parse error response", statusCode), "gemini")
	}

	errorType := ErrorTypeServerError
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		errorType = ErrorTypeAuthentication
	case http.StatusTooManyRequests:
		errorType = ErrorTypeRateLimit
	case http.StatusBadRequest:
		errorType = ErrorTypeInvalidRequest
	case http.StatusRequestTimeout:
		errorType = ErrorTypeTimeout
	}

	return NewProviderError(errorType, errorResp.Error.Message, "gemini")
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

// getDefaultSafetySettings returns default safety settings
func (c *Client) getDefaultSafetySettings() []SafetySetting {
	return []SafetySetting{
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_MEDIUM_AND_ABOVE"},
	}
}

// estimateTokens estimates token count for text
func (c *Client) estimateTokens(text string) int {
	// Rough estimation: ~4 characters per token
	return len(text) / 4
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
}
