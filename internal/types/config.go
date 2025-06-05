// Package types provides strongly typed structures to replace interface{} usage
package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// ToolConfig represents configuration for tools
// Replaces map[string]interface{} with type-safe structure
type ToolConfig struct {
	Timeout       time.Duration     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	MaxRetries    int               `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
	EnableCache   bool              `json:"enable_cache,omitempty" yaml:"enable_cache,omitempty"`
	EnableLogging bool              `json:"enable_logging,omitempty" yaml:"enable_logging,omitempty"`
	WorkingDir    string            `json:"working_dir,omitempty" yaml:"working_dir,omitempty"`
	Environment   map[string]string `json:"environment,omitempty" yaml:"environment,omitempty"`

	// Tool-specific configurations
	GoConfig       *GoToolConfig   `json:"go,omitempty" yaml:"go,omitempty"`
	DatabaseConfig *DatabaseConfig `json:"database,omitempty" yaml:"database,omitempty"`
	K8sConfig      *K8sConfig      `json:"k8s,omitempty" yaml:"k8s,omitempty"`
}

// GoToolConfig contains Go-specific tool configuration
type GoToolConfig struct {
	Version       string   `json:"version,omitempty" yaml:"version,omitempty"`
	BuildTags     []string `json:"build_tags,omitempty" yaml:"build_tags,omitempty"`
	EnableModules bool     `json:"enable_modules,omitempty" yaml:"enable_modules,omitempty"`
	GOOS          string   `json:"goos,omitempty" yaml:"goos,omitempty"`
	GOARCH        string   `json:"goarch,omitempty" yaml:"goarch,omitempty"`
}

// DatabaseConfig contains database-specific configuration
type DatabaseConfig struct {
	ConnectionTimeout time.Duration `json:"connection_timeout,omitempty" yaml:"connection_timeout,omitempty"`
	QueryTimeout      time.Duration `json:"query_timeout,omitempty" yaml:"query_timeout,omitempty"`
	MaxConnections    int           `json:"max_connections,omitempty" yaml:"max_connections,omitempty"`
	EnableSSL         bool          `json:"enable_ssl,omitempty" yaml:"enable_ssl,omitempty"`
}

// K8sConfig contains Kubernetes-specific configuration
type K8sConfig struct {
	Namespace   string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Context     string `json:"context,omitempty" yaml:"context,omitempty"`
	ConfigPath  string `json:"config_path,omitempty" yaml:"config_path,omitempty"`
	EnableWatch bool   `json:"enable_watch,omitempty" yaml:"enable_watch,omitempty"`
}

// ToolInput represents typed input for tool execution
// Replaces map[string]interface{} for tool inputs
type ToolInput struct {
	// Common fields
	Action  string            `json:"action,omitempty"`
	Target  string            `json:"target,omitempty"`
	Options map[string]string `json:"options,omitempty"`

	// Specific input types
	Code    *CodeInput    `json:"code,omitempty"`
	File    *FileInput    `json:"file,omitempty"`
	Query   *QueryInput   `json:"query,omitempty"`
	Command *CommandInput `json:"command,omitempty"`
}

// CodeInput represents code-related input
type CodeInput struct {
	Language    string `json:"language"`
	Content     string `json:"content"`
	FilePath    string `json:"file_path,omitempty"`
	PackageName string `json:"package_name,omitempty"`
}

// FileInput represents file-related input
type FileInput struct {
	Path      string `json:"path"`
	Content   string `json:"content,omitempty"`
	Operation string `json:"operation"` // read, write, append, delete
	Mode      string `json:"mode,omitempty"`
}

// QueryInput represents database query input
type QueryInput struct {
	SQL        string            `json:"sql"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Database   string            `json:"database,omitempty"`
	Timeout    time.Duration     `json:"timeout,omitempty"`
}

// CommandInput represents command execution input
type CommandInput struct {
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	WorkingDir  string            `json:"working_dir,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
}

// ToolOutput represents typed output from tool execution
// Replaces map[string]interface{} for tool outputs
type ToolOutput struct {
	// Standard fields
	Success  bool              `json:"success"`
	Message  string            `json:"message,omitempty"`
	Error    string            `json:"error,omitempty"`
	Duration time.Duration     `json:"duration"`
	Metadata map[string]string `json:"metadata,omitempty"`

	// Specific output types
	Code     *CodeOutput     `json:"code,omitempty"`
	File     *FileOutput     `json:"file,omitempty"`
	Query    *QueryOutput    `json:"query,omitempty"`
	Command  *CommandOutput  `json:"command,omitempty"`
	Analysis *AnalysisOutput `json:"analysis,omitempty"`
}

// CodeOutput represents code-related output
type CodeOutput struct {
	FormattedCode string      `json:"formatted_code,omitempty"`
	Issues        []CodeIssue `json:"issues,omitempty"`
	Metrics       CodeMetrics `json:"metrics"`
	Suggestions   []string    `json:"suggestions,omitempty"`
}

// CodeIssue represents a code issue found during analysis
type CodeIssue struct {
	Type     string `json:"type"` // warning, error, info
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity"` // low, medium, high, critical
}

