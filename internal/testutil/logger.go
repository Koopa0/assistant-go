package testutil

import (
	"log/slog"
	"os"
	"runtime"
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

// GetMemStats returns current memory statistics for testing
func GetMemStats() runtime.MemStats {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	return stats
}

// ForceGC forces garbage collection for testing
func ForceGC() {
	runtime.GC()
	runtime.GC() // Run twice to ensure thorough cleanup
}
