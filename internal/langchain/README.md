# LangChain Integration Framework

## Overview

The LangChain integration framework provides a comprehensive AI-powered workflow system for GoAssistant. This implementation leverages the LangChain Go library to create sophisticated agent-based interactions, multi-step reasoning, and advanced memory management.

## Architecture

### Core Components

1. **Service Layer** (`service.go`)
   - Central orchestration for all LangChain operations
   - Chain execution management
   - Provider abstraction for multiple AI backends

2. **Agent System** (`agents/`)
   - Specialized agents for different domains
   - Multi-step reasoning capabilities
   - Tool integration and execution

3. **Chain Framework** (`chains/`)
   - Sequential, parallel, and conditional execution patterns
   - RAG (Retrieval-Augmented Generation) capabilities
   - Enhanced chain composition

4. **Memory Management** (`memory/`)
   - Short-term conversational memory
   - Long-term personalized memory
   - Tool-specific memory caching
   - Vector-based semantic memory

5. **Vector Store** (`vectorstore/`)
   - PostgreSQL + pgvector integration
   - Semantic search capabilities
   - Document embedding and retrieval

6. **Document Processing** (`documentloader/`)
   - Multi-format document ingestion
   - Text splitting and chunking
   - Metadata preservation

## Design Philosophy

### Interface-Driven Architecture

The framework follows Go's principle of "accept interfaces, return structs":

```go
type ChainExecutor interface {
    Execute(ctx context.Context, request *ChainRequest) (*ChainResponse, error)
}

type MemoryManager interface {
    Store(ctx context.Context, entry *MemoryEntry) error
    Retrieve(ctx context.Context, query *MemoryQuery) ([]*MemoryEntry, error)
}
```

### Type Safety

All operations are type-safe with structured request/response patterns:

```go
type ChainRequest struct {
    Input       string                 `json:"input"`
    Context     map[string]interface{} `json:"context,omitempty"`
    Parameters  map[string]interface{} `json:"parameters,omitempty"`
    MaxSteps    int                    `json:"max_steps,omitempty"`
    Temperature float64                `json:"temperature,omitempty"`
}
```

### Error Handling

Comprehensive error handling with context preservation:

```go
if err != nil {
    return fmt.Errorf("chain execution failed at step %d: %w", stepNum, err)
}
```

## Agent System

### Specialized Agents

#### Development Agent
- **Purpose**: Code analysis, generation, and optimization
- **Capabilities**: 
  - AST analysis and code understanding
  - Code generation with best practices
  - Performance analysis and optimization suggestions
  - Test generation and refactoring recommendations
- **Tools**: Go development tools, linters, formatters

#### Database Agent
- **Purpose**: Database operations and optimization
- **Capabilities**:
  - SQL query generation and optimization
  - Schema exploration and analysis
  - Data analysis and insights
  - Migration assistance
- **Tools**: PostgreSQL tools, query analyzers

#### Infrastructure Agent
- **Purpose**: DevOps and infrastructure management
- **Capabilities**:
  - Kubernetes resource management
  - Docker container operations
  - Cloudflare service configuration
  - System monitoring and diagnostics
- **Tools**: kubectl, docker, cloudflare CLI equivalents

#### Research Agent
- **Purpose**: Information gathering and analysis
- **Capabilities**:
  - Web search and content analysis
  - Document processing and summarization
  - Knowledge synthesis
  - Fact verification
- **Tools**: Search engines, document processors

### Agent Execution Flow

1. **Request Processing**: Parse and validate incoming requests
2. **Tool Selection**: Determine appropriate tools for the task
3. **Multi-Step Reasoning**: Execute complex workflows
4. **Memory Integration**: Leverage past interactions
5. **Response Generation**: Synthesize results with explanations

## Chain Types

### Sequential Chains
Execute steps in a predefined order:

```go
type SequentialChain struct {
    Steps []ChainStep
}
```

### Parallel Chains
Execute multiple steps concurrently:

