package context

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

// TemporalContext maintains temporal awareness and development history
type TemporalContext struct {
	timeline       *DevelopmentTimeline
	patterns       *TemporalPatterns
	sessions       map[string]*DevelopmentSession
	currentSession *DevelopmentSession
	logger         *slog.Logger
	mu             sync.RWMutex
}

// DevelopmentTimeline represents the temporal sequence of development activities
type DevelopmentTimeline struct {
	Events    []TemporalEvent
	Sessions  []DevelopmentSession
	Patterns  []RecurringPattern
	MaxEvents int
	mu        sync.RWMutex
}

// TemporalEvent represents an event in the development timeline
type TemporalEvent struct {
	ID        string
	Type      EventType
	Timestamp time.Time
	Duration  time.Duration
	Context   string
	Action    string
	Result    string
	Success   bool
	Metadata  map[string]interface{}
	SessionID string
}

// EventType defines types of temporal events
type EventType string

const (
	EventCommand      EventType = "command"
	EventFileEdit     EventType = "file_edit"
	EventDebug        EventType = "debug"
	EventTest         EventType = "test"
	EventBuild        EventType = "build"
	EventDeploy       EventType = "deploy"
	EventResearch     EventType = "research"
	EventRefactor     EventType = "refactor"
	EventSessionStart EventType = "session_start"
	EventSessionEnd   EventType = "session_end"
)

// DevelopmentSession represents a development work session
type DevelopmentSession struct {
	ID            string
	StartTime     time.Time
	EndTime       *time.Time
	Duration      time.Duration
	Project       string
	Focus         string
	Activities    []TemporalEvent
	Productivity  float64
	Interruptions int
	Context       SessionContext
}

// SessionContext represents the context of a development session
type SessionContext struct {
	Goals       []string
	MainTask    string
	Blockers    []string
	Discoveries []string
	Learnings   []string
}

// TemporalPatterns manages pattern recognition across time
type TemporalPatterns struct {
	DailyPatterns    []DailyPattern
	WeeklyPatterns   []WeeklyPattern
	WorkflowPatterns []WorkflowPattern
	SeasonalPatterns []SeasonalPattern
	mu               sync.RWMutex
}

// DailyPattern represents patterns within a day
type DailyPattern struct {
	Hour         int
	Activity     string
	Frequency    int
	Productivity float64
	Confidence   float64
}

// WeeklyPattern represents patterns within a week
type WeeklyPattern struct {
	DayOfWeek  time.Weekday
	Activity   string
	Frequency  int
	Duration   time.Duration
	Confidence float64
}

// WorkflowPattern represents recurring workflow patterns
type WorkflowPattern struct {
	ID          string
	Name        string
	Steps       []WorkflowStep
	Frequency   int
	AvgDuration time.Duration
	SuccessRate float64
	Confidence  float64
}

// WorkflowStep represents a step in a workflow pattern
type WorkflowStep struct {
	Order       int
	Action      string
	Tool        string
	Duration    time.Duration
	SuccessRate float64
}

// SeasonalPattern represents long-term seasonal patterns
type SeasonalPattern struct {
	Month      time.Month
	Activity   string
	Intensity  float64
	Confidence float64
}

// RecurringPattern represents a pattern that recurs over time
type RecurringPattern struct {
	ID          string
	Type        PatternType
	Events      []TemporalEvent
	Frequency   time.Duration
	Confidence  float64
	LastSeen    time.Time
	Predictions []PatternPrediction
}

// PatternType defines types of recurring patterns
type PatternType string

const (
	PatternDaily    PatternType = "daily"
	PatternWeekly   PatternType = "weekly"
	PatternMonthly  PatternType = "monthly"
	PatternWorkflow PatternType = "workflow"
	PatternTrend    PatternType = "trend"
)

// PatternPrediction represents a prediction based on a pattern
type PatternPrediction struct {
	ExpectedTime    time.Time
	Confidence      float64
	SuggestedAction string
	Context         string
}

// TemporalInfo contains relevant temporal context for a request
type TemporalInfo struct {
	RecentActions  []TemporalEvent
	CurrentSession *DevelopmentSession
	Patterns       []RecurringPattern
	SimilarPast    []TemporalEvent
	Predictions    []PatternPrediction
	TimeContext    TimeContext
	WorkflowState  WorkflowState
}

