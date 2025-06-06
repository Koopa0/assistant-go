// Package conversation provides domain-specific error handling for conversation operations
// following CLAUDE.md best practices with proper error hierarchy and context.
package conversation

import (
	"fmt"
	"time"

	"github.com/koopa0/assistant-go/internal/errors"
)

// Conversation-specific error codes
const (
	// Conversation lifecycle errors
	CodeConversationCreation     = "CONVERSATION_CREATION"
	CodeConversationNotFound     = "CONVERSATION_NOT_FOUND"
	CodeConversationArchived     = "CONVERSATION_ARCHIVED"
	CodeConversationExpired      = "CONVERSATION_EXPIRED"
	CodeConversationLocked       = "CONVERSATION_LOCKED"
	CodeConversationInvalidState = "CONVERSATION_INVALID_STATE"
	CodeConversationUpdate       = "CONVERSATION_UPDATE"
	CodeConversationDeletion     = "CONVERSATION_DELETION"

	// Message errors
	CodeMessageCreation    = "MESSAGE_CREATION"
	CodeMessageNotFound    = "MESSAGE_NOT_FOUND"
	CodeMessageUpdate      = "MESSAGE_UPDATE"
	CodeMessageTooLong     = "MESSAGE_TOO_LONG"
	CodeMessageRateLimit   = "MESSAGE_RATE_LIMIT"
	CodeMessageDuplication = "MESSAGE_DUPLICATION"
	CodeMessageOrdering    = "MESSAGE_ORDERING"

	// Context errors
	CodeContextNotFound          = "CONTEXT_NOT_FOUND"
	CodeContextExpired           = "CONTEXT_EXPIRED"
	CodeContextCreation          = "CONTEXT_CREATION"
	CodeContextSerialization     = "CONTEXT_SERIALIZATION"
	CodeContextCorrupted         = "CONTEXT_CORRUPTED"
	CodeContextSizeLimitExceeded = "CONTEXT_SIZE_LIMIT_EXCEEDED"

	// Turn management errors
	CodeTurnCreation   = "TURN_CREATION"
	CodeTurnCompletion = "TURN_COMPLETION"
	CodeTurnTimeout    = "TURN_TIMEOUT"
	CodeTurnAborted    = "TURN_ABORTED"
	CodeTurnRateLimit  = "TURN_RATE_LIMIT"

	// Storage errors
	CodeConversationStorageFailure = "CONVERSATION_STORAGE_FAILURE"
	CodeConversationLoadFailure    = "CONVERSATION_LOAD_FAILURE"
	CodeConversationSyncFailure    = "CONVERSATION_SYNC_FAILURE"
)

// Conversation Lifecycle Error Constructors

// NewConversationCreationError creates a conversation creation error
func NewConversationCreationError(userID string, cause error) *errors.AssistantError {
	return errors.NewBusinessError(CodeConversationCreation, "failed to create conversation", cause).
		WithComponent("conversation").
		WithOperation("create").
		WithContext("user_id", userID).
		WithUserMessage("Failed to start a new conversation. Please try again.").
		WithActions("Check user permissions", "Verify storage availability", "Retry creation").
		WithRetryable(true)
}

// NewConversationNotFoundError creates a conversation not found error
func NewConversationNotFoundError(conversationID string) *errors.AssistantError {
	return errors.NewBusinessError(CodeConversationNotFound, "conversation not found", nil).
		WithComponent("conversation").
		WithOperation("retrieve").
		WithContext("conversation_id", conversationID).
		WithUserMessage("The conversation was not found. It may have been deleted or expired.").
		WithActions("Start a new conversation", "Check conversation ID", "List available conversations")
}

// NewConversationArchivedError creates a conversation archived error
func NewConversationArchivedError(conversationID string, archivedAt time.Time) *errors.AssistantError {
	return errors.NewBusinessError(CodeConversationArchived, "conversation is archived", nil).
		WithComponent("conversation").
		WithOperation("access").
		WithContext("conversation_id", conversationID).
		WithContext("archived_at", archivedAt.Format(time.RFC3339)).
		WithUserMessage("This conversation has been archived and is read-only.").
		WithActions("Start a new conversation", "Request conversation restoration", "View archived conversation")
}

// NewConversationExpiredError creates a conversation expired error
func NewConversationExpiredError(conversationID string, expiresAt time.Time) *errors.AssistantError {
	return errors.NewBusinessError(CodeConversationExpired, "conversation has expired", nil).
		WithComponent("conversation").
		WithOperation("access").
		WithContext("conversation_id", conversationID).
		WithContext("expired_at", expiresAt.Format(time.RFC3339)).
		WithUserMessage("This conversation has expired and is no longer available.").
		WithActions("Start a new conversation", "Check conversation retention policy")
}

