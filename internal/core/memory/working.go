package memory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/koopa0/assistant-go/internal/core/events"
)

// WorkingMemory provides short-term, task-focused memory storage with limited capacity.
// It implements cognitive psychology principles including:
// - Limited capacity (Miller's 7±2 rule)
// - Activation decay over time
// - Attention management and focus
// - Consolidation to long-term memory
//
// Working memory is essential for maintaining task context, active goals,
// and immediate information needed for current operations.
type WorkingMemory struct {
	capacity       int
	ttl            time.Duration
	items          map[string]*WorkingMemoryItem
	taskContexts   map[string]*TaskContext
	activeBuffers  map[string]*MemoryBuffer
	attentionFocus *AttentionManager
	consolidator   *MemoryConsolidator
	eventBus       *events.EventBus
	logger         *slog.Logger
	mu             sync.RWMutex
}

// WorkingMemoryItem represents a single piece of information in working memory.
// Items have activation levels that decay over time and increase with access.
// They can be associated with other items and contain rich metadata.
type WorkingMemoryItem struct {
	ID           string
	Type         ItemType
	Content      interface{}
	Context      string
	Priority     float64
	Activation   float64
	CreatedAt    time.Time
	LastAccessed time.Time
	AccessCount  int
	Associations []string
	Metadata     map[string]interface{}
}

// TaskContext maintains comprehensive context for active tasks including:
// - Goal tracking and progress monitoring
// - Constraint management
// - Relevant memory items
// - Sub-task decomposition
// - Result accumulation
//
// Task contexts enable the system to manage multiple concurrent activities
// while maintaining focus and tracking progress.
type TaskContext struct {
	TaskID         string
	Goal           string
	Constraints    []string
	RelevantItems  []string
	SubTasks       []string
	Progress       float64
	State          TaskState
	StartTime      time.Time
	LastUpdate     time.Time
	CompletionTime *time.Time
	Results        map[string]interface{}
}

// TaskState defines the state of a task
type TaskState string

const (
	TaskStateActive    TaskState = "active"
	TaskStateSuspended TaskState = "suspended"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
)

// MemoryBuffer provides temporary storage for related items
type MemoryBuffer struct {
	ID         string
	Name       string
	Type       BufferType
	Capacity   int
	Items      []string
	Priority   float64
	CreatedAt  time.Time
	LastUpdate time.Time
	TTL        time.Duration
}

// BufferType defines types of memory buffers
type BufferType string

const (
	BufferTypePhonological BufferType = "phonological" // Language/text
	BufferTypeVisuoSpatial BufferType = "visuospatial" // Visual/spatial
	BufferTypeEpisodic     BufferType = "episodic"     // Episode buffer
	BufferTypeExecutive    BufferType = "executive"    // Central executive
)

// AttentionManager implements attention mechanisms for working memory.
// It simulates human attention constraints including:
// - Limited attention span (typically 7 items)
// - Switch costs when changing focus
// - Distraction handling
// - Strength-based focus allocation
type AttentionManager struct {
	focusItems    []string
	focusStrength map[string]float64
	distractions  []Distraction
	attentionSpan int
	switchCost    float64
	mu            sync.RWMutex
}

// Distraction represents something that diverts attention
type Distraction struct {
	Source    string
	Strength  float64
	Timestamp time.Time
	Handled   bool
}

// MemoryConsolidator transfers items to long-term memory
type MemoryConsolidator struct {
	threshold      float64
	batchSize      int
	checkInterval  time.Duration
	consolidationQ chan ConsolidationRequest
	mu             sync.Mutex
}

// ConsolidationRequest represents a request to consolidate memory
type ConsolidationRequest struct {
	Items      []string
	TargetType MemoryType
	Priority   float64
	Context    string
}

// MemoryType is defined in types.go

// WorkingMemoryQuery defines query parameters for working memory
type WorkingMemoryQuery struct {
	Type          ItemType
	Context       string
	MinPriority   float64
	MinActivation float64
	Limit         int
	SortBy        SortCriteria
}

// SortCriteria is defined in types.go

