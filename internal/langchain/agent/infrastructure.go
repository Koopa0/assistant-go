package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// InfrastructureAgent specializes in infrastructure management and DevOps tasks
type InfrastructureAgent struct {
	*BaseAgent
}

// NewInfrastructureAgent creates a new infrastructure agent
func NewInfrastructureAgent(llm llms.Model, logger *slog.Logger) *InfrastructureAgent {
	return &InfrastructureAgent{
		BaseAgent: NewBaseAgent(TypeInfrastructure, llm, logger),
	}
}

// executeStep overrides base implementation for infrastructure-specific logic
func (a *InfrastructureAgent) executeStep(ctx context.Context, request *Request, stepNum int, previousResult string) (string, bool, error) {
	// Build infrastructure-specific prompt
	prompt := a.buildInfrastructurePrompt(request, stepNum, previousResult)

	// Generate response
	response, err := llms.GenerateFromSinglePrompt(ctx, a.llm, prompt)
	if err != nil {
		return "", false, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Check for completion indicators
	done := false
	if stepNum >= request.MaxSteps-1 ||
		strings.Contains(strings.ToLower(response), "deployment complete") ||
		strings.Contains(strings.ToLower(response), "configuration complete") {
		done = true
	}

	return response, done, nil
}

// buildInfrastructurePrompt builds specialized prompts for infrastructure tasks
func (a *InfrastructureAgent) buildInfrastructurePrompt(request *Request, stepNum int, previousResult string) string {
	prompt := "You are an expert infrastructure agent specializing in Kubernetes, Docker, CI/CD, and cloud operations.\n\n"

	if stepNum == 0 {
		prompt += fmt.Sprintf("Task: %s\n\n", request.Query)

		// Add infrastructure-specific guidelines
		prompt += "Guidelines:\n"
		prompt += "- Follow infrastructure as code principles\n"
		prompt += "- Use declarative configurations\n"
		prompt += "- Consider security best practices\n"
		prompt += "- Implement proper monitoring and logging\n"
		prompt += "- Ensure scalability and high availability\n"
		prompt += "- Use version control for configurations\n\n"

		if len(request.Context) > 0 {
			prompt += "Context:\n"
			for k, v := range request.Context {
				// Handle environment information specially
				if k == "environment" || k == "cluster" || k == "services" {
					prompt += fmt.Sprintf("- %s:\n%v\n", k, v)
				} else {
					prompt += fmt.Sprintf("- %s: %v\n", k, v)
				}
			}
			prompt += "\n"
		}

		// Add available tools if specified
		if len(request.Tools) > 0 {
			prompt += "Available tools: " + strings.Join(request.Tools, ", ") + "\n"
			prompt += "(e.g., docker for container management, kubernetes for orchestration)\n\n"
		}

		prompt += "Provide configuration files, deployment strategies, or infrastructure solutions."
	} else {
		prompt += fmt.Sprintf("Continue working on: %s\n\n", request.Query)
		prompt += fmt.Sprintf("Previous work:\n%s\n\n", previousResult)
		prompt += "Continue the deployment/configuration or finalize with 'Deployment complete' or 'Configuration complete'."
	}

	return prompt
}

// Execute implements the Agent interface with infrastructure-specific execution
func (a *InfrastructureAgent) Execute(ctx context.Context, request *Request) (*Response, error) {
	// Log infrastructure-specific request details
	a.logger.Info("Infrastructure agent executing request",
		slog.String("query", request.Query),
		slog.Any("environment", request.Context["environment"]),
		slog.Int("available_tools", len(request.Tools)))

	// Delegate to base implementation which will call our overridden executeStep
	return a.BaseAgent.Execute(ctx, request)
}
