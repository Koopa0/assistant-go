# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with the Assistant intelligent development companion project.

## Project Overview

Assistant is an **intelligent development companion** that transcends traditional development tools by understanding context, learning from patterns, and actively participating in your development workflow. This is not just a tool collection but an evolving AI system that grows smarter with every interaction.

### Architectural Philosophy

- **Contextual Intelligence**: Every component maintains awareness of the broader development context
- **Collaborative Agency**: Specialized AI agents work together like a development team
- **Temporal Awareness**: Understands development as a journey, not disconnected events
- **Personal Adaptation**: Learns your development style and provides personalized assistance

## Build Commands

### Essential Development Commands

- `make setup` - Complete development environment setup (installs tools, dependencies)
- `make dev` - Start development server with hot reload
- `make build` - Build the application binary
- `make test` - Run all unit tests
- `make test-integration` - Run integration tests with real dependencies
- `make lint` - Run comprehensive linting with golangci-lint
- `make generate` - Generate sqlc code (run after SQL changes)

### Code Quality Commands (CRITICAL: Use these after every task)

- `make quick-check` - **ESSENTIAL**: Fast quality checks for daily development (recommended)
- `make quality-check` - **COMPREHENSIVE**: Full code quality analysis with security scanning
- `make verify` - **BASIC**: Simple verification (compilation, vet, format)
- `make verify-lint` - **CI-SAFE**: Compatibility-safe linting for CI/CD pipelines

### Database Operations

- `make migrate-up` - Apply database migrations
- `make migrate-down` - Rollback last migration
- `make sqlc-generate` - Generate type-safe SQL queries

### Quality Assurance

- `make test-coverage` - Run tests with coverage report
- `make lint-fix` - Auto-fix linting issues where possible
- `make fmt` - Format Go code

## Architecture Overview

### Core Design Philosophy

This project implements an **intelligent agent architecture** with pure Go patterns:

- **Agent-based intelligence** - Specialized AI agents collaborate like a development team
- **Context-aware systems** - Every component understands broader development context
- **Learning-driven adaptation** - Continuous learning from user patterns and outcomes
- **Interface-driven design** - "Accept interfaces, return structs" for maximum flexibility
- **Type-safe database access** - sqlc-generated queries only, never raw SQL
- **Explicit error handling** - wrapped errors with `fmt.Errorf` and `%w`
- **Structured logging** - Go 1.24+ `log/slog` throughout
- **Event-driven foundation** - All actions generate events for learning and automation

### Key Architectural Layers

#### 1. Intelligent Orchestration Core (`/internal/core/`)

The "brain" of Assistant that provides high-level reasoning and coordination:

- **Agent Network**: Supervisor agent coordinates specialist agents (development, database, infrastructure, etc.)
- **Context Engine**: Maintains living understanding of development environment
- **Learning System**: Recognizes patterns, learns preferences, and adapts behavior
- **Reasoning Engine**: Makes intelligent decisions based on context and experience

#### 2. Specialized Agent Network (`/internal/agents/`)

Domain experts that collaborate to solve complex problems:

- **Development Agent**: Understands code semantically, suggests refactorings, recognizes patterns
- **Database Agent**: Optimizes queries, understands schema relationships, predicts performance issues
- **Infrastructure Agent**: Manages K8s/Docker deployments, predicts resource needs, troubleshoots issues
- **Research Agent**: Synthesizes information, extracts knowledge, provides contextual insights

#### 3. Intelligent Memory Systems (`/internal/memory/`)

Multi-layered memory architecture mimicking human cognition:

- **Working Memory**: Current task context with limited capacity but fast access
- **Episodic Memory**: Development experiences with context, intent, and outcomes
- **Semantic Memory**: Factual knowledge organized as a graph for rich associations
- **Procedural Memory**: How-to knowledge for automating learned procedures
- **Prospective Memory**: Future intentions and planned actions

#### 4. Personal Knowledge Graph (`/internal/knowledge/`)

Comprehensive understanding of your development universe:

- **Personal Graph**: Projects, technologies, patterns, and their relationships
- **Knowledge Evolution**: Tracks how understanding and practices change over time
- **Pattern Extraction**: Identifies and catalogs development patterns from your work
- **Connection Discovery**: Finds relationships between seemingly unrelated elements

#### 5. Intelligent Tool Ecosystem (`/internal/tools/`)

Tools that understand intent and collaborate toward goals:

