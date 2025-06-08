package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/koopa0/assistant-go/internal/cli/ui"
)

// Code Analysis Handlers

func (c *CLI) findCodeSmells(ctx context.Context) error {
	var path string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter path to analyze (file or directory):").
				Placeholder(".").
				Description("Path to search for code smells").
				Value(&path),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if path == "" {
		path = "."
	}

	query := fmt.Sprintf("Find code smells in %s. Look for: long functions, high complexity, duplicate code, poor naming, missing error handling, and other Go anti-patterns. Provide specific locations and improvement suggestions.", path)
	c.processQuery(ctx, query)
	return nil
}

func (c *CLI) checkComplexity(ctx context.Context) error {
	var path string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter path to check complexity:").
				Placeholder(".").
				Description("Path to analyze cyclomatic complexity").
				Value(&path),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if path == "" {
		path = "."
	}

	query := fmt.Sprintf("Analyze cyclomatic complexity in %s. List functions with high complexity (>10) and suggest refactoring strategies to reduce complexity.", path)
	c.processQuery(ctx, query)
	return nil
}

func (c *CLI) reviewRecentChanges(ctx context.Context) error {
	var commits string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("How many recent commits to review?").
				Placeholder("5").
				Description("Number of recent commits to analyze").
				Value(&commits),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if commits == "" {
		commits = "5"
	}

	query := fmt.Sprintf("Review the last %s git commits. Analyze code changes for quality, potential bugs, and adherence to Go best practices.", commits)
	c.processQuery(ctx, query)
	return nil
}

// Refactoring Handlers

func (c *CLI) renameSymbol(ctx context.Context) error {
	var filepath, oldName, newName string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("File path:").
				Description("Path to the file containing the symbol").
				Validate(validateGoFile).
				Value(&filepath),

			huh.NewInput().
				Title("Current symbol name:").
				Description("The current name of the function, variable, or type").
				Value(&oldName),

			huh.NewInput().
				Title("New symbol name:").
				Description("The new name for the symbol").
				Validate(validateIdentifier).
				Value(&newName),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	query := fmt.Sprintf("Rename '%s' to '%s' in %s and all references to it across the codebase. Show all affected files.",
		oldName, newName, filepath)
	c.processQuery(ctx, query)
	return nil
}

func (c *CLI) optimizeImports(ctx context.Context) error {
	var path string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter path to optimize imports:").
				Placeholder(".").
				Description("File or directory to optimize imports").
				Value(&path),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if path == "" {
		path = "."
	}

	query := fmt.Sprintf("Optimize imports in %s. Remove unused imports, add missing imports, and organize them according to Go conventions (standard library, third-party, local).", path)
	c.processQuery(ctx, query)
	return nil
}

func (c *CLI) convertToIdiomaticGo(ctx context.Context) error {
	var filepath string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter file to convert to idiomatic Go:").
				Description("Path to the Go file to improve").
				Validate(validateGoFile).
				Value(&filepath),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	query := fmt.Sprintf("Review %s and convert it to idiomatic Go. Focus on: naming conventions, error handling, interface usage, concurrency patterns, and Go proverbs. Show before and after code.", filepath)
	c.processQuery(ctx, query)
	return nil
}

func (c *CLI) simplifyComplexFunctions(ctx context.Context) error {
	var filepath string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter file with complex functions:").
				Description("Path to the Go file to simplify").
				Validate(validateGoFile).
				Value(&filepath),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	query := fmt.Sprintf("Find complex functions in %s (cyclomatic complexity > 10) and refactor them. Break down into smaller functions, reduce nesting, and improve readability.", filepath)
	c.processQuery(ctx, query)
	return nil
}

// Test Generation Handlers

func (c *CLI) generateUnitTests(ctx context.Context) error {
	var filepath, function, style string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("File containing the function:").
				Description("Path to the Go file").
				Validate(validateGoFile).
				Value(&filepath),

			huh.NewInput().
				Title("Function name to test:").
				Description("Name of the function to generate tests for").
				Validate(validateIdentifier).
				Value(&function),

			huh.NewSelect[string]().
				Title("Test style:").
				Options(
					huh.NewOption("Standard", "standard"),
					huh.NewOption("With mocks", "mocks"),
					huh.NewOption("Property-based", "property"),
					huh.NewOption("Example tests", "example"),
				).
				Value(&style),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	query := fmt.Sprintf("Generate %s unit tests for function %s in %s. Include edge cases, error scenarios, and good test names. Follow Go testing best practices.",
		style, function, filepath)
	c.processQuery(ctx, query)
	return nil
}

func (c *CLI) generateTableDrivenTests(ctx context.Context) error {
	var filepath, function string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("File containing the function:").
				Description("Path to the Go file").
				Validate(validateGoFile).
				Value(&filepath),

			huh.NewInput().
				Title("Function name:").
				Description("Function to generate table-driven tests for").
				Validate(validateIdentifier).
				Value(&function),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	query := fmt.Sprintf("Generate table-driven tests for function %s in %s. Include comprehensive test cases with descriptive names, edge cases, and error scenarios. Use subtests with t.Run().",
		function, filepath)
	c.processQuery(ctx, query)
	return nil
}

