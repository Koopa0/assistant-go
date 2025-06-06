package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
)

// Conversation represents a complete conversation with all its messages
// and metadata. Conversations provide continuity across multiple interactions
// and maintain context for better responses.
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

// Message represents a single message within a conversation.
// Messages can be from users or the assistant and include metadata
// for enhanced context tracking.
type Message struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversation_id"`
	Role           string                 `json:"role"`
	Content        string                 `json:"content"`
	Metadata       map[string]interface{} `json:"metadata"`
	TokenCount     *int                   `json:"token_count,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// Manager handles all conversation-related operations including:
// - Creating and managing conversations
// - Storing and retrieving messages
// - Maintaining conversation history
// - Archiving and cleanup operations
//
// It provides the persistence layer for maintaining conversation context across
// multiple interactions with users.
type Manager struct {
	db     postgres.DB  // Database client for persistence
	logger *slog.Logger // Structured logger for debugging
}

// NewManager creates a new conversation manager
func NewManager(db postgres.DB, logger *slog.Logger) (*Manager, error) {
	if db == nil {
		return nil, fmt.Errorf("database client is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &Manager{
		db:     db,
		logger: logger,
	}, nil
}

// CreateConversation creates a new conversation
func (cm *Manager) CreateConversation(ctx context.Context, userID, title string) (*Conversation, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
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
		return nil, NewConversationStorageFailureError("create_conversation", err)
	}

	conversation.Summary = summary
	if len(metadata) > 0 {
		// Parse metadata JSON
		if err := json.Unmarshal(metadata, &conversation.Metadata); err != nil {
			cm.logger.Warn("Failed to unmarshal conversation metadata",
				slog.String("conversation_id", conversation.ID),
				slog.Any("error", err))
			conversation.Metadata = make(map[string]interface{})
		}
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
func (cm *Manager) GetConversation(ctx context.Context, conversationID string) (*Conversation, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("conversation_id is required")
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
			return nil, NewConversationNotFoundError(conversationID)
		}
		return nil, NewConversationLoadFailureError(conversationID, err)
	}

	conversation.Summary = summary
	if len(metadata) > 0 {
		// Parse metadata JSON
		if err := json.Unmarshal(metadata, &conversation.Metadata); err != nil {
			cm.logger.Warn("Failed to unmarshal conversation metadata",
				slog.String("conversation_id", conversation.ID),
				slog.Any("error", err))
			conversation.Metadata = make(map[string]interface{})
		}
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
func (cm *Manager) ListConversations(ctx context.Context, userID string, limit, offset int) ([]*Conversation, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
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
		return nil, NewConversationStorageFailureError("list_conversations", err)
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
			return nil, NewConversationStorageFailureError("scan_conversation", err)
		}

		conversation.Summary = summary
		if len(metadata) > 0 {
			// Parse metadata JSON
			if err := json.Unmarshal(metadata, &conversation.Metadata); err != nil {
				cm.logger.Warn("Failed to unmarshal conversation metadata",
					slog.String("conversation_id", conversation.ID),
					slog.Any("error", err))
				conversation.Metadata = make(map[string]interface{})
			}
		} else {
			conversation.Metadata = make(map[string]interface{})
		}

		conversations = append(conversations, &conversation)
	}

	if err := rows.Err(); err != nil {
		return nil, NewConversationStorageFailureError("list_conversations_rows", err)
	}

	return conversations, nil
}

// AddMessage adds a message to a conversation
func (cm *Manager) AddMessage(ctx context.Context, conversationID, role, content string, metadata map[string]interface{}) (*Message, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("conversation_id is required")
	}
	if role == "" {
		return nil, fmt.Errorf("role is required")
	}
	if content == "" {
		return nil, fmt.Errorf("content is required")
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
		return nil, NewMessageCreationError(conversationID, role, err)
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
func (cm *Manager) GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("conversation_id is required")
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
		return nil, NewConversationStorageFailureError("get_messages", err)
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
			return nil, NewConversationStorageFailureError("scan_message", err)
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
		return nil, NewConversationStorageFailureError("get_messages_rows", err)
	}

	return messages, nil
}

// DeleteConversation deletes a conversation and all its messages
func (cm *Manager) DeleteConversation(ctx context.Context, conversationID string) error {
	if conversationID == "" {
		return fmt.Errorf("conversation_id is required")
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
			return NewConversationNotFoundError(conversationID)
		}

		return nil
	})
}

// ArchiveConversation archives a conversation
func (cm *Manager) ArchiveConversation(ctx context.Context, conversationID string) error {
	if conversationID == "" {
		return fmt.Errorf("conversation_id is required")
	}

	query := "UPDATE conversations SET is_archived = true, updated_at = $1 WHERE id = $2"
	result, err := cm.db.Exec(ctx, query, time.Now(), conversationID)
	if err != nil {
		return NewConversationStorageFailureError("archive_conversation", err)
	}

	if result.RowsAffected() == 0 {
		return NewConversationNotFoundError(conversationID)
	}

	cm.logger.Info("Archived conversation", slog.String("conversation_id", conversationID))
	return nil
}

// Close closes the context manager
func (cm *Manager) Close(ctx context.Context) error {
	// No cleanup needed for context manager
	return nil
}
