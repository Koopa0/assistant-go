package tools

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
)

// TestBaseTool tests the BaseTool implementation
func TestBaseTool(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name        string
		toolName    string
		description string
		category    string
		config      map[string]interface{}
		wantName    string
		wantDesc    string
		wantCat     string
	}{
		{
			name:        "basic_tool_creation",
			toolName:    "test_tool",
			description: "Test tool description",
			category:    "testing",
			config:      map[string]interface{}{"key": "value"},
			wantName:    "test_tool",
			wantDesc:    "Test tool description",
			wantCat:     "testing",
		},
		{
			name:        "empty_config_tool",
			toolName:    "empty_tool",
			description: "Tool with no config",
			category:    "general",
			config:      nil,
			wantName:    "empty_tool",
			wantDesc:    "Tool with no config",
			wantCat:     "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewBaseTool(tt.toolName, tt.description, tt.category, tt.config, logger)

			if got := tool.Name(); got != tt.wantName {
				t.Errorf("Name() = %v, want %v", got, tt.wantName)
			}

			if got := tool.Description(); got != tt.wantDesc {
				t.Errorf("Description() = %v, want %v", got, tt.wantDesc)
			}

			if got := tool.Category(); got != tt.wantCat {
				t.Errorf("Category() = %v, want %v", got, tt.wantCat)
			}

			if got := tool.Version(); got != "1.0.0" {
				t.Errorf("Version() = %v, want %v", got, "1.0.0")
			}
		})
	}
}

// TestBaseToolConfig tests configuration methods
func TestBaseToolConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name   string
		config map[string]interface{}
		tests  []configTest
	}{
		{
			name: "various_config_types",
			config: map[string]interface{}{
				"string_key":  "test_value",
				"int_key":     42,
				"float_key":   3.14,
				"bool_key":    true,
				"missing_key": nil,
			},
			tests: []configTest{
				{key: "string_key", expectString: true, expectedString: "test_value"},
				{key: "int_key", expectInt: true, expectedInt: 42},
				{key: "float_key", expectInt: true, expectedInt: 3}, // float64 converted to int
				{key: "bool_key", expectBool: true, expectedBool: true},
				{key: "nonexistent", expectString: false, expectedString: ""},
			},
		},
		{
			name:   "nil_config",
			config: nil,
			tests: []configTest{
				{key: "any_key", expectString: false, expectedString: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := NewBaseTool("test", "desc", "cat", tt.config, logger)

			for _, ct := range tt.tests {
				t.Run(ct.key, func(t *testing.T) {
					// Test GetConfig
					value, exists := tool.GetConfig(ct.key)
					if ct.expectString || ct.expectInt || ct.expectBool {
						if !exists {
							t.Errorf("GetConfig(%s) exists = false, want true", ct.key)
						}
					}

					// Test GetConfigString
					if ct.expectString {
						str, ok := tool.GetConfigString(ct.key)
						if !ok {
							t.Errorf("GetConfigString(%s) ok = false, want true", ct.key)
						}
						if str != ct.expectedString {
							t.Errorf("GetConfigString(%s) = %s, want %s", ct.key, str, ct.expectedString)
						}
					} else {
						_, ok := tool.GetConfigString(ct.key)
						if ok {
							t.Errorf("GetConfigString(%s) ok = true, want false", ct.key)
						}
					}

					// Test GetConfigInt
					if ct.expectInt {
						intVal, ok := tool.GetConfigInt(ct.key)
						if !ok {
							t.Errorf("GetConfigInt(%s) ok = false, want true", ct.key)
						}
						if intVal != ct.expectedInt {
							t.Errorf("GetConfigInt(%s) = %d, want %d", ct.key, intVal, ct.expectedInt)
						}
					}

					// Test GetConfigBool
					if ct.expectBool {
						boolVal, ok := tool.GetConfigBool(ct.key)
						if !ok {
							t.Errorf("GetConfigBool(%s) ok = false, want true", ct.key)
						}
						if boolVal != ct.expectedBool {
							t.Errorf("GetConfigBool(%s) = %t, want %t", ct.key, boolVal, ct.expectedBool)
						}
					}

					_ = value // Use value to avoid unused variable
				})
			}
		})
	}
}

type configTest struct {
	key            string
	expectString   bool
	expectedString string
	expectInt      bool
	expectedInt    int
	expectBool     bool
	expectedBool   bool
}

