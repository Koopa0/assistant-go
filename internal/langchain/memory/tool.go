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

// ToolMemory implements caching for tool execution results to optimize performance
type ToolMemory struct {
	cache      map[string]*ToolCacheEntry
	userCaches map[string][]string // userID -> list of cache keys
	maxSize    int
	maxAge     time.Duration
	config     config.LangChain
	logger     *slog.Logger
	mu         sync.RWMutex
}

// ToolCacheEntry represents a cached tool execution result
type ToolCacheEntry struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	ToolName      string                 `json:"tool_name"`
	InputHash     string                 `json:"input_hash"`
	Input         map[string]interface{} `json:"input"`
	Output        interface{}            `json:"output"`
	ExecutionTime time.Duration          `json:"execution_time"`
	Success       bool                   `json:"success"`
	Error         string                 `json:"error,omitempty"`
	HitCount      int                    `json:"hit_count"`
	LastHit       time.Time              `json:"last_hit"`
	CreatedAt     time.Time              `json:"created_at"`
	ExpiresAt     time.Time              `json:"expires_at"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// NewToolMemory creates a new tool memory instance
func NewToolMemory(config config.LangChain, logger *slog.Logger) *ToolMemory {
	return &ToolMemory{
		cache:      make(map[string]*ToolCacheEntry),
		userCaches: make(map[string][]string),
		maxSize:    1000,          // Default cache size
		maxAge:     time.Hour * 6, // Default 6 hours
		config:     config,
		logger:     logger,
	}
}

// Store stores a memory entry (tool cache entry) in tool memory
func (tm *ToolMemory) Store(ctx context.Context, entry *MemoryEntry) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Convert memory entry to tool cache entry
	cacheEntry, err := tm.memoryEntryToCacheEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to convert memory entry to cache entry: %w", err)
	}

	// Generate cache key
	cacheKey := tm.generateCacheKey(cacheEntry.UserID, cacheEntry.ToolName, cacheEntry.InputHash)
	cacheEntry.ID = cacheKey

	// Set expiration
	cacheEntry.ExpiresAt = time.Now().Add(tm.maxAge)

	// Store in cache
	tm.cache[cacheKey] = cacheEntry

	// Add to user cache tracking
	if _, exists := tm.userCaches[cacheEntry.UserID]; !exists {
		tm.userCaches[cacheEntry.UserID] = make([]string, 0)
	}
	tm.userCaches[cacheEntry.UserID] = append(tm.userCaches[cacheEntry.UserID], cacheKey)

	// Enforce size limits
	tm.enforceSize()

	tm.logger.Debug("Stored tool cache entry",
		slog.String("cache_key", cacheKey),
		slog.String("tool_name", cacheEntry.ToolName),
		slog.String("user_id", cacheEntry.UserID))

	return nil
}

// Search searches for cached tool results
func (tm *ToolMemory) Search(ctx context.Context, query *MemoryQuery) ([]*MemorySearchResult, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	results := make([]*MemorySearchResult, 0)

	// Get user's cache entries
	userCaches, exists := tm.userCaches[query.UserID]
	if !exists {
		return results, nil
	}

	for _, cacheKey := range userCaches {
		cacheEntry, exists := tm.cache[cacheKey]
		if !exists {
			continue
		}

		// Check if entry has expired
		if time.Now().After(cacheEntry.ExpiresAt) {
			continue
		}

		// Convert cache entry back to memory entry for matching
		memoryEntry := tm.cacheEntryToMemoryEntry(cacheEntry)

		// Apply filters
		if !tm.matchesQuery(memoryEntry, query) {
			continue
		}

		// Calculate similarity and relevance
		similarity := tm.calculateSimilarity(memoryEntry, query)
		relevance := tm.calculateRelevance(cacheEntry, query)

		result := &MemorySearchResult{
			Entry:      memoryEntry,
			Similarity: similarity,
			Relevance:  relevance,
		}

		results = append(results, result)
	}

	tm.logger.Debug("Tool memory search completed",
		slog.String("user_id", query.UserID),
		slog.Int("results", len(results)))

	return results, nil
}

// GetCachedResult retrieves a cached tool result by tool name and input hash
func (tm *ToolMemory) GetCachedResult(ctx context.Context, userID, toolName, inputHash string) (*ToolCacheEntry, bool) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	cacheKey := tm.generateCacheKey(userID, toolName, inputHash)
	cacheEntry, exists := tm.cache[cacheKey]

	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(cacheEntry.ExpiresAt) {
		// Remove expired entry
		delete(tm.cache, cacheKey)
		tm.removeFromUserCache(userID, cacheKey)
		return nil, false
	}

	// Update hit statistics
	cacheEntry.HitCount++
	cacheEntry.LastHit = time.Now()

	tm.logger.Debug("Cache hit",
		slog.String("tool_name", toolName),
		slog.String("user_id", userID),
		slog.Int("hit_count", cacheEntry.HitCount))

	return cacheEntry, true
}

// CacheToolResult caches a tool execution result
func (tm *ToolMemory) CacheToolResult(ctx context.Context, userID, toolName string, input map[string]interface{}, output interface{}, executionTime time.Duration, success bool, errorMsg string) error {
	inputHash := tm.hashInput(input)

	cacheEntry := &ToolCacheEntry{
		UserID:        userID,
		ToolName:      toolName,
		InputHash:     inputHash,
		Input:         input,
		Output:        output,
		ExecutionTime: executionTime,
		Success:       success,
		Error:         errorMsg,
		HitCount:      0,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(tm.maxAge),
		Metadata:      make(map[string]interface{}),
	}

	// Convert to memory entry and store
	memoryEntry := tm.cacheEntryToMemoryEntry(cacheEntry)
	return tm.Store(ctx, memoryEntry)
}

// Update updates an existing memory entry
func (tm *ToolMemory) Update(ctx context.Context, entry *MemoryEntry) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	_, exists := tm.cache[entry.ID]
	if !exists {
		return fmt.Errorf("cache entry not found: %s", entry.ID)
	}

	// Update cache entry from memory entry
	updatedCacheEntry, err := tm.memoryEntryToCacheEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to convert memory entry: %w", err)
	}

	updatedCacheEntry.ID = entry.ID
	tm.cache[entry.ID] = updatedCacheEntry

	tm.logger.Debug("Updated tool cache entry",
		slog.String("id", entry.ID))

	return nil
}

// Delete deletes a memory entry
func (tm *ToolMemory) Delete(ctx context.Context, entryID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	cacheEntry, exists := tm.cache[entryID]
	if !exists {
		return fmt.Errorf("cache entry not found: %s", entryID)
	}

	// Remove from cache
	delete(tm.cache, entryID)

	// Remove from user cache tracking
	tm.removeFromUserCache(cacheEntry.UserID, entryID)

	tm.logger.Debug("Deleted tool cache entry",
		slog.String("id", entryID))

	return nil
}

// Clear clears tool cache for a user
func (tm *ToolMemory) Clear(ctx context.Context, userID string, olderThan *time.Time) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	userCaches, exists := tm.userCaches[userID]
	if !exists {
		return nil
	}

	entriesToDelete := make([]string, 0)

	for _, cacheKey := range userCaches {
		cacheEntry, exists := tm.cache[cacheKey]
		if !exists {
			continue
		}

		// Check if entry should be deleted based on age
		if olderThan != nil && cacheEntry.CreatedAt.After(*olderThan) {
			continue
		}

		entriesToDelete = append(entriesToDelete, cacheKey)
	}

	// Delete entries
	for _, cacheKey := range entriesToDelete {
		delete(tm.cache, cacheKey)
	}

	// Update user cache tracking
	if len(entriesToDelete) == len(userCaches) {
		delete(tm.userCaches, userID)
	} else {
		newUserCaches := make([]string, 0)
		for _, cacheKey := range userCaches {
			found := false
			for _, deletedKey := range entriesToDelete {
				if cacheKey == deletedKey {
					found = true
					break
				}
			}
			if !found {
				newUserCaches = append(newUserCaches, cacheKey)
			}
		}
		tm.userCaches[userID] = newUserCaches
	}

	tm.logger.Info("Cleared tool cache",
		slog.String("user_id", userID),
		slog.Int("deleted_count", len(entriesToDelete)))

	return nil
}

// GetStats returns statistics for tool memory
func (tm *ToolMemory) GetStats(ctx context.Context, userID string) (*MemoryTypeStats, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := &MemoryTypeStats{
		EntryCount: 0,
		TotalSize:  0,
	}

	userCaches, exists := tm.userCaches[userID]
	if !exists {
		return stats, nil
	}

	var oldestTime, newestTime time.Time
	validEntries := 0

	for _, cacheKey := range userCaches {
		cacheEntry, exists := tm.cache[cacheKey]
		if !exists {
			continue
		}

		// Skip expired entries
		if time.Now().After(cacheEntry.ExpiresAt) {
			continue
		}

		validEntries++
		// Estimate size based on content
		stats.TotalSize += int64(len(fmt.Sprintf("%v", cacheEntry.Output)))

		if oldestTime.IsZero() || cacheEntry.CreatedAt.Before(oldestTime) {
			oldestTime = cacheEntry.CreatedAt
		}
		if newestTime.IsZero() || cacheEntry.CreatedAt.After(newestTime) {
			newestTime = cacheEntry.CreatedAt
		}
	}

	stats.EntryCount = validEntries
	stats.OldestEntry = oldestTime
	stats.NewestEntry = newestTime

	return stats, nil
}

// Cleanup removes expired cache entries
func (tm *ToolMemory) Cleanup(ctx context.Context) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	expiredEntries := make([]string, 0)

	// Find expired entries
	for cacheKey, cacheEntry := range tm.cache {
		if now.After(cacheEntry.ExpiresAt) {
			expiredEntries = append(expiredEntries, cacheKey)
		}
	}

	// Remove expired entries
	for _, cacheKey := range expiredEntries {
		cacheEntry := tm.cache[cacheKey]
		delete(tm.cache, cacheKey)
		tm.removeFromUserCache(cacheEntry.UserID, cacheKey)
	}

	tm.logger.Info("Tool memory cleanup completed",
		slog.Int("expired_entries", len(expiredEntries)))

	return nil
}

// Helper methods

func (tm *ToolMemory) generateCacheKey(userID, toolName, inputHash string) string {
	return fmt.Sprintf("tool_%s_%s_%s", userID, toolName, inputHash)
}

func (tm *ToolMemory) hashInput(input map[string]interface{}) string {
	// Simple hash based on string representation
	// In production, use a proper hash function
	inputStr := fmt.Sprintf("%v", input)
	return fmt.Sprintf("%x", len(inputStr)) // Simplified hash
}

func (tm *ToolMemory) memoryEntryToCacheEntry(entry *MemoryEntry) (*ToolCacheEntry, error) {
	// Extract tool-specific information from memory entry context
	toolName, _ := entry.Context["tool_name"].(string)
	inputHash, _ := entry.Context["input_hash"].(string)
	input, _ := entry.Context["input"].(map[string]interface{})
	output := entry.Context["output"]
	executionTime, _ := entry.Context["execution_time"].(time.Duration)
	success, _ := entry.Context["success"].(bool)
	errorMsg, _ := entry.Context["error"].(string)

	// Handle ExpiresAt safely
	var expiresAt time.Time
	if entry.ExpiresAt != nil {
		expiresAt = *entry.ExpiresAt
	} else {
		expiresAt = time.Now().Add(time.Hour * 6) // Default 6 hours
	}

	return &ToolCacheEntry{
		ID:            entry.ID,
		UserID:        entry.UserID,
		ToolName:      toolName,
		InputHash:     inputHash,
		Input:         input,
		Output:        output,
		ExecutionTime: executionTime,
		Success:       success,
		Error:         errorMsg,
		CreatedAt:     entry.CreatedAt,
		ExpiresAt:     expiresAt,
		Metadata:      entry.Metadata,
	}, nil
}

func (tm *ToolMemory) cacheEntryToMemoryEntry(cacheEntry *ToolCacheEntry) *MemoryEntry {
	return &MemoryEntry{
		ID:      cacheEntry.ID,
		Type:    MemoryTypeTool,
		UserID:  cacheEntry.UserID,
		Content: fmt.Sprintf("Tool: %s, Success: %v", cacheEntry.ToolName, cacheEntry.Success),
		Context: map[string]interface{}{
			"tool_name":      cacheEntry.ToolName,
			"input_hash":     cacheEntry.InputHash,
			"input":          cacheEntry.Input,
			"output":         cacheEntry.Output,
			"execution_time": cacheEntry.ExecutionTime,
			"success":        cacheEntry.Success,
			"error":          cacheEntry.Error,
		},
		Importance:  0.5, // Default importance for tool cache
		AccessCount: cacheEntry.HitCount,
		LastAccess:  cacheEntry.LastHit,
		CreatedAt:   cacheEntry.CreatedAt,
		ExpiresAt:   &cacheEntry.ExpiresAt,
		Metadata:    cacheEntry.Metadata,
	}
}

func (tm *ToolMemory) removeFromUserCache(userID, cacheKey string) {
	if userCaches, exists := tm.userCaches[userID]; exists {
		for i, key := range userCaches {
			if key == cacheKey {
				tm.userCaches[userID] = append(userCaches[:i], userCaches[i+1:]...)
				break
			}
		}
	}
}

func (tm *ToolMemory) enforceSize() {
	if len(tm.cache) <= tm.maxSize {
		return
	}

	// Remove oldest entries to stay within limit
	entriesToRemove := len(tm.cache) - tm.maxSize

	// Find oldest entries
	oldestKeys := make([]string, 0)
	for i := 0; i < entriesToRemove; i++ {
		oldestKey := ""
		var oldestTime time.Time

		for cacheKey, cacheEntry := range tm.cache {
			// Skip already selected entries
			found := false
			for _, selected := range oldestKeys {
				if selected == cacheKey {
					found = true
					break
				}
			}
			if found {
				continue
			}

			if oldestKey == "" || cacheEntry.CreatedAt.Before(oldestTime) {
				oldestKey = cacheKey
				oldestTime = cacheEntry.CreatedAt
			}
		}

		if oldestKey != "" {
			oldestKeys = append(oldestKeys, oldestKey)
		}
	}

	// Remove oldest entries
	for _, cacheKey := range oldestKeys {
		cacheEntry := tm.cache[cacheKey]
		delete(tm.cache, cacheKey)
		tm.removeFromUserCache(cacheEntry.UserID, cacheKey)
	}

	tm.logger.Debug("Enforced size limit for tool memory",
		slog.Int("removed_entries", len(oldestKeys)))
}

func (tm *ToolMemory) matchesQuery(entry *MemoryEntry, query *MemoryQuery) bool {
	// Check content match if specified
	if query.Content != "" {
		if !strings.Contains(strings.ToLower(entry.Content), strings.ToLower(query.Content)) {
			return false
		}
	}

	// Check time range
	if query.TimeRange != nil {
		if entry.CreatedAt.Before(query.TimeRange.Start) || entry.CreatedAt.After(query.TimeRange.End) {
			return false
		}
	}

	return true
}

func (tm *ToolMemory) calculateSimilarity(entry *MemoryEntry, query *MemoryQuery) float64 {
	if query.Content == "" {
		return 1.0
	}

	// Simple text similarity
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

func (tm *ToolMemory) calculateRelevance(cacheEntry *ToolCacheEntry, query *MemoryQuery) float64 {
	// Factor in hit count and recency
	hitScore := float64(cacheEntry.HitCount) / 10.0 // Normalize hit count
	if hitScore > 1.0 {
		hitScore = 1.0
	}

	recency := 1.0 - (time.Since(cacheEntry.CreatedAt).Hours() / 24.0) // Decay over 24 hours
	if recency < 0 {
		recency = 0
	}

	// Success factor
	successScore := 0.0
	if cacheEntry.Success {
		successScore = 1.0
	}

	// Weighted combination
	relevance := (hitScore * 0.4) + (recency * 0.3) + (successScore * 0.3)

	return relevance
}
