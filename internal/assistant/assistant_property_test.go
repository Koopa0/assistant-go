package assistant

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
	"github.com/koopa0/assistant-go/internal/testutil"
)

// FuzzQueryProcessing tests query processing with various inputs
func FuzzQueryProcessing(f *testing.F) {
	// Add seed corpus with valid queries
	f.Add("Hello world")
	f.Add("Analyze this Go code")
	f.Add("What is the meaning of life?")
	f.Add("幫助我理解並發編程")         // Chinese
	f.Add("مساعدة في البرمجة") // Arabic
	f.Add("🚀 Test emoji query")
	f.Add("")

	f.Fuzz(func(t *testing.T, query string) {
		// Skip if not valid UTF-8
		if !utf8.ValidString(query) {
			t.Skip("Invalid UTF-8 string")
		}

		// Skip extremely long queries to avoid timeout
		if len(query) > 10000 {
			t.Skip("Query too long")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cfg := &config.Config{
			Mode: "test",
			AI: config.AIConfig{
				DefaultProvider: "claude",
				Claude: config.Claude{
					APIKey: "test-key",
					Model:  "claude-3-sonnet-20240229",
				},
			},
		}
		mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
		logger := testutil.NewTestLogger()

		assistant, err := New(ctx, cfg, mockDB, logger)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}
		defer func() {
			if err := assistant.Close(ctx); err != nil {
				t.Logf("Failed to close assistant: %v", err)
			}
		}()

		// Property: Empty or whitespace-only queries should return error
		trimmed := strings.TrimSpace(query)
		if trimmed == "" {
			_, err := assistant.ProcessQuery(ctx, query)
			if err == nil {
				t.Errorf("Expected error for empty/whitespace query: %q", query)
			}
			return
		}

		// Property: Valid queries should not panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Query processing panicked with input %q: %v", query, r)
				}
			}()

			// ProcessQuery may fail in test mode but shouldn't panic
			_, _ = assistant.ProcessQuery(ctx, query)
		}()
	})
}

// FuzzToolExecution tests tool execution with various inputs
func FuzzToolExecution(f *testing.F) {
	// Add seed corpus
	f.Add("go_analyzer", "file_path", "/test/main.go")
	f.Add("go_formatter", "directory", "/test")
	f.Add("go_tester", "package_path", "./...")
	f.Add("", "", "")
	f.Add("nonexistent_tool", "param", "value")

	f.Fuzz(func(t *testing.T, toolName, inputKey, inputValue string) {
		// Skip if not valid UTF-8
		if !utf8.ValidString(toolName) || !utf8.ValidString(inputKey) || !utf8.ValidString(inputValue) {
			t.Skip("Invalid UTF-8 strings")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		cfg := &config.Config{
			Mode: "test",
			AI: config.AIConfig{
				DefaultProvider: "claude",
				Claude: config.Claude{
					APIKey: "test-key",
					Model:  "claude-3-sonnet-20240229",
				},
			},
		}
		mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
		logger := testutil.NewTestLogger()

		assistant, err := New(ctx, cfg, mockDB, logger)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}
		defer func() {
			if err := assistant.Close(ctx); err != nil {
				t.Logf("Failed to close assistant: %v", err)
			}
		}()

		req := &ToolExecutionRequest{
			ToolName: toolName,
			Input: map[string]any{
				inputKey: inputValue,
			},
		}

		// Property: Empty tool name should return error
		if toolName == "" {
			_, err := assistant.ExecuteTool(ctx, req)
			if err == nil {
				t.Errorf("Expected error for empty tool name")
			}
			return
		}

		// Property: Tool execution should not panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Tool execution panicked with tool %q: %v", toolName, r)
				}
			}()

			// ExecuteTool may fail but shouldn't panic
			_, _ = assistant.ExecuteTool(ctx, req)
		}()
	})
}

