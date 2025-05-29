# Vector Store Package

The vectorstore package provides a PostgreSQL-based vector store implementation that integrates seamlessly with LangChain Go, enabling semantic search capabilities using pgvector extension.

## Overview

This package implements LangChain's `VectorStore` interface using PostgreSQL with the pgvector extension, providing persistent vector storage and similarity search functionality for the AI assistant's RAG (Retrieval-Augmented Generation) capabilities.

## Core Components

### PGVectorStore (`pgvector.go`)

The main vector store implementation that bridges LangChain's interface with PostgreSQL pgvector storage.

```go
// Create vector store
vectorStore := vectorstore.NewPGVectorStore(dbClient, embedder, logger)

// Configure collection
vectorStore.SetCollection("documents")

// Add documents with automatic embedding generation
docs := []schema.Document{
    {
        PageContent: "Kubernetes deployment best practices",
        Metadata: map[string]any{
            "source": "documentation",
            "type":   "guide",
        },
    },
}
ids, err := vectorStore.AddDocuments(ctx, docs)

// Semantic search
results, err := vectorStore.SimilaritySearch(ctx, "deployment strategies", 5)
```

## Key Features

### LangChain Interface Compliance

The `PGVectorStore` fully implements the `vectorstores.VectorStore` interface:

```go
// Interface methods implemented
type VectorStore interface {
    AddDocuments(ctx context.Context, docs []schema.Document, options ...Option) ([]string, error)
    SimilaritySearch(ctx context.Context, query string, numDocuments int, options ...Option) ([]schema.Document, error)
    RemoveCollection(ctx context.Context, collection string) error
    GetNumDocuments(ctx context.Context) (int, error)
    Delete(ctx context.Context, ids []string) error
    Close() error
}

// Additional methods for enhanced functionality
func (vs *PGVectorStore) SimilaritySearchWithScore(ctx context.Context, query string, numDocuments int, options ...Option) ([]schema.Document, []float32, error)
func (vs *PGVectorStore) AsRetriever(ctx context.Context, options ...Option) schema.Retriever
```

### Automatic Embedding Generation

The vector store automatically generates embeddings for document content using the configured embedder:

```go
// Embeddings are generated automatically during document addition
docs := []schema.Document{
    {PageContent: "Your document content here"},
}

// Embedding generation happens internally
ids, err := vectorStore.AddDocuments(ctx, docs)
```

### Flexible Similarity Search

Multiple search options with configurable parameters:

```go
// Basic similarity search
docs, err := vectorStore.SimilaritySearch(ctx, "query", 10)

// Search with similarity threshold
docs, err := vectorStore.SimilaritySearch(ctx, "query", 10, 
    vectorstores.WithScoreThreshold(0.8))

// Search with scores
docs, scores, err := vectorStore.SimilaritySearchWithScore(ctx, "query", 10)

// Access similarity scores from metadata
for _, doc := range docs {
    if score, ok := doc.Metadata["similarity_score"].(float64); ok {
        fmt.Printf("Document score: %f\n", score)
    }
}
```

### Collection-based Organization

Organize documents into collections for better data management:

```go
// Set collection for document organization
vectorStore.SetCollection("technical_docs")
vectorStore.AddDocuments(ctx, techDocs)

vectorStore.SetCollection("user_guides") 
vectorStore.AddDocuments(ctx, userGuides)

// Search within specific collection context
vectorStore.SetCollection("technical_docs")
results, err := vectorStore.SimilaritySearch(ctx, "API endpoints", 5)
```

## Configuration

### Database Setup

The vector store requires PostgreSQL with pgvector extension:

```sql
-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Tables are automatically created by the SQLCClient
-- See internal/storage/postgres/migrations/ for schema
```

### Environment Configuration

```bash
# Database connection
DATABASE_URL=postgresql://user:pass@localhost/assistant

# AI provider for embeddings  
CLAUDE_API_KEY=your_claude_key
GEMINI_API_KEY=your_gemini_key
```

### Application Configuration

```yaml
langchain:
  vector_dimensions: 1536    # Embedding dimensions
  similarity_threshold: 0.7  # Default similarity threshold
```

## Usage Examples

### Basic Document Storage and Retrieval

