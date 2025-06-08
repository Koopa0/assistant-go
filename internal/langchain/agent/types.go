package agent

import (
	"context"
	"time"
)

// AgentType represents the type of agent
type AgentType string

const (
	TypeDevelopment    AgentType = "development"
	TypeGeneral        AgentType = "general"
	TypeDatabase       AgentType = "database"
	TypeInfrastructure AgentType = "infrastructure"
	TypeResearch       AgentType = "research"
)

// Request represents an agent request
type Request struct {
	Query       string                 `json:"query"`
	Context     map[string]interface{} `json:"context"`
	Tools       []string               `json:"tools"`
	MaxSteps    int                    `json:"max_steps"`
	Temperature float64                `json:"temperature"`
}

// Response represents an agent response
type Response struct {
	Result        string        `json:"result"`
	Success       bool          `json:"success"`
	Confidence    float64       `json:"confidence"`
	ExecutionTime time.Duration `json:"execution_time"`
	Steps         []Step        `json:"steps"`
}

// Step represents a single step in agent execution
type Step struct {
	Action string `json:"action"`
	Result string `json:"result"`
}

// SimpleManager manages agents
type SimpleManager struct {
	agents map[AgentType]interface{}
}

// NewSimpleManager creates a new simple manager
func NewSimpleManager() *SimpleManager {
	return &SimpleManager{
		agents: make(map[AgentType]interface{}),
	}
}

// Agent interface for all agent types
type Agent interface {
	Execute(ctx context.Context, request *Request) (*Response, error)
}
