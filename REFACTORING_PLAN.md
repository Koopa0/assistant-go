# Agent System Refactoring Plan

## Decision: Keep `/internal/agents/` as the Base

**Rationale:**
- Simplest, most practical interface
- Follows Go principles: small interfaces, discovered through use
- Already has concrete implementations
- Less abstraction overhead

## Phase 1: Agent Consolidation Strategy

### Step 1: Backup and Prepare
1. Create backup branch for safety
2. Document current state
3. Identify all agent usages

### Step 2: Merge Useful Features

**From `/internal/core/agents/`:**
- [ ] CollaborationResult and CollaborationStrategy (useful for multi-agent)
- [ ] Feedback and learning interfaces (if needed)
- [ ] Performance tracking concepts
- [ ] Priority and RequestType enums

**From `/internal/langchain/agents/`:**
- [ ] AgentCapability concept (good for discovery)
- [ ] Chain integration patterns
- [ ] Tool management from BaseAgent
- [ ] AST analyzer and code generation helpers

### Step 3: Update Agent Interface

```go
// Enhanced agent interface combining best of all three
type Agent interface {
    // Core methods (keep simple)
    Execute(ctx context.Context, request Request) (*Response, error)
    Name() string
    Type() AgentType
    
    // Optional capabilities (via type assertion)
    // Capabilities() []Capability // if needed
}

// Enhanced Request with useful fields from all implementations
type Request struct {
    // From current
    Query       string
    Context     map[string]interface{}
    Tools       []string
    MaxSteps    int
    Temperature float64
    
    // From core/agents
    Priority    Priority      // useful addition
    Timeout     time.Duration // useful addition
    
    // From langchain/agents  
    Metadata    map[string]interface{} // useful for extensibility
}

// Keep Response simple but add useful fields
type Response struct {
    // Current fields
    Result        string
    Steps         []Step
    ToolsUsed     []string
    ExecutionTime time.Duration
    TokensUsed    int
    
    // From core/agents
    Confidence  float64  // useful metric
    Suggestions []string // helpful addition
}
```

### Step 4: Migration Order

1. **Update base interface** in `/internal/agents/agent.go`
2. **Migrate Development Agent features**:
   - Copy AST analyzer from langchain version
   - Add code generation capabilities
   - Keep simple execution model

3. **Create adapters** for existing code:
   - LangChainAgentAdapter (wraps agents for langchain use)
   - CoreAgentAdapter (if any code uses core interfaces)

4. **Update all imports** systematically:
   ```bash
   # Find all agent imports
   grep -r "internal/core/agents" --include="*.go"
   grep -r "internal/langchain/agents" --include="*.go"
   ```

5. **Remove old implementations**:
   - Delete `/internal/core/agents/`
   - Delete `/internal/langchain/agents/`

### Step 5: Testing Strategy

1. Create comprehensive tests for new unified agent
2. Ensure backward compatibility where needed
3. Test all agent types (development, database, etc.)
4. Verify LangChain integration still works

## Implementation Checklist

- [ ] Create feature branch: `refactor/consolidate-agents`
- [ ] Update `/internal/agents/agent.go` with enhanced interface
- [ ] Migrate DevelopmentAgent features
- [ ] Create LangChain adapter
- [ ] Update all import paths
- [ ] Remove duplicate implementations
- [ ] Update tests
- [ ] Update documentation

## Success Criteria

1. Single agent package at `/internal/agents/`
2. All existing functionality preserved
3. Cleaner, simpler interface
4. No duplicate agent definitions
5. All tests passing
6. Documentation updated

## Risk Mitigation

1. **Gradual migration**: Use adapters initially
2. **Feature flags**: Can toggle between old/new if needed
3. **Comprehensive testing**: Before removing old code
4. **Rollback plan**: Keep backup branch

## Timeline

- Day 1: Interface design and base implementation
- Day 2: Migrate features and create adapters
- Day 3: Update all usages
- Day 4: Testing and cleanup
- Day 5: Documentation and review