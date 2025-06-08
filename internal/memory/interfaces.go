package memory

import (
	"context"
)

// MemoryReader defines methods for reading memory entries
type MemoryReader interface {
	Retrieve(ctx context.Context, id string) (*Entry, error)
	Search(ctx context.Context, criteria SearchCriteria) ([]Entry, error)
}

// MemoryWriter defines methods for writing memory entries
type MemoryWriter interface {
	Store(ctx context.Context, entry Entry) error
	Update(ctx context.Context, entry Entry) error
	Delete(ctx context.Context, id string) error
}

// WorkingMemoryAccess provides direct access to working memory
type WorkingMemoryAccess interface {
	GetWorkingMemory() *WorkingMemory
}

// MemoryService combines all memory operations
// This is what most consumers will use
type MemoryService interface {
	MemoryReader
	MemoryWriter
	WorkingMemoryAccess
}

// WorkingMemoryInterface defines the interface for working memory
type WorkingMemoryInterface interface {
	Store(entry Entry) error
	Get(id string) *Entry
	Update(entry Entry) error
	Delete(id string) bool
	Search(criteria SearchCriteria) []Entry
	Clear()
	Size() int
}

// HTTPMemoryService defines methods for HTTP memory operations
type HTTPMemoryServiceInterface interface {
	GetMemoryNodes(ctx context.Context, userID string, filters MemoryFilters) ([]MemoryNode, error)
	GetMemoryGraph(ctx context.Context, userID string, filters MemoryFilters) (*MemoryGraph, error)
	UpdateMemoryNode(ctx context.Context, userID string, nodeID string, updates MemoryNodeUpdate) (*MemoryNode, error)
}
