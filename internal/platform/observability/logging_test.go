package observability

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// TestSetupLogging tests the logging setup with different configurations
func TestSetupLogging(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		format      string
		expectLevel slog.Level
		expectJSON  bool
	}{
		{
			name:        "debug_json",
			level:       "debug",
			format:      "json",
			expectLevel: slog.LevelDebug,
			expectJSON:  true,
		},
		{
			name:        "info_text",
			level:       "info",
			format:      "text",
			expectLevel: slog.LevelInfo,
			expectJSON:  false,
		},
		{
			name:        "warn_json",
			level:       "warn",
			format:      "json",
			expectLevel: slog.LevelWarn,
			expectJSON:  true,
		},
		{
			name:        "error_default",
			level:       "error",
			format:      "invalid",
			expectLevel: slog.LevelError,
			expectJSON:  true, // Default to JSON
		},
		{
			name:        "invalid_level_default",
			level:       "invalid",
			format:      "json",
			expectLevel: slog.LevelInfo, // Default to Info
			expectJSON:  true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable (golang_guide.md critical requirement)
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // Run tests in parallel (golang_guide.md performance best practice)

			var buf bytes.Buffer
			logger := SetupLoggingWithWriter(&buf, tt.level, tt.format)

			// Test that logger was created
			if logger == nil {
				t.Fatal("Logger should not be nil")
			}

			// Test logging at the configured level
			logger.Info("test message")
			output := buf.String()

			// Verify JSON vs text format
			if tt.expectJSON {
				if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
					t.Errorf("Expected JSON format but got: %s", output)
				}
				if !strings.Contains(output, "\"message\":\"test message\"") && !strings.Contains(output, "\"msg\":\"test message\"") {
					t.Errorf("Expected message field in JSON output: %s", output)
				}
			}

			// Verify timestamp is included
			if !strings.Contains(output, "timestamp") && !strings.Contains(output, "time") {
				t.Errorf("Expected timestamp in output: %s", output)
			}
		})
	}
}

// TestWithContext tests context extraction for logging
func TestWithContext(t *testing.T) {
	tests := []struct {
		name            string
		setupContext    func() context.Context
		expectTraceID   bool
		expectSpanID    bool
		expectRequestID bool
		expectUserID    bool
	}{
		{
			name: "empty_context",
			setupContext: func() context.Context {
				return context.Background()
			},
			expectTraceID:   false,
			expectSpanID:    false,
			expectRequestID: false,
			expectUserID:    false,
		},
		{
			name: "context_with_trace",
			setupContext: func() context.Context {
				tracer := otel.Tracer("test")
				ctx, span := tracer.Start(context.Background(), "test-span")
				defer span.End()
				return ctx
			},
			expectTraceID: true,
			expectSpanID:  true,
		},
		{
			name: "context_with_request_id",
			setupContext: func() context.Context {
				ctx := context.WithValue(context.Background(), RequestIDKey, "req-123")
				return ctx
			},
			expectRequestID: true,
		},
		{
			name: "context_with_user_id",
			setupContext: func() context.Context {
				ctx := context.WithValue(context.Background(), UserIDKey, "user-456")
				return ctx
			},
			expectUserID: true,
		},
		{
			name: "context_with_all",
			setupContext: func() context.Context {
				tracer := otel.Tracer("test")
				ctx, span := tracer.Start(context.Background(), "test-span")
				defer span.End()
				ctx = context.WithValue(ctx, RequestIDKey, "req-789")
				ctx = context.WithValue(ctx, UserIDKey, "user-101")
				return ctx
			},
			expectTraceID:   true,
			expectSpanID:    true,
			expectRequestID: true,
			expectUserID:    true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			baseLogger := SetupLoggingWithWriter(&buf, "debug", "json")

			ctx := tt.setupContext()
			contextLogger := WithContext(ctx, baseLogger)

			// Log a test message
			contextLogger.Info("test message with context")
			output := buf.String()

			// Check for trace ID
			if tt.expectTraceID {
				if !strings.Contains(output, "trace_id") {
					t.Errorf("Expected trace_id in output: %s", output)
				}
			} else {
				if strings.Contains(output, "trace_id") {
					t.Errorf("Unexpected trace_id in output: %s", output)
				}
			}

			// Check for span ID
			if tt.expectSpanID {
				if !strings.Contains(output, "span_id") {
					t.Errorf("Expected span_id in output: %s", output)
				}
			} else {
				if strings.Contains(output, "span_id") {
					t.Errorf("Unexpected span_id in output: %s", output)
				}
			}

			// Check for request ID
			if tt.expectRequestID {
				if !strings.Contains(output, "request_id") {
					t.Errorf("Expected request_id in output: %s", output)
				}
			}

			// Check for user ID
			if tt.expectUserID {
				if !strings.Contains(output, "user_id") {
					t.Errorf("Expected user_id in output: %s", output)
				}
			}
		})
	}
}

