# Document Loader Package

The documentloader package provides comprehensive document loading and processing capabilities for the LangChain integration, enabling the assistant to ingest, process, and prepare various document types for RAG (Retrieval-Augmented Generation) workflows.

## Overview

This package handles the entire document ingestion pipeline, from loading raw documents to splitting them into optimized chunks for vector storage and semantic search. It supports multiple file formats, sources, and provides intelligent text splitting strategies.

## Core Components

### DocumentProcessor (`loader.go`)

The central component that orchestrates document loading, processing, and text splitting operations.

```go
// Create document processor
processor := documentloader.NewDocumentProcessor(logger)

// Load and process a single file
docs, err := processor.ProcessFile(ctx, "/path/to/document.md")

// Load and process an entire directory
docs, err := processor.ProcessDirectory(ctx, "/docs", true, []string{".md", ".txt"})

// Process string content directly
docs, err := processor.ProcessString(ctx, "Your content here", metadata)
```

## Supported File Types

The package supports a wide range of document formats:

### Text-based Formats
- **`.txt`** - Plain text files
- **`.md`** - Markdown documents
- **`.html`, `.htm`** - HTML documents
- **`.json`** - JSON files
- **`.yaml`, `.yml`** - YAML configuration files

### Code Files
- **`.go`** - Go source code
- **`.py`** - Python source code
- **`.js`** - JavaScript files
- **`.ts`** - TypeScript files

### Document Formats
- **`.pdf`** - PDF documents (basic support, extensible)

```go
// Check supported extensions
extensions := processor.GetSupportedExtensions()
fmt.Printf("Supported formats: %v\n", extensions)
```

## Loading Sources

### File Loading

Load individual files with automatic format detection:

```go
// Load a single file
docs, err := processor.LoadFile(ctx, "/path/to/document.md")
if err != nil {
    log.Fatal(err)
}

// Metadata is automatically added
for _, doc := range docs {
    fmt.Printf("Source: %s\n", doc.Metadata["source"])
    fmt.Printf("File name: %s\n", doc.Metadata["file_name"])
    fmt.Printf("Extension: %s\n", doc.Metadata["file_extension"])
}
```

### Directory Loading

Load all documents from a directory with flexible filtering:

```go
// Load all supported files from directory
docs, err := processor.LoadDirectory(ctx, "/docs", true, nil)

// Load specific file types recursively
docs, err := processor.LoadDirectory(ctx, "/src", true, []string{".go", ".md"})

// Load non-recursively
docs, err := processor.LoadDirectory(ctx, "/config", false, []string{".yaml", ".json"})
```

### String Content

Process content directly from strings:

```go
// Load from string with custom metadata
content := "This is important documentation about our API"
metadata := map[string]any{
    "type":     "api_docs",
    "version":  "1.0",
    "priority": "high",
}

docs, err := processor.LoadFromString(ctx, content, metadata)
```

### Reader Interface

Load from any `io.Reader` source:

```go
// Load from HTTP response, file, or any reader
resp, err := http.Get("https://example.com/docs")
defer resp.Body.Close()

metadata := map[string]any{
    "source_url": "https://example.com/docs",
    "fetched_at": time.Now(),
}

docs, err := processor.LoadFromReader(ctx, resp.Body, metadata)
```

### URL Loading

Load documents from web URLs (extensible):

```go
// Load from URL (basic implementation, extensible)
docs, err := processor.LoadFromURL(ctx, "https://example.com/documentation")

// Metadata includes source URL
for _, doc := range docs {
    fmt.Printf("Loaded from: %s\n", doc.Metadata["source"])
    fmt.Printf("Source type: %s\n", doc.Metadata["source_type"])
}
```

## Text Splitting Strategies

The package provides intelligent text splitting to optimize documents for vector search and retrieval.

### Default Recursive Character Splitter

