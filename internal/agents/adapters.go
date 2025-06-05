package agents

import (
	"context"
	"fmt"
	"time"

	core "github.com/koopa0/assistant-go/internal/core/agents"
)

// RequestAdapter adapts between different request types
type RequestAdapter interface {
	// ToCoreRequest converts to core.Request
	ToCoreRequest(req Request) core.Request
	// FromCoreRequest converts from core.Request
	FromCoreRequest(req core.Request) Request
}

// ResponseAdapter adapts between different response types
type ResponseAdapter interface {
	// ToCoreResponse converts to core.Response
	ToCoreResponse(resp *Response, agentName string, requestID string) *core.Response
	// FromCoreResponse converts from core.Response
	FromCoreResponse(resp *core.Response) *Response
}

// DefaultRequestAdapter provides standard request adaptation
type DefaultRequestAdapter struct{}

// ToCoreRequest converts agents.Request to core.Request
func (a *DefaultRequestAdapter) ToCoreRequest(req Request) core.Request {
	// Map Priority type (they're the same, just need type conversion)
	var priority core.Priority
	switch req.Priority {
	case PriorityLow:
		priority = core.PriorityLow
	case PriorityMedium:
		priority = core.PriorityMedium
	case PriorityHigh:
		priority = core.PriorityHigh
	case PriorityCritical:
		priority = core.PriorityCritical
	default:
		priority = core.PriorityMedium
	}

	// Generate request ID if not in metadata
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	if id, ok := req.Metadata["request_id"].(string); ok {
		requestID = id
	}

	// Extract user ID and session ID from metadata if available
	var userID, sessionID string
	if uid, ok := req.Metadata["user_id"].(string); ok {
		userID = uid
	}
	if sid, ok := req.Metadata["session_id"].(string); ok {
		sessionID = sid
	}

	// Determine request type based on query content or metadata
	requestType := core.RequestAnalyze // Default
	if rt, ok := req.Metadata["request_type"].(string); ok {
		requestType = core.RequestType(rt)
	}

	// Merge context and tools into core context
	coreContext := make(map[string]interface{})
	for k, v := range req.Context {
		coreContext[k] = v
	}
	if len(req.Tools) > 0 {
		coreContext["requested_tools"] = req.Tools
	}
	if req.MaxSteps > 0 {
		coreContext["max_steps"] = req.MaxSteps
	}
	if req.Temperature > 0 {
		coreContext["temperature"] = req.Temperature
	}

	return core.Request{
		ID:        requestID,
		Type:      requestType,
		Content:   req.Query, // Map Query to Content
		Context:   coreContext,
		Priority:  priority,
		Timeout:   req.Timeout,
		UserID:    userID,
		SessionID: sessionID,
		CreatedAt: time.Now(),
	}
}

// FromCoreRequest converts core.Request to agents.Request
func (a *DefaultRequestAdapter) FromCoreRequest(req core.Request) Request {
	// Map Priority type
	var priority Priority
	switch req.Priority {
	case core.PriorityLow:
		priority = PriorityLow
	case core.PriorityMedium:
		priority = PriorityMedium
	case core.PriorityHigh:
		priority = PriorityHigh
	case core.PriorityCritical:
		priority = PriorityCritical
	default:
		priority = PriorityMedium
	}

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
	metadata["created_at"] = req.CreatedAt

	return Request{
		Query:       req.Content, // Map Content to Query
		Context:     req.Context,
		Tools:       tools,
		MaxSteps:    maxSteps,
		Temperature: temperature,
		Priority:    priority,
		Timeout:     req.Timeout,
		Metadata:    metadata,
	}
}

// DefaultResponseAdapter provides standard response adaptation
type DefaultResponseAdapter struct{}

// ToCoreResponse converts agents.Response to core.Response
func (a *DefaultResponseAdapter) ToCoreResponse(resp *Response, agentName string, requestID string) *core.Response {
	// Determine success based on result
	success := resp.Result != ""
	var errorMsg string
	if !success {
		errorMsg = "No result generated"
	}

	// Extract suggestions from metadata if not already present
	suggestions := resp.Suggestions
	if suggestions == nil {
		if s, ok := resp.Metadata["suggestions"].([]string); ok {
			suggestions = s
		}
	}

	// Add steps information to metadata
	metadata := make(map[string]interface{})
	for k, v := range resp.Metadata {
		metadata[k] = v
	}
	metadata["steps_count"] = len(resp.Steps)
	metadata["tools_used"] = resp.ToolsUsed
	metadata["tokens_used"] = resp.TokensUsed

	// Add detailed steps if needed
	if len(resp.Steps) > 0 {
		stepsData := make([]map[string]interface{}, len(resp.Steps))
		for i, step := range resp.Steps {
			stepsData[i] = map[string]interface{}{
				"number":    step.Number,
				"action":    step.Action,
				"tool":      step.Tool,
				"reasoning": step.Reasoning,
				"duration":  step.Duration.String(),
			}
		}
		metadata["execution_steps"] = stepsData
	}

	return &core.Response{
		RequestID:   requestID,
		AgentName:   agentName,
		Content:     resp.Result, // Map Result to Content
		Success:     success,
		Error:       errorMsg,
		Confidence:  resp.Confidence,
		Duration:    resp.ExecutionTime, // Map ExecutionTime to Duration
		Metadata:    metadata,
		Suggestions: suggestions,
		CreatedAt:   time.Now(),
	}
}

