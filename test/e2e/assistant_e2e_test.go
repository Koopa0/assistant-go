//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// E2ETestSuite represents the end-to-end test suite
type E2ETestSuite struct {
	serverProcess *exec.Cmd
	serverURL     string
	tempDir       string
	configFile    string
	dbContainer   *testutil.DatabaseContainer
	cleanup       func()
}

// SetupE2ETestSuite initializes the complete test environment
func SetupE2ETestSuite(t *testing.T) *E2ETestSuite {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	// Setup test database
	dbContainer, cleanup := testutil.SetupTestDatabase(t)

	// Create temporary directory for test artifacts
	tempDir, err := os.MkdirTemp("", "assistant-e2e-*")
	require.NoError(t, err, "Should create temp directory")

	// Create test configuration
	configFile := filepath.Join(tempDir, "config.yaml")
	err = createTestConfig(configFile, dbContainer.URL)
	require.NoError(t, err, "Should create test config")

	suite := &E2ETestSuite{
		serverURL:   "http://localhost:8081", // Use different port for E2E
		tempDir:     tempDir,
		configFile:  configFile,
		dbContainer: dbContainer,
		cleanup: func() {
			cleanup()
			os.RemoveAll(tempDir)
		},
	}

	return suite
}

// StartServer starts the assistant server for E2E testing
func (s *E2ETestSuite) StartServer(t *testing.T) {
	// Build the assistant binary
	binaryPath := filepath.Join(s.tempDir, "assistant")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/assistant")
	buildCmd.Dir = s.tempDir
	err := buildCmd.Run()
	require.NoError(t, err, "Should build assistant binary")

	// Start server with test configuration
	s.serverProcess = exec.Command(binaryPath, "serve")
	s.serverProcess.Env = append(os.Environ(),
		"CONFIG_FILE="+s.configFile,
		"SERVER_ADDRESS=:8081",
		"LOG_LEVEL=debug",
		"CLAUDE_API_KEY=test-key",
		"ASSISTANT_DEMO_MODE=true",
	)

	// Capture server output for debugging
	var stdout, stderr bytes.Buffer
	s.serverProcess.Stdout = &stdout
	s.serverProcess.Stderr = &stderr

	err = s.serverProcess.Start()
	require.NoError(t, err, "Should start server process")

	// Wait for server to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = s.waitForServerReady(ctx)
	if err != nil {
		t.Logf("Server stdout: %s", stdout.String())
		t.Logf("Server stderr: %s", stderr.String())
		require.NoError(t, err, "Server should be ready")
	}
}

// StopServer stops the assistant server
func (s *E2ETestSuite) StopServer() {
	if s.serverProcess != nil {
		s.serverProcess.Process.Kill()
		s.serverProcess.Wait()
	}
}

// Cleanup cleans up all test resources
func (s *E2ETestSuite) Cleanup() {
	s.StopServer()
	s.cleanup()
}

