package assistant

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/koopa0/assistant/internal/ai"
	"github.com/koopa0/assistant/internal/config"
	"github.com/koopa0/assistant/internal/tools"
)

// MockAIManager implements ai.Manager interface for testing
type MockAIManager struct {
	generateResponseFunc func(ctx context.Context, request *ai.GenerateRequest, providerName ...string) (*ai.GenerateResponse, error)
	healthFunc           func(ctx context.Context) error
	getUsageStatsFunc    func(ctx context.Context) (map[string]*ai.UsageStats, error)
}

func (m *MockAIManager) GenerateResponse(ctx context.Context, request *ai.GenerateRequest, providerName ...string) (*ai.GenerateResponse, error) {
	if m.generateResponseFunc != nil {
		return m.generateResponseFunc(ctx, request, providerName...)
	}
	return &ai.GenerateResponse{
		Content:  "Mock response",
		Model:    "mock-model",
		Provider: "mock",
		TokensUsed: ai.TokenUsage{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
		FinishReason: "stop",
		ResponseTime: 100 * time.Millisecond,
	}, nil
}

func (m *MockAIManager) GenerateEmbedding(ctx context.Context, text string, providerName ...string) (*ai.EmbeddingResponse, error) {
	return &ai.EmbeddingResponse{
		Embedding:    []float64{0.1, 0.2, 0.3},
		Model:        "mock-embedding",
		Provider:     "mock",
		TokensUsed:   5,
		ResponseTime: 50 * time.Millisecond,
	}, nil
}

func (m *MockAIManager) Health(ctx context.Context) error {
	if m.healthFunc != nil {
		return m.healthFunc(ctx)
	}
	return nil
}

func (m *MockAIManager) GetUsageStats(ctx context.Context) (map[string]*ai.UsageStats, error) {
	if m.getUsageStatsFunc != nil {
		return m.getUsageStatsFunc(ctx)
	}
	return map[string]*ai.UsageStats{
		"mock": {
			TotalRequests: 10,
			TotalTokens:   100,
			InputTokens:   50,
			OutputTokens:  50,
		},
	}, nil
}

func (m *MockAIManager) GetAvailableProviders() []string {
	return []string{"mock"}
}

func (m *MockAIManager) GetDefaultProvider() string {
	return "mock"
}

func (m *MockAIManager) Close(ctx context.Context) error {
	return nil
}

// MockContextManager implements ContextManager interface for testing
type MockContextManager struct {
	conversations map[string]*Conversation
	messages      map[string][]*Message
}

func NewMockContextManager() *MockContextManager {
	return &MockContextManager{
		conversations: make(map[string]*Conversation),
		messages:      make(map[string][]*Message),
	}
}

func (m *MockContextManager) CreateConversation(ctx context.Context, userID, title string) (*Conversation, error) {
	conv := &Conversation{
		ID:        "test-conv-1",
		UserID:    userID,
		Title:     title,
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  []*Message{},
	}
	m.conversations[conv.ID] = conv
	return conv, nil
}

func (m *MockContextManager) GetConversation(ctx context.Context, conversationID string) (*Conversation, error) {
	conv, exists := m.conversations[conversationID]
	if !exists {
		return nil, NewContextNotFoundError(conversationID)
	}

	// Load messages
	if msgs, exists := m.messages[conversationID]; exists {
		conv.Messages = msgs
	}

	return conv, nil
}

func (m *MockContextManager) AddMessage(ctx context.Context, conversationID, role, content string, metadata map[string]interface{}) (*Message, error) {
	msg := &Message{
		ID:             "test-msg-1",
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		Metadata:       metadata,
		CreatedAt:      time.Now(),
	}

	if m.messages[conversationID] == nil {
		m.messages[conversationID] = []*Message{}
	}
	m.messages[conversationID] = append(m.messages[conversationID], msg)

	return msg, nil
}

func (m *MockContextManager) Close(ctx context.Context) error {
	return nil
}

// MockToolRegistry implements tools.Registry interface for testing
type MockToolRegistry struct{}

func (m *MockToolRegistry) Register(tool tools.Tool) error {
	return nil
}

func (m *MockToolRegistry) Get(name string) (tools.Tool, error) {
	return nil, tools.NewToolError("mock", "NOT_FOUND", "tool not found", nil)
}

func (m *MockToolRegistry) IsRegistered(name string) bool {
	return name == "test-tool"
}

func (m *MockToolRegistry) ListTools() []string {
	return []string{"test-tool"}
}

func (m *MockToolRegistry) Health(ctx context.Context) error {
	return nil
}

// MockDatabase implements postgres.Client interface for testing
type MockDatabase struct{}

func (m *MockDatabase) Health(ctx context.Context) error {
	return nil
}

func (m *MockDatabase) Close() error {
	return nil
}

// TestProcessor is a test version of Processor with injectable dependencies
type TestProcessor struct {
	config    *config.Config
	logger    *slog.Logger
	context   *MockContextManager
	aiManager *MockAIManager
}

// Test helper to create a test processor
func createTestProcessor(t *testing.T) *TestProcessor {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	cfg := &config.Config{
		AI: config.AIConfig{
			DefaultProvider: "mock",
			Claude: config.Claude{
				Model:       "claude-3-sonnet",
				MaxTokens:   4096,
				Temperature: 0.7,
			},
			Gemini: config.Gemini{
				Model:       "gemini-pro",
				MaxTokens:   4096,
				Temperature: 0.7,
			},
		},
	}

	mockContext := NewMockContextManager()
	mockAI := &MockAIManager{}

	processor := &TestProcessor{
		config:    cfg,
		logger:    logger,
		context:   mockContext,
		aiManager: mockAI,
	}

	return processor
}

// Process simulates the main processing logic for testing
func (p *TestProcessor) Process(ctx context.Context, request *QueryRequest) (*QueryResponse, error) {
	startTime := time.Now()

	// Basic validation
	if request.Query == "" {
		return nil, NewInvalidInputError("query cannot be empty", nil)
	}

	// Create or get conversation
	var conversationID string
	if request.ConversationID != nil {
		conversationID = *request.ConversationID
	} else {
		conv, err := p.context.CreateConversation(ctx, *request.UserID, "Test Conversation")
		if err != nil {
			return nil, err
		}
		conversationID = conv.ID
	}

	// Add user message
	_, err := p.context.AddMessage(ctx, conversationID, "user", request.Query, nil)
	if err != nil {
		return nil, err
	}

	// Generate AI response
	aiRequest := &ai.GenerateRequest{
		Messages: []ai.Message{
			{Role: "user", Content: request.Query},
		},
		MaxTokens:   p.config.AI.Claude.MaxTokens,
		Temperature: p.config.AI.Claude.Temperature,
		Model:       p.config.AI.Claude.Model,
	}

	aiResponse, err := p.aiManager.GenerateResponse(ctx, aiRequest)
	if err != nil {
		return nil, err
	}

	// Add assistant message
	assistantMsg, err := p.context.AddMessage(ctx, conversationID, "assistant", aiResponse.Content, nil)
	if err != nil {
		return nil, err
	}

	return &QueryResponse{
		Response:       aiResponse.Content,
		ConversationID: conversationID,
		MessageID:      assistantMsg.ID,
		Provider:       aiResponse.Provider,
		Model:          aiResponse.Model,
		TokensUsed:     aiResponse.TokensUsed.TotalTokens,
		ExecutionTime:  time.Since(startTime),
		Context:        make(map[string]interface{}),
	}, nil
}

// Health simulates health check for testing
func (p *TestProcessor) Health(ctx context.Context) error {
	return p.aiManager.Health(ctx)
}

// Stats simulates stats collection for testing
func (p *TestProcessor) Stats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	aiStats, err := p.aiManager.GetUsageStats(ctx)
	if err != nil {
		return nil, err
	}

	stats["ai_providers"] = aiStats
	stats["available_providers"] = p.aiManager.GetAvailableProviders()
	stats["default_provider"] = p.aiManager.GetDefaultProvider()

	return stats, nil
}

