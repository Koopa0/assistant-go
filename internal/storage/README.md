# Storage Systems

## Overview

The Storage Systems provide a comprehensive data persistence layer for the Assistant intelligent development companion. Built on PostgreSQL 17+ with pgvector extension, it offers type-safe database access through sqlc, intelligent query optimization, and seamless integration with the memory and context systems. The architecture supports both relational and vector data, enabling sophisticated semantic search capabilities.

## Architecture

```
internal/storage/
├── postgres/
│   ├── client.go              # Database client and connection management
│   ├── interface.go           # Storage interfaces and contracts
│   ├── migrations.go          # Database migration management
│   ├── mock_client.go         # Mock client for testing
│   ├── sqlc_client.go         # SQLC-generated client wrapper
│   ├── sqlc_improvements.go   # Enhanced SQLC functionality
│   ├── types.go              # Custom database types
│   ├── examples_test.go       # Usage examples
│   ├── storage_test.go.bak    # Storage tests (backup)
│   ├── migrations/            # SQL migration files
│   ├── queries/               # SQL query definitions
│   ├── schema/                # Database schema definitions
│   └── sqlc/                  # SQLC-generated code
└── cache/
    └── (cache implementations)
```

## Key Features

### 🗄️ **PostgreSQL 17+ Optimization**
- **Advanced Features**: Leverages PostgreSQL 17's performance improvements
- **pgvector Support**: Native vector operations for embeddings
- **Connection Pooling**: Optimized pgx v5 connection management
- **Intelligent Indexing**: B-tree, GIN, GiST, BRIN index strategies

### 🔒 **Type-Safe Database Access**
- **SQLC Integration**: 100% type-safe generated queries
- **No Raw SQL**: All queries validated at compile time
- **Generic Collections**: pgx v5 generic row collection
- **Custom Types**: Rich domain-specific type mappings

### 🎯 **Production Features**
- **Migration System**: Version-controlled schema evolution
- **Health Monitoring**: Connection pool and query performance tracking
- **Query Optimization**: Automatic EXPLAIN ANALYZE integration
- **Transaction Management**: ACID compliance with intelligent batching

## Core Components

### Storage Interface

```go
type Storage interface {
    // Connection management
    Connect(ctx context.Context) error
    Close() error
    Ping(ctx context.Context) error
    
    // Transaction management
    BeginTx(ctx context.Context, opts *sql.TxOptions) (Transaction, error)
    
    // Repository access
    Conversations() ConversationRepository
    Messages() MessageRepository
    Embeddings() EmbeddingRepository
    Memory() MemoryRepository
    Tools() ToolRepository
    
    // Migration management
    Migrate(ctx context.Context) error
    MigrateDown(ctx context.Context, steps int) error
    
    // Health and metrics
    Health(ctx context.Context) (*HealthStatus, error)
    Metrics() *StorageMetrics
}
```

### Repository Pattern

```go
type Repository[T any] interface {
    // CRUD operations
    Create(ctx context.Context, entity T) (*T, error)
    Get(ctx context.Context, id string) (*T, error)
    Update(ctx context.Context, id string, entity T) (*T, error)
    Delete(ctx context.Context, id string) error
    
    // Batch operations
    CreateBatch(ctx context.Context, entities []T) ([]T, error)
    GetBatch(ctx context.Context, ids []string) ([]T, error)
    
    // Query operations
    List(ctx context.Context, filter Filter, opts ...ListOption) ([]T, error)
    Count(ctx context.Context, filter Filter) (int64, error)
    Exists(ctx context.Context, id string) (bool, error)
}
```

## PostgreSQL Client Implementation

### Connection Management