// TestLogFunctions tests the convenience logging functions
func TestLogFunctions(t *testing.T) {
	tests := []struct {
		name    string
		logFunc func(context.Context, *slog.Logger, string, ...slog.Attr)
		level   slog.Level
		message string
		attrs   []slog.Attr
	}{
		{
			name: "log_error",
			logFunc: func(ctx context.Context, logger *slog.Logger, msg string, attrs ...slog.Attr) {
				// LogError requires an error parameter, so we create a test error
				testErr := fmt.Errorf("test error")
				LogError(ctx, logger, msg, testErr, attrs...)
			},
			level:   slog.LevelError,
			message: "error occurred",
			attrs:   []slog.Attr{slog.String("component", "test")},
		},
		{
			name:    "log_info",
			logFunc: LogInfo,
			level:   slog.LevelInfo,
			message: "info message",
			attrs:   []slog.Attr{slog.String("operation", "test")},
		},
		{
			name:    "log_debug",
			logFunc: LogDebug,
			level:   slog.LevelDebug,
			message: "debug message",
			attrs:   []slog.Attr{slog.Int("count", 42)},
		},
		{
			name:    "log_warn",
			logFunc: LogWarn,
			level:   slog.LevelWarn,
			message: "warning message",
			attrs:   []slog.Attr{slog.Bool("deprecated", true)},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			logger := SetupLoggingWithWriter(&buf, "debug", "json")
			ctx := context.Background()

			// Call the log function
			tt.logFunc(ctx, logger, tt.message, tt.attrs...)
			output := buf.String()

			// Verify message is present
			if !strings.Contains(output, tt.message) {
				t.Errorf("Expected message '%s' in output: %s", tt.message, output)
			}

			// Verify attributes are present
			for _, attr := range tt.attrs {
				if !strings.Contains(output, attr.Key) {
					t.Errorf("Expected attribute key '%s' in output: %s", attr.Key, output)
				}
			}
		})
	}
}

