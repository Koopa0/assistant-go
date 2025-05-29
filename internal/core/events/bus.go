package events

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"
)

// EventBus is the central event management system
type EventBus struct {
	subscribers     map[EventType][]Subscriber
	eventHandlers   map[EventType][]EventHandler
	middleware      []Middleware
	metrics         *EventMetrics
	deadLetterQueue *DeadLetterQueue
	config          EventBusConfig
	logger          *slog.Logger
	mu              sync.RWMutex
	running         bool
	stopCh          chan struct{}
}

// Event represents a system event
type Event struct {
	ID          string
	Type        EventType
	Source      string
	Target      string
	Timestamp   time.Time
	Data        interface{}
	Metadata    map[string]interface{}
	Priority    Priority
	TTL         time.Duration
	Retry       RetryConfig
	Correlation CorrelationInfo
}

// EventType defines the type of event
type EventType string

const (
	// System events
	EventSystemStartup     EventType = "system.startup"
	EventSystemShutdown    EventType = "system.shutdown"
	EventSystemError       EventType = "system.error"
	EventSystemHealthCheck EventType = "system.health_check"

	// Context events
	EventContextUpdate   EventType = "context.update"
	EventContextChange   EventType = "context.change"
	EventWorkspaceChange EventType = "workspace.change"
	EventFileActivity    EventType = "file.activity"
	EventProjectSwitch   EventType = "project.switch"

	// Agent events
	EventAgentRegistered    EventType = "agent.registered"
	EventAgentUnregistered  EventType = "agent.unregistered"
	EventAgentStatusChange  EventType = "agent.status_change"
	EventAgentTaskStart     EventType = "agent.task_start"
	EventAgentTaskComplete  EventType = "agent.task_complete"
	EventAgentTaskFailed    EventType = "agent.task_failed"
	EventAgentCollaboration EventType = "agent.collaboration"
	EventAgentLearning      EventType = "agent.learning"

	// User events
	EventUserRequest    EventType = "user.request"
	EventUserFeedback   EventType = "user.feedback"
	EventUserPreference EventType = "user.preference"
	EventUserSession    EventType = "user.session"

	// Tool events
	EventToolExecution    EventType = "tool.execution"
	EventToolRegistration EventType = "tool.registration"
	EventToolError        EventType = "tool.error"

	// Learning events
	EventLearningSession   EventType = "learning.session"
	EventKnowledgeUpdate   EventType = "knowledge.update"
	EventPatternDiscovered EventType = "pattern.discovered"
	EventInsightGenerated  EventType = "insight.generated"

	// Performance events
	EventPerformanceMetric  EventType = "performance.metric"
	EventResourceUsage      EventType = "resource.usage"
	EventBottleneckDetected EventType = "bottleneck.detected"

	// Custom events
	EventCustom EventType = "custom"
)

// Priority defines event priority levels
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// RetryConfig configures event retry behavior
type RetryConfig struct {
	Enabled    bool
	MaxRetries int
	Backoff    BackoffStrategy
	Conditions []RetryCondition
}

// BackoffStrategy defines retry backoff strategies
type BackoffStrategy string

const (
	BackoffFixed       BackoffStrategy = "fixed"
	BackoffExponential BackoffStrategy = "exponential"
	BackoffLinear      BackoffStrategy = "linear"
	BackoffRandom      BackoffStrategy = "random"
)

// RetryCondition defines when to retry an event
type RetryCondition struct {
	ErrorType string
	Predicate func(error) bool
}

// CorrelationInfo tracks event relationships
type CorrelationInfo struct {
	CorrelationID string
	CausationID   string
	SessionID     string
	UserID        string
	TraceID       string
}

// Subscriber represents an event subscriber
type Subscriber interface {
	OnEvent(ctx context.Context, event Event) error
	GetEventTypes() []EventType
	GetID() string
}

