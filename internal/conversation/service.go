// Package conversation manages conversation state, message history, and context tracking.
// It provides persistence, retrieval, and management of conversations between users and the AI assistant,
// including metadata tracking, archiving, and conversation statistics.
package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// Service handles all conversation-related operations
// Merges the functionality of Repository and Manager into a single service
type Service struct {
	queries sqlc.Querier
	logger  *slog.Logger
}

// NewService creates a new conversation service
func NewService(queries sqlc.Querier, logger *slog.Logger) *Service {
	return &Service{
		queries: queries,
		logger:  logger,
	}
}

// NewConversationSystem creates a new conversation service (backward compatibility)
func NewConversationSystem(queries sqlc.Querier, logger *slog.Logger) *Service {
	return NewService(queries, logger)
}

// CreateConversation creates a new conversation with business logic and database operations
func (s *Service) CreateConversation(ctx context.Context, userID, title string) (*Conversation, error) {
	// Validation (business logic)
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if title == "" {
		title = "New Conversation"
	}

	// Create metadata with business defaults
	metadata := ConversationMetadata{
		MessageCount: 0,
		TokensUsed:   0,
		LastActive:   time.Now(),
	}

	// Marshal metadata
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	// Parse user UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Create database params
	params := sqlc.CreateConversationParams{
		UserID:   pgtype.UUID{Bytes: userUUID, Valid: true},
		Title:    title,
		Summary:  pgtype.Text{Valid: false},
		Metadata: metadataJSON,
	}

	// Execute query
	row, err := s.queries.CreateConversation(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}

	// Convert to domain type
	conv, err := s.toDomainConversation(row)
	if err != nil {
		return nil, fmt.Errorf("convert conversation: %w", err)
	}

	s.logger.Info("Created conversation",
		slog.String("conversation_id", conv.ID),
		slog.String("user_id", userID),
		slog.String("title", title))

	return conv, nil
}

// GetConversation retrieves a conversation by ID
func (s *Service) GetConversation(ctx context.Context, conversationID string) (*Conversation, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("conversation ID is required")
	}

	// Parse conversation UUID
	id, err := uuid.Parse(conversationID)
	if err != nil {
		return nil, fmt.Errorf("invalid conversation ID: %w", err)
	}

	// Execute query
	row, err := s.queries.GetConversation(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ConversationNotFoundError{ConversationID: conversationID}
		}
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	// Convert to domain type
	return s.toDomainConversation(row)
}

// ListConversations retrieves all active conversations for a user
func (s *Service) ListConversations(ctx context.Context, userID string) ([]*Conversation, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Parse user UUID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Execute query
	rows, err := s.queries.GetConversationsByUser(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}

	// Convert all rows and filter active conversations
	conversations := make([]*Conversation, 0)
	for _, row := range rows {
		conv, err := s.toDomainConversation(row)
		if err != nil {
			s.logger.Warn("Failed to convert conversation",
				slog.String("conversation_id", row.ID.String()),
				slog.Any("error", err))
			continue
		}
		// Business logic: filter out archived conversations by default
		if !conv.IsArchived {
			conversations = append(conversations, conv)
		}
	}

	return conversations, nil
}

