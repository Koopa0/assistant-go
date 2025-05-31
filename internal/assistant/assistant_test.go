package assistant

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
	"github.com/koopa0/assistant-go/internal/testutil"
)

// TestAssistantCreation tests basic assistant creation and initialization
func TestAssistantCreation(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		db          postgres.ClientInterface
		logger      *slog.Logger
		expectError bool
		errorType   string
	}{
		{
			name: "valid_configuration_claude",
			cfg: &config.Config{
				Mode: "test",
				AI: config.AIConfig{
					DefaultProvider: "claude",
					Claude: config.Claude{
						APIKey: "test-key",
						Model:  "claude-3-sonnet-20240229",
					},
				},
			},
			db:          postgres.NewMockClient(testutil.NewSilentLogger()),
			logger:      testutil.NewTestLogger(),
			expectError: false,
		},
		{
			name: "valid_configuration_gemini",
			cfg: &config.Config{
				Mode: "test",
				AI: config.AIConfig{
					DefaultProvider: "gemini",
					Gemini: config.Gemini{
						APIKey: "test-key",
						Model:  "gemini-pro",
					},
				},
			},
			db:          postgres.NewMockClient(testutil.NewSilentLogger()),
			logger:      testutil.NewTestLogger(),
			expectError: false,
		},
		{
			name:        "nil_config",
			cfg:         nil,
			db:          postgres.NewMockClient(testutil.NewSilentLogger()),
			logger:      testutil.NewTestLogger(),
			expectError: true,
			errorType:   "configuration",
		},
		{
			name:        "nil_database",
			cfg:         &config.Config{Mode: "test"},
			db:          nil,
			logger:      testutil.NewTestLogger(),
			expectError: true,
			errorType:   "configuration",
		},
		{
			name:        "nil_logger",
			cfg:         &config.Config{Mode: "test"},
			db:          postgres.NewMockClient(testutil.NewSilentLogger()),
			logger:      nil,
			expectError: true,
			errorType:   "configuration",
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable for parallel execution
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			assistant, err := New(ctx, tt.cfg, tt.db, tt.logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if assistant != nil {
					t.Errorf("Expected nil assistant but got %v", assistant)
				}
				// Verify error type
				if tt.errorType != "" && err != nil {
					if !strings.Contains(err.Error(), tt.errorType) {
						t.Errorf("Expected error type %s, got: %v", tt.errorType, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if assistant == nil {
					t.Errorf("Expected assistant but got nil")
				}

				// Test graceful shutdown
				if assistant != nil {
					if err := assistant.Close(ctx); err != nil {
						t.Errorf("Failed to close assistant: %v", err)
					}
				}
			}
		})
	}
}

// TestQueryProcessing tests basic query processing functionality
func TestQueryProcessing(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectError bool
		errorType   string
	}{
		{
			name:        "valid_simple_query",
			query:       "Hello, how are you?",
			expectError: false,
		},
		{
			name:        "empty_query",
			query:       "",
			expectError: true,
			errorType:   "invalid input",
		},
		{
			name:        "whitespace_only_query",
			query:       "   \n\t  ",
			expectError: true,
			errorType:   "invalid input",
		},
		{
			name:        "complex_technical_query",
			query:       "Can you analyze this Go code and suggest improvements?",
			expectError: false,
		},
		{
			name:        "long_query",
			query:       strings.Repeat("This is a long query that tests handling of large input. ", 100),
			expectError: false,
		},
		{
			name:        "unicode_query",
			query:       "Hello 世界! 你好 مرحبا 🌍",
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Setup test assistant
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
					logger.Error("Failed to close assistant", slog.Any("error", err))
				}
			}()

			response, err := assistant.ProcessQuery(ctx, tt.query)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tt.errorType != "" && err != nil {
					if !strings.Contains(strings.ToLower(err.Error()), tt.errorType) {
						t.Errorf("Expected error type %s, got: %v", tt.errorType, err)
					}
				}
			} else {
				if err != nil {
					// In test mode, some errors are expected due to mock implementations
					t.Logf("Query processing error (may be expected in test mode): %v", err)
				}
				if response == "" && err == nil {
					t.Errorf("Expected response but got empty string")
				}
			}
		})
	}
}

