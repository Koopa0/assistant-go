package memory

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/koopa0/assistant-go/internal/core/events"
)

// ProceduralMemory stores how-to knowledge, skills, and procedures
type ProceduralMemory struct {
	procedures         map[string]*Procedure
	skills             map[string]*Skill
	workflows          map[string]*Workflow
	patterns           map[string]*Pattern
	executionEngine    *ExecutionEngine
	learningEngine     *SkillLearningEngine
	optimizationEngine *OptimizationEngine
	eventBus           *events.EventBus
	logger             *slog.Logger
	mu                 sync.RWMutex
}

// Procedure represents a sequence of steps to accomplish a task
type Procedure struct {
	ID              string
	Name            string
	Category        ProcedureCategory
	Description     string
	Steps           []Step
	Prerequisites   []string
	Outcomes        []string
	Context         ProcedureContext
	SuccessRate     float64
	AverageTime     time.Duration
	ComplexityScore float64
	Reliability     float64
	LastExecuted    *time.Time
	ExecutionCount  int
	Variations      []ProcedureVariation
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Metadata        map[string]interface{}
}

// ProcedureCategory defines categories of procedures
type ProcedureCategory string

const (
	ProcedureDevelopment     ProcedureCategory = "development"
	ProcedureDebugging       ProcedureCategory = "debugging"
	ProcedureOptimization    ProcedureCategory = "optimization"
	ProcedureDeployment      ProcedureCategory = "deployment"
	ProcedureMaintenance     ProcedureCategory = "maintenance"
	ProcedureTroubleshooting ProcedureCategory = "troubleshooting"
	ProcedureAnalysis        ProcedureCategory = "analysis"
)

// Step represents a single step in a procedure
type Step struct {
	ID              string
	Order           int
	Action          string
	Description     string
	Type            StepType
	Parameters      map[string]interface{}
	Conditions      []Condition
	ExpectedOutcome string
	Alternatives    []Alternative
	TimeEstimate    time.Duration
	SkillsRequired  []string
	ToolsRequired   []string
	RiskLevel       RiskLevel
	Checkpoints     []Checkpoint
}

// StepType defines types of steps
type StepType string

const (
	StepTypeAction       StepType = "action"
	StepTypeDecision     StepType = "decision"
	StepTypeValidation   StepType = "validation"
	StepTypeLoop         StepType = "loop"
	StepTypeParallel     StepType = "parallel"
	StepTypeConditional  StepType = "conditional"
	StepTypeSubprocedure StepType = "subprocedure"
)

// Condition represents a condition for a step
type Condition struct {
	Type       ConditionType
	Expression string
	Value      interface{}
	Required   bool
}

// ProceduralConditionType defines types of conditions for procedural memory
type ProceduralConditionType string

const (
	ProceduralConditionPrecondition  ProceduralConditionType = "precondition"
	ProceduralConditionPostcondition ProceduralConditionType = "postcondition"
	ProceduralConditionInvariant     ProceduralConditionType = "invariant"
	ProceduralConditionTrigger       ProceduralConditionType = "trigger"
)

// Alternative represents an alternative approach
type Alternative struct {
	ID          string
	Condition   string
	Steps       []Step
	Preference  float64
	SuccessRate float64
}

// RiskLevel defines risk levels
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// Checkpoint represents a validation point
type Checkpoint struct {
	ID         string
	Name       string
	Validation string
	OnFailure  string
	Critical   bool
}

// ProcedureContext defines the context for a procedure
type ProcedureContext struct {
	Domain      string
	Environment []string
	Constraints []string
	Resources   []string
}

// ProcedureVariation represents a variation of a procedure
type ProcedureVariation struct {
	ID            string
	Name          string
	Conditions    []string
	ModifiedSteps map[int]Step
	SuccessRate   float64
	UsageCount    int
}

