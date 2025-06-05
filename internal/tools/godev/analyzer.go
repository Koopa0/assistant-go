package godev

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/tools"
)

// GoAnalyzer is a tool for analyzing Go code
type GoAnalyzer struct {
	logger *slog.Logger
	config map[string]interface{}
}

// NewGoAnalyzer creates a new Go analyzer tool
func NewGoAnalyzer(config *tools.ToolConfig, logger *slog.Logger) (tools.Tool, error) {
	// Convert ToolConfig to legacy map format for now
	legacyConfig := make(map[string]interface{})
	if config != nil {
		legacyConfig["timeout"] = config.Timeout
		legacyConfig["debug"] = config.Debug
		legacyConfig["working_dir"] = config.WorkingDir
	}

	return &GoAnalyzer{
		logger: logger,
		config: legacyConfig,
	}, nil
}

// Name returns the tool name
func (g *GoAnalyzer) Name() string {
	return "go_analyzer"
}

// Description returns the tool description
func (g *GoAnalyzer) Description() string {
	return "Analyzes Go source code for structure, complexity, and potential issues"
}

// Parameters returns the tool parameters schema
func (g *GoAnalyzer) Parameters() *tools.ToolParametersSchema {
	return &tools.ToolParametersSchema{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"path": {
				Type:        "string",
				Description: "Path to Go file or directory to analyze",
				Required:    true,
			},
			"analysis_type": {
				Type:        "string",
				Description: "Type of analysis to perform",
				Enum:        []string{"structure", "complexity", "issues", "all"},
				Default:     "all",
			},
			"recursive": {
				Type:        "boolean",
				Description: "Recursively analyze subdirectories",
				Default:     true,
			},
			"include_tests": {
				Type:        "boolean",
				Description: "Include test files in analysis",
				Default:     true,
			},
		},
		Required: []string{"path"},
	}
}

// Execute executes the Go analyzer
func (g *GoAnalyzer) Execute(ctx context.Context, input *tools.ToolInput) (*tools.ToolResult, error) {
	startTime := time.Now()

	// Parse input parameters
	path, ok := input.Parameters["path"].(string)
	if !ok || path == "" {
		return &tools.ToolResult{
			Success: false,
			Error:   "path parameter is required",
		}, fmt.Errorf("path parameter is required")
	}

	analysisType := "all"
	if at, ok := input.Parameters["analysis_type"].(string); ok {
		analysisType = at
	}

	recursive := true
	if r, ok := input.Parameters["recursive"].(bool); ok {
		recursive = r
	}

	includeTests := true
	if it, ok := input.Parameters["include_tests"].(bool); ok {
		includeTests = it
	}

	g.logger.Info("Starting Go code analysis",
		slog.String("path", path),
		slog.String("analysis_type", analysisType),
		slog.Bool("recursive", recursive),
		slog.Bool("include_tests", includeTests))

	// Perform analysis
	result, err := g.analyzeGoCode(ctx, path, analysisType, recursive, includeTests)
	if err != nil {
		return &tools.ToolResult{
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: time.Since(startTime),
		}, err
	}

	// Convert AnalysisResult to ToolResultData
	toolData := &tools.ToolResultData{
		Analysis: &tools.AnalysisResult{
			Issues:  convertGoIssuesToToolIssues(result.Issues),
			Metrics: convertGoMetricsToToolMetrics(result.Metrics),
			// TODO: Add dependencies and test coverage conversion
		},
		LinesProcessed: int64(result.Summary.TotalLines),
	}

	// Create metadata
	metadata := &tools.ToolMetadata{
		StartTime:     startTime,
		EndTime:       time.Now(),
		ExecutionTime: time.Since(startTime),
		Parameters: map[string]string{
			"analysis_type": analysisType,
			"path":          path,
			"recursive":     fmt.Sprintf("%t", recursive),
			"include_tests": fmt.Sprintf("%t", includeTests),
		},
	}

	return &tools.ToolResult{
		Success:       true,
		Data:          toolData,
		Metadata:      metadata,
		ExecutionTime: time.Since(startTime),
	}, nil
}

// Health checks if the tool is healthy
func (g *GoAnalyzer) Health(ctx context.Context) error {
	// Check if we can parse a simple Go file
	testCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, "test.go", testCode, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("Go parser not working: %w", err)
	}

	return nil
}

// Close closes the tool and cleans up resources
func (g *GoAnalyzer) Close(ctx context.Context) error {
	g.logger.Debug("Go analyzer tool closed")
	return nil
}

// AnalysisResult represents the result of Go code analysis
type AnalysisResult struct {
	Summary        AnalysisSummary     `json:"summary"`
	Files          []FileAnalysis      `json:"files"`
	Issues         []CodeIssue         `json:"issues"`
	Suggestions    []Suggestion        `json:"suggestions"`
	Metrics        ProjectMetrics      `json:"metrics"`
	Dependencies   DependencyAnalysis  `json:"dependencies"`
	SecurityIssues []SecurityIssue     `json:"security_issues"`
	Performance    PerformanceAnalysis `json:"performance"`
}