// TestSpecializedLoggers tests the specialized logger creation functions
func TestSpecializedLoggers(t *testing.T) {
	tests := []struct {
		name       string
		createFunc func(*slog.Logger) *slog.Logger
		expectKeys []string
	}{
		{
			name: "request_logger",
			createFunc: func(logger *slog.Logger) *slog.Logger {
				return RequestLogger(logger, "GET", "/api/query", "req-123")
			},
			expectKeys: []string{"method", "path", "request_id"},
		},
		{
			name: "database_logger",
			createFunc: func(logger *slog.Logger) *slog.Logger {
				return DatabaseLogger(logger, "SELECT", "conversations")
			},
			expectKeys: []string{"component", "operation", "table"},
		},
		{
			name: "tool_logger",
			createFunc: func(logger *slog.Logger) *slog.Logger {
				return ToolLogger(logger, "go_analyzer", "analyze")
			},
			expectKeys: []string{"component", "tool", "operation"},
		},
		{
			name: "ai_logger",
			createFunc: func(logger *slog.Logger) *slog.Logger {
				return AILogger(logger, "claude", "claude-3-sonnet")
			},
			expectKeys: []string{"component", "provider", "model"},
		},
		{
			name: "server_logger",
			createFunc: func(logger *slog.Logger) *slog.Logger {
				return ServerLogger(logger, "http")
			},
			expectKeys: []string{"component", "subcomponent"},
		},
		{
			name: "cli_logger",
			createFunc: func(logger *slog.Logger) *slog.Logger {
				return CLILogger(logger, "query")
			},
			expectKeys: []string{"component", "command"},
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			baseLogger := SetupLoggingWithWriter(&buf, "debug", "json")

			specializedLogger := tt.createFunc(baseLogger)

			// Log a test message
			specializedLogger.Info("test message")
			output := buf.String()

			// Verify all expected keys are present
			for _, key := range tt.expectKeys {
				if !strings.Contains(output, key) {
					t.Errorf("Expected key '%s' in output: %s", key, output)
				}
			}

			// Verify message is present
			if !strings.Contains(output, "test message") {
				t.Errorf("Expected test message in output: %s", output)
			}
		})
	}
}

// TestLoggerPerformance tests logging performance under concurrent load
func TestLoggerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	var buf bytes.Buffer
	logger := SetupLoggingWithWriter(&buf, "info", "json")

	// Benchmark concurrent logging
	const numGoroutines = 10
	const messagesPerGoroutine = 100

	start := time.Now()

	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Info("performance test message",
					slog.Int("goroutine", id),
					slog.Int("message", j),
					slog.String("data", "some test data"))
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	duration := time.Since(start)
	totalMessages := numGoroutines * messagesPerGoroutine
	messagesPerSecond := float64(totalMessages) / duration.Seconds()

	t.Logf("Logged %d messages in %v (%.2f messages/second)",
		totalMessages, duration, messagesPerSecond)

	// Sanity check: we should be able to log at least 1000 messages per second
	if messagesPerSecond < 1000 {
		t.Errorf("Logging performance too slow: %.2f messages/second", messagesPerSecond)
	}
}

// BenchmarkLogging benchmarks different logging configurations
func BenchmarkLogging(b *testing.B) {
	benchmarks := []struct {
		name   string
		level  string
		format string
	}{
		{"json_info", "info", "json"},
		{"text_info", "info", "text"},
		{"json_debug", "debug", "json"},
		{"text_debug", "debug", "text"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			var buf bytes.Buffer
			logger := SetupLoggingWithWriter(&buf, bm.level, bm.format)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					logger.Info("benchmark message",
						slog.String("key1", "value1"),
						slog.Int("key2", 42),
						slog.Bool("key3", true))
				}
			})
		})
	}
}

// Mock tracer for testing
type mockSpan struct {
	trace.Span
	spanContext trace.SpanContext
}

func (m *mockSpan) SpanContext() trace.SpanContext {
	return m.spanContext
}

// TestWithContextMockTracing tests context extraction with mocked tracing
func TestWithContextMockTracing(t *testing.T) {
	// Create a mock span context
	traceID, _ := trace.TraceIDFromHex("12345678901234567890123456789012")
	spanID, _ := trace.SpanIDFromHex("1234567890123456")

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	})

	mockSpan := &mockSpan{spanContext: spanContext}
	ctx := trace.ContextWithSpan(context.Background(), mockSpan)

	var buf bytes.Buffer
	baseLogger := SetupLoggingWithWriter(&buf, "debug", "json")
	contextLogger := WithContext(ctx, baseLogger)

	contextLogger.Info("test with mock span")
	output := buf.String()

	// Verify trace ID and span ID are included
	if !strings.Contains(output, traceID.String()) {
		t.Errorf("Expected trace ID %s in output: %s", traceID.String(), output)
	}

	if !strings.Contains(output, spanID.String()) {
		t.Errorf("Expected span ID %s in output: %s", spanID.String(), output)
	}
}
