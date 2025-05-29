# Memory Package

The memory package provides a comprehensive memory system for the LangChain integration, enabling the assistant to maintain context, learn from interactions, and optimize performance across multiple types of memory storage.

## Overview

The memory system is designed around four distinct memory types, each serving specific purposes in the AI assistant's cognitive architecture:

- **Short-term Memory**: Buffer-based storage for immediate conversation context
- **Long-term Memory**: Persistent semantic storage for important information
- **Tool Memory**: Caching system for tool execution results  
- **Personalization Memory**: User preferences and contextual information storage

## Core Components

### Memory Manager (`manager.go`)

The central orchestrator that coordinates all memory types and provides a unified interface.

```go
// Create memory manager
manager := memory.NewMemoryManager(dbClient, config, logger)

// Store memory entry
entry := &memory.MemoryEntry{
    Type:       memory.MemoryTypeLongTerm,
    UserID:     "user123",
    Content:    "Important information to remember",
    Importance: 0.8,
}
err := manager.Store(ctx, entry)

// Search across all memory types
query := &memory.MemoryQuery{
    UserID:  "user123",
    Content: "search query",
    Limit:   10,
}
results, err := manager.Retrieve(ctx, query)
```

**Key Features:**
- Unified interface across all memory types
- Concurrent search across multiple memory systems
- Automatic routing based on memory type
- Relevance-based result ranking and limiting

### Short-term Memory (`shortterm.go`)

Buffer-based memory for immediate conversation context with automatic expiration.

```go
// Create short-term memory
stm := memory.NewShortTermMemory(config, logger)

// Store conversation context
entry := &memory.MemoryEntry{
    Type:      memory.MemoryTypeShortTerm,
    UserID:    "user123", 
    SessionID: "session456",
    Content:   "User asked about deployment strategies",
    ExpiresAt: &expirationTime,
}
err := stm.Store(ctx, entry)
```

**Characteristics:**
- **Size Limit**: Configurable maximum entries per user
- **Time-based Expiration**: Automatic cleanup of old entries
- **Session Tracking**: Entries can be scoped to specific sessions
- **FIFO Eviction**: Oldest entries removed when size limit exceeded

### Long-term Memory (`longterm.go`)

Persistent semantic memory with vector search capabilities for important information retention.

```go
// Create with vector store integration
ltm := memory.NewLongTermMemoryWithVectorStore(dbClient, vectorStore, embedder, config, logger)

// Store important knowledge
entry := &memory.MemoryEntry{
    Type:       memory.MemoryTypeLongTerm,
    UserID:     "user123",
    Content:    "Project uses microservices architecture with Kubernetes",
    Importance: 0.9,
    Context: map[string]interface{}{
        "project": "main-app",
        "domain":  "architecture",
    },
}
err := ltm.Store(ctx, entry)

// Semantic search
query := &memory.MemoryQuery{
    UserID:        "user123",
    Content:       "architecture patterns",
    Similarity:    0.7,
    MinImportance: 0.5,
}
results, err := ltm.Search(ctx, query)
```

**Features:**
- **Dual Storage**: LangChain vectorstore + PostgreSQL fallback
- **Semantic Search**: Vector similarity with configurable thresholds
- **Importance Weighting**: Relevance calculated from similarity, importance, and recency
- **Rich Metadata**: Context information and custom metadata support
- **Embedding Generation**: Automatic or custom embedding support

### Tool Memory (`tool.go`)

Intelligent caching system for tool execution results to optimize performance.

```go
// Create tool memory
tm := memory.NewToolMemory(config, logger)

// Cache tool execution result
err := tm.CacheToolResult(
    ctx,
    "user123",
    "kubernetes_get_pods",
    map[string]interface{}{"namespace": "production"},
    podList,
    500*time.Millisecond,
    true,
    "",
)

// Check for cached result
cacheEntry, found := tm.GetCachedResult(ctx, "user123", "kubernetes_get_pods", inputHash)
if found && !cacheEntry.Error {
    return cacheEntry.Output // Use cached result
}
```

**Benefits:**
- **Performance Optimization**: Avoid expensive tool re-execution
- **Hit Count Tracking**: Popular cache entries stay longer
- **Input-based Hashing**: Precise cache key generation
- **Success/Failure Tracking**: Cache both successful and failed executions
- **Automatic Expiration**: Configurable cache lifetime