// AnalysisSummary provides a high-level summary
type AnalysisSummary struct {
	TotalFiles        int     `json:"total_files"`
	TotalLines        int     `json:"total_lines"`
	TotalFunctions    int     `json:"total_functions"`
	TotalStructs      int     `json:"total_structs"`
	TotalInterfaces   int     `json:"total_interfaces"`
	AverageComplexity float64 `json:"average_complexity"`
	IssueCount        int     `json:"issue_count"`
	TestCoverage      float64 `json:"test_coverage"`
}

// FileAnalysis represents analysis of a single file
type FileAnalysis struct {
	Path       string            `json:"path"`
	Package    string            `json:"package"`
	Lines      int               `json:"lines"`
	Functions  []FunctionInfo    `json:"functions"`
	Structs    []StructInfo      `json:"structs"`
	Interfaces []InterfaceInfo   `json:"interfaces"`
	Imports    []string          `json:"imports"`
	Issues     []CodeIssue       `json:"issues"`
	Complexity ComplexityMetrics `json:"complexity"`
}

// FunctionInfo represents information about a function
type FunctionInfo struct {
	Name       string `json:"name"`
	Receiver   string `json:"receiver,omitempty"`
	Parameters int    `json:"parameters"`
	Returns    int    `json:"returns"`
	Lines      int    `json:"lines"`
	Complexity int    `json:"complexity"`
	IsExported bool   `json:"is_exported"`
	IsTest     bool   `json:"is_test"`
	DocComment string `json:"doc_comment,omitempty"`
}

// StructInfo represents information about a struct
type StructInfo struct {
	Name       string      `json:"name"`
	Fields     int         `json:"fields"`
	Methods    int         `json:"methods"`
	IsExported bool        `json:"is_exported"`
	DocComment string      `json:"doc_comment,omitempty"`
	FieldList  []FieldInfo `json:"field_list,omitempty"`
}

// FieldInfo represents information about a struct field
type FieldInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Tag        string `json:"tag,omitempty"`
	IsExported bool   `json:"is_exported"`
}

// InterfaceInfo represents information about an interface
type InterfaceInfo struct {
	Name       string       `json:"name"`
	Methods    int          `json:"methods"`
	IsExported bool         `json:"is_exported"`
	DocComment string       `json:"doc_comment,omitempty"`
	MethodList []MethodInfo `json:"method_list,omitempty"`
}

// MethodInfo represents information about an interface method
type MethodInfo struct {
	Name       string `json:"name"`
	Parameters int    `json:"parameters"`
	Returns    int    `json:"returns"`
	Signature  string `json:"signature"`
}

// CodeIssue represents a potential issue in the code
type CodeIssue struct {
	Type       IssueType `json:"type"`
	Severity   Severity  `json:"severity"`
	File       string    `json:"file"`
	Line       int       `json:"line"`
	Column     int       `json:"column"`
	Message    string    `json:"message"`
	Suggestion string    `json:"suggestion,omitempty"`
	Rule       string    `json:"rule"`
}

// IssueType defines types of code issues
type IssueType string

const (
	IssueTypeStyle       IssueType = "style"
	IssueTypeComplexity  IssueType = "complexity"
	IssueTypeNaming      IssueType = "naming"
	IssueTypeStructure   IssueType = "structure"
	IssueTypePerformance IssueType = "performance"
	IssueTypeSecurity    IssueType = "security"
	IssueTypeBugRisk     IssueType = "bug_risk"
)

// Severity defines issue severity levels
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Suggestion represents a suggestion for improvement
type Suggestion struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Example     string `json:"example,omitempty"`
	Priority    int    `json:"priority"`
}

// ComplexityMetrics represents complexity metrics
type ComplexityMetrics struct {
	Cyclomatic      int     `json:"cyclomatic"`
	Cognitive       int     `json:"cognitive"`
	Halstead        float64 `json:"halstead"`
	Lines           int     `json:"lines"`
	Maintainability float64 `json:"maintainability"`
}

// ProjectMetrics represents overall project metrics
type ProjectMetrics struct {
	TotalComplexity   int     `json:"total_complexity"`
	AverageComplexity float64 `json:"average_complexity"`
	MaxComplexity     int     `json:"max_complexity"`
	TechnicalDebt     float64 `json:"technical_debt"`
	Maintainability   float64 `json:"maintainability"`
	TestToCodeRatio   float64 `json:"test_to_code_ratio"`
	DuplicationRate   float64 `json:"duplication_rate"`
}

// DependencyAnalysis represents dependency analysis results
type DependencyAnalysis struct {
	DirectDependencies   []string          `json:"direct_dependencies"`
	IndirectDependencies []string          `json:"indirect_dependencies"`
	TotalDependencies    int               `json:"total_dependencies"`
	UnusedDependencies   []string          `json:"unused_dependencies"`
	MissingDependencies  []string          `json:"missing_dependencies"`
	DependencyHealth     map[string]string `json:"dependency_health"`
}

