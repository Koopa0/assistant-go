package tools

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/koopa0/assistant-go/internal/observability"
)

// Pipeline represents a tool execution pipeline with monitoring and optimization
type Pipeline struct {
	registry         *Registry
	metricsCollector *observability.MetricsCollector
	rateLimiter      *observability.RateLimiter
	logger           *slog.Logger
	config           *PipelineConfig
	executionHistory map[string]*ExecutionHistory
	mutex            sync.RWMutex
}

// PipelineConfig contains configuration for the tool pipeline
type PipelineConfig struct {
	MaxConcurrentExecutions int           `json:"max_concurrent_executions"`
	DefaultTimeout          time.Duration `json:"default_timeout"`
	RetryAttempts           int           `json:"retry_attempts"`
	RetryDelay              time.Duration `json:"retry_delay"`
	EnableMetrics           bool          `json:"enable_metrics"`
	EnableRateLimiting      bool          `json:"enable_rate_limiting"`
	CacheResults            bool          `json:"cache_results"`
	CacheTTL                time.Duration `json:"cache_ttl"`
}

// ExecutionHistory tracks execution history for a tool
type ExecutionHistory struct {
	ToolName         string                `json:"tool_name"`
	TotalExecutions  int64                 `json:"total_executions"`
	SuccessfulRuns   int64                 `json:"successful_runs"`
	FailedRuns       int64                 `json:"failed_runs"`
	AverageExecTime  time.Duration         `json:"average_execution_time"`
	LastExecution    time.Time             `json:"last_execution"`
	RecentResults    []*ExecutionResult    `json:"recent_results"`
	PerformanceStats *ToolPerformanceStats `json:"performance_stats"`
}

