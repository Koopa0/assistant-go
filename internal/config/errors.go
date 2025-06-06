// Package config provides domain-specific error handling for configuration operations
// following CLAUDE.md best practices with proper error hierarchy and context.
package config

import (
	"fmt"

	"github.com/koopa0/assistant-go/internal/errors"
)

// Configuration-specific error codes
const (
	// Configuration loading errors
	CodeConfigFileNotFound    = "CONFIG_FILE_NOT_FOUND"
	CodeConfigFileReadError   = "CONFIG_FILE_READ_ERROR"
	CodeConfigParseError      = "CONFIG_PARSE_ERROR"
	CodeConfigInvalidFormat   = "CONFIG_INVALID_FORMAT"
	CodeConfigSchemaViolation = "CONFIG_SCHEMA_VIOLATION"

	// Configuration validation errors
	CodeConfigValidationFailed = "CONFIG_VALIDATION_FAILED"
	CodeConfigMissingRequired  = "CONFIG_MISSING_REQUIRED"
	CodeConfigInvalidValue     = "CONFIG_INVALID_VALUE"
	CodeConfigTypeError        = "CONFIG_TYPE_ERROR"
	CodeConfigRangeError       = "CONFIG_RANGE_ERROR"
	CodeConfigPatternError     = "CONFIG_PATTERN_ERROR"

	// Environment configuration errors
	CodeEnvVarNotFound  = "CONFIG_ENV_VAR_NOT_FOUND"
	CodeEnvVarInvalid   = "CONFIG_ENV_VAR_INVALID"
	CodeEnvVarTypeError = "CONFIG_ENV_VAR_TYPE_ERROR"

	// Configuration merge errors
	CodeConfigMergeConflict     = "CONFIG_MERGE_CONFLICT"
	CodeConfigMergeTypeMismatch = "CONFIG_MERGE_TYPE_MISMATCH"

	// Configuration watch errors
	CodeConfigWatchError  = "CONFIG_WATCH_ERROR"
	CodeConfigReloadError = "CONFIG_RELOAD_ERROR"

	// Secret configuration errors
	CodeSecretNotFound         = "CONFIG_SECRET_NOT_FOUND"
	CodeSecretAccessDenied     = "CONFIG_SECRET_ACCESS_DENIED"
	CodeSecretDecryptionFailed = "CONFIG_SECRET_DECRYPTION_FAILED"
)

// Configuration Loading Error Constructors

// NewConfigFileNotFoundError creates a config file not found error
func NewConfigFileNotFoundError(path string) *errors.AssistantError {
	return errors.NewValidationError(CodeConfigFileNotFound, "configuration file not found", nil).
		WithComponent("config").
		WithOperation("load_file").
		WithContext("path", path).
		WithUserMessage("Configuration file not found.").
		WithActions("Check file path", "Create configuration file", "Use default configuration")
}

// NewConfigFileReadError creates a config file read error
func NewConfigFileReadError(path string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeConfigFileReadError, "failed to read configuration file", cause).
		WithComponent("config").
		WithOperation("read_file").
		WithContext("path", path).
		WithUserMessage("Failed to read configuration file.").
		WithActions("Check file permissions", "Verify file exists", "Check disk space").
		WithRetryable(true)
}

// NewConfigParseError creates a config parse error
func NewConfigParseError(path, format string, line, column int, cause error) *errors.AssistantError {
	err := errors.NewValidationError(CodeConfigParseError, "failed to parse configuration", cause).
		WithComponent("config").
		WithOperation("parse").
		WithContext("path", path).
		WithContext("format", format).
		WithUserMessage("Configuration file contains syntax errors.")

	if line > 0 {
		err.WithContext("line", line)
	}
	if column > 0 {
		err.WithContext("column", column)
	}

	return err.WithActions("Check configuration syntax", "Validate against schema", "Use configuration linter")
}

// NewConfigInvalidFormatError creates an invalid format error
func NewConfigInvalidFormatError(path, expectedFormat, actualFormat string) *errors.AssistantError {
	return errors.NewValidationError(CodeConfigInvalidFormat, "invalid configuration format", nil).
		WithComponent("config").
		WithOperation("detect_format").
		WithContext("path", path).
		WithContext("expected_format", expectedFormat).
		WithContext("actual_format", actualFormat).
		WithUserMessage(fmt.Sprintf("Configuration file must be in %s format.", expectedFormat)).
		WithActions("Convert to correct format", "Check file extension", "Use proper serialization")
}

