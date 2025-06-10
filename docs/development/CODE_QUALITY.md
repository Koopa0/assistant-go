# Code Quality Assurance

This document describes the code quality tools and processes for the Assistant project.

## Available Quality Check Tools

### 1. Quick Check (Recommended for Development)
```bash
make quick-check
# or
./scripts/quick-check.sh
```

**Purpose**: Fast essential checks for daily development
**Time**: ~10-30 seconds
**Checks**:
- ✅ Go modules integrity
- ✅ Code compilation
- ✅ Go vet analysis
- ✅ Code formatting
- ✅ Binary build
- ✅ Essential linting (compatibility-safe)

**When to Use**: 
- Before committing changes
- After making significant code changes
- Daily development workflow

### 2. Basic Verification
```bash
make verify
```

**Purpose**: Simple verification for CI/CD
**Time**: ~5-15 seconds
**Checks**:
- ✅ Code compilation
- ✅ Go vet
- ✅ Code formatting

### 3. Comprehensive Quality Check
```bash
make quality-check
# or
./scripts/check-code-quality.sh
```

**Purpose**: Complete code quality analysis
**Time**: ~2-5 minutes
**Checks**:
- ✅ All basic checks
- ✅ Security analysis (gosec)
- ✅ Static analysis (staticcheck)
- ✅ Ineffective assignments
- ✅ Spelling checks
- ✅ Vulnerability scanning
- ✅ Test coverage
- ✅ Race condition detection

**When to Use**:
- Before releases
- Weekly quality reviews
- After major refactoring

### 4. Individual Quality Tools

#### Linting
```bash
make lint              # Standard linting
make lint-fix          # Auto-fix issues
make verify-lint       # Compatibility-safe linting
```

#### Testing
```bash
make test              # Run all tests
make test-short        # Quick tests only
make test-integration  # Integration tests
make test-coverage     # Generate coverage report
```

#### Security
```bash
make security-scan     # Security analysis
make deps-check        # Dependency vulnerabilities
```

#### Code Formatting
```bash
make fmt               # Format code
```

## Quality Standards

### Compilation
- ✅ **MUST**: Code must compile without errors
- ✅ **MUST**: All packages must build successfully

### Code Analysis
- ✅ **MUST**: Pass `go vet` without errors
- ✅ **MUST**: No critical linting issues
- ⚠️ **SHOULD**: Address most linting warnings
- ⚠️ **SHOULD**: Maintain consistent code style

### Testing
- ✅ **MUST**: Core functionality tests pass
- ⚠️ **SHOULD**: Maintain test coverage >70%
- ⚠️ **SHOULD**: No race conditions in tests

### Security
- ✅ **MUST**: No high-severity security issues
- ⚠️ **SHOULD**: Address medium-severity findings
- ⚠️ **SHOULD**: Keep dependencies up to date

### Performance
- ✅ **MUST**: Binary builds successfully
- ⚠️ **SHOULD**: Binary size <50MB for distribution
- ⚠️ **SHOULD**: No obvious performance issues

## Known Compatibility Issues

### Go 1.24.2 + golangci-lint v1.55.2
**Issue**: Typecheck linter has compatibility problems
**Workaround**: We disable typecheck and rely on `go build` for compilation verification
**Status**: Tracked in our `.golangci.yml` configuration

### Demo Mode Testing
**Issue**: Some tests fail when running without database (demo mode)
**Workaround**: Expected behavior - tests require real database for full functionality
**Status**: Normal operation

### StaticCheck Panics
**Issue**: staticcheck may panic with certain Go 1.24.2 configurations
**Workaround**: We handle panics gracefully and continue with other checks
**Status**: Third-party tool compatibility issue

## Integration with Development Workflow

### Pre-Commit Checklist
1. Run `make quick-check` ✅
2. Ensure all essential checks pass ✅
3. Fix any critical issues ✅
4. Commit changes ✅

### Pre-Push Checklist
1. Run `make quick-check` ✅
2. Run `make test-short` ✅
3. Ensure CI will pass ✅
4. Push changes ✅

### Pre-Release Checklist
1. Run `make quality-check` ✅
2. Review all findings ✅
3. Address critical and high-priority issues ✅
4. Update documentation if needed ✅
5. Test in production-like environment ✅

### Daily Development
```bash
# Start of day - verify environment
make quick-check

# During development - frequent checks
make verify

# End of day - comprehensive check
make quality-check
```

## Troubleshooting

### "Code doesn't compile"
```bash
go build ./...         # Check specific errors
go mod tidy           # Fix module issues
make clean && make build  # Clean rebuild
```

### "Linting issues"
```bash
make lint-fix         # Auto-fix what's possible
make fmt              # Fix formatting
golangci-lint run --help  # Check specific rules
```

### "Tests failing"
```bash
make test-short       # Run quick tests only
go test -v ./...      # Verbose test output
make test-integration # Check integration tests
```

### "Tools not found"
```bash
make install-tools    # Install missing tools
make setup           # Full environment setup
```

## Custom Configuration

### Adding New Checks
1. Edit `scripts/check-code-quality.sh`
2. Add your check function
3. Call it in the main execution flow
4. Test with `make quality-check`

### Adjusting Standards
1. Edit `.golangci.yml` for linting rules
2. Edit `scripts/quick-check.sh` for essential checks
3. Update this document with new standards

### CI/CD Integration
```yaml
# Example GitHub Actions step
- name: Quality Check
  run: make quick-check

# Example GitLab CI step
quality_check:
  script:
    - make quick-check
```

## Best Practices

1. **Run quick-check frequently** during development
2. **Fix issues immediately** - don't accumulate technical debt
3. **Use comprehensive check** before important releases
4. **Monitor trends** in code quality metrics
5. **Keep tools updated** but test compatibility first
6. **Document exceptions** when standards can't be met

## Tool Versions

| Tool | Version | Purpose |
|------|---------|---------|
| golangci-lint | v1.55.2 | Comprehensive linting |
| staticcheck | latest | Static analysis |
| gosec | latest | Security scanning |
| ineffassign | latest | Dead code detection |
| misspell | latest | Spelling checks |
| govulncheck | latest | Vulnerability scanning |

## Support

For issues with code quality tools:
1. Check this documentation first
2. Review tool-specific documentation
3. Check for compatibility issues with Go version
4. Create an issue in the project repository

---

*Last updated: 2025-05-29*
*Go version: 1.24.2*
*Project: Assistant AI Development Tool*