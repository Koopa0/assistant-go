package chains

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"

	"github.com/koopa0/assistant/internal/config"
)

// ConditionalChain executes different paths based on conditions and intermediate results
type ConditionalChain struct {
	*BaseChain
	conditionalNodes []ConditionalNode
	defaultPath      string
}

// ConditionalNode represents a decision point in the conditional chain
type ConditionalNode struct {
	Name          string                 `json:"name"`
	ConditionType string                 `json:"condition_type"` // "content", "length", "sentiment", "custom"
	Condition     string                 `json:"condition"`      // The actual condition to evaluate
	TruePath      []ConditionalStep      `json:"true_path"`      // Steps to execute if condition is true
	FalsePath     []ConditionalStep      `json:"false_path"`     // Steps to execute if condition is false
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
}

// ConditionalStep represents a step within a conditional path
type ConditionalStep struct {
	Name           string                 `json:"name"`
	PromptTemplate string                 `json:"prompt_template"`
	OutputKey      string                 `json:"output_key"`
	NextCondition  string                 `json:"next_condition,omitempty"` // Optional next condition to evaluate
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
}

// ConditionResult represents the result of evaluating a condition
type ConditionResult struct {
	NodeName    string `json:"node_name"`
	Condition   string `json:"condition"`
	Result      bool   `json:"result"`
	Explanation string `json:"explanation"`
	PathTaken   string `json:"path_taken"` // "true" or "false"
}

// NewConditionalChain creates a new conditional chain
func NewConditionalChain(llm llms.Model, config config.LangChain, logger *slog.Logger) *ConditionalChain {
	base := NewBaseChain(ChainTypeConditional, llm, config, logger)

	chain := &ConditionalChain{
		BaseChain:        base,
		conditionalNodes: make([]ConditionalNode, 0),
		defaultPath:      "general_processing",
	}

	// Add default conditional logic for intelligent routing
	chain.initializeDefaultNodes()

	return chain
}

// initializeDefaultNodes sets up default conditional processing nodes
func (cc *ConditionalChain) initializeDefaultNodes() {
	defaultNodes := []ConditionalNode{
		{
			Name:          "input_type_detection",
			ConditionType: "content",
			Condition:     "contains_code",
			TruePath: []ConditionalStep{
				{
					Name:           "code_analysis",
					PromptTemplate: "Analyze the following code:\n\nCode: {input}\n\nProvide:\n1. Programming language\n2. Code structure\n3. Potential issues\n4. Recommendations\n\nCode Analysis:",
					OutputKey:      "code_analysis_result",
				},
			},
			FalsePath: []ConditionalStep{
				{
					Name:           "text_analysis",
					PromptTemplate: "Analyze the following text:\n\nText: {input}\n\nProvide:\n1. Content type\n2. Main topics\n3. Key insights\n4. Summary\n\nText Analysis:",
					OutputKey:      "text_analysis_result",
				},
			},
		},
		{
			Name:          "complexity_assessment",
			ConditionType: "length",
			Condition:     "length_gt_1000",
			TruePath: []ConditionalStep{
				{
					Name:           "detailed_processing",
					PromptTemplate: "Perform detailed analysis of this complex input:\n\nInput: {input}\n\nProvide comprehensive analysis with:\n1. Detailed breakdown\n2. Multiple perspectives\n3. In-depth insights\n4. Actionable recommendations\n\nDetailed Analysis:",
					OutputKey:      "detailed_result",
				},
			},
			FalsePath: []ConditionalStep{
				{
					Name:           "simple_processing",
					PromptTemplate: "Provide a focused analysis of this input:\n\nInput: {input}\n\nProvide:\n1. Key points\n2. Main insights\n3. Brief recommendations\n\nFocused Analysis:",
					OutputKey:      "simple_result",
				},
			},
		},
		{
			Name:          "sentiment_routing",
			ConditionType: "sentiment",
			Condition:     "negative_sentiment",
			TruePath: []ConditionalStep{
				{
					Name:           "problem_solving",
					PromptTemplate: "Address the concerns in this input:\n\nInput: {input}\n\nProvide:\n1. Problem identification\n2. Root cause analysis\n3. Solution recommendations\n4. Prevention strategies\n\nProblem-Solving Response:",
					OutputKey:      "problem_solving_result",
				},
			},
			FalsePath: []ConditionalStep{
				{
					Name:           "enhancement_focus",
					PromptTemplate: "Build upon the positive aspects of this input:\n\nInput: {input}\n\nProvide:\n1. Strengths identification\n2. Enhancement opportunities\n3. Optimization suggestions\n4. Growth recommendations\n\nEnhancement Response:",
					OutputKey:      "enhancement_result",
				},
			},
		},
	}

	for _, node := range defaultNodes {
		cc.AddConditionalNode(node)
	}
}

