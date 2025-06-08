package conversation

import (
	"context"
)

// ConversationReader defines methods for reading conversations
type ConversationReader interface {
	GetConversation(ctx context.Context, conversationID string) (*Conversation, error)
	ListConversations(ctx context.Context, userID string) ([]*Conversation, error)
}

// ConversationWriter defines methods for creating/updating conversations
type ConversationWriter interface {
	CreateConversation(ctx context.Context, userID, title string) (*Conversation, error)
	UpdateConversation(ctx context.Context, conv *Conversation) error
	ArchiveConversation(ctx context.Context, conversationID string) error
	DeleteConversation(ctx context.Context, conversationID string) error
}

// MessageReader defines methods for reading messages
type MessageReader interface {
	GetMessages(ctx context.Context, conversationID string) ([]*Message, error)
}

// MessageWriter defines methods for creating messages
type MessageWriter interface {
	AddMessage(ctx context.Context, conversationID, role, content string) (*Message, error)
}

// StatsProvider defines methods for conversation statistics
type StatsProvider interface {
	GetConversationStats(ctx context.Context, conversationID string) (*ConversationStats, error)
}

// ConversationService combines all conversation operations
// This is what most consumers will use
type ConversationService interface {
	ConversationReader
	ConversationWriter
	MessageReader
	MessageWriter
	StatsProvider
}

// QueryResponse represents the response from a query operation
type QueryResponse struct {
	Response       string                 `json:"response"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	MessageID      string                 `json:"message_id,omitempty"`
	TokensUsed     int                    `json:"tokens_used,omitempty"`
	ProcessingTime float64                `json:"processing_time,omitempty"`
	ToolsUsed      []string               `json:"tools_used,omitempty"`
	Model          string                 `json:"model,omitempty"`
	Error          string                 `json:"error,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// QueryRequest represents a query request
type QueryRequest struct {
	Query          string                 `json:"query"`
	UserID         *string                `json:"user_id,omitempty"`
	ConversationID *string                `json:"conversation_id,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
	Model          *string                `json:"model,omitempty"`
	Temperature    *float64               `json:"temperature,omitempty"`
	MaxTokens      *int                   `json:"max_tokens,omitempty"`
	Stream         bool                   `json:"stream,omitempty"`
}

// AssistantInterface defines the interface that ConversationService needs
// This breaks the circular dependency with the assistant package
type AssistantInterface interface {
	ListConversations(ctx context.Context, userID string, limit, offset int) ([]*Conversation, error)
	GetConversation(ctx context.Context, conversationID string) (*Conversation, error)
	GetConversationMessages(ctx context.Context, conversationID string) ([]*Message, error)
	DeleteConversation(ctx context.Context, conversationID string) error
	ProcessQueryRequest(ctx context.Context, req *QueryRequest) (*QueryResponse, error)
}
