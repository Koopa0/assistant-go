package agents

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	corecontext "github.com/koopa0/assistant/internal/core/context"
)

// AgentManager orchestrates multiple AI agents
type AgentManager struct {
	agents          map[string]Agent
	scheduler       *AgentScheduler
	collaborator    *CollaborationEngine
	learningEngine  *LearningEngine
	resourceManager *ResourceManager
	evaluator       *PerformanceEvaluator
	logger          *slog.Logger
	mu              sync.RWMutex
}

// AgentScheduler manages agent task scheduling and load balancing
type AgentScheduler struct {
	taskQueue      []ScheduledTask
	agentWorkloads map[string]float64
	priorities     map[string]int
	policies       []SchedulingPolicy
	metrics        SchedulingMetrics
	mu             sync.RWMutex
}

// ScheduledTask represents a task scheduled for execution
type ScheduledTask struct {
	ID                string
	Request           *corecontext.ContextualRequest
	RequiredDomain    Domain
	Priority          Priority
	Deadline          *time.Time
	EstimatedDuration time.Duration
	AssignedAgent     string
	Status            TaskStatus
	CreatedAt         time.Time
	ScheduledAt       *time.Time
	StartedAt         *time.Time
	CompletedAt       *time.Time
}

// SchedulingPolicy defines how tasks are scheduled
type SchedulingPolicy struct {
	Name       string
	Type       PolicyType
	Parameters map[string]interface{}
	Weight     float64
	Active     bool
}

// PolicyType defines types of scheduling policies
type PolicyType string

const (
	PolicyRoundRobin     PolicyType = "round_robin"
	PolicyLeastLoaded    PolicyType = "least_loaded"
	PolicyBestFit        PolicyType = "best_fit"
	PolicyPriority       PolicyType = "priority"
	PolicyDeadline       PolicyType = "deadline"
	PolicySpecialization PolicyType = "specialization"
	PolicyLearning       PolicyType = "learning"
)

// SchedulingMetrics tracks scheduling performance
type SchedulingMetrics struct {
	TotalTasks          int
	CompletedTasks      int
	AverageWaitTime     time.Duration
	AverageResponseTime time.Duration
	ThroughputRate      float64
	LoadBalance         float64
	LastUpdated         time.Time
}

// CollaborationEngine manages agent collaboration
type CollaborationEngine struct {
	activeCollaborations map[string]*Collaboration
	collaborationRules   []CollaborationRule
	trustMatrix          map[string]map[string]float64
	metrics              CollaborationMetrics
	mu                   sync.RWMutex
}

// Collaboration represents an active collaboration between agents
type Collaboration struct {
	ID           string
	Type         CollaborationType
	Participants []string
	Coordinator  string
	Task         *ScheduledTask
	Status       CollaborationStatus
	StartTime    time.Time
	EndTime      *time.Time
	Results      []AgentResponse
	Quality      float64
	Conflicts    []Conflict
}

// CollaborationRule defines rules for agent collaboration
type CollaborationRule struct {
	ID        string
	Trigger   string
	Condition string
	Action    string
	Agents    []string
	Priority  int
	Active    bool
}

// Conflict represents a conflict in collaboration
type Conflict struct {
	ID           string
	Type         ConflictType
	Description  string
	Participants []string
	Resolution   *ConflictResolution
	Timestamp    time.Time
}

// ConflictType defines types of conflicts
type ConflictType string

const (
	ConflictApproach ConflictType = "approach"
	ConflictResource ConflictType = "resource"
	ConflictPriority ConflictType = "priority"
	ConflictQuality  ConflictType = "quality"
	ConflictTimeline ConflictType = "timeline"
)

// CollaborationMetrics tracks collaboration performance
type CollaborationMetrics struct {
	TotalCollaborations      int
	SuccessfulCollaborations int
	AverageQuality           float64
	ConflictRate             float64
	ResolutionTime           time.Duration
	TrustScore               float64
	LastUpdated              time.Time
}

// LearningEngine manages agent learning and knowledge sharing
type LearningEngine struct {
	knowledgeBase     *SharedKnowledgeBase
	learningScheduler *LearningScheduler
	feedbackProcessor *FeedbackProcessor
	transferEngine    *KnowledgeTransferEngine
	metrics           LearningMetrics
	mu                sync.RWMutex
}

