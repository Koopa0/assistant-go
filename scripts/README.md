# Development Scripts

This directory contains useful scripts for development, testing, and maintenance tasks.

## Available Scripts

### check-code-quality.sh
Comprehensive code quality analysis including:
- Static analysis with staticcheck
- Security scanning with gosec  
- Code formatting checks
- Import organization validation
- Linting with golangci-lint

Usage:
```bash
./scripts/check-code-quality.sh
```

### quick-check.sh
Fast development checks for daily use:
- Compilation verification
- Go vet analysis
- Code formatting
- Basic linting (with typecheck disabled for Go 1.24 compatibility)

Usage:
```bash
./scripts/quick-check.sh
```

### upgrade-dependencies.sh
Automated dependency management:
- Updates Go dependencies to latest versions
- Resolves version conflicts
- Runs tests to verify compatibility
- Updates go.mod and go.sum files

Usage:
```bash
./scripts/upgrade-dependencies.sh
```

### verify-dependencies.go
Standalone dependency verification program:
- Validates all required dependencies are available
- Tests critical package imports
- Verifies database driver functionality
- Checks AI provider SDK compatibility

Usage:
```bash
go run ./scripts/verify-dependencies.go
```

## Script Categories

### Quality Assurance
- `check-code-quality.sh`: Comprehensive quality analysis
- `quick-check.sh`: Fast development checks

### Dependency Management  
- `upgrade-dependencies.sh`: Dependency updates
- `verify-dependencies.go`: Dependency verification

## Integration with Make

These scripts are integrated with the project Makefile:

```bash
make quality-check    # Runs check-code-quality.sh
make quick-check      # Runs quick-check.sh  
make upgrade-deps     # Runs upgrade-dependencies.sh
```

## CI/CD Integration

Scripts are designed for both local development and CI/CD environments:
- Proper exit codes for automation
- Colored output for local use
- Plain text output for CI systems
- Configurable verbosity levels