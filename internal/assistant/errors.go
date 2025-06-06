// Package assistant provides domain-specific error handling for assistant operations
// following CLAUDE.md best practices with proper error hierarchy and context.
package assistant

import (
	"fmt"
	"time"

	configerrors "github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/errors"
)

// Assistant-specific error codes
const (
	// Processing errors
	CodeAssistantInitialization  = "ASSISTANT_INITIALIZATION"
	CodeAssistantProcessing      = "ASSISTANT_PROCESSING"
	CodeAssistantTimeout         = "ASSISTANT_TIMEOUT"
	CodeAssistantMemoryFull      = "ASSISTANT_MEMORY_FULL"
	CodeAssistantContextOverflow = "ASSISTANT_CONTEXT_OVERFLOW"

	// Input/Output errors
	CodeAssistantInvalidInput   = "ASSISTANT_INVALID_INPUT"
	CodeAssistantEmptyInput     = "ASSISTANT_EMPTY_INPUT"
	CodeAssistantOutputTooLarge = "ASSISTANT_OUTPUT_TOO_LARGE"

	// Integration errors
	CodeAssistantProviderError = "ASSISTANT_PROVIDER_ERROR"
	CodeAssistantToolError     = "ASSISTANT_TOOL_ERROR"
	CodeAssistantDatabaseError = "ASSISTANT_DATABASE_ERROR"
	CodeAssistantMemoryError   = "ASSISTANT_MEMORY_ERROR"

	// State errors
	CodeAssistantStateCorrupted = "ASSISTANT_STATE_CORRUPTED"
	CodeAssistantStateMismatch  = "ASSISTANT_STATE_MISMATCH"
)

// Processing Error Constructors

// NewAssistantInitializationError creates an initialization error
func NewAssistantInitializationError(component string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeAssistantInitialization, "assistant initialization failed", cause).
		WithComponent("assistant").
		WithContext("failed_component", component).
		WithUserMessage("Failed to initialize assistant. Please try again.").
		WithActions("Check configuration", "Verify dependencies", "Restart application").
		WithSeverity(errors.SeverityHigh)
}

// NewAssistantProcessingError creates a processing error
func NewAssistantProcessingError(stage string, cause error) *errors.AssistantError {
	return errors.NewBusinessError(CodeAssistantProcessing, "assistant processing failed", cause).
		WithComponent("assistant").
		WithOperation("process").
		WithContext("stage", stage).
		WithUserMessage("Failed to process your request. Please try again.").
		WithActions("Simplify request", "Check input format", "Try alternative approach").
		WithRetryable(true)
}

// NewAssistantTimeoutError creates a timeout error
func NewAssistantTimeoutError(operation string, timeout time.Duration) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeAssistantTimeout, "assistant operation timed out", nil).
		WithComponent("assistant").
		WithOperation(operation).
		WithContext("timeout", timeout.String()).
		WithDuration(timeout).
		WithUserMessage("Request took too long to process. Please try again.").
		WithActions("Simplify request", "Break into smaller parts", "Check system resources").
		WithRetryAfter(time.Second * 5)
}

// NewAssistantMemoryFullError creates a memory full error
func NewAssistantMemoryFullError(used, limit int64) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeAssistantMemoryFull, "assistant memory full", nil).
		WithComponent("assistant").
		WithOperation("memory_allocation").
		WithContext("used_bytes", used).
		WithContext("limit_bytes", limit).
		WithContext("usage_percent", float64(used)/float64(limit)*100).
		WithUserMessage("Assistant memory is full. Starting a new session.").
		WithActions("Start new conversation", "Clear old conversations", "Increase memory limit")
}

// NewAssistantContextOverflowError creates a context overflow error
func NewAssistantContextOverflowError(tokens, maxTokens int) *errors.AssistantError {
	return errors.NewValidationError(CodeAssistantContextOverflow, "context window exceeded", nil).
		WithComponent("assistant").
		WithOperation("context_management").
		WithContext("tokens", tokens).
		WithContext("max_tokens", maxTokens).
		WithContext("overflow", tokens-maxTokens).
		WithUserMessage("Conversation has become too long. Please start a new one.").
		WithActions("Start new conversation", "Summarize previous context", "Remove old messages")
}

// Input/Output Error Constructors

// NewAssistantInvalidInputError creates an invalid input error
func NewAssistantInvalidInputError(reason string, input interface{}) *errors.AssistantError {
	return errors.NewValidationError(CodeAssistantInvalidInput, fmt.Sprintf("invalid input: %s", reason), nil).
		WithComponent("assistant").
		WithOperation("validate_input").
		WithContext("reason", reason).
		WithContext("input_type", fmt.Sprintf("%T", input)).
		WithUserMessage(fmt.Sprintf("Invalid input: %s", reason)).
		WithActions("Check input format", "Review requirements", "Try different input")
}

// NewAssistantEmptyInputError creates an empty input error
func NewAssistantEmptyInputError() *errors.AssistantError {
	return errors.NewValidationError(CodeAssistantEmptyInput, "empty input provided", nil).
		WithComponent("assistant").
		WithOperation("validate_input").
		WithUserMessage("Please provide some input to process.").
		WithActions("Enter a message", "Ask a question", "Provide a command")
}

