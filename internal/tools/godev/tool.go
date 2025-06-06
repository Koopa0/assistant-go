package godev

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"

	"github.com/koopa0/assistant-go/internal/ai/prompts"
	"github.com/koopa0/assistant-go/internal/tools"
)

// GoDevTool implements the Tool interface for Go development functionality
type GoDevTool struct {
	detector *WorkspaceDetector
	logger   *slog.Logger
}

// NewGoDevTool creates a new Go development tool
func NewGoDevTool(logger *slog.Logger) *GoDevTool {
	return &GoDevTool{
		detector: NewWorkspaceDetector(logger),
		logger:   logger,
	}
}

// Name returns the tool name
func (t *GoDevTool) Name() string {
	return "godev"
}

// Description returns the tool description
func (t *GoDevTool) Description() string {
	return "Go development workspace analyzer - Detects Go projects, analyzes code structure, dependencies, and provides intelligent suggestions for Go developers"
}

// Parameters returns the tool parameters schema
func (t *GoDevTool) Parameters() *tools.ToolParametersSchema {
	return &tools.ToolParametersSchema{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"action": {
				Type:        tools.ParameterTypeString,
				Description: "Action to perform: 'analyze', 'detect', 'coverage', 'dependencies', 'metrics'",
				Enum:        []string{"analyze", "detect", "coverage", "dependencies", "metrics"},
				Required:    true,
			},
			"path": {
				Type:        tools.ParameterTypeString,
				Description: "Path to Go project directory (defaults to current directory)",
				Required:    false,
			},
			"include_tests": {
				Type:        tools.ParameterTypeBoolean,
				Description: "Include test files in analysis (default: true)",
				Required:    false,
			},
			"include_dependencies": {
				Type:        tools.ParameterTypeBoolean,
				Description: "Include dependency analysis (default: true)",
				Required:    false,
			},
			"include_git_info": {
				Type:        tools.ParameterTypeBoolean,
				Description: "Include Git repository information (default: true)",
				Required:    false,
			},
			"include_coverage": {
				Type:        tools.ParameterTypeBoolean,
				Description: "Run test coverage analysis (default: false, can be slow)",
				Required:    false,
			},
			"include_build_info": {
				Type:        tools.ParameterTypeBoolean,
				Description: "Include build information (default: false, can be slow)",
				Required:    false,
			},
			"max_depth": {
				Type:        tools.ParameterTypeInteger,
				Description: "Maximum directory depth to analyze (default: 10)",
				Required:    false,
			},
		},
		Required: []string{"action"},
	}
}

// GoDevInput represents input for the Go development tool
type GoDevInput struct {
	Action              string `json:"action"`
	Path                string `json:"path,omitempty"`
	IncludeTests        *bool  `json:"include_tests,omitempty"`
	IncludeDependencies *bool  `json:"include_dependencies,omitempty"`
	IncludeGitInfo      *bool  `json:"include_git_info,omitempty"`
	IncludeCoverage     *bool  `json:"include_coverage,omitempty"`
	IncludeBuildInfo    *bool  `json:"include_build_info,omitempty"`
	MaxDepth            *int   `json:"max_depth,omitempty"`
}

// Execute executes the Go development tool with the given input
func (t *GoDevTool) Execute(ctx context.Context, input *tools.ToolInput) (*tools.ToolResult, error) {
	// Marshal the parameters map to JSON first
	paramsJSON, err := json.Marshal(input.Parameters)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to marshal parameters: %v", err),
		}, nil
	}

	// Parse the input parameters
	var goInput GoDevInput
	if err := json.Unmarshal(paramsJSON, &goInput); err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Invalid input parameters: %v", err),
		}, nil
	}
	t.logger.Info("Executing Go development tool",
		"action", goInput.Action,
		"path", goInput.Path)

	// Set default path if not provided
	if goInput.Path == "" {
		goInput.Path = "."
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(goInput.Path)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Invalid path: %v", err),
		}, nil
	}

	// Create analysis options
	options := t.createAnalysisOptions(&goInput)

	// Execute based on action
	switch goInput.Action {
	case "analyze":
		return t.executeAnalyze(ctx, absPath, options)
	case "detect":
		return t.executeDetect(ctx, absPath, options)
	case "coverage":
		return t.executeCoverage(ctx, absPath, options)
	case "dependencies":
		return t.executeDependencies(ctx, absPath, options)
	case "metrics":
		return t.executeMetrics(ctx, absPath, options)
	default:
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Unknown action: %s", goInput.Action),
		}, nil
	}
}