// AddConditionalNode adds a conditional node to the chain
func (cc *ConditionalChain) AddConditionalNode(node ConditionalNode) {
	cc.conditionalNodes = append(cc.conditionalNodes, node)
	cc.logger.Debug("Added conditional node",
		slog.String("node_name", node.Name),
		slog.String("condition_type", node.ConditionType),
		slog.String("condition", node.Condition))
}

// executeSteps implements conditional chain execution logic
func (cc *ConditionalChain) executeSteps(ctx context.Context, request *ChainRequest) (string, []ChainStep, error) {
	steps := make([]ChainStep, 0)
	context := make(map[string]string)

	// Initialize context
	context["input"] = request.Input

	cc.logger.Debug("Starting conditional chain execution",
		slog.Int("node_count", len(cc.conditionalNodes)),
		slog.String("input", request.Input))

	// Process each conditional node
	for _, node := range cc.conditionalNodes {
		stepStart := time.Now()

		// Evaluate condition
		conditionResult, err := cc.evaluateCondition(ctx, node, context)
		if err != nil {
			cc.logger.Warn("Failed to evaluate condition",
				slog.String("node", node.Name),
				slog.Any("error", err))
			continue
		}

		// Record condition evaluation step
		conditionStep := ChainStep{
			StepNumber: len(steps) + 1,
			StepType:   fmt.Sprintf("condition_%s", node.Name),
			Input:      fmt.Sprintf("Condition: %s", node.Condition),
			Output:     fmt.Sprintf("Result: %v, Path: %s", conditionResult.Result, conditionResult.PathTaken),
			Duration:   time.Since(stepStart),
			Success:    true,
			Metadata: map[string]interface{}{
				"node_name":   node.Name,
				"condition":   node.Condition,
				"result":      conditionResult.Result,
				"path_taken":  conditionResult.PathTaken,
				"explanation": conditionResult.Explanation,
			},
		}
		steps = append(steps, conditionStep)

		// Execute appropriate path
		var pathSteps []ConditionalStep
		if conditionResult.Result {
			pathSteps = node.TruePath
		} else {
			pathSteps = node.FalsePath
		}

		// Execute steps in the chosen path
		for j, step := range pathSteps {
			stepStart = time.Now()

			// Build prompt from template
			prompt, err := cc.buildPromptFromTemplate(step.PromptTemplate, context)
			if err != nil {
				return "", steps, fmt.Errorf("failed to build prompt for step %s: %w", step.Name, err)
			}

			// Execute LLM call
			options := []llms.CallOption{
				llms.WithMaxTokens(2000),
			}

			if request.Temperature > 0 {
				options = append(options, llms.WithTemperature(request.Temperature))
			}

			output, err := cc.llm.Call(ctx, prompt, options...)
			if err != nil {
				executionStep := ChainStep{
					StepNumber: len(steps) + 1,
					StepType:   step.Name,
					Input:      prompt,
					Output:     "",
					Duration:   time.Since(stepStart),
					Success:    false,
					Error:      err.Error(),
					Metadata: map[string]interface{}{
						"node_name":  node.Name,
						"path_taken": conditionResult.PathTaken,
						"step_index": j,
					},
				}
				steps = append(steps, executionStep)
				return "", steps, fmt.Errorf("step %s failed: %w", step.Name, err)
			}

			// Store output in context
			context[step.OutputKey] = output

			// Record execution step
			executionStep := ChainStep{
				StepNumber: len(steps) + 1,
				StepType:   step.Name,
				Input:      prompt,
				Output:     output,
				Duration:   time.Since(stepStart),
				Success:    true,
				Metadata: map[string]interface{}{
					"node_name":  node.Name,
					"path_taken": conditionResult.PathTaken,
					"step_index": j,
					"output_key": step.OutputKey,
				},
			}
			steps = append(steps, executionStep)

			cc.logger.Debug("Conditional step completed",
				slog.String("node", node.Name),
				slog.String("step", step.Name),
				slog.String("path", conditionResult.PathTaken),
				slog.Duration("duration", executionStep.Duration))
		}
	}

	// Generate final output by synthesizing all results
	finalOutput, err := cc.synthesizeConditionalResults(ctx, request, context, steps)
	if err != nil {
		return "", steps, fmt.Errorf("result synthesis failed: %w", err)
	}

	cc.logger.Info("Conditional chain execution completed",
		slog.Int("steps_executed", len(steps)),
		slog.Int("nodes_processed", len(cc.conditionalNodes)))

	return finalOutput, steps, nil
}

