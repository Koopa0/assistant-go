package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
)

// Conversation represents a conversation
type Conversation struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	Title      string                 `json:"title"`
	Summary    *string                `json:"summary,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	IsArchived bool                   `json:"is_archived"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Messages   []*Message             `json:"messages,omitempty"`
}

// Message represents a message in a conversation
type Message struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversation_id"`
	Role           string                 `json:"role"`
	Content        string                 `json:"content"`
	Metadata       map[string]interface{} `json:"metadata"`
	TokenCount     *int                   `json:"token_count,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// ContextManager manages conversation context and history
type ContextManager struct {
	db     *postgres.Client
	logger *slog.Logger
}

// NewContextManager creates a new context manager
func NewContextManager(db *postgres.Client, logger *slog.Logger) (*ContextManager, error) {
	if db == nil {
		return nil, fmt.Errorf("database client is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &ContextManager{
		db:     db,
		logger: logger,
	}, nil
}

// CreateConversation creates a new conversation
func (cm *ContextManager) CreateConversation(ctx context.Context, userID, title string) (*Conversation, error) {
	if userID == "" {
		return nil, NewInvalidInputError("user_id is required", nil)
	}
	if title == "" {
		title = "New Conversation"
	}

	conversationID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO conversations (id, user_id, title, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, title, summary, metadata, is_archived, created_at, updated_at
	`

	var conversation Conversation
	var metadata []byte
	var summary *string

	err := cm.db.QueryRow(ctx, query, conversationID, userID, title, now, now).Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.Title,
		&summary,
		&metadata,
		&conversation.IsArchived,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		return nil, NewDatabaseError("create_conversation", err)
	}

	conversation.Summary = summary
	if len(metadata) > 0 {
		// TODO: Unmarshal metadata JSON
		conversation.Metadata = make(map[string]interface{})
	} else {
		conversation.Metadata = make(map[string]interface{})
	}

	cm.logger.Info("Created conversation",
		slog.String("conversation_id", conversation.ID),
		slog.String("user_id", userID),
		slog.String("title", title))

	return &conversation, nil
}

// GetConversation retrieves a conversation by ID
func (cm *ContextManager) GetConversation(ctx context.Context, conversationID string) (*Conversation, error) {
	if conversationID == "" {
		return nil, NewInvalidInputError("conversation_id is required", nil)
	}

	query := `
		SELECT id, user_id, title, summary, metadata, is_archived, created_at, updated_at
		FROM conversations
		WHERE id = $1
	`

	var conversation Conversation
	var metadata []byte
	var summary *string

	err := cm.db.QueryRow(ctx, query, conversationID).Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.Title,
		&summary,
		&metadata,
		&conversation.IsArchived,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, NewContextNotFoundError(conversationID)
		}
		return nil, NewDatabaseError("get_conversation", err)
	}

	conversation.Summary = summary
	if len(metadata) > 0 {
		// TODO: Unmarshal metadata JSON
		conversation.Metadata = make(map[string]interface{})
	} else {
		conversation.Metadata = make(map[string]interface{})
	}

	// Load messages
	messages, err := cm.GetMessages(ctx, conversationID, 0, 100) // Default limit
	if err != nil {
		cm.logger.Warn("Failed to load messages for conversation",
			slog.String("conversation_id", conversationID),
			slog.Any("error", err))
	} else {
		conversation.Messages = messages
	}

	return &conversation, nil
}

// ListConversations lists conversations for a user
func (cm *ContextManager) ListConversations(ctx context.Context, userID string, limit, offset int) ([]*Conversation, error) {
	if userID == "" {
		return nil, NewInvalidInputError("user_id is required", nil)
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, user_id, title, summary, metadata, is_archived, created_at, updated_at
		FROM conversations
		WHERE user_id = $1 AND is_archived = false
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := cm.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, NewDatabaseError("list_conversations", err)
	}
	defer rows.Close()

	var conversations []*Conversation
	for rows.Next() {
		var conversation Conversation
		var metadata []byte
		var summary *string

		err := rows.Scan(
			&conversation.ID,
			&conversation.UserID,
			&conversation.Title,
			&summary,
			&metadata,
			&conversation.IsArchived,
			&conversation.CreatedAt,
			&conversation.UpdatedAt,
		)
		if err != nil {
			return nil, NewDatabaseError("scan_conversation", err)
		}

		conversation.Summary = summary
		if len(metadata) > 0 {
			// TODO: Unmarshal metadata JSON
			conversation.Metadata = make(map[string]interface{})
		} else {
			conversation.Metadata = make(map[string]interface{})
		}

		conversations = append(conversations, &conversation)
	}

	if err := rows.Err(); err != nil {
		return nil, NewDatabaseError("list_conversations_rows", err)
	}

	return conversations, nil
}

