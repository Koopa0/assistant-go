# Public Packages

This directory contains public packages that can be imported and used by external applications. These packages provide reusable utilities and abstractions that are not specific to the Assistant application.

## Structure

```
pkg/
├── errors/             # Error handling utilities
├── retry/              # Retry mechanisms and backoff strategies
└── validation/         # Input validation utilities
```

## Packages

### errors/
Common error handling patterns and utilities:
- Error wrapping and unwrapping
- Error categorization and classification
- Structured error types
- Error context management

### retry/
Robust retry mechanisms for external service calls:
- Exponential backoff strategies
- Configurable retry policies
- Circuit breaker patterns
- Timeout management

### validation/
Input validation utilities for consistent data validation:
- Common validation rules
- Custom validator functions
- Structured validation errors
- Validation middleware

## Design Principles

### Public API
All packages in `pkg/` are designed to be imported by external applications:
- Stable and well-documented APIs
- Minimal dependencies
- Clear version compatibility
- Comprehensive examples

### Reusability
Packages focus on common patterns that can be reused across projects:
- Framework-agnostic implementations
- Configurable behavior
- Composable components
- Testable design

### Quality Standards
High standards for public packages:
- Extensive documentation
- Comprehensive test coverage
- Performance benchmarks
- API stability guarantees

## Usage Example

```go
import (
    "github.com/koopa0/assistant-go/pkg/errors"
    "github.com/koopa0/assistant-go/pkg/retry"
    "github.com/koopa0/assistant-go/pkg/validation"
)

// Error handling
if err := someOperation(); err != nil {
    return errors.Wrap(err, "operation failed")
}

// Retry with backoff
err := retry.Do(func() error {
    return externalServiceCall()
}, retry.WithMaxAttempts(3), retry.WithBackoff(retry.ExponentialBackoff))

// Input validation
validator := validation.New()
if err := validator.Validate(input); err != nil {
    return fmt.Errorf("validation failed: %w", err)
}
```

## Testing

Each package includes:
- Unit tests with high coverage
- Integration tests where applicable
- Benchmark tests for performance
- Example tests for documentation