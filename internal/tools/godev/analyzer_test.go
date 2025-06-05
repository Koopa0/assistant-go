package godev

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/tools"
)

// TestGoAnalyzerBasics tests basic analyzer functionality
func TestGoAnalyzerBasics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	if analyzer.Name() != "go_analyzer" {
		t.Errorf("Name() = %s, want go_analyzer", analyzer.Name())
	}

	if analyzer.Description() == "" {
		t.Error("Description() should not be empty")
	}

	params := analyzer.Parameters()
	if params == nil {
		t.Error("Parameters() should not be nil")
	}

	// Check required parameters
	if params.Properties == nil {
		t.Error("Parameters should have properties")
		return
	}

	if _, exists := params.Properties["path"]; !exists {
		t.Error("Parameters should include 'path' property")
	}

	if len(params.Required) == 0 || params.Required[0] != "path" {
		t.Error("'path' should be required parameter")
	}
}

// TestGoAnalyzerHealth tests health check
func TestGoAnalyzerHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	ctx := context.Background()
	err = analyzer.Health(ctx)
	if err != nil {
		t.Errorf("Health() error = %v, want nil", err)
	}
}

// TestGoAnalyzerClose tests close functionality
func TestGoAnalyzerClose(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	ctx := context.Background()
	err = analyzer.Close(ctx)
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

// TestGoAnalyzerValidation tests input validation
func TestGoAnalyzerValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name      string
		input     *tools.ToolInput
		wantError bool
		errorMsg  string
	}{
		{
			name:      "missing_path",
			input:     &tools.ToolInput{Parameters: map[string]interface{}{}},
			wantError: true,
			errorMsg:  "path parameter is required",
		},
		{
			name:      "empty_path",
			input:     &tools.ToolInput{Parameters: map[string]interface{}{"path": ""}},
			wantError: true,
			errorMsg:  "path parameter is required",
		},
		{
			name:      "invalid_path_type",
			input:     &tools.ToolInput{Parameters: map[string]interface{}{"path": 123}},
			wantError: true,
			errorMsg:  "path parameter is required",
		},
		{
			name:      "nonexistent_path",
			input:     &tools.ToolInput{Parameters: map[string]interface{}{"path": "/nonexistent/path"}},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.Execute(ctx, tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("Execute() error = nil, want error")
				}
				if result == nil || result.Success {
					t.Error("Execute() should return failed result")
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Execute() error = %v, want to contain %v", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
				}
			}
		})
	}
}