```go
package main

import (
    "context"
    "log"
    
    "github.com/tmc/langchaingo/schema"
    
    "github.com/koopa0/assistant-go/internal/langchain/vectorstore"
    "github.com/koopa0/assistant-go/internal/storage/postgres"
    "github.com/koopa0/assistant-go/internal/ai"
)

func main() {
    ctx := context.Background()
    
    // Initialize dependencies
    dbClient := postgres.NewSQLCClient(config, logger)
    embedder := ai.NewEmbeddingService(aiFactory, config.AI, logger)
    
    // Create vector store
    vs := vectorstore.NewPGVectorStore(dbClient, embedder, logger)
    vs.SetCollection("knowledge_base")
    
    // Add documents
    docs := []schema.Document{
        {
            PageContent: "Go is a programming language developed by Google",
            Metadata: map[string]any{
                "topic":    "programming",
                "language": "go",
                "source":   "documentation",
            },
        },
        {
            PageContent: "Docker containers provide application isolation",
            Metadata: map[string]any{
                "topic":      "devops",
                "technology": "docker",
                "source":     "guide",
            },
        },
    }
    
    ids, err := vs.AddDocuments(ctx, docs)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Added documents with IDs: %v", ids)
    
    // Search for relevant documents
    results, err := vs.SimilaritySearch(ctx, "container technology", 3)
    if err != nil {
        log.Fatal(err)
    }
    
    for i, doc := range results {
        score := doc.Metadata["similarity_score"].(float64)
        log.Printf("Result %d (score: %.3f): %s", i+1, score, doc.PageContent)
    }
}
```

### Integration with LangChain Chains

```go
// Use vector store with LangChain retrieval chains
retriever := vectorStore.AsRetriever(ctx, 
    vectorstores.WithScoreThreshold(0.7))

// Create retrieval chain (simplified example)
chain := chains.NewRetrievalQA(
    llm,           // Language model
    retriever,     // Vector store retriever
    chains.WithMaxTokens(1000),
)

// Execute chain with query
response, err := chain.Call(ctx, map[string]any{
    "query": "How do I deploy applications with Docker?",
})
```

### Advanced Filtering and Search

```go
// Search with multiple filters
options := []vectorstores.Option{
    vectorstores.WithScoreThreshold(0.8),
    vectorstores.WithFilters(map[string]any{
        "topic":  "devops",
        "source": "guide",
    }),
}

results, err := vectorStore.SimilaritySearch(ctx, "deployment", 5, options...)

// Search with scores for analysis
docs, scores, err := vectorStore.SimilaritySearchWithScore(ctx, "kubernetes", 10)
for i, doc := range docs {
    log.Printf("Document %d: score=%.3f, content=%s", 
        i+1, scores[i], doc.PageContent[:100])
}
```

### Batch Document Processing

```go
// Process multiple document batches efficiently
func processBatchDocuments(vs *vectorstore.PGVectorStore, docBatches [][]schema.Document) error {
    for i, batch := range docBatches {
        log.Printf("Processing batch %d/%d (%d documents)", 
            i+1, len(docBatches), len(batch))
        
        ids, err := vs.AddDocuments(ctx, batch)
        if err != nil {
            return fmt.Errorf("failed to process batch %d: %w", i+1, err)
        }
        
        log.Printf("Batch %d processed, document IDs: %v", i+1, ids)
    }
    return nil
}
```

## Integration with Memory System

The vector store integrates seamlessly with the memory system for enhanced functionality:

```go
// Enhanced long-term memory with vector store
ltm := memory.NewLongTermMemoryWithVectorStore(dbClient, vectorStore, embedder, config, logger)

// Store memory with semantic indexing
entry := &memory.MemoryEntry{
    Type:       memory.MemoryTypeLongTerm,
    UserID:     "user123",
    Content:    "User prefers microservices architecture for scalability",
    Importance: 0.8,
    Context: map[string]interface{}{
        "domain":     "architecture",
        "preference": "microservices",
    },
}
err := ltm.Store(ctx, entry)

// Semantic search through memory
query := &memory.MemoryQuery{
    UserID:     "user123",
    Content:    "scalable architecture patterns",
    Similarity: 0.7,
}
results, err := ltm.Search(ctx, query)
```

## Performance Optimization

### Indexing Strategy

The vector store uses pgvector indexes for optimal performance:

```sql
-- Indexes are automatically created during migration
-- Custom indexes can be added for specific use cases

-- Example: Additional index for metadata filtering
CREATE INDEX idx_embeddings_metadata_topic 
ON embeddings USING gin ((metadata->>'topic'));
```

### Batch Operations

```go
// Batch document addition for better performance
const batchSize = 100

func addDocumentsInBatches(vs *vectorstore.PGVectorStore, docs []schema.Document) error {
    for i := 0; i < len(docs); i += batchSize {
        end := i + batchSize
        if end > len(docs) {
            end = len(docs)
        }
        
        batch := docs[i:end]
        _, err := vs.AddDocuments(ctx, batch)
        if err != nil {
            return fmt.Errorf("batch %d-%d failed: %w", i, end-1, err)
        }
        
        // Optional: Add delay between batches to avoid overwhelming the database
        time.Sleep(100 * time.Millisecond)
    }
    return nil
}
```

### Memory Usage

