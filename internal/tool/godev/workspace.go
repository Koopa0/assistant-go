package godev

import (
	"bufio"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// WorkspaceDetector handles Go workspace detection and analysis
type WorkspaceDetector struct {
	logger interface{} // Will be *slog.Logger in practice
}

// NewWorkspaceDetector creates a new workspace detector
func NewWorkspaceDetector(logger interface{}) *WorkspaceDetector {
	return &WorkspaceDetector{
		logger: logger,
	}
}

// DetectWorkspace detects and analyzes a Go workspace starting from the given path
func (w *WorkspaceDetector) DetectWorkspace(ctx context.Context, startPath string, options *AnalysisOptions) (*AnalysisResult, error) {
	if options == nil {
		options = DefaultAnalysisOptions()
	}

	// Find the Go module root
	rootPath, err := w.findModuleRoot(startPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find Go module root: %w", err)
	}

	// Create workspace info
	workspace := &WorkspaceInfo{
		RootPath:   rootPath,
		AnalyzedAt: time.Now(),
		Version:    "1.0.0",
	}

	// Parse go.mod file
	if err := w.parseGoMod(workspace); err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	// Detect project type
	workspace.ProjectType = w.detectProjectType(workspace.RootPath)

	// Analyze packages
	if err := w.analyzePackages(ctx, workspace, options); err != nil {
		return nil, fmt.Errorf("failed to analyze packages: %w", err)
	}

	// Analyze dependencies if requested
	if options.IncludeDependencies {
		if err := w.analyzeDependencies(workspace); err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: failed to analyze dependencies: %v\n", err)
		}
	}

	// Get Git information if requested
	if options.IncludeGitInfo {
		if gitInfo, err := w.getGitInfo(workspace.RootPath); err == nil {
			workspace.GitInfo = gitInfo
		}
	}

	// Run test coverage if requested
	if options.IncludeCoverage {
		if coverage, err := w.analyzeTestCoverage(ctx, workspace.RootPath); err == nil {
			workspace.TestCoverage = coverage
		}
	}

	// Get build information if requested
	if options.IncludeBuildInfo {
		if buildInfo, err := w.getBuildInfo(ctx, workspace.RootPath); err == nil {
			workspace.BuildInfo = buildInfo
		}
	}

	// Generate analysis results
	result := &AnalysisResult{
		Workspace:   workspace,
		Issues:      w.analyzeIssues(workspace),
		Suggestions: w.generateSuggestions(workspace),
		Metrics:     w.calculateMetrics(workspace),
	}

	return result, nil
}

// findModuleRoot finds the root directory containing go.mod
func (w *WorkspaceDetector) findModuleRoot(startPath string) (string, error) {
	abs, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	current := abs
	for {
		goModPath := filepath.Join(current, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break // Reached root directory
		}
		current = parent
	}

	return "", fmt.Errorf("no go.mod found in %s or any parent directory", startPath)
}

// parseGoMod parses the go.mod file to extract module information
func (w *WorkspaceDetector) parseGoMod(workspace *WorkspaceInfo) error {
	goModPath := filepath.Join(workspace.RootPath, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var inRequire bool

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse module declaration
		if strings.HasPrefix(line, "module ") {
			workspace.ModulePath = strings.TrimSpace(strings.TrimPrefix(line, "module"))
			continue
		}

		// Parse Go version
		if strings.HasPrefix(line, "go ") {
			workspace.GoVersion = strings.TrimSpace(strings.TrimPrefix(line, "go"))
			continue
		}

		// Track require block
		if strings.HasPrefix(line, "require (") {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		// Parse dependencies
		if inRequire || strings.HasPrefix(line, "require ") {
			if dep := w.parseDependencyLine(line); dep != nil {
				workspace.Dependencies = append(workspace.Dependencies, *dep)
			}
		}
	}

	return nil
}

// parseDependencyLine parses a single dependency line from go.mod
func (w *WorkspaceDetector) parseDependencyLine(line string) *DependencyInfo {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "//") {
		return nil
	}

	// Remove "require " prefix if present
	line = strings.TrimPrefix(line, "require ")

	// Handle indirect dependencies
	isIndirect := strings.Contains(line, "// indirect")
	line = strings.Split(line, "//")[0]
	line = strings.TrimSpace(line)

	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	return &DependencyInfo{
		ModulePath: parts[0],
		Version:    parts[1],
		IsIndirect: isIndirect,
	}
}

