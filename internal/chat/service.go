// Package chat provides chat services for the Assistant API server.
package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/conversation"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/user"
)

// ChatService handles chat completion and conversation logic
type ChatService struct {
	assistant *assistant.Assistant
	logger    *slog.Logger
	metrics   *observability.Metrics
}

// NewChatService creates a new chat service
func NewChatService(assistant *assistant.Assistant, logger *slog.Logger, metrics *observability.Metrics) *ChatService {
	return &ChatService{
		assistant: assistant,
		logger:    observability.ServerLogger(logger, "chat_service"),
		metrics:   metrics,
	}
}

// ChatCompletionRequest represents a chat completion request (OpenAI compatible)
type ChatCompletionRequest struct {
	Model          string                 `json:"model"`
	Messages       []ChatMessage          `json:"messages"`
	Temperature    *float64               `json:"temperature,omitempty"`
	MaxTokens      *int                   `json:"max_tokens,omitempty"`
	Stream         bool                   `json:"stream,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	SystemContext  map[string]interface{} `json:"system_context,omitempty"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ChatCompletionResponse represents a chat completion response
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage represents token usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ProcessChatCompletion processes a chat completion request
func (s *ChatService) ProcessChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// Extract the last user message
	var userQuery string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userQuery = req.Messages[i].Content
			break
		}
	}

	if userQuery == "" {
		return nil, fmt.Errorf("no user message found")
	}

	// Build context from conversation history
	conversationContext := make(map[string]interface{})
	if req.SystemContext != nil {
		conversationContext = req.SystemContext
	}

	// Add conversation history to context
	if len(req.Messages) > 1 {
		history := make([]map[string]string, 0, len(req.Messages)-1)
		for i := 0; i < len(req.Messages)-1; i++ {
			history = append(history, map[string]string{
				"role":    req.Messages[i].Role,
				"content": req.Messages[i].Content,
			})
		}
		conversationContext["history"] = history
	}

	// Create assistant query request
	userID := user.GetUserID(ctx)
	if userID == "" {
		return nil, errors.New("no authenticated user")
	}
	queryReq := &assistant.QueryRequest{
		Query:          userQuery,
		UserID:         &userID,
		ConversationID: &req.ConversationID,
		Context:        conversationContext,
	}

	// Process through assistant
	resp, err := s.assistant.ProcessQueryRequest(ctx, queryReq)
	if err != nil {
		return nil, fmt.Errorf("failed to process query: %w", err)
	}

	// Build OpenAI-compatible response
	response := &ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", resp.MessageID),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   resp.Model,
		Choices: []Choice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: resp.Response,
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     estimateTokens(userQuery),
			CompletionTokens: estimateTokens(resp.Response),
			TotalTokens:      estimateTokens(userQuery) + estimateTokens(resp.Response),
		},
	}

	// Update conversation ID if new
	if req.ConversationID == "" && resp.ConversationID != "" {
		req.ConversationID = resp.ConversationID
	}

	return response, nil
}