// EventHandler handles specific event types
type EventHandler interface {
	Handle(ctx context.Context, event Event) error
	CanHandle(eventType EventType) bool
	Priority() int
}

// Middleware provides event processing middleware
type Middleware interface {
	Process(ctx context.Context, event Event, next MiddlewareFunc) error
	Order() int
}

// MiddlewareFunc represents a middleware function
type MiddlewareFunc func(ctx context.Context, event Event) error

// EventBusConfig configures the event bus
type EventBusConfig struct {
	BufferSize       int
	MaxConcurrency   int
	EnableMetrics    bool
	EnableDeadLetter bool
	DeadLetterTTL    time.Duration
	RetryEnabled     bool
	AsyncProcessing  bool
}

// EventMetrics tracks event bus performance
type EventMetrics struct {
	TotalEvents     int64
	ProcessedEvents int64
	FailedEvents    int64
	AverageLatency  time.Duration
	ThroughputRate  float64
	ErrorRate       float64
	DeadLetterCount int64
	LastUpdated     time.Time
	mu              sync.RWMutex
}

// DeadLetterQueue handles failed events
type DeadLetterQueue struct {
	events     []DeadLetterEvent
	maxSize    int
	ttl        time.Duration
	processors []DeadLetterProcessor
	mu         sync.RWMutex
}

// DeadLetterEvent represents a failed event
type DeadLetterEvent struct {
	OriginalEvent Event
	FailureReason string
	FailureTime   time.Time
	RetryCount    int
	LastError     error
}

// DeadLetterProcessor processes dead letter events
type DeadLetterProcessor interface {
	Process(ctx context.Context, deadEvent DeadLetterEvent) error
	CanProcess(deadEvent DeadLetterEvent) bool
}

// EventFilter filters events based on criteria
type EventFilter struct {
	EventTypes []EventType
	Sources    []string
	Targets    []string
	Priority   Priority
	Metadata   map[string]interface{}
	Predicate  func(Event) bool
}

// EventStream represents a stream of events
type EventStream struct {
	ID          string
	Name        string
	Filter      EventFilter
	Subscribers []StreamSubscriber
	BufferSize  int
	events      chan Event
	mu          sync.RWMutex
}

// StreamSubscriber subscribes to event streams
type StreamSubscriber interface {
	OnStreamEvent(ctx context.Context, event Event) error
	GetStreamID() string
}

// EventPattern represents an event pattern for detection
type EventPattern struct {
	ID          string
	Name        string
	Description string
	Events      []EventType
	Sequence    []SequenceStep
	TimeWindow  time.Duration
	Condition   func([]Event) bool
	Action      PatternAction
}

// SequenceStep defines a step in an event sequence
type SequenceStep struct {
	EventType EventType
	Condition func(Event) bool
	Timeout   time.Duration
	Optional  bool
}

// PatternAction defines action to take when pattern is detected
type PatternAction interface {
	Execute(ctx context.Context, events []Event) error
}

// EventAggregator aggregates events for analysis
type EventAggregator struct {
	aggregators map[string]Aggregator
	windows     map[string]TimeWindow
	results     map[string]AggregationResult
	mu          sync.RWMutex
}

// Aggregator aggregates events
type Aggregator interface {
	Aggregate(events []Event) AggregationResult
	GetType() AggregationType
}

// AggregationType defines types of aggregation
type AggregationType string

const (
	AggregationCount   AggregationType = "count"
	AggregationSum     AggregationType = "sum"
	AggregationAverage AggregationType = "average"
	AggregationMin     AggregationType = "min"
	AggregationMax     AggregationType = "max"
	AggregationCustom  AggregationType = "custom"
)

// TimeWindow defines a time window for aggregation
type TimeWindow struct {
	Duration time.Duration
	Sliding  bool
	Overlap  time.Duration
}

// AggregationResult represents aggregation results
type AggregationResult struct {
	Type      AggregationType
	Value     interface{}
	Count     int
	TimeRange TimeRange
	Metadata  map[string]interface{}
}

