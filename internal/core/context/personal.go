package context

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

// PersonalContext manages personal preferences, habits, and learned behaviors
type PersonalContext struct {
	preferences     *UserPreferences
	habits          *DevelopmentHabits
	learningProfile *LearningProfile
	adaptations     *AdaptationEngine
	logger          *slog.Logger
	mu              sync.RWMutex
}

// UserPreferences stores user preferences and settings
type UserPreferences struct {
	General     GeneralPreferences
	Development DevelopmentPreferences
	Interface   InterfacePreferences
	Tools       ToolPreferences
	AI          AIPreferences
	mu          sync.RWMutex
}

// GeneralPreferences contains general user preferences
type GeneralPreferences struct {
	Language          string
	Timezone          string
	WorkingHours      WorkingHours
	NotificationLevel NotificationLevel
	FeedbackFrequency FeedbackFrequency
	PrivacyLevel      PrivacyLevel
}

// WorkingHours defines user's working schedule
type WorkingHours struct {
	Start    time.Time
	End      time.Time
	Timezone string
	Weekdays []time.Weekday
	Breaks   []TimeSlot
}

// TimeSlot represents a time period
type TimeSlot struct {
	Start time.Time
	End   time.Time
	Name  string
}

// DevelopmentPreferences contains development-specific preferences
type DevelopmentPreferences struct {
	PreferredLanguages  []string
	PreferredFrameworks []string
	CodingStyle         CodingStyle
	TestingApproach     TestingApproach
	DocumentationStyle  DocumentationStyle
	CodeReviewStyle     CodeReviewStyle
	DeploymentStyle     DeploymentStyle
}

// CodingStyle defines coding preferences
type CodingStyle struct {
	IndentStyle      string // "tabs" or "spaces"
	IndentSize       int
	LineLength       int
	NamingConvention string
	CommentStyle     string
	ErrorHandling    string
}

// TestingApproach defines testing preferences
type TestingApproach string

const (
	TestingTDD         TestingApproach = "tdd"
	TestingBDD         TestingApproach = "bdd"
	TestingIntegration TestingApproach = "integration"
	TestingManual      TestingApproach = "manual"
)

// DocumentationStyle defines documentation preferences
type DocumentationStyle string

const (
	DocumentationMinimal  DocumentationStyle = "minimal"
	DocumentationBalanced DocumentationStyle = "balanced"
	DocumentationDetailed DocumentationStyle = "detailed"
	DocumentationInline   DocumentationStyle = "inline"
	DocumentationExternal DocumentationStyle = "external"
)

// CodeReviewStyle defines code review preferences
type CodeReviewStyle string

const (
	ReviewStrict   CodeReviewStyle = "strict"
	ReviewBalanced CodeReviewStyle = "balanced"
	ReviewLenient  CodeReviewStyle = "lenient"
)

// DeploymentStyle defines deployment preferences
type DeploymentStyle string

const (
	DeploymentCI      DeploymentStyle = "ci_cd"
	DeploymentManual  DeploymentStyle = "manual"
	DeploymentStaging DeploymentStyle = "staging_first"
)

// InterfacePreferences contains UI/UX preferences
type InterfacePreferences struct {
	ResponseLength     ResponseLength
	ExplanationLevel   ExplanationLevel
	VisualizationStyle VisualizationStyle
	InteractionMode    InteractionMode
}

// ResponseLength defines preferred response length
type ResponseLength string

const (
	ResponseBrief    ResponseLength = "brief"
	ResponseModerate ResponseLength = "moderate"
	ResponseDetailed ResponseLength = "detailed"
)

// ExplanationLevel defines preferred explanation depth
type ExplanationLevel string

const (
	ExplanationBasic        ExplanationLevel = "basic"
	ExplanationIntermediate ExplanationLevel = "intermediate"
	ExplanationAdvanced     ExplanationLevel = "advanced"
)

// VisualizationStyle defines preferred visualization
type VisualizationStyle string

const (
	VisualizationText     VisualizationStyle = "text"
	VisualizationDiagrams VisualizationStyle = "diagrams"
	VisualizationMixed    VisualizationStyle = "mixed"
)

// InteractionMode defines preferred interaction style
type InteractionMode string

