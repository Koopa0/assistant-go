package langchain

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/koopa0/assistant-go/internal/langchain"
	"github.com/koopa0/assistant-go/internal/langchain/agents"
	"github.com/koopa0/assistant-go/internal/langchain/chains"
	"github.com/koopa0/assistant-go/internal/langchain/memory"
	"github.com/koopa0/assistant-go/internal/observability"
)

// TestLangChainService_GetAvailableAgents tests listing available agents
func TestLangChainService_GetAvailableAgents(t *testing.T) {
	// Create logger
	logger := observability.ServerLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)), "langchain_test")

	// Create service
	service := NewLangChainService(nil, logger)

	// Test listing agents with nil service (should return empty list)
	agents := service.GetAvailableAgents()
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents with nil service, got %d", len(agents))
	}

	// Create mock service with agents
	mockService := &langchain.Service{}
	service.service = mockService

	// Test listing agents (should return empty list as service is still mock)
	agents = service.GetAvailableAgents()
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents with mock service, got %d", len(agents))
	}
}

// TestLangChainService_GetAvailableChains tests listing available chains
func TestLangChainService_GetAvailableChains(t *testing.T) {
	// Create logger
	logger := observability.ServerLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)), "langchain_test")

	// Create service
	service := NewLangChainService(nil, logger)

	// Test listing chains with nil service (should return empty list)
	chains := service.GetAvailableChains()
	if len(chains) != 0 {
		t.Errorf("Expected 0 chains with nil service, got %d", len(chains))
	}
}

// TestExecuteAgentRequest_JSON tests JSON marshalling/unmarshalling
func TestExecuteAgentRequest_JSON(t *testing.T) {
	req := &ExecuteAgentRequest{
		UserID:   "user123",
		Query:    "分析這個 Go 專案的架構",
		MaxSteps: 5,
		Context: map[string]interface{}{
			"project_path": "/path/to/project",
		},
	}

	// Test JSON marshalling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Test JSON unmarshalling
	var unmarshalled ExecuteAgentRequest
	err = json.Unmarshal(data, &unmarshalled)
	if err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	// Verify fields
	if unmarshalled.UserID != req.UserID {
		t.Errorf("UserID mismatch: got %s, want %s", unmarshalled.UserID, req.UserID)
	}
	if unmarshalled.Query != req.Query {
		t.Errorf("Query mismatch: got %s, want %s", unmarshalled.Query, req.Query)
	}
	if unmarshalled.MaxSteps != req.MaxSteps {
		t.Errorf("MaxSteps mismatch: got %d, want %d", unmarshalled.MaxSteps, req.MaxSteps)
	}
}

// TestExecuteChainRequest_JSON tests JSON marshalling/unmarshalling
func TestExecuteChainRequest_JSON(t *testing.T) {
	req := &ExecuteChainRequest{
		UserID: "user123",
		Input:  "請幫我設計一個使用者認證系統",
		Context: map[string]interface{}{
			"technology": "Go",
			"database":   "PostgreSQL",
		},
	}

	// Test JSON marshalling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Test JSON unmarshalling
	var unmarshalled ExecuteChainRequest
	err = json.Unmarshal(data, &unmarshalled)
	if err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	// Verify fields
	if unmarshalled.UserID != req.UserID {
		t.Errorf("UserID mismatch: got %s, want %s", unmarshalled.UserID, req.UserID)
	}
	if unmarshalled.Input != req.Input {
		t.Errorf("Input mismatch: got %s, want %s", unmarshalled.Input, req.Input)
	}
}

// TestStoreMemoryRequest_JSON tests JSON marshalling/unmarshalling
func TestStoreMemoryRequest_JSON(t *testing.T) {
	req := &StoreMemoryRequest{
		UserID:     "user123",
		Type:       "episodic",
		Content:    "使用者偏好使用 PostgreSQL 作為主資料庫",
		Importance: 0.8,
		Metadata: map[string]interface{}{
			"category":   "preference",
			"technology": "database",
		},
	}

	// Test JSON marshalling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Test JSON unmarshalling
	var unmarshalled StoreMemoryRequest
	err = json.Unmarshal(data, &unmarshalled)
	if err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	// Verify fields
	if unmarshalled.UserID != req.UserID {
		t.Errorf("UserID mismatch: got %s, want %s", unmarshalled.UserID, req.UserID)
	}
	if unmarshalled.Type != req.Type {
		t.Errorf("Type mismatch: got %s, want %s", unmarshalled.Type, req.Type)
	}
	if unmarshalled.Content != req.Content {
		t.Errorf("Content mismatch: got %s, want %s", unmarshalled.Content, req.Content)
	}
	if unmarshalled.Importance != req.Importance {
		t.Errorf("Importance mismatch: got %f, want %f", unmarshalled.Importance, req.Importance)
	}
}

