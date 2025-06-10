// Package memory implements a multi-layered memory system for the assistant.
// It provides working memory for active context, episodic memory for conversation history,
// semantic memory for knowledge storage, and procedural memory for learned patterns and procedures.
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// Service provides memory operations combining business logic and data access
// Following Go principle: simple and direct, no unnecessary abstractions
type Service struct {
	queries sqlc.Querier
	logger  *slog.Logger
	working *WorkingMemory
	mu      sync.RWMutex
}

// NewService creates a new memory service
func NewService(queries sqlc.Querier, logger *slog.Logger) *Service {
	return &Service{
		queries: queries,
		logger:  logger,
		working: NewWorkingMemory(100),
	}
}

// Store saves an entry to memory
func (s *Service) Store(ctx context.Context, entry Entry) error {
	// Validate required fields
	if entry.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if entry.Content == "" {
		return fmt.Errorf("content is required")
	}

	// Set defaults
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Importance == 0 {
		entry.Importance = ImportanceMedium
	}

	now := time.Now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	if entry.UpdatedAt.IsZero() {
		entry.UpdatedAt = now
	}
	if entry.LastAccess.IsZero() {
		entry.LastAccess = now
	}

	// Map legacy types to database types
	entry.Type = mapLegacyType(entry.Type)

	s.logger.Debug("Storing memory entry",
		slog.String("id", entry.ID),
		slog.String("type", string(entry.Type)),
		slog.String("user_id", entry.UserID))

	// Working memory goes to in-memory store only
	if entry.Type == TypeWorking {
		return s.working.Store(entry)
	}

	// Other types go to database
	_, err := s.createInDB(ctx, &entry)
	return err
}

// Retrieve gets an entry by ID
func (s *Service) Retrieve(ctx context.Context, id string) (*Entry, error) {
	if id == "" {
		return nil, fmt.Errorf("memory ID is required")
	}

	// Try working memory first
	if entry := s.working.Get(id); entry != nil {
		return entry, nil
	}

	// Then try database
	return s.getFromDB(ctx, id)
}

// Search finds entries matching criteria
func (s *Service) Search(ctx context.Context, criteria SearchCriteria) ([]Entry, error) {
	// Validate criteria
	if criteria.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	// Set defaults
	if criteria.Limit <= 0 {
		criteria.Limit = DefaultSearchLimit
	}

	// Map legacy types in criteria
	for i, t := range criteria.Types {
		criteria.Types[i] = mapLegacyType(t)
	}

	s.logger.Debug("Searching memory",
		slog.String("user_id", criteria.UserID),
		slog.Any("types", criteria.Types))

	var results []Entry

	// Search working memory if included
	if shouldSearchType(TypeWorking, criteria.Types) {
		workingResults := s.working.Search(criteria)
		results = append(results, workingResults...)
	}

	// Search database for other types
	if shouldSearchOtherTypes(criteria.Types) {
		dbEntries, err := s.searchInDB(ctx, criteria)
		if err != nil {
			return nil, fmt.Errorf("database search failed: %w", err)
		}
		results = append(results, dbEntries...)
	}

	// Sort by importance and limit
	return sortAndLimit(results, criteria.Limit), nil
}

// Update updates an existing memory entry
func (s *Service) Update(ctx context.Context, entry Entry) error {
	if entry.ID == "" {
		return fmt.Errorf("memory ID is required")
	}

	// Working memory updates
	if entry.Type == TypeWorking {
		return s.working.Update(entry)
	}

	// Database updates
	_, err := s.updateInDB(ctx, &entry)
	return err
}

// Delete removes an entry by ID
func (s *Service) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("memory ID is required")
	}

	s.logger.Debug("Deleting memory entry", slog.String("id", id))

	// Try working memory first
	if s.working.Delete(id) {
		return nil
	}

	// Then try database
	return s.deleteFromDB(ctx, id)
}

// GetWorkingMemory returns the working memory instance for direct access
func (s *Service) GetWorkingMemory() *WorkingMemory {
	return s.working
}

// Database operations - direct sqlc usage

func (s *Service) createInDB(ctx context.Context, entry *Entry) (*Entry, error) {
	// Parse user UUID
	userUUID, err := uuid.Parse(entry.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Convert metadata to JSON
	metadataJSON, err := marshalMetadata(entry)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	// Prepare parameters
	params := sqlc.CreateMemoryEntryParams{
		MemoryType:  string(entry.Type),
		UserID:      pgtype.UUID{Bytes: userUUID, Valid: true},
		Content:     entry.Content,
		Importance:  entry.Importance,
		AccessCount: int32(entry.AccessCount),
		LastAccess:  pgtype.Timestamptz{Time: entry.LastAccess, Valid: true},
		Metadata:    metadataJSON,
	}

	// Handle optional fields
	if entry.SessionID != nil {
		params.SessionID = *entry.SessionID
	}
	if entry.ExpiresAt != nil {
		params.ExpiresAt = pgtype.Timestamptz{Time: *entry.ExpiresAt, Valid: true}
	}

	// Execute query
	row, err := s.queries.CreateMemoryEntry(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create memory entry: %w", err)
	}

	// Convert back to domain type
	return convertCreateMemoryEntryRow(row, s.logger), nil
}

func (s *Service) getFromDB(ctx context.Context, id string) (*Entry, error) {
	// Parse UUID
	entryUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid memory ID: %w", err)
	}

	// Execute query
	row, err := s.queries.GetMemoryEntry(ctx, pgtype.UUID{Bytes: entryUUID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, MemoryNotFoundError{MemoryID: id}
		}
		return nil, fmt.Errorf("get memory entry: %w", err)
	}

	// Increment access count asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.queries.IncrementMemoryAccess(ctx, pgtype.UUID{Bytes: entryUUID, Valid: true}); err != nil {
			s.logger.Error("Failed to increment memory access count",
				slog.String("memory_id", id),
				slog.Any("error", err))
		}
	}()

	return convertGetMemoryEntryRow(row, s.logger), nil
}