const (
	InteractionDirective     InteractionMode = "directive"
	InteractionCollaborative InteractionMode = "collaborative"
	InteractionGuided        InteractionMode = "guided"
)

// ToolPreferences contains tool-specific preferences
type ToolPreferences struct {
	PreferredEditor   string
	PreferredTerminal string
	PreferredBrowser  string
	PreferredVCS      string
	DatabaseTools     []string
	MonitoringTools   []string
	DeploymentTools   []string
	CustomTools       map[string]ToolConfig
}

// ToolConfig defines configuration for a specific tool
type ToolConfig struct {
	Name       string
	Version    string
	ConfigPath string
	Settings   map[string]interface{}
	LastUsed   time.Time
	UsageCount int
}

// AIPreferences contains AI interaction preferences
type AIPreferences struct {
	ModelPreference     string
	CreativityLevel     float64
	ProactivityLevel    float64
	ExplanationDetail   float64
	CodeSuggestionStyle CodeSuggestionStyle
	LearningRate        float64
}

// CodeSuggestionStyle defines how code suggestions are presented
type CodeSuggestionStyle string

const (
	SuggestionInline      CodeSuggestionStyle = "inline"
	SuggestionSeparate    CodeSuggestionStyle = "separate"
	SuggestionInteractive CodeSuggestionStyle = "interactive"
)

// NotificationLevel defines notification preferences
type NotificationLevel string

const (
	NotificationAll       NotificationLevel = "all"
	NotificationImportant NotificationLevel = "important"
	NotificationMinimal   NotificationLevel = "minimal"
)

// FeedbackFrequency defines feedback frequency preferences
type FeedbackFrequency string

const (
	FeedbackContinuous FeedbackFrequency = "continuous"
	FeedbackRegular    FeedbackFrequency = "regular"
	FeedbackOnRequest  FeedbackFrequency = "on_request"
)

// PrivacyLevel defines privacy preferences
type PrivacyLevel string

const (
	PrivacyPublic  PrivacyLevel = "public"
	PrivacyLimited PrivacyLevel = "limited"
	PrivacyPrivate PrivacyLevel = "private"
)

// DevelopmentHabits tracks learned development patterns and habits
type DevelopmentHabits struct {
	WorkflowPatterns    map[string]UserWorkflowPattern
	ToolUsagePatterns   map[string]ToolUsagePattern
	TimePatterns        map[string]TimePattern
	ErrorPatterns       map[string]ErrorPattern
	ProductivityMetrics ProductivityMetrics
	mu                  sync.RWMutex
}

// UserWorkflowPattern represents a learned workflow pattern
type UserWorkflowPattern struct {
	ID         string
	Name       string
	Steps      []WorkflowStep
	Frequency  int
	Success    float64
	LastUsed   time.Time
	Variations []WorkflowVariation
}

// WorkflowVariation represents a variation of a workflow
type WorkflowVariation struct {
	Condition    string
	Modification string
	Success      float64
}

// ToolUsagePattern represents tool usage patterns
type ToolUsagePattern struct {
	ToolName      string
	UsageContext  []string
	Frequency     map[string]int
	Effectiveness float64
	LastUsed      time.Time
}

// TimePattern represents time-based behavioral patterns
type TimePattern struct {
	TimeOfDay    string
	ActivityType string
	Productivity float64
	Frequency    int
	Confidence   float64
}

// ErrorPattern represents common error patterns and solutions
type ErrorPattern struct {
	ErrorType  string
	Context    string
	Solution   string
	Frequency  int
	LastSeen   time.Time
	Confidence float64
}

// ProductivityMetrics tracks productivity patterns
type ProductivityMetrics struct {
	AverageSessionLength time.Duration
	TaskCompletionRate   float64
	CodeQualityScore     float64
	LearningVelocity     float64
	FocusPatterns        map[string]float64
}

// LearningProfile tracks user's learning patterns and preferences
type LearningProfile struct {
	LearningStyle     LearningStyle
	PreferredSources  []string
	SkillLevels       map[string]SkillLevel
	LearningGoals     []LearningGoal
	KnowledgeGaps     []KnowledgeGap
	LearningHistory   []LearningEvent
	AdaptationHistory []AdaptationEvent
	mu                sync.RWMutex
}

