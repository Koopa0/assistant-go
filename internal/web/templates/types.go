package templates

import "time"

// Common types for template data

// BaseLayoutData contains data for the base layout
type BaseLayoutData struct {
	Title       string
	Description string
	Lang        string
	Theme       string
}

// AppLayoutData contains data for the app layout
type AppLayoutData struct {
	BaseLayoutData
	CurrentPage string
}

// DashboardData contains data for the dashboard page
type DashboardData struct {
	Lang             string
	ActiveAgents     int
	TasksCompleted   int
	SystemHealth     string
	Uptime           time.Duration
	AgentStatuses    []AgentStatus
	RecentActivities []Activity
	RecentChats      []ChatSummary
}

// AgentStatus represents the status of an AI agent
type AgentStatus struct {
	Name        string
	Description string
	Status      string
	IsActive    bool
	LastUsed    time.Time
	Href        string
}

// Activity represents a recent activity in the system
type Activity struct {
	ID          string
	Type        string
	Description string
	Timestamp   time.Time
	Icon        string
	Status      string
}

// ChatSummary represents a summary of a chat conversation
type ChatSummary struct {
	ID        string
	Title     string
	Agent     string
	Timestamp time.Time
	Preview   string
}

// ChatData contains data for the chat interface
type ChatData struct {
	Lang            string
	ConversationID  string
	AvailableAgents []Agent
	CurrentAgent    Agent
	Messages        []Message
	IsStreaming     bool
}

// Agent represents an AI agent
type Agent struct {
	ID           string
	Name         string
	Description  string
	Avatar       string
	Status       string
	Capabilities []string
}

// Message represents a chat message
type Message struct {
	ID        string
	Content   string
	Role      string // "user" or "assistant"
	Timestamp time.Time
	Agent     string
	Status    string
	Metadata  map[string]interface{}
}

// ToolData contains data for tools management
type ToolData struct {
	Lang           string
	AvailableTools []Tool
	ActiveTools    []Tool
	ToolCategories []ToolCategory
}

// Tool represents a tool in the system
type Tool struct {
	ID          string
	Name        string
	Description string
	Category    string
	Status      string
	IsActive    bool
	LastUsed    time.Time
	Config      map[string]interface{}
}

// ToolCategory represents a category of tools
type ToolCategory struct {
	ID    string
	Name  string
	Icon  string
	Tools []Tool
}

// SettingsData contains data for the settings page
type SettingsData struct {
	Lang            string
	UserPreferences UserPreferences
	APIKeys         []APIKey
	Connections     []Connection
	Notifications   NotificationSettings
}

// UserPreferences represents user preferences
type UserPreferences struct {
	Language    string
	Theme       string
	Timezone    string
	DateFormat  string
	TimeFormat  string
	Density     string // "compact", "comfortable", "spacious"
	FontSize    string
	AccentColor string
}

// APIKey represents an API key configuration
type APIKey struct {
	ID          string
	Provider    string
	Name        string
	IsActive    bool
	LastUsed    time.Time
	ExpiresAt   *time.Time
	IsValid     bool
	Permissions []string
}

// Connection represents a database or service connection
type Connection struct {
	ID         string
	Name       string
	Type       string
	Host       string
	Port       int
	Database   string
	IsActive   bool
	IsValid    bool
	LastTested time.Time
	Config     map[string]interface{}
}

// NotificationSettings represents notification preferences
type NotificationSettings struct {
	EmailEnabled    bool
	InAppEnabled    bool
	DesktopEnabled  bool
	AgentAlerts     bool
	SystemAlerts    bool
	ErrorAlerts     bool
	WeeklyReports   bool
	MonthlyReports  bool
	AlertThresholds map[string]float64
}

// DevelopmentData contains data for the development assistant page
type DevelopmentData struct {
	Lang              string
	ProjectInfo       ProjectInfo
	RecentFiles       []FileInfo
	CodeAnalysis      CodeAnalysis
	PerformanceData   PerformanceData
	AvailableFeatures []DevFeature
}

// ProjectInfo represents current project information
type ProjectInfo struct {
	Name        string
	Path        string
	Language    string
	Framework   string
	Version     string
	LastUpdated time.Time
	Status      string
}

