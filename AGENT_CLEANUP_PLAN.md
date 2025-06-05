# Agent System Cleanup Plan

## Final Decision

1. **Keep only `/internal/agent/`** as the single source of truth
2. **Update langchain service** to use the new agent package with adapters
3. **Delete all other agent implementations**

## Cleanup Steps

### Step 1: Update LangChain Service
- Modify `/internal/langchain/service.go` to import from `/internal/agent/`
- Use `LangChainAdapter` for compatibility
- Remove references to old agent packages

### Step 2: Delete Old Implementations
```bash
# Delete duplicate agent packages
rm -rf /internal/core/agents/
rm -rf /internal/agents/
rm -rf /internal/langchain/agents/
```

### Step 3: Update Imports
- Find and replace all old imports
- Update to use `agent.Agent` everywhere

### Step 4: Test
- Ensure everything compiles
- Run tests to verify functionality

## Benefits
- Single package to maintain
- Clear import path: `internal/agent`
- No confusion about which agent to use
- Easier to understand and modify

## Migration for Existing Code

If code was using:
- `core.Agent` → use `agent.Agent`
- `agents.Agent` → use `agent.Agent`
- `langchain/agents.*` → use `agent.LangChainAdapter`

## Note on LangChain Integration

The LangChain service can still work with agents through the adapter:
```go
// In langchain service
devAgent := agent.NewDevelopmentAgent(llm, logger)
lcAgent := agent.NewLangChainAdapter(devAgent)
// Use lcAgent with LangChain-specific code
```

This keeps the separation clean while allowing integration.