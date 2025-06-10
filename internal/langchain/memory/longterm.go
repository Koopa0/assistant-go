package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/langchain/vectorstore"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// LongTermMemory implements persistent long-term memory with semantic search using LangChain vectorstore
type LongTermMemory struct {
	queries     sqlc.Querier
	vectorStore vectorstores.VectorStore
	embedder    embeddings.Embedder
	config      config.LangChain
	logger      *slog.Logger
}

// NewLongTermMemory creates a new long-term memory instance
func NewLongTermMemory(queries sqlc.Querier, config config.LangChain, logger *slog.Logger) *LongTermMemory {
	return &LongTermMemory{
		queries: queries,
		config:  config,
		logger:  logger,
	}
}

// NewLongTermMemoryWithVectorStore creates a new long-term memory instance with custom vectorstore and embedder
func NewLongTermMemoryWithVectorStore(queries sqlc.Querier, vectorStore vectorstores.VectorStore, embedder embeddings.Embedder, config config.LangChain, logger *slog.Logger) *LongTermMemory {
	return &LongTermMemory{
		queries:     queries,
		vectorStore: vectorStore,
		embedder:    embedder,
		config:      config,
		logger:      logger,
	}
}

// SetVectorStore sets the vectorstore for this memory instance
func (ltm *LongTermMemory) SetVectorStore(vectorStore vectorstores.VectorStore) {
	ltm.vectorStore = vectorStore
}

// SetEmbedder sets the embedder for this memory instance
func (ltm *LongTermMemory) SetEmbedder(embedder embeddings.Embedder) {
	ltm.embedder = embedder
}

// Store stores a memory entry in long-term memory with semantic indexing
func (ltm *LongTermMemory) Store(ctx context.Context, entry *MemoryEntry) error {
	// Generate ID if not provided
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("lt_%s_%d", entry.UserID, time.Now().UnixNano())
	}

	// Use LangChain vectorstore if available
	if ltm.vectorStore != nil {
		return ltm.storeWithVectorStore(ctx, entry)
	}

	// Fallback to direct database storage
	return ltm.storeWithDatabase(ctx, entry)
}

// storeWithVectorStore stores memory using LangChain vectorstore
func (ltm *LongTermMemory) storeWithVectorStore(ctx context.Context, entry *MemoryEntry) error {
	ltm.logger.Debug("Storing memory entry with vectorstore",
		slog.String("entry_id", entry.ID),
		slog.String("user_id", entry.UserID))

	// Create schema.Document from memory entry
	doc := schema.Document{
		PageContent: entry.Content,
		Metadata:    make(map[string]any),
	}

	// Add memory metadata
	doc.Metadata["memory_type"] = string(entry.Type)
	doc.Metadata["user_id"] = entry.UserID
	doc.Metadata["session_id"] = entry.SessionID
	doc.Metadata["importance"] = entry.Importance
	doc.Metadata["access_count"] = entry.AccessCount
	doc.Metadata["last_access"] = entry.LastAccess.Format(time.RFC3339)
	doc.Metadata["created_at"] = entry.CreatedAt.Format(time.RFC3339)
	doc.Metadata["entry_id"] = entry.ID

	// Add custom metadata
	for key, value := range entry.Metadata {
		doc.Metadata[key] = value
	}

	// Add context information
	for key, value := range entry.Context {
		doc.Metadata[fmt.Sprintf("context_%s", key)] = value
	}

	// Store in vectorstore
	ids, err := ltm.vectorStore.AddDocuments(ctx, []schema.Document{doc})
	if err != nil {
		return fmt.Errorf("failed to store memory in vectorstore: %w", err)
	}

	ltm.logger.Debug("Stored memory entry in vectorstore",
		slog.String("id", entry.ID),
		slog.String("user_id", entry.UserID),
		slog.Any("vector_ids", ids))

	return nil
}

