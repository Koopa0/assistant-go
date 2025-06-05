# Detailed Architecture Issues with Code Examples

## 1. Agent System Triplication

### Issue: Three Different Agent Interfaces

**Location 1: `/internal/core/agents/agent.go`**
```go
type Agent interface {
    Execute(ctx context.Context, request Request) (*Response, error)
    Name() string
    Type() AgentType
}

type Request struct {
    ID        string
    Type      RequestType
    Content   string
    Context   map[string]interface{}
    Priority  Priority
    Timeout   time.Duration
    UserID    string
    SessionID string
    CreatedAt time.Time
}
```

**Location 2: `/internal/agents/agent.go`**
```go
type Agent interface {
    Execute(ctx context.Context, request Request) (*Response, error)
    Name() string
    Type() AgentType
}

type Request struct {
    Query       string
    Context     map[string]interface{}
    Tools       []string
    MaxSteps    int
    Temperature float64
}
```

**Location 3: `/internal/langchain/agents/base.go`**
```go
type AgentRequest struct {
    Query       string
    Context     map[string]interface{}
    Tools       []string
    MaxSteps    int
    Temperature float64
    Metadata    map[string]interface{}
}
```

### Impact
- Same method names with different signatures
- Incompatible Request/Response types
- Cannot interchange implementations

## 2. Memory System Fragmentation

### Issue: Multiple Memory Architectures

**Core Memory Interface (`/internal/core/memory/memory.go`):**
```go
type Memory interface {
    Store(ctx context.Context, item *MemoryItem) error
    Retrieve(ctx context.Context, query MemoryQuery) ([]*MemoryItem, error)
    Update(ctx context.Context, id string, updates map[string]interface{}) error
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, criteria SearchCriteria) ([]*MemoryItem, error)
    GetGraph(ctx context.Context) (*MemoryGraph, error)
    CreateRelation(ctx context.Context, relation *Relation) error
    TraverseRelations(ctx context.Context, startID string, options TraversalOptions) ([]*MemoryItem, error)
}
```

**Practical Memory (`/internal/memory/store.go`):**
```go
type Store struct {
    working   *WorkingMemory
    shortTerm ShortTermMemory
    longTerm  LongTermMemory
    db        postgres.DB
    logger    *slog.Logger
}
// Implements only basic Store/Retrieve, no graph features
```

**LangChain Memory (`/internal/langchain/memory/`):**
- Separate shortterm.go, longterm.go
- Different interfaces
- No integration with core Memory

## 3. Tool Interface Proliferation

### Issue: Three Different Tool Interfaces

**Tools Package (`/internal/tools/registry.go`):**
```go
type Tool interface {
    Name() string
    Description() string
    Parameters() *ToolParametersSchema
    Execute(ctx context.Context, input *ToolInput) (*ToolResult, error)
    Health(ctx context.Context) error
    Close(ctx context.Context) error
}
```

**Agents Package (`/internal/agents/agent.go`):**
```go
type Tool interface {
    Name() string
    Execute(ctx context.Context, input string) (string, error)
}
```

**LangChain Adapter (`/internal/langchain/tools/`):**
- Adapts internal tools to LangChain interface
- Different Execute signature

## 4. Handler/Service Pattern Chaos

### Issue: Inconsistent HTTP Handler Organization

**Pattern 1: Separate Handler + Service**
```
/internal/server/tools/
├── http.go          # HTTP handlers
├── tools.go         # Service implementation
├── enhanced_handler.go
└── enhanced_service.go
```

**Pattern 2: Just Service**
```
/internal/server/chat/
├── http.go          # HTTP handlers
└── chat.go          # Service implementation
```

**Pattern 3: All in http.go**
```
/internal/server/system/
└── http.go          # Everything mixed
```

## 5. Context Management Confusion

### Issue: Multiple Context Systems

**Core Context (`/internal/core/context/`):**
- Engine, Personal, Semantic, Temporal, Workspace contexts
- Rich relationship modeling
- Complex initialization

**Assistant Context (`/internal/assistant/context.go`):**
- ContextManager for conversations
- Different purpose, similar name

**Request Contexts:**
- Agent requests have Context map[string]interface{}
- Tool inputs have Context fields
- HTTP handlers use standard context.Context

## 6. Database Interface Duplication

### Issue: Multiple Database Access Patterns

**Interface Definition (`/internal/storage/postgres/interface.go`):**
```go
type DB interface {
    Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error)
    QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row
    Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error)
    // ... more methods
}
```

**SQLC Generated (`/internal/storage/postgres/sqlc/`):**
- Separate Queries struct
- Different method signatures
- Some services use DB, others use Queries

**Direct SQL in Services:**
Some services write SQL directly instead of using SQLC

## 7. Configuration Sprawl

### Issue: Config Types Everywhere

**Main Config (`/internal/config/config.go`):**
```go
type Config struct {
    Mode        string
    Server      Server
    Database    Database
    AI          AI
    Tools       Tools
    // ...
}
```

**Types Config (`/internal/types/config.go`):**
Additional config types (if exists)

**Component Configs:**
- Each major component has own config structures
- No unified config management

## 8. Import Cycle Risks

### Issue: Packages Importing Each Other

**Example Risk Areas:**
- core packages importing implementation packages
- Implementation packages importing core interfaces
- Circular dependencies between agent, memory, and context

## Recommendations Priority Matrix

### P0 - Immediate (Block Release)
1. **Consolidate Agent System**
   - Keep `/internal/agents/` as the single implementation
   - Move useful features from core and langchain
   - Create single Agent interface

2. **Unify Memory Architecture**
   - Keep `/internal/memory/` as base
   - Add graph features if needed
   - Create LangChain adapters

3. **Fix Handler/Service Pattern**
   - Standardize on http.go + service.go pattern
   - Separate HTTP concerns from business logic

### P1 - Short Term (Next Sprint)
1. **Tool Interface Consolidation**
   - Single Tool interface in `/internal/tools/`
   - Remove duplicate definitions

2. **Context Management Cleanup**
   - Define clear context boundaries
   - Use standard context.Context where possible

3. **Database Access Standardization**
   - Always use SQLC for queries
   - Consistent interface usage

### P2 - Long Term (Tech Debt)
1. **Package Reorganization**
   - Follow functional cohesion
   - Reduce package count

2. **Configuration Unification**
   - Single config loading mechanism
   - Environment-based overrides

3. **Documentation**
   - Architecture decision records
   - Package relationship diagrams

## Migration Strategy

### Phase 1: Stop the Bleeding
- No new duplicate implementations
- Mark deprecated interfaces
- Document which to use

### Phase 2: Consolidation
- Merge implementations
- Update all usages
- Remove duplicates

### Phase 3: Cleanup
- Refactor package structure
- Update documentation
- Add architecture tests

## Success Metrics
- Single implementation per concept
- No circular dependencies
- Consistent patterns throughout
- Clear package boundaries
- Reduced code duplication by 50%+