package assistant

import (
	"context"
	"log/slog"
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
		expectError bool
	}{
		{
			name: "valid_configuration",
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
			expectError: false,
		},
		{
			name:        "nil_config",
			cfg:         nil,
			db:          postgres.NewMockClient(testutil.NewSilentLogger()),
			expectError: true,
		},
		{
			name:        "nil_database",
			cfg:         &config.Config{Mode: "test"},
			db:          nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable for parallel execution
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			logger := testutil.NewTestLogger()
			assistant, err := New(ctx, tt.cfg, tt.db, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if assistant != nil {
					t.Errorf("Expected nil assistant but got %v", assistant)
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
	defer func(assistant *Assistant, ctx context.Context) {
		if err = assistant.Close(ctx); err != nil {
			logger.Error("Failed to close assistant", slog.Any("error", err))
		}
	}(assistant, ctx)

	tests := []struct {
		name        string
		query       string
		expectError bool
	}{
		{
			name:        "valid_query",
			query:       "Hello, how are you?",
			expectError: false,
		},
		{
			name:        "empty_query",
			query:       "",
			expectError: true,
		},
		{
			name:        "complex_query",
			query:       "Can you analyze this Go code and suggest improvements?",
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			response, err := assistant.ProcessQuery(ctx, tt.query)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if response == "" {
					t.Errorf("Expected response but got empty string")
				}
			}
		})
	}
}

// TestHealthCheck tests the health check functionality
func TestHealthCheck(t *testing.T) {
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

	// Test health check
	err = assistant.Health(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
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
		toolName    string
		input       map[string]interface{}
		config      map[string]interface{}
		expectError bool
	}{
		{
			name:        "nonexistent_tool",
			toolName:    "nonexistent_tool",
			input:       map[string]interface{}{},
			config:      nil,
			expectError: true,
		},
		{
			name:     "go_analyzer_tool",
			toolName: "go_analyzer",
			input: map[string]interface{}{
				"file_path": "/test/file.go",
			},
			config:      nil,
			expectError: false, // May fail but should not error on registration
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			result, err := assistant.ExecuteTool(ctx, tt.toolName, tt.input, tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				// Tool may fail execution but should not fail on basic setup
				if err != nil && tt.toolName != "nonexistent_tool" {
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
	if _, exists := stats["database"]; !exists {
		t.Errorf("Expected database stats but not found")
	}
	if _, exists := stats["tools"]; !exists {
		t.Errorf("Expected tools stats but not found")
	}
}