// waitForServerReady waits for the server to be ready to accept requests
func (s *E2ETestSuite) waitForServerReady(ctx context.Context) error {
	client := &http.Client{Timeout: 5 * time.Second}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for server to be ready: %w", ctx.Err())
		default:
			resp, err := client.Get(s.serverURL + "/health")
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// TestCompleteUserWorkflow tests the complete user workflow
func TestCompleteUserWorkflow(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	suite.StartServer(t)

	t.Run("health_check", func(t *testing.T) {
		resp, err := http.Get(suite.serverURL + "/health")
		require.NoError(t, err, "Health check should succeed")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Health check should return 200")
	})

	t.Run("query_processing_workflow", func(t *testing.T) {
		// Test the complete query processing workflow
		testCases := []struct {
			name     string
			query    string
			expected string
		}{
			{
				name:     "simple_greeting",
				query:    "Hello, can you help me?",
				expected: "help",
			},
			{
				name:     "go_development_question",
				query:    "How do I create a Go HTTP server?",
				expected: "HTTP",
			},
			{
				name:     "code_analysis_request",
				query:    "Analyze this Go code for potential issues",
				expected: "analyze",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create query request
				queryReq := map[string]interface{}{
					"query": tc.query,
				}

				reqBody, err := json.Marshal(queryReq)
				require.NoError(t, err)

				// Send request to server
				resp, err := http.Post(
					suite.serverURL+"/api/v1/query",
					"application/json",
					bytes.NewBuffer(reqBody),
				)
				require.NoError(t, err, "Query request should succeed")
				defer resp.Body.Close()

				// Verify response
				assert.Equal(t, http.StatusOK, resp.StatusCode, "Query should return 200")

				var response map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err, "Should decode response")

				assert.Contains(t, response, "response", "Response should contain response field")
				responseText := response["response"].(string)
				assert.Contains(t, strings.ToLower(responseText), strings.ToLower(tc.expected),
					"Response should contain expected content")
			})
		}
	})

	t.Run("tool_execution_workflow", func(t *testing.T) {
		// Test tool execution through the API
		toolReq := map[string]interface{}{
			"tool_name": "go-analyzer",
			"input": map[string]interface{}{
				"path": ".",
			},
		}

		reqBody, err := json.Marshal(toolReq)
		require.NoError(t, err)

		resp, err := http.Post(
			suite.serverURL+"/api/v1/tools/execute",
			"application/json",
			bytes.NewBuffer(reqBody),
		)
		require.NoError(t, err, "Tool execution request should succeed")
		defer resp.Body.Close()

		// Tool might not be available in test environment, so we accept both success and error
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest,
			"Tool execution should return valid status")
	})

	t.Run("conversation_management", func(t *testing.T) {
		// Test conversation creation and management
		convReq := map[string]interface{}{
			"title": "Test Conversation",
		}

		reqBody, err := json.Marshal(convReq)
		require.NoError(t, err)

		// Create conversation
		resp, err := http.Post(
			suite.serverURL+"/api/v1/conversations",
			"application/json",
			bytes.NewBuffer(reqBody),
		)
		require.NoError(t, err, "Conversation creation should succeed")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode, "Conversation should be created")

		var conversation map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&conversation)
		require.NoError(t, err, "Should decode conversation response")

		conversationID := conversation["id"].(string)
		assert.NotEmpty(t, conversationID, "Conversation should have ID")

		// List conversations
		resp, err = http.Get(suite.serverURL + "/api/v1/conversations")
		require.NoError(t, err, "List conversations should succeed")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "List conversations should return 200")

		var conversations []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&conversations)
		require.NoError(t, err, "Should decode conversations list")

		assert.NotEmpty(t, conversations, "Should have at least one conversation")
	})
}

// TestCLIWorkflow tests the CLI interface end-to-end
func TestCLIWorkflow(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	// Build the assistant binary
	binaryPath := filepath.Join(suite.tempDir, "assistant")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/assistant")
	buildCmd.Dir = suite.tempDir
	err := buildCmd.Run()
	require.NoError(t, err, "Should build assistant binary")

	t.Run("direct_query_mode", func(t *testing.T) {
		// Test direct query mode
		cmd := exec.Command(binaryPath, "ask", "What is Go?")
		cmd.Env = append(os.Environ(),
			"CONFIG_FILE="+suite.configFile,
			"CLAUDE_API_KEY=test-key",
			"ASSISTANT_DEMO_MODE=true",
		)

		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Direct query should succeed")

		outputStr := string(output)
		assert.NotEmpty(t, outputStr, "Should have output")
		assert.Contains(t, strings.ToLower(outputStr), "go", "Output should mention Go")
	})

	t.Run("version_command", func(t *testing.T) {
		// Test version command
		cmd := exec.Command(binaryPath, "version")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Version command should succeed")

		outputStr := string(output)
		assert.Contains(t, outputStr, "Assistant", "Version should mention Assistant")
		assert.Contains(t, outputStr, "0.1.0", "Version should show version number")
	})

	t.Run("help_command", func(t *testing.T) {
		// Test help command
		cmd := exec.Command(binaryPath, "help")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Help command should succeed")

		outputStr := string(output)
		assert.Contains(t, outputStr, "Usage:", "Help should show usage")
		assert.Contains(t, outputStr, "Commands:", "Help should show commands")
		assert.Contains(t, outputStr, "serve", "Help should mention serve command")
		assert.Contains(t, outputStr, "cli", "Help should mention cli command")
	})
}

