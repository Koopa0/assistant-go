package godev

import (
	"log/slog"
	"os"
	"testing"

	"github.com/koopa0/assistant-go/internal/tools"
)

// TestRegisterGoTools tests the registration of all Go development tools
func TestRegisterGoTools(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := tools.NewRegistry(logger)

	err := RegisterGoTools(registry)
	if err != nil {
		t.Fatalf("RegisterGoTools() error = %v, want nil", err)
	}

	expectedTools := []string{
		"go_analyzer",
		"go_formatter",
		"go_tester",
		"go_builder",
		"go_dependency_analyzer",
	}

	for _, toolName := range expectedTools {
		t.Run(toolName, func(t *testing.T) {
			if !registry.IsRegistered(toolName) {
				t.Errorf("Tool %s not registered", toolName)
			}

			// Test tool creation
			tool, err := registry.GetTool(toolName, nil)
			if err != nil {
				t.Errorf("Failed to create tool %s: %v", toolName, err)
			}
			if tool == nil {
				t.Errorf("Tool %s creation returned nil", toolName)
			}
			if tool.Name() != toolName {
				t.Errorf("Tool name = %s, want %s", tool.Name(), toolName)
			}
		})
	}
}

// TestGoToolFactories tests individual tool factory functions
func TestGoToolFactories(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tests := []struct {
		name         string
		factory      func(config map[string]interface{}, logger *slog.Logger) (tools.Tool, error)
		expectedName string
	}{
		{
			name:         "go_analyzer",
			factory:      NewGoAnalyzer,
			expectedName: "go_analyzer",
		},
		{
			name:         "go_formatter",
			factory:      NewGoFormatter,
			expectedName: "go_formatter",
		},
		{
			name:         "go_tester",
			factory:      NewGoTester,
			expectedName: "go_tester",
		},
		{
			name:         "go_builder",
			factory:      NewGoBuilder,
			expectedName: "go_builder",
		},
		{
			name:         "go_dependency_analyzer",
			factory:      NewGoDependencyAnalyzer,
			expectedName: "go_dependency_analyzer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := tt.factory(nil, logger)
			if err != nil {
				t.Errorf("Factory %s error = %v, want nil", tt.name, err)
			}
			if tool == nil {
				t.Errorf("Factory %s returned nil tool", tt.name)
			}
			if tool.Name() != tt.expectedName {
				t.Errorf("Tool name = %s, want %s", tool.Name(), tt.expectedName)
			}
			if tool.Description() == "" {
				t.Errorf("Tool %s has empty description", tt.name)
			}
		})
	}
}

// TestGoToolsIntegration tests integration between tools and registry
func TestGoToolsIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	registry := tools.NewRegistry(logger)

	// Register tools
	err := RegisterGoTools(registry)
	if err != nil {
		t.Fatalf("RegisterGoTools() error = %v", err)
	}

	// Test listing tools
	toolsList := registry.ListTools()
	if len(toolsList) != 5 {
		t.Errorf("ListTools() returned %d tools, want 5", len(toolsList))
	}

	// Test all tools are development category
	for _, toolInfo := range toolsList {
		if toolInfo.Category != "development" {
			t.Errorf("Tool %s category = %s, want development", toolInfo.Name, toolInfo.Category)
		}
		if !toolInfo.IsEnabled {
			t.Errorf("Tool %s is not enabled", toolInfo.Name)
		}
	}

	// Test tool info retrieval
	for _, toolName := range []string{"go_analyzer", "go_formatter", "go_tester", "go_builder", "go_dependency_analyzer"} {
		info, err := registry.GetToolInfo(toolName)
		if err != nil {
			t.Errorf("GetToolInfo(%s) error = %v", toolName, err)
		}
		if info.Name != toolName {
			t.Errorf("Tool info name = %s, want %s", info.Name, toolName)
		}
	}
}

// TestGoToolsConfiguration tests tool configuration handling
func TestGoToolsConfiguration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	config := map[string]interface{}{
		"timeout":    30,
		"verbose":    true,
		"output_dir": "/tmp/test",
	}

	tests := []struct {
		name    string
		factory func(config map[string]interface{}, logger *slog.Logger) (tools.Tool, error)
	}{
		{"go_analyzer", NewGoAnalyzer},
		{"go_formatter", NewGoFormatter},
		{"go_tester", NewGoTester},
		{"go_builder", NewGoBuilder},
		{"go_dependency_analyzer", NewGoDependencyAnalyzer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := tt.factory(config, logger)
			if err != nil {
				t.Errorf("Factory with config error = %v", err)
			}
			if tool == nil {
				t.Error("Factory with config returned nil tool")
			}

			// Test that tool can be created with nil config too
			tool2, err := tt.factory(nil, logger)
			if err != nil {
				t.Errorf("Factory with nil config error = %v", err)
			}
			if tool2 == nil {
				t.Error("Factory with nil config returned nil tool")
			}
		})
	}
}

// BenchmarkGoToolsRegistration benchmarks tool registration
func BenchmarkGoToolsRegistration(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	b.Run("RegisterGoTools", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			registry := tools.NewRegistry(logger)
			_ = RegisterGoTools(registry)
		}
	})

	b.Run("ToolCreation", func(b *testing.B) {
		registry := tools.NewRegistry(logger)
		_ = RegisterGoTools(registry)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = registry.GetTool("go_analyzer", nil)
		}
	})
}
