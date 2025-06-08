package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/koopa0/assistant-go/internal/cli/ui"
)

// showMainMenu shows an interactive task selection menu using huh
func (c *CLI) showMainMenu(ctx context.Context) error {
	for {
		var choice string

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("What would you like to do?").
					Options(
						huh.NewOption("🔍 Code Analysis & Review", "analysis"),
						huh.NewOption("🔧 Refactoring & Optimization", "refactor"),
						huh.NewOption("🧪 Generate Tests", "tests"),
						huh.NewOption("📝 Documentation", "docs"),
						huh.NewOption("🐛 Debug & Fix Issues", "debug"),
						huh.NewOption("🏗️ Architecture & Design", "architecture"),
						huh.NewOption("🔐 Security Audit", "security"),
						huh.NewOption("🎯 Custom Query", "custom"),
						huh.NewOption("📊 View Project Stats", "stats"),
						huh.NewOption("⚙️ Settings", "settings"),
						huh.NewOption("❌ Exit Menu", "exit"),
					).
					Value(&choice),
			),
		)

		if err := form.Run(); err != nil {
			return err
		}

		switch choice {
		case "analysis":
			if err := c.codeAnalysisMenu(ctx); err != nil {
				return err
			}
		case "refactor":
			if err := c.refactoringMenu(ctx); err != nil {
				return err
			}
		case "tests":
			if err := c.testGenerationMenu(ctx); err != nil {
				return err
			}
		case "docs":
			if err := c.documentationMenu(ctx); err != nil {
				return err
			}
		case "debug":
			if err := c.debugMenu(ctx); err != nil {
				return err
			}
		case "architecture":
			if err := c.architectureMenu(ctx); err != nil {
				return err
			}
		case "security":
			if err := c.securityMenu(ctx); err != nil {
				return err
			}
		case "custom":
			if err := c.customQueryMenu(ctx); err != nil {
				return err
			}
		case "stats":
			if err := c.projectStatsMenu(ctx); err != nil {
				return err
			}
		case "settings":
			if err := c.settingsMenu(ctx); err != nil {
				return err
			}
		case "exit":
			return nil
		}
	}
}

// codeAnalysisMenu shows code analysis options
func (c *CLI) codeAnalysisMenu(ctx context.Context) error {
	var choice string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Code Analysis Options:").
				Options(
					huh.NewOption("Analyze current file", "current"),
					huh.NewOption("Analyze directory", "directory"),
					huh.NewOption("Find code smells", "smells"),
					huh.NewOption("Check complexity", "complexity"),
					huh.NewOption("Review recent changes", "changes"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	switch choice {
	case "current":
		return c.analyzeCurrentFile(ctx)
	case "directory":
		return c.analyzeDirectory(ctx)
	case "smells":
		return c.findCodeSmells(ctx)
	case "complexity":
		return c.checkComplexity(ctx)
	case "changes":
		return c.reviewRecentChanges(ctx)
	case "back":
		return nil
	}

	return nil
}

// refactoringMenu shows refactoring options
func (c *CLI) refactoringMenu(ctx context.Context) error {
	var choice string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Refactoring Options:").
				Options(
					huh.NewOption("Extract function/method", "extract"),
					huh.NewOption("Rename symbol", "rename"),
					huh.NewOption("Optimize imports", "imports"),
					huh.NewOption("Convert to idiomatic Go", "idiomatic"),
					huh.NewOption("Simplify complex functions", "simplify"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	switch choice {
	case "extract":
		return c.extractFunction(ctx)
	case "rename":
		return c.renameSymbol(ctx)
	case "imports":
		return c.optimizeImports(ctx)
	case "idiomatic":
		return c.convertToIdiomaticGo(ctx)
	case "simplify":
		return c.simplifyComplexFunctions(ctx)
	case "back":
		return nil
	}

	return nil
}

// testGenerationMenu shows test generation options
func (c *CLI) testGenerationMenu(ctx context.Context) error {
	var choice string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Test Generation Options:").
				Options(
					huh.NewOption("Generate unit tests for function", "unit"),
					huh.NewOption("Generate table-driven tests", "table"),
					huh.NewOption("Generate integration tests", "integration"),
					huh.NewOption("Generate benchmark tests", "benchmark"),
					huh.NewOption("Generate fuzz tests", "fuzz"),
					huh.NewOption("← Back", "back"),
				).
				Value(&choice),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	switch choice {
	case "unit":
		return c.generateUnitTests(ctx)
	case "table":
		return c.generateTableDrivenTests(ctx)
	case "integration":
		return c.generateIntegrationTests(ctx)
	case "benchmark":
		return c.generateBenchmarkTests(ctx)
	case "fuzz":
		return c.generateFuzzTests(ctx)
	case "back":
		return nil
	}

	return nil
}

// Interactive helper functions using huh

func (c *CLI) analyzeCurrentFile(ctx context.Context) error {
	var filepath string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter file path to analyze:").
				Placeholder("path/to/file.go").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("file path cannot be empty")
					}
					if !strings.HasSuffix(s, ".go") {
						return fmt.Errorf("file must be a Go file (.go)")
					}
					return nil
				}).
				Value(&filepath),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	query := fmt.Sprintf("Analyze this Go file and provide detailed feedback on code quality, potential issues, and suggestions for improvement: %s", filepath)
	c.processQuery(ctx, query)
	return nil
}

func (c *CLI) analyzeDirectory(ctx context.Context) error {
	var dirpath string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter directory path:").
				Placeholder(".").
				Value(&dirpath),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if dirpath == "" {
		dirpath = "."
	}

	query := fmt.Sprintf("Analyze all Go files in directory %s and provide a comprehensive code review with suggestions", dirpath)
	c.processQuery(ctx, query)
	return nil
}

func (c *CLI) extractFunction(ctx context.Context) error {
	var filepath, startLine, endLine, funcName string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("File path:").
				Description("Path to the file containing the code to extract").
				Validate(validateGoFile).
				Value(&filepath),

			huh.NewInput().
				Title("Start line number:").
				Description("Line number where the code to extract begins").
				Validate(validateLineNumber).
				Value(&startLine),

			huh.NewInput().
				Title("End line number:").
				Description("Line number where the code to extract ends").
				Validate(validateLineNumber).
				Value(&endLine),

			huh.NewInput().
				Title("New function name:").
				Description("Name for the extracted function").
				Validate(validateIdentifier).
				Value(&funcName),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	query := fmt.Sprintf("Extract lines %s-%s from %s into a new function called %s. Show the refactored code.",
		startLine, endLine, filepath, funcName)
	c.processQuery(ctx, query)
	return nil
}

// Workflow examples using huh

func (c *CLI) interactiveCodeReview(ctx context.Context) error {
	ui.Header.Println("Interactive Code Review")
	fmt.Println(ui.Divider())

	var scope string
	var reviewTypes []string
	var severity string

	// Multi-step form
	form := huh.NewForm(
		// Step 1: Select scope
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Review scope:").
				Options(
					huh.NewOption("Single file", "file"),
					huh.NewOption("Directory", "directory"),
					huh.NewOption("Recent changes", "changes"),
					huh.NewOption("Entire project", "project"),
				).
				Value(&scope),
		),

		// Step 2: Select review types
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("What to review?").
				Options(
					huh.NewOption("Code style and conventions", "style"),
					huh.NewOption("Performance issues", "performance"),
					huh.NewOption("Security vulnerabilities", "security"),
					huh.NewOption("Test coverage", "tests"),
					huh.NewOption("Documentation", "docs"),
					huh.NewOption("Error handling", "errors"),
					huh.NewOption("Concurrency issues", "concurrency"),
				).
				Value(&reviewTypes),
		),

		// Step 3: Severity level
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Issue severity to report:").
				Options(
					huh.NewOption("All issues", "all"),
					huh.NewOption("Medium and above", "medium"),
					huh.NewOption("High only", "high"),
					huh.NewOption("Critical only", "critical"),
				).
				Value(&severity),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	// Build query
	query := fmt.Sprintf("Perform a code review with scope: %s, checking for: %s, reporting %s",
		scope, strings.Join(reviewTypes, ", "), severity)

	c.processQuery(ctx, query)
	return nil
}

