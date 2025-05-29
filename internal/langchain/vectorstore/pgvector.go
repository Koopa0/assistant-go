package vectorstore

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"

	"github.com/koopa0/assistant/internal/storage/postgres"
)

// PGVectorStore implements LangChain's VectorStore interface using PostgreSQL with pgvector
type PGVectorStore struct {
	dbClient   *postgres.SQLCClient
	embedder   embeddings.Embedder
	logger     *slog.Logger
	collection string
	dimensions int
}

// NewPGVectorStore creates a new PGVector store
func NewPGVectorStore(dbClient *postgres.SQLCClient, embedder embeddings.Embedder, logger *slog.Logger) *PGVectorStore {
	return &PGVectorStore{
		dbClient:   dbClient,
		embedder:   embedder,
		logger:     logger,
		collection: "langchain_documents",
		dimensions: 1536, // Default OpenAI embedding dimension
	}
}

// SetCollection sets the collection name for storing documents
func (vs *PGVectorStore) SetCollection(collection string) {
	vs.collection = collection
}

// AddDocuments adds documents to the vector store
func (vs *PGVectorStore) AddDocuments(ctx context.Context, docs []schema.Document, options ...vectorstores.Option) ([]string, error) {
	vs.logger.Debug("Adding documents to vector store",
		slog.Int("count", len(docs)),
		slog.String("collection", vs.collection))

	// Parse options
	opts := &vectorstores.Options{}
	for _, opt := range options {
		opt(opts)
	}

	var documentIDs []string

	for i, doc := range docs {
		// Generate embedding for document content
		embedding, err := vs.embedder.EmbedQuery(ctx, doc.PageContent)
		if err != nil {
			vs.logger.Error("Failed to generate embedding for document",
				slog.Int("doc_index", i),
				slog.Any("error", err))
			return nil, fmt.Errorf("failed to generate embedding for document %d: %w", i, err)
		}

		// Prepare metadata
		metadata := make(map[string]interface{})
		for key, value := range doc.Metadata {
			metadata[key] = value
		}

		// Add document metadata
		metadata["page_content"] = doc.PageContent
		metadata["collection"] = vs.collection

		// Generate document ID if not provided
		docID := fmt.Sprintf("doc_%d_%s", i, vs.collection)
		if len(documentIDs) > i {
			docID = documentIDs[i]
		}

		// Convert float32 slice to float64 slice for database
		embeddingFloat64 := make([]float64, len(embedding))
		for j, v := range embedding {
			embeddingFloat64[j] = float64(v)
		}

		// Store in database
		_, err = vs.dbClient.CreateEmbedding(
			ctx,
			vs.collection,
			docID,
			doc.PageContent,
			embeddingFloat64,
			metadata,
		)
		if err != nil {
			vs.logger.Error("Failed to store document embedding",
				slog.String("doc_id", docID),
				slog.Any("error", err))
			return nil, fmt.Errorf("failed to store document %s: %w", docID, err)
		}

		documentIDs = append(documentIDs, docID)
	}

	vs.logger.Info("Documents added to vector store",
		slog.Int("count", len(docs)),
		slog.String("collection", vs.collection))

	return documentIDs, nil
}

// SimilaritySearch performs similarity search and returns documents
func (vs *PGVectorStore) SimilaritySearch(ctx context.Context, query string, numDocuments int, options ...vectorstores.Option) ([]schema.Document, error) {
	vs.logger.Debug("Performing similarity search",
		slog.String("query", query),
		slog.Int("num_documents", numDocuments),
		slog.String("collection", vs.collection))

	// Parse options
	opts := &vectorstores.Options{}
	for _, opt := range options {
		opt(opts)
	}

	// Generate query embedding
	queryEmbedding, err := vs.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Convert query embedding to float64
	queryEmbeddingFloat64 := make([]float64, len(queryEmbedding))
	for i, v := range queryEmbedding {
		queryEmbeddingFloat64[i] = float64(v)
	}

	// Set default similarity threshold
	similarity := 0.7
	if opts.Filters != nil {
		if filters, ok := opts.Filters.(map[string]interface{}); ok {
			if sim, ok := filters["similarity_threshold"].(float64); ok {
				similarity = sim
			}
		}
	}

	// Search similar embeddings
	results, err := vs.dbClient.SearchSimilarEmbeddings(
		ctx,
		queryEmbeddingFloat64,
		vs.collection,
		numDocuments,
		similarity,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar embeddings: %w", err)
	}

	// Convert results to schema.Document
	documents := make([]schema.Document, 0, len(results))
	for _, result := range results {
		doc := schema.Document{
			PageContent: result.Record.ContentText,
			Metadata:    make(map[string]any),
		}

		// Copy metadata, excluding internal fields
		for key, value := range result.Record.Metadata {
			if key != "page_content" && key != "collection" {
				doc.Metadata[key] = value
			}
		}

		// Add similarity score to metadata
		doc.Metadata["similarity_score"] = result.Similarity

		documents = append(documents, doc)
	}

	vs.logger.Debug("Similarity search completed",
		slog.String("query", query),
		slog.Int("results", len(documents)))

	return documents, nil
}

