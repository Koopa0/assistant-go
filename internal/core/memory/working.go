package memory

import (
	"strings"
	"sync"
)

// WorkingMemory provides fast in-memory storage
// Simple LRU-style implementation for working memory
type WorkingMemory struct {
	entries  map[string]Entry
	order    []string // Track insertion order
	capacity int
	mu       sync.RWMutex
}

// NewWorkingMemory creates a new working memory
func NewWorkingMemory(capacity int) *WorkingMemory {
	return &WorkingMemory{
		entries:  make(map[string]Entry),
		order:    make([]string, 0, capacity),
		capacity: capacity,
	}
}

// Store adds an entry to working memory
func (w *WorkingMemory) Store(entry Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if entry already exists
	if _, exists := w.entries[entry.ID]; exists {
		// Update existing entry
		w.entries[entry.ID] = entry
		return nil
	}

	// Add new entry
	w.entries[entry.ID] = entry
	w.order = append(w.order, entry.ID)

	// Remove oldest if at capacity
	if len(w.order) > w.capacity {
		oldest := w.order[0]
		delete(w.entries, oldest)
		w.order = w.order[1:]
	}

	return nil
}

// Get retrieves an entry by ID
func (w *WorkingMemory) Get(id string) *Entry {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if entry, exists := w.entries[id]; exists {
		// Create a copy to avoid mutation
		entryCopy := entry
		return &entryCopy
	}
	return nil
}

// Search finds entries matching criteria
func (w *WorkingMemory) Search(criteria SearchCriteria) []Entry {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var results []Entry

	for _, entry := range w.entries {
		if w.matchesCriteria(entry, criteria) {
			results = append(results, entry)
		}
	}

	return results
}

// Delete removes an entry by ID
func (w *WorkingMemory) Delete(id string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.entries[id]; exists {
		delete(w.entries, id)

		// Remove from order slice
		for i, entryID := range w.order {
			if entryID == id {
				w.order = append(w.order[:i], w.order[i+1:]...)
				break
			}
		}
		return true
	}
	return false
}

// Clear removes all entries
func (w *WorkingMemory) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.entries = make(map[string]Entry)
	w.order = w.order[:0]
}

// Size returns the current number of entries
func (w *WorkingMemory) Size() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.entries)
}

// Capacity returns the maximum capacity
func (w *WorkingMemory) Capacity() int {
	return w.capacity
}

// matchesCriteria checks if an entry matches search criteria
func (w *WorkingMemory) matchesCriteria(entry Entry, criteria SearchCriteria) bool {
	// Check user ID
	if criteria.UserID != "" && entry.UserID != criteria.UserID {
		return false
	}

	// Check type filter
	if len(criteria.Types) > 0 {
		found := false
		for _, t := range criteria.Types {
			if entry.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check query (simple text search)
	if criteria.Query != "" {
		query := strings.ToLower(criteria.Query)
		content := strings.ToLower(entry.Content)
		if !strings.Contains(content, query) {
			return false
		}
	}

	// Check importance minimum
	if criteria.ImportanceMin > 0 && entry.Importance < criteria.ImportanceMin {
		return false
	}

	// Check time range
	if criteria.TimeFrom != nil && entry.CreatedAt.Before(*criteria.TimeFrom) {
		return false
	}
	if criteria.TimeTo != nil && entry.CreatedAt.After(*criteria.TimeTo) {
		return false
	}

	return true
}

// GetStats returns statistics about working memory
func (w *WorkingMemory) GetStats() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	stats := map[string]interface{}{
		"size":       len(w.entries),
		"capacity":   w.capacity,
		"usage":      float64(len(w.entries)) / float64(w.capacity),
		"order_size": len(w.order),
	}

	// Calculate importance statistics
	var totalImportance float64
	var maxImportance float64
	var minImportance float64 = 1.0

	for _, entry := range w.entries {
		totalImportance += entry.Importance
		if entry.Importance > maxImportance {
			maxImportance = entry.Importance
		}
		if entry.Importance < minImportance {
			minImportance = entry.Importance
		}
	}

	if len(w.entries) > 0 {
		stats["avg_importance"] = totalImportance / float64(len(w.entries))
		stats["max_importance"] = maxImportance
		stats["min_importance"] = minImportance
	}

	return stats
}