func (s *Service) updateInDB(ctx context.Context, entry *Entry) (*Entry, error) {
	// Parse UUID
	entryUUID, err := uuid.Parse(entry.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid memory ID: %w", err)
	}

	// Update timestamps
	entry.UpdatedAt = time.Now()
	entry.LastAccess = time.Now()

	// Convert metadata to JSON
	metadataJSON, err := marshalMetadata(entry)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	// Prepare parameters
	params := sqlc.UpdateMemoryEntryParams{
		ID:          pgtype.UUID{Bytes: entryUUID, Valid: true},
		Content:     entry.Content,
		Importance:  entry.Importance,
		AccessCount: int32(entry.AccessCount),
		LastAccess:  pgtype.Timestamptz{Time: entry.LastAccess, Valid: true},
		Metadata:    metadataJSON,
	}

	// Handle optional fields
	if entry.ExpiresAt != nil {
		params.ExpiresAt = pgtype.Timestamptz{Time: *entry.ExpiresAt, Valid: true}
	}

	// For optimistic locking support (set to 0 and invalid time to skip)
	params.ExpectedVersion = 0
	params.ExpectedUpdatedAt = pgtype.Timestamptz{Valid: false}

	// Execute query
	row, err := s.queries.UpdateMemoryEntry(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, MemoryNotFoundError{MemoryID: entry.ID}
		}
		return nil, fmt.Errorf("update memory entry: %w", err)
	}

	return convertUpdateMemoryEntryRow(row, s.logger), nil
}

func (s *Service) deleteFromDB(ctx context.Context, id string) error {
	// Parse UUID
	entryUUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid memory ID: %w", err)
	}

	err = s.queries.DeleteMemoryEntry(ctx, pgtype.UUID{Bytes: entryUUID, Valid: true})
	if err != nil {
		return fmt.Errorf("delete memory entry: %w", err)
	}

	return nil
}

func (s *Service) searchInDB(ctx context.Context, criteria SearchCriteria) ([]Entry, error) {
	// Parse user UUID
	userUUID, err := uuid.Parse(criteria.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Convert types to string array
	var memoryTypes []string
	if len(criteria.Types) > 0 {
		memoryTypes = make([]string, len(criteria.Types))
		for i, t := range criteria.Types {
			memoryTypes[i] = string(t)
		}
	}

	params := sqlc.SearchMemoryEntriesParams{
		UserID:    pgtype.UUID{Bytes: userUUID, Valid: true},
		LimitVal:  int32(criteria.Limit),
		OffsetVal: int32(criteria.Offset),
	}

	// Handle optional parameters
	if criteria.Query != "" {
		params.SearchQuery = criteria.Query
	}
	if len(memoryTypes) > 0 {
		params.MemoryTypes = memoryTypes
	}
	if criteria.ImportanceMin > 0 {
		params.MinImportance = criteria.ImportanceMin
	}

	// Execute query
	rows, err := s.queries.SearchMemoryEntries(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("search memory entries: %w", err)
	}

	// Convert results
	entries := make([]Entry, 0, len(rows))
	for _, row := range rows {
		entry := convertSearchMemoryEntriesRow(row, s.logger)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries, nil
}

// Helper functions

func shouldSearchType(t Type, types []Type) bool {
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

func shouldSearchOtherTypes(types []Type) bool {
	if len(types) == 0 {
		return true // Search all types
	}
	for _, t := range types {
		if t != TypeWorking {
			return true
		}
	}
	return false
}

func sortAndLimit(entries []Entry, limit int) []Entry {
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

func mapLegacyType(t Type) Type {
	// Since TypeEpisodic and TypeSemantic are already defined as TypeLongTerm,
	// and TypeProcedural is already TypeTool, we only need to handle the base types
	switch t {
	case TypeWorking, TypeShortTerm, TypeLongTerm, TypeTool, TypePersonalization:
		return t // Already correct
	default:
		return TypeShortTerm // Default
	}
}

func marshalMetadata(entry *Entry) ([]byte, error) {
	// Prefer Metadata over Context
	if len(entry.Metadata.Category) > 0 || len(entry.Metadata.Tags) > 0 ||
		entry.Metadata.Source != "" || entry.Metadata.Confidence > 0 {
		return json.Marshal(entry.Metadata)
	}

	// Fall back to Context for backward compatibility
	if entry.Context != nil && len(entry.Context) > 0 {
		return json.Marshal(entry.Context)
	}

	// Default to empty JSON object
	return []byte("{}"), nil
}
