package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ConfigEvent represents a configuration change event
type ConfigEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	Source      string                 `json:"source"`
	Field       string                 `json:"field,omitempty"`
	OldValue    interface{}            `json:"old_value,omitempty"`
	NewValue    interface{}            `json:"new_value,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Environment string                 `json:"environment"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ConfigMetrics holds OpenTelemetry metrics for configuration observability
type ConfigMetrics struct {
	ConfigLoads            metric.Int64Counter
	ConfigLoadDuration     metric.Float64Histogram
	ConfigValidations      metric.Int64Counter
	ConfigValidationErrors metric.Int64Counter
	ConfigChanges          metric.Int64Counter
	ConfigReloads          metric.Int64Counter
	SecurityEvents         metric.Int64Counter
}

// ConfigObserver provides configuration observability and change tracking
type ConfigObserver struct {
	events      []ConfigEvent
	eventsMutex sync.RWMutex
	maxEvents   int
	listeners   []func(ConfigEvent)
	metrics     *ConfigMetrics
	currentHash string
	hashMutex   sync.RWMutex
}

// NewConfigObserver creates a new configuration observer
func NewConfigObserver(maxEvents int) (*ConfigObserver, error) {
	meter := otel.Meter("assistant.config")

	metrics, err := newConfigMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create config metrics: %w", err)
	}

	return &ConfigObserver{
		events:    make([]ConfigEvent, 0, maxEvents),
		maxEvents: maxEvents,
		listeners: make([]func(ConfigEvent), 0),
		metrics:   metrics,
	}, nil
}

