package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koopa0/assistant-go/internal/storage/postgres/sqlc"
	"github.com/pgvector/pgvector-go"
)

// SQLCImprovements provides enhanced SQLC integration with proper vector operations
type SQLCImprovements struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *slog.Logger
}

// NewSQLCImprovements creates an enhanced SQLC client with vector support
func NewSQLCImprovements(pool *pgxpool.Pool, logger *slog.Logger) *SQLCImprovements {
	return &SQLCImprovements{
		pool:    pool,
		queries: sqlc.New(pool),
		logger:  logger,
	}
}

// float64ToFloat32 converts []float64 to []float32 for pgvector compatibility
func float64ToFloat32(f64 []float64) []float32 {
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32
}

// float32ToFloat64 converts []float32 to []float64 for domain compatibility
func float32ToFloat64(f32 []float32) []float64 {
	f64 := make([]float64, len(f32))
	for i, v := range f32 {
		f64[i] = float64(v)
	}
	return f64
}

// SearchSimilarEmbeddingsOptimized uses pgx v5 generic row collection following golang_guide.md best practices
func (s *SQLCImprovements) SearchSimilarEmbeddingsOptimized(ctx context.Context, queryEmbedding []float64, contentType string, limit int) ([]EmbeddingRecord, error) {
	// Convert to pgvector.Vector following golang_guide.md type safety practices
	vector := pgvector.NewVector(float64ToFloat32(queryEmbedding))

	query := `
		SELECT id, content_type, content_id, content_text, embedding, metadata, created_at,
		       1 - (embedding <=> $1::vector) as similarity
		FROM embeddings 
		WHERE content_type = $2
		  AND 1 - (embedding <=> $1::vector) > 0.3
		ORDER BY embedding <=> $1::vector
		LIMIT $3`

	rows, err := s.pool.Query(ctx, query, vector, contentType, limit)
	if err != nil {
		return nil, fmt.Errorf("similarity search failed: %w", err)
	}
	defer rows.Close()

	// pgx v5 泛型行收集（遵循 golang_guide.md 的類型安全實踐）
	type EmbeddingSearchResult struct {
		ID          string          `db:"id"`
		ContentType string          `db:"content_type"`
		ContentID   string          `db:"content_id"`
		ContentText string          `db:"content_text"`
		Embedding   pgvector.Vector `db:"embedding"`
		Metadata    []byte          `db:"metadata"`
		CreatedAt   time.Time       `db:"created_at"`
		Similarity  float64         `db:"similarity"`
	}

	searchResults, err := pgx.CollectRows(rows, pgx.RowToStructByName[EmbeddingSearchResult])
	if err != nil {
		return nil, fmt.Errorf("failed to collect search results: %w", err)
	}

	// Convert to domain objects (following "Accept interfaces, return structs" principle)
	result := make([]EmbeddingRecord, len(searchResults))
	for i, row := range searchResults {
		var metadata map[string]interface{}
		if len(row.Metadata) > 0 {
			if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
				s.logger.Warn("Failed to unmarshal metadata",
					slog.String("content_id", row.ContentID),
					slog.Any("error", err))
			}
		}

		result[i] = EmbeddingRecord{
			ID:          row.ID,
			ContentType: row.ContentType,
			ContentID:   row.ContentID,
			ContentText: row.ContentText,
			Embedding:   float32ToFloat64(row.Embedding.Slice()),
			Metadata:    metadata,
			CreatedAt:   row.CreatedAt,
		}
	}

	return result, nil
}