// LearningStyle defines how the user prefers to learn
type LearningStyle struct {
	Primary    string // "visual", "auditory", "kinesthetic", "reading"
	Secondary  string
	Examples   bool // Prefers examples
	StepByStep bool // Prefers step-by-step instructions
	BigPicture bool // Prefers big picture first
}

// SkillLevel represents proficiency in a skill
type SkillLevel struct {
	Skill       string
	Level       string // "beginner", "intermediate", "advanced", "expert"
	Confidence  float64
	LastUpdated time.Time
	Evidence    []string
}

// LearningGoal represents a learning objective
type LearningGoal struct {
	ID        string
	Skill     string
	Target    string
	Deadline  time.Time
	Progress  float64
	Priority  string
	Resources []string
}

// KnowledgeGap represents an identified knowledge gap
type KnowledgeGap struct {
	ID         string
	Area       string
	Severity   float64
	Impact     float64
	Identified time.Time
	Addressed  bool
	Resources  []string
}

// LearningEvent represents a learning activity
type LearningEvent struct {
	ID        string
	Skill     string
	Activity  string
	Outcome   string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// AdaptationEvent represents system adaptation to user
type AdaptationEvent struct {
	ID            string
	Type          string
	Before        interface{}
	After         interface{}
	Reason        string
	Timestamp     time.Time
	Effectiveness float64
}

// AdaptationEngine manages system adaptation to user behavior
type AdaptationEngine struct {
	Adaptations      map[string]Adaptation
	Rules            []AdaptationRule
	ActiveStrategies []AdaptationStrategy
	mu               sync.RWMutex
}

// Adaptation represents a system adaptation
type Adaptation struct {
	ID          string
	Type        AdaptationType
	Trigger     string
	Action      string
	Confidence  float64
	Success     float64
	LastApplied time.Time
}

// AdaptationType defines types of adaptations
type AdaptationType string

const (
	AdaptationResponse  AdaptationType = "response"
	AdaptationTool      AdaptationType = "tool"
	AdaptationWorkflow  AdaptationType = "workflow"
	AdaptationInterface AdaptationType = "interface"
	AdaptationLearning  AdaptationType = "learning"
)

// AdaptationRule defines rules for system adaptation
type AdaptationRule struct {
	ID        string
	Condition string
	Action    string
	Priority  int
	Active    bool
}

// AdaptationStrategy defines adaptation strategies
type AdaptationStrategy struct {
	ID          string
	Name        string
	Description string
	Rules       []string
	Success     float64
	Active      bool
}

// PersonalPreferencesSnapshot contains a snapshot of user preferences without mutexes
type PersonalPreferencesSnapshot struct {
	General     GeneralPreferences
	Development DevelopmentPreferences
	Interface   InterfacePreferences
	Tools       ToolPreferences
	AI          AIPreferences
}

// PersonalInfo contains relevant personal context for a request
type PersonalInfo struct {
	PreferencesSnapshot PersonalPreferencesSnapshot
	RelevantHabits      []string
	SkillContext        []SkillLevel
	LearningContext     []LearningGoal
	Adaptations         []string
	PersonalityScore    float64
}

// PersonalState represents current personal context state
type PersonalState struct {
	PreferencesSnapshot PersonalPreferencesSnapshot
	ActiveHabits        []string
	CurrentGoals        []LearningGoal
	ActiveAdaptations   []string
	LastUpdate          time.Time
}

// NewPersonalContext creates a new personal context
func NewPersonalContext(logger *slog.Logger) (*PersonalContext, error) {
	preferences := &UserPreferences{
		General: GeneralPreferences{
			Language:          "en",
			Timezone:          "UTC",
			NotificationLevel: NotificationImportant,
			FeedbackFrequency: FeedbackRegular,
			PrivacyLevel:      PrivacyLimited,
		},
		Development: DevelopmentPreferences{
			PreferredLanguages:  []string{"go"},
			PreferredFrameworks: []string{},
			CodingStyle: CodingStyle{
				IndentStyle:      "tabs",
				IndentSize:       4,
				LineLength:       100,
				NamingConvention: "camelCase",
				CommentStyle:     "descriptive",
				ErrorHandling:    "explicit",
			},
			TestingApproach:    TestingTDD,
			DocumentationStyle: DocumentationBalanced,
			CodeReviewStyle:    ReviewBalanced,
			DeploymentStyle:    DeploymentCI,
		},
		Interface: InterfacePreferences{
			ResponseLength:     ResponseModerate,
			ExplanationLevel:   ExplanationIntermediate,
			VisualizationStyle: VisualizationMixed,
			InteractionMode:    InteractionCollaborative,
		},
		Tools: ToolPreferences{
			PreferredEditor:   "vscode",
			PreferredTerminal: "zsh",
			PreferredBrowser:  "chrome",
			PreferredVCS:      "git",
			CustomTools:       make(map[string]ToolConfig),
		},
		AI: AIPreferences{
			ModelPreference:     "claude",
			CreativityLevel:     0.5,
			ProactivityLevel:    0.6,
			ExplanationDetail:   0.7,
			CodeSuggestionStyle: SuggestionInteractive,
			LearningRate:        0.5,
		},
	}

	habits := &DevelopmentHabits{
		WorkflowPatterns:  make(map[string]UserWorkflowPattern),
		ToolUsagePatterns: make(map[string]ToolUsagePattern),
		TimePatterns:      make(map[string]TimePattern),
		ErrorPatterns:     make(map[string]ErrorPattern),
		ProductivityMetrics: ProductivityMetrics{
			FocusPatterns: make(map[string]float64),
		},
	}

	learningProfile := &LearningProfile{
		LearningStyle: LearningStyle{
			Primary:    "visual",
			Secondary:  "kinesthetic",
			Examples:   true,
			StepByStep: true,
			BigPicture: false,
		},
		PreferredSources:  []string{"documentation", "examples", "interactive"},
		SkillLevels:       make(map[string]SkillLevel),
		LearningGoals:     make([]LearningGoal, 0),
		KnowledgeGaps:     make([]KnowledgeGap, 0),
		LearningHistory:   make([]LearningEvent, 0),
		AdaptationHistory: make([]AdaptationEvent, 0),
	}

	adaptations := &AdaptationEngine{
		Adaptations:      make(map[string]Adaptation),
		Rules:            make([]AdaptationRule, 0),
		ActiveStrategies: make([]AdaptationStrategy, 0),
	}

	return &PersonalContext{
		preferences:     preferences,
		habits:          habits,
		learningProfile: learningProfile,
		adaptations:     adaptations,
		logger:          logger,
	}, nil
}

// GetPreferences gets relevant personal preferences for a request
func (pc *PersonalContext) GetPreferences(ctx context.Context, request Request) (PersonalInfo, error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	pc.logger.Debug("Getting personal preferences",
		slog.String("request_id", request.ID),
		slog.String("query", request.Query))

	// Get relevant habits based on request context
	relevantHabits := pc.getRelevantHabits(request.Query)

	// Get skill context
	skillContext := pc.getSkillContext(request.Query)

	// Get learning context
	learningContext := pc.getLearningContext(request.Query)

	// Get relevant adaptations
	adaptations := pc.getRelevantAdaptations(request.Query)

	// Calculate personality score based on request alignment
	personalityScore := pc.calculatePersonalityAlignment(request)

	// Create a snapshot of preferences without mutexes
	pc.preferences.mu.RLock()
	preferencesSnapshot := PersonalPreferencesSnapshot{
		General:     pc.preferences.General,
		Development: pc.preferences.Development,
		Interface:   pc.preferences.Interface,
		Tools:       pc.preferences.Tools,
		AI:          pc.preferences.AI,
	}
	pc.preferences.mu.RUnlock()

	info := PersonalInfo{
		PreferencesSnapshot: preferencesSnapshot,
		RelevantHabits:      relevantHabits,
		SkillContext:        skillContext,
		LearningContext:     learningContext,
		Adaptations:         adaptations,
		PersonalityScore:    personalityScore,
	}

	pc.logger.Info("Personal preferences retrieved",
		slog.String("request_id", request.ID),
		slog.Int("habits", len(relevantHabits)),
		slog.Int("skills", len(skillContext)),
		slog.Float64("personality_score", personalityScore))

	return info, nil
}

// ProcessUpdate processes a personal context update
func (pc *PersonalContext) ProcessUpdate(ctx context.Context, update ContextUpdate) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	switch update.Type {
	case PreferenceChange:
		return pc.handlePreferenceChange(update)
	default:
		// Learn from other types of updates
		return pc.learnFromUpdate(update)
	}
}

