package godev

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGoTesterBasics tests basic tester functionality
func TestGoTesterBasics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tester, err := NewGoTester(nil, logger)
	if err != nil {
		t.Fatalf("NewGoTester() error = %v", err)
	}

	if tester.Name() != "go_tester" {
		t.Errorf("Name() = %s, want go_tester", tester.Name())
	}

	if tester.Description() == "" {
		t.Error("Description() should not be empty")
	}

	params := tester.Parameters()
	if params == nil {
		t.Error("Parameters() should not be nil")
	}

	// Check parameters structure
	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Error("Parameters should have properties")
	}

	expectedProps := []string{"path", "pattern", "coverage", "verbose", "benchmark"}
	for _, prop := range expectedProps {
		if _, exists := properties[prop]; !exists {
			t.Errorf("Parameters should include '%s' property", prop)
		}
	}

	// Check that no parameters are required (all have defaults)
	required, ok := params["required"].([]string)
	if !ok {
		t.Error("Parameters should have required array")
	}
	if len(required) != 0 {
		t.Error("No parameters should be required (all have defaults)")
	}
}

// TestGoTesterHealth tests health check
func TestGoTesterHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tester, _ := NewGoTester(nil, logger)

	ctx := context.Background()
	err := tester.Health(ctx)

	// Check if go command is available
	if _, lookErr := exec.LookPath("go"); lookErr != nil {
		// If go is not available, health check should fail
		if err == nil {
			t.Error("Health() should fail when go command is not available")
		}
		t.Skip("go command not available, skipping test")
	} else {
		// If go is available, health check should pass
		if err != nil {
			t.Errorf("Health() error = %v, want nil when go is available", err)
		}
	}
}

// TestGoTesterClose tests close functionality
func TestGoTesterClose(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tester, _ := NewGoTester(nil, logger)

	ctx := context.Background()
	err := tester.Close(ctx)
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

// TestGoTesterParameterParsing tests parameter parsing
func TestGoTesterParameterParsing(t *testing.T) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tester, _ := NewGoTester(nil, logger)

	// Create a simple test project
	tempDir := t.TempDir()
	createSimpleGoProject(t, tempDir)

	ctx := context.Background()

	tests := []struct {
		name             string
		input            map[string]interface{}
		expectSuccess    bool
		checkMetadata    bool
		expectedPath     string
		expectedCoverage bool
		expectedVerbose  bool
	}{
		{
			name:             "default_parameters",
			input:            map[string]interface{}{},
			expectSuccess:    true,
			checkMetadata:    true,
			expectedPath:     "./...",
			expectedCoverage: true,
			expectedVerbose:  false,
		},
		{
			name: "explicit_parameters",
			input: map[string]interface{}{
				"path":     tempDir,
				"pattern":  "TestExample",
				"coverage": false,
				"verbose":  true,
				"short":    true,
				"timeout":  "5s",
			},
			expectSuccess:    true,
			checkMetadata:    true,
			expectedPath:     tempDir,
			expectedCoverage: false,
			expectedVerbose:  true,
		},
		{
			name: "with_parallel",
			input: map[string]interface{}{
				"path":     tempDir,
				"parallel": 2,
			},
			expectSuccess: true,
		},
		{
			name: "with_benchmark",
			input: map[string]interface{}{
				"path":      tempDir,
				"benchmark": true,
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tester.Execute(ctx, tt.input)

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
				}
				if result == nil {
					t.Fatal("Execute() returned nil result")
				}

				// Check metadata
				if tt.checkMetadata && result.Metadata != nil {
					if path, ok := result.Metadata["path"].(string); ok {
						if path != tt.expectedPath {
							t.Errorf("Metadata path = %s, want %s", path, tt.expectedPath)
						}
					}
					if coverage, ok := result.Metadata["coverage"].(bool); ok {
						if coverage != tt.expectedCoverage {
							t.Errorf("Metadata coverage = %t, want %t", coverage, tt.expectedCoverage)
						}
					}
					if verbose, ok := result.Metadata["verbose"].(bool); ok {
						if verbose != tt.expectedVerbose {
							t.Errorf("Metadata verbose = %t, want %t", verbose, tt.expectedVerbose)
						}
					}
				}

				// Check execution time
				if result.ExecutionTime <= 0 {
					t.Error("Should track execution time")
				}

			} else {
				if err == nil {
					t.Error("Execute() error = nil, want error")
				}
			}
		})
	}
}

// TestGoTesterWithRealProject tests with actual Go project
func TestGoTesterWithRealProject(t *testing.T) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tester, _ := NewGoTester(nil, logger)

	tempDir := t.TempDir()

	// Create a more comprehensive test project
	projectFiles := map[string]string{
		"go.mod": `module testproject

go 1.21`,
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println(Add(2, 3))
}

