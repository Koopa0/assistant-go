package agents

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// Agent represents a practical AI agent that can execute tasks
// Following Go principles: small interface, discovered through use
type Agent interface {
	// Execute processes a request and returns a response
	Execute(ctx context.Context, request Request) (*Response, error)
	// Name returns the agent's name for identification
	Name() string
	// Type returns the agent's specialized type
	Type() AgentType
}

// AgentType identifies the agent's specialization
type AgentType string

const (
	TypeDevelopment    AgentType = "development"
	TypeDatabase       AgentType = "database"
	TypeInfrastructure AgentType = "infrastructure"
	TypeResearch       AgentType = "research"
	TypeGeneral        AgentType = "general"
	TypeSecurity       AgentType = "security"
	TypeTesting        AgentType = "testing"
	TypeDeployment     AgentType = "deployment"
	TypeMonitoring     AgentType = "monitoring"
	TypeOptimization   AgentType = "optimization"
)

// Priority defines request priority levels
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// Request contains the input for agent execution
// Enhanced with useful fields from all implementations
type Request struct {
	Query       string                 // The main query or task
	Context     map[string]interface{} // Additional context
	Tools       []string               // Specific tools to use
	MaxSteps    int                    // Maximum execution steps
	Temperature float64                // LLM temperature (0-2)

	// Enhanced fields
	Priority Priority               // Request priority
	Timeout  time.Duration          // Request timeout
	Metadata map[string]interface{} // Extensible metadata
}

// Response contains the agent's execution result
// Enhanced with useful fields from all implementations
type Response struct {
	Result        string        // The main result
	Steps         []Step        // Execution steps taken
	ToolsUsed     []string      // Tools actually used
	ExecutionTime time.Duration // Total execution time
	TokensUsed    int           // LLM tokens consumed

	// Enhanced fields
	Confidence  float64                // Confidence in the result (0-1)
	Suggestions []string               // Additional suggestions
	Metadata    map[string]interface{} // Response metadata
}

// Step represents a single execution step
// Enhanced with reasoning field
type Step struct {
	Number    int                    // Step number (1-based)
	Action    string                 // What action was taken
	Tool      string                 // Tool used (if any)
	Input     string                 // Input to the action
	Output    string                 // Output from the action
	Reasoning string                 // Why this step was taken
	Duration  time.Duration          // Step duration
	Metadata  map[string]interface{} // Step metadata
}

// Capability represents something an agent can do
type Capability struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// BaseAgent provides common functionality for all agents
// Concrete type, not an interface - compose, don't inherit
type BaseAgent struct {
	name         string
	agentType    AgentType
	llm          llms.Model
	tools        map[string]Tool // Simple map, not complex registry
	logger       *slog.Logger
	capabilities []Capability // Agent capabilities for discovery
}

// Tool represents a capability that an agent can use
// Simple interface discovered from actual tool usage
type Tool interface {
	// Name returns the tool's name
	Name() string
	// Execute runs the tool with given input
	Execute(ctx context.Context, input string) (string, error)
}

