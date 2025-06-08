package tool

import (
	"context"
	"time"
)

// Parameter type constants
const (
	ParameterTypeString  = "string"
	ParameterTypeNumber  = "number"
	ParameterTypeInteger = "integer"
	ParameterTypeBoolean = "boolean"
	ParameterTypeArray   = "array"
	ParameterTypeObject  = "object"
)

// ToolParametersSchema represents the schema for tool parameters
type ToolParametersSchema struct {
	Type        string                       `json:"type"` // Usually "object"
	Properties  map[string]ParameterProperty `json:"properties"`
	Required    []string                     `json:"required,omitempty"`
	Description string                       `json:"description,omitempty"`
}

// ParameterProperty represents a single parameter property
type ParameterProperty struct {
	Type        string      `json:"type"` // "string", "number", "boolean", "array", "object"
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Format      string      `json:"format,omitempty"` // e.g., "date-time", "email"
	MinLength   *int        `json:"minLength,omitempty"`
	MaxLength   *int        `json:"maxLength,omitempty"`
	Minimum     *float64    `json:"minimum,omitempty"`
	Maximum     *float64    `json:"maximum,omitempty"`
}

// ToolInput represents the input to a tool execution
type ToolInput struct {
	// Parameters contains the actual tool parameters
	Parameters map[string]interface{} `json:"parameters"`

	// Context provides execution context
	Context *ToolContext `json:"context,omitempty"`

	// Config provides tool-specific configuration
	Config *ToolConfig `json:"config,omitempty"`
}

// AssistantToolInfo represents tool information from the assistant
type AssistantToolInfo struct {
	Name        string
	Description string
	Category    string
	Version     string
	Author      string
	IsEnabled   bool
}

// AssistantToolInterface defines the interface that HTTPHandler needs from assistant
// This breaks the circular dependency with the assistant package
type AssistantToolInterface interface {
	GetAvailableTools() []AssistantToolInfo
	ExecuteTool(ctx context.Context, req *struct {
		ToolName string
		Input    map[string]interface{}
		Config   map[string]interface{}
	}) (*struct {
		Success     bool
		Result      interface{}
		Error       string
		ToolsUsed   []string
		ElapsedTime time.Duration
		Metadata    map[string]interface{}
	}, error)
}

