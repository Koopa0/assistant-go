package integration

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/observability"
	"github.com/koopa0/assistant-go/internal/testutil"
)

// TestObservabilityStackIntegration tests the complete observability stack integration
func TestObservabilityStackIntegration(t *testing.T) {
	// Table-driven test with different configurations (golang_guide.md pattern)
	tests := []struct {
		name            string
		config          *observability.ObservabilityConfig
		expectError     bool
		expectOTel      bool
		expectMetrics   bool
		expectProfiling bool
	}{
		{
			name: "minimal_config",
			config: &observability.ObservabilityConfig{
				ServiceName:                  "test-service",
				ServiceVersion:               "1.0.0",
				Environment:                  "test",
				LogLevel:                     "debug",
				LogFormat:                    "json",
				OTelEnabled:                  false,
				MetricsEnabled:               false,
				ProfilingEnabled:             false,
				PerformanceMonitoringEnabled: false,
			},
			expectError:     false,
			expectOTel:      false,
			expectMetrics:   false,
			expectProfiling: false,
		},
		{
			name: "full_observability",
			config: &observability.ObservabilityConfig{
				ServiceName:                  "test-service-full",
				ServiceVersion:               "1.0.0",
				Environment:                  "test",
				LogLevel:                     "info",
				LogFormat:                    "json",
				OTelEnabled:                  true,
				TraceEndpoint:                "http://localhost:4318",
				MetricsEndpoint:              "http://localhost:4318",
				SamplingRate:                 1.0, // 100% sampling for tests
				MetricsEnabled:               true,
				ProfilingEnabled:             true,
				ProfilingInterval:            1 * time.Minute,
				PerformanceMonitoringEnabled: true,
				BaselineMetrics: map[string]float64{
					"test_metric": 100.0,
				},
				RegressionThresholds: map[string]float64{
					"test_metric": 50.0,
				},
			},
			expectError:     false,
			expectOTel:      true,
			expectMetrics:   true,
			expectProfiling: true,
		},
		{
			name:            "production_config",
			config:          observability.NewProductionConfig(),
			expectError:     false,
			expectOTel:      true,
			expectMetrics:   true,
			expectProfiling: true,
		},
		{
			name:            "default_config",
			config:          observability.NewDefaultConfig(),
			expectError:     false,
			expectOTel:      true,
			expectMetrics:   true,
			expectProfiling: false, // Default is disabled
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable (golang_guide.md critical requirement)
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // Run tests in parallel (golang_guide.md performance best practice)

			// Adjust config for test environment to avoid external dependencies
			testConfig := *tt.config
			testConfig.OTelEnabled = false // Disable OTel to avoid external dependencies in tests

			// Initialize observability stack
			stack, err := observability.Initialize(&testConfig)

			// Check for initialization errors
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if err != nil {
				return // Skip remaining tests if initialization failed
			}

			// Verify components are initialized as expected
			if stack.Logger == nil {
				t.Error("Logger should always be initialized")
			}

			// Test logger functionality
			stack.Logger.Info("Test log message",
				slog.String("test", "integration"),
				slog.String("component", "observability"))

			// Check OpenTelemetry setup
			if tt.expectOTel {
				if testConfig.OTelEnabled { // Only check if we actually enabled it
					if stack.OTelSetup == nil {
						t.Error("Expected OTel setup but got nil")
					}
					if stack.DatabaseTracer == nil {
						t.Error("Expected database tracer but got nil")
					}
					if stack.AITracer == nil {
						t.Error("Expected AI tracer but got nil")
					}
					if stack.ToolTracer == nil {
						t.Error("Expected tool tracer but got nil")
					}
				}
			}

			// Check metrics setup
			if tt.expectMetrics && testConfig.MetricsEnabled {
				if stack.Metrics == nil {
					t.Error("Expected metrics but got nil")
				}

				// Test metrics functionality
				ctx := context.Background()
				stack.Metrics.RecordRequest(ctx, "GET", "/test", 100*time.Millisecond, 200)
				stack.Metrics.RecordCacheOperation(ctx, "get", "hit")
			}

			// Check performance monitoring
			if testConfig.PerformanceMonitoringEnabled {
				if stack.PerformanceMonitor == nil {
					t.Error("Expected performance monitor but got nil")
				}

				// Test performance monitor health
				ctx := context.Background()
				if err := stack.PerformanceMonitor.Health(ctx); err != nil {
					t.Errorf("Performance monitor health check failed: %v", err)
				}
			}

			// Check profiling setup
			if stack.ProfileManager == nil {
				t.Error("Profile manager should always be initialized")
			}

			// Test health status
			ctx := context.Background()
			health := stack.GetHealthStatus(ctx)

			if health == nil {
				t.Error("Health status should not be nil")
			}

			// Verify health status structure
			if service, ok := health["service"].(string); !ok || service != testConfig.ServiceName {
				t.Errorf("Expected service name %s, got %v", testConfig.ServiceName, health["service"])
			}

			if components, ok := health["components"].(map[string]interface{}); !ok {
				t.Error("Expected components in health status")
			} else {
				// Verify logging component
				if logging, ok := components["logging"].(map[string]interface{}); !ok {
					t.Error("Expected logging component in health status")
				} else {
					if status, ok := logging["status"].(string); !ok || status != "healthy" {
						t.Errorf("Expected logging status to be healthy, got %v", logging["status"])
					}
				}
			}

			// Test graceful shutdown
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := stack.Shutdown(shutdownCtx); err != nil {
				t.Errorf("Shutdown failed: %v", err)
			}
		})
	}
}

