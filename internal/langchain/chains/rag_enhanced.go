package chains

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"

	"github.com/koopa0/assistant/internal/config"
	//"github.com/koopa0/assistant/internal/langchain/documentloader" // Used via embedded RAGChain.docProcessor
)

// EnhancedRAGChain provides advanced RAG capabilities with document processing
type EnhancedRAGChain struct {
	*RAGChain
	documentCache map[string][]schema.Document
}

// NewEnhancedRAGChain creates a new enhanced RAG chain
func NewEnhancedRAGChain(llm llms.Model, vectorStore vectorstores.VectorStore, embedder embeddings.Embedder, config config.LangChain, logger *slog.Logger) *EnhancedRAGChain {
	ragChain := NewRAGChainWithComponents(llm, vectorStore, embedder, config, logger)

	eachain := &EnhancedRAGChain{
		RAGChain:      ragChain,
		documentCache: make(map[string][]schema.Document),
	}

	// Ensure docProcessor is accessible (uses documentloader package)
	_ = eachain.docProcessor

	return eachain
}

// IngestDocument processes and ingests a single document into the knowledge base
func (erc *EnhancedRAGChain) IngestDocument(ctx context.Context, filePath string, metadata map[string]any) error {
	erc.logger.Info("Starting document ingestion",
		slog.String("file_path", filePath))

	if erc.vectorStore == nil {
		return fmt.Errorf("vector store not configured")
	}

	// Process the document
	docs, err := erc.docProcessor.ProcessFile(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to process document: %w", err)
	}

	// Add additional metadata
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]any)
		}

		// Add custom metadata
		for key, value := range metadata {
			docs[i].Metadata[key] = value
		}

		// Add ingestion metadata
		docs[i].Metadata["ingested_at"] = time.Now().Format(time.RFC3339)
		docs[i].Metadata["ingestion_source"] = "enhanced_rag_chain"
	}

	// Store in vector store
	ids, err := erc.vectorStore.AddDocuments(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to store documents in vector store: %w", err)
	}

	// Cache the documents
	erc.documentCache[filePath] = docs

	erc.logger.Info("Document ingestion completed",
		slog.String("file_path", filePath),
		slog.Int("chunks_created", len(docs)),
		slog.Any("document_ids", ids))

	return nil
}

// IngestDirectory processes and ingests all documents in a directory
func (erc *EnhancedRAGChain) IngestDirectory(ctx context.Context, dirPath string, recursive bool, extensions []string, metadata map[string]any) error {
	erc.logger.Info("Starting directory ingestion",
		slog.String("dir_path", dirPath),
		slog.Bool("recursive", recursive))

	if erc.vectorStore == nil {
		return fmt.Errorf("vector store not configured")
	}

	// Process all documents in the directory
	docs, err := erc.docProcessor.ProcessDirectory(ctx, dirPath, recursive, extensions)
	if err != nil {
		return fmt.Errorf("failed to process directory: %w", err)
	}

	// Add metadata to all documents
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]any)
		}

		// Add custom metadata
		for key, value := range metadata {
			docs[i].Metadata[key] = value
		}

		// Add ingestion metadata
		docs[i].Metadata["ingested_at"] = time.Now().Format(time.RFC3339)
		docs[i].Metadata["ingestion_source"] = "enhanced_rag_chain"
		docs[i].Metadata["directory_source"] = dirPath
	}

	// Store in vector store in batches
	batchSize := 50
	totalIngested := 0

	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}

		batch := docs[i:end]
		ids, err := erc.vectorStore.AddDocuments(ctx, batch)
		if err != nil {
			erc.logger.Error("Failed to ingest document batch",
				slog.Int("batch_start", i),
				slog.Int("batch_end", end),
				slog.Any("error", err))
			continue
		}

		totalIngested += len(batch)
		erc.logger.Debug("Document batch ingested",
			slog.Int("batch_size", len(batch)),
			slog.Any("document_ids", ids))
	}

	// Cache the documents
	erc.documentCache[dirPath] = docs

	erc.logger.Info("Directory ingestion completed",
		slog.String("dir_path", dirPath),
		slog.Int("total_documents", len(docs)),
		slog.Int("ingested_documents", totalIngested))

	return nil
}