// Skill represents a learned ability or competence
type Skill struct {
	ID                   string
	Name                 string
	Type                 SkillType
	Domain               string
	ProficiencyLevel     float64
	Components           []SkillComponent
	Prerequisites        []string
	Applications         []string
	PracticeTime         time.Duration
	LastPracticed        *time.Time
	RetentionRate        float64
	TransferabilityScore float64
	Complexity           float64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// SkillType defines types of skills
type SkillType string

const (
	SkillTypeCognitive     SkillType = "cognitive"
	SkillTypeTechnical     SkillType = "technical"
	SkillTypeAnalytical    SkillType = "analytical"
	SkillTypeCreative      SkillType = "creative"
	SkillTypeSocial        SkillType = "social"
	SkillTypeMetacognitive SkillType = "metacognitive"
)

// SkillComponent represents a component of a skill
type SkillComponent struct {
	ID          string
	Name        string
	Type        ComponentType
	Proficiency float64
	Weight      float64
}

// ComponentType defines types of skill components
type ComponentType string

const (
	ComponentKnowledge   ComponentType = "knowledge"
	ComponentApplication ComponentType = "application"
	ComponentJudgment    ComponentType = "judgment"
	ComponentAdaptation  ComponentType = "adaptation"
)

// Workflow represents a complex process with multiple procedures
type Workflow struct {
	ID                 string
	Name               string
	Description        string
	Type               WorkflowType
	Procedures         []WorkflowNode
	Transitions        []Transition
	DecisionPoints     []DecisionPoint
	DataFlow           DataFlowSpec
	ErrorHandling      ErrorHandlingStrategy
	Optimization       OptimizationStrategy
	PerformanceMetrics WorkflowMetrics
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// WorkflowType defines types of workflows
type WorkflowType string

const (
	WorkflowSequential  WorkflowType = "sequential"
	WorkflowParallel    WorkflowType = "parallel"
	WorkflowConditional WorkflowType = "conditional"
	WorkflowIterative   WorkflowType = "iterative"
	WorkflowAdaptive    WorkflowType = "adaptive"
)

// WorkflowNode represents a node in a workflow
type WorkflowNode struct {
	ID          string
	ProcedureID string
	Type        NodeType
	Position    Position
	Inputs      []string
	Outputs     []string
	Timeout     time.Duration
	Retries     int
}

// NodeType defines types of workflow nodes
type NodeType string

const (
	NodeTypeStart     NodeType = "start"
	NodeTypeEnd       NodeType = "end"
	NodeTypeProcedure NodeType = "procedure"
	NodeTypeDecision  NodeType = "decision"
	NodeTypeFork      NodeType = "fork"
	NodeTypeJoin      NodeType = "join"
)

// Position represents a position in the workflow
type Position struct {
	X int
	Y int
}

// Transition represents a transition between nodes
type Transition struct {
	ID          string
	From        string
	To          string
	Condition   string
	Probability float64
}

// DecisionPoint represents a decision in the workflow
type DecisionPoint struct {
	ID       string
	NodeID   string
	Criteria []DecisionCriterion
	Strategy DecisionStrategy
}

// DecisionCriterion represents a criterion for decision
type DecisionCriterion struct {
	Name       string
	Weight     float64
	Evaluation string
}

// DecisionStrategy defines decision strategies
type DecisionStrategy string

const (
	DecisionStrategyFirst         DecisionStrategy = "first_match"
	DecisionStrategyBest          DecisionStrategy = "best_score"
	DecisionStrategyProbabilistic DecisionStrategy = "probabilistic"
	DecisionStrategyLearned       DecisionStrategy = "learned"
)

// DataFlowSpec specifies data flow in workflow
type DataFlowSpec struct {
	Inputs          []DataSpecification
	Outputs         []DataSpecification
	Intermediates   []DataSpecification
	Transformations []DataTransformation
}

// DataSpecification specifies data requirements
type DataSpecification struct {
	Name     string
	Type     string
	Required bool
	Default  interface{}
}

// DataTransformation represents data transformation
type DataTransformation struct {
	From      string
	To        string
	Transform string
}

// ErrorHandlingStrategy defines error handling
type ErrorHandlingStrategy struct {
	Type          ErrorHandlingType
	RetryPolicy   RetryPolicy
	Fallbacks     []string
	Notifications []string
}

// ErrorHandlingType defines types of error handling
type ErrorHandlingType string

const (
	ErrorHandlingFail       ErrorHandlingType = "fail_fast"
	ErrorHandlingRetry      ErrorHandlingType = "retry"
	ErrorHandlingFallback   ErrorHandlingType = "fallback"
	ErrorHandlingCompensate ErrorHandlingType = "compensate"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries int
	Backoff    BackoffStrategy
	Timeout    time.Duration
}

// BackoffStrategy defines backoff strategies
type BackoffStrategy string

const (
	BackoffConstant    BackoffStrategy = "constant"
	BackoffLinear      BackoffStrategy = "linear"
	BackoffExponential BackoffStrategy = "exponential"
)

// OptimizationStrategy defines optimization approach
type OptimizationStrategy struct {
	Type        OptimizationType
	Objectives  []OptimizationObjective
	Constraints []OptimizationConstraint
}

// OptimizationType defines types of optimization
type OptimizationType string

const (
	OptimizationSpeed       OptimizationType = "speed"
	OptimizationReliability OptimizationType = "reliability"
	OptimizationResource    OptimizationType = "resource"
	OptimizationQuality     OptimizationType = "quality"
)

// OptimizationObjective represents an optimization goal
type OptimizationObjective struct {
	Metric    string
	Target    float64
	Weight    float64
	Direction Direction
}

// Direction defines optimization direction
type Direction string

const (
	DirectionMinimize Direction = "minimize"
	DirectionMaximize Direction = "maximize"
)

// OptimizationConstraint represents a constraint
type OptimizationConstraint struct {
	Resource string
	Limit    float64
	Type     ConstraintType
}

// WorkflowMetrics tracks workflow performance
type WorkflowMetrics struct {
	ExecutionCount    int
	SuccessRate       float64
	AverageTime       time.Duration
	ResourceUsage     map[string]float64
	BottleneckNodes   []string
	OptimizationScore float64
}

// Pattern represents a reusable pattern
type Pattern struct {
	ID              string
	Name            string
	Type            PatternType
	Context         []string
	Problem         string
	Solution        PatternSolution
	Consequences    []string
	Examples        []PatternExample
	RelatedPatterns []string
	UsageCount      int
	SuccessRate     float64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PatternType defines types of patterns
type PatternType string

const (
	PatternDesign         PatternType = "design"
	PatternArchitecture   PatternType = "architecture"
	PatternAlgorithm      PatternType = "algorithm"
	PatternProcess        PatternType = "process"
	PatternProblemSolving PatternType = "problem_solving"
)

// PatternSolution represents the solution part of a pattern
type PatternSolution struct {
	Structure      string
	Participants   []PatternParticipant
	Implementation string
	Variations     []PatternVariation
}

// PatternParticipant represents a participant in a pattern
type PatternParticipant struct {
	Name             string
	Role             string
	Responsibilities []string
}

// PatternVariation represents a variation of a pattern
type PatternVariation struct {
	Name        string
	When        string
	Differences []string
}

// PatternExample represents an example of pattern usage
type PatternExample struct {
	Context        string
	Application    string
	Result         string
	LessonsLearned []string
}

// ExecutionEngine executes procedures
type ExecutionEngine struct {
	executor  ProcedureExecutor
	validator ExecutionValidator
	monitor   ExecutionMonitor
	optimizer ExecutionOptimizer
	mu        sync.RWMutex
}

// ProcedureExecutor executes procedures
type ProcedureExecutor interface {
	Execute(ctx context.Context, procedure *Procedure, params map[string]interface{}) (*ExecutionResult, error)
	ExecuteStep(ctx context.Context, step *Step, state *ExecutionState) (*StepResult, error)
}

// ExecutionResult represents the result of procedure execution
type ExecutionResult struct {
	ProcedureID string
	Success     bool
	Duration    time.Duration
	Steps       []StepResult
	Outputs     map[string]interface{}
	Errors      []ExecutionError
	Metrics     ExecutionMetrics
}

// StepResult represents the result of step execution
type StepResult struct {
	StepID      string
	Success     bool
	Duration    time.Duration
	Output      interface{}
	Error       *ExecutionError
	Checkpoints []CheckpointResult
}

// ExecutionError represents an execution error
type ExecutionError struct {
	StepID      string
	Type        ErrorType
	Message     string
	Recoverable bool
	Timestamp   time.Time
}

// ErrorType defines types of execution errors
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeExecution  ErrorType = "execution"
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeResource   ErrorType = "resource"
)

// CheckpointResult represents checkpoint validation result
type CheckpointResult struct {
	CheckpointID string
	Passed       bool
	Message      string
}

// ExecutionState maintains execution state
type ExecutionState struct {
	CurrentStep    int
	Variables      map[string]interface{}
	History        []StepResult
	Stack          []string
	StartTime      time.Time
	LastCheckpoint string
}

// ExecutionValidator validates execution
type ExecutionValidator interface {
	ValidatePreconditions(procedure *Procedure, params map[string]interface{}) error
	ValidateStep(step *Step, state *ExecutionState) error
	ValidatePostconditions(procedure *Procedure, result *ExecutionResult) error
}

// ExecutionMonitor monitors execution
type ExecutionMonitor interface {
	StartMonitoring(executionID string)
	RecordStep(executionID string, step StepResult)
	GetMetrics(executionID string) ExecutionMetrics
	StopMonitoring(executionID string)
}

// ExecutionMetrics tracks execution metrics
type ExecutionMetrics struct {
	TotalSteps       int
	CompletedSteps   int
	FailedSteps      int
	RetryCount       int
	ResourceUsage    map[string]float64
	PerformanceScore float64
}

// ExecutionOptimizer optimizes execution
type ExecutionOptimizer interface {
	OptimizeProcedure(procedure *Procedure, history []ExecutionResult) *Procedure
	SuggestImprovements(procedure *Procedure) []Improvement
}

// Improvement represents a suggested improvement
type Improvement struct {
	Type        ImprovementType
	Target      string
	Description string
	Impact      float64
	Effort      float64
	Priority    Priority
}

// ImprovementType defines types of improvements
type ImprovementType string

const (
	ImprovementParallelization ImprovementType = "parallelization"
	ImprovementElimination     ImprovementType = "elimination"
	ImprovementReordering      ImprovementType = "reordering"
	ImprovementCaching         ImprovementType = "caching"
	ImprovementAutomation      ImprovementType = "automation"
)

// Priority defines priority levels
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// SkillLearningEngine learns new skills
type SkillLearningEngine struct {
	strategies     []LearningStrategy
	practiceEngine PracticeEngine
	evaluator      SkillEvaluator
	mu             sync.RWMutex
}

// ProceduralLearningStrategy defines how to learn skills
type ProceduralLearningStrategy interface {
	Learn(examples []SkillExample, feedback []Feedback) (*Skill, error)
	Improve(skill *Skill, practice []PracticeSession) error
}

// SkillExample represents an example for learning
type SkillExample struct {
	Context     string
	Actions     []string
	Result      string
	Success     bool
	Annotations map[string]interface{}
}

// Feedback represents feedback on skill execution
type Feedback struct {
	Type        FeedbackType
	Content     string
	Rating      float64
	Suggestions []string
	Timestamp   time.Time
}

// ProceduralFeedbackType defines types of feedback for procedural memory
type ProceduralFeedbackType string

const (
	ProceduralFeedbackPositive     ProceduralFeedbackType = "positive"
	ProceduralFeedbackNegative     ProceduralFeedbackType = "negative"
	ProceduralFeedbackConstructive ProceduralFeedbackType = "constructive"
	ProceduralFeedbackCorrectional ProceduralFeedbackType = "correctional"
)

// PracticeEngine manages skill practice
type PracticeEngine interface {
	GeneratePractice(skill *Skill) *PracticeSession
	EvaluatePractice(session *PracticeSession) *PracticeResult
}

// PracticeSession represents a practice session
type PracticeSession struct {
	ID         string
	SkillID    string
	Exercises  []Exercise
	Duration   time.Duration
	Difficulty float64
	StartTime  time.Time
	EndTime    *time.Time
}

// Exercise represents a practice exercise
type Exercise struct {
	ID          string
	Type        ExerciseType
	Description string
	Target      string
	Difficulty  float64
	TimeLimit   time.Duration
}

// ExerciseType defines types of exercises
type ExerciseType string

const (
	ExerciseRecall      ExerciseType = "recall"
	ExerciseApplication ExerciseType = "application"
	ExerciseAdaptation  ExerciseType = "adaptation"
	ExerciseIntegration ExerciseType = "integration"
)

// PracticeResult represents practice results
type PracticeResult struct {
	SessionID    string
	Score        float64
	Improvements []string
	Weaknesses   []string
	NextSteps    []string
}

// SkillEvaluator evaluates skills
type SkillEvaluator interface {
	Evaluate(skill *Skill, context string) *SkillEvaluation
	Compare(skill1, skill2 *Skill) *SkillComparison
}

// SkillEvaluation represents skill evaluation
type SkillEvaluation struct {
	SkillID          string
	ProficiencyScore float64
	Strengths        []string
	Weaknesses       []string
	ReadinessLevel   ReadinessLevel
}

// ReadinessLevel defines readiness levels
type ReadinessLevel string

const (
	ReadinessBeginner     ReadinessLevel = "beginner"
	ReadinessIntermediate ReadinessLevel = "intermediate"
	ReadinessAdvanced     ReadinessLevel = "advanced"
	ReadinessExpert       ReadinessLevel = "expert"
)

// SkillComparison compares two skills
type SkillComparison struct {
	Similarity   float64
	Differences  []string
	Transferable []string
	Synergies    []string
}

// OptimizationEngine optimizes procedures and workflows
type OptimizationEngine struct {
	optimizers []Optimizer
	analyzer   PerformanceAnalyzer
	simulator  ExecutionSimulator
	mu         sync.RWMutex
}

// Optimizer optimizes procedures
type Optimizer interface {
	Optimize(target interface{}, constraints []OptimizationConstraint) (interface{}, error)
	GetName() string
}

// PerformanceAnalyzer analyzes performance
type PerformanceAnalyzer interface {
	Analyze(history []ExecutionResult) *PerformanceAnalysis
	IdentifyBottlenecks(workflow *Workflow) []Bottleneck
}

// PerformanceAnalysis represents performance analysis
type PerformanceAnalysis struct {
	AverageTime           time.Duration
	SuccessRate           float64
	ResourceUsage         map[string]float64
	Bottlenecks           []Bottleneck
	OptimizationPotential float64
}

// Bottleneck represents a performance bottleneck
type Bottleneck struct {
	Location    string
	Type        BottleneckType
	Impact      float64
	Suggestions []string
}

// BottleneckType defines types of bottlenecks
type BottleneckType string

const (
	BottleneckTime     BottleneckType = "time"
	BottleneckResource BottleneckType = "resource"
	BottleneckSequence BottleneckType = "sequence"
	BottleneckDecision BottleneckType = "decision"
)

// ExecutionSimulator simulates execution
type ExecutionSimulator interface {
	Simulate(procedure *Procedure, scenarios []Scenario) []SimulationResult
	PredictOutcome(procedure *Procedure, params map[string]interface{}) *OutcomePrediction
}

// Scenario represents a simulation scenario
type Scenario struct {
	Name       string
	Parameters map[string]interface{}
	Conditions []string
	Expected   interface{}
}

// SimulationResult represents simulation results
type SimulationResult struct {
	ScenarioName  string
	Success       bool
	Duration      time.Duration
	ResourceUsage map[string]float64
	Deviations    []string
}

// OutcomePrediction predicts execution outcome
type OutcomePrediction struct {
	SuccessProbability float64
	ExpectedDuration   time.Duration
	RiskFactors        []string
	Confidence         float64
}

// ProceduralQuery represents a query for procedural memory
type ProceduralQuery struct {
	Type        ProceduralQueryType
	Goal        string
	Context     []string
	Constraints []string
	Preferences QueryPreferences
	Limit       int
}

// ProceduralQueryType defines types of procedural queries
type ProceduralQueryType string

const (
	ProceduralQueryHowTo       ProceduralQueryType = "how_to"
	ProceduralQueryBestWay     ProceduralQueryType = "best_way"
	ProceduralQueryAlternative ProceduralQueryType = "alternative"
	ProceduralQueryOptimal     ProceduralQueryType = "optimal"
	ProceduralQueryFastest     ProceduralQueryType = "fastest"
	ProceduralQuerySafest      ProceduralQueryType = "safest"
)

// QueryPreferences represents query preferences
type QueryPreferences struct {
	OptimizeFor  []OptimizationType
	AvoidRisks   []RiskLevel
	PreferSkills []string
	TimeLimit    time.Duration
}

// NewProceduralMemory creates a new procedural memory
func NewProceduralMemory(eventBus *events.EventBus, logger *slog.Logger) (*ProceduralMemory, error) {
	executionEngine := &ExecutionEngine{
		// Initialize with default implementations
	}

	learningEngine := &SkillLearningEngine{
		strategies: make([]LearningStrategy, 0),
	}

	optimizationEngine := &OptimizationEngine{
		optimizers: make([]Optimizer, 0),
	}

	pm := &ProceduralMemory{
		procedures:         make(map[string]*Procedure),
		skills:             make(map[string]*Skill),
		workflows:          make(map[string]*Workflow),
		patterns:           make(map[string]*Pattern),
		executionEngine:    executionEngine,
		learningEngine:     learningEngine,
		optimizationEngine: optimizationEngine,
		eventBus:           eventBus,
		logger:             logger,
	}

	// Start background processes
	go pm.runMaintenanceLoop()
	go pm.runOptimizationLoop()

	return pm, nil
}

// StoreProcedure stores a procedure
func (pm *ProceduralMemory) StoreProcedure(ctx context.Context, procedure *Procedure) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Update timestamps
	now := time.Now()
	if procedure.CreatedAt.IsZero() {
		procedure.CreatedAt = now
	}
	procedure.UpdatedAt = now

	// Store procedure
	pm.procedures[procedure.ID] = procedure

	pm.logger.Info("Stored procedure",
		slog.String("procedure_id", procedure.ID),
		slog.String("name", procedure.Name),
		slog.String("category", string(procedure.Category)))

	// Publish event
	if pm.eventBus != nil {
		event := events.Event{
			Type:   events.EventCustom,
			Source: "procedural_memory",
			Data: map[string]interface{}{
				"action":       "store_procedure",
				"procedure_id": procedure.ID,
				"category":     procedure.Category,
			},
		}
		pm.eventBus.Publish(ctx, event)
	}

	return nil
}

// GetProcedure retrieves a procedure by ID
func (pm *ProceduralMemory) GetProcedure(ctx context.Context, procedureID string) (*Procedure, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	procedure, exists := pm.procedures[procedureID]
	if !exists {
		return nil, fmt.Errorf("procedure not found: %s", procedureID)
	}

	return procedure, nil
}

// Query searches procedural memory
func (pm *ProceduralMemory) Query(ctx context.Context, query ProceduralQuery) ([]*Procedure, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var results []*Procedure

	// Filter procedures based on query
	for _, procedure := range pm.procedures {
		if pm.matchesQuery(procedure, query) {
			results = append(results, procedure)
		}
	}

	// Sort and rank results
	results = pm.rankResults(results, query)

	// Apply limit
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	pm.logger.Debug("Queried procedural memory",
		slog.String("type", string(query.Type)),
		slog.Int("results", len(results)))

	return results, nil
}

// ExecuteProcedure executes a procedure
func (pm *ProceduralMemory) ExecuteProcedure(ctx context.Context, procedureID string, params map[string]interface{}) (*ExecutionResult, error) {
	pm.mu.RLock()
	procedure, exists := pm.procedures[procedureID]
	pm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("procedure not found: %s", procedureID)
	}

	// Validate preconditions
	if pm.executionEngine.validator != nil {
		if err := pm.executionEngine.validator.ValidatePreconditions(procedure, params); err != nil {
			return nil, fmt.Errorf("precondition validation failed: %w", err)
		}
	}

	// Execute procedure
	result, err := pm.executionEngine.executor.Execute(ctx, procedure, params)
	if err != nil {
		return nil, err
	}

	// Update procedure statistics
	pm.mu.Lock()
	procedure.ExecutionCount++
	now := time.Now()
	procedure.LastExecuted = &now
	if result.Success {
		procedure.SuccessRate = (procedure.SuccessRate*float64(procedure.ExecutionCount-1) + 1.0) / float64(procedure.ExecutionCount)
	} else {
		procedure.SuccessRate = (procedure.SuccessRate * float64(procedure.ExecutionCount-1)) / float64(procedure.ExecutionCount)
	}
	pm.mu.Unlock()

	pm.logger.Info("Executed procedure",
		slog.String("procedure_id", procedureID),
		slog.Bool("success", result.Success),
		slog.Duration("duration", result.Duration))

	return result, nil
}