// SecurityIssue represents a security vulnerability or concern
type SecurityIssue struct {
	Type        string   `json:"type"`
	Severity    Severity `json:"severity"`
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Column      int      `json:"column"`
	Message     string   `json:"message"`
	CWE         string   `json:"cwe,omitempty"`
	OWASP       string   `json:"owasp,omitempty"`
	Remediation string   `json:"remediation"`
}

// PerformanceAnalysis represents performance-related analysis
type PerformanceAnalysis struct {
	MemoryLeaks        []MemoryLeak      `json:"memory_leaks"`
	IneffientLoops     []IneffientCode   `json:"inefficient_loops"`
	GoroutineLeaks     []GoroutineLeak   `json:"goroutine_leaks"`
	UnbufferedChannels []ChannelIssue    `json:"unbuffered_channels"`
	LargeAllocations   []LargeAllocation `json:"large_allocations"`
}

// MemoryLeak represents a potential memory leak
type MemoryLeak struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// IneffientCode represents inefficient code patterns
type IneffientCode struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// GoroutineLeak represents a potential goroutine leak
type GoroutineLeak struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Description string `json:"description"`
	Risk        string `json:"risk"`
}

// ChannelIssue represents channel-related issues
type ChannelIssue struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// LargeAllocation represents large memory allocations
type LargeAllocation struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Size        string `json:"size"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// analyzeGoCode performs the actual analysis
func (g *GoAnalyzer) analyzeGoCode(ctx context.Context, path, analysisType string, recursive, includeTests bool) (*AnalysisResult, error) {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	result := &AnalysisResult{
		Files:          make([]FileAnalysis, 0),
		Issues:         make([]CodeIssue, 0),
		Suggestions:    make([]Suggestion, 0),
		SecurityIssues: make([]SecurityIssue, 0),
		Dependencies: DependencyAnalysis{
			DirectDependencies:   make([]string, 0),
			IndirectDependencies: make([]string, 0),
			UnusedDependencies:   make([]string, 0),
			MissingDependencies:  make([]string, 0),
			DependencyHealth:     make(map[string]string),
		},
		Performance: PerformanceAnalysis{
			MemoryLeaks:        make([]MemoryLeak, 0),
			IneffientLoops:     make([]IneffientCode, 0),
			GoroutineLeaks:     make([]GoroutineLeak, 0),
			UnbufferedChannels: make([]ChannelIssue, 0),
			LargeAllocations:   make([]LargeAllocation, 0),
		},
	}

	if info.IsDir() {
		// Analyze directory
		err = g.analyzeDirectory(ctx, path, recursive, includeTests, result)
	} else {
		// Analyze single file
		err = g.analyzeFile(ctx, path, result)
	}

	if err != nil {
		return nil, err
	}

	// Calculate summary and metrics
	g.calculateSummary(result)
	g.generateSuggestions(result)

	// Perform advanced analysis
	g.analyzeSecurity(result)
	g.analyzePerformance(result)
	g.analyzeDependencies(path, result)

	// Filter results based on analysis type
	if analysisType != "all" {
		g.filterResults(result, analysisType)
	}

	return result, nil
}

// analyzeDirectory analyzes all Go files in a directory
func (g *GoAnalyzer) analyzeDirectory(ctx context.Context, dirPath string, recursive, includeTests bool, result *AnalysisResult) error {
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files if not included
		if !includeTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Skip vendor directories
		if strings.Contains(path, "/vendor/") {
			return nil
		}

		// Skip if not recursive and in subdirectory
		if !recursive {
			rel, err := filepath.Rel(dirPath, path)
			if err != nil {
				return err
			}
			if strings.Contains(rel, string(filepath.Separator)) {
				return nil
			}
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		return g.analyzeFile(ctx, path, result)
	})
}

// analyzeFile analyzes a single Go file
func (g *GoAnalyzer) analyzeFile(ctx context.Context, filePath string, result *AnalysisResult) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse the file
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		// Add parsing error as an issue
		result.Issues = append(result.Issues, CodeIssue{
			Type:     IssueTypeBugRisk,
			Severity: SeverityError,
			File:     filePath,
			Line:     1,
			Column:   1,
			Message:  fmt.Sprintf("Parse error: %v", err),
			Rule:     "syntax",
		})
		return nil // Continue with other files
	}

	fileAnalysis := FileAnalysis{
		Path:       filePath,
		Package:    file.Name.Name,
		Imports:    make([]string, 0),
		Functions:  make([]FunctionInfo, 0),
		Structs:    make([]StructInfo, 0),
		Interfaces: make([]InterfaceInfo, 0),
		Issues:     make([]CodeIssue, 0),
	}

	// Count lines
	fileAnalysis.Lines = strings.Count(string(content), "\n") + 1

	// Analyze imports
	for _, importSpec := range file.Imports {
		importPath := strings.Trim(importSpec.Path.Value, `"`)
		fileAnalysis.Imports = append(fileAnalysis.Imports, importPath)
	}

	// Walk the AST
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			g.analyzeFunctionDecl(node, fset, &fileAnalysis)
		case *ast.GenDecl:
			g.analyzeGenDecl(node, fset, &fileAnalysis)
		}
		return true
	})

	// Calculate complexity metrics
	fileAnalysis.Complexity = g.calculateFileComplexity(&fileAnalysis)

	// Detect issues
	g.detectIssues(file, fset, &fileAnalysis)

	result.Files = append(result.Files, fileAnalysis)
	result.Issues = append(result.Issues, fileAnalysis.Issues...)

	return nil
}

