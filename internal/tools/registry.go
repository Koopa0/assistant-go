package tools

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Tool represents a tool that can be executed by the assistant
type Tool interface {
	// Name returns the tool name
	Name() string

	// Description returns the tool description
	Description() string

	// Parameters returns the tool parameters schema
	Parameters() *ToolParametersSchema

	// Execute executes the tool with the given input
	Execute(ctx context.Context, input *ToolInput) (*ToolResult, error)

	// Health checks if the tool is healthy
	Health(ctx context.Context) error

	// Close closes the tool and cleans up resources
	Close(ctx context.Context) error
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success       bool            `json:"success"`
	Data          *ToolResultData `json:"data,omitempty"`
	Error         string          `json:"error,omitempty"`
	Metadata      *ToolMetadata   `json:"metadata,omitempty"`
	ExecutionTime time.Duration   `json:"execution_time"`
}

// ToolInfo represents information about a tool
type ToolInfo struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Parameters  *ToolParametersSchema `json:"parameters"`
	Category    string                `json:"category"`
	Version     string                `json:"version"`
	Author      string                `json:"author"`
	IsEnabled   bool                  `json:"is_enabled"`
}

// ToolFactory is a function that creates a new tool instance
type ToolFactory func(config *ToolConfig, logger *slog.Logger) (Tool, error)

// Registry manages tool registration and execution
type Registry struct {
	tools     map[string]Tool
	factories map[string]ToolFactory
	info      map[string]ToolInfo
	mutex     sync.RWMutex
	logger    *slog.Logger
}

// NewRegistry creates a new tool registry
func NewRegistry(logger *slog.Logger) *Registry {
	return &Registry{
		tools:     make(map[string]Tool),
		factories: make(map[string]ToolFactory),
		info:      make(map[string]ToolInfo),
		logger:    logger,
	}
}

// Register registers a tool factory
func (r *Registry) Register(name string, factory ToolFactory) error {
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if factory == nil {
		return fmt.Errorf("tool factory cannot be nil")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("tool %s is already registered", name)
	}

	r.factories[name] = factory
	r.logger.Debug("Tool factory registered", slog.String("tool", name))

	return nil
}

// Unregister unregisters a tool
func (r *Registry) Unregister(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Close the tool if it's instantiated
	if tool, exists := r.tools[name]; exists {
		if err := tool.Close(context.Background()); err != nil {
			r.logger.Warn("Failed to close tool during unregistration",
				slog.String("tool", name),
				slog.Any("error", err))
		}
		delete(r.tools, name)
	}

	// Remove factory and info
	delete(r.factories, name)
	delete(r.info, name)

	r.logger.Debug("Tool unregistered", slog.String("tool", name))
	return nil
}

// GetTool gets or creates a tool instance
func (r *Registry) GetTool(name string, config *ToolConfig) (Tool, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Return existing instance if available
	if tool, exists := r.tools[name]; exists {
		return tool, nil
	}

	// Get factory
	factory, exists := r.factories[name]
	if !exists {
		return nil, fmt.Errorf("tool %s is not registered", name)
	}

	// Use default config if none provided
	if config == nil {
		config = &ToolConfig{}
	}

	// Create new instance with panic recovery
	var tool Tool
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("tool factory panic: %v", r)
			}
		}()
		tool, err = factory(config, r.logger)
	}()

	if err != nil {
		return nil, fmt.Errorf("failed to create tool %s: %w", name, err)
	}

	// Store instance
	r.tools[name] = tool

	// Store tool info
	r.info[name] = ToolInfo{
		Name:        tool.Name(),
		Description: tool.Description(),
		Parameters:  tool.Parameters(),
		Category:    r.getToolCategory(name),
		Version:     "1.0.0", // TODO: Get from tool metadata
		Author:      "GoAssistant",
		IsEnabled:   true,
	}

	r.logger.Debug("Tool instance created", slog.String("tool", name))
	return tool, nil
}

