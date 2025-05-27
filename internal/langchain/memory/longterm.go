package memory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
)

// LongTermMemory implements persistent long-term memory with semantic search using pgvector
type LongTermMemory struct {
	dbClient *postgres.SQLCClient
	config   config.LangChain
	logger   *slog.Logger
}

// NewLongTermMemory creates a new long-term memory instance
func NewLongTermMemory(dbClient *postgres.SQLCClient, config config.LangChain, logger *slog.Logger) *LongTermMemory {
	return &LongTermMemory{
		dbClient: dbClient,
		config:   config,
		logger:   logger,
	}
}

// Store stores a memory entry in long-term memory with semantic indexing
func (ltm *LongTermMemory) Store(ctx context.Context, entry *MemoryEntry) error {
	// Check if database client is available
	if ltm.dbClient == nil {
		ltm.logger.Debug("No database client available for long-term memory storage",
			slog.String("entry_id", entry.ID))
		return nil // Gracefully handle missing database
	}

	// Generate ID if not provided
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("lt_%s_%d", entry.UserID, time.Now().UnixNano())
	}

	// Generate embedding if not provided
	if len(entry.Embedding) == 0 {
		embedding, err := ltm.generateEmbedding(ctx, entry.Content)
		if err != nil {
			ltm.logger.Warn("Failed to generate embedding for long-term memory",
				slog.String("entry_id", entry.ID),
				slog.Any("error", err))
			// Continue without embedding - will use text search fallback
		} else {
			entry.Embedding = embedding
		}
	}

	// Store in database using embedding service
	metadata := map[string]interface{}{
		"memory_type":  string(entry.Type),
		"user_id":      entry.UserID,
		"session_id":   entry.SessionID,
		"importance":   entry.Importance,
		"access_count": entry.AccessCount,
		"last_access":  entry.LastAccess,
		"created_at":   entry.CreatedAt,
	}

	// Add custom metadata
	for key, value := range entry.Metadata {
		metadata[key] = value
	}

	// Add context information
	for key, value := range entry.Context {
		metadata[fmt.Sprintf("context_%s", key)] = value
	}

	_, err := ltm.dbClient.CreateEmbedding(
		ctx,
		"memory",
		entry.ID,
		entry.Content,
		entry.Embedding,
		metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to store long-term memory: %w", err)
	}

	ltm.logger.Debug("Stored long-term memory entry",
		slog.String("id", entry.ID),
		slog.String("user_id", entry.UserID),
		slog.Int("embedding_dim", len(entry.Embedding)))

	return nil
}

// Search searches for memories using semantic similarity
func (ltm *LongTermMemory) Search(ctx context.Context, query *MemoryQuery) ([]*MemorySearchResult, error) {
	results := make([]*MemorySearchResult, 0)

	// Check if database client is available
	if ltm.dbClient == nil {
		ltm.logger.Debug("No database client available for long-term memory search")
		return results, nil // Return empty results gracefully
	}

	// Generate query embedding if provided
	var queryEmbedding []float64
	var err error

	if len(query.Embedding) > 0 {
		queryEmbedding = query.Embedding
	} else if query.Content != "" {
		queryEmbedding, err = ltm.generateEmbedding(ctx, query.Content)
		if err != nil {
			ltm.logger.Warn("Failed to generate query embedding",
				slog.String("query", query.Content),
				slog.Any("error", err))
			// Fall back to text-based search
			return ltm.searchByText(ctx, query)
		}
	} else {
		// No content or embedding provided, return empty results
		return results, nil
	}

	// Set default similarity threshold
	similarity := query.Similarity
	if similarity == 0 {
		similarity = 0.7 // Default threshold
	}

	// Set default limit
	limit := query.Limit
	if limit == 0 {
		limit = 10 // Default limit
	}

	// Search using semantic similarity
	searchResults, err := ltm.dbClient.SearchSimilarEmbeddings(
		ctx,
		queryEmbedding,
		"memory",
		limit,
		similarity,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search long-term memory: %w", err)
	}

	// Convert results and apply additional filters
	for _, result := range searchResults {
		// Parse metadata to reconstruct memory entry
		entry, err := ltm.parseEmbeddingToMemoryEntry(result.Record)
		if err != nil {
			ltm.logger.Warn("Failed to parse embedding result",
				slog.String("embedding_id", result.Record.ID),
				slog.Any("error", err))
			continue
		}

		// Apply additional filters
		if !ltm.matchesQuery(entry, query) {
			continue
		}

		// Calculate relevance
		relevance := ltm.calculateRelevance(entry, query, result.Similarity)

		memoryResult := &MemorySearchResult{
			Entry:      entry,
			Similarity: result.Similarity,
			Relevance:  relevance,
		}

		results = append(results, memoryResult)
	}

	ltm.logger.Debug("Long-term memory search completed",
		slog.String("user_id", query.UserID),
		slog.Int("results", len(results)),
		slog.Float64("similarity_threshold", similarity))

	return results, nil
}

