package langchain

import (
	"context"
	"testing"

	"github.com/tmc/langchaingo/llms"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/langchain/agents"
	"github.com/koopa0/assistant-go/internal/langchain/chains"
	"github.com/koopa0/assistant-go/internal/testutil"
)

// MockLLM implements a mock LLM for testing
type MockLLM struct {
	responses map[string]string
}

func NewMockLLM() *MockLLM {
	return &MockLLM{
		responses: map[string]string{
			"default": "This is a mock response from the LLM.",
		},
	}
}

func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	if response, exists := m.responses[prompt]; exists {
		return response, nil
	}
	return m.responses["default"], nil
}

func (m *MockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	// Extract the last message content for simplicity
	prompt := "default"
	if len(messages) > 0 && len(messages[len(messages)-1].Parts) > 0 {
		if textPart, ok := messages[len(messages)-1].Parts[0].(llms.TextContent); ok {
			prompt = textPart.Text
		}
	}

	response, err := m.Call(ctx, prompt, options...)
	if err != nil {
		return nil, err
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: response,
			},
		},
	}, nil
}

func (m *MockLLM) SetResponse(prompt, response string) {
	m.responses[prompt] = response
}

func TestNewService(t *testing.T) {
	logger := testutil.NewTestLogger()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	// Test with missing logger
	_, err := NewService(ServiceConfig{
		Config:       config,
		LLMProviders: map[string]llms.Model{"mock": NewMockLLM()},
	})
	if err == nil {
		t.Error("Expected error for missing logger")
	}

	// Test with missing LLM providers
	_, err = NewService(ServiceConfig{
		Config: config,
		Logger: logger,
	})
	if err == nil {
		t.Error("Expected error for missing LLM providers")
	}

	// Test successful creation
	service, err := NewService(ServiceConfig{
		Config:       config,
		Logger:       logger,
		LLMProviders: map[string]llms.Model{"mock": NewMockLLM()},
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	if service == nil {
		t.Error("Service should not be nil")
	}

	// Test service methods
	agents := service.GetAvailableAgents()
	if len(agents) == 0 {
		t.Error("Should have available agents")
	}

	chains := service.GetAvailableChains()
	if len(chains) == 0 {
		t.Error("Should have available chains")
	}

	providers := service.GetLLMProviders()
	if len(providers) != 1 || providers[0] != "mock" {
		t.Errorf("Expected 1 provider 'mock', got %v", providers)
	}
}

func TestServiceExecuteAgent(t *testing.T) {
	logger := testutil.NewTestLogger()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	mockLLM := NewMockLLM()
	mockLLM.SetResponse("test query", "test response")

	service, err := NewService(ServiceConfig{
		Config:       config,
		Logger:       logger,
		LLMProviders: map[string]llms.Model{"mock": mockLLM},
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	request := &agents.AgentRequest{
		Query:    "test query",
		MaxSteps: 3,
		Context:  map[string]interface{}{"test": "context"},
	}

	agentExecRequest := &AgentExecutionRequest{
		UserID:       "test-user",
		AgentRequest: request,
	}
	response, err := service.ExecuteAgent(ctx, agents.AgentTypeDevelopment, agentExecRequest)
	if err != nil {
		t.Fatalf("Failed to execute agent: %v", err)
	}

	if response == nil {
		t.Error("Response should not be nil")
	}

	// AgentResponse doesn't have Success field, check if result is not empty
	if response.Result == "" {
		t.Error("Agent execution should return a result")
	}
}

func TestServiceExecuteChain(t *testing.T) {
	logger := testutil.NewTestLogger()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	mockLLM := NewMockLLM()
	mockLLM.SetResponse("test input", "test output")

	service, err := NewService(ServiceConfig{
		Config:       config,
		Logger:       logger,
		LLMProviders: map[string]llms.Model{"mock": mockLLM},
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	request := &chains.ChainRequest{
		UserID:  "test-user",
		Input:   "test input",
		Context: map[string]interface{}{"test": "context"},
	}

	response, err := service.ExecuteChain(ctx, chains.ChainTypeSequential, request)
	if err != nil {
		t.Fatalf("Failed to execute chain: %v", err)
	}

	if response == nil {
		t.Error("Response should not be nil")
	}

	if !response.Success {
		t.Error("Chain execution should be successful")
	}
}

func TestServiceHealthCheck(t *testing.T) {
	logger := testutil.NewTestLogger()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	service, err := NewService(ServiceConfig{
		Config:       config,
		Logger:       logger,
		LLMProviders: map[string]llms.Model{"mock": NewMockLLM()},
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	ctx := context.Background()
	err = service.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Health check should pass: %v", err)
	}
}

func TestServiceClose(t *testing.T) {
	logger := testutil.NewTestLogger()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	service, err := NewService(ServiceConfig{
		Config:       config,
		Logger:       logger,
		LLMProviders: map[string]llms.Model{"mock": NewMockLLM()},
	})
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	err = service.Close()
	if err != nil {
		t.Errorf("Service close should not error: %v", err)
	}
}