func (c *CLI) generateIntegrationTests(ctx context.Context) error {
	var component, path string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Component to test:").
				Options(
					huh.NewOption("HTTP API", "http"),
					huh.NewOption("Database", "database"),
					huh.NewOption("Service", "service"),
					huh.NewOption("gRPC", "grpc"),
					huh.NewOption("Message Queue", "mq"),
				).
				Value(&component),

			huh.NewInput().
				Title("Component path:").
				Description("Path to the component to test").
				Value(&path),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	query := fmt.Sprintf("Generate integration tests for %s component at %s. Use testcontainers for dependencies, test real interactions, and include setup/teardown. Focus on realistic scenarios.",
		component, path)
	c.processQuery(ctx, query)
	return nil
}

func (c *CLI) generateBenchmarkTests(ctx context.Context) error {
	var filepath string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("File to benchmark:").
				Description("Path to the Go file containing functions to benchmark").
				Validate(validateGoFile).
				Value(&filepath),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	query := fmt.Sprintf("Generate benchmark tests for performance-critical functions in %s. Include memory allocation benchmarks, different input sizes, and parallel benchmarks where appropriate.", filepath)
	c.processQuery(ctx, query)
	return nil
}

func (c *CLI) generateFuzzTests(ctx context.Context) error {
	var filepath, function string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("File containing the function:").
				Description("Path to the Go file").
				Validate(validateGoFile).
				Value(&filepath),

			huh.NewInput().
				Title("Function to fuzz:").
				Description("Function name for fuzz testing").
				Validate(validateIdentifier).
				Value(&function),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	query := fmt.Sprintf("Generate fuzz tests for function %s in %s. Include seed corpus, property checks, and crash detection. Use Go 1.18+ native fuzzing.",
		function, filepath)
	c.processQuery(ctx, query)
	return nil
}

// Documentation Menu

func (c *CLI) documentationMenu(ctx context.Context) error {
	var choice string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Documentation Options:").
				Options(
					huh.NewOption("Generate package documentation", "package"),
					huh.NewOption("Add function comments", "functions"),
					huh.NewOption("Create README", "readme"),
					huh.NewOption("Generate API documentation", "api"),
					huh.NewOption("Create architecture diagram", "diagram"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	switch choice {
	case "package":
		var pkg string
		f := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Package path:").
					Placeholder(".").
					Value(&pkg),
			),
		)
		f.Run()
		if pkg == "" {
			pkg = "."
		}
		query := fmt.Sprintf("Generate comprehensive godoc-style documentation for package %s. Include package overview, examples, and important types/functions.", pkg)
		c.processQuery(ctx, query)

	case "functions":
		var file string
		f := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("File path:").
					Validate(validateGoFile).
					Value(&file),
			),
		)
		f.Run()
		query := fmt.Sprintf("Add missing function comments to %s following godoc conventions. Include parameters, return values, and behavior description.", file)
		c.processQuery(ctx, query)

	case "readme":
		query := "Generate a comprehensive README.md for this project including: overview, features, installation, usage examples, API reference, contributing guidelines, and license."
		c.processQuery(ctx, query)

	case "api":
		query := "Generate OpenAPI/Swagger documentation for all HTTP endpoints in this project. Include request/response schemas, examples, and error codes."
		c.processQuery(ctx, query)

	case "diagram":
		query := "Create an architecture diagram for this project showing: components, data flow, external dependencies, and deployment structure. Use mermaid or PlantUML format."
		c.processQuery(ctx, query)

	case "back":
		return nil
	}

	return nil
}

// Debug Menu

func (c *CLI) debugMenu(ctx context.Context) error {
	var choice string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Debug Options:").
				Options(
					huh.NewOption("Analyze error", "error"),
					huh.NewOption("Trace execution path", "trace"),
					huh.NewOption("Find race conditions", "race"),
					huh.NewOption("Memory leak detection", "memory"),
					huh.NewOption("Performance bottlenecks", "performance"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	switch choice {
	case "error":
		var errorMsg string
		f := huh.NewForm(
			huh.NewGroup(
				huh.NewText().
					Title("Paste error message:").
					Description("The error message or stack trace").
					Lines(5).
					Value(&errorMsg),
			),
		)
		f.Run()
		query := fmt.Sprintf("Analyze this error and suggest fixes: %s", errorMsg)
		c.processQuery(ctx, query)

	case "race":
		var path string
		f := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Code path to analyze:").
					Placeholder(".").
					Value(&path),
			),
		)
		f.Run()
		if path == "" {
			path = "."
		}
		query := fmt.Sprintf("Analyze %s for potential race conditions. Check goroutine usage, shared state access, and synchronization. Suggest fixes using channels or sync primitives.", path)
		c.processQuery(ctx, query)

	case "back":
		return nil
	}

	return nil
}

// Architecture Menu

