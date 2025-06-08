package memory

import (
	"fmt"
)

// MemoryNotFoundError indicates a memory entry was not found
type MemoryNotFoundError struct {
	MemoryID string
}

func (e MemoryNotFoundError) Error() string {
	return fmt.Sprintf("memory entry not found: %s", e.MemoryID)
}

// MemoryValidationError indicates invalid memory entry data
type MemoryValidationError struct {
	Field   string
	Message string
}

func (e MemoryValidationError) Error() string {
	return fmt.Sprintf("memory validation error on field %s: %s", e.Field, e.Message)
}

// WorkingMemoryFullError indicates working memory is at capacity
type WorkingMemoryFullError struct {
	Capacity int
	Current  int
}

func (e WorkingMemoryFullError) Error() string {
	return fmt.Sprintf("working memory full: %d/%d entries", e.Current, e.Capacity)
}
