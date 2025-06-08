package embedding

import (
	"fmt"
	"math"
	"time"
)

// EmbeddingRecord represents an embedding with proper types
// Following Go best practice: explicit types instead of map[string]interface{}
type EmbeddingRecord struct {
	ID          string            `json:"id"`
	ContentType string            `json:"content_type"`
	ContentID   string            `json:"content_id"`
	ContentText string            `json:"content_text"`
	Embedding   []float64         `json:"embedding"`
	Metadata    EmbeddingMetadata `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
}

// EmbeddingMetadata contains structured metadata for embeddings
// Replaces map[string]interface{} with explicit fields
type EmbeddingMetadata struct {
	// Source information
	Source    string `json:"source,omitempty"`     // e.g., "conversation", "document", "code"
	SourceURL string `json:"source_url,omitempty"` // Original location if applicable

	// Content information
	ContentHash string `json:"content_hash,omitempty"` // For deduplication
	ChunkIndex  int    `json:"chunk_index,omitempty"`  // If part of larger document
	ChunkTotal  int    `json:"chunk_total,omitempty"`  // Total chunks

	// Processing information
	Model       string    `json:"model,omitempty"`      // Model used for embedding
	Dimensions  int       `json:"dimensions,omitempty"` // Embedding dimensions
	ProcessedAt time.Time `json:"processed_at"`         // When embedding was created

	// Categorization
	Category string   `json:"category,omitempty"` // Content category
	Tags     []string `json:"tags,omitempty"`     // Associated tags
	Language string   `json:"language,omitempty"` // Content language
}

// SearchResult represents an embedding search result
type SearchResult struct {
	Record     *EmbeddingRecord `json:"record"`
	Similarity float64          `json:"similarity"`
	Distance   float64          `json:"distance"`
}

// GenerateRequest contains parameters for embedding generation
type GenerateRequest struct {
	ContentType string            `json:"content_type"`
	ContentID   string            `json:"content_id"`
	ContentText string            `json:"content_text"`
	Metadata    EmbeddingMetadata `json:"metadata"`
}

// SearchRequest contains parameters for similarity search
type SearchRequest struct {
	Query         string   `json:"query"`          // Text to search for
	ContentTypes  []string `json:"content_types"`  // Filter by content types
	Limit         int      `json:"limit"`          // Max results
	MinSimilarity float64  `json:"min_similarity"` // Minimum similarity threshold
}

// BatchGenerateRequest for generating multiple embeddings
type BatchGenerateRequest struct {
	Requests []GenerateRequest `json:"requests"`
}

// EmbeddingStats provides statistics about stored embeddings
type EmbeddingStats struct {
	TotalCount      int64            `json:"total_count"`
	ByContentType   map[string]int64 `json:"by_content_type"`
	TotalDimensions int64            `json:"total_dimensions"`
	OldestEmbedding time.Time        `json:"oldest_embedding"`
	NewestEmbedding time.Time        `json:"newest_embedding"`
}

// Custom errors following Go patterns
type EmbeddingNotFoundError struct {
	ContentID string
}

func (e EmbeddingNotFoundError) Error() string {
	return "embedding not found for content: " + e.ContentID
}

type EmbeddingGenerationError struct {
	ContentID string
	Err       error
}

func (e EmbeddingGenerationError) Error() string {
	return "failed to generate embedding for " + e.ContentID + ": " + e.Err.Error()
}

func (e EmbeddingGenerationError) Unwrap() error {
	return e.Err
}

// Helper functions

// NewEmbeddingMetadata creates metadata with defaults
func NewEmbeddingMetadata(source string) EmbeddingMetadata {
	return EmbeddingMetadata{
		Source:      source,
		ProcessedAt: time.Now(),
		Tags:        []string{},
	}
}

// ValidateEmbedding ensures embedding data is valid
func ValidateEmbedding(embedding []float64) error {
	if len(embedding) == 0 {
		return fmt.Errorf("embedding cannot be empty")
	}

	// Check for NaN or Inf values
	for i, val := range embedding {
		if math.IsNaN(val) || math.IsInf(val, 0) {
			return fmt.Errorf("invalid embedding value at index %d: %f", i, val)
		}
	}

	return nil
}
