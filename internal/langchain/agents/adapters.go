package agents

import (
	"context"
	"fmt"
	"time"

	core "github.com/koopa0/assistant-go/internal/core/agents"
)

// LangChainToCoreAdapter adapts LangChain agent types to core agent types
type LangChainToCoreAdapter struct {
	baseAgent *BaseAgent
}

// NewLangChainToCoreAdapter creates a new adapter for LangChain agents
func NewLangChainToCoreAdapter(baseAgent *BaseAgent) *LangChainToCoreAdapter {
	return &LangChainToCoreAdapter{
		baseAgent: baseAgent,
	}
}

// Execute implements core.Agent interface
func (a *LangChainToCoreAdapter) Execute(ctx context.Context, request core.Request) (*core.Response, error) {
	// Convert core.Request to AgentRequest
	agentReq := a.toCoreRequest(request)

	// Execute using BaseAgent
	agentResp, err := a.baseAgent.Execute(ctx, agentReq)
	if err != nil {
		return &core.Response{
			RequestID:  request.ID,
			AgentName:  a.Name(),
			Content:    "",
			Success:    false,
			Error:      err.Error(),
			Confidence: 0,
			Duration:   0,
			CreatedAt:  time.Now(),
		}, nil
	}

	// Convert AgentResponse to core.Response
	return a.fromAgentResponse(agentResp, request.ID), nil
}

// Name returns the agent's name
func (a *LangChainToCoreAdapter) Name() string {
	return fmt.Sprintf("langchain_%s", a.baseAgent.agentType)
}

// Type returns the agent's type
func (a *LangChainToCoreAdapter) Type() core.AgentType {
	switch a.baseAgent.agentType {
	case AgentTypeDevelopment:
		return core.TypeDevelopment
	case AgentTypeDatabase:
		return core.TypeDatabase
	case AgentTypeInfrastructure:
		return core.TypeInfrastructure
	case AgentTypeResearch:
		return core.TypeResearch
	default:
		return core.TypeGeneral
	}
}

// toCoreRequest converts core.Request to AgentRequest
func (a *LangChainToCoreAdapter) toCoreRequest(req core.Request) *AgentRequest {
	// Extract tools, max steps, and temperature from context
	var tools []string
	var maxSteps int
	var temperature float64

	if t, ok := req.Context["requested_tools"].([]string); ok {
		tools = t
	}
	if ms, ok := req.Context["max_steps"].(int); ok {
		maxSteps = ms
	}
	if temp, ok := req.Context["temperature"].(float64); ok {
		temperature = temp
	}

	// Build metadata
	metadata := make(map[string]interface{})
	metadata["request_id"] = req.ID
	metadata["request_type"] = string(req.Type)
	metadata["user_id"] = req.UserID
	metadata["session_id"] = req.SessionID
	metadata["priority"] = string(req.Priority)
	metadata["created_at"] = req.CreatedAt

	return &AgentRequest{
		Query:       req.Content,
		Context:     req.Context,
		Tools:       tools,
		MaxSteps:    maxSteps,
		Temperature: temperature,
		Metadata:    metadata,
	}
}

// fromAgentResponse converts AgentResponse to core.Response
func (a *LangChainToCoreAdapter) fromAgentResponse(resp *AgentResponse, requestID string) *core.Response {
	// Build metadata including execution details
	metadata := make(map[string]interface{})
	for k, v := range resp.Metadata {
		metadata[k] = v
	}
	metadata["steps_count"] = len(resp.Steps)
	metadata["tools_used"] = resp.ToolsUsed
	metadata["tokens_used"] = resp.TokensUsed

	// Convert steps to a serializable format
	if len(resp.Steps) > 0 {
		stepsData := make([]map[string]interface{}, len(resp.Steps))
		for i, step := range resp.Steps {
			stepsData[i] = map[string]interface{}{
				"number":    step.StepNumber,
				"action":    step.Action,
				"tool":      step.Tool,
				"input":     step.Input,
				"output":    step.Output,
				"reasoning": step.Reasoning,
				"duration":  step.Duration.String(),
			}
		}
		metadata["execution_steps"] = stepsData
	}

	// Extract suggestions from steps or metadata
	var suggestions []string
	if s, ok := resp.Metadata["suggestions"].([]string); ok {
		suggestions = s
	}

	// Determine confidence based on result
	confidence := 0.8 // Default confidence
	if resp.Result == "" {
		confidence = 0.0
	}

	return &core.Response{
		RequestID:   requestID,
		AgentName:   a.Name(),
		Content:     resp.Result,
		Success:     resp.Result != "",
		Error:       "",
		Confidence:  confidence,
		Duration:    resp.ExecutionTime,
		Metadata:    metadata,
		Suggestions: suggestions,
		CreatedAt:   time.Now(),
	}
}

// CoreToLangChainRequestAdapter adapts core requests to LangChain requests
type CoreToLangChainRequestAdapter struct{}

// Adapt converts a core.Request to an AgentRequest
func (a *CoreToLangChainRequestAdapter) Adapt(req core.Request) *AgentRequest {
	// Extract LangChain-specific fields from context
	var tools []string
	var maxSteps int
	var temperature float64

	if t, ok := req.Context["requested_tools"].([]string); ok {
		tools = t
	}
	if ms, ok := req.Context["max_steps"].(int); ok {
		maxSteps = ms
	}
	if temp, ok := req.Context["temperature"].(float64); ok {
		temperature = temp
	}

	// Build metadata
	metadata := make(map[string]interface{})
	metadata["request_id"] = req.ID
	metadata["request_type"] = string(req.Type)
	metadata["user_id"] = req.UserID
	metadata["session_id"] = req.SessionID
	metadata["priority"] = string(req.Priority)
	metadata["timeout"] = req.Timeout
	metadata["created_at"] = req.CreatedAt

	return &AgentRequest{
		Query:       req.Content,
		Context:     req.Context,
		Tools:       tools,
		MaxSteps:    maxSteps,
		Temperature: temperature,
		Metadata:    metadata,
	}
}

