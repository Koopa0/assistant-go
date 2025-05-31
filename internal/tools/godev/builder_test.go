package godev

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestGoBuilderBasics tests basic builder functionality
func TestGoBuilderBasics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	builder, err := NewGoBuilder(nil, logger)
	if err != nil {
		t.Fatalf("NewGoBuilder() error = %v", err)
	}

	if builder.Name() != "go_builder" {
		t.Errorf("Name() = %s, want go_builder", builder.Name())
	}

	if builder.Description() == "" {
		t.Error("Description() should not be empty")
	}

	params := builder.Parameters()
	if params == nil {
		t.Error("Parameters() should not be nil")
	}

	// Check parameters structure
	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Error("Parameters should have properties")
	}

	expectedProps := []string{"path", "output", "target_os", "target_arch", "ldflags", "static"}
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

// TestGoBuilderHealth tests health check
func TestGoBuilderHealth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	builder, _ := NewGoBuilder(nil, logger)

	ctx := context.Background()
	err := builder.Health(ctx)

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

// TestGoBuilderClose tests close functionality
func TestGoBuilderClose(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	builder, _ := NewGoBuilder(nil, logger)

	ctx := context.Background()
	err := builder.Close(ctx)
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

// TestGoBuilderParameterParsing tests parameter parsing
func TestGoBuilderParameterParsing(t *testing.T) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	builder, _ := NewGoBuilder(nil, logger)

	// Create a simple Go project
	tempDir := t.TempDir()
	createBuildableGoProject(t, tempDir)

	ctx := context.Background()

	tests := []struct {
		name             string
		input            map[string]interface{}
		expectSuccess    bool
		checkMetadata    bool
		expectedTargetOS string
		expectedStatic   bool
	}{
		{
			name:             "default_parameters",
			input:            map[string]interface{}{"path": tempDir},
			expectSuccess:    true,
			checkMetadata:    true,
			expectedTargetOS: runtime.GOOS,
			expectedStatic:   false,
		},
		{
			name: "explicit_parameters",
			input: map[string]interface{}{
				"path":        tempDir,
				"output":      "custom_binary",
				"target_os":   "linux",
				"target_arch": "amd64",
				"static":      true,
				"trimpath":    true,
				"ldflags":     "-s -w",
			},
			expectSuccess:    true,
			checkMetadata:    true,
			expectedTargetOS: "linux",
			expectedStatic:   true,
		},
		{
			name: "with_build_tags",
			input: map[string]interface{}{
				"path": tempDir,
				"tags": []interface{}{"dev", "debug"},
			},
			expectSuccess: true,
		},
		{
			name: "cross_compile_windows",
			input: map[string]interface{}{
				"path":        tempDir,
				"target_os":   "windows",
				"target_arch": "amd64",
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.Execute(ctx, tt.input)

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
				}
				if result == nil {
					t.Fatal("Execute() returned nil result")
				}

				// Check result data
				buildResult, ok := result.Data.(*BuildResult)
				if !ok {
					t.Fatal("Execute() result data should be *BuildResult")
				}

				if !buildResult.Success {
					t.Errorf("Build should succeed, got: %s", buildResult.Output)
				}

				// Check metadata
				if tt.checkMetadata && result.Metadata != nil {
					if targetOS, ok := result.Metadata["target_os"].(string); ok {
						if targetOS != tt.expectedTargetOS {
							t.Errorf("Metadata target_os = %s, want %s", targetOS, tt.expectedTargetOS)
						}
					}
					if static, ok := result.Metadata["static"].(bool); ok {
						if static != tt.expectedStatic {
							t.Errorf("Metadata static = %t, want %t", static, tt.expectedStatic)
						}
					}
				}

				// Check build result fields
				if buildResult.BinaryPath == "" {
					t.Error("BinaryPath should not be empty")
				}
				if buildResult.BuildTime <= 0 {
					t.Error("BuildTime should be positive")
				}
				if buildResult.GoVersion == "" {
					t.Error("GoVersion should not be empty")
				}

				// Check target platform
				if buildResult.TargetPlatform.OS != tt.expectedTargetOS {
					t.Errorf("TargetPlatform.OS = %s, want %s",
						buildResult.TargetPlatform.OS, tt.expectedTargetOS)
				}
				if buildResult.TargetPlatform.CGOEnabled == tt.expectedStatic {
					t.Errorf("TargetPlatform.CGOEnabled = %t, should be inverse of static (%t)",
						buildResult.TargetPlatform.CGOEnabled, tt.expectedStatic)
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

// TestGoBuilderWithRealProject tests building real Go projects
func TestGoBuilderWithRealProject(t *testing.T) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	builder, _ := NewGoBuilder(nil, logger)

	tempDir := t.TempDir()

	// Create a more comprehensive Go project
	projectFiles := map[string]string{
		"go.mod": `module buildtest

go 1.21

require (
	github.com/google/uuid v1.3.0
)`,
		"main.go": `package main

import (
	"fmt"
	"os"

	"github.com/google/uuid"
)

var version = "dev"
var buildTime = "unknown"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("Version: %s\nBuild Time: %s\n", version, buildTime)
		return
	}
	
	id := uuid.New()
	fmt.Printf("Hello from build test! UUID: %s\n", id.String())
}`,
		"README.md": "# Build Test Project",
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
		checkBinary       bool
		checkDependencies bool
		checkLdflags      bool
	}{
		{
			name: "simple_build",
			input: map[string]interface{}{
				"path": tempDir,
			},
			checkBinary:       true,
			checkDependencies: true,
		},
		{
			name: "build_with_ldflags",
			input: map[string]interface{}{
				"path":    tempDir,
				"output":  "versioned_binary",
				"ldflags": `-X main.version=1.0.0 -X main.buildTime=2023-01-01`,
			},
			checkBinary:  true,
			checkLdflags: true,
		},
		{
			name: "static_build",
			input: map[string]interface{}{
				"path":     tempDir,
				"output":   "static_binary",
				"static":   true,
				"trimpath": true,
				"ldflags":  "-s -w",
			},
			checkBinary: true,
		},
		{
			name: "cross_compile",
			input: map[string]interface{}{
				"path":        tempDir,
				"target_os":   "linux",
				"target_arch": "amd64",
				"output":      "linux_binary",
			},
			checkBinary: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.Execute(ctx, tt.input)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
			}
			if result == nil || !result.Success {
				t.Error("Execute() should return successful result")
			}

			buildResult := result.Data.(*BuildResult)

			if tt.checkBinary {
				// Check that binary was created
				if buildResult.BinaryPath == "" {
					t.Error("BinaryPath should not be empty")
				}

				// For same-OS builds, check if binary exists and is executable
				if buildResult.TargetPlatform.OS == runtime.GOOS {
					if _, err := os.Stat(buildResult.BinaryPath); err != nil {
						t.Errorf("Binary should exist at %s: %v", buildResult.BinaryPath, err)
					}

					// Check binary size
					if buildResult.BinarySize <= 0 {
						t.Error("BinarySize should be positive")
					}
				}
			}

			if tt.checkDependencies {
				if len(buildResult.Dependencies) == 0 {
					t.Error("Should find dependencies (google/uuid)")
				}

				foundUUID := false
				for _, dep := range buildResult.Dependencies {
					if strings.Contains(dep.Path, "google/uuid") {
						foundUUID = true
						break
					}
				}
				if !foundUUID {
					t.Error("Should find google/uuid dependency")
				}
			}

			if tt.checkLdflags {
				if !strings.Contains(buildResult.BuildInfo.LDFlags, "main.version") {
					t.Error("Should preserve ldflags in BuildInfo")
				}
			}

			// Check build info
			if buildResult.BuildInfo.Static != (tt.input["static"] == true) {
				t.Errorf("BuildInfo.Static = %t, want %t",
					buildResult.BuildInfo.Static, tt.input["static"] == true)
			}

			// Check output
			if buildResult.Output == "" {
				t.Error("Should capture build output")
			}

			// Cleanup binary for next test
			if buildResult.BinaryPath != "" {
				os.Remove(buildResult.BinaryPath)
			}
		})
	}
}

// TestGoBuilderBuildModes tests different build modes
func TestGoBuilderBuildModes(t *testing.T) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	builder, _ := NewGoBuilder(nil, logger)

	tempDir := t.TempDir()
	createBuildableGoProject(t, tempDir)

	ctx := context.Background()

	tests := []struct {
		name       string
		buildMode  string
		shouldWork bool
	}{
		{"default", "default", true},
		{"exe", "exe", true},
		{"pie", "pie", true},
		// Note: c-archive and c-shared require CGO and special setup
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]interface{}{
				"path":       tempDir,
				"build_mode": tt.buildMode,
				"output":     "test_" + tt.buildMode,
			}

			result, err := builder.Execute(ctx, input)

			if tt.shouldWork {
				if err != nil {
					t.Errorf("Execute() error = %v", err)
				}
				if result == nil || !result.Success {
					t.Error("Execute() should succeed")
				}

				buildResult := result.Data.(*BuildResult)
				if buildResult.BuildInfo.BuildMode != tt.buildMode {
					t.Errorf("BuildInfo.BuildMode = %s, want %s",
						buildResult.BuildInfo.BuildMode, tt.buildMode)
				}

				// Cleanup
				if buildResult.BinaryPath != "" {
					os.Remove(buildResult.BinaryPath)
				}
			}
		})
	}
}