### Personalization Memory (`personalization.go`)

Storage for user preferences and contextual information that personalizes the assistant's behavior.

```go
// Create personalization memory
pm := memory.NewPersonalizationMemory(dbClient, config, logger)

// Store user preference
prefEntry := &memory.MemoryEntry{
    Type:    memory.MemoryTypePersonalization,
    UserID:  "user123",
    Content: "User prefers detailed technical explanations",
    Context: map[string]interface{}{
        "category": "communication",
        "key":      "explanation_detail",
        "value":    "detailed",
        "type":     "string",
    },
}
err := pm.Store(ctx, prefEntry)

// Store user context
contextEntry := &memory.MemoryEntry{
    Type:       memory.MemoryTypePersonalization,
    UserID:     "user123", 
    Content:    "Current working on e-commerce project",
    Importance: 0.8,
    Context: map[string]interface{}{
        "context_type": "project",
        "context_key":  "current_project",
        "context_value": map[string]interface{}{
            "name":   "e-commerce-platform",
            "stack":  "Go, React, PostgreSQL",
            "stage":  "development",
        },
    },
}
err := pm.Store(ctx, contextEntry)

// Retrieve preferences
preferences, err := pm.GetUserPreferences(ctx, "user123", "communication")
```

**Use Cases:**
- **User Preferences**: UI settings, communication style, content preferences
- **Project Context**: Current projects, technology stacks, team information
- **Domain Knowledge**: User expertise areas, learning goals
- **Interaction History**: Patterns in user behavior and preferences

## Memory Types and Entry Structure

### MemoryEntry

The core data structure for all memory operations:

```go
type MemoryEntry struct {
    ID          string                 `json:"id"`
    Type        MemoryType             `json:"type"`
    UserID      string                 `json:"user_id"`
    SessionID   string                 `json:"session_id,omitempty"`
    Content     string                 `json:"content"`
    Context     map[string]interface{} `json:"context,omitempty"`
    Embedding   []float64              `json:"embedding,omitempty"`
    Importance  float64                `json:"importance"`     // 0.0 to 1.0
    AccessCount int                    `json:"access_count"`
    LastAccess  time.Time              `json:"last_access"`
    CreatedAt   time.Time              `json:"created_at"`
    ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
```

### MemoryQuery

Flexible query structure for memory retrieval:

```go
type MemoryQuery struct {
    UserID        string                 `json:"user_id"`
    SessionID     string                 `json:"session_id,omitempty"`
    Types         []MemoryType           `json:"types,omitempty"`
    Content       string                 `json:"content,omitempty"`
    Embedding     []float64              `json:"embedding,omitempty"`
    Similarity    float64                `json:"similarity,omitempty"`
    Limit         int                    `json:"limit,omitempty"`
    TimeRange     *TimeRange             `json:"time_range,omitempty"`
    MinImportance float64                `json:"min_importance,omitempty"`
    Context       map[string]interface{} `json:"context,omitempty"`
}
```

## Configuration

### LangChain Configuration

```yaml
langchain:
  memory_size: 1000        # Short-term memory buffer size
  vector_dimensions: 1536  # Embedding dimensions
  similarity_threshold: 0.7 # Default similarity threshold
```

### Environment Variables

```bash
# Database connection for persistent memory
DATABASE_URL=postgresql://user:pass@localhost/assistant

# AI provider for embeddings
CLAUDE_API_KEY=your_claude_key
GEMINI_API_KEY=your_gemini_key
```

## Advanced Usage

### Custom Memory Strategies

```go
// Multi-type search with custom filtering
query := &memory.MemoryQuery{
    UserID: "user123",
    Types:  []memory.MemoryType{
        memory.MemoryTypeLongTerm,
        memory.MemoryTypePersonalization,
    },
    Content:       "kubernetes deployment",
    MinImportance: 0.6,
    TimeRange: &memory.TimeRange{
        Start: time.Now().AddDate(0, -1, 0), // Last month
        End:   time.Now(),
    },
}

results, err := manager.Retrieve(ctx, query)
```

### Memory Cleanup and Maintenance

