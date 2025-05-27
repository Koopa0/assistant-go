package memory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
)

// ShortTermMemory implements buffer-based short-term memory storage
type ShortTermMemory struct {
	entries      map[string]*MemoryEntry
	userSessions map[string][]string // userID -> list of entry IDs
	maxSize      int
	maxAge       time.Duration
	config       config.LangChain
	logger       *slog.Logger
	mu           sync.RWMutex
}

// NewShortTermMemory creates a new short-term memory instance
func NewShortTermMemory(config config.LangChain, logger *slog.Logger) *ShortTermMemory {
	return &ShortTermMemory{
		entries:      make(map[string]*MemoryEntry),
		userSessions: make(map[string][]string),
		maxSize:      config.MemorySize,
		maxAge:       time.Hour * 24, // Default 24 hours
		config:       config,
		logger:       logger,
	}
}

// Store stores a memory entry in short-term memory
func (stm *ShortTermMemory) Store(ctx context.Context, entry *MemoryEntry) error {
	stm.mu.Lock()
	defer stm.mu.Unlock()

	// Generate ID if not provided
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("st_%s_%d", entry.UserID, time.Now().UnixNano())
	}

	// Set expiration if not set
	if entry.ExpiresAt == nil {
		expiresAt := time.Now().Add(stm.maxAge)
		entry.ExpiresAt = &expiresAt
	}

	// Store the entry
	stm.entries[entry.ID] = entry

	// Add to user session tracking
	if _, exists := stm.userSessions[entry.UserID]; !exists {
		stm.userSessions[entry.UserID] = make([]string, 0)
	}
	stm.userSessions[entry.UserID] = append(stm.userSessions[entry.UserID], entry.ID)

	// Enforce size limits
	stm.enforceSize(entry.UserID)

	stm.logger.Debug("Stored short-term memory entry",
		slog.String("id", entry.ID),
		slog.String("user_id", entry.UserID),
		slog.String("session_id", entry.SessionID))

	return nil
}

// Search searches for memories matching the query
func (stm *ShortTermMemory) Search(ctx context.Context, query *MemoryQuery) ([]*MemorySearchResult, error) {
	stm.mu.RLock()
	defer stm.mu.RUnlock()

	results := make([]*MemorySearchResult, 0)

	// Get user's entries
	userEntries, exists := stm.userSessions[query.UserID]
	if !exists {
		return results, nil
	}

	for _, entryID := range userEntries {
		entry, exists := stm.entries[entryID]
		if !exists {
			continue
		}

		// Check if entry has expired
		if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
			continue
		}

		// Apply filters
		if !stm.matchesQuery(entry, query) {
			continue
		}

		// Calculate similarity and relevance
		similarity := stm.calculateSimilarity(entry, query)
		relevance := stm.calculateRelevance(entry, query)

		result := &MemorySearchResult{
			Entry:      entry,
			Similarity: similarity,
			Relevance:  relevance,
		}

		results = append(results, result)
	}

	stm.logger.Debug("Short-term memory search completed",
		slog.String("user_id", query.UserID),
		slog.Int("results", len(results)))

	return results, nil
}

// Update updates an existing memory entry
func (stm *ShortTermMemory) Update(ctx context.Context, entry *MemoryEntry) error {
	stm.mu.Lock()
	defer stm.mu.Unlock()

	if _, exists := stm.entries[entry.ID]; !exists {
		return fmt.Errorf("entry not found: %s", entry.ID)
	}

	// Update the entry
	stm.entries[entry.ID] = entry

	stm.logger.Debug("Updated short-term memory entry",
		slog.String("id", entry.ID))

	return nil
}

