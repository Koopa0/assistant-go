// Package errors provides the base error types and utilities for the Assistant application
// following CLAUDE.md best practices with proper error hierarchy and structured information.
//
// This package contains only the core error types and helper functions. Domain-specific
// errors are defined in their respective feature packages to follow the "package by feature"
// principle.
package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "LOW"      // Non-critical errors that don't affect functionality
	SeverityMedium   ErrorSeverity = "MEDIUM"   // Errors that affect some functionality
	SeverityHigh     ErrorSeverity = "HIGH"     // Critical errors that affect core functionality
	SeverityCritical ErrorSeverity = "CRITICAL" // System-wide failures
)

// ErrorCategory represents the category of error for better classification
type ErrorCategory string

const (
	// Infrastructure Layer Errors
	CategoryInfrastructure ErrorCategory = "INFRASTRUCTURE"
	CategoryDatabase       ErrorCategory = "DATABASE"
	CategoryNetwork        ErrorCategory = "NETWORK"
	CategoryFileSystem     ErrorCategory = "FILESYSTEM"
	CategorySecurity       ErrorCategory = "SECURITY"

	// Business Logic Layer Errors
	CategoryBusiness     ErrorCategory = "BUSINESS"
	CategoryProcessing   ErrorCategory = "PROCESSING"
	CategoryTool         ErrorCategory = "TOOL"
	CategoryProvider     ErrorCategory = "PROVIDER"
	CategoryConversation ErrorCategory = "CONVERSATION"

	// Validation Layer Errors
	CategoryValidation     ErrorCategory = "VALIDATION"
	CategoryConfiguration  ErrorCategory = "CONFIGURATION"
	CategoryAuthentication ErrorCategory = "AUTHENTICATION"
	CategoryAuthorization  ErrorCategory = "AUTHORIZATION"
)

// AssistantError represents a comprehensive domain-specific error with structured information
// for debugging, monitoring, and operational support.
type AssistantError struct {
	// Core error information
	Code     string        `json:"code"`     // Machine-readable error code
	Message  string        `json:"message"`  // Human-readable error message
	Cause    error         `json:"-"`        // Underlying error cause
	Category ErrorCategory `json:"category"` // Error category for classification
	Severity ErrorSeverity `json:"severity"` // Error severity level

	// Context and debugging information
	Context       map[string]interface{} `json:"context,omitempty"`        // Additional context data
	Operation     string                 `json:"operation,omitempty"`      // Operation that caused the error
	Component     string                 `json:"component,omitempty"`      // Component/service that generated the error
	StackTrace    string                 `json:"stack_trace,omitempty"`    // Stack trace for debugging
	CorrelationID string                 `json:"correlation_id,omitempty"` // Request correlation ID

	// Temporal information
	Timestamp time.Time     `json:"timestamp"`          // When the error occurred
	Duration  time.Duration `json:"duration,omitempty"` // Operation duration before error

	// Actionable information for operations teams
	UserMessage string             `json:"user_message,omitempty"` // User-friendly error message
	Actions     []string           `json:"actions,omitempty"`      // Suggested remediation actions
	Metrics     map[string]float64 `json:"metrics,omitempty"`      // Related metrics for analysis
	Retryable   bool               `json:"retryable"`              // Whether the operation can be retried
	RetryAfter  *time.Duration     `json:"retry_after,omitempty"`  // Suggested retry delay
}

// Error implements the error interface with comprehensive formatting
func (e *AssistantError) Error() string {
	var parts []string

	// Add category and severity for context
	parts = append(parts, fmt.Sprintf("[%s:%s]", e.Category, e.Severity))

	// Add component if available
	if e.Component != "" {
		parts = append(parts, fmt.Sprintf("component=%s", e.Component))
	}

	// Add operation if available
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation=%s", e.Operation))
	}

	// Add core error information
	parts = append(parts, fmt.Sprintf("%s: %s", e.Code, e.Message))

	// Add cause if available
	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("cause: %v", e.Cause))
	}

	return strings.Join(parts, " | ")
}

