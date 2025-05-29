# LangChain Tools Adapter Package

The langchain tools adapter package provides seamless integration between the internal tool system and LangChain's tool interface, enabling AI agents to use all available tools through standardized interfaces.

## Overview

This package bridges the gap between the assistant's internal tool architecture and LangChain's tool interface, providing automatic adaptation, input/output formatting, and agent-specific tool provisioning.

## Core Components

### LangChainToolAdapter (`adapter.go`)

The main adapter that wraps internal tools to make them compatible with LangChain's `tools.Tool` interface.

```go
// Create tool adapter
internalTool := registry.GetTool("postgres", config)
adapter := langchain.NewLangChainToolAdapter(internalTool, logger)

// Use with LangChain agents
agent := agents.NewDevelopmentAgent(llm, []tools.Tool{adapter}, logger)
```

### ToolRegistry

Manages the creation and lifecycle of tool adapters with intelligent caching and agent-specific tool provisioning.

```go
// Create tool registry
registry := langchain.NewToolRegistry(internalRegistry, logger)

// Get specific tool for agents
postgresAdapter, err := registry.GetLangChainTool("postgres", config)

// Get all tools for an agent type
devTools, err := registry.CreateToolsForAgent("development", config)
```

## Key Features

### Automatic Input/Output Conversion

The adapter handles conversion between LangChain's string-based interface and the internal tool's structured interface:

```go
// LangChain agent provides string input
input := `{"query": "SELECT * FROM users", "database": "production"}`

// Adapter converts to internal tool format
inputMap := map[string]interface{}{
    "query":    "SELECT * FROM users",
    "database": "production",
}

// Internal tool returns structured result
result := &tools.ToolResult{
    Success: true,
    Data: map[string]interface{}{
        "rows":     []map[string]interface{}{...},
        "count":    42,
        "duration": "15ms",
    },
}

// Adapter converts back to string for LangChain
output := `{"rows":[...],"count":42,"duration":"15ms"}`
```

### Flexible Input Parsing

The adapter intelligently handles different input formats from agents:

```go
// JSON input (structured)
jsonInput := `{"query": "get pods", "namespace": "production"}`

// Plain text input (simple queries)
textInput := "list all running pods in production"

// Both are handled automatically:
// - JSON is parsed to map[string]interface{}
// - Text creates map with multiple key options (query, text, input)
```

### Agent-Specific Tool Provisioning

Tools are automatically selected based on agent specialization:

```go
// Development Agent Tools
devTools := []string{"godev", "search"}

// Database Agent Tools  
dbTools := []string{"postgres", "search"}

// Infrastructure Agent Tools
infraTools := []string{"k8s", "docker", "search"}

// Research Agent Tools
researchTools := []string{"search", "cloudflare"}

// Automatic provisioning
tools, err := registry.CreateToolsForAgent("development", config)
```

## Usage Examples

### Basic Tool Adaptation

```go
package main

import (
    "context"
    "log"
    
    "github.com/koopa0/assistant-go/internal/tools"
    "github.com/koopa0/assistant-go/internal/tools/langchain"
)

func main() {
    ctx := context.Background()
    
    // Create internal tool registry
    internalRegistry := tools.NewRegistry(logger)
    internalRegistry.RegisterTool("postgres", postgres.NewTool(dbConfig))
    
    // Create LangChain adapter registry
    langchainRegistry := langchain.NewToolRegistry(internalRegistry, logger)
    
    // Get adapted tool
    postgresAdapter, err := langchainRegistry.GetLangChainTool("postgres", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use with LangChain
    result, err := postgresAdapter.Call(ctx, `{"query": "SELECT version()"}`)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("Database version: %s", result)
}
```

### Agent Integration

```go
// Create specialized agent with appropriate tools
func CreateDatabaseAgent(llm llms.Model, config map[string]interface{}) (*agents.DatabaseAgent, error) {
    // Create tool registry
    langchainRegistry := langchain.NewToolRegistry(internalRegistry, logger)
    
    // Get database-specific tools
    tools, err := langchainRegistry.CreateToolsForAgent("database", config)
    if err != nil {
        return nil, fmt.Errorf("failed to create tools: %w", err)
    }
    
    // Create specialized agent
    agent := agents.NewDatabaseAgent(llm, tools, logger)
    
    return agent, nil
}

// Use agent with natural language queries
func QueryDatabase(agent *agents.DatabaseAgent, query string) (string, error) {
    ctx := context.Background()
    
    // Agent automatically selects and uses appropriate tools
    response, err := agent.ExecuteQuery(ctx, query)
    if err != nil {
        return "", fmt.Errorf("agent execution failed: %w", err)
    }
    
    return response, nil
}
```

