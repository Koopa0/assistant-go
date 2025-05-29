package events

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// LoggingHandler logs all events
type LoggingHandler struct {
	logger *slog.Logger
	config LoggingConfig
}

// LoggingConfig configures logging behavior
type LoggingConfig struct {
	LogLevel     LogLevel
	IncludeData  bool
	FilterTypes  []EventType
	ExcludeTypes []EventType
}

// LogLevel defines logging levels
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// MetricsHandler tracks event metrics
type MetricsHandler struct {
	metrics map[EventType]*EventTypeMetrics
	windows map[string]*MetricWindow
	mu      sync.RWMutex
}

// EventTypeMetrics tracks metrics for a specific event type
type EventTypeMetrics struct {
	Count       int64
	LastSeen    time.Time
	AverageSize int64
	ErrorCount  int64
	MinLatency  time.Duration
	MaxLatency  time.Duration
	AvgLatency  time.Duration
}

// MetricWindow represents a time window for metrics
type MetricWindow struct {
	Duration  time.Duration
	StartTime time.Time
	Events    []Event
	Metrics   EventTypeMetrics
}

// PersistenceHandler persists events to storage
type PersistenceHandler struct {
	storage  EventStorage
	config   PersistenceConfig
	buffer   []Event
	bufferMu sync.Mutex
}

// PersistenceConfig configures event persistence
type PersistenceConfig struct {
	Enabled       bool
	BatchSize     int
	FlushInterval time.Duration
	Retention     time.Duration
	Compress      bool
	EventTypes    []EventType
}

// EventStorage interface for event storage
type EventStorage interface {
	Store(ctx context.Context, events []Event) error
	Retrieve(ctx context.Context, filter EventFilter, limit int) ([]Event, error)
	Delete(ctx context.Context, olderThan time.Time) error
}

// PatternDetectionHandler detects event patterns
type PatternDetectionHandler struct {
	patterns  []EventPattern
	sequences map[string]*SequenceTracker
	mu        sync.RWMutex
}

// SequenceTracker tracks event sequences for pattern detection
type SequenceTracker struct {
	PatternID   string
	Events      []Event
	StartTime   time.Time
	LastEvent   time.Time
	CurrentStep int
	Completed   bool
}

// AlertingHandler generates alerts based on events
type AlertingHandler struct {
	rules    []AlertRule
	channels []AlertChannel
	history  []Alert
	mu       sync.RWMutex
}

// AlertRule defines when to generate alerts
type AlertRule struct {
	ID          string
	Name        string
	Description string
	EventTypes  []EventType
	Condition   AlertCondition
	Severity    AlertSeverity
	Throttle    time.Duration
	LastFired   time.Time
	Enabled     bool
}

// AlertCondition defines alert conditions
type AlertCondition struct {
	Type        ConditionType
	Threshold   float64
	TimeWindow  time.Duration
	Aggregation AggregationType
	Predicate   func(Event) bool
}

// ConditionType defines types of alert conditions
type ConditionType string

const (
	ConditionCount     ConditionType = "count"
	ConditionRate      ConditionType = "rate"
	ConditionThreshold ConditionType = "threshold"
	ConditionAnomaly   ConditionType = "anomaly"
	ConditionPattern   ConditionType = "pattern"
)

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityError    AlertSeverity = "error"
	SeverityCritical AlertSeverity = "critical"
)

// Alert represents a generated alert
type Alert struct {
	ID           string
	RuleID       string
	Title        string
	Description  string
	Severity     AlertSeverity
	Events       []Event
	Timestamp    time.Time
	Acknowledged bool
	Resolved     bool
	ResolvedAt   *time.Time
}

// AlertChannel defines how alerts are delivered
type AlertChannel interface {
	Send(ctx context.Context, alert Alert) error
	GetType() ChannelType
}

// ChannelType defines types of alert channels
type ChannelType string

const (
	ChannelLog     ChannelType = "log"
	ChannelEmail   ChannelType = "email"
	ChannelSlack   ChannelType = "slack"
	ChannelWebhook ChannelType = "webhook"
)

// CorrelationHandler correlates related events
type CorrelationHandler struct {
	correlations map[string]*EventCorrelation
	rules        []CorrelationRule
	mu           sync.RWMutex
}

// EventCorrelation represents a correlation between events
type EventCorrelation struct {
	ID         string
	Events     []Event
	Type       CorrelationType
	Confidence float64
	StartTime  time.Time
	EndTime    *time.Time
	Metadata   map[string]interface{}
}

// CorrelationType defines types of correlations
type CorrelationType string

