# Comprehensive Testing Guide

This guide covers the complete testing strategy and implementation for the Assistant project, following Go 1.24+ best practices and the principles outlined in `golang_guide.md`.

## Testing Philosophy

Our testing approach follows Go's simplicity principle and the **"Discover abstractions, don't create them"** philosophy:

- **Use real databases with testcontainers instead of mocks**
- **Test behavior, not implementation details**
- **Rely on `sqlc.Querier` interface for database operations**
- **Prefer integration tests for real confidence**
- **Avoid over-abstraction and unnecessary patterns**

## Testing Architecture

### Test Organization

```
test/
├── testutil/           # Test utilities and helpers
│   ├── database.go     # Database test containers
│   ├── ai_mocks.go     # AI provider mocks
│   └── factories.go    # Test data factories
├── fixtures/           # Test data and SQL scripts
│   └── init.sql        # Database initialization
├── integration/        # Integration tests
│   ├── assistant_integration_test.go
│   └── database_integration_test.go
└── e2e/               # End-to-end tests
    └── (future E2E tests)
```

### Test Types

1. **Unit Tests** (`*_test.go` alongside source files)
   - Test individual functions and methods
   - Target: 85-95% statement coverage
   - Fast execution (< 1s per test)

2. **Integration Tests** (`test/integration/`)
   - Test component interactions
   - Use real dependencies (testcontainers)
   - Target: 70-80% statement coverage

3. **End-to-End Tests** (`test/e2e/`)
   - Test complete user journeys
   - Target: 60-70% statement coverage