// newConfigMetrics creates OpenTelemetry metrics for configuration observability
func newConfigMetrics(meter metric.Meter) (*ConfigMetrics, error) {
	configLoads, err := meter.Int64Counter(
		"config_loads_total",
		metric.WithDescription("Total number of configuration loads"),
	)
	if err != nil {
		return nil, err
	}

	configLoadDuration, err := meter.Float64Histogram(
		"config_load_duration_seconds",
		metric.WithDescription("Configuration load duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	configValidations, err := meter.Int64Counter(
		"config_validations_total",
		metric.WithDescription("Total number of configuration validations"),
	)
	if err != nil {
		return nil, err
	}

	configValidationErrors, err := meter.Int64Counter(
		"config_validation_errors_total",
		metric.WithDescription("Total number of configuration validation errors"),
	)
	if err != nil {
		return nil, err
	}

	configChanges, err := meter.Int64Counter(
		"config_changes_total",
		metric.WithDescription("Total number of configuration changes"),
	)
	if err != nil {
		return nil, err
	}

	configReloads, err := meter.Int64Counter(
		"config_reloads_total",
		metric.WithDescription("Total number of configuration reloads"),
	)
	if err != nil {
		return nil, err
	}

	securityEvents, err := meter.Int64Counter(
		"config_security_events_total",
		metric.WithDescription("Total number of configuration security events"),
	)
	if err != nil {
		return nil, err
	}

	return &ConfigMetrics{
		ConfigLoads:            configLoads,
		ConfigLoadDuration:     configLoadDuration,
		ConfigValidations:      configValidations,
		ConfigValidationErrors: configValidationErrors,
		ConfigChanges:          configChanges,
		ConfigReloads:          configReloads,
		SecurityEvents:         securityEvents,
	}, nil
}

// RecordLoad records a configuration load event
func (co *ConfigObserver) RecordLoad(ctx context.Context, source string, duration time.Duration, success bool) {
	// Record metrics
	attrs := []attribute.KeyValue{
		attribute.String("source", source),
		attribute.Bool("success", success),
	}

	co.metrics.ConfigLoads.Add(ctx, 1, metric.WithAttributes(attrs...))
	co.metrics.ConfigLoadDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	// Record event
	event := ConfigEvent{
		Timestamp: time.Now(),
		EventType: "config_load",
		Source:    source,
		Metadata: map[string]interface{}{
			"duration_ms": duration.Milliseconds(),
			"success":     success,
		},
	}

	co.recordEvent(event)
	slog.Info("Configuration load recorded",
		"source", source,
		"duration", duration,
		"success", success)
}

// RecordValidation records a configuration validation event
func (co *ConfigObserver) RecordValidation(ctx context.Context, errors []ValidationError) {
	hasErrors := len(errors) > 0

	// Record metrics
	attrs := []attribute.KeyValue{
		attribute.Bool("has_errors", hasErrors),
		attribute.Int("error_count", len(errors)),
	}

	co.metrics.ConfigValidations.Add(ctx, 1, metric.WithAttributes(attrs...))
	if hasErrors {
		co.metrics.ConfigValidationErrors.Add(ctx, int64(len(errors)), metric.WithAttributes(attrs...))
	}

	// Record event
	event := ConfigEvent{
		Timestamp: time.Now(),
		EventType: "config_validation",
		Source:    "validator",
		Metadata: map[string]interface{}{
			"error_count": len(errors),
			"has_errors":  hasErrors,
		},
	}

	if hasErrors {
		errorDetails := make([]map[string]interface{}, len(errors))
		for i, err := range errors {
			errorDetails[i] = map[string]interface{}{
				"field":   err.Field,
				"message": err.Message,
				"code":    err.Code,
			}
		}
		event.Metadata["errors"] = errorDetails
	}

	co.recordEvent(event)

	if hasErrors {
		slog.Warn("Configuration validation completed with errors",
			"error_count", len(errors))
	} else {
		slog.Info("Configuration validation successful")
	}
}

// RecordChange records a configuration change event
func (co *ConfigObserver) RecordChange(ctx context.Context, field string, oldValue, newValue interface{}, userID string) {
	// Record metrics
	attrs := []attribute.KeyValue{
		attribute.String("field", field),
	}
	if userID != "" {
		attrs = append(attrs, attribute.String("user_id", userID))
	}

	co.metrics.ConfigChanges.Add(ctx, 1, metric.WithAttributes(attrs...))

	// Record event
	event := ConfigEvent{
		Timestamp: time.Now(),
		EventType: "config_change",
		Source:    "user",
		Field:     field,
		OldValue:  oldValue,
		NewValue:  newValue,
		UserID:    userID,
		Metadata: map[string]interface{}{
			"change_type": determineChangeType(oldValue, newValue),
		},
	}

	co.recordEvent(event)
	slog.Info("Configuration change recorded",
		"field", field,
		"user_id", userID,
		"change_type", determineChangeType(oldValue, newValue))
}

// RecordReload records a configuration reload event
func (co *ConfigObserver) RecordReload(ctx context.Context, trigger string, success bool) {
	// Record metrics
	attrs := []attribute.KeyValue{
		attribute.String("trigger", trigger),
		attribute.Bool("success", success),
	}

	co.metrics.ConfigReloads.Add(ctx, 1, metric.WithAttributes(attrs...))

	// Record event
	event := ConfigEvent{
		Timestamp: time.Now(),
		EventType: "config_reload",
		Source:    trigger,
		Metadata: map[string]interface{}{
			"success": success,
		},
	}

	co.recordEvent(event)
	slog.Info("Configuration reload recorded",
		"trigger", trigger,
		"success", success)
}

// RecordSecurityEvent records a configuration security event
func (co *ConfigObserver) RecordSecurityEvent(ctx context.Context, eventType, description string, severity string) {
	// Record metrics
	attrs := []attribute.KeyValue{
		attribute.String("event_type", eventType),
		attribute.String("severity", severity),
	}

	co.metrics.SecurityEvents.Add(ctx, 1, metric.WithAttributes(attrs...))

	// Record event
	event := ConfigEvent{
		Timestamp: time.Now(),
		EventType: "security_event",
		Source:    "security_monitor",
		Metadata: map[string]interface{}{
			"security_event_type": eventType,
			"description":         description,
			"severity":            severity,
		},
	}

	co.recordEvent(event)

	// Log with appropriate level based on severity
	switch severity {
	case "critical", "high":
		slog.Error("Configuration security event",
			"event_type", eventType,
			"description", description,
			"severity", severity)
	case "medium":
		slog.Warn("Configuration security event",
			"event_type", eventType,
			"description", description,
			"severity", severity)
	default:
		slog.Info("Configuration security event",
			"event_type", eventType,
			"description", description,
			"severity", severity)
	}
}

// TrackConfigChanges compares two configurations and records changes
func (co *ConfigObserver) TrackConfigChanges(ctx context.Context, oldConfig, newConfig *Config, userID string) {
	changes := co.detectChanges(oldConfig, newConfig)

	for _, change := range changes {
		co.RecordChange(ctx, change.Field, change.OldValue, change.NewValue, userID)

		// Check for security-relevant changes
		co.checkSecurityImplications(ctx, change)
	}
}

// detectChanges detects changes between two configurations
func (co *ConfigObserver) detectChanges(oldConfig, newConfig *Config) []ConfigChange {
	var changes []ConfigChange

	// Compare configurations using reflection or specific field comparisons
	// For brevity, implementing key fields that are security-sensitive

	if oldConfig.Security.JWTSecret != newConfig.Security.JWTSecret {
		changes = append(changes, ConfigChange{
			Field:    "Security.JWTSecret",
			OldValue: "[REDACTED]",
			NewValue: "[REDACTED]",
		})
	}

	if oldConfig.Security.RateLimitRPS != newConfig.Security.RateLimitRPS {
		changes = append(changes, ConfigChange{
			Field:    "Security.RateLimitRPS",
			OldValue: oldConfig.Security.RateLimitRPS,
			NewValue: newConfig.Security.RateLimitRPS,
		})
	}

	if oldConfig.Database.URL != newConfig.Database.URL {
		changes = append(changes, ConfigChange{
			Field:    "Database.URL",
			OldValue: maskSensitiveValue(oldConfig.Database.URL),
			NewValue: maskSensitiveValue(newConfig.Database.URL),
		})
	}

	if oldConfig.AI.DefaultProvider != newConfig.AI.DefaultProvider {
		changes = append(changes, ConfigChange{
			Field:    "AI.DefaultProvider",
			OldValue: oldConfig.AI.DefaultProvider,
			NewValue: newConfig.AI.DefaultProvider,
		})
	}

	// Add more field comparisons as needed

	return changes
}

// ConfigChange represents a detected configuration change
type ConfigChange struct {
	Field    string
	OldValue interface{}
	NewValue interface{}
}

// checkSecurityImplications checks for security implications of configuration changes
func (co *ConfigObserver) checkSecurityImplications(ctx context.Context, change ConfigChange) {
	switch change.Field {
	case "Security.JWTSecret":
		co.RecordSecurityEvent(ctx, "jwt_secret_change", "JWT secret was changed", "high")
	case "Security.RateLimitRPS":
		if newVal, ok := change.NewValue.(int); ok && newVal > 1000 {
			co.RecordSecurityEvent(ctx, "rate_limit_increase", "Rate limit increased significantly", "medium")
		}
	case "Database.URL":
		co.RecordSecurityEvent(ctx, "database_url_change", "Database connection URL was changed", "high")
	case "Server.EnableTLS":
		if newVal, ok := change.NewValue.(bool); ok && !newVal {
			co.RecordSecurityEvent(ctx, "tls_disabled", "TLS was disabled", "critical")
		}
	}
}

// AddListener adds an event listener
func (co *ConfigObserver) AddListener(listener func(ConfigEvent)) {
	co.eventsMutex.Lock()
	defer co.eventsMutex.Unlock()
	co.listeners = append(co.listeners, listener)
}

// GetEvents returns recent configuration events
func (co *ConfigObserver) GetEvents(limit int) []ConfigEvent {
	co.eventsMutex.RLock()
	defer co.eventsMutex.RUnlock()

	if limit <= 0 || limit > len(co.events) {
		limit = len(co.events)
	}

	// Return most recent events
	start := len(co.events) - limit
	if start < 0 {
		start = 0
	}

	events := make([]ConfigEvent, limit)
	copy(events, co.events[start:])
	return events
}

// GetEventsByType returns events filtered by type
func (co *ConfigObserver) GetEventsByType(eventType string, limit int) []ConfigEvent {
	co.eventsMutex.RLock()
	defer co.eventsMutex.RUnlock()

	var filtered []ConfigEvent
	for i := len(co.events) - 1; i >= 0 && len(filtered) < limit; i-- {
		if co.events[i].EventType == eventType {
			filtered = append(filtered, co.events[i])
		}
	}

	return filtered
}

// recordEvent adds an event to the history
func (co *ConfigObserver) recordEvent(event ConfigEvent) {
	co.eventsMutex.Lock()
	defer co.eventsMutex.Unlock()

	// Add current environment to event
	event.Environment = getCurrentEnvironment()

	co.events = append(co.events, event)

	// Maintain maximum event count
	if len(co.events) > co.maxEvents {
		copy(co.events, co.events[len(co.events)-co.maxEvents:])
		co.events = co.events[:co.maxEvents]
	}

	// Notify listeners
	for _, listener := range co.listeners {
		go listener(event) // Non-blocking notification
	}
}

// Helper functions

// determineChangeType determines the type of configuration change
func determineChangeType(oldValue, newValue interface{}) string {
	if oldValue == nil && newValue != nil {
		return "added"
	}
	if oldValue != nil && newValue == nil {
		return "removed"
	}
	return "modified"
}

// maskSensitiveValue masks sensitive values in configuration
func maskSensitiveValue(value string) string {
	if len(value) <= 8 {
		return "[REDACTED]"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

// getCurrentEnvironment gets the current environment name
func getCurrentEnvironment() string {
	// This would typically come from environment variables or config
	return "unknown" // Placeholder
}

// ExportEvents exports configuration events as JSON
func (co *ConfigObserver) ExportEvents() ([]byte, error) {
	co.eventsMutex.RLock()
	defer co.eventsMutex.RUnlock()

	return json.MarshalIndent(co.events, "", "  ")
}

// ClearEvents clears the event history
func (co *ConfigObserver) ClearEvents() {
	co.eventsMutex.Lock()
	defer co.eventsMutex.Unlock()

	co.events = co.events[:0]
	slog.Info("Configuration event history cleared")
}