// Configuration Validation Error Constructors

// NewConfigValidationFailedError creates a validation failed error
func NewConfigValidationFailedError(field string, value interface{}, reason string) *errors.AssistantError {
	return errors.NewValidationError(CodeConfigValidationFailed, "configuration validation failed", nil).
		WithComponent("config").
		WithOperation("validate").
		WithContext("field", field).
		WithContext("value", value).
		WithContext("reason", reason).
		WithUserMessage(fmt.Sprintf("Configuration validation failed: %s", reason)).
		WithActions("Fix configuration value", "Check validation rules", "Use example configuration")
}

// NewConfigMissingRequiredError creates a missing required field error
func NewConfigMissingRequiredError(field string, configSection string) *errors.AssistantError {
	return errors.NewValidationError(CodeConfigMissingRequired, fmt.Sprintf("missing required configuration: %s", field), nil).
		WithComponent("config").
		WithOperation("validate").
		WithContext("field", field).
		WithContext("section", configSection).
		WithUserMessage(fmt.Sprintf("Required configuration '%s' is missing.", field)).
		WithActions("Add required field", "Check configuration template", "Set via environment variable").
		WithSeverity(errors.SeverityHigh)
}

// NewConfigInvalidValueError creates an invalid value error
func NewConfigInvalidValueError(field string, value interface{}, validValues []interface{}) *errors.AssistantError {
	return errors.NewValidationError(CodeConfigInvalidValue, "invalid configuration value", nil).
		WithComponent("config").
		WithOperation("validate_value").
		WithContext("field", field).
		WithContext("value", value).
		WithContext("valid_values", validValues).
		WithUserMessage(fmt.Sprintf("Invalid value for configuration '%s'.", field)).
		WithActions("Use valid value", "Check allowed values", "Review documentation")
}

// NewConfigTypeError creates a type error
func NewConfigTypeError(field string, expectedType, actualType string, value interface{}) *errors.AssistantError {
	return errors.NewValidationError(CodeConfigTypeError, "configuration type mismatch", nil).
		WithComponent("config").
		WithOperation("validate_type").
		WithContext("field", field).
		WithContext("expected_type", expectedType).
		WithContext("actual_type", actualType).
		WithContext("value", value).
		WithUserMessage(fmt.Sprintf("Configuration '%s' must be %s, got %s.", field, expectedType, actualType)).
		WithActions("Fix value type", "Use proper format", "Check type conversion")
}

// NewConfigRangeError creates a range error
func NewConfigRangeError(field string, value, min, max interface{}) *errors.AssistantError {
	return errors.NewValidationError(CodeConfigRangeError, "configuration value out of range", nil).
		WithComponent("config").
		WithOperation("validate_range").
		WithContext("field", field).
		WithContext("value", value).
		WithContext("min", min).
		WithContext("max", max).
		WithUserMessage(fmt.Sprintf("Configuration '%s' must be between %v and %v.", field, min, max)).
		WithActions("Adjust value", "Check limits", "Use default value")
}

// Environment Configuration Error Constructors

// NewEnvVarNotFoundError creates an environment variable not found error
func NewEnvVarNotFoundError(varName string, fallbackConfig string) *errors.AssistantError {
	return errors.NewValidationError(CodeEnvVarNotFound, "environment variable not found", nil).
		WithComponent("config").
		WithOperation("read_env").
		WithContext("variable", varName).
		WithContext("fallback_config", fallbackConfig).
		WithUserMessage(fmt.Sprintf("Required environment variable '%s' is not set.", varName)).
		WithActions("Set environment variable", "Use configuration file", "Check .env file")
}

// NewEnvVarInvalidError creates an invalid environment variable error
func NewEnvVarInvalidError(varName string, value, reason string) *errors.AssistantError {
	return errors.NewValidationError(CodeEnvVarInvalid, "invalid environment variable", nil).
		WithComponent("config").
		WithOperation("parse_env").
		WithContext("variable", varName).
		WithContext("value", value).
		WithContext("reason", reason).
		WithUserMessage(fmt.Sprintf("Environment variable '%s' has invalid value: %s", varName, reason)).
		WithActions("Fix variable value", "Check format requirements", "Use valid format")
}

// Configuration Merge Error Constructors