// CreateEmbeddingWithVector creates an embedding using proper pgvector types
func (s *SQLCImprovements) CreateEmbeddingWithVector(ctx context.Context, contentType, contentID, contentText string, embedding []float64, metadata map[string]interface{}) (*EmbeddingRecord, error) {
	// Convert to pgvector.Vector for proper type safety
	vector := pgvector.NewVector(float64ToFloat32(embedding))

	// Use raw SQL with proper vector type handling
	query := `
		INSERT INTO embeddings (content_type, content_id, content_text, embedding, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var id string
	var createdAt, updatedAt time.Time

	err := s.pool.QueryRow(ctx, query, contentType, contentID, contentText, vector, metadata).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		s.logger.Error("Failed to create embedding",
			slog.String("content_type", contentType),
			slog.String("content_id", contentID),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	s.logger.Debug("Created embedding with vector",
		slog.String("id", id),
		slog.String("content_type", contentType),
		slog.Int("dimensions", len(embedding)))

	return &EmbeddingRecord{
		ID:          id,
		ContentType: contentType,
		ContentID:   contentID,
		ContentText: contentText,
		Embedding:   embedding,
		Metadata:    metadata,
		CreatedAt:   createdAt,
		UpdatedAt:   &updatedAt,
	}, nil
}

// SearchSimilarEmbeddingsWithVector performs proper vector similarity search
func (s *SQLCImprovements) SearchSimilarEmbeddingsWithVector(ctx context.Context, queryEmbedding []float64, contentType string, limit int, threshold float64) ([]*EmbeddingSearchResult, error) {
	vector := pgvector.NewVector(float64ToFloat32(queryEmbedding))

	// Use proper vector similarity search with cosine distance
	query := `
		SELECT 
			id, content_type, content_id, content_text, embedding, metadata, created_at,
			1 - (embedding <=> $1) as similarity
		FROM embeddings 
		WHERE content_type = $2 
			AND (1 - (embedding <=> $1)) >= $3
		ORDER BY embedding <=> $1
		LIMIT $4
	`

	rows, err := s.pool.Query(ctx, query, vector, contentType, threshold, limit)
	if err != nil {
		s.logger.Error("Failed to search similar embeddings",
			slog.String("content_type", contentType),
			slog.Float64("threshold", threshold),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to search similar embeddings: %w", err)
	}
	defer rows.Close()

	var results []*EmbeddingSearchResult
	for rows.Next() {
		var record EmbeddingRecord
		var similarity float64
		var embeddingVector pgvector.Vector

		err := rows.Scan(
			&record.ID,
			&record.ContentType,
			&record.ContentID,
			&record.ContentText,
			&embeddingVector,
			&record.Metadata,
			&record.CreatedAt,
			&similarity,
		)
		if err != nil {
			s.logger.Error("Failed to scan embedding result", slog.Any("error", err))
			continue
		}

		// Convert pgvector.Vector back to []float64
		record.Embedding = float32ToFloat64(embeddingVector.Slice())

		results = append(results, &EmbeddingSearchResult{
			Record:     &record,
			Similarity: similarity,
			Distance:   1.0 - similarity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating embedding results: %w", err)
	}

	s.logger.Debug("Found similar embeddings",
		slog.String("content_type", contentType),
		slog.Int("count", len(results)),
		slog.Float64("threshold", threshold))

	return results, nil
}

// BatchCreateEmbeddings creates multiple embeddings in a single transaction
func (s *SQLCImprovements) BatchCreateEmbeddings(ctx context.Context, embeddings []*EmbeddingRecord) error {
	if len(embeddings) == 0 {
		return nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Prepare batch insert
	batch := &pgx.Batch{}
	query := `
		INSERT INTO embeddings (content_type, content_id, content_text, embedding, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`

	for _, embedding := range embeddings {
		vector := pgvector.NewVector(float64ToFloat32(embedding.Embedding))
		batch.Queue(query, embedding.ContentType, embedding.ContentID, embedding.ContentText, vector, embedding.Metadata)
	}

	// Execute batch
	batchResults := tx.SendBatch(ctx, batch)
	defer batchResults.Close()

	// Process results
	for i := 0; i < len(embeddings); i++ {
		_, err := batchResults.Exec()
		if err != nil {
			s.logger.Error("Failed to insert embedding in batch",
				slog.Int("index", i),
				slog.String("content_type", embeddings[i].ContentType),
				slog.Any("error", err))
			return fmt.Errorf("failed to insert embedding %d: %w", i, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit batch embeddings: %w", err)
	}

	s.logger.Info("Batch created embeddings",
		slog.Int("count", len(embeddings)))

	return nil
}

// GetEmbeddingsByContentIDs retrieves embeddings by multiple content IDs
func (s *SQLCImprovements) GetEmbeddingsByContentIDs(ctx context.Context, contentType string, contentIDs []string) ([]*EmbeddingRecord, error) {
	if len(contentIDs) == 0 {
		return []*EmbeddingRecord{}, nil
	}

	// Use ANY operator for efficient IN query
	query := `
		SELECT id, content_type, content_id, content_text, embedding, metadata, created_at, updated_at
		FROM embeddings 
		WHERE content_type = $1 AND content_id = ANY($2)
		ORDER BY created_at DESC
	`

	rows, err := s.pool.Query(ctx, query, contentType, contentIDs)
	if err != nil {
		s.logger.Error("Failed to get embeddings by content IDs",
			slog.String("content_type", contentType),
			slog.Int("id_count", len(contentIDs)),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to get embeddings by content IDs: %w", err)
	}
	defer rows.Close()

	var results []*EmbeddingRecord
	for rows.Next() {
		var record EmbeddingRecord
		var embeddingVector pgvector.Vector

		err := rows.Scan(
			&record.ID,
			&record.ContentType,
			&record.ContentID,
			&record.ContentText,
			&embeddingVector,
			&record.Metadata,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan embedding record", slog.Any("error", err))
			continue
		}

		// Convert pgvector.Vector back to []float64
		record.Embedding = float32ToFloat64(embeddingVector.Slice())
		results = append(results, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating embedding records: %w", err)
	}

	s.logger.Debug("Retrieved embeddings by content IDs",
		slog.String("content_type", contentType),
		slog.Int("requested", len(contentIDs)),
		slog.Int("found", len(results)))

	return results, nil
}

// UpdateEmbeddingMetadata updates only the metadata of an embedding
func (s *SQLCImprovements) UpdateEmbeddingMetadata(ctx context.Context, embeddingID string, metadata map[string]interface{}) error {
	query := `
		UPDATE embeddings 
		SET metadata = $2, updated_at = NOW()
		WHERE id = $1
	`

	result, err := s.pool.Exec(ctx, query, embeddingID, metadata)
	if err != nil {
		s.logger.Error("Failed to update embedding metadata",
			slog.String("embedding_id", embeddingID),
			slog.Any("error", err))
		return fmt.Errorf("failed to update embedding metadata: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("embedding not found: %s", embeddingID)
	}

	s.logger.Debug("Updated embedding metadata",
		slog.String("embedding_id", embeddingID))

	return nil
}

// DeleteEmbeddingsByContentType deletes all embeddings of a specific content type
func (s *SQLCImprovements) DeleteEmbeddingsByContentType(ctx context.Context, contentType string) (int64, error) {
	query := `DELETE FROM embeddings WHERE content_type = $1`

	result, err := s.pool.Exec(ctx, query, contentType)
	if err != nil {
		s.logger.Error("Failed to delete embeddings by content type",
			slog.String("content_type", contentType),
			slog.Any("error", err))
		return 0, fmt.Errorf("failed to delete embeddings by content type: %w", err)
	}

	rowsAffected := result.RowsAffected()
	s.logger.Info("Deleted embeddings by content type",
		slog.String("content_type", contentType),
		slog.Int64("count", rowsAffected))

	return rowsAffected, nil
}

// GetEmbeddingStats returns statistics about embeddings
func (s *SQLCImprovements) GetEmbeddingStats(ctx context.Context) (*EmbeddingStats, error) {
	query := `
		SELECT 
			content_type,
			COUNT(*) as count,
			AVG(LENGTH(content_text)) as avg_text_length,
			MIN(created_at) as earliest_created,
			MAX(created_at) as latest_created
		FROM embeddings 
		GROUP BY content_type
		ORDER BY count DESC
	`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		s.logger.Error("Failed to get embedding stats", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get embedding stats: %w", err)
	}
	defer rows.Close()

	stats := &EmbeddingStats{
		ByContentType: make(map[string]*ContentTypeStats),
	}

	var totalCount int64
	for rows.Next() {
		var contentType string
		var count int64
		var avgTextLength float64
		var earliestCreated, latestCreated time.Time

		err := rows.Scan(&contentType, &count, &avgTextLength, &earliestCreated, &latestCreated)
		if err != nil {
			s.logger.Error("Failed to scan embedding stats", slog.Any("error", err))
			continue
		}

		stats.ByContentType[contentType] = &ContentTypeStats{
			Count:           count,
			AvgTextLength:   avgTextLength,
			EarliestCreated: earliestCreated,
			LatestCreated:   latestCreated,
		}

		totalCount += count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating embedding stats: %w", err)
	}

	stats.TotalCount = totalCount
	stats.LastUpdated = time.Now()

	s.logger.Debug("Retrieved embedding stats",
		slog.Int64("total_count", totalCount),
		slog.Int("content_types", len(stats.ByContentType)))

	return stats, nil
}

// EmbeddingStats represents statistics about embeddings
type EmbeddingStats struct {
	TotalCount    int64                        `json:"total_count"`
	ByContentType map[string]*ContentTypeStats `json:"by_content_type"`
	LastUpdated   time.Time                    `json:"last_updated"`
}

// ContentTypeStats represents statistics for a specific content type
type ContentTypeStats struct {
	Count           int64     `json:"count"`
	AvgTextLength   float64   `json:"avg_text_length"`
	EarliestCreated time.Time `json:"earliest_created"`
	LatestCreated   time.Time `json:"latest_created"`
}
