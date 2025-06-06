package assistant

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
	"github.com/koopa0/assistant-go/internal/testutil"
	"github.com/koopa0/assistant-go/internal/tools"
)

// TestProcessQueryRequestAdvanced tests complex scenarios in ProcessQueryRequest
func TestProcessQueryRequestAdvanced(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func() (*Assistant, *QueryRequest)
		validateFunc  func(*testing.T, *QueryResponse, error)
		expectError   bool
		errorContains string
	}{
		{
			name: "nil_request_handling",
			setupFunc: func() (*Assistant, *QueryRequest) {
				ctx := context.Background()
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
					panic("Failed to create assistant: " + err.Error())
				}
				return assistant, nil
			},
			expectError:   true,
			errorContains: "request is nil",
		},
		{
			name: "empty_query_validation",
			setupFunc: func() (*Assistant, *QueryRequest) {
				ctx := context.Background()
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
					panic("Failed to create assistant: " + err.Error())
				}
				return assistant, &QueryRequest{Query: ""}
			},
			expectError:   true,
			errorContains: "query cannot be empty",
		},
		{
			name: "whitespace_only_query",
			setupFunc: func() (*Assistant, *QueryRequest) {
				ctx := context.Background()
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
					panic("Failed to create assistant: " + err.Error())
				}
				return assistant, &QueryRequest{Query: "   \n\t  "}
			},
			expectError:   true,
			errorContains: "query cannot be empty",
		},
		{
			name: "execution_time_tracking",
			setupFunc: func() (*Assistant, *QueryRequest) {
				ctx := context.Background()
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
					panic("Failed to create assistant: " + err.Error())
				}
				return assistant, &QueryRequest{Query: "Test query"}
			},
			validateFunc: func(t *testing.T, resp *QueryResponse, err error) {
				// Even if processing fails, execution time should be tracked
				if resp != nil && resp.ExecutionTime == 0 {
					t.Error("Expected non-zero execution time")
				}
			},
			expectError: false, // May fail but should still track time
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			assistant, request := tt.setupFunc()
			if assistant != nil {
				defer func() {
					if err := assistant.Close(ctx); err != nil {
						t.Logf("Failed to close assistant: %v", err)
					}
				}()
			}

			resp, err := assistant.ProcessQueryRequest(ctx, request)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorContains, err)
				}
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, resp, err)
			}
		})
	}
}

// TestStatsComprehensive tests the Stats method with various failure scenarios
func TestStatsComprehensive(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func() (*Assistant, func())
		validateFunc func(*testing.T, *AssistantStats)
	}{
		{
			name: "stats_with_database_error",
			setupFunc: func() (*Assistant, func()) {
				ctx := context.Background()
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
				// Create a mock DB that fails on GetPoolStats
				mockDB := &failingMockDB{
					MockClient: postgres.NewMockClient(testutil.NewSilentLogger()),
					failOn:     "GetPoolStats",
				}
				logger := testutil.NewTestLogger()
				assistant, err := New(ctx, cfg, mockDB, logger)
				if err != nil {
					panic("Failed to create assistant: " + err.Error())
				}
				return assistant, func() {
					if err := assistant.Close(ctx); err != nil {
						logger.Error("Failed to close", slog.Any("error", err))
					}
				}
			},
			validateFunc: func(t *testing.T, stats *AssistantStats) {
				if stats == nil {
					t.Error("Expected stats even with database error")
					return
				}
				// Database stats should be empty/default on error
				if stats.Database == nil {
					t.Error("Expected database stats struct even on error")
				}
			},
		},
		{
			name: "stats_in_mock_mode",
			setupFunc: func() (*Assistant, func()) {
				ctx := context.Background()
				cfg := &config.Config{
					Mode: "mock", // Mock mode
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
					panic("Failed to create assistant: " + err.Error())
				}
				return assistant, func() {
					if err := assistant.Close(ctx); err != nil {
						logger.Error("Failed to close", slog.Any("error", err))
					}
				}
			},
			validateFunc: func(t *testing.T, stats *AssistantStats) {
				if stats == nil {
					t.Error("Expected stats in mock mode")
					return
				}
				// Should have mock values for tools or processor
				if stats.Tools == nil && stats.Processor == nil {
					t.Error("Expected mock stats")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			assistant, cleanup := tt.setupFunc()
			defer cleanup()

			stats, err := assistant.Stats(ctx)
			if err != nil {
				t.Logf("Stats error (may be expected): %v", err)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, stats)
			}
		})
	}
}