// NewConversationLockedError creates a conversation locked error
func NewConversationLockedError(conversationID string, lockedBy string, lockedUntil *time.Time) *errors.AssistantError {
	err := errors.NewBusinessError(CodeConversationLocked, "conversation is locked", nil).
		WithComponent("conversation").
		WithOperation("access").
		WithContext("conversation_id", conversationID).
		WithContext("locked_by", lockedBy).
		WithUserMessage("This conversation is temporarily locked by another process.").
		WithActions("Wait and retry", "Check active processes", "Contact support if persists")

	if lockedUntil != nil {
		err.WithContext("locked_until", lockedUntil.Format(time.RFC3339)).
			WithRetryAfter(time.Until(*lockedUntil))
	}

	return err
}

// NewConversationInvalidStateError creates an invalid conversation state error
func NewConversationInvalidStateError(conversationID, currentState, expectedState string) *errors.AssistantError {
	return errors.NewBusinessError(CodeConversationInvalidState, "conversation in invalid state", nil).
		WithComponent("conversation").
		WithOperation("state_transition").
		WithContext("conversation_id", conversationID).
		WithContext("current_state", currentState).
		WithContext("expected_state", expectedState).
		WithUserMessage("The conversation is not in the expected state for this operation.").
		WithActions("Check conversation state", "Complete pending operations", "Reset conversation state")
}

// Message Error Constructors

// NewMessageCreationError creates a message creation error
func NewMessageCreationError(conversationID string, role string, cause error) *errors.AssistantError {
	return errors.NewBusinessError(CodeMessageCreation, "failed to create message", cause).
		WithComponent("conversation").
		WithOperation("create_message").
		WithContext("conversation_id", conversationID).
		WithContext("role", role).
		WithUserMessage("Failed to send message. Please try again.").
		WithActions("Check message content", "Verify conversation state", "Retry sending").
		WithRetryable(true)
}

// NewMessageNotFoundError creates a message not found error
func NewMessageNotFoundError(messageID, conversationID string) *errors.AssistantError {
	return errors.NewBusinessError(CodeMessageNotFound, "message not found", nil).
		WithComponent("conversation").
		WithOperation("retrieve_message").
		WithContext("message_id", messageID).
		WithContext("conversation_id", conversationID).
		WithUserMessage("The requested message was not found.").
		WithActions("Check message ID", "List conversation messages", "Verify conversation access")
}

// NewMessageTooLongError creates a message too long error
func NewMessageTooLongError(length, maxLength int) *errors.AssistantError {
	return errors.NewValidationError(CodeMessageTooLong, "message exceeds maximum length", nil).
		WithComponent("conversation").
		WithOperation("validate_message").
		WithContext("length", length).
		WithContext("max_length", maxLength).
		WithContext("excess_chars", length-maxLength).
		WithUserMessage(fmt.Sprintf("Message is too long (%d characters, max %d).", length, maxLength)).
		WithActions("Shorten message", "Split into multiple messages", "Remove unnecessary content")
}

// NewMessageRateLimitError creates a message rate limit error
func NewMessageRateLimitError(conversationID string, limit int, window time.Duration, retryAfter time.Duration) *errors.AssistantError {
	return errors.NewValidationError(CodeMessageRateLimit, "message rate limit exceeded", nil).
		WithComponent("conversation").
		WithOperation("send_message").
		WithContext("conversation_id", conversationID).
		WithContext("limit", limit).
		WithContext("window", window.String()).
		WithUserMessage("Sending messages too quickly. Please wait before sending another.").
		WithActions("Wait before retry", "Reduce message frequency", "Batch messages").
		WithRetryAfter(retryAfter)
}

// Context Error Constructors

// NewContextNotFoundError creates a context not found error
func NewContextNotFoundError(contextID string) *errors.AssistantError {
	return errors.NewBusinessError(CodeContextNotFound, "conversation context not found", nil).
		WithComponent("conversation").
		WithOperation("retrieve_context").
		WithContext("context_id", contextID).
		WithUserMessage("The conversation context was not found. Please start a new conversation.").
		WithActions("Create new context", "Check context ID", "Verify context persistence")
}

// NewContextExpiredError creates a context expired error
func NewContextExpiredError(contextID string, expiredAt time.Time) *errors.AssistantError {
	return errors.NewBusinessError(CodeContextExpired, "conversation context expired", nil).
		WithComponent("conversation").
		WithOperation("access_context").
		WithContext("context_id", contextID).
		WithContext("expired_at", expiredAt.Format(time.RFC3339)).
		WithUserMessage("The conversation context has expired. Please start a new conversation.").
		WithActions("Create new context", "Extend context lifetime", "Check retention policy")
}