// createAnalysisOptions creates analysis options from tool input
func (t *GoDevTool) createAnalysisOptions(input *GoDevInput) *AnalysisOptions {
	options := DefaultAnalysisOptions()

	if input.IncludeTests != nil {
		options.IncludeTestFiles = *input.IncludeTests
	}
	if input.IncludeDependencies != nil {
		options.IncludeDependencies = *input.IncludeDependencies
	}
	if input.IncludeGitInfo != nil {
		options.IncludeGitInfo = *input.IncludeGitInfo
	}
	if input.IncludeCoverage != nil {
		options.IncludeCoverage = *input.IncludeCoverage
	}
	if input.IncludeBuildInfo != nil {
		options.IncludeBuildInfo = *input.IncludeBuildInfo
	}
	if input.MaxDepth != nil {
		options.MaxDepth = *input.MaxDepth
	}

	return options
}

// executeAnalyze performs full workspace analysis
func (t *GoDevTool) executeAnalyze(ctx context.Context, path string, options *AnalysisOptions) (*tools.ToolResult, error) {
	result, err := t.detector.DetectWorkspace(ctx, path, options)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Analysis failed: %v", err),
		}, nil
	}

	// Convert result to JSON for generic storage
	resultJSON, _ := json.Marshal(result)

	return &tools.ToolResult{
		Success: true,
		Data: &tools.ToolResultData{
			Output:  fmt.Sprintf("Successfully analyzed Go workspace at %s", path),
			Content: string(resultJSON),
		},
	}, nil
}

// executeDetect performs basic workspace detection
func (t *GoDevTool) executeDetect(ctx context.Context, path string, options *AnalysisOptions) (*tools.ToolResult, error) {
	// Simplified options for detection only
	detectOptions := &AnalysisOptions{
		IncludeTestFiles:    false,
		IncludeDependencies: true,
		IncludeGitInfo:      true,
		IncludeCoverage:     false,
		IncludeBuildInfo:    false,
		MaxDepth:            5,
		ExcludeVendor:       true,
	}

	result, err := t.detector.DetectWorkspace(ctx, path, detectOptions)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Detection failed: %v", err),
		}, nil
	}

	// Return only workspace info for detection
	detectionResult := map[string]interface{}{
		"workspace":    result.Workspace,
		"project_type": result.Workspace.ProjectType,
		"module_path":  result.Workspace.ModulePath,
		"go_version":   result.Workspace.GoVersion,
		"packages":     len(result.Workspace.Packages),
		"dependencies": len(result.Workspace.Dependencies),
	}

	// Convert result to JSON for generic storage
	resultJSON, _ := json.Marshal(detectionResult)

	return &tools.ToolResult{
		Success: true,
		Data: &tools.ToolResultData{
			Output:  fmt.Sprintf("Detected Go %s project: %s", result.Workspace.ProjectType, result.Workspace.ModulePath),
			Content: string(resultJSON),
		},
	}, nil
}

