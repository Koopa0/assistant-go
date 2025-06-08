// Package tools provides domain-specific error handling for tool operations
// following CLAUDE.md best practices with proper error hierarchy and context.
package tool

import (
	"fmt"
	"time"

	"github.com/koopa0/assistant-go/internal/errors"
)

// Tool-specific error codes
const (
	// Tool registry errors
	CodeToolRegistration   = "TOOL_REGISTRATION"
	CodeToolDeregistration = "TOOL_DEREGISTRATION"
	CodeToolNotRegistered  = "TOOL_NOT_REGISTERED"
	CodeToolAlreadyExists  = "TOOL_ALREADY_EXISTS"
	CodeDuplicateToolName  = "DUPLICATE_TOOL_NAME"

	// Tool execution errors
	CodeToolExecutionTimeout  = "TOOL_EXECUTION_TIMEOUT"
	CodeToolExecutionPanic    = "TOOL_EXECUTION_PANIC"
	CodeToolPermissionDenied  = "TOOL_PERMISSION_DENIED"
	CodeToolDependencyMissing = "TOOL_DEPENDENCY_MISSING"
	CodeToolResourceExhausted = "TOOL_RESOURCE_EXHAUSTED"

	// Tool validation errors
	CodeInvalidToolConfig    = "INVALID_TOOL_CONFIG"
	CodeInvalidToolInput     = "INVALID_TOOL_INPUT"
	CodeMissingRequiredParam = "MISSING_REQUIRED_PARAM"
	CodeInvalidParamType     = "INVALID_PARAM_TYPE"
	CodeInvalidParamValue    = "INVALID_PARAM_VALUE"

	// Tool output errors
	CodeToolOutputTooLarge            = "TOOL_OUTPUT_TOO_LARGE"
	CodeToolOutputInvalid             = "TOOL_OUTPUT_INVALID"
	CodeToolOutputSerializationFailed = "TOOL_OUTPUT_SERIALIZATION_FAILED"

	// Pipeline errors
	CodePipelineCreation           = "PIPELINE_CREATION"
	CodePipelineExecution          = "PIPELINE_EXECUTION"
	CodePipelineValidation         = "PIPELINE_VALIDATION"
	CodePipelineCircularDependency = "PIPELINE_CIRCULAR_DEPENDENCY"
)

// Tool Registry Error Constructors

// NewToolRegistrationError creates a tool registration error
func NewToolRegistrationError(toolName string, cause error) *errors.AssistantError {
	return errors.NewBusinessError(CodeToolRegistration, "tool registration failed", cause).
		WithComponent("tools").
		WithOperation("register").
		WithContext("tool_name", toolName).
		WithUserMessage("Failed to register tool. Please try again.").
		WithActions("Check tool implementation", "Verify tool metadata", "Review tool dependencies")
}

// NewToolNotRegisteredError creates a tool not registered error
func NewToolNotRegisteredError(toolName string) *errors.AssistantError {
	return errors.NewBusinessError(CodeToolNotRegistered, "tool not registered", nil).
		WithComponent("tools").
		WithOperation("lookup").
		WithContext("tool_name", toolName).
		WithUserMessage(fmt.Sprintf("Tool '%s' is not available.", toolName)).
		WithActions("Check tool name spelling", "Verify tool is registered", "List available tools")
}

// NewToolAlreadyExistsError creates a tool already exists error
func NewToolAlreadyExistsError(toolName string) *errors.AssistantError {
	return errors.NewValidationError(CodeToolAlreadyExists, "tool already exists", nil).
		WithComponent("tools").
		WithOperation("register").
		WithContext("tool_name", toolName).
		WithUserMessage(fmt.Sprintf("Tool '%s' is already registered.", toolName)).
		WithActions("Use different tool name", "Unregister existing tool first", "Update existing tool")
}

// NewDuplicateToolNameError creates a duplicate tool name error
func NewDuplicateToolNameError(toolName string, existingTool string) *errors.AssistantError {
	return errors.NewValidationError(CodeDuplicateToolName, "duplicate tool name", nil).
		WithComponent("tools").
		WithOperation("register").
		WithContext("tool_name", toolName).
		WithContext("existing_tool", existingTool).
		WithUserMessage(fmt.Sprintf("Tool name '%s' conflicts with existing tool.", toolName)).
		WithActions("Choose unique tool name", "Use namespaced name", "Remove conflicting tool")
}

// Tool Execution Error Constructors

// NewToolExecutionTimeoutError creates a tool execution timeout error
func NewToolExecutionTimeoutError(toolName string, timeout time.Duration) *errors.AssistantError {
	return errors.NewBusinessError(CodeToolExecutionTimeout, "tool execution timed out", nil).
		WithComponent("tools").
		WithOperation("execute").
		WithContext("tool_name", toolName).
		WithContext("timeout", timeout.String()).
		WithDuration(timeout).
		WithUserMessage("Tool execution timed out. Please try again.").
		WithActions("Increase timeout", "Optimize tool implementation", "Use simpler parameters").
		WithRetryable(true)
}

