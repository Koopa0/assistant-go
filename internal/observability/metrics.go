package observability

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a single metric
type Metric struct {
	Name      string                 `json:"name"`
	Type      MetricType             `json:"type"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MetricsCollector collects and manages application metrics
type MetricsCollector struct {
	metrics map[string]*Metric
	mutex   sync.RWMutex
	logger  *slog.Logger
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger *slog.Logger) *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*Metric),
		logger:  logger,
	}
}

// Counter increments a counter metric
func (mc *MetricsCollector) Counter(name string, labels map[string]string, value float64) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	key := mc.buildKey(name, labels)
	if metric, exists := mc.metrics[key]; exists {
		metric.Value += value
		metric.Timestamp = time.Now()
	} else {
		mc.metrics[key] = &Metric{
			Name:      name,
			Type:      MetricTypeCounter,
			Value:     value,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}

	mc.logger.Debug("Counter metric updated",
		slog.String("name", name),
		slog.Float64("value", value),
		slog.Any("labels", labels))
}

// Gauge sets a gauge metric value
func (mc *MetricsCollector) Gauge(name string, labels map[string]string, value float64) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	key := mc.buildKey(name, labels)
	mc.metrics[key] = &Metric{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}

	mc.logger.Debug("Gauge metric set",
		slog.String("name", name),
		slog.Float64("value", value),
		slog.Any("labels", labels))
}

// Histogram records a histogram metric (simplified as gauge for now)
func (mc *MetricsCollector) Histogram(name string, labels map[string]string, value float64) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	key := mc.buildKey(name, labels)
	mc.metrics[key] = &Metric{
		Name:      name,
		Type:      MetricTypeHistogram,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}

	mc.logger.Debug("Histogram metric recorded",
		slog.String("name", name),
		slog.Float64("value", value),
		slog.Any("labels", labels))
}

// GetMetrics returns all collected metrics
func (mc *MetricsCollector) GetMetrics() map[string]*Metric {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]*Metric)
	for k, v := range mc.metrics {
		result[k] = &Metric{
			Name:      v.Name,
			Type:      v.Type,
			Value:     v.Value,
			Labels:    v.Labels,
			Timestamp: v.Timestamp,
			Metadata:  v.Metadata,
		}
	}

	return result
}

// GetMetric returns a specific metric
func (mc *MetricsCollector) GetMetric(name string, labels map[string]string) *Metric {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	key := mc.buildKey(name, labels)
	if metric, exists := mc.metrics[key]; exists {
		return &Metric{
			Name:      metric.Name,
			Type:      metric.Type,
			Value:     metric.Value,
			Labels:    metric.Labels,
			Timestamp: metric.Timestamp,
			Metadata:  metric.Metadata,
		}
	}

	return nil
}

// Reset clears all metrics
func (mc *MetricsCollector) Reset() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.metrics = make(map[string]*Metric)
	mc.logger.Debug("Metrics reset")
}

// buildKey creates a unique key for a metric
func (mc *MetricsCollector) buildKey(name string, labels map[string]string) string {
	key := name
	if labels != nil {
		for k, v := range labels {
			key += ":" + k + "=" + v
		}
	}
	return key
}

// AIMetrics tracks AI-specific performance metrics
type AIMetrics struct {
	collector *MetricsCollector
	logger    *slog.Logger
}

// NewAIMetrics creates a new AI metrics tracker
func NewAIMetrics(collector *MetricsCollector, logger *slog.Logger) *AIMetrics {
	return &AIMetrics{
		collector: collector,
		logger:    logger,
	}
}

// RecordRequest records an AI request
func (am *AIMetrics) RecordRequest(provider, model string, startTime time.Time, success bool, tokenUsage map[string]int, errorType string) {
	duration := time.Since(startTime)
	labels := map[string]string{
		"provider": provider,
		"model":    model,
	}

	// Record request count
	am.collector.Counter("ai_requests_total", labels, 1)

	// Record response time
	am.collector.Histogram("ai_response_time_seconds", labels, duration.Seconds())

	// Record success/failure
	if success {
		am.collector.Counter("ai_requests_success_total", labels, 1)
	} else {
		errorLabels := map[string]string{
			"provider":   provider,
			"model":      model,
			"error_type": errorType,
		}
		am.collector.Counter("ai_requests_error_total", errorLabels, 1)
	}

	// Record token usage
	if tokenUsage != nil {
		if inputTokens, ok := tokenUsage["input"]; ok {
			am.collector.Counter("ai_tokens_input_total", labels, float64(inputTokens))
		}
		if outputTokens, ok := tokenUsage["output"]; ok {
			am.collector.Counter("ai_tokens_output_total", labels, float64(outputTokens))
		}
		if totalTokens, ok := tokenUsage["total"]; ok {
			am.collector.Counter("ai_tokens_total", labels, float64(totalTokens))
		}
	}

	am.logger.Debug("AI request metrics recorded",
		slog.String("provider", provider),
		slog.String("model", model),
		slog.Duration("duration", duration),
		slog.Bool("success", success),
		slog.Any("token_usage", tokenUsage))
}

// RecordEmbedding records an embedding generation request
func (am *AIMetrics) RecordEmbedding(provider, model string, startTime time.Time, success bool, tokensUsed int, dimensions int, errorType string) {
	duration := time.Since(startTime)
	labels := map[string]string{
		"provider": provider,
		"model":    model,
	}

	// Record embedding request count
	am.collector.Counter("ai_embeddings_total", labels, 1)

	// Record response time
	am.collector.Histogram("ai_embedding_response_time_seconds", labels, duration.Seconds())

	// Record success/failure
	if success {
		am.collector.Counter("ai_embeddings_success_total", labels, 1)
		am.collector.Gauge("ai_embedding_dimensions", labels, float64(dimensions))
	} else {
		errorLabels := map[string]string{
			"provider":   provider,
			"model":      model,
			"error_type": errorType,
		}
		am.collector.Counter("ai_embeddings_error_total", errorLabels, 1)
	}

	// Record token usage
	if tokensUsed > 0 {
		am.collector.Counter("ai_embedding_tokens_total", labels, float64(tokensUsed))
	}

	am.logger.Debug("AI embedding metrics recorded",
		slog.String("provider", provider),
		slog.String("model", model),
		slog.Duration("duration", duration),
		slog.Bool("success", success),
		slog.Int("tokens_used", tokensUsed),
		slog.Int("dimensions", dimensions))
}

// GetAIStats returns aggregated AI statistics
func (am *AIMetrics) GetAIStats() map[string]interface{} {
	metrics := am.collector.GetMetrics()
	stats := make(map[string]interface{})

	// Aggregate by provider
	providerStats := make(map[string]map[string]interface{})

	for _, metric := range metrics {
		if provider, exists := metric.Labels["provider"]; exists {
			if providerStats[provider] == nil {
				providerStats[provider] = make(map[string]interface{})
			}

			switch metric.Name {
			case "ai_requests_total":
				providerStats[provider]["total_requests"] = metric.Value
			case "ai_requests_success_total":
				providerStats[provider]["successful_requests"] = metric.Value
			case "ai_requests_error_total":
				providerStats[provider]["failed_requests"] = metric.Value
			case "ai_tokens_total":
				providerStats[provider]["total_tokens"] = metric.Value
			case "ai_tokens_input_total":
				providerStats[provider]["input_tokens"] = metric.Value
			case "ai_tokens_output_total":
				providerStats[provider]["output_tokens"] = metric.Value
			}
		}
	}

	stats["providers"] = providerStats
	stats["last_updated"] = time.Now()

	return stats
}

// PerformanceMonitor monitors overall system performance
type PerformanceMonitor struct {
	collector *MetricsCollector
	aiMetrics *AIMetrics
	logger    *slog.Logger
	startTime time.Time
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(logger *slog.Logger) *PerformanceMonitor {
	collector := NewMetricsCollector(logger)
	aiMetrics := NewAIMetrics(collector, logger)

	return &PerformanceMonitor{
		collector: collector,
		aiMetrics: aiMetrics,
		logger:    logger,
		startTime: time.Now(),
	}
}

// GetCollector returns the metrics collector
func (pm *PerformanceMonitor) GetCollector() *MetricsCollector {
	return pm.collector
}

// GetAIMetrics returns the AI metrics tracker
func (pm *PerformanceMonitor) GetAIMetrics() *AIMetrics {
	return pm.aiMetrics
}

// GetSystemStats returns overall system statistics
func (pm *PerformanceMonitor) GetSystemStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// System uptime
	stats["uptime_seconds"] = time.Since(pm.startTime).Seconds()

	// AI statistics
	stats["ai"] = pm.aiMetrics.GetAIStats()

	// Metrics count
	metrics := pm.collector.GetMetrics()
	stats["metrics_count"] = len(metrics)

	return stats
}

// Health checks the health of the performance monitor
func (pm *PerformanceMonitor) Health(ctx context.Context) error {
	// Performance monitor is always healthy if it can respond
	pm.collector.Gauge("system_health", map[string]string{"component": "performance_monitor"}, 1)
	return nil
}

// RateLimiter implements AI provider rate limiting
type RateLimiter struct {
	limits    map[string]*ProviderLimit
	mutex     sync.RWMutex
	logger    *slog.Logger
	collector *MetricsCollector
}

// ProviderLimit represents rate limiting configuration for a provider
type ProviderLimit struct {
	RequestsPerMinute int
	TokensPerMinute   int
	RequestCount      int
	TokenCount        int
	LastReset         time.Time
	Blocked           bool
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(collector *MetricsCollector, logger *slog.Logger) *RateLimiter {
	return &RateLimiter{
		limits:    make(map[string]*ProviderLimit),
		logger:    logger,
		collector: collector,
	}
}

// SetProviderLimit sets rate limiting for a provider
func (rl *RateLimiter) SetProviderLimit(provider string, requestsPerMinute, tokensPerMinute int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.limits[provider] = &ProviderLimit{
		RequestsPerMinute: requestsPerMinute,
		TokensPerMinute:   tokensPerMinute,
		LastReset:         time.Now(),
	}

	rl.logger.Info("Rate limit set for provider",
		slog.String("provider", provider),
		slog.Int("requests_per_minute", requestsPerMinute),
		slog.Int("tokens_per_minute", tokensPerMinute))
}

// CheckLimit checks if a request is within rate limits
func (rl *RateLimiter) CheckLimit(provider string, tokensRequested int) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	limit, exists := rl.limits[provider]
	if !exists {
		// No limit set, allow request
		return true
	}

	now := time.Now()

	// Reset counters if a minute has passed
	if now.Sub(limit.LastReset) >= time.Minute {
		limit.RequestCount = 0
		limit.TokenCount = 0
		limit.LastReset = now
		limit.Blocked = false
	}

	// Check if adding this request would exceed limits
	wouldExceedRequests := limit.RequestCount+1 > limit.RequestsPerMinute
	wouldExceedTokens := limit.TokenCount+tokensRequested > limit.TokensPerMinute

	if wouldExceedRequests || wouldExceedTokens {
		limit.Blocked = true

		// Record rate limit hit
		labels := map[string]string{"provider": provider}
		rl.collector.Counter("rate_limit_hits_total", labels, 1)

		rl.logger.Warn("Rate limit exceeded",
			slog.String("provider", provider),
			slog.Int("current_requests", limit.RequestCount),
			slog.Int("current_tokens", limit.TokenCount),
			slog.Int("requested_tokens", tokensRequested),
			slog.Bool("requests_exceeded", wouldExceedRequests),
			slog.Bool("tokens_exceeded", wouldExceedTokens))

		return false
	}

	// Update counters
	limit.RequestCount++
	limit.TokenCount += tokensRequested

	return true
}

// GetLimitStatus returns current rate limit status for all providers
func (rl *RateLimiter) GetLimitStatus() map[string]*ProviderLimit {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	result := make(map[string]*ProviderLimit)
	for provider, limit := range rl.limits {
		result[provider] = &ProviderLimit{
			RequestsPerMinute: limit.RequestsPerMinute,
			TokensPerMinute:   limit.TokensPerMinute,
			RequestCount:      limit.RequestCount,
			TokenCount:        limit.TokenCount,
			LastReset:         limit.LastReset,
			Blocked:           limit.Blocked,
		}
	}
	return result
}

// PerformanceRegression detects performance regressions
type PerformanceRegression struct {
	collector  *MetricsCollector
	logger     *slog.Logger
	baselines  map[string]float64
	thresholds map[string]float64
	mutex      sync.RWMutex
}

// NewPerformanceRegression creates a new performance regression detector
func NewPerformanceRegression(collector *MetricsCollector, logger *slog.Logger) *PerformanceRegression {
	return &PerformanceRegression{
		collector:  collector,
		logger:     logger,
		baselines:  make(map[string]float64),
		thresholds: make(map[string]float64),
	}
}

// SetBaseline sets a performance baseline for a metric
func (pr *PerformanceRegression) SetBaseline(metricName string, baseline float64, threshold float64) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()

	pr.baselines[metricName] = baseline
	pr.thresholds[metricName] = threshold

	pr.logger.Info("Performance baseline set",
		slog.String("metric", metricName),
		slog.Float64("baseline", baseline),
		slog.Float64("threshold", threshold))
}

// CheckRegression checks for performance regressions
func (pr *PerformanceRegression) CheckRegression(metricName string, currentValue float64) bool {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	baseline, hasBaseline := pr.baselines[metricName]
	threshold, hasThreshold := pr.thresholds[metricName]

	if !hasBaseline || !hasThreshold {
		return false // No baseline set, can't detect regression
	}

	// Calculate percentage change
	percentChange := ((currentValue - baseline) / baseline) * 100

	if percentChange > threshold {
		pr.logger.Warn("Performance regression detected",
			slog.String("metric", metricName),
			slog.Float64("baseline", baseline),
			slog.Float64("current", currentValue),
			slog.Float64("percent_change", percentChange),
			slog.Float64("threshold", threshold))

		// Record regression metric
		labels := map[string]string{"metric": metricName}
		pr.collector.Counter("performance_regressions_total", labels, 1)
		pr.collector.Gauge("performance_regression_percent", labels, percentChange)

		return true
	}

	return false
}
