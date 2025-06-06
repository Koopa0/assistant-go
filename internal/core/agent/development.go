package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// DevelopmentAgent specializes in software development tasks
type DevelopmentAgent struct {
	*BaseAgent
}

// NewDevelopmentAgent creates a new development-focused agent
func NewDevelopmentAgent(llm llms.Model, logger *slog.Logger) *DevelopmentAgent {
	base := NewBaseAgent("development_agent", TypeDevelopment, llm, logger)

	agent := &DevelopmentAgent{
		BaseAgent: base,
	}

	// Add development-specific capabilities
	agent.addCapabilities()

	return agent
}

// addCapabilities registers what this agent can do
func (d *DevelopmentAgent) addCapabilities() {
	capabilities := []Capability{
		{
			Name:        "code_review",
			Description: "Review code for quality, bugs, and improvements",
			Examples:    []string{"Review this Go function", "Check this code for bugs"},
		},
		{
			Name:        "code_generation",
			Description: "Generate code based on requirements",
			Examples:    []string{"Write a function to parse JSON", "Create a REST API endpoint"},
		},
		{
			Name:        "debugging",
			Description: "Help debug code issues",
			Examples:    []string{"Why is this code failing?", "Debug this error message"},
		},
		{
			Name:        "refactoring",
			Description: "Suggest code improvements and refactoring",
			Examples:    []string{"How can I improve this code?", "Refactor for better performance"},
		},
		{
			Name:        "testing",
			Description: "Write tests and test strategies",
			Examples:    []string{"Write unit tests for this function", "Create test cases"},
		},
		{
			Name:        "architecture",
			Description: "Design software architecture and patterns",
			Examples:    []string{"Design a microservice architecture", "What pattern should I use?"},
		},
	}

	for _, cap := range capabilities {
		d.AddCapability(cap)
	}
}

// Execute processes development-related requests
func (d *DevelopmentAgent) Execute(ctx context.Context, req Request) (*Response, error) {
	startTime := time.Now()

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Enhance request with development context
	enhancedReq := d.enhanceRequest(req)

	// Determine task type
	taskType := d.identifyTaskType(enhancedReq.Query)

	d.logger.Info("executing development task",
		slog.String("task_type", taskType),
		slog.String("query", req.Query))

	// Initialize response
	response := &Response{
		Success: true,
		Steps:   make([]Step, 0),
		Metadata: map[string]interface{}{
			"agent_type": string(d.Type()),
			"task_type":  taskType,
		},
	}

	// Execute based on task type
	var result string
	var err error

	stepStart := time.Now()

	switch taskType {
	case "code_review":
		result, err = d.executeCodeReview(ctx, enhancedReq)
	case "code_generation":
		result, err = d.executeCodeGeneration(ctx, enhancedReq)
	case "debugging":
		result, err = d.executeDebugging(ctx, enhancedReq)
	case "testing":
		result, err = d.executeTestGeneration(ctx, enhancedReq)
	default:
		// Fall back to general LLM execution
		result, err = d.BaseAgent.executeLLM(ctx, enhancedReq)
	}

	if err != nil {
		response.SetError(err)
		return response, err
	}

	// Record the step
	response.AddStep(taskType, req.Query, result, time.Since(stepStart))
	response.Result = result
	response.ExecutionTime = time.Since(startTime)

	// Set confidence based on task type
	response.Confidence = d.calculateConfidence(taskType, result)

	return response, nil
}

// enhanceRequest adds development-specific context
func (d *DevelopmentAgent) enhanceRequest(req Request) Request {
	enhanced := req

	// Add development expertise to context
	if enhanced.Context == nil {
		enhanced.Context = make(map[string]interface{})
	}

	enhanced.Context["expertise"] = "software development, code review, debugging, architecture"
	enhanced.Context["languages"] = "Go, Python, JavaScript, TypeScript, SQL"
	enhanced.Context["focus"] = "clean code, best practices, performance, security"

	return enhanced
}