// FileInfo represents a file in the project
type FileInfo struct {
	Path         string
	Name         string
	Type         string
	Size         int64
	LastModified time.Time
	Language     string
}

// CodeAnalysis represents code analysis results
type CodeAnalysis struct {
	LinesOfCode  int
	TestCoverage float64
	Complexity   int
	Issues       []CodeIssue
	Suggestions  []string
	Dependencies []Dependency
}

// CodeIssue represents a code issue
type CodeIssue struct {
	File       string
	Line       int
	Column     int
	Severity   string
	Message    string
	Rule       string
	Suggestion string
}

// Dependency represents a project dependency
type Dependency struct {
	Name           string
	Version        string
	LatestVersion  string
	IsOutdated     bool
	SecurityIssues int
	License        string
}

// PerformanceData represents performance metrics
type PerformanceData struct {
	CPUUsage       float64
	MemoryUsage    float64
	GoroutineCount int
	HeapSize       int64
	GCStats        GCStats
	Profiles       []ProfileData
}

// GCStats represents garbage collection statistics
type GCStats struct {
	NumGC      uint32
	PauseTotal time.Duration
	PauseAvg   time.Duration
	PauseMax   time.Duration
	LastGC     time.Time
}

// ProfileData represents profiling data
type ProfileData struct {
	Type         string
	Timestamp    time.Time
	Duration     time.Duration
	SampleCount  int
	TopFunctions []FunctionProfile
}

// FunctionProfile represents a function in profiling data
type FunctionProfile struct {
	Name       string
	Package    string
	File       string
	Line       int
	Cumulative float64
	Flat       float64
	Percentage float64
}

// DevFeature represents a development feature
type DevFeature struct {
	ID          string
	Name        string
	Description string
	IsEnabled   bool
	IsActive    bool
	Icon        string
	Category    string
}

// DatabaseData contains data for the database management page
type DatabaseData struct {
	Lang         string
	Connections  []DatabaseConnection
	CurrentDB    *DatabaseConnection
	Schema       DatabaseSchema
	QueryHistory []QueryInfo
	Performance  DatabasePerformance
}

// DatabaseConnection represents a database connection
type DatabaseConnection struct {
	ID          string
	Name        string
	Type        string
	Host        string
	Port        int
	Database    string
	Username    string
	IsConnected bool
	LastUsed    time.Time
	Config      map[string]interface{}
}

// DatabaseSchema represents database schema information
type DatabaseSchema struct {
	Tables    []TableInfo
	Views     []ViewInfo
	Indexes   []IndexInfo
	Functions []FunctionInfo
	Triggers  []TriggerInfo
}

// TableInfo represents database table information
type TableInfo struct {
	Name        string
	Schema      string
	RowCount    int64
	Size        int64
	Columns     []ColumnInfo
	Constraints []ConstraintInfo
	Indexes     []IndexInfo
}

// ColumnInfo represents database column information
type ColumnInfo struct {
	Name         string
	Type         string
	IsNullable   bool
	IsPrimaryKey bool
	IsForeignKey bool
	DefaultValue *string
	Comment      string
}

// ConstraintInfo represents database constraint information
type ConstraintInfo struct {
	Name       string
	Type       string
	Columns    []string
	References *ReferenceInfo
}

// ReferenceInfo represents foreign key reference information
type ReferenceInfo struct {
	Table   string
	Columns []string
}

// ViewInfo represents database view information
type ViewInfo struct {
	Name       string
	Schema     string
	Definition string
	Columns    []ColumnInfo
}

// IndexInfo represents database index information
type IndexInfo struct {
	Name      string
	Table     string
	Columns   []string
	IsUnique  bool
	IsPrimary bool
	Type      string
	Size      int64
}

// FunctionInfo represents database function information
type FunctionInfo struct {
	Name       string
	Schema     string
	Language   string
	Definition string
	Parameters []ParameterInfo
	ReturnType string
}

// ParameterInfo represents function parameter information
type ParameterInfo struct {
	Name string
	Type string
	Mode string // IN, OUT, INOUT
}

// TriggerInfo represents database trigger information
type TriggerInfo struct {
	Name       string
	Table      string
	Event      string
	Timing     string
	Definition string
}

