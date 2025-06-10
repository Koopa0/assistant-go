package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"sync"
	"time"
)

// ProfileManager manages performance profiling operations
// Implements golang_guide.md recommendations for production profiling
type ProfileManager struct {
	logger   *slog.Logger
	enabled  bool
	interval time.Duration
	mu       sync.RWMutex // mu protects access to 'enabled' and 'interval' fields.
}

// NewProfileManager creates a new profile manager
func NewProfileManager(logger *slog.Logger) *ProfileManager {
	return &ProfileManager{
		logger:   logger,
		enabled:  false,
		interval: time.Minute * 5, // Default 5-minute intervals
	}
}

// EnableProfiling enables periodic profiling collection
func (pm *ProfileManager) EnableProfiling(interval time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.enabled = true
	pm.interval = interval

	pm.logger.Info("Profiling enabled",
		slog.Duration("interval", interval))
}

// DisableProfiling disables profiling collection
func (pm *ProfileManager) DisableProfiling() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.enabled = false
	pm.logger.Info("Profiling disabled")
}

// CollectCPUProfile collects a CPU profile for the specified duration
func (pm *ProfileManager) CollectCPUProfile(ctx context.Context, duration time.Duration, filename string) error {
	pm.logger.Info("Starting CPU profile collection",
		slog.Duration("duration", duration),
		slog.String("file", filename))

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile file: %w", err)
	}
	defer file.Close()

	if err := pprof.StartCPUProfile(file); err != nil {
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}
	defer pprof.StopCPUProfile()

	// Wait for the specified duration or context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
	}

	pm.logger.Info("CPU profile collected successfully",
		slog.String("file", filename))

	return nil
}

// CollectMemProfile collects a memory profile
func (pm *ProfileManager) CollectMemProfile(filename string) error {
	pm.logger.Info("Collecting memory profile",
		slog.String("file", filename))

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create memory profile file: %w", err)
	}
	defer file.Close()

	// Force garbage collection to get more accurate profile
	runtime.GC()

	if err := pprof.WriteHeapProfile(file); err != nil {
		return fmt.Errorf("failed to write memory profile: %w", err)
	}

	pm.logger.Info("Memory profile collected successfully",
		slog.String("file", filename))

	return nil
}

// CollectGoroutineProfile collects a goroutine profile
func (pm *ProfileManager) CollectGoroutineProfile(filename string) error {
	pm.logger.Info("Collecting goroutine profile",
		slog.String("file", filename))

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create goroutine profile file: %w", err)
	}
	defer file.Close()

	profile := pprof.Lookup("goroutine")
	if profile == nil {
		return fmt.Errorf("goroutine profile not found")
	}

	if err := profile.WriteTo(file, 1); err != nil {
		return fmt.Errorf("failed to write goroutine profile: %w", err)
	}

	pm.logger.Info("Goroutine profile collected successfully",
		slog.String("file", filename))

	return nil
}

// CollectTrace collects an execution trace
func (pm *ProfileManager) CollectTrace(ctx context.Context, duration time.Duration, filename string) error {
	pm.logger.Info("Starting trace collection",
		slog.Duration("duration", duration),
		slog.String("file", filename))

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create trace file: %w", err)
	}
	defer file.Close()

	if err := trace.Start(file); err != nil {
		return fmt.Errorf("failed to start trace: %w", err)
	}
	defer trace.Stop()

	// Wait for the specified duration or context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
	}

	pm.logger.Info("Trace collected successfully",
		slog.String("file", filename))

	return nil
}

// StartPeriodicProfiling starts collecting profiles periodically in the background
// This implements the golang_guide.md recommendation for continuous profiling
func (pm *ProfileManager) StartPeriodicProfiling(ctx context.Context) {
	// This goroutine periodically collects various performance profiles
	// (CPU, memory, goroutine) if profiling is enabled.
	// It listens for context cancellation via ctx.Done() for graceful shutdown.
	// The ticker uses the 'pm.interval' value that was set when StartPeriodicProfiling
	// was initially called or last configured via EnableProfiling before this goroutine started.
	// Dynamic changes to 'pm.interval' after this point will not affect this running ticker.
	go func() {
		// Read interval for the ticker.
		// RLock is used to safely read pm.interval for NewTicker,
		// ensuring consistency if EnableProfiling (which takes a WriteLock) is called concurrently.
		pm.mu.RLock()
		intervalForTicker := pm.interval
		pm.mu.RUnlock()
		ticker := time.NewTicker(intervalForTicker)

		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				pm.logger.Info("Stopping periodic profiling")
				return
			case <-ticker.C:
				// Check if profiling is enabled under a read lock before collecting.
				pm.mu.RLock()
				enabled := pm.enabled
				// currentInterval := pm.interval // Read interval in case ticker needs to be reset (not done here)
				pm.mu.RUnlock()
				// Note: If currentInterval changed and ticker needs reset, logic would be more complex.
				// Current design: ticker uses interval set at StartPeriodicProfiling or last EnableProfiling before start.

				if !enabled {
					continue
				}

				// Collect profiles with timestamp
				timestamp := time.Now().Format("20060102-150405")

				// Collect CPU profile
				cpuFile := fmt.Sprintf("cpu-profile-%s.pprof", timestamp)
				if err := pm.CollectCPUProfile(ctx, time.Second*30, cpuFile); err != nil {
					pm.logger.Error("Failed to collect CPU profile",
						slog.Any("error", err))
				}

				// Collect memory profile
				memFile := fmt.Sprintf("mem-profile-%s.pprof", timestamp)
				if err := pm.CollectMemProfile(memFile); err != nil {
					pm.logger.Error("Failed to collect memory profile",
						slog.Any("error", err))
				}

				// Collect goroutine profile
				goroutineFile := fmt.Sprintf("goroutine-profile-%s.pprof", timestamp)
				if err := pm.CollectGoroutineProfile(goroutineFile); err != nil {
					pm.logger.Error("Failed to collect goroutine profile",
						slog.Any("error", err))
				}

				pm.logger.Info("Periodic profiling completed",
					slog.String("timestamp", timestamp))
			}
		}
	}()

	pm.logger.Info("Periodic profiling started",
		slog.Duration("interval", pm.interval))
}

// GetProfileStats returns current profiling statistics
func (pm *ProfileManager) GetProfileStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]interface{}{
		"profiling_enabled":  pm.enabled,
		"profiling_interval": pm.interval.String(),
		"goroutines":         runtime.NumGoroutine(),
		"memory": map[string]interface{}{
			"alloc_mb":         bToMb(memStats.Alloc),
			"total_alloc_mb":   bToMb(memStats.TotalAlloc),
			"sys_mb":           bToMb(memStats.Sys),
			"heap_alloc_mb":    bToMb(memStats.HeapAlloc),
			"heap_sys_mb":      bToMb(memStats.HeapSys),
			"heap_idle_mb":     bToMb(memStats.HeapIdle),
			"heap_inuse_mb":    bToMb(memStats.HeapInuse),
			"heap_released_mb": bToMb(memStats.HeapReleased),
			"gc_cycles":        memStats.NumGC,
			"last_gc_ns":       memStats.LastGC,
		},
		"cpu": map[string]interface{}{
			"num_cpu":    runtime.NumCPU(),
			"gomaxprocs": runtime.GOMAXPROCS(0),
		},
	}
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