// NewAssistantOutputTooLargeError creates an output too large error
func NewAssistantOutputTooLargeError(size, maxSize int64) *errors.AssistantError {
	return errors.NewBusinessError(CodeAssistantOutputTooLarge, "assistant output too large", nil).
		WithComponent("assistant").
		WithOperation("generate_output").
		WithContext("size", size).
		WithContext("max_size", maxSize).
		WithContext("excess", size-maxSize).
		WithUserMessage("Response is too large. Truncating output.").
		WithActions("Request summary", "Ask for specific parts", "Break into smaller queries")
}

// Integration Error Constructors

// NewAssistantProviderError creates a provider integration error
func NewAssistantProviderError(provider string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeAssistantProviderError, "AI provider error", cause).
		WithComponent("assistant").
		WithOperation("provider_call").
		WithContext("provider", provider).
		WithUserMessage("AI service temporarily unavailable. Please try again.").
		WithActions("Retry request", "Try alternative provider", "Check service status").
		WithRetryAfter(time.Minute)
}

// NewAssistantToolError creates a tool integration error
func NewAssistantToolError(toolName string, cause error) *errors.AssistantError {
	return errors.NewBusinessError(CodeAssistantToolError, "tool execution error", cause).
		WithComponent("assistant").
		WithOperation("tool_execution").
		WithContext("tool", toolName).
		WithUserMessage("Tool execution failed. Please try again.").
		WithActions("Check tool availability", "Verify parameters", "Use alternative tool").
		WithRetryable(true)
}

// NewAssistantDatabaseError creates a database integration error
func NewAssistantDatabaseError(operation string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeAssistantDatabaseError, "database operation failed", cause).
		WithComponent("assistant").
		WithOperation(operation).
		WithUserMessage("Failed to save/load conversation data.").
		WithActions("Check database connection", "Retry operation", "Contact support").
		WithRetryable(true)
}

// NewAssistantMemoryError creates a memory system error
func NewAssistantMemoryError(operation string, cause error) *errors.AssistantError {
	return errors.NewBusinessError(CodeAssistantMemoryError, "memory system error", cause).
		WithComponent("assistant").
		WithOperation(operation).
		WithUserMessage("Failed to access conversation memory.").
		WithActions("Restart conversation", "Clear memory cache", "Check memory service")
}

// State Error Constructors

// NewAssistantStateCorruptedError creates a state corrupted error
func NewAssistantStateCorruptedError(stateID string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeAssistantStateCorrupted, "assistant state corrupted", cause).
		WithComponent("assistant").
		WithOperation("load_state").
		WithContext("state_id", stateID).
		WithUserMessage("Assistant state is corrupted. Starting fresh.").
		WithActions("Reset assistant", "Start new session", "Report issue").
		WithSeverity(errors.SeverityHigh)
}

// NewAssistantStateMismatchError creates a state mismatch error
func NewAssistantStateMismatchError(expected, actual string) *errors.AssistantError {
	return errors.NewBusinessError(CodeAssistantStateMismatch, "assistant state mismatch", nil).
		WithComponent("assistant").
		WithOperation("validate_state").
		WithContext("expected_state", expected).
		WithContext("actual_state", actual).
		WithUserMessage("Assistant state mismatch detected.").
		WithActions("Refresh state", "Restart conversation", "Clear cache")
}

// Legacy error constructors for backward compatibility

// NewConfigurationError creates a configuration error (deprecated)
func NewConfigurationError(field string, cause error) error {
	return configerrors.NewConfigMissingRequiredError(field, "assistant").
		WithComponent("assistant")
}

// NewInvalidInputError creates an invalid input error (deprecated)
func NewInvalidInputError(message string, cause error) error {
	return NewAssistantInvalidInputError(message, nil)
}

// NewProcessingFailedError creates a processing failed error (deprecated)
func NewProcessingFailedError(stage string, cause error) error {
	return NewAssistantProcessingError(stage, cause)
}

// NewTimeoutError creates a timeout error (deprecated)
func NewTimeoutError(operation string, timeout string) error {
	duration, _ := time.ParseDuration(timeout)
	return NewAssistantTimeoutError(operation, duration)
}

// NewToolNotFoundError creates a tool not found error (deprecated)
func NewToolNotFoundError(toolName string) error {
	return NewAssistantToolError(toolName, fmt.Errorf("tool not found: %s", toolName))
}

// NewDatabaseError creates a database error (deprecated)
func NewDatabaseError(operation string, cause error) error {
	return NewAssistantDatabaseError(operation, cause)
}

// Assistant-specific error helpers

// IsAssistantError checks if an error is assistant-related
func IsAssistantError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		return assistantErr.Component == "assistant"
	}
	return false
}

// IsRetryableAssistantError checks if an assistant error is retryable
func IsRetryableAssistantError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		return assistantErr.Retryable && assistantErr.Component == "assistant"
	}
	return false
}

// IsMemoryError checks if an error is related to memory constraints
func IsMemoryError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeAssistantMemoryFull, CodeAssistantContextOverflow,
			CodeAssistantMemoryError:
			return true
		}
	}
	return false
}

// GetRetryDelay extracts retry delay from assistant errors
func GetRetryDelay(err error) *time.Duration {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		return assistantErr.RetryAfter
	}
	return nil
}
