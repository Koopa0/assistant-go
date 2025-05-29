package agents

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	ctxpkg "github.com/koopa0/assistant/internal/core/context"
)

// Agent represents an AI agent with specialized capabilities
type Agent interface {
	// Core agent interface
	GetID() string
	GetName() string
	GetDomain() Domain
	GetCapabilities() []Capability
	GetConfidence() float64
	GetStatus() AgentStatus

	// Execution methods
	CanHandle(ctx context.Context, request *ctxpkg.ContextualRequest) (bool, float64)
	Execute(ctx context.Context, request *ctxpkg.ContextualRequest) (*AgentResponse, error)
	Collaborate(ctx context.Context, other Agent, request *ctxpkg.ContextualRequest) (*CollaborationResult, error)

	// Learning and adaptation
	Learn(ctx context.Context, feedback *Feedback) error
	Adapt(ctx context.Context, environment *Environment) error

	// State management
	GetState() AgentState
	UpdateState(state AgentState) error

	// Resource management
	GetResources() Resources
	AllocateResources(resources Resources) error
	ReleaseResources() error

	// Lifecycle
	Initialize(ctx context.Context, config AgentConfig) error
	Shutdown(ctx context.Context) error
}

// BaseAgent provides common functionality for all agents
type BaseAgent struct {
	ID           string
	Name         string
	Domain       Domain
	Capabilities []Capability
	Confidence   float64
	Status       AgentStatus
	State        AgentState
	Resources    Resources
	Memory       *AgentMemory
	Logger       *slog.Logger
	mu           sync.RWMutex
}

// Domain represents the area of expertise for an agent
type Domain string

const (
	DomainDevelopment    Domain = "development"
	DomainDatabase       Domain = "database"
	DomainInfrastructure Domain = "infrastructure"
	DomainResearch       Domain = "research"
	DomainGeneral        Domain = "general"
	DomainSecurity       Domain = "security"
	DomainTesting        Domain = "testing"
	DomainDeployment     Domain = "deployment"
	DomainMonitoring     Domain = "monitoring"
	DomainOptimization   Domain = "optimization"
)

// Capability represents a specific capability of an agent
type Capability struct {
	Name        string
	Description string
	Proficiency float64
	Tools       []string
	LastUsed    time.Time
	SuccessRate float64
}

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	StatusIdle          AgentStatus = "idle"
	StatusActive        AgentStatus = "active"
	StatusBusy          AgentStatus = "busy"
	StatusLearning      AgentStatus = "learning"
	StatusCollaborating AgentStatus = "collaborating"
	StatusError         AgentStatus = "error"
	StatusShutdown      AgentStatus = "shutdown"
)

// AgentState represents the internal state of an agent
type AgentState struct {
	CurrentTask    *Task
	ActiveContext  *ctxpkg.ContextualRequest
	RecentActions  []Action
	Collaborations []CollaborationInfo
	Performance    PerformanceMetrics
	LearningState  LearningState
	LastUpdate     time.Time
}

// Task represents a task being executed by an agent
type Task struct {
	ID           string
	Type         TaskType
	Description  string
	Priority     Priority
	Status       TaskStatus
	StartTime    time.Time
	EndTime      *time.Time
	Progress     float64
	SubTasks     []Task
	Dependencies []string
	Results      map[string]interface{}
}

// TaskType defines the type of task
type TaskType string

const (
	TaskAnalysis       TaskType = "analysis"
	TaskImplementation TaskType = "implementation"
	TaskOptimization   TaskType = "optimization"
	TaskDebug          TaskType = "debug"
	TaskResearch       TaskType = "research"
	TaskTesting        TaskType = "testing"
	TaskDocumentation  TaskType = "documentation"
	TaskReview         TaskType = "review"
)

// Priority defines task priority levels
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// TaskStatus defines task execution status
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusBlocked   TaskStatus = "blocked"
)

// Action represents an action taken by an agent
type Action struct {
	ID          string
	Type        ActionType
	Description string
	Timestamp   time.Time
	Tool        string
	Input       map[string]interface{}
	Output      map[string]interface{}
	Success     bool
	Duration    time.Duration
	Context     string
}

