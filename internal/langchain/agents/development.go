package agents

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"

	"github.com/koopa0/assistant-go/internal/config"
)

// DevelopmentAgent specializes in development assistance tasks
type DevelopmentAgent struct {
	*BaseAgent
	astAnalyzer   *ASTAnalyzer
	codeGenerator *CodeGenerator
	perfAnalyzer  *PerformanceAnalyzer
}

// NewDevelopmentAgent creates a new development assistant agent
func NewDevelopmentAgent(llm llms.Model, config config.LangChain, logger *slog.Logger) *DevelopmentAgent {
	base := NewBaseAgent(AgentTypeDevelopment, llm, config, logger)

	agent := &DevelopmentAgent{
		BaseAgent:     base,
		astAnalyzer:   NewASTAnalyzer(logger),
		codeGenerator: NewCodeGenerator(llm, logger),
		perfAnalyzer:  NewPerformanceAnalyzer(logger),
	}

	// Add development-specific capabilities
	agent.initializeCapabilities()

	return agent
}

// initializeCapabilities sets up the development agent's capabilities
func (d *DevelopmentAgent) initializeCapabilities() {
	capabilities := []AgentCapability{
		{
			Name:        "code_analysis",
			Description: "Analyze Go code structure, dependencies, and patterns",
			Parameters: map[string]interface{}{
				"file_path":     "string",
				"analysis_type": "string (structure|dependencies|patterns|all)",
			},
		},
		{
			Name:        "code_generation",
			Description: "Generate Go code based on specifications",
			Parameters: map[string]interface{}{
				"specification": "string",
				"code_type":     "string (function|struct|interface|package)",
				"style_guide":   "string",
			},
		},
		{
			Name:        "performance_analysis",
			Description: "Analyze code performance and suggest optimizations",
			Parameters: map[string]interface{}{
				"code":           "string",
				"analysis_depth": "string (basic|detailed|comprehensive)",
			},
		},
		{
			Name:        "refactoring_suggestions",
			Description: "Suggest code refactoring improvements",
			Parameters: map[string]interface{}{
				"code":             "string",
				"refactoring_type": "string (structure|performance|readability)",
			},
		},
		{
			Name:        "test_generation",
			Description: "Generate unit tests for Go code",
			Parameters: map[string]interface{}{
				"code":            "string",
				"test_type":       "string (unit|integration|benchmark)",
				"coverage_target": "number",
			},
		},
	}

	for _, capability := range capabilities {
		d.AddCapability(capability)
	}
}

// executeSteps implements specialized development agent logic
func (d *DevelopmentAgent) executeSteps(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Analyze the request to determine the development task
	taskType, err := d.analyzeRequestType(request.Query)
	if err != nil {
		return "", nil, fmt.Errorf("failed to analyze request type: %w", err)
	}

	d.logger.Debug("Development task identified",
		slog.String("task_type", taskType),
		slog.String("query", request.Query))

	// Execute based on task type
	switch taskType {
	case "code_analysis":
		return d.executeCodeAnalysis(ctx, request, maxSteps)
	case "code_generation":
		return d.executeCodeGeneration(ctx, request, maxSteps)
	case "performance_analysis":
		return d.executePerformanceAnalysis(ctx, request, maxSteps)
	case "refactoring":
		return d.executeRefactoring(ctx, request, maxSteps)
	case "test_generation":
		return d.executeTestGeneration(ctx, request, maxSteps)
	default:
		return d.executeGeneralDevelopment(ctx, request, maxSteps)
	}
}

// analyzeRequestType determines what type of development task is being requested
func (d *DevelopmentAgent) analyzeRequestType(query string) (string, error) {
	query = strings.ToLower(query)

	// Simple keyword-based analysis (could be enhanced with ML)
	if strings.Contains(query, "analyze") || strings.Contains(query, "analysis") {
		if strings.Contains(query, "performance") || strings.Contains(query, "optimize") {
			return "performance_analysis", nil
		}
		return "code_analysis", nil
	}

	if strings.Contains(query, "generate") || strings.Contains(query, "create") || strings.Contains(query, "write") {
		if strings.Contains(query, "test") {
			return "test_generation", nil
		}
		return "code_generation", nil
	}

	if strings.Contains(query, "refactor") || strings.Contains(query, "improve") || strings.Contains(query, "restructure") {
		return "refactoring", nil
	}

	if strings.Contains(query, "test") {
		return "test_generation", nil
	}

	return "general", nil
}