4. **Property-Based Tests** (using Go's fuzzing)
   - Test invariants and properties
   - Discover edge cases automatically

## Quick Start

### Running Tests

```bash
# Run all unit tests
make test-unit

# Run integration tests (requires Docker)
make test-integration

# Run comprehensive test suite
make test-comprehensive

# Run specific test types
make test-race          # Race condition detection
make test-fuzz          # Fuzz testing
make test-security      # Security testing
make benchmark          # Performance benchmarks
```

### Test Coverage

```bash
# Generate coverage reports
make test-unit          # Creates coverage/unit.html
make test-integration   # Creates coverage/integration.html
make test-comprehensive # Creates coverage/combined.html
```

## Test Utilities

### Database Testing

Use `testutil.SetupTestDatabase()` for tests requiring a database:

```go
func TestWithDatabase(t *testing.T) {
    dbContainer, cleanup := testutil.SetupTestDatabase(t)
    defer cleanup()
    
    // Use dbContainer.URL for connections
    // Database includes pgvector extension and test schema
}
```

### Real Database Testing

We use real PostgreSQL databases with testcontainers instead of mocks. This approach:

- **Tests actual SQL behavior** - No surprises from mock mismatches
- **Validates migrations** - Ensures schema changes work correctly
- **Tests transactions** - Real rollback and commit behavior
- **Uses sqlc.Querier** - Same interface in tests and production

```go
func TestServiceWithRealDB(t *testing.T) {
    // Setup test database with testcontainers
    dbContainer, cleanup := testutil.SetupTestDatabase(t)
    defer cleanup()
    
    // Get connection pool
    pool, err := pgxpool.New(ctx, dbContainer.URL)
    require.NoError(t, err)
    defer pool.Close()
    
    // Create sqlc queries - same as production
    queries := sqlc.New(pool)
    
    // Test with real database operations
    service := user.NewService(queries, logger)
    
    // All operations hit real database
    user, err := service.CreateUser(ctx, &user.CreateUserRequest{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "secure123",
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, user.ID)
}
```

### AI Provider Testing

For AI providers, we use simple test doubles instead of complex mocks:

```go
type TestAIProvider struct {
    responses map[string]string
    calls     []string
}

func (p *TestAIProvider) Complete(ctx context.Context, prompt string) (string, error) {
    p.calls = append(p.calls, prompt)
    if resp, ok := p.responses[prompt]; ok {
        return resp, nil
    }
    return "default response", nil
}
```

### Test Data Factories

Use `testutil.NewTestDataFactory()` for consistent test data:

```go
func TestWithTestData(t *testing.T) {
    factory := testutil.NewTestDataFactory()
    
    request := factory.CreateAssistantRequest(
        testutil.WithMessage("test message"),
        testutil.WithUserID("test-user"),
    )
}
```

## Testing Best Practices

### 1. Database Testing with sqlc.Querier

```go
func TestUserOperations(t *testing.T) {
    // Setup real database
    db, cleanup := testutil.SetupTestDatabase(t)
    defer cleanup()
    
    queries := sqlc.New(db.Pool)
    service := user.NewService(queries, logger)
    
    t.Run("create_and_retrieve_user", func(t *testing.T) {
        // Test with real database - no mocks needed
        created, err := service.CreateUser(ctx, &user.CreateUserRequest{
            Username: "john",
            Email:    "john@example.com",
            Password: "password123",
        })
        require.NoError(t, err)
        
        // Verify in database
        retrieved, err := service.GetUserByID(ctx, created.ID)
        require.NoError(t, err)
        assert.Equal(t, created.Username, retrieved.Username)
    })
}
```

### 2. Table-Driven Tests

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantErr  bool
    }{
        {
            name:     "valid_input",
            input:    validInput,
            expected: expectedOutput,
            wantErr:  false,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 2. Parallel Testing

```go
func TestConcurrentOperations(t *testing.T) {
    tests := []struct {
        name string
        // test cases
    }{
        // test cases
    }
    
    for _, tt := range tests {
        tt := tt // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // Test implementation
        })
    }
}
```

### 3. Context and Timeouts

```go
func TestWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    result, err := service.Operation(ctx)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### 4. Error Path Testing

```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name          string
        setupError    func()
        expectedError string
    }{
        {
            name: "database_connection_error",
            setupError: func() {
                // Setup condition that causes DB error
            },
            expectedError: "database connection failed",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.setupError()
            
            _, err := service.Operation()
            assert.Error(t, err)
            assert.Contains(t, err.Error(), tt.expectedError)
        })
    }
}
```

## Fuzz Testing

### Writing Fuzz Tests

```go
func FuzzJSONProcessing(f *testing.F) {
    // Seed with valid examples
    f.Add(`{"name": "test", "value": 42}`)
    f.Add(`[]`)
    f.Add(`null`)
    
    f.Fuzz(func(t *testing.T, data []byte) {
        var v interface{}
        
        // Skip invalid JSON
        if err := json.Unmarshal(data, &v); err != nil {
            t.Skip()
        }
        
        // Test property: valid JSON should round-trip
        marshaled, err := json.Marshal(v)
        if err != nil {
            t.Errorf("Marshal failed: %v", err)
        }
        
        var v2 interface{}
        if err := json.Unmarshal(marshaled, &v2); err != nil {
            t.Errorf("Re-unmarshal failed: %v", err)
        }
    })
}
```

## Benchmark Testing

### Writing Benchmarks

```go
func BenchmarkOperation(b *testing.B) {
    // Setup outside the loop
    data := generateTestData()
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        result := operation(data)
        _ = result // Prevent optimization
    }
}

func BenchmarkOperationParallel(b *testing.B) {
    data := generateTestData()
    
    b.ResetTimer()
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            result := operation(data)
            _ = result
        }
    })
}
```

## Integration Testing

### Real Database Integration

All our tests use real databases by default. We don't separate "unit" and "integration" tests artificially:

```go
func TestConversationWorkflow(t *testing.T) {
    // Always use real database
    db, cleanup := testutil.SetupTestDatabase(t)
    defer cleanup()
    
    queries := sqlc.New(db.Pool)
    
    // Create services with real dependencies
    userService := user.NewService(queries, logger)
    convService := conversation.NewService(queries, logger)
    
    // Test complete workflow
    user, err := userService.CreateUser(ctx, &user.CreateUserRequest{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "password",
    })
    require.NoError(t, err)
    
    conv, err := convService.CreateConversation(ctx, user.ID, "Test Chat")
    require.NoError(t, err)
    
    msg, err := convService.AddMessage(ctx, conv.ID, "user", "Hello!")
    require.NoError(t, err)
    
    // Verify everything in real database
    messages, err := convService.GetMessages(ctx, conv.ID)
    require.NoError(t, err)
    assert.Len(t, messages, 1)
    assert.Equal(t, "Hello!", messages[0].Content)
}
```

### Database Setup Best Practices

```go
// test/testutil/database.go
func SetupTestDatabase(t *testing.T) (*TestDatabase, func()) {
    ctx := context.Background()
    
    // Start PostgreSQL container
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "pgvector/pgvector:pg17",
            ExposedPorts: []string{"5432/tcp"},
            Env: map[string]string{
                "POSTGRES_USER":     "test",
                "POSTGRES_PASSWORD": "test",
                "POSTGRES_DB":       "test",
            },
            WaitingFor: wait.ForSQL("5432/tcp", "postgres", func(host string, port nat.Port) string {
                return fmt.Sprintf("postgres://test:test@%s:%s/test?sslmode=disable", host, port.Port())
            }),
        },
        Started: true,
    })
    require.NoError(t, err)
    
    // Get connection details
    host, err := container.Host(ctx)
    require.NoError(t, err)
    
    port, err := container.MappedPort(ctx, "5432")
    require.NoError(t, err)
    
    // Connect and run migrations
    url := fmt.Sprintf("postgres://test:test@%s:%s/test?sslmode=disable", host, port.Port())
    pool, err := pgxpool.New(ctx, url)
    require.NoError(t, err)
    
    // Run migrations
    runMigrations(t, pool)
    
    return &TestDatabase{
        Container: container,
        Pool:      pool,
        URL:       url,
    }, func() {
        pool.Close()
        container.Terminate(ctx)
    }
}
```

### Testing Without Mocks

```go
//go:build integration

func TestDatabaseIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    dbContainer, cleanup := testutil.SetupTestDatabase(t)
    defer cleanup()
    
    // Test with real database
    storage, err := postgres.NewStorage(ctx, config, logger)
    require.NoError(t, err)
    defer storage.Close(ctx)
    
    // Test operations
}
```

### AI Integration

```go
//go:build integration

func TestAIIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Use real AI providers with test keys
    aiManager := ai.NewManager(config, logger)
    defer aiManager.Close(ctx)
    
    // Test real AI operations
}
```

## Performance Testing

### Memory Leak Detection

```go
func TestMemoryUsage(t *testing.T) {
    var m1, m2 runtime.MemStats
    
    runtime.GC()
    runtime.ReadMemStats(&m1)
    
    // Perform operations
    for i := 0; i < 1000; i++ {
        service.Operation()
    }
    
    runtime.GC()
    runtime.ReadMemStats(&m2)
    
    // Check memory growth
    memGrowth := m2.Alloc - m1.Alloc
    assert.Less(t, memGrowth, uint64(1024*1024), "Memory growth should be < 1MB")
}
```

### Race Condition Testing

```go
func TestConcurrentAccess(t *testing.T) {
    service := NewService()
    
    var wg sync.WaitGroup
    numGoroutines := 100
    
    wg.Add(numGoroutines)
    for i := 0; i < numGoroutines; i++ {
        go func() {
            defer wg.Done()
            service.Operation()
        }()
    }
    
    wg.Wait()
}
```

## Test Coverage Goals

- **Unit Tests**: 85-95% statement coverage
- **Integration Tests**: 70-80% statement coverage
- **Critical Systems**: 90%+ coverage with multiple test types
- **Overall Project**: 80%+ combined coverage

## Continuous Integration

Tests are automatically run in CI/CD pipeline:

1. **Unit tests** on every commit
2. **Integration tests** on pull requests
3. **E2E tests** on main branch
4. **Security tests** on releases
5. **Performance benchmarks** weekly

## Troubleshooting

### Common Issues

1. **Tests fail in CI but pass locally**
   - Check for race conditions: `make test-race`
   - Verify test isolation and cleanup

2. **Integration tests timeout**
   - Increase timeout values
   - Check Docker daemon status
   - Verify network connectivity

3. **Flaky tests**
   - Add proper synchronization
   - Use deterministic test data
   - Implement retry mechanisms for external dependencies

### Debug Commands

```bash
# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestSpecificFunction ./package

# Run tests with race detection
go test -race ./...

# Profile tests
go test -cpuprofile=cpu.prof -memprofile=mem.prof ./...
```

## Contributing

When adding new tests:

1. Follow the established patterns in existing tests
2. Use the test utilities in `test/testutil/`
3. Add integration tests for new features
4. Ensure tests are deterministic and isolated
5. Update this guide for new testing patterns

For questions or improvements to the testing infrastructure, please refer to the team or create an issue.
