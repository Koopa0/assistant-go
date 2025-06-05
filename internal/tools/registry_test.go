package tools

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"
)

// TestNewRegistry tests registry creation
func TestNewRegistry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := NewRegistry(logger)

	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if registry.tools == nil {
		t.Error("NewRegistry() tools map is nil")
	}
	if registry.factories == nil {
		t.Error("NewRegistry() factories map is nil")
	}
	if registry.info == nil {
		t.Error("NewRegistry() info map is nil")
	}
	if registry.logger != logger {
		t.Error("NewRegistry() logger not set correctly")
	}
}

// TestRegistryRegister tests tool registration
func TestRegistryRegister(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := NewRegistry(logger)

	factory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "test_tool"}, nil
	}

	tests := []struct {
		name      string
		toolName  string
		factory   ToolFactory
		wantError bool
		errorMsg  string
		setupFunc func(*Registry)
	}{
		{
			name:      "successful_registration",
			toolName:  "test_tool",
			factory:   factory,
			wantError: false,
		},
		{
			name:      "empty_name",
			toolName:  "",
			factory:   factory,
			wantError: true,
			errorMsg:  "tool name cannot be empty",
		},
		{
			name:      "nil_factory",
			toolName:  "test_tool",
			factory:   nil,
			wantError: true,
			errorMsg:  "tool factory cannot be nil",
		},
		{
			name:     "duplicate_registration",
			toolName: "duplicate_tool",
			factory:  factory,
			setupFunc: func(r *Registry) {
				_ = r.Register("duplicate_tool", factory)
			},
			wantError: true,
			errorMsg:  "tool duplicate_tool is already registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc(registry)
			}

			err := registry.Register(tt.toolName, tt.factory)
			if tt.wantError {
				if err == nil {
					t.Errorf("Register() error = nil, want error")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("Register() error = %v, want %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Register() error = %v, want nil", err)
				}
				if !registry.IsRegistered(tt.toolName) {
					t.Errorf("Tool %s not registered after successful registration", tt.toolName)
				}
			}
		})
	}
}

// TestRegistryUnregister tests tool unregistration
func TestRegistryUnregister(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := NewRegistry(logger)

	factory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "test_tool"}, nil
	}

	// Register a tool first
	err := registry.Register("test_tool", factory)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	// Create an instance to test cleanup
	_, err = registry.GetTool("test_tool", nil)
	if err != nil {
		t.Fatalf("Failed to create tool instance: %v", err)
	}

	// Unregister the tool
	err = registry.Unregister("test_tool")
	if err != nil {
		t.Errorf("Unregister() error = %v, want nil", err)
	}

	// Verify tool is no longer registered
	if registry.IsRegistered("test_tool") {
		t.Error("Tool still registered after unregistration")
	}

	// Verify tool instance is removed
	if len(registry.tools) != 0 {
		t.Error("Tool instance not removed after unregistration")
	}

	// Unregistering non-existent tool should not error
	err = registry.Unregister("nonexistent_tool")
	if err != nil {
		t.Errorf("Unregister() for non-existent tool error = %v, want nil", err)
	}
}

