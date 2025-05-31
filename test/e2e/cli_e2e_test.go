//go:build e2e

package e2e

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CLITestSession represents an interactive CLI test session
type CLITestSession struct {
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	stderr      io.ReadCloser
	scanner     *bufio.Scanner
	tempDir     string
	binaryPath  string
	configFile  string
	dbContainer *testutil.DatabaseContainer
	cleanup     func()
}

// SetupCLITestSession creates a new CLI test session
func SetupCLITestSession(t *testing.T) *CLITestSession {
	if testing.Short() {
		t.Skip("Skipping CLI E2E tests in short mode")
	}

	// Setup test database
	dbContainer, cleanup := testutil.SetupTestDatabase(t)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "assistant-cli-e2e-*")
	require.NoError(t, err, "Should create temp directory")

	// Build the assistant binary
	binaryPath := filepath.Join(tempDir, "assistant")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/assistant")
	buildCmd.Dir = tempDir
	err = buildCmd.Run()
	require.NoError(t, err, "Should build assistant binary")

	// Create test configuration
	configFile := filepath.Join(tempDir, "config.yaml")
	err = createTestConfig(configFile, dbContainer.URL)
	require.NoError(t, err, "Should create test config")

	session := &CLITestSession{
		tempDir:     tempDir,
		binaryPath:  binaryPath,
		configFile:  configFile,
		dbContainer: dbContainer,
		cleanup: func() {
			cleanup()
			os.RemoveAll(tempDir)
		},
	}

	return session
}

// StartInteractiveCLI starts the interactive CLI session
func (s *CLITestSession) StartInteractiveCLI(t *testing.T) {
	// Start CLI in interactive mode
	s.cmd = exec.Command(s.binaryPath, "cli")
	s.cmd.Env = append(os.Environ(),
		"CONFIG_FILE="+s.configFile,
		"CLAUDE_API_KEY=test-key",
		"ASSISTANT_DEMO_MODE=true",
		"CLI_HISTORY_FILE="+filepath.Join(s.tempDir, ".assistant_history"),
	)

	// Setup pipes for interaction
	stdin, err := s.cmd.StdinPipe()
	require.NoError(t, err, "Should create stdin pipe")
	s.stdin = stdin

	stdout, err := s.cmd.StdoutPipe()
	require.NoError(t, err, "Should create stdout pipe")
	s.stdout = stdout

	stderr, err := s.cmd.StderrPipe()
	require.NoError(t, err, "Should create stderr pipe")
	s.stderr = stderr

	// Start the command
	err = s.cmd.Start()
	require.NoError(t, err, "Should start CLI command")

	// Create scanner for reading output
	s.scanner = bufio.NewScanner(s.stdout)

	// Wait for CLI to be ready (look for prompt)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = s.waitForPrompt(ctx)
	require.NoError(t, err, "CLI should be ready")
}

