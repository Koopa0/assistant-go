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

	"github.com/koopa0/assistant-go/internal/platform/storage/postgres" // Added for postgres.DB
)

// Service handles all conversation-related operations
// Merges the functionality of Repository and Manager into a single service
type Service struct {
	db      postgres.DB // Changed from sqlc.Querier to postgres.DB for transaction support
	logger  *slog.Logger
}

// NewService creates a new conversation service
func NewService(db postgres.DB, logger *slog.Logger) *Service { // Parameter changed
	return &Service{
		db:      db,
		logger:  logger,
	}
}

// NewConversationSystem creates a new conversation service (backward compatibility)
func NewConversationSystem(db postgres.DB, logger *slog.Logger) *Service { // Parameter changed
	return NewService(db, logger)
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
	row, err := s.db.GetQueries().CreateConversation(ctx, params) // Use s.db.GetQueries()
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
	row, err := s.db.GetQueries().GetConversation(ctx, pgtype.UUID{Bytes: id, Valid: true}) // Use s.db.GetQueries()
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
	rows, err := s.db.GetQueries().GetConversationsByUser(ctx, pgtype.UUID{Bytes: uid, Valid: true}) // Use s.db.GetQueries()
	rows, err := s.db.GetQueries().GetConversationsByUser(ctx, pgtype.UUID{Bytes: uid, Valid: true}) // Use s.db.GetQueries()
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}

	// Convert all rows and filter active conversations.
	// If an individual conversation entry fails to convert (e.g., due to malformed metadata),
	// the error is logged, and that specific entry is skipped. The operation
	// will still return a list of successfully converted conversations.
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

	// The original GetConversation call is kept outside the transaction for now.
	// If GetConversation itself needed to be part of the same transaction (e.g., for read-committed),
	// it would also need to accept a Querier or pgx.Tx.
	conv, err := s.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get conversation for validation: %w", err)
	}
	if conv.IsArchived {
		return nil, fmt.Errorf("cannot add message to archived conversation")
	}

	var createdMessage *Message

	// Use a transaction to ensure atomicity of creating a message
	// and updating the parent conversation's metadata.
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		qtx := s.db.GetQueries().WithTx(tx) // Get transaction-aware querier

		// Create message with initial metadata
		msg := &Message{
			ConversationID: conversationID,
			Role:           role,
			Content:        content,
			Metadata:       MessageMetadata{}, // Will be populated by AI service
		}
		metadataJSON, err := json.Marshal(msg.Metadata)
		if err != nil {
			return fmt.Errorf("marshal message metadata in tx: %w", err)
		}
		convUUID, err := uuid.Parse(conversationID)
		if err != nil {
			return fmt.Errorf("invalid conversation ID in tx: %w", err)
		}
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

		// Execute CreateMessage query using transaction-aware querier (qtx)
		row, err := qtx.CreateMessage(ctx, params)
		if err != nil {
			return fmt.Errorf("create message in tx: %w", err)
		}
		createdMsgInternal, err := s.toDomainMessageFromCreateRow(row)
		if err != nil {
			return fmt.Errorf("convert message in tx: %w", err)
		}
		createdMessage = createdMsgInternal // Assign to outer scope variable

		// Update conversation metadata (in-memory part from the pre-transaction read)
		// This uses the 'conv' object fetched before the transaction.
		// If other transactions could modify 'conv' concurrently, this might lead to stale data.
		// For higher consistency, 'conv' could be re-fetched (SELECT FOR UPDATE) within the transaction.
		// However, for typical chat message additions, this level might be acceptable.
		conv.Metadata.MessageCount++
		conv.Metadata.LastActive = time.Now()

		// Update conversation in DB using transaction-aware querier (qtx)
		metadataJSONForUpdate, err := json.Marshal(conv.Metadata)
		if err != nil {
			return fmt.Errorf("marshal updated conv metadata in tx: %w", err)
		}
		// conv.ID is the same as conversationID, which was parsed into convUUID
		updateParams := sqlc.UpdateConversationParams{
			ID:       pgtype.UUID{Bytes: convUUID, Valid: true}, // Use convUUID from above
			Title:    conv.Title,                               // Title might not change here, ensure it's correct
			Metadata: metadataJSONForUpdate,
		}
		_, err = qtx.UpdateConversation(ctx, updateParams)
		if err != nil {
			return fmt.Errorf("update conversation metadata in tx: %w", err)
		}
		return nil // Commit transaction
	})

	if err != nil {
		// Log the top-level transaction error. Specific errors inside were already wrapped.
		s.logger.Error("Failed to add message with transaction", slog.String("conversation_id", conversationID), slog.Any("error", err))
		return nil, err // Return the error from WithTransaction
	}
	return createdMessage, nil
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
	rows, err := s.db.GetQueries().GetAllMessagesByConversation(ctx, pgtype.UUID{Bytes: id, Valid: true}) // Use s.db.GetQueries()
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

	_, err = s.db.GetQueries().UpdateConversation(ctx, params) // Use s.db.GetQueries()
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

	err = s.db.GetQueries().ArchiveConversation(ctx, pgtype.UUID{Bytes: id, Valid: true}) // Use s.db.GetQueries()
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

	err = s.db.GetQueries().DeleteConversation(ctx, pgtype.UUID{Bytes: id, Valid: true}) // Use s.db.GetQueries()
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