// TestObservabilityStackConcurrency tests concurrent operations on the observability stack
func TestObservabilityStackConcurrency(t *testing.T) {
	config := &observability.ObservabilityConfig{
		ServiceName:                  "test-concurrent",
		ServiceVersion:               "1.0.0",
		Environment:                  "test",
		LogLevel:                     "info",
		LogFormat:                    "json",
		OTelEnabled:                  false, // Disable to avoid external dependencies
		MetricsEnabled:               true,
		ProfilingEnabled:             false,
		PerformanceMonitoringEnabled: true,
	}

	stack, err := observability.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize observability stack: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		stack.Shutdown(ctx)
	}()

	// Test concurrent logging
	const numGoroutines = 10
	const operationsPerGoroutine = 50

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Concurrent logging
				stack.Logger.Info("Concurrent test",
					slog.Int("goroutine", id),
					slog.Int("operation", j))

				// Concurrent metrics (if available)
				if stack.Metrics != nil {
					ctx := context.Background()
					stack.Metrics.RecordRequest(ctx, "POST", "/concurrent",
						time.Duration(j)*time.Millisecond, 200)
					stack.Metrics.RecordCacheOperation(ctx, "set", "success")
				}

				// Concurrent health checks
				if stack.PerformanceMonitor != nil {
					ctx := context.Background()
					stack.PerformanceMonitor.Health(ctx)
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Goroutine completed successfully
		case <-time.After(30 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations to complete")
		}
	}

	// Verify system is still healthy after concurrent operations
	ctx := context.Background()
	health := stack.GetHealthStatus(ctx)

	if health == nil {
		t.Error("Health status should not be nil after concurrent operations")
	}
}

// TestObservabilityEnvironmentVariables tests environment variable configuration
func TestObservabilityEnvironmentVariables(t *testing.T) {
	// Save original environment
	originalEnv := make(map[string]string)
	envVars := []string{
		"ENVIRONMENT", "LOG_LEVEL", "LOG_FORMAT", "OTEL_ENABLED",
		"METRICS_ENABLED", "PROFILING_ENABLED", "PERFORMANCE_MONITORING_ENABLED",
	}

	for _, envVar := range envVars {
		originalEnv[envVar] = os.Getenv(envVar)
	}

	// Restore environment after test
	defer func() {
		for _, envVar := range envVars {
			if original, exists := originalEnv[envVar]; exists && original != "" {
				os.Setenv(envVar, original)
			} else {
				os.Unsetenv(envVar)
			}
		}
	}()

	// Set test environment variables
	os.Setenv("ENVIRONMENT", "test-env")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_FORMAT", "text")
	os.Setenv("OTEL_ENABLED", "false")
	os.Setenv("METRICS_ENABLED", "true")
	os.Setenv("PROFILING_ENABLED", "false")
	os.Setenv("PERFORMANCE_MONITORING_ENABLED", "true")

	// Create config that should read from environment
	config := observability.NewDefaultConfig()

	// Verify environment variables were read correctly
	if config.Environment != "test-env" {
		t.Errorf("Expected environment 'test-env', got '%s'", config.Environment)
	}

	if config.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", config.LogLevel)
	}

	if config.LogFormat != "text" {
		t.Errorf("Expected log format 'text', got '%s'", config.LogFormat)
	}

	if config.OTelEnabled != false {
		t.Errorf("Expected OTel enabled false, got %v", config.OTelEnabled)
	}

	if config.MetricsEnabled != true {
		t.Errorf("Expected metrics enabled true, got %v", config.MetricsEnabled)
	}

	if config.ProfilingEnabled != false {
		t.Errorf("Expected profiling enabled false, got %v", config.ProfilingEnabled)
	}

	if config.PerformanceMonitoringEnabled != true {
		t.Errorf("Expected performance monitoring enabled true, got %v", config.PerformanceMonitoringEnabled)
	}

	// Test that stack initializes correctly with environment config
	stack, err := observability.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize stack with environment config: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		stack.Shutdown(ctx)
	}()

	// Verify stack components match environment configuration
	if stack.Logger == nil {
		t.Error("Logger should be initialized")
	}

	if config.MetricsEnabled && stack.Metrics == nil {
		t.Error("Metrics should be initialized when enabled in environment")
	}

	if config.PerformanceMonitoringEnabled && stack.PerformanceMonitor == nil {
		t.Error("Performance monitor should be initialized when enabled in environment")
	}
}

