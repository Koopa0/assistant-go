// Package agent provides the unified agent system for the intelligent assistant.
// All agent-related functionality is centralized here for clarity and simplicity.
package agent

import (
	"context"
	"fmt"
	"time"
)

// Agent defines the core interface for all agents
// Simple and practical - what agents actually need to do
type Agent interface {
	// Execute processes a request and returns a response
	Execute(ctx context.Context, request Request) (*Response, error)
	// Name returns the agent's identifier
	Name() string
	// Type returns the agent's specialization
	Type() AgentType
}

// AgentType identifies what kind of agent this is
type AgentType string

const (
	TypeDevelopment    AgentType = "development"
	TypeDatabase       AgentType = "database"
	TypeInfrastructure AgentType = "infrastructure"
	TypeResearch       AgentType = "research"
	TypeSecurity       AgentType = "security"
	TypeTesting        AgentType = "testing"
	TypeGeneral        AgentType = "general"
)

// Request contains everything needed to process an agent request
// Practical fields based on actual usage
type Request struct {
	// Core fields
	Query   string                 `json:"query"`           // The user's question or task
	Context map[string]interface{} `json:"context"`         // Contextual information
	
	// Control fields
	MaxSteps    int           `json:"max_steps,omitempty"`    // Limit execution steps
	Timeout     time.Duration `json:"timeout,omitempty"`      // Request timeout
	Temperature float64       `json:"temperature,omitempty"`  // LLM temperature (0-2)
	
	// Extension fields
	Tools      []string               `json:"tools,omitempty"`      // Specific tools to use
	Parameters map[string]interface{} `json:"parameters,omitempty"` // Additional parameters
	Metadata   map[string]interface{} `json:"metadata,omitempty"`   // Request metadata
}

// Response contains the agent's execution result
// Everything needed to understand what happened
type Response struct {
	// Core fields
	Result  string `json:"result"`  // The actual response
	Success bool   `json:"success"` // Whether execution succeeded
	Error   string `json:"error,omitempty"` // Error message if failed
	
	// Execution details
	Steps         []Step        `json:"steps,omitempty"`    // What the agent did
	ExecutionTime time.Duration `json:"execution_time"`     // How long it took
	TokensUsed    int           `json:"tokens_used,omitempty"` // LLM tokens consumed
	
	// Extension fields
	Confidence float64                `json:"confidence,omitempty"` // Result confidence (0-1)
	Metadata   map[string]interface{} `json:"metadata,omitempty"`   // Response metadata
}

// Step represents one action taken during execution
// Helps understand agent reasoning
type Step struct {
	Number    int           `json:"number"`           // Step sequence (1-based)
	Action    string        `json:"action"`           // What was done
	Input     string        `json:"input,omitempty"`  // Input to the action
	Output    string        `json:"output,omitempty"` // Result of the action
	Duration  time.Duration `json:"duration"`         // How long this step took
	ToolUsed  string        `json:"tool_used,omitempty"` // Tool used (if any)
}

// Capability describes something an agent can do
// Used for agent discovery and routing
type Capability struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Examples    []string `json:"examples,omitempty"`
}

// Tool represents an external capability an agent can use
// Simple interface - tools just need to execute
type Tool interface {
	Name() string
	Execute(ctx context.Context, input string) (string, error)
}

// Manager coordinates multiple agents
type Manager interface {
	// Register adds an agent to the manager
	Register(agent Agent) error
	// Get returns a specific agent by type
	Get(agentType AgentType) (Agent, error)
	// Execute routes a request to the appropriate agent
	Execute(ctx context.Context, agentType AgentType, request Request) (*Response, error)
	// List returns all registered agents
	List() []Agent
}

// NewRequest creates a request with defaults
func NewRequest(query string) Request {
	return Request{
		Query:       query,
		Context:     make(map[string]interface{}),
		MaxSteps:    5,
		Temperature: 0.7,
		Timeout:     30 * time.Second,
		Parameters:  make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}
}

// Validate checks if a request is valid
func (r Request) Validate() error {
	if r.Query == "" {
		return fmt.Errorf("query cannot be empty")
	}
	if r.Temperature < 0 || r.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	if r.MaxSteps < 0 {
		return fmt.Errorf("max_steps cannot be negative")
	}
	return nil
}

// AddStep adds a step to the response
func (r *Response) AddStep(action, input, output string, duration time.Duration) {
	step := Step{
		Number:   len(r.Steps) + 1,
		Action:   action,
		Input:    input,
		Output:   output,
		Duration: duration,
	}
	r.Steps = append(r.Steps, step)
}

// SetError marks the response as failed with an error
func (r *Response) SetError(err error) {
	r.Success = false
	r.Error = err.Error()
}