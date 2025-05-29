package context

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ContextEngine maintains a living understanding of the development environment
type ContextEngine struct {
	workspace   *WorkspaceContext
	temporal    *TemporalContext
	semantic    *SemanticContext
	personal    *PersonalContext
	logger      *slog.Logger
	mu          sync.RWMutex
	subscribers []ContextSubscriber
}

// ContextSubscriber interface for components that need context updates
type ContextSubscriber interface {
	OnContextUpdate(ctx context.Context, update ContextUpdate) error
}

// ContextUpdate represents a change in context
type ContextUpdate struct {
	Type      ContextUpdateType
	Timestamp time.Time
	Source    string
	Data      map[string]interface{}
}

// ContextUpdateType defines types of context updates
type ContextUpdateType string

const (
	WorkspaceChange  ContextUpdateType = "workspace_change"
	FileActivity     ContextUpdateType = "file_activity"
	CommandExecution ContextUpdateType = "command_execution"
	ProjectSwitch    ContextUpdateType = "project_switch"
	PreferenceChange ContextUpdateType = "preference_change"
)

// Request represents an incoming request with context
type Request struct {
	ID        string
	Query     string
	Type      string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// ContextualRequest is a request enriched with context
type ContextualRequest struct {
	Original   Request
	Workspace  WorkspaceInfo
	History    TemporalInfo
	Semantics  SemanticInfo
	Personal   PersonalInfo
	Confidence float64
}

// NewContextEngine creates a new context engine
func NewContextEngine(logger *slog.Logger) (*ContextEngine, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	workspace, err := NewWorkspaceContext(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace context: %w", err)
	}

	temporal, err := NewTemporalContext(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporal context: %w", err)
	}

	semantic, err := NewSemanticContext(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create semantic context: %w", err)
	}

	personal, err := NewPersonalContext(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create personal context: %w", err)
	}

	return &ContextEngine{
		workspace:   workspace,
		temporal:    temporal,
		semantic:    semantic,
		personal:    personal,
		logger:      logger,
		subscribers: make([]ContextSubscriber, 0),
	}, nil
}

// EnrichRequest enriches a request with comprehensive context
func (ce *ContextEngine) EnrichRequest(ctx context.Context, request Request) (*ContextualRequest, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	ce.logger.Debug("Enriching request with context",
		slog.String("request_id", request.ID),
		slog.String("query", request.Query))

	// Get workspace context
	workspaceInfo, err := ce.workspace.GetRelevantContext(ctx, request)
	if err != nil {
		ce.logger.Warn("Failed to get workspace context", slog.Any("error", err))
		workspaceInfo = WorkspaceInfo{} // Use empty context
	}

	// Get temporal context
	historyInfo, err := ce.temporal.GetRelatedHistory(ctx, request)
	if err != nil {
		ce.logger.Warn("Failed to get temporal context", slog.Any("error", err))
		historyInfo = TemporalInfo{} // Use empty context
	}

	// Extract semantic meaning
	semanticInfo, err := ce.semantic.ExtractMeaning(ctx, request)
	if err != nil {
		ce.logger.Warn("Failed to extract semantic meaning", slog.Any("error", err))
		semanticInfo = SemanticInfo{} // Use empty context
	}

	// Get personal preferences
	personalInfo, err := ce.personal.GetPreferences(ctx, request)
	if err != nil {
		ce.logger.Warn("Failed to get personal preferences", slog.Any("error", err))
		personalInfo = PersonalInfo{} // Use empty context
	}

	// Calculate overall confidence based on context quality
	confidence := ce.calculateConfidence(workspaceInfo, historyInfo, semanticInfo, personalInfo)

	enriched := &ContextualRequest{
		Original:   request,
		Workspace:  workspaceInfo,
		History:    historyInfo,
		Semantics:  semanticInfo,
		Personal:   personalInfo,
		Confidence: confidence,
	}

	ce.logger.Info("Request enriched with context",
		slog.String("request_id", request.ID),
		slog.Float64("confidence", confidence),
		slog.String("active_project", workspaceInfo.ActiveProject),
		slog.Int("history_items", len(historyInfo.RecentActions)))

	return enriched, nil
}

// UpdateContext updates the context engine with new information
func (ce *ContextEngine) UpdateContext(ctx context.Context, update ContextUpdate) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	ce.logger.Debug("Updating context",
		slog.String("type", string(update.Type)),
		slog.String("source", update.Source))

	// Route update to appropriate context subsystem
	switch update.Type {
	case WorkspaceChange, FileActivity, ProjectSwitch:
		if err := ce.workspace.ProcessUpdate(ctx, update); err != nil {
			return fmt.Errorf("failed to update workspace context: %w", err)
		}
	case CommandExecution:
		if err := ce.temporal.ProcessUpdate(ctx, update); err != nil {
			return fmt.Errorf("failed to update temporal context: %w", err)
		}
	case PreferenceChange:
		if err := ce.personal.ProcessUpdate(ctx, update); err != nil {
			return fmt.Errorf("failed to update personal context: %w", err)
		}
	}

	// Notify subscribers
	go ce.notifySubscribers(ctx, update)

	return nil
}

