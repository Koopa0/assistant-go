package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// PersonalizationMemory implements persistent storage for user preferences and contextual information
type PersonalizationMemory struct {
	queries sqlc.Querier
	config  config.LangChain
	logger  *slog.Logger
}

// UserPreference represents a user preference or setting
type UserPreference struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Category    string                 `json:"category"`    // e.g., "ui", "behavior", "content"
	Key         string                 `json:"key"`         // preference key
	Value       interface{}            `json:"value"`       // preference value
	Type        string                 `json:"type"`        // "string", "number", "boolean", "object"
	Description string                 `json:"description"` // human-readable description
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UserContext represents contextual information about a user
type UserContext struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	ContextType  string                 `json:"context_type"`  // e.g., "session", "project", "domain"
	ContextKey   string                 `json:"context_key"`   // context identifier
	ContextValue map[string]interface{} `json:"context_value"` // context data
	Importance   float64                `json:"importance"`    // 0.0 to 1.0
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CreatedAt    time.Time              `json:"created_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NewPersonalizationMemory creates a new personalization memory instance
func NewPersonalizationMemory(queries sqlc.Querier, config config.LangChain, logger *slog.Logger) *PersonalizationMemory {
	return &PersonalizationMemory{
		queries: queries,
		config:  config,
		logger:  logger,
	}
}

// Store stores a memory entry (preference or context) in personalization memory
func (pm *PersonalizationMemory) Store(ctx context.Context, entry *MemoryEntry) error {
	// Check if database client is available
	if pm.queries == nil {
		pm.logger.Debug("No database client available for personalization memory storage",
			slog.String("entry_id", entry.ID))
		return nil // Gracefully handle missing database
	}

	// Determine if this is a preference or context based on metadata
	if category, exists := entry.Context["category"]; exists {
		// This is a preference
		return pm.storePreference(ctx, entry, category.(string))
	} else if contextType, exists := entry.Context["context_type"]; exists {
		// This is a context
		return pm.storeContext(ctx, entry, contextType.(string))
	}

	return fmt.Errorf("unable to determine personalization type from memory entry")
}

// storePreference stores a user preference
func (pm *PersonalizationMemory) storePreference(ctx context.Context, entry *MemoryEntry, category string) error {
	preference := &UserPreference{
		ID:          entry.ID,
		UserID:      entry.UserID,
		Category:    category,
		Key:         entry.Context["key"].(string),
		Value:       entry.Context["value"],
		Type:        entry.Context["type"].(string),
		Description: entry.Content,
		UpdatedAt:   time.Now(),
		CreatedAt:   entry.CreatedAt,
		Metadata:    entry.Metadata,
	}

	if preference.ID == "" {
		preference.ID = fmt.Sprintf("pref_%s_%s_%s", preference.UserID, preference.Category, preference.Key)
	}

	// Store as embedding for semantic search
	content := fmt.Sprintf("User preference: %s = %v (%s)", preference.Key, preference.Value, preference.Description)
	metadata := map[string]interface{}{
		"type":       "preference",
		"user_id":    preference.UserID,
		"category":   preference.Category,
		"key":        preference.Key,
		"value_type": preference.Type,
		"updated_at": preference.UpdatedAt,
		"created_at": preference.CreatedAt,
	}

	// Add custom metadata
	for key, value := range preference.Metadata {
		metadata[key] = value
	}

	// Generate embedding for semantic search
	embedding, err := pm.generateEmbedding(ctx, content)
	if err != nil {
		pm.logger.Warn("Failed to generate embedding for preference",
			slog.String("preference_id", preference.ID),
			slog.Any("error", err))
		embedding = nil // Continue without embedding
	}

	// Convert preference ID to UUID
	prefUUID, err := postgres.ParseUUID(preference.ID)
	if err != nil {
		return fmt.Errorf("failed to parse preference ID: %w", err)
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create embedding with queries
	_, err = pm.queries.CreateEmbedding(ctx, sqlc.CreateEmbeddingParams{
		ContentType: "personalization",
		ContentID:   prefUUID,
		ContentText: content,
		Embedding:   postgres.VectorToPgVector(embedding),
		Metadata:    metadataJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to store preference: %w", err)
	}

	pm.logger.Debug("Stored user preference",
		slog.String("id", preference.ID),
		slog.String("user_id", preference.UserID),
		slog.String("category", preference.Category),
		slog.String("key", preference.Key))

	return nil
}

// storeContext stores user context information
func (pm *PersonalizationMemory) storeContext(ctx context.Context, entry *MemoryEntry, contextType string) error {
	userContext := &UserContext{
		ID:           entry.ID,
		UserID:       entry.UserID,
		ContextType:  contextType,
		ContextKey:   entry.Context["context_key"].(string),
		ContextValue: entry.Context["context_value"].(map[string]interface{}),
		Importance:   entry.Importance,
		UpdatedAt:    time.Now(),
		CreatedAt:    entry.CreatedAt,
		Metadata:     entry.Metadata,
	}

	if userContext.ID == "" {
		userContext.ID = fmt.Sprintf("ctx_%s_%s_%s", userContext.UserID, userContext.ContextType, userContext.ContextKey)
	}

	// Set expiration if provided
	if entry.ExpiresAt != nil {
		userContext.ExpiresAt = entry.ExpiresAt
	}

	// Store as embedding for semantic search
	contextValueJSON, _ := json.Marshal(userContext.ContextValue)
	content := fmt.Sprintf("User context: %s (%s) = %s", userContext.ContextKey, userContext.ContextType, string(contextValueJSON))

	metadata := map[string]interface{}{
		"type":         "context",
		"user_id":      userContext.UserID,
		"context_type": userContext.ContextType,
		"context_key":  userContext.ContextKey,
		"importance":   userContext.Importance,
		"updated_at":   userContext.UpdatedAt,
		"created_at":   userContext.CreatedAt,
	}

	if userContext.ExpiresAt != nil {
		metadata["expires_at"] = userContext.ExpiresAt
	}

	// Add custom metadata
	for key, value := range userContext.Metadata {
		metadata[key] = value
	}

	// Generate embedding for semantic search
	embedding, err := pm.generateEmbedding(ctx, content)
	if err != nil {
		pm.logger.Warn("Failed to generate embedding for context",
			slog.String("context_id", userContext.ID),
			slog.Any("error", err))
		embedding = nil // Continue without embedding
	}

	// Convert context ID to UUID
	ctxUUID, err := postgres.ParseUUID(userContext.ID)
	if err != nil {
		return fmt.Errorf("failed to parse context ID: %w", err)
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create embedding with queries
	_, err = pm.queries.CreateEmbedding(ctx, sqlc.CreateEmbeddingParams{
		ContentType: "personalization",
		ContentID:   ctxUUID,
		ContentText: content,
		Embedding:   postgres.VectorToPgVector(embedding),
		Metadata:    metadataJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to store context: %w", err)
	}

	pm.logger.Debug("Stored user context",
		slog.String("id", userContext.ID),
		slog.String("user_id", userContext.UserID),
		slog.String("context_type", userContext.ContextType),
		slog.String("context_key", userContext.ContextKey))

	return nil
}

// Search searches for personalization data
func (pm *PersonalizationMemory) Search(ctx context.Context, query *MemoryQuery) ([]*MemorySearchResult, error) {
	results := make([]*MemorySearchResult, 0)

	// Check if database client is available
	if pm.queries == nil {
		pm.logger.Debug("No database client available for personalization memory search")
		return results, nil // Return empty results gracefully
	}

	// Generate query embedding if content provided
	var queryEmbedding []float64
	var err error

	if query.Content != "" {
		queryEmbedding, err = pm.generateEmbedding(ctx, query.Content)
		if err != nil {
			pm.logger.Warn("Failed to generate query embedding",
				slog.String("query", query.Content),
				slog.Any("error", err))
			// Continue with text-based search
		}
	}

	// Set default similarity threshold
	similarity := query.Similarity
	if similarity == 0 {
		similarity = 0.6 // Lower threshold for personalization data
	}

	// Set default limit
	limit := query.Limit
	if limit == 0 {
		limit = 20 // Higher default for personalization
	}

	// Search using semantic similarity if embedding available
	if len(queryEmbedding) > 0 {
		// Search for similar embeddings
		searchResults, err := pm.queries.SearchSimilarEmbeddings(ctx, sqlc.SearchSimilarEmbeddingsParams{
			QueryEmbedding: postgres.VectorToPgVector(queryEmbedding),
			ContentType:    "personalization",
			Threshold:      similarity,
			ResultLimit:    int32(limit),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to search personalization memory: %w", err)
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
			entry, err := pm.parseEmbeddingToMemoryEntry(embRecord)
			if err != nil {
				pm.logger.Warn("Failed to parse embedding result",
					slog.String("embedding_id", searchResult.ID.String()),
					slog.Any("error", err))
				continue
			}

			// Apply additional filters
			if !pm.matchesQuery(entry, query) {
				continue
			}

			// Convert similarity from int32 to float64
			similarityFloat := float64(searchResult.Similarity) / 100.0

			// Calculate relevance
			relevance := pm.calculateRelevance(entry, query, similarityFloat)

			memoryResult := &MemorySearchResult{
				Entry:      entry,
				Similarity: similarityFloat,
				Relevance:  relevance,
			}

			results = append(results, memoryResult)
		}

		pm.logger.Debug("Personalization memory search completed",
			slog.String("user_id", query.UserID),
			slog.Int("results", len(results)),
			slog.Float64("similarity_threshold", similarity))

		return results, nil
	}

	pm.logger.Debug("Personalization memory search completed",
		slog.String("user_id", query.UserID),
		slog.Int("results", len(results)))

	return results, nil
}

// GetUserPreferences retrieves all preferences for a user
func (pm *PersonalizationMemory) GetUserPreferences(ctx context.Context, userID string, category string) ([]*UserPreference, error) {
	query := &MemoryQuery{
		UserID: userID,
		Limit:  100,
	}

	if category != "" {
		query.Content = fmt.Sprintf("category:%s", category)
	}

	results, err := pm.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	preferences := make([]*UserPreference, 0)
	for _, result := range results {
		if result.Entry.Context["type"] == "preference" {
			pref := &UserPreference{
				ID:          result.Entry.ID,
				UserID:      result.Entry.UserID,
				Category:    result.Entry.Context["category"].(string),
				Key:         result.Entry.Context["key"].(string),
				Value:       result.Entry.Context["value"],
				Type:        result.Entry.Context["value_type"].(string),
				Description: result.Entry.Content,
				CreatedAt:   result.Entry.CreatedAt,
				Metadata:    result.Entry.Metadata,
			}
			preferences = append(preferences, pref)
		}
	}

	return preferences, nil
}

// GetUserContext retrieves context information for a user
func (pm *PersonalizationMemory) GetUserContext(ctx context.Context, userID string, contextType string) ([]*UserContext, error) {
	query := &MemoryQuery{
		UserID: userID,
		Limit:  50,
	}

	if contextType != "" {
		query.Content = fmt.Sprintf("context_type:%s", contextType)
	}

	results, err := pm.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	contexts := make([]*UserContext, 0)
	for _, result := range results {
		if result.Entry.Context["type"] == "context" {
			ctx := &UserContext{
				ID:           result.Entry.ID,
				UserID:       result.Entry.UserID,
				ContextType:  result.Entry.Context["context_type"].(string),
				ContextKey:   result.Entry.Context["context_key"].(string),
				ContextValue: result.Entry.Context["context_value"].(map[string]interface{}),
				Importance:   result.Entry.Importance,
				CreatedAt:    result.Entry.CreatedAt,
				Metadata:     result.Entry.Metadata,
			}
			contexts = append(contexts, ctx)
		}
	}

	return contexts, nil
}

// Update updates an existing memory entry
func (pm *PersonalizationMemory) Update(ctx context.Context, entry *MemoryEntry) error {
	// Update in embedding store
	return pm.Store(ctx, entry) // Store will overwrite existing entry
}

// Delete deletes a memory entry
func (pm *PersonalizationMemory) Delete(ctx context.Context, entryID string) error {
	// Check if database client is available
	if pm.queries == nil {
		pm.logger.Debug("No database client available for deletion")
		return nil
	}

	// Convert entry ID to UUID
	entryUUID, err := postgres.ParseUUID(entryID)
	if err != nil {
		return fmt.Errorf("failed to parse entry ID: %w", err)
	}

	// Delete by content type and content ID
	err = pm.queries.DeleteEmbedding(ctx, sqlc.DeleteEmbeddingParams{
		ContentType: "personalization",
		ContentID:   entryUUID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete personalization entry: %w", err)
	}

	pm.logger.Debug("Deleted personalization entry",
		slog.String("id", entryID))

	return nil
}

// Clear clears personalization data for a user
func (pm *PersonalizationMemory) Clear(ctx context.Context, userID string, olderThan *time.Time) error {
	// Check if database client is available
	if pm.queries == nil {
		pm.logger.Debug("No database client available for clearing memories")
		return nil
	}

	pm.logger.Info("Personalization memory clear requested",
		slog.String("user_id", userID),
		slog.Any("older_than", olderThan))

	// If olderThan is specified, delete expired embeddings
	if olderThan != nil {
		err := pm.queries.DeleteExpiredEmbeddings(ctx, sqlc.DeleteExpiredEmbeddingsParams{
			CreatedAt: *olderThan,
			Column2:   "personalization", // content_type parameter
		})
		if err != nil {
			return fmt.Errorf("failed to delete expired personalization data: %w", err)
		}
	} else {
		// Delete all personalization data for the user
		metadataFilter := map[string]interface{}{
			"user_id": userID,
		}
		metadataJSON, err := json.Marshal(metadataFilter)
		if err != nil {
			return fmt.Errorf("failed to create metadata filter: %w", err)
		}

		err = pm.queries.DeleteEmbeddingsByMetadata(ctx, metadataJSON)
		if err != nil {
			return fmt.Errorf("failed to delete user personalization data: %w", err)
		}
	}

	return nil
}

// GetStats returns statistics for personalization memory
func (pm *PersonalizationMemory) GetStats(ctx context.Context, userID string) (*MemoryTypeStats, error) {
	// Check if database client is available
	if pm.queries == nil {
		return &MemoryTypeStats{
			EntryCount:        0,
			TotalSize:         0,
			AverageImportance: 0,
		}, nil
	}

	pm.logger.Debug("Personalization memory stats requested",
		slog.String("user_id", userID))

	// Get count of personalization embeddings
	count, err := pm.queries.CountEmbeddingsByType(ctx, "personalization")
	if err != nil {
		pm.logger.Warn("Failed to get personalization count",
			slog.Any("error", err))
		count = 0
	}

	// Basic stats
	stats := &MemoryTypeStats{
		EntryCount:        int(count),
		TotalSize:         int64(count) * 1536 * 4, // Approximate size
		AverageImportance: 0.7,                     // Higher importance for personalization
	}

	return stats, nil
}

// Cleanup removes expired entries
func (pm *PersonalizationMemory) Cleanup(ctx context.Context) error {
	pm.logger.Info("Personalization memory cleanup started")

	// Check if database client is available
	if pm.queries == nil {
		pm.logger.Debug("No database client available for cleanup")
		return nil
	}

	// Delete personalization data older than 6 months
	expirationDate := time.Now().AddDate(0, -6, 0)
	err := pm.queries.DeleteExpiredEmbeddings(ctx, sqlc.DeleteExpiredEmbeddingsParams{
		CreatedAt: expirationDate,
		Column2:   "personalization", // content_type parameter
	})
	if err != nil {
		return fmt.Errorf("failed to cleanup expired personalization data: %w", err)
	}

	pm.logger.Info("Personalization memory cleanup completed",
		slog.Time("older_than", expirationDate))

	return nil
}

// Helper methods

func (pm *PersonalizationMemory) generateEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Mock embedding generation
	mockEmbedding := make([]float64, 1536)
	for i := range mockEmbedding {
		mockEmbedding[i] = 0.1
	}
	return mockEmbedding, nil
}

func (pm *PersonalizationMemory) parseEmbeddingToMemoryEntry(record *postgres.EmbeddingRecord) (*MemoryEntry, error) {
	entry := &MemoryEntry{
		ID:        record.ContentID,
		Type:      MemoryTypePersonalization,
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
			case "user_id":
				if userID, ok := value.(string); ok {
					entry.UserID = userID
				}
			case "importance":
				if importance, ok := value.(float64); ok {
					entry.Importance = importance
				}
			default:
				entry.Context[key] = value
			}
		}
	}

	return entry, nil
}

func (pm *PersonalizationMemory) matchesQuery(entry *MemoryEntry, query *MemoryQuery) bool {
	// Check user ID
	if entry.UserID != query.UserID {
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

func (pm *PersonalizationMemory) calculateRelevance(entry *MemoryEntry, query *MemoryQuery, similarity float64) float64 {
	// Factor in importance and recency
	importance := entry.Importance

	// Recency factor (personalization data should be relatively stable)
	daysSinceCreation := time.Since(entry.CreatedAt).Hours() / 24.0
	recency := 1.0 / (1.0 + daysSinceCreation*0.01) // Very slow decay for personalization

	// Weighted combination
	relevance := (similarity * 0.6) + (importance * 0.3) + (recency * 0.1)

	return relevance
}