// ActionType defines types of actions
type ActionType string

const (
	ActionAnalyze  ActionType = "analyze"
	ActionExecute  ActionType = "execute"
	ActionQuery    ActionType = "query"
	ActionCreate   ActionType = "create"
	ActionModify   ActionType = "modify"
	ActionDelete   ActionType = "delete"
	ActionValidate ActionType = "validate"
	ActionOptimize ActionType = "optimize"
)

// Resources represents computational and external resources
type Resources struct {
	Memory      int64
	CPU         float64
	Storage     int64
	NetworkBW   int64
	Tools       []ToolAccess
	Databases   []DatabaseAccess
	APIs        []APIAccess
	Permissions []Permission
}

// ToolAccess represents access to a specific tool
type ToolAccess struct {
	ToolName    string
	AccessLevel AccessLevel
	LastUsed    time.Time
	UsageCount  int
}

// DatabaseAccess represents access to a database
type DatabaseAccess struct {
	DatabaseID  string
	AccessLevel AccessLevel
	LastUsed    time.Time
	QueryCount  int
}

// APIAccess represents access to an external API
type APIAccess struct {
	APIID       string
	AccessLevel AccessLevel
	LastUsed    time.Time
	CallCount   int
	RateLimit   int
}

// Permission represents a permission granted to an agent
type Permission struct {
	Resource  string
	Action    string
	Scope     string
	ExpiresAt *time.Time
}

// AccessLevel defines access levels
type AccessLevel string

const (
	AccessRead    AccessLevel = "read"
	AccessWrite   AccessLevel = "write"
	AccessExecute AccessLevel = "execute"
	AccessAdmin   AccessLevel = "admin"
)

// AgentMemory represents agent's working memory
type AgentMemory struct {
	WorkingMemory    map[string]interface{}
	EpisodicMemory   []MemoryEpisode
	SemanticMemory   map[string]Concept
	ProceduralMemory map[string]Procedure
	mu               sync.RWMutex
}

// MemoryEpisode represents an episodic memory
type MemoryEpisode struct {
	ID         string
	Timestamp  time.Time
	Event      string
	Context    map[string]interface{}
	Outcome    string
	Importance float64
}

// Concept represents a semantic concept
type Concept struct {
	Name         string
	Definition   string
	Relations    map[string]float64
	Confidence   float64
	LastAccessed time.Time
}

// Procedure represents a learned procedure
type Procedure struct {
	Name           string
	Steps          []ProcedureStep
	Preconditions  []string
	Postconditions []string
	SuccessRate    float64
	LastUsed       time.Time
}

// ProcedureStep represents a step in a procedure
type ProcedureStep struct {
	Order       int
	Description string
	Action      string
	Parameters  map[string]interface{}
	Expected    string
}

// CollaborationInfo represents information about a collaboration
type CollaborationInfo struct {
	PartnerID string
	Type      CollaborationType
	StartTime time.Time
	EndTime   *time.Time
	Status    CollaborationStatus
	Outcome   string
	Quality   float64
}

// CollaborationType defines types of collaboration
type CollaborationType string

const (
	CollaborationPeerReview     CollaborationType = "peer_review"
	CollaborationJointExecution CollaborationType = "joint_execution"
	CollaborationKnowledgeShare CollaborationType = "knowledge_share"
	CollaborationDelegation     CollaborationType = "delegation"
	CollaborationConsultation   CollaborationType = "consultation"
)

// CollaborationStatus defines collaboration status
type CollaborationStatus string

const (
	CollaborationActive    CollaborationStatus = "active"
	CollaborationCompleted CollaborationStatus = "completed"
	CollaborationFailed    CollaborationStatus = "failed"
	CollaborationCancelled CollaborationStatus = "cancelled"
)

// PerformanceMetrics tracks agent performance
type PerformanceMetrics struct {
	TaskCompletionRate  float64
	AverageTaskDuration time.Duration
	SuccessRate         float64
	QualityScore        float64
	EfficiencyScore     float64
	CollaborationScore  float64
	LearningRate        float64
	AdaptationRate      float64
	ResourceUtilization float64
	LastCalculated      time.Time
}

