package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// DevelopmentAgent specializes in code-related queries and development tasks
type DevelopmentAgent struct {
	*BaseAgent
}

// NewDevelopmentAgent creates a new development agent
func NewDevelopmentAgent(llm llms.Model, logger *slog.Logger) *DevelopmentAgent {
	return &DevelopmentAgent{
		BaseAgent: NewBaseAgent(TypeDevelopment, llm, logger),
	}
}

// executeStep overrides base implementation for development-specific logic
func (a *DevelopmentAgent) executeStep(ctx context.Context, request *Request, stepNum int, previousResult string) (string, bool, error) {
	// Build development-specific prompt
	prompt := a.buildDevelopmentPrompt(request, stepNum, previousResult)

	// Generate response
	response, err := llms.GenerateFromSinglePrompt(ctx, a.llm, prompt)
	if err != nil {
		return "", false, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Check for completion indicators
	done := false
	if stepNum >= request.MaxSteps-1 ||
		strings.Contains(strings.ToLower(response), "implementation complete") ||
		strings.Contains(strings.ToLower(response), "code complete") {
		done = true
	}

	return response, done, nil
}

// buildDevelopmentPrompt builds specialized prompts for development tasks
func (a *DevelopmentAgent) buildDevelopmentPrompt(request *Request, stepNum int, previousResult string) string {
	prompt := "You are an expert development agent specializing in Go programming, software architecture, and best practices.\n\n"

	if stepNum == 0 {
		prompt += fmt.Sprintf("Task: %s\n\n", request.Query)

		// Add development-specific context
		prompt += "Guidelines:\n"
		prompt += "- Follow Go best practices and idiomatic patterns\n"
		prompt += "- Use clear, descriptive naming conventions\n"
		prompt += "- Include error handling with wrapped errors\n"
		prompt += "- Write clean, maintainable code\n"
		prompt += "- Consider performance and scalability\n\n"

		if len(request.Context) > 0 {
			prompt += "Context:\n"
			for k, v := range request.Context {
				prompt += fmt.Sprintf("- %s: %v\n", k, v)
			}
			prompt += "\n"
		}

		// Add available tools if specified
		if len(request.Tools) > 0 {
			prompt += "Available tools: " + strings.Join(request.Tools, ", ") + "\n\n"
		}

		prompt += "Provide a detailed solution with code examples where appropriate."
	} else {
		prompt += fmt.Sprintf("Continue working on: %s\n\n", request.Query)
		prompt += fmt.Sprintf("Previous work:\n%s\n\n", previousResult)
		prompt += "Continue the implementation or finalize with 'Implementation complete'."
	}

	return prompt
}

// Execute implements the Agent interface with development-specific execution
func (a *DevelopmentAgent) Execute(ctx context.Context, request *Request) (*Response, error) {
	// Log development-specific request details
	a.logger.Info("Development agent executing request",
		slog.String("query", request.Query),
		slog.Int("available_tools", len(request.Tools)))

	// Delegate to base implementation which will call our overridden executeStep
	return a.BaseAgent.Execute(ctx, request)
}
