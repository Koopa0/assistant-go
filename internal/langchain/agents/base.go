package agents

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/tools"

	"github.com/koopa0/assistant-go/internal/config"
)

// AgentType represents the type of specialized agent
type AgentType string

const (
	AgentTypeDevelopment    AgentType = "development"
	AgentTypeDatabase       AgentType = "database"
	AgentTypeInfrastructure AgentType = "infrastructure"
	AgentTypeResearch       AgentType = "research"
)

// AgentCapability represents a capability that an agent can perform
type AgentCapability struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// AgentRequest represents a request to an agent
type AgentRequest struct {
	Query       string                 `json:"query"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Tools       []string               `json:"tools,omitempty"`
	MaxSteps    int                    `json:"max_steps,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AgentResponse represents a response from an agent
type AgentResponse struct {
	Result        string                 `json:"result"`
	Steps         []AgentStep            `json:"steps"`
	ToolsUsed     []string               `json:"tools_used"`
	ExecutionTime time.Duration          `json:"execution_time"`
	TokensUsed    int                    `json:"tokens_used"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AgentStep represents a single step in agent execution
type AgentStep struct {
	StepNumber int                    `json:"step_number"`
	Action     string                 `json:"action"`
	Tool       string                 `json:"tool,omitempty"`
	Input      string                 `json:"input"`
	Output     string                 `json:"output"`
	Reasoning  string                 `json:"reasoning,omitempty"`
	Duration   time.Duration          `json:"duration"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// BaseAgent provides common functionality for all specialized agents
type BaseAgent struct {
	agentType    AgentType
	llm          llms.Model
	memory       schema.Memory
	tools        []tools.Tool
	chains       map[string]chains.Chain
	config       config.LangChain
	logger       *slog.Logger
	capabilities []AgentCapability
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(agentType AgentType, llm llms.Model, config config.LangChain, logger *slog.Logger) *BaseAgent {
	// Initialize memory based on configuration
	var mem schema.Memory
	if config.EnableMemory {
		mem = memory.NewConversationBuffer()
	}

	return &BaseAgent{
		agentType:    agentType,
		llm:          llm,
		memory:       mem,
		tools:        make([]tools.Tool, 0),
		chains:       make(map[string]chains.Chain),
		config:       config,
		logger:       logger,
		capabilities: make([]AgentCapability, 0),
	}
}

// GetType returns the agent type
func (a *BaseAgent) GetType() AgentType {
	return a.agentType
}

// GetCapabilities returns the agent's capabilities
func (a *BaseAgent) GetCapabilities() []AgentCapability {
	return a.capabilities
}

// AddCapability adds a capability to the agent
func (a *BaseAgent) AddCapability(capability AgentCapability) {
	a.capabilities = append(a.capabilities, capability)
	a.logger.Debug("Added capability to agent",
		slog.String("agent_type", string(a.agentType)),
		slog.String("capability", capability.Name))
}

// AddTool adds a tool to the agent
func (a *BaseAgent) AddTool(tool tools.Tool) {
	a.tools = append(a.tools, tool)
	a.logger.Debug("Added tool to agent",
		slog.String("agent_type", string(a.agentType)),
		slog.String("tool", tool.Name()))
}

// AddChain adds a chain to the agent
func (a *BaseAgent) AddChain(name string, chain chains.Chain) {
	a.chains[name] = chain
	a.logger.Debug("Added chain to agent",
		slog.String("agent_type", string(a.agentType)),
		slog.String("chain", name))
}

// Execute executes a request using the agent
func (a *BaseAgent) Execute(ctx context.Context, request *AgentRequest) (*AgentResponse, error) {
	startTime := time.Now()

	// Validate request first
	if err := a.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	a.logger.Info("Executing agent request",
		slog.String("agent_type", string(a.agentType)),
		slog.String("query", request.Query),
		slog.Int("max_steps", request.MaxSteps))

	// Initialize response
	response := &AgentResponse{
		Steps:     make([]AgentStep, 0),
		ToolsUsed: make([]string, 0),
		Metadata:  make(map[string]interface{}),
	}

	// Set default max steps if not provided
	maxSteps := request.MaxSteps
	if maxSteps <= 0 {
		maxSteps = a.config.MaxIterations
	}

	// Execute agent logic (to be implemented by specialized agents)
	result, steps, err := a.executeSteps(ctx, request, maxSteps)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	// Build response
	response.Result = result
	response.Steps = steps
	response.ExecutionTime = time.Since(startTime)
	response.Metadata["agent_type"] = string(a.agentType)
	response.Metadata["steps_executed"] = len(steps)

	// Extract tools used
	toolsUsed := make(map[string]bool)
	for _, step := range steps {
		if step.Tool != "" {
			toolsUsed[step.Tool] = true
		}
	}
	for tool := range toolsUsed {
		response.ToolsUsed = append(response.ToolsUsed, tool)
	}

	a.logger.Info("Agent execution completed",
		slog.String("agent_type", string(a.agentType)),
		slog.Int("steps", len(steps)),
		slog.Duration("execution_time", response.ExecutionTime))

	return response, nil
}

// executeSteps executes the agent steps (to be overridden by specialized agents)
func (a *BaseAgent) executeSteps(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Default implementation - just call the LLM directly
	steps := make([]AgentStep, 0)

	stepStart := time.Now()
	result, err := a.llm.Call(ctx, request.Query)
	if err != nil {
		return "", nil, fmt.Errorf("LLM call failed: %w", err)
	}

	step := AgentStep{
		StepNumber: 1,
		Action:     "direct_llm_call",
		Input:      request.Query,
		Output:     result,
		Reasoning:  "Direct LLM call for simple query",
		Duration:   time.Since(stepStart),
		Metadata:   make(map[string]interface{}),
	}

	steps = append(steps, step)
	return result, steps, nil
}

// validateRequest validates an agent request
func (a *BaseAgent) validateRequest(request *AgentRequest) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if request.Query == "" {
		return fmt.Errorf("query cannot be empty")
	}

	if request.MaxSteps < 0 {
		return fmt.Errorf("max_steps cannot be negative")
	}

	if request.Temperature < 0 || request.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	return nil
}

// Health checks the health of the agent
func (a *BaseAgent) Health(ctx context.Context) error {
	// Check LLM health
	_, err := a.llm.Call(ctx, "Health check", llms.WithMaxTokens(1))
	if err != nil {
		return fmt.Errorf("LLM health check failed: %w", err)
	}

	// Check memory if enabled
	if a.memory != nil {
		// Memory doesn't have a direct health check, so we assume it's healthy
		a.logger.Debug("Memory health check passed")
	}

	// Check tools
	for _, tool := range a.tools {
		// Tools don't have a standard health check interface
		a.logger.Debug("Tool available", slog.String("tool", tool.Name()))
	}

	return nil
}

// ClearMemory clears the agent's memory
func (a *BaseAgent) ClearMemory(ctx context.Context) error {
	if a.memory != nil {
		a.memory.Clear(ctx)
		a.logger.Debug("Agent memory cleared", slog.String("agent_type", string(a.agentType)))
	}
	return nil
}

// GetMemorySize returns the current memory size
func (a *BaseAgent) GetMemorySize() int {
	if a.memory == nil {
		return 0
	}

	// This is a simplified implementation
	// In practice, you'd need to implement proper memory size calculation
	return 0 // TODO: Implement proper memory size calculation
}
