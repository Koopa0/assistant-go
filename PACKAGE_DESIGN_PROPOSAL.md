# Package Design Proposal for Agent System Refactoring

## Current Problem
- 3 different agent implementations scattered across packages
- Flat structure vs deep hierarchy confusion
- No clear separation of concerns

## Proposed Package Structure

```
internal/
├── core/               # Core domain interfaces and types
│   ├── agent/         # Agent interfaces and base types
│   │   ├── agent.go   # Core Agent interface
│   │   ├── types.go   # Request, Response, Step types
│   │   └── capability.go # Capability definition
│   │
│   ├── memory/        # Memory interfaces
│   └── context/       # Context interfaces
│
├── agent/             # Concrete agent implementations
│   ├── base.go        # BaseAgent implementation
│   ├── development.go # DevelopmentAgent
│   ├── database.go    # DatabaseAgent
│   ├── manager.go     # Agent Manager
│   └── registry.go    # Agent Registry
│
├── integration/       # Integration with external systems
│   └── langchain/
│       ├── adapter/
│       │   ├── agent.go    # LangChain agent adapter
│       │   ├── memory.go   # LangChain memory adapter
│       │   └── tool.go     # LangChain tool adapter
│       └── service.go
│
└── server/           # HTTP handlers and services
    └── agent/
        ├── http.go   # HTTP handlers
        └── service.go # Business logic
```

## Design Principles

1. **Core Package**: Contains only interfaces and types
   - No implementations
   - Minimal dependencies
   - Stable contracts

2. **Implementation Packages**: Concrete implementations
   - Depend on core interfaces
   - Can have external dependencies
   - Business logic lives here

3. **Integration Package**: External system adapters
   - Isolates external dependencies
   - Clear boundaries
   - Easy to swap implementations

## Benefits

1. **Clear Separation**: Interfaces vs implementations
2. **Dependency Direction**: Implementation depends on core, not vice versa
3. **Testability**: Easy to mock interfaces
4. **Flexibility**: Can swap implementations
5. **Clarity**: Clear where to find things

## Migration Strategy

### Phase 1: Core Interfaces
1. Create `/internal/core/agent/` with clean interfaces
2. Define Agent, Request, Response, Capability types
3. No external dependencies in core

### Phase 2: Implementations
1. Move concrete agents to `/internal/agent/`
2. Implement core interfaces
3. Keep practical focus

### Phase 3: Integration
1. Create `/internal/integration/langchain/`
2. Build adapters for LangChain compatibility
3. Isolate LangChain dependencies

### Phase 4: Cleanup
1. Remove old implementations
2. Update imports
3. Fix tests

## Alternative Considered

### Option A: Everything in `/internal/agents/` (Too Flat)
- Pros: Simple
- Cons: Mixes interfaces with implementations

### Option B: Deep Hierarchy (Too Complex)
```
internal/
└── domain/
    └── agent/
        └── core/
            └── interfaces/
```
- Pros: Very organized
- Cons: Too many levels, hard to navigate

### Option C: By Feature (Current Approach - Chosen)
- Pros: Clear separation, reasonable depth
- Cons: Need discipline to maintain boundaries

## Decision

Go with Option C - balanced approach that:
- Separates interfaces from implementations
- Groups related functionality
- Maintains reasonable package depth
- Follows Go community practices