// AddMessage adds a message to a conversation
func (cm *ContextManager) AddMessage(ctx context.Context, conversationID, role, content string, metadata map[string]interface{}) (*Message, error) {
	if conversationID == "" {
		return nil, NewInvalidInputError("conversation_id is required", nil)
	}
	if role == "" {
		return nil, NewInvalidInputError("role is required", nil)
	}
	if content == "" {
		return nil, NewInvalidInputError("content is required", nil)
	}

	messageID := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO messages (id, conversation_id, role, content, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, conversation_id, role, content, metadata, token_count, created_at
	`

	var message Message
	var metadataBytes []byte
	var tokenCount *int

	// Marshal metadata to JSON
	metadataJSON := "{}"
	if metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			cm.logger.Warn("Failed to marshal message metadata, using empty object",
				slog.String("conversation_id", conversationID),
				slog.String("role", role),
				slog.Any("error", err))
		} else {
			metadataJSON = string(metadataBytes)
		}
	}

	err := cm.db.QueryRow(ctx, query, messageID, conversationID, role, content, metadataJSON, now).Scan(
		&message.ID,
		&message.ConversationID,
		&message.Role,
		&message.Content,
		&metadataBytes,
		&tokenCount,
		&message.CreatedAt,
	)
	if err != nil {
		return nil, NewDatabaseError("add_message", err)
	}

	message.TokenCount = tokenCount
	if metadata != nil {
		message.Metadata = metadata
	} else {
		message.Metadata = make(map[string]interface{})
	}

	// Update conversation updated_at timestamp
	_, err = cm.db.Exec(ctx, "UPDATE conversations SET updated_at = $1 WHERE id = $2", now, conversationID)
	if err != nil {
		cm.logger.Warn("Failed to update conversation timestamp",
			slog.String("conversation_id", conversationID),
			slog.Any("error", err))
	}

	cm.logger.Debug("Added message to conversation",
		slog.String("message_id", message.ID),
		slog.String("conversation_id", conversationID),
		slog.String("role", role))

	return &message, nil
}

// GetMessages retrieves messages for a conversation
func (cm *ContextManager) GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error) {
	if conversationID == "" {
		return nil, NewInvalidInputError("conversation_id is required", nil)
	}
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, conversation_id, role, content, metadata, token_count, created_at
		FROM messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := cm.db.Query(ctx, query, conversationID, limit, offset)
	if err != nil {
		return nil, NewDatabaseError("get_messages", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var message Message
		var metadata []byte
		var tokenCount *int

		err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.Role,
			&message.Content,
			&metadata,
			&tokenCount,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, NewDatabaseError("scan_message", err)
		}

		message.TokenCount = tokenCount
		message.Metadata = make(map[string]interface{})
		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &message.Metadata); err != nil {
				cm.logger.Warn("Failed to unmarshal message metadata",
					slog.String("message_id", message.ID),
					slog.String("metadata", string(metadata)),
					slog.Any("error", err))
			}
		}

		messages = append(messages, &message)
	}

	if err := rows.Err(); err != nil {
		return nil, NewDatabaseError("get_messages_rows", err)
	}

	return messages, nil
}

// DeleteConversation deletes a conversation and all its messages
func (cm *ContextManager) DeleteConversation(ctx context.Context, conversationID string) error {
	if conversationID == "" {
		return NewInvalidInputError("conversation_id is required", nil)
	}

	return cm.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		// Delete messages first (due to foreign key constraint)
		_, err := tx.Exec(ctx, "DELETE FROM messages WHERE conversation_id = $1", conversationID)
		if err != nil {
			return fmt.Errorf("failed to delete messages: %w", err)
		}

		// Delete conversation
		result, err := tx.Exec(ctx, "DELETE FROM conversations WHERE id = $1", conversationID)
		if err != nil {
			return fmt.Errorf("failed to delete conversation: %w", err)
		}

		if result.RowsAffected() == 0 {
			return NewContextNotFoundError(conversationID)
		}

		return nil
	})
}

// ArchiveConversation archives a conversation
func (cm *ContextManager) ArchiveConversation(ctx context.Context, conversationID string) error {
	if conversationID == "" {
		return NewInvalidInputError("conversation_id is required", nil)
	}

	query := "UPDATE conversations SET is_archived = true, updated_at = $1 WHERE id = $2"
	result, err := cm.db.Exec(ctx, query, time.Now(), conversationID)
	if err != nil {
		return NewDatabaseError("archive_conversation", err)
	}

	if result.RowsAffected() == 0 {
		return NewContextNotFoundError(conversationID)
	}

	cm.logger.Info("Archived conversation", slog.String("conversation_id", conversationID))
	return nil
}

// Close closes the context manager
func (cm *ContextManager) Close(ctx context.Context) error {
	// No cleanup needed for context manager
	return nil
}