// SharedKnowledgeBase contains shared knowledge across agents
type SharedKnowledgeBase struct {
	Concepts    map[string]SharedConcept
	Procedures  map[string]SharedProcedure
	Experiences []SharedExperience
	Patterns    map[string]Pattern
	mu          sync.RWMutex
}

// SharedConcept represents a concept shared among agents
type SharedConcept struct {
	ID           string
	Name         string
	Definition   string
	Domain       Domain
	Contributors []string
	Confidence   float64
	Usage        int
	LastUpdated  time.Time
}

// SharedProcedure represents a procedure shared among agents
type SharedProcedure struct {
	ID           string
	Name         string
	Domain       Domain
	Steps        []ProcedureStep
	Success      float64
	Contributors []string
	Usage        int
	LastUpdated  time.Time
}

// SharedExperience represents a shared experience
type SharedExperience struct {
	ID         string
	Agent      string
	Context    string
	Action     string
	Outcome    string
	Lessons    []string
	Applicable []string
	Timestamp  time.Time
}

// Pattern represents a learned pattern
type Pattern struct {
	ID       string
	Type     PatternType
	Context  string
	Trigger  string
	Response string
	Success  float64
	Usage    int
	LastSeen time.Time
}

// PatternType defines types of patterns
type PatternType string

const (
	PatternBehavioral PatternType = "behavioral"
	PatternTechnical  PatternType = "technical"
	PatternProcess    PatternType = "process"
	PatternError      PatternType = "error"
	PatternSuccess    PatternType = "success"
)

// LearningScheduler schedules learning activities
type LearningScheduler struct {
	learningTasks map[string][]LearningTask
	schedule      map[string]time.Time
	priorities    map[string]float64
	mu            sync.RWMutex
}

// LearningTask represents a learning task for an agent
type LearningTask struct {
	ID         string
	AgentID    string
	Type       LearningTaskType
	Content    string
	Difficulty float64
	Priority   Priority
	Deadline   *time.Time
	Status     TaskStatus
	Progress   float64
}

// LearningTaskType defines types of learning tasks
type LearningTaskType string

const (
	LearningTaskStudy      LearningTaskType = "study"
	LearningTaskPractice   LearningTaskType = "practice"
	LearningTaskReflection LearningTaskType = "reflection"
	LearningTaskSharing    LearningTaskType = "sharing"
)

// FeedbackProcessor processes feedback for learning
type FeedbackProcessor struct {
	feedbackQueue []ProcessableFeedback
	processors    map[FeedbackType]FeedbackHandler
	metrics       FeedbackMetrics
	mu            sync.RWMutex
}

// ProcessableFeedback represents feedback ready for processing
type ProcessableFeedback struct {
	Feedback       *Feedback
	ProcessingType ProcessingType
	Priority       Priority
	Timestamp      time.Time
}

// ProcessingType defines how feedback should be processed
type ProcessingType string

const (
	ProcessingImmediate ProcessingType = "immediate"
	ProcessingBatch     ProcessingType = "batch"
	ProcessingDeferred  ProcessingType = "deferred"
)

// FeedbackHandler handles specific types of feedback
type FeedbackHandler interface {
	Handle(ctx context.Context, feedback *Feedback) error
	Priority() Priority
	CanHandle(feedback *Feedback) bool
}

// FeedbackMetrics tracks feedback processing
type FeedbackMetrics struct {
	TotalFeedback         int
	ProcessedFeedback     int
	AverageProcessingTime time.Duration
	QualityScore          float64
	ActionableRate        float64
	LastUpdated           time.Time
}

// KnowledgeTransferEngine manages knowledge transfer between agents
type KnowledgeTransferEngine struct {
	transferRules   []TransferRule
	transferHistory []KnowledgeTransfer
	compatibility   map[string]map[string]float64
	metrics         TransferMetrics
	mu              sync.RWMutex
}

// TransferRule defines rules for knowledge transfer
type TransferRule struct {
	ID         string
	Source     Domain
	Target     Domain
	Knowledge  string
	Conditions []string
	Adaptation string
	Success    float64
}

// KnowledgeTransfer represents a knowledge transfer event
type KnowledgeTransfer struct {
	ID        string
	Source    string
	Target    string
	Knowledge interface{}
	Method    TransferMethod
	Success   bool
	Quality   float64
	Timestamp time.Time
}

