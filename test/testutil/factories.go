package testutil

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/koopa0/assistant-go/internal/ai"
	"github.com/koopa0/assistant-go/internal/assistant"
)

// TestDataFactory provides factory methods for creating test data
type TestDataFactory struct {
	rand *rand.Rand
}

// NewTestDataFactory creates a new test data factory
func NewTestDataFactory() *TestDataFactory {
	return &TestDataFactory{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// CreateAssistantRequest creates a test assistant request
func (f *TestDataFactory) CreateAssistantRequest(options ...func(*assistant.QueryRequest)) *assistant.QueryRequest {
	userID := f.GenerateUserID()
	req := &assistant.QueryRequest{
		Query:  "Test message",
		UserID: &userID,
		Context: map[string]any{
			"test": true,
		},
	}

	for _, option := range options {
		option(req)
	}

	return req
}

// CreateContext creates a test context map
func (f *TestDataFactory) CreateContext() map[string]any {
	return map[string]any{
		"user_id":      f.GenerateUserID(),
		"session_id":   f.GenerateSessionID(),
		"workspace_id": f.GenerateWorkspaceID(),
		"timestamp":    time.Now(),
		"test_context": true,
	}
}

// CreateAIGenerateRequest creates a test AI generate request
func (f *TestDataFactory) CreateAIGenerateRequest(options ...func(*ai.GenerateRequest)) *ai.GenerateRequest {
	systemPrompt := "You are a helpful assistant for testing."
	req := &ai.GenerateRequest{
		Messages: []ai.Message{
			{Role: "user", Content: "Test prompt"},
		},
		SystemPrompt: &systemPrompt,
		MaxTokens:    1000,
		Temperature:  0.7,
		Metadata: &ai.RequestMetadata{
			RequestID: f.GenerateID(),
			UserID:    "test-user",
			SessionID: "test-session",
			Tags:      []string{"test"},
			Features:  map[string]string{"test": "true"},
		},
	}

	for _, option := range options {
		option(req)
	}

	return req
}

// CreateAIGenerateResponse creates a test AI generate response
func (f *TestDataFactory) CreateAIGenerateResponse(options ...func(*ai.GenerateResponse)) *ai.GenerateResponse {
	resp := &ai.GenerateResponse{
		Content:  "Test response content",
		Model:    "test-model",
		Provider: "test-provider",
		TokensUsed: ai.TokenUsage{
			InputTokens:  f.rand.Intn(200) + 50,
			OutputTokens: f.rand.Intn(300) + 50,
			TotalTokens:  f.rand.Intn(500) + 100,
		},
		ResponseTime: time.Duration(f.rand.Intn(1000)) * time.Millisecond,
		Metadata: &ai.ResponseMetadata{
			ProcessingTime: time.Duration(f.rand.Intn(1000)) * time.Millisecond,
			Provider:       "test-provider",
			Model:          "test-model",
			Debug: &ai.DebugInfo{
				PromptTokens:     f.rand.Intn(200) + 50,
				CompletionTokens: f.rand.Intn(300) + 50,
			},
		},
	}

	for _, option := range options {
		option(resp)
	}

	return resp
}

// CreateEmbeddingResponse creates a test embedding response
func (f *TestDataFactory) CreateEmbeddingResponse(options ...func(*ai.EmbeddingResponse)) *ai.EmbeddingResponse {
	resp := &ai.EmbeddingResponse{
		Embedding:    f.GenerateEmbedding(1536),
		Model:        "test-embedding-model",
		Provider:     "test-provider",
		TokensUsed:   f.rand.Intn(100) + 10,
		ResponseTime: time.Duration(f.rand.Intn(500)) * time.Millisecond,
	}

	for _, option := range options {
		option(resp)
	}

	return resp
}

// CreateAssistantResponse creates a test assistant response
func (f *TestDataFactory) CreateAssistantResponse(options ...func(*assistant.QueryResponse)) *assistant.QueryResponse {
	resp := &assistant.QueryResponse{
		Response:       "Test response content",
		ConversationID: f.GenerateID(),
		MessageID:      f.GenerateID(),
		Provider:       "claude",
		Model:          "claude-3-sonnet-20240229",
		TokensUsed:     100,
		ExecutionTime:  time.Millisecond * 500,
		Context: map[string]interface{}{
			"test": true,
		},
	}

	for _, option := range options {
		option(resp)
	}

	return resp
}

// GenerateID generates a random ID for testing
func (f *TestDataFactory) GenerateID() string {
	return fmt.Sprintf("test-%d-%d", time.Now().UnixNano(), f.rand.Intn(10000))
}

// GenerateUserID generates a random user ID for testing
func (f *TestDataFactory) GenerateUserID() string {
	return fmt.Sprintf("user-%d", f.rand.Intn(1000))
}

// GenerateSessionID generates a random session ID for testing
func (f *TestDataFactory) GenerateSessionID() string {
	return fmt.Sprintf("session-%d", f.rand.Intn(10000))
}

// GenerateWorkspaceID generates a random workspace ID for testing
func (f *TestDataFactory) GenerateWorkspaceID() string {
	return fmt.Sprintf("workspace-%d", f.rand.Intn(1000))
}

// GenerateEmbedding generates a random embedding vector for testing
func (f *TestDataFactory) GenerateEmbedding(dimensions int) []float64 {
	embedding := make([]float64, dimensions)
	for i := range embedding {
		embedding[i] = f.rand.Float64()*2 - 1 // Values between -1 and 1
	}
	return embedding
}

// GenerateRandomString generates a random string of specified length
func (f *TestDataFactory) GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[f.rand.Intn(len(charset))]
	}
	return string(b)
}