// executeCodeAnalysis performs code analysis tasks
func (d *DevelopmentAgent) executeCodeAnalysis(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	steps := make([]AgentStep, 0)

	// Step 1: Extract code from request or context
	stepStart := time.Now()
	code, err := d.extractCodeFromRequest(request)
	if err != nil {
		return "", nil, fmt.Errorf("failed to extract code: %w", err)
	}

	step1 := AgentStep{
		StepNumber: 1,
		Action:     "extract_code",
		Input:      "Extract code from request",
		Output:     fmt.Sprintf("Extracted %d characters of code", len(code)),
		Reasoning:  "Need to extract code before analysis",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"code_length": len(code)},
	}
	steps = append(steps, step1)

	// Step 2: Perform AST analysis
	stepStart = time.Now()
	analysis, err := d.astAnalyzer.AnalyzeCode(code)
	if err != nil {
		return "", nil, fmt.Errorf("AST analysis failed: %w", err)
	}

	step2 := AgentStep{
		StepNumber: 2,
		Action:     "ast_analysis",
		Tool:       "ast_analyzer",
		Input:      code,
		Output:     analysis.Summary,
		Reasoning:  "Perform structural analysis of the code",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"functions_found": analysis.FunctionCount},
	}
	steps = append(steps, step2)

	// Step 3: Generate analysis report using LLM
	stepStart = time.Now()
	prompt := d.buildAnalysisPrompt(code, analysis)
	report, err := d.llm.Call(ctx, prompt, llms.WithMaxTokens(2000))
	if err != nil {
		return "", nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	step3 := AgentStep{
		StepNumber: 3,
		Action:     "generate_report",
		Tool:       "llm",
		Input:      prompt,
		Output:     report,
		Reasoning:  "Generate comprehensive analysis report",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"report_length": len(report)},
	}
	steps = append(steps, step3)

	return report, steps, nil
}

// executeCodeGeneration performs code generation tasks
func (d *DevelopmentAgent) executeCodeGeneration(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	steps := make([]AgentStep, 0)

	// Step 1: Analyze requirements
	stepStart := time.Now()
	requirements := d.extractRequirements(request.Query)

	step1 := AgentStep{
		StepNumber: 1,
		Action:     "analyze_requirements",
		Input:      request.Query,
		Output:     fmt.Sprintf("Identified %d requirements", len(requirements)),
		Reasoning:  "Extract and analyze code generation requirements",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"requirements_count": len(requirements)},
	}
	steps = append(steps, step1)

	// Step 2: Generate code using LLM
	stepStart = time.Now()
	requirementsStr := strings.Join(requirements, ", ")
	prompt := d.buildGenerationPrompt(request.Query, requirementsStr)
	code, err := d.llm.Call(ctx, prompt, llms.WithMaxTokens(3000))
	if err != nil {
		return "", nil, fmt.Errorf("code generation failed: %w", err)
	}

	step2 := AgentStep{
		StepNumber: 2,
		Action:     "generate_code",
		Tool:       "llm",
		Input:      prompt,
		Output:     code,
		Reasoning:  "Generate code based on requirements",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"code_length": len(code)},
	}
	steps = append(steps, step2)

	// Step 3: Validate generated code
	stepStart = time.Now()
	validation := d.validateGeneratedCode(code)

	step3 := AgentStep{
		StepNumber: 3,
		Action:     "validate_code",
		Tool:       "code_validator",
		Input:      code,
		Output:     validation.Summary,
		Reasoning:  "Validate syntax and structure of generated code",
		Duration:   time.Since(stepStart),
		Metadata:   map[string]interface{}{"is_valid": validation.IsValid},
	}
	steps = append(steps, step3)

	return code, steps, nil
}

// executePerformanceAnalysis performs performance analysis tasks
func (d *DevelopmentAgent) executePerformanceAnalysis(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Implementation similar to code analysis but focused on performance
	return d.executeCodeAnalysis(ctx, request, maxSteps) // Simplified for now
}

// executeRefactoring performs refactoring tasks
func (d *DevelopmentAgent) executeRefactoring(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Implementation for refactoring suggestions
	return d.executeCodeGeneration(ctx, request, maxSteps) // Simplified for now
}