// Subscribe allows components to receive context updates
func (ce *ContextEngine) Subscribe(subscriber ContextSubscriber) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	ce.subscribers = append(ce.subscribers, subscriber)
	ce.logger.Debug("New context subscriber added", slog.Int("total_subscribers", len(ce.subscribers)))
}

// GetCurrentContext returns the current context state
func (ce *ContextEngine) GetCurrentContext(ctx context.Context) (*CurrentContext, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	workspace := ce.workspace.GetCurrentState()
	temporal := ce.temporal.GetCurrentState()
	semantic := ce.semantic.GetCurrentState()
	personal := ce.personal.GetCurrentState()

	return &CurrentContext{
		Workspace: workspace,
		Temporal:  temporal,
		Semantic:  semantic,
		Personal:  personal,
		Timestamp: time.Now(),
	}, nil
}

// calculateConfidence calculates overall context confidence
func (ce *ContextEngine) calculateConfidence(workspace WorkspaceInfo, history TemporalInfo, semantic SemanticInfo, personal PersonalInfo) float64 {
	// Base confidence factors
	workspaceConfidence := 0.0
	if workspace.ActiveProject != "" {
		workspaceConfidence = 0.3
	}
	if len(workspace.OpenFiles) > 0 {
		workspaceConfidence += 0.2
	}

	historyConfidence := 0.0
	if len(history.RecentActions) > 0 {
		historyConfidence = 0.2
	}
	if len(history.Patterns) > 0 {
		historyConfidence += 0.1
	}

	semanticConfidence := semantic.Confidence * 0.3

	personalConfidence := 0.0
	if personal.PersonalityScore > 0 {
		personalConfidence = 0.1
	}

	return workspaceConfidence + historyConfidence + semanticConfidence + personalConfidence
}

// notifySubscribers notifies all subscribers of context updates
func (ce *ContextEngine) notifySubscribers(ctx context.Context, update ContextUpdate) {
	for _, subscriber := range ce.subscribers {
		if err := subscriber.OnContextUpdate(ctx, update); err != nil {
			ce.logger.Warn("Failed to notify context subscriber", slog.Any("error", err))
		}
	}
}

// Close gracefully shuts down the context engine
func (ce *ContextEngine) Close(ctx context.Context) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	ce.logger.Info("Shutting down context engine")

	// Close all context subsystems
	if err := ce.workspace.Close(ctx); err != nil {
		ce.logger.Error("Failed to close workspace context", slog.Any("error", err))
	}

	if err := ce.temporal.Close(ctx); err != nil {
		ce.logger.Error("Failed to close temporal context", slog.Any("error", err))
	}

	if err := ce.semantic.Close(ctx); err != nil {
		ce.logger.Error("Failed to close semantic context", slog.Any("error", err))
	}

	if err := ce.personal.Close(ctx); err != nil {
		ce.logger.Error("Failed to close personal context", slog.Any("error", err))
	}

	ce.subscribers = nil
	return nil
}

// CurrentContext represents the current state of all context subsystems
type CurrentContext struct {
	Workspace WorkspaceState
	Temporal  TemporalState
	Semantic  SemanticState
	Personal  PersonalState
	Timestamp time.Time
}
