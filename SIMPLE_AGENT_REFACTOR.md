# Simplified Agent Architecture Refactoring

## Principle: Keep It Simple

Instead of 3 different agent packages, we'll have just ONE.

## Proposed Structure

```
internal/
└── agent/                    # ALL agent-related code here
    ├── agent.go             # Core interface and types
    ├── base.go              # Base implementation
    ├── development.go       # Development agent
    ├── database.go          # Database agent
    ├── manager.go           # Agent manager
    └── langchain_adapter.go # Adapter for LangChain (if needed)
```

That's it. One package, clear organization.

## What Goes Where

### agent.go
```go
// Core interface - simple and practical
type Agent interface {
    Execute(ctx context.Context, request Request) (*Response, error)
    Name() string
    Type() AgentType
}

// Request/Response types that work for everyone
type Request struct {
    Query       string
    Context     map[string]interface{}
    Parameters  map[string]interface{} // For extensibility
}

type Response struct {
    Result      string
    Success     bool
    Error       error
    Metadata    map[string]interface{} // For extensibility
}
```

### base.go
- BaseAgent struct
- Common functionality
- Tool management

### Specific agents (development.go, database.go, etc.)
- Embed BaseAgent
- Add specific behavior

### langchain_adapter.go
- ONLY if we need LangChain compatibility
- Simple adapter pattern
- Converts between our types and LangChain types

## Migration Plan

1. **Create `/internal/agent/`**
2. **Move the most practical implementation there** (from `/internal/agents/`)
3. **Add useful features from other implementations**
4. **Delete everything else**:
   - `/internal/core/agents/`
   - `/internal/agents/`
   - `/internal/langchain/agents/`

## Benefits

1. **One place to look** - No confusion about which agent to use
2. **Simple imports** - `agent.Agent` not `core.Agent` vs `agents.Agent`
3. **Clear ownership** - One package owns all agent logic
4. **Easy to understand** - New developers know where to find things
5. **Less code** - No duplicate types or interfaces

## What We're NOT Doing

- NOT creating multiple packages for agents
- NOT over-abstracting with too many interfaces
- NOT maintaining parallel implementations
- NOT creating deep hierarchies

## Decision

Keep it simple. One agent package. Clear types. Practical implementation.