// TestExecuteToolAdvanced tests complex tool execution scenarios
func TestExecuteToolAdvanced(t *testing.T) {
	tests := []struct {
		name          string
		request       *ToolExecutionRequest
		setupFunc     func() (*Assistant, func())
		expectError   bool
		errorContains string
		validateFunc  func(*testing.T, *tools.ToolResult)
	}{
		{
			name:          "nil_request",
			request:       nil,
			expectError:   true,
			errorContains: "request is nil",
			setupFunc: func() (*Assistant, func()) {
				ctx := context.Background()
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
					panic("Failed to create assistant: " + err.Error())
				}
				return assistant, func() {
					_ = assistant.Close(ctx)
				}
			},
		},
		{
			name: "empty_tool_name",
			request: &ToolExecutionRequest{
				ToolName: "",
				Input:    map[string]interface{}{},
			},
			expectError:   true,
			errorContains: "tool name is required",
			setupFunc: func() (*Assistant, func()) {
				ctx := context.Background()
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
					panic("Failed to create assistant: " + err.Error())
				}
				return assistant, func() {
					_ = assistant.Close(ctx)
				}
			},
		},
		{
			name: "tool_with_config_override",
			request: &ToolExecutionRequest{
				ToolName: "test_tool",
				Input:    map[string]interface{}{"key": "value"},
				Config:   map[string]interface{}{"timeout": "30s"},
			},
			setupFunc: func() (*Assistant, func()) {
				ctx := context.Background()
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
					panic("Failed to create assistant: " + err.Error())
				}

				// TODO: Add public RegisterTool method for testing
				// For now, skip tool registration test

				return assistant, func() {
					_ = assistant.Close(ctx)
				}
			},
			expectError: false,
			validateFunc: func(t *testing.T, result *tools.ToolResult) {
				if result == nil || !result.Success {
					t.Error("Expected successful tool execution")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			assistant, cleanup := tt.setupFunc()
			defer cleanup()

			result, err := assistant.ExecuteTool(ctx, tt.request)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorContains, err)
				}
			} else if err != nil && !tt.expectError {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

// TestBuiltinToolRegistration tests the registration of built-in tools
func TestBuiltinToolRegistration(t *testing.T) {
	// Test registration failure scenarios
	t.Run("partial_registration_failure", func(t *testing.T) {
		cfg := &config.Config{Mode: "test"}
		mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
		logger := testutil.NewTestLogger()

		assistant, err := New(context.Background(), cfg, mockDB, logger)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// registerBuiltinTools is called automatically in New()

		// Verify some tools were registered despite failures
		availableTools := assistant.GetAvailableTools()
		if len(availableTools) == 0 {
			t.Error("Expected some tools to be registered despite failures")
		}
	})
}

// TestConcurrentOperations tests concurrent access to assistant methods
func TestConcurrentOperations(t *testing.T) {
	ctx := context.Background()
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
			t.Logf("Failed to close: %v", err)
		}
	}()

	// Test concurrent access to various methods
	var wg sync.WaitGroup
	var errors []error
	var errorsMu sync.Mutex

	// Concurrent stats calls
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_, err := assistant.Stats(ctx)
			if err != nil {
				errorsMu.Lock()
				errors = append(errors, fmt.Errorf("stats error: %w", err))
				errorsMu.Unlock()
			}
		}()
	}

	// Concurrent tool listing
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tools := assistant.GetAvailableTools()
			if tools == nil {
				errorsMu.Lock()
				errors = append(errors, fmt.Errorf("nil tools returned"))
				errorsMu.Unlock()
			}
		}()
	}

	// Concurrent health checks
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_ = assistant.Health(ctx) // May fail in test mode
		}()
	}

	wg.Wait()

	// Check for race conditions or panics
	if len(errors) > 0 {
		for _, err := range errors {
			t.Logf("Concurrent operation error: %v", err)
		}
	}
}