### Multi-Tool Workflows

```go
// Complex workflow using multiple adapted tools
func DeploymentWorkflow(registry *langchain.ToolRegistry, config map[string]interface{}) error {
    ctx := context.Background()
    
    // Get all infrastructure tools
    infraTools, err := registry.CreateToolsForAgent("infrastructure", config)
    if err != nil {
        return err
    }
    
    // Create tool map for easy access
    toolMap := make(map[string]tools.Tool)
    for _, tool := range infraTools {
        toolMap[tool.Name()] = tool
    }
    
    // Step 1: Check cluster status
    k8sTool := toolMap["k8s"]
    clusterStatus, err := k8sTool.Call(ctx, `{"action": "get_nodes"}`)
    if err != nil {
        return fmt.Errorf("cluster check failed: %w", err)
    }
    
    // Step 2: Build and push image
    dockerTool := toolMap["docker"]
    buildResult, err := dockerTool.Call(ctx, `{
        "action": "build",
        "dockerfile": "./Dockerfile",
        "tag": "myapp:latest"
    }`)
    if err != nil {
        return fmt.Errorf("docker build failed: %w", err)
    }
    
    // Step 3: Deploy to Kubernetes
    deployResult, err := k8sTool.Call(ctx, `{
        "action": "apply",
        "manifest": "./k8s/deployment.yaml"
    }`)
    if err != nil {
        return fmt.Errorf("deployment failed: %w", err)
    }
    
    log.Printf("Deployment workflow completed successfully")
    return nil
}
```

### Custom String Tools

For simple tools that work better with direct string input:

```go
// Create simple string-based tool
searchTool := langchain.NewSimpleStringTool(
    "web_search",
    "Search the web for information",
    func(ctx context.Context, query string) (string, error) {
        // Implement web search logic
        results, err := performWebSearch(ctx, query)
        if err != nil {
            return "", err
        }
        
        // Format results as string
        return formatSearchResults(results), nil
    },
)

// Use with agents
agent := agents.NewResearchAgent(llm, []tools.Tool{searchTool}, logger)
```

## Advanced Configuration

### Error Handling and Resilience

```go
// Robust tool adapter with retry logic
type ResilientToolAdapter struct {
    *langchain.LangChainToolAdapter
    maxRetries int
    backoff    time.Duration
}

func NewResilientToolAdapter(internalTool internaltools.Tool, logger *slog.Logger, maxRetries int) *ResilientToolAdapter {
    base := langchain.NewLangChainToolAdapter(internalTool, logger)
    return &ResilientToolAdapter{
        LangChainToolAdapter: base,
        maxRetries:          maxRetries,
        backoff:             time.Second,
    }
}

func (r *ResilientToolAdapter) Call(ctx context.Context, input string) (string, error) {
    var lastErr error
    
    for attempt := 1; attempt <= r.maxRetries; attempt++ {
        result, err := r.LangChainToolAdapter.Call(ctx, input)
        if err == nil {
            return result, nil
        }
        
        lastErr = err
        
        // Don't retry on certain error types
        if isNonRetryableError(err) {
            break
        }
        
        if attempt < r.maxRetries {
            time.Sleep(r.backoff * time.Duration(attempt))
        }
    }
    
    return "", fmt.Errorf("tool failed after %d attempts: %w", r.maxRetries, lastErr)
}
```

### Performance Monitoring