// LearnProcedure learns a new procedure from examples
func (pm *ProceduralMemory) LearnProcedure(ctx context.Context, examples []ExecutionExample, context string) (*Procedure, error) {
	// Analyze examples to extract procedure
	procedure := pm.extractProcedureFromExamples(examples, context)

	// Validate and optimize the learned procedure
	if pm.optimizationEngine != nil {
		optimized, err := pm.optimizationEngine.optimizers[0].Optimize(procedure, nil)
		if err == nil {
			procedure = optimized.(*Procedure)
		}
	}

	// Store the learned procedure
	if err := pm.StoreProcedure(ctx, procedure); err != nil {
		return nil, err
	}

	pm.logger.Info("Learned new procedure",
		slog.String("procedure_id", procedure.ID),
		slog.String("name", procedure.Name),
		slog.Int("steps", len(procedure.Steps)))

	return procedure, nil
}

// StoreSkill stores a skill
func (pm *ProceduralMemory) StoreSkill(ctx context.Context, skill *Skill) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Update timestamps
	now := time.Now()
	if skill.CreatedAt.IsZero() {
		skill.CreatedAt = now
	}
	skill.UpdatedAt = now

	// Store skill
	pm.skills[skill.ID] = skill

	pm.logger.Info("Stored skill",
		slog.String("skill_id", skill.ID),
		slog.String("name", skill.Name),
		slog.Float64("proficiency", skill.ProficiencyLevel))

	return nil
}