// executeTestGeneration performs test generation tasks
func (d *DevelopmentAgent) executeTestGeneration(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Implementation for test generation
	return d.executeCodeGeneration(ctx, request, maxSteps) // Simplified for now
}

// executeGeneralDevelopment handles general development queries
func (d *DevelopmentAgent) executeGeneralDevelopment(ctx context.Context, request *AgentRequest, maxSteps int) (string, []AgentStep, error) {
	// Fallback to base agent implementation
	return d.BaseAgent.executeSteps(ctx, request, maxSteps)
}

// Helper methods

func (d *DevelopmentAgent) extractCodeFromRequest(request *AgentRequest) (string, error) {
	// Try to extract code from context
	if code, exists := request.Context["code"]; exists {
		if codeStr, ok := code.(string); ok {
			return codeStr, nil
		}
	}

	// Try to extract code from query (look for code blocks)
	if strings.Contains(request.Query, "```") {
		parts := strings.Split(request.Query, "```")
		if len(parts) >= 3 {
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("no code found in request")
}

func (d *DevelopmentAgent) extractRequirements(query string) []string {
	// Simple requirement extraction (could be enhanced)
	requirements := make([]string, 0)

	// Look for common requirement patterns
	if strings.Contains(query, "function") {
		requirements = append(requirements, "function_implementation")
	}
	if strings.Contains(query, "struct") {
		requirements = append(requirements, "struct_definition")
	}
	if strings.Contains(query, "interface") {
		requirements = append(requirements, "interface_definition")
	}

	return requirements
}

func (d *DevelopmentAgent) buildAnalysisPrompt(code string, analysis *ASTAnalysis) string {
	return fmt.Sprintf(`Analyze the following Go code and provide a comprehensive report:

Code:
%s

AST Analysis Summary:
%s

Please provide:
1. Code structure analysis
2. Potential issues or improvements
3. Best practices recommendations
4. Performance considerations

Analysis:`, code, analysis.Summary)
}

func (d *DevelopmentAgent) buildGenerationPrompt(query, requirements string) string {
	return fmt.Sprintf(`Generate Go code based on the following requirements:

Query: %s
Requirements: %s

Please provide:
1. Clean, well-documented Go code
2. Follow Go best practices
3. Include error handling
4. Add appropriate comments

Code:`, query, requirements)
}

func (d *DevelopmentAgent) validateGeneratedCode(code string) *CodeValidation {
	// Simple validation using Go parser
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "", code, parser.ParseComments)

	return &CodeValidation{
		IsValid: err == nil,
		Summary: fmt.Sprintf("Code validation: %v", err == nil),
		Errors:  []string{},
	}
}

// Supporting types

type ASTAnalysis struct {
	Summary        string
	FunctionCount  int
	StructCount    int
	InterfaceCount int
}

type CodeValidation struct {
	IsValid bool
	Summary string
	Errors  []string
}

// ASTAnalyzer analyzes Go code structure
type ASTAnalyzer struct {
	logger *slog.Logger
}

func NewASTAnalyzer(logger *slog.Logger) *ASTAnalyzer {
	return &ASTAnalyzer{logger: logger}
}

func (a *ASTAnalyzer) AnalyzeCode(code string) (*ASTAnalysis, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", code, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse code: %w", err)
	}

	analysis := &ASTAnalysis{}

	// Count functions, structs, interfaces
	ast.Inspect(node, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.FuncDecl:
			analysis.FunctionCount++
		case *ast.StructType:
			analysis.StructCount++
		case *ast.InterfaceType:
			analysis.InterfaceCount++
		}
		return true
	})

	analysis.Summary = fmt.Sprintf("Found %d functions, %d structs, %d interfaces",
		analysis.FunctionCount, analysis.StructCount, analysis.InterfaceCount)

	return analysis, nil
}

// CodeGenerator generates code using LLM
type CodeGenerator struct {
	llm    llms.Model
	logger *slog.Logger
}

func NewCodeGenerator(llm llms.Model, logger *slog.Logger) *CodeGenerator {
	return &CodeGenerator{llm: llm, logger: logger}
}

// PerformanceAnalyzer analyzes code performance
type PerformanceAnalyzer struct {
	logger *slog.Logger
}

func NewPerformanceAnalyzer(logger *slog.Logger) *PerformanceAnalyzer {
	return &PerformanceAnalyzer{logger: logger}
}
