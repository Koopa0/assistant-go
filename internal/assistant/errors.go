package assistant

import (
	"errors"
	"fmt"
)

// Error types for the assistant package
var (
	// ErrInvalidInput indicates invalid input was provided
	ErrInvalidInput = errors.New("invalid input")

	// ErrProcessingFailed indicates request processing failed
	ErrProcessingFailed = errors.New("processing failed")

	// ErrContextNotFound indicates conversation context was not found
	ErrContextNotFound = errors.New("context not found")

	// ErrProviderUnavailable indicates AI provider is unavailable
	ErrProviderUnavailable = errors.New("provider unavailable")

	// ErrToolNotFound indicates requested tool was not found
	ErrToolNotFound = errors.New("tool not found")

	// ErrToolExecutionFailed indicates tool execution failed
	ErrToolExecutionFailed = errors.New("tool execution failed")

	// ErrRateLimited indicates rate limit was exceeded
	ErrRateLimited = errors.New("rate limited")

	// ErrUnauthorized indicates unauthorized access
	ErrUnauthorized = errors.New("unauthorized")

	// ErrTimeout indicates operation timed out
	ErrTimeout = errors.New("timeout")
)

// AssistantError represents a domain-specific error
type AssistantError struct {
	Code    string
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error implements the error interface
func (e *AssistantError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
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

// NewAssistantError creates a new AssistantError
func NewAssistantError(code, message string, cause error) *AssistantError {
	return &AssistantError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context to the error
func (e *AssistantError) WithContext(key string, value interface{}) *AssistantError {
	e.Context[key] = value
	return e
}

// Error codes
const (
	CodeInvalidInput        = "INVALID_INPUT"
	CodeProcessingFailed    = "PROCESSING_FAILED"
	CodeContextNotFound     = "CONTEXT_NOT_FOUND"
	CodeProviderUnavailable = "PROVIDER_UNAVAILABLE"
	CodeToolNotFound        = "TOOL_NOT_FOUND"
	CodeToolExecutionFailed = "TOOL_EXECUTION_FAILED"
	CodeRateLimited         = "RATE_LIMITED"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeTimeout             = "TIMEOUT"
	CodeDatabaseError       = "DATABASE_ERROR"
	CodeConfigurationError  = "CONFIGURATION_ERROR"
	CodeValidationError     = "VALIDATION_ERROR"
)

// Predefined error constructors
func NewInvalidInputError(message string, cause error) *AssistantError {
	return NewAssistantError(CodeInvalidInput, message, cause)
}

func NewProcessingFailedError(message string, cause error) *AssistantError {
	return NewAssistantError(CodeProcessingFailed, message, cause)
}

func NewContextNotFoundError(contextID string) *AssistantError {
	return NewAssistantError(CodeContextNotFound, "conversation context not found", nil).
		WithContext("context_id", contextID)
}

func NewProviderUnavailableError(provider string, cause error) *AssistantError {
	return NewAssistantError(CodeProviderUnavailable, "AI provider unavailable", cause).
		WithContext("provider", provider)
}

func NewToolNotFoundError(toolName string) *AssistantError {
	return NewAssistantError(CodeToolNotFound, "tool not found", nil).
		WithContext("tool", toolName)
}

func NewToolExecutionFailedError(toolName string, cause error) *AssistantError {
	return NewAssistantError(CodeToolExecutionFailed, "tool execution failed", cause).
		WithContext("tool", toolName)
}

func NewRateLimitedError(limit int, window string) *AssistantError {
	return NewAssistantError(CodeRateLimited, "rate limit exceeded", nil).
		WithContext("limit", limit).
		WithContext("window", window)
}

func NewUnauthorizedError(message string) *AssistantError {
	return NewAssistantError(CodeUnauthorized, message, nil)
}

func NewTimeoutError(operation string, timeout string) *AssistantError {
	return NewAssistantError(CodeTimeout, "operation timed out", nil).
		WithContext("operation", operation).
		WithContext("timeout", timeout)
}

func NewDatabaseError(operation string, cause error) *AssistantError {
	return NewAssistantError(CodeDatabaseError, "database operation failed", cause).
		WithContext("operation", operation)
}

func NewConfigurationError(field string, cause error) *AssistantError {
	return NewAssistantError(CodeConfigurationError, "configuration error", cause).
		WithContext("field", field)
}

func NewValidationError(field string, message string) *AssistantError {
	return NewAssistantError(CodeValidationError, message, nil).
		WithContext("field", field)
}

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

// ErrorResponse represents an error response for API endpoints
type ErrorResponse struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ToErrorResponse converts an error to an ErrorResponse
func ToErrorResponse(err error) *ErrorResponse {
	if assistantErr := GetAssistantError(err); assistantErr != nil {
		return &ErrorResponse{
			Code:    assistantErr.Code,
			Message: assistantErr.Message,
			Details: assistantErr.Context,
		}
	}

	return &ErrorResponse{
		Code:    "INTERNAL_ERROR",
		Message: "An internal error occurred",
		Details: map[string]interface{}{
			"error": err.Error(),
		},
	}
}
