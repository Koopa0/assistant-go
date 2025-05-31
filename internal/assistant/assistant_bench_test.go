package assistant

import (
	"context"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
	"github.com/koopa0/assistant-go/internal/testutil"
)

// BenchmarkAssistantCreation benchmarks assistant creation performance
func BenchmarkAssistantCreation(b *testing.B) {
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

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		assistant, err := New(ctx, cfg, mockDB, logger)
		if err != nil {
			b.Fatalf("Failed to create assistant: %v", err)
		}

		if err := assistant.Close(ctx); err != nil {
			b.Fatalf("Failed to close assistant: %v", err)
		}

		cancel()
	}
}

// BenchmarkQueryProcessing benchmarks query processing performance
func BenchmarkQueryProcessing(b *testing.B) {
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
		b.Fatalf("Failed to create assistant: %v", err)
	}
	defer func() {
		if err := assistant.Close(ctx); err != nil {
			b.Fatalf("Failed to close assistant: %v", err)
		}
	}()

	queries := []string{
		"Hello",
		"What is Go?",
		"Analyze this code",
		"Help me with debugging",
		"Explain concurrent programming",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		query := queries[i%len(queries)]

		// Use a context with timeout for each query
		queryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)

		_, _ = assistant.ProcessQuery(queryCtx, query)
		// Error is expected in test mode, so we don't fail on it
		cancel()
	}
}

// BenchmarkToolExecution benchmarks tool execution performance
func BenchmarkToolExecution(b *testing.B) {
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
		b.Fatalf("Failed to create assistant: %v", err)
	}
	defer func() {
		if err := assistant.Close(ctx); err != nil {
			b.Fatalf("Failed to close assistant: %v", err)
		}
	}()

	request := &ToolExecutionRequest{
		ToolName: "go_analyzer",
		Input: map[string]any{
			"file_path": "/test/file.go",
		},
		Config: map[string]any{
			"timeout": 5,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		toolCtx, cancel := context.WithTimeout(ctx, 3*time.Second)

		_, _ = assistant.ExecuteTool(toolCtx, request)
		// Error is expected in test mode, so we don't fail on it
		cancel()
	}
}

// BenchmarkStatsCollection benchmarks statistics collection performance
func BenchmarkStatsCollection(b *testing.B) {
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
		b.Fatalf("Failed to create assistant: %v", err)
	}
	defer func() {
		if err := assistant.Close(ctx); err != nil {
			b.Fatalf("Failed to close assistant: %v", err)
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		statsCtx, cancel := context.WithTimeout(ctx, 1*time.Second)

		stats, err := assistant.Stats(statsCtx)
		if err != nil {
			b.Fatalf("Failed to get stats: %v", err)
		}

		// Prevent compiler optimization
		_ = stats
		cancel()
	}
}

// BenchmarkHealthCheck benchmarks health check performance
func BenchmarkHealthCheck(b *testing.B) {
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
		b.Fatalf("Failed to create assistant: %v", err)
	}
	defer func() {
		if err := assistant.Close(ctx); err != nil {
			b.Fatalf("Failed to close assistant: %v", err)
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		healthCtx, cancel := context.WithTimeout(ctx, 1*time.Second)

		err := assistant.Health(healthCtx)
		if err != nil {
			b.Fatalf("Health check failed: %v", err)
		}

		cancel()
	}
}

// BenchmarkConversationManagement benchmarks conversation operations
func BenchmarkConversationManagement(b *testing.B) {
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
		b.Fatalf("Failed to create assistant: %v", err)
	}
	defer func() {
		if err := assistant.Close(ctx); err != nil {
			b.Fatalf("Failed to close assistant: %v", err)
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		convCtx, cancel := context.WithTimeout(ctx, 1*time.Second)

		// Benchmark conversation listing
		conversations, err := assistant.ListConversations(convCtx, "test_user", 10, 0)
		if err != nil {
			b.Fatalf("Failed to list conversations: %v", err)
		}

		// Prevent compiler optimization
		_ = conversations
		cancel()
	}
}

// BenchmarkToolRegistry benchmarks tool registry operations
func BenchmarkToolRegistry(b *testing.B) {
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
		b.Fatalf("Failed to create assistant: %v", err)
	}
	defer func() {
		if err := assistant.Close(ctx); err != nil {
			b.Fatalf("Failed to close assistant: %v", err)
		}
	}()

	tools := []string{"go_analyzer", "go_formatter", "go_tester", "go_builder", "go_dependency_analyzer"}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Benchmark getting available tools
		availableTools := assistant.GetAvailableTools()
		_ = availableTools

		// Benchmark getting specific tool info
		toolName := tools[i%len(tools)]
		toolInfo, err := assistant.GetToolInfo(toolName)
		if err != nil {
			b.Fatalf("Failed to get tool info for %s: %v", toolName, err)
		}
		_ = toolInfo
	}
}
