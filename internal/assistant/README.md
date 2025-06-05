# Assistant Core Orchestration

This package implements the main orchestration engine for the Assistant intelligent development companion. It coordinates between AI providers, tools, memory systems, and user interactions to provide contextual and intelligent assistance.

## Overview

The assistant package serves as the central coordination point for all intelligence systems. It manages:

- **Request Processing**: Handles user queries and commands with context awareness
- **Agent Coordination**: Orchestrates multiple AI agents for complex tasks
- **Tool Execution**: Manages tool registry and execution pipeline
- **Memory Integration**: Maintains working, episodic, and semantic memory
- **Context Management**: Tracks development environment and user patterns

## Key Components

### Assistant (`assistant.go`)
Main orchestration engine that:
- Processes user requests with full context
- Coordinates between multiple AI agents
- Manages tool execution and results
- Handles memory storage and retrieval

### Context Manager (`context.go`)
Maintains rich contextual awareness:
- Tracks current development environment
- Manages conversation history
- Monitors workspace changes
- Provides contextual embeddings

### Request Processor (`processor.go`)
Advanced request processing with:
- Intent recognition and classification
- Multi-step reasoning coordination
- Tool chain optimization
- Result synthesis and presentation

## Usage Example

```go
// Initialize the assistant with configuration
assistant, err := assistant.New(&Config{
    AIProvider: "claude",
    DatabaseURL: "postgres://...",
    WorkspacePath: "/path/to/project",
})

// Process a user request
response, err := assistant.ProcessRequest(ctx, &Request{
    Query: "Help me optimize this slow database query",
    Context: RequestContext{
        CurrentFile: "queries.sql",
        RecentChanges: []string{"schema.sql", "migrations/001.sql"},
    },
})
```

## Testing

The package includes comprehensive tests:
- Unit tests for core functionality
- Integration tests with real AI providers
- Property-based tests for edge cases
- Benchmark tests for performance validation

## Error Handling

All functions provide detailed error context using Go's error wrapping:
- Request validation errors
- AI provider communication errors  
- Tool execution errors
- Memory system errors