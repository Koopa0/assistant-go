# Migration Guide: Simplifying Agents and Memory

This guide explains how to migrate from the complex dual implementations to the simplified, Go-idiomatic versions.

## Overview of Changes

### Before (Over-abstracted)
```
/internal/
├── core/
│   ├── agents/      # 800+ lines of theoretical abstractions
│   └── memory/      # Complex cognitive models
└── langchain/
    ├── agents/      # Practical but disconnected implementation
    └── memory/      # Database-backed but incompatible
```

### After (Simplified)
```
/internal/
├── agents/          # Single, practical implementation
└── memory/          # Unified memory system
```

## Key Principles Applied

1. **"Discover abstractions, don't create them"**
   - Removed premature abstractions
   - Interfaces emerged from actual use

2. **Value semantics over pointer semantics**
   - Structs are values that can be safely copied
   - Pointers only where mutation is needed

3. **Composition over inheritance**
   - BaseAgent is embedded, not inherited
   - No complex hierarchies

4. **Small interfaces**
   - Agent interface: 3 methods (was 15+)
   - Memory interface: 3 methods (was 10+)

## Migration Steps

### 1. Agent Migration

#### Old Complex Way
```go
// core/agents/base.go - Overly complex
type Agent interface {
    GetID() string
    GetName() string
    GetDomain() Domain
    GetCapabilities() []Capability
    GetConfidence() float64
    GetStatus() AgentStatus
    CanHandle(ctx context.Context, request *corecontext.ContextualRequest) (bool, float64)
    Execute(ctx context.Context, request *corecontext.ContextualRequest) (*AgentResponse, error)
    Collaborate(ctx context.Context, other Agent, request *corecontext.ContextualRequest) (*CollaborationResult, error)
    Learn(ctx context.Context, feedback *Feedback) error
    Adapt(ctx context.Context, environment *Environment) error
    GetState() AgentState
    UpdateState(state AgentState) error
    GetResources() Resources
    AllocateResources(resources Resources) error
    ReleaseResources() error
    Initialize(ctx context.Context, config AgentConfig) error
    Shutdown(ctx context.Context) error
}
```

#### New Simple Way
```go
// agents/agent.go - Practical and focused
type Agent interface {
    Execute(ctx context.Context, request Request) (*Response, error)
    Name() string
    Type() AgentType
}
```

#### Migration Example
```go
// Before
agent := core.NewBaseAgent("id", "name", core.DomainDevelopment, logger)
agent.AddCapability(core.Capability{
    Name: "code_analysis",
    Proficiency: 0.8,
})
agent.AllocateResources(resources)
response, err := agent.Execute(ctx, complexRequest)

// After
agent := agents.NewDevelopmentAgent(llm, logger)
response, err := agent.Execute(ctx, agents.Request{
    Query: "analyze this code",
})
```

### 2. Memory Migration

#### Old Dual System
```go
// Two incompatible systems
coreMemory := core.NewWorkingMemory()
langchainMemory := langchain.NewMemoryManager()

// Different types
coreMemory.Store(core.ItemTypeGoal, data)
langchainMemory.Store(langchain.MemoryTypeShortTerm, entry)
```

#### New Unified System
```go
// Single memory store
store := memory.NewStore(db, logger)

// Consistent types
store.Store(ctx, memory.Entry{
    Type:    memory.TypeWorking,
    UserID:  userID,
    Content: content,
})
```

### 3. Removing Unnecessary Abstractions

#### Remove Theoretical Concepts
- ❌ `AgentMemory` with episodic/semantic/procedural subdivisions
- ❌ `CollaborationResult` with conflict resolution
- ❌ `LearningState` with learning modes
- ❌ `PerformanceMetrics` with 10+ metrics
- ✅ Simple execution tracking and basic memory storage

#### Simplify Data Structures
```go
// Before: Complex nested structures
type AgentState struct {
    CurrentTask    *Task
    ActiveContext  *corecontext.ContextualRequest
    RecentActions  []Action
    Collaborations []CollaborationInfo
    Performance    PerformanceMetrics
    LearningState  LearningState
    LastUpdate     time.Time
}

// After: Just what's needed
type Response struct {
    Result        string
    Steps         []Step
    ToolsUsed     []string
    ExecutionTime time.Duration
    TokensUsed    int
}
```

## Benefits of Simplification

1. **Reduced Code Size**: ~80% less code to maintain
2. **Better Performance**: Less abstraction overhead
3. **Easier Testing**: Concrete types are easier to test
4. **Clear Purpose**: Each struct/interface has obvious use
5. **Go Idiomatic**: Follows community best practices

## Next Steps

1. Update imports from `core/agents` and `langchain/agents` to `agents`
2. Update imports from `core/memory` and `langchain/memory` to `memory`
3. Simplify agent creation and usage
4. Consolidate memory operations
5. Remove unused theoretical code

## Compatibility Note

The simplified versions maintain the essential functionality while removing theoretical overhead. Any advanced features can be added back when actually needed, following the principle of "discover abstractions through use."