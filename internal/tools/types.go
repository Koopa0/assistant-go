package tools

import (
	"time"
)

// ToolParameterType represents the type of a tool parameter
type ToolParameterType string

const (
	ParameterTypeString  ToolParameterType = "string"
	ParameterTypeInteger ToolParameterType = "integer"
	ParameterTypeNumber  ToolParameterType = "number"
	ParameterTypeBoolean ToolParameterType = "boolean"
	ParameterTypeArray   ToolParameterType = "array"
	ParameterTypeObject  ToolParameterType = "object"
)

// ToolParameter represents a single parameter definition for a tool
type ToolParameter struct {
	Type        ToolParameterType `json:"type"`
	Description string            `json:"description"`
	Required    bool              `json:"required"`
	Default     interface{}       `json:"default,omitempty"` // Keep as interface{} for JSON compatibility
	Enum        []string          `json:"enum,omitempty"`
	Pattern     string            `json:"pattern,omitempty"`
	MinLength   *int              `json:"min_length,omitempty"`
	MaxLength   *int              `json:"max_length,omitempty"`
	Minimum     *float64          `json:"minimum,omitempty"`
	Maximum     *float64          `json:"maximum,omitempty"`
}

// ToolParametersSchema represents the schema for tool parameters
type ToolParametersSchema struct {
	Type       string                   `json:"type"`
	Properties map[string]ToolParameter `json:"properties"`
	Required   []string                 `json:"required"`
}