// IngestFromString processes and ingests text content
func (erc *EnhancedRAGChain) IngestFromString(ctx context.Context, content string, metadata map[string]any) error {
	erc.logger.Debug("Starting string content ingestion",
		slog.Int("content_length", len(content)))

	if erc.vectorStore == nil {
		return fmt.Errorf("vector store not configured")
	}

	// Process the string content
	docs, err := erc.docProcessor.ProcessString(ctx, content, metadata)
	if err != nil {
		return fmt.Errorf("failed to process string content: %w", err)
	}

	// Add ingestion metadata
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]any)
		}
		docs[i].Metadata["ingested_at"] = time.Now().Format(time.RFC3339)
		docs[i].Metadata["ingestion_source"] = "enhanced_rag_chain"
	}

	// Store in vector store
	ids, err := erc.vectorStore.AddDocuments(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to store string content in vector store: %w", err)
	}

	erc.logger.Info("String content ingestion completed",
		slog.Int("chunks_created", len(docs)),
		slog.Any("document_ids", ids))

	return nil
}

// IngestFromURL processes and ingests content from a URL
func (erc *EnhancedRAGChain) IngestFromURL(ctx context.Context, url string, metadata map[string]any) error {
	erc.logger.Info("Starting URL content ingestion",
		slog.String("url", url))

	if erc.vectorStore == nil {
		return fmt.Errorf("vector store not configured")
	}

	// Load content from URL
	docs, err := erc.docProcessor.LoadFromURL(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to load URL content: %w", err)
	}

	// Split documents into chunks
	docs, err = erc.docProcessor.SplitDocuments(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to split URL documents: %w", err)
	}

	// Add metadata
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]any)
		}

		// Add custom metadata
		for key, value := range metadata {
			docs[i].Metadata[key] = value
		}

		// Add ingestion metadata
		docs[i].Metadata["ingested_at"] = time.Now().Format(time.RFC3339)
		docs[i].Metadata["ingestion_source"] = "enhanced_rag_chain"
		docs[i].Metadata["url_source"] = url
	}

	// Store in vector store
	ids, err := erc.vectorStore.AddDocuments(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to store URL content in vector store: %w", err)
	}

	// Cache the documents
	erc.documentCache[url] = docs

	erc.logger.Info("URL content ingestion completed",
		slog.String("url", url),
		slog.Int("chunks_created", len(docs)),
		slog.Any("document_ids", ids))

	return nil
}

// QueryWithSources performs RAG query and returns sources
func (erc *EnhancedRAGChain) QueryWithSources(ctx context.Context, query string, options map[string]interface{}) (*RAGQueryResult, error) {
	erc.logger.Debug("Performing RAG query with sources",
		slog.String("query", query))

	if erc.retriever == nil {
		return nil, fmt.Errorf("retriever not configured")
	}

	// Get relevant documents
	docs, err := erc.retriever.GetRelevantDocuments(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

	// If we have the native LangChain RAG chain, use it
	if erc.langchainRAGChain != nil {
		return erc.queryWithLangChainRAG(ctx, query, docs, options)
	}

	// Fallback to custom RAG implementation
	return erc.queryWithCustomRAG(ctx, query, docs, options)
}

// queryWithLangChainRAG uses the native LangChain RAG chain
func (erc *EnhancedRAGChain) queryWithLangChainRAG(ctx context.Context, query string, docs []schema.Document, options map[string]interface{}) (*RAGQueryResult, error) {
	// Use native LangChain RAG chain
	result, err := erc.langchainRAGChain.Call(ctx, map[string]any{
		"query": query,
	})
	if err != nil {
		return nil, fmt.Errorf("LangChain RAG call failed: %w", err)
	}

	// Extract answer from result
	answer := ""
	if resultStr, ok := result["text"].(string); ok {
		answer = resultStr
	} else if resultStr, ok := result["answer"].(string); ok {
		answer = resultStr
	} else {
		answer = fmt.Sprintf("%v", result)
	}

	// Build sources from retrieved documents
	sources := make([]DocumentSource, 0, len(docs))
	for i, doc := range docs {
		source := DocumentSource{
			ID:       fmt.Sprintf("doc_%d", i),
			Content:  doc.PageContent,
			Metadata: doc.Metadata,
			Score:    0.0, // LangChain doesn't return scores directly
		}

		// Extract source information from metadata
		if sourceFile, ok := doc.Metadata["source"].(string); ok {
			source.Source = sourceFile
		}
		if fileName, ok := doc.Metadata["file_name"].(string); ok {
			source.Title = fileName
		}

		sources = append(sources, source)
	}

	return &RAGQueryResult{
		Query:     query,
		Answer:    answer,
		Sources:   sources,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"method":         "langchain_rag",
			"documents_used": len(docs),
			"total_sources":  len(sources),
		},
	}, nil
}