- **Semantic Framework**: Tools communicate capabilities and collaborate intelligently
- **SQL Tools**: Understand schema, optimize queries, predict performance impacts
- **Code Tools**: AST analysis, semantic understanding, intelligent refactoring
- **Infrastructure Tools**: K8s/Docker management with predictive capabilities
- **Integration Tools**: Cloudflare, CI/CD, and other service integrations

#### 6. Event-Driven Foundation (`/internal/infrastructure/`)

Everything built on events for learning and automation:

- **Event Store**: Captures all system activities for learning and analysis
- **Storage Systems**: Postgres with pgvector, time-series data, intelligent caching
- **Security Layer**: Privacy-preserving learning, encryption, audit trails
- **Performance Optimization**: Predictive caching, parallel processing, resource scheduling

#### 7. Integration Systems (`/internal/integration/`)

Bridges to external systems and protocols:

- **LangChain-Go Integration**: Current AI orchestration capabilities
- **MCP Preparation**: Ready for Model Context Protocol when Go support arrives
- **IDE Integration**: Language server protocol for editor enhancement
- **CI/CD Integration**: Intelligent build analysis and automation

### Entry Points

The application supports three modes via `/cmd/assistant/main.go`:

- `assistant serve` - API server mode
- `assistant cli` - Interactive CLI mode
- `assistant ask "query"` - Direct query mode

## Development Standards for Intelligent Systems

### Go Package Design Philosophy

**Follow Go's package naming and organization principles**:

**Package Naming Standards** (learned from golang_guide.md):

- **Avoid generic package names**: Don't use `handlers`, `models`, `utils`, `common` and other semantically meaningless package names
- **Function-oriented naming**: Package names should clearly express their functionality, such as `user`, `order`, `intelligence`, `memory`
- **Short and clear**: Follow standard library patterns, use simple nouns like `time`, `json`, `sql`
- **Lowercase without underscores**: Use `bufio` instead of `buf_io`

**Correct Package Organization**:

```go
// Wrong: Generic package organization
models/
├── user.go
├── agent.go
handlers/
├── user_handler.go
├── agent_handler.go

// Correct: Function-oriented organization
user/
├── user.go          // User domain types
├── service.go       // User business logic
├── memory.go        // User-related memory management
└── http.go          // User HTTP handling

agents/
├── base.go          // Base agent interface
├── development.go   // Development specialist agent
├── database.go      // Database specialist agent
└── manager.go       // Agent coordination management
```

### Agent Development Pattern

**Build agents that understand and collaborate**:

```go
// Agents implement semantic understanding, not just execution
type SpecialistAgent struct {
    domain      Domain
    expertise   []Capability
    confidence  ConfidenceEstimator
    memory      *DomainMemory
    tools       []SemanticTool
}

func (a *SpecialistAgent) ProcessIntent(ctx context.Context, intent Intent) (*Response, error) {
    // Understand the intent semantically
    understanding := a.analyzeIntent(intent)

    // Check confidence and collaborate if needed
    if understanding.Confidence < 0.8 {
        return a.requestCollaboration(ctx, intent)
    }

    // Execute with context awareness
    return a.executeWithContext(ctx, understanding)
}
```

### Semantic Tool Development

**Tools must understand intent and collaborate**:

```go
type SemanticTool interface {
    // Traditional execution
    Execute(ctx context.Context, params Parameters) (Result, error)

    // Semantic understanding
    UnderstandCapabilities() []Capability
    EstimateRelevance(intent Intent) float64
    SuggestCollaborations(intent Intent) []ToolCollaboration

    // Learning and adaptation
    LearnFromUsage(usage Usage) error
    AdaptToContext(ctx ToolContext) error
}
```

### Context-Aware Implementation

**Every component must maintain context awareness**:

```go
func (c *Component) ProcessRequest(ctx context.Context, req Request) (*Response, error) {
    // Enrich with context
    enriched := c.contextEngine.EnrichRequest(req)

    // Consider temporal context
    history := c.temporalContext.GetRelevantHistory(req)

    // Apply personal preferences
    preferences := c.personalContext.GetPreferences(req)

    // Process with full context awareness
    return c.processContextual(ctx, enriched, history, preferences)
}
```

### Learning-Driven Development

**Components must contribute to system learning**:

```go
func (s *Service) ExecuteAction(ctx context.Context, action Action) error {
    // Capture pre-execution context
    preContext := s.captureContext()

    // Execute the action
    result, err := s.execute(ctx, action)

    // Learn from the outcome
    s.learningSystem.LearnFromInteraction(Interaction{
        Action:     action,
        Context:    preContext,
        Result:     result,
        Error:      err,
        Timestamp:  time.Now(),
    })

    return err
}
```

### Database Operations with Intelligence

**Use sqlc with PostgreSQL 17+ best practices** (learned from golang_guide.md):

```go
// Intelligent SQL operations with pgx v5 and PostgreSQL 17
type IntelligentSQLClient struct {
    pool      *pgxpool.Pool
    queries   *sqlc.Queries
    schema    *SchemaKnowledge
    optimizer *QueryOptimizer
}

// Production-grade connection pool setup
func NewIntelligentSQLClient(ctx context.Context, databaseURL string) (*IntelligentSQLClient, error) {
    config, err := pgxpool.ParseConfig(databaseURL)
    if err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    // PostgreSQL 17 optimization settings
    config.MaxConns = 30                          // Based on CPU cores
    config.MinConns = 5                           // Maintain minimum connections
    config.MaxConnLifetime = time.Hour            // Connection rotation
    config.MaxConnIdleTime = time.Minute * 15     // Close idle connections
    config.HealthCheckPeriod = time.Minute        // Regular health checks

    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        return nil, fmt.Errorf("create pool: %w", err)
    }

    return &IntelligentSQLClient{
        pool:    pool,
        queries: sqlc.New(pool),
    }, nil
}

// Type-safe queries with intelligent optimization
func (c *IntelligentSQLClient) GetOrdersByStatus(ctx context.Context, status string) ([]Order, error) {
    // Use EXPLAIN ANALYZE for query analysis
    explainQuery := `EXPLAIN (ANALYZE, BUFFERS, TIMING)
        SELECT * FROM orders WHERE status = $1`

    // Intelligent index suggestions
    if c.optimizer.ShouldUseIndex("orders", "status") {
        // Suggest creating partial index
        c.logSuggestion("CREATE INDEX idx_orders_status ON orders(status) WHERE status = 'active'")
    }

    // Use pgx v5 generic row collection
    rows, err := c.pool.Query(ctx, query, status)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    return pgx.CollectRows(rows, pgx.RowToStructByName[Order])
}
```

### Go Function and API Design

**Clear function naming and error handling** (learned from golang_guide.md):

```go
// Use package name to provide context
package user

// Correct: Concise use of package name
func New() *User              // Call: user.New()
func ByID(id int) *User       // Call: user.ByID()

// Error wrapping with context
func ProcessFile(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return fmt.Errorf("failed to open file %s: %w", filename, err)
    }
    defer file.Close()

    data, err := io.ReadAll(file)
    if err != nil {
        return fmt.Errorf("failed to read file %s: %w", filename, err)
    }

    if err := validateData(data); err != nil {
        return fmt.Errorf("invalid data in file %s: %w", filename, err)
    }

    return nil
}

// Functional options pattern for evolving APIs
type Option func(*Server)

func WithTimeout(timeout time.Duration) Option {
    return func(s *Server) {
        s.timeout = timeout
    }
}

func NewServer(opts ...Option) *Server {
    s := &Server{
        timeout: 30 * time.Second, // Reasonable default value
    }

    for _, opt := range opts {
        opt(s)
    }

    return s
}
```

## Configuration System

Environment-based configuration (12-factor app methodology):

- **Required**: `CLAUDE_API_KEY` or `GEMINI_API_KEY`, `DATABASE_URL`
- **Development**: Copy `.env.example` to `.env`
- **YAML configs**: `/configs/development.yaml` and `/configs/production.yaml`
- Environment variables override YAML settings

## Testing Strategy

### Go Testing Philosophy (learned from golang_guide.md)

**"Discover abstractions rather than create abstractions" in testing**:

- Black box testing tests abstractions, but understanding underlying implementation provides more confidence
- Perfect abstractions are rare; tests passing for wrong reasons are more dangerous than tests failing for wrong reasons
- Tests should understand and trust the "translation" between interfaces and implementations

**Strategy for not using mock interfaces**:

```go
// Use test doubles and fake objects
type FakeEmailSender struct {
    SentEmails []Email
    ShouldFail bool
}

func (f *FakeEmailSender) SendEmail(email Email) error {
    if f.ShouldFail {
        return errors.New("email service unavailable")
    }
    f.SentEmails = append(f.SentEmails, email)
    return nil
}

// In-memory implementation
type InMemoryUserStore struct {
    users map[string]User
    mutex sync.RWMutex
}
```