// Unwrap returns the underlying error
func (e *AssistantError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target
func (e *AssistantError) Is(target error) bool {
	if t, ok := target.(*AssistantError); ok {
		return e.Code == t.Code
	}
	return errors.Is(e.Cause, target)
}

// NewAssistantError creates a new AssistantError with default values
func NewAssistantError(code, message string, cause error) *AssistantError {
	return &AssistantError{
		Code:      code,
		Message:   message,
		Cause:     cause,
		Category:  CategoryBusiness, // Default category
		Severity:  SeverityMedium,   // Default severity
		Context:   make(map[string]interface{}),
		Timestamp: time.Now(),
		Retryable: false, // Default to non-retryable
	}
}

// NewInfrastructureError creates an infrastructure-level error
func NewInfrastructureError(code, message string, cause error) *AssistantError {
	err := NewAssistantError(code, message, cause)
	err.Category = CategoryInfrastructure
	err.Severity = SeverityHigh
	err.Retryable = true // Infrastructure errors are often retryable
	captureStackTrace(err)
	return err
}

// NewBusinessError creates a business logic error
func NewBusinessError(code, message string, cause error) *AssistantError {
	err := NewAssistantError(code, message, cause)
	err.Category = CategoryBusiness
	err.Severity = SeverityMedium
	captureStackTrace(err)
	return err
}

// NewValidationError creates a validation error
func NewValidationError(code, message string, cause error) *AssistantError {
	err := NewAssistantError(code, message, cause)
	err.Category = CategoryValidation
	err.Severity = SeverityLow
	err.Retryable = false // Validation errors typically aren't retryable
	captureStackTrace(err)
	return err
}

// captureStackTrace captures the current stack trace for debugging
func captureStackTrace(err *AssistantError) {
	buf := make([]byte, 2048)
	n := runtime.Stack(buf, false)
	err.StackTrace = string(buf[:n])
}

// Builder methods for AssistantError

// WithContext adds context to the error
func (e *AssistantError) WithContext(key string, value interface{}) *AssistantError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithOperation sets the operation that caused the error
func (e *AssistantError) WithOperation(operation string) *AssistantError {
	e.Operation = operation
	return e
}

// WithComponent sets the component that generated the error
func (e *AssistantError) WithComponent(component string) *AssistantError {
	e.Component = component
	return e
}

// WithSeverity sets the error severity
func (e *AssistantError) WithSeverity(severity ErrorSeverity) *AssistantError {
	e.Severity = severity
	return e
}

// WithCategory sets the error category
func (e *AssistantError) WithCategory(category ErrorCategory) *AssistantError {
	e.Category = category
	return e
}

// WithUserMessage sets a user-friendly error message
func (e *AssistantError) WithUserMessage(message string) *AssistantError {
	e.UserMessage = message
	return e
}

// WithActions adds suggested remediation actions
func (e *AssistantError) WithActions(actions ...string) *AssistantError {
	e.Actions = append(e.Actions, actions...)
	return e
}

// WithMetric adds a metric for analysis
func (e *AssistantError) WithMetric(name string, value float64) *AssistantError {
	if e.Metrics == nil {
		e.Metrics = make(map[string]float64)
	}
	e.Metrics[name] = value
	return e
}

// WithRetryable marks the error as retryable
func (e *AssistantError) WithRetryable(retryable bool) *AssistantError {
	e.Retryable = retryable
	return e
}

// WithRetryAfter sets the suggested retry delay
func (e *AssistantError) WithRetryAfter(delay time.Duration) *AssistantError {
	e.RetryAfter = &delay
	e.Retryable = true
	return e
}

// WithCorrelationID sets the correlation ID for request tracking
func (e *AssistantError) WithCorrelationID(id string) *AssistantError {
	e.CorrelationID = id
	return e
}

// WithDuration sets the operation duration before error
func (e *AssistantError) WithDuration(duration time.Duration) *AssistantError {
	e.Duration = duration
	return e
}

// Helper functions

// IsAssistantError checks if an error is an AssistantError
func IsAssistantError(err error) bool {
	var assistantErr *AssistantError
	return errors.As(err, &assistantErr)
}

// GetAssistantError extracts an AssistantError from an error chain
func GetAssistantError(err error) *AssistantError {
	var assistantErr *AssistantError
	if errors.As(err, &assistantErr) {
		return assistantErr
	}
	return nil
}

// WrapError wraps an error with additional context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// ErrorResponse represents a comprehensive error response for API endpoints
// with structured information for debugging, monitoring, and user experience.
type ErrorResponse struct {
	// Core error information
	Code     string        `json:"code"`     // Machine-readable error code
	Message  string        `json:"message"`  // Human-readable error message
	Category ErrorCategory `json:"category"` // Error category
	Severity ErrorSeverity `json:"severity"` // Error severity level

	// Context and debugging
	Operation     string                 `json:"operation,omitempty"`      // Operation that failed
	Component     string                 `json:"component,omitempty"`      // Component that generated error
	Context       map[string]interface{} `json:"context,omitempty"`        // Additional context
	CorrelationID string                 `json:"correlation_id,omitempty"` // Request correlation ID

	// Temporal information
	Timestamp string `json:"timestamp"`          // ISO 8601 timestamp
	Duration  string `json:"duration,omitempty"` // Operation duration

	// User and operational guidance
	UserMessage string   `json:"user_message,omitempty"` // User-friendly message
	Actions     []string `json:"actions,omitempty"`      // Suggested actions
	Retryable   bool     `json:"retryable"`              // Whether retryable
	RetryAfter  string   `json:"retry_after,omitempty"`  // Retry delay (ISO 8601 duration)

	// Debugging information (only in development)
	StackTrace string             `json:"stack_trace,omitempty"` // Stack trace
	Metrics    map[string]float64 `json:"metrics,omitempty"`     // Related metrics
}

// ToErrorResponse converts an error to a comprehensive ErrorResponse
func ToErrorResponse(err error) *ErrorResponse {
	if assistantErr := GetAssistantError(err); assistantErr != nil {
		return &ErrorResponse{
			Code:          assistantErr.Code,
			Message:       assistantErr.Message,
			Category:      assistantErr.Category,
			Severity:      assistantErr.Severity,
			Operation:     assistantErr.Operation,
			Component:     assistantErr.Component,
			Context:       assistantErr.Context,
			CorrelationID: assistantErr.CorrelationID,
			Timestamp:     assistantErr.Timestamp.Format(time.RFC3339),
			Duration:      formatDuration(assistantErr.Duration),
			UserMessage:   assistantErr.UserMessage,
			Actions:       assistantErr.Actions,
			Retryable:     assistantErr.Retryable,
			RetryAfter:    formatRetryAfter(assistantErr.RetryAfter),
			StackTrace:    assistantErr.StackTrace,
			Metrics:       assistantErr.Metrics,
		}
	}

	// Handle standard errors
	return &ErrorResponse{
		Code:      "INTERNAL_ERROR",
		Message:   "An internal error occurred",
		Category:  CategoryInfrastructure,
		Severity:  SeverityCritical,
		Timestamp: time.Now().Format(time.RFC3339),
		Context: map[string]interface{}{
			"error": err.Error(),
		},
		UserMessage: "We encountered an unexpected error. Please try again later.",
		Actions:     []string{"Try again later", "Contact support if the problem persists"},
		Retryable:   true,
	}
}

// formatDuration formats a duration for API response
func formatDuration(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return d.String()
}

// formatRetryAfter formats retry delay for API response
func formatRetryAfter(d *time.Duration) string {
	if d == nil {
		return ""
	}
	return d.String()
}

// GetErrorSummary provides a summary of error information for monitoring
func GetErrorSummary(err error) map[string]interface{} {
	summary := map[string]interface{}{
		"error_occurred": true,
		"timestamp":      time.Now().Format(time.RFC3339),
	}

	if assistantErr := GetAssistantError(err); assistantErr != nil {
		summary["code"] = assistantErr.Code
		summary["category"] = assistantErr.Category
		summary["severity"] = assistantErr.Severity
		summary["component"] = assistantErr.Component
		summary["operation"] = assistantErr.Operation
		summary["retryable"] = assistantErr.Retryable

		if assistantErr.Duration > 0 {
			summary["duration_ms"] = assistantErr.Duration.Milliseconds()
		}

		// Add metrics if available
		for k, v := range assistantErr.Metrics {
			summary["metric_"+k] = v
		}
	} else {
		summary["code"] = "UNKNOWN_ERROR"
		summary["category"] = CategoryInfrastructure
		summary["severity"] = SeverityHigh
		summary["message"] = err.Error()
	}

	return summary
}