// NewWorkingMemory creates and initializes a new working memory instance.
// It starts background maintenance and consolidation processes.
//
// Parameters:
//   - capacity: Maximum number of items (recommend 50-100)
//   - ttl: Time-to-live for items before eligible for eviction
//   - eventBus: Event bus for publishing memory events
//   - logger: Structured logger for debugging
//
// Returns:
//   - *WorkingMemory: Initialized working memory instance
//   - error: Configuration error if capacity <= 0
func NewWorkingMemory(capacity int, ttl time.Duration, eventBus *events.EventBus, logger *slog.Logger) (*WorkingMemory, error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("capacity must be positive")
	}

	attention := &AttentionManager{
		focusItems:    make([]string, 0),
		focusStrength: make(map[string]float64),
		distractions:  make([]Distraction, 0),
		attentionSpan: 7, // Miller's magical number
		switchCost:    0.1,
	}

	consolidator := &MemoryConsolidator{
		threshold:      0.8,
		batchSize:      10,
		checkInterval:  5 * time.Minute,
		consolidationQ: make(chan ConsolidationRequest, 100),
	}

	wm := &WorkingMemory{
		capacity:       capacity,
		ttl:            ttl,
		items:          make(map[string]*WorkingMemoryItem),
		taskContexts:   make(map[string]*TaskContext),
		activeBuffers:  make(map[string]*MemoryBuffer),
		attentionFocus: attention,
		consolidator:   consolidator,
		eventBus:       eventBus,
		logger:         logger,
	}

	// Start background processes
	go wm.runMaintenanceLoop()
	go wm.runConsolidationLoop()

	return wm, nil
}

// Store adds an item to working memory with automatic eviction if at capacity.
// Items are initialized with timestamps and activation levels.
// An event is published for each successful store operation.
//
// Parameters:
//   - ctx: Context for cancellation
//   - item: The item to store (ID must be set)
//
// Returns:
//   - error: Storage error or eviction failure
func (wm *WorkingMemory) Store(ctx context.Context, item *WorkingMemoryItem) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Check capacity
	if len(wm.items) >= wm.capacity {
		// Evict least recently used item with lowest activation
		if err := wm.evictLRU(); err != nil {
			return fmt.Errorf("failed to evict item: %w", err)
		}
	}

	// Set timestamps
	now := time.Now()
	item.CreatedAt = now
	item.LastAccessed = now
	item.AccessCount = 0

	// Initialize activation if not set
	if item.Activation == 0 {
		item.Activation = item.Priority
	}

	// Store item
	wm.items[item.ID] = item

	wm.logger.Debug("Stored working memory item",
		slog.String("item_id", item.ID),
		slog.String("type", string(item.Type)),
		slog.Float64("priority", item.Priority))

	// Publish event
	if wm.eventBus != nil {
		event := events.Event{
			Type:   events.EventCustom,
			Source: "working_memory",
			Data: map[string]interface{}{
				"action":  "store",
				"item_id": item.ID,
				"type":    item.Type,
			},
		}
		wm.eventBus.Publish(ctx, event)
	}

	return nil
}

// Retrieve gets an item from working memory
func (wm *WorkingMemory) Retrieve(ctx context.Context, itemID string) (*WorkingMemoryItem, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	item, exists := wm.items[itemID]
	if !exists {
		return nil, fmt.Errorf("item not found: %s", itemID)
	}

	// Update access information
	item.LastAccessed = time.Now()
	item.AccessCount++

	// Increase activation with each access
	item.Activation = min(1.0, item.Activation*1.1)

	wm.logger.Debug("Retrieved working memory item",
		slog.String("item_id", itemID),
		slog.Int("access_count", item.AccessCount))

	return item, nil
}

// Query searches working memory based on criteria
func (wm *WorkingMemory) Query(ctx context.Context, query WorkingMemoryQuery) ([]*WorkingMemoryItem, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var results []*WorkingMemoryItem

	// Filter items
	for _, item := range wm.items {
		if wm.matchesQuery(item, query) {
			results = append(results, item)
		}
	}

	// Sort results
	results = wm.sortResults(results, query.SortBy)

	// Apply limit
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	wm.logger.Debug("Queried working memory",
		slog.Int("results", len(results)),
		slog.String("type", string(query.Type)))

	return results, nil
}