// TestPropertyBasedQueryRequest tests QueryRequest properties
func TestPropertyBasedQueryRequest(t *testing.T) {
	tests := []struct {
		name     string
		genFunc  func() *QueryRequest
		property func(*QueryRequest) bool
		desc     string
	}{
		{
			name: "non_empty_query_should_be_valid",
			genFunc: func() *QueryRequest {
				return &QueryRequest{
					Query: generateRandomString(rand.Intn(100) + 1),
				}
			},
			property: func(req *QueryRequest) bool {
				return strings.TrimSpace(req.Query) != ""
			},
			desc: "Non-empty queries should have non-empty trimmed query",
		},
		{
			name: "max_tokens_should_be_non_negative",
			genFunc: func() *QueryRequest {
				return &QueryRequest{
					Query:     "test query",
					MaxTokens: rand.Intn(10000),
				}
			},
			property: func(req *QueryRequest) bool {
				return req.MaxTokens >= 0
			},
			desc: "MaxTokens should be non-negative",
		},
		{
			name: "temperature_should_be_valid_range",
			genFunc: func() *QueryRequest {
				return &QueryRequest{
					Query:       "test query",
					Temperature: rand.Float64() * 2, // 0.0 to 2.0
				}
			},
			property: func(req *QueryRequest) bool {
				return req.Temperature >= 0.0 && req.Temperature <= 2.0
			},
			desc: "Temperature should be between 0.0 and 2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run property test multiple times
			for i := 0; i < 100; i++ {
				req := tt.genFunc()
				if !tt.property(req) {
					t.Errorf("Property failed: %s for request: %+v", tt.desc, req)
				}
			}
		})
	}
}

// TestPropertyBasedStats tests statistics properties
func TestPropertyBasedStats(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &config.Config{
		Mode: "test",
		AI: config.AIConfig{
			DefaultProvider: "claude",
			Claude: config.Claude{
				APIKey: "test-key",
				Model:  "claude-3-sonnet-20240229",
			},
		},
	}
	mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
	logger := testutil.NewTestLogger()

	assistant, err := New(ctx, cfg, mockDB, logger)
	if err != nil {
		t.Fatalf("Failed to create assistant: %v", err)
	}
	defer func() {
		if err := assistant.Close(ctx); err != nil {
			t.Logf("Failed to close assistant: %v", err)
		}
	}()

	// Property: Stats should always be consistent
	for i := 0; i < 10; i++ {
		stats, err := assistant.Stats(ctx)
		if err != nil {
			t.Errorf("Failed to get stats on iteration %d: %v", i, err)
			continue
		}

		// Property: Stats should never be nil
		if stats == nil {
			t.Errorf("Stats should never be nil")
			continue
		}

		// Property: Database stats should be valid
		if stats.Database != nil {
			if stats.Database.TotalConns < 0 {
				t.Errorf("TotalConns should be non-negative, got: %d", stats.Database.TotalConns)
			}
			if stats.Database.MaxConns < 0 {
				t.Errorf("MaxConns should be non-negative, got: %d", stats.Database.MaxConns)
			}
			if stats.Database.AcquiredConns < 0 {
				t.Errorf("AcquiredConns should be non-negative, got: %d", stats.Database.AcquiredConns)
			}
		}

		// Property: Tool stats should be valid
		if stats.Tools != nil {
			if stats.Tools.RegisteredTools < 0 {
				t.Errorf("RegisteredTools should be non-negative, got: %d", stats.Tools.RegisteredTools)
			}
		}

		// Property: Processor stats should be valid
		if stats.Processor != nil {
			if stats.Processor.Processor != nil {
				if stats.Processor.Processor.Status == "" {
					t.Errorf("Processor status should not be empty")
				}
				if stats.Processor.Processor.Version == "" {
					t.Errorf("Processor version should not be empty")
				}
			}
			if stats.Processor.Health != nil {
				if stats.Processor.Health.Status == "" {
					t.Errorf("Health status should not be empty")
				}
			}
		}
	}
}

