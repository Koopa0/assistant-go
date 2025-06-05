# Go Idioms Refactoring Plan

This document outlines the refactoring needed to align the codebase with Go best practices and principles from golang_guide.md.

## Priority 1: Remove Base Types and Inheritance

### 1.1 Remove BaseTool Pattern
**Current Issue**: Java-style inheritance with `BaseTool`
```go
// BAD: Current approach
type BaseTool struct {
    name string
    // ... common fields
}

type GoAnalyzer struct {
    *BaseTool
    // ... specific fields
}
```

**Solution**: Use composition where truly needed
```go
// GOOD: Go approach
type GoAnalyzer struct {
    name        string
    description string
    logger      *slog.Logger
}

func (g *GoAnalyzer) Name() string { return g.name }
func (g *GoAnalyzer) Execute(ctx context.Context, input string) (string, error) {
    // Direct implementation
}
```

### 1.2 Remove BaseAgent Pattern
**Files to refactor**:
- `/internal/core/agents/base.go` (delete)
- `/internal/langchain/agents/base.go` (simplify)
- `/internal/agents/agent.go` (already improved)

## Priority 2: Simplify Factory and Manager Patterns

### 2.1 Replace Factory with Simple Constructors
**Current Issue**: Complex factory pattern
```go
// BAD: Current factory
factory.RegisterProvider("claude", claudeConstructor)
provider := factory.CreateProvider("claude", config)
```

**Solution**: Direct constructors
```go
// GOOD: Simple constructor
func NewClaudeProvider(config Config, logger *slog.Logger) (*ClaudeProvider, error) {
    // Direct initialization
}
```

### 2.2 Replace Manager with Specific Services
**Current Issue**: Generic "Manager" types
```go
// BAD: Manager anti-pattern
type Manager struct {
    factory *Factory
    providers map[string]Provider
}
```

**Solution**: Domain-specific services
```go
// GOOD: Specific service
type AIService struct {
    claude *ClaudeProvider  // Concrete types
    gemini *GeminiProvider
}

func (s *AIService) GenerateText(ctx context.Context, prompt string) (string, error) {
    // Clear responsibility
}
```

## Priority 3: Fix Interface Design

### 3.1 Remove Interface Suffix
**Files to fix**:
- `/internal/storage/postgres/interface.go` → rename `ClientInterface` to `Storage`

### 3.2 Break Large Interfaces
**Current Issue**: 15+ method interfaces
```go
// BAD: Large interface
type Agent interface {
    GetID() string
    GetName() string
    GetDomain() Domain
    // ... 12 more methods
}
```

**Solution**: Small, focused interfaces
```go
// GOOD: Small interfaces
type Agent interface {
    Execute(ctx context.Context, req Request) (*Response, error)
}

type Named interface {
    Name() string
}
```

## Priority 4: Replace interface{} with Specific Types

### 4.1 Configuration
**Current Issue**: `map[string]interface{}` everywhere
```go
// BAD
config map[string]interface{}
```

**Solution**: Typed structs
```go
// GOOD
type ToolConfig struct {
    Timeout     time.Duration
    MaxRetries  int
    EnableCache bool
}
```

### 4.2 Use Generics Where Appropriate
```go
// GOOD: Type-safe result
type Result[T any] struct {
    Value T
    Error error
}
```

## Priority 5: Fix Package Organization

### 5.1 Merge Handler/Service Splits
**Current Structure**:
```
/server/tools/
├── enhanced_service.go
└── enhanced_handler.go
```

**Target Structure**:
```
/server/tools/
└── tools.go  // Both HTTP handling and business logic
```

### 5.2 Remove Empty Packages
- Delete packages that only contain interfaces or types
- Move types to where they're used

## Priority 6: Value vs Pointer Semantics

### 6.1 Use Value Receivers for Read-Only Methods
```go
// BAD: Unnecessary pointer
func (c *Config) GetTimeout() time.Duration {
    return c.timeout
}

// GOOD: Value receiver
func (c Config) Timeout() time.Duration {
    return c.timeout
}
```

### 6.2 Return Structs, Not Pointers
```go
// BAD: Returning pointer for small struct
func NewConfig() *Config {
    return &Config{timeout: 30 * time.Second}
}

// GOOD: Return value
func NewConfig() Config {
    return Config{timeout: 30 * time.Second}
}
```

## Implementation Order

1. **Week 1**: Remove base types and flatten hierarchies
   - Delete `/internal/core/agents/base.go`
   - Refactor tools to not use `BaseTool`
   - Simplify agent implementations

2. **Week 2**: Simplify factories and managers
   - Replace AI factory with direct constructors
   - Convert managers to specific services
   - Remove global registries

3. **Week 3**: Fix interfaces and types
   - Remove Interface suffix
   - Break large interfaces
   - Replace interface{} with concrete types

4. **Week 4**: Package reorganization
   - Merge handler/service files
   - Organize by feature, not type
   - Clean up imports

## Success Criteria

1. **No Base Types**: Zero files named `base.go`
2. **Small Interfaces**: No interface with >3 methods
3. **Type Safety**: <10% usage of `interface{}`
4. **Clear Packages**: Packages named by functionality
5. **Value Semantics**: Configs and small types use values

## Go Principles Checklist

- [ ] "Accept interfaces, return structs"
- [ ] "The bigger the interface, the weaker the abstraction"
- [ ] "Make the zero value useful"
- [ ] "Errors are values"
- [ ] "Don't communicate by sharing memory, share memory by communicating"
- [ ] "Concurrency is not parallelism"
- [ ] "Channels orchestrate; mutexes serialize"
- [ ] "Interface names should describe behavior"

## References

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Proverbs](https://go-proverbs.github.io/)
- Internal `golang_guide.md`