// evaluateCondition evaluates a condition and returns the result
func (cc *ConditionalChain) evaluateCondition(ctx context.Context, node ConditionalNode, context map[string]string) (*ConditionResult, error) {
	input := context["input"]

	result := &ConditionResult{
		NodeName:  node.Name,
		Condition: node.Condition,
	}

	switch node.ConditionType {
	case "content":
		result.Result = cc.evaluateContentCondition(input, node.Condition)
		result.Explanation = fmt.Sprintf("Content condition '%s' evaluated to %v", node.Condition, result.Result)

	case "length":
		result.Result = cc.evaluateLengthCondition(input, node.Condition)
		result.Explanation = fmt.Sprintf("Length condition '%s' evaluated to %v (input length: %d)", node.Condition, result.Result, len(input))

	case "sentiment":
		var err error
		result.Result, err = cc.evaluateSentimentCondition(ctx, input, node.Condition)
		if err != nil {
			return nil, fmt.Errorf("sentiment evaluation failed: %w", err)
		}
		result.Explanation = fmt.Sprintf("Sentiment condition '%s' evaluated to %v", node.Condition, result.Result)

	case "custom":
		var err error
		result.Result, err = cc.evaluateCustomCondition(ctx, input, node.Condition, context)
		if err != nil {
			return nil, fmt.Errorf("custom condition evaluation failed: %w", err)
		}
		result.Explanation = fmt.Sprintf("Custom condition '%s' evaluated to %v", node.Condition, result.Result)

	default:
		return nil, fmt.Errorf("unknown condition type: %s", node.ConditionType)
	}

	if result.Result {
		result.PathTaken = "true"
	} else {
		result.PathTaken = "false"
	}

	return result, nil
}

// evaluateContentCondition evaluates content-based conditions
func (cc *ConditionalChain) evaluateContentCondition(input, condition string) bool {
	input = strings.ToLower(input)

	switch condition {
	case "contains_code":
		return strings.Contains(input, "function") || strings.Contains(input, "class") ||
			strings.Contains(input, "def ") || strings.Contains(input, "import") ||
			strings.Contains(input, "```")
	case "contains_question":
		return strings.Contains(input, "?") || strings.Contains(input, "what") ||
			strings.Contains(input, "how") || strings.Contains(input, "why")
	case "contains_data":
		return strings.Contains(input, "data") || strings.Contains(input, "table") ||
			strings.Contains(input, "csv") || strings.Contains(input, "json")
	default:
		return strings.Contains(input, condition)
	}
}

// evaluateLengthCondition evaluates length-based conditions
func (cc *ConditionalChain) evaluateLengthCondition(input, condition string) bool {
	length := len(input)

	switch condition {
	case "length_gt_1000":
		return length > 1000
	case "length_gt_500":
		return length > 500
	case "length_lt_100":
		return length < 100
	case "length_lt_500":
		return length < 500
	default:
		// Parse custom length conditions like "length_gt_X" or "length_lt_X"
		if strings.HasPrefix(condition, "length_gt_") {
			thresholdStr := strings.TrimPrefix(condition, "length_gt_")
			if threshold, err := strconv.Atoi(thresholdStr); err == nil {
				return length > threshold
			}
		} else if strings.HasPrefix(condition, "length_lt_") {
			thresholdStr := strings.TrimPrefix(condition, "length_lt_")
			if threshold, err := strconv.Atoi(thresholdStr); err == nil {
				return length < threshold
			}
		}
		return false
	}
}

