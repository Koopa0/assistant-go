// Package observability provides comprehensive monitoring and observability capabilities.
// It includes structured logging with slog, OpenTelemetry integration for tracing and metrics,
// profiling support, and unified context propagation for request tracking.
package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogFormat represents the logging format
type LogFormat string

const (
	LogFormatJSON LogFormat = "json"
	LogFormatText LogFormat = "text"
)

// ContextKey represents a context key for logging
type ContextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
	// TraceIDKey is the context key for trace ID
	TraceIDKey ContextKey = "trace_id"
	// SpanIDKey is the context key for span ID
	SpanIDKey ContextKey = "span_id"
)

// SetupLogging configures and returns a structured logger
func SetupLogging(level, format string) *slog.Logger {
	return SetupLoggingWithWriter(os.Stdout, level, format)
}

// SetupLoggingWithWriter configures and returns a structured logger with custom writer
func SetupLoggingWithWriter(writer io.Writer, level, format string) *slog.Logger {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   "timestamp",
					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339)),
				}
			}
			// Customize level key
			if a.Key == slog.LevelKey {
				return slog.Attr{
					Key:   "level",
					Value: a.Value,
				}
			}
			// Customize message key
			if a.Key == slog.MessageKey {
				return slog.Attr{
					Key:   "message",
					Value: a.Value,
				}
			}
			return a
		},
	}

	switch strings.ToLower(format) {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		handler = slog.NewJSONHandler(writer, opts)
	}

	return slog.New(handler)
}

// NewLogger creates a new logger with the specified writer and options
func NewLogger(w io.Writer, level LogLevel, format LogFormat) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case LogLevelDebug:
		logLevel = slog.LevelDebug
	case LogLevelInfo:
		logLevel = slog.LevelInfo
	case LogLevelWarn:
		logLevel = slog.LevelWarn
	case LogLevelError:
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   "timestamp",
					Value: slog.StringValue(a.Value.Time().Format(time.RFC3339)),
				}
			}
			return a
		},
	}

	var handler slog.Handler
	switch format {
	case LogFormatJSON:
		handler = slog.NewJSONHandler(w, opts)
	case LogFormatText:
		handler = slog.NewTextHandler(w, opts)
	default:
		handler = slog.NewJSONHandler(w, opts)
	}

	return slog.New(handler)
}

// WithContext creates a logger with context values following golang_guide.md OpenTelemetry best practices
func WithContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	attrs := make([]slog.Attr, 0)

	// Extract OpenTelemetry trace information (golang_guide.md pattern for Loki integration)
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// Add low-cardinality context values (following golang_guide.md Loki labeling strategy)
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		// Store detailed information in log content, not as labels
		attrs = append(attrs, slog.String("request_id", requestID.(string)))
	}

	// Avoid high-cardinality labels for Loki - store in content instead
	if userID := ctx.Value(UserIDKey); userID != nil {
		attrs = append(attrs, slog.String("user_id", userID.(string)))
	}

	if len(attrs) == 0 {
		return logger
	}

	// Convert slog.Attr to any for With method
	args := make([]any, len(attrs))
	for i, attr := range attrs {
		args[i] = attr
	}
	return logger.With(args...)
}

// LogError logs an error with additional context
func LogError(ctx context.Context, logger *slog.Logger, msg string, err error, attrs ...slog.Attr) {
	allAttrs := make([]slog.Attr, 0, len(attrs)+1)
	allAttrs = append(allAttrs, slog.Any("error", err))
	allAttrs = append(allAttrs, attrs...)

	contextLogger := WithContext(ctx, logger)
	contextLogger.LogAttrs(ctx, slog.LevelError, msg, allAttrs...)
}

// LogInfo logs an info message with context
func LogInfo(ctx context.Context, logger *slog.Logger, msg string, attrs ...slog.Attr) {
	contextLogger := WithContext(ctx, logger)
	contextLogger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
}

// LogDebug logs a debug message with context
func LogDebug(ctx context.Context, logger *slog.Logger, msg string, attrs ...slog.Attr) {
	contextLogger := WithContext(ctx, logger)
	contextLogger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

// LogWarn logs a warning message with context
func LogWarn(ctx context.Context, logger *slog.Logger, msg string, attrs ...slog.Attr) {
	contextLogger := WithContext(ctx, logger)
	contextLogger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
}

// RequestLogger creates a logger for HTTP requests
func RequestLogger(logger *slog.Logger, method, path, requestID string) *slog.Logger {
	return logger.With(
		slog.String("method", method),
		slog.String("path", path),
		slog.String("request_id", requestID),
	)
}

// DatabaseLogger creates a logger for database operations
func DatabaseLogger(logger *slog.Logger, operation, table string) *slog.Logger {
	return logger.With(
		slog.String("component", "database"),
		slog.String("operation", operation),
		slog.String("table", table),
	)
}

// ToolLogger creates a logger for tool operations
func ToolLogger(logger *slog.Logger, toolName, operation string) *slog.Logger {
	return logger.With(
		slog.String("component", "tool"),
		slog.String("tool", toolName),
		slog.String("operation", operation),
	)
}

// AILogger creates a logger for AI operations
func AILogger(logger *slog.Logger, provider, model string) *slog.Logger {
	return logger.With(
		slog.String("component", "ai"),
		slog.String("provider", provider),
		slog.String("model", model),
	)
}

// ServerLogger creates a logger for server operations
func ServerLogger(logger *slog.Logger, component string) *slog.Logger {
	return logger.With(
		slog.String("component", "server"),
		slog.String("subcomponent", component),
	)
}

// CLILogger creates a logger for CLI operations
func CLILogger(logger *slog.Logger, command string) *slog.Logger {
	return logger.With(
		slog.String("component", "cli"),
		slog.String("command", command),
	)
}