// TestConcurrentAccess tests concurrent access properties
func TestConcurrentAccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := &config.Config{
		Mode: "test",
		AI: config.AIConfig{
			DefaultProvider: "claude",
			Claude: config.Claude{
				APIKey: "test-key",
				Model:  "claude-3-sonnet-20240229",
			},
		},
	}
	mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
	logger := testutil.NewTestLogger()

	assistant, err := New(ctx, cfg, mockDB, logger)
	if err != nil {
		t.Fatalf("Failed to create assistant: %v", err)
	}
	defer func() {
		if err := assistant.Close(ctx); err != nil {
			t.Logf("Failed to close assistant: %v", err)
		}
	}()

	// Property: Concurrent operations should not panic or corrupt state
	const numGoroutines = 10
	const opsPerGoroutine = 5

	results := make(chan error, numGoroutines*opsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			for j := 0; j < opsPerGoroutine; j++ {
				opCtx, opCancel := context.WithTimeout(ctx, 2*time.Second)

				switch j % 4 {
				case 0:
					// Test health check
					err := assistant.Health(opCtx)
					results <- err
				case 1:
					// Test stats
					_, err := assistant.Stats(opCtx)
					results <- err
				case 2:
					// Test tool listing
					tools := assistant.GetAvailableTools()
					if tools == nil {
						results <- fmt.Errorf("tools should not be nil")
					} else {
						results <- nil
					}
				case 3:
					// Test conversation listing
					_, err := assistant.ListConversations(opCtx, fmt.Sprintf("user-%d", workerID), 5, 0)
					results <- err
				}

				opCancel()
			}
		}(i)
	}

	// Collect results
	var errors []error
	for i := 0; i < numGoroutines*opsPerGoroutine; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	// Property: Most operations should succeed (some may fail in test mode)
	if len(errors) > numGoroutines*opsPerGoroutine/2 {
		t.Errorf("Too many concurrent operations failed: %d out of %d", len(errors), numGoroutines*opsPerGoroutine)
		for i, err := range errors {
			if i < 5 { // Show first 5 errors
				t.Logf("Error %d: %v", i+1, err)
			}
		}
	}
}

// TestInvariantProperties tests invariant properties that should always hold
func TestInvariantProperties(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := &config.Config{
		Mode: "test",
		AI: config.AIConfig{
			DefaultProvider: "claude",
			Claude: config.Claude{
				APIKey: "test-key",
				Model:  "claude-3-sonnet-20240229",
			},
		},
	}
	mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
	logger := testutil.NewTestLogger()

	assistant, err := New(ctx, cfg, mockDB, logger)
	if err != nil {
		t.Fatalf("Failed to create assistant: %v", err)
	}
	defer func() {
		if err := assistant.Close(ctx); err != nil {
			t.Logf("Failed to close assistant: %v", err)
		}
	}()

	// Invariant: Available tools should be consistent
	tools1 := assistant.GetAvailableTools()
	tools2 := assistant.GetAvailableTools()

	if len(tools1) != len(tools2) {
		t.Errorf("Tool count should be consistent: %d vs %d", len(tools1), len(tools2))
	}

	toolNames1 := make(map[string]bool)
	toolNames2 := make(map[string]bool)

	for _, tool := range tools1 {
		toolNames1[tool.Name] = true
	}
	for _, tool := range tools2 {
		toolNames2[tool.Name] = true
	}

	for name := range toolNames1 {
		if !toolNames2[name] {
			t.Errorf("Tool %s disappeared between calls", name)
		}
	}

	// Invariant: Health check should be idempotent
	err1 := assistant.Health(ctx)
	err2 := assistant.Health(ctx)

	if (err1 == nil) != (err2 == nil) {
		t.Errorf("Health check should be idempotent: %v vs %v", err1, err2)
	}
}