// GetSkill retrieves a skill by ID
func (pm *ProceduralMemory) GetSkill(ctx context.Context, skillID string) (*Skill, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	skill, exists := pm.skills[skillID]
	if !exists {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}

	return skill, nil
}

// PracticeSkill practices a skill
func (pm *ProceduralMemory) PracticeSkill(ctx context.Context, skillID string) (*PracticeResult, error) {
	pm.mu.RLock()
	skill, exists := pm.skills[skillID]
	pm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}

	// Generate practice session
	session := pm.learningEngine.practiceEngine.GeneratePractice(skill)

	// Evaluate practice
	result := pm.learningEngine.practiceEngine.EvaluatePractice(session)

	// Update skill based on practice
	pm.mu.Lock()
	now := time.Now()
	skill.LastPracticed = &now
	skill.PracticeTime += session.Duration
	// Adjust proficiency based on practice result
	if result.Score > 0.8 {
		skill.ProficiencyLevel = min(1.0, skill.ProficiencyLevel*1.05)
	}
	pm.mu.Unlock()

	pm.logger.Info("Practiced skill",
		slog.String("skill_id", skillID),
		slog.Float64("score", result.Score),
		slog.Duration("duration", session.Duration))

	return result, nil
}

// StoreWorkflow stores a workflow
func (pm *ProceduralMemory) StoreWorkflow(ctx context.Context, workflow *Workflow) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Update timestamps
	now := time.Now()
	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = now
	}
	workflow.UpdatedAt = now

	// Store workflow
	pm.workflows[workflow.ID] = workflow

	pm.logger.Info("Stored workflow",
		slog.String("workflow_id", workflow.ID),
		slog.String("name", workflow.Name),
		slog.String("type", string(workflow.Type)))

	return nil
}

