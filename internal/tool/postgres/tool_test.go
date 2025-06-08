package postgres

import (
	"context"
	"log/slog"
	"testing"

	"github.com/koopa0/assistant-go/internal/tool"
)

func TestPostgresTool_Basic(t *testing.T) {
	logger := slog.Default()
	pgTool := NewPostgresTool(logger)

	// Test tool metadata
	if pgTool.Name() != "postgres" {
		t.Errorf("Expected tool name 'postgres', got '%s'", pgTool.Name())
	}

	if pgTool.Description() == "" {
		t.Error("Tool description should not be empty")
	}

	// Test parameters schema
	params := pgTool.Parameters()
	if params == nil {
		t.Fatal("Parameters schema should not be nil")
	}

	if params.Type != "object" {
		t.Errorf("Expected parameters type 'object', got '%s'", params.Type)
	}

	// Test required parameters
	if len(params.Required) != 1 || params.Required[0] != "action" {
		t.Error("Expected 'action' to be the only required parameter")
	}

}

func TestPostgresTool_AnalyzeQuery(t *testing.T) {
	logger := slog.Default()
	pgTool := NewPostgresTool(logger)
	ctx := context.Background()

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "Simple SELECT",
			query:   "SELECT * FROM users WHERE id = 1",
			wantErr: false,
		},
		{
			name:    "Complex JOIN",
			query:   "SELECT u.*, o.* FROM users u JOIN orders o ON u.id = o.user_id WHERE u.active = true",
			wantErr: false,
		},
		{
			name:    "Missing query",
			query:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &tool.ToolInput{
				Parameters: map[string]interface{}{
					"action": "analyze_query",
					"query":  tt.query,
				},
			}

			result, err := pgTool.Execute(ctx, input)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if tt.wantErr && result.Success {
				t.Error("Expected error but got success")
			}
			if !tt.wantErr && !result.Success {
				t.Errorf("Expected success but got error: %s", result.Error)
			}
		})
	}
}

func TestPostgresTool_ValidateMigration(t *testing.T) {
	logger := slog.Default()
	pgTool := NewPostgresTool(logger)
	ctx := context.Background()

	tests := []struct {
		name      string
		migration string
		wantErr   bool
	}{
		{
			name: "Valid CREATE TABLE",
			migration: `
				CREATE TABLE IF NOT EXISTS users (
					id SERIAL PRIMARY KEY,
					email VARCHAR(255) UNIQUE NOT NULL,
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);`,
			wantErr: false,
		},
		{
			name:      "Dangerous DROP TABLE",
			migration: `DROP TABLE users;`,
			wantErr:   false, // Validation should succeed but with warnings
		},
		{
			name:      "Empty migration",
			migration: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &tool.ToolInput{
				Parameters: map[string]interface{}{
					"action":    "validate_migration",
					"migration": tt.migration,
				},
			}

			result, err := pgTool.Execute(ctx, input)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if tt.wantErr && result.Success {
				t.Error("Expected error but got success")
			}
			if !tt.wantErr && !result.Success {
				t.Errorf("Expected success but got error: %s", result.Error)
			}

			// Check for output
			if result.Success && result.Data != nil && result.Data.Output == nil {
				t.Error("Expected non-empty output for successful validation")
			}
		})
	}
}

func TestPostgresTool_GenerateMigration(t *testing.T) {
	logger := slog.Default()
	pgTool := NewPostgresTool(logger)
	ctx := context.Background()

	input := &tool.ToolInput{
		Parameters: map[string]interface{}{
			"action":         "generate_migration",
			"migration_type": "create_table",
			"table":          "products",
			"schema":         "public",
		},
	}

	result, err := pgTool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected success but got error: %s", result.Error)
	}

	if result.Data == nil || result.Data.Output == nil {
		t.Error("Expected non-empty output for migration generation")
	}
}

func TestPostgresTool_CheckPerformance_NoConnection(t *testing.T) {
	logger := slog.Default()
	pgTool := NewPostgresTool(logger)
	ctx := context.Background()

	input := &tool.ToolInput{
		Parameters: map[string]interface{}{
			"action": "check_performance",
		},
	}

	result, err := pgTool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Should succeed but indicate connection required
	if !result.Success {
		t.Errorf("Expected success (with error message) but got failure: %s", result.Error)
	}
}
