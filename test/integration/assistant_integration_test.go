//go:build integration

package integration

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/ai"
	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
	"github.com/koopa0/assistant-go/internal/tools"
	"github.com/koopa0/assistant-go/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAssistantIntegration tests the complete assistant workflow with real dependencies
func TestAssistantIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	dbContainer, cleanup := testutil.SetupTestDatabase(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wait for database to be ready
	err := dbContainer.WaitForHealthy(ctx, 10*time.Second)
	require.NoError(t, err, "Database should be healthy")

	// Create test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL:             dbContainer.URL,
			MaxConnections:  5,
			ConnMaxLifetime: time.Hour,
		},
		AI: config.AIConfig{
			DefaultProvider: "mock",
		},
		Assistant: config.AssistantConfig{
			MaxConcurrentRequests: 10,
			RequestTimeout:        30 * time.Second,
		},
	}

	// Create logger
	logger := testutil.CreateTestLogger(slog.LevelDebug)

	// Setup storage
	storage, err := postgres.NewStorage(ctx, cfg.Database, logger)
	require.NoError(t, err, "Should create storage")
	defer storage.Close(ctx)

	// Setup mock AI manager
	aiManager := testutil.NewMockAIManager(logger)

	// Setup tool registry
	toolRegistry := tools.NewRegistry(logger)

	// Create assistant
	assistantService, err := assistant.NewAssistant(cfg.Assistant, storage, aiManager, toolRegistry, logger)
	require.NoError(t, err, "Should create assistant")
	defer assistantService.Close(ctx)

	// Test data factory
	factory := testutil.NewTestDataFactory()

	t.Run("ProcessRequest_Success", func(t *testing.T) {
		// Create test request
		request := factory.CreateAssistantRequest(
			testutil.WithMessage("Hello, can you help me with Go development?"),
			testutil.WithUserID("test-user-1"),
		)

		// Set up mock AI response
		expectedResponse := factory.CreateAIGenerateResponse(
			testutil.WithContent("I'd be happy to help you with Go development! What specific aspect would you like assistance with?"),
			testutil.WithProvider("mock"),
		)
		aiManager.SetResponse(request.Message, expectedResponse)

		// Process request
		response, err := assistantService.ProcessRequest(ctx, request)
		require.NoError(t, err, "Should process request successfully")
		assert.NotNil(t, response, "Response should not be nil")
		assert.Equal(t, request.ID, response.RequestID, "Response should reference request ID")
		assert.NotEmpty(t, response.Content, "Response should have content")
		assert.NotEmpty(t, response.ID, "Response should have ID")
	})

	t.Run("ProcessRequest_WithContext", func(t *testing.T) {
		// Create request with context
		request := factory.CreateAssistantRequest(
			testutil.WithMessage("Analyze this Go code for potential issues"),
			testutil.WithUserID("test-user-2"),
		)

		// Set up mock AI response
		expectedResponse := factory.CreateAIGenerateResponse(
			testutil.WithContent("I'll analyze your Go code. Please provide the code you'd like me to review."),
		)
		aiManager.SetResponse(request.Message, expectedResponse)

		// Process request
		response, err := assistantService.ProcessRequest(ctx, request)
		require.NoError(t, err, "Should process request with context")
		assert.NotNil(t, response)
		assert.Contains(t, response.Content, "analyze")
	})

	t.Run("ProcessRequest_Concurrent", func(t *testing.T) {
		// Test concurrent request processing
		numRequests := 5
		responses := make(chan *assistant.Response, numRequests)
		errors := make(chan error, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(requestNum int) {
				request := factory.CreateAssistantRequest(
					testutil.WithMessage("Concurrent test request"),
					testutil.WithUserID("test-user-concurrent"),
				)

				expectedResponse := factory.CreateAIGenerateResponse(
					testutil.WithContent("Concurrent response"),
				)
				aiManager.SetResponse(request.Message, expectedResponse)

				response, err := assistantService.ProcessRequest(ctx, request)
				if err != nil {
					errors <- err
					return
				}
				responses <- response
			}(i)
		}

		// Collect results
		var successCount int
		for i := 0; i < numRequests; i++ {
			select {
			case response := <-responses:
				assert.NotNil(t, response)
				successCount++
			case err := <-errors:
				t.Errorf("Concurrent request failed: %v", err)
			case <-time.After(10 * time.Second):
				t.Error("Timeout waiting for concurrent requests")
				return
			}
		}

		assert.Equal(t, numRequests, successCount, "All concurrent requests should succeed")
	})

	t.Run("ProcessRequest_AIError", func(t *testing.T) {
		// Test handling of AI provider errors
		aiManager.EnableErrorSimulation(1.0) // 100% error rate
		defer aiManager.DisableErrorSimulation()

		request := factory.CreateAssistantRequest(
			testutil.WithMessage("This should trigger an AI error"),
			testutil.WithUserID("test-user-error"),
		)

		response, err := assistantService.ProcessRequest(ctx, request)
		assert.Error(t, err, "Should return error when AI provider fails")
		assert.Nil(t, response, "Response should be nil on error")
	})

	t.Run("ProcessRequest_Timeout", func(t *testing.T) {
		// Test request timeout handling
		shortCtx, shortCancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer shortCancel()

		request := factory.CreateAssistantRequest(
			testutil.WithMessage("This should timeout"),
			testutil.WithUserID("test-user-timeout"),
		)

		response, err := assistantService.ProcessRequest(shortCtx, request)
		assert.Error(t, err, "Should return error on timeout")
		assert.Nil(t, response, "Response should be nil on timeout")
		assert.Contains(t, err.Error(), "context deadline exceeded", "Should be a timeout error")
	})

	t.Run("Health_Check", func(t *testing.T) {
		// Test health check
		err := assistantService.Health(ctx)
		assert.NoError(t, err, "Health check should pass")
	})

	t.Run("GetStats", func(t *testing.T) {
		// Test statistics retrieval
		stats, err := assistantService.GetStats(ctx)
		assert.NoError(t, err, "Should get stats")
		assert.NotNil(t, stats, "Stats should not be nil")
	})
}