// analyzeFunctionDecl analyzes a function declaration
func (g *GoAnalyzer) analyzeFunctionDecl(funcDecl *ast.FuncDecl, fset *token.FileSet, fileAnalysis *FileAnalysis) {
	if funcDecl.Name == nil {
		return
	}

	funcInfo := FunctionInfo{
		Name:       funcDecl.Name.Name,
		IsExported: ast.IsExported(funcDecl.Name.Name),
		IsTest:     strings.HasPrefix(funcDecl.Name.Name, "Test"),
	}

	// Get receiver if it's a method
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		if len(funcDecl.Recv.List[0].Names) > 0 {
			funcInfo.Receiver = funcDecl.Recv.List[0].Names[0].Name
		}
	}

	// Count parameters
	if funcDecl.Type.Params != nil {
		funcInfo.Parameters = len(funcDecl.Type.Params.List)
	}

	// Count return values
	if funcDecl.Type.Results != nil {
		funcInfo.Returns = len(funcDecl.Type.Results.List)
	}

	// Calculate lines
	if funcDecl.Body != nil {
		start := fset.Position(funcDecl.Pos()).Line
		end := fset.Position(funcDecl.End()).Line
		funcInfo.Lines = end - start + 1
	}

	// Calculate cyclomatic complexity
	funcInfo.Complexity = g.calculateCyclomaticComplexity(funcDecl)

	// Get documentation
	if funcDecl.Doc != nil {
		funcInfo.DocComment = funcDecl.Doc.Text()
	}

	fileAnalysis.Functions = append(fileAnalysis.Functions, funcInfo)
}

// analyzeGenDecl analyzes a general declaration (types, consts, vars)
func (g *GoAnalyzer) analyzeGenDecl(genDecl *ast.GenDecl, fset *token.FileSet, fileAnalysis *FileAnalysis) {
	for _, spec := range genDecl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			g.analyzeTypeSpec(s, genDecl, fileAnalysis)
		}
	}
}

// analyzeTypeSpec analyzes a type specification
func (g *GoAnalyzer) analyzeTypeSpec(typeSpec *ast.TypeSpec, genDecl *ast.GenDecl, fileAnalysis *FileAnalysis) {
	typeName := typeSpec.Name.Name
	isExported := ast.IsExported(typeName)
	docComment := ""
	if genDecl.Doc != nil {
		docComment = genDecl.Doc.Text()
	}

	switch t := typeSpec.Type.(type) {
	case *ast.StructType:
		structInfo := StructInfo{
			Name:       typeName,
			IsExported: isExported,
			DocComment: docComment,
			FieldList:  make([]FieldInfo, 0),
		}

		if t.Fields != nil {
			structInfo.Fields = len(t.Fields.List)
			for _, field := range t.Fields.List {
				for _, name := range field.Names {
					fieldInfo := FieldInfo{
						Name:       name.Name,
						Type:       g.typeToString(field.Type),
						IsExported: ast.IsExported(name.Name),
					}
					if field.Tag != nil {
						fieldInfo.Tag = field.Tag.Value
					}
					structInfo.FieldList = append(structInfo.FieldList, fieldInfo)
				}
			}
		}

		fileAnalysis.Structs = append(fileAnalysis.Structs, structInfo)

	case *ast.InterfaceType:
		interfaceInfo := InterfaceInfo{
			Name:       typeName,
			IsExported: isExported,
			DocComment: docComment,
			MethodList: make([]MethodInfo, 0),
		}

		if t.Methods != nil {
			interfaceInfo.Methods = len(t.Methods.List)
			for _, method := range t.Methods.List {
				if len(method.Names) > 0 {
					methodInfo := MethodInfo{
						Name: method.Names[0].Name,
					}
					if funcType, ok := method.Type.(*ast.FuncType); ok {
						if funcType.Params != nil {
							methodInfo.Parameters = len(funcType.Params.List)
						}
						if funcType.Results != nil {
							methodInfo.Returns = len(funcType.Results.List)
						}
						methodInfo.Signature = g.funcTypeToString(funcType)
					}
					interfaceInfo.MethodList = append(interfaceInfo.MethodList, methodInfo)
				}
			}
		}

		fileAnalysis.Interfaces = append(fileAnalysis.Interfaces, interfaceInfo)
	}
}

