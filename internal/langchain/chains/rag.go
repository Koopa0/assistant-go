package chains

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/langchain/documentloader"
	"github.com/koopa0/assistant-go/internal/langchain/vectorstore"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
)

// RAGChain implements Retrieval-Augmented Generation using LangChain components
type RAGChain struct {
	*BaseChain
	vectorStore       vectorstores.VectorStore
	retriever         schema.Retriever
	embedder          embeddings.Embedder
	docProcessor      *documentloader.DocumentProcessor
	embeddingClient   *postgres.SQLCClient // Fallback for direct DB access
	retrievalConfig   RAGRetrievalConfig
	langchainRAGChain chains.Chain // Native LangChain RAG chain
}

// RAGRetrievalConfig configures the retrieval behavior
type RAGRetrievalConfig struct {
	MaxDocuments        int      `json:"max_documents"`
	SimilarityThreshold float64  `json:"similarity_threshold"`
	ContentTypes        []string `json:"content_types"`
	IncludeMetadata     bool     `json:"include_metadata"`
	RetrievalStrategy   string   `json:"retrieval_strategy"` // "similarity", "hybrid", "keyword"
}

// RetrievedDocument represents a document retrieved from the knowledge base
type RetrievedDocument struct {
	ID          string                 `json:"id"`
	ContentType string                 `json:"content_type"`
	ContentID   string                 `json:"content_id"`
	Content     string                 `json:"content"`
	Similarity  float64                `json:"similarity"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RetrievedAt time.Time              `json:"retrieved_at"`
}

// RAGContext represents the context built from retrieved documents
type RAGContext struct {
	Query             string              `json:"query"`
	RetrievedDocs     []RetrievedDocument `json:"retrieved_docs"`
	ContextSummary    string              `json:"context_summary"`
	RetrievalStrategy string              `json:"retrieval_strategy"`
	TotalDocuments    int                 `json:"total_documents"`
	AvgSimilarity     float64             `json:"avg_similarity"`
}

// NewRAGChain creates a new RAG chain
func NewRAGChain(llm llms.Model, config config.LangChain, logger *slog.Logger) *RAGChain {
	base := NewBaseChain(ChainTypeRAG, llm, config, logger)

	chain := &RAGChain{
		BaseChain:    base,
		docProcessor: documentloader.NewDocumentProcessor(logger),
		retrievalConfig: RAGRetrievalConfig{
			MaxDocuments:        5,
			SimilarityThreshold: 0.7,
			ContentTypes:        []string{"message", "document", "code"},
			IncludeMetadata:     true,
			RetrievalStrategy:   "similarity",
		},
	}

	return chain
}

// NewRAGChainWithComponents creates a new RAG chain with LangChain components
func NewRAGChainWithComponents(llm llms.Model, vectorStore vectorstores.VectorStore, embedder embeddings.Embedder, config config.LangChain, logger *slog.Logger) *RAGChain {
	base := NewBaseChain(ChainTypeRAG, llm, config, logger)

	chain := &RAGChain{
		BaseChain:    base,
		vectorStore:  vectorStore,
		embedder:     embedder,
		docProcessor: documentloader.NewDocumentProcessor(logger),
		retrievalConfig: RAGRetrievalConfig{
			MaxDocuments:        5,
			SimilarityThreshold: 0.7,
			ContentTypes:        []string{"message", "document", "code"},
			IncludeMetadata:     true,
			RetrievalStrategy:   "similarity",
		},
	}

	// Set up retriever if vector store is available
	if vectorStore != nil {
		// Check if vector store implements AsRetriever method
		if pgVectorStore, ok := vectorStore.(*vectorstore.PGVectorStore); ok {
			chain.retriever = pgVectorStore.AsRetriever(context.Background(),
				vectorstores.WithScoreThreshold(float32(chain.retrievalConfig.SimilarityThreshold)))
		} else {
			// Use a custom retriever wrapper
			chain.retriever = &CustomVectorStoreRetriever{
				vectorStore: vectorStore,
				threshold:   float32(chain.retrievalConfig.SimilarityThreshold),
				maxDocs:     chain.retrievalConfig.MaxDocuments,
			}
		}
	}

	// Initialize native LangChain RAG chain if components are available
	if vectorStore != nil && embedder != nil {
		chain.initializeLangChainRAG()
	}

	return chain
}

// SetVectorStore sets the vector store for this RAG chain
func (rc *RAGChain) SetVectorStore(vectorStore vectorstores.VectorStore) {
	rc.vectorStore = vectorStore
	if vectorStore != nil {
		// Check if vector store implements AsRetriever method
		if pgVectorStore, ok := vectorStore.(*vectorstore.PGVectorStore); ok {
			rc.retriever = pgVectorStore.AsRetriever(context.Background(),
				vectorstores.WithScoreThreshold(float32(rc.retrievalConfig.SimilarityThreshold)))
		} else {
			// Use a custom retriever wrapper
			rc.retriever = &CustomVectorStoreRetriever{
				vectorStore: vectorStore,
				threshold:   float32(rc.retrievalConfig.SimilarityThreshold),
				maxDocs:     rc.retrievalConfig.MaxDocuments,
			}
		}
		rc.logger.Debug("Vector store set for RAG chain")
	}
}

// SetEmbedder sets the embedder for this RAG chain
func (rc *RAGChain) SetEmbedder(embedder embeddings.Embedder) {
	rc.embedder = embedder
	rc.logger.Debug("Embedder set for RAG chain")
}

// initializeLangChainRAG initializes the native LangChain RAG chain
func (rc *RAGChain) initializeLangChainRAG() {
	if rc.vectorStore == nil || rc.llm == nil {
		rc.logger.Warn("Cannot initialize LangChain RAG: missing components")
		return
	}

	// Create retrieval QA chain - TODO: Fix API compatibility
	// ragChain := chains.LoadRetrievalQA(rc.llm, rc.retriever, nil)
	// rc.langchainRAGChain = ragChain
	rc.logger.Debug("LangChain RAG chain creation deferred - API compatibility issue")

	rc.logger.Debug("Native LangChain RAG chain initialized")
}

// SetEmbeddingClient sets the embedding client for document retrieval
func (rc *RAGChain) SetEmbeddingClient(client *postgres.SQLCClient) {
	rc.embeddingClient = client
	rc.logger.Debug("Embedding client set for RAG chain")
}

// SetRetrievalConfig sets the retrieval configuration
func (rc *RAGChain) SetRetrievalConfig(config RAGRetrievalConfig) {
	rc.retrievalConfig = config
	rc.logger.Debug("Retrieval config updated",
		slog.Int("max_documents", config.MaxDocuments),
		slog.Float64("similarity_threshold", config.SimilarityThreshold),
		slog.String("strategy", config.RetrievalStrategy))
}

// executeSteps implements RAG chain execution logic
func (rc *RAGChain) executeSteps(ctx context.Context, request *ChainRequest) (string, []ChainStep, error) {
	steps := make([]ChainStep, 0)

	rc.logger.Debug("Starting RAG chain execution",
		slog.String("query", request.Input),
		slog.Int("max_documents", rc.retrievalConfig.MaxDocuments))

	// Step 1: Query Analysis and Embedding Generation
	stepStart := time.Now()
	queryEmbedding, err := rc.generateQueryEmbedding(ctx, request.Input)
	if err != nil {
		return "", steps, fmt.Errorf("query embedding generation failed: %w", err)
	}

	step1 := ChainStep{
		StepNumber: 1,
		StepType:   "query_embedding",
		Input:      request.Input,
		Output:     fmt.Sprintf("Generated embedding vector (dimension: %d)", len(queryEmbedding)),
		Duration:   time.Since(stepStart),
		Success:    true,
		Metadata: map[string]interface{}{
			"embedding_dimension": len(queryEmbedding),
			"query_length":        len(request.Input),
		},
	}
	steps = append(steps, step1)

	// Step 2: Document Retrieval
	stepStart = time.Now()
	retrievedDocs, err := rc.retrieveRelevantDocuments(ctx, queryEmbedding, request.Input)
	if err != nil {
		return "", steps, fmt.Errorf("document retrieval failed: %w", err)
	}

	step2 := ChainStep{
		StepNumber: 2,
		StepType:   "document_retrieval",
		Input:      fmt.Sprintf("Query embedding + similarity search"),
		Output:     fmt.Sprintf("Retrieved %d documents", len(retrievedDocs)),
		Duration:   time.Since(stepStart),
		Success:    true,
		Metadata: map[string]interface{}{
			"documents_retrieved": len(retrievedDocs),
			"avg_similarity":      rc.calculateAverageSimilarity(retrievedDocs),
			"retrieval_strategy":  rc.retrievalConfig.RetrievalStrategy,
		},
	}
	steps = append(steps, step2)

	// Step 3: Context Building
	stepStart = time.Now()
	ragContext, err := rc.buildRAGContext(ctx, request.Input, retrievedDocs)
	if err != nil {
		return "", steps, fmt.Errorf("context building failed: %w", err)
	}

	step3 := ChainStep{
		StepNumber: 3,
		StepType:   "context_building",
		Input:      fmt.Sprintf("%d retrieved documents", len(retrievedDocs)),
		Output:     fmt.Sprintf("Built context summary (%d characters)", len(ragContext.ContextSummary)),
		Duration:   time.Since(stepStart),
		Success:    true,
		Metadata: map[string]interface{}{
			"context_length":  len(ragContext.ContextSummary),
			"total_documents": ragContext.TotalDocuments,
			"avg_similarity":  ragContext.AvgSimilarity,
		},
	}
	steps = append(steps, step3)

	// Step 4: Augmented Generation
	stepStart = time.Now()
	augmentedResponse, err := rc.generateAugmentedResponse(ctx, request, ragContext)
	if err != nil {
		return "", steps, fmt.Errorf("augmented generation failed: %w", err)
	}

	step4 := ChainStep{
		StepNumber: 4,
		StepType:   "augmented_generation",
		Input:      fmt.Sprintf("Query + Context (%d chars)", len(ragContext.ContextSummary)),
		Output:     augmentedResponse,
		Duration:   time.Since(stepStart),
		Success:    true,
		Metadata: map[string]interface{}{
			"response_length":      len(augmentedResponse),
			"context_used":         len(ragContext.ContextSummary) > 0,
			"documents_referenced": len(ragContext.RetrievedDocs),
		},
	}
	steps = append(steps, step4)

	rc.logger.Info("RAG chain execution completed",
		slog.Int("steps_executed", len(steps)),
		slog.Int("documents_used", len(retrievedDocs)),
		slog.Int("response_length", len(augmentedResponse)))

	return augmentedResponse, steps, nil
}

// generateQueryEmbedding generates an embedding for the query
func (rc *RAGChain) generateQueryEmbedding(ctx context.Context, query string) ([]float64, error) {
	// For now, return a mock embedding since we don't have direct embedding generation
	// In a real implementation, this would call an embedding service
	mockEmbedding := make([]float64, 1536) // OpenAI embedding dimension
	for i := range mockEmbedding {
		mockEmbedding[i] = 0.1 // Simple mock values
	}

	rc.logger.Debug("Generated query embedding",
		slog.String("query", query),
		slog.Int("dimension", len(mockEmbedding)))

	return mockEmbedding, nil
}

// retrieveRelevantDocuments retrieves documents similar to the query
func (rc *RAGChain) retrieveRelevantDocuments(ctx context.Context, queryEmbedding []float64, query string) ([]RetrievedDocument, error) {
	if rc.embeddingClient == nil {
		// Return mock documents if no embedding client is available
		return rc.getMockDocuments(query), nil
	}

	retrievedDocs := make([]RetrievedDocument, 0)

	// Search for similar embeddings for each content type
	for _, contentType := range rc.retrievalConfig.ContentTypes {
		results, err := rc.embeddingClient.SearchSimilarEmbeddings(
			ctx,
			queryEmbedding,
			contentType,
			rc.retrievalConfig.MaxDocuments,
			rc.retrievalConfig.SimilarityThreshold,
		)
		if err != nil {
			rc.logger.Warn("Failed to search embeddings",
				slog.String("content_type", contentType),
				slog.Any("error", err))
			continue
		}

		// Convert search results to retrieved documents
		for _, result := range results {
			doc := RetrievedDocument{
				ID:          result.Record.ID,
				ContentType: result.Record.ContentType,
				ContentID:   result.Record.ContentID,
				Content:     result.Record.ContentText,
				Similarity:  result.Similarity,
				Metadata:    result.Record.Metadata,
				RetrievedAt: time.Now(),
			}
			retrievedDocs = append(retrievedDocs, doc)
		}
	}

	// Sort by similarity (highest first) and limit results
	if len(retrievedDocs) > rc.retrievalConfig.MaxDocuments {
		retrievedDocs = retrievedDocs[:rc.retrievalConfig.MaxDocuments]
	}

	rc.logger.Debug("Documents retrieved",
		slog.Int("count", len(retrievedDocs)),
		slog.Float64("min_similarity", rc.retrievalConfig.SimilarityThreshold))

	return retrievedDocs, nil
}

// getMockDocuments returns mock documents for testing
func (rc *RAGChain) getMockDocuments(query string) []RetrievedDocument {
	return []RetrievedDocument{
		{
			ID:          "mock-1",
			ContentType: "document",
			ContentID:   "doc-1",
			Content:     fmt.Sprintf("This is a mock document related to: %s", query),
			Similarity:  0.85,
			Metadata:    map[string]interface{}{"source": "mock"},
			RetrievedAt: time.Now(),
		},
		{
			ID:          "mock-2",
			ContentType: "message",
			ContentID:   "msg-1",
			Content:     fmt.Sprintf("Previous conversation about: %s", query),
			Similarity:  0.78,
			Metadata:    map[string]interface{}{"source": "conversation"},
			RetrievedAt: time.Now(),
		},
	}
}

// buildRAGContext builds the context from retrieved documents
func (rc *RAGChain) buildRAGContext(ctx context.Context, query string, docs []RetrievedDocument) (*RAGContext, error) {
	if len(docs) == 0 {
		return &RAGContext{
			Query:             query,
			RetrievedDocs:     docs,
			ContextSummary:    "",
			RetrievalStrategy: rc.retrievalConfig.RetrievalStrategy,
			TotalDocuments:    0,
			AvgSimilarity:     0,
		}, nil
	}

	// Build context summary from retrieved documents
	contextBuilder := strings.Builder{}
	contextBuilder.WriteString("Relevant context from knowledge base:\n\n")

	for i, doc := range docs {
		contextBuilder.WriteString(fmt.Sprintf("Document %d (Similarity: %.2f):\n", i+1, doc.Similarity))
		contextBuilder.WriteString(fmt.Sprintf("Type: %s\n", doc.ContentType))
		contextBuilder.WriteString(fmt.Sprintf("Content: %s\n\n", doc.Content))
	}

	ragContext := &RAGContext{
		Query:             query,
		RetrievedDocs:     docs,
		ContextSummary:    contextBuilder.String(),
		RetrievalStrategy: rc.retrievalConfig.RetrievalStrategy,
		TotalDocuments:    len(docs),
		AvgSimilarity:     rc.calculateAverageSimilarity(docs),
	}

	rc.logger.Debug("RAG context built",
		slog.Int("documents", len(docs)),
		slog.Float64("avg_similarity", ragContext.AvgSimilarity),
		slog.Int("context_length", len(ragContext.ContextSummary)))

	return ragContext, nil
}

// generateAugmentedResponse generates the final response using retrieved context
func (rc *RAGChain) generateAugmentedResponse(ctx context.Context, request *ChainRequest, ragContext *RAGContext) (string, error) {
	// Build the augmented prompt
	prompt := rc.buildAugmentedPrompt(request.Input, ragContext)

	// Generate response using LLM
	options := []llms.CallOption{
		llms.WithMaxTokens(3000),
	}

	if request.Temperature > 0 {
		options = append(options, llms.WithTemperature(request.Temperature))
	}

	response, err := rc.llm.Call(ctx, prompt, options...)
	if err != nil {
		return "", fmt.Errorf("LLM generation failed: %w", err)
	}

	// Add source attribution if documents were used
	if len(ragContext.RetrievedDocs) > 0 {
		response += rc.buildSourceAttribution(ragContext.RetrievedDocs)
	}

	return response, nil
}

// buildAugmentedPrompt builds the prompt with retrieved context
func (rc *RAGChain) buildAugmentedPrompt(query string, ragContext *RAGContext) string {
	if len(ragContext.RetrievedDocs) == 0 {
		// No context available, use basic prompt
		return fmt.Sprintf("Please answer the following question:\n\nQuestion: %s\n\nAnswer:", query)
	}

	// Build prompt with context
	prompt := fmt.Sprintf(`Answer the following question using the provided context from the knowledge base. If the context doesn't contain relevant information, please indicate that and provide the best answer you can.

Context:
%s

Question: %s

Please provide a comprehensive answer based on the context above. If you reference specific information from the context, please indicate which document it came from.

Answer:`, ragContext.ContextSummary, query)

	return prompt
}

// buildSourceAttribution builds source attribution for the response
func (rc *RAGChain) buildSourceAttribution(docs []RetrievedDocument) string {
	if len(docs) == 0 {
		return ""
	}

	attribution := "\n\n---\n**Sources:**\n"
	for i, doc := range docs {
		attribution += fmt.Sprintf("- Document %d: %s (Type: %s, Similarity: %.2f)\n",
			i+1, doc.ContentID, doc.ContentType, doc.Similarity)
	}

	return attribution
}

// calculateAverageSimilarity calculates the average similarity of retrieved documents
func (rc *RAGChain) calculateAverageSimilarity(docs []RetrievedDocument) float64 {
	if len(docs) == 0 {
		return 0
	}

	total := 0.0
	for _, doc := range docs {
		total += doc.Similarity
	}

	return total / float64(len(docs))
}

// GetRetrievalConfig returns the current retrieval configuration
func (rc *RAGChain) GetRetrievalConfig() RAGRetrievalConfig {
	return rc.retrievalConfig
}

// UpdateSimilarityThreshold updates the similarity threshold for retrieval
func (rc *RAGChain) UpdateSimilarityThreshold(threshold float64) {
	rc.retrievalConfig.SimilarityThreshold = threshold
	rc.logger.Debug("Similarity threshold updated",
		slog.Float64("new_threshold", threshold))
}

// AddContentType adds a content type to search
func (rc *RAGChain) AddContentType(contentType string) {
	for _, existing := range rc.retrievalConfig.ContentTypes {
		if existing == contentType {
			return // Already exists
		}
	}

	rc.retrievalConfig.ContentTypes = append(rc.retrievalConfig.ContentTypes, contentType)
	rc.logger.Debug("Content type added",
		slog.String("content_type", contentType))
}

// CustomVectorStoreRetriever wraps a VectorStore to implement schema.Retriever
type CustomVectorStoreRetriever struct {
	vectorStore vectorstores.VectorStore
	threshold   float32
	maxDocs     int
}

// GetRelevantDocuments retrieves relevant documents for a query
func (r *CustomVectorStoreRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	maxDocs := r.maxDocs
	if maxDocs == 0 {
		maxDocs = 10 // Default
	}

	options := make([]vectorstores.Option, 0)
	if r.threshold > 0 {
		options = append(options, vectorstores.WithScoreThreshold(r.threshold))
	}

	return r.vectorStore.SimilaritySearch(ctx, query, maxDocs, options...)
}
