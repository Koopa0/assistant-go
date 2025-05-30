// Package tools provides a registry and base implementation for tools that can be
// executed by the assistant. It includes tool registration, execution, and management.
package tools

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// BaseTool provides a base implementation for tools
type BaseTool struct {
	name        string
	description string
	category    string
	version     string
	logger      *slog.Logger
	config      map[string]interface{}
}

// NewBaseTool creates a new base tool
func NewBaseTool(name, description, category string, config map[string]interface{}, logger *slog.Logger) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		category:    category,
		version:     "1.0.0",
		logger:      logger,
		config:      config,
	}
}

// Name returns the tool name
func (bt *BaseTool) Name() string {
	return bt.name
}

// Description returns the tool description
func (bt *BaseTool) Description() string {
	return bt.description
}

// Category returns the tool category
func (bt *BaseTool) Category() string {
	return bt.category
}

// Version returns the tool version
func (bt *BaseTool) Version() string {
	return bt.version
}

// Parameters returns the tool parameters schema (to be overridden by specific tools)
func (bt *BaseTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

// Execute executes the tool (to be overridden by specific tools)
func (bt *BaseTool) Execute(ctx context.Context, input map[string]interface{}) (*ToolResult, error) {
	return nil, fmt.Errorf("execute method not implemented for tool %s", bt.name)
}

// Health checks if the tool is healthy (default implementation)
func (bt *BaseTool) Health(ctx context.Context) error {
	// Default health check - can be overridden by specific tools
	return nil
}

// Close closes the tool (default implementation)
func (bt *BaseTool) Close(ctx context.Context) error {
	// Default close - can be overridden by specific tools
	return nil
}

// GetConfig returns a configuration value for the given key.
// It returns the value and a boolean indicating whether the key exists.
func (bt *BaseTool) GetConfig(key string) (interface{}, bool) {
	if bt.config == nil {
		return nil, false
	}
	value, exists := bt.config[key]
	return value, exists
}

// GetConfigString returns a string configuration value for the given key.
// It returns the string value and a boolean indicating whether the key exists and is a string.
func (bt *BaseTool) GetConfigString(key string) (string, bool) {
	value, exists := bt.GetConfig(key)
	if !exists {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// GetConfigInt returns an int configuration value for the given key.
// It returns the int value and a boolean indicating whether the key exists and is convertible to int.
// It supports both int and float64 types from JSON unmarshaling.
func (bt *BaseTool) GetConfigInt(key string) (int, bool) {
	value, exists := bt.GetConfig(key)
	if !exists {
		return 0, false
	}

	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// GetConfigBool returns a bool configuration value
func (bt *BaseTool) GetConfigBool(key string) (bool, bool) {
	value, exists := bt.GetConfig(key)
	if !exists {
		return false, false
	}
	b, ok := value.(bool)
	return b, ok
}

// LogInfo logs an info message with tool context
func (bt *BaseTool) LogInfo(ctx context.Context, message string, args ...slog.Attr) {
	attrs := append([]slog.Attr{
		slog.String("tool", bt.name),
		slog.String("category", bt.category),
	}, args...)
	bt.logger.LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// LogError logs an error message with tool context
func (bt *BaseTool) LogError(ctx context.Context, message string, err error, args ...slog.Attr) {
	attrs := append([]slog.Attr{
		slog.String("tool", bt.name),
		slog.String("category", bt.category),
		slog.Any("error", err),
	}, args...)
	bt.logger.LogAttrs(ctx, slog.LevelError, message, attrs...)
}

// LogDebug logs a debug message with tool context
func (bt *BaseTool) LogDebug(ctx context.Context, message string, args ...slog.Attr) {
	attrs := append([]slog.Attr{
		slog.String("tool", bt.name),
		slog.String("category", bt.category),
	}, args...)
	bt.logger.LogAttrs(ctx, slog.LevelDebug, message, attrs...)
}

// ValidateInput validates tool input against the parameters schema
func (bt *BaseTool) ValidateInput(input map[string]interface{}) error {
	// Basic validation - can be enhanced with JSON schema validation
	if input == nil {
		return fmt.Errorf("input cannot be nil")
	}

	// Get required parameters
	params := bt.Parameters()
	if params == nil {
		return nil
	}

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		return nil
	}

	required, ok := params["required"].([]string)
	if !ok {
		return nil
	}

	// Check required parameters
	for _, reqParam := range required {
		if _, exists := input[reqParam]; !exists {
			return fmt.Errorf("required parameter '%s' is missing", reqParam)
		}
	}

	// Check parameter types (basic validation)
	for paramName, paramDef := range properties {
		if value, exists := input[paramName]; exists {
			if err := bt.validateParameterType(paramName, value, paramDef); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateParameterType validates a parameter type
func (bt *BaseTool) validateParameterType(name string, value interface{}, definition interface{}) error {
	def, ok := definition.(map[string]interface{})
	if !ok {
		return nil // Skip validation if definition is not a map
	}

	expectedType, ok := def["type"].(string)
	if !ok {
		return nil // Skip validation if type is not specified
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter '%s' must be a string", name)
		}
	case "integer":
		switch value.(type) {
		case int, int64, float64:
			// Accept various numeric types
		default:
			return fmt.Errorf("parameter '%s' must be an integer", name)
		}
	case "number":
		switch value.(type) {
		case int, int64, float64:
			// Accept various numeric types
		default:
			return fmt.Errorf("parameter '%s' must be a number", name)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter '%s' must be a boolean", name)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("parameter '%s' must be an array", name)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("parameter '%s' must be an object", name)
		}
	}

	return nil
}

// CreateSuccessResult creates a successful tool result
func (bt *BaseTool) CreateSuccessResult(data interface{}, metadata map[string]interface{}) *ToolResult {
	return &ToolResult{
		Success:  true,
		Data:     data,
		Metadata: metadata,
	}
}

// CreateErrorResult creates an error tool result
func (bt *BaseTool) CreateErrorResult(err error, metadata map[string]interface{}) *ToolResult {
	return &ToolResult{
		Success:  false,
		Error:    err.Error(),
		Metadata: metadata,
	}
}

// MeasureExecution measures execution time and creates a result
func (bt *BaseTool) MeasureExecution(fn func() (*ToolResult, error)) (*ToolResult, error) {
	start := time.Now()
	result, err := fn()
	duration := time.Since(start)

	if result != nil {
		result.ExecutionTime = duration
	}

	return result, err
}

// ToolError represents a tool-specific error
type ToolError struct {
	Tool    string `json:"tool"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

// Error implements the error interface
func (e *ToolError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s [%s]: %s: %v", e.Tool, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s [%s]: %s", e.Tool, e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *ToolError) Unwrap() error {
	return e.Cause
}

// NewToolError creates a new tool error
func NewToolError(tool, code, message string, cause error) *ToolError {
	return &ToolError{
		Tool:    tool,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Common error codes
const (
	ErrorCodeInvalidInput         = "INVALID_INPUT"
	ErrorCodeConnectionFailed     = "CONNECTION_FAILED"
	ErrorCodeAuthenticationFailed = "AUTHENTICATION_FAILED"
	ErrorCodePermissionDenied     = "PERMISSION_DENIED"
	ErrorCodeResourceNotFound     = "RESOURCE_NOT_FOUND"
	ErrorCodeTimeout              = "TIMEOUT"
	ErrorCodeInternalError        = "INTERNAL_ERROR"
	ErrorCodeConfigurationError   = "CONFIGURATION_ERROR"
)