// calculateCyclomaticComplexity calculates the cyclomatic complexity of a function
func (g *GoAnalyzer) calculateCyclomaticComplexity(funcDecl *ast.FuncDecl) int {
	complexity := 1 // Base complexity

	if funcDecl.Body == nil {
		return complexity
	}

	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		}
		return true
	})

	return complexity
}

// calculateFileComplexity calculates complexity metrics for a file
func (g *GoAnalyzer) calculateFileComplexity(fileAnalysis *FileAnalysis) ComplexityMetrics {
	totalComplexity := 0
	for _, function := range fileAnalysis.Functions {
		totalComplexity += function.Complexity
	}

	avgComplexity := 0.0
	if len(fileAnalysis.Functions) > 0 {
		avgComplexity = float64(totalComplexity) / float64(len(fileAnalysis.Functions))
	}

	// Simple maintainability calculation (0-100 scale)
	maintainability := 100.0
	if avgComplexity > 10 {
		maintainability -= (avgComplexity - 10) * 5
	}
	if maintainability < 0 {
		maintainability = 0
	}

	return ComplexityMetrics{
		Cyclomatic:      totalComplexity,
		Cognitive:       totalComplexity, // Simplified
		Lines:           fileAnalysis.Lines,
		Maintainability: maintainability,
	}
}

// detectIssues detects potential issues in the code
func (g *GoAnalyzer) detectIssues(file *ast.File, fset *token.FileSet, fileAnalysis *FileAnalysis) {
	// Check for missing package documentation
	if file.Doc == nil && file.Name.Name != "main" {
		fileAnalysis.Issues = append(fileAnalysis.Issues, CodeIssue{
			Type:       IssueTypeStyle,
			Severity:   SeverityWarning,
			File:       fileAnalysis.Path,
			Line:       1,
			Column:     1,
			Message:    "Package is missing documentation",
			Suggestion: "Add a package comment explaining the purpose of this package",
			Rule:       "missing_package_doc",
		})
	}

	// Check functions for issues
	for _, function := range fileAnalysis.Functions {
		// High complexity warning
		if function.Complexity > 10 {
			fileAnalysis.Issues = append(fileAnalysis.Issues, CodeIssue{
				Type:       IssueTypeComplexity,
				Severity:   SeverityWarning,
				File:       fileAnalysis.Path,
				Message:    fmt.Sprintf("Function '%s' has high cyclomatic complexity (%d)", function.Name, function.Complexity),
				Suggestion: "Consider breaking this function into smaller, more focused functions",
				Rule:       "high_complexity",
			})
		}

		// Missing documentation for exported functions
		if function.IsExported && function.DocComment == "" && !function.IsTest {
			fileAnalysis.Issues = append(fileAnalysis.Issues, CodeIssue{
				Type:       IssueTypeStyle,
				Severity:   SeverityWarning,
				File:       fileAnalysis.Path,
				Message:    fmt.Sprintf("Exported function '%s' is missing documentation", function.Name),
				Suggestion: fmt.Sprintf("Add a comment starting with '%s' to document this function", function.Name),
				Rule:       "missing_exported_doc",
			})
		}

		// Very long functions
		if function.Lines > 50 {
			fileAnalysis.Issues = append(fileAnalysis.Issues, CodeIssue{
				Type:       IssueTypeStructure,
				Severity:   SeverityInfo,
				File:       fileAnalysis.Path,
				Message:    fmt.Sprintf("Function '%s' is very long (%d lines)", function.Name, function.Lines),
				Suggestion: "Consider breaking this function into smaller, more focused functions",
				Rule:       "long_function",
			})
		}
	}

	// Check structs for issues
	for _, structInfo := range fileAnalysis.Structs {
		// Missing documentation for exported structs
		if structInfo.IsExported && structInfo.DocComment == "" {
			fileAnalysis.Issues = append(fileAnalysis.Issues, CodeIssue{
				Type:       IssueTypeStyle,
				Severity:   SeverityWarning,
				File:       fileAnalysis.Path,
				Message:    fmt.Sprintf("Exported struct '%s' is missing documentation", structInfo.Name),
				Suggestion: fmt.Sprintf("Add a comment starting with '%s' to document this struct", structInfo.Name),
				Rule:       "missing_exported_doc",
			})
		}

		// Large structs
		if structInfo.Fields > 20 {
			fileAnalysis.Issues = append(fileAnalysis.Issues, CodeIssue{
				Type:       IssueTypeStructure,
				Severity:   SeverityInfo,
				File:       fileAnalysis.Path,
				Message:    fmt.Sprintf("Struct '%s' has many fields (%d)", structInfo.Name, structInfo.Fields),
				Suggestion: "Consider breaking this struct into smaller, more focused structs",
				Rule:       "large_struct",
			})
		}
	}
}

