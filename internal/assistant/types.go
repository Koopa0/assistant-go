package assistant

import (
	"time"
)

// ProcessorContext represents enriched context data for request processing
// Replaces map[string]interface{} with strongly typed structure
type ProcessorContext struct {
	Workspace *WorkspaceContext `json:"workspace,omitempty"`
	Memory    *MemoryContext    `json:"memory,omitempty"`
	Query     *QueryContext     `json:"query,omitempty"`
	User      *UserContext      `json:"user,omitempty"`
}

// WorkspaceContext contains information about the current workspace
type WorkspaceContext struct {
	ProjectType        string            `json:"project_type"`
	Languages          []string          `json:"languages"`
	Framework          string            `json:"framework,omitempty"`
	Dependencies       []string          `json:"dependencies,omitempty"`
	StructureType      string            `json:"structure_type,omitempty"`
	ConfigFiles        []string          `json:"config_files,omitempty"`
	DocumentationStyle string            `json:"documentation_style,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
}

// MemoryContext contains relevant memory and preferences
type MemoryContext struct {
	UserPreferences *UserPreferences `json:"user_preferences,omitempty"`
	SessionContext  *SessionContext  `json:"session_context,omitempty"`
	WorkingMemory   string           `json:"working_memory,omitempty"`
	RecentTopics    []string         `json:"recent_topics,omitempty"`
}

// UserPreferences represents user's preferences and settings
type UserPreferences struct {
	PreferredLanguage   string `json:"preferred_language"`
	CodeStyle           string `json:"code_style"`
	Documentation       string `json:"documentation"`
	ResponseLanguage    string `json:"response_language"`
	LanguageRestriction string `json:"language_restriction,omitempty"`
}

// SessionContext contains information about the current session
type SessionContext struct {
	PreviousQueries int       `json:"previous_queries"`
	SessionStart    time.Time `json:"session_start"`
	TopicsDiscussed []string  `json:"topics_discussed"`
	LastActivity    time.Time `json:"last_activity,omitempty"`
}

// QueryContext contains analysis of the current query
type QueryContext struct {
	Intent          string            `json:"intent"`
	Category        string            `json:"category,omitempty"`
	Complexity      string            `json:"complexity,omitempty"`
	RequiredTools   []string          `json:"required_tools,omitempty"`
	EstimatedTokens int               `json:"estimated_tokens,omitempty"`
	Keywords        []string          `json:"keywords,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// UserContext contains user-specific information
type UserContext struct {
	UserID      string    `json:"user_id"`
	SessionID   string    `json:"session_id,omitempty"`
	Timezone    string    `json:"timezone,omitempty"`
	Locale      string    `json:"locale,omitempty"`
	Permissions []string  `json:"permissions,omitempty"`
	LastSeen    time.Time `json:"last_seen,omitempty"`
}

// ProcessorStats represents comprehensive processor statistics
// Replaces map[string]interface{} in Stats() function
type ProcessorStats struct {
	Processor     *ProcessorInfo     `json:"processor"`
	AIProviders   *AIProviderStats   `json:"ai_providers,omitempty"`
	Conversations *ConversationStats `json:"conversations,omitempty"`
	Tools         *ToolStats         `json:"tools,omitempty"`
	Performance   *PerformanceStats  `json:"performance,omitempty"`
	Health        *HealthStatus      `json:"health"`
}

// ProcessorInfo contains basic processor information
type ProcessorInfo struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
	Build   string `json:"build,omitempty"`
}

// AIProviderStats contains AI provider usage statistics
type AIProviderStats struct {
	Status    string                    `json:"status"`
	Providers map[string]*ProviderStats `json:"providers,omitempty"`
	Error     string                    `json:"error,omitempty"`
}

// ProviderStats represents statistics for a single AI provider
type ProviderStats struct {
	RequestCount    int64   `json:"request_count"`
	SuccessCount    int64   `json:"success_count"`
	ErrorCount      int64   `json:"error_count"`
	SuccessRate     float64 `json:"success_rate"`
	AvgResponseTime string  `json:"avg_response_time"`
	TokensUsed      int64   `json:"tokens_used"`
	LastUsed        string  `json:"last_used,omitempty"`
}