// executeCoverage performs test coverage analysis
func (t *GoDevTool) executeCoverage(ctx context.Context, path string, options *AnalysisOptions) (*tools.ToolResult, error) {
	// Force enable coverage analysis
	coverageOptions := *options
	coverageOptions.IncludeCoverage = true

	result, err := t.detector.DetectWorkspace(ctx, path, &coverageOptions)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Coverage analysis failed: %v", err),
		}, nil
	}

	if result.Workspace.TestCoverage == nil {
		return &tools.ToolResult{
			Success: false,
			Error:   "No test coverage data available. Make sure you have tests in your project.",
		}, nil
	}

	coverageResult := map[string]interface{}{
		"coverage":         result.Workspace.TestCoverage,
		"total_lines":      result.Workspace.TestCoverage.TotalLines,
		"covered_lines":    result.Workspace.TestCoverage.CoveredLines,
		"percentage":       result.Workspace.TestCoverage.Percentage,
		"package_coverage": result.Workspace.TestCoverage.PackageCoverage,
	}

	// Convert result to JSON for generic storage
	resultJSON, _ := json.Marshal(coverageResult)

	return &tools.ToolResult{
		Success: true,
		Data: &tools.ToolResultData{
			Output: fmt.Sprintf("Test coverage: %.1f%% (%d/%d lines)",
				result.Workspace.TestCoverage.Percentage,
				result.Workspace.TestCoverage.CoveredLines,
				result.Workspace.TestCoverage.TotalLines),
			Content: string(resultJSON),
		},
	}, nil
}

// executeDependencies analyzes project dependencies
func (t *GoDevTool) executeDependencies(ctx context.Context, path string, options *AnalysisOptions) (*tools.ToolResult, error) {
	result, err := t.detector.DetectWorkspace(ctx, path, options)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Dependency analysis failed: %v", err),
		}, nil
	}

	// Group dependencies by type
	var direct, indirect []DependencyInfo
	for _, dep := range result.Workspace.Dependencies {
		if dep.IsIndirect {
			indirect = append(indirect, dep)
		} else {
			direct = append(direct, dep)
		}
	}

	dependencyResult := map[string]interface{}{
		"total_dependencies":    len(result.Workspace.Dependencies),
		"direct_dependencies":   len(direct),
		"indirect_dependencies": len(indirect),
		"dependencies":          result.Workspace.Dependencies,
		"direct":                direct,
		"indirect":              indirect,
	}

	// Convert result to JSON for generic storage
	resultJSON, _ := json.Marshal(dependencyResult)

	return &tools.ToolResult{
		Success: true,
		Data: &tools.ToolResultData{
			Output: fmt.Sprintf("Found %d dependencies (%d direct, %d indirect)",
				len(result.Workspace.Dependencies), len(direct), len(indirect)),
			Content: string(resultJSON),
		},
	}, nil
}

// executeMetrics calculates code metrics
func (t *GoDevTool) executeMetrics(ctx context.Context, path string, options *AnalysisOptions) (*tools.ToolResult, error) {
	result, err := t.detector.DetectWorkspace(ctx, path, options)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Metrics calculation failed: %v", err),
		}, nil
	}

	metricsResult := map[string]interface{}{
		"metrics":     result.Metrics,
		"issues":      result.Issues,
		"suggestions": result.Suggestions,
		"summary": map[string]interface{}{
			"packages":       result.Metrics.TotalPackages,
			"files":          result.Metrics.TotalFiles,
			"lines":          result.Metrics.TotalLines,
			"functions":      result.Metrics.TotalFunctions,
			"avg_complexity": result.Metrics.AvgComplexity,
			"max_complexity": result.Metrics.MaxComplexity,
			"issues_count":   len(result.Issues),
		},
	}

	// Convert result to JSON for generic storage
	resultJSON, _ := json.Marshal(metricsResult)

	return &tools.ToolResult{
		Success: true,
		Data: &tools.ToolResultData{
			Output: fmt.Sprintf("Code metrics: %d packages, %d files, %d lines, %d functions",
				result.Metrics.TotalPackages,
				result.Metrics.TotalFiles,
				result.Metrics.TotalLines,
				result.Metrics.TotalFunctions),
			Content: string(resultJSON),
		},
	}, nil
}

// Health checks if the tool is healthy and ready to use
func (t *GoDevTool) Health(ctx context.Context) error {
	// Check if Go is installed and accessible
	cmd := exec.CommandContext(ctx, "go", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Go is not accessible: %w", err)
	}
	return nil
}

// Close performs cleanup when the tool is no longer needed
func (t *GoDevTool) Close(ctx context.Context) error {
	// No cleanup needed for this tool
	return nil
}