const (
	CorrelationCausal   CorrelationType = "causal"
	CorrelationTemporal CorrelationType = "temporal"
	CorrelationSpatial  CorrelationType = "spatial"
	CorrelationSemantic CorrelationType = "semantic"
)

// CorrelationRule defines how to correlate events
type CorrelationRule struct {
	ID         string
	Name       string
	EventTypes []EventType
	TimeWindow time.Duration
	Condition  func([]Event) bool
	Type       CorrelationType
	Confidence float64
}

// NewLoggingHandler creates a new logging handler
func NewLoggingHandler(logger *slog.Logger, config LoggingConfig) *LoggingHandler {
	return &LoggingHandler{
		logger: logger,
		config: config,
	}
}

// Handle logs events
func (lh *LoggingHandler) Handle(ctx context.Context, event Event) error {
	// Check if this event type should be logged
	if len(lh.config.FilterTypes) > 0 {
		found := false
		for _, eventType := range lh.config.FilterTypes {
			if event.Type == eventType {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	// Check if this event type should be excluded
	for _, eventType := range lh.config.ExcludeTypes {
		if event.Type == eventType {
			return nil
		}
	}

	attrs := []slog.Attr{
		slog.String("event_id", event.ID),
		slog.String("type", string(event.Type)),
		slog.String("source", event.Source),
		slog.Time("timestamp", event.Timestamp),
	}

	if event.Target != "" {
		attrs = append(attrs, slog.String("target", event.Target))
	}

	if event.Priority != "" {
		attrs = append(attrs, slog.String("priority", string(event.Priority)))
	}

	if lh.config.IncludeData && event.Data != nil {
		attrs = append(attrs, slog.Any("data", event.Data))
	}

	if len(event.Metadata) > 0 {
		attrs = append(attrs, slog.Any("metadata", event.Metadata))
	}

	switch lh.config.LogLevel {
	case LogLevelDebug:
		lh.logger.LogAttrs(ctx, slog.LevelDebug, "Event received", attrs...)
	case LogLevelInfo:
		lh.logger.LogAttrs(ctx, slog.LevelInfo, "Event received", attrs...)
	case LogLevelWarn:
		lh.logger.LogAttrs(ctx, slog.LevelWarn, "Event received", attrs...)
	case LogLevelError:
		lh.logger.LogAttrs(ctx, slog.LevelError, "Event received", attrs...)
	default:
		lh.logger.LogAttrs(ctx, slog.LevelInfo, "Event received", attrs...)
	}

	return nil
}

// CanHandle checks if this handler can handle the event type
func (lh *LoggingHandler) CanHandle(eventType EventType) bool {
	return true // Logging handler can handle all event types
}

// Priority returns the handler priority
func (lh *LoggingHandler) Priority() int {
	return 1000 // Low priority, run last
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{
		metrics: make(map[EventType]*EventTypeMetrics),
		windows: make(map[string]*MetricWindow),
	}
}

// Handle updates metrics for events
func (mh *MetricsHandler) Handle(ctx context.Context, event Event) error {
	mh.mu.Lock()
	defer mh.mu.Unlock()

	// Update metrics for this event type
	if mh.metrics[event.Type] == nil {
		mh.metrics[event.Type] = &EventTypeMetrics{
			MinLatency: time.Hour, // Initialize with high value
		}
	}

	metrics := mh.metrics[event.Type]
	metrics.Count++
	metrics.LastSeen = event.Timestamp

	// Calculate latency if possible
	latency := time.Since(event.Timestamp)
	if latency < metrics.MinLatency {
		metrics.MinLatency = latency
	}
	if latency > metrics.MaxLatency {
		metrics.MaxLatency = latency
	}
	metrics.AvgLatency = (metrics.AvgLatency + latency) / 2

	return nil
}

// CanHandle checks if this handler can handle the event type
func (mh *MetricsHandler) CanHandle(eventType EventType) bool {
	return true // Metrics handler can handle all event types
}

// Priority returns the handler priority
func (mh *MetricsHandler) Priority() int {
	return 100 // High priority, run early
}

// GetMetrics returns metrics for an event type
func (mh *MetricsHandler) GetMetrics(eventType EventType) *EventTypeMetrics {
	mh.mu.RLock()
	defer mh.mu.RUnlock()

	if metrics, exists := mh.metrics[eventType]; exists {
		// Return a copy to avoid race conditions
		copy := *metrics
		return &copy
	}

	return nil
}

// GetAllMetrics returns all event type metrics
func (mh *MetricsHandler) GetAllMetrics() map[EventType]*EventTypeMetrics {
	mh.mu.RLock()
	defer mh.mu.RUnlock()

	result := make(map[EventType]*EventTypeMetrics)
	for eventType, metrics := range mh.metrics {
		copy := *metrics
		result[eventType] = &copy
	}

	return result
}

// NewPersistenceHandler creates a new persistence handler
func NewPersistenceHandler(storage EventStorage, config PersistenceConfig) *PersistenceHandler {
	handler := &PersistenceHandler{
		storage: storage,
		config:  config,
		buffer:  make([]Event, 0, config.BatchSize),
	}

	// Start background flusher if enabled
	if config.Enabled && config.FlushInterval > 0 {
		go handler.backgroundFlusher()
	}

	return handler
}

// Handle persists events to storage
func (ph *PersistenceHandler) Handle(ctx context.Context, event Event) error {
	if !ph.config.Enabled {
		return nil
	}

	// Check if this event type should be persisted
	if len(ph.config.EventTypes) > 0 {
		found := false
		for _, eventType := range ph.config.EventTypes {
			if event.Type == eventType {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	ph.bufferMu.Lock()
	defer ph.bufferMu.Unlock()

	ph.buffer = append(ph.buffer, event)

	// Flush if buffer is full
	if len(ph.buffer) >= ph.config.BatchSize {
		return ph.flush(ctx)
	}

	return nil
}

// CanHandle checks if this handler can handle the event type
func (ph *PersistenceHandler) CanHandle(eventType EventType) bool {
	return ph.config.Enabled
}

// Priority returns the handler priority
func (ph *PersistenceHandler) Priority() int {
	return 500 // Medium priority
}

// flush flushes buffered events to storage
func (ph *PersistenceHandler) flush(ctx context.Context) error {
	if len(ph.buffer) == 0 {
		return nil
	}

	events := make([]Event, len(ph.buffer))
	copy(events, ph.buffer)
	ph.buffer = ph.buffer[:0] // Clear buffer

	return ph.storage.Store(ctx, events)
}

// backgroundFlusher flushes events periodically
func (ph *PersistenceHandler) backgroundFlusher() {
	ticker := time.NewTicker(ph.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ph.bufferMu.Lock()
			if len(ph.buffer) > 0 {
				ph.flush(context.Background())
			}
			ph.bufferMu.Unlock()
		}
	}
}

// NewPatternDetectionHandler creates a new pattern detection handler
func NewPatternDetectionHandler(patterns []EventPattern) *PatternDetectionHandler {
	return &PatternDetectionHandler{
		patterns:  patterns,
		sequences: make(map[string]*SequenceTracker),
	}
}

// Handle detects event patterns
func (pdh *PatternDetectionHandler) Handle(ctx context.Context, event Event) error {
	pdh.mu.Lock()
	defer pdh.mu.Unlock()

	// Check each pattern
	for _, pattern := range pdh.patterns {
		pdh.checkPattern(ctx, pattern, event)
	}

	return nil
}

// CanHandle checks if this handler can handle the event type
func (pdh *PatternDetectionHandler) CanHandle(eventType EventType) bool {
	pdh.mu.RLock()
	defer pdh.mu.RUnlock()

	for _, pattern := range pdh.patterns {
		for _, patternEventType := range pattern.Events {
			if eventType == patternEventType {
				return true
			}
		}
	}

	return false
}

// Priority returns the handler priority
func (pdh *PatternDetectionHandler) Priority() int {
	return 200 // High priority for pattern detection
}

// checkPattern checks if an event matches a pattern
func (pdh *PatternDetectionHandler) checkPattern(ctx context.Context, pattern EventPattern, event Event) {
	// Get or create sequence tracker
	trackerKey := fmt.Sprintf("%s_%s", pattern.ID, event.Correlation.CorrelationID)
	tracker, exists := pdh.sequences[trackerKey]

	if !exists {
		tracker = &SequenceTracker{
			PatternID: pattern.ID,
			Events:    make([]Event, 0),
			StartTime: event.Timestamp,
		}
		pdh.sequences[trackerKey] = tracker
	}

	// Check if event matches current step
	if tracker.CurrentStep < len(pattern.Sequence) {
		step := pattern.Sequence[tracker.CurrentStep]
		if event.Type == step.EventType {
			if step.Condition == nil || step.Condition(event) {
				tracker.Events = append(tracker.Events, event)
				tracker.LastEvent = event.Timestamp
				tracker.CurrentStep++

				// Check if pattern is complete
				if tracker.CurrentStep >= len(pattern.Sequence) {
					tracker.Completed = true

					// Execute pattern action
					if pattern.Action != nil {
						go pattern.Action.Execute(ctx, tracker.Events)
					}

					// Clean up completed tracker
					delete(pdh.sequences, trackerKey)
				}
			}
		}
	}

	// Clean up expired trackers
	now := time.Now()
	for key, tracker := range pdh.sequences {
		if now.Sub(tracker.StartTime) > pattern.TimeWindow {
			delete(pdh.sequences, key)
		}
	}
}

// NewAlertingHandler creates a new alerting handler
func NewAlertingHandler(rules []AlertRule, channels []AlertChannel) *AlertingHandler {
	return &AlertingHandler{
		rules:    rules,
		channels: channels,
		history:  make([]Alert, 0),
	}
}

// Handle generates alerts based on events
func (ah *AlertingHandler) Handle(ctx context.Context, event Event) error {
	ah.mu.Lock()
	defer ah.mu.Unlock()

	// Check each alert rule
	for i := range ah.rules {
		rule := &ah.rules[i]
		if !rule.Enabled {
			continue
		}

		// Check if rule applies to this event type
		ruleApplies := false
		for _, eventType := range rule.EventTypes {
			if event.Type == eventType {
				ruleApplies = true
				break
			}
		}

		if !ruleApplies {
			continue
		}

		// Check throttling
		if rule.Throttle > 0 && time.Since(rule.LastFired) < rule.Throttle {
			continue
		}

		// Check condition
		if ah.checkAlertCondition(rule.Condition, event) {
			alert := Alert{
				ID:          fmt.Sprintf("alert_%d", time.Now().UnixNano()),
				RuleID:      rule.ID,
				Title:       rule.Name,
				Description: rule.Description,
				Severity:    rule.Severity,
				Events:      []Event{event},
				Timestamp:   time.Now(),
			}

			ah.history = append(ah.history, alert)
			rule.LastFired = time.Now()

			// Send alert through channels
			for _, channel := range ah.channels {
				go channel.Send(ctx, alert)
			}
		}
	}

	return nil
}

// CanHandle checks if this handler can handle the event type
func (ah *AlertingHandler) CanHandle(eventType EventType) bool {
	ah.mu.RLock()
	defer ah.mu.RUnlock()

	for _, rule := range ah.rules {
		for _, ruleEventType := range rule.EventTypes {
			if eventType == ruleEventType {
				return true
			}
		}
	}

	return false
}

// Priority returns the handler priority
func (ah *AlertingHandler) Priority() int {
	return 300 // High priority for alerting
}

// checkAlertCondition checks if an alert condition is met
func (ah *AlertingHandler) checkAlertCondition(condition AlertCondition, event Event) bool {
	// Apply custom predicate if present
	if condition.Predicate != nil {
		return condition.Predicate(event)
	}

	// For now, return true for simple implementation
	// In a full implementation, this would check various condition types
	return true
}

// NewCorrelationHandler creates a new correlation handler
func NewCorrelationHandler(rules []CorrelationRule) *CorrelationHandler {
	return &CorrelationHandler{
		correlations: make(map[string]*EventCorrelation),
		rules:        rules,
	}
}

// Handle correlates events
func (ch *CorrelationHandler) Handle(ctx context.Context, event Event) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Check each correlation rule
	for _, rule := range ch.rules {
		ch.checkCorrelation(rule, event)
	}

	return nil
}

// CanHandle checks if this handler can handle the event type
func (ch *CorrelationHandler) CanHandle(eventType EventType) bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	for _, rule := range ch.rules {
		for _, ruleEventType := range rule.EventTypes {
			if eventType == ruleEventType {
				return true
			}
		}
	}

	return false
}

// Priority returns the handler priority
func (ch *CorrelationHandler) Priority() int {
	return 400 // Medium-high priority for correlation
}

// checkCorrelation checks if events can be correlated
func (ch *CorrelationHandler) checkCorrelation(rule CorrelationRule, event Event) {
	// Get or create correlation
	correlationID := event.Correlation.CorrelationID
	if correlationID == "" {
		correlationID = fmt.Sprintf("corr_%d", time.Now().UnixNano())
	}

	correlation, exists := ch.correlations[correlationID]
	if !exists {
		correlation = &EventCorrelation{
			ID:        correlationID,
			Events:    make([]Event, 0),
			Type:      rule.Type,
			StartTime: event.Timestamp,
			Metadata:  make(map[string]interface{}),
		}
		ch.correlations[correlationID] = correlation
	}

	correlation.Events = append(correlation.Events, event)

	// Check if correlation condition is met
	if rule.Condition != nil && rule.Condition(correlation.Events) {
		correlation.Confidence = rule.Confidence
		now := time.Now()
		correlation.EndTime = &now
	}

	// Clean up old correlations
	cutoff := time.Now().Add(-rule.TimeWindow)
	for id, corr := range ch.correlations {
		if corr.StartTime.Before(cutoff) {
			delete(ch.correlations, id)
		}
	}
}
