package godev

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGoFormatterBasics tests basic formatter functionality
func TestGoFormatterBasics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	formatter, err := NewGoFormatter(nil, logger)
	if err != nil {
		t.Fatalf("NewGoFormatter() error = %v", err)
	}

	if formatter.Name() != "go_formatter" {
		t.Errorf("Name() = %s, want go_formatter", formatter.Name())
	}

	if formatter.Description() == "" {
		t.Error("Description() should not be empty")
	}

	params := formatter.Parameters()
	if params == nil {
		t.Error("Parameters() should not be nil")
	}

	// Check required parameters
	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Error("Parameters should have properties")
	}

	if _, exists := properties["path"]; !exists {
		t.Error("Parameters should include 'path' property")
	}

	required, ok := params["required"].([]string)
	if !ok {
		t.Error("Parameters should have required array")
	}

	if len(required) == 0 || required[0] != "path" {
		t.Error("'path' should be required parameter")
	}
}

// TestGoFormatterHealth tests health check
func TestGoFormatterHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	formatter, _ := NewGoFormatter(nil, logger)

	ctx := context.Background()
	err := formatter.Health(ctx)
	if err != nil {
		t.Errorf("Health() error = %v, want nil", err)
	}
}

// TestGoFormatterClose tests close functionality
func TestGoFormatterClose(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	formatter, _ := NewGoFormatter(nil, logger)

	ctx := context.Background()
	err := formatter.Close(ctx)
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

// TestGoFormatterValidation tests input validation
func TestGoFormatterValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	formatter, _ := NewGoFormatter(nil, logger)

	ctx := context.Background()

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name:      "missing_path",
			input:     map[string]interface{}{},
			wantError: true,
			errorMsg:  "path parameter is required",
		},
		{
			name:      "empty_path",
			input:     map[string]interface{}{"path": ""},
			wantError: true,
			errorMsg:  "path parameter is required",
		},
		{
			name:      "invalid_path_type",
			input:     map[string]interface{}{"path": 123},
			wantError: true,
			errorMsg:  "path parameter is required",
		},
		{
			name:      "nonexistent_path",
			input:     map[string]interface{}{"path": "/nonexistent/path"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.Execute(ctx, tt.input)
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

// TestGoFormatterWithFiles tests formatting with actual files
func TestGoFormatterWithFiles(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	formatter, _ := NewGoFormatter(nil, logger)

	tempDir := t.TempDir()

	// Create test files with different formatting states
	testFiles := map[string]struct {
		original    string
		formatted   string
		needsFormat bool
	}{
		"well_formatted.go": {
			original: `package main

import "fmt"

// main is the entry point
func main() {
	fmt.Println("Hello, World!")
}`,
			formatted: `package main

import "fmt"

// main is the entry point
func main() {
	fmt.Println("Hello, World!")
}`,
			needsFormat: false,
		},
		"needs_formatting.go": {
			original: `package main
import "fmt"
func main(){fmt.Println("Hello, World!")}`,
			formatted: `package main

import "fmt"

func main() { fmt.Println("Hello, World!") }`,
			needsFormat: true,
		},
		"syntax_error.go": {
			original: `package main
func main( {
	fmt.Println("syntax error"
}`,
			formatted:   "",
			needsFormat: false, // Will have format error
		},
	}

	for filename, testCase := range testFiles {
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(testCase.original), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	ctx := context.Background()

	tests := []struct {
		name              string
		input             map[string]interface{}
		expectSuccess     bool
		checkFileCount    bool
		expectedFormatted int
		expectedUnchanged int
		expectedErrors    int
		checkWrite        bool
	}{
		{
			name: "check_only_directory",
			input: map[string]interface{}{
				"path":       tempDir,
				"check_only": true,
				"recursive":  true,
			},
			expectSuccess:     true,
			checkFileCount:    true,
			expectedFormatted: 1, // needs_formatting.go
			expectedUnchanged: 1, // well_formatted.go
			expectedErrors:    1, // syntax_error.go
		},
		{
			name: "format_single_file",
			input: map[string]interface{}{
				"path":       filepath.Join(tempDir, "needs_formatting.go"),
				"check_only": true,
			},
			expectSuccess:     true,
			checkFileCount:    true,
			expectedFormatted: 1,
			expectedUnchanged: 0,
			expectedErrors:    0,
		},
		{
			name: "format_with_write",
			input: map[string]interface{}{
				"path":  filepath.Join(tempDir, "needs_formatting.go"),
				"write": true,
			},
			expectSuccess:     true,
			checkFileCount:    true,
			expectedFormatted: 1,
			checkWrite:        true,
		},
		{
			name: "exclude_recursive",
			input: map[string]interface{}{
				"path":      tempDir,
				"recursive": false,
			},
			expectSuccess:  true,
			checkFileCount: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.Execute(ctx, tt.input)

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
				}
				if result == nil || !result.Success {
					t.Error("Execute() should return successful result")
				}

				// Parse the result data
				formatterResult, ok := result.Data.(*FormatterResult)
				if !ok {
					t.Fatal("Execute() result data should be *FormatterResult")
				}

				if tt.checkFileCount {
					if tt.expectedFormatted >= 0 && len(formatterResult.FormattedFiles) != tt.expectedFormatted {
						t.Errorf("FormattedFiles count = %d, want %d",
							len(formatterResult.FormattedFiles), tt.expectedFormatted)
					}
					if tt.expectedUnchanged >= 0 && len(formatterResult.UnchangedFiles) != tt.expectedUnchanged {
						t.Errorf("UnchangedFiles count = %d, want %d",
							len(formatterResult.UnchangedFiles), tt.expectedUnchanged)
					}
					if tt.expectedErrors >= 0 && len(formatterResult.Errors) != tt.expectedErrors {
						t.Errorf("Errors count = %d, want %d",
							len(formatterResult.Errors), tt.expectedErrors)
					}
				}

				// Check summary
				summary := formatterResult.Summary
				expectedTotal := len(formatterResult.FormattedFiles) +
					len(formatterResult.UnchangedFiles) +
					len(formatterResult.Errors)
				if summary.TotalFiles != expectedTotal {
					t.Errorf("Summary TotalFiles = %d, want %d", summary.TotalFiles, expectedTotal)
				}
				if summary.FormattedFiles != len(formatterResult.FormattedFiles) {
					t.Errorf("Summary FormattedFiles = %d, want %d",
						summary.FormattedFiles, len(formatterResult.FormattedFiles))
				}

				if tt.checkWrite {
					// Verify file was actually written
					content, err := os.ReadFile(filepath.Join(tempDir, "needs_formatting.go"))
					if err != nil {
						t.Fatalf("Failed to read written file: %v", err)
					}

					// Should be properly formatted now
					contentStr := string(content)
					if !strings.Contains(contentStr, "import \"fmt\"") {
						t.Error("File should be properly formatted after write")
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

// TestGoFormatterParameters tests various parameter combinations
func TestGoFormatterParameters(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	formatter, _ := NewGoFormatter(nil, logger)

	tempDir := t.TempDir()

	// Create a file that needs formatting
	unformattedCode := `package main
import "fmt"
func main(){fmt.Println("test")}`

	testFile := filepath.Join(tempDir, "test.go")
	err := os.WriteFile(testFile, []byte(unformattedCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name           string
		input          map[string]interface{}
		checkParameter func(*testing.T, *FormatterResult, map[string]interface{})
	}{
		{
			name: "default_parameters",
			input: map[string]interface{}{
				"path": testFile,
			},
			checkParameter: func(t *testing.T, result *FormatterResult, metadata map[string]interface{}) {
				// Check defaults: write=false, recursive=true, simplify=true, check_only=false
				if write, ok := metadata["write"].(bool); !ok || write {
					t.Error("Default write should be false")
				}
				if recursive, ok := metadata["recursive"].(bool); !ok || !recursive {
					t.Error("Default recursive should be true")
				}
				if simplify, ok := metadata["simplify"].(bool); !ok || !simplify {
					t.Error("Default simplify should be true")
				}
				if checkOnly, ok := metadata["check_only"].(bool); !ok || checkOnly {
					t.Error("Default check_only should be false")
				}
			},
		},
		{
			name: "explicit_false_parameters",
			input: map[string]interface{}{
				"path":       testFile,
				"write":      false,
				"recursive":  false,
				"simplify":   false,
				"check_only": false,
			},
			checkParameter: func(t *testing.T, result *FormatterResult, metadata map[string]interface{}) {
				if write, ok := metadata["write"].(bool); !ok || write {
					t.Error("Write should be false")
				}
				if recursive, ok := metadata["recursive"].(bool); !ok || recursive {
					t.Error("Recursive should be false")
				}
				if simplify, ok := metadata["simplify"].(bool); !ok || simplify {
					t.Error("Simplify should be false")
				}
				if checkOnly, ok := metadata["check_only"].(bool); !ok || checkOnly {
					t.Error("Check_only should be false")
				}
			},
		},
		{
			name: "explicit_true_parameters",
			input: map[string]interface{}{
				"path":       testFile,
				"write":      true,
				"recursive":  true,
				"simplify":   true,
				"check_only": true,
			},
			checkParameter: func(t *testing.T, result *FormatterResult, metadata map[string]interface{}) {
				if write, ok := metadata["write"].(bool); !ok || !write {
					t.Error("Write should be true")
				}
				if recursive, ok := metadata["recursive"].(bool); !ok || !recursive {
					t.Error("Recursive should be true")
				}
				if simplify, ok := metadata["simplify"].(bool); !ok || !simplify {
					t.Error("Simplify should be true")
				}
				if checkOnly, ok := metadata["check_only"].(bool); !ok || !checkOnly {
					t.Error("Check_only should be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.Execute(ctx, tt.input)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			formatterResult := result.Data.(*FormatterResult)
			tt.checkParameter(t, formatterResult, result.Metadata)
		})
	}
}

// TestGoFormatterDirectoryStructure tests directory handling
func TestGoFormatterDirectoryStructure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	formatter, _ := NewGoFormatter(nil, logger)

	tempDir := t.TempDir()

	// Create directory structure
	// root/
	//   main.go
	//   vendor/
	//     vendor.go (should be skipped)
	//   subdir/
	//     sub.go
	//     subsubdir/
	//       subsub.go

	files := map[string]string{
		"main.go":                    `package main\nfunc main(){}`,
		"vendor/vendor.go":           `package vendor\nfunc vendor(){}`,
		"subdir/sub.go":              `package sub\nfunc sub(){}`,
		"subdir/subsubdir/subsub.go": `package subsub\nfunc subsub(){}`,
		"README.md":                  "# README", // non-Go file
	}

	for filePath, content := range files {
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

	tests := []struct {
		name          string
		input         map[string]interface{}
		expectedFiles int
		checkVendor   bool
	}{
		{
			name: "recursive_formatting",
			input: map[string]interface{}{
				"path":      tempDir,
				"recursive": true,
			},
			expectedFiles: 3, // main.go, sub.go, subsub.go (vendor skipped)
			checkVendor:   true,
		},
		{
			name: "non_recursive_formatting",
			input: map[string]interface{}{
				"path":      tempDir,
				"recursive": false,
			},
			expectedFiles: 1, // only main.go
		},
		{
			name: "subdir_recursive",
			input: map[string]interface{}{
				"path":      filepath.Join(tempDir, "subdir"),
				"recursive": true,
			},
			expectedFiles: 2, // sub.go, subsub.go
		},
		{
			name: "subdir_non_recursive",
			input: map[string]interface{}{
				"path":      filepath.Join(tempDir, "subdir"),
				"recursive": false,
			},
			expectedFiles: 1, // only sub.go
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatter.Execute(ctx, tt.input)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
			}

			formatterResult := result.Data.(*FormatterResult)
			totalFiles := len(formatterResult.FormattedFiles) +
				len(formatterResult.UnchangedFiles) +
				len(formatterResult.Errors)

			if totalFiles != tt.expectedFiles {
				t.Errorf("Total processed files = %d, want %d", totalFiles, tt.expectedFiles)
			}

			if tt.checkVendor {
				// Verify vendor files were skipped
				allPaths := make([]string, 0)
				for _, file := range formatterResult.FormattedFiles {
					allPaths = append(allPaths, file.Path)
				}
				for _, path := range formatterResult.UnchangedFiles {
					allPaths = append(allPaths, path)
				}
				for _, err := range formatterResult.Errors {
					allPaths = append(allPaths, err.Path)
				}

				for _, path := range allPaths {
					if strings.Contains(path, "/vendor/") {
						t.Errorf("Vendor file should be skipped: %s", path)
					}
				}
			}
		})
	}
}

// TestGoFormatterWriteOperations tests file writing
func TestGoFormatterWriteOperations(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	formatter, _ := NewGoFormatter(nil, logger)

	tempDir := t.TempDir()

	tests := []struct {
		name         string
		filename     string
		original     string
		expectWrite  bool
		writeMode    os.FileMode
		checkContent bool
	}{
		{
			name:     "write_formatted_file",
			filename: "writable.go",
			original: `package main
func main(){}`,
			expectWrite:  true,
			writeMode:    0644,
			checkContent: true,
		},
		{
			name:     "readonly_file",
			filename: "readonly.go",
			original: `package main
func main(){}`,
			expectWrite: false,
			writeMode:   0444, // readonly
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tt.filename)
			err := os.WriteFile(testFile, []byte(tt.original), tt.writeMode)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			ctx := context.Background()
			input := map[string]interface{}{
				"path":  testFile,
				"write": true,
			}

			result, err := formatter.Execute(ctx, input)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
			}

			formatterResult := result.Data.(*FormatterResult)

			if tt.expectWrite {
				if len(formatterResult.Errors) > 0 {
					t.Errorf("Should not have write errors for writable file")
				}
				if len(formatterResult.FormattedFiles) == 0 {
					t.Error("Should format writable file")
				}

				if tt.checkContent {
					content, err := os.ReadFile(testFile)
					if err != nil {
						t.Fatalf("Failed to read formatted file: %v", err)
					}
					if !strings.Contains(string(content), "func main() {") {
						t.Error("File should be properly formatted")
					}
				}
			} else {
				// Should have write error for readonly file
				if len(formatterResult.Errors) == 0 {
					t.Error("Should have write error for readonly file")
				}
			}
		})
	}
}

// TestGoFormatterConcurrency tests concurrent execution
func TestGoFormatterConcurrency(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	formatter, _ := NewGoFormatter(nil, logger)

	// Create test files
	tempDir := t.TempDir()
	for i := 0; i < 5; i++ {
		testCode := `package main
func main(){}`
		testFile := filepath.Join(tempDir, "test"+string(rune('0'+i))+".go")
		err := os.WriteFile(testFile, []byte(testCode), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	ctx := context.Background()
	input := map[string]interface{}{
		"path": tempDir,
	}

	// Run multiple concurrent formats
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := formatter.Execute(ctx, input)
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

// TestGoFormatterContextCancellation tests context cancellation
func TestGoFormatterContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	formatter, _ := NewGoFormatter(nil, logger)

	// Create multiple test files to slow down formatting
	tempDir := t.TempDir()
	for i := 0; i < 10; i++ {
		testCode := `package main
func test(){}`
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

	_, err := formatter.Execute(ctx, input)
	if err == nil {
		t.Error("Execute() with cancelled context should return error")
	} else if !strings.Contains(err.Error(), "context") {
		t.Errorf("Error should mention context cancellation, got: %v", err)
	}
}

// TestGoFormatterErrorHandling tests various error conditions
func TestGoFormatterErrorHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	formatter, _ := NewGoFormatter(nil, logger)

	tempDir := t.TempDir()

	tests := []struct {
		name          string
		setupFunc     func() string
		input         map[string]interface{}
		expectErrors  bool
		expectFormats bool
	}{
		{
			name: "syntax_error_file",
			setupFunc: func() string {
				invalidCode := `package main
func main( {
	// syntax error
}`
				testFile := filepath.Join(tempDir, "syntax_error.go")
				err := os.WriteFile(testFile, []byte(invalidCode), 0644)
				if err != nil {
					t.Fatal(err)
				}
				return testFile
			},
			expectErrors:  true,
			expectFormats: false,
		},
		{
			name: "mixed_valid_invalid",
			setupFunc: func() string {
				// Create directory with mix of valid and invalid files
				subDir := filepath.Join(tempDir, "mixed")
				err := os.Mkdir(subDir, 0755)
				if err != nil {
					t.Fatal(err)
				}

				validCode := `package main
func main(){}`
				err = os.WriteFile(filepath.Join(subDir, "valid.go"), []byte(validCode), 0644)
				if err != nil {
					t.Fatal(err)
				}

				invalidCode := `package main
func main( {}`
				err = os.WriteFile(filepath.Join(subDir, "invalid.go"), []byte(invalidCode), 0644)
				if err != nil {
					t.Fatal(err)
				}

				return subDir
			},
			expectErrors:  true,
			expectFormats: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFunc()
			input := map[string]interface{}{
				"path": path,
			}
			if tt.input != nil {
				for k, v := range tt.input {
					input[k] = v
				}
			}

			ctx := context.Background()
			result, err := formatter.Execute(ctx, input)

			// Should not fail completely, but continue processing
			if err != nil {
				t.Errorf("Execute() should handle errors gracefully, got: %v", err)
			}
			if result == nil || !result.Success {
				t.Error("Execute() should succeed even with some file errors")
			}

			formatterResult := result.Data.(*FormatterResult)

			if tt.expectErrors && len(formatterResult.Errors) == 0 {
				t.Error("Should have formatting errors")
			}

			if tt.expectFormats && len(formatterResult.FormattedFiles) == 0 {
				t.Error("Should have successfully formatted some files")
			}

			// Check that errors have proper details
			for _, formatErr := range formatterResult.Errors {
				if formatErr.Path == "" {
					t.Error("Error should include file path")
				}
				if formatErr.Error == "" {
					t.Error("Error should include error message")
				}
			}
		})
	}
}

// BenchmarkGoFormatter benchmarks formatter performance
func BenchmarkGoFormatter(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	formatter, _ := NewGoFormatter(nil, logger)

	// Create test file
	tempDir := b.TempDir()
	testCode := `package main
import "fmt"
func main(){fmt.Println("test")}`
	testFile := filepath.Join(tempDir, "benchmark.go")
	err := os.WriteFile(testFile, []byte(testCode), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()

	b.Run("format_single_file", func(b *testing.B) {
		input := map[string]interface{}{
			"path":       testFile,
			"check_only": true,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := formatter.Execute(ctx, input)
			if err != nil {
				b.Fatalf("Execute() error = %v", err)
			}
		}
	})

	b.Run("format_with_write", func(b *testing.B) {
		input := map[string]interface{}{
			"path":  testFile,
			"write": true,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Restore unformatted content
			err := os.WriteFile(testFile, []byte(testCode), 0644)
			if err != nil {
				b.Fatal(err)
			}

			_, err = formatter.Execute(ctx, input)
			if err != nil {
				b.Fatalf("Execute() error = %v", err)
			}
		}
	})
}

// TestGoFormatterEdgeCases tests edge cases
func TestGoFormatterEdgeCases(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	formatter, _ := NewGoFormatter(nil, logger)

	ctx := context.Background()

	t.Run("empty_directory", func(t *testing.T) {
		emptyDir := t.TempDir()
		input := map[string]interface{}{
			"path": emptyDir,
		}

		result, err := formatter.Execute(ctx, input)
		if err != nil {
			t.Errorf("Execute() on empty directory error = %v", err)
		}
		if result == nil || !result.Success {
			t.Error("Execute() on empty directory should succeed")
		}

		formatterResult := result.Data.(*FormatterResult)
		if formatterResult.Summary.TotalFiles != 0 {
			t.Errorf("Empty directory should have 0 files, got %d", formatterResult.Summary.TotalFiles)
		}
	})

	t.Run("already_formatted_file", func(t *testing.T) {
		tempDir := t.TempDir()
		wellFormattedCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`
		testFile := filepath.Join(tempDir, "formatted.go")
		err := os.WriteFile(testFile, []byte(wellFormattedCode), 0644)
		if err != nil {
			t.Fatal(err)
		}

		input := map[string]interface{}{
			"path": testFile,
		}

		result, err := formatter.Execute(ctx, input)
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}

		formatterResult := result.Data.(*FormatterResult)
		if len(formatterResult.UnchangedFiles) != 1 {
			t.Errorf("Should have 1 unchanged file, got %d", len(formatterResult.UnchangedFiles))
		}
		if len(formatterResult.FormattedFiles) != 0 {
			t.Errorf("Should have 0 formatted files, got %d", len(formatterResult.FormattedFiles))
		}
	})
}