// identifyTaskType determines what kind of development task this is
func (d *DevelopmentAgent) identifyTaskType(query string) string {
	q := strings.ToLower(query)

	switch {
	case strings.Contains(q, "review") || strings.Contains(q, "check"):
		return "code_review"
	case strings.Contains(q, "write") || strings.Contains(q, "generate") || strings.Contains(q, "create"):
		return "code_generation"
	case strings.Contains(q, "debug") || strings.Contains(q, "error") || strings.Contains(q, "fix"):
		return "debugging"
	case strings.Contains(q, "test") || strings.Contains(q, "testing"):
		return "testing"
	case strings.Contains(q, "refactor") || strings.Contains(q, "improve"):
		return "refactoring"
	case strings.Contains(q, "architect") || strings.Contains(q, "design"):
		return "architecture"
	default:
		return "general_development"
	}
}

// executeCodeReview performs code review
func (d *DevelopmentAgent) executeCodeReview(ctx context.Context, req Request) (string, error) {
	prompt := fmt.Sprintf(`As an expert code reviewer, analyze the following:

Query: %s

Please provide:
1. Code quality assessment
2. Potential bugs or issues
3. Performance considerations
4. Security concerns
5. Suggested improvements

Be specific and actionable in your feedback.`, req.Query)

	result, err := d.llm.Call(ctx, prompt,
		llms.WithTemperature(0.3), // Lower temperature for more focused analysis
		llms.WithMaxTokens(2000))

	if err != nil {
		return "", fmt.Errorf("code review failed: %w", err)
	}

	return result, nil
}

// executeCodeGeneration generates code
func (d *DevelopmentAgent) executeCodeGeneration(ctx context.Context, req Request) (string, error) {
	prompt := fmt.Sprintf(`As an expert software developer, generate code for:

Query: %s

Requirements:
- Write clean, well-documented code
- Follow best practices
- Include error handling
- Make it production-ready

Provide the code with explanations.`, req.Query)

	result, err := d.llm.Call(ctx, prompt,
		llms.WithTemperature(0.7), // Balanced temperature for creativity
		llms.WithMaxTokens(3000))

	if err != nil {
		return "", fmt.Errorf("code generation failed: %w", err)
	}

	return result, nil
}

// executeDebugging helps with debugging
func (d *DevelopmentAgent) executeDebugging(ctx context.Context, req Request) (string, error) {
	prompt := fmt.Sprintf(`As an expert debugger, help with:

Query: %s

Provide:
1. Likely cause of the issue
2. Step-by-step debugging approach
3. Potential solutions
4. Prevention strategies

Be thorough and explain your reasoning.`, req.Query)

	result, err := d.llm.Call(ctx, prompt,
		llms.WithTemperature(0.4), // Lower temperature for focused problem-solving
		llms.WithMaxTokens(2000))

	if err != nil {
		return "", fmt.Errorf("debugging assistance failed: %w", err)
	}

	return result, nil
}

// executeTestGeneration generates tests
func (d *DevelopmentAgent) executeTestGeneration(ctx context.Context, req Request) (string, error) {
	prompt := fmt.Sprintf(`As a testing expert, create tests for:

Query: %s

Include:
1. Unit tests
2. Edge cases
3. Error scenarios
4. Test data
5. Assertions

Use appropriate testing frameworks and best practices.`, req.Query)

	result, err := d.llm.Call(ctx, prompt,
		llms.WithTemperature(0.5),
		llms.WithMaxTokens(2500))

	if err != nil {
		return "", fmt.Errorf("test generation failed: %w", err)
	}

	return result, nil
}

// calculateConfidence estimates confidence in the response
func (d *DevelopmentAgent) calculateConfidence(taskType string, result string) float64 {
	// Simple heuristic - can be enhanced with more sophisticated logic
	baseConfidence := 0.7

	// Adjust based on task type
	switch taskType {
	case "code_review", "debugging":
		baseConfidence = 0.8 // Higher confidence in analysis tasks
	case "code_generation":
		baseConfidence = 0.75 // Moderate confidence in generation
	case "architecture":
		baseConfidence = 0.65 // Lower confidence in high-level design
	}

	// Adjust based on result characteristics
	if len(result) > 1000 {
		baseConfidence += 0.05 // More detailed response
	}

	if strings.Contains(result, "```") {
		baseConfidence += 0.05 // Contains code blocks
	}

	// Cap at 0.95
	if baseConfidence > 0.95 {
		baseConfidence = 0.95
	}

	return baseConfidence
}
