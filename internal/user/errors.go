// Package users provides domain-specific error handling for user authentication and authorization
// following CLAUDE.md best practices with proper error hierarchy and context.
package user

import (
	"fmt"
	"time"

	"github.com/koopa0/assistant-go/internal/errors"
)

// Authentication and authorization error codes
const (
	// Authentication errors
	CodeAuthenticationFailed  = "AUTH_FAILED"
	CodeInvalidCredentials    = "AUTH_INVALID_CREDENTIALS"
	CodeTokenExpired          = "AUTH_TOKEN_EXPIRED"
	CodeTokenInvalid          = "AUTH_TOKEN_INVALID"
	CodeTokenRevoked          = "AUTH_TOKEN_REVOKED"
	CodeTokenGenerationFailed = "AUTH_TOKEN_GENERATION_FAILED"
	CodeRefreshTokenInvalid   = "AUTH_REFRESH_TOKEN_INVALID"
	CodeMFARequired           = "AUTH_MFA_REQUIRED"
	CodeMFAFailed             = "AUTH_MFA_FAILED"
	CodeSessionExpired        = "AUTH_SESSION_EXPIRED"
	CodeSessionInvalid        = "AUTH_SESSION_INVALID"

	// Authorization errors
	CodeUnauthorized            = "AUTHZ_UNAUTHORIZED"
	CodeForbidden               = "AUTHZ_FORBIDDEN"
	CodeInsufficientPermissions = "AUTHZ_INSUFFICIENT_PERMISSIONS"
	CodeResourceAccessDenied    = "AUTHZ_RESOURCE_ACCESS_DENIED"
	CodeRoleNotFound            = "AUTHZ_ROLE_NOT_FOUND"
	CodePermissionNotFound      = "AUTHZ_PERMISSION_NOT_FOUND"
	CodeScopeInvalid            = "AUTHZ_SCOPE_INVALID"

	// User management errors
	CodeUserNotFound         = "USER_NOT_FOUND"
	CodeUserAlreadyExists    = "USER_ALREADY_EXISTS"
	CodeUserCreationFailed   = "USER_CREATION_FAILED"
	CodeUserUpdateFailed     = "USER_UPDATE_FAILED"
	CodeUserDeletionFailed   = "USER_DELETION_FAILED"
	CodeUserDeactivated      = "USER_DEACTIVATED"
	CodeUserLocked           = "USER_LOCKED"
	CodeUserEmailNotVerified = "USER_EMAIL_NOT_VERIFIED"

	// Password errors
	CodePasswordInvalid           = "PASSWORD_INVALID"
	CodePasswordTooWeak           = "PASSWORD_TOO_WEAK"
	CodePasswordResetRequired     = "PASSWORD_RESET_REQUIRED"
	CodePasswordResetTokenInvalid = "PASSWORD_RESET_TOKEN_INVALID"
	CodePasswordResetTokenExpired = "PASSWORD_RESET_TOKEN_EXPIRED"
	CodePasswordHistoryViolation  = "PASSWORD_HISTORY_VIOLATION"

	// API key errors
	CodeAPIKeyInvalid       = "API_KEY_INVALID"
	CodeAPIKeyExpired       = "API_KEY_EXPIRED"
	CodeAPIKeyRevoked       = "API_KEY_REVOKED"
	CodeAPIKeyQuotaExceeded = "API_KEY_QUOTA_EXCEEDED"
	CodeAPIKeyRateLimited   = "API_KEY_RATE_LIMITED"
)

// Authentication Error Constructors

// NewAuthenticationFailedError creates a generic authentication failed error
func NewAuthenticationFailedError(method string, cause error) *errors.AssistantError {
	return errors.NewValidationError(CodeAuthenticationFailed, "authentication failed", cause).
		WithComponent("auth").
		WithOperation("authenticate").
		WithContext("method", method).
		WithUserMessage("Authentication failed. Please check your credentials.").
		WithActions("Verify credentials", "Reset password if needed", "Contact support")
}