// Confirmation dialogs using huh

func (c *CLI) confirmAction(action string) bool {
	var confirm bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Are you sure you want to %s?", action)).
				Affirmative("Yes").
				Negative("No").
				Value(&confirm),
		),
	)

	if err := form.Run(); err != nil {
		return false
	}

	return confirm
}

// Advanced forms with validation

func (c *CLI) createNewFeature(ctx context.Context) error {
	var name, description, packagePath string
	var includeTests, includeAPI bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Feature name:").
				Placeholder("MyNewFeature").
				Validate(validateIdentifier).
				Value(&name),

			huh.NewText().
				Title("Description:").
				Placeholder("Describe what this feature does...").
				Lines(3).
				Value(&description),

			huh.NewInput().
				Title("Package path:").
				Placeholder("internal/features/mynewfeature").
				Description("Where to create the feature package").
				Value(&packagePath),
		),

		huh.NewGroup(
			huh.NewConfirm().
				Title("Include unit tests?").
				Value(&includeTests),

			huh.NewConfirm().
				Title("Include API endpoints?").
				Value(&includeAPI),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	query := fmt.Sprintf(`Create a new feature called "%s" with the following requirements:
Description: %s
Package: %s
Include tests: %v
Include API: %v

Generate the complete implementation following Go best practices.`,
		name, description, packagePath, includeTests, includeAPI)

	c.processQuery(ctx, query)
	return nil
}

// Validation functions

func validateGoFile(s string) error {
	if s == "" {
		return fmt.Errorf("file path cannot be empty")
	}
	if !strings.HasSuffix(s, ".go") {
		return fmt.Errorf("file must be a Go file (.go)")
	}
	return nil
}

func validateLineNumber(s string) error {
	if s == "" {
		return fmt.Errorf("line number cannot be empty")
	}
	// Simple check - could be enhanced
	for _, r := range s {
		if r < '0' || r > '9' {
			return fmt.Errorf("line number must contain only digits")
		}
	}
	return nil
}

func validateIdentifier(s string) error {
	if s == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if !isValidGoIdentifier(s) {
		return fmt.Errorf("must be a valid Go identifier")
	}
	return nil
}