// TimeContext provides time-based context
type TimeContext struct {
	CurrentTime    time.Time
	TimeOfDay      string
	DayOfWeek      time.Weekday
	WorkingHours   bool
	SessionLength  time.Duration
	RecentActivity string
}

// WorkflowState represents the current state in known workflows
type WorkflowState struct {
	ActiveWorkflows []string
	CurrentStep     map[string]int
	Predictions     []WorkflowPrediction
}

// WorkflowPrediction predicts next steps in a workflow
type WorkflowPrediction struct {
	WorkflowID    string
	NextStep      string
	Confidence    float64
	EstimatedTime time.Duration
}

// TemporalState represents current temporal state
type TemporalState struct {
	CurrentSession *DevelopmentSession
	RecentEvents   []TemporalEvent
	ActivePatterns []RecurringPattern
	LastUpdate     time.Time
}

// NewTemporalContext creates a new temporal context
func NewTemporalContext(logger *slog.Logger) (*TemporalContext, error) {
	timeline := &DevelopmentTimeline{
		Events:    make([]TemporalEvent, 0),
		Sessions:  make([]DevelopmentSession, 0),
		Patterns:  make([]RecurringPattern, 0),
		MaxEvents: 1000,
	}

	patterns := &TemporalPatterns{
		DailyPatterns:    make([]DailyPattern, 0),
		WeeklyPatterns:   make([]WeeklyPattern, 0),
		WorkflowPatterns: make([]WorkflowPattern, 0),
		SeasonalPatterns: make([]SeasonalPattern, 0),
	}

	return &TemporalContext{
		timeline:       timeline,
		patterns:       patterns,
		sessions:       make(map[string]*DevelopmentSession),
		currentSession: nil,
		logger:         logger,
	}, nil
}

// GetRelatedHistory gets temporal context related to a request
func (tc *TemporalContext) GetRelatedHistory(ctx context.Context, request Request) (TemporalInfo, error) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	now := time.Now()

	// Get recent actions (last hour)
	recentActions := tc.getRecentActions(1 * time.Hour)

	// Find similar past events
	similarPast := tc.findSimilarEvents(request.Query, 5)

	// Get relevant patterns
	patterns := tc.getRelevantPatterns(request.Query)

	// Generate predictions
	predictions := tc.generatePredictions(patterns, now)

	// Build time context
	timeContext := TimeContext{
		CurrentTime:    now,
		TimeOfDay:      tc.getTimeOfDay(now),
		DayOfWeek:      now.Weekday(),
		WorkingHours:   tc.isWorkingHours(now),
		RecentActivity: tc.getRecentActivitySummary(),
	}

	if tc.currentSession != nil {
		timeContext.SessionLength = now.Sub(tc.currentSession.StartTime)
	}

	// Build workflow state
	workflowState := tc.analyzeWorkflowState(request)

	return TemporalInfo{
		RecentActions:  recentActions,
		CurrentSession: tc.currentSession,
		Patterns:       patterns,
		SimilarPast:    similarPast,
		Predictions:    predictions,
		TimeContext:    timeContext,
		WorkflowState:  workflowState,
	}, nil
}

// ProcessUpdate processes a temporal context update
func (tc *TemporalContext) ProcessUpdate(ctx context.Context, update ContextUpdate) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Create temporal event from update
	event := TemporalEvent{
		ID:        fmt.Sprintf("event_%d", time.Now().UnixNano()),
		Type:      tc.mapUpdateToEventType(update),
		Timestamp: update.Timestamp,
		Context:   update.Source,
		Metadata:  update.Data,
	}

	if tc.currentSession != nil {
		event.SessionID = tc.currentSession.ID
		tc.currentSession.Activities = append(tc.currentSession.Activities, event)
	}

	// Add to timeline
	tc.timeline.mu.Lock()
	tc.timeline.Events = append(tc.timeline.Events, event)

	// Maintain timeline size
	if len(tc.timeline.Events) > tc.timeline.MaxEvents {
		tc.timeline.Events = tc.timeline.Events[1:]
	}
	tc.timeline.mu.Unlock()

	// Update patterns
	go tc.updatePatterns(event)

	tc.logger.Debug("Temporal event recorded",
		slog.String("event_id", event.ID),
		slog.String("type", string(event.Type)),
		slog.String("context", event.Context))

	return nil
}