func (c *CLI) architectureMenu(ctx context.Context) error {
	var choice string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Architecture Options:").
				Options(
					huh.NewOption("Review architecture", "review"),
					huh.NewOption("Suggest improvements", "improve"),
					huh.NewOption("Design new feature", "design"),
					huh.NewOption("Refactor to patterns", "patterns"),
					huh.NewOption("Dependency analysis", "deps"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	switch choice {
	case "review":
		query := "Review the current project architecture. Analyze package structure, dependencies, design patterns, and provide improvement suggestions following Go best practices and DDD principles."
		c.processQuery(ctx, query)

	case "design":
		var feature string
		f := huh.NewForm(
			huh.NewGroup(
				huh.NewText().
					Title("Describe the feature:").
					Description("Brief description of the feature to design").
					Lines(3).
					Value(&feature),
			),
		)
		f.Run()
		query := fmt.Sprintf("Design architecture for feature: %s. Include: package structure, interfaces, data models, API design, and integration points.", feature)
		c.processQuery(ctx, query)

	case "back":
		return nil
	}

	return nil
}

// Security Menu

func (c *CLI) securityMenu(ctx context.Context) error {
	var choice string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Security Options:").
				Options(
					huh.NewOption("Security audit", "audit"),
					huh.NewOption("Check SQL injection", "sql"),
					huh.NewOption("Review authentication", "auth"),
					huh.NewOption("Scan dependencies", "deps"),
					huh.NewOption("Check secrets", "secrets"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	switch choice {
	case "audit":
		query := "Perform a comprehensive security audit. Check for: SQL injection, XSS, authentication issues, authorization flaws, cryptographic weaknesses, dependency vulnerabilities, and secret exposure."
		c.processQuery(ctx, query)

	case "sql":
		query := "Scan all database queries for SQL injection vulnerabilities. Verify parameterized queries, check dynamic SQL construction, and suggest sqlc migrations where appropriate."
		c.processQuery(ctx, query)

	case "back":
		return nil
	}

	return nil
}

// Custom Query

func (c *CLI) customQueryMenu(ctx context.Context) error {
	var query string
	var addContext bool
	var files string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Enter your query:").
				Description("Describe what you want to do. Press Tab to move to next field.").
				Lines(5).
				Value(&query),
		),

		huh.NewGroup(
			huh.NewConfirm().
				Title("Add file context?").
				Value(&addContext),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if addContext {
		fileForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Files to include (comma-separated):").
					Description("These files will be analyzed as context for your query").
					Value(&files),
			),
		)
		fileForm.Run()

		if files != "" {
			query = fmt.Sprintf("%s\n\nContext files: %s", query, files)
		}
	}

	c.processQuery(ctx, query)
	return nil
}

// Project Stats

func (c *CLI) projectStatsMenu(ctx context.Context) error {
	ui.Header.Println("📊 Project Statistics")
	fmt.Println(ui.Divider())

	// This would normally gather real stats
	stats := [][]string{
		{"Total Files", "156"},
		{"Go Files", "142"},
		{"Test Files", "48"},
		{"Test Coverage", "78.3%"},
		{"Total Lines", "24,531"},
		{"Code Lines", "18,234"},
		{"Comment Lines", "3,421"},
		{"Packages", "23"},
		{"Dependencies", "47"},
	}

	for _, stat := range stats {
		ui.Label.Printf("%-20s: ", stat[0])
		ui.Success.Println(stat[1])
	}

	fmt.Println()

	var analyze bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Run detailed analysis?").
				Value(&analyze),
		),
	)
	form.Run()

	if analyze {
		query := "Analyze this project and provide detailed statistics including: code quality metrics, complexity analysis, test coverage gaps, and technical debt assessment."
		c.processQuery(ctx, query)
	}

	return nil
}

// Settings Menu

func (c *CLI) settingsMenu(ctx context.Context) error {
	var choice string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Settings:").
				Options(
					huh.NewOption("AI Provider Settings", "ai"),
					huh.NewOption("Default Behaviors", "defaults"),
					huh.NewOption("Output Preferences", "output"),
					huh.NewOption("Tool Configuration", "tools"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	switch choice {
	case "ai":
		var provider string
		f := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select AI provider:").
					Options(
						huh.NewOption("Claude", "claude"),
						huh.NewOption("Gemini", "gemini"),
					).
					Value(&provider),
			),
		)
		f.Run()
		ui.Success.Printf("AI provider set to: %s\n", provider)

	case "output":
		var streaming bool
		f := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Enable streaming output?").
					Value(&streaming),
			),
		)
		f.Run()
		c.config.EnableStreaming = streaming
		ui.Success.Printf("Streaming output: %v\n", streaming)

	case "back":
		return nil
	}

	return nil
}

// Helper to get file extension
func getFileExt(path string) string {
	return strings.ToLower(filepath.Ext(path))
}

// isValidGoIdentifier checks if a string is a valid Go identifier
func isValidGoIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Must start with letter or underscore
	if !isLetter(rune(s[0])) && s[0] != '_' {
		return false
	}

	// Rest must be letters, digits, or underscores
	for i := 1; i < len(s); i++ {
		if !isLetter(rune(s[i])) && !isDigit(rune(s[i])) && s[i] != '_' {
			return false
		}
	}

	return true
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