// Update modifies an existing item
func (wm *WorkingMemory) Update(ctx context.Context, itemID string, updates map[string]interface{}) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	item, exists := wm.items[itemID]
	if !exists {
		return fmt.Errorf("item not found: %s", itemID)
	}

	// Apply updates
	if priority, ok := updates["priority"].(float64); ok {
		item.Priority = priority
	}
	if activation, ok := updates["activation"].(float64); ok {
		item.Activation = activation
	}
	if content, ok := updates["content"]; ok {
		item.Content = content
	}
	if metadata, ok := updates["metadata"].(map[string]interface{}); ok {
		for k, v := range metadata {
			item.Metadata[k] = v
		}
	}

	item.LastAccessed = time.Now()

	wm.logger.Debug("Updated working memory item",
		slog.String("item_id", itemID))

	return nil
}

// Delete removes an item from working memory
func (wm *WorkingMemory) Delete(ctx context.Context, itemID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.items[itemID]; !exists {
		return fmt.Errorf("item not found: %s", itemID)
	}

	delete(wm.items, itemID)

	wm.logger.Debug("Deleted working memory item",
		slog.String("item_id", itemID))

	return nil
}

// CreateTaskContext creates a new task context
func (wm *WorkingMemory) CreateTaskContext(ctx context.Context, taskID, goal string, constraints []string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	taskCtx := &TaskContext{
		TaskID:        taskID,
		Goal:          goal,
		Constraints:   constraints,
		RelevantItems: make([]string, 0),
		SubTasks:      make([]string, 0),
		Progress:      0.0,
		State:         TaskStateActive,
		StartTime:     time.Now(),
		LastUpdate:    time.Now(),
		Results:       make(map[string]interface{}),
	}

	wm.taskContexts[taskID] = taskCtx

	wm.logger.Info("Created task context",
		slog.String("task_id", taskID),
		slog.String("goal", goal))

	return nil
}

// UpdateTaskProgress updates the progress of a task
func (wm *WorkingMemory) UpdateTaskProgress(ctx context.Context, taskID string, progress float64, state TaskState) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	taskCtx, exists := wm.taskContexts[taskID]
	if !exists {
		return fmt.Errorf("task context not found: %s", taskID)
	}

	taskCtx.Progress = progress
	taskCtx.State = state
	taskCtx.LastUpdate = time.Now()

	if state == TaskStateCompleted || state == TaskStateFailed {
		now := time.Now()
		taskCtx.CompletionTime = &now
	}

	wm.logger.Debug("Updated task progress",
		slog.String("task_id", taskID),
		slog.Float64("progress", progress),
		slog.String("state", string(state)))

	return nil
}

// Focus sets attention focus on specific items
func (wm *WorkingMemory) Focus(ctx context.Context, itemIDs []string, strength float64) error {
	wm.attentionFocus.mu.Lock()
	defer wm.attentionFocus.mu.Unlock()

	// Clear current focus if switching
	if len(wm.attentionFocus.focusItems) > 0 {
		// Apply switch cost
		for id := range wm.attentionFocus.focusStrength {
			wm.attentionFocus.focusStrength[id] *= (1 - wm.attentionFocus.switchCost)
		}
	}

	// Set new focus
	wm.attentionFocus.focusItems = itemIDs
	for _, id := range itemIDs {
		wm.attentionFocus.focusStrength[id] = strength
	}

	wm.logger.Debug("Set attention focus",
		slog.Int("items", len(itemIDs)),
		slog.Float64("strength", strength))

	return nil
}

// GetFocusedItems returns currently focused items
func (wm *WorkingMemory) GetFocusedItems(ctx context.Context) ([]*WorkingMemoryItem, error) {
	wm.attentionFocus.mu.RLock()
	focusIDs := wm.attentionFocus.focusItems
	wm.attentionFocus.mu.RUnlock()

	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var focused []*WorkingMemoryItem
	for _, id := range focusIDs {
		if item, exists := wm.items[id]; exists {
			focused = append(focused, item)
		}
	}

	return focused, nil
}