// StartSession starts a new development session
func (tc *TemporalContext) StartSession(project, focus string) *DevelopmentSession {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	session := &DevelopmentSession{
		ID:         fmt.Sprintf("session_%d", time.Now().UnixNano()),
		StartTime:  time.Now(),
		Project:    project,
		Focus:      focus,
		Activities: make([]TemporalEvent, 0),
		Context: SessionContext{
			Goals:       make([]string, 0),
			Blockers:    make([]string, 0),
			Discoveries: make([]string, 0),
			Learnings:   make([]string, 0),
		},
	}

	tc.currentSession = session
	tc.sessions[session.ID] = session

	// Record session start event
	startEvent := TemporalEvent{
		ID:        fmt.Sprintf("session_start_%d", time.Now().UnixNano()),
		Type:      EventSessionStart,
		Timestamp: session.StartTime,
		Context:   "session",
		Action:    "start",
		SessionID: session.ID,
		Metadata: map[string]interface{}{
			"project": project,
			"focus":   focus,
		},
	}

	tc.timeline.mu.Lock()
	tc.timeline.Events = append(tc.timeline.Events, startEvent)
	tc.timeline.mu.Unlock()

	tc.logger.Info("Development session started",
		slog.String("session_id", session.ID),
		slog.String("project", project),
		slog.String("focus", focus))

	return session
}

// EndSession ends the current development session
func (tc *TemporalContext) EndSession() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.currentSession == nil {
		return
	}

	endTime := time.Now()
	tc.currentSession.EndTime = &endTime
	tc.currentSession.Duration = endTime.Sub(tc.currentSession.StartTime)

	// Calculate productivity score
	tc.currentSession.Productivity = tc.calculateProductivity(tc.currentSession)

	// Record session end event
	endEvent := TemporalEvent{
		ID:        fmt.Sprintf("session_end_%d", time.Now().UnixNano()),
		Type:      EventSessionEnd,
		Timestamp: endTime,
		Context:   "session",
		Action:    "end",
		SessionID: tc.currentSession.ID,
		Metadata: map[string]interface{}{
			"duration":     tc.currentSession.Duration.String(),
			"productivity": tc.currentSession.Productivity,
			"activities":   len(tc.currentSession.Activities),
		},
	}

	tc.timeline.mu.Lock()
	tc.timeline.Events = append(tc.timeline.Events, endEvent)
	tc.timeline.Sessions = append(tc.timeline.Sessions, *tc.currentSession)
	tc.timeline.mu.Unlock()

	tc.logger.Info("Development session ended",
		slog.String("session_id", tc.currentSession.ID),
		slog.Duration("duration", tc.currentSession.Duration),
		slog.Float64("productivity", tc.currentSession.Productivity))

	tc.currentSession = nil
}

// GetCurrentState returns current temporal state
func (tc *TemporalContext) GetCurrentState() TemporalState {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	recentEvents := tc.getRecentActions(1 * time.Hour)
	activePatterns := tc.getActivePatterns()

	return TemporalState{
		CurrentSession: tc.currentSession,
		RecentEvents:   recentEvents,
		ActivePatterns: activePatterns,
		LastUpdate:     time.Now(),
	}
}

// Helper methods

func (tc *TemporalContext) getRecentActions(duration time.Duration) []TemporalEvent {
	cutoff := time.Now().Add(-duration)
	var recent []TemporalEvent

	tc.timeline.mu.RLock()
	defer tc.timeline.mu.RUnlock()

	for i := len(tc.timeline.Events) - 1; i >= 0; i-- {
		event := tc.timeline.Events[i]
		if event.Timestamp.After(cutoff) {
			recent = append([]TemporalEvent{event}, recent...)
		} else {
			break
		}
	}

	return recent
}

func (tc *TemporalContext) findSimilarEvents(query string, limit int) []TemporalEvent {
	// Simplified similarity matching - would use better NLP in full implementation
	var similar []TemporalEvent

	tc.timeline.mu.RLock()
	defer tc.timeline.mu.RUnlock()

	for _, event := range tc.timeline.Events {
		if len(similar) >= limit {
			break
		}
		// Simple string containment for now
		if event.Action == query || event.Context == query {
			similar = append(similar, event)
		}
	}

	return similar
}