// TestRegistryGetTool tests tool instance creation and retrieval
func TestRegistryGetTool(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := NewRegistry(logger)

	successFactory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "success_tool", config: map[string]interface{}{"timeout": config.Timeout}}, nil
	}

	failureFactory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return nil, errors.New("factory error")
	}

	// Register tools
	_ = registry.Register("success_tool", successFactory)
	_ = registry.Register("failure_tool", failureFactory)

	tests := []struct {
		name      string
		toolName  string
		config    *ToolConfig
		wantError bool
		errorMsg  string
	}{
		{
			name:     "successful_creation",
			toolName: "success_tool",
			config:   &ToolConfig{WorkingDir: "/test"},
		},
		{
			name:      "unregistered_tool",
			toolName:  "nonexistent_tool",
			wantError: true,
			errorMsg:  "tool nonexistent_tool is not registered",
		},
		{
			name:      "factory_failure",
			toolName:  "failure_tool",
			wantError: true,
			errorMsg:  "failed to create tool failure_tool: factory error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := registry.GetTool(tt.toolName, tt.config)
			if tt.wantError {
				if err == nil {
					t.Errorf("GetTool() error = nil, want error")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("GetTool() error = %v, want %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("GetTool() error = %v, want nil", err)
				}
				if tool == nil {
					t.Error("GetTool() returned nil tool")
				}
				if tool.Name() != tt.toolName {
					t.Errorf("GetTool() tool name = %s, want %s", tool.Name(), tt.toolName)
				}

				// Second call should return the same instance
				tool2, err2 := registry.GetTool(tt.toolName, tt.config)
				if err2 != nil {
					t.Errorf("GetTool() second call error = %v, want nil", err2)
				}
				if tool != tool2 {
					t.Error("GetTool() second call returned different instance")
				}
			}
		})
	}
}

// TestRegistryExecute tests tool execution through registry
func TestRegistryExecute(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := NewRegistry(logger)

	successFactory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{
			name:          "success_tool",
			executeResult: &ToolResult{Success: true, Data: &ToolResultData{Output: "test result"}},
		}, nil
	}

	errorFactory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{
			name:       "error_tool",
			executeErr: errors.New("execution error"),
		}, nil
	}

	nilResultFactory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "nil_result_tool"}, nil
	}

	// Register tools
	_ = registry.Register("success_tool", successFactory)
	_ = registry.Register("error_tool", errorFactory)
	_ = registry.Register("nil_result_tool", nilResultFactory)

	ctx := context.Background()
	input := &ToolInput{Parameters: map[string]interface{}{"test": "input"}}
	config := &ToolConfig{WorkingDir: "/test"}

	tests := []struct {
		name        string
		toolName    string
		wantSuccess bool
		wantError   bool
		wantData    string
	}{
		{
			name:        "successful_execution",
			toolName:    "success_tool",
			wantSuccess: true,
			wantData:    "test result",
		},
		{
			name:      "unregistered_tool",
			toolName:  "nonexistent_tool",
			wantError: true,
		},
		{
			name:      "execution_error",
			toolName:  "error_tool",
			wantError: true,
		},
		{
			name:        "nil_result",
			toolName:    "nil_result_tool",
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := registry.Execute(ctx, tt.toolName, input, config)

			if tt.wantError {
				if err == nil {
					t.Errorf("Execute() error = nil, want error")
				}
				if result == nil || result.Success {
					t.Error("Execute() should return failed result for error case")
				}
			} else {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
				}
				if result == nil {
					t.Fatal("Execute() returned nil result")
				}
				if result.Success != tt.wantSuccess {
					t.Errorf("Execute() Success = %v, want %v", result.Success, tt.wantSuccess)
				}
				if tt.wantData != "" && result.Data != nil && result.Data.Output != tt.wantData {
					t.Errorf("Execute() Data = %v, want %v", result.Data.Output, tt.wantData)
				}
				if result.ExecutionTime <= 0 {
					t.Error("Execute() should set execution time")
				}
			}
		})
	}
}

