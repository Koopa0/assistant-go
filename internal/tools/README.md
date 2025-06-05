# Tools Registry and Execution

This package provides an extensible tool registry and execution framework for the Assistant. It enables the system to dynamically discover, register, and execute various development tools with type safety and proper error handling.

## Overview

The tools package implements a plugin-like architecture where tools can be:
- **Dynamically registered** at runtime
- **Type-safe** with structured input/output
- **Context-aware** with execution tracking
- **Health-monitored** with automatic recovery
- **Concurrency-safe** for parallel execution

## Architecture

```
tools/
├── base.go             # Core tool interfaces and base implementation
├── registry.go         # Tool registry and management
├── pipeline.go         # Tool execution pipeline
├── godev/              # Go development tools
│   ├── analyzer.go     # Go code analysis
│   ├── formatter.go    # Code formatting
│   ├── tester.go       # Test execution
│   └── builder.go      # Build automation
├── docker/             # Docker tools (placeholder)
├── k8s/                # Kubernetes tools (placeholder)
└── cloudflare/         # Cloudflare tools (placeholder)
```

## Core Interfaces

### Tool Interface
```go
type Tool interface {
    Name() string
    Description() string
    Execute(ctx context.Context, params Parameters) (Result, error)
    Validate(params Parameters) error
    Close() error
}
```

### Tool Factory
```go
type ToolFactory interface {
    Create(config ToolConfig) (Tool, error)
    SupportsType(toolType string) bool
}
```

## Available Tools

### Go Development Tools
- **go_analyzer**: Analyzes Go source code for complexity, imports, and patterns
- **go_formatter**: Formats Go code with gofmt and goimports
- **go_tester**: Executes Go tests with coverage reporting
- **go_builder**: Builds Go applications with optimization options
- **go_dependency_analyzer**: Analyzes Go module dependencies

## Usage Example

```go
// Create and configure registry
registry := tools.NewRegistry()

// Register Go development tools
if err := registry.RegisterGoTools(); err != nil {
    return fmt.Errorf("failed to register Go tools: %w", err)
}

// Get and execute a tool
tool, err := registry.GetTool("go_analyzer", nil)
if err != nil {
    return fmt.Errorf("failed to get analyzer tool: %w", err)
}

result, err := tool.Execute(ctx, tools.Parameters{
    "path": "/path/to/go/code",
    "recursive": true,
})
```

## Tool Development

To create a new tool:

1. **Implement the Tool interface**:
```go
type MyTool struct {
    name string
    config ToolConfig
}

func (t *MyTool) Name() string { return t.name }
func (t *MyTool) Execute(ctx context.Context, params Parameters) (Result, error) {
    // Implementation
}
```

2. **Create a factory**:
```go
type MyToolFactory struct{}

func (f *MyToolFactory) Create(config ToolConfig) (Tool, error) {
    return &MyTool{config: config}, nil
}
```

3. **Register with the registry**:
```go
registry.RegisterFactory("my_tool", &MyToolFactory{})
```

## Testing

The package includes comprehensive testing:
- Unit tests for individual tools
- Integration tests with real development environments
- Performance benchmarks for tool execution
- Edge case testing with invalid inputs