func (tc *TemporalContext) getRelevantPatterns(query string) []RecurringPattern {
	// Return patterns relevant to the query
	var relevant []RecurringPattern

	tc.timeline.mu.RLock()
	defer tc.timeline.mu.RUnlock()

	for _, pattern := range tc.timeline.Patterns {
		if pattern.Confidence > 0.7 {
			relevant = append(relevant, pattern)
		}
	}

	return relevant
}

func (tc *TemporalContext) generatePredictions(patterns []RecurringPattern, now time.Time) []PatternPrediction {
	var predictions []PatternPrediction

	for _, pattern := range patterns {
		for _, pred := range pattern.Predictions {
			if pred.ExpectedTime.After(now) && pred.Confidence > 0.5 {
				predictions = append(predictions, pred)
			}
		}
	}

	// Sort by expected time
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].ExpectedTime.Before(predictions[j].ExpectedTime)
	})

	return predictions
}

func (tc *TemporalContext) getTimeOfDay(t time.Time) string {
	hour := t.Hour()
	switch {
	case hour < 6:
		return "night"
	case hour < 12:
		return "morning"
	case hour < 18:
		return "afternoon"
	default:
		return "evening"
	}
}

func (tc *TemporalContext) isWorkingHours(t time.Time) bool {
	hour := t.Hour()
	weekday := t.Weekday()
	return weekday >= time.Monday && weekday <= time.Friday && hour >= 9 && hour < 17
}

func (tc *TemporalContext) getRecentActivitySummary() string {
	recent := tc.getRecentActions(30 * time.Minute)
	if len(recent) == 0 {
		return "no recent activity"
	}

	// Summarize recent activity types
	activityCount := make(map[EventType]int)
	for _, event := range recent {
		activityCount[event.Type]++
	}

	// Find most common activity
	var mostCommon EventType
	maxCount := 0
	for eventType, count := range activityCount {
		if count > maxCount {
			maxCount = count
			mostCommon = eventType
		}
	}

	return string(mostCommon)
}

func (tc *TemporalContext) analyzeWorkflowState(request Request) WorkflowState {
	// Simplified workflow analysis
	return WorkflowState{
		ActiveWorkflows: []string{},
		CurrentStep:     make(map[string]int),
		Predictions:     []WorkflowPrediction{},
	}
}

func (tc *TemporalContext) mapUpdateToEventType(update ContextUpdate) EventType {
	switch update.Type {
	case CommandExecution:
		return EventCommand
	case FileActivity:
		return EventFileEdit
	default:
		return EventCommand
	}
}

func (tc *TemporalContext) updatePatterns(event TemporalEvent) {
	// Pattern update logic would be implemented here
	// This would analyze the event in context of existing patterns
	// and update pattern confidence, frequency, etc.
}

func (tc *TemporalContext) calculateProductivity(session *DevelopmentSession) float64 {
	// Simplified productivity calculation
	if len(session.Activities) == 0 {
		return 0.0
	}

	// Base productivity on activity variety and success rate
	activityTypes := make(map[EventType]bool)
	successCount := 0

	for _, activity := range session.Activities {
		activityTypes[activity.Type] = true
		if activity.Success {
			successCount++
		}
	}

	varietyScore := float64(len(activityTypes)) / 5.0 // Normalize to 0-1
	if varietyScore > 1.0 {
		varietyScore = 1.0
	}

	successRate := float64(successCount) / float64(len(session.Activities))

	return (varietyScore + successRate) / 2.0
}

func (tc *TemporalContext) getActivePatterns() []RecurringPattern {
	var active []RecurringPattern

	tc.timeline.mu.RLock()
	defer tc.timeline.mu.RUnlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for _, pattern := range tc.timeline.Patterns {
		if pattern.LastSeen.After(cutoff) && pattern.Confidence > 0.6 {
			active = append(active, pattern)
		}
	}

	return active
}

// Close shuts down the temporal context
func (tc *TemporalContext) Close(ctx context.Context) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// End current session if active
	if tc.currentSession != nil {
		tc.EndSession()
	}

	tc.logger.Info("Shutting down temporal context")

	// Clear data
	tc.timeline = nil
	tc.patterns = nil
	tc.sessions = nil

	return nil
}