// detectProjectType determines the type of Go project
func (w *WorkspaceDetector) detectProjectType(rootPath string) ProjectType {
	// Check for main packages
	hasMain := false
	hasHTTPServer := false
	hasMultipleMain := false
	mainCount := 0

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			if content, err := os.ReadFile(path); err == nil {
				if strings.Contains(string(content), "package main") {
					hasMain = true
					mainCount++
					if mainCount > 1 {
						hasMultipleMain = true
					}

					// Check for HTTP server indicators
					if strings.Contains(string(content), "http.ListenAndServe") ||
						strings.Contains(string(content), "gin.Engine") ||
						strings.Contains(string(content), "fiber.App") ||
						strings.Contains(string(content), "echo.Echo") ||
						strings.Contains(string(content), "mux.Router") {
						hasHTTPServer = true
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return ProjectTypeUnknown
	}

	// Determine project type based on analysis
	if hasMultipleMain {
		return ProjectTypeMonorepo
	}

	if hasMain {
		if hasHTTPServer {
			// Check for microservice patterns
			if w.hasMicroservicePatterns(rootPath) {
				return ProjectTypeMicroservice
			}
			return ProjectTypeWebService
		}
		return ProjectTypeCLI
	}

	return ProjectTypeLibrary
}

// hasMicroservicePatterns checks for common microservice patterns
func (w *WorkspaceDetector) hasMicroservicePatterns(rootPath string) bool {
	// Check for common microservice directories/files
	patterns := []string{
		"cmd/", "internal/", "pkg/",
		"Dockerfile", "docker-compose.yml",
		"k8s/", "kubernetes/", "helm/",
		"api/", "proto/", "grpc/",
	}

	for _, pattern := range patterns {
		if _, err := os.Stat(filepath.Join(rootPath, pattern)); err == nil {
			return true
		}
	}

	return false
}

// analyzePackages analyzes all Go packages in the workspace
func (w *WorkspaceDetector) analyzePackages(ctx context.Context, workspace *WorkspaceInfo, options *AnalysisOptions) error {
	fset := token.NewFileSet()

	err := filepath.WalkDir(workspace.RootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		// Skip vendor directory if requested
		if options.ExcludeVendor && strings.Contains(path, "vendor/") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Process Go files
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			// Skip test files if not requested
			if !options.IncludeTestFiles && strings.HasSuffix(path, "_test.go") {
				return nil
			}

			if err := w.analyzeGoFile(fset, workspace, path); err != nil {
				// Log error but continue
				fmt.Printf("Warning: failed to analyze %s: %v\n", path, err)
			}
		}

		return nil
	})

	return err
}

// analyzeGoFile analyzes a single Go file
func (w *WorkspaceDetector) analyzeGoFile(fset *token.FileSet, workspace *WorkspaceInfo, filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Parse the file
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return err
	}

	// Get or create package info
	pkgPath := filepath.Dir(filePath)
	relPath, _ := filepath.Rel(workspace.RootPath, pkgPath)
	importPath := filepath.Join(workspace.ModulePath, relPath)

	var pkg *PackageInfo
	for i := range workspace.Packages {
		if workspace.Packages[i].Path == pkgPath {
			pkg = &workspace.Packages[i]
			break
		}
	}

	if pkg == nil {
		pkg = &PackageInfo{
			Name:       file.Name.Name,
			Path:       pkgPath,
			ImportPath: importPath,
			IsMain:     file.Name.Name == "main",
		}
		workspace.Packages = append(workspace.Packages, *pkg)
		pkg = &workspace.Packages[len(workspace.Packages)-1]
	}

	// Count lines
	lines := strings.Split(string(content), "\n")
	pkg.LineCount += len(lines)
	pkg.FileCount++

	// Check if it's a test file
	if strings.HasSuffix(filePath, "_test.go") {
		pkg.TestFiles = append(pkg.TestFiles, filePath)
	}

	// Analyze imports
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		pkg.Imports = append(pkg.Imports, importPath)

		// Check if it's an external dependency
		if !strings.HasPrefix(importPath, workspace.ModulePath) &&
			!strings.Contains(importPath, ".") == false {
			pkg.ExternalDeps = append(pkg.ExternalDeps, importPath)
		}
	}

	// Analyze declarations
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			w.analyzeFunctionDecl(pkg, d, fset)
		case *ast.GenDecl:
			w.analyzeGenDecl(pkg, d, fset)
		}
	}

	return nil
}

