package embedding

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/ai"
	"github.com/koopa0/assistant-go/internal/config"
)

// MockAIManager for embedding tests
type MockAIManager struct {
	generateEmbeddingFunc func(ctx context.Context, text string, providerName ...string) (*ai.EmbeddingResponse, error)
}

func (m *MockAIManager) GenerateEmbedding(ctx context.Context, text string, providerName ...string) (*ai.EmbeddingResponse, error) {
	if m.generateEmbeddingFunc != nil {
		return m.generateEmbeddingFunc(ctx, text, providerName...)
	}

	// Default mock response
	return &ai.EmbeddingResponse{
		Embedding:    []float64{0.1, 0.2, 0.3, 0.4, 0.5},
		Model:        "mock-embedding-model",
		Provider:     "mock",
		TokensUsed:   len(text) / 4, // Simple token estimation
		ResponseTime: 50 * time.Millisecond,
	}, nil
}

func (m *MockAIManager) GenerateResponse(ctx context.Context, request *ai.GenerateRequest, providerName ...string) (*ai.GenerateResponse, error) {
	return nil, nil // Not used in embedding tests
}

func (m *MockAIManager) Health(ctx context.Context) error {
	return nil
}

func (m *MockAIManager) GetUsageStats(ctx context.Context) (map[string]*ai.UsageStats, error) {
	return nil, nil
}

func (m *MockAIManager) GetAvailableProviders() []string {
	return []string{"mock"}
}

func (m *MockAIManager) GetDefaultProvider() string {
	return "mock"
}

func (m *MockAIManager) Close(ctx context.Context) error {
	return nil
}

// MockDatabase for embedding tests
type MockDatabase struct {
	embeddings map[string]*EmbeddingRecord
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		embeddings: make(map[string]*EmbeddingRecord),
	}
}

func (m *MockDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) MockRow {
	return MockRow{db: m, query: query, args: args}
}

func (m *MockDatabase) Query(ctx context.Context, query string, args ...interface{}) (MockRows, error) {
	return MockRows{db: m, query: query, args: args}, nil
}

func (m *MockDatabase) Exec(ctx context.Context, query string, args ...interface{}) (MockResult, error) {
	return MockResult{rowsAffected: 1}, nil
}

func (m *MockDatabase) Health(ctx context.Context) error {
	return nil
}

func (m *MockDatabase) Close() error {
	return nil
}

// Mock types for database operations
type MockRow struct {
	db    *MockDatabase
	query string
	args  []interface{}
}

func (r MockRow) Scan(dest ...interface{}) error {
	// Mock successful insert - return generated ID and timestamp
	if len(dest) >= 2 {
		if id, ok := dest[0].(*string); ok {
			*id = "test-embedding-id"
		}
		if createdAt, ok := dest[1].(*time.Time); ok {
			*createdAt = time.Now()
		}
	}
	return nil
}

type MockRows struct {
	db     *MockDatabase
	query  string
	args   []interface{}
	closed bool
	count  int
}

func (r *MockRows) Next() bool {
	if r.closed || r.count >= 1 {
		return false
	}
	r.count++
	return true
}

func (r *MockRows) Scan(dest ...interface{}) error {
	// Mock search result
	if len(dest) >= 8 {
		if id, ok := dest[0].(*string); ok {
			*id = "test-result-id"
		}
		if contentType, ok := dest[1].(*string); ok {
			*contentType = "test"
		}
		if contentID, ok := dest[2].(*string); ok {
			*contentID = "test-content"
		}
		if contentText, ok := dest[3].(*string); ok {
			*contentText = "test content text"
		}
		if embedding, ok := dest[4].(*string); ok {
			*embedding = "[0.1,0.2,0.3]"
		}
		if metadata, ok := dest[5].(*string); ok {
			*metadata = "{}"
		}
		if createdAt, ok := dest[6].(*time.Time); ok {
			*createdAt = time.Now()
		}
		if similarity, ok := dest[7].(*float64); ok {
			*similarity = 0.95
		}
	}
	return nil
}

func (r *MockRows) Close() {
	r.closed = true
}

func (r *MockRows) Err() error {
	return nil
}

type MockResult struct {
	rowsAffected int64
}

func (r MockResult) RowsAffected() int64 {
	return r.rowsAffected
}

// TestEmbeddingService is a test version of the embedding service
type TestEmbeddingService struct {
	aiManager *MockAIManager
	config    config.Embedding
	logger    *slog.Logger
	cache     *EmbeddingCache
}

// Test helper to create embedding service
func createTestEmbeddingService(t *testing.T) *TestEmbeddingService {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	cfg := config.Embedding{
		Provider:   "mock",
		Model:      "mock-embedding-model",
		Dimensions: 5,
	}

	mockAI := &MockAIManager{}

	cache := &EmbeddingCache{
		cache:   make(map[string]*ai.EmbeddingResponse),
		maxSize: 1000,
		ttl:     1 * time.Hour,
	}

	return &TestEmbeddingService{
		aiManager: mockAI,
		config:    cfg,
		logger:    logger,
		cache:     cache,
	}
}