// ExecuteWorkflow executes a workflow
func (pm *ProceduralMemory) ExecuteWorkflow(ctx context.Context, workflowID string, inputs map[string]interface{}) (*WorkflowExecutionResult, error) {
	pm.mu.RLock()
	workflow, exists := pm.workflows[workflowID]
	pm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	// Execute workflow nodes based on type
	result := &WorkflowExecutionResult{
		WorkflowID:  workflowID,
		StartTime:   time.Now(),
		NodeResults: make(map[string]*ExecutionResult),
	}

	// Simple sequential execution for now
	for _, node := range workflow.Procedures {
		if node.Type == NodeTypeProcedure {
			nodeResult, err := pm.ExecuteProcedure(ctx, node.ProcedureID, inputs)
			if err != nil {
				result.Success = false
				result.Error = err
				break
			}
			result.NodeResults[node.ID] = nodeResult

			// Update inputs for next node
			for k, v := range nodeResult.Outputs {
				inputs[k] = v
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if result.Error == nil {
		result.Success = true
	}

	return result, nil
}

// WorkflowExecutionResult represents workflow execution result
type WorkflowExecutionResult struct {
	WorkflowID  string
	Success     bool
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	NodeResults map[string]*ExecutionResult
	Outputs     map[string]interface{}
	Error       error
}

// StorePattern stores a pattern
func (pm *ProceduralMemory) StorePattern(ctx context.Context, pattern *Pattern) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Update timestamps
	now := time.Now()
	if pattern.CreatedAt.IsZero() {
		pattern.CreatedAt = now
	}
	pattern.UpdatedAt = now

	// Store pattern
	pm.patterns[pattern.ID] = pattern

	pm.logger.Info("Stored pattern",
		slog.String("pattern_id", pattern.ID),
		slog.String("name", pattern.Name),
		slog.String("type", string(pattern.Type)))

	return nil
}

// FindPatterns finds applicable patterns
func (pm *ProceduralMemory) FindPatterns(ctx context.Context, context string, problem string) ([]*Pattern, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var results []*Pattern

	for _, pattern := range pm.patterns {
		if pm.patternApplies(pattern, context, problem) {
			results = append(results, pattern)
		}
	}

	// Sort by success rate and usage
	sort.Slice(results, func(i, j int) bool {
		scoreI := results[i].SuccessRate * float64(results[i].UsageCount)
		scoreJ := results[j].SuccessRate * float64(results[j].UsageCount)
		return scoreI > scoreJ
	})

	return results, nil
}

// GetMetrics returns procedural memory metrics
func (pm *ProceduralMemory) GetMetrics() ProceduralMemoryMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metrics := ProceduralMemoryMetrics{
		TotalProcedures: len(pm.procedures),
		TotalSkills:     len(pm.skills),
		TotalWorkflows:  len(pm.workflows),
		TotalPatterns:   len(pm.patterns),
	}

	// Calculate average metrics
	totalSuccessRate := 0.0
	totalExecution := 0

	for _, proc := range pm.procedures {
		totalSuccessRate += proc.SuccessRate
		totalExecution += proc.ExecutionCount
	}

	if len(pm.procedures) > 0 {
		metrics.AverageSuccessRate = totalSuccessRate / float64(len(pm.procedures))
	}

	// Count by category
	metrics.ProceduresByCategory = make(map[ProcedureCategory]int)
	for _, proc := range pm.procedures {
		metrics.ProceduresByCategory[proc.Category]++
	}

	// Skill proficiency distribution
	metrics.SkillsByProficiency = make(map[string]int)
	for _, skill := range pm.skills {
		level := "beginner"
		if skill.ProficiencyLevel > 0.8 {
			level = "expert"
		} else if skill.ProficiencyLevel > 0.6 {
			level = "advanced"
		} else if skill.ProficiencyLevel > 0.4 {
			level = "intermediate"
		}
		metrics.SkillsByProficiency[level]++
	}

	return metrics
}

// ProceduralMemoryMetrics contains metrics about procedural memory
type ProceduralMemoryMetrics struct {
	TotalProcedures      int
	TotalSkills          int
	TotalWorkflows       int
	TotalPatterns        int
	AverageSuccessRate   float64
	ProceduresByCategory map[ProcedureCategory]int
	SkillsByProficiency  map[string]int
}

// Helper methods

func (pm *ProceduralMemory) matchesQuery(procedure *Procedure, query ProceduralQuery) bool {
	// Check if procedure matches query goal
	if query.Goal != "" && !pm.matchesGoal(procedure, query.Goal) {
		return false
	}

	// Check context match
	if len(query.Context) > 0 && !pm.matchesContext(procedure, query.Context) {
		return false
	}

	// Check constraints
	if len(query.Constraints) > 0 && !pm.meetsConstraints(procedure, query.Constraints) {
		return false
	}

	return true
}

func (pm *ProceduralMemory) matchesGoal(procedure *Procedure, goal string) bool {
	// Simple keyword matching for now
	goal = strings.ToLower(goal)
	return strings.Contains(strings.ToLower(procedure.Name), goal) ||
		strings.Contains(strings.ToLower(procedure.Description), goal)
}

func (pm *ProceduralMemory) matchesContext(procedure *Procedure, context []string) bool {
	for _, ctx := range context {
		if procedure.Context.Domain == ctx {
			return true
		}
		for _, env := range procedure.Context.Environment {
			if env == ctx {
				return true
			}
		}
	}
	return false
}

func (pm *ProceduralMemory) meetsConstraints(procedure *Procedure, constraints []string) bool {
	// Check if procedure meets all constraints
	for _, constraint := range constraints {
		// Simple constraint checking
		if strings.Contains(constraint, "time<") {
			// Check time constraint
			// Implementation would parse and check actual time
		}
		if strings.Contains(constraint, "risk=low") && pm.hasHighRiskSteps(procedure) {
			return false
		}
	}
	return true
}

func (pm *ProceduralMemory) hasHighRiskSteps(procedure *Procedure) bool {
	for _, step := range procedure.Steps {
		if step.RiskLevel == RiskLevelHigh || step.RiskLevel == RiskLevelCritical {
			return true
		}
	}
	return false
}

func (pm *ProceduralMemory) rankResults(procedures []*Procedure, query ProceduralQuery) []*Procedure {
	// Score and sort procedures based on query preferences
	type scoredProcedure struct {
		procedure *Procedure
		score     float64
	}

	scored := make([]scoredProcedure, len(procedures))
	for i, proc := range procedures {
		score := pm.scoreProcedure(proc, query)
		scored[i] = scoredProcedure{procedure: proc, score: score}
	}

	// Sort by score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Extract sorted procedures
	result := make([]*Procedure, len(scored))
	for i, s := range scored {
		result[i] = s.procedure
	}

	return result
}

func (pm *ProceduralMemory) scoreProcedure(procedure *Procedure, query ProceduralQuery) float64 {
	score := 0.0

	// Base score from success rate
	score += procedure.SuccessRate * 0.3

	// Score based on query type
	switch query.Type {
	case ProceduralQueryFastest:
		// Prefer procedures with shorter average time
		if procedure.AverageTime < 1*time.Minute {
			score += 0.3
		}
	case ProceduralQuerySafest:
		// Prefer procedures without high-risk steps
		if !pm.hasHighRiskSteps(procedure) {
			score += 0.4
		}
	case ProceduralQueryOptimal:
		// Balance between success rate and efficiency
		score += procedure.Reliability * 0.2
		score += (1.0 - procedure.ComplexityScore) * 0.2
	}

	// Score based on usage
	if procedure.ExecutionCount > 10 {
		score += 0.1
	}

	return score
}

func (pm *ProceduralMemory) patternApplies(pattern *Pattern, context string, problem string) bool {
	// Check if pattern context matches
	for _, ctx := range pattern.Context {
		if strings.Contains(strings.ToLower(context), strings.ToLower(ctx)) {
			return true
		}
	}

	// Check if pattern addresses the problem
	if strings.Contains(strings.ToLower(pattern.Problem), strings.ToLower(problem)) {
		return true
	}

	return false
}

func (pm *ProceduralMemory) extractProcedureFromExamples(examples []ExecutionExample, context string) *Procedure {
	// Simplified procedure extraction
	procedure := &Procedure{
		ID:          fmt.Sprintf("proc_%d", time.Now().UnixNano()),
		Name:        fmt.Sprintf("Learned procedure for %s", context),
		Category:    ProcedureDevelopment,
		Description: "Automatically learned procedure",
		Steps:       make([]Step, 0),
		CreatedAt:   time.Now(),
	}

	// Extract common steps from examples
	// This is a simplified implementation
	for i, example := range examples {
		if len(example.Steps) > 0 {
			for j, stepAction := range example.Steps {
				step := Step{
					ID:          fmt.Sprintf("step_%d_%d", i, j),
					Order:       j + 1,
					Action:      stepAction.Action,
					Description: stepAction.Description,
					Type:        StepTypeAction,
				}
				procedure.Steps = append(procedure.Steps, step)
			}
			break // Use first example as template for now
		}
	}

	return procedure
}

// ExecutionExample represents an example execution
type ExecutionExample struct {
	Context string
	Steps   []StepExample
	Result  interface{}
	Success bool
}

// StepExample represents an example step
type StepExample struct {
	Action      string
	Description string
	Parameters  map[string]interface{}
	Result      interface{}
}

func (pm *ProceduralMemory) runMaintenanceLoop() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		pm.mu.Lock()

		// Update skill retention
		for _, skill := range pm.skills {
			if skill.LastPracticed != nil {
				daysSinceLastPractice := time.Since(*skill.LastPracticed).Hours() / 24
				retentionDecay := 1.0 - (daysSinceLastPractice * 0.01) // 1% decay per day
				if retentionDecay < 0.5 {
					retentionDecay = 0.5 // Minimum 50% retention
				}
				skill.RetentionRate = retentionDecay
			}
		}

		pm.mu.Unlock()
	}
}

