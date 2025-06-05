# Agent Architecture Migration Guide

This guide explains the migration strategy for unifying the three different agent implementations in the Assistant codebase into a single, coherent architecture.

## Overview

The Assistant codebase currently has three separate agent implementations:

1. **`/internal/core/agents/`** - The canonical interface definitions with comprehensive types
2. **`/internal/agents/`** - A practical implementation with simpler types
3. **`/internal/langchain/agents/`** - LangChain-specific implementation

The goal is to establish `core/agents` as the single source of truth for all agent interfaces while maintaining backward compatibility.

## Key Incompatibilities

### 1. Request Types

The main incompatibility is in the request structure:

| Field | core/agents.Request | agents.Request | langchain/agents.AgentRequest |
|-------|-------------------|----------------|------------------------------|
| Main Query | `Content` | `Query` | `Query` |
| Unique ID | `ID` | - | - |
| Request Type | `Type` | - | - |
| User Tracking | `UserID` | - | - |
| Session Tracking | `SessionID` | - | - |
| Created Time | `CreatedAt` | - | - |
| LLM Settings | - | `Temperature` | `Temperature` |
| Execution Control | - | `MaxSteps` | `MaxSteps` |
| Tool Selection | - | `Tools` | `Tools` |

### 2. Response Types

Similar incompatibilities exist in responses:

| Field | core/agents.Response | agents.Response | langchain/agents.AgentResponse |
|-------|---------------------|-----------------|--------------------------------|
| Main Result | `Content` | `Result` | `Result` |
| Request Link | `RequestID` | - | - |
| Agent Name | `AgentName` | - | - |
| Success Flag | `Success` | - | - |
| Error Message | `Error` | - | - |
| Time Taken | `Duration` | `ExecutionTime` | `ExecutionTime` |
| Created Time | `CreatedAt` | - | - |
| Execution Details | - | `Steps` | `Steps` |
| Token Usage | - | `TokensUsed` | `TokensUsed` |

## Migration Strategy

### Phase 1: Add Adapters (Complete)

We've created adapter layers to bridge between the different implementations:

1. **`/internal/agents/adapters.go`**:
   - `CoreAgentAdapter` - Wraps `agents.Agent` to implement `core.Agent`
   - `AgentAdapter` - Wraps `core.Agent` to implement `agents.Agent`
   - Request/Response adapters for type conversion

2. **`/internal/langchain/agents/adapters.go`**:
   - `LangChainToCoreAdapter` - Wraps LangChain agents to implement `core.Agent`
   - `SpecializedAgentAdapter` - Wraps specific LangChain agents
   - Request/Response conversion utilities

### Phase 2: Update Implementations (Next)

Update existing code to use adapters:

```go
// Before: Direct use of agents.Agent
agent := agents.NewDevelopmentAgent(llm, logger)
resp, err := agent.Execute(ctx, agentRequest)

// After: Use with core interfaces via adapter
agent := agents.NewDevelopmentAgent(llm, logger)
coreAgent := agents.NewCoreAgentAdapter(agent)
coreResp, err := coreAgent.Execute(ctx, coreRequest)
```

### Phase 3: Gradual Migration

1. **Update agent registrations** to use core interfaces:
```go
// In agent managers
func (m *Manager) RegisterAgent(agent core.Agent) error {
    // Implementation
}
```

2. **Update consumers** to use core types:
```go
// Service layer
func (s *Service) ProcessRequest(ctx context.Context, req core.Request) (*core.Response, error) {
    agent, err := s.manager.GetBestAgent(req)
    if err != nil {
        return nil, err
    }
    return agent.Execute(ctx, req)
}
```

3. **Migrate specialized agents** to implement core interfaces directly:
```go
// Future: Direct implementation
type DevelopmentAgent struct {
    // fields
}

func (a *DevelopmentAgent) Execute(ctx context.Context, req core.Request) (*core.Response, error) {
    // Direct implementation using core types
}
```

## Using the Adapters

### Example 1: Wrapping an agents.Agent for core compatibility

```go
import (
    "github.com/koopa0/assistant-go/internal/agents"
    core "github.com/koopa0/assistant-go/internal/core/agents"
)

// Create an agent using the current implementation
devAgent := agents.NewDevelopmentAgent(llm, logger)

// Wrap it to be core-compatible
coreAgent := agents.NewCoreAgentAdapter(devAgent)

// Now it can be used with core interfaces
var agent core.Agent = coreAgent
response, err := agent.Execute(ctx, coreRequest)
```

### Example 2: Using a core.Agent with legacy code

```go
// You have a core agent
var coreAgent core.Agent

// Need to use it with code expecting agents.Agent
legacyAgent := agents.NewAgentAdapter(coreAgent)

// Now it works with the legacy interface
var agent agents.Agent = legacyAgent
response, err := agent.Execute(ctx, agentRequest)
```

### Example 3: LangChain agent integration

```go
import (
    langchainAgents "github.com/koopa0/assistant-go/internal/langchain/agents"
    core "github.com/koopa0/assistant-go/internal/core/agents"
)

// Create a LangChain agent
lcAgent := langchainAgents.NewDevelopmentAgent(llm, tools, config, logger)

// Wrap it for core compatibility
coreAgent := langchainAgents.NewDevelopmentAgentAdapter(lcAgent)

// Register with core agent manager
manager.RegisterAgent(coreAgent)
```

## Field Mapping Reference

### Request Mapping

When converting between types, the adapters handle these mappings:

```go
// core.Request → agents.Request
agents.Request{
    Query:       core.Content,
    Context:     core.Context,
    Priority:    core.Priority,
    Timeout:     core.Timeout,
    // Tools, MaxSteps, Temperature extracted from Context
    Metadata: {
        "request_id": core.ID,
        "request_type": core.Type,
        "user_id": core.UserID,
        "session_id": core.SessionID,
    },
}

// agents.Request → core.Request
core.Request{
    ID:        generated or from metadata,
    Type:      inferred or from metadata,
    Content:   agents.Query,
    Context:   agents.Context + tools/steps/temp,
    Priority:  agents.Priority,
    Timeout:   agents.Timeout,
    UserID:    from metadata,
    SessionID: from metadata,
}
```

### Response Mapping

```go
// core.Response → agents.Response
agents.Response{
    Result:        core.Content,
    ExecutionTime: core.Duration,
    Confidence:    core.Confidence,
    Suggestions:   core.Suggestions,
    // Steps, ToolsUsed, TokensUsed extracted from Metadata
}

// agents.Response → core.Response
core.Response{
    Content:     agents.Result,
    Duration:    agents.ExecutionTime,
    Success:     agents.Result != "",
    Confidence:  agents.Confidence,
    Suggestions: agents.Suggestions,
    Metadata: {
        "steps": agents.Steps,
        "tools_used": agents.ToolsUsed,
        "tokens_used": agents.TokensUsed,
    },
}
```

## Best Practices

1. **Always use core interfaces** in new code
2. **Use adapters** for backward compatibility
3. **Preserve all data** during conversions using metadata fields
4. **Test thoroughly** when migrating existing code
5. **Document adapter usage** in code comments

## Future Considerations

1. **Deprecation Timeline**: Plan to phase out non-core implementations
2. **Performance**: Monitor adapter overhead (should be minimal)
3. **Type Safety**: Consider code generation for adapters if patterns emerge
4. **Testing**: Ensure comprehensive adapter testing

## Conclusion

This migration provides a path to unify the agent architecture while maintaining backward compatibility. The adapter pattern allows gradual migration without breaking existing code, ensuring a smooth transition to the unified core agent interfaces.