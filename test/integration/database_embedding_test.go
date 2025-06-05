package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/storage/postgres"
	"github.com/koopa0/assistant-go/internal/testutil"
	testutil2 "github.com/koopa0/assistant-go/test/testutil"
)

// TestDatabaseEmbeddingOperations demonstrates golang_guide.md testing best practices
func TestDatabaseEmbeddingOperations(t *testing.T) {
	// Setup real database using testcontainers (following golang_guide.md "use real implementations" principle)
	container, cleanup := testutil2.SetupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	pool, err := container.GetConnectionPool(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection pool: %v", err)
	}
	defer pool.Close()

	client := postgres.NewSQLCClient(pool, testutil.NewSilentLogger())

	// Table-driven tests with parallel execution (golang_guide.md pattern)
	tests := []struct {
		name        string
		contentType string
		contentID   string
		contentText string
		embedding   []float64
		metadata    map[string]interface{}
		expectError bool
	}{
		{
			name:        "valid_embedding_document",
			contentType: "document",
			contentID:   "doc-123",
			contentText: "This is a test document for embedding",
			embedding:   []float64{0.1, 0.2, 0.3, 0.4, 0.5},
			metadata:    map[string]interface{}{"source": "test", "type": "document"},
			expectError: false,
		},
		{
			name:        "valid_embedding_code",
			contentType: "code",
			contentID:   "code-456",
			contentText: "func main() { fmt.Println(\"Hello, World!\") }",
			embedding:   []float64{0.9, 0.8, 0.7, 0.6, 0.5},
			metadata:    map[string]interface{}{"language": "go", "function": "main"},
			expectError: false,
		},
		{
			name:        "empty_embedding",
			contentType: "empty",
			contentID:   "empty-789",
			contentText: "Empty embedding test",
			embedding:   []float64{},
			metadata:    nil,
			expectError: false,
		},
		{
			name:        "large_embedding",
			contentType: "large",
			contentID:   "large-001",
			contentText: "Large embedding test with many dimensions",
			embedding:   generateLargeEmbedding(1536), // OpenAI embedding size
			metadata:    map[string]interface{}{"dimensions": 1536},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable (golang_guide.md critical requirement)
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel() // Run tests in parallel (golang_guide.md performance best practice)

			// Test embedding creation
			record, err := client.CreateEmbedding(ctx, tt.contentType, tt.contentID, tt.contentText, tt.embedding, tt.metadata)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if err != nil {
				return // Skip remaining assertions if creation failed
			}

			// Verify creation results
			if record.ContentType != tt.contentType {
				t.Errorf("Expected content type %s, got %s", tt.contentType, record.ContentType)
			}
			if record.ContentID != tt.contentID {
				t.Errorf("Expected content ID %s, got %s", tt.contentID, record.ContentID)
			}
			if record.ContentText != tt.contentText {
				t.Errorf("Expected content text %s, got %s", tt.contentText, record.ContentText)
			}

			// Verify embedding vector consistency (property-based testing aspect)
			if len(record.Embedding) != len(tt.embedding) {
				t.Errorf("Embedding dimension mismatch: expected %d, got %d", len(tt.embedding), len(record.Embedding))
			}

			// Verify embedding values with tolerance for float precision
			for i, expected := range tt.embedding {
				if i >= len(record.Embedding) {
					break
				}
				actual := record.Embedding[i]
				if !floatEqual(expected, actual, 1e-6) {
					t.Errorf("Embedding value mismatch at index %d: expected %f, got %f", i, expected, actual)
				}
			}
		})
	}
}

// TestDatabaseConcurrentOperations tests concurrent database operations
func TestDatabaseConcurrentOperations(t *testing.T) {
	container, cleanup := testutil2.SetupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	pool, err := container.GetConnectionPool(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection pool: %v", err)
	}
	defer pool.Close()

	client := postgres.NewSQLCClient(pool, testutil.NewSilentLogger())

	// Test concurrent embedding creation
	const numGoroutines = 10
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			_, err := client.CreateEmbedding(
				ctx,
				"concurrent",
				fmt.Sprintf("concurrent-%d", id),
				fmt.Sprintf("Concurrent test content %d", id),
				[]float64{float64(id), float64(id) * 0.1, float64(id) * 0.2},
				map[string]interface{}{"goroutine_id": id},
			)
			errors <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errors:
			if err != nil {
				t.Errorf("Concurrent operation failed: %v", err)
			}
		case <-time.After(30 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}
}

// TestEmbeddingSearchSimilarity tests vector similarity search functionality
func TestEmbeddingSearchSimilarity(t *testing.T) {
	container, cleanup := testutil2.SetupTestDatabase(t)
	defer cleanup()

	ctx := context.Background()
	pool, err := container.GetConnectionPool(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection pool: %v", err)
	}
	defer pool.Close()

	client := postgres.NewSQLCClient(pool, testutil.NewSilentLogger())

	// Create test embeddings with known relationships
	testEmbeddings := []struct {
		id        string
		text      string
		embedding []float64
	}{
		{"similar-1", "This is about cats", []float64{1.0, 0.8, 0.6}},
		{"similar-2", "This is about felines", []float64{0.9, 0.8, 0.7}},
		{"different", "This is about cars", []float64{0.1, 0.2, 0.9}},
	}

	// Insert test embeddings
	for _, te := range testEmbeddings {
		_, err := client.CreateEmbedding(ctx, "test", te.id, te.text, te.embedding, nil)
		if err != nil {
			t.Fatalf("Failed to create test embedding %s: %v", te.id, err)
		}
	}

	// Test similarity search with threshold
	queryEmbedding := []float64{1.0, 0.8, 0.6} // Similar to "similar-1"
	results, err := client.SearchSimilarEmbeddings(ctx, queryEmbedding, "test", 2, 0.5)
	if err != nil {
		t.Fatalf("Failed to search similar embeddings: %v", err)
	}

	// Verify results (property-based testing: similar embeddings should rank higher)
	if len(results) == 0 {
		t.Fatal("Expected search results but got none")
	}

	// The most similar result should be "similar-1"
	if results[0].Record.ContentID != "similar-1" {
		t.Errorf("Expected most similar result to be 'similar-1', got '%s'", results[0].Record.ContentID)
	}
}

// generateLargeEmbedding creates a test embedding with specified dimensions
func generateLargeEmbedding(dimensions int) []float64 {
	embedding := make([]float64, dimensions)
	for i := range embedding {
		embedding[i] = float64(i%100) / 100.0 // Normalize to [0, 1)
	}
	return embedding
}

// floatEqual compares two float64 values with tolerance
func floatEqual(a, b, tolerance float64) bool {
	return abs(a-b) <= tolerance
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
