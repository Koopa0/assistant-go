package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// DatabaseAgent specializes in database operations and SQL queries
type DatabaseAgent struct {
	*BaseAgent
}

// NewDatabaseAgent creates a new database agent
func NewDatabaseAgent(llm llms.Model, logger *slog.Logger) *DatabaseAgent {
	return &DatabaseAgent{
		BaseAgent: NewBaseAgent(TypeDatabase, llm, logger),
	}
}

// executeStep overrides base implementation for database-specific logic
func (a *DatabaseAgent) executeStep(ctx context.Context, request *Request, stepNum int, previousResult string) (string, bool, error) {
	// Build database-specific prompt
	prompt := a.buildDatabasePrompt(request, stepNum, previousResult)

	// Generate response
	response, err := llms.GenerateFromSinglePrompt(ctx, a.llm, prompt)
	if err != nil {
		return "", false, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Check for completion indicators
	done := false
	if stepNum >= request.MaxSteps-1 ||
		strings.Contains(strings.ToLower(response), "query complete") ||
		strings.Contains(strings.ToLower(response), "migration complete") {
		done = true
	}

	return response, done, nil
}

// buildDatabasePrompt builds specialized prompts for database tasks
func (a *DatabaseAgent) buildDatabasePrompt(request *Request, stepNum int, previousResult string) string {
	prompt := "You are an expert database agent specializing in PostgreSQL, query optimization, and database design.\n\n"

	if stepNum == 0 {
		prompt += fmt.Sprintf("Task: %s\n\n", request.Query)

		// Add database-specific guidelines
		prompt += "Guidelines:\n"
		prompt += "- Use PostgreSQL 17+ best practices\n"
		prompt += "- Optimize queries for performance\n"
		prompt += "- Consider indexes and query planning\n"
		prompt += "- Use proper constraints and data types\n"
		prompt += "- Follow normalization principles where appropriate\n"
		prompt += "- Include comments for complex queries\n\n"

		if len(request.Context) > 0 {
			prompt += "Context:\n"
			for k, v := range request.Context {
				// Handle schema information specially
				if k == "schema" || k == "tables" {
					prompt += fmt.Sprintf("- %s:\n%v\n", k, v)
				} else {
					prompt += fmt.Sprintf("- %s: %v\n", k, v)
				}
			}
			prompt += "\n"
		}

		prompt += "Provide SQL queries, schema designs, or optimization suggestions as appropriate."
	} else {
		prompt += fmt.Sprintf("Continue working on: %s\n\n", request.Query)
		prompt += fmt.Sprintf("Previous work:\n%s\n\n", previousResult)
		prompt += "Continue the query/migration or finalize with 'Query complete' or 'Migration complete'."
	}

	return prompt
}

// Execute implements the Agent interface with database-specific execution
func (a *DatabaseAgent) Execute(ctx context.Context, request *Request) (*Response, error) {
	// Log database-specific request details
	a.logger.Info("Database agent executing request",
		slog.String("query", request.Query),
		slog.Any("has_schema_context", request.Context["schema"] != nil))

	// Delegate to base implementation which will call our overridden executeStep
	return a.BaseAgent.Execute(ctx, request)
}
