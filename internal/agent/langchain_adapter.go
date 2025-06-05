package agent

import (
	"context"
	"time"
)

// LangChainAdapter provides compatibility with LangChain agent interface
// Only use this when you need to integrate with LangChain
type LangChainAdapter struct {
	agent Agent
}

// NewLangChainAdapter wraps an agent for LangChain compatibility
func NewLangChainAdapter(agent Agent) *LangChainAdapter {
	return &LangChainAdapter{agent: agent}
}

// Execute converts LangChain-style parameters to our format
func (a *LangChainAdapter) Execute(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	// Extract parameters from LangChain format
	query, _ := params["query"].(string)
	context, _ := params["context"].(map[string]interface{})
	tools, _ := params["tools"].([]string)
	maxSteps, _ := params["max_steps"].(int)
	temperature, _ := params["temperature"].(float64)
	
	// Create our request
	req := Request{
		Query:       query,
		Context:     context,
		Tools:       tools,
		MaxSteps:    maxSteps,
		Temperature: temperature,
		Parameters:  params, // Store original params
	}
	
	// Set defaults if not provided
	if req.MaxSteps == 0 {
		req.MaxSteps = 5
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	
	// Execute using our agent
	resp, err := a.agent.Execute(ctx, req)
	if err != nil {
		return nil, err
	}
	
	// Convert response to LangChain format
	result := map[string]interface{}{
		"result":         resp.Result,
		"success":        resp.Success,
		"error":          resp.Error,
		"execution_time": resp.ExecutionTime.Seconds(),
		"tokens_used":    resp.TokensUsed,
		"confidence":     resp.Confidence,
		"metadata":       resp.Metadata,
	}
	
	// Add steps if present
	if len(resp.Steps) > 0 {
		steps := make([]map[string]interface{}, len(resp.Steps))
		for i, step := range resp.Steps {
			steps[i] = map[string]interface{}{
				"step_number": step.Number,
				"action":      step.Action,
				"input":       step.Input,
				"output":      step.Output,
				"duration":    step.Duration.Seconds(),
				"tool_used":   step.ToolUsed,
			}
		}
		result["steps"] = steps
	}
	
	return result, nil
}

// GetType returns the wrapped agent's type
func (a *LangChainAdapter) GetType() string {
	return string(a.agent.Type())
}

// GetCapabilities returns capabilities in LangChain format
func (a *LangChainAdapter) GetCapabilities() []map[string]interface{} {
	// Try to get capabilities if the agent supports them
	if baseAgent, ok := a.agent.(*BaseAgent); ok {
		caps := baseAgent.GetCapabilities()
		result := make([]map[string]interface{}, len(caps))
		for i, cap := range caps {
			result[i] = map[string]interface{}{
				"name":        cap.Name,
				"description": cap.Description,
				"examples":    cap.Examples,
			}
		}
		return result
	}
	
	// Return empty if agent doesn't support capabilities
	return []map[string]interface{}{}
}

// ConvertRequestFromLangChain converts a LangChain agent request to our format
func ConvertRequestFromLangChain(lcReq map[string]interface{}) Request {
	req := NewRequest("")
	
	if query, ok := lcReq["query"].(string); ok {
		req.Query = query
	}
	
	if context, ok := lcReq["context"].(map[string]interface{}); ok {
		req.Context = context
	}
	
	if tools, ok := lcReq["tools"].([]string); ok {
		req.Tools = tools
	}
	
	if maxSteps, ok := lcReq["max_steps"].(int); ok {
		req.MaxSteps = maxSteps
	}
	
	if temperature, ok := lcReq["temperature"].(float64); ok {
		req.Temperature = temperature
	}
	
	if timeout, ok := lcReq["timeout"].(time.Duration); ok {
		req.Timeout = timeout
	}
	
	if metadata, ok := lcReq["metadata"].(map[string]interface{}); ok {
		req.Metadata = metadata
	}
	
	return req
}

// ConvertResponseToLangChain converts our response to LangChain format
func ConvertResponseToLangChain(resp *Response) map[string]interface{} {
	result := map[string]interface{}{
		"result":         resp.Result,
		"steps":          convertStepsToLangChain(resp.Steps),
		"tools_used":     []string{}, // Extract from steps if needed
		"execution_time": resp.ExecutionTime,
		"tokens_used":    resp.TokensUsed,
		"metadata":       resp.Metadata,
	}
	
	// Extract tools used from steps
	toolsUsed := make(map[string]bool)
	for _, step := range resp.Steps {
		if step.ToolUsed != "" {
			toolsUsed[step.ToolUsed] = true
		}
	}
	
	tools := make([]string, 0, len(toolsUsed))
	for tool := range toolsUsed {
		tools = append(tools, tool)
	}
	result["tools_used"] = tools
	
	return result
}

// convertStepsToLangChain converts steps to LangChain format
func convertStepsToLangChain(steps []Step) []map[string]interface{} {
	result := make([]map[string]interface{}, len(steps))
	for i, step := range steps {
		result[i] = map[string]interface{}{
			"step_number": step.Number,
			"action":      step.Action,
			"tool":        step.ToolUsed,
			"input":       step.Input,
			"output":      step.Output,
			"reasoning":   "", // LangChain expects this
			"duration":    step.Duration,
			"metadata":    map[string]interface{}{},
		}
	}
	return result
}