// calculateSummary calculates the analysis summary
func (g *GoAnalyzer) calculateSummary(result *AnalysisResult) {
	summary := AnalysisSummary{
		TotalFiles: len(result.Files),
		IssueCount: len(result.Issues),
	}

	totalComplexity := 0
	functionCount := 0

	for _, file := range result.Files {
		summary.TotalLines += file.Lines
		summary.TotalFunctions += len(file.Functions)
		summary.TotalStructs += len(file.Structs)
		summary.TotalInterfaces += len(file.Interfaces)

		for _, function := range file.Functions {
			totalComplexity += function.Complexity
			functionCount++
		}
	}

	if functionCount > 0 {
		summary.AverageComplexity = float64(totalComplexity) / float64(functionCount)
	}

	// Calculate test coverage estimation (simplified)
	testFiles := 0
	for _, file := range result.Files {
		if strings.HasSuffix(file.Path, "_test.go") {
			testFiles++
		}
	}
	if summary.TotalFiles > 0 {
		summary.TestCoverage = float64(testFiles) / float64(summary.TotalFiles) * 100
	}

	// Calculate project metrics
	result.Metrics = ProjectMetrics{
		TotalComplexity:   totalComplexity,
		AverageComplexity: summary.AverageComplexity,
		Maintainability:   g.calculateMaintainability(result),
		TestToCodeRatio:   summary.TestCoverage / 100,
	}

	// Find max complexity
	for _, file := range result.Files {
		for _, function := range file.Functions {
			if function.Complexity > result.Metrics.MaxComplexity {
				result.Metrics.MaxComplexity = function.Complexity
			}
		}
	}

	result.Summary = summary
}

// generateSuggestions generates improvement suggestions
func (g *GoAnalyzer) generateSuggestions(result *AnalysisResult) {
	suggestions := make([]Suggestion, 0)

	// High-level suggestions based on metrics
	if result.Summary.AverageComplexity > 5 {
		suggestions = append(suggestions, Suggestion{
			Type:        "complexity",
			Title:       "Reduce Code Complexity",
			Description: "The average cyclomatic complexity is high. Consider refactoring complex functions.",
			Priority:    8,
		})
	}

	if result.Summary.TestCoverage < 50 {
		suggestions = append(suggestions, Suggestion{
			Type:        "testing",
			Title:       "Improve Test Coverage",
			Description: "Test coverage is low. Consider adding more test files.",
			Priority:    7,
		})
	}

	// Issue-based suggestions
	issueTypes := make(map[IssueType]int)
	for _, issue := range result.Issues {
		issueTypes[issue.Type]++
	}

	if issueTypes[IssueTypeStyle] > 5 {
		suggestions = append(suggestions, Suggestion{
			Type:        "style",
			Title:       "Improve Code Documentation",
			Description: "Multiple style issues detected. Focus on improving documentation for exported items.",
			Priority:    6,
		})
	}

	if issueTypes[IssueTypeComplexity] > 0 {
		suggestions = append(suggestions, Suggestion{
			Type:        "refactoring",
			Title:       "Refactor Complex Functions",
			Description: "Some functions have high complexity. Consider breaking them into smaller functions.",
			Priority:    9,
		})
	}

	result.Suggestions = suggestions
}

// calculateMaintainability calculates overall maintainability score
func (g *GoAnalyzer) calculateMaintainability(result *AnalysisResult) float64 {
	if len(result.Files) == 0 {
		return 100.0
	}

	totalMaintainability := 0.0
	for _, file := range result.Files {
		totalMaintainability += file.Complexity.Maintainability
	}

	return totalMaintainability / float64(len(result.Files))
}

// filterResults filters results based on analysis type
func (g *GoAnalyzer) filterResults(result *AnalysisResult, analysisType string) {
	switch analysisType {
	case "structure":
		// Keep only structural information, remove complexity and issues
		result.Issues = make([]CodeIssue, 0)
		for i := range result.Files {
			result.Files[i].Issues = make([]CodeIssue, 0)
		}
	case "complexity":
		// Focus on complexity metrics
		filteredIssues := make([]CodeIssue, 0)
		for _, issue := range result.Issues {
			if issue.Type == IssueTypeComplexity {
				filteredIssues = append(filteredIssues, issue)
			}
		}
		result.Issues = filteredIssues
	case "issues":
		// Keep only issues, remove detailed structural info
		for i := range result.Files {
			result.Files[i].Functions = make([]FunctionInfo, 0)
			result.Files[i].Structs = make([]StructInfo, 0)
			result.Files[i].Interfaces = make([]InterfaceInfo, 0)
		}
	}
}

// Helper functions