// Delete deletes a memory entry
func (stm *ShortTermMemory) Delete(ctx context.Context, entryID string) error {
	stm.mu.Lock()
	defer stm.mu.Unlock()

	entry, exists := stm.entries[entryID]
	if !exists {
		return fmt.Errorf("entry not found: %s", entryID)
	}

	// Remove from entries
	delete(stm.entries, entryID)

	// Remove from user session tracking
	if userEntries, exists := stm.userSessions[entry.UserID]; exists {
		for i, id := range userEntries {
			if id == entryID {
				stm.userSessions[entry.UserID] = append(userEntries[:i], userEntries[i+1:]...)
				break
			}
		}
	}

	stm.logger.Debug("Deleted short-term memory entry",
		slog.String("id", entryID))

	return nil
}

// Clear clears memories for a user
func (stm *ShortTermMemory) Clear(ctx context.Context, userID string, olderThan *time.Time) error {
	stm.mu.Lock()
	defer stm.mu.Unlock()

	userEntries, exists := stm.userSessions[userID]
	if !exists {
		return nil
	}

	entriesToDelete := make([]string, 0)

	for _, entryID := range userEntries {
		entry, exists := stm.entries[entryID]
		if !exists {
			continue
		}

		// Check if entry should be deleted based on age
		if olderThan != nil && entry.CreatedAt.After(*olderThan) {
			continue
		}

		entriesToDelete = append(entriesToDelete, entryID)
	}

	// Delete entries
	for _, entryID := range entriesToDelete {
		delete(stm.entries, entryID)
	}

	// Update user session tracking
	if len(entriesToDelete) == len(userEntries) {
		// All entries deleted
		delete(stm.userSessions, userID)
	} else {
		// Remove deleted entries from tracking
		newUserEntries := make([]string, 0)
		for _, entryID := range userEntries {
			found := false
			for _, deletedID := range entriesToDelete {
				if entryID == deletedID {
					found = true
					break
				}
			}
			if !found {
				newUserEntries = append(newUserEntries, entryID)
			}
		}
		stm.userSessions[userID] = newUserEntries
	}

	stm.logger.Info("Cleared short-term memories",
		slog.String("user_id", userID),
		slog.Int("deleted_count", len(entriesToDelete)))

	return nil
}

// GetStats returns statistics for short-term memory
func (stm *ShortTermMemory) GetStats(ctx context.Context, userID string) (*MemoryTypeStats, error) {
	stm.mu.RLock()
	defer stm.mu.RUnlock()

	stats := &MemoryTypeStats{
		EntryCount: 0,
		TotalSize:  0,
	}

	userEntries, exists := stm.userSessions[userID]
	if !exists {
		return stats, nil
	}

	var oldestTime, newestTime time.Time
	var totalImportance float64
	validEntries := 0

	for _, entryID := range userEntries {
		entry, exists := stm.entries[entryID]
		if !exists {
			continue
		}

		// Skip expired entries
		if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
			continue
		}

		validEntries++
		stats.TotalSize += int64(len(entry.Content))
		totalImportance += entry.Importance

		if oldestTime.IsZero() || entry.CreatedAt.Before(oldestTime) {
			oldestTime = entry.CreatedAt
		}
		if newestTime.IsZero() || entry.CreatedAt.After(newestTime) {
			newestTime = entry.CreatedAt
		}
	}

	stats.EntryCount = validEntries
	stats.OldestEntry = oldestTime
	stats.NewestEntry = newestTime

	if validEntries > 0 {
		stats.AverageImportance = totalImportance / float64(validEntries)
	}

	return stats, nil
}

// Cleanup removes expired entries
func (stm *ShortTermMemory) Cleanup(ctx context.Context) error {
	stm.mu.Lock()
	defer stm.mu.Unlock()

	now := time.Now()
	expiredEntries := make([]string, 0)

	// Find expired entries
	for entryID, entry := range stm.entries {
		if entry.ExpiresAt != nil && now.After(*entry.ExpiresAt) {
			expiredEntries = append(expiredEntries, entryID)
		}
	}

	// Remove expired entries
	for _, entryID := range expiredEntries {
		entry := stm.entries[entryID]
		delete(stm.entries, entryID)

		// Remove from user session tracking
		if userEntries, exists := stm.userSessions[entry.UserID]; exists {
			for i, id := range userEntries {
				if id == entryID {
					stm.userSessions[entry.UserID] = append(userEntries[:i], userEntries[i+1:]...)
					break
				}
			}
		}
	}

	stm.logger.Info("Short-term memory cleanup completed",
		slog.Int("expired_entries", len(expiredEntries)))

	return nil
}