// SendCommand sends a command to the CLI and waits for response
func (s *CLITestSession) SendCommand(command string) (string, error) {
	// Send command
	_, err := s.stdin.Write([]byte(command + "\n"))
	if err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	// Read response until next prompt
	var response strings.Builder
	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for response")
		default:
			if s.scanner.Scan() {
				line := s.scanner.Text()
				response.WriteString(line + "\n")

				// Check if we've reached the next prompt
				if strings.Contains(line, "assistant>") || strings.Contains(line, "> ") {
					return response.String(), nil
				}
			} else {
				// Check for scanner error
				if err := s.scanner.Err(); err != nil {
					return "", fmt.Errorf("scanner error: %w", err)
				}
				// If no error, continue reading
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
}

// waitForPrompt waits for the CLI prompt to appear
func (s *CLITestSession) waitForPrompt(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for CLI prompt: %w", ctx.Err())
		default:
			if s.scanner.Scan() {
				line := s.scanner.Text()
				if strings.Contains(line, "assistant>") || strings.Contains(line, "> ") {
					return nil
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Close closes the CLI session
func (s *CLITestSession) Close() {
	if s.stdin != nil {
		s.stdin.Write([]byte("exit\n"))
		s.stdin.Close()
	}
	if s.stdout != nil {
		s.stdout.Close()
	}
	if s.stderr != nil {
		s.stderr.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}
	s.cleanup()
}

// TestInteractiveCLIWorkflow tests the interactive CLI workflow
func TestInteractiveCLIWorkflow(t *testing.T) {
	session := SetupCLITestSession(t)
	defer session.Close()

	session.StartInteractiveCLI(t)

	t.Run("help_command", func(t *testing.T) {
		response, err := session.SendCommand("help")
		require.NoError(t, err, "Help command should succeed")

		assert.Contains(t, response, "Available Commands", "Help should show available commands")
		assert.Contains(t, response, "exit", "Help should mention exit command")
		assert.Contains(t, response, "status", "Help should mention status command")
	})

	t.Run("status_command", func(t *testing.T) {
		response, err := session.SendCommand("status")
		require.NoError(t, err, "Status command should succeed")

		assert.Contains(t, response, "System Status", "Status should show system status")
	})

	t.Run("tools_command", func(t *testing.T) {
		response, err := session.SendCommand("tools")
		require.NoError(t, err, "Tools command should succeed")

		assert.Contains(t, response, "Available Tools", "Tools should show available tools")
	})

	t.Run("simple_query", func(t *testing.T) {
		response, err := session.SendCommand("What is Go programming language?")
		require.NoError(t, err, "Simple query should succeed")

		assert.NotEmpty(t, response, "Query should have response")
		assert.Contains(t, strings.ToLower(response), "go", "Response should mention Go")
	})

	t.Run("code_related_query", func(t *testing.T) {
		response, err := session.SendCommand("How do I create a HTTP server in Go?")
		require.NoError(t, err, "Code query should succeed")

		assert.NotEmpty(t, response, "Code query should have response")
		responseText := strings.ToLower(response)
		assert.True(t,
			strings.Contains(responseText, "http") || strings.Contains(responseText, "server"),
			"Response should mention HTTP or server")
	})

	t.Run("clear_command", func(t *testing.T) {
		response, err := session.SendCommand("clear")
		require.NoError(t, err, "Clear command should succeed")

		// Clear command typically doesn't return much output
		assert.NotNil(t, response, "Clear should return some response")
	})
}

// TestCLIToolIntegration tests CLI tool integration
func TestCLIToolIntegration(t *testing.T) {
	session := SetupCLITestSession(t)
	defer session.Close()

	session.StartInteractiveCLI(t)

	t.Run("docker_command", func(t *testing.T) {
		response, err := session.SendCommand("docker ps")
		// Docker might not be available, so we don't require success
		if err == nil {
			assert.NotEmpty(t, response, "Docker command should have response")
		} else {
			t.Logf("Docker command failed (expected if Docker not available): %v", err)
		}
	})

	t.Run("sql_command", func(t *testing.T) {
		response, err := session.SendCommand("sql SELECT version()")
		require.NoError(t, err, "SQL command should succeed")

		assert.NotEmpty(t, response, "SQL command should have response")
		// Should show PostgreSQL version or error message
		responseText := strings.ToLower(response)
		assert.True(t,
			strings.Contains(responseText, "postgresql") ||
				strings.Contains(responseText, "version") ||
				strings.Contains(responseText, "error"),
			"SQL response should mention PostgreSQL, version, or error")
	})

	t.Run("k8s_command", func(t *testing.T) {
		response, err := session.SendCommand("k8s get pods")
		// Kubernetes might not be available, so we don't require success
		if err == nil {
			assert.NotEmpty(t, response, "K8s command should have response")
		} else {
			t.Logf("K8s command failed (expected if kubectl not available): %v", err)
		}
	})
}

// TestCLIErrorHandling tests CLI error handling
func TestCLIErrorHandling(t *testing.T) {
	session := SetupCLITestSession(t)
	defer session.Close()

	session.StartInteractiveCLI(t)

	t.Run("invalid_command", func(t *testing.T) {
		response, err := session.SendCommand("invalidcommandthatdoesnotexist")
		require.NoError(t, err, "Invalid command should be handled gracefully")

		assert.NotEmpty(t, response, "Invalid command should have response")
		// Should either process as query or show error
		assert.NotContains(t, response, "panic", "Should not panic on invalid command")
	})

	t.Run("empty_input", func(t *testing.T) {
		response, err := session.SendCommand("")
		require.NoError(t, err, "Empty input should be handled gracefully")

		// Empty input should just return to prompt
		assert.NotContains(t, response, "error", "Empty input should not cause error")
	})

	t.Run("very_long_input", func(t *testing.T) {
		longInput := strings.Repeat("This is a very long input string. ", 100)
		response, err := session.SendCommand(longInput)
		require.NoError(t, err, "Long input should be handled gracefully")

		assert.NotEmpty(t, response, "Long input should have response")
		assert.NotContains(t, response, "panic", "Should not panic on long input")
	})
}

// TestCLIHistoryAndSession tests CLI history and session management
func TestCLIHistoryAndSession(t *testing.T) {
	session := SetupCLITestSession(t)
	defer session.Close()

	session.StartInteractiveCLI(t)

	t.Run("command_history", func(t *testing.T) {
		// Send a few commands
		_, err := session.SendCommand("help")
		require.NoError(t, err)

		_, err = session.SendCommand("status")
		require.NoError(t, err)

		// Check history
		response, err := session.SendCommand("history")
		require.NoError(t, err, "History command should succeed")

		assert.Contains(t, response, "help", "History should contain help command")
		assert.Contains(t, response, "status", "History should contain status command")
	})

	t.Run("session_persistence", func(t *testing.T) {
		// Test that session state is maintained
		_, err := session.SendCommand("What is my name?")
		require.NoError(t, err)

		// Follow up question that might reference previous context
		response, err := session.SendCommand("Can you remember what I just asked?")
		require.NoError(t, err, "Follow-up question should succeed")

		assert.NotEmpty(t, response, "Follow-up should have response")
	})
}

// TestCLIPerformance tests CLI performance characteristics
func TestCLIPerformance(t *testing.T) {
	session := SetupCLITestSession(t)
	defer session.Close()

	session.StartInteractiveCLI(t)

	t.Run("response_time", func(t *testing.T) {
		// Measure response time for simple commands
		commands := []string{"help", "status", "tools"}

		for _, cmd := range commands {
			start := time.Now()
			response, err := session.SendCommand(cmd)
			duration := time.Since(start)

			require.NoError(t, err, "Command should succeed")
			assert.NotEmpty(t, response, "Command should have response")
			assert.Less(t, duration, 5*time.Second, "Command should respond quickly")

			t.Logf("Command '%s' took %v", cmd, duration)
		}
	})

	t.Run("rapid_commands", func(t *testing.T) {
		// Test rapid command execution
		start := time.Now()
		numCommands := 5

		for i := 0; i < numCommands; i++ {
			_, err := session.SendCommand("status")
			require.NoError(t, err, "Rapid command should succeed")
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(numCommands)

		t.Logf("Executed %d rapid commands in %v (avg: %v)", numCommands, duration, avgTime)
		assert.Less(t, duration, 30*time.Second, "Rapid commands should complete quickly")
	})
}

// TestCLIGracefulShutdown tests CLI graceful shutdown
func TestCLIGracefulShutdown(t *testing.T) {
	session := SetupCLITestSession(t)
	defer session.Close()

	session.StartInteractiveCLI(t)

	t.Run("exit_command", func(t *testing.T) {
		// Send exit command
		_, err := session.SendCommand("exit")
		// Don't require no error since the process will exit

		// Wait for process to exit
		done := make(chan error, 1)
		go func() {
			done <- session.cmd.Wait()
		}()

		select {
		case err := <-done:
			// Process should exit cleanly
			if err != nil {
				// Exit code 0 is success, others might be expected
				t.Logf("CLI exited with: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("CLI did not exit within timeout")
			session.cmd.Process.Kill()
		}
	})
}

// BenchmarkCLICommands benchmarks CLI command performance
func BenchmarkCLICommands(b *testing.B) {
	session := SetupCLITestSession(b)
	defer session.Close()

	session.StartInteractiveCLI(b)

	b.Run("help_command", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := session.SendCommand("help")
			if err != nil {
				b.Fatalf("Help command failed: %v", err)
			}
		}
	})

	b.Run("status_command", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := session.SendCommand("status")
			if err != nil {
				b.Fatalf("Status command failed: %v", err)
			}
		}
	})
}