// TimeRange represents a time range
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// NewEventBus creates a new event bus
func NewEventBus(config EventBusConfig, logger *slog.Logger) (*EventBus, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	deadLetterQueue := &DeadLetterQueue{
		events:     make([]DeadLetterEvent, 0),
		maxSize:    1000,
		ttl:        config.DeadLetterTTL,
		processors: make([]DeadLetterProcessor, 0),
	}

	metrics := &EventMetrics{
		LastUpdated: time.Now(),
	}

	bus := &EventBus{
		subscribers:     make(map[EventType][]Subscriber),
		eventHandlers:   make(map[EventType][]EventHandler),
		middleware:      make([]Middleware, 0),
		metrics:         metrics,
		deadLetterQueue: deadLetterQueue,
		config:          config,
		logger:          logger,
		stopCh:          make(chan struct{}),
	}

	return bus, nil
}

// Start starts the event bus
func (eb *EventBus) Start(ctx context.Context) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.running {
		return fmt.Errorf("event bus is already running")
	}

	eb.running = true
	eb.logger.Info("Event bus started")

	return nil
}

// Stop stops the event bus
func (eb *EventBus) Stop(ctx context.Context) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if !eb.running {
		return fmt.Errorf("event bus is not running")
	}

	close(eb.stopCh)
	eb.running = false
	eb.logger.Info("Event bus stopped")

	return nil
}

// Subscribe adds a subscriber for specific event types
func (eb *EventBus) Subscribe(subscriber Subscriber) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eventTypes := subscriber.GetEventTypes()
	for _, eventType := range eventTypes {
		eb.subscribers[eventType] = append(eb.subscribers[eventType], subscriber)
	}

	eb.logger.Info("Subscriber registered",
		slog.String("subscriber_id", subscriber.GetID()),
		slog.Any("event_types", eventTypes))

	return nil
}

// Unsubscribe removes a subscriber
func (eb *EventBus) Unsubscribe(subscriberID string) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for eventType, subscribers := range eb.subscribers {
		for i, subscriber := range subscribers {
			if subscriber.GetID() == subscriberID {
				eb.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
				break
			}
		}
	}

	eb.logger.Info("Subscriber unregistered", slog.String("subscriber_id", subscriberID))
	return nil
}

// AddHandler adds an event handler
func (eb *EventBus) AddHandler(handler EventHandler) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Determine which event types this handler can handle
	for _, eventType := range getAllEventTypes() {
		if handler.CanHandle(eventType) {
			eb.eventHandlers[eventType] = append(eb.eventHandlers[eventType], handler)
		}
	}

	eb.logger.Info("Event handler added", slog.String("handler", reflect.TypeOf(handler).String()))
	return nil
}

// AddMiddleware adds middleware to the processing chain
func (eb *EventBus) AddMiddleware(middleware Middleware) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.middleware = append(eb.middleware, middleware)

	// Sort middleware by order
	for i := 0; i < len(eb.middleware); i++ {
		for j := i + 1; j < len(eb.middleware); j++ {
			if eb.middleware[i].Order() > eb.middleware[j].Order() {
				eb.middleware[i], eb.middleware[j] = eb.middleware[j], eb.middleware[i]
			}
		}
	}

	eb.logger.Info("Middleware added", slog.String("middleware", reflect.TypeOf(middleware).String()))
	return nil
}