func (g *GoAnalyzer) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + g.typeToString(t.X)
	case *ast.ArrayType:
		return "[]" + g.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + g.typeToString(t.Key) + "]" + g.typeToString(t.Value)
	case *ast.ChanType:
		return "chan " + g.typeToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.FuncType:
		return g.funcTypeToString(t)
	case *ast.SelectorExpr:
		return g.typeToString(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

func (g *GoAnalyzer) funcTypeToString(funcType *ast.FuncType) string {
	result := "func("

	if funcType.Params != nil {
		for i, param := range funcType.Params.List {
			if i > 0 {
				result += ", "
			}
			result += g.typeToString(param.Type)
		}
	}

	result += ")"

	if funcType.Results != nil && len(funcType.Results.List) > 0 {
		result += " "
		if len(funcType.Results.List) > 1 {
			result += "("
		}
		for i, resultField := range funcType.Results.List {
			if i > 0 {
				result += ", "
			}
			result += g.typeToString(resultField.Type)
		}
		if len(funcType.Results.List) > 1 {
			result += ")"
		}
	}

	return result
}

// analyzeSecurity performs security analysis on the code
func (g *GoAnalyzer) analyzeSecurity(result *AnalysisResult) {
	for _, file := range result.Files {
		// Check for hardcoded credentials
		g.checkHardcodedCredentials(&file, result)

		// Check for SQL injection vulnerabilities
		g.checkSQLInjection(&file, result)

		// Check for command injection
		g.checkCommandInjection(&file, result)

		// Check for path traversal
		g.checkPathTraversal(&file, result)

		// Check for insecure random number generation
		g.checkInsecureRandom(&file, result)
	}
}

// checkHardcodedCredentials checks for hardcoded passwords and API keys
func (g *GoAnalyzer) checkHardcodedCredentials(file *FileAnalysis, result *AnalysisResult) {
	// Common patterns for credentials
	credentialPatterns := []string{
		"password", "passwd", "pwd", "secret", "api_key", "apikey",
		"access_key", "private_key", "token", "auth",
	}

	for _, function := range file.Functions {
		for _, pattern := range credentialPatterns {
			if strings.Contains(strings.ToLower(function.Name), pattern) {
				result.SecurityIssues = append(result.SecurityIssues, SecurityIssue{
					Type:        "hardcoded_credentials",
					Severity:    SeverityCritical,
					File:        file.Path,
					Line:        0, // Would need AST position
					Message:     fmt.Sprintf("Function name '%s' suggests credential handling - ensure no hardcoded values", function.Name),
					CWE:         "CWE-798",
					Remediation: "Use environment variables or secure credential management systems",
				})
			}
		}
	}
}

// checkSQLInjection checks for potential SQL injection vulnerabilities
func (g *GoAnalyzer) checkSQLInjection(file *FileAnalysis, result *AnalysisResult) {
	// Check for SQL query construction patterns
	for _, imp := range file.Imports {
		if strings.Contains(imp, "database/sql") {
			// Flag files using database/sql for manual review
			result.SecurityIssues = append(result.SecurityIssues, SecurityIssue{
				Type:        "sql_injection_risk",
				Severity:    SeverityWarning,
				File:        file.Path,
				Line:        0,
				Message:     "File uses database/sql - ensure parameterized queries are used",
				CWE:         "CWE-89",
				OWASP:       "A03:2021",
				Remediation: "Always use parameterized queries or prepared statements",
			})
			break
		}
	}
}

// checkCommandInjection checks for command injection vulnerabilities
func (g *GoAnalyzer) checkCommandInjection(file *FileAnalysis, result *AnalysisResult) {
	// Check for os/exec usage
	for _, imp := range file.Imports {
		if strings.Contains(imp, "os/exec") {
			result.SecurityIssues = append(result.SecurityIssues, SecurityIssue{
				Type:        "command_injection_risk",
				Severity:    SeverityWarning,
				File:        file.Path,
				Line:        0,
				Message:     "File uses os/exec - ensure user input is properly sanitized",
				CWE:         "CWE-78",
				OWASP:       "A03:2021",
				Remediation: "Validate and sanitize all user input before passing to exec commands",
			})
			break
		}
	}
}

// checkPathTraversal checks for path traversal vulnerabilities
func (g *GoAnalyzer) checkPathTraversal(file *FileAnalysis, result *AnalysisResult) {
	// Check for file operations
	filePackages := []string{"os", "io/ioutil", "path/filepath"}
	for _, imp := range file.Imports {
		for _, pkg := range filePackages {
			if strings.Contains(imp, pkg) {
				result.SecurityIssues = append(result.SecurityIssues, SecurityIssue{
					Type:        "path_traversal_risk",
					Severity:    SeverityInfo,
					File:        file.Path,
					Line:        0,
					Message:     fmt.Sprintf("File uses %s - ensure paths are properly validated", pkg),
					CWE:         "CWE-22",
					Remediation: "Use filepath.Clean() and validate paths against allowed directories",
				})
				break
			}
		}
	}
}

// checkInsecureRandom checks for insecure random number generation
func (g *GoAnalyzer) checkInsecureRandom(file *FileAnalysis, result *AnalysisResult) {
	// Check for math/rand usage
	for _, imp := range file.Imports {
		if imp == "math/rand" {
			result.SecurityIssues = append(result.SecurityIssues, SecurityIssue{
				Type:        "insecure_random",
				Severity:    SeverityWarning,
				File:        file.Path,
				Line:        0,
				Message:     "Using math/rand for security-sensitive operations is insecure",
				CWE:         "CWE-338",
				Remediation: "Use crypto/rand for cryptographic operations",
			})
			break
		}
	}
}

// analyzePerformance performs performance analysis on the code
func (g *GoAnalyzer) analyzePerformance(result *AnalysisResult) {
	for i := range result.Files {
		file := &result.Files[i]

		// Check for performance issues in functions
		for _, function := range file.Functions {
			// Check for inefficient patterns
			if function.Complexity > 15 {
				result.Performance.IneffientLoops = append(result.Performance.IneffientLoops, IneffientCode{
					File:        file.Path,
					Line:        0,
					Pattern:     "high_complexity",
					Description: fmt.Sprintf("Function %s has high complexity (%d)", function.Name, function.Complexity),
					Suggestion:  "Consider breaking down into smaller functions",
				})
			}

			// Check for large functions that might have memory issues
			if function.Lines > 100 {
				result.Performance.LargeAllocations = append(result.Performance.LargeAllocations, LargeAllocation{
					File:        file.Path,
					Line:        0,
					Size:        fmt.Sprintf("%d lines", function.Lines),
					Type:        "large_function",
					Description: fmt.Sprintf("Function %s is very large", function.Name),
				})
			}
		}

		// Check for unbuffered channel patterns
		for _, imp := range file.Imports {
			if strings.Contains(imp, "sync") {
				result.Performance.UnbufferedChannels = append(result.Performance.UnbufferedChannels, ChannelIssue{
					File:        file.Path,
					Line:        0,
					Type:        "sync_usage",
					Description: "File uses sync package - check for proper channel buffering",
				})
				break
			}
		}
	}
}

// analyzeDependencies analyzes project dependencies
func (g *GoAnalyzer) analyzeDependencies(projectPath string, result *AnalysisResult) {
	// Check for go.mod file
	goModPath := filepath.Join(projectPath, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		// Parse go.mod file (simplified version)
		content, err := os.ReadFile(goModPath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "require") && strings.Contains(line, "(") {
					// Multi-line require block
					continue
				}
				if strings.HasPrefix(line, "require ") {
					// Single line require
					parts := strings.Fields(line)
					if len(parts) >= 3 {
						dep := parts[1]
						result.Dependencies.DirectDependencies = append(result.Dependencies.DirectDependencies, dep)
					}
				}
				// Simple parsing - in production would use proper go.mod parser
				if strings.Contains(line, "//") && strings.Contains(line, "indirect") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						dep := parts[0]
						result.Dependencies.IndirectDependencies = append(result.Dependencies.IndirectDependencies, dep)
					}
				}
			}
		}
	}

	// Update dependency counts
	result.Dependencies.TotalDependencies = len(result.Dependencies.DirectDependencies) + len(result.Dependencies.IndirectDependencies)

	// Check dependency health (simplified)
	for _, dep := range result.Dependencies.DirectDependencies {
		if strings.Contains(dep, "deprecated") {
			result.Dependencies.DependencyHealth[dep] = "deprecated"
		} else {
			result.Dependencies.DependencyHealth[dep] = "ok"
		}
	}
}

