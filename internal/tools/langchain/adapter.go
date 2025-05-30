package langchain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/tmc/langchaingo/tools"

	internaltools "github.com/koopa0/assistant-go/internal/tools"
)

// LangChainToolAdapter adapts internal tools to LangChain's tool interface
type LangChainToolAdapter struct {
	internalTool internaltools.Tool
	logger       *slog.Logger
}

// NewLangChainToolAdapter creates a new adapter for an internal tool
func NewLangChainToolAdapter(internalTool internaltools.Tool, logger *slog.Logger) *LangChainToolAdapter {
	return &LangChainToolAdapter{
		internalTool: internalTool,
		logger:       logger,
	}
}

// Name returns the tool name (implements tools.Tool interface)
func (a *LangChainToolAdapter) Name() string {
	return a.internalTool.Name()
}

// Description returns the tool description (implements tools.Tool interface)
func (a *LangChainToolAdapter) Description() string {
	return a.internalTool.Description()
}

// Call executes the tool with string input (implements tools.Tool interface)
func (a *LangChainToolAdapter) Call(ctx context.Context, input string) (string, error) {
	a.logger.Debug("LangChain tool adapter call",
		slog.String("tool", a.Name()),
		slog.String("input", input))

	// Parse string input to map
	inputMap, err := a.parseStringInput(input)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	// Execute internal tool
	result, err := a.internalTool.Execute(ctx, inputMap)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	// Convert result to string
	output, err := a.formatResult(result)
	if err != nil {
		return "", fmt.Errorf("failed to format result: %w", err)
	}

	a.logger.Debug("LangChain tool adapter call completed",
		slog.String("tool", a.Name()),
		slog.Bool("success", result.Success))

	return output, nil
}

// parseStringInput attempts to parse string input as JSON, falls back to plain text
func (a *LangChainToolAdapter) parseStringInput(input string) (map[string]interface{}, error) {
	// First, try to parse as JSON
	var inputMap map[string]interface{}
	if err := json.Unmarshal([]byte(input), &inputMap); err == nil {
		return inputMap, nil
	}

	// If JSON parsing fails, create a simple map with the input as "query" or "text"
	// This handles cases where agents pass simple text inputs
	inputMap = map[string]interface{}{
		"query": input,
		"text":  input,
		"input": input,
	}

	return inputMap, nil
}

// formatResult converts ToolResult to string output
func (a *LangChainToolAdapter) formatResult(result *internaltools.ToolResult) (string, error) {
	if !result.Success {
		return fmt.Sprintf("Error: %s", result.Error), nil
	}

	// If data is already a string, return it directly
	if str, ok := result.Data.(string); ok {
		return str, nil
	}

	// Try to convert data to JSON string
	if result.Data != nil {
		jsonData, err := json.Marshal(result.Data)
		if err != nil {
			return fmt.Sprintf("Result: %v", result.Data), nil
		}
		return string(jsonData), nil
	}

	return "Operation completed successfully", nil
}

// ToolRegistry manages LangChain tool adapters
type ToolRegistry struct {
	internalRegistry *internaltools.Registry
	adapters         map[string]*LangChainToolAdapter
	logger           *slog.Logger
}

// NewToolRegistry creates a new LangChain tool registry
func NewToolRegistry(internalRegistry *internaltools.Registry, logger *slog.Logger) *ToolRegistry {
	return &ToolRegistry{
		internalRegistry: internalRegistry,
		adapters:         make(map[string]*LangChainToolAdapter),
		logger:           logger,
	}
}

// GetLangChainTool returns a LangChain-compatible tool adapter
func (tr *ToolRegistry) GetLangChainTool(name string, config map[string]interface{}) (tools.Tool, error) {
	// Check if adapter already exists
	if adapter, exists := tr.adapters[name]; exists {
		return adapter, nil
	}

	// Get internal tool
	internalTool, err := tr.internalRegistry.GetTool(name, config)
	if err != nil {
		return nil, fmt.Errorf("failed to get internal tool: %w", err)
	}

	// Create adapter
	adapter := NewLangChainToolAdapter(internalTool, tr.logger)
	tr.adapters[name] = adapter

	tr.logger.Debug("Created LangChain tool adapter",
		slog.String("tool", name))

	return adapter, nil
}