```go
type PostgresClient struct {
    pool      *pgxpool.Pool
    queries   *sqlc.Queries
    config    PostgresConfig
    logger    *slog.Logger
    metrics   *ClientMetrics
    
    // Schema knowledge
    schema    *SchemaKnowledge
    optimizer *QueryOptimizer
}

type PostgresConfig struct {
    // Connection settings
    DatabaseURL      string        `env:"DATABASE_URL" required:"true"`
    MaxConnections   int32         `env:"MAX_CONNECTIONS" default:"30"`
    MinConnections   int32         `env:"MIN_CONNECTIONS" default:"5"`
    MaxConnLifetime  time.Duration `env:"MAX_CONN_LIFETIME" default:"1h"`
    MaxConnIdleTime  time.Duration `env:"MAX_CONN_IDLE_TIME" default:"15m"`
    HealthCheckPeriod time.Duration `env:"HEALTH_CHECK_PERIOD" default:"1m"`
    
    // Performance settings
    StatementTimeout time.Duration `env:"STATEMENT_TIMEOUT" default:"30s"`
    LockTimeout      time.Duration `env:"LOCK_TIMEOUT" default:"10s"`
    
    // Query optimization
    EnableQueryStats bool `env:"ENABLE_QUERY_STATS" default:"true"`
    SlowQueryThreshold time.Duration `env:"SLOW_QUERY_THRESHOLD" default:"1s"`
}

func NewPostgresClient(ctx context.Context, config PostgresConfig) (*PostgresClient, error) {
    // Parse connection config
    poolConfig, err := pgxpool.ParseConfig(config.DatabaseURL)
    if err != nil {
        return nil, fmt.Errorf("parsing config: %w", err)
    }
    
    // PostgreSQL 17 optimizations
    poolConfig.MaxConns = config.MaxConnections
    poolConfig.MinConns = config.MinConnections
    poolConfig.MaxConnLifetime = config.MaxConnLifetime
    poolConfig.MaxConnIdleTime = config.MaxConnIdleTime
    poolConfig.HealthCheckPeriod = config.HealthCheckPeriod
    
    // Connection configuration
    poolConfig.ConnConfig.Config.RuntimeParams["statement_timeout"] = 
        fmt.Sprintf("%d", config.StatementTimeout.Milliseconds())
    poolConfig.ConnConfig.Config.RuntimeParams["lock_timeout"] = 
        fmt.Sprintf("%d", config.LockTimeout.Milliseconds())
    
    // Query logging and tracing
    poolConfig.ConnConfig.Tracer = &pgx.QueryTracer{
        TraceQueryStart: func(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
            return context.WithValue(ctx, "query_start", time.Now())
        },
        TraceQueryEnd: func(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
            if start, ok := ctx.Value("query_start").(time.Time); ok {
                duration := time.Since(start)
                if duration > config.SlowQueryThreshold {
                    log.Warn("Slow query detected",
                        slog.String("query", data.SQL),
                        slog.Duration("duration", duration),
                        slog.Any("args", data.Args))
                }
            }
        },
    }
    
    // Create connection pool
    pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
    if err != nil {
        return nil, fmt.Errorf("creating pool: %w", err)
    }
    
    // Verify connection
    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return nil, fmt.Errorf("ping failed: %w", err)
    }
    
    client := &PostgresClient{
        pool:      pool,
        queries:   sqlc.New(pool),
        config:    config,
        logger:    slog.With("component", "postgres_client"),
        metrics:   NewClientMetrics(),
        schema:    NewSchemaKnowledge(),
        optimizer: NewQueryOptimizer(config),
    }
    
    // Load schema knowledge
    if err := client.loadSchemaKnowledge(ctx); err != nil {
        client.logger.Warn("Failed to load schema knowledge", 
            slog.String("error", err.Error()))
    }
    
    return client, nil
}
```

### Transaction Management

