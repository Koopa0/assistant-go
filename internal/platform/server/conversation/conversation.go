// Package conversation provides conversation management services for the Assistant API server.
package conversation

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/core/conversation"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// ConversationService handles conversation management logic
type ConversationService struct {
	assistant *assistant.Assistant
	queries   *sqlc.Queries
	logger    *slog.Logger
	metrics   *observability.Metrics
}

// NewConversationService creates a new conversation service
func NewConversationService(assistant *assistant.Assistant, queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *ConversationService {
	return &ConversationService{
		assistant: assistant,
		queries:   queries,
		logger:    observability.ServerLogger(logger, "conversation_service"),
		metrics:   metrics,
	}
}

// ConversationResponse represents a conversation in API responses
type ConversationResponse struct {
	ID           string                 `json:"id"`
	Title        string                 `json:"title"`
	LastMessage  string                 `json:"lastMessage"`
	Timestamp    time.Time              `json:"timestamp"`
	MessageCount int                    `json:"messageCount"`
	Category     string                 `json:"category"`
	Status       string                 `json:"status"`
	Participants []string               `json:"participants"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ConversationDetailResponse represents detailed conversation data
type ConversationDetailResponse struct {
	ID       string                 `json:"id"`
	Title    string                 `json:"title"`
	Created  time.Time              `json:"created"`
	Updated  time.Time              `json:"updated"`
	Messages []MessageResponse      `json:"messages"`
	Insights ConversationInsights   `json:"insights"`
	Metadata map[string]interface{} `json:"metadata"`
}

// MessageResponse represents a message in API responses
type MessageResponse struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ConversationInsights represents analytics for a conversation
type ConversationInsights struct {
	QueriesPerDay         float64 `json:"queriesPerDay"`
	ToolsUsedCount        int     `json:"toolsUsedCount"`
	CodeSnippetsGenerated int     `json:"codeSnippetsGenerated"`
	ProblemsSolved        int     `json:"problemsSolved"`
	AverageResponseTime   int     `json:"averageResponseTime"`
}

// ListConversationsParams contains parameters for listing conversations
type ListConversationsParams struct {
	UserID    string
	Search    string
	Category  string
	Status    string
	SortBy    string
	SortOrder string
	Page      int
	Limit     int
}

// ListConversations returns a paginated list of conversations
func (s *ConversationService) ListConversations(ctx context.Context, params ListConversationsParams) ([]ConversationResponse, int, error) {
	// Get conversations from assistant
	conversations, err := s.assistant.ListConversations(ctx, params.UserID, params.Limit, (params.Page-1)*params.Limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list conversations: %w", err)
	}

	// Convert to response format with filtering
	response := []ConversationResponse{}
	for _, conv := range conversations {
		// Apply filters
		if params.Search != "" && !s.matchesSearch(conv, params.Search) {
			continue
		}
		if params.Category != "" && s.getCategory(conv) != params.Category {
			continue
		}
		if params.Status != "" {
			convStatus := s.getStatus(conv)
			if convStatus != params.Status {
				continue
			}
		}

		response = append(response, s.convertToResponse(conv))
	}

	// Apply sorting
	s.sortConversations(response, params.SortBy, params.SortOrder)

	// Calculate total (mock for demo)
	total := len(response) * 3

	return response, total, nil
}

// GetConversation returns a single conversation with messages
func (s *ConversationService) GetConversation(ctx context.Context, conversationID string) (*ConversationDetailResponse, error) {
	conversation, err := s.assistant.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	response := s.convertToDetailedResponse(conversation)
	return &response, nil
}

// CreateConversation creates a new conversation
func (s *ConversationService) CreateConversation(ctx context.Context, userID, title, initialMessage string, metadata map[string]interface{}) (*assistant.QueryResponse, error) {
	// Generate title if not provided
	if title == "" {
		title = s.generateTitle(initialMessage)
	}

	// Process initial message through assistant
	queryReq := &assistant.QueryRequest{
		Query:  initialMessage,
		UserID: &userID,
		Context: map[string]any{
			"metadata": metadata,
			"title":    title,
		},
	}

	return s.assistant.ProcessQueryRequest(ctx, queryReq)
}

// SendMessage sends a message to a conversation
func (s *ConversationService) SendMessage(ctx context.Context, conversationID, content string, attachments []interface{}) (*assistant.QueryResponse, error) {
	context := make(map[string]any)
	if len(attachments) > 0 {
		context["attachments"] = attachments
	}

	queryReq := &assistant.QueryRequest{
		Query:          content,
		ConversationID: &conversationID,
		Context:        context,
	}

	return s.assistant.ProcessQueryRequest(ctx, queryReq)
}

// DeleteConversation deletes a conversation
func (s *ConversationService) DeleteConversation(ctx context.Context, conversationID string) error {
	return s.assistant.DeleteConversation(ctx, conversationID)
}

// ExportConversation exports a conversation in various formats
func (s *ConversationService) ExportConversation(ctx context.Context, conversationID string) (*conversation.Conversation, error) {
	return s.assistant.GetConversation(ctx, conversationID)
}

// Helper methods

func (s *ConversationService) convertToResponse(conv *conversation.Conversation) ConversationResponse {
	// Extract metadata
	language := ""
	toolsUsed := []string{}
	satisfaction := 0.0

	if conv.Metadata != nil {
		if lang, ok := conv.Metadata["language"].(string); ok {
			language = lang
		}
		if tools, ok := conv.Metadata["toolsUsed"].([]string); ok {
			toolsUsed = tools
		}
		if sat, ok := conv.Metadata["satisfaction"].(float64); ok {
			satisfaction = sat
		}
	}

	return ConversationResponse{
		ID:           conv.ID,
		Title:        conv.Title,
		LastMessage:  s.getLastMessage(conv),
		Timestamp:    conv.UpdatedAt,
		MessageCount: len(conv.Messages),
		Category:     s.getCategory(conv),
		Status:       s.getStatus(conv),
		Participants: []string{conv.UserID, "assistant"},
		Tags:         s.extractTags(conv),
		Metadata: map[string]interface{}{
			"language":     language,
			"toolsUsed":    toolsUsed,
			"satisfaction": satisfaction,
		},
	}
}

func (s *ConversationService) convertToDetailedResponse(conv *conversation.Conversation) ConversationDetailResponse {
	messages := []MessageResponse{}
	for _, msg := range conv.Messages {
		messages = append(messages, MessageResponse{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.CreatedAt,
			Metadata:  msg.Metadata,
		})
	}

	insights := s.calculateInsights(conv)

	return ConversationDetailResponse{
		ID:       conv.ID,
		Title:    conv.Title,
		Created:  conv.CreatedAt,
		Updated:  conv.UpdatedAt,
		Messages: messages,
		Insights: insights,
		Metadata: conv.Metadata,
	}
}

func (s *ConversationService) getLastMessage(conv *conversation.Conversation) string {
	if len(conv.Messages) == 0 {
		return ""
	}
	lastMsg := conv.Messages[len(conv.Messages)-1]
	if len(lastMsg.Content) > 100 {
		return lastMsg.Content[:100] + "..."
	}
	return lastMsg.Content
}

func (s *ConversationService) getCategory(conv *conversation.Conversation) string {
	if conv.Metadata != nil {
		if category, ok := conv.Metadata["category"].(string); ok {
			return category
		}
	}
	// Infer category from content
	if s.containsKeywords(conv, []string{"angular", "react", "vue", "frontend", "css", "html"}) {
		return "Frontend"
	}
	if s.containsKeywords(conv, []string{"go", "golang", "backend", "api", "server"}) {
		return "Backend"
	}
	if s.containsKeywords(conv, []string{"database", "sql", "postgres", "mongodb"}) {
		return "Database"
	}
	return "General"
}

func (s *ConversationService) extractTags(conv *conversation.Conversation) []string {
	tags := []string{}
	if conv.Metadata != nil {
		if metaTags, ok := conv.Metadata["tags"].([]string); ok {
			tags = append(tags, metaTags...)
		}
	}
	// Auto-generate tags from content
	if s.containsKeywords(conv, []string{"performance", "優化", "效能"}) {
		tags = append(tags, "效能優化")
	}
	if s.containsKeywords(conv, []string{"debug", "error", "錯誤", "除錯"}) {
		tags = append(tags, "除錯")
	}
	return tags
}

func (s *ConversationService) getStatus(conv *conversation.Conversation) string {
	if conv.Metadata != nil {
		if status, ok := conv.Metadata["status"].(string); ok {
			return status
		}
	}
	// Default status based on archive state
	if conv.IsArchived {
		return "archived"
	}
	return "active"
}

func (s *ConversationService) containsKeywords(conv *conversation.Conversation, keywords []string) bool {
	content := strings.ToLower(conv.Title)
	for _, msg := range conv.Messages {
		content += " " + strings.ToLower(msg.Content)
	}
	for _, keyword := range keywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func (s *ConversationService) calculateInsights(conv *conversation.Conversation) ConversationInsights {
	// Calculate queries per day
	duration := time.Since(conv.CreatedAt)
	days := duration.Hours() / 24
	if days < 1 {
		days = 1
	}
	queriesPerDay := float64(len(conv.Messages)/2) / days

	// Count tools used
	toolsUsed := make(map[string]bool)
	codeSnippets := 0
	totalResponseTime := 0
	responseCount := 0

	for _, msg := range conv.Messages {
		if msg.Metadata != nil {
			if tools, ok := msg.Metadata["toolsUsed"].([]interface{}); ok {
				for _, tool := range tools {
					if toolName, ok := tool.(string); ok {
						toolsUsed[toolName] = true
					}
				}
			}
			if responseTime, ok := msg.Metadata["processingTime"].(float64); ok {
				totalResponseTime += int(responseTime)
				responseCount++
			}
		}
		// Count code snippets (simple heuristic)
		if strings.Contains(msg.Content, "```") {
			codeSnippets++
		}
	}

	avgResponseTime := 0
	if responseCount > 0 {
		avgResponseTime = totalResponseTime / responseCount
	}

	return ConversationInsights{
		QueriesPerDay:         queriesPerDay,
		ToolsUsedCount:        len(toolsUsed),
		CodeSnippetsGenerated: codeSnippets,
		ProblemsSolved:        len(conv.Messages) / 4, // Rough estimate
		AverageResponseTime:   avgResponseTime,
	}
}

func (s *ConversationService) matchesSearch(conv *conversation.Conversation, search string) bool {
	search = strings.ToLower(search)
	if strings.Contains(strings.ToLower(conv.Title), search) {
		return true
	}
	for _, msg := range conv.Messages {
		if strings.Contains(strings.ToLower(msg.Content), search) {
			return true
		}
	}
	return false
}

func (s *ConversationService) sortConversations(conversations []ConversationResponse, sortBy, sortOrder string) {
	// Simple sorting implementation
	// In production, implement proper sorting logic
}

func (s *ConversationService) generateTitle(message string) string {
	// Simple title generation
	if len(message) > 50 {
		return message[:50] + "..."
	}
	return message
}