// TransferMethod defines knowledge transfer methods
type TransferMethod string

const (
	TransferDirect      TransferMethod = "direct"
	TransferAnalogy     TransferMethod = "analogy"
	TransferAbstraction TransferMethod = "abstraction"
	TransferExample     TransferMethod = "example"
)

// TransferMetrics tracks knowledge transfer performance
type TransferMetrics struct {
	TotalTransfers      int
	SuccessfulTransfers int
	AverageQuality      float64
	CompatibilityScore  float64
	LastUpdated         time.Time
}

// LearningMetrics tracks overall learning performance
type LearningMetrics struct {
	KnowledgeGrowth        float64
	SkillImprovement       map[string]float64
	LearningVelocity       float64
	KnowledgeSharing       float64
	CollectiveIntelligence float64
	LastUpdated            time.Time
}

// ResourceManager manages agent resources
type ResourceManager struct {
	totalResources     Resources
	allocatedResources map[string]Resources
	reservations       []ResourceReservation
	policies           []AllocationPolicy
	metrics            ResourceMetrics
	mu                 sync.RWMutex
}

// ResourceReservation represents a resource reservation
type ResourceReservation struct {
	ID        string
	AgentID   string
	Resources Resources
	StartTime time.Time
	EndTime   time.Time
	Priority  Priority
	Status    ReservationStatus
}

// ReservationStatus defines reservation status
type ReservationStatus string

const (
	ReservationPending   ReservationStatus = "pending"
	ReservationActive    ReservationStatus = "active"
	ReservationCompleted ReservationStatus = "completed"
	ReservationCancelled ReservationStatus = "cancelled"
)

// AllocationPolicy defines resource allocation policies
type AllocationPolicy struct {
	ID         string
	Type       AllocationPolicyType
	Parameters map[string]interface{}
	Priority   int
	Active     bool
}

// AllocationPolicyType defines types of allocation policies
type AllocationPolicyType string

const (
	AllocationFairShare   AllocationPolicyType = "fair_share"
	AllocationPriority    AllocationPolicyType = "priority"
	AllocationDemandBased AllocationPolicyType = "demand_based"
	AllocationPerformance AllocationPolicyType = "performance"
)

// ResourceMetrics tracks resource utilization
type ResourceMetrics struct {
	TotalUtilization float64
	EfficiencyScore  float64
	WastePercentage  float64
	ContentionRate   float64
	AllocationTime   time.Duration
	LastUpdated      time.Time
}

// PerformanceEvaluator evaluates agent performance
type PerformanceEvaluator struct {
	evaluations map[string][]PerformanceEvaluation
	benchmarks  map[string]Benchmark
	comparisons []AgentComparison
	trends      map[string]PerformanceTrend
	mu          sync.RWMutex
}

// PerformanceEvaluation represents a performance evaluation
type PerformanceEvaluation struct {
	ID        string
	AgentID   string
	Period    TimePeriod
	Metrics   PerformanceMetrics
	Score     float64
	Ranking   int
	Feedback  []string
	Timestamp time.Time
}

// TimePeriod represents a time period for evaluation
type TimePeriod struct {
	Start time.Time
	End   time.Time
	Type  PeriodType
}

// PeriodType defines types of evaluation periods
type PeriodType string

const (
	PeriodHourly  PeriodType = "hourly"
	PeriodDaily   PeriodType = "daily"
	PeriodWeekly  PeriodType = "weekly"
	PeriodMonthly PeriodType = "monthly"
)

// Benchmark represents a performance benchmark
type Benchmark struct {
	ID          string
	Name        string
	Domain      Domain
	Metrics     []BenchmarkMetric
	Baseline    float64
	Target      float64
	LastUpdated time.Time
}

// BenchmarkMetric represents a specific metric in a benchmark
type BenchmarkMetric struct {
	Name        string
	Weight      float64
	Target      float64
	Unit        string
	Description string
}

// AgentComparison represents a comparison between agents
type AgentComparison struct {
	ID         string
	Agents     []string
	Metrics    []string
	Results    map[string]map[string]float64
	Winner     string
	Confidence float64
	Timestamp  time.Time
}