// ListConversations returns a list of conversations
func (s *ChatService) ListConversations(ctx context.Context, userID string, limit, offset int) ([]*conversation.Conversation, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	conversations, err := s.assistant.ListConversations(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	return conversations, nil
}

// GetConversation retrieves a specific conversation
func (s *ChatService) GetConversation(ctx context.Context, conversationID string) (*conversation.Conversation, error) {
	conversation, err := s.assistant.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	return conversation, nil
}

// GetWorkingMemory retrieves working memory content
func (s *ChatService) GetWorkingMemory(ctx context.Context, userID string) (map[string]interface{}, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	// TODO: Implement actual working memory retrieval
	workingMemory := map[string]interface{}{
		"user_id": userID,
		"context": map[string]interface{}{
			"current_task":     "API interaction",
			"recent_topics":    []string{"chat completion", "memory management"},
			"active_tools":     []string{"web_search", "code_analyzer"},
			"conversation_ids": []string{},
		},
		"short_term_items": []map[string]interface{}{
			{
				"type":      "query",
				"content":   "Recent chat query",
				"timestamp": time.Now().Add(-5 * time.Minute),
				"relevance": 0.9,
			},
		},
		"capacity_used": 0.45,
		"last_updated":  time.Now(),
	}

	return workingMemory, nil
}

// ListConcepts returns knowledge concepts
func (s *ChatService) ListConcepts(ctx context.Context, conceptType string, limit int) ([]map[string]interface{}, error) {
	// TODO: Implement actual concept retrieval
	concepts := []map[string]interface{}{
		{
			"id":          "concept-1",
			"name":        "Go Programming",
			"type":        "technology",
			"description": "Modern programming language for building scalable systems",
			"related_concepts": []string{
				"concurrency",
				"interfaces",
				"goroutines",
			},
			"confidence":  0.95,
			"usage_count": 150,
		},
		{
			"id":          "concept-2",
			"name":        "Microservices Architecture",
			"type":        "architecture",
			"description": "Architectural pattern for building distributed systems",
			"related_concepts": []string{
				"api_gateway",
				"service_mesh",
				"distributed_tracing",
			},
			"confidence":  0.88,
			"usage_count": 89,
		},
		{
			"id":          "concept-3",
			"name":        "Test-Driven Development",
			"type":        "methodology",
			"description": "Software development process relying on test-first approach",
			"related_concepts": []string{
				"unit_testing",
				"red_green_refactor",
				"test_coverage",
			},
			"confidence":  0.92,
			"usage_count": 120,
		},
	}

	// Filter by type if specified
	if conceptType != "" {
		filtered := []map[string]interface{}{}
		for _, concept := range concepts {
			if concept["type"] == conceptType {
				filtered = append(filtered, concept)
			}
		}
		concepts = filtered
	}

	// Apply limit
	if limit > 0 && len(concepts) > limit {
		concepts = concepts[:limit]
	}

	return concepts, nil
}

// ListTools returns available tools
func (s *ChatService) ListTools(ctx context.Context) ([]map[string]interface{}, error) {
	// Get tools from assistant
	availableTools := s.assistant.GetAvailableTools()

	// Convert to API response format
	tools := []map[string]interface{}{}
	for _, toolInfo := range availableTools {
		tools = append(tools, map[string]interface{}{
			"name":        toolInfo.Name,
			"description": toolInfo.Description,
			"category":    toolInfo.Category,
			"available":   true,
			"usage_count": 0, // TODO: Track actual usage
			"version":     toolInfo.Version,
			"parameters":  toolInfo.Parameters,
		})
	}

	return tools, nil
}

// Search performs a semantic search
func (s *ChatService) Search(ctx context.Context, query, searchType string, limit int) ([]map[string]interface{}, error) {
	// TODO: Implement actual semantic search
	results := []map[string]interface{}{
		{
			"id":        "result-1",
			"type":      "conversation",
			"title":     "Discussion about Go concurrency",
			"snippet":   "...using goroutines and channels for concurrent processing...",
			"relevance": 0.92,
			"timestamp": time.Now().Add(-24 * time.Hour),
			"metadata": map[string]interface{}{
				"conversation_id": "conv-123",
				"message_count":   15,
			},
		},
		{
			"id":        "result-2",
			"type":      "knowledge",
			"title":     "Best practices for error handling",
			"snippet":   "...always wrap errors with context using fmt.Errorf...",
			"relevance": 0.85,
			"timestamp": time.Now().Add(-48 * time.Hour),
			"metadata": map[string]interface{}{
				"concept_id": "concept-456",
				"category":   "best_practice",
			},
		},
	}

	// Filter by type if specified
	if searchType != "" {
		filtered := []map[string]interface{}{}
		for _, result := range results {
			if result["type"] == searchType {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// Helper functions

func estimateTokens(text string) int {
	// Rough estimation: 1 token ≈ 4 characters
	return len(text) / 4
}