// TestGoAnalyzerWithTestFiles tests analysis with actual test files
func TestGoAnalyzerWithTestFiles(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	// Create temporary test files
	tempDir := t.TempDir()

	testFiles := map[string]string{
		"simple.go": `package main

import "fmt"

// main is the entry point
func main() {
	fmt.Println("Hello, World!")
}`,
		"complex.go": `package main

import (
	"fmt"
	"os"
	"strings"
)

// User represents a user in the system
type User struct {
	ID   int    ` + "`json:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
	Age  int    ` + "`json:\"age\"`" + `
}

// UserService provides user operations
type UserService interface {
	GetUser(id int) (*User, error)
	CreateUser(name string, age int) (*User, error)
}

// processUser is a complex function with high cyclomatic complexity
func processUser(user *User, options map[string]string) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}
	
	if user.Name == "" {
		return fmt.Errorf("user name cannot be empty")
	}
	
	if user.Age < 0 {
		return fmt.Errorf("user age cannot be negative")
	}
	
	if user.Age > 150 {
		return fmt.Errorf("user age seems unrealistic")
	}
	
	for key, value := range options {
		switch key {
		case "format":
			if value == "upper" {
				user.Name = strings.ToUpper(user.Name)
			} else if value == "lower" {
				user.Name = strings.ToLower(user.Name)
			} else if value == "title" {
				user.Name = strings.Title(user.Name)
			}
		case "validate":
			if value == "strict" {
				if len(user.Name) < 2 {
					return fmt.Errorf("name too short")
				}
				if len(user.Name) > 50 {
					return fmt.Errorf("name too long")
				}
			}
		case "output":
			if value == "json" {
				fmt.Printf("User: %+v\n", user)
			} else if value == "yaml" {
				fmt.Printf("name: %s\nage: %d\n", user.Name, user.Age)
			}
		}
	}
	
	return nil
}

// GetUser gets a user by ID (exported but no doc comment)
func GetUser(id int) *User {
	return &User{ID: id, Name: "Test", Age: 25}
}`,
		"test_file_test.go": `package main

import "testing"

func TestGetUser(t *testing.T) {
	user := GetUser(1)
	if user == nil {
		t.Error("GetUser should not return nil")
	}
}`,
		"invalid_syntax.go": `package main

// This file has syntax errors
func main( {
	fmt.Println("Missing closing parenthesis"
}`,
	}

	for filename, content := range testFiles {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	ctx := context.Background()

	tests := []struct {
		name             string
		input            map[string]interface{}
		expectSuccess    bool
		checkFileCount   bool
		expectedMinFiles int
		checkComplexity  bool
		checkIssues      bool
		checkSecurity    bool
		checkStructs     bool
		checkInterfaces  bool
		checkSuggestions bool
	}{
		{
			name: "analyze_directory_all",
			input: map[string]interface{}{
				"path":          tempDir,
				"analysis_type": "all",
				"recursive":     true,
				"include_tests": true,
			},
			expectSuccess:    true,
			checkFileCount:   true,
			expectedMinFiles: 3, // Should parse 3 valid files
			checkComplexity:  true,
			checkIssues:      true,
			checkSecurity:    true,
			checkStructs:     true,
			checkInterfaces:  true,
			checkSuggestions: true,
		},
		{
			name: "analyze_single_file",
			input: map[string]interface{}{
				"path":          filepath.Join(tempDir, "simple.go"),
				"analysis_type": "all",
			},
			expectSuccess:    true,
			checkFileCount:   true,
			expectedMinFiles: 1,
		},
		{
			name: "analyze_structure_only",
			input: map[string]interface{}{
				"path":          tempDir,
				"analysis_type": "structure",
				"recursive":     true,
			},
			expectSuccess: true,
			checkStructs:  true,
			checkIssues:   false, // Issues should be filtered out
		},
		{
			name: "analyze_complexity_only",
			input: map[string]interface{}{
				"path":          tempDir,
				"analysis_type": "complexity",
				"recursive":     true,
			},
			expectSuccess:   true,
			checkComplexity: true,
		},
		{
			name: "exclude_tests",
			input: map[string]interface{}{
				"path":          tempDir,
				"include_tests": false,
				"recursive":     true,
			},
			expectSuccess:    true,
			checkFileCount:   true,
			expectedMinFiles: 2, // Should exclude test file
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.Execute(ctx, tt.input)

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
				}
				if result == nil || !result.Success {
					t.Error("Execute() should return successful result")
				}

				// Parse the result data
				analysisResult, ok := result.Data.(*AnalysisResult)
				if !ok {
					t.Fatal("Execute() result data should be *AnalysisResult")
				}

				if tt.checkFileCount {
					if len(analysisResult.Files) < tt.expectedMinFiles {
						t.Errorf("Analysis result files count = %d, want >= %d",
							len(analysisResult.Files), tt.expectedMinFiles)
					}
				}

				if tt.checkComplexity {
					if analysisResult.Summary.TotalFunctions == 0 {
						t.Error("Should find functions in analysis")
					}
					if analysisResult.Summary.AverageComplexity <= 0 {
						t.Error("Should calculate average complexity")
					}
				}

				if tt.checkIssues {
					// Should find issues (like missing docs, high complexity)
					if len(analysisResult.Issues) == 0 {
						t.Error("Should find code issues in test files")
					}
				}

				if tt.checkStructs {
					foundStruct := false
					for _, file := range analysisResult.Files {
						if len(file.Structs) > 0 {
							foundStruct = true
							break
						}
					}
					if !foundStruct {
						t.Error("Should find struct definitions")
					}
				}

				if tt.checkInterfaces {
					foundInterface := false
					for _, file := range analysisResult.Files {
						if len(file.Interfaces) > 0 {
							foundInterface = true
							break
						}
					}
					if !foundInterface {
						t.Error("Should find interface definitions")
					}
				}

				if tt.checkSecurity {
					// Security analysis should run
					if analysisResult.SecurityIssues == nil {
						t.Error("SecurityIssues should be initialized")
					}
				}

				if tt.checkSuggestions {
					if len(analysisResult.Suggestions) == 0 {
						t.Error("Should generate improvement suggestions")
					}
				}

				// Check metadata
				if result.Metadata == nil {
					t.Error("Result should include metadata")
				}
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

// TestGoAnalyzerComplexityCalculation tests complexity calculation
func TestGoAnalyzerComplexityCalculation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	tempDir := t.TempDir()

	// Create a file with known complexity
	complexCode := `package main

func simpleFunction() {
	// Complexity = 1 (base)
}

func conditionalFunction(x int) {
	if x > 0 {  // +1
		// do something
	}
	// Total complexity = 2
}

func loopFunction(items []int) {
	for _, item := range items {  // +1
		if item > 0 {  // +1
			// do something
		}
	}
	// Total complexity = 3
}

func switchFunction(x int) {
	switch x {  // +1
	case 1:     // +1
		// case 1
	case 2:     // +1
		// case 2
	default:
		// default
	}
	// Total complexity = 4
}`

	testFile := filepath.Join(tempDir, "complexity.go")
	err = os.WriteFile(testFile, []byte(complexCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	input := map[string]interface{}{
		"path":          testFile,
		"analysis_type": "all",
	}

	result, err := analyzer.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	analysisResult := result.Data.(*AnalysisResult)
	if len(analysisResult.Files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(analysisResult.Files))
	}

	file := analysisResult.Files[0]
	if len(file.Functions) != 4 {
		t.Fatalf("Expected 4 functions, got %d", len(file.Functions))
	}

	expectedComplexities := map[string]int{
		"simpleFunction":      1,
		"conditionalFunction": 2,
		"loopFunction":        3,
		"switchFunction":      4,
	}

	for _, function := range file.Functions {
		expected, exists := expectedComplexities[function.Name]
		if !exists {
			t.Errorf("Unexpected function: %s", function.Name)
			continue
		}
		if function.Complexity != expected {
			t.Errorf("Function %s complexity = %d, want %d",
				function.Name, function.Complexity, expected)
		}
	}
}

// TestGoAnalyzerIssueDetection tests various issue detection
func TestGoAnalyzerIssueDetection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	tempDir := t.TempDir()

	// Create file with various issues
	issueCode := `// Package without documentation
package main

import "fmt"

// ExportedStruct is documented  
type ExportedStruct struct {
	Field1 string
	Field2 int
	Field3 bool
	Field4 float64
	Field5 []string
	Field6 map[string]int
	Field7 interface{}
	Field8 string
	Field9 int
	Field10 bool
	Field11 float64
	Field12 []string
	Field13 map[string]int
	Field14 interface{}
	Field15 string
	Field16 int
	Field17 bool
	Field18 float64
	Field19 []string
	Field20 map[string]int
	Field21 interface{}
}

type UndocumentedExportedStruct struct {
	Field string
}

// ExportedFunction is documented
func ExportedFunction() {
	fmt.Println("documented")
}

func UndocumentedExportedFunction() {
	fmt.Println("undocumented")
}

func VeryLongFunction() {
	// This function will be over 50 lines
	line1 := 1
	line2 := 2
	line3 := 3
	line4 := 4
	line5 := 5
	line6 := 6
	line7 := 7
	line8 := 8
	line9 := 9
	line10 := 10
	line11 := 11
	line12 := 12
	line13 := 13
	line14 := 14
	line15 := 15
	line16 := 16
	line17 := 17
	line18 := 18
	line19 := 19
	line20 := 20
	line21 := 21
	line22 := 22
	line23 := 23
	line24 := 24
	line25 := 25
	line26 := 26
	line27 := 27
	line28 := 28
	line29 := 29
	line30 := 30
	line31 := 31
	line32 := 32
	line33 := 33
	line34 := 34
	line35 := 35
	line36 := 36
	line37 := 37
	line38 := 38
	line39 := 39
	line40 := 40
	line41 := 41
	line42 := 42
	line43 := 43
	line44 := 44
	line45 := 45
	line46 := 46
	line47 := 47
	line48 := 48
	line49 := 49
	line50 := 50
	line51 := 51
	fmt.Println(line1, line2, line3, line50, line51)
}

func HighComplexityFunction(x, y, z int) int {
	if x > 0 {
		if y > 0 {
			if z > 0 {
				for i := 0; i < x; i++ {
					for j := 0; j < y; j++ {
						switch z {
						case 1:
							return 1
						case 2:
							return 2
						case 3:
							return 3
						case 4:
							return 4
						default:
							return 0
						}
					}
				}
			}
		}
	}
	return -1
}`

	testFile := filepath.Join(tempDir, "issues.go")
	err = os.WriteFile(testFile, []byte(issueCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	input := map[string]interface{}{
		"path":          testFile,
		"analysis_type": "all",
	}

	result, err := analyzer.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	analysisResult := result.Data.(*AnalysisResult)

	// Check that issues were detected
	if len(analysisResult.Issues) == 0 {
		t.Error("Should detect code issues")
	}

	issueTypes := make(map[IssueType]int)
	for _, issue := range analysisResult.Issues {
		issueTypes[issue.Type]++
	}

	// Should detect missing documentation
	if issueTypes[IssueTypeStyle] == 0 {
		t.Error("Should detect style issues (missing documentation)")
	}

	// Should detect high complexity
	if issueTypes[IssueTypeComplexity] == 0 {
		t.Error("Should detect complexity issues")
	}

	// Should detect large struct
	if issueTypes[IssueTypeStructure] == 0 {
		t.Error("Should detect structural issues (large struct/function)")
	}
}

// TestGoAnalyzerSecurityAnalysis tests security issue detection
func TestGoAnalyzerSecurityAnalysis(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	tempDir := t.TempDir()

	securityCode := `package main

import (
	"database/sql"
	"os/exec"
	"os"
	"math/rand"
	"path/filepath"
	"io/ioutil"
)

func processPassword(password string) {
	// Function name suggests credential handling
}

func executeQuery(db *sql.DB, userInput string) {
	// Using database/sql - should flag for SQL injection review
	query := "SELECT * FROM users WHERE id = " + userInput
	db.Query(query)
}

func executeCommand(userInput string) {
	// Using os/exec - should flag for command injection review
	cmd := exec.Command("ls", userInput)
	cmd.Run()
}

func readFile(userPath string) {
	// Using file operations - should flag for path traversal review
	content, _ := os.ReadFile(userPath)
	_ = content
}

func generateToken() {
	// Using math/rand - should flag for insecure random
	token := rand.Int()
	_ = token
}

func processPath(userPath string) {
	// Using path operations
	cleanPath := filepath.Clean(userPath)
	content, _ := ioutil.ReadFile(cleanPath)
	_ = content
}`

	testFile := filepath.Join(tempDir, "security.go")
	err = os.WriteFile(testFile, []byte(securityCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	input := map[string]interface{}{
		"path":          testFile,
		"analysis_type": "all",
	}

	result, err := analyzer.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	analysisResult := result.Data.(*AnalysisResult)

	if len(analysisResult.SecurityIssues) == 0 {
		t.Error("Should detect security issues")
	}

	securityTypes := make(map[string]int)
	for _, issue := range analysisResult.SecurityIssues {
		securityTypes[issue.Type]++
	}

	expectedIssues := []string{
		"hardcoded_credentials",
		"sql_injection_risk",
		"command_injection_risk",
		"path_traversal_risk",
		"insecure_random",
	}

	for _, expectedType := range expectedIssues {
		if securityTypes[expectedType] == 0 {
			t.Errorf("Should detect %s security issue", expectedType)
		}
	}
}

// TestGoAnalyzerPerformanceAnalysis tests performance issue detection
func TestGoAnalyzerPerformanceAnalysis(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	tempDir := t.TempDir()

	performanceCode := `package main

import "sync"

func highComplexityFunction() {
	// Very high complexity function
	for i := 0; i < 100; i++ {
		for j := 0; j < 100; j++ {
			if i > j {
				if i%2 == 0 {
					if j%2 == 1 {
						// Nested conditions
					}
				}
			}
		}
	}
}

func veryLargeFunction() {
	// Function with many lines (simulated with repetitive code)
` + strings.Repeat("	line := 1\n", 150) + `
}

func syncOperations() {
	// Function using sync package
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()
}`

	testFile := filepath.Join(tempDir, "performance.go")
	err = os.WriteFile(testFile, []byte(performanceCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	input := map[string]interface{}{
		"path":          testFile,
		"analysis_type": "all",
	}

	result, err := analyzer.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	analysisResult := result.Data.(*AnalysisResult)

	// Should detect performance issues
	if len(analysisResult.Performance.IneffientLoops) == 0 {
		t.Error("Should detect inefficient code patterns")
	}

	if len(analysisResult.Performance.LargeAllocations) == 0 {
		t.Error("Should detect large function allocations")
	}

	if len(analysisResult.Performance.UnbufferedChannels) == 0 {
		t.Error("Should detect sync usage patterns")
	}
}

// TestGoAnalyzerConcurrency tests concurrent execution
func TestGoAnalyzerConcurrency(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	// Create a test file
	tempDir := t.TempDir()
	testCode := `package main
func main() {}`
	testFile := filepath.Join(tempDir, "test.go")
	err = os.WriteFile(testFile, []byte(testCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	input := map[string]interface{}{
		"path": testFile,
	}

	// Run multiple concurrent analyses
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, execErr := analyzer.Execute(ctx, input)
			results <- execErr
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent execution %d failed: %v", i, err)
		}
	}
}

// TestGoAnalyzerContextCancellation tests context cancellation
func TestGoAnalyzerContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	// Create multiple test files to slow down analysis
	tempDir := t.TempDir()
	for i := 0; i < 10; i++ {
		testCode := `package main
func test() {}`
		testFile := filepath.Join(tempDir, "test"+string(rune('0'+i))+".go")
		err := os.WriteFile(testFile, []byte(testCode), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	input := map[string]interface{}{
		"path":      tempDir,
		"recursive": true,
	}

	_, err = analyzer.Execute(ctx, input)
	if err == nil {
		t.Error("Execute() with cancelled context should return error")
	} else if !strings.Contains(err.Error(), "context") {
		t.Errorf("Error should mention context cancellation, got: %v", err)
	}
}

// BenchmarkGoAnalyzer benchmarks analyzer performance
func BenchmarkGoAnalyzer(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		b.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	// Create a test file
	tempDir := b.TempDir()
	testCode := `package main

import "fmt"

type User struct {
	ID   int
	Name string
}

func GetUser(id int) *User {
	return &User{ID: id, Name: "test"}
}

func ProcessUser(user *User) error {
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}
	if user.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	return nil
}

func main() {
	user := GetUser(1)
	if err := ProcessUser(user); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}`

	testFile := filepath.Join(tempDir, "benchmark.go")
	err = os.WriteFile(testFile, []byte(testCode), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	input := map[string]interface{}{
		"path":          testFile,
		"analysis_type": "all",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.Execute(ctx, input)
		if err != nil {
			b.Fatalf("Execute() error = %v", err)
		}
	}
}

// TestGoAnalyzerTypeConversions tests type conversion utilities
func TestGoAnalyzerTypeConversions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	tool, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}
	analyzer, ok := tool.(*GoAnalyzer)
	if !ok {
		t.Fatal("NewGoAnalyzer() type assertion failed")
	}

	// These are internal methods, so we test them indirectly by creating
	// a file that exercises various type scenarios
	tempDir := t.TempDir()
	typeTestCode := `package main

type ComplexType struct {
	SimpleString  string
	PointerInt    *int
	SliceStrings  []string
	MapStringInt  map[string]int
	Channel       chan string
	Interface     interface{}
	InnerStruct   struct{ Field int }
	Function      func(int) string
	SelectorType  os.File
}

type SimpleInterface interface {
	Method1() string
	Method2(int) error
	Method3(string, bool) (int, error)
}`

	testFile := filepath.Join(tempDir, "types.go")
	err = os.WriteFile(testFile, []byte(typeTestCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	input := map[string]interface{}{
		"path":          testFile,
		"analysis_type": "structure",
	}

	result, err := analyzer.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	analysisResult := result.Data.(*AnalysisResult)
	if len(analysisResult.Files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(analysisResult.Files))
	}

	file := analysisResult.Files[0]

	// Check struct analysis
	if len(file.Structs) != 1 {
		t.Fatalf("Expected 1 struct, got %d", len(file.Structs))
	}

	complexTypeStruct := file.Structs[0]
	if complexTypeStruct.Name != "ComplexType" {
		t.Errorf("Struct name = %s, want ComplexType", complexTypeStruct.Name)
	}

	// Check field types were converted properly
	expectedFields := []string{
		"SimpleString", "PointerInt", "SliceStrings", "MapStringInt",
		"Channel", "Interface", "InnerStruct", "Function", "SelectorType",
	}

	if len(complexTypeStruct.FieldList) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(complexTypeStruct.FieldList))
	}

	// Check interface analysis
	if len(file.Interfaces) != 1 {
		t.Fatalf("Expected 1 interface, got %d", len(file.Interfaces))
	}

	simpleInterface := file.Interfaces[0]
	if simpleInterface.Name != "SimpleInterface" {
		t.Errorf("Interface name = %s, want SimpleInterface", simpleInterface.Name)
	}

	if len(simpleInterface.MethodList) != 3 {
		t.Errorf("Expected 3 methods, got %d", len(simpleInterface.MethodList))
	}

	// Verify method signatures were parsed
	for _, method := range simpleInterface.MethodList {
		if method.Signature == "" {
			t.Errorf("Method %s should have signature", method.Name)
		}
	}
}

// TestGoAnalyzerEdgeCases tests edge cases and error conditions
func TestGoAnalyzerEdgeCases(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	analyzer, err := NewGoAnalyzer(nil, logger)
	if err != nil {
		t.Fatalf("NewGoAnalyzer() error = %v", err)
	}

	ctx := context.Background()

	t.Run("empty_directory", func(t *testing.T) {
		emptyDir := t.TempDir()
		input := map[string]interface{}{
			"path": emptyDir,
		}

		result, err := analyzer.Execute(ctx, input)
		if err != nil {
			t.Errorf("Execute() on empty directory error = %v", err)
		}
		if result == nil || !result.Success {
			t.Error("Execute() on empty directory should succeed")
		}

		analysisResult := result.Data.(*AnalysisResult)
		if len(analysisResult.Files) != 0 {
			t.Errorf("Empty directory should have 0 files, got %d", len(analysisResult.Files))
		}
	})

	t.Run("directory_with_only_non_go_files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create non-Go files
		err := os.WriteFile(filepath.Join(tempDir, "readme.txt"), []byte("readme"), 0644)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(tempDir, "config.json"), []byte("{}"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		input := map[string]interface{}{
			"path": tempDir,
		}

		result, err := analyzer.Execute(ctx, input)
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}

		analysisResult := result.Data.(*AnalysisResult)
		if len(analysisResult.Files) != 0 {
			t.Errorf("Directory with no Go files should have 0 files, got %d", len(analysisResult.Files))
		}
	})

	t.Run("recursive_vs_non_recursive", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create Go file in root
		err := os.WriteFile(filepath.Join(tempDir, "root.go"), []byte("package main"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Create subdirectory with Go file
		subDir := filepath.Join(tempDir, "subdir")
		err = os.Mkdir(subDir, 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(subDir, "sub.go"), []byte("package sub"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Test non-recursive
		input := map[string]interface{}{
			"path":      tempDir,
			"recursive": false,
		}

		result, err := analyzer.Execute(ctx, input)
		if err != nil {
			t.Errorf("Execute() non-recursive error = %v", err)
		}

		analysisResult := result.Data.(*AnalysisResult)
		if len(analysisResult.Files) != 1 {
			t.Errorf("Non-recursive should find 1 file, got %d", len(analysisResult.Files))
		}

		// Test recursive
		input["recursive"] = true
		result, err = analyzer.Execute(ctx, input)
		if err != nil {
			t.Errorf("Execute() recursive error = %v", err)
		}

		analysisResult = result.Data.(*AnalysisResult)
		if len(analysisResult.Files) != 2 {
			t.Errorf("Recursive should find 2 files, got %d", len(analysisResult.Files))
		}
	})
}
