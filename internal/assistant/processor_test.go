package assistant

import (
	"context"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
	"github.com/koopa0/assistant-go/internal/testutil"
	"github.com/koopa0/assistant-go/internal/tools"
)

// TestProcessorCreation tests processor creation and initialization
func TestProcessorCreation(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		db          postgres.ClientInterface
		registry    *tools.Registry
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
			registry:    tools.NewRegistry(testutil.NewTestLogger()),
			expectError: false,
		},
		{
			name:        "nil_config",
			cfg:         nil,
			db:          postgres.NewMockClient(testutil.NewSilentLogger()),
			registry:    tools.NewRegistry(testutil.NewTestLogger()),
			expectError: true,
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
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if processor == nil {
					t.Errorf("Expected processor but got nil")
				}

				// Test cleanup
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				if err = processor.Close(ctx); err != nil {
					t.Errorf("Failed to close processor: %v", err)
				}
			}
		})
	}
}