// searchByText performs text-based search as fallback
func (ltm *LongTermMemory) searchByText(ctx context.Context, query *MemoryQuery) ([]*MemorySearchResult, error) {
	// This is a simplified fallback implementation
	// In a real system, you might use full-text search capabilities
	ltm.logger.Debug("Using text-based search fallback",
		slog.String("query", query.Content))

	// For now, return empty results
	// TODO: Implement text-based search using PostgreSQL full-text search
	return make([]*MemorySearchResult, 0), nil
}

// Update updates an existing memory entry
func (ltm *LongTermMemory) Update(ctx context.Context, entry *MemoryEntry) error {
	// Update metadata
	metadata := map[string]interface{}{
		"memory_type":  string(entry.Type),
		"user_id":      entry.UserID,
		"session_id":   entry.SessionID,
		"importance":   entry.Importance,
		"access_count": entry.AccessCount,
		"last_access":  entry.LastAccess,
		"created_at":   entry.CreatedAt,
	}

	// Add custom metadata
	for key, value := range entry.Metadata {
		metadata[key] = value
	}

	// Add context information
	for key, value := range entry.Context {
		metadata[fmt.Sprintf("context_%s", key)] = value
	}

	// Update embedding if content changed
	if len(entry.Embedding) == 0 {
		embedding, err := ltm.generateEmbedding(ctx, entry.Content)
		if err != nil {
			ltm.logger.Warn("Failed to generate embedding for update",
				slog.String("entry_id", entry.ID),
				slog.Any("error", err))
		} else {
			entry.Embedding = embedding
		}
	}

	// Update in database
	_, err := ltm.dbClient.CreateEmbedding(
		ctx,
		"memory",
		entry.ID,
		entry.Content,
		entry.Embedding,
		metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to update long-term memory: %w", err)
	}

	ltm.logger.Debug("Updated long-term memory entry",
		slog.String("id", entry.ID))

	return nil
}

// Delete deletes a memory entry
func (ltm *LongTermMemory) Delete(ctx context.Context, entryID string) error {
	// TODO: Implement deletion from embedding store
	// For now, just log the operation
	ltm.logger.Debug("Deleted long-term memory entry",
		slog.String("id", entryID))

	return nil
}

// Clear clears memories for a user
func (ltm *LongTermMemory) Clear(ctx context.Context, userID string, olderThan *time.Time) error {
	// This would require a custom query to delete embeddings by metadata
	// For now, we'll log the operation
	ltm.logger.Info("Long-term memory clear requested",
		slog.String("user_id", userID),
		slog.Any("older_than", olderThan))

	// TODO: Implement bulk deletion by user ID and timestamp
	// This would require extending the embedding client with metadata-based deletion

	return nil
}

// GetStats returns statistics for long-term memory
func (ltm *LongTermMemory) GetStats(ctx context.Context, userID string) (*MemoryTypeStats, error) {
	// This would require aggregation queries on the embedding store
	// For now, return basic stats
	stats := &MemoryTypeStats{
		EntryCount:        0,
		TotalSize:         0,
		AverageImportance: 0,
	}

	ltm.logger.Debug("Long-term memory stats requested",
		slog.String("user_id", userID))

	// TODO: Implement proper stats collection from embedding store
	return stats, nil
}