```go
type Transaction struct {
    tx      pgx.Tx
    queries *sqlc.Queries
    logger  *slog.Logger
    start   time.Time
}

func (pc *PostgresClient) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Transaction, error) {
    pgxOpts := pgx.TxOptions{}
    
    if opts != nil {
        pgxOpts.IsoLevel = pgx.TxIsoLevel(opts.Isolation)
        pgxOpts.AccessMode = pgx.ReadWrite
        if opts.ReadOnly {
            pgxOpts.AccessMode = pgx.ReadOnly
        }
    }
    
    tx, err := pc.pool.BeginTx(ctx, pgxOpts)
    if err != nil {
        return nil, fmt.Errorf("beginning transaction: %w", err)
    }
    
    return &Transaction{
        tx:      tx,
        queries: pc.queries.WithTx(tx),
        logger:  pc.logger.With("tx_id", generateTxID()),
        start:   time.Now(),
    }, nil
}

func (t *Transaction) Commit(ctx context.Context) error {
    duration := time.Since(t.start)
    t.logger.Info("Committing transaction", slog.Duration("duration", duration))
    
    if err := t.tx.Commit(ctx); err != nil {
        return fmt.Errorf("committing transaction: %w", err)
    }
    
    return nil
}

func (t *Transaction) Rollback(ctx context.Context) error {
    duration := time.Since(t.start)
    t.logger.Warn("Rolling back transaction", slog.Duration("duration", duration))
    
    if err := t.tx.Rollback(ctx); err != nil {
        return fmt.Errorf("rolling back transaction: %w", err)
    }
    
    return nil
}
```

## SQLC Integration

### Query Definitions

```sql
-- queries/conversations.sql

-- name: CreateConversation :one
INSERT INTO conversations (
    id, user_id, title, metadata, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetConversation :one
SELECT * FROM conversations 
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListConversations :many
SELECT * FROM conversations
WHERE user_id = $1 
  AND deleted_at IS NULL
  AND ($2::timestamptz IS NULL OR created_at >= $2)
  AND ($3::timestamptz IS NULL OR created_at <= $3)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: UpdateConversation :one
UPDATE conversations 
SET title = $2, 
    metadata = $3,
    updated_at = $4
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteConversation :exec
UPDATE conversations 
SET deleted_at = NOW() 
WHERE id = $1;
```

### SQLC Configuration

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/storage/postgres/queries"
    schema: "internal/storage/postgres/schema"
    gen:
      go:
        package: "sqlc"
        out: "internal/storage/postgres/sqlc"
        emit_json_tags: true
        emit_interface: true
        emit_empty_slices: true
        emit_exact_table_names: false
        emit_exported_queries: true
        emit_result_struct_pointers: true
        overrides:
          - db_type: "timestamptz"
            go_type: "time.Time"
          - db_type: "uuid"
            go_type: "string"
          - db_type: "jsonb"
            go_type: "json.RawMessage"
          - db_type: "vector"
            go_type:
              import: "github.com/pgvector/pgvector-go"
              type: "Vector"
```

### Enhanced SQLC Client

```go
type EnhancedSQLCClient struct {
    *sqlc.Queries
    pool *pgxpool.Pool
    
    // Query optimization
    explainer *QueryExplainer
    cache     *QueryCache
    
    // Monitoring
    stats     *QueryStats
    tracer    trace.Tracer
}

func (ec *EnhancedSQLCClient) GetConversationWithStats(ctx context.Context, id string) (*Conversation, *QueryStats, error) {
    // Start tracing
    ctx, span := ec.tracer.Start(ctx, "GetConversation")
    defer span.End()
    
    start := time.Now()
    
    // Check cache first
    if cached, found := ec.cache.Get(cacheKey("conversation", id)); found {
        span.SetAttributes(attribute.Bool("cache_hit", true))
        return cached.(*Conversation), nil, nil
    }
    
    // Execute query with EXPLAIN ANALYZE in development
    var conv *Conversation
    var err error
    
    if ec.isDevelopment() {
        plan, planErr := ec.explainer.ExplainQuery(ctx, 
            "SELECT * FROM conversations WHERE id = $1", id)
        if planErr == nil {
            ec.logger.Debug("Query plan", 
                slog.String("query", "GetConversation"),
                slog.Any("plan", plan))
        }
    }
    
    // Execute actual query
    conv, err = ec.Queries.GetConversation(ctx, id)
    duration := time.Since(start)
    
    // Record stats
    stats := &QueryStats{
        Query:    "GetConversation",
        Duration: duration,
        RowCount: 1,
        CacheHit: false,
    }
    ec.stats.Record(stats)
    
    if err != nil {
        span.RecordError(err)
        return nil, stats, err
    }
    
    // Cache result
    ec.cache.Set(cacheKey("conversation", id), conv, 5*time.Minute)
    
    return conv, stats, nil
}
```

## Migration System

### Migration Management

```go
type MigrationManager struct {
    db         *sql.DB
    migrations embed.FS
    logger     *slog.Logger
}

