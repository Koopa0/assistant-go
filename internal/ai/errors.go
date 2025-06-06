// Package ai provides domain-specific error handling for AI operations
// following CLAUDE.md best practices with proper error hierarchy and context.
package ai

import (
	"fmt"
	"time"

	"github.com/koopa0/assistant-go/internal/errors"
)

// AI-specific error codes
const (
	// Provider errors
	CodeProviderInitialization = "AI_PROVIDER_INITIALIZATION"
	CodeProviderAuthentication = "AI_PROVIDER_AUTHENTICATION"
	CodeProviderQuotaExceeded  = "AI_PROVIDER_QUOTA_EXCEEDED"
	CodeProviderModelNotFound  = "AI_PROVIDER_MODEL_NOT_FOUND"
	CodeProviderTimeout        = "AI_PROVIDER_TIMEOUT"

	// Request processing errors
	CodeInvalidPrompt     = "AI_INVALID_PROMPT"
	CodePromptTooLong     = "AI_PROMPT_TOO_LONG"
	CodeInvalidParameters = "AI_INVALID_PARAMETERS"
	CodeResponseTruncated = "AI_RESPONSE_TRUNCATED"
	CodeResponseFiltered  = "AI_RESPONSE_FILTERED"

	// Token management errors
	CodeTokenCountExceeded     = "AI_TOKEN_COUNT_EXCEEDED"
	CodeTokenCalculationFailed = "AI_TOKEN_CALCULATION_FAILED"

	// Embedding errors
	CodeEmbeddingGeneration       = "AI_EMBEDDING_GENERATION"
	CodeEmbeddingInvalidDimension = "AI_EMBEDDING_INVALID_DIMENSION"
)

// AI Provider Error Constructors

// NewProviderInitializationError creates a provider initialization error
func NewProviderInitializationError(provider string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeProviderInitialization, "AI provider initialization failed", cause).
		WithComponent("ai").
		WithContext("provider", provider).
		WithUserMessage("AI service is temporarily unavailable. Please try again.").
		WithActions("Check provider configuration", "Verify API keys", "Check provider status").
		WithRetryAfter(time.Minute * 2)
}

// NewProviderAuthenticationError creates a provider authentication error
func NewProviderAuthenticationError(provider string, cause error) *errors.AssistantError {
	return errors.NewValidationError(CodeProviderAuthentication, "AI provider authentication failed", cause).
		WithComponent("ai").
		WithContext("provider", provider).
		WithUserMessage("AI service authentication failed. Please contact support.").
		WithActions("Check API key validity", "Verify provider credentials", "Contact provider support").
		WithSeverity(errors.SeverityHigh)
}

// NewProviderQuotaExceededError creates a quota exceeded error
func NewProviderQuotaExceededError(provider string, quotaType string, resetTime *time.Time) *errors.AssistantError {
	err := errors.NewValidationError(CodeProviderQuotaExceeded, "AI provider quota exceeded", nil).
		WithComponent("ai").
		WithContext("provider", provider).
		WithContext("quota_type", quotaType).
		WithUserMessage("AI service quota exceeded. Please wait before trying again.").
		WithActions("Wait for quota reset", "Use alternative provider", "Upgrade quota if available")

	if resetTime != nil {
		err.WithContext("reset_time", resetTime.Format(time.RFC3339)).
			WithRetryAfter(time.Until(*resetTime))
	}

	return err
}

// NewProviderModelNotFoundError creates a model not found error
func NewProviderModelNotFoundError(provider, model string) *errors.AssistantError {
	return errors.NewBusinessError(CodeProviderModelNotFound, "AI model not found", nil).
		WithComponent("ai").
		WithContext("provider", provider).
		WithContext("model", model).
		WithUserMessage("The requested AI model is not available.").
		WithActions("Check model name", "Use alternative model", "Verify provider capabilities")
}

// NewProviderTimeoutError creates a provider timeout error
func NewProviderTimeoutError(provider string, timeout time.Duration, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeProviderTimeout, "AI provider request timed out", cause).
		WithComponent("ai").
		WithContext("provider", provider).
		WithContext("timeout", timeout.String()).
		WithDuration(timeout).
		WithUserMessage("AI request timed out. Please try again.").
		WithActions("Simplify prompt", "Reduce token limit", "Try again later").
		WithRetryAfter(time.Second * 30)
}

// Request Processing Error Constructors

// NewInvalidPromptError creates an invalid prompt error
func NewInvalidPromptError(reason string, prompt string) *errors.AssistantError {
	// Truncate prompt for logging to avoid exposing sensitive data
	truncatedPrompt := prompt
	if len(prompt) > 100 {
		truncatedPrompt = prompt[:100] + "..."
	}

	return errors.NewValidationError(CodeInvalidPrompt, fmt.Sprintf("invalid prompt: %s", reason), nil).
		WithComponent("ai").
		WithContext("reason", reason).
		WithContext("prompt_length", len(prompt)).
		WithContext("prompt_preview", truncatedPrompt).
		WithUserMessage("Invalid prompt format. Please check your input.").
		WithActions("Review prompt format", "Check for forbidden content", "Simplify prompt")
}