// GetCurrentState returns current personal state
func (pc *PersonalContext) GetCurrentState() PersonalState {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	activeHabits := pc.getActiveHabits()
	currentGoals := pc.getCurrentGoals()
	activeAdaptations := pc.getActiveAdaptations()

	// Create a snapshot of preferences without mutexes
	pc.preferences.mu.RLock()
	preferencesSnapshot := PersonalPreferencesSnapshot{
		General:     pc.preferences.General,
		Development: pc.preferences.Development,
		Interface:   pc.preferences.Interface,
		Tools:       pc.preferences.Tools,
		AI:          pc.preferences.AI,
	}
	pc.preferences.mu.RUnlock()

	return PersonalState{
		PreferencesSnapshot: preferencesSnapshot,
		ActiveHabits:        activeHabits,
		CurrentGoals:        currentGoals,
		ActiveAdaptations:   activeAdaptations,
		LastUpdate:          time.Now(),
	}
}

// Helper methods

func (pc *PersonalContext) getRelevantHabits(query string) []string {
	pc.habits.mu.RLock()
	defer pc.habits.mu.RUnlock()

	var relevant []string
	queryLower := strings.ToLower(query)

	// Check workflow patterns
	for _, pattern := range pc.habits.WorkflowPatterns {
		for _, step := range pattern.Steps {
			if strings.Contains(queryLower, strings.ToLower(step.Action)) {
				relevant = append(relevant, pattern.Name)
				break
			}
		}
	}

	// Check tool usage patterns
	for toolName, pattern := range pc.habits.ToolUsagePatterns {
		for _, context := range pattern.UsageContext {
			if strings.Contains(queryLower, strings.ToLower(context)) {
				relevant = append(relevant, toolName)
				break
			}
		}
	}

	return relevant
}

