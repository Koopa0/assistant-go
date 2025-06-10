package event

import (
	"context"
)

// EventPublisher defines the interface for components that only need to publish events.
type EventPublisher interface {
	Publish(ctx context.Context, event Event) error
}

// EventBusService defines a comprehensive interface for managing and interacting
// with the event bus.
type EventBusService interface {
	EventPublisher // Embeds Publish method

	Subscribe(subscriber Subscriber) error
	Unsubscribe(subscriberID string) error
	AddHandler(handler EventHandler) error
	AddMiddleware(middleware Middleware) error // Added as it's a common management task

	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	GetMetrics() EventMetricsSnapshot // For components that might need to monitor the bus
	// GetDeadLetterEvents() []DeadLetterEvent // Optional, less common for typical service consumers
	// CreateStream(id, name string, filter EventFilter) (*EventStream, error) // Optional, if stream creation is managed externally
}

// Note: Types Event, Subscriber, EventHandler, Middleware, EventMetricsSnapshot
// are expected to be defined in bus.go or other files within the event package.