//go:embed migrations/*.sql
var migrationFiles embed.FS

func NewMigrationManager(db *sql.DB) *MigrationManager {
    return &MigrationManager{
        db:         db,
        migrations: migrationFiles,
        logger:     slog.With("component", "migration_manager"),
    }
}

func (mm *MigrationManager) Migrate(ctx context.Context) error {
    // Create migration table if not exists
    if err := mm.createMigrationTable(ctx); err != nil {
        return fmt.Errorf("creating migration table: %w", err)
    }
    
    // Get applied migrations
    applied, err := mm.getAppliedMigrations(ctx)
    if err != nil {
        return fmt.Errorf("getting applied migrations: %w", err)
    }
    
    // Get pending migrations
    pending, err := mm.getPendingMigrations(applied)
    if err != nil {
        return fmt.Errorf("getting pending migrations: %w", err)
    }
    
    if len(pending) == 0 {
        mm.logger.Info("No pending migrations")
        return nil
    }
    
    // Apply migrations in transaction
    tx, err := mm.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("beginning transaction: %w", err)
    }
    defer tx.Rollback()
    
    for _, migration := range pending {
        mm.logger.Info("Applying migration", 
            slog.String("version", migration.Version),
            slog.String("description", migration.Description))
        
        if err := mm.applyMigration(ctx, tx, migration); err != nil {
            return fmt.Errorf("applying migration %s: %w", 
                migration.Version, err)
        }
    }
    
    return tx.Commit()
}
```

### Migration Files

```sql
-- migrations/001_initial_schema.up.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgvector";

-- Conversations table
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);
CREATE INDEX idx_conversations_created_at ON conversations(created_at DESC);
CREATE INDEX idx_conversations_deleted_at ON conversations(deleted_at) 
    WHERE deleted_at IS NULL;

-- Messages table
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_conversation 
        FOREIGN KEY (conversation_id) 
        REFERENCES conversations(id)
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);

-- Embeddings table with pgvector
CREATE TABLE embeddings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content_type VARCHAR(50) NOT NULL,
    content_id UUID NOT NULL,
    embedding vector(1536) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_embeddings_content ON embeddings(content_type, content_id);
CREATE INDEX idx_embeddings_vector ON embeddings 
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);
```

## Vector Operations

### pgvector Integration

```go
type VectorStore struct {
    db      *pgxpool.Pool
    queries *sqlc.Queries
    config  VectorConfig
}

type VectorConfig struct {
    Dimensions      int     `default:"1536"`
    IndexLists      int     `default:"100"`
    SimilarityFunc  string  `default:"cosine"`
    SearchThreshold float32 `default:"0.7"`
}

func (vs *VectorStore) StoreEmbedding(ctx context.Context, embedding Embedding) error {
    // Convert to pgvector format
    vec := pgvector.NewVector(embedding.Vector)
    
    _, err := vs.queries.CreateEmbedding(ctx, sqlc.CreateEmbeddingParams{
        ID:          embedding.ID,
        ContentType: embedding.ContentType,
        ContentID:   embedding.ContentID,
        Embedding:   vec,
        Metadata:    embedding.Metadata,
        CreatedAt:   time.Now(),
    })
    
    if err != nil {
        return fmt.Errorf("storing embedding: %w", err)
    }
    
    return nil
}