func (pc *PersonalContext) getSkillContext(query string) []SkillLevel {
	pc.learningProfile.mu.RLock()
	defer pc.learningProfile.mu.RUnlock()

	var relevant []SkillLevel
	queryLower := strings.ToLower(query)

	for _, skill := range pc.learningProfile.SkillLevels {
		if strings.Contains(queryLower, strings.ToLower(skill.Skill)) {
			relevant = append(relevant, skill)
		}
	}

	// Sort by confidence
	sort.Slice(relevant, func(i, j int) bool {
		return relevant[i].Confidence > relevant[j].Confidence
	})

	return relevant
}

func (pc *PersonalContext) getLearningContext(query string) []LearningGoal {
	pc.learningProfile.mu.RLock()
	defer pc.learningProfile.mu.RUnlock()

	var relevant []LearningGoal
	queryLower := strings.ToLower(query)

	for _, goal := range pc.learningProfile.LearningGoals {
		if strings.Contains(queryLower, strings.ToLower(goal.Skill)) ||
			strings.Contains(queryLower, strings.ToLower(goal.Target)) {
			relevant = append(relevant, goal)
		}
	}

	return relevant
}

func (pc *PersonalContext) getRelevantAdaptations(query string) []string {
	pc.adaptations.mu.RLock()
	defer pc.adaptations.mu.RUnlock()

	var relevant []string

	for _, adaptation := range pc.adaptations.Adaptations {
		if adaptation.Confidence > 0.7 && adaptation.Success > 0.6 {
			relevant = append(relevant, adaptation.ID)
		}
	}

	return relevant
}

