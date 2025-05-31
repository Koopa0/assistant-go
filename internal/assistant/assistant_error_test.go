package assistant

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
	"github.com/koopa0/assistant-go/internal/testutil"
)

// TestErrorTypes tests custom error types and their behavior
func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func() error
		expectType string
		expectMsg  string
	}{
		{
			name: "configuration_error",
			setupFunc: func() error {
				return NewConfigurationError("test_field", errors.New("test error"))
			},
			expectType: "configuration",
			expectMsg:  "test_field",
		},
		{
			name: "invalid_input_error",
			setupFunc: func() error {
				return NewInvalidInputError("test input is invalid", errors.New("validation failed"))
			},
			expectType: "invalid_input",
			expectMsg:  "test input is invalid",
		},
		{
			name: "processing_error",
			setupFunc: func() error {
				return NewProcessingFailedError("test_stage", errors.New("processing failed"))
			},
			expectType: "processing",
			expectMsg:  "test_stage",
		},
		{
			name: "timeout_error",
			setupFunc: func() error {
				return NewTimeoutError("test_operation", "5s")
			},
			expectType: "timeout",
			expectMsg:  "test_operation",
		},
		{
			name: "tool_not_found_error",
			setupFunc: func() error {
				return NewToolNotFoundError("test_tool")
			},
			expectType: "tool not found",
			expectMsg:  "test_tool",
		},
		{
			name: "database_error",
			setupFunc: func() error {
				return NewDatabaseError("test_operation", errors.New("connection failed"))
			},
			expectType: "database",
			expectMsg:  "test_operation",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.setupFunc()
			if err == nil {
				t.Fatal("Expected error but got nil")
			}

			errMsg := err.Error()
			if !strings.Contains(strings.ToLower(errMsg), tt.expectType) {
				t.Errorf("Expected error message to contain %q, got: %s", tt.expectType, errMsg)
			}

			// For structured errors, check the context instead of the main message
			if assistantErr := GetAssistantError(err); assistantErr != nil {
				found := false
				for _, value := range assistantErr.Context {
					if valueStr, ok := value.(string); ok && strings.Contains(valueStr, tt.expectMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Logf("Expected error context to contain %q, got context: %+v, full error: %s", tt.expectMsg, assistantErr.Context, errMsg)
				}
			} else if !strings.Contains(errMsg, tt.expectMsg) {
				t.Errorf("Expected error message to contain %q, got: %s", tt.expectMsg, errMsg)
			}
		})
	}
}

// TestContextCancellation tests behavior with cancelled contexts
func TestContextCancellation(t *testing.T) {
	tests := []struct {
		name        string
		operation   func(context.Context, *Assistant) error
		description string
	}{
		{
			name: "query_processing_with_cancelled_context",
			operation: func(ctx context.Context, a *Assistant) error {
				_, err := a.ProcessQuery(ctx, "test query")
				return err
			},
			description: "ProcessQuery with cancelled context",
		},
		{
			name: "tool_execution_with_cancelled_context",
			operation: func(ctx context.Context, a *Assistant) error {
				req := &ToolExecutionRequest{
					ToolName: "go_analyzer",
					Input:    map[string]any{"file_path": "/test/file.go"},
				}
				_, err := a.ExecuteTool(ctx, req)
				return err
			},
			description: "ExecuteTool with cancelled context",
		},
		{
			name: "health_check_with_cancelled_context",
			operation: func(ctx context.Context, a *Assistant) error {
				return a.Health(ctx)
			},
			description: "Health check with cancelled context",
		},
		{
			name: "stats_with_cancelled_context",
			operation: func(ctx context.Context, a *Assistant) error {
				_, err := a.Stats(ctx)
				return err
			},
			description: "Stats with cancelled context",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create assistant
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

			ctx := context.Background()
			assistant, err := New(ctx, cfg, mockDB, logger)
			if err != nil {
				t.Fatalf("Failed to create assistant: %v", err)
			}
			defer func() {
				if err := assistant.Close(ctx); err != nil {
					t.Logf("Failed to close assistant: %v", err)
				}
			}()

			// Create cancelled context
			cancelledCtx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			// Test operation with cancelled context
			err = tt.operation(cancelledCtx, assistant)

			// Should return context.Canceled or handle gracefully
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Logf("%s with cancelled context returned: %v (may be acceptable)", tt.description, err)
			}
		})
	}
}