func (vs *VectorStore) SearchSimilar(ctx context.Context, query []float32, limit int) ([]SearchResult, error) {
    // Create query vector
    queryVec := pgvector.NewVector(query)
    
    // Perform similarity search
    rows, err := vs.db.Query(ctx, `
        SELECT 
            id,
            content_type,
            content_id,
            embedding,
            metadata,
            1 - (embedding <=> $1) as similarity
        FROM embeddings
        WHERE 1 - (embedding <=> $1) > $2
        ORDER BY embedding <=> $1
        LIMIT $3
    `, queryVec, vs.config.SearchThreshold, limit)
    
    if err != nil {
        return nil, fmt.Errorf("searching embeddings: %w", err)
    }
    defer rows.Close()
    
    results, err := pgx.CollectRows(rows, pgx.RowToStructByName[SearchResult])
    if err != nil {
        return nil, fmt.Errorf("collecting results: %w", err)
    }
    
    return results, nil
}
```

### Vector Indexing Strategies

```go
func (vs *VectorStore) OptimizeIndex(ctx context.Context) error {
    // Analyze vector distribution
    stats, err := vs.analyzeVectorDistribution(ctx)
    if err != nil {
        return fmt.Errorf("analyzing distribution: %w", err)
    }
    
    // Determine optimal index parameters
    lists := vs.calculateOptimalLists(stats.TotalVectors)
    
    // Recreate index with optimized parameters
    _, err = vs.db.Exec(ctx, `
        DROP INDEX IF EXISTS idx_embeddings_vector;
        CREATE INDEX idx_embeddings_vector ON embeddings 
        USING ivfflat (embedding vector_cosine_ops)
        WITH (lists = $1);
    `, lists)
    
    if err != nil {
        return fmt.Errorf("recreating index: %w", err)
    }
    
    // Vacuum analyze for statistics update
    _, err = vs.db.Exec(ctx, "VACUUM ANALYZE embeddings")
    if err != nil {
        return fmt.Errorf("vacuum analyze: %w", err)
    }
    
    return nil
}

func (vs *VectorStore) calculateOptimalLists(totalVectors int64) int {
    // PostgreSQL recommendation: lists = max(64, sqrt(total_vectors))
    lists := int(math.Sqrt(float64(totalVectors)))
    if lists < 64 {
        lists = 64
    }
    if lists > 1000 {
        lists = 1000 // Cap at reasonable maximum
    }
    return lists
}
```

## Query Optimization

### Intelligent Query Analysis

```go
type QueryOptimizer struct {
    pool       *pgxpool.Pool
    analyzer   *QueryAnalyzer
    statistics *TableStatistics
    cache      *OptimizationCache
}

func (qo *QueryOptimizer) SuggestIndexes(ctx context.Context, slowQueries []SlowQuery) ([]IndexSuggestion, error) {
    suggestions := make([]IndexSuggestion, 0)
    
    for _, query := range slowQueries {
        // Analyze query plan
        plan, err := qo.analyzer.AnalyzePlan(ctx, query.SQL, query.Args)
        if err != nil {
            continue
        }
        
        // Check for sequential scans
        if seqScans := plan.FindSequentialScans(); len(seqScans) > 0 {
            for _, scan := range seqScans {
                suggestion := qo.suggestIndexForScan(scan)
                if suggestion != nil {
                    suggestions = append(suggestions, *suggestion)
                }
            }
        }
        
        // Check for missing join indexes
        if joins := plan.FindHashJoins(); len(joins) > 0 {
            for _, join := range joins {
                suggestion := qo.suggestJoinIndex(join)
                if suggestion != nil {
                    suggestions = append(suggestions, *suggestion)
                }
            }
        }
    }
    
    // Deduplicate and prioritize suggestions
    return qo.prioritizeSuggestions(suggestions), nil
}

func (qo *QueryOptimizer) AutoCreateIndexes(ctx context.Context, suggestions []IndexSuggestion) error {
    for _, suggestion := range suggestions {
        if suggestion.Priority < PriorityHigh {
            continue
        }
        
        // Generate CREATE INDEX statement
        indexSQL := qo.generateIndexSQL(suggestion)
        
        // Create index concurrently to avoid blocking
        concurrentSQL := strings.Replace(indexSQL, "CREATE INDEX", 
            "CREATE INDEX CONCURRENTLY", 1)
        
        qo.logger.Info("Creating suggested index",
            slog.String("index", suggestion.Name),
            slog.String("table", suggestion.Table),
            slog.Any("columns", suggestion.Columns))
        
        if _, err := qo.pool.Exec(ctx, concurrentSQL); err != nil {
            return fmt.Errorf("creating index %s: %w", suggestion.Name, err)
        }
    }
    
    return nil
}
```

### Performance Monitoring

```go
type PerformanceMonitor struct {
    pool     *pgxpool.Pool
    metrics  *PerformanceMetrics
    alerter  *Alerter
    config   MonitorConfig
}