**Advanced patterns for table-driven tests**:

```go
func TestConcurrentValidation(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected bool
    }{
        {"valid_email", "test@example.com", true},
        {"invalid_email", "not-an-email", false},
    }

    for _, tt := range tests {
        tt := tt // Critical: capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            result := ValidateEmail(tt.input)
            if result != tt.expected {
                t.Errorf("got %v; want %v", result, tt.expected)
            }
        })
    }
}
```

**Fuzz testing in Go 1.18+**:

```go
func FuzzJSONMarshaling(f *testing.F) {
    // Use valid JSON examples as seeds
    f.Add(`{"name": "test", "value": 42}`)
    f.Add(`[]`)
    f.Add(`null`)

    f.Fuzz(func(t *testing.T, data []byte) {
        var v interface{}

        // Skip invalid JSON
        if err := json.Unmarshal(data, &v); err != nil {
            t.Skip()
        }

        // Property: valid JSON should be re-marshalable
        marshaled, err := json.Marshal(v)
        if err != nil {
            t.Errorf("Marshal failed: %v", err)
        }
    })
}
```

### Testing Organization

- **Unit tests**: Alongside implementation files (`*_test.go`)
- **Integration tests**: `/test/integration/` with real dependencies (testcontainers)
- **E2E tests**: `/test/e2e/` for complete user journeys
- **Test utilities**: `/test/testutil/` for common helpers
- **Coverage targets**:
  - Unit tests: 85-95% statement coverage
  - Integration tests: 70-80% statement coverage
  - Critical systems: 90%+ coverage

## Key Dependencies

- `github.com/jackc/pgx/v5` - PostgreSQL driver (not database/sql)
- `github.com/tmc/langchaingo` - LangChain Go implementation
- `github.com/pgvector/pgvector-go` - Vector operations for RAG
- `gopkg.in/yaml.v3` - Configuration parsing

## Development Workflow

1. **Environment Setup**: `make setup` (installs sqlc, golangci-lint, air)
2. **Database Setup**: PostgreSQL 15+ with pgvector extension required
3. **Migration**: `make migrate-up` before first run
4. **Development**: `make dev` for server mode, `make run-cli` for CLI mode
5. **Code Generation**: Run `make generate` after SQL changes
6. **Testing**: `make test` for unit tests, `make test-integration` for full testing
7. **Quality Checks**: `make lint` and `make fmt` before commits

## Critical Implementation Guidelines for Intelligent Systems

### Core Principles for Assistant Development

**🧠 Intelligence-First Design**: Every component must contribute to the system's overall intelligence

- Implement semantic understanding, not just syntactic processing
- Enable learning from every interaction and outcome
- Maintain context awareness across all operations
- Design for collaboration between agents and tools

**🤝 Agent Collaboration**: Build systems that work together like a development team

- Implement confidence estimation and collaboration protocols
- Enable knowledge sharing between specialist agents
- Design for emergent behaviors through agent interaction
- Support dynamic team formation based on task requirements

**📚 Memory and Learning**: Create systems that remember and evolve

- Capture all interactions with context and outcomes
- Implement multi-layered memory (working, episodic, semantic, procedural)
- Enable pattern recognition and preference learning
- Support temporal awareness and evolution tracking

**🔒 Privacy-Preserving Intelligence**: Learn while respecting privacy

- Keep all personal data and learning local by default
- Implement anonymization for any optional data sharing
- Provide granular control over learning and synchronization
- Ensure audit trails for all intelligent decisions

### Technical Implementation Standards

**Database Access with Intelligence**:

- Use sqlc-generated queries only, never raw SQL
- Implement schema understanding and query optimization
- Enable performance prediction and resource planning
- Support intelligent migration suggestions

**Error Handling with Context**:

- Always wrap errors with meaningful context using `%w` verb
- Include contextual information for learning systems
- Enable error pattern recognition and prevention
- Support intelligent error recovery suggestions

**Event-Driven Architecture**:

- Generate events for all significant actions
- Enable real-time learning and adaptation
- Support automation opportunity detection
- Maintain complete audit trails for intelligence

**Configuration with Intelligence**:

- Environment vars > YAML files (12-factor methodology)
- Support dynamic configuration based on learned preferences
- Enable context-aware configuration suggestions
- Maintain configuration evolution history