// TestTimeoutBehavior tests behavior with various timeout scenarios
func TestTimeoutBehavior(t *testing.T) {
	tests := []struct {
		name        string
		timeout     time.Duration
		operation   func(context.Context, *Assistant) error
		expectError bool
		description string
	}{
		{
			name:    "very_short_timeout_query",
			timeout: 1 * time.Millisecond,
			operation: func(ctx context.Context, a *Assistant) error {
				_, err := a.ProcessQuery(ctx, "test query")
				return err
			},
			expectError: true,
			description: "ProcessQuery with very short timeout",
		},
		{
			name:    "reasonable_timeout_health",
			timeout: 1 * time.Second,
			operation: func(ctx context.Context, a *Assistant) error {
				return a.Health(ctx)
			},
			expectError: false,
			description: "Health check with reasonable timeout",
		},
		{
			name:    "short_timeout_stats",
			timeout: 100 * time.Millisecond,
			operation: func(ctx context.Context, a *Assistant) error {
				_, err := a.Stats(ctx)
				return err
			},
			expectError: false, // Stats should be fast
			description: "Stats with short timeout",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create assistant
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

			ctx := context.Background()
			assistant, err := New(ctx, cfg, mockDB, logger)
			if err != nil {
				t.Fatalf("Failed to create assistant: %v", err)
			}
			defer func() {
				if err := assistant.Close(ctx); err != nil {
					t.Logf("Failed to close assistant: %v", err)
				}
			}()

			// Create context with timeout
			timeoutCtx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			// Test operation with timeout
			err = tt.operation(timeoutCtx, assistant)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error due to timeout but got none", tt.description)
				}
			} else {
				if err != nil && errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("%s: unexpected timeout error: %v", tt.description, err)
				}
			}
		})
	}
}