// LearningState represents the learning state of an agent
type LearningState struct {
	LearningMode   LearningMode
	CurrentLessons []Lesson
	RecentFeedback []Feedback
	KnowledgeBase  map[string]Knowledge
	LearningGoals  []LearningGoal
	Progress       map[string]float64
	LastUpdate     time.Time
}

// LearningMode defines learning modes
type LearningMode string

const (
	LearningPassive       LearningMode = "passive"
	LearningActive        LearningMode = "active"
	LearningReinforcement LearningMode = "reinforcement"
	LearningTransfer      LearningMode = "transfer"
)

// Lesson represents a learning lesson
type Lesson struct {
	ID         string
	Topic      string
	Content    string
	Type       LessonType
	Difficulty float64
	Progress   float64
	StartTime  time.Time
	Deadline   *time.Time
}

// LessonType defines types of lessons
type LessonType string

const (
	LessonTutorial     LessonType = "tutorial"
	LessonExercise     LessonType = "exercise"
	LessonCase         LessonType = "case"
	LessonExperimental LessonType = "experimental"
)

// Knowledge represents a piece of knowledge
type Knowledge struct {
	Topic       string
	Content     string
	Source      string
	Confidence  float64
	Verified    bool
	LastUpdated time.Time
	UsageCount  int
}

// LearningGoal represents a learning objective
type LearningGoal struct {
	ID         string
	Skill      string
	Target     string
	Priority   Priority
	Progress   float64
	Deadline   time.Time
	Milestones []Milestone
}

// Milestone represents a learning milestone
type Milestone struct {
	ID          string
	Description string
	Target      float64
	Achieved    bool
	AchievedAt  *time.Time
}

// AgentResponse represents a response from an agent
type AgentResponse struct {
	AgentID     string
	RequestID   string
	Content     string
	Confidence  float64
	Actions     []Action
	Resources   []string
	Suggestions []Suggestion
	Metadata    map[string]interface{}
	Duration    time.Duration
	Success     bool
	ErrorMsg    string
}

// Suggestion represents a suggestion from an agent
type Suggestion struct {
	Type       SuggestionType
	Content    string
	Confidence float64
	Rationale  string
	Actions    []string
}

// SuggestionType defines types of suggestions
type SuggestionType string

const (
	SuggestionImprovement  SuggestionType = "improvement"
	SuggestionOptimization SuggestionType = "optimization"
	SuggestionAlternative  SuggestionType = "alternative"
	SuggestionPrevention   SuggestionType = "prevention"
	SuggestionLearning     SuggestionType = "learning"
)

// CollaborationResult represents the result of agent collaboration
type CollaborationResult struct {
	PrimaryResponse     *AgentResponse
	SecondaryResponse   *AgentResponse
	SynthesizedResponse *AgentResponse
	CollaborationType   CollaborationType
	Quality             float64
	Duration            time.Duration
	ConflictResolution  []ConflictResolution
}

// ConflictResolution represents how conflicts were resolved
type ConflictResolution struct {
	Conflict   string
	Resolution string
	Method     string
	Confidence float64
}

// Feedback represents feedback for agent learning
type Feedback struct {
	ID         string
	AgentID    string
	Source     FeedbackSource
	Type       FeedbackType
	Content    string
	Rating     float64
	Timestamp  time.Time
	Context    map[string]interface{}
	Actionable bool
}

// FeedbackSource defines sources of feedback
type FeedbackSource string

const (
	FeedbackUser        FeedbackSource = "user"
	FeedbackSystem      FeedbackSource = "system"
	FeedbackPeer        FeedbackSource = "peer"
	FeedbackEnvironment FeedbackSource = "environment"
)

// FeedbackType defines types of feedback
type FeedbackType string

const (
	FeedbackPositive   FeedbackType = "positive"
	FeedbackNegative   FeedbackType = "negative"
	FeedbackNeutral    FeedbackType = "neutral"
	FeedbackSuggestion FeedbackType = "suggestion"
	FeedbackCorrection FeedbackType = "correction"
)