```go
type ParallelChain struct {
    Branches []ChainBranch
    Merger   ResultMerger
}
```

### Conditional Chains
Dynamic execution based on conditions:

```go
type ConditionalChain struct {
    Condition func(context.Context, *ChainState) bool
    TrueBranch ChainExecutor
    FalseBranch ChainExecutor
}
```

### RAG Chains
Retrieval-Augmented Generation with vector search:

```go
type RAGChain struct {
    VectorStore VectorStore
    LLM        llms.Model
    Retriever  DocumentRetriever
}
```

## Memory System

### Memory Types

#### Short-Term Memory
- **Scope**: Single conversation session
- **Retention**: Until session ends
- **Use Case**: Context maintenance within conversations

#### Long-Term Memory
- **Scope**: User-specific, persistent
- **Retention**: Configurable (days, weeks, months)
- **Use Case**: Personalization and user preferences

#### Tool Memory
- **Scope**: Tool-specific caching
- **Retention**: Based on tool requirements
- **Use Case**: Performance optimization and result caching

### Memory Operations

```go
// Store memory entry
entry := &MemoryEntry{
    UserID:     "user123",
    Type:       MemoryTypeShortTerm,
    Content:    "User prefers Python over Go",
    Importance: 0.8,
    Metadata:   map[string]interface{}{"topic": "programming"},
}
err := memoryManager.Store(ctx, entry)

// Retrieve relevant memories
query := &MemoryQuery{
    UserID:   "user123",
    Query:    "programming preferences",
    Limit:    10,
    MinScore: 0.7,
}
memories, err := memoryManager.Retrieve(ctx, query)
```

## Vector Store Integration

### PostgreSQL + pgvector

The vector store uses PostgreSQL with the pgvector extension for high-performance semantic search:

```sql
CREATE TABLE embeddings (
    id UUID PRIMARY KEY,
    content_text TEXT NOT NULL,
    embedding vector(1536),
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX idx_embeddings_vector 
ON embeddings USING ivfflat (embedding vector_cosine_ops) 
WITH (lists = 100);
```

### Document Operations

```go
// Add documents
docs := []*schema.Document{
    {
        PageContent: "LangChain enables building applications with LLMs",
        Metadata:    map[string]any{"source": "documentation"},
    },
}
err := vectorStore.AddDocuments(ctx, docs)

// Similarity search
results, err := vectorStore.SimilaritySearch(ctx, "How to use LangChain?", 5)
```

## Usage Examples

### Basic Chain Execution

```go
service := NewService(cfg, logger)

request := &ChainExecutionRequest{
    UserID: "user123",
    ChainRequest: &chains.ChainRequest{
        Input:       "Analyze this Go code for performance issues",
        MaxSteps:    5,
        Temperature: 0.7,
    },
}

response, err := service.ExecuteChain(ctx, chains.ChainTypeSequential, request)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Result: %s\n", response.Output)
fmt.Printf("Steps taken: %d\n", len(response.Steps))
```

### Agent Interaction

```go
agent := agents.NewDevelopmentAgent(logger, tools, cfg)

result, err := agent.Execute(ctx, &agents.AgentRequest{
    Query:    "Review this code and suggest improvements",
    Context:  map[string]interface{}{"language": "go"},
    MaxSteps: 3,
})

if err != nil {
    log.Fatal(err)
}

fmt.Printf("Analysis: %s\n", result.Output)
```

### Memory Management

```go
memManager := memory.NewManager(db, embeddingService, logger)

// Store user interaction
entry := &memory.MemoryEntry{
    UserID:     userID,
    Type:       memory.MemoryTypeShortTerm,
    Content:    "User asked about Go concurrency patterns",
    Importance: 0.8,
}
err := memManager.Store(ctx, entry)

// Retrieve relevant context
memories, err := memManager.Search(ctx, userID, "concurrency", 5)
```

### RAG Implementation