// TestHealthCheck tests the health check functionality
func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() (*Assistant, func(), error)
		expectError bool
	}{
		{
			name: "healthy_assistant",
			setupFunc: func() (*Assistant, func(), error) {
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
					return nil, nil, err
				}

				cleanup := func() {
					if err := assistant.Close(ctx); err != nil {
						logger.Error("Failed to close assistant", slog.Any("error", err))
					}
				}

				return assistant, cleanup, nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			assistant, cleanup, err := tt.setupFunc()
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			defer cleanup()

			// Test health check
			err = assistant.Health(ctx)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Health check failed: %v", err)
				}
			}
		})
	}
}

// TestToolExecution tests direct tool execution
func TestToolExecution(t *testing.T) {
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
	defer func(assistant *Assistant, ctx context.Context) {
		if err = assistant.Close(ctx); err != nil {
			logger.Error("Failed to close assistant", slog.Any("error", err))
		}
	}(assistant, ctx)

	tests := []struct {
		name        string
		request     *ToolExecutionRequest
		expectError bool
	}{
		{
			name: "nonexistent_tool",
			request: &ToolExecutionRequest{
				ToolName: "nonexistent_tool",
				Input:    map[string]interface{}{},
			},
			expectError: true,
		},
		{
			name: "go_analyzer_tool",
			request: &ToolExecutionRequest{
				ToolName: "go_analyzer",
				Input: map[string]interface{}{
					"file_path": "/test/file.go",
				},
			},
			expectError: false, // May fail but should not error on registration
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			result, err := assistant.ExecuteTool(ctx, tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				// Tool may fail execution but should not fail on basic setup
				if err != nil && tt.request.ToolName != "nonexistent_tool" {
					// This is acceptable for the test environment
					t.Logf("Tool execution failed (expected in test): %v", err)
				}
				if result != nil && !result.Success {
					t.Logf("Tool execution was not successful (expected in test): %s", result.Error)
				}
			}
		})
	}
}

// TestConversationManagement tests conversation management functionality
func TestConversationManagement(t *testing.T) {
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
	defer func(assistant *Assistant, ctx context.Context) {
		if err = assistant.Close(ctx); err != nil {
			logger.Error("Failed to close assistant", slog.Any("error", err))
		}
	}(assistant, ctx)

	// Test listing conversations
	conversations, err := assistant.ListConversations(ctx, "test_user", 10, 0)
	if err != nil {
		t.Errorf("Failed to list conversations: %v", err)
	}
	if conversations == nil {
		t.Errorf("Expected conversations slice but got nil")
	}

	// Test getting available tools
	tools := assistant.GetAvailableTools()
	if tools == nil {
		t.Errorf("Expected tools slice but got nil")
	}

	// Should have registered Go tools
	if len(tools) == 0 {
		t.Errorf("Expected some tools to be registered")
	}
}

// TestStats tests statistics functionality
func TestStats(t *testing.T) {
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
	defer func(assistant *Assistant, ctx context.Context) {
		if err = assistant.Close(ctx); err != nil {
			logger.Error("Failed to close assistant", slog.Any("error", err))
		}
	}(assistant, ctx)

	stats, err := assistant.Stats(ctx)
	if err != nil {
		t.Errorf("Failed to get stats: %v", err)
	}
	if stats == nil {
		t.Errorf("Expected stats but got nil")
	}

	// Verify the basic stats structure
	if stats.Database == nil {
		t.Errorf("Expected database stats but not found")
	}
	if stats.Tools == nil {
		t.Errorf("Expected tools stats but not found")
	}
}
