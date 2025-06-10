package testutil

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/koopa0/assistant-go/internal/ai"
)

// MockAIManager provides a mock implementation of AI manager for testing
type MockAIManager struct {
	mu                    sync.RWMutex // 保護共享狀態
	logger                *slog.Logger
	responses             map[string]*ai.GenerateResponse
	embeddings            map[string]*ai.EmbeddingResponse
	generateResponseFunc  func(ctx context.Context, request *ai.GenerateRequest, providerName ...string) (*ai.GenerateResponse, error)
	generateEmbeddingFunc func(ctx context.Context, text string, providerName ...string) (*ai.EmbeddingResponse, error)
	healthFunc            func(ctx context.Context) error
	usageStats            map[string]*ai.UsageStats
	availableProviders    []string
	defaultProvider       string
	simulateLatency       bool
	simulateErrors        bool
	errorRate             float64
}

// NewMockAIManager creates a new mock AI manager
func NewMockAIManager(logger *slog.Logger) *MockAIManager {
	return &MockAIManager{
		logger:             logger,
		responses:          make(map[string]*ai.GenerateResponse),
		embeddings:         make(map[string]*ai.EmbeddingResponse),
		usageStats:         make(map[string]*ai.UsageStats),
		availableProviders: []string{"mock-claude", "mock-gemini"},
		defaultProvider:    "mock-claude",
		simulateLatency:    true,
		simulateErrors:     false,
		errorRate:          0.0,
	}
}

// SetResponse sets a predefined response for a specific prompt
func (m *MockAIManager) SetResponse(prompt string, response *ai.GenerateResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[prompt] = response
}

// SetEmbedding sets a predefined embedding for specific text
func (m *MockAIManager) SetEmbedding(text string, embedding *ai.EmbeddingResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embeddings[text] = embedding
}

// SetCustomResponseFunc sets a custom function for generating responses
func (m *MockAIManager) SetCustomResponseFunc(fn func(ctx context.Context, request *ai.GenerateRequest, providerName ...string) (*ai.GenerateResponse, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generateResponseFunc = fn
}

// SetCustomEmbeddingFunc sets a custom function for generating embeddings
func (m *MockAIManager) SetCustomEmbeddingFunc(fn func(ctx context.Context, text string, providerName ...string) (*ai.EmbeddingResponse, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generateEmbeddingFunc = fn
}

// EnableErrorSimulation enables error simulation with specified rate
func (m *MockAIManager) EnableErrorSimulation(rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateErrors = true
	m.errorRate = rate
}

// DisableErrorSimulation disables error simulation
func (m *MockAIManager) DisableErrorSimulation() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateErrors = false
	m.errorRate = 0.0
}

// SetLatencySimulation enables/disables latency simulation
func (m *MockAIManager) SetLatencySimulation(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateLatency = enabled
}

// GenerateResponse implements ai.Manager interface
func (m *MockAIManager) GenerateResponse(ctx context.Context, request *ai.GenerateRequest, providerName ...string) (*ai.GenerateResponse, error) {
	// 讀取配置（使用 RLock）
	m.mu.RLock()
	simulateLatency := m.simulateLatency
	simulateErrors := m.simulateErrors
	errorRate := m.errorRate
	generateResponseFunc := m.generateResponseFunc
	defaultProvider := m.defaultProvider
	m.mu.RUnlock()

	// Simulate latency
	if simulateLatency {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(rand.Intn(100)) * time.Millisecond):
		}
	}

	// Simulate errors
	if simulateErrors && rand.Float64() < errorRate {
		return nil, fmt.Errorf("simulated AI provider error")
	}

	// Use custom function if set
	if generateResponseFunc != nil {
		return generateResponseFunc(ctx, request, providerName...)
	}

	// Extract the content from messages for prompt matching
	prompt := m.extractPromptFromMessages(request.Messages)

	// Check for predefined response（再次使用 RLock）
	m.mu.RLock()
	response, exists := m.responses[prompt]
	m.mu.RUnlock()

	if exists {
		return response, nil
	}

	// Generate default mock response
	provider := defaultProvider
	if len(providerName) > 0 {
		provider = providerName[0]
	}

	// Simple token estimation
	estimatedTokens := len(prompt)/4 + rand.Intn(100)
	response = &ai.GenerateResponse{
		Content:  m.generateMockContent(prompt),
		Model:    fmt.Sprintf("%s-model", provider),
		Provider: provider,
		TokensUsed: ai.TokenUsage{
			InputTokens:  estimatedTokens / 2,
			OutputTokens: estimatedTokens / 2,
			TotalTokens:  estimatedTokens,
		},
		ResponseTime: time.Duration(rand.Intn(200)) * time.Millisecond,
		Metadata: &ai.ResponseMetadata{
			ProcessingTime: time.Duration(rand.Intn(200)) * time.Millisecond,
			Provider:       provider,
			Model:          fmt.Sprintf("%s-model", provider),
			Debug: &ai.DebugInfo{
				PromptTokens:     estimatedTokens / 2,
				CompletionTokens: estimatedTokens / 2,
			},
		},
	}

	// Update usage stats
	m.updateUsageStats(provider, response.TokensUsed.TotalTokens)

	return response, nil
}