// Environment represents the agent's operating environment
type Environment struct {
	Context     *ctxpkg.ContextualRequest
	Resources   Resources
	Constraints []Constraint
	Goals       []Goal
	Metrics     map[string]float64
	State       map[string]interface{}
}

// Constraint represents an environmental constraint
type Constraint struct {
	Type        ConstraintType
	Description string
	Value       interface{}
	Enforced    bool
}

// ConstraintType defines types of constraints
type ConstraintType string

const (
	ConstraintTime     ConstraintType = "time"
	ConstraintResource ConstraintType = "resource"
	ConstraintQuality  ConstraintType = "quality"
	ConstraintSecurity ConstraintType = "security"
	ConstraintPolicy   ConstraintType = "policy"
)

// Goal represents an environmental goal
type Goal struct {
	ID          string
	Description string
	Priority    Priority
	Target      float64
	Current     float64
	Deadline    *time.Time
	Achieved    bool
}

// AgentConfig represents configuration for an agent
type AgentConfig struct {
	Name                string
	Domain              Domain
	Capabilities        []string
	Resources           Resources
	LearningConfig      LearningConfig
	CollaborationConfig CollaborationConfig
	Settings            map[string]interface{}
}

// LearningConfig configures agent learning
type LearningConfig struct {
	Mode            LearningMode
	Rate            float64
	MaxMemory       int
	RetentionPeriod time.Duration
	Sources         []string
}