```go
// Tool adapter with performance monitoring
type MonitoredToolAdapter struct {
    *langchain.LangChainToolAdapter
    metrics *metrics.ToolMetrics
}

func NewMonitoredToolAdapter(internalTool internaltools.Tool, logger *slog.Logger, metrics *metrics.ToolMetrics) *MonitoredToolAdapter {
    base := langchain.NewLangChainToolAdapter(internalTool, logger)
    return &MonitoredToolAdapter{
        LangChainToolAdapter: base,
        metrics:             metrics,
    }
}

func (m *MonitoredToolAdapter) Call(ctx context.Context, input string) (string, error) {
    start := time.Now()
    toolName := m.Name()
    
    // Record invocation
    m.metrics.ToolInvoked(toolName)
    
    result, err := m.LangChainToolAdapter.Call(ctx, input)
    
    duration := time.Since(start)
    
    if err != nil {
        m.metrics.ToolFailed(toolName, duration)
        return "", err
    }
    
    m.metrics.ToolSucceeded(toolName, duration)
    return result, nil
}
```

### Dynamic Tool Loading

```go
// Registry with dynamic tool loading capability
type DynamicToolRegistry struct {
    *langchain.ToolRegistry
    configWatcher *config.Watcher
}

func NewDynamicToolRegistry(internalRegistry *internaltools.Registry, logger *slog.Logger) *DynamicToolRegistry {
    base := langchain.NewToolRegistry(internalRegistry, logger)
    
    return &DynamicToolRegistry{
        ToolRegistry:  base,
        configWatcher: config.NewWatcher(logger),
    }
}

func (d *DynamicToolRegistry) WatchForConfigChanges(ctx context.Context) {
    configChan := d.configWatcher.Watch(ctx, "tools.yaml")
    
    for {
        select {
        case <-ctx.Done():
            return
        case configUpdate := <-configChan:
            d.reloadTools(configUpdate)
        }
    }
}

func (d *DynamicToolRegistry) reloadTools(config map[string]interface{}) {
    log.Printf("Reloading tools with new configuration")
    
    // Clear existing adapters to force recreation
    d.adapters = make(map[string]*langchain.LangChainToolAdapter)
    
    // Tools will be recreated with new config on next access
    log.Printf("Tool adapters cleared, will be recreated on demand")
}
```

## Integration with Agent Workflows

### Chain of Tools Pattern

```go
// Execute tools in sequence with context passing
func ExecuteToolChain(registry *langchain.ToolRegistry, steps []ToolStep) (string, error) {
    ctx := context.Background()
    var context strings.Builder
    
    for i, step := range steps {
        tool, err := registry.GetLangChainTool(step.ToolName, step.Config)
        if err != nil {
            return "", fmt.Errorf("step %d: failed to get tool %s: %w", i+1, step.ToolName, err)
        }
        
        // Build input with context from previous steps
        input := step.Input
        if context.Len() > 0 {
            input = fmt.Sprintf("%s\n\nContext from previous steps:\n%s", step.Input, context.String())
        }
        
        result, err := tool.Call(ctx, input)
        if err != nil {
            return "", fmt.Errorf("step %d: tool %s execution failed: %w", i+1, step.ToolName, err)
        }
        
        // Add result to context for next steps
        context.WriteString(fmt.Sprintf("Step %d (%s): %s\n", i+1, step.ToolName, result))
        
        log.Printf("Step %d completed: %s", i+1, step.ToolName)
    }
    
    return context.String(), nil
}

type ToolStep struct {
    ToolName string                 `json:"tool_name"`
    Input    string                 `json:"input"`
    Config   map[string]interface{} `json:"config"`
}
```

### Parallel Tool Execution

```go
// Execute multiple tools in parallel
func ExecuteToolsInParallel(registry *langchain.ToolRegistry, tasks []ToolTask) (map[string]string, error) {
    ctx := context.Background()
    
    // Create channels for results and errors
    resultChan := make(chan ToolTaskResult, len(tasks))
    
    // Launch goroutines for each task
    for _, task := range tasks {
        go func(t ToolTask) {
            tool, err := registry.GetLangChainTool(t.ToolName, t.Config)
            if err != nil {
                resultChan <- ToolTaskResult{ID: t.ID, Error: err}
                return
            }
            
            result, err := tool.Call(ctx, t.Input)
            resultChan <- ToolTaskResult{
                ID:     t.ID,
                Result: result,
                Error:  err,
            }
        }(task)
    }
    
    // Collect results
    results := make(map[string]string)
    var errors []error
    
    for i := 0; i < len(tasks); i++ {
        taskResult := <-resultChan
        
        if taskResult.Error != nil {
            errors = append(errors, fmt.Errorf("task %s: %w", taskResult.ID, taskResult.Error))
            continue
        }
        
        results[taskResult.ID] = taskResult.Result
    }
    
    if len(errors) > 0 {
        return results, fmt.Errorf("some tasks failed: %v", errors)
    }
    
    return results, nil
}

type ToolTask struct {
    ID       string                 `json:"id"`
    ToolName string                 `json:"tool_name"`
    Input    string                 `json:"input"`
    Config   map[string]interface{} `json:"config"`
}

type ToolTaskResult struct {
    ID     string
    Result string
    Error  error
}
```

