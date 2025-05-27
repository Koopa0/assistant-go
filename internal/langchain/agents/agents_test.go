package agents

import (
	"context"
	"testing"

	"github.com/tmc/langchaingo/llms"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
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

func TestDevelopmentAgent(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	agent := NewDevelopmentAgent(mockLLM, config, logger)

	// Test agent creation
	if agent == nil {
		t.Fatal("Failed to create development agent")
	}

	if agent.GetType() != AgentTypeDevelopment {
		t.Errorf("Expected agent type %s, got %s", AgentTypeDevelopment, agent.GetType())
	}

	// Test capabilities
	capabilities := agent.GetCapabilities()
	if len(capabilities) == 0 {
		t.Error("Development agent should have capabilities")
	}

	expectedCapabilities := []string{"code_analysis", "code_generation", "performance_analysis", "refactoring_suggestions", "test_generation"}
	for _, expected := range expectedCapabilities {
		found := false
		for _, capability := range capabilities {
			if capability.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected capability: %s", expected)
		}
	}

	// Test execution
	ctx := context.Background()
	request := &AgentRequest{
		Query:       "Analyze this Go code: func main() { fmt.Println(\"Hello\") }",
		MaxSteps:    3,
		Temperature: 0.7,
		Metadata:    map[string]interface{}{"test": true},
	}

	mockLLM.SetResponse("Analyze the following code and provide a comprehensive report:\n\nCode:\nfunc main() { fmt.Println(\"Hello\") }\n\nAST Analysis Summary:\nFound 1 functions, 0 structs, 0 interfaces\n\nPlease provide:\n1. Code structure analysis\n2. Potential issues or improvements\n3. Best practices recommendations\n4. Performance considerations\n\nAnalysis:", "This is a simple Go program that prints 'Hello' to the console.")

	response, err := agent.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Agent execution failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if len(response.Steps) == 0 {
		t.Error("Response should contain execution steps")
	}

	if response.ExecutionTime == 0 {
		t.Error("Execution time should be recorded")
	}
}

func TestDatabaseAgent(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	// Create mock database client
	dbClient := &postgres.SQLCClient{} // This would be properly mocked in real tests

	agent := NewDatabaseAgent(mockLLM, config, dbClient, logger)

	// Test agent creation
	if agent == nil {
		t.Fatal("Failed to create database agent")
	}

	if agent.GetType() != AgentTypeDatabase {
		t.Errorf("Expected agent type %s, got %s", AgentTypeDatabase, agent.GetType())
	}

	// Test capabilities
	capabilities := agent.GetCapabilities()
	expectedCapabilities := []string{"sql_generation", "query_optimization", "schema_exploration", "data_analysis", "migration_assistance"}
	for _, expected := range expectedCapabilities {
		found := false
		for _, capability := range capabilities {
			if capability.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected capability: %s", expected)
		}
	}

	// Test execution
	ctx := context.Background()
	request := &AgentRequest{
		Query:       "Generate a SQL query to find all users created in the last 30 days",
		MaxSteps:    3,
		Temperature: 0.5,
	}

	response, err := agent.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Agent execution failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}
}

func TestInfrastructureAgent(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	agent := NewInfrastructureAgent(mockLLM, config, logger)

	// Test agent creation
	if agent == nil {
		t.Fatal("Failed to create infrastructure agent")
	}

	if agent.GetType() != AgentTypeInfrastructure {
		t.Errorf("Expected agent type %s, got %s", AgentTypeInfrastructure, agent.GetType())
	}

	// Test capabilities
	capabilities := agent.GetCapabilities()
	expectedCapabilities := []string{"kubernetes_management", "docker_management", "log_analysis", "troubleshooting", "resource_monitoring", "deployment_assistance"}
	for _, expected := range expectedCapabilities {
		found := false
		for _, capability := range capabilities {
			if capability.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected capability: %s", expected)
		}
	}

	// Test execution
	ctx := context.Background()
	request := &AgentRequest{
		Query:       "List all pods in the default namespace",
		MaxSteps:    3,
		Temperature: 0.5,
	}

	response, err := agent.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Agent execution failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}
}

func TestResearchAgent(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	agent := NewResearchAgent(mockLLM, config, logger)

	// Test agent creation
	if agent == nil {
		t.Fatal("Failed to create research agent")
	}

	if agent.GetType() != AgentTypeResearch {
		t.Errorf("Expected agent type %s, got %s", AgentTypeResearch, agent.GetType())
	}

	// Test capabilities
	capabilities := agent.GetCapabilities()
	expectedCapabilities := []string{"information_gathering", "fact_checking", "report_generation", "source_analysis", "trend_analysis", "comparative_analysis"}
	for _, expected := range expectedCapabilities {
		found := false
		for _, capability := range capabilities {
			if capability.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected capability: %s", expected)
		}
	}

	// Test execution
	ctx := context.Background()
	request := &AgentRequest{
		Query:       "Research the latest trends in Go programming language",
		MaxSteps:    3,
		Temperature: 0.7,
	}

	response, err := agent.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Agent execution failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}
}

func TestBaseAgent(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	agent := NewBaseAgent(AgentTypeDevelopment, mockLLM, config, logger)

	// Test agent creation
	if agent == nil {
		t.Fatal("Failed to create base agent")
	}

	// Test adding capabilities
	capability := AgentCapability{
		Name:        "test_capability",
		Description: "A test capability",
		Parameters:  map[string]interface{}{"param1": "value1"},
	}

	agent.AddCapability(capability)
	capabilities := agent.GetCapabilities()
	if len(capabilities) != 1 {
		t.Errorf("Expected 1 capability, got %d", len(capabilities))
	}

	if capabilities[0].Name != "test_capability" {
		t.Errorf("Expected capability name 'test_capability', got '%s'", capabilities[0].Name)
	}

	// Test health check
	ctx := context.Background()
	err := agent.Health(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test execution with invalid request
	invalidRequest := &AgentRequest{
		Query: "", // Empty query should fail validation
	}

	_, err = agent.Execute(ctx, invalidRequest)
	if err == nil {
		t.Error("Expected error for invalid request")
	}

	// Test execution with valid request
	validRequest := &AgentRequest{
		Query:       "Test query",
		MaxSteps:    3,
		Temperature: 0.5,
	}

	response, err := agent.Execute(ctx, validRequest)
	if err != nil {
		t.Fatalf("Agent execution failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if len(response.Steps) == 0 {
		t.Error("Response should contain execution steps")
	}
}

func TestAgentRequestValidation(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	agent := NewBaseAgent(AgentTypeDevelopment, mockLLM, config, logger)

	testCases := []struct {
		name        string
		request     *AgentRequest
		expectError bool
	}{
		{
			name:        "nil request",
			request:     nil,
			expectError: true,
		},
		{
			name: "empty query",
			request: &AgentRequest{
				Query: "",
			},
			expectError: true,
		},
		{
			name: "negative max steps",
			request: &AgentRequest{
				Query:    "Test query",
				MaxSteps: -1,
			},
			expectError: true,
		},
		{
			name: "invalid temperature",
			request: &AgentRequest{
				Query:       "Test query",
				Temperature: 3.0, // Should be between 0 and 2
			},
			expectError: true,
		},
		{
			name: "valid request",
			request: &AgentRequest{
				Query:       "Test query",
				MaxSteps:    3,
				Temperature: 0.7,
			},
			expectError: false,
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := agent.Execute(ctx, tc.request)
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
