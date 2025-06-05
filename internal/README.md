# Internal Packages

This directory contains all internal packages that implement the core functionality of the Assistant intelligent development companion.

## Architecture Overview

The internal packages are organized by functional domains and follow Go's best practices for package organization:

```
internal/
├── ai/                 # AI providers and embeddings
├── assistant/          # Core assistant orchestration
├── cli/                # Command-line interface components
├── config/             # Configuration management
├── core/               # Core intelligence systems
├── infrastructure/     # Infrastructure abstractions
├── langchain/          # LangChain Go integration
├── observability/      # Logging, metrics, and profiling
├── ratelimit/          # Rate limiting utilities
├── server/             # HTTP server and middleware
├── storage/            # Data persistence layer
├── testutil/           # Testing utilities
└── tools/              # Development tools and registry
```

## Key Principles

### Package Organization
- **Domain-driven**: Packages are organized by business domain rather than technical layer
- **Interface-driven**: Accept interfaces, return structs
- **Dependency-aware**: Clear dependency relationships with no circular imports

### Code Quality
- **Type Safety**: Leverages Go's type system for correctness
- **Error Handling**: Explicit error handling with context wrapping
- **Testing**: Comprehensive test coverage with various testing strategies
- **Documentation**: Clear package documentation and examples

## Core Systems

### Intelligence Layer (`core/`)
Implements the multi-agent intelligence system with memory, learning, and reasoning capabilities.

### Assistant Orchestration (`assistant/`)
Main orchestration engine that coordinates between different AI agents and tools.

### Storage Layer (`storage/`)
Unified data access layer with PostgreSQL integration and type-safe database operations.

### Tools Ecosystem (`tools/`)
Extensible tool registry supporting development tools like Go analyzers, formatters, and testers.

## Development Guidelines

When working with internal packages:

1. **Follow Go conventions**: Use standard Go package naming and organization
2. **Maintain boundaries**: Keep clear separation between package responsibilities  
3. **Test thoroughly**: Each package should have comprehensive tests
4. **Document interfaces**: Provide clear documentation for public APIs
5. **Handle errors**: Always provide meaningful error context