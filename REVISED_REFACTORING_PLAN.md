# Revised Agent System Refactoring Plan

## Design Philosophy
Based on the existing `/internal/core/` structure, the project already follows a layered architecture. We should respect and enhance this pattern.

## Proposed Approach

### 1. Keep and Enhance `/internal/core/agents/`
- This is the RIGHT place for agent interfaces
- Clean up to have only interfaces and core types
- Remove any implementation details

### 2. Consolidate Implementations in `/internal/agents/`
- This becomes the concrete implementation layer
- Implements interfaces from `/internal/core/agents/`
- Contains BaseAgent, DevelopmentAgent, etc.

### 3. Move LangChain Integration to `/internal/langchain/agents/`
- Keep it where it is, but refactor to use core interfaces
- Acts as an adapter layer
- Maintains LangChain-specific logic

## Revised Architecture

```
internal/
├── core/                    # Domain layer (interfaces & types)
│   └── agents/
│       ├── agent.go         # Core Agent interface
│       ├── types.go         # Request, Response, Priority types
│       ├── capability.go    # Capability interface
│       └── manager.go       # Manager interface
│
├── agents/                  # Implementation layer
│   ├── base.go             # BaseAgent implementing core.Agent
│   ├── development.go      # DevelopmentAgent
│   ├── database.go         # DatabaseAgent
│   ├── manager.go          # Manager implementation
│   └── builder.go          # Agent builder/factory
│
└── langchain/              # Integration layer
    └── agents/
        ├── adapter.go      # Adapts core agents to LangChain
        ├── development.go  # LangChain-specific development agent
        └── registry.go     # LangChain tool registry
```

## Key Design Decisions

1. **Three Clear Layers**:
   - **Core**: Pure interfaces, no dependencies
   - **Implementation**: Concrete types, depends on core
   - **Integration**: External system adapters

2. **Interface Segregation**:
   - Small, focused interfaces in core
   - Composed implementations in agents
   - Adapters for external systems

3. **Dependency Direction**:
   ```
   langchain/agents → agents → core/agents
   (integration)    (impl)    (interfaces)
   ```

## Implementation Steps

### Step 1: Clean `/internal/core/agents/`
```go
// agent.go - ONLY interfaces
type Agent interface {
    Execute(ctx context.Context, req Request) (*Response, error)
    Name() string
    Type() AgentType
}

type CapableAgent interface {
    Agent
    Capabilities() []Capability
}

type CollaborativeAgent interface {
    Agent
    CanHandle(ctx context.Context, req Request) (bool, float64)
    Collaborate(ctx context.Context, other Agent, req Request) (*Response, error)
}
```

### Step 2: Clean `/internal/agents/`
- Remove duplicate interface definitions
- Keep only concrete implementations
- Import from `core/agents`

### Step 3: Refactor `/internal/langchain/agents/`
- Import from `core/agents`
- Create adapters for core → LangChain
- Remove duplicate type definitions

## Benefits of This Approach

1. **Respects Existing Architecture**: Works with the grain, not against it
2. **Clear Boundaries**: Each package has a clear purpose
3. **Gradual Migration**: Can be done incrementally
4. **Maintains Compatibility**: Existing code continues to work
5. **Follows Go Idioms**: Interfaces in consumer packages

## What We're NOT Doing

1. **NOT creating new package structures** - using what exists
2. **NOT moving everything to one package** - maintaining separation
3. **NOT creating deep hierarchies** - keeping it practical
4. **NOT breaking existing code** - using adapters

## Next Steps

1. Start with `/internal/core/agents/` cleanup
2. Update `/internal/agents/` to use core interfaces
3. Create adapters in `/internal/langchain/agents/`
4. Test each step before proceeding
5. Remove duplicates only after everything works

This approach is more conservative but safer and more aligned with the existing architecture.