// TestRegistryListTools tests tool listing
func TestRegistryListTools(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := NewRegistry(logger)

	factory1 := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "tool1", description: "Tool 1"}, nil
	}
	factory2 := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "tool2", description: "Tool 2"}, nil
	}

	t.Run("empty_registry", func(t *testing.T) {
		tools := registry.ListTools()
		if len(tools) != 0 {
			t.Errorf("ListTools() empty registry returned %d tools, want 0", len(tools))
		}
	})

	// Register tools
	_ = registry.Register("tool1", factory1)
	_ = registry.Register("tool2", factory2)

	t.Run("with_factories_only", func(t *testing.T) {
		tools := registry.ListTools()
		if len(tools) != 2 {
			t.Errorf("ListTools() returned %d tools, want 2", len(tools))
		}

		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Name] = true
		}

		if !toolNames["tool1"] || !toolNames["tool2"] {
			t.Error("ListTools() did not return expected tools")
		}
	})

	// Create instances
	_, _ = registry.GetTool("tool1", nil)
	_, _ = registry.GetTool("tool2", nil)

	t.Run("with_instances", func(t *testing.T) {
		tools := registry.ListTools()
		if len(tools) != 2 {
			t.Errorf("ListTools() returned %d tools, want 2", len(tools))
		}

		for _, tool := range tools {
			if tool.Description == "" || tool.Description == "Tool description not available" {
				t.Error("ListTools() should return actual descriptions for instantiated tools")
			}
		}
	})
}

// TestRegistryGetToolInfo tests getting tool information
func TestRegistryGetToolInfo(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := NewRegistry(logger)

	factory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "test_tool", description: "Test tool description"}, nil
	}

	_ = registry.Register("test_tool", factory)

	tests := []struct {
		name      string
		toolName  string
		wantError bool
		checkInfo bool
	}{
		{
			name:      "existing_tool_factory_only",
			toolName:  "test_tool",
			checkInfo: true,
		},
		{
			name:      "nonexistent_tool",
			toolName:  "nonexistent",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := registry.GetToolInfo(tt.toolName)
			if tt.wantError {
				if err == nil {
					t.Errorf("GetToolInfo() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("GetToolInfo() error = %v, want nil", err)
				}
				if tt.checkInfo {
					if info.Name != tt.toolName {
						t.Errorf("GetToolInfo() Name = %s, want %s", info.Name, tt.toolName)
					}
					if !info.IsEnabled {
						t.Error("GetToolInfo() IsEnabled = false, want true")
					}
				}
			}
		})
	}

	// Test with instantiated tool
	_, _ = registry.GetTool("test_tool", nil)
	info, err := registry.GetToolInfo("test_tool")
	if err != nil {
		t.Fatalf("GetToolInfo() for instantiated tool error = %v", err)
	}
	if info.Description != "Test tool description" {
		t.Errorf("GetToolInfo() Description = %s, want %s", info.Description, "Test tool description")
	}
}

// TestRegistryHealth tests registry health check
func TestRegistryHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := NewRegistry(logger)

	healthyFactory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "healthy_tool"}, nil
	}

	unhealthyFactory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "unhealthy_tool", healthErr: errors.New("health check failed")}, nil
	}

	_ = registry.Register("healthy_tool", healthyFactory)
	_ = registry.Register("unhealthy_tool", unhealthyFactory)

	ctx := context.Background()

	t.Run("no_instances", func(t *testing.T) {
		err := registry.Health(ctx)
		if err != nil {
			t.Errorf("Health() with no instances error = %v, want nil", err)
		}
	})

	// Create healthy instance
	_, _ = registry.GetTool("healthy_tool", nil)

	t.Run("healthy_instances", func(t *testing.T) {
		err := registry.Health(ctx)
		if err != nil {
			t.Errorf("Health() with healthy instances error = %v, want nil", err)
		}
	})

	// Create unhealthy instance
	_, _ = registry.GetTool("unhealthy_tool", nil)

	t.Run("unhealthy_instance", func(t *testing.T) {
		err := registry.Health(ctx)
		if err == nil {
			t.Error("Health() with unhealthy instance error = nil, want error")
		}
	})
}