// PerformanceTrend represents performance trends over time
type PerformanceTrend struct {
	AgentID     string
	Metric      string
	Trend       TrendDirection
	Slope       float64
	Confidence  float64
	Period      TimePeriod
	Predictions []TrendPrediction
}

// TrendDirection defines trend directions
type TrendDirection string

const (
	TrendImproving TrendDirection = "improving"
	TrendDeclining TrendDirection = "declining"
	TrendStable    TrendDirection = "stable"
	TrendVolatile  TrendDirection = "volatile"
)

// TrendPrediction represents a trend prediction
type TrendPrediction struct {
	Time       time.Time
	Value      float64
	Confidence float64
}

// NewAgentManager creates a new agent manager
func NewAgentManager(logger *slog.Logger) (*AgentManager, error) {
	scheduler := &AgentScheduler{
		taskQueue:      make([]ScheduledTask, 0),
		agentWorkloads: make(map[string]float64),
		priorities:     make(map[string]int),
		policies:       make([]SchedulingPolicy, 0),
		metrics: SchedulingMetrics{
			LastUpdated: time.Now(),
		},
	}

	collaborator := &CollaborationEngine{
		activeCollaborations: make(map[string]*Collaboration),
		collaborationRules:   make([]CollaborationRule, 0),
		trustMatrix:          make(map[string]map[string]float64),
		metrics: CollaborationMetrics{
			LastUpdated: time.Now(),
		},
	}

	learningEngine := &LearningEngine{
		knowledgeBase: &SharedKnowledgeBase{
			Concepts:    make(map[string]SharedConcept),
			Procedures:  make(map[string]SharedProcedure),
			Experiences: make([]SharedExperience, 0),
			Patterns:    make(map[string]Pattern),
		},
		learningScheduler: &LearningScheduler{
			learningTasks: make(map[string][]LearningTask),
			schedule:      make(map[string]time.Time),
			priorities:    make(map[string]float64),
		},
		feedbackProcessor: &FeedbackProcessor{
			feedbackQueue: make([]ProcessableFeedback, 0),
			processors:    make(map[FeedbackType]FeedbackHandler),
			metrics: FeedbackMetrics{
				LastUpdated: time.Now(),
			},
		},
		transferEngine: &KnowledgeTransferEngine{
			transferRules:   make([]TransferRule, 0),
			transferHistory: make([]KnowledgeTransfer, 0),
			compatibility:   make(map[string]map[string]float64),
			metrics: TransferMetrics{
				LastUpdated: time.Now(),
			},
		},
		metrics: LearningMetrics{
			SkillImprovement: make(map[string]float64),
			LastUpdated:      time.Now(),
		},
	}

	resourceManager := &ResourceManager{
		allocatedResources: make(map[string]Resources),
		reservations:       make([]ResourceReservation, 0),
		policies:           make([]AllocationPolicy, 0),
		metrics: ResourceMetrics{
			LastUpdated: time.Now(),
		},
	}

	evaluator := &PerformanceEvaluator{
		evaluations: make(map[string][]PerformanceEvaluation),
		benchmarks:  make(map[string]Benchmark),
		comparisons: make([]AgentComparison, 0),
		trends:      make(map[string]PerformanceTrend),
	}

	return &AgentManager{
		agents:          make(map[string]Agent),
		scheduler:       scheduler,
		collaborator:    collaborator,
		learningEngine:  learningEngine,
		resourceManager: resourceManager,
		evaluator:       evaluator,
		logger:          logger,
	}, nil
}

// RegisterAgent registers a new agent with the manager
func (am *AgentManager) RegisterAgent(agent Agent) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	agentID := agent.GetID()
	if _, exists := am.agents[agentID]; exists {
		return fmt.Errorf("agent with ID %s already registered", agentID)
	}

	am.agents[agentID] = agent
	am.scheduler.agentWorkloads[agentID] = 0.0
	am.scheduler.priorities[agentID] = 1

	am.logger.Info("Agent registered",
		slog.String("agent_id", agentID),
		slog.String("name", agent.GetName()),
		slog.String("domain", string(agent.GetDomain())))

	return nil
}