// NewInvalidCredentialsError creates an invalid credentials error
func NewInvalidCredentialsError(username string) *errors.AssistantError {
	return errors.NewValidationError(CodeInvalidCredentials, "invalid username or password", nil).
		WithComponent("auth").
		WithOperation("authenticate").
		WithContext("username", username).
		WithUserMessage("Invalid username or password.").
		WithActions("Check credentials", "Reset password", "Verify account status")
}

// NewTokenExpiredError creates a token expired error
func NewTokenExpiredError(tokenType string, expiredAt time.Time) *errors.AssistantError {
	return errors.NewValidationError(CodeTokenExpired, fmt.Sprintf("%s token expired", tokenType), nil).
		WithComponent("auth").
		WithOperation("validate_token").
		WithContext("token_type", tokenType).
		WithContext("expired_at", expiredAt.Format(time.RFC3339)).
		WithUserMessage("Your session has expired. Please sign in again.").
		WithActions("Sign in again", "Use refresh token", "Request new token")
}

// NewTokenInvalidError creates a token invalid error
func NewTokenInvalidError(tokenType string, reason string) *errors.AssistantError {
	return errors.NewValidationError(CodeTokenInvalid, fmt.Sprintf("%s token invalid", tokenType), nil).
		WithComponent("auth").
		WithOperation("validate_token").
		WithContext("token_type", tokenType).
		WithContext("reason", reason).
		WithUserMessage("Invalid authentication token.").
		WithActions("Sign in again", "Check token format", "Verify token source")
}

// NewTokenGenerationFailedError creates a token generation failed error
func NewTokenGenerationFailedError(tokenType string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeTokenGenerationFailed, "token generation failed", cause).
		WithComponent("auth").
		WithOperation("generate_token").
		WithContext("token_type", tokenType).
		WithUserMessage("Failed to generate authentication token. Please try again.").
		WithActions("Retry sign in", "Check system status", "Contact support").
		WithRetryable(true)
}

// NewMFARequiredError creates an MFA required error
func NewMFARequiredError(userID, method string) *errors.AssistantError {
	return errors.NewValidationError(CodeMFARequired, "multi-factor authentication required", nil).
		WithComponent("auth").
		WithOperation("authenticate").
		WithContext("user_id", userID).
		WithContext("mfa_method", method).
		WithUserMessage("Multi-factor authentication is required.").
		WithActions("Complete MFA challenge", "Setup MFA if needed", "Use backup codes")
}

// NewSessionExpiredError creates a session expired error
func NewSessionExpiredError(sessionID string, expiredAt time.Time) *errors.AssistantError {
	return errors.NewValidationError(CodeSessionExpired, "session expired", nil).
		WithComponent("auth").
		WithOperation("validate_session").
		WithContext("session_id", sessionID).
		WithContext("expired_at", expiredAt.Format(time.RFC3339)).
		WithUserMessage("Your session has expired. Please sign in again.").
		WithActions("Sign in again", "Enable remember me", "Check session settings")
}