// TestRegistryStats tests registry statistics
func TestRegistryStats(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := NewRegistry(logger)

	factory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "test_tool"}, nil
	}

	ctx := context.Background()

	t.Run("empty_registry", func(t *testing.T) {
		stats, err := registry.Stats(ctx)
		if err != nil {
			t.Errorf("Stats() error = %v, want nil", err)
		}
		if stats["registered_factories"] != 0 {
			t.Errorf("Stats() registered_factories = %v, want 0", stats["registered_factories"])
		}
		if stats["active_instances"] != 0 {
			t.Errorf("Stats() active_instances = %v, want 0", stats["active_instances"])
		}
	})

	// Register tool
	_ = registry.Register("test_tool", factory)

	t.Run("with_factory", func(t *testing.T) {
		stats, err := registry.Stats(ctx)
		if err != nil {
			t.Errorf("Stats() error = %v, want nil", err)
		}
		if stats["registered_factories"] != 1 {
			t.Errorf("Stats() registered_factories = %v, want 1", stats["registered_factories"])
		}
		if stats["active_instances"] != 0 {
			t.Errorf("Stats() active_instances = %v, want 0", stats["active_instances"])
		}
	})

	// Create instance
	_, _ = registry.GetTool("test_tool", nil)

	t.Run("with_instance", func(t *testing.T) {
		stats, err := registry.Stats(ctx)
		if err != nil {
			t.Errorf("Stats() error = %v, want nil", err)
		}
		if stats["registered_factories"] != 1 {
			t.Errorf("Stats() registered_factories = %v, want 1", stats["registered_factories"])
		}
		if stats["active_instances"] != 1 {
			t.Errorf("Stats() active_instances = %v, want 1", stats["active_instances"])
		}
	})
}

// TestRegistryClose tests registry cleanup
func TestRegistryClose(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := NewRegistry(logger)

	normalFactory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "normal_tool"}, nil
	}

	errorCloseFactory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "error_close_tool", closeErr: errors.New("close error")}, nil
	}

	_ = registry.Register("normal_tool", normalFactory)
	_ = registry.Register("error_close_tool", errorCloseFactory)

	// Create instances
	_, _ = registry.GetTool("normal_tool", nil)
	_, _ = registry.GetTool("error_close_tool", nil)

	ctx := context.Background()
	err := registry.Close(ctx)

	// Should return the last error encountered
	if err == nil {
		t.Error("Close() error = nil, want error from failed close")
	}

	// Verify all maps are cleared
	if len(registry.tools) != 0 {
		t.Error("Close() did not clear tools map")
	}
	if len(registry.factories) != 0 {
		t.Error("Close() did not clear factories map")
	}
	if len(registry.info) != 0 {
		t.Error("Close() did not clear info map")
	}
}

// TestRegistryConcurrency tests registry thread safety
func TestRegistryConcurrency(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := NewRegistry(logger)

	factory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "concurrent_tool"}, nil
	}

	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent registration
	t.Run("concurrent_registration", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				toolName := "tool_" + string(rune('0'+id))
				_ = registry.Register(toolName, factory)
			}(i)
		}
		wg.Wait()
	})

	// Test concurrent tool creation
	t.Run("concurrent_tool_creation", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				toolName := "tool_" + string(rune('0'+id))
				_, _ = registry.GetTool(toolName, nil)
			}(i)
		}
		wg.Wait()
	})

	// Test concurrent execution
	t.Run("concurrent_execution", func(t *testing.T) {
		ctx := context.Background()
		input := &ToolInput{Parameters: map[string]interface{}{"test": "input"}}

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				toolName := "tool_" + string(rune('0'+id))
				_, _ = registry.Execute(ctx, toolName, input, nil)
			}(i)
		}
		wg.Wait()
	})
}

// TestRegistryGetToolCategory tests category determination
func TestRegistryGetToolCategory(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := NewRegistry(logger)

	tests := []struct {
		toolName string
		expected string
	}{
		{"postgres", "database"},
		{"database", "database"},
		{"kubernetes", "infrastructure"},
		{"k8s", "infrastructure"},
		{"docker", "containers"},
		{"cloudflare", "cloud"},
		{"search", "search"},
		{"searxng", "search"},
		{"langchain", "ai"},
		{"agent", "ai"},
		{"godev", "development"},
		{"go", "development"},
		{"unknown_tool", "general"},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			category := registry.getToolCategory(tt.toolName)
			if category != tt.expected {
				t.Errorf("getToolCategory(%s) = %s, want %s", tt.toolName, category, tt.expected)
			}
		})
	}
}