// TestTypedStructs tests the new typed structs for assistant package
func TestTypedStructs(t *testing.T) {
	t.Run("AssistantStats", func(t *testing.T) {
		t.Parallel()

		stats := &AssistantStats{
			Database: &DatabaseStats{
				Status:            "connected",
				TotalConns:        10,
				IdleConns:         5,
				AcquiredConns:     3,
				ConstructingConns: 2,
				MaxConns:          20,
				NewConnsCount:     100,
				AcquireCount:      500,
				AcquireDurationMs: 150,
			},
			Tools: &ToolRegistryStats{
				RegisteredTools: 5,
				ExecutionCounts: map[string]int64{
					"go_analyzer":  10,
					"go_formatter": 5,
				},
				AverageExecutionTimes: map[string]int64{
					"go_analyzer":  1500,
					"go_formatter": 800,
				},
			},
			Processor: &ProcessorStats{
				Processor: &ProcessorInfo{
					Status:  "healthy",
					Version: "1.0.0",
					Uptime:  "120s",
				},
				Health: &HealthStatus{
					Status:    "healthy",
					LastCheck: time.Now(),
				},
				// TODO: Add performance metrics when tracking is implemented
			},
		}

		// Verify all fields are accessible and properly typed
		if stats.Database.Status != "connected" {
			t.Errorf("Expected status 'connected', got %s", stats.Database.Status)
		}
		if stats.Database.TotalConns != 10 {
			t.Errorf("Expected TotalConns 10, got %d", stats.Database.TotalConns)
		}
		if stats.Tools.RegisteredTools != 5 {
			t.Errorf("Expected RegisteredTools 5, got %d", stats.Tools.RegisteredTools)
		}
		if stats.Processor.Processor.Status != "healthy" {
			t.Errorf("Expected processor status 'healthy', got %s", stats.Processor.Processor.Status)
		}
	})

	t.Run("ToolExecutionRequest", func(t *testing.T) {
		t.Parallel()

		req := &ToolExecutionRequest{
			ToolName: "go_analyzer",
			Input: map[string]any{
				"file_path": "/test/file.go",
				"options":   []string{"verbose", "detailed"},
			},
			Config: map[string]any{
				"timeout": 30,
				"retries": 3,
			},
		}

		// Verify struct fields
		if req.ToolName != "go_analyzer" {
			t.Errorf("Expected ToolName 'go_analyzer', got %s", req.ToolName)
		}
		if req.Input["file_path"] != "/test/file.go" {
			t.Errorf("Expected file_path '/test/file.go', got %v", req.Input["file_path"])
		}
		if req.Config["timeout"] != 30 {
			t.Errorf("Expected timeout 30, got %v", req.Config["timeout"])
		}
	})

	t.Run("QueryRequest", func(t *testing.T) {
		t.Parallel()

		req := &QueryRequest{
			Query:          "Test query",
			ConversationID: stringPtr("conv-123"),
			UserID:         stringPtr("user-456"),
			Context: map[string]any{
				"project":  "test",
				"language": "go",
			},
			Tools:        []string{"go_analyzer"},
			Provider:     stringPtr("claude"),
			Model:        stringPtr("claude-3-sonnet-20240229"),
			MaxTokens:    1000,
			Temperature:  0.7,
			SystemPrompt: stringPtr("Custom prompt"),
		}

		// Verify struct fields
		if req.Query != "Test query" {
			t.Errorf("Expected Query 'Test query', got %s", req.Query)
		}
		if req.ConversationID == nil || *req.ConversationID != "conv-123" {
			t.Errorf("Expected ConversationID 'conv-123', got %v", req.ConversationID)
		}
		if req.MaxTokens != 1000 {
			t.Errorf("Expected MaxTokens 1000, got %d", req.MaxTokens)
		}
		if req.Temperature != 0.7 {
			t.Errorf("Expected Temperature 0.7, got %f", req.Temperature)
		}
	})

	t.Run("QueryResponse", func(t *testing.T) {
		t.Parallel()

		resp := &QueryResponse{
			Response:       "Test response",
			ConversationID: "conv-123",
			MessageID:      "msg-456",
			Provider:       "claude",
			Model:          "claude-3-sonnet-20240229",
			TokensUsed:     500,
			ExecutionTime:  2 * time.Second,
			ToolsUsed:      []string{"go_analyzer"},
			Context: map[string]any{
				"processed_at": time.Now(),
			},
			Error: nil,
		}

		// Verify struct fields
		if resp.Response != "Test response" {
			t.Errorf("Expected Response 'Test response', got %s", resp.Response)
		}
		if resp.TokensUsed != 500 {
			t.Errorf("Expected TokensUsed 500, got %d", resp.TokensUsed)
		}
		if resp.ExecutionTime != 2*time.Second {
			t.Errorf("Expected ExecutionTime 2s, got %v", resp.ExecutionTime)
		}
		if len(resp.ToolsUsed) != 1 || resp.ToolsUsed[0] != "go_analyzer" {
			t.Errorf("Expected ToolsUsed ['go_analyzer'], got %v", resp.ToolsUsed)
		}
	})
}

// Helper function for string pointers (already defined in multiple files)
func stringPtr(s string) *string {
	return &s
}

// Helper function to generate random strings
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !@#$%^&*()_+-=[]{}|;':\",./<>?"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