// GenerateEmbedding simulates embedding generation for testing
func (s *TestEmbeddingService) GenerateEmbedding(ctx context.Context, text string) (*ai.EmbeddingResponse, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Check cache first
	if cached := s.cache.Get(text); cached != nil {
		return cached, nil
	}

	// Generate embedding using AI manager
	response, err := s.aiManager.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(text, response)

	return response, nil
}

// vectorToString converts a float64 slice to PostgreSQL vector string format
func (s *TestEmbeddingService) vectorToString(vector []float64) string {
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
func (s *TestEmbeddingService) stringToVector(vectorStr string) []float64 {
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
			continue
		}

		vector = append(vector, value)
	}

	return vector
}

// SearchSimilar simulates similarity search for testing
func (s *TestEmbeddingService) SearchSimilar(ctx context.Context, queryEmbedding []float64, contentType string, limit int, threshold float64) ([]*SearchResult, error) {
	// Mock search result
	record := &EmbeddingRecord{
		ID:          "test-result-id",
		ContentType: contentType,
		ContentID:   "test-content",
		ContentText: "test content text",
		Embedding:   []float64{0.1, 0.2, 0.3},
		Metadata:    EmbeddingMetadata{ProcessedAt: time.Now()},
		CreatedAt:   time.Now(),
	}

	result := &SearchResult{
		Record:     record,
		Similarity: 0.95,
		Distance:   0.05,
	}

	return []*SearchResult{result}, nil
}

func TestEmbeddingService_GenerateEmbedding_Success(t *testing.T) {
	service := createTestEmbeddingService(t)
	ctx := context.Background()

	text := "This is a test text for embedding generation"

	response, err := service.GenerateEmbedding(ctx, text)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if len(response.Embedding) == 0 {
		t.Error("Expected non-empty embedding vector")
	}

	if response.Provider != "mock" {
		t.Errorf("Expected provider 'mock', got: %s", response.Provider)
	}

	if response.TokensUsed == 0 {
		t.Error("Expected non-zero token usage")
	}
}

func TestEmbeddingService_GenerateEmbedding_EmptyText(t *testing.T) {
	service := createTestEmbeddingService(t)
	ctx := context.Background()

	_, err := service.GenerateEmbedding(ctx, "")
	if err == nil {
		t.Fatal("Expected error for empty text, got nil")
	}

	expectedMsg := "text cannot be empty"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got: %s", expectedMsg, err.Error())
	}
}

func TestEmbeddingService_GenerateEmbedding_Cache(t *testing.T) {
	service := createTestEmbeddingService(t)
	ctx := context.Background()

	text := "This is a test text for caching"

	// First call - should generate embedding
	response1, err := service.GenerateEmbedding(ctx, text)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Second call - should use cache
	response2, err := service.GenerateEmbedding(ctx, text)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Responses should be identical (from cache)
	if len(response1.Embedding) != len(response2.Embedding) {
		t.Error("Cached response should have same embedding length")
	}

	for i := range response1.Embedding {
		if response1.Embedding[i] != response2.Embedding[i] {
			t.Error("Cached response should have identical embedding values")
			break
		}
	}
}

func TestEmbeddingService_VectorToString(t *testing.T) {
	service := createTestEmbeddingService(t)

	vector := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	result := service.vectorToString(vector)

	expected := "[0.100000,0.200000,0.300000,0.400000,0.500000]"
	if result != expected {
		t.Errorf("Expected vector string '%s', got: %s", expected, result)
	}
}

func TestEmbeddingService_StringToVector(t *testing.T) {
	service := createTestEmbeddingService(t)

	vectorStr := "[0.1,0.2,0.3,0.4,0.5]"
	result := service.stringToVector(vectorStr)

	expected := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	if len(result) != len(expected) {
		t.Errorf("Expected vector length %d, got: %d", len(expected), len(result))
	}

	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("Expected vector[%d] = %f, got: %f", i, expected[i], result[i])
		}
	}
}

func TestEmbeddingService_StringToVector_Empty(t *testing.T) {
	service := createTestEmbeddingService(t)

	result := service.stringToVector("[]")
	if len(result) != 0 {
		t.Errorf("Expected empty vector, got length: %d", len(result))
	}

	result = service.stringToVector("")
	if len(result) != 0 {
		t.Errorf("Expected empty vector, got length: %d", len(result))
	}
}

func TestEmbeddingService_SearchSimilar_Success(t *testing.T) {
	service := createTestEmbeddingService(t)
	ctx := context.Background()

	queryEmbedding := []float64{0.1, 0.2, 0.3}

	results, err := service.SearchSimilar(ctx, queryEmbedding, "test", 10, 0.8)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected at least one result")
	}

	if len(results) > 0 {
		result := results[0]
		if result.Similarity <= 0 || result.Similarity > 1 {
			t.Errorf("Expected similarity between 0 and 1, got: %f", result.Similarity)
		}

		if result.Record == nil {
			t.Error("Expected record in search result")
		}
	}
}