func Add(a, b int) int {
	return a + b
}

func Subtract(a, b int) int {
	return a - b
}`,
		"main_test.go": `package main

import "testing"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("Add(2, 3) = %d, want 5", result)
	}
}

func TestSubtract(t *testing.T) {
	result := Subtract(5, 3)
	if result != 2 {
		t.Errorf("Subtract(5, 3) = %d, want 2", result)
	}
}

func BenchmarkAdd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Add(2, 3)
	}
}

func TestSkipped(t *testing.T) {
	t.Skip("This test is skipped")
}`,
		"utils/utils.go": `package utils

func Multiply(a, b int) int {
	return a * b
}`,
		"utils/utils_test.go": `package utils

import "testing"

func TestMultiply(t *testing.T) {
	result := Multiply(3, 4)
	if result != 12 {
		t.Errorf("Multiply(3, 4) = %d, want 12", result)
	}
}`,
	}

	for filePath, content := range projectFiles {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}

	// Initialize go module
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize go module: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name              string
		input             map[string]interface{}
		checkTests        bool
		checkCoverage     bool
		checkBenchmarks   bool
		expectedMinTests  int
		expectedMinPassed int
	}{
		{
			name: "run_all_tests_with_coverage",
			input: map[string]interface{}{
				"path":     tempDir,
				"coverage": true,
				"verbose":  true,
			},
			checkTests:        true,
			checkCoverage:     true,
			expectedMinTests:  3, // TestAdd, TestSubtract, TestMultiply, TestSkipped
			expectedMinPassed: 3, // TestAdd, TestSubtract, TestMultiply (TestSkipped is skipped)
		},
		{
			name: "run_specific_test_pattern",
			input: map[string]interface{}{
				"path":    tempDir,
				"pattern": "TestAdd",
			},
			checkTests:        true,
			expectedMinTests:  1,
			expectedMinPassed: 1,
		},
		{
			name: "run_benchmarks",
			input: map[string]interface{}{
				"path":      tempDir,
				"benchmark": true,
			},
			checkBenchmarks: true,
		},
		{
			name: "short_tests_only",
			input: map[string]interface{}{
				"path":  tempDir,
				"short": true,
			},
			checkTests: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tester.Execute(ctx, tt.input)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
			}
			if result == nil || !result.Success {
				t.Error("Execute() should return successful result")
			}

			// Parse result data
			testResult, ok := result.Data.(*TestResult)
			if !ok {
				t.Fatal("Execute() result data should be *TestResult")
			}

			// Check summary
			if testResult.Summary.TotalPackages == 0 {
				t.Error("Should find at least one package")
			}

			if tt.checkTests {
				if testResult.Summary.TotalTests < tt.expectedMinTests {
					t.Errorf("Total tests = %d, want >= %d",
						testResult.Summary.TotalTests, tt.expectedMinTests)
				}
				if testResult.Summary.PassedTests < tt.expectedMinPassed {
					t.Errorf("Passed tests = %d, want >= %d",
						testResult.Summary.PassedTests, tt.expectedMinPassed)
				}
				if testResult.Summary.SkippedTests < 1 {
					t.Error("Should have at least one skipped test")
				}
			}

			if tt.checkCoverage {
				if testResult.Coverage == nil {
					t.Error("Should have coverage report")
				} else {
					if testResult.Coverage.TotalCoverage <= 0 {
						t.Error("Should have non-zero coverage")
					}
				}
			}

			if tt.checkBenchmarks {
				if len(testResult.Benchmarks) == 0 {
					t.Error("Should find benchmark results")
				}
			}

			// Check output
			if testResult.Output == "" {
				t.Error("Should capture test output")
			}

			// Check execution time
			if testResult.ExecutionTime <= 0 {
				t.Error("Should track execution time")
			}
		})
	}
}

// TestGoTesterFailures tests handling of test failures
func TestGoTesterFailures(t *testing.T) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tester, _ := NewGoTester(nil, logger)

	tempDir := t.TempDir()

	// Create project with failing tests
	projectFiles := map[string]string{
		"go.mod": `module testproject

go 1.21`,
		"main.go": `package main

func Add(a, b int) int {
	return a + b
}`,
		"main_test.go": `package main

import "testing"

func TestAddPass(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("Add(2, 3) = %d, want 5", result)
	}
}