// TestBaseToolValidation tests input validation
func TestBaseToolValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tool := &mockTool{
		BaseTool: NewBaseTool("test_tool", "Test tool", "testing", nil, logger),
		params: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"required_string": map[string]interface{}{
					"type": "string",
				},
				"optional_int": map[string]interface{}{
					"type": "integer",
				},
				"optional_bool": map[string]interface{}{
					"type": "boolean",
				},
				"optional_array": map[string]interface{}{
					"type": "array",
				},
				"optional_object": map[string]interface{}{
					"type": "object",
				},
			},
			"required": []string{"required_string"},
		},
	}

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid_input",
			input: map[string]interface{}{
				"required_string": "test",
				"optional_int":    42,
				"optional_bool":   true,
			},
			wantError: false,
		},
		{
			name:      "nil_input",
			input:     nil,
			wantError: true,
			errorMsg:  "input cannot be nil",
		},
		{
			name:      "missing_required_param",
			input:     map[string]interface{}{"optional_int": 42},
			wantError: true,
			errorMsg:  "required parameter 'required_string' is missing",
		},
		{
			name: "wrong_string_type",
			input: map[string]interface{}{
				"required_string": 123,
			},
			wantError: true,
			errorMsg:  "parameter 'required_string' must be a string",
		},
		{
			name: "wrong_int_type",
			input: map[string]interface{}{
				"required_string": "test",
				"optional_int":    "not_an_int",
			},
			wantError: true,
			errorMsg:  "parameter 'optional_int' must be an integer",
		},
		{
			name: "wrong_bool_type",
			input: map[string]interface{}{
				"required_string": "test",
				"optional_bool":   "not_a_bool",
			},
			wantError: true,
			errorMsg:  "parameter 'optional_bool' must be a boolean",
		},
		{
			name: "wrong_array_type",
			input: map[string]interface{}{
				"required_string": "test",
				"optional_array":  "not_an_array",
			},
			wantError: true,
			errorMsg:  "parameter 'optional_array' must be an array",
		},
		{
			name: "wrong_object_type",
			input: map[string]interface{}{
				"required_string": "test",
				"optional_object": "not_an_object",
			},
			wantError: true,
			errorMsg:  "parameter 'optional_object' must be an object",
		},
		{
			name: "valid_types",
			input: map[string]interface{}{
				"required_string": "test",
				"optional_int":    42,
				"optional_bool":   true,
				"optional_array":  []interface{}{"a", "b"},
				"optional_object": map[string]interface{}{"key": "value"},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tool.ValidateInput(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateInput() error = nil, want error")
					return
				}
				if err.Error() != tt.errorMsg {
					t.Errorf("ValidateInput() error = %v, want %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateInput() error = %v, want nil", err)
				}
			}
		})
	}
}

// TestBaseToolResults tests result creation methods
func TestBaseToolResults(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tool := NewBaseTool("test_tool", "Test tool", "testing", nil, logger)

	t.Run("success_result", func(t *testing.T) {
		data := map[string]interface{}{"result": "success"}
		metadata := map[string]interface{}{"timestamp": "now"}

		result := tool.CreateSuccessResult(data, metadata)

		if !result.Success {
			t.Errorf("CreateSuccessResult() Success = false, want true")
		}
		// Compare map contents, not references
		resultData, ok := result.Data.(map[string]interface{})
		if !ok {
			t.Errorf("CreateSuccessResult() Data type assertion failed")
		} else if resultData["result"] != "success" {
			t.Errorf("CreateSuccessResult() Data[\"result\"] = %v, want %v", resultData["result"], "success")
		}
		if result.Metadata["timestamp"] != "now" {
			t.Errorf("CreateSuccessResult() Metadata incorrect")
		}
		if result.Error != "" {
			t.Errorf("CreateSuccessResult() Error = %s, want empty", result.Error)
		}
	})

	t.Run("error_result", func(t *testing.T) {
		err := errors.New("test error")
		metadata := map[string]interface{}{"error_code": "TEST_ERROR"}

		result := tool.CreateErrorResult(err, metadata)

		if result.Success {
			t.Errorf("CreateErrorResult() Success = true, want false")
		}
		if result.Error != "test error" {
			t.Errorf("CreateErrorResult() Error = %s, want %s", result.Error, "test error")
		}
		if result.Metadata["error_code"] != "TEST_ERROR" {
			t.Errorf("CreateErrorResult() Metadata incorrect")
		}
		if result.Data != nil {
			t.Errorf("CreateErrorResult() Data = %v, want nil", result.Data)
		}
	})
}

// TestBaseToolMeasureExecution tests execution measurement
func TestBaseToolMeasureExecution(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tool := NewBaseTool("test_tool", "Test tool", "testing", nil, logger)

	t.Run("measure_successful_execution", func(t *testing.T) {
		expectedResult := &ToolResult{Success: true, Data: "test"}

		result, err := tool.MeasureExecution(func() (*ToolResult, error) {
			time.Sleep(10 * time.Millisecond) // Simulate work
			return expectedResult, nil
		})

		if err != nil {
			t.Errorf("MeasureExecution() error = %v, want nil", err)
		}
		if result != expectedResult {
			t.Errorf("MeasureExecution() result = %v, want %v", result, expectedResult)
		}
		if result.ExecutionTime <= 0 {
			t.Errorf("MeasureExecution() ExecutionTime = %v, want > 0", result.ExecutionTime)
		}
		if result.ExecutionTime < 5*time.Millisecond {
			t.Errorf("MeasureExecution() ExecutionTime = %v, want >= 5ms", result.ExecutionTime)
		}
	})

	t.Run("measure_failed_execution", func(t *testing.T) {
		expectedErr := errors.New("test error")

		result, err := tool.MeasureExecution(func() (*ToolResult, error) {
			time.Sleep(5 * time.Millisecond)
			return nil, expectedErr
		})

		if err != expectedErr {
			t.Errorf("MeasureExecution() error = %v, want %v", err, expectedErr)
		}
		if result != nil {
			t.Errorf("MeasureExecution() result = %v, want nil", result)
		}
	})
}

