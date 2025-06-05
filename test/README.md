# Testing Infrastructure

This directory contains the comprehensive testing infrastructure for the Assistant project, including unit tests, integration tests, end-to-end tests, and testing utilities.

## Structure

```
test/
├── e2e/                # End-to-end tests
│   ├── api_e2e_test.go
│   ├── assistant_e2e_test.go
│   └── cli_e2e_test.go
├── fixtures/           # Test data and fixtures
│   └── init.sql
├── integration/        # Integration tests
│   ├── assistant_integration_test.go
│   └── database_integration_test.go
└── testutil/           # Testing utilities and helpers
    ├── ai_mocks.go
    ├── database.go
    └── factories.go
```

## Test Categories

### Unit Tests
Located alongside source code with `*_test.go` naming:
- Fast execution (< 1 second per test)
- No external dependencies
- Isolated component testing
- High code coverage target (85-95%)

### Integration Tests (`integration/`)
Test component interactions with real dependencies:
- Database integration with testcontainers
- AI provider integration with real APIs
- Tool execution with actual binaries
- Memory system integration

### End-to-End Tests (`e2e/`)
Full application workflow testing:
- **API E2E**: REST API endpoint testing
- **CLI E2E**: Command-line interface testing
- **Assistant E2E**: Complete assistant workflows

## Testing Utilities (`testutil/`)

### AI Mocks (`ai_mocks.go`)
Mock implementations for AI providers:
- Predictable responses for testing
- Error simulation capabilities
- Performance testing support

### Database Utilities (`database.go`)
Database testing infrastructure:
- PostgreSQL testcontainers setup
- Schema migration for tests
- Test data cleanup

### Test Factories (`factories.go`)
Factory functions for creating test data:
- Valid and invalid test objects
- Randomized test data generation
- Builder pattern for complex objects

## Running Tests

### All Tests
```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

### Specific Test Categories
```bash
# Unit tests only
go test ./internal/... ./cmd/...

# Integration tests only
go test -tags=integration ./test/integration/...

# End-to-end tests only
go test -tags=e2e ./test/e2e/...
```

### Performance Tests
```bash
# Benchmark tests
go test -bench=. ./...

# Memory profiling
go test -memprofile=mem.prof ./...

# CPU profiling  
go test -cpuprofile=cpu.prof ./...
```

## Test Environment

### Database Setup
Tests use PostgreSQL with pgvector extension via testcontainers:
- Automatic container lifecycle management
- Isolated test databases
- Real database behavior testing

### AI Provider Setup
Tests support both mock and real AI providers:
- Mock providers for unit testing
- Real providers for integration testing (requires API keys)
- Configurable via environment variables

## Best Practices

### Test Organization
- Place unit tests next to source code
- Use descriptive test names
- Group related tests in subtests
- Use table-driven tests for multiple scenarios

### Test Data
- Use factories for creating test objects
- Avoid hardcoded test data
- Clean up test data after tests
- Use meaningful test data that reflects real usage

### Error Testing
- Test both success and failure scenarios
- Validate error messages and types
- Test edge cases and boundary conditions
- Use property-based testing for complex logic