// queryWithCustomRAG uses the custom RAG implementation
func (erc *EnhancedRAGChain) queryWithCustomRAG(ctx context.Context, query string, docs []schema.Document, options map[string]interface{}) (*RAGQueryResult, error) {
	// Build context from documents
	contextBuilder := strings.Builder{}
	contextBuilder.WriteString("Based on the following information:\n\n")

	sources := make([]DocumentSource, 0, len(docs))
	for i, doc := range docs {
		contextBuilder.WriteString(fmt.Sprintf("Source %d:\n%s\n\n", i+1, doc.PageContent))

		source := DocumentSource{
			ID:       fmt.Sprintf("doc_%d", i),
			Content:  doc.PageContent,
			Metadata: doc.Metadata,
			Score:    0.0,
		}

		// Extract source information
		if sourceFile, ok := doc.Metadata["source"].(string); ok {
			source.Source = sourceFile
		}
		if fileName, ok := doc.Metadata["file_name"].(string); ok {
			source.Title = fileName
		}

		sources = append(sources, source)
	}

	// Build the prompt
	prompt := fmt.Sprintf(`%s

Question: %s

Please provide a comprehensive answer based on the information above. If the information doesn't contain relevant details, please indicate that and provide what you can.

Answer:`, contextBuilder.String(), query)

	// Generate response using LLM
	response, err := erc.llm.Call(ctx, prompt, llms.WithMaxTokens(2000))
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	return &RAGQueryResult{
		Query:     query,
		Answer:    response,
		Sources:   sources,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"method":         "custom_rag",
			"documents_used": len(docs),
			"total_sources":  len(sources),
			"context_length": contextBuilder.Len(),
		},
	}, nil
}

// GetCachedDocuments returns cached documents for a source
func (erc *EnhancedRAGChain) GetCachedDocuments(source string) ([]schema.Document, bool) {
	docs, exists := erc.documentCache[source]
	return docs, exists
}

// ClearCache clears the document cache
func (erc *EnhancedRAGChain) ClearCache() {
	erc.documentCache = make(map[string][]schema.Document)
	erc.logger.Debug("Document cache cleared")
}

// GetCacheStats returns cache statistics
func (erc *EnhancedRAGChain) GetCacheStats() map[string]interface{} {
	totalDocs := 0
	for _, docs := range erc.documentCache {
		totalDocs += len(docs)
	}

	return map[string]interface{}{
		"cached_sources":    len(erc.documentCache),
		"total_cached_docs": totalDocs,
	}
}

// RAGQueryResult represents the result of a RAG query
type RAGQueryResult struct {
	Query     string                 `json:"query"`
	Answer    string                 `json:"answer"`
	Sources   []DocumentSource       `json:"sources"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// DocumentSource represents a source document used in RAG
type DocumentSource struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title,omitempty"`
	Source   string                 `json:"source,omitempty"`
	Content  string                 `json:"content"`
	Score    float64                `json:"score"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}
