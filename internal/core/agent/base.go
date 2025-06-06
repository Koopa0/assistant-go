package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// BaseAgent provides common functionality for all agents
// Embed this to create specialized agents
type BaseAgent struct {
	name         string
	agentType    AgentType
	llm          llms.Model
	tools        map[string]Tool
	capabilities []Capability
	logger       *slog.Logger
}

// NewBaseAgent creates a new base agent with common functionality
func NewBaseAgent(name string, agentType AgentType, llm llms.Model, logger *slog.Logger) *BaseAgent {
	return &BaseAgent{
		name:         name,
		agentType:    agentType,
		llm:          llm,
		tools:        make(map[string]Tool),
		capabilities: make([]Capability, 0),
		logger:       logger.With(slog.String("agent", name)),
	}
}

// Name returns the agent's name
func (a *BaseAgent) Name() string {
	return a.name
}

// Type returns the agent's type
func (a *BaseAgent) Type() AgentType {
	return a.agentType
}

// AddTool registers a tool with the agent
func (a *BaseAgent) AddTool(tool Tool) {
	a.tools[tool.Name()] = tool
	a.logger.Debug("added tool", slog.String("tool", tool.Name()))
}

// AddCapability registers a capability
func (a *BaseAgent) AddCapability(cap Capability) {
	a.capabilities = append(a.capabilities, cap)
	a.logger.Debug("added capability", slog.String("capability", cap.Name))
}

// GetCapabilities returns all agent capabilities
func (a *BaseAgent) GetCapabilities() []Capability {
	return a.capabilities
}

// Execute implements basic agent execution
// Override this in specialized agents for custom behavior
func (a *BaseAgent) Execute(ctx context.Context, req Request) (*Response, error) {
	startTime := time.Now()

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Apply timeout if specified
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	a.logger.Info("executing request",
		slog.String("query", req.Query),
		slog.Int("max_steps", req.MaxSteps))

	// Initialize response
	response := &Response{
		Success:  true,
		Steps:    make([]Step, 0),
		Metadata: make(map[string]interface{}),
	}

	// Execute with LLM
	stepStart := time.Now()
	result, err := a.executeLLM(ctx, req)
	if err != nil {
		response.SetError(err)
		return response, err
	}

	// Record the step
	response.AddStep("llm_call", req.Query, result, time.Since(stepStart))
	response.Result = result

	// Set execution time
	response.ExecutionTime = time.Since(startTime)

	a.logger.Info("request completed",
		slog.Bool("success", response.Success),
		slog.Duration("duration", response.ExecutionTime))

	return response, nil
}

// executeLLM handles the actual LLM call
func (a *BaseAgent) executeLLM(ctx context.Context, req Request) (string, error) {
	// Prepare options
	opts := []llms.CallOption{
		llms.WithTemperature(req.Temperature),
	}

	// Add any context as part of the query if needed
	query := req.Query
	if req.Context != nil && len(req.Context) > 0 {
		// Convert context to string representation
		// This is simplified - in practice you'd format this better
		contextStr := fmt.Sprintf("Context: %v\n\nQuery: %s", req.Context, req.Query)
		query = contextStr
	}

	// Call LLM
	result, err := a.llm.Call(ctx, query, opts...)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	return result, nil
}

// executeTool executes a tool if available
func (a *BaseAgent) executeTool(ctx context.Context, toolName, input string) (string, error) {
	tool, exists := a.tools[toolName]
	if !exists {
		return "", fmt.Errorf("tool not found: %s", toolName)
	}

	a.logger.Debug("executing tool",
		slog.String("tool", toolName),
		slog.String("input", input))

	result, err := tool.Execute(ctx, input)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	return result, nil
}

// SimpleManager provides basic agent management
type SimpleManager struct {
	agents map[AgentType]Agent
	logger *slog.Logger
}

// NewSimpleManager creates a new agent manager
func NewSimpleManager(logger *slog.Logger) *SimpleManager {
	return &SimpleManager{
		agents: make(map[AgentType]Agent),
		logger: logger,
	}
}

// Register adds an agent to the manager
func (m *SimpleManager) Register(agent Agent) error {
	if agent == nil {
		return fmt.Errorf("agent cannot be nil")
	}

	agentType := agent.Type()
	if _, exists := m.agents[agentType]; exists {
		return fmt.Errorf("agent type already registered: %s", agentType)
	}

	m.agents[agentType] = agent
	m.logger.Info("registered agent",
		slog.String("name", agent.Name()),
		slog.String("type", string(agentType)))

	return nil
}

// Get returns an agent by type
func (m *SimpleManager) Get(agentType AgentType) (Agent, error) {
	agent, exists := m.agents[agentType]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentType)
	}
	return agent, nil
}

// Execute routes a request to the appropriate agent
func (m *SimpleManager) Execute(ctx context.Context, agentType AgentType, req Request) (*Response, error) {
	agent, err := m.Get(agentType)
	if err != nil {
		// Try general agent as fallback
		if general, exists := m.agents[TypeGeneral]; exists {
			m.logger.Info("falling back to general agent",
				slog.String("requested", string(agentType)))
			agent = general
		} else {
			return nil, err
		}
	}

	return agent.Execute(ctx, req)
}

// List returns all registered agents
func (m *SimpleManager) List() []Agent {
	agents := make([]Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	return agents
}
