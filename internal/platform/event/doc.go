// Package event provides a comprehensive system for event-driven architecture within the application.
// It includes an EventBus for publishing and subscribing to events, various predefined event types,
// and interfaces for handlers, subscribers, and middleware.
//
// The system supports features like asynchronous processing, dead-letter queues,
// metrics, event streams, and pattern detection. Interfaces like EventPublisher
// and EventBusService are provided for decoupled interaction with the event bus.
package event