// CreateBuffer creates a temporary memory buffer
func (wm *WorkingMemory) CreateBuffer(ctx context.Context, name string, bufferType BufferType, capacity int) (*MemoryBuffer, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	bufferID := fmt.Sprintf("buffer_%s_%d", name, time.Now().UnixNano())
	buffer := &MemoryBuffer{
		ID:         bufferID,
		Name:       name,
		Type:       bufferType,
		Capacity:   capacity,
		Items:      make([]string, 0, capacity),
		Priority:   0.5,
		CreatedAt:  time.Now(),
		LastUpdate: time.Now(),
		TTL:        wm.ttl,
	}

	wm.activeBuffers[bufferID] = buffer

	wm.logger.Debug("Created memory buffer",
		slog.String("buffer_id", bufferID),
		slog.String("name", name),
		slog.String("type", string(bufferType)))

	return buffer, nil
}

// AddToBuffer adds an item to a buffer
func (wm *WorkingMemory) AddToBuffer(ctx context.Context, bufferID, itemID string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	buffer, exists := wm.activeBuffers[bufferID]
	if !exists {
		return fmt.Errorf("buffer not found: %s", bufferID)
	}

	if len(buffer.Items) >= buffer.Capacity {
		// Remove oldest item
		buffer.Items = buffer.Items[1:]
	}

	buffer.Items = append(buffer.Items, itemID)
	buffer.LastUpdate = time.Now()

	return nil
}

// Consolidate triggers memory consolidation
func (wm *WorkingMemory) Consolidate(ctx context.Context, targetType MemoryType, threshold float64) error {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var itemsToConsolidate []string

	// Find items that meet consolidation criteria
	for id, item := range wm.items {
		if item.Activation >= threshold {
			itemsToConsolidate = append(itemsToConsolidate, id)
		}
	}

	if len(itemsToConsolidate) == 0 {
		return nil
	}

	// Send consolidation request
	request := ConsolidationRequest{
		Items:      itemsToConsolidate,
		TargetType: targetType,
		Priority:   0.8,
		Context:    "manual_consolidation",
	}

	select {
	case wm.consolidator.consolidationQ <- request:
		wm.logger.Info("Initiated memory consolidation",
			slog.Int("items", len(itemsToConsolidate)),
			slog.String("target_type", string(targetType)))
	default:
		return fmt.Errorf("consolidation queue full")
	}

	return nil
}

// GetMetrics returns working memory metrics
func (wm *WorkingMemory) GetMetrics() WorkingMemoryMetrics {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	metrics := WorkingMemoryMetrics{
		TotalItems:      len(wm.items),
		Capacity:        wm.capacity,
		UtilizationRate: float64(len(wm.items)) / float64(wm.capacity),
		ActiveTasks:     len(wm.taskContexts),
		ActiveBuffers:   len(wm.activeBuffers),
	}

	// Calculate average activation
	totalActivation := 0.0
	for _, item := range wm.items {
		totalActivation += item.Activation
	}
	if len(wm.items) > 0 {
		metrics.AverageActivation = totalActivation / float64(len(wm.items))
	}

	// Count by type
	metrics.ItemsByType = make(map[ItemType]int)
	for _, item := range wm.items {
		metrics.ItemsByType[item.Type]++
	}

	return metrics
}

// WorkingMemoryMetrics contains metrics about working memory
type WorkingMemoryMetrics struct {
	TotalItems        int
	Capacity          int
	UtilizationRate   float64
	AverageActivation float64
	ItemsByType       map[ItemType]int
	ActiveTasks       int
	ActiveBuffers     int
}

// Helper methods

func (wm *WorkingMemory) evictLRU() error {
	var lruItem *WorkingMemoryItem
	var lruID string

	// Find item with lowest activation and oldest access
	for id, item := range wm.items {
		if lruItem == nil ||
			item.Activation < lruItem.Activation ||
			(item.Activation == lruItem.Activation && item.LastAccessed.Before(lruItem.LastAccessed)) {
			lruItem = item
			lruID = id
		}
	}

	if lruID != "" {
		delete(wm.items, lruID)
		wm.logger.Debug("Evicted LRU item",
			slog.String("item_id", lruID),
			slog.Float64("activation", lruItem.Activation))
	}

	return nil
}

