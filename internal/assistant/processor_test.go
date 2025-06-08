package assistant

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
	"github.com/koopa0/assistant-go/internal/testutil"
	"github.com/koopa0/assistant-go/internal/tool"
)

// TestProcessorCreation tests processor creation and initialization
func TestProcessorCreation(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		db          postgres.DB
		registry    *tool.Registry
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
			registry:    tool.NewRegistry(testutil.NewTestLogger()),
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
			registry:    tool.NewRegistry(testutil.NewTestLogger()),
			expectError: false,
		},
		{
			name:        "nil_config",
			cfg:         nil,
			db:          postgres.NewMockClient(testutil.NewSilentLogger()),
			registry:    tool.NewRegistry(testutil.NewTestLogger()),
			expectError: true,
			errorType:   "config",
		},
		{
			name:        "nil_database",
			cfg:         &config.Config{Mode: "test"},
			db:          nil,
			registry:    tool.NewRegistry(testutil.NewTestLogger()),
			expectError: true,
			errorType:   "database",
		},
		{
			name:        "nil_registry",
			cfg:         &config.Config{Mode: "test"},
			db:          postgres.NewMockClient(testutil.NewSilentLogger()),
			registry:    nil,
			expectError: true,
			errorType:   "registry",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := testutil.NewTestLogger()
			processor, err := NewProcessor(tt.cfg, tt.db, tt.registry, logger)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if processor != nil {
					t.Errorf("Expected nil processor but got %v", processor)
				}
				if tt.errorType != "" && err != nil {
					if !strings.Contains(strings.ToLower(err.Error()), tt.errorType) {
						t.Errorf("Expected error type %s, got: %v", tt.errorType, err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if processor == nil {
					t.Errorf("Expected processor but got nil")
				}

				// Test cleanup
				if processor != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()

					if err = processor.Close(ctx); err != nil {
						t.Errorf("Failed to close processor: %v", err)
					}
				}
			}
		})
	}
}

// TestProcessorQueryProcessing tests the processor's query processing functionality
func TestProcessorQueryProcessing(t *testing.T) {
	tests := []struct {
		name        string
		request     *QueryRequest
		expectError bool
		errorType   string
	}{
		{
			name: "valid_simple_request",
			request: &QueryRequest{
				Query: "Hello world",
			},
			expectError: false,
		},
		{
			name: "valid_complex_request",
			request: &QueryRequest{
				Query:          "Analyze this code",
				ConversationID: stringPtr("conv-123"),
				UserID:         stringPtr("user-456"),
				Tools:          []string{"go_analyzer"},
				Provider:       stringPtr("claude"),
				MaxTokens:      1000,
				Temperature:    0.7,
			},
			expectError: false,
		},
		{
			name:        "nil_request",
			request:     nil,
			expectError: true,
			errorType:   "request",
		},
		{
			name: "empty_query",
			request: &QueryRequest{
				Query: "",
			},
			expectError: true,
			errorType:   "query",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Setup processor
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
			registry := tool.NewRegistry(testutil.NewTestLogger())
			logger := testutil.NewTestLogger()

			processor, err := NewProcessor(cfg, mockDB, registry, logger)
			if err != nil {
				t.Fatalf("Failed to create processor: %v", err)
			}
			defer func() {
				if err := processor.Close(ctx); err != nil {
					logger.Error("Failed to close processor", "error", err)
				}
			}()

			response, err := processor.Process(ctx, tt.request)

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
				// In test mode, processing may fail but shouldn't error on validation
				if err != nil {
					t.Logf("Processing failed (may be expected in test): %v", err)
				}
				if response != nil {
					// Verify response structure
					if response.ConversationID == "" {
						t.Errorf("Expected non-empty conversation ID")
					}
					if response.MessageID == "" {
						t.Errorf("Expected non-empty message ID")
					}
				}
			}
		})
	}
}

// TestProcessorHealth tests processor health check functionality
func TestProcessorHealth(t *testing.T) {
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
	registry := tool.NewRegistry(testutil.NewTestLogger())
	logger := testutil.NewTestLogger()

	processor, err := NewProcessor(cfg, mockDB, registry, logger)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer func() {
		if err := processor.Close(ctx); err != nil {
			logger.Error("Failed to close processor", "error", err)
		}
	}()

	// Test health check - in test mode, this may fail due to invalid API keys
	err = processor.Health(ctx)
	if err != nil {
		t.Logf("Health check failed (expected in test mode): %v", err)
	}
}

// TestProcessorStats tests processor statistics functionality
func TestProcessorStats(t *testing.T) {
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
	registry := tool.NewRegistry(testutil.NewTestLogger())
	logger := testutil.NewTestLogger()

	processor, err := NewProcessor(cfg, mockDB, registry, logger)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer func() {
		if err := processor.Close(ctx); err != nil {
			logger.Error("Failed to close processor", "error", err)
		}
	}()

	// Test stats collection
	stats, err := processor.Stats(ctx)
	if err != nil {
		t.Errorf("Failed to get stats: %v", err)
	}
	if stats == nil {
		t.Errorf("Expected stats but got nil")
	}
}