// ExecutionResult represents the result of a tool execution
type ExecutionResult struct {
	ID           string                 `json:"id"`
	ToolName     string                 `json:"tool_name"`
	Input        map[string]interface{} `json:"input"`
	Output       *ToolResult            `json:"output"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Duration     time.Duration          `json:"duration"`
	Success      bool                   `json:"success"`
	Error        string                 `json:"error,omitempty"`
	RetryAttempt int                    `json:"retry_attempt"`
	CacheHit     bool                   `json:"cache_hit"`
}

// ToolPerformanceStats contains performance statistics for a tool
type ToolPerformanceStats struct {
	MinExecutionTime    time.Duration `json:"min_execution_time"`
	MaxExecutionTime    time.Duration `json:"max_execution_time"`
	P50ExecutionTime    time.Duration `json:"p50_execution_time"`
	P95ExecutionTime    time.Duration `json:"p95_execution_time"`
	P99ExecutionTime    time.Duration `json:"p99_execution_time"`
	SuccessRate         float64       `json:"success_rate"`
	ErrorRate           float64       `json:"error_rate"`
	ThroughputPerMinute float64       `json:"throughput_per_minute"`
}

// ExecutionContext contains context for tool execution
type ExecutionContext struct {
	Context     context.Context
	RequestID   string
	UserID      string
	SessionID   string
	Priority    int
	Metadata    map[string]interface{}
	Timeout     time.Duration
	RetryPolicy *RetryPolicy
}

// RetryPolicy defines retry behavior for tool execution
type RetryPolicy struct {
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	Backoff     float64       `json:"backoff"`
	MaxDelay    time.Duration `json:"max_delay"`
}

// NewPipeline creates a new tool execution pipeline
func NewPipeline(registry *Registry, metricsCollector *observability.MetricsCollector, logger *slog.Logger) *Pipeline {
	config := &PipelineConfig{
		MaxConcurrentExecutions: 10,
		DefaultTimeout:          30 * time.Second,
		RetryAttempts:           3,
		RetryDelay:              1 * time.Second,
		EnableMetrics:           true,
		EnableRateLimiting:      true,
		CacheResults:            true,
		CacheTTL:                5 * time.Minute,
	}

	rateLimiter := observability.NewRateLimiter(metricsCollector, logger)

	// Set default rate limits for tools
	rateLimiter.SetProviderLimit("go-tools", 60, 0)     // 60 requests per minute
	rateLimiter.SetProviderLimit("docker-tools", 30, 0) // 30 requests per minute
	rateLimiter.SetProviderLimit("k8s-tools", 20, 0)    // 20 requests per minute

	return &Pipeline{
		registry:         registry,
		metricsCollector: metricsCollector,
		rateLimiter:      rateLimiter,
		logger:           logger,
		config:           config,
		executionHistory: make(map[string]*ExecutionHistory),
	}
}

// ExecuteWithPipeline executes a tool through the enhanced pipeline
func (p *Pipeline) ExecuteWithPipeline(execCtx *ExecutionContext, toolName string, input map[string]interface{}) (*ExecutionResult, error) {
	startTime := time.Now()

	// Generate execution ID
	executionID := fmt.Sprintf("%s-%d", toolName, startTime.UnixNano())

	p.logger.Info("Starting tool execution",
		slog.String("execution_id", executionID),
		slog.String("tool_name", toolName),
		slog.String("request_id", execCtx.RequestID),
		slog.String("user_id", execCtx.UserID))

	// Check rate limits if enabled
	if p.config.EnableRateLimiting {
		toolCategory := p.getToolCategory(toolName)
		if !p.rateLimiter.CheckLimit(toolCategory, 1) {
			return nil, fmt.Errorf("rate limit exceeded for tool category: %s", toolCategory)
		}
	}

	// Check cache if enabled
	if p.config.CacheResults {
		if cachedResult := p.getCachedResult(toolName, input); cachedResult != nil {
			cachedResult.ID = executionID
			cachedResult.CacheHit = true
			p.recordExecution(cachedResult)
			return cachedResult, nil
		}
	}

	// Set timeout
	timeout := p.config.DefaultTimeout
	if execCtx.Timeout > 0 {
		timeout = execCtx.Timeout
	}

	ctx, cancel := context.WithTimeout(execCtx.Context, timeout)
	defer cancel()

	// Execute with retry logic
	var result *ExecutionResult
	var err error

	retryPolicy := execCtx.RetryPolicy
	if retryPolicy == nil {
		retryPolicy = &RetryPolicy{
			MaxAttempts: p.config.RetryAttempts,
			Delay:       p.config.RetryDelay,
			Backoff:     2.0,
			MaxDelay:    30 * time.Second,
		}
	}

	for attempt := 1; attempt <= retryPolicy.MaxAttempts; attempt++ {
		attemptStart := time.Now()

		toolInput := ConvertLegacyInput(input)
		toolResult, execErr := p.registry.Execute(ctx, toolName, toolInput, nil)
		attemptEnd := time.Now()

		result = &ExecutionResult{
			ID:           executionID,
			ToolName:     toolName,
			Input:        input,
			Output:       toolResult,
			StartTime:    attemptStart,
			EndTime:      attemptEnd,
			Duration:     attemptEnd.Sub(attemptStart),
			Success:      execErr == nil,
			RetryAttempt: attempt,
		}

		if execErr != nil {
			result.Error = execErr.Error()
			err = execErr

			// Check if error is retryable
			if !p.isRetryableError(execErr) || attempt == retryPolicy.MaxAttempts {
				break
			}

			// Calculate delay for next attempt
			delay := time.Duration(float64(retryPolicy.Delay) *
				pow(retryPolicy.Backoff, float64(attempt-1)))
			if delay > retryPolicy.MaxDelay {
				delay = retryPolicy.MaxDelay
			}

			p.logger.Warn("Tool execution failed, retrying",
				slog.String("execution_id", executionID),
				slog.String("tool_name", toolName),
				slog.Int("attempt", attempt),
				slog.Duration("delay", delay),
				slog.String("error", execErr.Error()))

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		} else {
			// Success
			err = nil
			break
		}
	}

	// Update total execution time
	result.StartTime = startTime
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Cache successful results
	if p.config.CacheResults && result.Success {
		p.cacheResult(toolName, input, result)
	}

	// Record execution metrics and history
	p.recordExecution(result)

	if err != nil {
		p.logger.Error("Tool execution failed after all retries",
			slog.String("execution_id", executionID),
			slog.String("tool_name", toolName),
			slog.String("error", err.Error()))
		return result, err
	}

	p.logger.Info("Tool execution completed successfully",
		slog.String("execution_id", executionID),
		slog.String("tool_name", toolName),
		slog.Duration("total_duration", result.Duration))

	return result, nil
}

// recordExecution records execution metrics and updates history
func (p *Pipeline) recordExecution(result *ExecutionResult) {
	if p.config.EnableMetrics {
		// Record metrics
		labels := map[string]string{
			"tool": result.ToolName,
		}

		p.metricsCollector.Counter("tool_executions_total", labels, 1)
		p.metricsCollector.Histogram("tool_execution_duration_seconds", labels, result.Duration.Seconds())

		if result.Success {
			p.metricsCollector.Counter("tool_executions_success_total", labels, 1)
		} else {
			p.metricsCollector.Counter("tool_executions_error_total", labels, 1)
		}

		if result.CacheHit {
			p.metricsCollector.Counter("tool_cache_hits_total", labels, 1)
		}
	}

	// Update execution history
	p.updateExecutionHistory(result)
}

// updateExecutionHistory updates the execution history for a tool
func (p *Pipeline) updateExecutionHistory(result *ExecutionResult) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	history, exists := p.executionHistory[result.ToolName]
	if !exists {
		history = &ExecutionHistory{
			ToolName:         result.ToolName,
			RecentResults:    make([]*ExecutionResult, 0, 100),
			PerformanceStats: &ToolPerformanceStats{},
		}
		p.executionHistory[result.ToolName] = history
	}

	// Update counters
	history.TotalExecutions++
	if result.Success {
		history.SuccessfulRuns++
	} else {
		history.FailedRuns++
	}

	// Update average execution time
	if history.TotalExecutions == 1 {
		history.AverageExecTime = result.Duration
	} else {
		// Simple moving average
		history.AverageExecTime = time.Duration(
			(int64(history.AverageExecTime)*int64(history.TotalExecutions-1) + int64(result.Duration)) / int64(history.TotalExecutions),
		)
	}

	history.LastExecution = result.EndTime

	// Add to recent results (keep last 100)
	history.RecentResults = append(history.RecentResults, result)
	if len(history.RecentResults) > 100 {
		history.RecentResults = history.RecentResults[1:]
	}

	// Update performance stats
	p.updatePerformanceStats(history)
}

// updatePerformanceStats updates performance statistics
func (p *Pipeline) updatePerformanceStats(history *ExecutionHistory) {
	if len(history.RecentResults) == 0 {
		return
	}

	stats := history.PerformanceStats

	// Calculate min/max execution times
	var durations []time.Duration
	var successCount int64

	for _, result := range history.RecentResults {
		durations = append(durations, result.Duration)
		if result.Success {
			successCount++
		}
	}

	if len(durations) > 0 {
		// Sort durations for percentile calculations
		sortDurations(durations)

		stats.MinExecutionTime = durations[0]
		stats.MaxExecutionTime = durations[len(durations)-1]
		stats.P50ExecutionTime = durations[len(durations)/2]
		stats.P95ExecutionTime = durations[int(float64(len(durations))*0.95)]
		stats.P99ExecutionTime = durations[int(float64(len(durations))*0.99)]
	}

	// Calculate success/error rates
	if history.TotalExecutions > 0 {
		stats.SuccessRate = float64(history.SuccessfulRuns) / float64(history.TotalExecutions)
		stats.ErrorRate = float64(history.FailedRuns) / float64(history.TotalExecutions)
	}

	// Calculate throughput (executions per minute)
	if len(history.RecentResults) > 1 {
		timeSpan := history.RecentResults[len(history.RecentResults)-1].EndTime.Sub(
			history.RecentResults[0].StartTime)
		if timeSpan > 0 {
			stats.ThroughputPerMinute = float64(len(history.RecentResults)) / timeSpan.Minutes()
		}
	}
}

// GetExecutionHistory returns execution history for a tool
func (p *Pipeline) GetExecutionHistory(toolName string) *ExecutionHistory {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if history, exists := p.executionHistory[toolName]; exists {
		// Return a copy to avoid concurrent access issues
		return &ExecutionHistory{
			ToolName:         history.ToolName,
			TotalExecutions:  history.TotalExecutions,
			SuccessfulRuns:   history.SuccessfulRuns,
			FailedRuns:       history.FailedRuns,
			AverageExecTime:  history.AverageExecTime,
			LastExecution:    history.LastExecution,
			PerformanceStats: history.PerformanceStats,
		}
	}
	return nil
}

// GetAllExecutionHistory returns execution history for all tools
func (p *Pipeline) GetAllExecutionHistory() map[string]*ExecutionHistory {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make(map[string]*ExecutionHistory)
	for toolName, history := range p.executionHistory {
		result[toolName] = &ExecutionHistory{
			ToolName:         history.ToolName,
			TotalExecutions:  history.TotalExecutions,
			SuccessfulRuns:   history.SuccessfulRuns,
			FailedRuns:       history.FailedRuns,
			AverageExecTime:  history.AverageExecTime,
			LastExecution:    history.LastExecution,
			PerformanceStats: history.PerformanceStats,
		}
	}
	return result
}

// Helper functions

func (p *Pipeline) getToolCategory(toolName string) string {
	// Categorize tools for rate limiting
	if contains(toolName, "go-") {
		return "go-tools"
	}
	if contains(toolName, "docker-") {
		return "docker-tools"
	}
	if contains(toolName, "k8s-") || contains(toolName, "kubectl-") {
		return "k8s-tools"
	}
	return "general-tools"
}

func (p *Pipeline) isRetryableError(err error) bool {
	// Define which errors are retryable
	errorStr := err.Error()
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"temporary failure",
		"rate limit",
		"service unavailable",
	}

	for _, retryable := range retryableErrors {
		if contains(errorStr, retryable) {
			return true
		}
	}
	return false
}

func (p *Pipeline) getCachedResult(toolName string, input map[string]interface{}) *ExecutionResult {
	// Simple cache implementation - in production, use Redis or similar
	// For now, return nil (no cache hit)
	return nil
}

func (p *Pipeline) cacheResult(toolName string, input map[string]interface{}, result *ExecutionResult) {
	// Simple cache implementation - in production, use Redis or similar
	// For now, do nothing
}

// Utility functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr || containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

func sortDurations(durations []time.Duration) {
	// Simple bubble sort for durations
	n := len(durations)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if durations[j] > durations[j+1] {
				durations[j], durations[j+1] = durations[j+1], durations[j]
			}
		}
	}
}
