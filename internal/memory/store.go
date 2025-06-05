package memory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	coremem "github.com/koopa0/assistant-go/internal/core/memory"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
)

// Store implements the core memory interfaces
// Concrete implementation using PostgreSQL and in-memory storage
type Store struct {
	db      *postgres.SQLCClient
	logger  *slog.Logger
	working *WorkingMemory // In-memory for speed
	mu      sync.RWMutex
}

// Ensure Store implements core memory interfaces
var (
	_ coremem.Memory        = (*Store)(nil)
	_ coremem.MemoryManager = (*Store)(nil)
)

// NewStore creates a new memory store
func NewStore(db *postgres.SQLCClient, logger *slog.Logger) *Store {
	return &Store{
		db:      db,
		logger:  logger,
		working: NewWorkingMemory(100), // Keep last 100 working items
	}
}

// Store saves an entry to memory
func (s *Store) Store(ctx context.Context, entry coremem.Entry) error {
	// Set defaults
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	entry.UpdatedAt = time.Now()

	s.logger.Debug("Storing memory entry",
		slog.String("id", entry.ID),
		slog.String("type", string(entry.Type)),
		slog.String("user_id", entry.UserID))

	// Working memory goes to in-memory store
	if entry.Type == coremem.TypeWorking {
		return s.working.Store(entry)
	}

	// Other types go to database
	return s.storeInDB(ctx, entry)
}

// Retrieve gets an entry by ID
func (s *Store) Retrieve(ctx context.Context, id string) (*coremem.Entry, error) {
	// Try working memory first
	if entry := s.working.Get(id); entry != nil {
		return entry, nil
	}

	// Then try database
	return s.retrieveFromDB(ctx, id)
}

// Search finds entries matching criteria
func (s *Store) Search(ctx context.Context, criteria coremem.SearchCriteria) ([]coremem.Entry, error) {
	// Validate criteria
	if criteria.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	// Set defaults
	if criteria.Limit <= 0 {
		criteria.Limit = 10
	}

	s.logger.Debug("Searching memory",
		slog.String("user_id", criteria.UserID),
		slog.Any("types", criteria.Types))

	// Search working memory if included
	var results []coremem.Entry
	if s.shouldSearchType(coremem.TypeWorking, criteria.Types) {
		workingResults := s.working.Search(criteria)
		results = append(results, workingResults...)
	}

	// Search database for other types
	if s.shouldSearchOtherTypes(criteria.Types) {
		dbResults, err := s.searchInDB(ctx, criteria)
		if err != nil {
			return nil, fmt.Errorf("database search failed: %w", err)
		}
		results = append(results, dbResults...)
	}

	// Sort by importance and limit
	results = s.sortAndLimit(results, criteria.Limit)

	return results, nil
}

// Delete removes an entry by ID
func (s *Store) Delete(ctx context.Context, id string) error {
	s.logger.Debug("Deleting memory entry", slog.String("id", id))

	// Try working memory first
	if s.working.Delete(id) {
		return nil
	}

	// Then try database
	return s.deleteFromDB(ctx, id)
}

// MemoryManager interface implementations

// GetWorkingMemory returns the working memory instance
func (s *Store) GetWorkingMemory() coremem.Memory {
	return &workingMemoryWrapper{working: s.working}
}

// GetEpisodicMemory returns episodic memory implementation
func (s *Store) GetEpisodicMemory() coremem.Memory {
	return &dbMemoryWrapper{store: s, memType: coremem.TypeEpisodic}
}

// GetSemanticMemory returns semantic memory implementation
func (s *Store) GetSemanticMemory() coremem.Memory {
	return &dbMemoryWrapper{store: s, memType: coremem.TypeSemantic}
}

// GetProceduralMemory returns procedural memory implementation
func (s *Store) GetProceduralMemory() coremem.Memory {
	return &dbMemoryWrapper{store: s, memType: coremem.TypeProcedural}
}

// GetGraph returns memory graph implementation
func (s *Store) GetGraph() coremem.MemoryGraph {
	return NewGraphStore(s)
}

// Consolidate processes memory consolidation
func (s *Store) Consolidate(ctx context.Context, opts coremem.ConsolidationOptions) error {
	// TODO: Implement memory consolidation logic
	return fmt.Errorf("consolidation not yet implemented")
}

// GetStats returns memory statistics
func (s *Store) GetStats(ctx context.Context) (*coremem.MemoryStats, error) {
	// TODO: Implement statistics collection
	return &coremem.MemoryStats{
		TotalEntries:     0,
		EntriesByType:    make(map[coremem.Type]int),
		TotalRelations:   0,
		RelationsByType:  make(map[coremem.RelationType]int),
		StorageUsed:      0,
		LastConsolidated: time.Now(),
		Performance: coremem.PerformanceStats{
			AverageStoreTime:    time.Millisecond,
			AverageRetrieveTime: time.Millisecond,
			AverageSearchTime:   time.Millisecond * 10,
			CacheHitRate:        0.8,
			ConsolidationRate:   0.1,
		},
	}, nil
}

// Helper methods

func (s *Store) shouldSearchType(t coremem.Type, types []coremem.Type) bool {
	if len(types) == 0 {
		return true // Search all types
	}
	for _, typ := range types {
		if typ == t {
			return true
		}
	}
	return false
}

func (s *Store) shouldSearchOtherTypes(types []coremem.Type) bool {
	if len(types) == 0 {
		return true // Search all types
	}
	for _, t := range types {
		if t != coremem.TypeWorking {
			return true
		}
	}
	return false
}

func (s *Store) sortAndLimit(entries []coremem.Entry, limit int) []coremem.Entry {
	// Simple bubble sort by importance (good enough for small result sets)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].Importance < entries[j].Importance {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Apply limit
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	return entries
}

// Database operations (to be implemented)

func (s *Store) storeInDB(ctx context.Context, entry coremem.Entry) error {
	// TODO: Implement database storage using s.db and SQLC
	return fmt.Errorf("database storage not yet implemented")
}

func (s *Store) retrieveFromDB(ctx context.Context, id string) (*coremem.Entry, error) {
	// TODO: Implement database retrieval
	return nil, fmt.Errorf("database retrieval not yet implemented")
}

func (s *Store) searchInDB(ctx context.Context, criteria coremem.SearchCriteria) ([]coremem.Entry, error) {
	// TODO: Implement database search using s.db and SQLC
	return nil, fmt.Errorf("database search not yet implemented")
}

func (s *Store) deleteFromDB(ctx context.Context, id string) error {
	// TODO: Implement database deletion
	return fmt.Errorf("database deletion not yet implemented")
}