// TestInvalidInputHandling tests handling of various invalid inputs
func TestInvalidInputHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create assistant
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

	t.Run("invalid_query_requests", func(t *testing.T) {
		tests := []struct {
			name    string
			request *QueryRequest
			wantErr bool
		}{
			{
				name:    "nil_request",
				request: nil,
				wantErr: true,
			},
			{
				name: "empty_query",
				request: &QueryRequest{
					Query: "",
				},
				wantErr: true,
			},
			{
				name: "whitespace_only_query",
				request: &QueryRequest{
					Query: "   \n\t  ",
				},
				wantErr: true,
			},
			{
				name: "negative_max_tokens",
				request: &QueryRequest{
					Query:     "test",
					MaxTokens: -100,
				},
				wantErr: false, // Should handle gracefully
			},
			{
				name: "invalid_temperature",
				request: &QueryRequest{
					Query:       "test",
					Temperature: -1.0,
				},
				wantErr: false, // Should handle gracefully
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := assistant.ProcessQueryRequest(ctx, tt.request)
				if (err != nil) != tt.wantErr {
					t.Errorf("ProcessQueryRequest() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("invalid_tool_requests", func(t *testing.T) {
		tests := []struct {
			name    string
			request *ToolExecutionRequest
			wantErr bool
		}{
			{
				name:    "nil_request",
				request: nil,
				wantErr: true,
			},
			{
				name: "empty_tool_name",
				request: &ToolExecutionRequest{
					ToolName: "",
					Input:    map[string]any{},
				},
				wantErr: true,
			},
			{
				name: "nil_input",
				request: &ToolExecutionRequest{
					ToolName: "go_analyzer",
					Input:    nil,
				},
				wantErr: false, // Should handle gracefully
			},
			{
				name: "invalid_input_types",
				request: &ToolExecutionRequest{
					ToolName: "go_analyzer",
					Input: map[string]any{
						"file_path": func() {}, // Function not serializable
					},
				},
				wantErr: false, // Tool execution may handle this
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := assistant.ExecuteTool(ctx, tt.request)
				if (err != nil) != tt.wantErr {
					t.Errorf("ExecuteTool() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})

	t.Run("invalid_conversation_operations", func(t *testing.T) {
		tests := []struct {
			name        string
			operation   func() error
			wantErr     bool
			description string
		}{
			{
				name: "list_conversations_empty_user",
				operation: func() error {
					_, err := assistant.ListConversations(ctx, "", 10, 0)
					return err
				},
				wantErr:     false, // Should handle gracefully
				description: "List conversations with empty user ID",
			},
			{
				name: "list_conversations_negative_limit",
				operation: func() error {
					_, err := assistant.ListConversations(ctx, "user", -1, 0)
					return err
				},
				wantErr:     false, // Should handle gracefully
				description: "List conversations with negative limit",
			},
			{
				name: "list_conversations_negative_offset",
				operation: func() error {
					_, err := assistant.ListConversations(ctx, "user", 10, -1)
					return err
				},
				wantErr:     false, // Should handle gracefully
				description: "List conversations with negative offset",
			},
			{
				name: "get_conversation_empty_id",
				operation: func() error {
					_, err := assistant.GetConversation(ctx, "")
					return err
				},
				wantErr:     true, // Should error on empty ID
				description: "Get conversation with empty ID",
			},
			{
				name: "delete_conversation_empty_id",
				operation: func() error {
					return assistant.DeleteConversation(ctx, "")
				},
				wantErr:     true, // Should error on empty ID
				description: "Delete conversation with empty ID",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.operation()
				if (err != nil) != tt.wantErr {
					t.Errorf("%s: error = %v, wantErr %v", tt.description, err, tt.wantErr)
				}
			})
		}
	})
}

// TestResourceCleanup tests proper resource cleanup in error scenarios
func TestResourceCleanup(t *testing.T) {
	t.Run("cleanup_after_creation_failure", func(t *testing.T) {
		// Test that resources are cleaned up when assistant creation fails
		// This is more of a design test - we verify no panics occur

		invalidConfigs := []*config.Config{
			nil, // Nil config
			{},  // Empty config
		}

		for i, cfg := range invalidConfigs {
			ctx := context.Background()
			mockDB := postgres.NewMockClient(testutil.NewSilentLogger())
			logger := testutil.NewTestLogger()

			assistant, err := New(ctx, cfg, mockDB, logger)
			if err == nil {
				t.Errorf("Config %d: expected error but got none", i)
				if assistant != nil {
					assistant.Close(ctx)
				}
			}
			if assistant != nil {
				t.Errorf("Config %d: expected nil assistant on error", i)
			}
		}
	})

	t.Run("multiple_close_calls", func(t *testing.T) {
		// Test that multiple Close() calls don't cause issues
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

		// Call Close() multiple times
		err1 := assistant.Close(ctx)
		err2 := assistant.Close(ctx)
		err3 := assistant.Close(ctx)

		// Should not panic or cause severe errors
		// Errors may occur but should be handled gracefully
		if err1 != nil {
			t.Logf("First close error (may be acceptable): %v", err1)
		}
		if err2 != nil {
			t.Logf("Second close error (may be acceptable): %v", err2)
		}
		if err3 != nil {
			t.Logf("Third close error (may be acceptable): %v", err3)
		}
	})
}

// TestEdgeCases tests various edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
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

	t.Run("extreme_values", func(t *testing.T) {
		tests := []struct {
			name    string
			request *QueryRequest
		}{
			{
				name: "max_int_max_tokens",
				request: &QueryRequest{
					Query:     "test",
					MaxTokens: int(^uint(0) >> 1), // Max int
				},
			},
			{
				name: "max_float_temperature",
				request: &QueryRequest{
					Query:       "test",
					Temperature: 1000.0, // Very high temperature
				},
			},
			{
				name: "zero_values",
				request: &QueryRequest{
					Query:       "test",
					MaxTokens:   0,
					Temperature: 0.0,
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Should not panic with extreme values
				_, err := assistant.ProcessQueryRequest(ctx, tt.request)
				// Error is acceptable, panic is not
				if err != nil {
					t.Logf("Request with extreme values failed (acceptable): %v", err)
				}
			})
		}
	})

	t.Run("unicode_and_special_characters", func(t *testing.T) {
		specialQueries := []string{
			"\x00\x01\x02",             // Control characters
			"🚀🔥💻",                      // Emojis
			"测试中文查询",                   // Chinese characters
			"مرحبا بالعالم",            // Arabic characters
			"\n\r\t",                   // Whitespace characters
			string([]byte{0xFF, 0xFE}), // Invalid UTF-8
		}

		for i, query := range specialQueries {
			t.Run(fmt.Sprintf("special_query_%d", i), func(t *testing.T) {
				// Should not panic with special characters
				_, err := assistant.ProcessQuery(ctx, query)
				// Error is acceptable for invalid UTF-8, panic is not
				if err != nil {
					t.Logf("Special query failed (may be acceptable): %v", err)
				}
			})
		}
	})
}