```go
// Default splitter with sensible defaults
processor := documentloader.NewDocumentProcessor(logger)
// Uses: chunk_size=1000, chunk_overlap=200

// Customize the default splitter
customSplitter := documentloader.CreateCustomSplitter(
    1500,  // chunk_size
    300,   // chunk_overlap  
    nil,   // use default separators
)
processor.SetTextSplitter(customSplitter)
```

### Markdown-Optimized Splitter

```go
// Splitter optimized for Markdown documents
markdownSplitter := documentloader.CreateMarkdownSplitter(
    1200, // chunk_size
    250,  // chunk_overlap
)
processor.SetTextSplitter(markdownSplitter)

// Process markdown files
docs, err := processor.ProcessDirectory(ctx, "/docs", true, []string{".md"})
```

### Code-Optimized Splitter

```go
// Splitter optimized for source code
codeSplitter := documentloader.CreateCodeSplitter(
    "go",  // language hint
    800,   // chunk_size (smaller for code)
    150,   // chunk_overlap
)
processor.SetTextSplitter(codeSplitter)

// Process code files
docs, err := processor.ProcessDirectory(ctx, "/src", true, []string{".go", ".py"})
```

### Custom Splitter Configuration

```go
// Create highly customized splitter
customSplitter := documentloader.CreateCustomSplitter(
    2000,                    // chunk_size
    400,                     // chunk_overlap
    []string{"\n\n", "\n"}, // custom separators
)
processor.SetTextSplitter(customSplitter)
```

## Advanced Processing Workflows

### Batch Document Processing

```go
// Process multiple directories with different strategies
func ProcessDocumentationSuite(ctx context.Context, processor *documentloader.DocumentProcessor) ([]schema.Document, error) {
    var allDocs []schema.Document
    
    // Process API documentation with markdown splitter
    processor.SetTextSplitter(documentloader.CreateMarkdownSplitter(1200, 250))
    apiDocs, err := processor.ProcessDirectory(ctx, "/docs/api", true, []string{".md"})
    if err != nil {
        return nil, fmt.Errorf("failed to process API docs: %w", err)
    }
    allDocs = append(allDocs, apiDocs...)
    
    // Process source code with code splitter
    processor.SetTextSplitter(documentloader.CreateCodeSplitter("go", 800, 150))
    codeDocs, err := processor.ProcessDirectory(ctx, "/src", true, []string{".go"})
    if err != nil {
        return nil, fmt.Errorf("failed to process source code: %w", err)
    }
    allDocs = append(allDocs, codeDocs...)
    
    // Process configuration files with small chunks
    processor.SetTextSplitter(documentloader.CreateCustomSplitter(500, 100, nil))
    configDocs, err := processor.ProcessDirectory(ctx, "/config", false, []string{".yaml", ".json"})
    if err != nil {
        return nil, fmt.Errorf("failed to process config files: %w", err)
    }
    allDocs = append(allDocs, configDocs...)
    
    return allDocs, nil
}
```

### Metadata Enrichment