// NewContextSerializationError creates a context serialization error
func NewContextSerializationError(contextID string, operation string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeContextSerialization, "context serialization failed", cause).
		WithComponent("conversation").
		WithOperation(operation).
		WithContext("context_id", contextID).
		WithUserMessage("Failed to process conversation context.").
		WithActions("Check context format", "Verify context size", "Clear corrupted context")
}

// NewContextCorruptedError creates a context corrupted error
func NewContextCorruptedError(contextID string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeContextCorrupted, "conversation context corrupted", cause).
		WithComponent("conversation").
		WithOperation("load_context").
		WithContext("context_id", contextID).
		WithUserMessage("The conversation context is corrupted and cannot be loaded.").
		WithActions("Reset context", "Start new conversation", "Contact support").
		WithSeverity(errors.SeverityHigh)
}

// NewContextSizeLimitExceededError creates a context size limit exceeded error
func NewContextSizeLimitExceededError(contextID string, size, limit int64) *errors.AssistantError {
	return errors.NewValidationError(CodeContextSizeLimitExceeded, "context size limit exceeded", nil).
		WithComponent("conversation").
		WithOperation("update_context").
		WithContext("context_id", contextID).
		WithContext("size", size).
		WithContext("limit", limit).
		WithContext("excess_bytes", size-limit).
		WithUserMessage("Conversation context has grown too large.").
		WithActions("Start new conversation", "Clear old messages", "Summarize context")
}

// Turn Management Error Constructors

// NewTurnCreationError creates a turn creation error
func NewTurnCreationError(conversationID string, cause error) *errors.AssistantError {
	return errors.NewBusinessError(CodeTurnCreation, "failed to create conversation turn", cause).
		WithComponent("conversation").
		WithOperation("create_turn").
		WithContext("conversation_id", conversationID).
		WithUserMessage("Failed to process your request. Please try again.").
		WithActions("Check conversation state", "Retry request", "Start new conversation").
		WithRetryable(true)
}

// NewTurnTimeoutError creates a turn timeout error
func NewTurnTimeoutError(conversationID, turnID string, timeout time.Duration) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeTurnTimeout, "conversation turn timed out", nil).
		WithComponent("conversation").
		WithOperation("execute_turn").
		WithContext("conversation_id", conversationID).
		WithContext("turn_id", turnID).
		WithContext("timeout", timeout.String()).
		WithDuration(timeout).
		WithUserMessage("Request took too long to process. Please try again.").
		WithActions("Simplify request", "Check system load", "Retry with shorter input").
		WithRetryable(true)
}

// Storage Error Constructors

// NewConversationStorageFailureError creates a storage failure error
func NewConversationStorageFailureError(operation string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeConversationStorageFailure, "conversation storage operation failed", cause).
		WithComponent("conversation").
		WithOperation(operation).
		WithUserMessage("Failed to save conversation data. Please try again.").
		WithActions("Check storage availability", "Verify permissions", "Retry operation").
		WithRetryable(true)
}

// NewConversationLoadFailureError creates a load failure error
func NewConversationLoadFailureError(conversationID string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeConversationLoadFailure, "failed to load conversation", cause).
		WithComponent("conversation").
		WithOperation("load").
		WithContext("conversation_id", conversationID).
		WithUserMessage("Failed to load conversation. Please try again.").
		WithActions("Check conversation ID", "Verify storage connection", "Check data integrity").
		WithRetryable(true)
}

// Conversation-specific error helpers

// IsConversationNotFoundError checks if an error indicates a conversation was not found
func IsConversationNotFoundError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		return assistantErr.Code == CodeConversationNotFound
	}
	return false
}

// IsConversationAccessError checks if an error is related to conversation access
func IsConversationAccessError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeConversationArchived, CodeConversationExpired,
			CodeConversationLocked, CodeConversationInvalidState:
			return true
		}
	}
	return false
}

// IsMessageError checks if an error is related to messages
func IsMessageError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeMessageCreation, CodeMessageNotFound,
			CodeMessageUpdate, CodeMessageTooLong,
			CodeMessageRateLimit, CodeMessageDuplication,
			CodeMessageOrdering:
			return true
		}
	}
	return false
}

// IsContextError checks if an error is related to conversation context
func IsContextError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeContextNotFound, CodeContextExpired,
			CodeContextCreation, CodeContextSerialization,
			CodeContextCorrupted, CodeContextSizeLimitExceeded:
			return true
		}
	}
	return false
}

// IsRetryableConversationError checks if a conversation error is retryable
func IsRetryableConversationError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		return assistantErr.Retryable
	}
	return false
}

// GetConversationID extracts conversation ID from conversation errors
func GetConversationID(err error) string {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		if convID, ok := assistantErr.Context["conversation_id"].(string); ok {
			return convID
		}
	}
	return ""
}
