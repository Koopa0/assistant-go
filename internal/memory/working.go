package memory

import (
	"fmt"
	"sync"
	"time"
)

// WorkingMemory provides fast, in-memory storage for active context
// Using value semantics for entries to avoid pointer chasing
type WorkingMemory struct {
	entries  map[string]WorkingMemoryEntry
	capacity int
	mu       sync.RWMutex
}

// NewWorkingMemory creates a new working memory instance
func NewWorkingMemory(capacity int) *WorkingMemory {
	if capacity <= 0 {
		capacity = DefaultMaxWorkingSize
	}
	return &WorkingMemory{
		entries:  make(map[string]WorkingMemoryEntry, capacity),
		capacity: capacity,
	}
}

// Store adds an entry to working memory
func (wm *WorkingMemory) Store(entry Entry) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Check capacity
	if len(wm.entries) >= wm.capacity && wm.entries[entry.ID].Entry.ID == "" {
		// Evict least important entry
		if err := wm.evictLeastImportant(); err != nil {
			return fmt.Errorf("failed to evict entry: %w", err)
		}
	}

	// Create working memory entry
	wmEntry := WorkingMemoryEntry{
		Entry:       entry,
		Priority:    entry.Importance,
		LastUpdated: time.Now(),
	}

	// Ensure proper defaults
	if wmEntry.Entry.Type == "" {
		wmEntry.Entry.Type = TypeWorking
	}
	if wmEntry.Entry.CreatedAt.IsZero() {
		wmEntry.Entry.CreatedAt = time.Now()
	}
	if wmEntry.Entry.UpdatedAt.IsZero() {
		wmEntry.Entry.UpdatedAt = time.Now()
	}
	if wmEntry.Entry.LastAccess.IsZero() {
		wmEntry.Entry.LastAccess = time.Now()
	}

	wm.entries[entry.ID] = wmEntry
	return nil
}

// Get retrieves an entry by ID
func (wm *WorkingMemory) Get(id string) *Entry {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	if wmEntry, exists := wm.entries[id]; exists {
		// Update access time
		wm.mu.RUnlock()
		wm.mu.Lock()
		wmEntry.Entry.AccessCount++
		wmEntry.Entry.LastAccess = time.Now()
		wmEntry.LastUpdated = time.Now()
		wm.entries[id] = wmEntry
		wm.mu.Unlock()
		wm.mu.RLock()

		// Return a copy to prevent external modification
		entryCopy := wmEntry.Entry
		return &entryCopy
	}
	return nil
}

// Update updates an existing entry
func (wm *WorkingMemory) Update(entry Entry) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.entries[entry.ID]; !exists {
		return MemoryNotFoundError{MemoryID: entry.ID}
	}

	wmEntry := WorkingMemoryEntry{
		Entry:       entry,
		Priority:    entry.Importance,
		LastUpdated: time.Now(),
	}
	wmEntry.Entry.UpdatedAt = time.Now()
	wmEntry.Entry.LastAccess = time.Now()

	wm.entries[entry.ID] = wmEntry
	return nil
}

// Delete removes an entry by ID
func (wm *WorkingMemory) Delete(id string) bool {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.entries[id]; exists {
		delete(wm.entries, id)
		return true
	}
	return false
}

// Search finds entries matching criteria
func (wm *WorkingMemory) Search(criteria SearchCriteria) []Entry {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var results []Entry

	for _, wmEntry := range wm.entries {
		entry := wmEntry.Entry

		// Filter by user ID
		if criteria.UserID != "" && entry.UserID != criteria.UserID {
			continue
		}

		// Filter by types
		if len(criteria.Types) > 0 && !containsType(criteria.Types, entry.Type) {
			continue
		}

		// Filter by importance
		if entry.Importance < criteria.ImportanceMin {
			continue
		}

		// Filter by query (simple substring match)
		if criteria.Query != "" && !contains(entry.Content, criteria.Query) {
			continue
		}

		// Filter by tags
		if len(criteria.Tags) > 0 && !hasAnyTag(entry.Metadata.Tags, criteria.Tags) {
			continue
		}

		// Filter by category
		if criteria.Category != "" && entry.Metadata.Category != criteria.Category {
			continue
		}

		// Filter by time range
		if criteria.TimeRange != nil {
			if entry.CreatedAt.Before(criteria.TimeRange.Start) ||
				entry.CreatedAt.After(criteria.TimeRange.End) {
				continue
			}
		}

		results = append(results, entry)
	}

	return results
}

// Clear removes all entries
func (wm *WorkingMemory) Clear() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.entries = make(map[string]WorkingMemoryEntry, wm.capacity)
}

// Size returns the number of entries
func (wm *WorkingMemory) Size() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return len(wm.entries)
}

// GetAll returns all entries (for debugging/testing)
func (wm *WorkingMemory) GetAll() []Entry {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	results := make([]Entry, 0, len(wm.entries))
	for _, wmEntry := range wm.entries {
		results = append(results, wmEntry.Entry)
	}
	return results
}

// evictLeastImportant removes the least important entry
func (wm *WorkingMemory) evictLeastImportant() error {
	if len(wm.entries) == 0 {
		return nil
	}

	var leastImportantID string
	var lowestScore float64 = 999999

	// Find least important entry based on importance and last access
	for id, wmEntry := range wm.entries {
		// Calculate score: importance + recency bonus
		recencyBonus := 1.0 / (1.0 + time.Since(wmEntry.LastUpdated).Hours())
		score := wmEntry.Priority + recencyBonus

		if score < lowestScore {
			lowestScore = score
			leastImportantID = id
		}
	}

	if leastImportantID != "" {
		delete(wm.entries, leastImportantID)
	}

	return nil
}

// Helper functions

func containsType(types []Type, t Type) bool {
	for _, typ := range types {
		if typ == t {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || contains(s[1:], substr) || contains(s[:len(s)-1], substr))
}

func hasAnyTag(entryTags, searchTags []string) bool {
	for _, searchTag := range searchTags {
		for _, entryTag := range entryTags {
			if entryTag == searchTag {
				return true
			}
		}
	}
	return false
}