// LangChainToCoreResponseAdapter adapts LangChain responses to core responses
type LangChainToCoreResponseAdapter struct{}

// Adapt converts an AgentResponse to a core.Response
func (a *LangChainToCoreResponseAdapter) Adapt(resp *AgentResponse, agentName string, requestID string) *core.Response {
	// Build comprehensive metadata
	metadata := make(map[string]interface{})
	for k, v := range resp.Metadata {
		metadata[k] = v
	}

	// Add execution details
	metadata["steps_count"] = len(resp.Steps)
	metadata["tools_used"] = resp.ToolsUsed
	metadata["tokens_used"] = resp.TokensUsed

	// Convert steps to metadata
	if len(resp.Steps) > 0 {
		stepsData := make([]map[string]interface{}, len(resp.Steps))
		for i, step := range resp.Steps {
			stepsData[i] = map[string]interface{}{
				"step_number": step.StepNumber,
				"action":      step.Action,
				"tool":        step.Tool,
				"input":       step.Input,
				"output":      step.Output,
				"reasoning":   step.Reasoning,
				"duration":    step.Duration.String(),
				"metadata":    step.Metadata,
			}
		}
		metadata["execution_steps"] = stepsData
	}

	// Extract suggestions if available
	var suggestions []string
	if s, ok := resp.Metadata["suggestions"].([]string); ok {
		suggestions = s
	}

	// Determine success and confidence
	success := resp.Result != ""
	confidence := 0.8
	if !success {
		confidence = 0.0
	}

	return &core.Response{
		RequestID:   requestID,
		AgentName:   agentName,
		Content:     resp.Result,
		Success:     success,
		Error:       "",
		Confidence:  confidence,
		Duration:    resp.ExecutionTime,
		Metadata:    metadata,
		Suggestions: suggestions,
		CreatedAt:   time.Now(),
	}
}

// SpecializedAgentAdapter wraps specialized LangChain agents to implement core.Agent
type SpecializedAgentAdapter struct {
	agent       interface{} // Can be DevelopmentAgent, DatabaseAgent, etc.
	baseAgent   *BaseAgent
	agentType   core.AgentType
	executeFunc func(ctx context.Context, request *AgentRequest) (*AgentResponse, error)
}

// NewDevelopmentAgentAdapter creates an adapter for DevelopmentAgent
func NewDevelopmentAgentAdapter(agent *DevelopmentAgent) *SpecializedAgentAdapter {
	return &SpecializedAgentAdapter{
		agent:       agent,
		baseAgent:   agent.BaseAgent,
		agentType:   core.TypeDevelopment,
		executeFunc: agent.Execute,
	}
}

// NewDatabaseAgentAdapter creates an adapter for DatabaseAgent
func NewDatabaseAgentAdapter(agent *DatabaseAgent) *SpecializedAgentAdapter {
	return &SpecializedAgentAdapter{
		agent:       agent,
		baseAgent:   agent.BaseAgent,
		agentType:   core.TypeDatabase,
		executeFunc: agent.Execute,
	}
}

// NewInfrastructureAgentAdapter creates an adapter for InfrastructureAgent
func NewInfrastructureAgentAdapter(agent *InfrastructureAgent) *SpecializedAgentAdapter {
	return &SpecializedAgentAdapter{
		agent:       agent,
		baseAgent:   agent.BaseAgent,
		agentType:   core.TypeInfrastructure,
		executeFunc: agent.Execute,
	}
}

// NewResearchAgentAdapter creates an adapter for ResearchAgent
func NewResearchAgentAdapter(agent *ResearchAgent) *SpecializedAgentAdapter {
	return &SpecializedAgentAdapter{
		agent:       agent,
		baseAgent:   agent.BaseAgent,
		agentType:   core.TypeResearch,
		executeFunc: agent.Execute,
	}
}

// Execute implements core.Agent interface
func (a *SpecializedAgentAdapter) Execute(ctx context.Context, request core.Request) (*core.Response, error) {
	// Convert core.Request to AgentRequest
	adapter := &CoreToLangChainRequestAdapter{}
	agentReq := adapter.Adapt(request)

	// Execute using the specialized agent
	agentResp, err := a.executeFunc(ctx, agentReq)
	if err != nil {
		return &core.Response{
			RequestID:  request.ID,
			AgentName:  a.Name(),
			Content:    "",
			Success:    false,
			Error:      err.Error(),
			Confidence: 0,
			Duration:   0,
			Metadata:   map[string]interface{}{"error_type": "execution_error"},
			CreatedAt:  time.Now(),
		}, nil
	}

	// Convert AgentResponse to core.Response
	respAdapter := &LangChainToCoreResponseAdapter{}
	return respAdapter.Adapt(agentResp, a.Name(), request.ID), nil
}

// Name returns the agent's name
func (a *SpecializedAgentAdapter) Name() string {
	return fmt.Sprintf("langchain_%s_agent", a.agentType)
}

// Type returns the agent's type
func (a *SpecializedAgentAdapter) Type() core.AgentType {
	return a.agentType
}

