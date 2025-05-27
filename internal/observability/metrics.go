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