// NewToolExecutionPanicError creates a tool execution panic error
func NewToolExecutionPanicError(toolName string, panicValue interface{}, stackTrace string) *errors.AssistantError {
	return errors.NewBusinessError(CodeToolExecutionPanic, "tool execution panicked", nil).
		WithComponent("tools").
		WithOperation("execute").
		WithContext("tool_name", toolName).
		WithContext("panic_value", fmt.Sprintf("%v", panicValue)).
		WithContext("stack_trace", stackTrace).
		WithUserMessage("Tool execution failed unexpectedly. Please try again.").
		WithActions("Check tool implementation", "Report bug", "Use alternative tool").
		WithSeverity(errors.SeverityHigh)
}

// NewToolPermissionDeniedError creates a permission denied error
func NewToolPermissionDeniedError(toolName, operation, resource string) *errors.AssistantError {
	return errors.NewValidationError(CodeToolPermissionDenied, "tool permission denied", nil).
		WithComponent("tools").
		WithOperation("execute").
		WithContext("tool_name", toolName).
		WithContext("denied_operation", operation).
		WithContext("resource", resource).
		WithUserMessage("Permission denied for tool operation.").
		WithActions("Check tool permissions", "Request access", "Use alternative approach")
}

// NewToolDependencyMissingError creates a dependency missing error
func NewToolDependencyMissingError(toolName, dependency string) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeToolDependencyMissing, "tool dependency missing", nil).
		WithComponent("tools").
		WithOperation("execute").
		WithContext("tool_name", toolName).
		WithContext("missing_dependency", dependency).
		WithUserMessage("Required tool dependency is missing.").
		WithActions("Install dependency", "Check tool requirements", "Use alternative tool").
		WithSeverity(errors.SeverityHigh)
}

// NewToolResourceExhaustedError creates a resource exhausted error
func NewToolResourceExhaustedError(toolName, resource string, limit interface{}) *errors.AssistantError {
	return errors.NewInfrastructureError(CodeToolResourceExhausted, "tool resource exhausted", nil).
		WithComponent("tools").
		WithOperation("execute").
		WithContext("tool_name", toolName).
		WithContext("resource", resource).
		WithContext("limit", limit).
		WithUserMessage("Tool resource limit exceeded.").
		WithActions("Reduce resource usage", "Wait and retry", "Increase resource limits").
		WithRetryAfter(time.Minute * 5)
}

// Tool Validation Error Constructors

// NewInvalidToolConfigError creates an invalid tool configuration error
func NewInvalidToolConfigError(toolName, field, reason string) *errors.AssistantError {
	return errors.NewValidationError(CodeInvalidToolConfig, fmt.Sprintf("invalid tool configuration: %s", reason), nil).
		WithComponent("tools").
		WithOperation("validate").
		WithContext("tool_name", toolName).
		WithContext("invalid_field", field).
		WithContext("reason", reason).
		WithUserMessage("Tool configuration is invalid.").
		WithActions("Fix configuration", "Check configuration format", "Use default configuration")
}

// NewInvalidToolInputError creates an invalid tool input error
func NewInvalidToolInputError(toolName, paramName string, value interface{}, reason string) *errors.AssistantError {
	return errors.NewValidationError(CodeInvalidToolInput, fmt.Sprintf("invalid tool input: %s", reason), nil).
		WithComponent("tools").
		WithOperation("validate_input").
		WithContext("tool_name", toolName).
		WithContext("parameter", paramName).
		WithContext("value", value).
		WithContext("reason", reason).
		WithUserMessage(fmt.Sprintf("Invalid input for parameter '%s': %s", paramName, reason)).
		WithActions("Check parameter format", "Review parameter requirements", "Use valid values")
}

// NewMissingRequiredParamError creates a missing required parameter error
func NewMissingRequiredParamError(toolName, paramName string) *errors.AssistantError {
	return errors.NewValidationError(CodeMissingRequiredParam, fmt.Sprintf("missing required parameter: %s", paramName), nil).
		WithComponent("tools").
		WithOperation("validate_input").
		WithContext("tool_name", toolName).
		WithContext("missing_parameter", paramName).
		WithUserMessage(fmt.Sprintf("Required parameter '%s' is missing.", paramName)).
		WithActions("Provide required parameter", "Check parameter name", "Review tool documentation")
}