// UnregisterAgent removes an agent from the manager
func (am *AgentManager) UnregisterAgent(agentID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	agent, exists := am.agents[agentID]
	if !exists {
		return fmt.Errorf("agent with ID %s not found", agentID)
	}

	// Shutdown the agent
	if err := agent.Shutdown(context.Background()); err != nil {
		am.logger.Warn("Failed to shutdown agent", slog.String("agent_id", agentID), slog.Any("error", err))
	}

	delete(am.agents, agentID)
	delete(am.scheduler.agentWorkloads, agentID)
	delete(am.scheduler.priorities, agentID)

	am.logger.Info("Agent unregistered", slog.String("agent_id", agentID))

	return nil
}

// ProcessRequest processes a request using the most suitable agent(s)
func (am *AgentManager) ProcessRequest(ctx context.Context, request *corecontext.ContextualRequest) (*AgentResponse, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	am.logger.Debug("Processing request",
		slog.String("request_id", request.Original.ID),
		slog.String("query", request.Original.Query))

	// Find suitable agents
	candidates := am.findSuitableAgents(ctx, request)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no suitable agents found for request")
	}

	// Schedule the task
	task := ScheduledTask{
		ID:                fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Request:           request,
		RequiredDomain:    am.inferDomain(request),
		Priority:          am.inferPriority(request),
		EstimatedDuration: am.estimateDuration(request),
		Status:            TaskStatusPending,
		CreatedAt:         time.Now(),
	}

	selectedAgent := am.scheduler.scheduleTask(task, candidates)
	if selectedAgent == nil {
		return nil, fmt.Errorf("failed to schedule task")
	}

	// Execute the task
	response, err := am.executeTask(ctx, selectedAgent, &task)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task: %w", err)
	}

	// Update metrics and learning
	go am.updateMetrics(selectedAgent.GetID(), &task, response)
	go am.processLearning(ctx, selectedAgent.GetID(), &task, response)

	am.logger.Info("Request processed",
		slog.String("request_id", request.Original.ID),
		slog.String("agent_id", selectedAgent.GetID()),
		slog.Float64("confidence", response.Confidence),
		slog.Bool("success", response.Success))

	return response, nil
}

// findSuitableAgents finds agents suitable for handling a request
func (am *AgentManager) findSuitableAgents(ctx context.Context, request *corecontext.ContextualRequest) []AgentCandidate {
	var candidates []AgentCandidate

	for _, agent := range am.agents {
		if agent.GetStatus() == StatusShutdown || agent.GetStatus() == StatusError {
			continue
		}

		canHandle, confidence := agent.CanHandle(ctx, request)
		if canHandle && confidence > 0.3 { // Minimum confidence threshold
			candidates = append(candidates, AgentCandidate{
				Agent:      agent,
				Confidence: confidence,
				Workload:   am.scheduler.agentWorkloads[agent.GetID()],
			})
		}
	}

	// Sort by confidence and workload
	sort.Slice(candidates, func(i, j int) bool {
		// Higher confidence and lower workload preferred
		scoreI := candidates[i].Confidence - (candidates[i].Workload * 0.3)
		scoreJ := candidates[j].Confidence - (candidates[j].Workload * 0.3)
		return scoreI > scoreJ
	})

	return candidates
}

// AgentCandidate represents a candidate agent for task execution
type AgentCandidate struct {
	Agent      Agent
	Confidence float64
	Workload   float64
}

// executeTask executes a task using the selected agent
func (am *AgentManager) executeTask(ctx context.Context, agent Agent, task *ScheduledTask) (*AgentResponse, error) {
	task.AssignedAgent = agent.GetID()
	task.Status = TaskStatusRunning
	task.StartedAt = &[]time.Time{time.Now()}[0]

	// Update agent workload
	am.scheduler.mu.Lock()
	am.scheduler.agentWorkloads[agent.GetID()] += 1.0
	am.scheduler.mu.Unlock()

	defer func() {
		// Reduce agent workload
		am.scheduler.mu.Lock()
		am.scheduler.agentWorkloads[agent.GetID()] -= 1.0
		if am.scheduler.agentWorkloads[agent.GetID()] < 0 {
			am.scheduler.agentWorkloads[agent.GetID()] = 0
		}
		am.scheduler.mu.Unlock()

		// Update task completion
		now := time.Now()
		task.CompletedAt = &now
		if task.StartedAt != nil {
			task.Status = TaskStatusCompleted
		}
	}()

	// Execute the task
	response, err := agent.Execute(ctx, task.Request)
	if err != nil {
		task.Status = TaskStatusFailed
		return nil, err
	}

	return response, nil
}