### Observability Standards (learned from golang_guide.md)

**OpenTelemetry implementation priorities**:

- **Traces**: Stable (production-ready) - implement first
- **Metrics**: Beta (production-ready) - implement RED method
- **Logs**: Experimental - use slog with Loki integration

**Performance overhead control**:

```go
// Intelligent sampling strategy
func newProductionSampler() trace.Sampler {
    return trace.ParentBased(
        trace.TraceIDRatioBased(0.01), // 1% base sampling
        trace.WithRemoteParentSampled(trace.AlwaysSample()),
        trace.WithRemoteParentNotSampled(trace.NeverSample()),
    )
}

// RED method implementation
type REDMetrics struct {
    requestsTotal    metric.Int64Counter    // Rate
    requestDuration  metric.Float64Histogram // Duration
    requestErrors    metric.Int64Counter     // Errors
}
```

**Loki label strategy**:

- Only use low cardinality labels (service, environment, version)
- Avoid high cardinality labels (user_id, request_id)
- Include detailed information in log content

### Performance Optimization (learned from golang_guide.md)

**Profile-Guided Optimization (PGO)**:

```bash
# Collect production profile (recommended 6+ minutes)
curl -o production.pprof "http://localhost:6060/debug/pprof/profile?seconds=360"

# Apply PGO
cp production.pprof ./default.pgo
go build  # Automatically detect and use PGO
```

**Continuous performance analysis**:

```go
// Use Pyroscope for continuous analysis
pyroscope.Start(pyroscope.Config{
    ApplicationName: "assistant",
    ProfileTypes: []pyroscope.ProfileType{
        pyroscope.ProfileCPU,
        pyroscope.ProfileAllocObjects,
        pyroscope.ProfileInuseObjects,
        pyroscope.ProfileGoroutines,
    },
})
```

### Development Workflow for Intelligent Systems

1. **Design Phase**: Consider intelligence implications

   - How will this component contribute to overall understanding?
   - What learning opportunities does it create?
   - How will it collaborate with other agents/tools?
   - What context does it need and provide?

2. **Implementation Phase**: Build with intelligence in mind

   - Implement semantic interfaces, not just functional ones
   - Add context awareness and learning capabilities
   - Enable collaboration and confidence estimation
   - Include privacy and security considerations

3. **Testing Phase**: Validate intelligent behaviors

   - Test learning and adaptation capabilities
   - Verify collaboration between components
   - Validate context awareness and temporal understanding
   - Ensure privacy protection and security

4. **Integration Phase**: Connect to the intelligent ecosystem
   - Register capabilities with the orchestration system
   - Enable knowledge sharing with other components
   - Connect to event streams for learning
   - Integrate with memory and knowledge systems

## LangChain Integration

The project now has complete LangChain integration:

### VectorStore Integration

- **PGVectorStore** (`/internal/langchain/vectorstore/pgvector.go`) implements LangChain's VectorStore interface
- Uses PostgreSQL with pgvector extension for semantic search
- Supports document storage, similarity search, and retrieval operations

### Document Processing

- **DocumentProcessor** (`/internal/langchain/documentloader/loader.go`) handles file loading and text splitting
- Supports multiple file types: `.txt`, `.md`, `.go`, `.py`, `.js`, `.ts`, `.yaml`, `.json`, `.html`, `.pdf`
- Uses LangChain's text splitters (RecursiveCharacter, Markdown, etc.)
- Can process files, directories, URLs, and string content

### Tool Adapters

- **LangChainToolAdapter** (`/internal/tools/langchain/adapter.go`) bridges custom tools to LangChain interface
- Converts string inputs to structured tool calls
- Provides tool registry for agent integration
- Creates agent-specific tool sets

### Memory System

- **Enhanced LongTermMemory** supports both LangChain vectorstore and direct database access
- Uses embeddings for semantic memory retrieval
- Backward compatible with existing memory implementation

### RAG Implementation

- **EnhancedRAGChain** (`/internal/langchain/chains/rag_enhanced.go`) provides advanced RAG capabilities
- Document ingestion from files, directories, URLs, and strings
- Native LangChain RAG chain integration with fallback to custom implementation
- Source attribution and query result tracking

## Security Configuration Management (learned from golang_guide.md)

### Environment Variables vs Configuration Files

**Use environment variables for**:

