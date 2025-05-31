//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// APITestClient represents a test client for the API
type APITestClient struct {
	baseURL    string
	httpClient *http.Client
	suite      *E2ETestSuite
}

// NewAPITestClient creates a new API test client
func NewAPITestClient(suite *E2ETestSuite) *APITestClient {
	return &APITestClient{
		baseURL: suite.serverURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		suite: suite,
	}
}

// QueryRequest represents a query request to the API
type QueryRequest struct {
	Query          string                 `json:"query"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	UserID         string                 `json:"user_id,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Tools          []string               `json:"tools,omitempty"`
}

// QueryResponse represents a query response from the API
type QueryResponse struct {
	ID             string                 `json:"id"`
	Response       string                 `json:"response"`
	ConversationID string                 `json:"conversation_id"`
	Timestamp      time.Time              `json:"timestamp"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Tools          []ToolExecution        `json:"tools,omitempty"`
}

// ToolExecution represents a tool execution result
type ToolExecution struct {
	Name     string                 `json:"name"`
	Input    map[string]interface{} `json:"input"`
	Output   map[string]interface{} `json:"output"`
	Success  bool                   `json:"success"`
	Error    string                 `json:"error,omitempty"`
	Duration string                 `json:"duration"`
}

// ConversationRequest represents a conversation creation request
type ConversationRequest struct {
	Title  string                 `json:"title"`
	UserID string                 `json:"user_id,omitempty"`
	Tags   []string               `json:"tags,omitempty"`
	Meta   map[string]interface{} `json:"metadata,omitempty"`
}

// ConversationResponse represents a conversation response
type ConversationResponse struct {
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	UserID    string                 `json:"user_id"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Tags      []string               `json:"tags"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// TestAPIQueryEndpoint tests the query API endpoint
func TestAPIQueryEndpoint(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	suite.StartServer(t)
	client := NewAPITestClient(suite)

	t.Run("simple_query", func(t *testing.T) {
		req := QueryRequest{
			Query: "What is Go programming language?",
		}

		response, err := client.PostQuery(req)
		require.NoError(t, err, "Query should succeed")

		assert.NotEmpty(t, response.ID, "Response should have ID")
		assert.NotEmpty(t, response.Response, "Response should have content")
		assert.Contains(t, strings.ToLower(response.Response), "go", "Response should mention Go")
		assert.NotZero(t, response.Timestamp, "Response should have timestamp")
	})

	t.Run("query_with_context", func(t *testing.T) {
		req := QueryRequest{
			Query: "Analyze this code for potential issues",
			Context: map[string]interface{}{
				"language":  "go",
				"file_path": "./main.go",
				"project":   "test-project",
			},
		}

		response, err := client.PostQuery(req)
		require.NoError(t, err, "Query with context should succeed")

		assert.NotEmpty(t, response.Response, "Response should have content")
		responseText := strings.ToLower(response.Response)
		assert.True(t,
			strings.Contains(responseText, "analyze") ||
				strings.Contains(responseText, "code") ||
				strings.Contains(responseText, "go"),
			"Response should be relevant to code analysis")
	})

	t.Run("query_with_user_id", func(t *testing.T) {
		req := QueryRequest{
			Query:  "Hello, I'm a new user",
			UserID: "test-user-123",
		}

		response, err := client.PostQuery(req)
		require.NoError(t, err, "Query with user ID should succeed")

		assert.NotEmpty(t, response.Response, "Response should have content")
		assert.NotEmpty(t, response.ConversationID, "Response should have conversation ID")
	})

	t.Run("query_with_tools", func(t *testing.T) {
		req := QueryRequest{
			Query: "Check the status of my Go project",
			Tools: []string{"go-analyzer", "go-tester"},
		}

		response, err := client.PostQuery(req)
		require.NoError(t, err, "Query with tools should succeed")

		assert.NotEmpty(t, response.Response, "Response should have content")
		// Tools might not execute if not available, but request should succeed
	})

	t.Run("invalid_query_request", func(t *testing.T) {
		// Test empty query
		req := QueryRequest{
			Query: "",
		}

		_, err := client.PostQuery(req)
		assert.Error(t, err, "Empty query should fail")
	})

	t.Run("malformed_json", func(t *testing.T) {
		resp, err := http.Post(
			client.baseURL+"/api/v1/query",
			"application/json",
			strings.NewReader(`{"query": "test", "invalid": }`),
		)
		require.NoError(t, err, "Request should be sent")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Malformed JSON should return 400")
	})
}

// TestAPIConversationEndpoint tests the conversation API endpoints
func TestAPIConversationEndpoint(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	suite.StartServer(t)
	client := NewAPITestClient(suite)

	var conversationID string

	t.Run("create_conversation", func(t *testing.T) {
		req := ConversationRequest{
			Title:  "Test Conversation",
			UserID: "test-user-123",
			Tags:   []string{"test", "api"},
			Meta: map[string]interface{}{
				"project": "test-project",
			},
		}

		response, err := client.PostConversation(req)
		require.NoError(t, err, "Conversation creation should succeed")

		assert.NotEmpty(t, response.ID, "Conversation should have ID")
		assert.Equal(t, req.Title, response.Title, "Title should match")
		assert.Equal(t, req.UserID, response.UserID, "User ID should match")
		assert.Equal(t, req.Tags, response.Tags, "Tags should match")
		assert.NotZero(t, response.CreatedAt, "Should have creation time")

		conversationID = response.ID
	})

	t.Run("get_conversation", func(t *testing.T) {
		require.NotEmpty(t, conversationID, "Need conversation ID from previous test")

		response, err := client.GetConversation(conversationID)
		require.NoError(t, err, "Get conversation should succeed")

		assert.Equal(t, conversationID, response.ID, "ID should match")
		assert.Equal(t, "Test Conversation", response.Title, "Title should match")
	})

	t.Run("list_conversations", func(t *testing.T) {
		conversations, err := client.ListConversations()
		require.NoError(t, err, "List conversations should succeed")

		assert.NotEmpty(t, conversations, "Should have at least one conversation")

		// Find our test conversation
		found := false
		for _, conv := range conversations {
			if conv.ID == conversationID {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find our test conversation")
	})

	t.Run("query_in_conversation", func(t *testing.T) {
		require.NotEmpty(t, conversationID, "Need conversation ID")

		req := QueryRequest{
			Query:          "This is a follow-up question",
			ConversationID: conversationID,
		}

		response, err := client.PostQuery(req)
		require.NoError(t, err, "Query in conversation should succeed")

		assert.Equal(t, conversationID, response.ConversationID, "Should use existing conversation")
		assert.NotEmpty(t, response.Response, "Should have response")
	})

	t.Run("get_nonexistent_conversation", func(t *testing.T) {
		_, err := client.GetConversation("nonexistent-id")
		assert.Error(t, err, "Getting nonexistent conversation should fail")
	})
}

// TestAPIToolsEndpoint tests the tools API endpoints
func TestAPIToolsEndpoint(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	suite.StartServer(t)
	client := NewAPITestClient(suite)

	t.Run("list_tools", func(t *testing.T) {
		tools, err := client.ListTools()
		require.NoError(t, err, "List tools should succeed")

		assert.NotNil(t, tools, "Tools list should not be nil")
		// Tools list might be empty if no tools are registered
	})

	t.Run("execute_tool", func(t *testing.T) {
		toolReq := map[string]interface{}{
			"tool_name": "go-analyzer",
			"input": map[string]interface{}{
				"path": ".",
			},
		}

		// Tool execution might fail if tool is not available
		_, err := client.ExecuteTool(toolReq)
		// We don't require success since tools might not be available in test environment
		if err != nil {
			t.Logf("Tool execution failed (expected if tool not available): %v", err)
		}
	})

	t.Run("execute_invalid_tool", func(t *testing.T) {
		toolReq := map[string]interface{}{
			"tool_name": "nonexistent-tool",
			"input":     map[string]interface{}{},
		}

		_, err := client.ExecuteTool(toolReq)
		assert.Error(t, err, "Executing nonexistent tool should fail")
	})
}

// TestAPIHealthEndpoint tests the health check endpoint
func TestAPIHealthEndpoint(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	suite.StartServer(t)
	client := NewAPITestClient(suite)

	t.Run("health_check", func(t *testing.T) {
		health, err := client.GetHealth()
		require.NoError(t, err, "Health check should succeed")

		assert.Contains(t, health, "status", "Health should contain status")
		assert.Equal(t, "healthy", health["status"], "Status should be healthy")
	})
}

// TestAPIConcurrentRequests tests concurrent API requests
func TestAPIConcurrentRequests(t *testing.T) {
	suite := SetupE2ETestSuite(t)
	defer suite.Cleanup()

	suite.StartServer(t)
	client := NewAPITestClient(suite)

	t.Run("concurrent_queries", func(t *testing.T) {
		numRequests := 10
		done := make(chan *QueryResponse, numRequests)
		errors := make(chan error, numRequests)

		// Send concurrent queries
		for i := 0; i < numRequests; i++ {
			go func(index int) {
				req := QueryRequest{
					Query: fmt.Sprintf("Test concurrent query %d", index),
				}

				response, err := client.PostQuery(req)
				if err != nil {
					errors <- err
					return
				}
				done <- response
			}(i)
		}

		// Collect results
		successCount := 0
		for i := 0; i < numRequests; i++ {
			select {
			case response := <-done:
				assert.NotEmpty(t, response.Response, "Response should have content")
				successCount++
			case err := <-errors:
				t.Logf("Concurrent request failed: %v", err)
			case <-time.After(30 * time.Second):
				t.Error("Timeout waiting for concurrent requests")
				return
			}
		}

		assert.Equal(t, numRequests, successCount, "All concurrent requests should succeed")
	})
}

// PostQuery sends a query request to the API
func (c *APITestClient) PostQuery(req QueryRequest) (*QueryResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/v1/query",
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var response QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// PostConversation creates a new conversation
func (c *APITestClient) PostConversation(req ConversationRequest) (*ConversationResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/v1/conversations",
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var response ConversationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// GetConversation retrieves a conversation by ID
func (c *APITestClient) GetConversation(id string) (*ConversationResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/conversations/" + id)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var response ConversationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// ListConversations retrieves all conversations
func (c *APITestClient) ListConversations() ([]ConversationResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/conversations")
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var conversations []ConversationResponse
	if err := json.NewDecoder(resp.Body).Decode(&conversations); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return conversations, nil
}

// ListTools retrieves available tools
func (c *APITestClient) ListTools() ([]map[string]interface{}, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/v1/tools")
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var tools []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tools, nil
}

// ExecuteTool executes a tool
func (c *APITestClient) ExecuteTool(req map[string]interface{}) (map[string]interface{}, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/v1/tools/execute",
		"application/json",
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// GetHealth retrieves health status
func (c *APITestClient) GetHealth() (map[string]interface{}, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return health, nil
}
