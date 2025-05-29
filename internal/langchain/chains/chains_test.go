package chains

import (
	"context"
	"testing"

	"github.com/tmc/langchaingo/llms"

	"github.com/koopa0/assistant/internal/config"
	"github.com/koopa0/assistant/internal/testutil"
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

func TestChainFactory(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	factory := NewChainFactory(mockLLM, config, logger)

	// Test creating all chain types
	chainTypes := []ChainType{
		ChainTypeSequential,
		ChainTypeParallel,
		ChainTypeConditional,
		ChainTypeRAG,
	}

	for _, chainType := range chainTypes {
		t.Run(string(chainType), func(t *testing.T) {
			chain, err := factory.CreateChain(chainType)
			if err != nil {
				t.Fatalf("Failed to create %s chain: %v", chainType, err)
			}

			if chain == nil {
				t.Fatalf("Chain should not be nil")
			}

			if chain.GetType() != chainType {
				t.Errorf("Expected chain type %s, got %s", chainType, chain.GetType())
			}
		})
	}

	// Test creating all chains at once
	allChains, err := factory.CreateAllChains()
	if err != nil {
		t.Fatalf("Failed to create all chains: %v", err)
	}

	if len(allChains) != len(chainTypes) {
		t.Errorf("Expected %d chains, got %d", len(chainTypes), len(allChains))
	}

	// Test unsupported chain type
	_, err = factory.CreateChain(ChainType("unsupported"))
	if err == nil {
		t.Error("Expected error for unsupported chain type")
	}
}

func TestChainManager(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	manager := NewChainManager(logger)
	factory := NewChainFactory(mockLLM, config, logger)

	// Create and register chains
	sequentialChain, _ := factory.CreateChain(ChainTypeSequential)
	manager.RegisterChain(ChainTypeSequential, sequentialChain)

	// Test getting registered chain
	chain, err := manager.GetChain(ChainTypeSequential)
	if err != nil {
		t.Fatalf("Failed to get registered chain: %v", err)
	}

	if chain.GetType() != ChainTypeSequential {
		t.Errorf("Expected chain type %s, got %s", ChainTypeSequential, chain.GetType())
	}

	// Test getting non-existent chain
	_, err = manager.GetChain(ChainTypeParallel)
	if err == nil {
		t.Error("Expected error for non-existent chain")
	}

	// Test listing chains
	chainTypes := manager.ListChains()
	if len(chainTypes) != 1 {
		t.Errorf("Expected 1 chain type, got %d", len(chainTypes))
	}

	if chainTypes[0] != ChainTypeSequential {
		t.Errorf("Expected chain type %s, got %s", ChainTypeSequential, chainTypes[0])
	}

	// Test executing chain
	ctx := context.Background()
	request := &ChainRequest{
		Input:       "Test input",
		MaxSteps:    3,
		Temperature: 0.7,
	}

	response, err := manager.ExecuteChain(ctx, ChainTypeSequential, request)
	if err != nil {
		t.Fatalf("Chain execution failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	// Test health check
	err = manager.Health(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestSequentialChain(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	chain := NewSequentialChain(mockLLM, config, logger)

	// Test chain creation
	if chain == nil {
		t.Fatal("Failed to create sequential chain")
	}

	if chain.GetType() != ChainTypeSequential {
		t.Errorf("Expected chain type %s, got %s", ChainTypeSequential, chain.GetType())
	}

	// Test default steps
	templates := chain.GetStepTemplates()
	if len(templates) == 0 {
		t.Error("Sequential chain should have default step templates")
	}

	expectedSteps := []string{"task_analysis", "step_planning", "execution", "synthesis"}
	if len(templates) != len(expectedSteps) {
		t.Errorf("Expected %d step templates, got %d", len(expectedSteps), len(templates))
	}

	for i, expected := range expectedSteps {
		if templates[i].Name != expected {
			t.Errorf("Expected step %d to be %s, got %s", i, expected, templates[i].Name)
		}
	}

	// Test execution
	ctx := context.Background()
	request := &ChainRequest{
		Input:       "Analyze the performance of a web application",
		MaxSteps:    4,
		Temperature: 0.7,
	}

	response, err := chain.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Chain execution failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if len(response.Steps) == 0 {
		t.Error("Response should contain execution steps")
	}

	if !response.Success {
		t.Error("Chain execution should be successful")
	}

	// Test custom steps
	customSteps := []SequentialStepTemplate{
		{
			Name:           "custom_step",
			PromptTemplate: "Custom prompt: {input}",
			OutputKey:      "custom_output",
		},
	}

	chain.SetCustomSteps(customSteps)
	updatedTemplates := chain.GetStepTemplates()
	if len(updatedTemplates) != 1 {
		t.Errorf("Expected 1 custom step template, got %d", len(updatedTemplates))
	}

	if updatedTemplates[0].Name != "custom_step" {
		t.Errorf("Expected custom step name 'custom_step', got '%s'", updatedTemplates[0].Name)
	}
}

func TestParallelChain(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	chain := NewParallelChain(mockLLM, config, logger)

	// Test chain creation
	if chain == nil {
		t.Fatal("Failed to create parallel chain")
	}

	if chain.GetType() != ChainTypeParallel {
		t.Errorf("Expected chain type %s, got %s", ChainTypeParallel, chain.GetType())
	}

	// Test default tasks
	tasks := chain.GetParallelTasks()
	if len(tasks) == 0 {
		t.Error("Parallel chain should have default tasks")
	}

	expectedTasks := []string{"content_analysis", "sentiment_analysis", "keyword_extraction", "summary_generation", "quality_assessment"}
	if len(tasks) != len(expectedTasks) {
		t.Errorf("Expected %d parallel tasks, got %d", len(expectedTasks), len(tasks))
	}

	// Test execution
	ctx := context.Background()
	request := &ChainRequest{
		Input:       "This is a test document for parallel analysis",
		MaxSteps:    5,
		Temperature: 0.7,
	}

	response, err := chain.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Chain execution failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if len(response.Steps) == 0 {
		t.Error("Response should contain execution steps")
	}

	if !response.Success {
		t.Error("Chain execution should be successful")
	}

	// Test concurrency setting
	chain.SetMaxConcurrency(3)
	// Note: We can't easily test the actual concurrency behavior in unit tests
	// but we can verify the setting doesn't cause errors
}

func TestConditionalChain(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	chain := NewConditionalChain(mockLLM, config, logger)

	// Test chain creation
	if chain == nil {
		t.Fatal("Failed to create conditional chain")
	}

	if chain.GetType() != ChainTypeConditional {
		t.Errorf("Expected chain type %s, got %s", ChainTypeConditional, chain.GetType())
	}

	// Test default nodes
	nodes := chain.GetConditionalNodes()
	if len(nodes) == 0 {
		t.Error("Conditional chain should have default nodes")
	}

	expectedNodes := []string{"input_type_detection", "complexity_assessment", "sentiment_routing"}
	if len(nodes) != len(expectedNodes) {
		t.Errorf("Expected %d conditional nodes, got %d", len(expectedNodes), len(nodes))
	}

	// Test execution with code input
	ctx := context.Background()
	request := &ChainRequest{
		Input:       "func main() { fmt.Println(\"Hello, World!\") }",
		MaxSteps:    3,
		Temperature: 0.7,
	}

	response, err := chain.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Chain execution failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if len(response.Steps) == 0 {
		t.Error("Response should contain execution steps")
	}

	if !response.Success {
		t.Error("Chain execution should be successful")
	}

	// Test execution with text input
	textRequest := &ChainRequest{
		Input:       "This is a simple text document for analysis",
		MaxSteps:    3,
		Temperature: 0.7,
	}

	textResponse, err := chain.Execute(ctx, textRequest)
	if err != nil {
		t.Fatalf("Chain execution failed: %v", err)
	}

	if textResponse == nil {
		t.Fatal("Response should not be nil")
	}
}

func TestRAGChain(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	chain := NewRAGChain(mockLLM, config, logger)

	// Test chain creation
	if chain == nil {
		t.Fatal("Failed to create RAG chain")
	}

	if chain.GetType() != ChainTypeRAG {
		t.Errorf("Expected chain type %s, got %s", ChainTypeRAG, chain.GetType())
	}

	// Test retrieval config
	retrievalConfig := chain.GetRetrievalConfig()
	if retrievalConfig.MaxDocuments == 0 {
		t.Error("RAG chain should have default retrieval config")
	}

	// Test execution (without embedding client - will use mock documents)
	ctx := context.Background()
	request := &ChainRequest{
		Input:       "What is the purpose of Go programming language?",
		MaxSteps:    4,
		Temperature: 0.7,
	}

	response, err := chain.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Chain execution failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response should not be nil")
	}

	if len(response.Steps) == 0 {
		t.Error("Response should contain execution steps")
	}

	if !response.Success {
		t.Error("Chain execution should be successful")
	}

	// Test updating similarity threshold
	chain.UpdateSimilarityThreshold(0.8)
	updatedConfig := chain.GetRetrievalConfig()
	if updatedConfig.SimilarityThreshold != 0.8 {
		t.Errorf("Expected similarity threshold 0.8, got %f", updatedConfig.SimilarityThreshold)
	}

	// Test adding content type
	chain.AddContentType("test_content")
	finalConfig := chain.GetRetrievalConfig()
	found := false
	for _, contentType := range finalConfig.ContentTypes {
		if contentType == "test_content" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Content type 'test_content' should be added")
	}
}

func TestChainRequestValidation(t *testing.T) {
	logger := testutil.NewTestLogger()
	mockLLM := NewMockLLM()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	chain := NewSequentialChain(mockLLM, config, logger)

	testCases := []struct {
		name        string
		request     *ChainRequest
		expectError bool
	}{
		{
			name:        "nil request",
			request:     nil,
			expectError: true,
		},
		{
			name: "empty input",
			request: &ChainRequest{
				Input: "",
			},
			expectError: true,
		},
		{
			name: "negative max steps",
			request: &ChainRequest{
				Input:    "Test input",
				MaxSteps: -1,
			},
			expectError: true,
		},
		{
			name: "invalid temperature",
			request: &ChainRequest{
				Input:       "Test input",
				Temperature: 3.0, // Should be between 0 and 2
			},
			expectError: true,
		},
		{
			name: "valid request",
			request: &ChainRequest{
				Input:       "Test input",
				MaxSteps:    3,
				Temperature: 0.7,
			},
			expectError: false,
		},
	}

	ctx := context.Background()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := chain.Execute(ctx, tc.request)
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
