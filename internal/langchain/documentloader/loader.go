package documentloader

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

// DocumentProcessor handles loading and processing documents for RAG
type DocumentProcessor struct {
	textSplitter textsplitter.TextSplitter
	logger       *slog.Logger
}

// NewDocumentProcessor creates a new document processor
func NewDocumentProcessor(logger *slog.Logger) *DocumentProcessor {
	// Create recursive character text splitter with reasonable defaults
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(1000),
		textsplitter.WithChunkOverlap(200),
	)

	return &DocumentProcessor{
		textSplitter: splitter,
		logger:       logger,
	}
}

// SetTextSplitter sets a custom text splitter
func (dp *DocumentProcessor) SetTextSplitter(splitter textsplitter.TextSplitter) {
	dp.textSplitter = splitter
}

// LoadFile loads a document from a file
func (dp *DocumentProcessor) LoadFile(ctx context.Context, filePath string) ([]schema.Document, error) {
	dp.logger.Debug("Loading document from file",
		slog.String("file_path", filePath))

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Determine file type and create appropriate loader
	ext := strings.ToLower(filepath.Ext(filePath))

	var loader documentloaders.Loader
	var err error

	switch ext {
	case ".txt", ".md", ".go", ".py", ".js", ".ts", ".yaml", ".yml", ".json":
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()
		loader = documentloaders.NewText(file)
	case ".pdf":
		// For PDF files, we'll need a PDF loader
		// For now, treat as text (this would need proper PDF library integration)
		dp.logger.Warn("PDF loading not fully implemented, treating as text")
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()
		loader = documentloaders.NewText(file)
	case ".html", ".htm":
		// For HTML files, we could use an HTML loader
		dp.logger.Warn("HTML loading not fully implemented, treating as text")
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()
		loader = documentloaders.NewText(file)
	default:
		// Try to load as text for unknown extensions
		dp.logger.Warn("Unknown file extension, attempting to load as text",
			slog.String("extension", ext))
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()
		loader = documentloaders.NewText(file)
	}

	// Load the document
	docs, err := loader.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load document: %w", err)
	}

	// Add file metadata
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]any)
		}
		docs[i].Metadata["source"] = filePath
		docs[i].Metadata["file_extension"] = ext
		docs[i].Metadata["file_name"] = filepath.Base(filePath)
	}

	dp.logger.Info("Document loaded successfully",
		slog.String("file_path", filePath),
		slog.Int("document_count", len(docs)))

	return docs, nil
}

// LoadDirectory loads all documents from a directory
func (dp *DocumentProcessor) LoadDirectory(ctx context.Context, dirPath string, recursive bool, extensions []string) ([]schema.Document, error) {
	dp.logger.Debug("Loading documents from directory",
		slog.String("dir_path", dirPath),
		slog.Bool("recursive", recursive),
		slog.Any("extensions", extensions))

	var allDocs []schema.Document

	// Set default extensions if none provided
	if len(extensions) == 0 {
		extensions = []string{".txt", ".md", ".go", ".py", ".js", ".ts", ".yaml", ".yml", ".json"}
	}

	// Create extension map for faster lookup
	extMap := make(map[string]bool)
	for _, ext := range extensions {
		extMap[strings.ToLower(ext)] = true
	}

	// Walk directory
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			dp.logger.Warn("Error walking directory",
				slog.String("path", path),
				slog.Any("error", err))
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			// If not recursive, skip subdirectories
			if !recursive && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file extension
		ext := strings.ToLower(filepath.Ext(path))
		if !extMap[ext] {
			return nil // Skip files with unsupported extensions
		}

		// Load the file
		docs, err := dp.LoadFile(ctx, path)
		if err != nil {
			dp.logger.Warn("Failed to load file",
				slog.String("path", path),
				slog.Any("error", err))
			return nil // Continue with other files
		}

		allDocs = append(allDocs, docs...)
		return nil
	}

	err := filepath.Walk(dirPath, walkFunc)
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	dp.logger.Info("Directory loading completed",
		slog.String("dir_path", dirPath),
		slog.Int("total_documents", len(allDocs)))

	return allDocs, nil
}

// LoadFromReader loads a document from an io.Reader
func (dp *DocumentProcessor) LoadFromReader(ctx context.Context, reader io.Reader, metadata map[string]any) ([]schema.Document, error) {
	dp.logger.Debug("Loading document from reader")

	// Read all content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Create document
	doc := schema.Document{
		PageContent: string(content),
		Metadata:    metadata,
	}

	if doc.Metadata == nil {
		doc.Metadata = make(map[string]any)
	}
	doc.Metadata["source"] = "reader"

	dp.logger.Info("Document loaded from reader",
		slog.Int("content_length", len(content)))

	return []schema.Document{doc}, nil
}

// LoadFromString loads a document from a string
func (dp *DocumentProcessor) LoadFromString(ctx context.Context, content string, metadata map[string]any) ([]schema.Document, error) {
	dp.logger.Debug("Loading document from string",
		slog.Int("content_length", len(content)))

	doc := schema.Document{
		PageContent: content,
		Metadata:    metadata,
	}

	if doc.Metadata == nil {
		doc.Metadata = make(map[string]any)
	}
	doc.Metadata["source"] = "string"

	return []schema.Document{doc}, nil
}

