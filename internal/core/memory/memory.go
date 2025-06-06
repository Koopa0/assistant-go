package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
)

// Memory provides a simplified memory implementation using direct SQL
// This follows the same pattern as the conversation manager for consistency
type Memory struct {
	db      postgres.DB    // Database interface for persistence
	logger  *slog.Logger   // Structured logger for debugging
	working *WorkingMemory // In-memory for speed
	mu      sync.RWMutex
}

// New creates a new simplified memory instance
func New(db postgres.DB, logger *slog.Logger) *Memory {
	return &Memory{
		db:      db,
		logger:  logger,
		working: NewWorkingMemory(100), // Keep last 100 working items
	}
}

// Store saves an entry to memory
func (m *Memory) Store(ctx context.Context, entry Entry) error {
	// Set defaults
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	entry.UpdatedAt = time.Now()

	m.logger.Debug("Storing memory entry",
		slog.String("id", entry.ID),
		slog.String("type", string(entry.Type)),
		slog.String("user_id", entry.UserID))

	// Working memory goes to in-memory store
	if entry.Type == TypeWorking {
		return m.working.Store(entry)
	}

	// Other types go to database using simple SQL
	return m.storeInDB(ctx, entry)
}

// Retrieve gets an entry by ID
func (m *Memory) Retrieve(ctx context.Context, id string) (*Entry, error) {
	// Try working memory first
	if entry := m.working.Get(id); entry != nil {
		return entry, nil
	}

	// Then try database
	return m.retrieveFromDB(ctx, id)
}

// Search finds entries matching criteria
func (m *Memory) Search(ctx context.Context, criteria SearchCriteria) ([]Entry, error) {
	// Validate criteria
	if criteria.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	// Set defaults
	if criteria.Limit <= 0 {
		criteria.Limit = 10
	}

	m.logger.Debug("Searching memory",
		slog.String("user_id", criteria.UserID),
		slog.Any("types", criteria.Types))

	var results []Entry

	// Search working memory if included
	if m.shouldSearchType(TypeWorking, criteria.Types) {
		workingResults := m.working.Search(criteria)
		results = append(results, workingResults...)
	}

	// Search database for other types
	if m.shouldSearchOtherTypes(criteria.Types) {
		dbResults, err := m.searchInDB(ctx, criteria)
		if err != nil {
			return nil, fmt.Errorf("database search failed: %w", err)
		}
		results = append(results, dbResults...)
	}

	// Sort by importance and limit
	results = m.sortAndLimit(results, criteria.Limit)

	return results, nil
}

// Delete removes an entry by ID
func (m *Memory) Delete(ctx context.Context, id string) error {
	m.logger.Debug("Deleting memory entry", slog.String("id", id))

	// Try working memory first
	if m.working.Delete(id) {
		return nil
	}

	// Then try database
	return m.deleteFromDB(ctx, id)
}

// GetWorkingMemory returns the working memory instance for direct access
func (m *Memory) GetWorkingMemory() *WorkingMemory {
	return m.working
}

// Helper methods