// FromCoreResponse converts core.Response to agents.Response
func (a *DefaultResponseAdapter) FromCoreResponse(resp *core.Response) *Response {
	// Extract steps from metadata if available
	var steps []Step
	if stepsData, ok := resp.Metadata["execution_steps"].([]map[string]interface{}); ok {
		steps = make([]Step, len(stepsData))
		for i, stepData := range stepsData {
			steps[i] = Step{
				Number:    stepData["number"].(int),
				Action:    stepData["action"].(string),
				Tool:      stepData["tool"].(string),
				Reasoning: stepData["reasoning"].(string),
			}
		}
	}

	// Extract tools used and tokens from metadata
	var toolsUsed []string
	var tokensUsed int
	if tools, ok := resp.Metadata["tools_used"].([]string); ok {
		toolsUsed = tools
	}
	if tokens, ok := resp.Metadata["tokens_used"].(int); ok {
		tokensUsed = tokens
	}

	// Build metadata without extracted fields
	metadata := make(map[string]interface{})
	for k, v := range resp.Metadata {
		if k != "execution_steps" && k != "tools_used" && k != "tokens_used" && k != "steps_count" {
			metadata[k] = v
		}
	}

	return &Response{
		Result:        resp.Content, // Map Content to Result
		Steps:         steps,
		ToolsUsed:     toolsUsed,
		ExecutionTime: resp.Duration, // Map Duration to ExecutionTime
		TokensUsed:    tokensUsed,
		Confidence:    resp.Confidence,
		Suggestions:   resp.Suggestions,
		Metadata:      metadata,
	}
}

// CoreAgentAdapter wraps an agents.Agent to implement core.Agent
type CoreAgentAdapter struct {
	agent           Agent
	requestAdapter  RequestAdapter
	responseAdapter ResponseAdapter
}

// NewCoreAgentAdapter creates a new adapter that makes agents.Agent compatible with core.Agent
func NewCoreAgentAdapter(agent Agent) *CoreAgentAdapter {
	return &CoreAgentAdapter{
		agent:           agent,
		requestAdapter:  &DefaultRequestAdapter{},
		responseAdapter: &DefaultResponseAdapter{},
	}
}

// Execute implements core.Agent by adapting the request and response
func (a *CoreAgentAdapter) Execute(ctx context.Context, request core.Request) (*core.Response, error) {
	// Convert core request to agent request
	agentReq := a.requestAdapter.FromCoreRequest(request)

	// Execute using the wrapped agent
	agentResp, err := a.agent.Execute(ctx, agentReq)
	if err != nil {
		// Return error response
		return &core.Response{
			RequestID:  request.ID,
			AgentName:  a.agent.Name(),
			Content:    "",
			Success:    false,
			Error:      err.Error(),
			Confidence: 0,
			Duration:   0,
			CreatedAt:  time.Now(),
		}, nil
	}

	// Convert agent response to core response
	return a.responseAdapter.ToCoreResponse(agentResp, a.agent.Name(), request.ID), nil
}

// Name delegates to the wrapped agent
func (a *CoreAgentAdapter) Name() string {
	return a.agent.Name()
}

// Type delegates to the wrapped agent
func (a *CoreAgentAdapter) Type() core.AgentType {
	// Convert between agent types
	switch a.agent.Type() {
	case TypeDevelopment:
		return core.TypeDevelopment
	case TypeDatabase:
		return core.TypeDatabase
	case TypeInfrastructure:
		return core.TypeInfrastructure
	case TypeResearch:
		return core.TypeResearch
	case TypeGeneral:
		return core.TypeGeneral
	case TypeSecurity:
		return core.TypeSecurity
	case TypeTesting:
		return core.TypeTesting
	case TypeDeployment:
		return core.TypeDeployment
	case TypeMonitoring:
		return core.TypeMonitoring
	case TypeOptimization:
		return core.TypeOptimization
	default:
		return core.TypeGeneral
	}
}

// AgentAdapter wraps a core.Agent to implement agents.Agent
type AgentAdapter struct {
	coreAgent       core.Agent
	requestAdapter  RequestAdapter
	responseAdapter ResponseAdapter
}

// NewAgentAdapter creates a new adapter that makes core.Agent compatible with agents.Agent
func NewAgentAdapter(coreAgent core.Agent) *AgentAdapter {
	return &AgentAdapter{
		coreAgent:       coreAgent,
		requestAdapter:  &DefaultRequestAdapter{},
		responseAdapter: &DefaultResponseAdapter{},
	}
}

// Execute implements agents.Agent by adapting the request and response
func (a *AgentAdapter) Execute(ctx context.Context, request Request) (*Response, error) {
	// Convert agent request to core request
	coreReq := a.requestAdapter.ToCoreRequest(request)

	// Execute using the core agent
	coreResp, err := a.coreAgent.Execute(ctx, coreReq)
	if err != nil {
		return nil, err
	}

	// Check if the core response indicates an error
	if !coreResp.Success && coreResp.Error != "" {
		return nil, fmt.Errorf("agent execution failed: %s", coreResp.Error)
	}

	// Convert core response to agent response
	return a.responseAdapter.FromCoreResponse(coreResp), nil
}

// Name delegates to the core agent
func (a *AgentAdapter) Name() string {
	return a.coreAgent.Name()
}

// Type delegates to the core agent
func (a *AgentAdapter) Type() AgentType {
	// Convert between agent types
	switch a.coreAgent.Type() {
	case core.TypeDevelopment:
		return TypeDevelopment
	case core.TypeDatabase:
		return TypeDatabase
	case core.TypeInfrastructure:
		return TypeInfrastructure
	case core.TypeResearch:
		return TypeResearch
	case core.TypeGeneral:
		return TypeGeneral
	case core.TypeSecurity:
		return TypeSecurity
	case core.TypeTesting:
		return TypeTesting
	case core.TypeDeployment:
		return TypeDeployment
	case core.TypeMonitoring:
		return TypeMonitoring
	case core.TypeOptimization:
		return TypeOptimization
	default:
		return TypeGeneral
	}
}