```go
// Clean up expired entries
err := manager.Cleanup(ctx)

// Clear old memories for a user
olderThan := time.Now().AddDate(0, -6, 0) // 6 months ago
err := manager.Clear(ctx, "user123", nil, &olderThan)

// Get memory statistics
stats, err := manager.GetStats(ctx, "user123")
fmt.Printf("Total entries: %d, Total size: %d bytes\n", 
    stats.TotalEntries, stats.TotalSize)
```

### Vector Store Integration

```go
// Create with custom vector store
vectorStore := vectorstore.NewPGVectorStore(dbClient, embedder, config, logger)
ltm := memory.NewLongTermMemoryWithVectorStore(dbClient, vectorStore, embedder, config, logger)

// Use LangChain-compatible embedder
embedder := ai.NewEmbeddingService(aiFactory, config.AI, logger)
ltm.SetEmbedder(embedder)
```

## Error Handling

The memory system gracefully handles various error conditions:

```go
// Database unavailable - falls back to in-memory operation
if dbClient == nil {
    // Memory continues to work with reduced functionality
    logger.Warn("Database unavailable, using fallback memory")
}

// Embedding generation fails - continues with text-based search
if err := generateEmbedding(ctx, text); err != nil {
    logger.Warn("Embedding failed, using text search fallback")
    return searchByText(ctx, query)
}

// Cache miss - normal operation continues
if cacheEntry, found := toolMemory.GetCachedResult(ctx, userID, toolName, hash); !found {
    // Execute tool normally and cache result
    result := executeTool(ctx, input)
    toolMemory.CacheToolResult(ctx, userID, toolName, input, result, duration, true, "")
}
```

## Performance Considerations

### Memory Usage Optimization

- **Buffer Limits**: Short-term memory automatically enforces size limits
- **Expiration**: Automatic cleanup of expired entries
- **Lazy Loading**: Embeddings generated only when needed
- **Batch Operations**: Efficient bulk operations for memory management

### Search Performance

- **Index Usage**: PostgreSQL indexes on user_id, created_at, and vector columns
- **Similarity Thresholds**: Configurable to balance precision vs. recall
- **Result Limiting**: Built-in pagination and limiting support
- **Concurrent Search**: Parallel search across memory types

## Best Practices

### Memory Entry Design

```go
// Good: Specific, actionable content
entry := &memory.MemoryEntry{
    Content:    "User prefers Docker Compose over Kubernetes for local development",
    Importance: 0.7,
    Context: map[string]interface{}{
        "domain":     "development",
        "preference": "tools",
        "scope":      "local",
    },
}

// Good: Rich context for better retrieval
entry := &memory.MemoryEntry{
    Content: "Fixed CORS issue in API gateway by adding allowed origins configuration",
    Context: map[string]interface{}{
        "issue_type": "cors",
        "component":  "api_gateway", 
        "solution":   "configuration",
        "project":    "main-app",
    },
}
```

### Query Optimization

```go
// Use specific memory types when possible
query := &memory.MemoryQuery{
    UserID: userID,
    Types:  []memory.MemoryType{memory.MemoryTypeLongTerm}, // Specific type
    Content: "database migration",
    Limit:   5, // Reasonable limit
}

// Use importance filtering for high-quality results
query.MinImportance = 0.5 // Only moderately important or higher
```

### Integration Patterns

```go
// Memory-aware assistant response
func (a *Assistant) processQuery(ctx context.Context, userID, query string) (string, error) {
    // 1. Search relevant memories
    memQuery := &memory.MemoryQuery{
        UserID:  userID,
        Content: query,
        Limit:   5,
    }
    memories, err := a.memoryManager.Retrieve(ctx, memQuery)
    
    // 2. Build context from memories
    context := buildContextFromMemories(memories)
    
    // 3. Generate response with memory context
    response, err := a.aiProvider.GenerateResponse(ctx, query, context)
    
    // 4. Store important information from interaction
    if shouldRemember(response) {
        memory := &memory.MemoryEntry{
            Type:       memory.MemoryTypeLongTerm,
            UserID:     userID,
            Content:    extractKeyInformation(query, response),
            Importance: calculateImportance(query, response),
        }
        a.memoryManager.Store(ctx, memory)
    }
    
    return response, nil
}
```

This memory system provides the foundation for building intelligent, context-aware AI assistants that can learn, adapt, and optimize their performance over time.