func TestAddFail(t *testing.T) {
	result := Add(2, 3)
	if result != 6 { // This will fail
		t.Errorf("Add(2, 3) = %d, want 6", result)
	}
}`,
	}

	for filePath, content := range projectFiles {
		fullPath := filepath.Join(tempDir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}

	ctx := context.Background()
	input := map[string]interface{}{
		"path":    tempDir,
		"verbose": true,
	}

	result, err := tester.Execute(ctx, input)

	// Should not error out completely, but should report test failures
	if err != nil {
		t.Errorf("Execute() should handle test failures gracefully, got: %v", err)
	}
	if result == nil {
		t.Fatal("Execute() should return result even with test failures")
	}

	testResult := result.Data.(*TestResult)

	// Should have failed tests
	if testResult.Summary.FailedTests == 0 {
		t.Error("Should report failed tests")
	}
	if testResult.Summary.PassedTests == 0 {
		t.Error("Should also report passed tests")
	}
	if testResult.Summary.Success {
		t.Error("Summary should indicate failure when tests fail")
	}

	// Should have detailed failure information
	if len(testResult.FailedTests) == 0 {
		t.Error("Should provide details about failed tests")
	}

	for _, failedTest := range testResult.FailedTests {
		if failedTest.Status != TestStatusFail {
			t.Errorf("Failed test status = %s, want %s", failedTest.Status, TestStatusFail)
		}
		if failedTest.Name == "" {
			t.Error("Failed test should have name")
		}
		if failedTest.Package == "" {
			t.Error("Failed test should have package")
		}
	}
}

// TestGoTesterContextCancellation tests context cancellation
func TestGoTesterContextCancellation(t *testing.T) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tester, _ := NewGoTester(nil, logger)

	tempDir := t.TempDir()
	createSimpleGoProject(t, tempDir)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	input := map[string]interface{}{
		"path": tempDir,
	}

	_, err := tester.Execute(ctx, input)
	if err == nil {
		t.Error("Execute() with cancelled context should return error")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("Error should mention context cancellation, got: %v", err)
	}
}

// TestGoTesterInvalidProject tests with invalid Go project
func TestGoTesterInvalidProject(t *testing.T) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tester, _ := NewGoTester(nil, logger)

	ctx := context.Background()

	tests := []struct {
		name      string
		input     map[string]interface{}
		setupFunc func() string
	}{
		{
			name: "nonexistent_path",
			input: map[string]interface{}{
				"path": "/nonexistent/path",
			},
		},
		{
			name: "invalid_timeout",
			setupFunc: func() string {
				tempDir := t.TempDir()
				createSimpleGoProject(t, tempDir)
				return tempDir
			},
			input: map[string]interface{}{
				"timeout": "invalid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				path := tt.setupFunc()
				if tt.input["path"] == nil {
					tt.input["path"] = path
				}
			}

			result, err := tester.Execute(ctx, tt.input)

			// Should handle errors gracefully
			if err == nil {
				// If no error, check if result indicates failure
				if result == nil {
					t.Error("Execute() should return result or error")
				}
				if result.Success {
					t.Error("Execute() should indicate failure for invalid input")
				}
			}
		})
	}
}

// TestGoTesterConcurrency tests concurrent execution
func TestGoTesterConcurrency(t *testing.T) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	tester, _ := NewGoTester(nil, logger)

	tempDir := t.TempDir()
	createSimpleGoProject(t, tempDir)

	ctx := context.Background()
	input := map[string]interface{}{
		"path": tempDir,
	}

	// Run multiple concurrent tests
	const numGoroutines = 3 // Keep low to avoid overwhelming system
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := tester.Execute(ctx, input)
			results <- err
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent execution %d failed: %v", i, err)
		}
	}
}

// BenchmarkGoTester benchmarks tester performance
func BenchmarkGoTester(b *testing.B) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		b.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	tester, _ := NewGoTester(nil, logger)

	tempDir := b.TempDir()
	createSimpleGoProject(b, tempDir)

	ctx := context.Background()
	input := map[string]interface{}{
		"path":     tempDir,
		"coverage": false, // Disable coverage for faster benchmarks
		"verbose":  false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tester.Execute(ctx, input)
		if err != nil {
			b.Fatalf("Execute() error = %v", err)
		}
	}
}

// Helper function to create a simple Go project for testing
func createSimpleGoProject(t testing.TB, dir string) {
	files := map[string]string{
		"go.mod": `module testproject

go 1.21`,
		"main.go": `package main

func Add(a, b int) int {
	return a + b
}`,
		"main_test.go": `package main

import "testing"

func TestAdd(t *testing.T) {
	result := Add(2, 3)
	if result != 5 {
		t.Errorf("Add(2, 3) = %d, want 5", result)
	}
}`,
	}

	for filePath, content := range files {
		fullPath := filepath.Join(dir, filePath)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", filePath, err)
		}
	}
}