// TestGoBuilderErrorHandling tests error handling
func TestGoBuilderErrorHandling(t *testing.T) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	builder, _ := NewGoBuilder(nil, logger)

	ctx := context.Background()

	tests := []struct {
		name      string
		setupFunc func() string
		input     map[string]interface{}
		expectErr bool
	}{
		{
			name: "nonexistent_path",
			input: map[string]interface{}{
				"path": "/nonexistent/path",
			},
			expectErr: true,
		},
		{
			name: "invalid_go_code",
			setupFunc: func() string {
				tempDir := t.TempDir()

				// Create invalid Go code
				files := map[string]string{
					"go.mod": `module invalid

go 1.21`,
					"main.go": `package main

func main( {
	// syntax error - missing closing parenthesis
}`,
				}

				for filePath, content := range files {
					fullPath := filepath.Join(tempDir, filePath)
					os.MkdirAll(filepath.Dir(fullPath), 0755)
					os.WriteFile(fullPath, []byte(content), 0644)
				}

				return tempDir
			},
			expectErr: true,
		},
		{
			name: "missing_dependency",
			setupFunc: func() string {
				tempDir := t.TempDir()

				files := map[string]string{
					"go.mod": `module missing

go 1.21`,
					"main.go": `package main

import "github.com/nonexistent/package"

func main() {
	// This will fail to build
}`,
				}

				for filePath, content := range files {
					fullPath := filepath.Join(tempDir, filePath)
					os.MkdirAll(filepath.Dir(fullPath), 0755)
					os.WriteFile(fullPath, []byte(content), 0644)
				}

				return tempDir
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.input
			if tt.setupFunc != nil {
				path := tt.setupFunc()
				if input == nil {
					input = make(map[string]interface{})
				}
				input["path"] = path
			}

			result, err := builder.Execute(ctx, input)

			if tt.expectErr {
				if err == nil {
					t.Error("Execute() should return error for invalid input")
				}
				// Should still return result with error details
				if result != nil {
					buildResult := result.Data.(*BuildResult)
					if buildResult.Success {
						t.Error("BuildResult.Success should be false for failed builds")
					}
					if buildResult.Output == "" {
						t.Error("Should capture error output")
					}
				}
			} else {
				if err != nil {
					t.Errorf("Execute() error = %v, want nil", err)
				}
			}
		})
	}
}

