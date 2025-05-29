package langchain

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/schema"

	"github.com/koopa0/assistant/internal/config"
)

// Local type definitions to avoid import cycles

// Message represents a message in a conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GenerateRequest represents a request to generate a response
type GenerateRequest struct {
	Messages     []Message              `json:"messages"`
	Model        string                 `json:"model,omitempty"`
	MaxTokens    int                    `json:"max_tokens,omitempty"`
	Temperature  float64                `json:"temperature,omitempty"`
	SystemPrompt *string                `json:"system_prompt,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// TokenUsage represents token usage statistics
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// GenerateResponse represents a response from AI generation
type GenerateResponse struct {
	Content      string                 `json:"content"`
	Model        string                 `json:"model"`
	Provider     string                 `json:"provider"`
	TokensUsed   TokenUsage             `json:"tokens_used"`
	FinishReason string                 `json:"finish_reason"`
	ResponseTime time.Duration          `json:"response_time"`
	RequestID    string                 `json:"request_id"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// EmbeddingResponse represents a response from embedding generation
type EmbeddingResponse struct {
	Embedding    []float64     `json:"embedding"`
	Model        string        `json:"model"`
	Provider     string        `json:"provider"`
	TokensUsed   int           `json:"tokens_used"`
	ResponseTime time.Duration `json:"response_time"`
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

// LangChainClient wraps LangChain-Go functionality with our AI interface
type LangChainClient struct {
	llm      llms.Model
	memory   schema.Memory
	config   config.LangChain
	logger   *slog.Logger
	provider string
}

// NewLangChainClient creates a new LangChain-Go client
func NewLangChainClient(provider string, aiConfig interface{}, langchainConfig config.LangChain, logger *slog.Logger) (*LangChainClient, error) {
	var llm llms.Model
	var err error

	switch provider {
	case "claude":
		claudeConfig, ok := aiConfig.(config.Claude)
		if !ok {
			return nil, fmt.Errorf("invalid Claude configuration")
		}
		llm, err = anthropic.New(
			anthropic.WithToken(claudeConfig.APIKey),
			anthropic.WithModel(claudeConfig.Model),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Anthropic LLM: %w", err)
		}

	case "gemini":
		geminiConfig, ok := aiConfig.(config.Gemini)
		if !ok {
			return nil, fmt.Errorf("invalid Gemini configuration")
		}
		llm, err = googleai.New(
			context.Background(),
			googleai.WithAPIKey(geminiConfig.APIKey),
			googleai.WithDefaultModel(geminiConfig.Model),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Google AI LLM: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	// Initialize memory if enabled
	var mem schema.Memory
	if langchainConfig.EnableMemory {
		mem = memory.NewConversationBuffer()
	}

	return &LangChainClient{
		llm:      llm,
		memory:   mem,
		config:   langchainConfig,
		logger:   logger,
		provider: provider,
	}, nil
}

// Name returns the provider name
func (c *LangChainClient) Name() string {
	return fmt.Sprintf("langchain-%s", c.provider)
}

// GenerateResponse generates a response using LangChain-Go
func (c *LangChainClient) GenerateResponse(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	c.logger.Debug("Generating response with LangChain",
		slog.String("provider", c.provider),
		slog.String("model", request.Model),
		slog.Int("message_count", len(request.Messages)))

	// Convert messages to LangChain format
	messages := c.convertToLangChainMessages(request.Messages)

	// Add system prompt if provided (simplified for now)
	var systemPrompt string
	if request.SystemPrompt != nil && *request.SystemPrompt != "" {
		systemPrompt = *request.SystemPrompt
	}

	// Prepare generation options
	options := []llms.CallOption{
		llms.WithMaxTokens(request.MaxTokens),
		llms.WithTemperature(request.Temperature),
	}

	if request.Model != "" {
		options = append(options, llms.WithModel(request.Model))
	}

	// Prepare the input text with system prompt if provided
	inputText := c.convertMessagesToString(messages)
	if systemPrompt != "" {
		inputText = "System: " + systemPrompt + "\n" + inputText
	}

	// Generate response using simplified API
	content, err := c.llm.Call(ctx, inputText, options...)
	if err != nil {
		c.logger.Error("LangChain generation failed",
			slog.String("provider", c.provider),
			slog.Any("error", err))
		return nil, fmt.Errorf("LangChain generation failed: %w", err)
	}

	// Calculate token usage (approximate)
	tokenUsage := c.estimateTokenUsage(request.Messages, content)

	aiResponse := &GenerateResponse{
		Content:      content,
		Model:        c.getModelFromResponse(content),
		Provider:     c.Name(),
		TokensUsed:   tokenUsage,
		FinishReason: c.getFinishReason(content),
		ResponseTime: time.Since(startTime),
		RequestID:    c.generateRequestID(),
		Metadata:     request.Metadata,
	}

	c.logger.Debug("LangChain response generated",
		slog.String("provider", c.provider),
		slog.Int("response_length", len(content)),
		slog.Duration("response_time", aiResponse.ResponseTime))

	return aiResponse, nil
}

// GenerateEmbedding generates embeddings (not directly supported by LangChain-Go core)
func (c *LangChainClient) GenerateEmbedding(ctx context.Context, text string) (*EmbeddingResponse, error) {
	// LangChain-Go doesn't have direct embedding support in the core
	// This would need to be implemented using specific embedding models
	return nil, fmt.Errorf("embedding generation not implemented for LangChain provider")
}

// Health checks if the LangChain client is healthy
func (c *LangChainClient) Health(ctx context.Context) error {
	// Simple health check by generating a minimal response
	_, err := c.llm.Call(ctx, "Hello", llms.WithMaxTokens(1))
	if err != nil {
		return fmt.Errorf("LangChain health check failed: %w", err)
	}

	return nil
}

// Close closes the LangChain client
func (c *LangChainClient) Close(ctx context.Context) error {
	// LangChain-Go doesn't require explicit cleanup
	return nil
}

// GetUsage returns usage statistics
func (c *LangChainClient) GetUsage(ctx context.Context) (*UsageStats, error) {
	// LangChain-Go doesn't provide built-in usage tracking
	// This would need to be implemented separately
	return &UsageStats{
		TotalRequests:   0,
		TotalTokens:     0,
		InputTokens:     0,
		OutputTokens:    0,
		TotalCost:       0,
		AverageLatency:  0,
		ErrorRate:       0,
		LastRequestTime: nil,
		RequestsPerHour: 0,
	}, nil
}

// Helper methods

func (c *LangChainClient) convertToLangChainMessages(messages []Message) []Message {
	// For now, just return the messages as-is since we're using the simplified API
	return messages
}

func (c *LangChainClient) convertMessagesToString(messages []Message) string {
	// Convert messages to a simple string format for the Call API
	var result string
	for _, msg := range messages {
		switch msg.Role {
		case "user", "human":
			result += "Human: " + msg.Content + "\n"
		case "assistant", "ai":
			result += "Assistant: " + msg.Content + "\n"
		case "system":
			result += "System: " + msg.Content + "\n"
		default:
			result += "Human: " + msg.Content + "\n"
		}
	}
	return result
}

func (c *LangChainClient) estimateTokenUsage(messages []Message, response string) TokenUsage {
	// Simple token estimation (4 characters ≈ 1 token for English)
	inputChars := 0
	for _, msg := range messages {
		inputChars += len(msg.Content)
	}

	inputTokens := inputChars / 4
	outputTokens := len(response) / 4
	totalTokens := inputTokens + outputTokens

	return TokenUsage{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
	}
}

func (c *LangChainClient) getModelFromResponse(response string) string {
	// Since we're using the simplified Call API, we don't have response metadata
	// Return the configured model or a default
	return fmt.Sprintf("%s-model", c.provider)
}

func (c *LangChainClient) getFinishReason(response string) string {
	// Since we're using the simplified Call API, we assume completion
	return "stop"
}

func (c *LangChainClient) generateRequestID() string {
	return fmt.Sprintf("langchain-%s-%d", c.provider, time.Now().UnixNano())
}

// GetMemory returns the conversation memory if enabled
func (c *LangChainClient) GetMemory() schema.Memory {
	return c.memory
}

// ClearMemory clears the conversation memory
func (c *LangChainClient) ClearMemory(ctx context.Context) error {
	if c.memory != nil {
		c.memory.Clear(ctx)
	}
	return nil
}
