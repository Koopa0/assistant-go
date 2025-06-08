package prompt

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
)

// PromptService manages AI prompt generation and context enhancement
type PromptService struct {
	logger    *slog.Logger
	templates map[string]PromptTemplate
}

// NewPromptService creates a new prompt service
func NewPromptService(logger *slog.Logger) *PromptService {
	service := &PromptService{
		logger:    logger,
		templates: make(map[string]PromptTemplate),
	}

	// Load available templates
	for _, template := range AvailableTemplates() {
		service.templates[template.Name] = template
	}

	return service
}

// EnhanceQuery enhances a user query with appropriate context and prompts
func (s *PromptService) EnhanceQuery(ctx context.Context, userQuery string, promptCtx *PromptContext) (*EnhancedQuery, error) {
	s.logger.Info("Enhancing user query with prompt context",
		"query_length", len(userQuery),
		"module", promptCtx.ModulePath,
		"project_type", promptCtx.ProjectType)

	// Detect the type of task based on user query
	taskType := s.detectTaskType(userQuery)
	promptCtx.TaskType = taskType
	promptCtx.UserQuery = userQuery

	// Generate appropriate system prompt
	systemPrompt := s.generateSystemPrompt(taskType, promptCtx)

	// Enhance the user query with context
	enhancedQuery := s.enhanceUserQuery(userQuery, promptCtx)

	enhanced := &EnhancedQuery{
		OriginalQuery:  userQuery,
		EnhancedQuery:  enhancedQuery,
		SystemPrompt:   systemPrompt,
		TaskType:       taskType,
		Context:        promptCtx,
		PromptTemplate: taskType,
		Confidence:     s.calculateConfidence(userQuery, taskType),
	}

	s.logger.Debug("Query enhancement completed",
		"task_type", taskType,
		"confidence", enhanced.Confidence,
		"enhanced_length", len(enhancedQuery))

	return enhanced, nil
}

// EnhancedQuery represents a user query enhanced with AI prompts and context
type EnhancedQuery struct {
	OriginalQuery  string         `json:"original_query"`
	EnhancedQuery  string         `json:"enhanced_query"`
	SystemPrompt   string         `json:"system_prompt"`
	TaskType       string         `json:"task_type"`
	Context        *PromptContext `json:"context"`
	PromptTemplate string         `json:"prompt_template"`
	Confidence     float64        `json:"confidence"`
}

// detectTaskType analyzes the user query to determine the most appropriate task type
func (s *PromptService) detectTaskType(query string) string {
	query = strings.ToLower(query)

	// Define keywords for different task types
	patterns := map[string][]string{
		"code_analysis": {
			"analyze", "review", "check", "examine", "audit", "inspect",
			"issues", "problems", "bugs", "quality", "best practices",
			"idioms", "conventions", "standards",
		},
		"refactoring": {
			"refactor", "improve", "optimize", "clean", "simplify",
			"restructure", "reorganize", "enhance", "better",
			"readable", "maintainable", "cleaner",
		},
		"performance": {
			"performance", "speed", "faster", "optimize", "bottleneck",
			"memory", "cpu", "profile", "slow",
			"efficiency", "scalability", "throughput", "better performance",
		},
		"architecture": {
			"architecture", "design", "structure", "pattern", "organize",
			"packages", "modules", "dependencies", "interfaces",
			"separation", "coupling", "cohesion", "layers", "structure the",
		},
		"test_generation": {
			"test", "testing", "coverage", "unit test", "create benchmark",
			"example", "spec", "verify", "validate", "assert",
			"mock", "stub", "fixture", "generate test", "write test",
		},
		"error_diagnosis": {
			"error", "bug", "issue", "problem", "fix", "debug",
			"crash", "panic", "exception", "failure", "broken",
			"not working", "wrong", "incorrect",
		},
		"workspace_analysis": {
			"project", "workspace", "codebase", "overview", "summary",
			"health", "metrics", "statistics", "report", "assessment",
			"complete", "entire", "whole",
		},
	}

	// Score each task type based on keyword matches
	scores := make(map[string]int)
	for taskType, keywords := range patterns {
		for _, keyword := range keywords {
			if strings.Contains(query, keyword) {
				scores[taskType]++
			}
		}
	}

	// Find the task type with the highest score
	maxScore := 0
	bestMatch := "code_analysis" // default
	for taskType, score := range scores {
		if score > maxScore {
			maxScore = score
			bestMatch = taskType
		}
	}

	s.logger.Debug("Task type detection completed",
		"query", query,
		"detected_type", bestMatch,
		"score", maxScore,
		"all_scores", scores)

	return bestMatch
}

// generateSystemPrompt creates the appropriate system prompt for the task
func (s *PromptService) generateSystemPrompt(taskType string, ctx *PromptContext) string {
	switch taskType {
	case "code_analysis":
		return CodeAnalysisPrompt(ctx)
	case "refactoring":
		return RefactoringPrompt(ctx)
	case "performance":
		return PerformanceAnalysisPrompt(ctx)
	case "architecture":
		return ArchitectureReviewPrompt(ctx)
	case "test_generation":
		return TestGenerationPrompt(ctx)
	case "error_diagnosis":
		return ErrorDiagnosisPrompt(ctx)
	case "workspace_analysis":
		return WorkspaceAnalysisPrompt(ctx)
	default:
		return SystemPrompt()
	}
}

