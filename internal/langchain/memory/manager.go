package memory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/koopa0/assistant/internal/config"
	"github.com/koopa0/assistant/internal/storage/postgres"
)

// MemoryType represents different types of memory
type MemoryType string

const (
	MemoryTypeShortTerm       MemoryType = "short_term"
	MemoryTypeLongTerm        MemoryType = "long_term"
	MemoryTypeTool            MemoryType = "tool"
	MemoryTypePersonalization MemoryType = "personalization"
)

// MemoryEntry represents a single memory entry
type MemoryEntry struct {
	ID          string                 `json:"id"`
	Type        MemoryType             `json:"type"`
	UserID      string                 `json:"user_id"`
	SessionID   string                 `json:"session_id,omitempty"`
	Content     string                 `json:"content"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Embedding   []float64              `json:"embedding,omitempty"`
	Importance  float64                `json:"importance"` // 0.0 to 1.0
	AccessCount int                    `json:"access_count"`
	LastAccess  time.Time              `json:"last_access"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MemoryQuery represents a query for retrieving memories
type MemoryQuery struct {
	UserID        string                 `json:"user_id"`
	SessionID     string                 `json:"session_id,omitempty"`
	Types         []MemoryType           `json:"types,omitempty"`
	Content       string                 `json:"content,omitempty"`
	Embedding     []float64              `json:"embedding,omitempty"`
	Similarity    float64                `json:"similarity,omitempty"`
	Limit         int                    `json:"limit,omitempty"`
	TimeRange     *TimeRange             `json:"time_range,omitempty"`
	MinImportance float64                `json:"min_importance,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
}

// TimeRange represents a time range for memory queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MemorySearchResult represents a memory search result with similarity score
type MemorySearchResult struct {
	Entry      *MemoryEntry `json:"entry"`
	Similarity float64      `json:"similarity"`
	Relevance  float64      `json:"relevance"`
}

// MemoryManager manages all types of memory for the LangChain system
type MemoryManager struct {
	shortTermMemory       *ShortTermMemory
	longTermMemory        *LongTermMemory
	toolMemory            *ToolMemory
	personalizationMemory *PersonalizationMemory
	dbClient              *postgres.SQLCClient
	config                config.LangChain
	logger                *slog.Logger
	mu                    sync.RWMutex
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(dbClient *postgres.SQLCClient, config config.LangChain, logger *slog.Logger) *MemoryManager {
	manager := &MemoryManager{
		dbClient: dbClient,
		config:   config,
		logger:   logger,
	}

	// Initialize memory components
	manager.shortTermMemory = NewShortTermMemory(config, logger)
	manager.longTermMemory = NewLongTermMemory(dbClient, config, logger)
	manager.toolMemory = NewToolMemory(config, logger)
	manager.personalizationMemory = NewPersonalizationMemory(dbClient, config, logger)

	return manager
}

// Store stores a memory entry in the appropriate memory system
func (mm *MemoryManager) Store(ctx context.Context, entry *MemoryEntry) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Set creation time if not set
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	// Set last access time
	entry.LastAccess = time.Now()

	mm.logger.Debug("Storing memory entry",
		slog.String("type", string(entry.Type)),
		slog.String("user_id", entry.UserID),
		slog.Float64("importance", entry.Importance))

	// Route to appropriate memory system based on type
	switch entry.Type {
	case MemoryTypeShortTerm:
		return mm.shortTermMemory.Store(ctx, entry)
	case MemoryTypeLongTerm:
		return mm.longTermMemory.Store(ctx, entry)
	case MemoryTypeTool:
		return mm.toolMemory.Store(ctx, entry)
	case MemoryTypePersonalization:
		return mm.personalizationMemory.Store(ctx, entry)
	default:
		return fmt.Errorf("unknown memory type: %s", entry.Type)
	}
}

// Retrieve retrieves memories based on a query
func (mm *MemoryManager) Retrieve(ctx context.Context, query *MemoryQuery) ([]*MemorySearchResult, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	results := make([]*MemorySearchResult, 0)

	mm.logger.Debug("Retrieving memories",
		slog.String("user_id", query.UserID),
		slog.Any("types", query.Types),
		slog.Int("limit", query.Limit))

	// If no specific types are requested, search all types
	types := query.Types
	if len(types) == 0 {
		types = []MemoryType{
			MemoryTypeShortTerm,
			MemoryTypeLongTerm,
			MemoryTypeTool,
			MemoryTypePersonalization,
		}
	}

	// Search each requested memory type
	for _, memType := range types {
		var typeResults []*MemorySearchResult
		var err error

		switch memType {
		case MemoryTypeShortTerm:
			typeResults, err = mm.shortTermMemory.Search(ctx, query)
		case MemoryTypeLongTerm:
			typeResults, err = mm.longTermMemory.Search(ctx, query)
		case MemoryTypeTool:
			typeResults, err = mm.toolMemory.Search(ctx, query)
		case MemoryTypePersonalization:
			typeResults, err = mm.personalizationMemory.Search(ctx, query)
		default:
			mm.logger.Warn("Unknown memory type in query", slog.String("type", string(memType)))
			continue
		}

		if err != nil {
			mm.logger.Warn("Failed to search memory type",
				slog.String("type", string(memType)),
				slog.Any("error", err))
			continue
		}

		results = append(results, typeResults...)
	}

	// Sort results by relevance and limit
	results = mm.sortAndLimitResults(results, query.Limit)

	mm.logger.Debug("Memory retrieval completed",
		slog.Int("results_count", len(results)),
		slog.String("user_id", query.UserID))

	return results, nil
}

// Update updates an existing memory entry
func (mm *MemoryManager) Update(ctx context.Context, entry *MemoryEntry) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Update last access time
	entry.LastAccess = time.Now()
	entry.AccessCount++

	mm.logger.Debug("Updating memory entry",
		slog.String("id", entry.ID),
		slog.String("type", string(entry.Type)))

	// Route to appropriate memory system
	switch entry.Type {
	case MemoryTypeShortTerm:
		return mm.shortTermMemory.Update(ctx, entry)
	case MemoryTypeLongTerm:
		return mm.longTermMemory.Update(ctx, entry)
	case MemoryTypeTool:
		return mm.toolMemory.Update(ctx, entry)
	case MemoryTypePersonalization:
		return mm.personalizationMemory.Update(ctx, entry)
	default:
		return fmt.Errorf("unknown memory type: %s", entry.Type)
	}
}

// Delete deletes a memory entry
func (mm *MemoryManager) Delete(ctx context.Context, entryID string, memType MemoryType) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.logger.Debug("Deleting memory entry",
		slog.String("id", entryID),
		slog.String("type", string(memType)))

	// Route to appropriate memory system
	switch memType {
	case MemoryTypeShortTerm:
		return mm.shortTermMemory.Delete(ctx, entryID)
	case MemoryTypeLongTerm:
		return mm.longTermMemory.Delete(ctx, entryID)
	case MemoryTypeTool:
		return mm.toolMemory.Delete(ctx, entryID)
	case MemoryTypePersonalization:
		return mm.personalizationMemory.Delete(ctx, entryID)
	default:
		return fmt.Errorf("unknown memory type: %s", memType)
	}
}

// Clear clears memories based on criteria
func (mm *MemoryManager) Clear(ctx context.Context, userID string, memTypes []MemoryType, olderThan *time.Time) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.logger.Info("Clearing memories",
		slog.String("user_id", userID),
		slog.Any("types", memTypes),
		slog.Any("older_than", olderThan))

	// If no types specified, clear all types
	if len(memTypes) == 0 {
		memTypes = []MemoryType{
			MemoryTypeShortTerm,
			MemoryTypeLongTerm,
			MemoryTypeTool,
			MemoryTypePersonalization,
		}
	}

	// Clear each memory type
	for _, memType := range memTypes {
		var err error

		switch memType {
		case MemoryTypeShortTerm:
			err = mm.shortTermMemory.Clear(ctx, userID, olderThan)
		case MemoryTypeLongTerm:
			err = mm.longTermMemory.Clear(ctx, userID, olderThan)
		case MemoryTypeTool:
			err = mm.toolMemory.Clear(ctx, userID, olderThan)
		case MemoryTypePersonalization:
			err = mm.personalizationMemory.Clear(ctx, userID, olderThan)
		default:
			mm.logger.Warn("Unknown memory type in clear", slog.String("type", string(memType)))
			continue
		}

		if err != nil {
			mm.logger.Error("Failed to clear memory type",
				slog.String("type", string(memType)),
				slog.Any("error", err))
			return fmt.Errorf("failed to clear %s memory: %w", memType, err)
		}
	}

	return nil
}

// GetStats returns memory statistics
func (mm *MemoryManager) GetStats(ctx context.Context, userID string) (*MemoryStats, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	stats := &MemoryStats{
		UserID:    userID,
		Timestamp: time.Now(),
	}

	// Get stats from each memory type
	if shortStats, err := mm.shortTermMemory.GetStats(ctx, userID); err == nil {
		stats.ShortTerm = shortStats
	}

	if longStats, err := mm.longTermMemory.GetStats(ctx, userID); err == nil {
		stats.LongTerm = longStats
	}

	if toolStats, err := mm.toolMemory.GetStats(ctx, userID); err == nil {
		stats.Tool = toolStats
	}

	if personalStats, err := mm.personalizationMemory.GetStats(ctx, userID); err == nil {
		stats.Personalization = personalStats
	}

	// Calculate totals
	stats.TotalEntries = stats.ShortTerm.EntryCount + stats.LongTerm.EntryCount +
		stats.Tool.EntryCount + stats.Personalization.EntryCount
	stats.TotalSize = stats.ShortTerm.TotalSize + stats.LongTerm.TotalSize +
		stats.Tool.TotalSize + stats.Personalization.TotalSize

	return stats, nil
}

// Cleanup performs maintenance tasks like removing expired entries
func (mm *MemoryManager) Cleanup(ctx context.Context) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.logger.Info("Starting memory cleanup")

	// Cleanup each memory type
	if err := mm.shortTermMemory.Cleanup(ctx); err != nil {
		mm.logger.Error("Short-term memory cleanup failed", slog.Any("error", err))
	}

	if err := mm.longTermMemory.Cleanup(ctx); err != nil {
		mm.logger.Error("Long-term memory cleanup failed", slog.Any("error", err))
	}

	if err := mm.toolMemory.Cleanup(ctx); err != nil {
		mm.logger.Error("Tool memory cleanup failed", slog.Any("error", err))
	}

	if err := mm.personalizationMemory.Cleanup(ctx); err != nil {
		mm.logger.Error("Personalization memory cleanup failed", slog.Any("error", err))
	}

	mm.logger.Info("Memory cleanup completed")
	return nil
}

// sortAndLimitResults sorts results by relevance and applies limit
func (mm *MemoryManager) sortAndLimitResults(results []*MemorySearchResult, limit int) []*MemorySearchResult {
	// Sort by relevance (descending)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Relevance < results[j].Relevance {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	UserID          string           `json:"user_id"`
	Timestamp       time.Time        `json:"timestamp"`
	TotalEntries    int              `json:"total_entries"`
	TotalSize       int64            `json:"total_size"`
	ShortTerm       *MemoryTypeStats `json:"short_term"`
	LongTerm        *MemoryTypeStats `json:"long_term"`
	Tool            *MemoryTypeStats `json:"tool"`
	Personalization *MemoryTypeStats `json:"personalization"`
}

// MemoryTypeStats represents statistics for a specific memory type
type MemoryTypeStats struct {
	EntryCount        int       `json:"entry_count"`
	TotalSize         int64     `json:"total_size"`
	OldestEntry       time.Time `json:"oldest_entry"`
	NewestEntry       time.Time `json:"newest_entry"`
	AverageImportance float64   `json:"average_importance"`
}
