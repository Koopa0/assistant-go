# Known Issues

## 1. Assistant Identity Override Issue

### Problem
When asked "你是誰?" (Who are you?) or similar identity questions, the Assistant incorrectly identifies itself as being developed by Anthropic instead of Koopa, despite explicit system prompts.

### Root Cause
Claude's pre-training identity is extremely strong and tends to override custom identity instructions in system prompts. This is a known limitation of the Claude API where the base model's training can sometimes supersede system-level instructions.

### Current Mitigation
- Added strong identity instructions in `SystemPrompt()` at `internal/ai/prompts/templates.go`
- Explicitly states: "You are Assistant, developed by Koopa. You are NOT Claude, and you were NOT developed by Anthropic."
- Provides specific response template for identity questions

### Workarounds
1. **For production**: Consider implementing a post-processing filter that detects and replaces identity-related responses
2. **For users**: The Assistant functions correctly for all technical queries; only identity questions are affected
3. **Alternative**: Add a disclaimer in documentation about this limitation

### Status
- **Severity**: Low (does not affect core functionality)
- **Impact**: Cosmetic/branding issue only
- **Priority**: Can be addressed in future versions

## 2. Chinese UI Text in Tests

### Problem
Some E2E tests expect Chinese UI text (e.g., "系統狀態", "AI 服務") but the actual output is in English.

### Root Cause
The CLI interface outputs in English by default, but tests were written expecting Chinese output.

### Status
- **Severity**: Low (tests only)
- **Impact**: Some E2E tests fail but functionality works correctly
- **Priority**: Low - can be fixed by updating test expectations

## 3. Docker Container Tests

### Problem
Integration tests using testcontainers fail with Docker Hub authentication errors.

### Root Cause
Docker Hub requires authentication for pulling images, even public ones, due to rate limiting.

### Workaround
- Skip container-based tests in CI/CD
- Run integration tests with local PostgreSQL instance
- Document Docker login requirement for developers

### Status
- **Severity**: Medium (affects testing)
- **Impact**: Integration tests cannot run without Docker authentication
- **Priority**: Medium - document requirements for contributors