// extractPromptFromMessages extracts a prompt string from messages for matching
func (m *MockAIManager) extractPromptFromMessages(messages []ai.Message) string {
	if len(messages) == 0 {
		return ""
	}

	// Use the last user message as the prompt
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}

	// Fallback to the last message content
	return messages[len(messages)-1].Content
}

// GenerateEmbedding implements ai.Manager interface
func (m *MockAIManager) GenerateEmbedding(ctx context.Context, text string, providerName ...string) (*ai.EmbeddingResponse, error) {
	// Simulate latency
	if m.simulateLatency {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(rand.Intn(50)) * time.Millisecond):
		}
	}

	// Simulate errors
	if m.simulateErrors && rand.Float64() < m.errorRate {
		return nil, fmt.Errorf("simulated embedding provider error")
	}

	// Use custom function if set
	if m.generateEmbeddingFunc != nil {
		return m.generateEmbeddingFunc(ctx, text, providerName...)
	}

	// Check for predefined embedding
	if embedding, exists := m.embeddings[text]; exists {
		return embedding, nil
	}

	// Generate default mock embedding
	provider := m.defaultProvider
	if len(providerName) > 0 {
		provider = providerName[0]
	}

	embedding := &ai.EmbeddingResponse{
		Embedding:    m.generateMockEmbedding(text),
		Model:        fmt.Sprintf("%s-embedding-model", provider),
		Provider:     provider,
		TokensUsed:   len(text) / 4, // Simple token estimation
		ResponseTime: time.Duration(rand.Intn(100)) * time.Millisecond,
	}

	return embedding, nil
}

// Health implements ai.Manager interface
func (m *MockAIManager) Health(ctx context.Context) error {
	if m.healthFunc != nil {
		return m.healthFunc(ctx)
	}
	return nil
}

// GetUsageStats implements ai.Manager interface
func (m *MockAIManager) GetUsageStats(ctx context.Context) (map[string]*ai.UsageStats, error) {
	return m.usageStats, nil
}

// GetAvailableProviders implements ai.Manager interface
func (m *MockAIManager) GetAvailableProviders() []string {
	return m.availableProviders
}

// GetDefaultProvider implements ai.Manager interface
func (m *MockAIManager) GetDefaultProvider() string {
	return m.defaultProvider
}

// Close implements ai.Manager interface
func (m *MockAIManager) Close(ctx context.Context) error {
	return nil
}

// generateMockContent generates mock AI response content
func (m *MockAIManager) generateMockContent(prompt string) string {
	// Simple mock content generation based on prompt
	if strings.Contains(strings.ToLower(prompt), "code") {
		return "```go\nfunc MockFunction() {\n    // Mock implementation\n}\n```"
	}
	if strings.Contains(strings.ToLower(prompt), "test") {
		return "Here's a mock test implementation for your code."
	}
	if strings.Contains(strings.ToLower(prompt), "explain") {
		return "This is a mock explanation of the requested concept."
	}
	return fmt.Sprintf("Mock response for: %s", prompt)
}

// generateMockEmbedding generates a mock embedding vector
func (m *MockAIManager) generateMockEmbedding(text string) []float64 {
	// Generate deterministic embedding based on text hash
	hash := 0
	for _, char := range text {
		hash = hash*31 + int(char)
	}

	rand.Seed(int64(hash))
	embedding := make([]float64, 1536) // Standard embedding dimension

	for i := range embedding {
		embedding[i] = rand.Float64()*2 - 1 // Values between -1 and 1
	}

	return embedding
}

// updateUsageStats updates usage statistics
func (m *MockAIManager) updateUsageStats(provider string, tokens int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if stats, exists := m.usageStats[provider]; exists {
		stats.TotalRequests++
		stats.TotalTokens += int64(tokens)
	} else {
		now := time.Now()
		m.usageStats[provider] = &ai.UsageStats{
			TotalRequests:   1,
			TotalTokens:     int64(tokens),
			LastRequestTime: &now,
		}
	}
}

// ResetStats resets usage statistics
func (m *MockAIManager) ResetStats() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usageStats = make(map[string]*ai.UsageStats)
}

// GetRequestCount returns the total number of requests made
func (m *MockAIManager) GetRequestCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for _, stats := range m.usageStats {
		total += stats.TotalRequests
	}
	return total
}

// GetTokenCount returns the total number of tokens used
func (m *MockAIManager) GetTokenCount() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for _, stats := range m.usageStats {
		total += stats.TotalTokens
	}
	return total
}