// TestErrorHandlingWorkflow tests error handling in E2E scenarios
func TestErrorHandlingWorkflow(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	suite.StartServer(t)

	t.Run("invalid_query_request", func(t *testing.T) {
		// Test invalid query request
		invalidReq := map[string]interface{}{
			"invalid_field": "invalid_value",
		}

		reqBody, err := json.Marshal(invalidReq)
		require.NoError(t, err)

		resp, err := http.Post(
			suite.serverURL+"/api/v1/query",
			"application/json",
			bytes.NewBuffer(reqBody),
		)
		require.NoError(t, err, "Request should be sent")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Invalid request should return 400")
	})

	t.Run("malformed_json", func(t *testing.T) {
		// Test malformed JSON
		resp, err := http.Post(
			suite.serverURL+"/api/v1/query",
			"application/json",
			strings.NewReader("invalid json"),
		)
		require.NoError(t, err, "Request should be sent")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Malformed JSON should return 400")
	})

	t.Run("non_existent_endpoint", func(t *testing.T) {
		// Test non-existent endpoint
		resp, err := http.Get(suite.serverURL + "/api/v1/nonexistent")
		require.NoError(t, err, "Request should be sent")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode, "Non-existent endpoint should return 404")
	})
}

// TestPerformanceWorkflow tests performance characteristics in E2E scenarios
func TestPerformanceWorkflow(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	suite.StartServer(t)

	t.Run("concurrent_requests", func(t *testing.T) {
		// Test concurrent request handling
		numRequests := 10
		done := make(chan bool, numRequests)
		errors := make(chan error, numRequests)

		queryReq := map[string]interface{}{
			"query": "Test concurrent request",
		}
		reqBody, err := json.Marshal(queryReq)
		require.NoError(t, err)

		start := time.Now()

		for i := 0; i < numRequests; i++ {
			go func(index int) {
				resp, err := http.Post(
					suite.serverURL+"/api/v1/query",
					"application/json",
					bytes.NewBuffer(reqBody),
				)
				if err != nil {
					errors <- err
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					errors <- fmt.Errorf("request %d failed with status %d", index, resp.StatusCode)
					return
				}

				done <- true
			}(i)
		}

		// Wait for all requests to complete
		successCount := 0
		for i := 0; i < numRequests; i++ {
			select {
			case <-done:
				successCount++
			case err := <-errors:
				t.Logf("Request failed: %v", err)
			case <-time.After(30 * time.Second):
				t.Error("Timeout waiting for concurrent requests")
				return
			}
		}

		duration := time.Since(start)
		t.Logf("Processed %d concurrent requests in %v", successCount, duration)

		assert.Equal(t, numRequests, successCount, "All concurrent requests should succeed")
		assert.Less(t, duration, 30*time.Second, "Concurrent requests should complete within reasonable time")
	})

	t.Run("response_time_measurement", func(t *testing.T) {
		// Measure response times
		queryReq := map[string]interface{}{
			"query": "Simple test query",
		}
		reqBody, err := json.Marshal(queryReq)
		require.NoError(t, err)

		var responseTimes []time.Duration
		numSamples := 5

		for i := 0; i < numSamples; i++ {
			start := time.Now()

			resp, err := http.Post(
				suite.serverURL+"/api/v1/query",
				"application/json",
				bytes.NewBuffer(reqBody),
			)
			require.NoError(t, err, "Request should succeed")

			// Read response to ensure complete processing
			_, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			require.NoError(t, err, "Should read response")

			duration := time.Since(start)
			responseTimes = append(responseTimes, duration)

			time.Sleep(100 * time.Millisecond) // Small delay between requests
		}

		// Calculate average response time
		var total time.Duration
		for _, rt := range responseTimes {
			total += rt
		}
		avgResponseTime := total / time.Duration(numSamples)

		t.Logf("Average response time: %v", avgResponseTime)
		assert.Less(t, avgResponseTime, 5*time.Second, "Average response time should be reasonable")
	})
}

// createTestConfig creates a test configuration file
func createTestConfig(configFile, databaseURL string) error {
	config := fmt.Sprintf(`
server:
  address: ":8081"
  timeout: 30s

database:
  url: "%s"
  max_connections: 5
  conn_max_lifetime: 1h

ai:
  default_provider: "claude"
  claude:
    api_key: "test-key"
    model: "claude-3-haiku-20240307"

assistant:
  max_concurrent_requests: 10
  request_timeout: 30s

cli:
  prompt_template: "assistant> "
  history_file: ".assistant_history"
  max_history_size: 1000

logging:
  level: "debug"
  format: "json"
`, databaseURL)

	return os.WriteFile(configFile, []byte(config), 0644)
}