// ConversationStats contains conversation management statistics
type ConversationStats struct {
	Status        string `json:"status"`
	ActiveCount   int    `json:"active_count,omitempty"`
	TotalCount    int    `json:"total_count,omitempty"`
	AverageLength int    `json:"average_length,omitempty"`
	Note          string `json:"note,omitempty"`
}

// ToolStats contains tool execution statistics
type ToolStats struct {
	Status         string   `json:"status"`
	AvailableCount int      `json:"available_count"`
	ExecutionCount int64    `json:"execution_count"`
	SuccessRate    float64  `json:"success_rate"`
	PopularTools   []string `json:"popular_tools,omitempty"`
	RecentErrors   []string `json:"recent_errors,omitempty"`
}

// PerformanceStats contains system performance metrics
type PerformanceStats struct {
	AverageProcessingTime string  `json:"average_processing_time"`
	MemoryUsage           string  `json:"memory_usage,omitempty"`
	CPUUsage              float64 `json:"cpu_usage,omitempty"`
	RequestsPerSecond     float64 `json:"requests_per_second,omitempty"`
}

// HealthStatus represents overall system health
type HealthStatus struct {
	Status     string            `json:"status"`
	Components map[string]string `json:"components,omitempty"`
	LastCheck  time.Time         `json:"last_check"`
}

// ToolExecutionResult represents the result of tool execution
// Replaces map[string]interface{} in executeTools results
type ToolExecutionResult struct {
	ToolName      string            `json:"tool_name"`
	Status        string            `json:"status"`
	Result        interface{}       `json:"result,omitempty"` // Keep as interface{} for actual tool results
	Error         string            `json:"error,omitempty"`
	ExecutionTime time.Duration     `json:"execution_time"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// ToolExecutionResults represents results from multiple tool executions
type ToolExecutionResults struct {
	Results      []ToolExecutionResult `json:"results"`
	SuccessCount int                   `json:"success_count"`
	ErrorCount   int                   `json:"error_count"`
	TotalTime    time.Duration         `json:"total_time"`
}

// ToolParameters represents typed tool input parameters
// Replaces map[string]interface{} for tool inputs
type ToolParameters struct {
	ToolName   string            `json:"tool_name"`
	Parameters map[string]string `json:"parameters"` // Most tool params are strings
	Options    *ToolOptions      `json:"options,omitempty"`
}

// ToolOptions contains execution options for tools
type ToolOptions struct {
	Timeout     time.Duration `json:"timeout,omitempty"`
	Retries     int           `json:"retries,omitempty"`
	Async       bool          `json:"async,omitempty"`
	CacheResult bool          `json:"cache_result,omitempty"`
}

// MessageContext represents context data for messages
// Replaces map[string]interface{} in message handling
type MessageContext struct {
	ConversationMessageCount int               `json:"conversation_message_count,omitempty"`
	ProcessingSteps          []string          `json:"processing_steps,omitempty"`
	TokensUsed               int               `json:"tokens_used,omitempty"`
	Provider                 string            `json:"provider,omitempty"`
	Model                    string            `json:"model,omitempty"`
	Metadata                 map[string]string `json:"metadata,omitempty"`
}

// ConvertProcessorContextFromMap converts legacy map[string]interface{} to ProcessorContext
// Helper function to ease migration from interface{}
func ConvertProcessorContextFromMap(data map[string]interface{}) *ProcessorContext {
	if data == nil {
		return &ProcessorContext{}
	}

	ctx := &ProcessorContext{}

	// Convert workspace context
	if workspace, ok := data["workspace"].(map[string]interface{}); ok {
		ctx.Workspace = convertWorkspaceContext(workspace)
	}

	// Convert memory context
	if memory, ok := data["memory"].(map[string]interface{}); ok {
		ctx.Memory = convertMemoryContext(memory)
	}

	// Convert query context
	if query, ok := data["query"].(map[string]interface{}); ok {
		ctx.Query = convertQueryContext(query)
	}

	return ctx
}

// Helper conversion functions
func convertWorkspaceContext(data map[string]interface{}) *WorkspaceContext {
	ws := &WorkspaceContext{}

	if projectType, ok := data["project_type"].(string); ok {
		ws.ProjectType = projectType
	}

	if langs, ok := data["languages"].([]string); ok {
		ws.Languages = langs
	}

	if framework, ok := data["framework"].(string); ok {
		ws.Framework = framework
	}

	return ws
}

func convertMemoryContext(data map[string]interface{}) *MemoryContext {
	mem := &MemoryContext{}

	if userPref, ok := data["user_preferences"].(map[string]interface{}); ok {
		mem.UserPreferences = &UserPreferences{}
		if lang, ok := userPref["preferred_language"].(string); ok {
			mem.UserPreferences.PreferredLanguage = lang
		}
		if style, ok := userPref["code_style"].(string); ok {
			mem.UserPreferences.CodeStyle = style
		}
	}

	return mem
}

func convertQueryContext(data map[string]interface{}) *QueryContext {
	query := &QueryContext{}

	if intent, ok := data["intent"].(string); ok {
		query.Intent = intent
	}

	if category, ok := data["category"].(string); ok {
		query.Category = category
	}

	return query
}

// Stats represents assistant statistics for the status endpoint
type Stats struct {
	ConversationCount   int                    `json:"conversation_count"`
	MessageCount        int                    `json:"message_count"`
	ToolExecutions      int                    `json:"tool_executions"`
	AverageResponseTime string                 `json:"average_response_time"`
	LastActivityTime    string                 `json:"last_activity_time"`
	ActiveProviders     []string               `json:"active_providers"`
	ProviderUsage       map[string]interface{} `json:"provider_usage"`
}

// Tool represents basic tool information
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Version     string                 `json:"version"`
	Author      string                 `json:"author"`
	IsEnabled   bool                   `json:"is_enabled"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"` // Tool parameters schema
}