```go
// Enrich documents with additional context
func EnrichDocuments(docs []schema.Document, projectName string, version string) []schema.Document {
    enriched := make([]schema.Document, len(docs))
    
    for i, doc := range docs {
        enriched[i] = doc
        
        // Ensure metadata exists
        if enriched[i].Metadata == nil {
            enriched[i].Metadata = make(map[string]any)
        }
        
        // Add project context
        enriched[i].Metadata["project"] = projectName
        enriched[i].Metadata["version"] = version
        enriched[i].Metadata["processed_at"] = time.Now()
        
        // Add content analysis
        content := doc.PageContent
        enriched[i].Metadata["word_count"] = len(strings.Fields(content))
        enriched[i].Metadata["char_count"] = len(content)
        
        // Detect content type
        if strings.Contains(content, "func ") && strings.Contains(content, "package ") {
            enriched[i].Metadata["content_type"] = "go_source"
        } else if strings.HasPrefix(content, "#") || strings.Contains(content, "##") {
            enriched[i].Metadata["content_type"] = "markdown"
        } else if strings.Contains(content, "```") {
            enriched[i].Metadata["content_type"] = "code_documentation"
        } else {
            enriched[i].Metadata["content_type"] = "text"
        }
    }
    
    return enriched
}
```

### Intelligent Filtering

```go
// Filter documents based on content and metadata
func FilterRelevantDocuments(docs []schema.Document, keywords []string, minLength int) []schema.Document {
    var filtered []schema.Document
    
    for _, doc := range docs {
        // Filter by content length
        if len(doc.PageContent) < minLength {
            continue
        }
        
        // Filter by keyword relevance
        content := strings.ToLower(doc.PageContent)
        hasKeyword := false
        for _, keyword := range keywords {
            if strings.Contains(content, strings.ToLower(keyword)) {
                hasKeyword = true
                break
            }
        }
        
        if hasKeyword {
            // Add relevance metadata
            if doc.Metadata == nil {
                doc.Metadata = make(map[string]any)
            }
            doc.Metadata["filtered_for_keywords"] = keywords
            doc.Metadata["relevance_checked"] = true
            
            filtered = append(filtered, doc)
        }
    }
    
    return filtered
}
```

## Integration with RAG Pipeline

### Vector Store Integration

```go
// Complete RAG pipeline with document loading
func BuildRAGKnowledgeBase(ctx context.Context) error {
    // Initialize components
    processor := documentloader.NewDocumentProcessor(logger)
    vectorStore := vectorstore.NewPGVectorStore(dbClient, embedder, logger)
    
    // Configure for technical documentation
    processor.SetTextSplitter(documentloader.CreateMarkdownSplitter(1000, 200))
    
    // Load documents from multiple sources
    docs, err := processor.ProcessDirectory(ctx, "/docs", true, []string{".md", ".txt"})
    if err != nil {
        return fmt.Errorf("failed to load documents: %w", err)
    }
    
    // Enrich with metadata
    docs = EnrichDocuments(docs, "assistant-go", "1.0")
    
    // Filter for quality and relevance
    docs = FilterRelevantDocuments(docs, []string{"api", "guide", "tutorial"}, 100)
    
    // Store in vector database
    ids, err := vectorStore.AddDocuments(ctx, docs)
    if err != nil {
        return fmt.Errorf("failed to add documents to vector store: %w", err)
    }
    
    log.Printf("Successfully added %d documents to knowledge base", len(ids))
    return nil
}
```

### Memory System Integration

```go
// Integration with memory system for intelligent caching
func ProcessAndMemorizeDocuments(ctx context.Context, userID string) error {
    processor := documentloader.NewDocumentProcessor(logger)
    memoryManager := memory.NewMemoryManager(dbClient, config, logger)
    
    // Process user-specific documentation
    docs, err := processor.ProcessDirectory(ctx, fmt.Sprintf("/users/%s/docs", userID), true, nil)
    if err != nil {
        return err
    }
    
    // Store important documents in long-term memory
    for _, doc := range docs {
        // Calculate importance based on content and metadata
        importance := calculateDocumentImportance(doc)
        
        if importance > 0.5 { // Only store moderately important or higher
            memoryEntry := &memory.MemoryEntry{
                Type:       memory.MemoryTypeLongTerm,
                UserID:     userID,
                Content:    doc.PageContent,
                Importance: importance,
                Context: map[string]interface{}{
                    "source":      doc.Metadata["source"],
                    "document_id": doc.Metadata["doc_id"],
                    "content_type": doc.Metadata["content_type"],
                },
                Metadata: doc.Metadata,
            }
            
            err := memoryManager.Store(ctx, memoryEntry)
            if err != nil {
                log.Printf("Failed to store document in memory: %v", err)
            }
        }
    }
    
    return nil
}