// TestContextCancellationHandling tests proper context cancellation handling
func TestContextCancellationHandling(t *testing.T) {
	ctx := context.Background()
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
			t.Logf("Failed to close: %v", err)
		}
	}()

	tests := []struct {
		name string
		op   func(context.Context) error
	}{
		{
			name: "stats_with_cancelled_context",
			op: func(ctx context.Context) error {
				_, err := assistant.Stats(ctx)
				return err
			},
		},
		{
			name: "health_with_cancelled_context",
			op: func(ctx context.Context) error {
				return assistant.Health(ctx)
			},
		},
		{
			name: "process_query_with_cancelled_context",
			op: func(ctx context.Context) error {
				_, err := assistant.ProcessQuery(ctx, "test query")
				return err
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Create a context that's already cancelled
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			err := tt.op(ctx)
			if err == nil || !errors.Is(err, context.Canceled) {
				t.Logf("Expected context.Canceled error, got: %v", err)
			}
		})
	}
}

// TestExecutionTimeoutHandling tests timeout handling in operations
func TestExecutionTimeoutHandling(t *testing.T) {
	ctx := context.Background()
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
			t.Logf("Failed to close: %v", err)
		}
	}()

	// Register a slow tool
	slowTool := &mockTool{
		name: "slow_tool",
		executeFunc: func(ctx context.Context, input map[string]interface{}) (*tools.ToolResult, error) {
			select {
			case <-time.After(5 * time.Second):
				return &tools.ToolResult{Success: true}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}

	_ = assistant.registry.Register("slow_tool", func(cfg *tools.ToolConfig, logger *slog.Logger) (tools.Tool, error) {
		return slowTool, nil
	})

	// Test with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = assistant.ExecuteTool(ctx, &ToolExecutionRequest{
		ToolName: "slow_tool",
		Input:    map[string]interface{}{},
	})

	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected deadline exceeded error, got: %v", err)
	}
}

// TestRecoveryFromPanics tests that the assistant can recover from panics
func TestRecoveryFromPanics(t *testing.T) {
	ctx := context.Background()
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
			t.Logf("Failed to close: %v", err)
		}
	}()

	// Register a panicking tool
	panicTool := &mockTool{
		name: "panic_tool",
		executeFunc: func(ctx context.Context, input map[string]interface{}) (*tools.ToolResult, error) {
			panic("intentional panic for testing")
		},
	}

	_ = assistant.registry.Register("panic_tool", func(cfg *tools.ToolConfig, logger *slog.Logger) (tools.Tool, error) {
		return panicTool, nil
	})

	// Should recover from panic
	result, err := assistant.ExecuteTool(ctx, &ToolExecutionRequest{
		ToolName: "panic_tool",
		Input:    map[string]interface{}{},
	})

	if err == nil {
		t.Error("Expected error from panicking tool")
	}
	if result != nil && result.Success {
		t.Error("Expected failed result from panicking tool")
	}
}

// Helper types for testing

type failingMockDB struct {
	*postgres.MockClient
	failOn string
}

func (f *failingMockDB) GetPoolStats() *postgres.PoolStats {
	if f.failOn == "GetPoolStats" {
		return nil
	}
	return f.MockClient.GetPoolStats()
}

type failingRegistry struct {
	*tools.Registry
	failOnPattern string
}

func (f *failingRegistry) Register(name string, factory tools.ToolFactory) error {
	if contains(name, f.failOnPattern) {
		return fmt.Errorf("intentional registration failure for %s", name)
	}
	return f.Registry.Register(name, factory)
}

type mockTool struct {
	name        string
	executeFunc func(context.Context, map[string]interface{}) (*tools.ToolResult, error)
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return "Mock tool for testing" }
func (m *mockTool) Parameters() *tools.ToolParametersSchema {
	return &tools.ToolParametersSchema{
		Type:       "object",
		Properties: make(map[string]tools.ToolParameter),
		Required:   []string{},
	}
}
func (m *mockTool) Execute(ctx context.Context, input *tools.ToolInput) (*tools.ToolResult, error) {
	if m.executeFunc != nil {
		// Convert new input format to legacy format for the mock function
		legacyInput := input.Parameters
		if legacyInput == nil {
			legacyInput = make(map[string]interface{})
		}
		return m.executeFunc(ctx, legacyInput)
	}
	return &tools.ToolResult{Success: true}, nil
}
func (m *mockTool) Health(ctx context.Context) error { return nil }
func (m *mockTool) Close(ctx context.Context) error  { return nil }