// NewPromptTooLongError creates a prompt too long error
func NewPromptTooLongError(length, maxLength int) *errors.AssistantError {
	return errors.NewValidationError(CodePromptTooLong, "prompt exceeds maximum length", nil).
		WithComponent("ai").
		WithContext("length", length).
		WithContext("max_length", maxLength).
		WithContext("excess_chars", length-maxLength).
		WithUserMessage(fmt.Sprintf("Prompt is too long (%d characters, max %d).", length, maxLength)).
		WithActions("Shorten prompt", "Split into multiple requests", "Use a different model")
}

// NewInvalidParametersError creates an invalid parameters error
func NewInvalidParametersError(parameter string, value interface{}, reason string) *errors.AssistantError {
	return errors.NewValidationError(CodeInvalidParameters, fmt.Sprintf("invalid parameter %s: %s", parameter, reason), nil).
		WithComponent("ai").
		WithContext("parameter", parameter).
		WithContext("value", value).
		WithContext("reason", reason).
		WithUserMessage(fmt.Sprintf("Invalid parameter: %s", reason)).
		WithActions("Check parameter format", "Review parameter limits", "Use default values")
}

// NewResponseTruncatedError creates a response truncated error
func NewResponseTruncatedError(actualTokens, maxTokens int) *errors.AssistantError {
	return errors.NewBusinessError(CodeResponseTruncated, "AI response was truncated", nil).
		WithComponent("ai").
		WithContext("actual_tokens", actualTokens).
		WithContext("max_tokens", maxTokens).
		WithUserMessage("The response was truncated due to length limits.").
		WithActions("Increase token limit", "Request continuation", "Simplify query").
		WithSeverity(errors.SeverityLow)
}

// NewResponseFilteredError creates a response filtered error
func NewResponseFilteredError(reason string) *errors.AssistantError {
	return errors.NewBusinessError(CodeResponseFiltered, "AI response was filtered", nil).
		WithComponent("ai").
		WithContext("filter_reason", reason).
		WithUserMessage("The response was filtered due to content policies.").
		WithActions("Rephrase query", "Use different approach", "Contact support if needed").
		WithSeverity(errors.SeverityMedium)
}

// Token Management Error Constructors

// NewTokenCountExceededError creates a token count exceeded error
func NewTokenCountExceededError(tokens, limit int, tokenType string) *errors.AssistantError {
	return errors.NewValidationError(CodeTokenCountExceeded, "token count exceeded", nil).
		WithComponent("ai").
		WithContext("tokens", tokens).
		WithContext("limit", limit).
		WithContext("token_type", tokenType).
		WithContext("excess_tokens", tokens-limit).
		WithUserMessage(fmt.Sprintf("Token limit exceeded (%d/%d %s tokens).", tokens, limit, tokenType)).
		WithActions("Reduce content length", "Split into smaller requests", "Increase token limit")
}

// NewTokenCalculationFailedError creates a token calculation error
func NewTokenCalculationFailedError(content string, cause error) *errors.AssistantError {
	contentLength := len(content)
	if contentLength > 100 {
		content = content[:100] + "..."
	}

	return errors.NewBusinessError(CodeTokenCalculationFailed, "token calculation failed", cause).
		WithComponent("ai").
		WithContext("content_length", contentLength).
		WithContext("content_preview", content).
		WithUserMessage("Unable to calculate token count. Please try again.").
		WithActions("Verify content format", "Check encoding", "Contact support")
}

// Embedding Error Constructors

// NewEmbeddingGenerationError creates an embedding generation error
func NewEmbeddingGenerationError(content string, cause error) *errors.AssistantError {
	contentLength := len(content)
	if len(content) > 100 {
		content = content[:100] + "..."
	}

	return errors.NewBusinessError(CodeEmbeddingGeneration, "embedding generation failed", cause).
		WithComponent("ai").
		WithContext("content_length", contentLength).
		WithContext("content_preview", content).
		WithUserMessage("Unable to generate embeddings. Please try again.").
		WithActions("Check content format", "Verify content encoding", "Try alternative content").
		WithRetryable(true)
}

// NewEmbeddingInvalidDimensionError creates an embedding dimension error
func NewEmbeddingInvalidDimensionError(expected, actual int) *errors.AssistantError {
	return errors.NewValidationError(CodeEmbeddingInvalidDimension, "embedding dimension mismatch", nil).
		WithComponent("ai").
		WithContext("expected_dimension", expected).
		WithContext("actual_dimension", actual).
		WithUserMessage("Embedding dimension mismatch detected.").
		WithActions("Check embedding model", "Verify dimension configuration", "Regenerate embeddings")
}

// AI-specific error helpers

// IsQuotaError checks if an error is related to quota/rate limiting
func IsQuotaError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		return assistantErr.Code == CodeProviderQuotaExceeded
	}
	return false
}

// IsProviderError checks if an error is provider-related
func IsProviderError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeProviderInitialization, CodeProviderAuthentication,
			CodeProviderQuotaExceeded, CodeProviderModelNotFound,
			CodeProviderTimeout:
			return true
		}
	}
	return false
}

// IsRetryableProviderError checks if a provider error is retryable
func IsRetryableProviderError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeProviderInitialization, CodeProviderTimeout:
			return true
		case CodeProviderAuthentication, CodeProviderModelNotFound:
			return false
		case CodeProviderQuotaExceeded:
			// Retryable if reset time is available
			return assistantErr.RetryAfter != nil
		}
	}
	return false
}

// GetRetryDelay extracts retry delay from AI errors
func GetRetryDelay(err error) *time.Duration {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		return assistantErr.RetryAfter
	}
	return nil
}
