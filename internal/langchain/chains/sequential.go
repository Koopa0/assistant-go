package chains

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"

	"github.com/koopa0/assistant/internal/config"
)

// SequentialChain executes steps in sequence, passing output from one step to the next
type SequentialChain struct {
	*BaseChain
	stepTemplates []SequentialStepTemplate
}

// SequentialStepTemplate defines a template for sequential chain steps
type SequentialStepTemplate struct {
	Name           string                 `json:"name"`
	PromptTemplate string                 `json:"prompt_template"`
	OutputKey      string                 `json:"output_key"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
	Validation     string                 `json:"validation,omitempty"`
}

// NewSequentialChain creates a new sequential chain
func NewSequentialChain(llm llms.Model, config config.LangChain, logger *slog.Logger) *SequentialChain {
	base := NewBaseChain(ChainTypeSequential, llm, config, logger)

	chain := &SequentialChain{
		BaseChain:     base,
		stepTemplates: make([]SequentialStepTemplate, 0),
	}

	// Add default sequential steps for complex task processing
	chain.initializeDefaultSteps()

	return chain
}

// initializeDefaultSteps sets up default sequential processing steps
func (sc *SequentialChain) initializeDefaultSteps() {
	defaultSteps := []SequentialStepTemplate{
		{
			Name:           "task_analysis",
			PromptTemplate: "Analyze the following task and break it down into key components:\n\nTask: {input}\n\nProvide:\n1. Task type and complexity\n2. Required steps\n3. Expected output format\n\nAnalysis:",
			OutputKey:      "analysis",
		},
		{
			Name:           "step_planning",
			PromptTemplate: "Based on the task analysis, create a detailed execution plan:\n\nTask Analysis: {analysis}\n\nProvide:\n1. Step-by-step execution plan\n2. Dependencies between steps\n3. Success criteria\n\nExecution Plan:",
			OutputKey:      "plan",
		},
		{
			Name:           "execution",
			PromptTemplate: "Execute the planned steps for the task:\n\nOriginal Task: {input}\nExecution Plan: {plan}\n\nProvide:\n1. Detailed execution of each step\n2. Results and findings\n3. Quality verification\n\nExecution Results:",
			OutputKey:      "results",
		},
		{
			Name:           "synthesis",
			PromptTemplate: "Synthesize the execution results into a final response:\n\nOriginal Task: {input}\nExecution Results: {results}\n\nProvide:\n1. Comprehensive final answer\n2. Summary of key findings\n3. Confidence level and limitations\n\nFinal Response:",
			OutputKey:      "final_output",
		},
	}

	for _, step := range defaultSteps {
		sc.AddStepTemplate(step)
	}
}

// AddStepTemplate adds a step template to the sequential chain
func (sc *SequentialChain) AddStepTemplate(template SequentialStepTemplate) {
	sc.stepTemplates = append(sc.stepTemplates, template)
	sc.logger.Debug("Added step template to sequential chain",
		slog.String("step_name", template.Name),
		slog.String("output_key", template.OutputKey))
}

// executeSteps implements sequential chain execution logic
func (sc *SequentialChain) executeSteps(ctx context.Context, request *ChainRequest) (string, []ChainStep, error) {
	steps := make([]ChainStep, 0)
	context := make(map[string]string)

	// Initialize context with input
	context["input"] = request.Input

	// Add any additional context from request
	if request.Context != nil {
		for key, value := range request.Context {
			if strValue, ok := value.(string); ok {
				context[key] = strValue
			}
		}
	}

	sc.logger.Debug("Starting sequential chain execution",
		slog.Int("step_count", len(sc.stepTemplates)),
		slog.String("input", request.Input))

	// Execute each step in sequence
	for i, stepTemplate := range sc.stepTemplates {
		stepStart := time.Now()

		// Build prompt from template
		prompt, err := sc.buildPromptFromTemplate(stepTemplate.PromptTemplate, context)
		if err != nil {
			return "", steps, fmt.Errorf("failed to build prompt for step %s: %w", stepTemplate.Name, err)
		}

		// Execute LLM call
		options := []llms.CallOption{
			llms.WithMaxTokens(2000),
		}

		if request.Temperature > 0 {
			options = append(options, llms.WithTemperature(request.Temperature))
		}

		output, err := sc.llm.Call(ctx, prompt, options...)
		if err != nil {
			step := ChainStep{
				StepNumber: i + 1,
				StepType:   stepTemplate.Name,
				Input:      prompt,
				Output:     "",
				Duration:   time.Since(stepStart),
				Success:    false,
				Error:      err.Error(),
				Metadata:   map[string]interface{}{"template": stepTemplate.Name},
			}
			steps = append(steps, step)
			return "", steps, fmt.Errorf("step %s failed: %w", stepTemplate.Name, err)
		}

		// Validate output if validation is specified
		if stepTemplate.Validation != "" {
			if err := sc.validateStepOutput(output, stepTemplate.Validation); err != nil {
				sc.logger.Warn("Step output validation failed",
					slog.String("step", stepTemplate.Name),
					slog.String("validation", stepTemplate.Validation),
					slog.Any("error", err))
			}
		}

		// Store output in context for next steps
		context[stepTemplate.OutputKey] = output

		// Create step record
		step := ChainStep{
			StepNumber: i + 1,
			StepType:   stepTemplate.Name,
			Input:      prompt,
			Output:     output,
			Duration:   time.Since(stepStart),
			Success:    true,
			Metadata: map[string]interface{}{
				"template":      stepTemplate.Name,
				"output_key":    stepTemplate.OutputKey,
				"output_length": len(output),
			},
		}
		steps = append(steps, step)

		sc.logger.Debug("Sequential step completed",
			slog.String("step", stepTemplate.Name),
			slog.Int("step_number", i+1),
			slog.Duration("duration", step.Duration),
			slog.Int("output_length", len(output)))
	}

	// Return the final output (from the last step)
	finalOutput := context["final_output"]
	if finalOutput == "" && len(steps) > 0 {
		// Fallback to last step output if no final_output key
		finalOutput = steps[len(steps)-1].Output
	}

	sc.logger.Info("Sequential chain execution completed",
		slog.Int("steps_executed", len(steps)),
		slog.Int("final_output_length", len(finalOutput)))

	return finalOutput, steps, nil
}

// buildPromptFromTemplate builds a prompt by replacing placeholders in the template
func (sc *SequentialChain) buildPromptFromTemplate(template string, context map[string]string) (string, error) {
	prompt := template

	// Replace all placeholders in the format {key} with values from context
	for key, value := range context {
		placeholder := fmt.Sprintf("{%s}", key)
		prompt = strings.ReplaceAll(prompt, placeholder, value)
	}

	// Check for any remaining unreplaced placeholders
	if strings.Contains(prompt, "{") && strings.Contains(prompt, "}") {
		sc.logger.Warn("Prompt contains unreplaced placeholders",
			slog.String("prompt", prompt))
	}

	return prompt, nil
}

// validateStepOutput validates the output of a step based on validation criteria
func (sc *SequentialChain) validateStepOutput(output, validation string) error {
	// Simple validation based on criteria
	switch validation {
	case "not_empty":
		if strings.TrimSpace(output) == "" {
			return fmt.Errorf("output is empty")
		}
	case "min_length_50":
		if len(output) < 50 {
			return fmt.Errorf("output too short: %d characters", len(output))
		}
	case "contains_analysis":
		if !strings.Contains(strings.ToLower(output), "analysis") {
			return fmt.Errorf("output does not contain analysis")
		}
	case "contains_plan":
		if !strings.Contains(strings.ToLower(output), "plan") {
			return fmt.Errorf("output does not contain plan")
		}
	default:
		// No validation or unknown validation type
		return nil
	}

	return nil
}

// SetCustomSteps allows setting custom step templates for specific use cases
func (sc *SequentialChain) SetCustomSteps(steps []SequentialStepTemplate) {
	sc.stepTemplates = steps
	sc.logger.Info("Custom steps set for sequential chain",
		slog.Int("step_count", len(steps)))
}

// AddCustomStep adds a single custom step to the chain
func (sc *SequentialChain) AddCustomStep(step SequentialStepTemplate) {
	sc.stepTemplates = append(sc.stepTemplates, step)
	sc.logger.Debug("Custom step added to sequential chain",
		slog.String("step_name", step.Name))
}

// GetStepTemplates returns the current step templates
func (sc *SequentialChain) GetStepTemplates() []SequentialStepTemplate {
	return sc.stepTemplates
}

// CreateTaskSpecificChain creates a sequential chain optimized for a specific task type
func CreateTaskSpecificChain(taskType string, llm llms.Model, config config.LangChain, logger *slog.Logger) *SequentialChain {
	chain := NewSequentialChain(llm, config, logger)

	switch strings.ToLower(taskType) {
	case "code_analysis":
		chain.SetCustomSteps([]SequentialStepTemplate{
			{
				Name:           "code_understanding",
				PromptTemplate: "Analyze and understand the following code:\n\nCode: {input}\n\nProvide:\n1. Code structure overview\n2. Key functions and components\n3. Dependencies and imports\n\nCode Understanding:",
				OutputKey:      "understanding",
			},
			{
				Name:           "issue_identification",
				PromptTemplate: "Based on the code understanding, identify potential issues:\n\nCode Understanding: {understanding}\n\nProvide:\n1. Potential bugs or errors\n2. Performance issues\n3. Security concerns\n4. Code quality issues\n\nIssue Analysis:",
				OutputKey:      "issues",
			},
			{
				Name:           "recommendations",
				PromptTemplate: "Provide recommendations based on the identified issues:\n\nCode Understanding: {understanding}\nIdentified Issues: {issues}\n\nProvide:\n1. Specific recommendations for each issue\n2. Priority levels\n3. Implementation suggestions\n\nRecommendations:",
				OutputKey:      "final_output",
			},
		})
	case "research_synthesis":
		chain.SetCustomSteps([]SequentialStepTemplate{
			{
				Name:           "topic_breakdown",
				PromptTemplate: "Break down the research topic into key areas:\n\nTopic: {input}\n\nProvide:\n1. Main research areas\n2. Key questions to investigate\n3. Information sources needed\n\nTopic Breakdown:",
				OutputKey:      "breakdown",
			},
			{
				Name:           "information_gathering",
				PromptTemplate: "Based on the topic breakdown, gather relevant information:\n\nTopic Breakdown: {breakdown}\n\nProvide:\n1. Key facts and findings\n2. Different perspectives\n3. Supporting evidence\n\nGathered Information:",
				OutputKey:      "information",
			},
			{
				Name:           "synthesis",
				PromptTemplate: "Synthesize the gathered information into a comprehensive response:\n\nOriginal Topic: {input}\nGathered Information: {information}\n\nProvide:\n1. Comprehensive synthesis\n2. Key insights\n3. Conclusions and implications\n\nResearch Synthesis:",
				OutputKey:      "final_output",
			},
		})
	case "problem_solving":
		chain.SetCustomSteps([]SequentialStepTemplate{
			{
				Name:           "problem_definition",
				PromptTemplate: "Define and clarify the problem:\n\nProblem: {input}\n\nProvide:\n1. Clear problem statement\n2. Constraints and requirements\n3. Success criteria\n\nProblem Definition:",
				OutputKey:      "definition",
			},
			{
				Name:           "solution_generation",
				PromptTemplate: "Generate potential solutions:\n\nProblem Definition: {definition}\n\nProvide:\n1. Multiple solution approaches\n2. Pros and cons of each\n3. Feasibility assessment\n\nSolution Options:",
				OutputKey:      "solutions",
			},
			{
				Name:           "solution_selection",
				PromptTemplate: "Select and detail the best solution:\n\nProblem Definition: {definition}\nSolution Options: {solutions}\n\nProvide:\n1. Recommended solution with rationale\n2. Implementation steps\n3. Risk mitigation strategies\n\nRecommended Solution:",
				OutputKey:      "final_output",
			},
		})
	}

	return chain
}