```go
// Initialize RAG chain
ragChain := chains.NewEnhancedRAGChain(vectorStore, llm, logger)

// Ingest documents
err := ragChain.IngestDocuments(ctx, "path/to/docs")
if err != nil {
    log.Fatal(err)
}

// Query with context
response, err := ragChain.Execute(ctx, &chains.ChainRequest{
    Input: "How do I implement dependency injection in Go?",
})

fmt.Printf("Answer: %s\n", response.Output)
```

## Configuration

### Environment Variables

```bash
# LangChain Configuration
LANGCHAIN_ENABLE_MEMORY=true
LANGCHAIN_MEMORY_SIZE=10
LANGCHAIN_MAX_ITERATIONS=5
LANGCHAIN_TIMEOUT=60s

# Vector Store Configuration
EMBEDDING_PROVIDER=claude
EMBEDDING_MODEL=text-embedding-ada-002
EMBEDDING_DIMENSIONS=1536
```

### YAML Configuration

```yaml
langchain:
  enable_memory: true
  memory_size: 10
  max_iterations: 5
  timeout: "60s"
  
ai:
  embeddings:
    provider: "claude"
    model: "text-embedding-ada-002"
    dimensions: 1536
```

## Testing

### Unit Tests

Each component includes comprehensive unit tests:

```go
func TestSequentialChain_Execute(t *testing.T) {
    chain := chains.NewSequentialChain([]chains.ChainStep{
        mockStep1, mockStep2, mockStep3,
    })
    
    response, err := chain.Execute(ctx, request)
    
    assert.NoError(t, err)
    assert.Equal(t, 3, len(response.Steps))
    assert.Contains(t, response.Output, "expected result")
}
```

### Integration Tests

Integration tests use testcontainers for real database testing:

```go
func TestVectorStore_Integration(t *testing.T) {
    container := setupPostgreSQLContainer(t)
    defer container.Terminate(ctx)
    
    vectorStore := NewPGVectorStore(container.ConnectionString())
    
    // Test document ingestion and retrieval
    docs := []*schema.Document{{PageContent: "test content"}}
    err := vectorStore.AddDocuments(ctx, docs)
    assert.NoError(t, err)
    
    results, err := vectorStore.SimilaritySearch(ctx, "test", 1)
    assert.NoError(t, err)
    assert.Len(t, results, 1)
}
```

## Performance Considerations

### Memory Management
- Automatic cleanup of expired short-term memories
- Configurable importance-based retention for long-term memories
- LRU caching for frequently accessed tool results

### Vector Operations
- Batch embedding generation for efficiency
- Indexing strategies for large document collections
- Query optimization with similarity thresholds

### Concurrency
- Parallel chain execution where applicable
- Concurrent tool execution with proper synchronization
- Rate limiting for external API calls

## Best Practices

### Error Handling
- Always wrap errors with context
- Use structured logging for debugging
- Implement retry mechanisms for transient failures

### Memory Usage
- Set appropriate importance scores for memories
- Implement regular cleanup routines
- Monitor memory usage and adjust limits as needed

### Tool Integration
- Use tool-specific timeouts
- Implement proper input validation
- Cache results when appropriate

### Security
- Sanitize user inputs before processing
- Implement proper authentication for tool access
- Use secure connections for external services

## Monitoring and Observability

### Metrics
- Chain execution times and success rates
- Memory storage and retrieval performance
- Vector store operation metrics
- Tool execution statistics

### Logging
- Structured logging with context preservation
- Debug information for troubleshooting
- Performance metrics and timing information

### Health Checks
- Service availability monitoring
- Database connection health
- External service dependency status

## Future Enhancements

### Planned Features
- Multi-modal document processing (images, audio)
- Advanced agent collaboration patterns
- Real-time memory synchronization
- Enhanced RAG with graph-based retrieval

### Extensibility Points
- Custom agent implementations
- Pluggable memory backends
- Additional vector store providers
- Custom chain types and execution patterns

---

This LangChain integration provides a robust foundation for building sophisticated AI-powered applications with advanced reasoning capabilities, persistent memory, and seamless tool integration.