// CodeMetrics represents code metrics
type CodeMetrics struct {
	LinesOfCode          int     `json:"lines_of_code"`
	CyclomaticComplexity int     `json:"cyclomatic_complexity"`
	TestCoverage         float64 `json:"test_coverage,omitempty"`
	FunctionCount        int     `json:"function_count"`
	TypeCount            int     `json:"type_count"`
}

// FileOutput represents file operation output
type FileOutput struct {
	Path     string `json:"path"`
	Size     int64  `json:"size,omitempty"`
	Modified bool   `json:"modified"`
	Checksum string `json:"checksum,omitempty"`
}

// QueryOutput represents database query output
type QueryOutput struct {
	RowsAffected  int64                    `json:"rows_affected"`
	Results       []map[string]interface{} `json:"results,omitempty"`
	Schema        []ColumnInfo             `json:"schema,omitempty"`
	ExecutionTime time.Duration            `json:"execution_time"`
}

// ColumnInfo represents database column information
type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// CommandOutput represents command execution output
type CommandOutput struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Signal   string `json:"signal,omitempty"`
}

// AnalysisOutput represents analysis results
type AnalysisOutput struct {
	Summary     string                 `json:"summary"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Confidence  float64                `json:"confidence"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// Context represents execution context
// Replaces map[string]interface{} for context
type Context struct {
	UserID         string `json:"user_id,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	RequestID      string `json:"request_id,omitempty"`

	// Preferences and settings
	Preferences UserPreferences   `json:"preferences,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`

	// Execution context
	WorkingDir string        `json:"working_dir,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`

	// Custom fields for specific use cases
	Custom map[string]string `json:"custom,omitempty"`
}

// UserPreferences represents user preferences
type UserPreferences struct {
	Language    string  `json:"language,omitempty"`
	Theme       string  `json:"theme,omitempty"`
	Verbosity   string  `json:"verbosity,omitempty"` // minimal, normal, verbose
	ExplainCode bool    `json:"explain_code,omitempty"`
	ShowMetrics bool    `json:"show_metrics,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// Metadata represents generic metadata
// When map[string]interface{} is truly needed, use this instead
type Metadata map[string]string

// ToJSON converts the structure to JSON for storage
func (tc ToolConfig) ToJSON() ([]byte, error) {
	return json.Marshal(tc)
}

// FromJSON loads configuration from JSON
func (tc *ToolConfig) FromJSON(data []byte) error {
	return json.Unmarshal(data, tc)
}

// Validate checks if the configuration is valid
func (tc ToolConfig) Validate() error {
	if tc.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}
	if tc.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}
	return nil
}

// SetDefaults sets default values for the configuration
func (tc *ToolConfig) SetDefaults() {
	if tc.Timeout == 0 {
		tc.Timeout = 30 * time.Second
	}
	if tc.MaxRetries == 0 {
		tc.MaxRetries = 3
	}
}

// ToolInfo represents information about an available tool
type ToolInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Version     string    `json:"version"`
	Status      string    `json:"status"` // available, maintenance, deprecated
	Usage       ToolUsage `json:"usage"`
}

// ToolUsage represents tool usage statistics
type ToolUsage struct {
	TotalExecutions int32      `json:"total_executions"`
	SuccessfulRuns  int32      `json:"successful_runs"`
	FailedRuns      int32      `json:"failed_runs"`
	AverageExecTime float64    `json:"average_execution_time_ms"`
	LastUsed        *time.Time `json:"last_used,omitempty"`
	PopularityScore float64    `json:"popularity_score"`
}

// APIResponse represents a typed API response structure
// Replaces map[string]interface{} for HTTP responses
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
}

// APIError represents structured error information
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ToolListResponse represents a typed response for tool listing
type ToolListResponse struct {
	Tools []ToolInfo    `json:"tools"`
	Total int           `json:"total"`
	Stats ToolListStats `json:"stats"`
}

// ToolListStats represents statistics for tool listing
type ToolListStats struct {
	ByCategory map[string]int `json:"by_category"`
	ByStatus   map[string]int `json:"by_status"`
}