// TestAssistantWithRealTools tests assistant with actual tool integration
func TestAssistantWithRealTools(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	dbContainer, cleanup := testutil.SetupTestDatabase(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wait for database to be ready
	err := dbContainer.WaitForHealthy(ctx, 10*time.Second)
	require.NoError(t, err)

	// Create configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL:             dbContainer.URL,
			MaxConnections:  5,
			ConnMaxLifetime: time.Hour,
		},
		AI: config.AIConfig{
			DefaultProvider: "mock",
		},
		Assistant: config.AssistantConfig{
			MaxConcurrentRequests: 10,
			RequestTimeout:        30 * time.Second,
		},
	}

	logger := testutil.CreateTestLogger(slog.LevelDebug)

	// Setup storage
	storage, err := postgres.NewStorage(ctx, cfg.Database, logger)
	require.NoError(t, err)
	defer storage.Close(ctx)

	// Setup mock AI manager
	aiManager := testutil.NewMockAIManager(logger)

	// Setup tool registry with real tools
	toolRegistry := tools.NewRegistry(logger)

	// Register Go development tools (if available)
	// This would register actual tools like go-analyzer, go-tester, etc.

	// Create assistant
	assistantService, err := assistant.NewAssistant(cfg.Assistant, storage, aiManager, toolRegistry, logger)
	require.NoError(t, err)
	defer assistantService.Close(ctx)

	factory := testutil.NewTestDataFactory()

	t.Run("ProcessRequest_WithToolExecution", func(t *testing.T) {
		// Create request that should trigger tool usage
		request := factory.CreateAssistantRequest(
			testutil.WithMessage("Please analyze the Go code in the current directory"),
			testutil.WithUserID("test-user-tools"),
		)

		// Set up mock AI response that includes tool usage
		expectedResponse := factory.CreateAIGenerateResponse(
			testutil.WithContent("I'll analyze the Go code in your directory using the go-analyzer tool."),
		)
		aiManager.SetResponse(request.Message, expectedResponse)

		// Process request
		response, err := assistantService.ProcessRequest(ctx, request)
		require.NoError(t, err, "Should process request with tools")
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Content)
	})
}

// BenchmarkAssistantProcessing benchmarks assistant request processing
func BenchmarkAssistantProcessing(b *testing.B) {
	// Setup test database
	dbContainer, cleanup := testutil.SetupTestDatabase(b)
	defer cleanup()

	ctx := context.Background()

	// Create configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			URL:             dbContainer.URL,
			MaxConnections:  10,
			ConnMaxLifetime: time.Hour,
		},
		AI: config.AIConfig{
			DefaultProvider: "mock",
		},
		Assistant: config.AssistantConfig{
			MaxConcurrentRequests: 100,
			RequestTimeout:        30 * time.Second,
		},
	}

	logger := testutil.CreateTestLogger(slog.LevelError) // Reduce logging for benchmarks

	// Setup storage
	storage, err := postgres.NewStorage(ctx, cfg.Database, logger)
	require.NoError(b, err)
	defer storage.Close(ctx)

	// Setup mock AI manager
	aiManager := testutil.NewMockAIManager(logger)
	aiManager.SetLatencySimulation(false) // Disable latency simulation for benchmarks

	// Setup tool registry
	toolRegistry := tools.NewRegistry(logger)

	// Create assistant
	assistantService, err := assistant.NewAssistant(cfg.Assistant, storage, aiManager, toolRegistry, logger)
	require.NoError(b, err)
	defer assistantService.Close(ctx)

	factory := testutil.NewTestDataFactory()

	// Set up mock response
	expectedResponse := factory.CreateAIGenerateResponse(
		testutil.WithContent("Benchmark response"),
	)
	aiManager.SetResponse("Benchmark message", expectedResponse)

	b.ResetTimer()

	b.Run("ProcessRequest", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			request := factory.CreateAssistantRequest(
				testutil.WithMessage("Benchmark message"),
				testutil.WithUserID("benchmark-user"),
			)

			_, err := assistantService.ProcessRequest(ctx, request)
			if err != nil {
				b.Fatalf("Request processing failed: %v", err)
			}
		}
	})

	b.Run("ProcessRequest_Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				request := factory.CreateAssistantRequest(
					testutil.WithMessage("Benchmark message"),
					testutil.WithUserID("benchmark-user-parallel"),
				)

				_, err := assistantService.ProcessRequest(ctx, request)
				if err != nil {
					b.Fatalf("Parallel request processing failed: %v", err)
				}
			}
		})
	})
}