// TestSearchMemoryRequest_JSON tests JSON marshalling/unmarshalling
func TestSearchMemoryRequest_JSON(t *testing.T) {
	req := &SearchMemoryRequest{
		UserID:    "user123",
		Query:     "資料庫偏好",
		Types:     []string{"episodic", "semantic"},
		Limit:     10,
		Threshold: 0.7,
	}

	// Test JSON marshalling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Test JSON unmarshalling
	var unmarshalled SearchMemoryRequest
	err = json.Unmarshal(data, &unmarshalled)
	if err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	// Verify fields
	if unmarshalled.UserID != req.UserID {
		t.Errorf("UserID mismatch: got %s, want %s", unmarshalled.UserID, req.UserID)
	}
	if unmarshalled.Query != req.Query {
		t.Errorf("Query mismatch: got %s, want %s", unmarshalled.Query, req.Query)
	}
	if len(unmarshalled.Types) != len(req.Types) {
		t.Errorf("Types length mismatch: got %d, want %d", len(unmarshalled.Types), len(req.Types))
	}
	if unmarshalled.Limit != req.Limit {
		t.Errorf("Limit mismatch: got %d, want %d", unmarshalled.Limit, req.Limit)
	}
	if unmarshalled.Threshold != req.Threshold {
		t.Errorf("Threshold mismatch: got %f, want %f", unmarshalled.Threshold, req.Threshold)
	}
}

// Mock agent type for testing
var testAgentTypes = []agents.AgentType{
	agents.AgentTypeDevelopment,
	agents.AgentTypeDatabase,
	agents.AgentTypeInfrastructure,
	agents.AgentTypeResearch,
}

// Mock chain type for testing
var testChainTypes = []chains.ChainType{
	chains.ChainTypeSequential,
	chains.ChainTypeConditional,
	chains.ChainTypeParallel,
	chains.ChainTypeRAG,
}

// TestAgentTypes tests agent type constants
func TestAgentTypes(t *testing.T) {
	for _, agentType := range testAgentTypes {
		if string(agentType) == "" {
			t.Errorf("Agent type string is empty for %v", agentType)
		}
	}
}

// TestChainTypes tests chain type constants
func TestChainTypes(t *testing.T) {
	for _, chainType := range testChainTypes {
		if string(chainType) == "" {
			t.Errorf("Chain type string is empty for %v", chainType)
		}
	}
}

// TestLangChainService_ExecuteAgentWithNilService tests agent execution with nil service
func TestLangChainService_ExecuteAgentWithNilService(t *testing.T) {
	// Create logger
	logger := observability.ServerLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)), "langchain_test")

	// Create service with nil underlying service
	service := NewLangChainService(nil, logger)

	// Create request (using langchain package type)
	req := &langchain.AgentExecutionRequest{
		UserID: "user123",
		AgentRequest: &agents.AgentRequest{
			Query:    "測試查詢",
			MaxSteps: 3,
		},
	}

	// Test agent execution (should fail with service unavailable)
	ctx := context.Background()
	_, err := service.ExecuteAgent(ctx, agents.AgentTypeDevelopment, req)
	if err == nil {
		t.Error("Expected error when executing agent with nil service")
	}

	// Error should mention service not available
	if err.Error() != "LangChain service not available" {
		t.Errorf("Expected 'LangChain service not available' error, got: %v", err)
	}
}

// TestLangChainService_ExecuteChainWithNilService tests chain execution with nil service
func TestLangChainService_ExecuteChainWithNilService(t *testing.T) {
	// Create logger
	logger := observability.ServerLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)), "langchain_test")

	// Create service with nil underlying service
	service := NewLangChainService(nil, logger)

	// Create request (using langchain package type)
	req := &langchain.ChainExecutionRequest{
		UserID: "user123",
		ChainRequest: &chains.ChainRequest{
			Input: "測試輸入",
		},
	}

	// Test chain execution (should fail with service unavailable)
	ctx := context.Background()
	_, err := service.ExecuteChain(ctx, chains.ChainTypeSequential, req)
	if err == nil {
		t.Error("Expected error when executing chain with nil service")
	}

	// Error should mention service not available
	if err.Error() != "LangChain service not available" {
		t.Errorf("Expected 'LangChain service not available' error, got: %v", err)
	}
}

// TestLangChainService_MemoryWithNilService tests memory operations with nil service
func TestLangChainService_MemoryWithNilService(t *testing.T) {
	// Create logger
	logger := observability.ServerLogger(slog.New(slog.NewTextHandler(os.Stdout, nil)), "langchain_test")

	// Create service with nil underlying service
	service := NewLangChainService(nil, logger)

	ctx := context.Background()

	// Test store memory
	storeEntry := &memory.MemoryEntry{
		UserID:     "user123",
		Type:       memory.MemoryType("episodic"),
		Content:    "測試記憶",
		Importance: 0.8,
	}

	err := service.StoreMemory(ctx, storeEntry)
	if err == nil {
		t.Error("Expected error when storing memory with nil service")
	}

	// Test search memory
	searchQuery := &memory.MemoryQuery{
		UserID:        "user123",
		Content:       "測試",
		Types:         []memory.MemoryType{memory.MemoryType("episodic")},
		Limit:         10,
		Similarity:    0.7,
		MinImportance: 0.5,
	}

	_, err = service.SearchMemory(ctx, searchQuery)
	if err == nil {
		t.Error("Expected error when searching memory with nil service")
	}

	// Test memory stats
	_, err = service.GetMemoryStats(ctx, "user123")
	if err == nil {
		t.Error("Expected error when getting memory stats with nil service")
	}
}
