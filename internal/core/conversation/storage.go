package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
)

// ConversationStorage defines what the conversation package actually needs
// Small interface discovered from actual usage in Manager
type ConversationStorage interface {
	// Core conversation operations
	CreateConversation(ctx context.Context, userID, title string) (*Conversation, error)
	GetConversation(ctx context.Context, conversationID string) (*Conversation, error)
	ListConversations(ctx context.Context, userID string, limit, offset int) ([]*Conversation, error)
	ArchiveConversation(ctx context.Context, conversationID string) error

	// Message operations
	AddMessage(ctx context.Context, conversationID, role, content string, metadata map[string]interface{}) (*Message, error)
	GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error)
}

// PostgresConversationStorage implements ConversationStorage using PostgreSQL
type PostgresConversationStorage struct {
	db postgres.DB
}

// NewPostgresConversationStorage creates a PostgreSQL-based conversation storage
func NewPostgresConversationStorage(db postgres.DB) *PostgresConversationStorage {
	return &PostgresConversationStorage{db: db}
}

// CreateConversation implements ConversationStorage
func (s *PostgresConversationStorage) CreateConversation(ctx context.Context, userID, title string) (*Conversation, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	conversationID := generateID()
	now := time.Now()

	query := `
		INSERT INTO conversations (id, user_id, title, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, title, summary, metadata, is_archived, created_at, updated_at
	`

	conversation := &Conversation{}
	err := s.db.QueryRow(ctx, query, conversationID, userID, title, now, now).Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.Title,
		&conversation.Summary,
		&conversation.Metadata,
		&conversation.IsArchived,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return conversation, nil
}

// GetConversation implements ConversationStorage
func (s *PostgresConversationStorage) GetConversation(ctx context.Context, conversationID string) (*Conversation, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("conversation_id is required")
	}

	query := `
		SELECT id, user_id, title, summary, metadata, is_archived, created_at, updated_at
		FROM conversations 
		WHERE id = $1 AND is_archived = FALSE
	`

	conversation := &Conversation{}
	err := s.db.QueryRow(ctx, query, conversationID).Scan(
		&conversation.ID,
		&conversation.UserID,
		&conversation.Title,
		&conversation.Summary,
		&conversation.Metadata,
		&conversation.IsArchived,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("conversation not found: %s", conversationID)
		}
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	return conversation, nil
}

// ListConversations implements ConversationStorage
func (s *PostgresConversationStorage) ListConversations(ctx context.Context, userID string, limit, offset int) ([]*Conversation, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, user_id, title, summary, metadata, is_archived, created_at, updated_at
		FROM conversations 
		WHERE user_id = $1 AND is_archived = FALSE
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var conversations []*Conversation
	for rows.Next() {
		conversation := &Conversation{}
		err := rows.Scan(
			&conversation.ID,
			&conversation.UserID,
			&conversation.Title,
			&conversation.Summary,
			&conversation.Metadata,
			&conversation.IsArchived,
			&conversation.CreatedAt,
			&conversation.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conversations = append(conversations, conversation)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conversations: %w", err)
	}

	return conversations, nil
}

// ArchiveConversation implements ConversationStorage
func (s *PostgresConversationStorage) ArchiveConversation(ctx context.Context, conversationID string) error {
	if conversationID == "" {
		return fmt.Errorf("conversation_id is required")
	}

	query := `UPDATE conversations SET is_archived = TRUE, updated_at = $1 WHERE id = $2`
	result, err := s.db.Exec(ctx, query, time.Now(), conversationID)
	if err != nil {
		return fmt.Errorf("failed to archive conversation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found: %s", conversationID)
	}

	return nil
}

// AddMessage implements ConversationStorage
func (s *PostgresConversationStorage) AddMessage(ctx context.Context, conversationID, role, content string, metadata map[string]interface{}) (*Message, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("conversation_id is required")
	}
	if role == "" {
		return nil, fmt.Errorf("role is required")
	}
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}

	messageID := generateID()
	now := time.Now()

	// Convert metadata to JSON
	var metadataJSON []byte
	if metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO messages (id, conversation_id, role, content, metadata, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, conversation_id, role, content, metadata, created_at
	`

	message := &Message{}
	err := s.db.QueryRow(ctx, query, messageID, conversationID, role, content, metadataJSON, now).Scan(
		&message.ID,
		&message.ConversationID,
		&message.Role,
		&message.Content,
		&message.Metadata,
		&message.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to add message: %w", err)
	}

	// Update conversation timestamp
	_, err = s.db.Exec(ctx, "UPDATE conversations SET updated_at = $1 WHERE id = $2", now, conversationID)
	if err != nil {
		// Log but don't fail - message was created successfully
		// In production, consider using a transaction
	}

	return message, nil
}

// GetMessages implements ConversationStorage
func (s *PostgresConversationStorage) GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error) {
	if conversationID == "" {
		return nil, fmt.Errorf("conversation_id is required")
	}

	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, conversation_id, role, content, metadata, created_at
		FROM messages 
		WHERE conversation_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(ctx, query, conversationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		message := &Message{}
		err := rows.Scan(
			&message.ID,
			&message.ConversationID,
			&message.Role,
			&message.Content,
			&message.Metadata,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, message)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// Helper function to generate IDs (you might want to use a proper ID generation strategy)
func generateID() string {
	return fmt.Sprintf("conv_%d", time.Now().UnixNano())
}

// Helper function for metadata JSON conversion
func metadataToJSON(metadata map[string]interface{}) ([]byte, error) {
	if metadata == nil {
		return nil, nil
	}
	return json.Marshal(metadata)
}