// SimilaritySearchWithScore performs similarity search and returns documents with scores
func (vs *PGVectorStore) SimilaritySearchWithScore(ctx context.Context, query string, numDocuments int, options ...vectorstores.Option) ([]schema.Document, []float32, error) {
	documents, err := vs.SimilaritySearch(ctx, query, numDocuments, options...)
	if err != nil {
		return nil, nil, err
	}

	// Extract scores from metadata
	scores := make([]float32, len(documents))
	for i, doc := range documents {
		if score, ok := doc.Metadata["similarity_score"].(float64); ok {
			scores[i] = float32(score)
		}
	}

	return documents, scores, nil
}

// RemoveCollection removes all documents from a collection
func (vs *PGVectorStore) RemoveCollection(ctx context.Context, collection string) error {
	vs.logger.Info("Removing collection",
		slog.String("collection", collection))

	// TODO: Implement collection removal in the database client
	// This would require extending the SQLCClient with a method to delete embeddings by collection
	vs.logger.Warn("Collection removal not yet implemented",
		slog.String("collection", collection))

	return nil
}

// GetNumDocuments returns the number of documents in the store
func (vs *PGVectorStore) GetNumDocuments(ctx context.Context) (int, error) {
	// TODO: Implement document counting in the database client
	vs.logger.Debug("Getting document count",
		slog.String("collection", vs.collection))

	// Return 0 for now - this would need a proper implementation
	return 0, nil
}

// Delete removes documents by IDs
func (vs *PGVectorStore) Delete(ctx context.Context, ids []string) error {
	vs.logger.Info("Deleting documents",
		slog.Any("ids", ids),
		slog.String("collection", vs.collection))

	// TODO: Implement document deletion in the database client
	vs.logger.Warn("Document deletion not yet implemented",
		slog.Any("ids", ids))

	return nil
}

// Close closes the vector store
func (vs *PGVectorStore) Close() error {
	vs.logger.Debug("Closing vector store",
		slog.String("collection", vs.collection))

	// Vector store doesn't maintain persistent connections
	return nil
}

// AsRetriever returns the vector store as a retriever
func (vs *PGVectorStore) AsRetriever(ctx context.Context, options ...vectorstores.Option) schema.Retriever {
	return &PGVectorRetriever{
		vectorStore: vs,
		options:     options,
	}
}

// PGVectorRetriever implements the schema.Retriever interface
type PGVectorRetriever struct {
	vectorStore *PGVectorStore
	options     []vectorstores.Option
}

// GetRelevantDocuments retrieves relevant documents for a query
func (r *PGVectorRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	// Default to 10 documents if not specified
	numDocs := 10

	// Parse options to get number of documents
	opts := &vectorstores.Options{}
	for _, opt := range r.options {
		opt(opts)
	}

	if opts.ScoreThreshold != 0 {
		// Use score threshold if provided
		return r.vectorStore.SimilaritySearch(ctx, query, numDocs, r.options...)
	}

	return r.vectorStore.SimilaritySearch(ctx, query, numDocs, r.options...)
}

// Verify interface compliance at compile time
var (
	_ vectorstores.VectorStore = (*PGVectorStore)(nil)
	_ schema.Retriever         = (*PGVectorRetriever)(nil)
)