// evaluateSentimentCondition evaluates sentiment-based conditions using LLM
func (cc *ConditionalChain) evaluateSentimentCondition(ctx context.Context, input, condition string) (bool, error) {
	prompt := fmt.Sprintf("Analyze the sentiment of the following text and respond with only 'POSITIVE', 'NEGATIVE', or 'NEUTRAL':\n\nText: %s\n\nSentiment:", input)

	sentiment, err := cc.llm.Call(ctx, prompt, llms.WithMaxTokens(10))
	if err != nil {
		return false, err
	}

	sentiment = strings.ToLower(strings.TrimSpace(sentiment))

	switch condition {
	case "positive_sentiment":
		return strings.Contains(sentiment, "positive"), nil
	case "negative_sentiment":
		return strings.Contains(sentiment, "negative"), nil
	case "neutral_sentiment":
		return strings.Contains(sentiment, "neutral"), nil
	default:
		return false, fmt.Errorf("unknown sentiment condition: %s", condition)
	}
}

// evaluateCustomCondition evaluates custom conditions using LLM
func (cc *ConditionalChain) evaluateCustomCondition(ctx context.Context, input, condition string, context map[string]string) (bool, error) {
	prompt := fmt.Sprintf("Evaluate whether the following condition is true for the given input. Respond with only 'TRUE' or 'FALSE':\n\nInput: %s\n\nCondition: %s\n\nResult:", input, condition)

	result, err := cc.llm.Call(ctx, prompt, llms.WithMaxTokens(10))
	if err != nil {
		return false, err
	}

	result = strings.ToLower(strings.TrimSpace(result))
	return strings.Contains(result, "true"), nil
}

// buildPromptFromTemplate builds a prompt by replacing placeholders
func (cc *ConditionalChain) buildPromptFromTemplate(template string, context map[string]string) (string, error) {
	prompt := template

	for key, value := range context {
		placeholder := fmt.Sprintf("{%s}", key)
		prompt = strings.ReplaceAll(prompt, placeholder, value)
	}

	return prompt, nil
}

// synthesizeConditionalResults synthesizes all conditional results into final output
func (cc *ConditionalChain) synthesizeConditionalResults(ctx context.Context, request *ChainRequest, context map[string]string, steps []ChainStep) (string, error) {
	// Build synthesis prompt
	prompt := fmt.Sprintf("Synthesize the following conditional analysis results into a comprehensive response:\n\nOriginal Input: %s\n\nConditional Analysis Results:\n", request.Input)

	for key, value := range context {
		if key != "input" && value != "" {
			prompt += fmt.Sprintf("\n%s:\n%s\n", key, value)
		}
	}

	prompt += "\nPlease provide:\n1. Comprehensive synthesis of all analyses\n2. Key insights from the conditional processing\n3. Final recommendations\n4. Summary of decision paths taken\n\nFinal Response:"

	// Generate synthesis
	options := []llms.CallOption{
		llms.WithMaxTokens(3000),
	}

	if request.Temperature > 0 {
		options = append(options, llms.WithTemperature(request.Temperature))
	}

	synthesis, err := cc.llm.Call(ctx, prompt, options...)
	if err != nil {
		return "", fmt.Errorf("synthesis generation failed: %w", err)
	}

	return synthesis, nil
}

// SetCustomNodes allows setting custom conditional nodes for specific use cases
func (cc *ConditionalChain) SetCustomNodes(nodes []ConditionalNode) {
	cc.conditionalNodes = nodes
	cc.logger.Info("Custom nodes set for conditional chain",
		slog.Int("node_count", len(nodes)))
}

// GetConditionalNodes returns the current conditional nodes
func (cc *ConditionalChain) GetConditionalNodes() []ConditionalNode {
	return cc.conditionalNodes
}