// storeWithDatabase stores memory using direct database access (fallback)
func (ltm *LongTermMemory) storeWithDatabase(ctx context.Context, entry *MemoryEntry) error {
	// Check if database client is available
	if ltm.queries == nil {
		ltm.logger.Debug("No database client available for long-term memory storage",
			slog.String("entry_id", entry.ID))
		return nil // Gracefully handle missing database
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

	// Convert entry ID to UUID
	entryUUID, err := postgres.ParseUUID(entry.ID)
	if err != nil {
		return fmt.Errorf("failed to parse entry ID: %w", err)
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create embedding with queries
	_, err = ltm.queries.CreateEmbedding(ctx, sqlc.CreateEmbeddingParams{
		ContentType: "memory",
		ContentID:   entryUUID,
		ContentText: entry.Content,
		Embedding:   postgres.VectorToPgVector(entry.Embedding),
		Metadata:    metadataJSON,
	})
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
	// Use LangChain vectorstore if available
	if ltm.vectorStore != nil {
		return ltm.searchWithVectorStore(ctx, query)
	}

	// Fallback to direct database search
	return ltm.searchWithDatabase(ctx, query)
}

// searchWithVectorStore searches memories using LangChain vectorstore
func (ltm *LongTermMemory) searchWithVectorStore(ctx context.Context, query *MemoryQuery) ([]*MemorySearchResult, error) {
	results := make([]*MemorySearchResult, 0)

	if query.Content == "" {
		ltm.logger.Debug("No query content provided for vectorstore search")
		return results, nil
	}

	// Set default limit
	limit := query.Limit
	if limit == 0 {
		limit = 10
	}

	// Prepare search options
	options := []vectorstores.Option{}
	if query.Similarity > 0 {
		options = append(options, vectorstores.WithScoreThreshold(float32(query.Similarity)))
	}

	// Add user filter
	if query.UserID != "" {
		filters := map[string]any{"user_id": query.UserID}
		if query.SessionID != "" {
			filters["session_id"] = query.SessionID
		}
		if query.MinImportance > 0 {
			filters["min_importance"] = query.MinImportance
		}
		options = append(options, vectorstores.WithFilters(filters))
	}

	// Perform similarity search
	// Check if it's our custom PGVectorStore that has SimilaritySearchWithScore
	var docs []schema.Document
	var scores []float32
	var err error

	if pgVectorStore, ok := ltm.vectorStore.(*vectorstore.PGVectorStore); ok {
		docs, scores, err = pgVectorStore.SimilaritySearchWithScore(ctx, query.Content, limit, options...)
	} else {
		// Fallback to regular similarity search
		docs, err = ltm.vectorStore.SimilaritySearch(ctx, query.Content, limit, options...)
		if err == nil {
			// Create dummy scores
			scores = make([]float32, len(docs))
			for i := range scores {
				scores[i] = 0.8 // Default score
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("vectorstore search failed: %w", err)
	}

	// Convert results to MemorySearchResult
	for i, doc := range docs {
		entry, err := ltm.parseDocumentToMemoryEntry(doc)
		if err != nil {
			ltm.logger.Warn("Failed to parse document to memory entry",
				slog.Int("doc_index", i),
				slog.Any("error", err))
			continue
		}

		// Apply additional filters
		if !ltm.matchesQuery(entry, query) {
			continue
		}

		// Calculate relevance
		similarity := float64(scores[i])
		relevance := ltm.calculateRelevance(entry, query, similarity)

		result := &MemorySearchResult{
			Entry:      entry,
			Similarity: similarity,
			Relevance:  relevance,
		}

		results = append(results, result)
	}

	ltm.logger.Debug("Vectorstore memory search completed",
		slog.String("user_id", query.UserID),
		slog.Int("results", len(results)))

	return results, nil
}

// searchWithDatabase searches memories using direct database access (fallback)
func (ltm *LongTermMemory) searchWithDatabase(ctx context.Context, query *MemoryQuery) ([]*MemorySearchResult, error) {
	results := make([]*MemorySearchResult, 0)

	// Check if database client is available
	if ltm.queries == nil {
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

	// Note: The current SearchSimilarEmbeddings doesn't support metadata filtering
	// We'll filter the results after retrieval
	if query.UserID != "" || query.SessionID != "" || query.MinImportance > 0 {
		ltm.logger.Debug("Additional filtering will be applied after retrieval",
			slog.String("user_id", query.UserID),
			slog.String("session_id", query.SessionID),
			slog.Float64("min_importance", query.MinImportance))
	}

	// Search for similar embeddings
	searchResults, err := ltm.queries.SearchSimilarEmbeddings(ctx, sqlc.SearchSimilarEmbeddingsParams{
		QueryEmbedding: postgres.VectorToPgVector(queryEmbedding),
		ContentType:    "memory",
		Threshold:      similarity,
		ResultLimit:    int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search long-term memory: %w", err)
	}

	// Convert results
	for _, searchResult := range searchResults {
		// Convert to standard Embedding type
		emb := &sqlc.Embedding{
			ID:          searchResult.ID,
			ContentType: searchResult.ContentType,
			ContentID:   searchResult.ContentID,
			ContentText: searchResult.ContentText,
			Embedding:   searchResult.Embedding,
			Metadata:    searchResult.Metadata,
			CreatedAt:   searchResult.CreatedAt,
		}

		// Convert to domain embedding record
		embRecord := postgres.ConvertSQLCEmbedding(emb)

		// Parse to memory entry
		entry, err := ltm.parseEmbeddingToMemoryEntry(embRecord)
		if err != nil {
			ltm.logger.Warn("Failed to parse embedding result",
				slog.String("embedding_id", searchResult.ID.String()),
				slog.Any("error", err))
			continue
		}

		// Apply additional filters
		if !ltm.matchesQuery(entry, query) {
			continue
		}

		// Convert similarity from int32 to float64
		similarityFloat := float64(searchResult.Similarity) / 100.0

		// Calculate relevance
		relevance := ltm.calculateRelevance(entry, query, similarityFloat)

		memoryResult := &MemorySearchResult{
			Entry:      entry,
			Similarity: similarityFloat,
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

	// Convert entry ID to UUID
	entryUUID, err := postgres.ParseUUID(entry.ID)
	if err != nil {
		return fmt.Errorf("failed to parse entry ID: %w", err)
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Update in database
	_, err = ltm.queries.CreateEmbedding(ctx, sqlc.CreateEmbeddingParams{
		ContentType: "memory",
		ContentID:   entryUUID,
		ContentText: entry.Content,
		Embedding:   postgres.VectorToPgVector(entry.Embedding),
		Metadata:    metadataJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to update long-term memory: %w", err)
	}

	ltm.logger.Debug("Updated long-term memory entry",
		slog.String("id", entry.ID))

	return nil
}

// Delete deletes a memory entry
func (ltm *LongTermMemory) Delete(ctx context.Context, entryID string) error {
	// Check if database client is available
	if ltm.queries == nil {
		ltm.logger.Debug("No database client available for deletion")
		return nil
	}

	// Convert entry ID to UUID
	entryUUID, err := postgres.ParseUUID(entryID)
	if err != nil {
		return fmt.Errorf("failed to parse entry ID: %w", err)
	}

	// Delete by content type and content ID
	err = ltm.queries.DeleteEmbedding(ctx, sqlc.DeleteEmbeddingParams{
		ContentType: "memory",
		ContentID:   entryUUID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete long-term memory entry: %w", err)
	}

	ltm.logger.Debug("Deleted long-term memory entry",
		slog.String("id", entryID))

	return nil
}

// Clear clears memories for a user
func (ltm *LongTermMemory) Clear(ctx context.Context, userID string, olderThan *time.Time) error {
	// Check if database client is available
	if ltm.queries == nil {
		ltm.logger.Debug("No database client available for clearing memories")
		return nil
	}

	ltm.logger.Info("Long-term memory clear requested",
		slog.String("user_id", userID),
		slog.Any("older_than", olderThan))

	// If olderThan is specified, delete expired embeddings
	if olderThan != nil {
		err := ltm.queries.DeleteExpiredEmbeddings(ctx, sqlc.DeleteExpiredEmbeddingsParams{
			CreatedAt: *olderThan,
			Column2:   "memory", // content_type parameter
		})
		if err != nil {
			return fmt.Errorf("failed to delete expired memories: %w", err)
		}
	} else {
		// Delete all memories for the content type
		// Note: This deletes ALL memory type embeddings, not user-specific
		// For user-specific deletion, we would need to use metadata filtering
		metadataFilter := map[string]interface{}{
			"user_id": userID,
		}
		metadataJSON, err := json.Marshal(metadataFilter)
		if err != nil {
			return fmt.Errorf("failed to create metadata filter: %w", err)
		}

		err = ltm.queries.DeleteEmbeddingsByMetadata(ctx, metadataJSON)
		if err != nil {
			return fmt.Errorf("failed to delete user memories: %w", err)
		}
	}

	return nil
}

// GetStats returns statistics for long-term memory
func (ltm *LongTermMemory) GetStats(ctx context.Context, userID string) (*MemoryTypeStats, error) {
	// Check if database client is available
	if ltm.queries == nil {
		return &MemoryTypeStats{
			EntryCount:        0,
			TotalSize:         0,
			AverageImportance: 0,
		}, nil
	}

	ltm.logger.Debug("Long-term memory stats requested",
		slog.String("user_id", userID))

	// Get count of memory embeddings
	count, err := ltm.queries.CountEmbeddingsByType(ctx, "memory")
	if err != nil {
		ltm.logger.Warn("Failed to get memory count",
			slog.Any("error", err))
		count = 0
	}

	// Basic stats - we can't easily filter by user without metadata search
	stats := &MemoryTypeStats{
		EntryCount:        int(count),
		TotalSize:         int64(count) * 1536 * 4, // Approximate size (1536 dimensions * 4 bytes per float)
		AverageImportance: 0.5,                     // Default importance
	}

	return stats, nil
}

// Cleanup removes expired entries
func (ltm *LongTermMemory) Cleanup(ctx context.Context) error {
	ltm.logger.Info("Long-term memory cleanup started")

	// Check if database client is available
	if ltm.queries == nil {
		ltm.logger.Debug("No database client available for cleanup")
		return nil
	}

	// Delete embeddings older than 90 days
	expirationDate := time.Now().AddDate(0, -3, 0) // 3 months ago
	err := ltm.queries.DeleteExpiredEmbeddings(ctx, sqlc.DeleteExpiredEmbeddingsParams{
		CreatedAt: expirationDate,
		Column2:   "memory", // content_type parameter
	})
	if err != nil {
		return fmt.Errorf("failed to cleanup expired memories: %w", err)
	}

	ltm.logger.Info("Long-term memory cleanup completed",
		slog.Time("older_than", expirationDate))

	return nil
}

// generateEmbedding generates an embedding for the given text
func (ltm *LongTermMemory) generateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Use LangChain embedder if available
	if ltm.embedder != nil {
		embedding, err := ltm.embedder.EmbedQuery(ctx, text)
		if err != nil {
			ltm.logger.Warn("Failed to generate embedding with embedder",
				slog.Any("error", err))
			// Fall back to mock embedding
		} else {
			// Convert float32 to float64
			embeddingFloat64 := make([]float64, len(embedding))
			for i, v := range embedding {
				embeddingFloat64[i] = float64(v)
			}
			return embeddingFloat64, nil
		}
	}

	// For now, return a mock embedding
	// In a real implementation, this would call an embedding service
	mockEmbedding := make([]float64, 1536) // OpenAI embedding dimension
	for i := range mockEmbedding {
		mockEmbedding[i] = 0.1 // Simple mock values
	}

	ltm.logger.Debug("Generated mock embedding",
		slog.String("text", text[:min(50, len(text))]),
		slog.Int("dimension", len(mockEmbedding)))

	return mockEmbedding, nil
}

// parseDocumentToMemoryEntry converts a LangChain document to a memory entry
func (ltm *LongTermMemory) parseDocumentToMemoryEntry(doc schema.Document) (*MemoryEntry, error) {
	entry := &MemoryEntry{
		Content:  doc.PageContent,
		Type:     MemoryTypeLongTerm,
		Metadata: make(map[string]interface{}),
		Context:  make(map[string]interface{}),
	}

	// Parse metadata from document
	for key, value := range doc.Metadata {
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
		case "entry_id":
			if entryID, ok := value.(string); ok {
				entry.ID = entryID
			}
		case "importance":
			if importance, ok := value.(float64); ok {
				entry.Importance = importance
			}
		case "access_count":
			if count, ok := value.(float64); ok {
				entry.AccessCount = int(count)
			} else if count, ok := value.(int); ok {
				entry.AccessCount = count
			}
		case "last_access":
			if lastAccess, ok := value.(string); ok {
				if t, err := time.Parse(time.RFC3339, lastAccess); err == nil {
					entry.LastAccess = t
				}
			}
		case "created_at":
			if createdAt, ok := value.(string); ok {
				if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
					entry.CreatedAt = t
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

	// Generate ID if not found
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("doc_%s_%d", entry.UserID, time.Now().UnixNano())
	}

	return entry, nil
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
