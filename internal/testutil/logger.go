package testutil

import (
	"log/slog"
	"os"
)

// NewTestLogger creates a logger suitable for testing
func NewTestLogger() *slog.Logger {
	// Create a logger that outputs to stderr for tests
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return slog.New(handler)
}

// NewSilentLogger creates a logger that discards all output
func NewSilentLogger() *slog.Logger {
	// Create a logger that discards all output
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError + 1, // Higher than any log level
	})
	return slog.New(handler)
}