// GetAllLangChainTools returns all available tools as LangChain adapters
func (tr *ToolRegistry) GetAllLangChainTools(config map[string]interface{}) ([]tools.Tool, error) {
	toolInfos := tr.internalRegistry.ListTools()
	langchainTools := make([]tools.Tool, 0, len(toolInfos))

	for _, toolInfo := range toolInfos {
		adapter, err := tr.GetLangChainTool(toolInfo.Name, config)
		if err != nil {
			tr.logger.Warn("Failed to create adapter for tool",
				slog.String("tool", toolInfo.Name),
				slog.Any("error", err))
			continue
		}
		langchainTools = append(langchainTools, adapter)
	}

	return langchainTools, nil
}

// CreateToolsForAgent creates LangChain tools for a specific agent
func (tr *ToolRegistry) CreateToolsForAgent(agentType string, config map[string]interface{}) ([]tools.Tool, error) {
	var toolNames []string

	// Define tools available for each agent type
	switch agentType {
	case "development":
		toolNames = []string{"godev", "search"}
	case "database":
		toolNames = []string{"postgres", "search"}
	case "infrastructure":
		toolNames = []string{"k8s", "docker", "search"}
	case "research":
		toolNames = []string{"search", "cloudflare"}
	default:
		// For unknown agent types, provide basic tools
		toolNames = []string{"search"}
	}

	var langchainTools []tools.Tool
	for _, toolName := range toolNames {
		if tr.internalRegistry.IsRegistered(toolName) {
			adapter, err := tr.GetLangChainTool(toolName, config)
			if err != nil {
				tr.logger.Warn("Failed to create tool adapter for agent",
					slog.String("agent_type", agentType),
					slog.String("tool", toolName),
					slog.Any("error", err))
				continue
			}
			langchainTools = append(langchainTools, adapter)
		}
	}

	tr.logger.Info("Created tools for agent",
		slog.String("agent_type", agentType),
		slog.Int("tool_count", len(langchainTools)))

	return langchainTools, nil
}

// Health checks the health of all tool adapters
func (tr *ToolRegistry) Health(ctx context.Context) error {
	for name, adapter := range tr.adapters {
		// Check internal tool health
		if err := adapter.internalTool.Health(ctx); err != nil {
			return fmt.Errorf("tool adapter %s health check failed: %w", name, err)
		}
	}
	return nil
}

// Close closes all tool adapters
func (tr *ToolRegistry) Close(ctx context.Context) error {
	var lastErr error
	for name, adapter := range tr.adapters {
		if err := adapter.internalTool.Close(ctx); err != nil {
			tr.logger.Error("Failed to close tool adapter",
				slog.String("tool", name),
				slog.Any("error", err))
			lastErr = err
		}
	}

	// Clear adapters
	tr.adapters = make(map[string]*LangChainToolAdapter)
	return lastErr
}

// StringInputTool is a helper interface for tools that can handle string inputs directly
type StringInputTool interface {
	tools.Tool
	CallWithString(ctx context.Context, input string) (string, error)
}

// SimpleStringTool wraps tools that work better with direct string input
type SimpleStringTool struct {
	name        string
	description string
	handler     func(ctx context.Context, input string) (string, error)
}

// NewSimpleStringTool creates a simple string-based tool
func NewSimpleStringTool(name, description string, handler func(ctx context.Context, input string) (string, error)) *SimpleStringTool {
	return &SimpleStringTool{
		name:        name,
		description: description,
		handler:     handler,
	}
}

// Name returns the tool name
func (t *SimpleStringTool) Name() string {
	return t.name
}

// Description returns the tool description
func (t *SimpleStringTool) Description() string {
	return t.description
}

// Call executes the tool
func (t *SimpleStringTool) Call(ctx context.Context, input string) (string, error) {
	return t.handler(ctx, input)
}

// CallWithString provides the same functionality as Call for compatibility
func (t *SimpleStringTool) CallWithString(ctx context.Context, input string) (string, error) {
	return t.Call(ctx, input)
}

// Verify interface compliance at compile time
var (
	_ tools.Tool      = (*LangChainToolAdapter)(nil)
	_ tools.Tool      = (*SimpleStringTool)(nil)
	_ StringInputTool = (*SimpleStringTool)(nil)
)