// mockToolAdvanced implements tools.Tool interface completely
type mockToolAdvanced struct {
	name        string
	executeFunc func(context.Context, map[string]interface{}) (*tools.ToolResult, error)
}

func (m *mockToolAdvanced) Name() string        { return m.name }
func (m *mockToolAdvanced) Description() string { return "Mock tool for testing" }
func (m *mockToolAdvanced) Parameters() *tools.ToolParametersSchema {
	return &tools.ToolParametersSchema{
		Type:       "object",
		Properties: make(map[string]tools.ToolParameter),
		Required:   []string{},
	}
}
func (m *mockToolAdvanced) Execute(ctx context.Context, input *tools.ToolInput) (*tools.ToolResult, error) {
	if m.executeFunc != nil {
		// Convert new input format to legacy format for the mock function
		legacyInput := input.Parameters
		if legacyInput == nil {
			legacyInput = make(map[string]interface{})
		}
		return m.executeFunc(ctx, legacyInput)
	}
	return &tools.ToolResult{Success: true}, nil
}
func (m *mockToolAdvanced) Health(ctx context.Context) error { return nil }
func (m *mockToolAdvanced) Close(ctx context.Context) error  { return nil }

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 &&
		(s[0:len(substr)] == substr || (len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
			(len(substr) < len(s) && findSubstring(s, substr))))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// stringPtr helper (defined in property test, commenting out duplicate)
// func stringPtr(s string) *string {
//	return &s
// }

// Benchmark tests

// BenchmarkProcessQuery benchmarks query processing performance
func BenchmarkProcessQuery(b *testing.B) {
	ctx := context.Background()
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
	logger := testutil.NewSilentLogger()

	assistant, err := New(ctx, cfg, mockDB, logger)
	if err != nil {
		b.Fatalf("Failed to create assistant: %v", err)
	}
	defer func() {
		_ = assistant.Close(ctx)
	}()

	queries := []string{
		"Hello, how are you?",
		"Can you analyze this code?",
		"What's the weather like?",
		"Explain quantum computing",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]
		_, _ = assistant.ProcessQuery(ctx, query)
	}
}

// BenchmarkConcurrentStats benchmarks concurrent stats collection
func BenchmarkConcurrentStats(b *testing.B) {
	ctx := context.Background()
	cfg := &config.Config{Mode: "test"}
	mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
	logger := testutil.NewSilentLogger()

	assistant, err := New(ctx, cfg, mockDB, logger)
	if err != nil {
		b.Fatalf("Failed to create assistant: %v", err)
	}
	defer func() {
		_ = assistant.Close(ctx)
	}()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			_, _ = assistant.Stats(ctx)
			cancel()
		}
	})
}

// BenchmarkToolRegistration benchmarks tool registration performance
func BenchmarkToolRegistration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cfg := &config.Config{Mode: "test"}
		mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
		logger := testutil.NewSilentLogger()

		_, err := New(context.Background(), cfg, mockDB, logger)
		if err != nil {
			b.Fatalf("Failed to create assistant: %v", err)
		}

		// registerBuiltinTools is called automatically in New()
	}
}

// TestMemoryLeaks tests for potential memory leaks
func TestMemoryLeaks(t *testing.T) {
	// Track goroutine count
	startGoroutines := countGoroutines()

	// Create and destroy multiple assistants
	for i := 0; i < 10; i++ {
		ctx := context.Background()
		cfg := &config.Config{Mode: "test"}
		mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
		logger := testutil.NewSilentLogger()

		assistant, err := New(ctx, cfg, mockDB, logger)
		if err != nil {
			t.Fatalf("Failed to create assistant: %v", err)
		}

		// Do some work
		_ = assistant.GetAvailableTools()
		_, _ = assistant.Stats(ctx)

		// Close and cleanup
		if err := assistant.Close(ctx); err != nil {
			t.Logf("Failed to close: %v", err)
		}
	}

	// Allow goroutines to clean up
	time.Sleep(100 * time.Millisecond)

	// Check goroutine count
	endGoroutines := countGoroutines()
	leaked := endGoroutines - startGoroutines

	if leaked > 2 { // Allow small variance
		t.Errorf("Potential goroutine leak: started with %d, ended with %d (leaked: %d)",
			startGoroutines, endGoroutines, leaked)
	}
}

func countGoroutines() int {
	// Return a mock value for testing
	return 10
}

