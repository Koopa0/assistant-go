# Development Guide

## Getting Started

### Prerequisites

- Go 1.24+ 
- PostgreSQL 17+ with pgvector extension
- Make
- Docker (optional, for containerized development)

### Setup

1. Clone the repository:
```bash
git clone https://github.com/koopa0/assistant-go.git
cd assistant-go
```

2. Install dependencies:
```bash
make setup
```

3. Configure environment:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Start PostgreSQL:
```bash
docker-compose up -d postgres
```

5. Run migrations:
```bash
make migrate-up
```

6. Start development server:
```bash
make dev
```

## Code Organization

### Package Structure

Following Go best practices, packages are organized by feature:

```
/internal/
├── assistant/      # Core orchestration
├── conversation/   # Conversation management
├── memory/        # Memory system
├── ai/            # AI provider integration
├── tools/         # Development tools
└── platform/      # Infrastructure
    ├── server/    # HTTP/WebSocket servers
    ├── storage/   # Database access
    └── observability/
```

### Key Principles

1. **Direct Integration**: No unnecessary adapters or wrappers
2. **Value Semantics**: Use values for small structs
3. **Error Context**: Always wrap errors with `fmt.Errorf` and `%w`
4. **Type Safety**: No `map[string]interface{}` in business logic

## Making Changes

### Adding a New Feature

1. Create a new package under `/internal/`:
```go
// /internal/myfeature/service.go
package myfeature

type Service struct {
    queries *sqlc.Queries
    logger  *slog.Logger
}

func NewService(queries *sqlc.Queries, logger *slog.Logger) *Service {
    return &Service{queries: queries, logger: logger}
}
```

2. Define interfaces only if needed by consumers:
```go
// Small, focused interface
type MyFeatureReader interface {
    Get(ctx context.Context, id string) (*Item, error)
}
```

3. Add HTTP handlers if needed:
```go
// /internal/myfeature/http.go
func NewHTTPHandler(service *Service) *HTTPHandler {
    return &HTTPHandler{service: service}
}

func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("GET /myfeature/{id}", h.handleGet)
}
```

### Working with the Database

1. Add SQL queries to `/internal/platform/storage/postgres/queries/`:
```sql
-- name: GetFeatureItem :one
SELECT * FROM feature_items WHERE id = $1;
```

2. Generate Go code:
```bash
make sqlc-generate
```

3. Use generated queries:
```go
item, err := s.queries.GetFeatureItem(ctx, id)
if err != nil {
    return nil, fmt.Errorf("get feature item: %w", err)
}
```

### Testing

1. Write table-driven tests:
```go
func TestService_Get(t *testing.T) {
    tests := []struct {
        name    string
        id      string
        want    *Item
        wantErr bool
    }{
        {
            name: "valid item",
            id:   "123",
            want: &Item{ID: "123", Name: "Test"},
        },
        {
            name:    "not found",
            id:      "999",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

2. Use real implementations, not mocks:
```go
// Use in-memory database for tests
db := setupTestDB(t)
service := NewService(sqlc.New(db), testLogger)
```

3. Run tests:
```bash
make test
```

## Quality Checks

Always run quality checks before committing:

```bash
# Quick check (required)
make quick-check

# Full quality check (recommended)
make quality-check
```

## Common Patterns

### Error Handling
```go
// Always wrap errors with context
if err := s.doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Custom errors for business logic
type NotFoundError struct {
    ID string
}

func (e NotFoundError) Error() string {
    return fmt.Sprintf("item not found: %s", e.ID)
}
```

### Configuration
```go
// Use structured config, not maps
type Config struct {
    Port     int    `env:"PORT" default:"8080"`
    LogLevel string `env:"LOG_LEVEL" default:"info"`
}
```

### Logging
```go
// Use structured logging
logger.Info("Processing request",
    slog.String("id", requestID),
    slog.Int("items", len(items)),
    slog.Duration("elapsed", elapsed))
```

## Debugging

### Enable Debug Logging
```bash
export LOG_LEVEL=debug
make dev
```

### Database Queries
```bash
# Enable query logging
export DATABASE_LOG_QUERIES=true
```

### Performance Profiling
```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

## Contributing

1. Follow Go conventions and project patterns
2. Write tests for new functionality
3. Update documentation as needed
4. Run quality checks before submitting
5. Keep commits focused and well-described

## Resources

- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [sqlc Documentation](https://docs.sqlc.dev/)
- [pgx Documentation](https://github.com/jackc/pgx)