// TestBaseToolHealth tests default health check
func TestBaseToolHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tool := NewBaseTool("test_tool", "Test tool", "testing", nil, logger)

	ctx := context.Background()
	err := tool.Health(ctx)
	if err != nil {
		t.Errorf("Health() error = %v, want nil", err)
	}
}

// TestBaseToolClose tests default close
func TestBaseToolClose(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tool := NewBaseTool("test_tool", "Test tool", "testing", nil, logger)

	ctx := context.Background()
	err := tool.Close(ctx)
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

// TestBaseToolLogging tests logging methods
func TestBaseToolLogging(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tool := NewBaseTool("test_tool", "Test tool", "testing", nil, logger)

	ctx := context.Background()

	// These methods should not panic and should execute without error
	t.Run("log_info", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("LogInfo() panicked: %v", r)
			}
		}()
		tool.LogInfo(ctx, "test info message")
	})

	t.Run("log_error", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("LogError() panicked: %v", r)
			}
		}()
		tool.LogError(ctx, "test error message", errors.New("test error"))
	})

	t.Run("log_debug", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("LogDebug() panicked: %v", r)
			}
		}()
		tool.LogDebug(ctx, "test debug message")
	})
}

// TestToolError tests the ToolError type
func TestToolError(t *testing.T) {
	tests := []struct {
		name         string
		tool         string
		code         string
		message      string
		cause        error
		expectedMsg  string
		expectUnwrap bool
	}{
		{
			name:        "error_without_cause",
			tool:        "test_tool",
			code:        "TEST_ERROR",
			message:     "Test error message",
			cause:       nil,
			expectedMsg: "test_tool [TEST_ERROR]: Test error message",
		},
		{
			name:         "error_with_cause",
			tool:         "test_tool",
			code:         "TEST_ERROR",
			message:      "Test error message",
			cause:        errors.New("underlying error"),
			expectedMsg:  "test_tool [TEST_ERROR]: Test error message: underlying error",
			expectUnwrap: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolErr := NewToolError(tt.tool, tt.code, tt.message, tt.cause)

			if toolErr.Error() != tt.expectedMsg {
				t.Errorf("Error() = %s, want %s", toolErr.Error(), tt.expectedMsg)
			}

			if tt.expectUnwrap {
				if unwrapped := toolErr.Unwrap(); unwrapped != tt.cause {
					t.Errorf("Unwrap() = %v, want %v", unwrapped, tt.cause)
				}
			}

			if toolErr.Tool != tt.tool {
				t.Errorf("Tool = %s, want %s", toolErr.Tool, tt.tool)
			}
			if toolErr.Code != tt.code {
				t.Errorf("Code = %s, want %s", toolErr.Code, tt.code)
			}
			if toolErr.Message != tt.message {
				t.Errorf("Message = %s, want %s", toolErr.Message, tt.message)
			}
		})
	}
}

// BenchmarkBaseTool benchmarks basic tool operations
func BenchmarkBaseTool(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	tool := NewBaseTool("bench_tool", "Benchmark tool", "testing",
		map[string]interface{}{"key": "value"}, logger)

	b.Run("GetConfig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = tool.GetConfig("key")
		}
	})

	b.Run("GetConfigString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = tool.GetConfigString("key")
		}
	})

	b.Run("CreateSuccessResult", func(b *testing.B) {
		data := map[string]interface{}{"result": "success"}
		metadata := map[string]interface{}{"bench": true}

		for i := 0; i < b.N; i++ {
			_ = tool.CreateSuccessResult(data, metadata)
		}
	})

	b.Run("ValidateInput", func(b *testing.B) {
		mockTool := &mockTool{
			BaseTool: tool,
			params: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
				"required": []string{"name"},
			},
		}
		input := map[string]interface{}{"name": "test"}

		for i := 0; i < b.N; i++ {
			_ = mockTool.ValidateInput(input)
		}
	})
}

// mockTool is a mock implementation for testing
type mockTool struct {
	*BaseTool
	params map[string]interface{}
}

func (m *mockTool) Parameters() map[string]interface{} {
	return m.params
}

func (m *mockTool) Execute(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
	return &ToolResult{Success: true, Data: "mock result"}, nil
}