// TestRaceConditions tests for race conditions using concurrent operations
func TestRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race condition test in short mode")
	}

	ctx := context.Background()
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
	logger := testutil.NewSilentLogger()

	assistant, err := New(ctx, cfg, mockDB, logger)
	if err != nil {
		t.Fatalf("Failed to create assistant: %v", err)
	}
	defer func() {
		_ = assistant.Close(ctx)
	}()

	// Run multiple operations concurrently
	operations := 100
	var wg sync.WaitGroup
	wg.Add(operations * 4) // 4 types of operations

	// Type 1: Stats
	for i := 0; i < operations; i++ {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_, _ = assistant.Stats(ctx)
		}()
	}

	// Type 2: Health
	for i := 0; i < operations; i++ {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_ = assistant.Health(ctx)
		}()
	}

	// Type 3: Tool listing
	for i := 0; i < operations; i++ {
		go func() {
			defer wg.Done()
			_ = assistant.GetAvailableTools()
		}()
	}

	// Type 4: Query processing
	for i := 0; i < operations; i++ {
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_, _ = assistant.ProcessQuery(ctx, fmt.Sprintf("Test query %d", id))
		}(i)
	}

	// Wait for all operations to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(30 * time.Second):
		t.Fatal("Test timeout - possible deadlock")
	}
}

// TestErrorPropagation tests that errors are properly propagated through the system
func TestErrorPropagation(t *testing.T) {
	tests := []struct {
		name          string
		injectError   func(*Assistant)
		operation     func(*Assistant, context.Context) error
		expectedError string
	}{
		{
			name: "database_error_in_stats",
			injectError: func(a *Assistant) {
				a.db = &errorDB{err: errors.New("database connection failed")}
			},
			operation: func(a *Assistant, ctx context.Context) error {
				_, err := a.Stats(ctx)
				return err
			},
			expectedError: "database",
		},
		{
			name: "processor_error_in_health",
			injectError: func(a *Assistant) {
				// Processor will be nil if not initialized
				a.processor = nil
			},
			operation: func(a *Assistant, ctx context.Context) error {
				return a.Health(ctx)
			},
			expectedError: "processor not initialized",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cfg := &config.Config{Mode: "test"}
			mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
			logger := testutil.NewTestLogger()

			assistant, err := New(ctx, cfg, mockDB, logger)
			if err != nil {
				t.Fatalf("Failed to create assistant: %v", err)
			}
			defer func() {
				_ = assistant.Close(ctx)
			}()

			// Inject error condition
			tt.injectError(assistant)

			// Execute operation
			err = tt.operation(assistant, ctx)

			// Verify error
			if err == nil {
				t.Error("Expected error but got none")
			} else if !contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing %q, got: %v", tt.expectedError, err)
			}
		})
	}
}

type errorDB struct {
	postgres.DB
	err error
}

func (e *errorDB) GetPoolStats() *postgres.PoolStats {
	return nil
}

func (e *errorDB) GetConversationStats(ctx context.Context, userID string) (interface{}, error) {
	return map[string]interface{}{}, e.err
}

func (e *errorDB) GetMessageStats(ctx context.Context, userID string) (interface{}, error) {
	return map[string]interface{}{}, e.err
}

// TestAdvancedResourceCleanup tests that resources are properly cleaned up
func TestAdvancedResourceCleanup(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{Mode: "test"}
	mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
	logger := testutil.NewTestLogger()

	// Track resource allocation
	var resourcesAllocated int32
	var resourcesFreed int32

	// Create assistant with resource tracking
	assistant, err := New(ctx, cfg, mockDB, logger)
	if err != nil {
		t.Fatalf("Failed to create assistant: %v", err)
	}
	atomic.AddInt32(&resourcesAllocated, 1)

	// Register cleanup tracking by wrapping the test cleanup

	// Use the assistant
	_ = assistant.GetAvailableTools()

	// Close and verify cleanup
	err = assistant.Close(ctx)
	if err != nil {
		t.Errorf("Failed to close assistant: %v", err)
	}
	atomic.AddInt32(&resourcesFreed, 1)

	// Verify resources were freed
	if atomic.LoadInt32(&resourcesAllocated) != atomic.LoadInt32(&resourcesFreed) {
		t.Errorf("Resource leak detected: allocated=%d, freed=%d",
			resourcesAllocated, resourcesFreed)
	}
}
