package agent

import (
	"context"
	"testing"

	"github.com/koopa0/assistant-go/internal/testutil"
	"github.com/tmc/langchaingo/llms"
)

// MockLLM implements a simple mock LLM for testing
type MockLLM struct {
	response string
	err      error
}

func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *MockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	response, err := m.Call(ctx, "test", options...)
	if err != nil {
		return nil, err
	}
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{Content: response},
		},
	}, nil
}

func TestBaseAgent_Execute(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := &MockLLM{response: "Test response"}
	agent := NewBaseAgent(TypeGeneral, mockLLM, logger)

	tests := []struct {
		name    string
		request *Request
		wantErr bool
	}{
		{
			name: "valid request",
			request: &Request{
				Query:    "Test query",
				MaxSteps: 1,
			},
			wantErr: false,
		},
		{
			name: "empty query",
			request: &Request{
				Query: "",
			},
			wantErr: false, // Validation error returned in response
		},
		{
			name: "with context",
			request: &Request{
				Query:    "Test with context",
				MaxSteps: 2,
				Context: map[string]interface{}{
					"key": "value",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := agent.Execute(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil && resp == nil {
				t.Error("Execute() returned nil response without error")
			}

			if resp != nil && tt.request.Query == "" {
				if resp.Success {
					t.Error("Execute() should fail for empty query")
				}
			}
		})
	}
}

func TestManager_RegisterAgent(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := &MockLLM{response: "Test response"}
	manager := NewManager(nil, logger) // Start with no LLM

	// Test registering a new agent
	agent := NewBaseAgent(TypeGeneral, mockLLM, logger)
	err := manager.RegisterAgent(TypeGeneral, agent)
	if err != nil {
		t.Errorf("RegisterAgent() error = %v", err)
	}

	// Test retrieving registered agent
	retrieved, err := manager.GetAgent(TypeGeneral)
	if err != nil {
		t.Errorf("GetAgent() error = %v", err)
	}
	if retrieved == nil {
		t.Error("GetAgent() returned nil agent")
	}

	// Test getting non-existent agent
	_, err = manager.GetAgent(TypeDevelopment)
	if err == nil {
		t.Error("GetAgent() should error for non-existent agent")
	}
}

func TestManager_SelectBestAgent(t *testing.T) {
	logger := testutil.NewTestLogger()
	manager := NewManager(nil, logger)

	tests := []struct {
		name     string
		request  *Request
		expected AgentType
	}{
		{
			name: "development query",
			request: &Request{
				Query: "Write a function to parse JSON",
			},
			expected: TypeDevelopment,
		},
		{
			name: "database query",
			request: &Request{
				Query: "Create a SQL query to find users",
			},
			expected: TypeDatabase,
		},
		{
			name: "infrastructure query",
			request: &Request{
				Query: "Deploy application to Kubernetes",
			},
			expected: TypeInfrastructure,
		},
		{
			name: "research query",
			request: &Request{
				Query: "Explain best practices for microservices",
			},
			expected: TypeResearch,
		},
		{
			name: "general query",
			request: &Request{
				Query: "What is the weather today?",
			},
			expected: TypeGeneral,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentType, err := manager.SelectBestAgent(tt.request)
			if err != nil {
				t.Errorf("SelectBestAgent() error = %v", err)
				return
			}
			if agentType != tt.expected {
				t.Errorf("SelectBestAgent() = %v, want %v", agentType, tt.expected)
			}
		})
	}
}

func TestDevelopmentAgent_Execute(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := &MockLLM{response: "function parseJSON(data string) { return json.Unmarshal(data) }\nImplementation complete"}
	agent := NewDevelopmentAgent(mockLLM, logger)

	request := &Request{
		Query:    "Write a function to parse JSON",
		MaxSteps: 2,
		Tools:    []string{"godev"},
	}

	ctx := context.Background()
	resp, err := agent.Execute(ctx, request)

	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	if resp == nil {
		t.Fatal("Execute() returned nil response")
	}

	if !resp.Success {
		t.Error("Execute() should succeed")
	}

	if len(resp.Steps) == 0 {
		t.Error("Execute() should have steps")
	}
}