// Publish publishes an event to the bus
func (eb *EventBus) Publish(ctx context.Context, event Event) error {
	if !eb.running {
		return fmt.Errorf("event bus is not running")
	}

	// Set event ID if not provided
	if event.ID == "" {
		event.ID = fmt.Sprintf("event_%d", time.Now().UnixNano())
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	eb.logger.Debug("Publishing event",
		slog.String("event_id", event.ID),
		slog.String("type", string(event.Type)),
		slog.String("source", event.Source))

	// Update metrics
	eb.updateMetrics(func(m *EventMetrics) {
		m.TotalEvents++
	})

	if eb.config.AsyncProcessing {
		go eb.processEvent(ctx, event)
	} else {
		return eb.processEvent(ctx, event)
	}

	return nil
}

// processEvent processes an event through the middleware chain and subscribers
func (eb *EventBus) processEvent(ctx context.Context, event Event) error {
	startTime := time.Now()

	// Process through middleware chain
	err := eb.processMiddleware(ctx, event, 0)
	if err != nil {
		eb.handleEventError(event, err)
		return err
	}

	// Process through event handlers
	err = eb.processHandlers(ctx, event)
	if err != nil {
		eb.handleEventError(event, err)
		return err
	}

	// Notify subscribers
	err = eb.notifySubscribers(ctx, event)
	if err != nil {
		eb.handleEventError(event, err)
		return err
	}

	// Update metrics
	duration := time.Since(startTime)
	eb.updateMetrics(func(m *EventMetrics) {
		m.ProcessedEvents++
		m.AverageLatency = (m.AverageLatency + duration) / 2
	})

	eb.logger.Debug("Event processed successfully",
		slog.String("event_id", event.ID),
		slog.Duration("duration", duration))

	return nil
}

// processMiddleware processes event through middleware chain
func (eb *EventBus) processMiddleware(ctx context.Context, event Event, index int) error {
	if index >= len(eb.middleware) {
		return nil
	}

	middleware := eb.middleware[index]
	return middleware.Process(ctx, event, func(ctx context.Context, event Event) error {
		return eb.processMiddleware(ctx, event, index+1)
	})
}

// processHandlers processes event through handlers
func (eb *EventBus) processHandlers(ctx context.Context, event Event) error {
	eb.mu.RLock()
	handlers := eb.eventHandlers[event.Type]
	eb.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler.Handle(ctx, event); err != nil {
			eb.logger.Warn("Event handler failed",
				slog.String("event_id", event.ID),
				slog.String("handler", reflect.TypeOf(handler).String()),
				slog.Any("error", err))
			// Continue processing other handlers
		}
	}

	return nil
}

// notifySubscribers notifies all subscribers of the event
func (eb *EventBus) notifySubscribers(ctx context.Context, event Event) error {
	eb.mu.RLock()
	subscribers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	for _, subscriber := range subscribers {
		if err := subscriber.OnEvent(ctx, event); err != nil {
			eb.logger.Warn("Subscriber notification failed",
				slog.String("event_id", event.ID),
				slog.String("subscriber_id", subscriber.GetID()),
				slog.Any("error", err))
			// Continue notifying other subscribers
		}
	}

	return nil
}

// handleEventError handles event processing errors
func (eb *EventBus) handleEventError(event Event, err error) {
	eb.updateMetrics(func(m *EventMetrics) {
		m.FailedEvents++
		m.ErrorRate = float64(m.FailedEvents) / float64(m.TotalEvents)
	})

	// Add to dead letter queue if enabled
	if eb.config.EnableDeadLetter {
		deadEvent := DeadLetterEvent{
			OriginalEvent: event,
			FailureReason: err.Error(),
			FailureTime:   time.Now(),
			LastError:     err,
		}

		eb.deadLetterQueue.mu.Lock()
		eb.deadLetterQueue.events = append(eb.deadLetterQueue.events, deadEvent)
		if len(eb.deadLetterQueue.events) > eb.deadLetterQueue.maxSize {
			eb.deadLetterQueue.events = eb.deadLetterQueue.events[1:]
		}
		eb.deadLetterQueue.mu.Unlock()

		eb.updateMetrics(func(m *EventMetrics) {
			m.DeadLetterCount++
		})
	}

	eb.logger.Error("Event processing failed",
		slog.String("event_id", event.ID),
		slog.String("type", string(event.Type)),
		slog.Any("error", err))
}