// CollaborationConfig configures agent collaboration
type CollaborationConfig struct {
	Enabled           bool
	PreferredPartners []string
	MaxCollaborations int
	TrustThreshold    float64
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(id, name string, domain Domain, logger *slog.Logger) *BaseAgent {
	return &BaseAgent{
		ID:           id,
		Name:         name,
		Domain:       domain,
		Capabilities: make([]Capability, 0),
		Confidence:   0.5,
		Status:       StatusIdle,
		State: AgentState{
			RecentActions:  make([]Action, 0),
			Collaborations: make([]CollaborationInfo, 0),
			Performance: PerformanceMetrics{
				LastCalculated: time.Now(),
			},
			LearningState: LearningState{
				LearningMode:   LearningPassive,
				CurrentLessons: make([]Lesson, 0),
				RecentFeedback: make([]Feedback, 0),
				KnowledgeBase:  make(map[string]Knowledge),
				LearningGoals:  make([]LearningGoal, 0),
				Progress:       make(map[string]float64),
				LastUpdate:     time.Now(),
			},
			LastUpdate: time.Now(),
		},
		Resources: Resources{
			Tools:       make([]ToolAccess, 0),
			Databases:   make([]DatabaseAccess, 0),
			APIs:        make([]APIAccess, 0),
			Permissions: make([]Permission, 0),
		},
		Memory: &AgentMemory{
			WorkingMemory:    make(map[string]interface{}),
			EpisodicMemory:   make([]MemoryEpisode, 0),
			SemanticMemory:   make(map[string]Concept),
			ProceduralMemory: make(map[string]Procedure),
		},
		Logger: logger,
	}
}

// GetID returns the agent ID
func (ba *BaseAgent) GetID() string {
	return ba.ID
}

// GetName returns the agent name
func (ba *BaseAgent) GetName() string {
	return ba.Name
}

// GetDomain returns the agent's domain
func (ba *BaseAgent) GetDomain() Domain {
	return ba.Domain
}

// GetCapabilities returns the agent's capabilities
func (ba *BaseAgent) GetCapabilities() []Capability {
	ba.mu.RLock()
	defer ba.mu.RUnlock()
	return ba.Capabilities
}

// GetConfidence returns the agent's confidence level
func (ba *BaseAgent) GetConfidence() float64 {
	ba.mu.RLock()
	defer ba.mu.RUnlock()
	return ba.Confidence
}

// GetStatus returns the agent's current status
func (ba *BaseAgent) GetStatus() AgentStatus {
	ba.mu.RLock()
	defer ba.mu.RUnlock()
	return ba.Status
}

// GetState returns the agent's current state
func (ba *BaseAgent) GetState() AgentState {
	ba.mu.RLock()
	defer ba.mu.RUnlock()
	return ba.State
}

// UpdateState updates the agent's state
func (ba *BaseAgent) UpdateState(state AgentState) error {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	ba.State = state
	ba.State.LastUpdate = time.Now()

	ba.Logger.Debug("Agent state updated",
		slog.String("agent_id", ba.ID),
		slog.String("status", string(ba.Status)))

	return nil
}

// GetResources returns the agent's resources
func (ba *BaseAgent) GetResources() Resources {
	ba.mu.RLock()
	defer ba.mu.RUnlock()
	return ba.Resources
}

// AllocateResources allocates resources to the agent
func (ba *BaseAgent) AllocateResources(resources Resources) error {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	ba.Resources = resources

	ba.Logger.Info("Resources allocated to agent",
		slog.String("agent_id", ba.ID),
		slog.Int("tools", len(resources.Tools)),
		slog.Int("databases", len(resources.Databases)))

	return nil
}

// ReleaseResources releases the agent's resources
func (ba *BaseAgent) ReleaseResources() error {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	ba.Resources = Resources{
		Tools:       make([]ToolAccess, 0),
		Databases:   make([]DatabaseAccess, 0),
		APIs:        make([]APIAccess, 0),
		Permissions: make([]Permission, 0),
	}

	ba.Logger.Info("Resources released from agent", slog.String("agent_id", ba.ID))

	return nil
}

// SetStatus updates the agent's status
func (ba *BaseAgent) SetStatus(status AgentStatus) {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	oldStatus := ba.Status
	ba.Status = status

	ba.Logger.Debug("Agent status changed",
		slog.String("agent_id", ba.ID),
		slog.String("from", string(oldStatus)),
		slog.String("to", string(status)))
}

// AddCapability adds a capability to the agent
func (ba *BaseAgent) AddCapability(capability Capability) {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	ba.Capabilities = append(ba.Capabilities, capability)

	ba.Logger.Info("Capability added to agent",
		slog.String("agent_id", ba.ID),
		slog.String("capability", capability.Name),
		slog.Float64("proficiency", capability.Proficiency))
}

// UpdateConfidence updates the agent's confidence level
func (ba *BaseAgent) UpdateConfidence(confidence float64) {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	oldConfidence := ba.Confidence
	ba.Confidence = confidence

	if confidence > 1.0 {
		ba.Confidence = 1.0
	} else if confidence < 0.0 {
		ba.Confidence = 0.0
	}

	ba.Logger.Debug("Agent confidence updated",
		slog.String("agent_id", ba.ID),
		slog.Float64("from", oldConfidence),
		slog.Float64("to", ba.Confidence))
}

// RecordAction records an action taken by the agent
func (ba *BaseAgent) RecordAction(action Action) {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	ba.State.RecentActions = append(ba.State.RecentActions, action)

	// Keep only recent actions (last 100)
	if len(ba.State.RecentActions) > 100 {
		ba.State.RecentActions = ba.State.RecentActions[1:]
	}

	ba.Logger.Debug("Action recorded",
		slog.String("agent_id", ba.ID),
		slog.String("action_id", action.ID),
		slog.String("type", string(action.Type)))
}

// AddMemoryEpisode adds an episodic memory
func (ba *BaseAgent) AddMemoryEpisode(event, outcome string, context map[string]interface{}, importance float64) {
	ba.mu.Lock()
	defer ba.mu.Unlock()

	episode := MemoryEpisode{
		ID:         fmt.Sprintf("episode_%d", time.Now().UnixNano()),
		Timestamp:  time.Now(),
		Event:      event,
		Context:    context,
		Outcome:    outcome,
		Importance: importance,
	}

	ba.Memory.mu.Lock()
	ba.Memory.EpisodicMemory = append(ba.Memory.EpisodicMemory, episode)
	ba.Memory.mu.Unlock()

	ba.Logger.Debug("Memory episode added",
		slog.String("agent_id", ba.ID),
		slog.String("episode_id", episode.ID),
		slog.Float64("importance", importance))
}
