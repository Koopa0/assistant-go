package memory

import (
	"context"
	"fmt"
	"sync"

	coremem "github.com/koopa0/assistant-go/internal/core/memory"
)

// WorkingMemory provides fast in-memory storage
// Simple LRU-style implementation for working memory
type WorkingMemory struct {
	entries  map[string]coremem.Entry
	order    []string // Track insertion order
	capacity int
	mu       sync.RWMutex
}

// NewWorkingMemory creates a new working memory
func NewWorkingMemory(capacity int) *WorkingMemory {
	return &WorkingMemory{
		entries:  make(map[string]coremem.Entry),
		order:    make([]string, 0, capacity),
		capacity: capacity,
	}
}

// Store adds an entry to working memory
func (w *WorkingMemory) Store(entry coremem.Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// If at capacity, remove oldest
	if len(w.entries) >= w.capacity {
		oldest := w.order[0]
		delete(w.entries, oldest)
		w.order = w.order[1:]
	}

	// Add new entry
	w.entries[entry.ID] = entry
	w.order = append(w.order, entry.ID)

	return nil
}

// Get retrieves an entry by ID
func (w *WorkingMemory) Get(id string) *coremem.Entry {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if entry, exists := w.entries[id]; exists {
		// Move to end (most recently accessed)
		w.moveToEnd(id)
		return &entry
	}
	return nil
}

// Search finds entries in working memory
func (w *WorkingMemory) Search(criteria coremem.SearchCriteria) []coremem.Entry {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var results []coremem.Entry
	for _, entry := range w.entries {
		if entry.UserID != criteria.UserID {
			continue
		}

		// Apply filters
		if criteria.MinImportance > 0 && entry.Importance < criteria.MinImportance {
			continue
		}
		if criteria.Since != nil && entry.CreatedAt.Before(*criteria.Since) {
			continue
		}

		results = append(results, entry)
	}

	return results
}

// Delete removes an entry from working memory
func (w *WorkingMemory) Delete(id string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.entries[id]; exists {
		delete(w.entries, id)
		// Remove from order
		for i, oid := range w.order {
			if oid == id {
				w.order = append(w.order[:i], w.order[i+1:]...)
				break
			}
		}
		return true
	}
	return false
}

// moveToEnd moves an entry to the end of the order (without lock)
func (w *WorkingMemory) moveToEnd(id string) {
	// Find and remove from current position
	for i, oid := range w.order {
		if oid == id {
			w.order = append(w.order[:i], w.order[i+1:]...)
			break
		}
	}
	// Add to end
	w.order = append(w.order, id)
}

// workingMemoryWrapper adapts WorkingMemory to the core Memory interface
type workingMemoryWrapper struct {
	working *WorkingMemory
}

func (w *workingMemoryWrapper) Store(ctx context.Context, entry coremem.Entry) error {
	return w.working.Store(entry)
}

func (w *workingMemoryWrapper) Retrieve(ctx context.Context, id string) (*coremem.Entry, error) {
	if entry := w.working.Get(id); entry != nil {
		return entry, nil
	}
	return nil, fmt.Errorf("entry not found: %s", id)
}

func (w *workingMemoryWrapper) Search(ctx context.Context, criteria coremem.SearchCriteria) ([]coremem.Entry, error) {
	return w.working.Search(criteria), nil
}

func (w *workingMemoryWrapper) Delete(ctx context.Context, id string) error {
	if w.working.Delete(id) {
		return nil
	}
	return fmt.Errorf("entry not found: %s", id)
}
