package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// BaseAgent provides common functionality for all agents
type BaseAgent struct {
	agentType AgentType
	llm       llms.Model
	logger    *slog.Logger
	tools     []string
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(agentType AgentType, llm llms.Model, logger *slog.Logger) *BaseAgent {
	return &BaseAgent{
		agentType: agentType,
		llm:       llm,
		logger:    logger,
		tools:     []string{},
	}
}

// Execute implements the Agent interface with basic execution logic
func (a *BaseAgent) Execute(ctx context.Context, request *Request) (*Response, error) {
	start := time.Now()

	// Validate request
	if err := a.validateRequest(request); err != nil {
		return &Response{
			Result:        fmt.Sprintf("Validation error: %v", err),
			Success:       false,
			ExecutionTime: time.Since(start),
		}, nil
	}

	// Log the request
	a.logger.Info("Agent executing request",
		slog.String("agent_type", string(a.agentType)),
		slog.String("query", request.Query),
		slog.Int("max_steps", request.MaxSteps))

	// Execute steps
	steps := []Step{}
	result := ""

	for i := 0; i < request.MaxSteps; i++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return &Response{
				Result:        "Execution cancelled",
				Success:       false,
				ExecutionTime: time.Since(start),
				Steps:         steps,
			}, ctx.Err()
		default:
		}

		// Execute a step
		stepResult, done, err := a.executeStep(ctx, request, i, result)
		if err != nil {
			a.logger.Error("Step execution failed",
				slog.String("agent_type", string(a.agentType)),
				slog.Int("step", i),
				slog.String("error", err.Error()))

			return &Response{
				Result:        fmt.Sprintf("Step %d failed: %v", i, err),
				Success:       false,
				ExecutionTime: time.Since(start),
				Steps:         steps,
			}, nil
		}

		steps = append(steps, Step{
			Action: fmt.Sprintf("Step %d", i),
			Result: stepResult,
		})

		result = stepResult

		if done {
			break
		}
	}

	return &Response{
		Result:        result,
		Success:       true,
		Confidence:    0.8, // Default confidence
		ExecutionTime: time.Since(start),
		Steps:         steps,
	}, nil
}

// validateRequest validates the agent request
func (a *BaseAgent) validateRequest(request *Request) error {
	if request.Query == "" {
		return fmt.Errorf("query is required")
	}

	if request.MaxSteps <= 0 {
		request.MaxSteps = 5 // Default
	}

	if request.Temperature <= 0 {
		request.Temperature = 0.7 // Default
	}

	return nil
}

// executeStep executes a single step - to be overridden by specific agents
func (a *BaseAgent) executeStep(ctx context.Context, request *Request, stepNum int, previousResult string) (string, bool, error) {
	// Base implementation - just use LLM directly
	if a.llm == nil {
		return "No LLM configured", true, nil
	}

	// Build prompt
	prompt := a.buildPrompt(request, stepNum, previousResult)

	// Generate response
	response, err := llms.GenerateFromSinglePrompt(ctx, a.llm, prompt)
	if err != nil {
		return "", false, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Simple completion check - if response contains "DONE" or "COMPLETE"
	done := false
	if stepNum >= request.MaxSteps-1 {
		done = true
	}

	return response, done, nil
}

// buildPrompt builds the prompt for the LLM
func (a *BaseAgent) buildPrompt(request *Request, stepNum int, previousResult string) string {
	prompt := fmt.Sprintf("You are a %s agent. ", a.agentType)

	if stepNum == 0 {
		prompt += fmt.Sprintf("User query: %s\n", request.Query)

		if len(request.Context) > 0 {
			prompt += "Context:\n"
			for k, v := range request.Context {
				prompt += fmt.Sprintf("- %s: %v\n", k, v)
			}
		}

		prompt += "\nProvide a helpful response."
	} else {
		prompt += fmt.Sprintf("Continue working on: %s\n", request.Query)
		prompt += fmt.Sprintf("Previous result: %s\n", previousResult)
		prompt += "\nContinue or finalize your response."
	}

	return prompt
}