// analyzeFunctionDecl analyzes a function declaration
func (w *WorkspaceDetector) analyzeFunctionDecl(pkg *PackageInfo, fn *ast.FuncDecl, fset *token.FileSet) {
	if fn.Name == nil {
		return
	}

	pos := fset.Position(fn.Pos())
	end := fset.Position(fn.End())

	funcInfo := FunctionInfo{
		Name:       fn.Name.Name,
		Package:    pkg.Name,
		IsExported: ast.IsExported(fn.Name.Name),
		LineStart:  pos.Line,
		LineEnd:    end.Line,
		IsTest:     strings.HasPrefix(fn.Name.Name, "Test"),
		IsBench:    strings.HasPrefix(fn.Name.Name, "Benchmark"),
	}

	// Analyze parameters
	if fn.Type.Params != nil {
		for _, param := range fn.Type.Params.List {
			paramType := w.extractTypeString(param.Type)
			for _, name := range param.Names {
				funcInfo.Parameters = append(funcInfo.Parameters, fmt.Sprintf("%s %s", name.Name, paramType))
			}
		}
	}

	// Analyze return types
	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			returnType := w.extractTypeString(result.Type)
			funcInfo.Returns = append(funcInfo.Returns, returnType)
		}
	}

	// Calculate cyclomatic complexity (simplified)
	funcInfo.Complexity = w.calculateComplexity(fn)

	pkg.Functions = append(pkg.Functions, funcInfo)
}

// analyzeGenDecl analyzes a general declaration (type, const, var)
func (w *WorkspaceDetector) analyzeGenDecl(pkg *PackageInfo, gen *ast.GenDecl, fset *token.FileSet) {
	for _, spec := range gen.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			w.analyzeTypeSpec(pkg, s, fset)
		}
	}
}

// analyzeTypeSpec analyzes a type specification
func (w *WorkspaceDetector) analyzeTypeSpec(pkg *PackageInfo, spec *ast.TypeSpec, fset *token.FileSet) {
	if spec.Name == nil {
		return
	}

	switch t := spec.Type.(type) {
	case *ast.StructType:
		w.analyzeStructType(pkg, spec.Name.Name, t, fset)
	case *ast.InterfaceType:
		w.analyzeInterfaceType(pkg, spec.Name.Name, t, fset)
	}
}

// analyzeStructType analyzes a struct type
func (w *WorkspaceDetector) analyzeStructType(pkg *PackageInfo, name string, structType *ast.StructType, fset *token.FileSet) {
	structInfo := StructInfo{
		Name:       name,
		Package:    pkg.Name,
		IsExported: ast.IsExported(name),
	}

	if structType.Fields != nil {
		for _, field := range structType.Fields.List {
			fieldType := w.extractTypeString(field.Type)
			var tag string
			if field.Tag != nil {
				tag = field.Tag.Value
			}

			if len(field.Names) > 0 {
				for _, fieldName := range field.Names {
					fieldInfo := FieldInfo{
						Name:       fieldName.Name,
						Type:       fieldType,
						Tag:        tag,
						IsExported: ast.IsExported(fieldName.Name),
					}
					structInfo.Fields = append(structInfo.Fields, fieldInfo)
				}
			} else {
				// Anonymous field
				fieldInfo := FieldInfo{
					Name:       "",
					Type:       fieldType,
					Tag:        tag,
					IsExported: true, // Anonymous fields are always embedded
				}
				structInfo.Fields = append(structInfo.Fields, fieldInfo)
			}
		}
	}

	pkg.Structs = append(pkg.Structs, structInfo)
}

// analyzeInterfaceType analyzes an interface type
func (w *WorkspaceDetector) analyzeInterfaceType(pkg *PackageInfo, name string, interfaceType *ast.InterfaceType, fset *token.FileSet) {
	interfaceInfo := InterfaceInfo{
		Name:       name,
		Package:    pkg.Name,
		IsExported: ast.IsExported(name),
	}

	if interfaceType.Methods != nil {
		for _, method := range interfaceType.Methods.List {
			if len(method.Names) > 0 {
				for _, methodName := range method.Names {
					methodInfo := MethodInfo{
						Name:       methodName.Name,
						IsExported: ast.IsExported(methodName.Name),
					}

					if funcType, ok := method.Type.(*ast.FuncType); ok {
						// Analyze parameters
						if funcType.Params != nil {
							for _, param := range funcType.Params.List {
								paramType := w.extractTypeString(param.Type)
								methodInfo.Parameters = append(methodInfo.Parameters, paramType)
							}
						}

						// Analyze return types
						if funcType.Results != nil {
							for _, result := range funcType.Results.List {
								returnType := w.extractTypeString(result.Type)
								methodInfo.Returns = append(methodInfo.Returns, returnType)
							}
						}
					}

					interfaceInfo.Methods = append(interfaceInfo.Methods, methodInfo)
				}
			}
		}
	}

	pkg.Interfaces = append(pkg.Interfaces, interfaceInfo)
}