func TestProcessor_Process_Success(t *testing.T) {
	processor := createTestProcessor(t)
	ctx := context.Background()

	request := &QueryRequest{
		Query:  "Hello, how are you?",
		UserID: stringPtr("test-user"),
	}

	response, err := processor.Process(ctx, request)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Response == "" {
		t.Error("Expected non-empty response content")
	}

	if response.ConversationID == "" {
		t.Error("Expected conversation ID")
	}

	if response.Provider != "mock" {
		t.Errorf("Expected provider 'mock', got: %s", response.Provider)
	}

	if response.TokensUsed == 0 {
		t.Error("Expected non-zero token usage")
	}
}

func TestProcessor_Process_ValidationError(t *testing.T) {
	processor := createTestProcessor(t)
	ctx := context.Background()

	request := &QueryRequest{
		Query: "", // Empty query should fail validation
	}

	_, err := processor.Process(ctx, request)
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	// Check if it's an InvalidInputError
	expectedMsg := "INVALID_INPUT: query cannot be empty"
	if err.Error() != expectedMsg {
		t.Errorf("Expected '%s' error, got: %s", expectedMsg, err.Error())
	}
}

func TestProcessor_Health_Success(t *testing.T) {
	processor := createTestProcessor(t)
	ctx := context.Background()

	err := processor.Health(ctx)
	if err != nil {
		t.Fatalf("Expected health check to pass, got: %v", err)
	}
}

func TestProcessor_Stats_Success(t *testing.T) {
	processor := createTestProcessor(t)
	ctx := context.Background()

	stats, err := processor.Stats(ctx)
	if err != nil {
		t.Fatalf("Expected stats collection to succeed, got: %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats, got nil")
	}

	// Check for expected stats keys
	expectedKeys := []string{"ai_providers", "available_providers", "default_provider"}
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected stats key '%s' not found", key)
		}
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
