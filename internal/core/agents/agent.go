package agents

import (
	"context"
	"time"
)

// Agent defines the core interface for intelligent agents
// Small, focused interface following Go principles
type Agent interface {
	// Execute processes a request and returns a response
	Execute(ctx context.Context, request Request) (*Response, error)
	// Name returns the agent's name
	Name() string
	// Type returns the agent's type
	Type() AgentType
}

// Collaborator defines advanced agent collaboration capabilities
type Collaborator interface {
	Agent
	// CanHandle determines if the agent can handle a request with confidence level
	CanHandle(ctx context.Context, request Request) (bool, float64)
	// Collaborate works with other agents on a request
	Collaborate(ctx context.Context, other Agent, request Request) (*CollaborationResult, error)
}

// Learner defines agent learning capabilities
type Learner interface {
	Agent
	// Learn processes feedback to improve performance
	Learn(ctx context.Context, feedback Feedback) error
	// Adapt adjusts behavior based on environment
	Adapt(ctx context.Context, environment Environment) error
}

// AgentType identifies the type of agent
type AgentType string

const (
	TypeDevelopment    AgentType = "development"
	TypeDatabase       AgentType = "database"
	TypeInfrastructure AgentType = "infrastructure"
	TypeResearch       AgentType = "research"
	TypeGeneral        AgentType = "general"
	TypeSecurity       AgentType = "security"
	TypeTesting        AgentType = "testing"
	TypeDeployment     AgentType = "deployment"
	TypeMonitoring     AgentType = "monitoring"
	TypeOptimization   AgentType = "optimization"
)

// Request represents a request to an agent
type Request struct {
	ID        string                 `json:"id"`
	Type      RequestType            `json:"type"`
	Content   string                 `json:"content"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Priority  Priority               `json:"priority"`
	Timeout   time.Duration          `json:"timeout"`
	UserID    string                 `json:"user_id"`
	SessionID string                 `json:"session_id,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// Response represents a response from an agent
type Response struct {
	RequestID   string                 `json:"request_id"`
	AgentName   string                 `json:"agent_name"`
	Content     string                 `json:"content"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	Confidence  float64                `json:"confidence"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// RequestType defines types of requests
type RequestType string

const (
	RequestAnalyze   RequestType = "analyze"
	RequestImplement RequestType = "implement"
	RequestOptimize  RequestType = "optimize"
	RequestDebug     RequestType = "debug"
	RequestResearch  RequestType = "research"
	RequestTest      RequestType = "test"
	RequestDocument  RequestType = "document"
	RequestReview    RequestType = "review"
	RequestExplain   RequestType = "explain"
	RequestSummarize RequestType = "summarize"
)

// Priority defines request priority levels
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// CollaborationResult represents the result of agent collaboration
type CollaborationResult struct {
	PrimaryResponse   *Response              `json:"primary_response"`
	SecondaryResponse *Response              `json:"secondary_response"`
	CombinedResponse  *Response              `json:"combined_response"`
	Confidence        float64                `json:"confidence"`
	Strategy          CollaborationStrategy  `json:"strategy"`
	Duration          time.Duration          `json:"duration"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// CollaborationStrategy defines how agents collaborate
type CollaborationStrategy string

const (
	StrategyPeerReview     CollaborationStrategy = "peer_review"
	StrategyJointExecution CollaborationStrategy = "joint_execution"
	StrategyDelegation     CollaborationStrategy = "delegation"
	StrategyConsultation   CollaborationStrategy = "consultation"
	StrategySecondOpinion  CollaborationStrategy = "second_opinion"
)

// Feedback represents feedback for agent learning
type Feedback struct {
	ID        string       `json:"id"`
	AgentName string       `json:"agent_name"`
	RequestID string       `json:"request_id"`
	Type      FeedbackType `json:"type"`
	Content   string       `json:"content"`
	Rating    float64      `json:"rating"` // 0.0 to 1.0
	Source    string       `json:"source"` // "user", "system", "peer"
	Timestamp time.Time    `json:"timestamp"`
}

// FeedbackType defines types of feedback
type FeedbackType string

const (
	FeedbackPositive   FeedbackType = "positive"
	FeedbackNegative   FeedbackType = "negative"
	FeedbackSuggestion FeedbackType = "suggestion"
	FeedbackCorrection FeedbackType = "correction"
)

// Environment represents the agent's operating environment
type Environment struct {
	Context     map[string]interface{} `json:"context"`
	Resources   []string               `json:"resources"`
	Constraints []Constraint           `json:"constraints"`
	Goals       []Goal                 `json:"goals"`
	Metrics     map[string]float64     `json:"metrics"`
}

// Constraint represents an environmental constraint
type Constraint struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Value       interface{} `json:"value"`
	Enforced    bool        `json:"enforced"`
}

// Goal represents an environmental goal
type Goal struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Priority    Priority   `json:"priority"`
	Target      float64    `json:"target"`
	Current     float64    `json:"current"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	Achieved    bool       `json:"achieved"`
}

// AgentManager coordinates multiple agents
type AgentManager interface {
	// RegisterAgent adds an agent to the manager
	RegisterAgent(agent Agent) error
	// GetAgent retrieves an agent by name
	GetAgent(name string) (Agent, error)
	// ListAgents returns all registered agents
	ListAgents() []Agent
	// Route finds the best agent for a request
	Route(ctx context.Context, request Request) (Agent, error)
	// ExecuteWithBestAgent automatically routes and executes
	ExecuteWithBestAgent(ctx context.Context, request Request) (*Response, error)
}

// CapabilityMatcher helps determine which agent can best handle a request
type CapabilityMatcher interface {
	// Match returns agents that can handle the request with confidence scores
	Match(ctx context.Context, request Request, agents []Agent) ([]AgentMatch, error)
}

// AgentMatch represents an agent's suitability for a request
type AgentMatch struct {
	Agent      Agent   `json:"agent"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// PerformanceTracker tracks agent performance metrics
type PerformanceTracker interface {
	// RecordExecution records an agent execution
	RecordExecution(agent Agent, request Request, response *Response) error
	// GetStats returns performance statistics for an agent
	GetStats(agentName string) (*PerformanceStats, error)
	// GetOverallStats returns overall system performance
	GetOverallStats() (*OverallStats, error)
}

// PerformanceStats tracks individual agent performance
type PerformanceStats struct {
	AgentName           string        `json:"agent_name"`
	TotalRequests       int           `json:"total_requests"`
	SuccessfulRequests  int           `json:"successful_requests"`
	SuccessRate         float64       `json:"success_rate"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	AverageConfidence   float64       `json:"average_confidence"`
	LastActivity        time.Time     `json:"last_activity"`
}

// OverallStats tracks system-wide performance
type OverallStats struct {
	TotalAgents         int                 `json:"total_agents"`
	TotalRequests       int                 `json:"total_requests"`
	AverageResponseTime time.Duration       `json:"average_response_time"`
	SystemSuccessRate   float64             `json:"system_success_rate"`
	RequestsByType      map[RequestType]int `json:"requests_by_type"`
	TopPerformers       []string            `json:"top_performers"`
	LastUpdated         time.Time           `json:"last_updated"`
}