func (pc *PersonalContext) calculatePersonalityAlignment(request Request) float64 {
	// Simplified personality alignment calculation
	score := 0.5 // Base score

	// Adjust based on request characteristics
	if strings.Contains(strings.ToLower(request.Query), "explain") {
		score += 0.1 // User likes explanations
	}
	if strings.Contains(strings.ToLower(request.Query), "show") {
		score += 0.1 // User likes demonstrations
	}
	if strings.Contains(strings.ToLower(request.Query), "help") {
		score += 0.1 // User seeks assistance
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (pc *PersonalContext) handlePreferenceChange(update ContextUpdate) error {
	// Handle explicit preference changes
	pc.logger.Info("Processing preference change",
		slog.String("source", update.Source),
		slog.Any("data", update.Data))

	// Update preferences based on the change
	// This would be more sophisticated in a full implementation
	return nil
}

func (pc *PersonalContext) learnFromUpdate(update ContextUpdate) error {
	// Learn patterns from user behavior
	pc.logger.Debug("Learning from user behavior",
		slog.String("type", string(update.Type)),
		slog.String("source", update.Source))

	// Add learning logic here
	// This would analyze the update and adjust habits/preferences accordingly
	return nil
}

func (pc *PersonalContext) getActiveHabits() []string {
	pc.habits.mu.RLock()
	defer pc.habits.mu.RUnlock()

	var active []string
	cutoff := time.Now().Add(-7 * 24 * time.Hour) // Last week

	for _, pattern := range pc.habits.WorkflowPatterns {
		if pattern.LastUsed.After(cutoff) && pattern.Success > 0.7 {
			active = append(active, pattern.Name)
		}
	}

	return active
}

func (pc *PersonalContext) getCurrentGoals() []LearningGoal {
	pc.learningProfile.mu.RLock()
	defer pc.learningProfile.mu.RUnlock()

	var current []LearningGoal
	now := time.Now()

	for _, goal := range pc.learningProfile.LearningGoals {
		if goal.Deadline.After(now) && goal.Progress < 1.0 {
			current = append(current, goal)
		}
	}

	// Sort by priority and deadline
	sort.Slice(current, func(i, j int) bool {
		if current[i].Priority != current[j].Priority {
			return current[i].Priority == "high"
		}
		return current[i].Deadline.Before(current[j].Deadline)
	})

	return current
}

func (pc *PersonalContext) getActiveAdaptations() []string {
	pc.adaptations.mu.RLock()
	defer pc.adaptations.mu.RUnlock()

	var active []string

	for _, strategy := range pc.adaptations.ActiveStrategies {
		if strategy.Active && strategy.Success > 0.6 {
			active = append(active, strategy.Name)
		}
	}

	return active
}

// UpdatePreference updates a specific preference
func (pc *PersonalContext) UpdatePreference(category, key string, value interface{}) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.logger.Info("Updating preference",
		slog.String("category", category),
		slog.String("key", key),
		slog.Any("value", value))

	// Update preference based on category and key
	// This would be more sophisticated with reflection or a preference map

	return nil
}

// AddLearningGoal adds a new learning goal
func (pc *PersonalContext) AddLearningGoal(skill, target string, deadline time.Time, priority string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	goal := LearningGoal{
		ID:        fmt.Sprintf("goal_%d", time.Now().UnixNano()),
		Skill:     skill,
		Target:    target,
		Deadline:  deadline,
		Progress:  0.0,
		Priority:  priority,
		Resources: make([]string, 0),
	}

	pc.learningProfile.mu.Lock()
	pc.learningProfile.LearningGoals = append(pc.learningProfile.LearningGoals, goal)
	pc.learningProfile.mu.Unlock()

	pc.logger.Info("Learning goal added",
		slog.String("goal_id", goal.ID),
		slog.String("skill", skill),
		slog.String("target", target))

	return nil
}

// RecordLearningEvent records a learning activity
func (pc *PersonalContext) RecordLearningEvent(skill, activity, outcome string, metadata map[string]interface{}) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	event := LearningEvent{
		ID:        fmt.Sprintf("event_%d", time.Now().UnixNano()),
		Skill:     skill,
		Activity:  activity,
		Outcome:   outcome,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	pc.learningProfile.mu.Lock()
	pc.learningProfile.LearningHistory = append(pc.learningProfile.LearningHistory, event)
	pc.learningProfile.mu.Unlock()

	pc.logger.Debug("Learning event recorded",
		slog.String("event_id", event.ID),
		slog.String("skill", skill),
		slog.String("activity", activity))

	return nil
}

// Close shuts down the personal context
func (pc *PersonalContext) Close(ctx context.Context) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	pc.logger.Info("Shutting down personal context")

	// Clear data
	pc.preferences = nil
	pc.habits = nil
	pc.learningProfile = nil
	pc.adaptations = nil

	return nil
}