func calculateDocumentImportance(doc schema.Document) float64 {
    content := strings.ToLower(doc.PageContent)
    importance := 0.3 // Base importance
    
    // Boost for certain keywords
    highValueKeywords := []string{"api", "tutorial", "guide", "important", "critical"}
    for _, keyword := range highValueKeywords {
        if strings.Contains(content, keyword) {
            importance += 0.2
        }
    }
    
    // Boost for longer, more substantial content
    if len(doc.PageContent) > 1000 {
        importance += 0.1
    }
    
    // Boost for code examples
    if strings.Contains(content, "```") || strings.Contains(content, "func ") {
        importance += 0.15
    }
    
    // Cap at 1.0
    if importance > 1.0 {
        importance = 1.0
    }
    
    return importance
}
```

## Configuration and Optimization

### Chunk Size Guidelines

```go
// Different content types benefit from different chunk sizes

// Technical documentation - moderate chunks for context
techSplitter := documentloader.CreateMarkdownSplitter(1200, 250)

// API reference - smaller chunks for precision
apiSplitter := documentloader.CreateCustomSplitter(800, 150, nil)

// Source code - smaller chunks to preserve function boundaries
codeSplitter := documentloader.CreateCodeSplitter("go", 600, 100)

// Configuration files - very small chunks
configSplitter := documentloader.CreateCustomSplitter(300, 50, nil)

// Long-form content - larger chunks for narrative flow
contentSplitter := documentloader.CreateCustomSplitter(1500, 300, nil)
```

### Performance Optimization

```go
// Parallel processing for large document sets
func ProcessDocumentsInParallel(ctx context.Context, filePaths []string) ([]schema.Document, error) {
    const maxWorkers = 10
    
    // Create worker pool
    workChan := make(chan string, len(filePaths))
    resultChan := make(chan []schema.Document, len(filePaths))
    errorChan := make(chan error, len(filePaths))
    
    // Start workers
    for i := 0; i < maxWorkers; i++ {
        go func() {
            processor := documentloader.NewDocumentProcessor(logger)
            for filePath := range workChan {
                docs, err := processor.ProcessFile(ctx, filePath)
                if err != nil {
                    errorChan <- fmt.Errorf("failed to process %s: %w", filePath, err)
                    return
                }
                resultChan <- docs
            }
        }()
    }
    
    // Send work
    for _, filePath := range filePaths {
        workChan <- filePath
    }
    close(workChan)
    
    // Collect results
    var allDocs []schema.Document
    for i := 0; i < len(filePaths); i++ {
        select {
        case docs := <-resultChan:
            allDocs = append(allDocs, docs...)
        case err := <-errorChan:
            return nil, err
        case <-ctx.Done():
            return nil, ctx.Err()
        }
    }
    
    return allDocs, nil
}
```

### Error Handling and Resilience

```go
// Robust document processing with error recovery
func ProcessDocumentsWithRetry(ctx context.Context, processor *documentloader.DocumentProcessor, filePaths []string) ([]schema.Document, []error) {
    var allDocs []schema.Document
    var errors []error
    
    for _, filePath := range filePaths {
        // Try processing with retries
        docs, err := processFileWithRetry(ctx, processor, filePath, 3)
        if err != nil {
            errors = append(errors, fmt.Errorf("failed to process %s after retries: %w", filePath, err))
            continue
        }
        
        allDocs = append(allDocs, docs...)
    }
    
    return allDocs, errors
}