// ToolExecutionRequest represents a request to execute a tool
type ToolExecutionRequest struct {
	ToolName string                 `json:"tool_name"`
	Input    map[string]interface{} `json:"input"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Context  *ToolExecutionContext  `json:"context,omitempty"`
}

// ToolExecutionContext provides context for tool execution
type ToolExecutionContext struct {
	UserID         string            `json:"user_id,omitempty"`
	SessionID      string            `json:"session_id,omitempty"`
	ConversationID string            `json:"conversation_id,omitempty"`
	RequestID      string            `json:"request_id,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// ToolExecutionResponse represents the response from tool execution
type ToolExecutionResponse struct {
	Success       bool                   `json:"success"`
	Result        interface{}            `json:"result,omitempty"`
	Error         string                 `json:"error,omitempty"`
	ExecutionTime time.Duration          `json:"execution_time"`
	ToolsUsed     []string               `json:"tools_used"`
	Data          *ToolResultData        `json:"data,omitempty"`
	Metadata      *ToolExecutionMetadata `json:"metadata,omitempty"`
}

// ToolResultData contains the actual result data from tool execution
type ToolResultData struct {
	Result    interface{}            `json:"result"`
	Output    map[string]interface{} `json:"output,omitempty"`
	Artifacts []ToolArtifact         `json:"artifacts,omitempty"`
}

// ToolArtifact represents a file or artifact produced by a tool
type ToolArtifact struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Content     []byte `json:"content,omitempty"`
	Path        string `json:"path,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
}

// ToolExecutionMetadata contains metadata about tool execution
type ToolExecutionMetadata struct {
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	CPUTime    time.Duration          `json:"cpu_time,omitempty"`
	MemoryUsed int64                  `json:"memory_used,omitempty"`
	Custom     map[string]interface{} `json:"custom,omitempty"`
	Warnings   []string               `json:"warnings,omitempty"`
	Debug      map[string]interface{} `json:"debug,omitempty"`
}
