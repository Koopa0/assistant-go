package chains

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/tmc/langchaingo/llms"

	"github.com/koopa0/assistant-go/internal/config"
)

// ParallelChain executes multiple independent operations concurrently
type ParallelChain struct {
	*BaseChain
	parallelTasks  []ParallelTask
	maxConcurrency int
}

// ParallelTask defines a task that can be executed in parallel
type ParallelTask struct {
	Name           string                 `json:"name"`
	PromptTemplate string                 `json:"prompt_template"`
	OutputKey      string                 `json:"output_key"`
	Priority       int                    `json:"priority"` // Higher number = higher priority
	Timeout        time.Duration          `json:"timeout"`
	Parameters     map[string]interface{} `json:"parameters,omitempty"`
	Dependencies   []string               `json:"dependencies,omitempty"` // Tasks that must complete first
}

// ParallelResult represents the result of a parallel task execution
type ParallelResult struct {
	TaskName  string        `json:"task_name"`
	Output    string        `json:"output"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
}

// NewParallelChain creates a new parallel chain
func NewParallelChain(llm llms.Model, config config.LangChain, logger *slog.Logger) *ParallelChain {
	base := NewBaseChain(ChainTypeParallel, llm, config, logger)

	chain := &ParallelChain{
		BaseChain:      base,
		parallelTasks:  make([]ParallelTask, 0),
		maxConcurrency: 5, // Default max concurrent tasks
	}

	// Add default parallel tasks for comprehensive analysis
	chain.initializeDefaultTasks()

	return chain
}

// initializeDefaultTasks sets up default parallel processing tasks
func (pc *ParallelChain) initializeDefaultTasks() {
	defaultTasks := []ParallelTask{
		{
			Name:           "content_analysis",
			PromptTemplate: "Analyze the content and structure of the following input:\n\nInput: {input}\n\nProvide:\n1. Content type and format\n2. Key themes and topics\n3. Structural analysis\n\nContent Analysis:",
			OutputKey:      "content_analysis",
			Priority:       3,
			Timeout:        30 * time.Second,
		},
		{
			Name:           "sentiment_analysis",
			PromptTemplate: "Analyze the sentiment and tone of the following input:\n\nInput: {input}\n\nProvide:\n1. Overall sentiment (positive/negative/neutral)\n2. Emotional tone\n3. Confidence level\n\nSentiment Analysis:",
			OutputKey:      "sentiment_analysis",
			Priority:       2,
			Timeout:        20 * time.Second,
		},
		{
			Name:           "keyword_extraction",
			PromptTemplate: "Extract key terms and concepts from the following input:\n\nInput: {input}\n\nProvide:\n1. Primary keywords\n2. Secondary concepts\n3. Technical terms\n\nKeyword Extraction:",
			OutputKey:      "keyword_extraction",
			Priority:       2,
			Timeout:        20 * time.Second,
		},
		{
			Name:           "summary_generation",
			PromptTemplate: "Generate a concise summary of the following input:\n\nInput: {input}\n\nProvide:\n1. Brief summary (2-3 sentences)\n2. Key points\n3. Main takeaways\n\nSummary:",
			OutputKey:      "summary",
			Priority:       3,
			Timeout:        25 * time.Second,
		},
		{
			Name:           "quality_assessment",
			PromptTemplate: "Assess the quality and completeness of the following input:\n\nInput: {input}\n\nProvide:\n1. Quality score (1-10)\n2. Completeness assessment\n3. Areas for improvement\n\nQuality Assessment:",
			OutputKey:      "quality_assessment",
			Priority:       1,
			Timeout:        20 * time.Second,
		},
	}

	for _, task := range defaultTasks {
		pc.AddParallelTask(task)
	}
}

// AddParallelTask adds a task to be executed in parallel
func (pc *ParallelChain) AddParallelTask(task ParallelTask) {
	pc.parallelTasks = append(pc.parallelTasks, task)
	pc.logger.Debug("Added parallel task",
		slog.String("task_name", task.Name),
		slog.Int("priority", task.Priority),
		slog.Duration("timeout", task.Timeout))
}

// SetMaxConcurrency sets the maximum number of concurrent tasks
func (pc *ParallelChain) SetMaxConcurrency(max int) {
	pc.maxConcurrency = max
	pc.logger.Debug("Set max concurrency", slog.Int("max_concurrency", max))
}

// executeSteps implements parallel chain execution logic
func (pc *ParallelChain) executeSteps(ctx context.Context, request *ChainRequest) (string, []ChainStep, error) {
	steps := make([]ChainStep, 0)

	pc.logger.Debug("Starting parallel chain execution",
		slog.Int("task_count", len(pc.parallelTasks)),
		slog.Int("max_concurrency", pc.maxConcurrency))

	// Execute tasks in parallel with concurrency control
	results, err := pc.executeTasksInParallel(ctx, request)
	if err != nil {
		return "", steps, fmt.Errorf("parallel execution failed: %w", err)
	}

	// Convert results to chain steps
	for i, result := range results {
		step := ChainStep{
			StepNumber: i + 1,
			StepType:   result.TaskName,
			Input:      fmt.Sprintf("Task: %s", result.TaskName),
			Output:     result.Output,
			Duration:   result.Duration,
			Success:    result.Success,
			Error:      result.Error,
			Metadata: map[string]interface{}{
				"task_name":  result.TaskName,
				"start_time": result.StartTime,
				"end_time":   result.EndTime,
			},
		}
		steps = append(steps, step)
	}

	// Synthesize results into final output
	finalOutput, err := pc.synthesizeResults(ctx, request, results)
	if err != nil {
		return "", steps, fmt.Errorf("result synthesis failed: %w", err)
	}

	pc.logger.Info("Parallel chain execution completed",
		slog.Int("tasks_executed", len(results)),
		slog.Int("successful_tasks", pc.countSuccessfulResults(results)))

	return finalOutput, steps, nil
}

// executeTasksInParallel executes all tasks in parallel with concurrency control
func (pc *ParallelChain) executeTasksInParallel(ctx context.Context, request *ChainRequest) ([]ParallelResult, error) {
	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, pc.maxConcurrency)

	// Create channels for results and errors
	resultsChan := make(chan ParallelResult, len(pc.parallelTasks))
	var wg sync.WaitGroup

	// Start all tasks
	for _, task := range pc.parallelTasks {
		wg.Add(1)
		go func(t ParallelTask) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Execute task
			result := pc.executeTask(ctx, request, t)
			resultsChan <- result
		}(task)
	}

	// Wait for all tasks to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	results := make([]ParallelResult, 0, len(pc.parallelTasks))
	for result := range resultsChan {
		results = append(results, result)
	}

	return results, nil
}

// executeTask executes a single parallel task
func (pc *ParallelChain) executeTask(ctx context.Context, request *ChainRequest, task ParallelTask) ParallelResult {
	startTime := time.Now()

	result := ParallelResult{
		TaskName:  task.Name,
		StartTime: startTime,
		Success:   false,
	}

	// Create task-specific context with timeout
	taskCtx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()

	// Build prompt from template
	prompt, err := pc.buildPromptFromTemplate(task.PromptTemplate, map[string]string{
		"input": request.Input,
	})
	if err != nil {
		result.Error = fmt.Sprintf("failed to build prompt: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime)
		return result
	}

	// Execute LLM call
	options := []llms.CallOption{
		llms.WithMaxTokens(1500),
	}

	if request.Temperature > 0 {
		options = append(options, llms.WithTemperature(request.Temperature))
	}

	output, err := pc.llm.Call(taskCtx, prompt, options...)
	if err != nil {
		result.Error = fmt.Sprintf("LLM call failed: %v", err)
		result.EndTime = time.Now()
		result.Duration = time.Since(startTime)
		return result
	}

	// Success
	result.Output = output
	result.Success = true
	result.EndTime = time.Now()
	result.Duration = time.Since(startTime)

	pc.logger.Debug("Parallel task completed",
		slog.String("task", task.Name),
		slog.Duration("duration", result.Duration),
		slog.Bool("success", result.Success))

	return result
}

// synthesizeResults combines all parallel results into a final output
func (pc *ParallelChain) synthesizeResults(ctx context.Context, request *ChainRequest, results []ParallelResult) (string, error) {
	// Build synthesis prompt
	synthesisPrompt := pc.buildSynthesisPrompt(request.Input, results)

	// Generate synthesis using LLM
	options := []llms.CallOption{
		llms.WithMaxTokens(3000),
	}

	if request.Temperature > 0 {
		options = append(options, llms.WithTemperature(request.Temperature))
	}

	synthesis, err := pc.llm.Call(ctx, synthesisPrompt, options...)
	if err != nil {
		return "", fmt.Errorf("synthesis generation failed: %w", err)
	}

	return synthesis, nil
}

// buildPromptFromTemplate builds a prompt by replacing placeholders
func (pc *ParallelChain) buildPromptFromTemplate(template string, context map[string]string) (string, error) {
	prompt := template

	for key, value := range context {
		placeholder := fmt.Sprintf("{%s}", key)
		prompt = strings.ReplaceAll(prompt, placeholder, value)
	}

	return prompt, nil
}

// buildSynthesisPrompt creates a prompt for synthesizing parallel results
func (pc *ParallelChain) buildSynthesisPrompt(originalInput string, results []ParallelResult) string {
	prompt := fmt.Sprintf("Synthesize the following parallel analysis results into a comprehensive response:\n\nOriginal Input: %s\n\nAnalysis Results:\n", originalInput)

	for _, result := range results {
		if result.Success {
			prompt += fmt.Sprintf("\n%s:\n%s\n", result.TaskName, result.Output)
		} else {
			prompt += fmt.Sprintf("\n%s: FAILED - %s\n", result.TaskName, result.Error)
		}
	}

	prompt += "\nPlease provide:\n1. Comprehensive synthesis of all successful analyses\n2. Key insights and patterns\n3. Overall assessment and conclusions\n4. Any limitations based on failed analyses\n\nSynthesis:"

	return prompt
}

// countSuccessfulResults counts the number of successful results
func (pc *ParallelChain) countSuccessfulResults(results []ParallelResult) int {
	count := 0
	for _, result := range results {
		if result.Success {
			count++
		}
	}
	return count
}

// SetCustomTasks allows setting custom parallel tasks for specific use cases
func (pc *ParallelChain) SetCustomTasks(tasks []ParallelTask) {
	pc.parallelTasks = tasks
	pc.logger.Info("Custom tasks set for parallel chain",
		slog.Int("task_count", len(tasks)))
}

// GetParallelTasks returns the current parallel tasks
func (pc *ParallelChain) GetParallelTasks() []ParallelTask {
	return pc.parallelTasks
}

// CreateAnalysisSpecificChain creates a parallel chain optimized for specific analysis types
func CreateAnalysisSpecificChain(analysisType string, llm llms.Model, config config.LangChain, logger *slog.Logger) *ParallelChain {
	chain := NewParallelChain(llm, config, logger)

	switch analysisType {
	case "document_analysis":
		chain.SetCustomTasks([]ParallelTask{
			{
				Name:           "structure_analysis",
				PromptTemplate: "Analyze the document structure:\n\nDocument: {input}\n\nProvide structure analysis:",
				OutputKey:      "structure",
				Priority:       3,
				Timeout:        30 * time.Second,
			},
			{
				Name:           "content_themes",
				PromptTemplate: "Identify main themes:\n\nDocument: {input}\n\nProvide theme analysis:",
				OutputKey:      "themes",
				Priority:       3,
				Timeout:        30 * time.Second,
			},
			{
				Name:           "readability_analysis",
				PromptTemplate: "Analyze readability:\n\nDocument: {input}\n\nProvide readability assessment:",
				OutputKey:      "readability",
				Priority:       2,
				Timeout:        25 * time.Second,
			},
		})
	case "code_review":
		chain.SetCustomTasks([]ParallelTask{
			{
				Name:           "syntax_analysis",
				PromptTemplate: "Analyze code syntax:\n\nCode: {input}\n\nProvide syntax analysis:",
				OutputKey:      "syntax",
				Priority:       3,
				Timeout:        30 * time.Second,
			},
			{
				Name:           "performance_analysis",
				PromptTemplate: "Analyze performance aspects:\n\nCode: {input}\n\nProvide performance analysis:",
				OutputKey:      "performance",
				Priority:       3,
				Timeout:        30 * time.Second,
			},
			{
				Name:           "security_analysis",
				PromptTemplate: "Analyze security aspects:\n\nCode: {input}\n\nProvide security analysis:",
				OutputKey:      "security",
				Priority:       3,
				Timeout:        30 * time.Second,
			},
		})
	}

	return chain
}