// ListAllConversations retrieves all conversations including archived
func (s *Service) ListAllConversations(ctx context.Context, userID string) ([]*Conversation, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Parse user UUID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Execute query
	rows, err := s.queries.GetConversationsByUser(ctx, pgtype.UUID{Bytes: uid, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}

	// Convert all rows
	conversations := make([]*Conversation, 0, len(rows))
	for _, row := range rows {
		conv, err := s.toDomainConversation(row)
		if err != nil {
			s.logger.Warn("Failed to convert conversation",
				slog.String("conversation_id", row.ID.String()),
				slog.Any("error", err))
			continue
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// AddMessage adds a message to a conversation with business rules
func (s *Service) AddMessage(ctx context.Context, conversationID, role, content string) (*Message, error) {
	// Validate inputs
	if conversationID == "" {
		return nil, fmt.Errorf("conversation ID is required")
	}
	if role == "" {
		return nil, fmt.Errorf("role is required")
	}
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	// Validate role
	if role != "user" && role != "assistant" && role != "system" {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	// Validate conversation exists and is not archived
	conv, err := s.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	if conv.IsArchived {
		return nil, fmt.Errorf("cannot add message to archived conversation")
	}

	// Create message with initial metadata
	msg := &Message{
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		Metadata:       MessageMetadata{}, // Will be populated by AI service
	}

	// Marshal metadata
	metadataJSON, err := json.Marshal(msg.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	// Parse conversation UUID
	convUUID, err := uuid.Parse(conversationID)
	if err != nil {
		return nil, fmt.Errorf("invalid conversation ID: %w", err)
	}

	// Prepare token count
	var tokenCount pgtype.Int4
	if msg.TokenCount != nil {
		tokenCount = pgtype.Int4{Int32: int32(*msg.TokenCount), Valid: true}
	}

	params := sqlc.CreateMessageParams{
		ConversationID: pgtype.UUID{Bytes: convUUID, Valid: true},
		Role:           msg.Role,
		Content:        msg.Content,
		Metadata:       metadataJSON,
		TokenCount:     tokenCount,
	}

	// Execute query
	row, err := s.queries.CreateMessage(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create message: %w", err)
	}

	// Convert to domain type
	created, err := s.toDomainMessageFromCreateRow(row)
	if err != nil {
		return nil, fmt.Errorf("convert message: %w", err)
	}

	// Update conversation metadata
	conv.Metadata.MessageCount++
	conv.Metadata.LastActive = time.Now()

	// Update conversation
	if err := s.UpdateConversation(ctx, conv); err != nil {
		s.logger.Error("Failed to update conversation metadata",
			slog.String("conversation_id", conversationID),
			slog.Any("error", err))
		// Don't fail the operation
	}

	return created, nil
}

// GetMessages retrieves all messages for a conversation
func (s *Service) GetMessages(ctx context.Context, conversationID string) ([]*Message, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("conversation ID is required")
	}

	// Verify conversation exists
	_, err := s.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	// Parse UUID
	id, err := uuid.Parse(conversationID)
	if err != nil {
		return nil, fmt.Errorf("invalid conversation ID: %w", err)
	}

	// Execute query
	rows, err := s.queries.GetAllMessagesByConversation(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	// Convert all
	messages := make([]*Message, 0, len(rows))
	for _, row := range rows {
		msg, err := s.toDomainMessageFromGetRow(row)
		if err != nil {
			s.logger.Warn("Failed to convert message",
				slog.String("message_id", row.ID.String()),
				slog.Any("error", err))
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// UpdateConversation updates a conversation's metadata
func (s *Service) UpdateConversation(ctx context.Context, conv *Conversation) error {
	// Marshal metadata
	metadataJSON, err := json.Marshal(conv.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	// Parse UUID
	id, err := uuid.Parse(conv.ID)
	if err != nil {
		return fmt.Errorf("invalid conversation ID: %w", err)
	}

	params := sqlc.UpdateConversationParams{
		ID:       pgtype.UUID{Bytes: id, Valid: true},
		Title:    conv.Title,
		Metadata: metadataJSON,
	}

	_, err = s.queries.UpdateConversation(ctx, params)
	return err
}

// ArchiveConversation archives a conversation
func (s *Service) ArchiveConversation(ctx context.Context, conversationID string) error {
	if conversationID == "" {
		return fmt.Errorf("conversation ID is required")
	}

	// Verify conversation exists
	conv, err := s.GetConversation(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("get conversation: %w", err)
	}

	if conv.IsArchived {
		return fmt.Errorf("conversation already archived")
	}

	id, err := uuid.Parse(conversationID)
	if err != nil {
		return fmt.Errorf("invalid conversation ID: %w", err)
	}

	err = s.queries.ArchiveConversation(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return fmt.Errorf("archive conversation: %w", err)
	}

	s.logger.Info("Archived conversation",
		slog.String("conversation_id", conversationID))

	return nil
}

// DeleteConversation permanently deletes a conversation and all its messages
func (s *Service) DeleteConversation(ctx context.Context, conversationID string) error {
	if conversationID == "" {
		return fmt.Errorf("conversation ID is required")
	}

	// Verify conversation exists
	conv, err := s.GetConversation(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("get conversation: %w", err)
	}

	// Business rule: must be archived first
	if !conv.IsArchived {
		return fmt.Errorf("conversation must be archived before deletion")
	}

	id, err := uuid.Parse(conversationID)
	if err != nil {
		return fmt.Errorf("invalid conversation ID: %w", err)
	}

	err = s.queries.DeleteConversation(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}

	s.logger.Info("Deleted conversation",
		slog.String("conversation_id", conversationID))

	return nil
}

// GetConversationStats returns statistics for a conversation
func (s *Service) GetConversationStats(ctx context.Context, conversationID string) (*ConversationStats, error) {
	conv, err := s.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	messages, err := s.GetMessages(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	// Calculate statistics
	stats := &ConversationStats{
		ConversationID: conversationID,
		MessageCount:   len(messages),
		TokensUsed:     conv.Metadata.TokensUsed,
		CreatedAt:      conv.CreatedAt,
		LastActive:     conv.Metadata.LastActive,
		IsArchived:     conv.IsArchived,
	}

	// Count messages by role
	stats.UserMessages = 0
	stats.AssistantMessages = 0
	for _, msg := range messages {
		switch msg.Role {
		case "user":
			stats.UserMessages++
		case "assistant":
			stats.AssistantMessages++
		}
	}

	return stats, nil
}

// Private conversion methods - handle all sqlc type conversions here

func (s *Service) toDomainConversation(row *sqlc.Conversation) (*Conversation, error) {
	// Handle nil check
	if row == nil {
		return nil, fmt.Errorf("nil conversation row")
	}

	conv := &Conversation{
		ID:         row.ID.String(),
		UserID:     row.UserID.String(),
		Title:      row.Title,
		IsArchived: row.IsArchived.Bool,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}

	// Handle nullable summary
	if row.Summary.Valid {
		conv.Summary = &row.Summary.String
	}

	// Parse metadata with proper error handling
	var metadata ConversationMetadata
	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			s.logger.Warn("Failed to unmarshal conversation metadata",
				slog.String("conversation_id", conv.ID),
				slog.Any("error", err))
			// Use default metadata instead of failing
			metadata = ConversationMetadata{
				MessageCount: 0,
				TokensUsed:   0,
				LastActive:   conv.UpdatedAt,
			}
		}
	} else {
		// Initialize with defaults
		metadata = ConversationMetadata{
			MessageCount: 0,
			TokensUsed:   0,
			LastActive:   conv.UpdatedAt,
		}
	}
	conv.Metadata = metadata

	return conv, nil
}

func (s *Service) toDomainMessageFromCreateRow(row *sqlc.CreateMessageRow) (*Message, error) {
	// Handle nil check
	if row == nil {
		return nil, fmt.Errorf("nil message row")
	}

	msg := &Message{
		ID:             row.ID.String(),
		ConversationID: row.ConversationID.String(),
		Role:           row.Role,
		Content:        row.Content,
		CreatedAt:      row.CreatedAt,
	}

	// Handle nullable token count
	if row.TokenCount.Valid {
		count := int(row.TokenCount.Int32)
		msg.TokenCount = &count
	}

	// Parse metadata
	var metadata MessageMetadata
	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			s.logger.Warn("Failed to unmarshal message metadata",
				slog.String("message_id", msg.ID),
				slog.Any("error", err))
			// Use empty metadata
			metadata = MessageMetadata{}
		}
	}
	msg.Metadata = metadata

	return msg, nil
}

func (s *Service) toDomainMessageFromGetRow(row *sqlc.GetAllMessagesByConversationRow) (*Message, error) {
	// Handle nil check
	if row == nil {
		return nil, fmt.Errorf("nil message row")
	}

	msg := &Message{
		ID:             row.ID.String(),
		ConversationID: row.ConversationID.String(),
		Role:           row.Role,
		Content:        row.Content,
		CreatedAt:      row.CreatedAt,
	}

	// Handle nullable token count
	if row.TokenCount.Valid {
		count := int(row.TokenCount.Int32)
		msg.TokenCount = &count
	}

	// Parse metadata
	var metadata MessageMetadata
	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			s.logger.Warn("Failed to unmarshal message metadata",
				slog.String("message_id", msg.ID),
				slog.Any("error", err))
			// Use empty metadata
			metadata = MessageMetadata{}
		}
	}
	msg.Metadata = metadata

	return msg, nil
}