// ToolContext provides context for tool execution
type ToolContext struct {
	UserID         string            `json:"user_id,omitempty"`
	SessionID      string            `json:"session_id,omitempty"`
	ConversationID string            `json:"conversation_id,omitempty"`
	RequestID      string            `json:"request_id,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// ToolConfig represents tool-specific configuration
type ToolConfig struct {
	// Timeout for tool execution
	Timeout time.Duration `json:"timeout,omitempty"`

	// MaxRetries for failed executions
	MaxRetries int `json:"max_retries,omitempty"`

	// RetryDelay between retries
	RetryDelay time.Duration `json:"retry_delay,omitempty"`

	// Custom configuration for specific tools
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// ToolResultData represents the data returned by a tool
type ToolResultData struct {
	// The actual result data - structure depends on the tool
	Result interface{} `json:"result"`

	// Optional structured output
	Output map[string]interface{} `json:"output,omitempty"`

	// Any files or artifacts produced
	Artifacts []ToolArtifact `json:"artifacts,omitempty"`
}

// ToolArtifact represents a file or artifact produced by a tool
type ToolArtifact struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "file", "image", "code", etc.
	Content     []byte `json:"content,omitempty"`
	Path        string `json:"path,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

// ToolMetadata contains metadata about tool execution
type ToolMetadata struct {
	// Execution timestamps
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	// Resource usage
	CPUTime    time.Duration `json:"cpu_time,omitempty"`
	MemoryUsed int64         `json:"memory_used,omitempty"`

	// Tool-specific metadata
	Custom map[string]interface{} `json:"custom,omitempty"`

	// Warnings or notes
	Warnings []string `json:"warnings,omitempty"`

	// Debug information
	Debug map[string]interface{} `json:"debug,omitempty"`
}

// RegistryStats represents statistics about the tool registry
type RegistryStats struct {
	TotalTools      int                   `json:"total_tools"`
	EnabledTools    int                   `json:"enabled_tools"`
	DisabledTools   int                   `json:"disabled_tools"`
	ToolsByCategory map[string]int        `json:"tools_by_category"`
	ExecutionStats  map[string]*ToolStats `json:"execution_stats"`
	LastHealthCheck time.Time             `json:"last_health_check"`
	HealthyTools    int                   `json:"healthy_tools"`
	UnhealthyTools  int                   `json:"unhealthy_tools"`
}

// ToolStats represents statistics for a specific tool
type ToolStats struct {
	TotalExecutions   int64         `json:"total_executions"`
	SuccessfulRuns    int64         `json:"successful_runs"`
	FailedRuns        int64         `json:"failed_runs"`
	AverageRunTime    time.Duration `json:"average_run_time"`
	LastExecutionTime time.Time     `json:"last_execution_time"`
	ErrorRate         float64       `json:"error_rate"`
	IsHealthy         bool          `json:"is_healthy"`
}

// ToolRegistration represents a tool registration request
type ToolRegistration struct {
	Name       string      `json:"name"`
	Factory    ToolFactory `json:"-"`
	Info       ToolInfo    `json:"info"`
	Config     *ToolConfig `json:"config,omitempty"`
	AutoEnable bool        `json:"auto_enable"`
}

// ToolValidator interface for validating tool inputs
type ToolValidator interface {
	// ValidateInput validates the input against the tool's parameter schema
	ValidateInput(tool Tool, input *ToolInput) error

	// ValidateOutput validates the tool's output
	ValidateOutput(tool Tool, result *ToolResult) error
}

// ToolMiddleware interface for tool execution middleware
type ToolMiddleware interface {
	// Before is called before tool execution
	Before(ctx context.Context, tool Tool, input *ToolInput) error

	// After is called after tool execution
	After(ctx context.Context, tool Tool, input *ToolInput, result *ToolResult, err error) error
}

// ToolHook represents a hook that can be registered for tool events
type ToolHook func(event ToolEvent)

// ToolEvent represents an event in the tool lifecycle
type ToolEvent struct {
	Type      string      `json:"type"` // "registered", "executed", "failed", etc.
	ToolName  string      `json:"tool_name"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
	Error     error       `json:"error,omitempty"`
}

// ToolCategory represents a category of tools
type ToolCategory struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
}

// ToolParameter represents a parameter for legacy compatibility
type ToolParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// ConvertLegacyConfig converts a legacy map[string]interface{} config to ToolConfig
func ConvertLegacyConfig(legacy map[string]interface{}) *ToolConfig {
	if legacy == nil {
		return nil
	}

	config := &ToolConfig{
		Custom: make(map[string]interface{}),
	}

	// Extract known fields
	if timeout, ok := legacy["timeout"].(time.Duration); ok {
		config.Timeout = timeout
		delete(legacy, "timeout")
	} else if timeoutStr, ok := legacy["timeout"].(string); ok {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			config.Timeout = d
			delete(legacy, "timeout")
		}
	}

	if maxRetries, ok := legacy["max_retries"].(int); ok {
		config.MaxRetries = maxRetries
		delete(legacy, "max_retries")
	}

	if retryDelay, ok := legacy["retry_delay"].(time.Duration); ok {
		config.RetryDelay = retryDelay
		delete(legacy, "retry_delay")
	} else if delayStr, ok := legacy["retry_delay"].(string); ok {
		if d, err := time.ParseDuration(delayStr); err == nil {
			config.RetryDelay = d
			delete(legacy, "retry_delay")
		}
	}

	// Copy remaining fields to custom
	for k, v := range legacy {
		config.Custom[k] = v
	}

	return config
}

// ConvertLegacyInput converts a legacy map[string]interface{} input to ToolInput
func ConvertLegacyInput(legacy map[string]interface{}) *ToolInput {
	if legacy == nil {
		return &ToolInput{
			Parameters: make(map[string]interface{}),
		}
	}

	input := &ToolInput{
		Parameters: make(map[string]interface{}),
	}

	// Extract context if present
	if ctx, ok := legacy["context"].(map[string]interface{}); ok {
		input.Context = &ToolContext{
			Metadata: make(map[string]string),
		}

		if userID, ok := ctx["user_id"].(string); ok {
			input.Context.UserID = userID
		}
		if sessionID, ok := ctx["session_id"].(string); ok {
			input.Context.SessionID = sessionID
		}
		if conversationID, ok := ctx["conversation_id"].(string); ok {
			input.Context.ConversationID = conversationID
		}
		if requestID, ok := ctx["request_id"].(string); ok {
			input.Context.RequestID = requestID
		}
		if metadata, ok := ctx["metadata"].(map[string]string); ok {
			input.Context.Metadata = metadata
		}

		delete(legacy, "context")
	}

	// Extract config if present
	if cfg, ok := legacy["config"].(map[string]interface{}); ok {
		input.Config = ConvertLegacyConfig(cfg)
		delete(legacy, "config")
	}

	// Everything else goes to parameters
	input.Parameters = legacy

	return input
}