func (m *Memory) shouldSearchType(t Type, types []Type) bool {
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

func (m *Memory) shouldSearchOtherTypes(types []Type) bool {
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

func (m *Memory) sortAndLimit(entries []Entry, limit int) []Entry {
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

// Database operations - Simple SQL implementation like conversation manager

func (m *Memory) storeInDB(ctx context.Context, entry Entry) error {
	// Convert metadata to JSON
	metadataJSON := "{}"
	if entry.Context != nil {
		if data, err := json.Marshal(entry.Context); err == nil {
			metadataJSON = string(data)
		}
	}

	// Simple SQL insert
	query := `
		INSERT INTO memory_entries (id, memory_type, user_id, content, importance, access_count, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			importance = EXCLUDED.importance,
			access_count = EXCLUDED.access_count,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`

	_, err := m.db.Exec(ctx, query,
		entry.ID,
		string(entry.Type),
		entry.UserID,
		entry.Content,
		entry.Importance,
		entry.AccessCount,
		metadataJSON,
		entry.CreatedAt,
		entry.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to store memory entry: %w", err)
	}

	return nil
}

func (m *Memory) retrieveFromDB(ctx context.Context, id string) (*Entry, error) {
	query := `
		SELECT id, memory_type, user_id, content, importance, access_count, metadata, created_at, updated_at
		FROM memory_entries
		WHERE id = $1
	`

	var entry Entry
	var memoryType string
	var metadataJSON string

	err := m.db.QueryRow(ctx, query, id).Scan(
		&entry.ID,
		&memoryType,
		&entry.UserID,
		&entry.Content,
		&entry.Importance,
		&entry.AccessCount,
		&metadataJSON,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve memory entry: %w", err)
	}

	entry.Type = Type(memoryType)

	// Parse metadata
	if metadataJSON != "{}" && metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &entry.Context); err != nil {
			m.logger.Warn("Failed to unmarshal memory metadata",
				slog.String("id", entry.ID),
				slog.Any("error", err))
		}
	}
	if entry.Context == nil {
		entry.Context = make(map[string]interface{})
	}

	return &entry, nil
}

func (m *Memory) searchInDB(ctx context.Context, criteria SearchCriteria) ([]Entry, error) {
	// Build dynamic query based on criteria
	query := `
		SELECT id, memory_type, user_id, content, importance, access_count, metadata, created_at, updated_at
		FROM memory_entries
		WHERE user_id = $1
	`
	args := []interface{}{criteria.UserID}
	argIndex := 2

	// Add type filter if specified
	if len(criteria.Types) > 0 {
		typeValues := make([]string, len(criteria.Types))
		for i, t := range criteria.Types {
			typeValues[i] = string(t)
		}
		query += fmt.Sprintf(" AND memory_type = ANY($%d)", argIndex)
		args = append(args, typeValues)
		argIndex++
	}

	// Add importance filter if specified
	if criteria.ImportanceMin > 0 {
		query += fmt.Sprintf(" AND importance >= $%d", argIndex)
		args = append(args, criteria.ImportanceMin)
		argIndex++
	}

	// Add text search if specified
	if criteria.Query != "" {
		query += fmt.Sprintf(" AND content ILIKE $%d", argIndex)
		args = append(args, "%"+criteria.Query+"%")
		argIndex++
	}

	// Add time range filters
	if criteria.TimeFrom != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *criteria.TimeFrom)
		argIndex++
	}

	if criteria.TimeTo != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *criteria.TimeTo)
		argIndex++
	}

	// Add ordering and limiting
	query += " ORDER BY importance DESC, created_at DESC"
	if criteria.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, criteria.Limit)
		argIndex++
	}
	if criteria.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, criteria.Offset)
	}

	rows, err := m.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var entry Entry
		var memoryType string
		var metadataJSON string

		err := rows.Scan(
			&entry.ID,
			&memoryType,
			&entry.UserID,
			&entry.Content,
			&entry.Importance,
			&entry.AccessCount,
			&metadataJSON,
			&entry.CreatedAt,
			&entry.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan memory entry: %w", err)
		}

		entry.Type = Type(memoryType)

		// Parse metadata
		if metadataJSON != "{}" && metadataJSON != "" {
			if err := json.Unmarshal([]byte(metadataJSON), &entry.Context); err != nil {
				m.logger.Warn("Failed to unmarshal memory metadata",
					slog.String("id", entry.ID),
					slog.Any("error", err))
			}
		}
		if entry.Context == nil {
			entry.Context = make(map[string]interface{})
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return entries, nil
}

func (m *Memory) deleteFromDB(ctx context.Context, id string) error {
	query := "DELETE FROM memory_entries WHERE id = $1"
	result, err := m.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete memory entry: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("memory entry not found: %s", id)
	}

	return nil
}

// AddRelation creates a relationship between two memory entries (stub implementation)
// TODO: Implement full relation support when needed
func (m *Memory) AddRelation(ctx context.Context, relation Relation) error {
	m.logger.Debug("AddRelation called - not fully implemented yet",
		slog.String("from_id", relation.FromID),
		slog.String("to_id", relation.ToID),
		slog.String("type", string(relation.Type)))
	return fmt.Errorf("relations not implemented yet in simplified memory")
}