func (pm *PerformanceMonitor) MonitorQueries(ctx context.Context) {
    ticker := time.NewTicker(pm.config.CheckInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            stats, err := pm.collectQueryStats(ctx)
            if err != nil {
                pm.logger.Error("Failed to collect query stats", 
                    slog.String("error", err.Error()))
                continue
            }
            
            // Check for anomalies
            if anomalies := pm.detectAnomalies(stats); len(anomalies) > 0 {
                for _, anomaly := range anomalies {
                    pm.alerter.SendAlert(Alert{
                        Level:   AlertLevelWarning,
                        Type:    AlertTypePerformance,
                        Message: anomaly.Description,
                        Data:    anomaly,
                    })
                }
            }
            
            // Update metrics
            pm.metrics.Update(stats)
            
        case <-ctx.Done():
            return
        }
    }
}

func (pm *PerformanceMonitor) collectQueryStats(ctx context.Context) (*QueryStatistics, error) {
    var stats QueryStatistics
    
    // Get current query statistics
    err := pm.pool.QueryRow(ctx, `
        SELECT 
            COUNT(*) as total_queries,
            AVG(mean_exec_time) as avg_duration_ms,
            MAX(mean_exec_time) as max_duration_ms,
            SUM(calls) as total_calls
        FROM pg_stat_statements
        WHERE query NOT LIKE '%pg_stat_statements%'
    `).Scan(&stats.TotalQueries, &stats.AvgDuration, 
            &stats.MaxDuration, &stats.TotalCalls)
    
    if err != nil {
        return nil, fmt.Errorf("querying statistics: %w", err)
    }
    
    // Get slow queries
    rows, err := pm.pool.Query(ctx, `
        SELECT 
            query,
            calls,
            mean_exec_time,
            total_exec_time
        FROM pg_stat_statements
        WHERE mean_exec_time > $1
        ORDER BY mean_exec_time DESC
        LIMIT 10
    `, pm.config.SlowQueryThreshold.Milliseconds())
    
    if err != nil {
        return nil, fmt.Errorf("querying slow queries: %w", err)
    }
    defer rows.Close()
    
    stats.SlowQueries, err = pgx.CollectRows(rows, pgx.RowToStructByName[SlowQuery])
    if err != nil {
        return nil, fmt.Errorf("collecting slow queries: %w", err)
    }
    
    return &stats, nil
}
```

## Caching Layer

### Multi-Level Cache

```go
type CacheLayer struct {
    l1Cache   *MemoryCache    // In-memory LRU cache
    l2Cache   *RedisCache     // Distributed Redis cache
    l3Storage Storage         // Database fallback
    
    stats     *CacheStats
    config    CacheConfig
}

func (cl *CacheLayer) Get(ctx context.Context, key string) (interface{}, error) {
    // Check L1 cache
    if value, found := cl.l1Cache.Get(key); found {
        cl.stats.RecordHit(CacheLevelL1)
        return value, nil
    }
    
    // Check L2 cache
    if value, err := cl.l2Cache.Get(ctx, key); err == nil {
        cl.stats.RecordHit(CacheLevelL2)
        // Populate L1
        cl.l1Cache.Set(key, value, cl.config.L1TTL)
        return value, nil
    }
    
    // Fetch from storage
    value, err := cl.fetchFromStorage(ctx, key)
    if err != nil {
        cl.stats.RecordMiss()
        return nil, err
    }
    
    // Populate caches
    cl.populateCaches(ctx, key, value)
    
    return value, nil
}