// TestObservabilityStackBackgroundServices tests background service lifecycle
func TestObservabilityStackBackgroundServices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping background services test in short mode")
	}

	config := &observability.ObservabilityConfig{
		ServiceName:                  "test-background",
		ServiceVersion:               "1.0.0",
		Environment:                  "test",
		LogLevel:                     "info",
		LogFormat:                    "json",
		OTelEnabled:                  false,
		MetricsEnabled:               true,
		ProfilingEnabled:             true,
		ProfilingInterval:            100 * time.Millisecond, // Short interval for testing
		PerformanceMonitoringEnabled: true,
	}

	stack, err := observability.Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize observability stack: %v", err)
	}

	// Start background services
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	stack.StartBackgroundServices(ctx)

	// Let services run for a short time
	time.Sleep(500 * time.Millisecond)

	// Cancel context to stop background services
	cancel()

	// Give time for services to shutdown
	time.Sleep(200 * time.Millisecond)

	// Shutdown stack
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := stack.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Failed to shutdown stack: %v", err)
	}
}

// TestObservabilityStackMemoryLeak tests for memory leaks in the observability stack
func TestObservabilityStackMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	// Use test utility for memory monitoring
	initialStats := testutil.GetMemStats()

	// Create and destroy multiple stacks
	const iterations = 10

	for i := 0; i < iterations; i++ {
		config := &observability.ObservabilityConfig{
			ServiceName:                  "test-memory",
			ServiceVersion:               "1.0.0",
			Environment:                  "test",
			LogLevel:                     "info",
			LogFormat:                    "json",
			OTelEnabled:                  false,
			MetricsEnabled:               true,
			ProfilingEnabled:             false,
			PerformanceMonitoringEnabled: true,
		}

		stack, err := observability.Initialize(config)
		if err != nil {
			t.Fatalf("Iteration %d: Failed to initialize stack: %v", i, err)
		}

		// Use the stack
		stack.Logger.Info("Memory test iteration", slog.Int("iteration", i))

		if stack.Metrics != nil {
			ctx := context.Background()
			stack.Metrics.RecordRequest(ctx, "GET", "/memory-test", 100*time.Millisecond, 200)
		}

		// Shutdown stack
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		stack.Shutdown(ctx)
		cancel()
	}

	// Force garbage collection
	testutil.ForceGC()

	// Check memory usage
	finalStats := testutil.GetMemStats()

	// Allow for some growth but not excessive
	memoryIncrease := finalStats.Alloc - initialStats.Alloc
	maxAllowedIncrease := uint64(10 * 1024 * 1024) // 10MB threshold

	if memoryIncrease > maxAllowedIncrease {
		t.Errorf("Potential memory leak detected: memory increased by %d bytes (threshold: %d bytes)",
			memoryIncrease, maxAllowedIncrease)
	}

	t.Logf("Memory usage: initial=%d, final=%d, increase=%d bytes",
		initialStats.Alloc, finalStats.Alloc, memoryIncrease)
}