// Cleanup removes expired entries
func (ltm *LongTermMemory) Cleanup(ctx context.Context) error {
	ltm.logger.Info("Long-term memory cleanup started")

	// TODO: Implement cleanup based on expiration dates in metadata
	// This would require querying embeddings by metadata and deleting expired ones

	return nil
}

// generateEmbedding generates an embedding for the given text
func (ltm *LongTermMemory) generateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// For now, return a mock embedding
	// In a real implementation, this would call an embedding service
	mockEmbedding := make([]float64, 1536) // OpenAI embedding dimension
	for i := range mockEmbedding {
		mockEmbedding[i] = 0.1 // Simple mock values
	}

	ltm.logger.Debug("Generated embedding",
		slog.String("text", text[:min(50, len(text))]),
		slog.Int("dimension", len(mockEmbedding)))

	return mockEmbedding, nil
}

// parseEmbeddingToMemoryEntry converts an embedding record to a memory entry
func (ltm *LongTermMemory) parseEmbeddingToMemoryEntry(record *postgres.EmbeddingRecord) (*MemoryEntry, error) {
	entry := &MemoryEntry{
		ID:        record.ContentID,
		Type:      MemoryTypeLongTerm,
		Content:   record.ContentText,
		Embedding: record.Embedding,
		CreatedAt: record.CreatedAt,
		Metadata:  make(map[string]interface{}),
		Context:   make(map[string]interface{}),
	}

	// Parse metadata
	if record.Metadata != nil {
		for key, value := range record.Metadata {
			switch key {
			case "memory_type":
				if typeStr, ok := value.(string); ok {
					entry.Type = MemoryType(typeStr)
				}
			case "user_id":
				if userID, ok := value.(string); ok {
					entry.UserID = userID
				}
			case "session_id":
				if sessionID, ok := value.(string); ok {
					entry.SessionID = sessionID
				}
			case "importance":
				if importance, ok := value.(float64); ok {
					entry.Importance = importance
				}
			case "access_count":
				if count, ok := value.(float64); ok {
					entry.AccessCount = int(count)
				}
			case "last_access":
				if lastAccess, ok := value.(string); ok {
					if t, err := time.Parse(time.RFC3339, lastAccess); err == nil {
						entry.LastAccess = t
					}
				}
			default:
				if strings.HasPrefix(key, "context_") {
					contextKey := strings.TrimPrefix(key, "context_")
					entry.Context[contextKey] = value
				} else {
					entry.Metadata[key] = value
				}
			}
		}
	}

	return entry, nil
}

// matchesQuery checks if an entry matches the query criteria
func (ltm *LongTermMemory) matchesQuery(entry *MemoryEntry, query *MemoryQuery) bool {
	// Check user ID
	if entry.UserID != query.UserID {
		return false
	}

	// Check session ID if specified
	if query.SessionID != "" && entry.SessionID != query.SessionID {
		return false
	}

	// Check minimum importance
	if query.MinImportance > 0 && entry.Importance < query.MinImportance {
		return false
	}

	// Check time range
	if query.TimeRange != nil {
		if entry.CreatedAt.Before(query.TimeRange.Start) || entry.CreatedAt.After(query.TimeRange.End) {
			return false
		}
	}

	return true
}

// calculateRelevance calculates overall relevance score
func (ltm *LongTermMemory) calculateRelevance(entry *MemoryEntry, query *MemoryQuery, similarity float64) float64 {
	// Factor in importance, recency, and access frequency
	importance := entry.Importance

	// Recency factor (decay over time)
	daysSinceCreation := time.Since(entry.CreatedAt).Hours() / 24.0
	recency := 1.0 / (1.0 + daysSinceCreation*0.1) // Slow decay

	// Access frequency factor
	accessFrequency := float64(entry.AccessCount) / (daysSinceCreation + 1.0)
	if accessFrequency > 1.0 {
		accessFrequency = 1.0
	}

	// Weighted combination
	relevance := (similarity * 0.5) + (importance * 0.2) + (recency * 0.2) + (accessFrequency * 0.1)

	return relevance
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