// enforceSize ensures the memory doesn't exceed size limits
func (stm *ShortTermMemory) enforceSize(userID string) {
	userEntries := stm.userSessions[userID]

	if len(userEntries) <= stm.maxSize {
		return
	}

	// Remove oldest entries to stay within limit
	entriesToRemove := len(userEntries) - stm.maxSize

	// Sort by creation time to find oldest entries
	oldestEntries := make([]string, 0)
	for i := 0; i < entriesToRemove; i++ {
		oldestID := ""
		var oldestTime time.Time

		for _, entryID := range userEntries {
			// Skip already selected entries
			found := false
			for _, selected := range oldestEntries {
				if selected == entryID {
					found = true
					break
				}
			}
			if found {
				continue
			}

			entry := stm.entries[entryID]
			if oldestID == "" || entry.CreatedAt.Before(oldestTime) {
				oldestID = entryID
				oldestTime = entry.CreatedAt
			}
		}

		if oldestID != "" {
			oldestEntries = append(oldestEntries, oldestID)
		}
	}

	// Remove oldest entries
	for _, entryID := range oldestEntries {
		delete(stm.entries, entryID)

		// Remove from user session tracking
		for i, id := range userEntries {
			if id == entryID {
				userEntries = append(userEntries[:i], userEntries[i+1:]...)
				break
			}
		}
	}

	stm.userSessions[userID] = userEntries

	stm.logger.Debug("Enforced size limit for short-term memory",
		slog.String("user_id", userID),
		slog.Int("removed_entries", len(oldestEntries)))
}

// matchesQuery checks if an entry matches the query criteria
func (stm *ShortTermMemory) matchesQuery(entry *MemoryEntry, query *MemoryQuery) bool {
	// Check session ID if specified
	if query.SessionID != "" && entry.SessionID != query.SessionID {
		return false
	}

	// Check minimum importance
	if query.MinImportance > 0 && entry.Importance < query.MinImportance {
		return false
	}

	// Check time range
	if query.TimeRange != nil {
		if entry.CreatedAt.Before(query.TimeRange.Start) || entry.CreatedAt.After(query.TimeRange.End) {
			return false
		}
	}

	// Check content match if specified
	if query.Content != "" {
		if !strings.Contains(strings.ToLower(entry.Content), strings.ToLower(query.Content)) {
			return false
		}
	}

	return true
}

// calculateSimilarity calculates similarity between entry and query
func (stm *ShortTermMemory) calculateSimilarity(entry *MemoryEntry, query *MemoryQuery) float64 {
	if query.Content == "" {
		return 1.0 // No content to compare
	}

	// Simple text similarity based on common words
	entryWords := strings.Fields(strings.ToLower(entry.Content))
	queryWords := strings.Fields(strings.ToLower(query.Content))

	if len(entryWords) == 0 || len(queryWords) == 0 {
		return 0.0
	}

	commonWords := 0
	for _, queryWord := range queryWords {
		for _, entryWord := range entryWords {
			if queryWord == entryWord {
				commonWords++
				break
			}
		}
	}

	return float64(commonWords) / float64(len(queryWords))
}

// calculateRelevance calculates overall relevance score
func (stm *ShortTermMemory) calculateRelevance(entry *MemoryEntry, query *MemoryQuery) float64 {
	similarity := stm.calculateSimilarity(entry, query)

	// Factor in importance and recency
	importance := entry.Importance
	recency := 1.0 - (time.Since(entry.CreatedAt).Hours() / 24.0) // Decay over 24 hours
	if recency < 0 {
		recency = 0
	}

	// Weighted combination
	relevance := (similarity * 0.4) + (importance * 0.3) + (recency * 0.3)

	return relevance
}