// BenchmarkRegistry benchmarks registry operations
func BenchmarkRegistry(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	registry := NewRegistry(logger)

	factory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
		return &testTool{name: "bench_tool"}, nil
	}

	_ = registry.Register("bench_tool", factory)

	b.Run("GetTool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = registry.GetTool("bench_tool", nil)
		}
	})

	b.Run("Execute", func(b *testing.B) {
		ctx := context.Background()
		input := &ToolInput{Parameters: map[string]interface{}{"test": "input"}}

		for i := 0; i < b.N; i++ {
			_, _ = registry.Execute(ctx, "bench_tool", input, nil)
		}
	})

	b.Run("IsRegistered", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = registry.IsRegistered("bench_tool")
		}
	})

	b.Run("ListTools", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = registry.ListTools()
		}
	})
}

// testTool is a test implementation of the Tool interface
type testTool struct {
	name          string
	description   string
	config        map[string]interface{}
	executeResult *ToolResult
	executeErr    error
	healthErr     error
	closeErr      error
}

func (t *testTool) Name() string {
	if t.name == "" {
		return "test_tool"
	}
	return t.name
}

func (t *testTool) Description() string {
	if t.description == "" {
		return "Test tool description"
	}
	return t.description
}

func (t *testTool) Parameters() *ToolParametersSchema {
	return &ToolParametersSchema{
		Type:       "object",
		Properties: map[string]ToolParameter{},
		Required:   []string{},
	}
}

func (t *testTool) Execute(ctx context.Context, input *ToolInput) (*ToolResult, error) {
	if t.executeErr != nil {
		return nil, t.executeErr
	}
	if t.executeResult != nil {
		return t.executeResult, nil
	}
	return &ToolResult{Success: true, Data: &ToolResultData{Output: "test result"}}, nil
}

func (t *testTool) Health(ctx context.Context) error {
	return t.healthErr
}

func (t *testTool) Close(ctx context.Context) error {
	return t.closeErr
}

// TestRegistryEdgeCases tests edge cases and error scenarios
func TestRegistryEdgeCases(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := NewRegistry(logger)

	t.Run("execute_with_cancelled_context", func(t *testing.T) {
		factory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
			return &slowTool{}, nil
		}
		_ = registry.Register("slow_tool", factory)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		result, err := registry.Execute(ctx, "slow_tool", nil, nil)
		if err == nil {
			t.Error("Execute() with cancelled context should return error")
		}
		if result == nil || result.Success {
			t.Error("Execute() with cancelled context should return failed result")
		}
	})

	t.Run("tool_factory_panic_recovery", func(t *testing.T) {
		panicFactory := func(config *ToolConfig, logger *slog.Logger) (Tool, error) {
			panic("factory panic")
		}
		_ = registry.Register("panic_tool", panicFactory)

		// This should not crash the test
		tool, err := registry.GetTool("panic_tool", nil)
		if err == nil {
			t.Error("GetTool() with panicking factory should return error")
		}
		if tool != nil {
			t.Error("GetTool() with panicking factory should return nil tool")
		}
	})
}

// slowTool simulates a slow tool for context cancellation testing
type slowTool struct{}

func (s *slowTool) Name() string        { return "slow_tool" }
func (s *slowTool) Description() string { return "Slow tool" }
func (s *slowTool) Parameters() *ToolParametersSchema {
	return &ToolParametersSchema{Type: "object", Properties: map[string]ToolParameter{}, Required: []string{}}
}
func (s *slowTool) Execute(ctx context.Context, input *ToolInput) (*ToolResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return &ToolResult{Success: true}, nil
	}
}
func (s *slowTool) Health(ctx context.Context) error { return nil }
func (s *slowTool) Close(ctx context.Context) error  { return nil }