// QueryInfo represents a database query
type QueryInfo struct {
	ID           string
	SQL          string
	Timestamp    time.Time
	Duration     time.Duration
	RowsAffected int64
	Status       string
	Error        string
}

// DatabasePerformance represents database performance metrics
type DatabasePerformance struct {
	ConnectionCount   int
	ActiveConnections int
	SlowQueries       []SlowQuery
	IndexSuggestions  []IndexSuggestion
	TableStats        []TableStats
}

// SlowQuery represents a slow query
type SlowQuery struct {
	Query        string
	Duration     time.Duration
	Timestamp    time.Time
	Database     string
	RowsExamined int64
	RowsSent     int64
}

// IndexSuggestion represents an index suggestion
type IndexSuggestion struct {
	Table   string
	Columns []string
	Reason  string
	Impact  string
	Query   string
}

// TableStats represents table statistics
type TableStats struct {
	Name         string
	RowCount     int64
	DataSize     int64
	IndexSize    int64
	LastAnalyzed time.Time
}

// InfrastructureData contains data for infrastructure monitoring
type InfrastructureData struct {
	Lang           string
	Clusters       []ClusterInfo
	CurrentCluster *ClusterInfo
	Nodes          []NodeInfo
	Pods           []PodInfo
	Services       []ServiceInfo
	Deployments    []DeploymentInfo
	Metrics        InfraMetrics
}

// ClusterInfo represents Kubernetes cluster information
type ClusterInfo struct {
	Name        string
	Version     string
	Provider    string
	NodeCount   int
	PodCount    int
	Namespace   string
	Status      string
	LastUpdated time.Time
}

// NodeInfo represents Kubernetes node information
type NodeInfo struct {
	Name           string
	Status         string
	Role           string
	Version        string
	OS             string
	Architecture   string
	CPUCapacity    string
	MemoryCapacity string
	CPUUsage       float64
	MemoryUsage    float64
	PodCount       int
	Conditions     []NodeCondition
}

// NodeCondition represents a node condition
type NodeCondition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

// PodInfo represents Kubernetes pod information
type PodInfo struct {
	Name         string
	Namespace    string
	Status       string
	Node         string
	Containers   []ContainerInfo
	CreatedAt    time.Time
	RestartCount int
	CPUUsage     float64
	MemoryUsage  float64
}

// ContainerInfo represents container information
type ContainerInfo struct {
	Name         string
	Image        string
	Status       string
	RestartCount int
	CPUUsage     float64
	MemoryUsage  float64
	Ports        []PortInfo
}

// PortInfo represents container port information
type PortInfo struct {
	Name          string
	ContainerPort int
	Protocol      string
}

// ServiceInfo represents Kubernetes service information
type ServiceInfo struct {
	Name       string
	Namespace  string
	Type       string
	ClusterIP  string
	ExternalIP string
	Ports      []ServicePort
	Selector   map[string]string
	Endpoints  []string
}

// ServicePort represents service port information
type ServicePort struct {
	Name       string
	Port       int
	TargetPort int
	Protocol   string
	NodePort   int
}

// DeploymentInfo represents Kubernetes deployment information
type DeploymentInfo struct {
	Name          string
	Namespace     string
	Replicas      int32
	ReadyReplicas int32
	Strategy      string
	Image         string
	CreatedAt     time.Time
	Status        string
	Conditions    []DeploymentCondition
}

// DeploymentCondition represents a deployment condition
type DeploymentCondition struct {
	Type    string
	Status  string
	Reason  string
	Message string
}

// InfraMetrics represents infrastructure metrics
type InfraMetrics struct {
	ClusterCPU    float64
	ClusterMemory float64
	NetworkIO     NetworkIOStats
	StorageStats  StorageStats
	ErrorRate     float64
	Availability  float64
}

// NetworkIOStats represents network I/O statistics
type NetworkIOStats struct {
	BytesIn    int64
	BytesOut   int64
	PacketsIn  int64
	PacketsOut int64
}

// StorageStats represents storage statistics
type StorageStats struct {
	Total     int64
	Used      int64
	Available int64
	IOPS      int64
}