// LoadFromURL loads a document from a URL
func (dp *DocumentProcessor) LoadFromURL(ctx context.Context, url string) ([]schema.Document, error) {
	dp.logger.Debug("Loading document from URL",
		slog.String("url", url))

	// Use LangChain's URL loader - for now, just create a placeholder
	// TODO: Implement proper URL loading
	doc := schema.Document{
		PageContent: "URL content: " + url,
		Metadata:    make(map[string]any),
	}
	docs := []schema.Document{doc}

	// docs, err := loader.Load(ctx)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to load URL: %w", err)
	// }

	// Add URL metadata
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = make(map[string]any)
		}
		docs[i].Metadata["source"] = url
		docs[i].Metadata["source_type"] = "url"
	}

	dp.logger.Info("Document loaded from URL",
		slog.String("url", url),
		slog.Int("document_count", len(docs)))

	return docs, nil
}

// SplitDocuments splits documents into smaller chunks
func (dp *DocumentProcessor) SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error) {
	dp.logger.Debug("Splitting documents",
		slog.Int("document_count", len(docs)))

	if dp.textSplitter == nil {
		dp.logger.Warn("No text splitter configured, returning documents as-is")
		return docs, nil
	}

	var allChunks []schema.Document

	for i, doc := range docs {
		text := doc.PageContent
		texts, err := dp.textSplitter.SplitText(text)
		if err != nil {
			dp.logger.Warn("Failed to split document",
				slog.Int("doc_index", i),
				slog.Any("error", err))
			// Add original document if splitting fails
			allChunks = append(allChunks, doc)
			continue
		}

		// Convert split texts to documents
		for j, text := range texts {
			chunk := schema.Document{
				PageContent: text,
				Metadata:    make(map[string]any),
			}

			// Copy original metadata
			for key, value := range doc.Metadata {
				chunk.Metadata[key] = value
			}

			// Add chunk-specific metadata
			chunk.Metadata["chunk_index"] = j
			chunk.Metadata["total_chunks"] = len(texts)
			chunk.Metadata["parent_doc_index"] = i

			allChunks = append(allChunks, chunk)
		}
	}

	dp.logger.Info("Document splitting completed",
		slog.Int("original_docs", len(docs)),
		slog.Int("total_chunks", len(allChunks)))

	return allChunks, nil
}

// ProcessFile loads and splits a file into chunks ready for embedding
func (dp *DocumentProcessor) ProcessFile(ctx context.Context, filePath string) ([]schema.Document, error) {
	// Load the file
	docs, err := dp.LoadFile(ctx, filePath)
	if err != nil {
		return nil, err
	}

	// Split into chunks
	return dp.SplitDocuments(ctx, docs)
}

// ProcessDirectory loads and splits all files in a directory into chunks
func (dp *DocumentProcessor) ProcessDirectory(ctx context.Context, dirPath string, recursive bool, extensions []string) ([]schema.Document, error) {
	// Load all documents
	docs, err := dp.LoadDirectory(ctx, dirPath, recursive, extensions)
	if err != nil {
		return nil, err
	}

	// Split into chunks
	return dp.SplitDocuments(ctx, docs)
}

// ProcessString loads and splits a string into chunks
func (dp *DocumentProcessor) ProcessString(ctx context.Context, content string, metadata map[string]any) ([]schema.Document, error) {
	// Load the string
	docs, err := dp.LoadFromString(ctx, content, metadata)
	if err != nil {
		return nil, err
	}

	// Split into chunks
	return dp.SplitDocuments(ctx, docs)
}

// GetSupportedExtensions returns the list of supported file extensions
func (dp *DocumentProcessor) GetSupportedExtensions() []string {
	return []string{
		".txt", ".md", ".go", ".py", ".js", ".ts",
		".yaml", ".yml", ".json", ".html", ".htm",
		".pdf", // Note: PDF support requires additional implementation
	}
}

// CreateCustomSplitter creates a text splitter with custom parameters
func CreateCustomSplitter(chunkSize, chunkOverlap int, separators []string) textsplitter.TextSplitter {
	if len(separators) == 0 {
		// Use recursive character splitter with default separators
		return textsplitter.NewRecursiveCharacter(
			textsplitter.WithChunkSize(chunkSize),
			textsplitter.WithChunkOverlap(chunkOverlap),
		)
	}

	// Use character splitter with custom separators
	// TODO: Fix API compatibility with langchaingo text splitter
	return textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(chunkSize),
		textsplitter.WithChunkOverlap(chunkOverlap),
	)
}

// CreateMarkdownSplitter creates a splitter optimized for Markdown documents
func CreateMarkdownSplitter(chunkSize, chunkOverlap int) textsplitter.TextSplitter {
	// TODO: Fix API compatibility with langchaingo text splitter
	return textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(chunkSize),
		textsplitter.WithChunkOverlap(chunkOverlap),
	)
}

// CreateCodeSplitter creates a splitter optimized for code documents
func CreateCodeSplitter(language string, chunkSize, chunkOverlap int) textsplitter.TextSplitter {
	// Use recursive character splitter for code
	// In a more advanced implementation, you could use language-specific splitters
	return textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(chunkSize),
		textsplitter.WithChunkOverlap(chunkOverlap),
	)
}