// extractTypeString extracts a string representation of a type
func (w *WorkspaceDetector) extractTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + w.extractTypeString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + w.extractTypeString(t.Elt)
		}
		return "[...]" + w.extractTypeString(t.Elt)
	case *ast.MapType:
		return "map[" + w.extractTypeString(t.Key) + "]" + w.extractTypeString(t.Value)
	case *ast.ChanType:
		dir := ""
		if t.Dir == ast.SEND {
			dir = "chan<- "
		} else if t.Dir == ast.RECV {
			dir = "<-chan "
		} else {
			dir = "chan "
		}
		return dir + w.extractTypeString(t.Value)
	case *ast.FuncType:
		return "func"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.SelectorExpr:
		return w.extractTypeString(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

// calculateComplexity calculates the cyclomatic complexity of a function
func (w *WorkspaceDetector) calculateComplexity(fn *ast.FuncDecl) int {
	complexity := 1 // Base complexity

	ast.Inspect(fn, func(n ast.Node) bool {
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

// analyzeDependencies analyzes module dependencies
func (w *WorkspaceDetector) analyzeDependencies(workspace *WorkspaceInfo) error {
	// This is a simplified implementation
	// In a real implementation, you might want to use go list or go mod commands
	return nil
}

// getGitInfo retrieves Git repository information
func (w *WorkspaceDetector) getGitInfo(rootPath string) (*GitInfo, error) {
	gitInfo := &GitInfo{}

	// Check if it's a Git repository
	gitDir := filepath.Join(rootPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		gitInfo.IsRepo = false
		return gitInfo, nil
	}
	gitInfo.IsRepo = true

	// Get current branch
	if branch, err := w.runGitCommand(rootPath, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		gitInfo.Branch = strings.TrimSpace(branch)
	}

	// Get commit hash
	if hash, err := w.runGitCommand(rootPath, "rev-parse", "HEAD"); err == nil {
		gitInfo.CommitHash = strings.TrimSpace(hash)
	}

	// Get commit message
	if msg, err := w.runGitCommand(rootPath, "log", "-1", "--pretty=format:%s"); err == nil {
		gitInfo.CommitMessage = strings.TrimSpace(msg)
	}

	// Get commit author
	if author, err := w.runGitCommand(rootPath, "log", "-1", "--pretty=format:%an"); err == nil {
		gitInfo.CommitAuthor = strings.TrimSpace(author)
	}

	// Check if repository is dirty
	if status, err := w.runGitCommand(rootPath, "status", "--porcelain"); err == nil {
		gitInfo.IsDirty = strings.TrimSpace(status) != ""
	}

	// Get remote URL
	if url, err := w.runGitCommand(rootPath, "remote", "get-url", "origin"); err == nil {
		gitInfo.RemoteURL = strings.TrimSpace(url)
	}

	return gitInfo, nil
}

// runGitCommand runs a git command and returns the output
func (w *WorkspaceDetector) runGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	return string(output), err
}

// analyzeTestCoverage analyzes test coverage
func (w *WorkspaceDetector) analyzeTestCoverage(ctx context.Context, rootPath string) (*TestCoverageInfo, error) {
	// Run go test with coverage
	cmd := exec.CommandContext(ctx, "go", "test", "-coverprofile=coverage.out", "./...")
	cmd.Dir = rootPath

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	// Parse coverage file
	coveragePath := filepath.Join(rootPath, "coverage.out")
	defer os.Remove(coveragePath) // Clean up

	file, err := os.Open(coveragePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	coverage := &TestCoverageInfo{
		PackageCoverage: make(map[string]float64),
		LastRun:         time.Now(),
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			continue
		}

		// Parse coverage line
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			if count, err := strconv.Atoi(parts[2]); err == nil {
				coverage.TotalLines++
				if count > 0 {
					coverage.CoveredLines++
				}
			}
		}
	}

	if coverage.TotalLines > 0 {
		coverage.Percentage = float64(coverage.CoveredLines) / float64(coverage.TotalLines) * 100
	}

	return coverage, nil
}

// getBuildInfo gets build information
func (w *WorkspaceDetector) getBuildInfo(ctx context.Context, rootPath string) (*BuildInfo, error) {
	// This is a simplified implementation
	buildInfo := &BuildInfo{
		GoVersion:   "unknown",
		Platform:    "unknown",
		LastBuilt:   time.Now(),
		Environment: make(map[string]string),
	}

	// Get Go version
	if output, err := exec.CommandContext(ctx, "go", "version").Output(); err == nil {
		if matches := regexp.MustCompile(`go(\d+\.\d+(?:\.\d+)?)`).FindStringSubmatch(string(output)); len(matches) > 1 {
			buildInfo.GoVersion = matches[1]
		}
	}

	return buildInfo, nil
}

// analyzeIssues analyzes potential issues in the workspace
func (w *WorkspaceDetector) analyzeIssues(workspace *WorkspaceInfo) []Issue {
	var issues []Issue

	// Check for high complexity functions
	for _, pkg := range workspace.Packages {
		for _, fn := range pkg.Functions {
			if fn.Complexity > 10 {
				issues = append(issues, Issue{
					Type:       "complexity",
					Severity:   "warning",
					File:       pkg.Path,
					Line:       fn.LineStart,
					Message:    fmt.Sprintf("Function %s has high complexity (%d)", fn.Name, fn.Complexity),
					Rule:       "max-complexity",
					Suggestion: "Consider breaking this function into smaller functions",
				})
			}
		}
	}

	// Check for missing tests
	for _, pkg := range workspace.Packages {
		if pkg.IsMain {
			continue // Skip main packages
		}

		if len(pkg.TestFiles) == 0 {
			issues = append(issues, Issue{
				Type:       "testing",
				Severity:   "info",
				File:       pkg.Path,
				Message:    fmt.Sprintf("Package %s has no test files", pkg.Name),
				Rule:       "missing-tests",
				Suggestion: "Add test files to improve code quality and reliability",
			})
		}
	}

	return issues
}

// generateSuggestions generates improvement suggestions
func (w *WorkspaceDetector) generateSuggestions(workspace *WorkspaceInfo) []Suggestion {
	var suggestions []Suggestion

	// Suggest Go version upgrade if needed
	if workspace.GoVersion != "" && workspace.GoVersion < "1.23" {
		suggestions = append(suggestions, Suggestion{
			Type:        "upgrade",
			Priority:    "medium",
			Title:       "Consider upgrading Go version",
			Description: fmt.Sprintf("Current Go version is %s. Consider upgrading to Go 1.24+ for better performance and features.", workspace.GoVersion),
		})
	}

	// Suggest adding documentation
	hasReadme := false
	if _, err := os.Stat(filepath.Join(workspace.RootPath, "README.md")); err == nil {
		hasReadme = true
	}

	if !hasReadme {
		suggestions = append(suggestions, Suggestion{
			Type:        "documentation",
			Priority:    "medium",
			Title:       "Add README.md",
			Description: "Consider adding a README.md file to document your project",
		})
	}

	// Suggest CI/CD setup
	hasCICD := false
	cicdPaths := []string{".github/workflows", ".gitlab-ci.yml", ".travis.yml", "Jenkinsfile"}
	for _, path := range cicdPaths {
		if _, err := os.Stat(filepath.Join(workspace.RootPath, path)); err == nil {
			hasCICD = true
			break
		}
	}

	if !hasCICD {
		suggestions = append(suggestions, Suggestion{
			Type:        "automation",
			Priority:    "low",
			Title:       "Set up CI/CD pipeline",
			Description: "Consider setting up continuous integration and deployment",
		})
	}

	return suggestions
}

// calculateMetrics calculates various code metrics
func (w *WorkspaceDetector) calculateMetrics(workspace *WorkspaceInfo) *Metrics {
	metrics := &Metrics{
		TotalPackages: len(workspace.Packages),
	}

	var totalComplexity int
	var maxComplexity int

	for _, pkg := range workspace.Packages {
		metrics.TotalLines += pkg.LineCount
		metrics.TotalFiles += pkg.FileCount
		metrics.TotalFunctions += len(pkg.Functions)
		metrics.TotalStructs += len(pkg.Structs)
		metrics.TotalInterfaces += len(pkg.Interfaces)

		for _, fn := range pkg.Functions {
			totalComplexity += fn.Complexity
			if fn.Complexity > maxComplexity {
				maxComplexity = fn.Complexity
			}
		}
	}

	if metrics.TotalFunctions > 0 {
		metrics.AvgComplexity = float64(totalComplexity) / float64(metrics.TotalFunctions)
	}
	metrics.MaxComplexity = maxComplexity

	if workspace.TestCoverage != nil {
		metrics.TestCoverage = workspace.TestCoverage.Percentage
	}

	return metrics
}