// WithQuery sets the query for an assistant request
func WithQuery(query string) func(*assistant.QueryRequest) {
	return func(req *assistant.QueryRequest) {
		req.Query = query
	}
}

// WithUserID sets the user ID for an assistant request
func WithUserID(userID string) func(*assistant.QueryRequest) {
	return func(req *assistant.QueryRequest) {
		req.UserID = &userID
	}
}

// WithMessages sets the messages for an AI generate request
func WithMessages(messages []ai.Message) func(*ai.GenerateRequest) {
	return func(req *ai.GenerateRequest) {
		req.Messages = messages
	}
}

// WithSystemPrompt sets the system prompt for an AI generate request
func WithSystemPrompt(systemPrompt string) func(*ai.GenerateRequest) {
	return func(req *ai.GenerateRequest) {
		req.SystemPrompt = &systemPrompt
	}
}

// WithMaxTokens sets the max tokens for an AI generate request
func WithMaxTokens(maxTokens int) func(*ai.GenerateRequest) {
	return func(req *ai.GenerateRequest) {
		req.MaxTokens = maxTokens
	}
}

// WithTemperature sets the temperature for an AI generate request
func WithTemperature(temperature float64) func(*ai.GenerateRequest) {
	return func(req *ai.GenerateRequest) {
		req.Temperature = temperature
	}
}

// WithContent sets the content for an AI generate response
func WithContent(content string) func(*ai.GenerateResponse) {
	return func(resp *ai.GenerateResponse) {
		resp.Content = content
	}
}

// WithModel sets the model for an AI response
func WithModel(model string) func(*ai.GenerateResponse) {
	return func(resp *ai.GenerateResponse) {
		resp.Model = model
	}
}

// WithProvider sets the provider for an AI response
func WithProvider(provider string) func(*ai.GenerateResponse) {
	return func(resp *ai.GenerateResponse) {
		resp.Provider = provider
	}
}

// WithTokensUsed sets the tokens used for an AI response
func WithTokensUsed(tokens ai.TokenUsage) func(*ai.GenerateResponse) {
	return func(resp *ai.GenerateResponse) {
		resp.TokensUsed = tokens
	}
}

// WithEmbedding sets the embedding for an embedding response
func WithEmbedding(embedding []float64) func(*ai.EmbeddingResponse) {
	return func(resp *ai.EmbeddingResponse) {
		resp.Embedding = embedding
	}
}

// WithResponseContent sets the content for an assistant response
func WithResponseContent(content string) func(*assistant.QueryResponse) {
	return func(resp *assistant.QueryResponse) {
		resp.Response = content
	}
}

// WithConversationID sets the conversation ID for an assistant response
func WithConversationID(conversationID string) func(*assistant.QueryResponse) {
	return func(resp *assistant.QueryResponse) {
		resp.ConversationID = conversationID
	}
}

// CreateTestContext creates a context with timeout for testing
func CreateTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// CreateTestContextWithCancel creates a cancellable context for testing
func CreateTestContextWithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(ctx context.Context, condition func() bool, checkInterval time.Duration) error {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if condition() {
				return nil
			}
		}
	}
}