// NewInvalidParamTypeError creates an invalid parameter type error
func NewInvalidParamTypeError(toolName, paramName string, expectedType, actualType string) *errors.AssistantError {
	return errors.NewValidationError(CodeInvalidParamType, "invalid parameter type", nil).
		WithComponent("tools").
		WithOperation("validate_input").
		WithContext("tool_name", toolName).
		WithContext("parameter", paramName).
		WithContext("expected_type", expectedType).
		WithContext("actual_type", actualType).
		WithUserMessage(fmt.Sprintf("Parameter '%s' must be %s, got %s.", paramName, expectedType, actualType)).
		WithActions("Convert parameter type", "Check parameter format", "Use correct type")
}

// Tool Output Error Constructors

// NewToolOutputTooLargeError creates an output too large error
func NewToolOutputTooLargeError(toolName string, size, maxSize int64) *errors.AssistantError {
	return errors.NewBusinessError(CodeToolOutputTooLarge, "tool output too large", nil).
		WithComponent("tools").
		WithOperation("execute").
		WithContext("tool_name", toolName).
		WithContext("output_size", size).
		WithContext("max_size", maxSize).
		WithContext("excess_bytes", size-maxSize).
		WithUserMessage("Tool output exceeds size limit.").
		WithActions("Reduce output size", "Paginate results", "Use streaming output")
}

// NewToolOutputInvalidError creates an invalid output error
func NewToolOutputInvalidError(toolName, reason string, output interface{}) *errors.AssistantError {
	return errors.NewBusinessError(CodeToolOutputInvalid, fmt.Sprintf("invalid tool output: %s", reason), nil).
		WithComponent("tools").
		WithOperation("execute").
		WithContext("tool_name", toolName).
		WithContext("reason", reason).
		WithContext("output_type", fmt.Sprintf("%T", output)).
		WithUserMessage("Tool produced invalid output.").
		WithActions("Check tool implementation", "Validate output format", "Report tool bug")
}

// Pipeline Error Constructors

// NewPipelineCreationError creates a pipeline creation error
func NewPipelineCreationError(pipelineName string, cause error) *errors.AssistantError {
	return errors.NewBusinessError(CodePipelineCreation, "pipeline creation failed", cause).
		WithComponent("tools").
		WithOperation("create_pipeline").
		WithContext("pipeline_name", pipelineName).
		WithUserMessage("Failed to create tool pipeline.").
		WithActions("Check pipeline configuration", "Verify tool dependencies", "Simplify pipeline")
}

// NewPipelineExecutionError creates a pipeline execution error
func NewPipelineExecutionError(pipelineName string, failedStep string, cause error) *errors.AssistantError {
	return errors.NewBusinessError(CodePipelineExecution, "pipeline execution failed", cause).
		WithComponent("tools").
		WithOperation("execute_pipeline").
		WithContext("pipeline_name", pipelineName).
		WithContext("failed_step", failedStep).
		WithUserMessage("Pipeline execution failed.").
		WithActions("Check failed step", "Verify step inputs", "Retry pipeline").
		WithRetryable(true)
}

// NewPipelineCircularDependencyError creates a circular dependency error
func NewPipelineCircularDependencyError(pipelineName string, cycle []string) *errors.AssistantError {
	return errors.NewValidationError(CodePipelineCircularDependency, "pipeline has circular dependency", nil).
		WithComponent("tools").
		WithOperation("validate_pipeline").
		WithContext("pipeline_name", pipelineName).
		WithContext("dependency_cycle", cycle).
		WithUserMessage("Pipeline contains circular dependencies.").
		WithActions("Remove circular dependencies", "Reorder pipeline steps", "Simplify dependencies")
}

// Tool-specific error helpers

// IsToolError checks if an error is tool-related
func IsToolError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		return assistantErr.Component == "tools"
	}
	return false
}

// IsToolExecutionError checks if an error is related to tool execution
func IsToolExecutionError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeToolExecutionTimeout, CodeToolExecutionPanic,
			CodeToolPermissionDenied, CodeToolDependencyMissing,
			CodeToolResourceExhausted:
			return true
		}
	}
	return false
}

// IsRetryableToolError checks if a tool error is retryable
func IsRetryableToolError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		return assistantErr.Retryable
	}
	return false
}

// GetToolName extracts tool name from tool errors
func GetToolName(err error) string {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		if toolName, ok := assistantErr.Context["tool_name"].(string); ok {
			return toolName
		}
	}
	return ""
}

// IsValidationError checks if an error is related to tool validation
func IsValidationError(err error) bool {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		switch assistantErr.Code {
		case CodeInvalidToolConfig, CodeInvalidToolInput,
			CodeMissingRequiredParam, CodeInvalidParamType,
			CodeInvalidParamValue:
			return true
		}
	}
	return false
}

// GetFailedParameter extracts parameter name from validation errors
func GetFailedParameter(err error) string {
	if assistantErr := errors.GetAssistantError(err); assistantErr != nil {
		if param, ok := assistantErr.Context["parameter"].(string); ok {
			return param
		}
		if param, ok := assistantErr.Context["missing_parameter"].(string); ok {
			return param
		}
	}
	return ""
}