// Authorization Error Constructors

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(reason string) *errors.AssistantError {
	return errors.NewValidationError(CodeUnauthorized, "unauthorized access", nil).
		WithComponent("auth").
		WithOperation("authorize").
		WithContext("reason", reason).
		WithUserMessage("Access denied. Please sign in.").
		WithActions("Sign in", "Check credentials", "Verify account access")
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(resource, action string) *errors.AssistantError {
	return errors.NewValidationError(CodeForbidden, "access forbidden", nil).
		WithComponent("auth").
		WithOperation("authorize").
		WithContext("resource", resource).
		WithContext("action", action).
		WithUserMessage("You don't have permission to access this resource.").
		WithActions("Request access", "Check permissions", "Contact administrator")
}

// NewInsufficientPermissionsError creates an insufficient permissions error
func NewInsufficientPermissionsError(required, actual []string) *errors.AssistantError {
	return errors.NewValidationError(CodeInsufficientPermissions, "insufficient permissions", nil).
		WithComponent("auth").
		WithOperation("check_permissions").
		WithContext("required_permissions", required).
		WithContext("actual_permissions", actual).
		WithUserMessage("You don't have the required permissions for this action.").
		WithActions("Request additional permissions", "Contact administrator", "Check role assignments")
}

// NewResourceAccessDeniedError creates a resource access denied error
func NewResourceAccessDeniedError(resourceType, resourceID, reason string) *errors.AssistantError {
	return errors.NewValidationError(CodeResourceAccessDenied, "resource access denied", nil).
		WithComponent("auth").
		WithOperation("check_access").
		WithContext("resource_type", resourceType).
		WithContext("resource_id", resourceID).
		WithContext("reason", reason).
		WithUserMessage("Access to the requested resource is denied.").
		WithActions("Check resource permissions", "Request access", "Verify resource ownership")
}

// User Management Error Constructors

// NewUserNotFoundError creates a user not found error
func NewUserNotFoundError(identifier string) *errors.AssistantError {
	return errors.NewBusinessError(CodeUserNotFound, "user not found", nil).
		WithComponent("users").
		WithOperation("retrieve").
		WithContext("identifier", identifier).
		WithUserMessage("User account not found.").
		WithActions("Check user identifier", "Verify account exists", "Create new account")
}

// NewUserAlreadyExistsError creates a user already exists error
func NewUserAlreadyExistsError(email string) *errors.AssistantError {
	return errors.NewValidationError(CodeUserAlreadyExists, "user already exists", nil).
		WithComponent("users").
		WithOperation("create").
		WithContext("email", email).
		WithUserMessage("An account with this email already exists.").
		WithActions("Sign in instead", "Use different email", "Reset password")
}

// NewUserCreationFailedError creates a user creation failed error
func NewUserCreationFailedError(email string, cause error) *errors.AssistantError {
	return errors.NewBusinessError(CodeUserCreationFailed, "user creation failed", cause).
		WithComponent("users").
		WithOperation("create").
		WithContext("email", email).
		WithUserMessage("Failed to create user account. Please try again.").
		WithActions("Check input data", "Verify email format", "Try again later").
		WithRetryable(true)
}

// NewUserDeactivatedError creates a user deactivated error
func NewUserDeactivatedError(userID string, deactivatedAt time.Time) *errors.AssistantError {
	return errors.NewValidationError(CodeUserDeactivated, "user account deactivated", nil).
		WithComponent("users").
		WithOperation("authenticate").
		WithContext("user_id", userID).
		WithContext("deactivated_at", deactivatedAt.Format(time.RFC3339)).
		WithUserMessage("Your account has been deactivated.").
		WithActions("Contact support", "Request reactivation", "Create new account")
}

// NewUserLockedError creates a user locked error
func NewUserLockedError(userID string, lockedUntil *time.Time, reason string) *errors.AssistantError {
	err := errors.NewValidationError(CodeUserLocked, "user account locked", nil).
		WithComponent("users").
		WithOperation("authenticate").
		WithContext("user_id", userID).
		WithContext("reason", reason).
		WithUserMessage("Your account has been locked due to security reasons.").
		WithActions("Wait for unlock", "Contact support", "Reset password")

	if lockedUntil != nil {
		err.WithContext("locked_until", lockedUntil.Format(time.RFC3339)).
			WithRetryAfter(time.Until(*lockedUntil))
	}

	return err
}

// Password Error Constructors

// NewPasswordInvalidError creates a password invalid error
func NewPasswordInvalidError(reason string) *errors.AssistantError {
	return errors.NewValidationError(CodePasswordInvalid, "password validation failed", nil).
		WithComponent("users").
		WithOperation("validate_password").
		WithContext("reason", reason).
		WithUserMessage(fmt.Sprintf("Password is invalid: %s", reason)).
		WithActions("Check password requirements", "Use stronger password", "Follow password policy")
}

// NewPasswordTooWeakError creates a password too weak error
func NewPasswordTooWeakError(score int, minScore int) *errors.AssistantError {
	return errors.NewValidationError(CodePasswordTooWeak, "password too weak", nil).
		WithComponent("users").
		WithOperation("validate_password").
		WithContext("strength_score", score).
		WithContext("minimum_score", minScore).
		WithUserMessage("Password is too weak. Please use a stronger password.").
		WithActions("Add uppercase and lowercase letters", "Include numbers and symbols", "Make password longer")
}

// NewPasswordResetRequiredError creates a password reset required error
func NewPasswordResetRequiredError(userID string, reason string) *errors.AssistantError {
	return errors.NewValidationError(CodePasswordResetRequired, "password reset required", nil).
		WithComponent("users").
		WithOperation("authenticate").
		WithContext("user_id", userID).
		WithContext("reason", reason).
		WithUserMessage("Password reset is required before you can continue.").
		WithActions("Reset password", "Check email for reset link", "Contact support")
}

// API Key Error Constructors

// NewAPIKeyInvalidError creates an API key invalid error
func NewAPIKeyInvalidError(keyPrefix string) *errors.AssistantError {
	return errors.NewValidationError(CodeAPIKeyInvalid, "invalid API key", nil).
		WithComponent("auth").
		WithOperation("validate_api_key").
		WithContext("key_prefix", keyPrefix).
		WithUserMessage("Invalid API key.").
		WithActions("Check API key", "Generate new key", "Verify key permissions")
}

// NewAPIKeyExpiredError creates an API key expired error
func NewAPIKeyExpiredError(keyID string, expiredAt time.Time) *errors.AssistantError {
	return errors.NewValidationError(CodeAPIKeyExpired, "API key expired", nil).
		WithComponent("auth").
		WithOperation("validate_api_key").
		WithContext("key_id", keyID).
		WithContext("expired_at", expiredAt.Format(time.RFC3339)).
		WithUserMessage("API key has expired.").
		WithActions("Generate new API key", "Extend key validity", "Use non-expiring key")
}

// NewAPIKeyRateLimitedError creates an API key rate limited error
func NewAPIKeyRateLimitedError(keyID string, limit int, window time.Duration, retryAfter time.Duration) *errors.AssistantError {
	return errors.NewValidationError(CodeAPIKeyRateLimited, "API key rate limited", nil).
		WithComponent("auth").
		WithOperation("check_rate_limit").
		WithContext("key_id", keyID).
		WithContext("limit", limit).
		WithContext("window", window.String()).
		WithUserMessage("API rate limit exceeded.").
		WithActions("Wait before retry", "Upgrade API plan", "Distribute requests").
		WithRetryAfter(retryAfter)
}

// Authentication/Authorization error helpers

// IsAuthenticationError checks if an error is authentication-related
func IsAuthenticationError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeAuthenticationFailed, CodeInvalidCredentials,
			CodeTokenExpired, CodeTokenInvalid, CodeTokenRevoked,
			CodeMFARequired, CodeMFAFailed, CodeSessionExpired:
			return true
		}
	}
	return false
}

// IsAuthorizationError checks if an error is authorization-related
func IsAuthorizationError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeUnauthorized, CodeForbidden,
			CodeInsufficientPermissions, CodeResourceAccessDenied:
			return true
		}
	}
	return false
}

// IsUserError checks if an error is user-related
func IsUserError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		return assistantErr.Component == "users"
	}
	return false
}

// IsRetryableAuthError checks if an auth error is retryable
func IsRetryableAuthError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		// Most auth errors are not retryable except infrastructure issues
		switch assistantErr.Code {
		case CodeTokenGenerationFailed, CodeUserCreationFailed:
			return true
		}
		return assistantErr.Retryable
	}
	return false
}

// RequiresReauthentication checks if an error requires re-authentication
func RequiresReauthentication(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeTokenExpired, CodeTokenInvalid, CodeTokenRevoked,
			CodeSessionExpired, CodeSessionInvalid:
			return true
		}
	}
	return false
}