// TestGoBuilderConcurrency tests concurrent builds
func TestGoBuilderConcurrency(t *testing.T) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	builder, _ := NewGoBuilder(nil, logger)

	tempDir := t.TempDir()
	createBuildableGoProject(t, tempDir)

	ctx := context.Background()

	// Run multiple concurrent builds with different outputs
	const numGoroutines = 3
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			input := map[string]interface{}{
				"path":   tempDir,
				"output": "concurrent_binary_" + string(rune('0'+id)),
			}
			_, err := builder.Execute(ctx, input)
			results <- err
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent build %d failed: %v", i, err)
		}
	}
}

// TestGoBuilderContextCancellation tests context cancellation
func TestGoBuilderContextCancellation(t *testing.T) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	builder, _ := NewGoBuilder(nil, logger)

	tempDir := t.TempDir()
	createBuildableGoProject(t, tempDir)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	input := map[string]interface{}{
		"path": tempDir,
	}

	_, err := builder.Execute(ctx, input)
	if err == nil {
		t.Error("Execute() with cancelled context should return error")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Errorf("Error should mention context cancellation, got: %v", err)
	}
}

// BenchmarkGoBuilder benchmarks builder performance
func BenchmarkGoBuilder(b *testing.B) {
	// Skip if go is not available
	if _, err := exec.LookPath("go"); err != nil {
		b.Skip("go command not available")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	builder, _ := NewGoBuilder(nil, logger)

	tempDir := b.TempDir()
	createBuildableGoProject(b, tempDir)

	ctx := context.Background()
	input := map[string]interface{}{
		"path": tempDir,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := builder.Execute(ctx, input)
		if err != nil {
			b.Fatalf("Execute() error = %v", err)
		}
		// Clean up binary
		if result.Data != nil {
			if buildResult, ok := result.Data.(*BuildResult); ok && buildResult.BinaryPath != "" {
				os.Remove(buildResult.BinaryPath)
			}
		}
	}
}

// Helper function to create a buildable Go project
func createBuildableGoProject(t testing.TB, dir string) {
	files := map[string]string{
		"go.mod": `module testbuild

go 1.21`,
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello from test build!")
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