- Sensitive data (API keys, database passwords, JWT signing keys)
- Environment-specific values (hostnames, ports, deployment targets)
- 12-factor app compliance requirements
- Containerized deployments (Docker, Kubernetes)

**Hybrid approach: Security-first pattern**:

```go
func LoadConfig() (*Config, error) {
    cfg := &Config{}

    // 1. Load base configuration from file
    if err := cleanenv.ReadConfig("config.yaml", cfg); err != nil {
        return nil, fmt.Errorf("config file: %w", err)
    }

    // 2. Override with environment variables (including secrets)
    if err := cleanenv.ReadEnv(cfg); err != nil {
        return nil, fmt.Errorf("environment: %w", err)
    }

    // 3. Validate critical security settings
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("validation: %w", err)
    }

    return cfg, nil
}
```

### Key Rotation Implementation

```go
type RotatingKeys struct {
    current  string
    previous string
    mutex    sync.RWMutex
}

func (rk *RotatingKeys) ValidateSignature(token string) error {
    rk.mutex.RLock()
    defer rk.mutex.RUnlock()

    // Try current key first
    if err := validateWithKey(token, rk.current); err == nil {
        return nil
    }

    // Fall back to previous key during rotation window
    if rk.previous != "" {
        return validateWithKey(token, rk.previous)
    }

    return errors.New("token validation failed")
}
```

## ⚠️ CRITICAL CODE QUALITY WORKFLOW

**IMPORTANT**: After completing ANY task, ALWAYS run code quality checks:

1. **After every code change**: `make quick-check`
2. **Before committing**: `make quick-check` MUST pass
3. **Before major features**: `make quality-check`
4. **If checks fail**: Fix issues immediately, don't accumulate technical debt

### Quality Check Commands Summary:

```bash
# Daily development (REQUIRED after every task)
make quick-check

# Comprehensive analysis (weekly/before releases)
make quality-check

# Basic verification (CI/CD)
make verify

# Linting only
make verify-lint
```

### Known Compatibility Issues:

- **Go 1.24.2 + golangci-lint v1.55.2**: typecheck disabled due to compatibility
- **Demo mode**: Some tests expected to fail without database
- **StaticCheck**: May panic with Go 1.24.2, handled gracefully in scripts

### Quality Standards:

- ✅ **MUST PASS**: Compilation, go vet, essential linting
- ⚠️ **SHOULD ADDRESS**: Security warnings, code style issues
- 📊 **TARGET**: >70% test coverage, binary <50MB

See `/docs/CODE_QUALITY.md` for complete quality guidelines.

## Key Practices Summary (learned from golang_guide.md)

### Package Design Principles

1. **Organize by function, not type**: Avoid `models`, `handlers` and other generic package names
2. **Use package names to provide context**: Reduce function name repetition
3. **Keep packages focused**: Each package should have a clear single responsibility

### Database Optimization

1. **Leverage PostgreSQL 17 new features**: Vacuum optimizations, WAL performance improvements
2. **Choose the right index types**: B-tree, GIN, GiST, BRIN each have their use cases
3. **Use pgx v5 type-safe features**: Generic row collection, named parameters

### Observability Practices

1. **Start with tracing**: Most mature signal, provides immediate debugging value
2. **Sampling is critical**: Use head-based sampling for performance, tail-based for completeness
3. **Keep Loki labels low cardinality**: Use structured logging for detailed information

### Testing Strategy

1. **Prefer real implementations over mocks**
2. **Test behavior, not implementation details**
3. **Table-driven tests with subtests** for comprehensive coverage
4. **Fuzz testing** for property-based validation

### Performance Optimization

1. **Measurement-driven development**: Analyze first, optimize specific bottlenecks
2. **PGO provides automatic optimization benefits**: Collect production profiles and apply
3. **Continuous profiling**: Use Pyroscope or Parca for production monitoring

### Security Configuration

1. **Separation of concerns**: Store configuration structure in files, secrets in environment/external storage
2. **Defense in depth**: Multiple layers of security controls
3. **Key rotation**: Automatic rotation and graceful key transitions

### API Design

1. **Accept interfaces, return structs**
2. **Keep interfaces small and focused**
3. **Use functional options** for evolving APIs
4. **Provide meaningful error context**

These principles and practices represent the collective wisdom of the Go community, combining official documentation, expert guidance, and real production experience. When applying these principles, consider specific use cases and team needs, and apply them flexibly to the Assistant intelligent development companion implementation.