// Execute executes a tool with the given input
func (r *Registry) Execute(ctx context.Context, name string, input *ToolInput, config *ToolConfig) (*ToolResult, error) {
	startTime := time.Now()

	tool, err := r.GetTool(name, config)
	if err != nil {
		return &ToolResult{
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: time.Since(startTime),
		}, err
	}

	// Ensure input is not nil
	if input == nil {
		input = &ToolInput{
			Parameters: make(map[string]interface{}),
		}
	}

	r.logger.Debug("Executing tool",
		slog.String("tool", name),
		slog.String("task_description", input.TaskDescription),
		slog.String("context", input.Context))

	result, err := tool.Execute(ctx, input)
	if err != nil {
		r.logger.Error("Tool execution failed",
			slog.String("tool", name),
			slog.Any("error", err),
			slog.Duration("execution_time", time.Since(startTime)))

		return &ToolResult{
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: time.Since(startTime),
		}, err
	}

	if result == nil {
		result = &ToolResult{
			Success:       true,
			ExecutionTime: time.Since(startTime),
		}
	} else {
		result.ExecutionTime = time.Since(startTime)
	}

	r.logger.Debug("Tool execution completed",
		slog.String("tool", name),
		slog.Bool("success", result.Success),
		slog.Duration("execution_time", result.ExecutionTime))

	return result, nil
}

// IsRegistered checks if a tool is registered
func (r *Registry) IsRegistered(name string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.factories[name]
	return exists
}

// ListTools returns a list of all registered tools
func (r *Registry) ListTools() []ToolInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	tools := make([]ToolInfo, 0, len(r.info))
	for _, info := range r.info {
		tools = append(tools, info)
	}

	// If no instances exist, create info from factories
	if len(tools) == 0 {
		for name := range r.factories {
			tools = append(tools, ToolInfo{
				Name:        name,
				Description: "Tool description not available",
				Parameters: &ToolParametersSchema{
					Type:       "object",
					Properties: make(map[string]ToolParameter),
					Required:   []string{},
				},
				Category:  r.getToolCategory(name),
				Version:   "1.0.0",
				Author:    "GoAssistant",
				IsEnabled: true,
			})
		}
	}

	return tools
}

// GetToolInfo returns information about a specific tool
func (r *Registry) GetToolInfo(name string) (*ToolInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if info, exists := r.info[name]; exists {
		return &info, nil
	}

	// If no instance exists but factory is registered, create basic info
	if _, exists := r.factories[name]; exists {
		info := ToolInfo{
			Name:        name,
			Description: "Tool description not available",
			Parameters: &ToolParametersSchema{
				Type:       "object",
				Properties: make(map[string]ToolParameter),
				Required:   []string{},
			},
			Category:  r.getToolCategory(name),
			Version:   "1.0.0",
			Author:    "GoAssistant",
			IsEnabled: true,
		}
		return &info, nil
	}

	return nil, fmt.Errorf("tool %s is not registered", name)
}

// Health checks the health of all registered tools
func (r *Registry) Health(ctx context.Context) error {
	r.mutex.RLock()
	tools := make(map[string]Tool)
	for name, tool := range r.tools {
		tools[name] = tool
	}
	r.mutex.RUnlock()

	for name, tool := range tools {
		if err := tool.Health(ctx); err != nil {
			return fmt.Errorf("tool %s health check failed: %w", name, err)
		}
	}

	return nil
}

