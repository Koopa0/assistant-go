// Package conversation provides conversation management services for the Assistant API server.
package conversation

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// ConversationServiceImpl handles conversation management logic
type ConversationServiceImpl struct {
	assistant AssistantInterface
	queries   *sqlc.Queries
	logger    *slog.Logger
	metrics   *observability.Metrics
}

// NewConversationService creates a new conversation service
func NewConversationService(assistant AssistantInterface, queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *ConversationServiceImpl {
	return &ConversationServiceImpl{
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
// ListConversations returns conversations for a user (interface method)
func (s *ConversationServiceImpl) ListConversations(ctx context.Context, userID string) ([]*Conversation, error) {
	return s.assistant.ListConversations(ctx, userID, 100, 0)
}

// ListConversationsEnhanced returns all conversations with enhanced metadata
func (s *ConversationServiceImpl) ListConversationsEnhanced(ctx context.Context, params ListConversationsParams) ([]ConversationResponse, int, error) {
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

// GetConversation returns a conversation (interface method)
func (s *ConversationServiceImpl) GetConversation(ctx context.Context, conversationID string) (*Conversation, error) {
	return s.assistant.GetConversation(ctx, conversationID)
}

// GetConversationDetail returns a single conversation with messages
func (s *ConversationServiceImpl) GetConversationDetail(ctx context.Context, conversationID string) (*ConversationDetailResponse, error) {
	conversation, err := s.assistant.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	response, err := s.convertToDetailedResponse(ctx, conversation)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// CreateConversation creates a new conversation (interface method)
func (s *ConversationServiceImpl) CreateConversation(ctx context.Context, userID, title string) (*Conversation, error) {
	// Basic implementation for interface compliance
	conv := &Conversation{
		ID:         s.generateID(),
		UserID:     userID,
		Title:      title,
		Metadata:   ConversationMetadata{},
		IsArchived: false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	s.logger.Info("Created conversation",
		slog.String("conversation_id", conv.ID),
		slog.String("user_id", userID))

	return conv, nil
}

// CreateConversationWithMessage creates a new conversation and processes the initial message
func (s *ConversationServiceImpl) CreateConversationWithMessage(ctx context.Context, userID, title, initialMessage string, metadata map[string]interface{}) (*QueryResponse, error) {
	// Generate title if not provided
	if title == "" {
		title = s.generateTitle(initialMessage)
	}

	// Process initial message through assistant
	queryReq := &QueryRequest{
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
func (s *ConversationServiceImpl) SendMessage(ctx context.Context, conversationID, content string, attachments []interface{}) (*QueryResponse, error) {
	ctx2 := make(map[string]any)
	if len(attachments) > 0 {
		ctx2["attachments"] = attachments
	}

	queryReq := &QueryRequest{
		Query:          content,
		ConversationID: &conversationID,
		Context:        ctx2,
	}

	return s.assistant.ProcessQueryRequest(ctx, queryReq)
}

// DeleteConversation deletes a conversation
func (s *ConversationServiceImpl) DeleteConversation(ctx context.Context, conversationID string) error {
	return s.assistant.DeleteConversation(ctx, conversationID)
}

// ExportConversation exports a conversation in various formats
func (s *ConversationServiceImpl) ExportConversation(ctx context.Context, conversationID string) (*Conversation, error) {
	return s.assistant.GetConversation(ctx, conversationID)
}

// Helper methods

func (s *ConversationServiceImpl) convertToResponse(conv *Conversation) ConversationResponse {
	// Extract metadata - use defaults for now
	// TODO: Add these fields to ConversationMetadata struct if needed
	language := "en"
	toolsUsed := []string{}
	satisfaction := 0.0

	return ConversationResponse{
		ID:           conv.ID,
		Title:        conv.Title,
		LastMessage:  s.getLastMessage(conv),
		Timestamp:    conv.UpdatedAt,
		MessageCount: int(conv.Metadata.MessageCount),
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

func (s *ConversationServiceImpl) convertToDetailedResponse(ctx context.Context, conv *Conversation) (ConversationDetailResponse, error) {
	// Fetch messages separately
	messages, err := s.assistant.GetConversationMessages(ctx, conv.ID)
	if err != nil {
		return ConversationDetailResponse{}, fmt.Errorf("failed to get messages: %w", err)
	}

	messageResponses := []MessageResponse{}
	for _, msg := range messages {
		// Convert MessageMetadata to map for API response
		metadata := make(map[string]interface{})
		// TODO: Add MessageMetadata fields to map when needed

		messageResponses = append(messageResponses, MessageResponse{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.CreatedAt,
			Metadata:  metadata,
		})
	}

	insights := s.calculateInsights(ctx, conv, messages)

	return ConversationDetailResponse{
		ID:       conv.ID,
		Title:    conv.Title,
		Created:  conv.CreatedAt,
		Updated:  conv.UpdatedAt,
		Messages: messageResponses,
		Insights: insights,
		Metadata: s.convertMetadataToMap(conv.Metadata),
	}, nil
}

func (s *ConversationServiceImpl) convertMetadataToMap(metadata ConversationMetadata) map[string]interface{} {
	// Convert struct to map for API response
	result := make(map[string]interface{})

	// Add fields as needed
	if metadata.Context != "" {
		result["context"] = metadata.Context
	}
	if len(metadata.Tags) > 0 {
		result["tags"] = metadata.Tags
	}
	if metadata.Category != "" {
		result["category"] = metadata.Category
	}
	if metadata.Priority != "" {
		result["priority"] = metadata.Priority
	}
	if metadata.Model != "" {
		result["model"] = metadata.Model
	}
	if metadata.Temperature > 0 {
		result["temperature"] = metadata.Temperature
	}
	if metadata.MaxTokens > 0 {
		result["max_tokens"] = metadata.MaxTokens
	}
	result["message_count"] = metadata.MessageCount

	return result
}

func (s *ConversationServiceImpl) getLastMessage(conv *Conversation) string {
	// For now, return empty since we need context to fetch messages
	// TODO: Refactor to pass messages or context
	return ""
}

func (s *ConversationServiceImpl) getCategory(conv *Conversation) string {
	if conv.Metadata.Category != "" {
		return conv.Metadata.Category
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

func (s *ConversationServiceImpl) extractTags(conv *Conversation) []string {
	// Use tags from metadata if available
	tags := append([]string{}, conv.Metadata.Tags...)
	// Auto-generate tags from content
	if s.containsKeywords(conv, []string{"performance", "優化", "效能"}) {
		tags = append(tags, "效能優化")
	}
	if s.containsKeywords(conv, []string{"debug", "error", "錯誤", "除錯"}) {
		tags = append(tags, "除錯")
	}
	return tags
}

func (s *ConversationServiceImpl) getStatus(conv *Conversation) string {
	// Check priority field for status-like information
	if conv.Metadata.Priority != "" {
		return conv.Metadata.Priority
	}
	// Default status based on archive state
	if conv.IsArchived {
		return "archived"
	}
	return "active"
}

func (s *ConversationServiceImpl) containsKeywords(conv *Conversation, keywords []string) bool {
	// For now, just check title
	// TODO: Refactor to check messages when context is available
	content := strings.ToLower(conv.Title)
	for _, keyword := range keywords {
		if strings.Contains(content, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func (s *ConversationServiceImpl) calculateInsights(ctx context.Context, conv *Conversation, messages []*Message) ConversationInsights {
	// Calculate queries per day
	duration := time.Since(conv.CreatedAt)
	days := duration.Hours() / 24
	if days < 1 {
		days = 1
	}
	queriesPerDay := float64(len(messages)/2) / days

	// Count tools used
	toolsUsed := make(map[string]bool)
	codeSnippets := 0
	totalResponseTime := 0
	responseCount := 0

	for _, msg := range messages {
		// Count tools used
		for _, toolCall := range msg.Metadata.ToolCalls {
			toolsUsed[toolCall.ToolName] = true
		}

		// Track response times
		if msg.Metadata.ResponseTime > 0 {
			totalResponseTime += int(msg.Metadata.ResponseTime.Milliseconds())
			responseCount++
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
		ProblemsSolved:        len(messages) / 4, // Rough estimate
		AverageResponseTime:   avgResponseTime,
	}
}

func (s *ConversationServiceImpl) matchesSearch(conv *Conversation, search string) bool {
	search = strings.ToLower(search)
	// For now, just search in title
	// TODO: Refactor to search in messages when context is available
	return strings.Contains(strings.ToLower(conv.Title), search)
}

func (s *ConversationServiceImpl) sortConversations(conversations []ConversationResponse, sortBy, sortOrder string) {
	// Simple sorting implementation
	// In production, implement proper sorting logic
}

func (s *ConversationServiceImpl) generateTitle(message string) string {
	// Simple title generation
	if len(message) > 50 {
		return message[:50] + "..."
	}
	return message
}

// AddMessage adds a message to a conversation
func (s *ConversationServiceImpl) AddMessage(ctx context.Context, conversationID, role, content string) (*Message, error) {
	// For now, we'll create a basic message
	// In a real implementation, this would save to database
	message := &Message{
		ID:             s.generateID(),
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		Metadata:       MessageMetadata{},
		CreatedAt:      time.Now(),
	}

	s.logger.Info("Added message to conversation",
		slog.String("conversation_id", conversationID),
		slog.String("role", role),
		slog.String("message_id", message.ID))

	return message, nil
}

// generateID generates a unique ID
func (s *ConversationServiceImpl) generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ArchiveConversation archives a conversation
func (s *ConversationServiceImpl) ArchiveConversation(ctx context.Context, conversationID string) error {
	s.logger.Info("Archiving conversation", slog.String("conversation_id", conversationID))
	// In a real implementation, this would update the database
	// For now, just log and return success
	return nil
}

// UpdateConversation updates a conversation
func (s *ConversationServiceImpl) UpdateConversation(ctx context.Context, conv *Conversation) error {
	s.logger.Info("Updating conversation", slog.String("conversation_id", conv.ID))
	// In a real implementation, this would update the database
	// For now, just log and return success
	return nil
}

// GetMessages retrieves messages for a conversation
func (s *ConversationServiceImpl) GetMessages(ctx context.Context, conversationID string) ([]*Message, error) {
	return s.assistant.GetConversationMessages(ctx, conversationID)
}

// GetConversationStats returns statistics for a conversation
func (s *ConversationServiceImpl) GetConversationStats(ctx context.Context, conversationID string) (*ConversationStats, error) {
	// Mock implementation
	stats := &ConversationStats{
		ConversationID:    conversationID,
		MessageCount:      10,
		UserMessages:      5,
		AssistantMessages: 5,
		TokensUsed:        1000,
		CreatedAt:         time.Now().Add(-24 * time.Hour),
		LastActive:        time.Now(),
		IsArchived:        false,
	}
	return stats, nil
}