func processFileWithRetry(ctx context.Context, processor *documentloader.DocumentProcessor, filePath string, maxRetries int) ([]schema.Document, error) {
    for attempt := 1; attempt <= maxRetries; attempt++ {
        docs, err := processor.ProcessFile(ctx, filePath)
        if err == nil {
            return docs, nil
        }
        
        // Log retry attempt
        log.Printf("Attempt %d/%d failed for %s: %v", attempt, maxRetries, filePath, err)
        
        // Don't retry on certain types of errors
        if os.IsNotExist(err) || strings.Contains(err.Error(), "permission denied") {
            return nil, err
        }
        
        // Wait before retry
        if attempt < maxRetries {
            time.Sleep(time.Duration(attempt) * time.Second)
        }
    }
    
    return nil, fmt.Errorf("exhausted %d retry attempts", maxRetries)
}
```

## Testing

### Unit Tests

```go
func TestDocumentProcessor_LoadFile(t *testing.T) {
    processor := documentloader.NewDocumentProcessor(testLogger)
    
    // Create test file
    content := "This is test content for document loading"
    testFile := createTempFile(t, "test.txt", content)
    defer os.Remove(testFile)
    
    // Load document
    docs, err := processor.LoadFile(context.Background(), testFile)
    assert.NoError(t, err)
    assert.Len(t, docs, 1)
    assert.Equal(t, content, docs[0].PageContent)
    assert.Equal(t, testFile, docs[0].Metadata["source"])
}

func TestDocumentProcessor_SplitDocuments(t *testing.T) {
    processor := documentloader.NewDocumentProcessor(testLogger)
    
    // Create test document that will be split
    longContent := strings.Repeat("This is a sentence. ", 200) // ~4000 chars
    doc := schema.Document{
        PageContent: longContent,
        Metadata:    map[string]any{"test": true},
    }
    
    chunks, err := processor.SplitDocuments(context.Background(), []schema.Document{doc})
    assert.NoError(t, err)
    assert.Greater(t, len(chunks), 1) // Should be split into multiple chunks
    
    // Verify chunk metadata
    for i, chunk := range chunks {
        assert.Equal(t, i, chunk.Metadata["chunk_index"])
        assert.Equal(t, len(chunks), chunk.Metadata["total_chunks"])
        assert.True(t, chunk.Metadata["test"].(bool)) // Original metadata preserved
    }
}
```

### Integration Tests

```go
func TestDocumentProcessor_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    processor := documentloader.NewDocumentProcessor(testLogger)
    
    // Create test directory structure
    testDir := setupTestDirectory(t)
    defer os.RemoveAll(testDir)
    
    // Process directory
    docs, err := processor.ProcessDirectory(context.Background(), testDir, true, nil)
    require.NoError(t, err)
    require.NotEmpty(t, docs)
    
    // Verify all documents have required metadata
    for _, doc := range docs {
        assert.NotEmpty(t, doc.PageContent)
        assert.Contains(t, doc.Metadata, "source")
        assert.Contains(t, doc.Metadata, "file_name")
        assert.Contains(t, doc.Metadata, "file_extension")
    }
}
```

## Best Practices

### Content Preparation

```go
// Prepare content for optimal RAG performance
func PrepareContentForRAG(content string) string {
    // Normalize whitespace
    content = strings.TrimSpace(content)
    content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
    
    // Remove excessive punctuation
    content = regexp.MustCompile(`[.]{3,}`).ReplaceAllString(content, "...")
    
    // Ensure content ends with punctuation for better chunking
    if !strings.HasSuffix(content, ".") && !strings.HasSuffix(content, "!") && !strings.HasSuffix(content, "?") {
        content += "."
    }
    
    return content
}
```

### Metadata Design

```go
// Design metadata for effective filtering and routing
func CreateStandardMetadata(filePath string, contentType string) map[string]any {
    return map[string]any{
        // Source information
        "source":         filePath,
        "file_name":      filepath.Base(filePath),
        "file_extension": filepath.Ext(filePath),
        "directory":      filepath.Dir(filePath),
        
        // Content classification
        "content_type":   contentType,
        "language":       detectLanguage(filePath),
        "category":       categorizeContent(contentType),
        
        // Processing metadata
        "processed_at":   time.Now(),
        "processor_version": "1.0",
        
        // Quality indicators
        "quality_score":  calculateQualityScore(filePath),
        "is_complete":    true,
    }
}
```

This document loader package provides a comprehensive foundation for ingesting and processing documents in the AI assistant's RAG pipeline, enabling sophisticated document understanding and retrieval capabilities.