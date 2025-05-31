# Testing Framework

This directory contains the comprehensive testing infrastructure for the Assistant intelligent development companion.

## Testing Philosophy

Following Go 1.24+ best practices and the "discover abstractions, don't create them" principle:

- **Black-box testing** tests abstractions while understanding implementation
- **Real implementations** over mock interfaces where possible
- **Table-driven tests** with parallel execution
- **Property-based testing** using Go's built-in fuzzing
- **Integration testing** with real dependencies

## Directory Structure

### Test Utilities (`testutil/`)
Comprehensive testing utilities and helpers:
- **`database.go`** - Database test containers (PostgreSQL + pgvector)
- **`ai_mocks.go`** - AI provider mocks with configurable behavior
- **`factories.go`** - Test data factories for consistent data generation
- **`logger.go`** - Test-specific logging utilities

### Test Fixtures (`fixtures/`)
- **`init.sql`** - Database initialization for tests
- **Sample data** - Representative test data sets

### Integration Tests (`integration/`)
- **`assistant_integration_test.go`** - Core assistant functionality
- **`database_integration_test.go`** - Database operations
- **Real dependency testing** with testcontainers

### End-to-End Tests (`e2e/`)
- **`assistant_e2e_test.go`** - Complete user workflows
- **`cli_e2e_test.go`** - CLI interface testing
- **`api_e2e_test.go`** - API endpoint testing

## Testing Strategies

### 1. Unit Testing
Testing individual components in isolation:

```go
func TestProcessorExecute(t *testing.T) {
    tests := []struct {
        name     string
        input    ProcessorInput
        expected ProcessorOutput
        wantErr  bool
    }{
        {
            name: "valid_input",
            input: ProcessorInput{Query: "test query"},
            expected: ProcessorOutput{Result: "processed"},
            wantErr: false,
        },
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

### 2. Integration Testing
Testing component interactions with real dependencies:

```go
func TestDatabaseIntegration(t *testing.T) {
    // Use testcontainers for real PostgreSQL instance
    container := testutil.SetupTestDatabase(t)
    defer container.Cleanup()
    
    client := postgres.NewClient(container.ConnectionString())
    // Test real database operations
}
```

### 3. Property-Based Testing
Using Go 1.18+ fuzzing for property validation:

```go
func FuzzJSONProcessing(f *testing.F) {
    f.Add(`{"valid": "json"}`)
    f.Fuzz(func(t *testing.T, data []byte) {
        var v interface{}
        if err := json.Unmarshal(data, &v); err != nil {
            t.Skip() // Skip invalid JSON
        }
        
        // Property: valid JSON should round-trip
        marshaled, err := json.Marshal(v)
        if err != nil {
            t.Errorf("Marshal failed: %v", err)
        }
    })
}
```

### 4. Fake Implementations
Using test doubles instead of mocks:

```go
type FakeAIProvider struct {
    responses map[string]string
    callCount int
}

func (f *FakeAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
    f.callCount++
    if response, exists := f.responses[prompt]; exists {
        return response, nil
    }
    return "default response", nil
}
```

## Test Utilities

### Database Testing
```go
// Setup test database with migrations
container, cleanup := testutil.SetupTestDatabase(t)
defer cleanup()

db := container.DB
// Use real database for tests
```

### AI Mock Management
```go
// Create configurable AI mock
aiManager := testutil.NewMockAIManager(logger)
aiManager.SetResponse("test prompt", expectedResponse)
aiManager.SetError("error prompt", expectedError)
```

### Test Data Factory
```go
factory := testutil.NewTestDataFactory()

// Generate consistent test data
request := factory.CreateAssistantRequest(
    testutil.WithQuery("test message"),
    testutil.WithUserID("test-user"),
)
```

## Testing Commands

### Unit Tests
```bash
# Run all unit tests
make test-unit

# Run tests with coverage
make test-coverage

# Run race condition detection
make test-race
```

### Integration Tests
```bash
# Run integration tests (requires Docker)
make test-integration

# Run specific integration test
go test -tags=integration ./test/integration/
```

### End-to-End Tests
```bash
# Run complete E2E test suite
make test-e2e

# Run specific E2E test
go test -tags=e2e ./test/e2e/
```

### Comprehensive Testing
```bash
# Run all test types
make test-comprehensive

# Quick development testing
make test-quick
```

## Coverage Targets

### Unit Tests
- **Target**: 85-95% statement coverage
- **Critical systems**: 90%+ coverage
- **Focus**: Business logic and error paths

### Integration Tests
- **Target**: 70-80% statement coverage
- **Focus**: Component interactions
- **Real dependencies**: Database, external services

### Overall Project
- **Target**: 80%+ combined coverage
- **Quality gates**: Enforced in CI/CD
- **Trend monitoring**: Coverage should not decrease

## Best Practices

### Test Organization
1. **Group related tests** in table-driven format
2. **Use descriptive test names** that explain the scenario
3. **Test both success and error paths**
4. **Include edge cases and boundary conditions**

### Test Data Management
```go
// Use factories for consistent data
factory := testutil.NewTestDataFactory()
user := factory.CreateUser(testutil.WithRole("admin"))

// Use fixtures for complex scenarios
testutil.LoadFixtures(t, "user_scenarios.json")
```

### Assertion Patterns
```go
// Use testify for clear assertions
assert.NoError(t, err)
assert.Equal(t, expected, actual)
assert.Contains(t, response.Messages, expectedMessage)

// Custom assertions for domain objects
testutil.AssertUserEqual(t, expectedUser, actualUser)
```

### Context and Timeouts
```go
// Always use context with reasonable timeouts
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := service.ProcessWithContext(ctx, input)
```

## Continuous Integration

### Test Pipeline
1. **Unit tests** - Every commit
2. **Integration tests** - Pull requests
3. **E2E tests** - Main branch
4. **Performance tests** - Weekly schedule

### Quality Gates
- All tests must pass
- Coverage thresholds enforced
- No race conditions detected
- Security scanning passed

## Troubleshooting Tests

### Common Issues

#### Database Tests Failing
```bash
# Check Docker is running
docker ps

# Verify PostgreSQL container
docker logs <container_id>

# Reset test database
make test-db-reset
```

#### AI Mock Issues
```bash
# Verify mock configuration
go test -v -run TestAIMock ./test/testutil/

# Check mock response setup
```

#### Race Condition Detection
```bash
# Run with race detector
go test -race ./...

# Focus on specific package
go test -race -v ./internal/assistant/
```

This testing framework ensures the Assistant system maintains high quality and reliability through comprehensive testing at all levels.