// convertGoIssuesToToolIssues converts Go analyzer issues to tool issues
func convertGoIssuesToToolIssues(goIssues []CodeIssue) []tools.Issue {
	issues := make([]tools.Issue, 0, len(goIssues))
	for _, issue := range goIssues {
		toolIssue := tools.Issue{
			Type:        string(issue.Type),
			Severity:    string(issue.Severity),
			Message:     issue.Message,
			File:        issue.File,
			Line:        issue.Line,
			Column:      issue.Column,
			Rule:        issue.Rule,
			Suggestions: []string{issue.Suggestion}, // Convert single suggestion to slice
		}
		issues = append(issues, toolIssue)
	}
	return issues
}

// convertGoMetricsToToolMetrics converts Go metrics to tool metrics
func convertGoMetricsToToolMetrics(goMetrics ProjectMetrics) *tools.CodeMetrics {
	return &tools.CodeMetrics{
		LinesOfCode:          0,                                // ProjectMetrics doesn't have TotalLines
		CyclomaticComplexity: int(goMetrics.AverageComplexity), // Convert float64 to int
		CognitiveComplexity:  goMetrics.MaxComplexity,
		TestCoveragePercent:  goMetrics.TestToCodeRatio * 100,              // Use TestToCodeRatio
		DuplicationPercent:   goMetrics.DuplicationRate * 100,              // Use DuplicationRate
		TechnicalDebt:        fmt.Sprintf("%.2f", goMetrics.TechnicalDebt), // Use TechnicalDebt
	}
}
