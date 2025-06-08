package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// ResearchAgent specializes in research, documentation, and information synthesis
type ResearchAgent struct {
	*BaseAgent
}

// NewResearchAgent creates a new research agent
func NewResearchAgent(llm llms.Model, logger *slog.Logger) *ResearchAgent {
	return &ResearchAgent{
		BaseAgent: NewBaseAgent(TypeResearch, llm, logger),
	}
}

// executeStep overrides base implementation for research-specific logic
func (a *ResearchAgent) executeStep(ctx context.Context, request *Request, stepNum int, previousResult string) (string, bool, error) {
	// Build research-specific prompt
	prompt := a.buildResearchPrompt(request, stepNum, previousResult)

	// Generate response
	response, err := llms.GenerateFromSinglePrompt(ctx, a.llm, prompt)
	if err != nil {
		return "", false, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Check for completion indicators
	done := false
	if stepNum >= request.MaxSteps-1 ||
		strings.Contains(strings.ToLower(response), "research complete") ||
		strings.Contains(strings.ToLower(response), "analysis complete") {
		done = true
	}

	return response, done, nil
}

// buildResearchPrompt builds specialized prompts for research tasks
func (a *ResearchAgent) buildResearchPrompt(request *Request, stepNum int, previousResult string) string {
	prompt := "You are an expert research agent specializing in information synthesis, documentation, and technical analysis.\n\n"

	if stepNum == 0 {
		prompt += fmt.Sprintf("Research Topic: %s\n\n", request.Query)

		// Add research-specific guidelines
		prompt += "Guidelines:\n"
		prompt += "- Provide comprehensive and accurate information\n"
		prompt += "- Cite sources and provide references where possible\n"
		prompt += "- Structure information clearly and logically\n"
		prompt += "- Include practical examples and use cases\n"
		prompt += "- Highlight key insights and recommendations\n"
		prompt += "- Consider multiple perspectives and trade-offs\n\n"

		if len(request.Context) > 0 {
			prompt += "Research Context:\n"
			for k, v := range request.Context {
				// Handle research-specific context
				if k == "domain" || k == "focus_areas" || k == "existing_knowledge" {
					prompt += fmt.Sprintf("- %s:\n%v\n", k, v)
				} else {
					prompt += fmt.Sprintf("- %s: %v\n", k, v)
				}
			}
			prompt += "\n"
		}

		prompt += "Provide a thorough research summary with actionable insights."
	} else {
		prompt += fmt.Sprintf("Continue researching: %s\n\n", request.Query)
		prompt += fmt.Sprintf("Previous findings:\n%s\n\n", previousResult)
		prompt += "Expand on the research, add more details, or finalize with 'Research complete' or 'Analysis complete'."
	}

	return prompt
}

// Execute implements the Agent interface with research-specific execution
func (a *ResearchAgent) Execute(ctx context.Context, request *Request) (*Response, error) {
	// Log research-specific request details
	a.logger.Info("Research agent executing request",
		slog.String("topic", request.Query),
		slog.Any("domain", request.Context["domain"]),
		slog.Any("focus_areas", request.Context["focus_areas"]))

	// Delegate to base implementation which will call our overridden executeStep
	return a.BaseAgent.Execute(ctx, request)
}