func (pm *ProceduralMemory) runOptimizationLoop() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// Analyze and optimize frequently used procedures
		pm.optimizeFrequentProcedures()
	}
}

func (pm *ProceduralMemory) optimizeFrequentProcedures() {
	pm.mu.RLock()

	// Find procedures that need optimization
	candidatesForOptimization := make([]*Procedure, 0)
	for _, proc := range pm.procedures {
		if proc.ExecutionCount > 50 && proc.SuccessRate < 0.8 {
			candidatesForOptimization = append(candidatesForOptimization, proc)
		}
	}

	pm.mu.RUnlock()

	// Optimize each candidate
	for _, proc := range candidatesForOptimization {
		if pm.optimizationEngine != nil && len(pm.optimizationEngine.optimizers) > 0 {
			optimized, err := pm.optimizationEngine.optimizers[0].Optimize(proc, nil)
			if err == nil {
				pm.mu.Lock()
				pm.procedures[proc.ID] = optimized.(*Procedure)
				pm.mu.Unlock()

				pm.logger.Info("Optimized procedure",
					slog.String("procedure_id", proc.ID),
					slog.Float64("previous_success_rate", proc.SuccessRate))
			}
		}
	}
}

// min is defined in types.go

// Close gracefully shuts down procedural memory
func (pm *ProceduralMemory) Close() error {
	pm.logger.Info("Procedural memory shut down")
	return nil
}