// Helper methods

func (am *AgentManager) inferDomain(request *corecontext.ContextualRequest) Domain {
	// Simple domain inference based on request content
	query := strings.ToLower(request.Original.Query)

	if strings.Contains(query, "database") || strings.Contains(query, "sql") {
		return DomainDatabase
	}
	if strings.Contains(query, "deploy") || strings.Contains(query, "kubernetes") || strings.Contains(query, "docker") {
		return DomainInfrastructure
	}
	if strings.Contains(query, "code") || strings.Contains(query, "function") || strings.Contains(query, "debug") {
		return DomainDevelopment
	}
	if strings.Contains(query, "research") || strings.Contains(query, "find") || strings.Contains(query, "search") {
		return DomainResearch
	}

	return DomainGeneral
}

func (am *AgentManager) inferPriority(request *corecontext.ContextualRequest) Priority {
	query := strings.ToLower(request.Original.Query)

	if strings.Contains(query, "urgent") || strings.Contains(query, "critical") || strings.Contains(query, "emergency") {
		return PriorityCritical
	}
	if strings.Contains(query, "important") || strings.Contains(query, "asap") {
		return PriorityHigh
	}
	if strings.Contains(query, "low priority") || strings.Contains(query, "when you can") {
		return PriorityLow
	}

	return PriorityMedium
}

func (am *AgentManager) estimateDuration(request *corecontext.ContextualRequest) time.Duration {
	// Simple duration estimation
	queryLength := len(request.Original.Query)

	if queryLength < 50 {
		return 30 * time.Second
	} else if queryLength < 200 {
		return 2 * time.Minute
	} else {
		return 5 * time.Minute
	}
}

func (am *AgentManager) updateMetrics(agentID string, task *ScheduledTask, response *AgentResponse) {
	// Update scheduling metrics
	am.scheduler.mu.Lock()
	am.scheduler.metrics.TotalTasks++
	if response.Success {
		am.scheduler.metrics.CompletedTasks++
	}
	am.scheduler.metrics.LastUpdated = time.Now()
	am.scheduler.mu.Unlock()

	// Update agent performance metrics
	// This would be more sophisticated in a full implementation
}

func (am *AgentManager) processLearning(ctx context.Context, agentID string, task *ScheduledTask, response *AgentResponse) {
	// Process learning from task execution
	// This would analyze the task and response to extract learning opportunities
}

// scheduleTask schedules a task to the best available agent
func (as *AgentScheduler) scheduleTask(task ScheduledTask, candidates []AgentCandidate) Agent {
	as.mu.Lock()
	defer as.mu.Unlock()

	if len(candidates) == 0 {
		return nil
	}

	// For now, use simple best-fit scheduling
	// In a full implementation, this would consider various policies
	selected := candidates[0]

	task.AssignedAgent = selected.Agent.GetID()
	task.ScheduledAt = &[]time.Time{time.Now()}[0]

	as.taskQueue = append(as.taskQueue, task)

	return selected.Agent
}

// GetAgents returns all registered agents
func (am *AgentManager) GetAgents() map[string]Agent {
	am.mu.RLock()
	defer am.mu.RUnlock()

	agents := make(map[string]Agent)
	for id, agent := range am.agents {
		agents[id] = agent
	}

	return agents
}

// GetAgentStatus returns the status of all agents
func (am *AgentManager) GetAgentStatus() map[string]AgentStatus {
	am.mu.RLock()
	defer am.mu.RUnlock()

	status := make(map[string]AgentStatus)
	for id, agent := range am.agents {
		status[id] = agent.GetStatus()
	}

	return status
}

// Shutdown gracefully shuts down all agents
func (am *AgentManager) Shutdown(ctx context.Context) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.logger.Info("Shutting down agent manager")

	var errors []error
	for id, agent := range am.agents {
		if err := agent.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown agent %s: %w", id, err))
			am.logger.Error("Failed to shutdown agent", slog.String("agent_id", id), slog.Any("error", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors occurred during shutdown: %v", errors)
	}

	return nil
}