// Stats returns registry statistics
func (r *Registry) Stats(ctx context.Context) (*RegistryStats, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	toolStats := make(map[string]*ToolStats)
	activeCount := 0

	for name := range r.factories {
		isActive := r.tools[name] != nil
		if isActive {
			activeCount++
		}

		toolStats[name] = &ToolStats{
			Name:           name,
			ExecutionCount: 0,           // TODO: Implement execution tracking
			SuccessCount:   0,           // TODO: Implement success tracking
			ErrorCount:     0,           // TODO: Implement error tracking
			SuccessRate:    1.0,         // Default to 100% until tracking is implemented
			AverageTime:    0,           // TODO: Implement timing tracking
			LastExecuted:   time.Time{}, // TODO: Implement execution tracking
		}
	}

	stats := &RegistryStats{
		RegisteredTools:    len(r.factories),
		ActiveTools:        activeCount,
		TotalExecutions:    0,   // TODO: Implement tracking
		TotalSuccesses:     0,   // TODO: Implement tracking
		TotalErrors:        0,   // TODO: Implement tracking
		OverallSuccessRate: 1.0, // Default until tracking is implemented
		ToolStats:          toolStats,
		Uptime:             time.Since(time.Now()), // TODO: Track actual uptime
	}

	return stats, nil
}

// Close closes all tool instances
func (r *Registry) Close(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var lastErr error
	for name, tool := range r.tools {
		if err := tool.Close(ctx); err != nil {
			r.logger.Error("Failed to close tool",
				slog.String("tool", name),
				slog.Any("error", err))
			lastErr = err
		}
	}

	// Clear all maps
	r.tools = make(map[string]Tool)
	r.factories = make(map[string]ToolFactory)
	r.info = make(map[string]ToolInfo)

	r.logger.Info("Tool registry closed")
	return lastErr
}

// getToolCategory determines the category of a tool based on its name
func (r *Registry) getToolCategory(name string) string {
	switch name {
	case "postgres", "database":
		return "database"
	case "kubernetes", "k8s":
		return "infrastructure"
	case "docker":
		return "containers"
	case "cloudflare":
		return "cloud"
	case "search", "searxng":
		return "search"
	case "langchain", "agent":
		return "ai"
	case "godev", "go":
		return "development"
	default:
		return "general"
	}
}

// LEGACY COMPATIBILITY METHODS
// These methods provide backward compatibility with existing code that uses map[string]interface{}

// ExecuteLegacy executes a tool with legacy map[string]interface{} input for backward compatibility
func (r *Registry) ExecuteLegacy(ctx context.Context, name string, input map[string]interface{}, config map[string]interface{}) (*ToolResult, error) {
	// Convert legacy input and config to new types
	toolInput := ConvertLegacyInput(input)
	toolConfig := ConvertLegacyConfig(config)

	return r.Execute(ctx, name, toolInput, toolConfig)
}

// GetToolLegacy gets a tool with legacy map[string]interface{} config for backward compatibility
func (r *Registry) GetToolLegacy(name string, config map[string]interface{}) (Tool, error) {
	toolConfig := ConvertLegacyConfig(config)
	return r.GetTool(name, toolConfig)
}

// StatsLegacy returns registry statistics in legacy map[string]interface{} format for backward compatibility
func (r *Registry) StatsLegacy(ctx context.Context) (map[string]interface{}, error) {
	stats, err := r.Stats(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to legacy format
	legacyStats := map[string]interface{}{
		"registered_factories": stats.RegisteredTools,
		"active_instances":     stats.ActiveTools,
		"total_executions":     stats.TotalExecutions,
		"total_successes":      stats.TotalSuccesses,
		"total_errors":         stats.TotalErrors,
		"success_rate":         stats.OverallSuccessRate,
		"uptime":               stats.Uptime.String(),
		"tools":                make(map[string]interface{}),
	}

	// Convert tool stats
	toolStats := make(map[string]interface{})
	for name, stat := range stats.ToolStats {
		toolStats[name] = map[string]interface{}{
			"execution_count": stat.ExecutionCount,
			"success_count":   stat.SuccessCount,
			"error_count":     stat.ErrorCount,
			"success_rate":    stat.SuccessRate,
			"average_time":    stat.AverageTime.String(),
			"last_executed":   stat.LastExecuted,
		}
	}
	legacyStats["tools"] = toolStats

	return legacyStats, nil
}