// ExecuteWithJSON is a helper method that accepts JSON input
func (t *GoDevTool) ExecuteWithJSON(ctx context.Context, inputJSON string) (*tools.ToolResult, error) {
	var input GoDevInput
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Invalid JSON input: %v", err),
		}, nil
	}

	// Convert to map[string]interface{} for ToolInput
	var paramsMap map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &paramsMap); err != nil {
		return &tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to convert to parameters map: %v", err),
		}, nil
	}

	toolInput := &tools.ToolInput{
		Parameters: paramsMap,
	}

	return t.Execute(ctx, toolInput)
}

// GetSampleUsage returns sample usage examples for the tool
func (t *GoDevTool) GetSampleUsage() []string {
	return []string{
		`{"action": "detect", "path": "."}`,
		`{"action": "analyze", "path": "./my-project", "include_tests": true}`,
		`{"action": "coverage", "path": ".", "include_coverage": true}`,
		`{"action": "dependencies", "path": "."}`,
		`{"action": "metrics", "path": ".", "include_tests": true}`,
	}
}

// CreatePromptContext creates a prompt context from workspace analysis results
func (t *GoDevTool) CreatePromptContext(result *AnalysisResult, filePath string) *prompts.PromptContext {
	if result == nil || result.Workspace == nil {
		return &prompts.PromptContext{
			ProjectPath: ".",
			ProjectType: "unknown",
			GoVersion:   "1.24",
		}
	}

	workspace := result.Workspace

	// Extract issues as strings
	var issues []string
	for _, issue := range result.Issues {
		issues = append(issues, fmt.Sprintf("%s: %s", issue.Type, issue.Message))
	}

	// Extract dependencies as strings
	var dependencies []string
	for _, dep := range workspace.Dependencies {
		dependencies = append(dependencies, dep.ModulePath)
	}

	// Create metrics map
	metrics := make(map[string]interface{})
	if result.Metrics != nil {
		metrics["total_packages"] = result.Metrics.TotalPackages
		metrics["total_files"] = result.Metrics.TotalFiles
		metrics["total_lines"] = result.Metrics.TotalLines
		metrics["total_functions"] = result.Metrics.TotalFunctions
		metrics["avg_complexity"] = result.Metrics.AvgComplexity
		metrics["max_complexity"] = result.Metrics.MaxComplexity
		metrics["test_coverage"] = result.Metrics.TestCoverage
	}

	return &prompts.PromptContext{
		ProjectPath:  workspace.RootPath,
		ModulePath:   workspace.ModulePath,
		ProjectType:  string(workspace.ProjectType),
		GoVersion:    workspace.GoVersion,
		FileName:     filePath,
		Issues:       issues,
		Metrics:      metrics,
		Dependencies: dependencies,
	}
}

// AnalyzeCodeWithPrompt analyzes code and returns prompt-enhanced analysis
func (t *GoDevTool) AnalyzeCodeWithPrompt(ctx context.Context, codePath string, codeSnippet string) (*CodeAnalysisResult, error) {
	// First perform workspace analysis
	options := DefaultAnalysisOptions()
	result, err := t.detector.DetectWorkspace(ctx, codePath, options)
	if err != nil {
		return nil, fmt.Errorf("workspace analysis failed: %w", err)
	}

	// Create prompt context
	promptCtx := t.CreatePromptContext(result, codePath)
	promptCtx.CodeSnippet = codeSnippet

	return &CodeAnalysisResult{
		WorkspaceAnalysis: result,
		PromptContext:     promptCtx,
		EnhancedPrompt:    prompts.CodeAnalysisPrompt(promptCtx),
	}, nil
}

// CodeAnalysisResult represents the result of code analysis with prompt context
type CodeAnalysisResult struct {
	WorkspaceAnalysis *AnalysisResult        `json:"workspace_analysis"`
	PromptContext     *prompts.PromptContext `json:"prompt_context"`
	EnhancedPrompt    string                 `json:"enhanced_prompt"`
}
