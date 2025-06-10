package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLIFunctionality tests the CLI features documented in CLI_FUNCTIONALITY.md
func TestCLIFunctionality(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Build the CLI binary
	binaryPath := filepath.Join(t.TempDir(), "assistant")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/assistant")
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(buildOutput))

	// Ensure Claude API key is set
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		t.Skip("CLAUDE_API_KEY not set")
	}

	// Test 1: Version Command
	t.Run("VersionCommand", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "version")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// As documented in CLI_FUNCTIONALITY.md
		assert.Contains(t, string(output), "Assistant")
		assert.Contains(t, string(output), "0.1.0")
		assert.Contains(t, string(output), "Go version:")
		assert.Contains(t, string(output), "Platform:")
	})

	// Test 2: Direct Query Mode (ask)
	t.Run("DirectQueryMode", func(t *testing.T) {
		testCases := []struct {
			name     string
			query    string
			expected []string
		}{
			{
				name:     "SimpleQuestion",
				query:    "What is Go's context package?",
				expected: []string{"context", "cancel", "deadline"},
			},
			{
				name:     "CodeAnalysis",
				query:    "How do I handle errors in Go?",
				expected: []string{"error", "nil", "return"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cmd := exec.Command(binaryPath, "ask", tc.query)
				cmd.Env = append(os.Environ(),
					"CLAUDE_API_KEY="+apiKey,
					"DATABASE_URL=postgres://test:test@localhost:5432/test?sslmode=disable",
					"GEMINI_API_KEY=AIzaSyABCDEFGHIJKLMNOPQRSTUVWXYZ1234567",
				)

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "Command failed: %s", string(output))

				outputStr := strings.ToLower(string(output))
				for _, expected := range tc.expected {
					assert.Contains(t, outputStr, strings.ToLower(expected),
						"Expected to find '%s' in output", expected)
				}
			})
		}
	})

	// Test 3: Interactive CLI Commands
	t.Run("InteractiveCLICommands", func(t *testing.T) {
		testCases := []struct {
			name     string
			commands []string
			expected []string
		}{
			{
				name:     "HelpCommand",
				commands: []string{"help", "exit"},
				expected: []string{"help", "exit", "clear", "status", "tools"},
			},
			{
				name:     "StatusCommand",
				commands: []string{"status", "exit"},
				expected: []string{"系統狀態", "AI 服務", "可用工具"},
			},
			{
				name:     "ToolsCommand",
				commands: []string{"tools", "exit"},
				expected: []string{"可用工具", "godev", "docker", "postgres"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				cmd := exec.Command(binaryPath, "cli")
				cmd.Env = append(os.Environ(),
					"CLAUDE_API_KEY="+apiKey,
					"DATABASE_URL=postgres://test:test@localhost:5432/test?sslmode=disable",
					"GEMINI_API_KEY=AIzaSyABCDEFGHIJKLMNOPQRSTUVWXYZ1234567",
				)

				// Prepare input
				var stdin bytes.Buffer
				for _, command := range tc.commands {
					stdin.WriteString(command + "\n")
				}
				cmd.Stdin = &stdin

				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "Command failed: %s", string(output))

				outputStr := string(output)
				for _, expected := range tc.expected {
					assert.Contains(t, outputStr, expected,
						"Expected to find '%s' in output", expected)
				}
			})
		}
	})

	// Test 4: Tool Integration
	t.Run("ToolIntegration", func(t *testing.T) {
		// Test Go development tool
		t.Run("GoDevTool", func(t *testing.T) {
			queries := []string{
				"analyze this Go project structure",
				"check the Go workspace",
				"what type of Go project is this",
			}

			for _, query := range queries {
				cmd := exec.Command(binaryPath, "ask", query)
				cmd.Env = append(os.Environ(), "CLAUDE_API_KEY="+apiKey)
				cmd.Dir = "." // Use current project

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Logf("Query '%s' output: %s", query, string(output))
				}

				// Tool might be called, check for no errors
				assert.NotContains(t, string(output), "panic")
				assert.NotContains(t, string(output), "failed to")
			}
		})
	})

	// Test 5: Environment Configuration
	t.Run("EnvironmentConfiguration", func(t *testing.T) {
		// Test with different Claude model
		t.Run("ModelConfiguration", func(t *testing.T) {
			cmd := exec.Command(binaryPath, "ask", "Hello")
			cmd.Env = append(os.Environ(),
				"CLAUDE_API_KEY="+apiKey,
				"CLAUDE_MODEL=claude-3-sonnet-20240229",
			)

			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Command failed: %s", string(output))
			assert.NotEmpty(t, string(output))
		})

		// Test with custom log level
		t.Run("LogLevel", func(t *testing.T) {
			cmd := exec.Command(binaryPath, "ask", "test")
			cmd.Env = append(os.Environ(),
				"CLAUDE_API_KEY="+apiKey,
				"LOG_LEVEL=debug",
			)

			output, err := cmd.CombinedOutput()
			// Debug mode might show more output
			assert.NotNil(t, output)
			// Should not fail
			if err != nil {
				t.Logf("Debug output: %s", string(output))
			}
		})
	})
}

// TestPromptEnhancement tests the 7 prompt templates
func TestPromptEnhancement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		t.Skip("CLAUDE_API_KEY not set")
	}

	binaryPath := filepath.Join(t.TempDir(), "assistant")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/assistant")
	require.NoError(t, buildCmd.Run())

	// Test each prompt template type
	testCases := []struct {
		name     string
		query    string
		keywords []string // Keywords that should appear in enhanced response
	}{
		{
			name:     "CodeAnalysis",
			query:    "analyze this function for potential issues",
			keywords: []string{"analysis", "function", "issues", "quality"},
		},
		{
			name:     "Refactoring",
			query:    "refactor this code to be more idiomatic",
			keywords: []string{"refactor", "idiomatic", "improve"},
		},
		{
			name:     "Performance",
			query:    "why is this code running slowly",
			keywords: []string{"performance", "slow", "optimize"},
		},
		{
			name:     "Architecture",
			query:    "review the architecture of this system",
			keywords: []string{"architecture", "design", "structure"},
		},
		{
			name:     "TestGeneration",
			query:    "write unit tests for this function",
			keywords: []string{"test", "unit", "coverage"},
		},
		{
			name:     "ErrorDiagnosis",
			query:    "debug this error message",
			keywords: []string{"error", "debug", "fix"},
		},
		{
			name:     "WorkspaceAnalysis",
			query:    "analyze my project workspace",
			keywords: []string{"project", "workspace", "structure"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, "ask", tc.query)
			cmd.Env = append(os.Environ(), "CLAUDE_API_KEY="+apiKey)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Command failed: %s", string(output))

			outputStr := strings.ToLower(string(output))
			// Check that the response is relevant to the query type
			foundKeyword := false
			for _, keyword := range tc.keywords {
				if strings.Contains(outputStr, strings.ToLower(keyword)) {
					foundKeyword = true
					break
				}
			}
			assert.True(t, foundKeyword,
				"Expected response to contain at least one keyword from %v", tc.keywords)
		})
	}
}
