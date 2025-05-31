# Internal Modules

This directory contains the core implementation modules of the Assistant intelligent development companion.

## Directory Structure

### Core Systems
- **`assistant/`** - Main assistant logic and query processing
- **`ai/`** - AI provider integrations (Claude, Gemini) and embedding services
- **`config/`** - Configuration management and validation
- **`observability/`** - Logging, metrics, and performance profiling

### Intelligence & Memory
- **`core/`** - Core intelligence systems (agents, context, memory)
- **`langchain/`** - LangChain integration framework for advanced AI workflows

### Data & Storage
- **`storage/`** - Database clients and data persistence
  - **`postgres/`** - PostgreSQL client with pgvector support

### Tools & Infrastructure
- **`tools/`** - Development tools and integrations
  - **`godev/`** - Go development tools (analyzer, builder, formatter, tester)
  - **`cloudflare/`** - Cloudflare service management
  - **`docker/`** - Docker container operations
  - **`k8s/`** - Kubernetes resource management
  - **`postgres/`** - Database management tools
  - **`search/`** - Search and information retrieval

### Interfaces
- **`api/`** - HTTP API handlers and middleware
- **`cli/`** - Command-line interface implementation
- **`server/`** - Web server and WebSocket handlers

## Key Design Principles

### 1. Interface-Driven Design
Following Go's principle of "accept interfaces, return structs":
```go
type ToolExecutor interface {
    Execute(ctx context.Context, input ToolInput) (ToolResult, error)
}
```

### 2. Contextual Intelligence
Every component maintains awareness of the broader development context:
- File system state
- Project structure
- Recent changes
- User patterns

### 3. Learning and Adaptation
Components contribute to the system's learning:
- Capture interaction patterns
- Learn from outcomes
- Adapt to user preferences
- Improve over time

### 4. Robust Error Handling
Comprehensive error handling with context preservation:
```go
if err != nil {
    return fmt.Errorf("failed to execute tool %s: %w", toolName, err)
}
```

## Module Dependencies

```
assistant/
├── ai/ (AI providers)
├── config/ (configuration)
├── storage/ (data persistence)
├── tools/ (development tools)
├── core/ (intelligence systems)
└── observability/ (monitoring)

langchain/
├── ai/ (providers)
├── storage/ (vector store)
└── tools/ (tool adapters)

server/
├── assistant/ (core logic)
├── api/ (HTTP handlers)
└── observability/ (metrics)

cli/
├── assistant/ (core logic)
└── tools/ (tool integrations)
```

## Getting Started

### For Contributors

1. **Read the Architecture**: Start with `/Architecture.md` for system overview
2. **Choose a Module**: Pick a module aligned with your interests/expertise
3. **Follow Patterns**: Study existing implementations before adding new features
4. **Write Tests**: Comprehensive testing is required for all changes
5. **Document Changes**: Update relevant README files and documentation

### For Module Development

Each module should:
- Have clear interfaces and error handling
- Include comprehensive unit tests
- Follow Go best practices from `golang_guide.md`
- Implement observability (logging, metrics)
- Support graceful degradation

### Common Patterns

#### Tool Implementation
```go
type CustomTool struct {
    name   string
    logger *slog.Logger
}

func (t *CustomTool) Execute(ctx context.Context, input ToolInput) (ToolResult, error) {
    t.logger.Debug("Executing tool", slog.String("name", t.name))
    
    // Implementation
    result, err := t.performOperation(ctx, input)
    if err != nil {
        return ToolResult{}, fmt.Errorf("tool execution failed: %w", err)
    }
    
    return result, nil
}
```

#### Context-Aware Processing
```go
func (p *Processor) ProcessWithContext(ctx context.Context, request Request) (Response, error) {
    // Gather context
    context := p.contextEngine.GatherContext(request)
    
    // Apply learned patterns
    patterns := p.learningSystem.GetRelevantPatterns(context)
    
    // Process with intelligence
    response := p.processIntelligently(ctx, request, context, patterns)
    
    // Learn from outcome
    p.learningSystem.RecordOutcome(request, response)
    
    return response, nil
}
```

This modular architecture enables the Assistant to grow in capability while maintaining code quality and system performance.