package memory

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Convenience wrappers for common memory operations

// StoreWorking stores a working memory entry
func (m *Memory) StoreWorking(ctx context.Context, userID, content string) error {
	return m.Store(ctx, Entry{
		Type:       TypeWorking,
		UserID:     userID,
		Content:    content,
		Importance: 0.5, // Default medium importance
	})
}

// StoreEpisodic stores an episodic memory entry with context
func (m *Memory) StoreEpisodic(ctx context.Context, userID, content string, context map[string]interface{}) error {
	return m.Store(ctx, Entry{
		Type:       TypeEpisodic,
		UserID:     userID,
		Content:    content,
		Context:    context,
		Importance: 0.7, // Episodes are generally important
	})
}

// StoreSemantic stores a fact-based memory entry
func (m *Memory) StoreSemantic(ctx context.Context, userID, content string, importance float64) error {
	return m.Store(ctx, Entry{
		Type:       TypeSemantic,
		UserID:     userID,
		Content:    content,
		Importance: importance,
	})
}

// StoreProcedural stores a how-to memory entry
func (m *Memory) StoreProcedural(ctx context.Context, userID, procedure string, steps []string) error {
	context := map[string]interface{}{
		"procedure": procedure,
		"steps":     steps,
	}

	return m.Store(ctx, Entry{
		Type:       TypeProcedural,
		UserID:     userID,
		Content:    fmt.Sprintf("Procedure: %s", procedure),
		Context:    context,
		Importance: 0.8, // Procedures are very important
	})
}

// SearchByType searches for entries of a specific type
func (m *Memory) SearchByType(ctx context.Context, userID string, memType Type, limit int) ([]Entry, error) {
	return m.Search(ctx, SearchCriteria{
		UserID: userID,
		Types:  []Type{memType},
		Limit:  limit,
	})
}

// SearchRecent searches for recent entries
func (m *Memory) SearchRecent(ctx context.Context, userID string, duration time.Duration) ([]Entry, error) {
	from := time.Now().Add(-duration)
	return m.Search(ctx, SearchCriteria{
		UserID:   userID,
		TimeFrom: &from,
	})
}

// SearchImportant searches for important entries
func (m *Memory) SearchImportant(ctx context.Context, userID string, minImportance float64) ([]Entry, error) {
	return m.Search(ctx, SearchCriteria{
		UserID:        userID,
		ImportanceMin: minImportance,
	})
}

// SearchByQuery performs a text search
func (m *Memory) SearchByQuery(ctx context.Context, userID, query string) ([]Entry, error) {
	return m.Search(ctx, SearchCriteria{
		UserID: userID,
		Query:  query,
	})
}

// RelateSequence creates a sequence relationship between entries
func (m *Memory) RelateSequence(ctx context.Context, fromID, toID string) error {
	return m.AddRelation(ctx, Relation{
		FromID: fromID,
		ToID:   toID,
		Type:   RelationTypeSequence,
		Weight: 1.0,
	})
}

// RelateCause creates a causal relationship between entries
func (m *Memory) RelateCause(ctx context.Context, causeID, effectID string) error {
	return m.AddRelation(ctx, Relation{
		FromID: causeID,
		ToID:   effectID,
		Type:   RelationTypeCause,
		Weight: 1.0,
	})
}

// RelateSimilar creates a similarity relationship between entries
func (m *Memory) RelateSimilar(ctx context.Context, id1, id2 string, similarity float64) error {
	return m.AddRelation(ctx, Relation{
		FromID: id1,
		ToID:   id2,
		Type:   RelationTypeSimilar,
		Weight: similarity,
	})
}

// GetCauses finds entries that cause a specific entry (stub implementation)
func (m *Memory) GetCauses(ctx context.Context, effectID string) ([]Entry, error) {
	m.logger.Debug("GetCauses called - relations not implemented yet", slog.String("effect_id", effectID))
	return []Entry{}, fmt.Errorf("relations not implemented yet in simplified memory")
}

// GetEffects finds entries that are caused by a specific entry (stub implementation)
func (m *Memory) GetEffects(ctx context.Context, causeID string) ([]Entry, error) {
	m.logger.Debug("GetEffects called - relations not implemented yet", slog.String("cause_id", causeID))
	return []Entry{}, fmt.Errorf("relations not implemented yet in simplified memory")
}

// ClearWorking clears all working memory for a user
func (m *Memory) ClearWorking(userID string) {
	// Get all working memory entries for the user
	entries := m.working.Search(SearchCriteria{
		UserID: userID,
		Types:  []Type{TypeWorking},
	})

	// Delete each one
	for _, entry := range entries {
		m.working.Delete(entry.ID)
	}
}

// ExpireOldEntries removes expired entries (maintenance operation)
func (m *Memory) ExpireOldEntries(ctx context.Context) error {
	// Direct database operations are not exposed by SQLCClient
	// TODO: Create SQLC queries for memory operations
	return fmt.Errorf("expire old entries not yet implemented - need SQLC queries")
}