## Testing

### Unit Tests

```go
func TestLangChainToolAdapter_Call(t *testing.T) {
    // Create mock internal tool
    mockTool := &MockInternalTool{
        name:        "test_tool",
        description: "Test tool for unit testing",
    }
    
    // Create adapter
    adapter := langchain.NewLangChainToolAdapter(mockTool, testLogger)
    
    // Test JSON input
    jsonInput := `{"query": "test", "param": "value"}`
    result, err := adapter.Call(context.Background(), jsonInput)
    
    assert.NoError(t, err)
    assert.NotEmpty(t, result)
    assert.Equal(t, "test_tool", adapter.Name())
    assert.Equal(t, "Test tool for unit testing", adapter.Description())
}

func TestToolRegistry_CreateToolsForAgent(t *testing.T) {
    // Setup
    internalRegistry := setupMockInternalRegistry()
    registry := langchain.NewToolRegistry(internalRegistry, testLogger)
    
    // Test development agent tools
    tools, err := registry.CreateToolsForAgent("development", nil)
    assert.NoError(t, err)
    assert.NotEmpty(t, tools)
    
    // Verify expected tools are present
    toolNames := make([]string, len(tools))
    for i, tool := range tools {
        toolNames[i] = tool.Name()
    }
    
    assert.Contains(t, toolNames, "godev")
    assert.Contains(t, toolNames, "search")
}
```

### Integration Tests

```go
func TestToolAdapterIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Setup real tool registry
    registry := setupRealToolRegistry(t)
    defer registry.Close(context.Background())
    
    // Test real tool execution
    postgresAdapter, err := registry.GetLangChainTool("postgres", map[string]interface{}{
        "connection_string": testDBConnectionString,
    })
    require.NoError(t, err)
    
    // Execute real query
    result, err := postgresAdapter.Call(context.Background(), `{"query": "SELECT 1 as test"}`)
    require.NoError(t, err)
    require.Contains(t, result, "test")
}
```

## Best Practices

### Input Validation

```go
// Validate input before processing
func validateToolInput(toolName string, input string) error {
    switch toolName {
    case "postgres":
        return validatePostgresInput(input)
    case "k8s":
        return validateK8sInput(input)
    default:
        return validateGenericInput(input)
    }
}

func validatePostgresInput(input string) error {
    var inputMap map[string]interface{}
    if err := json.Unmarshal([]byte(input), &inputMap); err != nil {
        return fmt.Errorf("invalid JSON input: %w", err)
    }
    
    if _, ok := inputMap["query"]; !ok {
        return fmt.Errorf("missing required field: query")
    }
    
    return nil
}
```

### Result Formatting

```go
// Format results for optimal agent consumption
func formatResultForAgent(result *internaltools.ToolResult, agentType string) (string, error) {
    if !result.Success {
        return fmt.Sprintf("Error: %s", result.Error), nil
    }
    
    switch agentType {
    case "development":
        return formatForDevelopmentAgent(result.Data)
    case "database":
        return formatForDatabaseAgent(result.Data)
    default:
        return formatGenericResult(result.Data)
    }
}

func formatForDatabaseAgent(data interface{}) (string, error) {
    if queryResult, ok := data.(map[string]interface{}); ok {
        if rows, ok := queryResult["rows"].([]map[string]interface{}); ok {
            // Format as table for better readability
            return formatAsTable(rows), nil
        }
    }
    
    // Fallback to JSON
    jsonData, err := json.MarshalIndent(data, "", "  ")
    return string(jsonData), err
}
```

This LangChain tools adapter provides seamless integration between the internal tool ecosystem and LangChain's agent framework, enabling sophisticated AI-driven workflows with full access to all available tools and capabilities.