// NewConfigMergeConflictError creates a merge conflict error
func NewConfigMergeConflictError(field string, value1, value2 interface{}, source1, source2 string) *errors.AssistantError {
	return errors.NewBusinessError(CodeConfigMergeConflict, "configuration merge conflict", nil).
		WithComponent("config").
		WithOperation("merge").
		WithContext("field", field).
		WithContext("value1", value1).
		WithContext("value2", value2).
		WithContext("source1", source1).
		WithContext("source2", source2).
		WithUserMessage("Configuration conflict detected during merge.").
		WithActions("Resolve conflict manually", "Set priority order", "Use explicit override")
}

// Configuration Watch Error Constructors

// NewConfigWatchError creates a config watch error
func NewConfigWatchError(path string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeConfigWatchError, "configuration watch failed", cause).
		WithComponent("config").
		WithOperation("watch").
		WithContext("path", path).
		WithUserMessage("Failed to watch configuration changes.").
		WithActions("Check file system permissions", "Verify watcher limits", "Disable auto-reload")
}

// NewConfigReloadError creates a config reload error
func NewConfigReloadError(path string, cause error) *errors.AssistantError {
	return errors.NewBusinessError(CodeConfigReloadError, "configuration reload failed", cause).
		WithComponent("config").
		WithOperation("reload").
		WithContext("path", path).
		WithUserMessage("Failed to reload configuration.").
		WithActions("Check configuration validity", "Review changes", "Revert to previous version").
		WithRetryable(true)
}

// Secret Configuration Error Constructors

// NewSecretNotFoundError creates a secret not found error
func NewSecretNotFoundError(secretName, provider string) *errors.AssistantError {
	return errors.NewValidationError(CodeSecretNotFound, "secret not found", nil).
		WithComponent("config").
		WithOperation("fetch_secret").
		WithContext("secret_name", secretName).
		WithContext("provider", provider).
		WithUserMessage("Required secret configuration not found.").
		WithActions("Create secret", "Check secret name", "Verify provider configuration")
}

// NewSecretAccessDeniedError creates a secret access denied error
func NewSecretAccessDeniedError(secretName, provider string, cause error) *errors.AssistantError {
	return errors.NewValidationError(CodeSecretAccessDenied, "secret access denied", cause).
		WithComponent("config").
		WithOperation("fetch_secret").
		WithContext("secret_name", secretName).
		WithContext("provider", provider).
		WithUserMessage("Access denied to secret configuration.").
		WithActions("Check IAM permissions", "Verify authentication", "Request access")
}

// NewSecretDecryptionFailedError creates a secret decryption failed error
func NewSecretDecryptionFailedError(secretName string, cause error) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeSecretDecryptionFailed, "secret decryption failed", cause).
		WithComponent("config").
		WithOperation("decrypt_secret").
		WithContext("secret_name", secretName).
		WithUserMessage("Failed to decrypt secret configuration.").
		WithActions("Check encryption key", "Verify secret format", "Re-encrypt secret").
		WithSeverity(errors.SeverityHigh)
}

// Configuration-specific error helpers

// IsConfigurationError checks if an error is configuration-related
func IsConfigurationError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		return assistantErr.Component == "config"
	}
	return false
}

// IsValidationError checks if an error is a configuration validation error
func IsValidationError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeConfigValidationFailed, CodeConfigMissingRequired,
			CodeConfigInvalidValue, CodeConfigTypeError,
			CodeConfigRangeError, CodeConfigPatternError:
			return true
		}
	}
	return false
}

// IsCriticalConfigError checks if a configuration error is critical
func IsCriticalConfigError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		// Missing required config and secret failures are critical
		switch assistantErr.Code {
		case CodeConfigMissingRequired, CodeSecretNotFound,
			CodeSecretAccessDenied, CodeSecretDecryptionFailed:
			return true
		}
		return assistantErr.Severity == errors.SeverityCritical ||
			assistantErr.Severity == errors.SeverityHigh
	}
	return false
}

// GetConfigField extracts the configuration field from config errors
func GetConfigField(err error) string {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		if field, ok := assistantErr.Context["field"].(string); ok {
			return field
		}
	}
	return ""
}

// GetConfigPath extracts the configuration file path from config errors
func GetConfigPath(err error) string {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		if path, ok := assistantErr.Context["path"].(string); ok {
			return path
		}
	}
	return ""
}

// ShouldFallbackToDefault checks if the error suggests falling back to default config
func ShouldFallbackToDefault(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeConfigFileNotFound, CodeConfigFileReadError,
			CodeEnvVarNotFound:
			return true
		}
	}
	return false
}