// NewBaseAgent creates a new base agent
// Constructor follows Go conventions
func NewBaseAgent(name string, agentType AgentType, llm llms.Model, logger *slog.Logger) *BaseAgent {
	return &BaseAgent{
		name:         name,
		agentType:    agentType,
		llm:          llm,
		tools:        make(map[string]Tool),
		logger:       logger,
		capabilities: make([]Capability, 0),
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

// AddTool adds a tool to the agent
// Simple method, no complex registration
func (a *BaseAgent) AddTool(tool Tool) {
	a.tools[tool.Name()] = tool
	a.logger.Debug("Added tool to agent",
		slog.String("agent", a.name),
		slog.String("tool", tool.Name()))
}

// AddCapability adds a capability to the agent
func (a *BaseAgent) AddCapability(capability Capability) {
	a.capabilities = append(a.capabilities, capability)
	a.logger.Debug("Added capability to agent",
		slog.String("agent", a.name),
		slog.String("capability", capability.Name))
}

// GetCapabilities returns the agent's capabilities
func (a *BaseAgent) GetCapabilities() []Capability {
	return a.capabilities
}

// Execute implements basic execution logic
// Can be overridden by embedding types
func (a *BaseAgent) Execute(ctx context.Context, req Request) (*Response, error) {
	startTime := time.Now()

	// Validate request
	if req.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Set defaults
	if req.MaxSteps <= 0 {
		req.MaxSteps = 5
	}
	if req.Temperature < 0 || req.Temperature > 2 {
		req.Temperature = 0.7
	}

	a.logger.Info("Executing agent request",
		slog.String("agent", a.name),
		slog.String("query", req.Query))

	// Simple direct LLM call for base implementation
	result, err := a.llm.Call(ctx, req.Query,
		llms.WithTemperature(req.Temperature))
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Build response
	response := &Response{
		Result: result,
		Steps: []Step{
			{
				Number:   1,
				Action:   "llm_call",
				Input:    req.Query,
				Output:   result,
				Duration: time.Since(startTime),
			},
		},
		ExecutionTime: time.Since(startTime),
	}

	return response, nil
}

// DevelopmentAgent specializes in development tasks
// Concrete type that embeds BaseAgent
type DevelopmentAgent struct {
	*BaseAgent
}

// NewDevelopmentAgent creates a new development agent
func NewDevelopmentAgent(llm llms.Model, logger *slog.Logger) *DevelopmentAgent {
	base := NewBaseAgent("dev_agent", TypeDevelopment, llm, logger)
	return &DevelopmentAgent{BaseAgent: base}
}

// Execute overrides base execution with development-specific logic
func (a *DevelopmentAgent) Execute(ctx context.Context, req Request) (*Response, error) {
	// Add development-specific context
	if req.Context == nil {
		req.Context = make(map[string]interface{})
	}
	req.Context["expertise"] = "software development, debugging, code review"

	// Use base implementation with enhanced context
	return a.BaseAgent.Execute(ctx, req)
}

// DatabaseAgent specializes in database operations
type DatabaseAgent struct {
	*BaseAgent
}

// NewDatabaseAgent creates a new database agent
func NewDatabaseAgent(llm llms.Model, logger *slog.Logger) *DatabaseAgent {
	base := NewBaseAgent("db_agent", TypeDatabase, llm, logger)
	return &DatabaseAgent{BaseAgent: base}
}

// Execute overrides base execution with database-specific logic
func (a *DatabaseAgent) Execute(ctx context.Context, req Request) (*Response, error) {
	// Add database-specific context
	if req.Context == nil {
		req.Context = make(map[string]interface{})
	}
	req.Context["expertise"] = "SQL, query optimization, schema design"

	// Use base implementation with enhanced context
	return a.BaseAgent.Execute(ctx, req)
}

// Manager coordinates multiple agents
// Simple coordinator, not complex supervisor hierarchy
type Manager struct {
	agents map[AgentType]Agent
	logger *slog.Logger
}

// NewManager creates a new agent manager
func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		agents: make(map[AgentType]Agent),
		logger: logger,
	}
}

// RegisterAgent adds an agent to the manager
func (m *Manager) RegisterAgent(agent Agent) {
	m.agents[agent.Type()] = agent
	m.logger.Info("Registered agent",
		slog.String("name", agent.Name()),
		slog.String("type", string(agent.Type())))
}

// Execute routes request to appropriate agent
func (m *Manager) Execute(ctx context.Context, agentType AgentType, req Request) (*Response, error) {
	agent, exists := m.agents[agentType]
	if !exists {
		// Fallback to general agent if available
		if general, ok := m.agents[TypeGeneral]; ok {
			agent = general
		} else {
			return nil, fmt.Errorf("no agent available for type: %s", agentType)
		}
	}

	return agent.Execute(ctx, req)
}

// GetAgent returns a specific agent by type
func (m *Manager) GetAgent(agentType AgentType) (Agent, bool) {
	agent, exists := m.agents[agentType]
	return agent, exists
}