// ToolConfig represents typed configuration for tools
type ToolConfig struct {
	// Common configuration
	Timeout    time.Duration `json:"timeout,omitempty"`
	MaxRetries int           `json:"max_retries,omitempty"`
	Debug      bool          `json:"debug,omitempty"`

	// Tool-specific configuration
	WorkingDir  string            `json:"working_dir,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`

	// Advanced options
	AllowUnsafe  bool              `json:"allow_unsafe,omitempty"`
	CacheResults bool              `json:"cache_results,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ToolInput represents strongly typed tool input parameters
type ToolInput struct {
	// Common input parameters
	TaskDescription string       `json:"task_description,omitempty"`
	Context         string       `json:"context,omitempty"`
	Options         *ToolOptions `json:"options,omitempty"`

	// Tool-specific parameters stored as typed values
	Parameters map[string]interface{} `json:"parameters"` // Keep for JSON compatibility

	// Metadata
	RequestID     string   `json:"request_id,omitempty"`
	UserID        string   `json:"user_id,omitempty"`
	CorrelationID string   `json:"correlation_id,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

// ToolOptions represents execution options for tools
type ToolOptions struct {
	Timeout        time.Duration `json:"timeout,omitempty"`
	Async          bool          `json:"async,omitempty"`
	RetryCount     int           `json:"retry_count,omitempty"`
	CacheEnabled   bool          `json:"cache_enabled,omitempty"`
	ValidateInput  bool          `json:"validate_input,omitempty"`
	ValidateOutput bool          `json:"validate_output,omitempty"`
}

// ToolResultData represents strongly typed result data from tools
type ToolResultData struct {
	// Standard result fields
	Output   string `json:"output,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	FilePath string `json:"file_path,omitempty"`
	Content  string `json:"content,omitempty"`

	// Analysis results
	Analysis    *AnalysisResult `json:"analysis,omitempty"`
	Suggestions []string        `json:"suggestions,omitempty"`
	Warnings    []string        `json:"warnings,omitempty"`

	// Metrics and performance
	LinesProcessed int64 `json:"lines_processed,omitempty"`
	BytesProcessed int64 `json:"bytes_processed,omitempty"`

	// Build results
	Build *BuildResult `json:"build,omitempty"`

	// Additional structured data
	Results   []ResultItem   `json:"results,omitempty"`
	Artifacts []ArtifactInfo `json:"artifacts,omitempty"`
}

// AnalysisResult represents code analysis results
type AnalysisResult struct {
	Issues       []Issue       `json:"issues,omitempty"`
	Metrics      *CodeMetrics  `json:"metrics,omitempty"`
	Dependencies []Dependency  `json:"dependencies,omitempty"`
	TestCoverage *TestCoverage `json:"test_coverage,omitempty"`
}

// Issue represents a code issue found during analysis
type Issue struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Message     string   `json:"message"`
	File        string   `json:"file,omitempty"`
	Line        int      `json:"line,omitempty"`
	Column      int      `json:"column,omitempty"`
	Rule        string   `json:"rule,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// CodeMetrics represents code quality metrics
type CodeMetrics struct {
	LinesOfCode          int     `json:"lines_of_code"`
	CyclomaticComplexity int     `json:"cyclomatic_complexity"`
	CognitiveComplexity  int     `json:"cognitive_complexity"`
	TestCoveragePercent  float64 `json:"test_coverage_percent"`
	DuplicationPercent   float64 `json:"duplication_percent"`
	TechnicalDebt        string  `json:"technical_debt,omitempty"`
}

// Dependency represents a code dependency
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`
	Source  string `json:"source,omitempty"`
}

// TestCoverage represents test coverage information
type TestCoverage struct {
	LinesTotal   int             `json:"lines_total"`
	LinesCovered int             `json:"lines_covered"`
	Percentage   float64         `json:"percentage"`
	Branches     *BranchCoverage `json:"branches,omitempty"`
}

// BranchCoverage represents branch coverage details
type BranchCoverage struct {
	Total      int     `json:"total"`
	Covered    int     `json:"covered"`
	Percentage float64 `json:"percentage"`
}

// BuildResult represents the result of a build operation
type BuildResult struct {
	Success    bool                   `json:"success"`
	BinaryPath string                 `json:"binary_path,omitempty"`
	BinarySize int64                  `json:"binary_size,omitempty"`
	BuildTime  time.Duration          `json:"build_time"`
	Output     string                 `json:"output,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ResultItem represents a generic result item
type ResultItem struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Value       string            `json:"value,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ArtifactInfo represents information about generated artifacts
type ArtifactInfo struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Path      string            `json:"path"`
	Size      int64             `json:"size,omitempty"`
	Checksum  string            `json:"checksum,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// ToolMetadata represents metadata about a tool execution
type ToolMetadata struct {
	// Execution context
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	ExecutionTime time.Duration `json:"execution_time"`

	// System information
	Hostname     string `json:"hostname,omitempty"`
	Platform     string `json:"platform,omitempty"`
	Architecture string `json:"architecture,omitempty"`

	// Resource usage
	CPUUsage    float64 `json:"cpu_usage,omitempty"`
	MemoryUsage int64   `json:"memory_usage,omitempty"`
	DiskUsage   int64   `json:"disk_usage,omitempty"`

	// Tool specific metadata
	ToolVersion string            `json:"tool_version,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`

	// Error tracking
	RetryCount int      `json:"retry_count,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`

	// Tracing
	TraceID string `json:"trace_id,omitempty"`
	SpanID  string `json:"span_id,omitempty"`
}

// ToolStats represents statistics for tool usage
type ToolStats struct {
	Name                string        `json:"name"`
	ExecutionCount      int64         `json:"execution_count"`
	SuccessCount        int64         `json:"success_count"`
	ErrorCount          int64         `json:"error_count"`
	SuccessRate         float64       `json:"success_rate"`
	AverageTime         time.Duration `json:"average_time"`
	MinTime             time.Duration `json:"min_time"`
	MaxTime             time.Duration `json:"max_time"`
	LastExecuted        time.Time     `json:"last_executed,omitempty"`
	LastError           string        `json:"last_error,omitempty"`
	TotalProcessingTime time.Duration `json:"total_processing_time"`
}

// RegistryStats represents overall registry statistics
type RegistryStats struct {
	RegisteredTools    int                   `json:"registered_tools"`
	ActiveTools        int                   `json:"active_tools"`
	TotalExecutions    int64                 `json:"total_executions"`
	TotalSuccesses     int64                 `json:"total_successes"`
	TotalErrors        int64                 `json:"total_errors"`
	OverallSuccessRate float64               `json:"overall_success_rate"`
	ToolStats          map[string]*ToolStats `json:"tool_stats,omitempty"`
	Uptime             time.Duration         `json:"uptime"`
	LastReset          time.Time             `json:"last_reset,omitempty"`
}

// ConvertLegacyParameters converts legacy map[string]interface{} to ToolParametersSchema
func ConvertLegacyParameters(legacy map[string]interface{}) *ToolParametersSchema {
	if legacy == nil {
		return &ToolParametersSchema{
			Type:       "object",
			Properties: make(map[string]ToolParameter),
			Required:   []string{},
		}
	}

	schema := &ToolParametersSchema{
		Type:       "object",
		Properties: make(map[string]ToolParameter),
		Required:   []string{},
	}

	for name, paramData := range legacy {
		if paramMap, ok := paramData.(map[string]interface{}); ok {
			param := ToolParameter{
				Type:     ParameterTypeString, // Default type
				Required: false,
			}

			if typeVal, ok := paramMap["type"].(string); ok {
				param.Type = ToolParameterType(typeVal)
			}
			if desc, ok := paramMap["description"].(string); ok {
				param.Description = desc
			}
			if req, ok := paramMap["required"].(bool); ok {
				param.Required = req
				if req {
					schema.Required = append(schema.Required, name)
				}
			}
			if defaultVal, ok := paramMap["default"]; ok {
				param.Default = defaultVal
			}

			schema.Properties[name] = param
		}
	}

	return schema
}

// ConvertLegacyConfig converts legacy map[string]interface{} to ToolConfig
func ConvertLegacyConfig(legacy map[string]interface{}) *ToolConfig {
	if legacy == nil {
		return &ToolConfig{}
	}

	config := &ToolConfig{
		Metadata:    make(map[string]string),
		Environment: make(map[string]string),
	}

	if timeout, ok := legacy["timeout"].(time.Duration); ok {
		config.Timeout = timeout
	}
	if retries, ok := legacy["max_retries"].(int); ok {
		config.MaxRetries = retries
	}
	if debug, ok := legacy["debug"].(bool); ok {
		config.Debug = debug
	}
	if workingDir, ok := legacy["working_dir"].(string); ok {
		config.WorkingDir = workingDir
	}

	return config
}

// ConvertLegacyInput converts legacy map[string]interface{} to ToolInput
func ConvertLegacyInput(legacy map[string]interface{}) *ToolInput {
	if legacy == nil {
		return &ToolInput{
			Parameters: make(map[string]interface{}),
		}
	}

	input := &ToolInput{
		Parameters: make(map[string]interface{}),
		Tags:       []string{},
	}

	// Copy known fields
	if desc, ok := legacy["task_description"].(string); ok {
		input.TaskDescription = desc
	}
	if ctx, ok := legacy["context"].(string); ok {
		input.Context = ctx
	}
	if reqID, ok := legacy["request_id"].(string); ok {
		input.RequestID = reqID
	}
	if userID, ok := legacy["user_id"].(string); ok {
		input.UserID = userID
	}

	// Copy all other parameters
	for key, value := range legacy {
		switch key {
		case "task_description", "context", "request_id", "user_id":
			// Already handled above
		default:
			input.Parameters[key] = value
		}
	}

	return input
}