func (cl *CacheLayer) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    // Set in all cache levels
    cl.l1Cache.Set(key, value, ttl)
    
    if err := cl.l2Cache.Set(ctx, key, value, ttl); err != nil {
        cl.logger.Warn("Failed to set L2 cache", 
            slog.String("key", key),
            slog.String("error", err.Error()))
    }
    
    return nil
}

func (cl *CacheLayer) Invalidate(ctx context.Context, pattern string) error {
    // Invalidate matching keys in all levels
    cl.l1Cache.InvalidatePattern(pattern)
    
    if err := cl.l2Cache.InvalidatePattern(ctx, pattern); err != nil {
        return fmt.Errorf("invalidating L2 cache: %w", err)
    }
    
    return nil
}
```

## Testing Support

### Mock Storage Client

```go
type MockStorageClient struct {
    conversations map[string]*Conversation
    messages      map[string][]*Message
    embeddings    map[string]*Embedding
    
    failureRate   float32
    latency       time.Duration
    mutex         sync.RWMutex
}

func NewMockStorageClient() *MockStorageClient {
    return &MockStorageClient{
        conversations: make(map[string]*Conversation),
        messages:      make(map[string][]*Message),
        embeddings:    make(map[string]*Embedding),
        failureRate:   0.0,
        latency:       0,
    }
}

func (m *MockStorageClient) SetFailureRate(rate float32) {
    m.failureRate = rate
}

func (m *MockStorageClient) SetLatency(latency time.Duration) {
    m.latency = latency
}

func (m *MockStorageClient) CreateConversation(ctx context.Context, conv Conversation) (*Conversation, error) {
    // Simulate latency
    if m.latency > 0 {
        time.Sleep(m.latency)
    }
    
    // Simulate failures
    if rand.Float32() < m.failureRate {
        return nil, errors.New("simulated failure")
    }
    
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    conv.ID = generateID()
    conv.CreatedAt = time.Now()
    conv.UpdatedAt = conv.CreatedAt
    
    m.conversations[conv.ID] = &conv
    return &conv, nil
}
```

## Configuration

### Storage Configuration

```yaml
storage:
  postgres:
    # Connection settings
    database_url: "${DATABASE_URL}"
    max_connections: 30
    min_connections: 5
    max_conn_lifetime: "1h"
    max_conn_idle_time: "15m"
    health_check_period: "1m"
    
    # Performance settings
    statement_timeout: "30s"
    lock_timeout: "10s"
    enable_query_stats: true
    slow_query_threshold: "1s"
    
    # Query optimization
    auto_explain:
      enabled: true
      log_analyze: true
      log_buffers: true
      log_timing: true
      min_duration: "100ms"
    
    # Connection pool tuning
    pool:
      max_lifetime_jitter: "1m"
      health_check_period: "30s"
      lazy_connect: false
    
  cache:
    l1:
      type: "memory"
      size: 10000
      ttl: "5m"
      
    l2:
      type: "redis"
      endpoint: "${REDIS_URL}"
      ttl: "1h"
      max_connections: 100
      
    invalidation:
      strategy: "event-based"
      patterns:
        - "conversation:*"
        - "message:*"
        - "embedding:*"
        
  migrations:
    auto_migrate: true
    directory: "migrations"
    table: "schema_migrations"
    lock_timeout: "10s"
    
  vector:
    dimensions: 1536
    index_lists: 100
    similarity_function: "cosine"
    search_threshold: 0.7
    
  monitoring:
    enable_metrics: true
    metrics_interval: "30s"
    slow_query_log: true
    query_stats: true