// updateMetrics updates event metrics
func (eb *EventBus) updateMetrics(updater func(*EventMetrics)) {
	if !eb.config.EnableMetrics {
		return
	}

	eb.metrics.mu.Lock()
	defer eb.metrics.mu.Unlock()

	updater(eb.metrics)
	eb.metrics.LastUpdated = time.Now()
}

// GetMetrics returns current event metrics
func (eb *EventBus) GetMetrics() EventMetrics {
	eb.metrics.mu.RLock()
	defer eb.metrics.mu.RUnlock()

	return *eb.metrics
}

// GetDeadLetterEvents returns dead letter events
func (eb *EventBus) GetDeadLetterEvents() []DeadLetterEvent {
	eb.deadLetterQueue.mu.RLock()
	defer eb.deadLetterQueue.mu.RUnlock()

	events := make([]DeadLetterEvent, len(eb.deadLetterQueue.events))
	copy(events, eb.deadLetterQueue.events)
	return events
}

// CreateStream creates a new event stream
func (eb *EventBus) CreateStream(id, name string, filter EventFilter) (*EventStream, error) {
	stream := &EventStream{
		ID:          id,
		Name:        name,
		Filter:      filter,
		Subscribers: make([]StreamSubscriber, 0),
		BufferSize:  100,
		events:      make(chan Event, 100),
	}

	// Subscribe to relevant events
	streamSubscriber := &streamSubscriberAdapter{
		stream: stream,
		filter: filter,
	}

	eb.Subscribe(streamSubscriber)

	eb.logger.Info("Event stream created",
		slog.String("stream_id", id),
		slog.String("name", name))

	return stream, nil
}

// streamSubscriberAdapter adapts a stream to be a subscriber
type streamSubscriberAdapter struct {
	stream *EventStream
	filter EventFilter
}

func (ssa *streamSubscriberAdapter) OnEvent(ctx context.Context, event Event) error {
	if ssa.matchesFilter(event) {
		select {
		case ssa.stream.events <- event:
		default:
			// Buffer full, drop event
		}
	}
	return nil
}

func (ssa *streamSubscriberAdapter) GetEventTypes() []EventType {
	return ssa.filter.EventTypes
}

func (ssa *streamSubscriberAdapter) GetID() string {
	return fmt.Sprintf("stream_%s", ssa.stream.ID)
}

func (ssa *streamSubscriberAdapter) matchesFilter(event Event) bool {
	// Check event types
	if len(ssa.filter.EventTypes) > 0 {
		found := false
		for _, eventType := range ssa.filter.EventTypes {
			if event.Type == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check sources
	if len(ssa.filter.Sources) > 0 {
		found := false
		for _, source := range ssa.filter.Sources {
			if event.Source == source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check priority
	if ssa.filter.Priority != "" && event.Priority != ssa.filter.Priority {
		return false
	}

	// Check custom predicate
	if ssa.filter.Predicate != nil {
		return ssa.filter.Predicate(event)
	}

	return true
}

// Helper function to get all event types
func getAllEventTypes() []EventType {
	return []EventType{
		EventSystemStartup, EventSystemShutdown, EventSystemError, EventSystemHealthCheck,
		EventContextUpdate, EventContextChange, EventWorkspaceChange, EventFileActivity, EventProjectSwitch,
		EventAgentRegistered, EventAgentUnregistered, EventAgentStatusChange, EventAgentTaskStart,
		EventAgentTaskComplete, EventAgentTaskFailed, EventAgentCollaboration, EventAgentLearning,
		EventUserRequest, EventUserFeedback, EventUserPreference, EventUserSession,
		EventToolExecution, EventToolRegistration, EventToolError,
		EventLearningSession, EventKnowledgeUpdate, EventPatternDiscovered, EventInsightGenerated,
		EventPerformanceMetric, EventResourceUsage, EventBottleneckDetected,
		EventCustom,
	}
}
