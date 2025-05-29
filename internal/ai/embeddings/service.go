package embeddings

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/koopa0/assistant/internal/ai"
	"github.com/koopa0/assistant/internal/config"
	"github.com/koopa0/assistant/internal/observability"
	"github.com/koopa0/assistant/internal/storage/postgres"
)

// Service provides embedding generation and storage capabilities
type Service struct {
	aiManager *ai.Manager
	db        *postgres.Client
	config    config.Embedding
	logger    *slog.Logger
	cache     *EmbeddingCache
}

// EmbeddingRecord represents a stored embedding
type EmbeddingRecord struct {
	ID          string                 `json:"id"`
	ContentType string                 `json:"content_type"`
	ContentID   string                 `json:"content_id"`
	ContentText string                 `json:"content_text"`
	Embedding   []float64              `json:"embedding"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

// SearchResult represents a similarity search result
type SearchResult struct {
	Record     *EmbeddingRecord `json:"record"`
	Similarity float64          `json:"similarity"`
	Distance   float64          `json:"distance"`
}

// EmbeddingCache provides in-memory caching for embeddings
type EmbeddingCache struct {
	cache   map[string]*ai.EmbeddingResponse
	mutex   sync.RWMutex
	maxSize int
	ttl     time.Duration
}

// NewService creates a new embedding service
func NewService(aiManager *ai.Manager, db *postgres.Client, cfg config.Embedding, logger *slog.Logger) (*Service, error) {
	if aiManager == nil {
		return nil, fmt.Errorf("AI manager is required")
	}
	if db == nil {
		return nil, fmt.Errorf("database client is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	cache := &EmbeddingCache{
		cache:   make(map[string]*ai.EmbeddingResponse),
		maxSize: 1000,          // TODO: Make configurable
		ttl:     1 * time.Hour, // TODO: Make configurable
	}

	return &Service{
		aiManager: aiManager,
		db:        db,
		config:    cfg,
		logger:    observability.AILogger(logger, "embeddings", cfg.Model),
		cache:     cache,
	}, nil
}

// GenerateEmbedding generates an embedding for the given text
func (s *Service) GenerateEmbedding(ctx context.Context, text string) (*ai.EmbeddingResponse, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Check cache first
	if cached := s.cache.Get(text); cached != nil {
		s.logger.Debug("Embedding cache hit", slog.String("text_length", fmt.Sprintf("%d", len(text))))
		return cached, nil
	}

	s.logger.Debug("Generating embedding",
		slog.String("provider", s.config.Provider),
		slog.String("model", s.config.Model),
		slog.Int("text_length", len(text)))

	// Generate embedding using AI provider
	response, err := s.aiManager.GenerateEmbedding(ctx, text, s.config.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Validate embedding dimensions
	if len(response.Embedding) != s.config.Dimensions {
		s.logger.Warn("Embedding dimension mismatch",
			slog.Int("expected", s.config.Dimensions),
			slog.Int("actual", len(response.Embedding)))
	}

	// Cache the result
	s.cache.Set(text, response)

	s.logger.Debug("Embedding generated",
		slog.String("provider", response.Provider),
		slog.String("model", response.Model),
		slog.Int("dimensions", len(response.Embedding)),
		slog.Duration("response_time", response.ResponseTime))

	return response, nil
}

// StoreEmbedding stores an embedding in the database
func (s *Service) StoreEmbedding(ctx context.Context, contentType, contentID, contentText string, embedding []float64, metadata map[string]interface{}) (*EmbeddingRecord, error) {
	if contentType == "" {
		return nil, fmt.Errorf("content type is required")
	}
	if contentID == "" {
		return nil, fmt.Errorf("content ID is required")
	}
	if contentText == "" {
		return nil, fmt.Errorf("content text is required")
	}
	if len(embedding) == 0 {
		return nil, fmt.Errorf("embedding is required")
	}

	// Convert embedding to PostgreSQL vector format
	embeddingStr := s.vectorToString(embedding)

	// Prepare metadata JSON
	metadataJSON := "{}"
	if metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			s.logger.Warn("Failed to marshal metadata, using empty object",
				slog.String("content_type", contentType),
				slog.String("content_id", contentID),
				slog.Any("error", err))
		} else {
			metadataJSON = string(metadataBytes)
		}
	}

	query := `
		INSERT INTO embeddings (content_type, content_id, content_text, embedding, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	var id string
	var createdAt time.Time
	err := s.db.QueryRow(ctx, query, contentType, contentID, contentText, embeddingStr, metadataJSON, time.Now()).Scan(&id, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("failed to store embedding: %w", err)
	}

	record := &EmbeddingRecord{
		ID:          id,
		ContentType: contentType,
		ContentID:   contentID,
		ContentText: contentText,
		Embedding:   embedding,
		Metadata:    metadata,
		CreatedAt:   createdAt,
	}

	s.logger.Debug("Embedding stored",
		slog.String("id", id),
		slog.String("content_type", contentType),
		slog.String("content_id", contentID))

	return record, nil
}

// GenerateAndStore generates an embedding and stores it in the database
func (s *Service) GenerateAndStore(ctx context.Context, contentType, contentID, contentText string, metadata map[string]interface{}) (*EmbeddingRecord, error) {
	// Generate embedding
	response, err := s.GenerateEmbedding(ctx, contentText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Store embedding
	record, err := s.StoreEmbedding(ctx, contentType, contentID, contentText, response.Embedding, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to store embedding: %w", err)
	}

	return record, nil
}

// SearchSimilar searches for similar embeddings using cosine similarity
func (s *Service) SearchSimilar(ctx context.Context, queryEmbedding []float64, contentType string, limit int, threshold float64) ([]*SearchResult, error) {
	if len(queryEmbedding) == 0 {
		return nil, fmt.Errorf("query embedding is required")
	}
	if limit <= 0 {
		limit = 10
	}
	if threshold <= 0 {
		threshold = 0.7 // Default similarity threshold
	}

	// Convert embedding to PostgreSQL vector format
	embeddingStr := s.vectorToString(queryEmbedding)

	query := `
		SELECT
			id, content_type, content_id, content_text, embedding, metadata, created_at,
			1 - (embedding <=> $1::vector) AS similarity
		FROM embeddings
		WHERE ($2 = '' OR content_type = $2)
		AND 1 - (embedding <=> $1::vector) >= $3
		ORDER BY embedding <=> $1::vector
		LIMIT $4
	`

	rows, err := s.db.Query(ctx, query, embeddingStr, contentType, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search embeddings: %w", err)
	}
	defer rows.Close()

	var results []*SearchResult
	for rows.Next() {
		var record EmbeddingRecord
		var embeddingStr string
		var metadataJSON string
		var similarity float64

		err := rows.Scan(
			&record.ID,
			&record.ContentType,
			&record.ContentID,
			&record.ContentText,
			&embeddingStr,
			&metadataJSON,
			&record.CreatedAt,
			&similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan embedding result: %w", err)
		}

		// Parse embedding vector
		record.Embedding = s.stringToVector(embeddingStr)

		// Parse metadata JSON
		record.Metadata = make(map[string]interface{})
		if metadataJSON != "" && metadataJSON != "{}" {
			if err := json.Unmarshal([]byte(metadataJSON), &record.Metadata); err != nil {
				s.logger.Warn("Failed to unmarshal metadata JSON",
					slog.String("record_id", record.ID),
					slog.String("metadata", metadataJSON),
					slog.Any("error", err))
			}
		}

		result := &SearchResult{
			Record:     &record,
			Similarity: similarity,
			Distance:   1 - similarity,
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating embedding results: %w", err)
	}

	s.logger.Debug("Similarity search completed",
		slog.String("content_type", contentType),
		slog.Int("results_count", len(results)),
		slog.Float64("threshold", threshold))

	return results, nil
}

// SearchSimilarByText searches for similar embeddings by generating an embedding for the query text
func (s *Service) SearchSimilarByText(ctx context.Context, queryText, contentType string, limit int, threshold float64) ([]*SearchResult, error) {
	// Generate embedding for query text
	response, err := s.GenerateEmbedding(ctx, queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search using the generated embedding
	return s.SearchSimilar(ctx, response.Embedding, contentType, limit, threshold)
}

// DeleteEmbedding deletes an embedding by ID
func (s *Service) DeleteEmbedding(ctx context.Context, id string) error {
	query := "DELETE FROM embeddings WHERE id = $1"
	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete embedding: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("embedding not found: %s", id)
	}

	s.logger.Debug("Embedding deleted", slog.String("id", id))
	return nil
}

// DeleteEmbeddingsByContent deletes all embeddings for a specific content
func (s *Service) DeleteEmbeddingsByContent(ctx context.Context, contentType, contentID string) error {
	query := "DELETE FROM embeddings WHERE content_type = $1 AND content_id = $2"
	result, err := s.db.Exec(ctx, query, contentType, contentID)
	if err != nil {
		return fmt.Errorf("failed to delete embeddings: %w", err)
	}

	s.logger.Debug("Embeddings deleted",
		slog.String("content_type", contentType),
		slog.String("content_id", contentID),
		slog.Int64("count", result.RowsAffected()))

	return nil
}

// vectorToString converts a float64 slice to PostgreSQL vector string format
func (s *Service) vectorToString(vector []float64) string {
	if len(vector) == 0 {
		return "[]"
	}

	result := "["
	for i, v := range vector {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf("%f", v)
	}
	result += "]"
	return result
}

// stringToVector converts a PostgreSQL vector string to float64 slice
func (s *Service) stringToVector(vectorStr string) []float64 {
	if vectorStr == "" || vectorStr == "[]" {
		return make([]float64, 0)
	}

	// Remove brackets and split by comma
	vectorStr = strings.Trim(vectorStr, "[]")
	parts := strings.Split(vectorStr, ",")

	vector := make([]float64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		value, err := strconv.ParseFloat(part, 64)
		if err != nil {
			s.logger.Warn("Failed to parse vector component",
				slog.String("component", part),
				slog.String("vector_string", vectorStr),
				slog.Any("error", err))
			continue
		}

		vector = append(vector, value)
	}

	return vector
}

// EmbeddingCache methods

// Get retrieves an embedding from cache
func (c *EmbeddingCache) Get(key string) *ai.EmbeddingResponse {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if response, exists := c.cache[key]; exists {
		return response
	}
	return nil
}

// Set stores an embedding in cache
func (c *EmbeddingCache) Set(key string, response *ai.EmbeddingResponse) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Simple cache eviction if at max size
	if len(c.cache) >= c.maxSize {
		// Remove oldest entry (simple FIFO)
		for k := range c.cache {
			delete(c.cache, k)
			break
		}
	}

	c.cache[key] = response
}

// Clear clears the cache
func (c *EmbeddingCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache = make(map[string]*ai.EmbeddingResponse)
}