```go
// Monitor vector store memory usage
func monitorVectorStoreUsage(vs *vectorstore.PGVectorStore) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        count, err := vs.GetNumDocuments(ctx)
        if err != nil {
            log.Printf("Failed to get document count: %v", err)
            continue
        }
        
        log.Printf("Vector store contains %d documents", count)
    }
}
```

## Error Handling

The vector store implements comprehensive error handling:

```go
// Handle embedding generation failures
docs, err := vectorStore.AddDocuments(ctx, documents)
if err != nil {
    if strings.Contains(err.Error(), "failed to generate embedding") {
        log.Printf("Embedding generation failed, retrying with smaller batch")
        // Implement retry logic with smaller batches
    } else if strings.Contains(err.Error(), "database connection") {
        log.Printf("Database connection issue, implementing fallback")
        // Implement fallback storage mechanism
    } else {
        return fmt.Errorf("unexpected error: %w", err)
    }
}

// Handle search failures gracefully
results, err := vectorStore.SimilaritySearch(ctx, query, limit)
if err != nil {
    log.Printf("Vector search failed, falling back to keyword search: %v", err)
    // Implement fallback search mechanism
    return fallbackKeywordSearch(query)
}
```

## Testing

### Unit Tests

```go
func TestPGVectorStore_AddDocuments(t *testing.T) {
    // Setup test dependencies
    dbClient := setupTestDB(t)
    embedder := &mockEmbedder{}
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    
    vs := vectorstore.NewPGVectorStore(dbClient, embedder, logger)
    
    // Test document addition
    docs := []schema.Document{
        {PageContent: "test content", Metadata: map[string]any{"test": true}},
    }
    
    ids, err := vs.AddDocuments(context.Background(), docs)
    assert.NoError(t, err)
    assert.Len(t, ids, 1)
}

func TestPGVectorStore_SimilaritySearch(t *testing.T) {
    // Test similarity search functionality
    vs := setupTestVectorStore(t)
    
    // Add test documents
    addTestDocuments(t, vs)
    
    // Perform search
    results, err := vs.SimilaritySearch(ctx, "test query", 5)
    assert.NoError(t, err)
    assert.NotEmpty(t, results)
    
    // Verify similarity scores
    for _, doc := range results {
        score, exists := doc.Metadata["similarity_score"]
        assert.True(t, exists)
        assert.GreaterOrEqual(t, score.(float64), 0.0)
    }
}
```

### Integration Tests

```go
func TestVectorStoreIntegration(t *testing.T) {
    // Test with real database and embeddings
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    ctx := context.Background()
    vs := setupRealVectorStore(t)
    
    // Test end-to-end workflow
    docs := createTestDocuments()
    ids, err := vs.AddDocuments(ctx, docs)
    require.NoError(t, err)
    
    // Test search
    results, err := vs.SimilaritySearch(ctx, "relevant query", 3)
    require.NoError(t, err)
    require.NotEmpty(t, results)
    
    // Test retriever interface
    retriever := vs.AsRetriever(ctx)
    retrievedDocs, err := retriever.GetRelevantDocuments(ctx, "test query")
    require.NoError(t, err)
    require.NotEmpty(t, retrievedDocs)
}
```

## Best Practices

### Document Structure

```go
// Good: Well-structured documents with meaningful metadata
doc := schema.Document{
    PageContent: "Detailed explanation of the feature or concept",
    Metadata: map[string]any{
        "source":     "documentation",
        "category":   "api",
        "last_updated": time.Now(),
        "author":     "team-name",
        "version":    "1.0",
    },
}

// Good: Consistent content formatting
doc := schema.Document{
    PageContent: fmt.Sprintf("Title: %s\n\nContent: %s\n\nSummary: %s", 
        title, content, summary),
    Metadata: map[string]any{
        "title":   title,
        "summary": summary,
        "type":    "article",
    },
}
```

### Search Optimization

```go
// Use appropriate similarity thresholds
results, err := vectorStore.SimilaritySearch(ctx, query, 10,
    vectorstores.WithScoreThreshold(0.7)) // Adjust based on use case

// Limit results to prevent overwhelming responses
maxResults := 5
if userExperienceLevel == "beginner" {
    maxResults = 3 // Fewer, more targeted results for beginners
}

results, err := vectorStore.SimilaritySearch(ctx, query, maxResults)
```

### Resource Management

```go
// Proper resource cleanup
defer func() {
    if err := vectorStore.Close(); err != nil {
        log.Printf("Failed to close vector store: %v", err)
    }
}()

// Connection pooling and timeout management
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results, err := vectorStore.SimilaritySearch(ctx, query, limit)
```

This vector store implementation provides a robust, scalable foundation for semantic search capabilities in the AI assistant, enabling sophisticated RAG workflows and intelligent information retrieval.