func (wm *WorkingMemory) matchesQuery(item *WorkingMemoryItem, query WorkingMemoryQuery) bool {
	if query.Type != "" && item.Type != query.Type {
		return false
	}
	if query.Context != "" && item.Context != query.Context {
		return false
	}
	if item.Priority < query.MinPriority {
		return false
	}
	if item.Activation < query.MinActivation {
		return false
	}
	return true
}

func (wm *WorkingMemory) sortResults(items []*WorkingMemoryItem, criteria SortCriteria) []*WorkingMemoryItem {
	// Sort based on criteria
	switch criteria {
	case SortByPriority:
		// Sort by priority descending
		for i := 0; i < len(items); i++ {
			for j := i + 1; j < len(items); j++ {
				if items[i].Priority < items[j].Priority {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
	case SortByActivation:
		// Sort by activation descending
		for i := 0; i < len(items); i++ {
			for j := i + 1; j < len(items); j++ {
				if items[i].Activation < items[j].Activation {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
	case SortByRecency:
		// Sort by last accessed descending
		for i := 0; i < len(items); i++ {
			for j := i + 1; j < len(items); j++ {
				if items[i].LastAccessed.Before(items[j].LastAccessed) {
					items[i], items[j] = items[j], items[i]
				}
			}
		}
	}
	return items
}

func (wm *WorkingMemory) runMaintenanceLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		wm.mu.Lock()

		// Decay activation levels
		for _, item := range wm.items {
			age := time.Since(item.LastAccessed)
			decayFactor := 1.0 - (age.Minutes()/60.0)*0.1
			if decayFactor < 0.1 {
				decayFactor = 0.1
			}
			item.Activation *= decayFactor
		}

		// Remove expired items
		for id, item := range wm.items {
			if time.Since(item.CreatedAt) > wm.ttl && item.Activation < 0.2 {
				delete(wm.items, id)
				wm.logger.Debug("Removed expired item",
					slog.String("item_id", id))
			}
		}

		// Clean up completed task contexts
		for id, taskCtx := range wm.taskContexts {
			if taskCtx.CompletionTime != nil && time.Since(*taskCtx.CompletionTime) > 1*time.Hour {
				delete(wm.taskContexts, id)
				wm.logger.Debug("Removed completed task context",
					slog.String("task_id", id))
			}
		}

		// Clean up expired buffers
		for id, buffer := range wm.activeBuffers {
			if time.Since(buffer.CreatedAt) > buffer.TTL {
				delete(wm.activeBuffers, id)
				wm.logger.Debug("Removed expired buffer",
					slog.String("buffer_id", id))
			}
		}

		wm.mu.Unlock()
	}
}

func (wm *WorkingMemory) runConsolidationLoop() {
	for request := range wm.consolidator.consolidationQ {
		// Process consolidation request
		wm.processConsolidation(request)
	}
}

func (wm *WorkingMemory) processConsolidation(request ConsolidationRequest) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	consolidatedItems := make([]*WorkingMemoryItem, 0, len(request.Items))
	for _, itemID := range request.Items {
		if item, exists := wm.items[itemID]; exists {
			consolidatedItems = append(consolidatedItems, item)
		}
	}

	if len(consolidatedItems) == 0 {
		return
	}

	// Publish consolidation event
	if wm.eventBus != nil {
		event := events.Event{
			Type:   events.EventCustom,
			Source: "working_memory",
			Data: map[string]interface{}{
				"action":      "consolidate",
				"items_count": len(consolidatedItems),
				"target_type": request.TargetType,
				"priority":    request.Priority,
			},
		}
		wm.eventBus.Publish(context.Background(), event)
	}

	wm.logger.Info("Processed consolidation request",
		slog.Int("items", len(consolidatedItems)),
		slog.String("target_type", string(request.TargetType)))
}

// min is defined in types.go

// Close gracefully shuts down working memory
func (wm *WorkingMemory) Close() error {
	close(wm.consolidator.consolidationQ)
	wm.logger.Info("Working memory shut down")
	return nil
}