```

## Usage Examples

### Basic Operations

```go
func ExampleBasicOperations() {
    // Initialize storage
    storage, err := postgres.NewPostgresClient(context.Background(), config)
    if err != nil {
        log.Fatal(err)
    }
    defer storage.Close()
    
    // Create conversation
    conv, err := storage.Conversations().Create(context.Background(), Conversation{
        UserID: "user-123",
        Title:  "Database optimization discussion",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Add message
    msg, err := storage.Messages().Create(context.Background(), Message{
        ConversationID: conv.ID,
        Role:          "user",
        Content:       "How can I optimize my PostgreSQL queries?",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Store embedding
    embedding := []float32{0.1, 0.2, 0.3...} // 1536 dimensions
    err = storage.Embeddings().Store(context.Background(), Embedding{
        ContentType: "message",
        ContentID:   msg.ID,
        Vector:      embedding,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Created conversation: %s\n", conv.ID)
}
```

### Transaction Example

```go
func ExampleTransaction() {
    storage, _ := postgres.NewPostgresClient(context.Background(), config)
    
    // Start transaction
    tx, err := storage.BeginTx(context.Background(), &sql.TxOptions{
        Isolation: sql.LevelSerializable,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Perform operations in transaction
    err = func() error {
        // Create conversation
        conv, err := tx.Conversations().Create(context.Background(), Conversation{
            UserID: "user-456",
            Title:  "Transactional operations",
        })
        if err != nil {
            return err
        }
        
        // Add multiple messages
        for i := 0; i < 10; i++ {
            _, err := tx.Messages().Create(context.Background(), Message{
                ConversationID: conv.ID,
                Role:          "user",
                Content:       fmt.Sprintf("Message %d", i),
            })
            if err != nil {
                return err
            }
        }
        
        return nil
    }()
    
    if err != nil {
        tx.Rollback(context.Background())
        log.Fatal(err)
    }
    
    // Commit transaction
    if err := tx.Commit(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

### Vector Search

```go
func ExampleVectorSearch() {
    storage, _ := postgres.NewPostgresClient(context.Background(), config)
    vectorStore := storage.Vectors()
    
    // Generate query embedding
    queryText := "How to improve database performance?"
    queryEmbedding := generateEmbedding(queryText) // Your embedding function
    
    // Search similar embeddings
    results, err := vectorStore.SearchSimilar(context.Background(), 
        queryEmbedding, 10)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d similar results:\n", len(results))
    for _, result := range results {
        fmt.Printf("- ID: %s, Similarity: %.3f\n", 
            result.ContentID, result.Similarity)
    }
}
```

## Performance Best Practices

### Connection Pool Optimization

```go
func OptimizeConnectionPool(config *PostgresConfig) {
    // Calculate optimal pool size
    // Formula: connections = ((core_count * 2) + effective_spindle_count)
    cores := runtime.NumCPU()
    config.MaxConnections = int32(cores*2 + 1)
    
    // Set minimum connections to 25% of max
    config.MinConnections = config.MaxConnections / 4
    
    // Connection lifetime based on workload
    if config.HighThroughput {
        config.MaxConnLifetime = 5 * time.Minute
        config.MaxConnIdleTime = 1 * time.Minute
    } else {
        config.MaxConnLifetime = 1 * time.Hour
        config.MaxConnIdleTime = 15 * time.Minute
    }
}
```

### Query Batching

```go
func (s *Storage) BatchInsertMessages(ctx context.Context, messages []Message) error {
    const batchSize = 1000
    
    for i := 0; i < len(messages); i += batchSize {
        end := i + batchSize
        if end > len(messages) {
            end = len(messages)
        }
        
        batch := messages[i:end]
        
        // Use COPY for bulk insert
        conn, err := s.pool.Acquire(ctx)
        if err != nil {
            return fmt.Errorf("acquiring connection: %w", err)
        }
        defer conn.Release()
        
        _, err = conn.CopyFrom(
            ctx,
            pgx.Identifier{"messages"},
            []string{"id", "conversation_id", "role", "content", "created_at"},
            pgx.CopyFromSlice(len(batch), func(i int) ([]interface{}, error) {
                return []interface{}{
                    batch[i].ID,
                    batch[i].ConversationID,
                    batch[i].Role,
                    batch[i].Content,
                    batch[i].CreatedAt,
                }, nil
            }),
        )
        
        if err != nil {
            return fmt.Errorf("batch insert: %w", err)
        }
    }
    
    return nil
}
```

## Related Documentation

- [Memory Systems](../core/memory/README.md) - Memory persistence integration
- [Context Engine](../core/context/README.md) - Context storage
- [AI Embeddings](../ai/embeddings/README.md) - Vector generation
- [Configuration](../config/README.md) - Storage configuration