// enhanceUserQuery adds context and clarification to the user's query
func (s *PromptService) enhanceUserQuery(query string, ctx *PromptContext) string {
	var enhancements []string

	// Add project context if available
	if ctx.ModulePath != "" {
		enhancements = append(enhancements,
			fmt.Sprintf("For the Go project '%s'", ctx.ModulePath))
	}

	if ctx.ProjectType != "" {
		enhancements = append(enhancements,
			fmt.Sprintf("(project type: %s)", ctx.ProjectType))
	}

	// Add Go version context
	if ctx.GoVersion != "" {
		enhancements = append(enhancements,
			fmt.Sprintf("using Go %s", ctx.GoVersion))
	}

	// Add file context if specific file is being analyzed
	if ctx.FileName != "" {
		enhancements = append(enhancements,
			fmt.Sprintf("in file '%s'", ctx.FileName))
	}

	// Build the enhanced query
	enhanced := query
	if len(enhancements) > 0 {
		enhanced = fmt.Sprintf("%s %s", query, strings.Join(enhancements, " "))
	}

	// Add specific instructions based on task type
	switch ctx.TaskType {
	case "code_analysis":
		enhanced += ". Please focus on Go idioms, best practices, potential issues, and improvement suggestions."
	case "refactoring":
		enhanced += ". Please provide refactored code that follows Go best practices while maintaining the same functionality."
	case "performance":
		enhanced += ". Please identify performance bottlenecks and provide optimization suggestions with benchmarks where appropriate."
	case "architecture":
		enhanced += ". Please evaluate the overall architecture and suggest improvements following Go design principles."
	case "test_generation":
		enhanced += ". Please generate comprehensive tests including table-driven tests, error cases, and benchmarks."
	case "error_diagnosis":
		enhanced += ". Please explain the root cause of the error and provide a complete solution with explanation."
	case "workspace_analysis":
		enhanced += ". Please provide a comprehensive analysis of the entire project with actionable recommendations."
	}

	return enhanced
}

// calculateConfidence estimates confidence in the task type detection
func (s *PromptService) calculateConfidence(query string, taskType string) float64 {
	query = strings.ToLower(query)

	// Get keywords for the detected task type
	patterns := map[string][]string{
		"code_analysis":      {"analyze", "review", "check", "examine", "issues", "problems"},
		"refactoring":        {"refactor", "improve", "optimize", "clean", "simplify", "better"},
		"performance":        {"performance", "speed", "faster", "bottleneck", "memory", "cpu"},
		"architecture":       {"architecture", "design", "structure", "pattern", "organize"},
		"test_generation":    {"test", "testing", "coverage", "unit test", "benchmark"},
		"error_diagnosis":    {"error", "bug", "issue", "problem", "fix", "debug"},
		"workspace_analysis": {"project", "workspace", "codebase", "overview", "summary"},
	}

	keywords, exists := patterns[taskType]
	if !exists {
		return 0.5 // Default confidence for unknown task types
	}

	matches := 0
	for _, keyword := range keywords {
		if strings.Contains(query, keyword) {
			matches++
		}
	}

	// Calculate confidence based on keyword matches
	confidence := float64(matches) / float64(len(keywords))

	// Boost confidence if query contains very specific terms
	specificTerms := map[string]float64{
		"analyze code":          0.9,
		"review code":           0.9,
		"refactor this":         0.9,
		"optimize performance":  0.9,
		"better performance":    0.9,
		"generate tests":        0.9,
		"create benchmarks":     0.9,
		"fix error":             0.9,
		"project overview":      0.9,
		"structure the project": 0.9,
	}

	for term, boost := range specificTerms {
		if strings.Contains(query, term) {
			confidence = math.Max(confidence, boost)
		}
	}

	// Ensure confidence is between 0.1 and 1.0
	if confidence < 0.1 {
		confidence = 0.1
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// GetTemplate returns a specific prompt template
func (s *PromptService) GetTemplate(name string) (PromptTemplate, error) {
	template, exists := s.templates[name]
	if !exists {
		return PromptTemplate{}, fmt.Errorf("template '%s' not found", name)
	}
	return template, nil
}

// ListTemplates returns all available templates
func (s *PromptService) ListTemplates() []PromptTemplate {
	var templates []PromptTemplate
	for _, template := range s.templates {
		templates = append(templates, template)
	}
	return templates
}

// CreateContextFromWorkspace creates a prompt context from workspace information
func (s *PromptService) CreateContextFromWorkspace(workspace interface{}) *PromptContext {
	// This would be implemented to extract context from workspace analysis results
	// For now, return a basic context
	return &PromptContext{
		ProjectPath: ".",
		ProjectType: "unknown",
		GoVersion:   "1.24",
	}
}

// ValidateTemplate validates a prompt template
func (s *PromptService) ValidateTemplate(template PromptTemplate) error {
	if template.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if template.Template == "" {
		return fmt.Errorf("template content is required")
	}
	if template.Category == "" {
		return fmt.Errorf("template category is required")
	}
	return nil
}
