package conversation

import (
	"time"
)

// Conversation represents a conversation with explicit types
// Following Go best practice: explicit types instead of map[string]interface{}
type Conversation struct {
	ID         string               `json:"id"`
	UserID     string               `json:"user_id"`
	Title      string               `json:"title"`
	Summary    *string              `json:"summary,omitempty"`
	Metadata   ConversationMetadata `json:"metadata"`
	IsArchived bool                 `json:"is_archived"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

// ConversationMetadata contains structured metadata for conversations
// Replaces map[string]interface{} with explicit fields
type ConversationMetadata struct {
	// Context information
	Context  string   `json:"context,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Category string   `json:"category,omitempty"`
	Priority string   `json:"priority,omitempty"`

	// AI-related metadata
	Model       string  `json:"model,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`

	// User preferences for this conversation
	Language     string `json:"language,omitempty"`
	ResponseMode string `json:"response_mode,omitempty"`

	// Statistics
	MessageCount int       `json:"message_count"`
	TokensUsed   int       `json:"tokens_used"`
	LastActive   time.Time `json:"last_active,omitempty"`
}

// Message represents a message in a conversation
type Message struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversation_id"`
	Role           string          `json:"role"`
	Content        string          `json:"content"`
	Metadata       MessageMetadata `json:"metadata"`
	TokenCount     *int            `json:"token_count,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// MessageMetadata contains structured metadata for messages
// Replaces map[string]interface{} with explicit fields
type MessageMetadata struct {
	// Tool execution related
	ToolCalls   []ToolCall   `json:"tool_calls,omitempty"`
	ToolResults []ToolResult `json:"tool_results,omitempty"`

	// AI provider information
	Provider     string        `json:"provider,omitempty"`
	Model        string        `json:"model,omitempty"`
	TokensUsed   int           `json:"tokens_used,omitempty"`
	ResponseTime time.Duration `json:"response_time,omitempty"`

	// Processing information
	ProcessingSteps []string   `json:"processing_steps,omitempty"`
	ErrorInfo       *ErrorInfo `json:"error_info,omitempty"`
}

// ToolCall represents a tool invocation
type ToolCall struct {
	ID         string                 `json:"id"`
	ToolName   string                 `json:"tool_name"`
	Parameters map[string]interface{} `json:"parameters"` // Keep as interface{} for tool flexibility
	Timestamp  time.Time              `json:"timestamp"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID    string        `json:"tool_call_id"`
	Success       bool          `json:"success"`
	Output        interface{}   `json:"output"` // Tool-specific output
	Error         string        `json:"error,omitempty"`
	ExecutionTime time.Duration `json:"execution_time"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Storage errors following Go error pattern
type ConversationNotFoundError struct {
	ConversationID string
}

func (e ConversationNotFoundError) Error() string {
	return "conversation not found: " + e.ConversationID
}

type ConversationStorageError struct {
	Operation string
	Err       error
}

func (e ConversationStorageError) Error() string {
	return "conversation storage error during " + e.Operation + ": " + e.Err.Error()
}

func (e ConversationStorageError) Unwrap() error {
	return e.Err
}

// Helper functions for type safety

// NewConversation creates a new conversation with proper defaults
func NewConversation(userID, title string) *Conversation {
	now := time.Now()
	return &Conversation{
		UserID: userID,
		Title:  title,
		Metadata: ConversationMetadata{
			MessageCount: 0,
			TokensUsed:   0,
			LastActive:   now,
		},
		IsArchived: false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewMessage creates a new message with proper defaults
func NewMessage(conversationID, role, content string) *Message {
	return &Message{
		ConversationID: conversationID,
		Role:           role,
		Content:        content,
		Metadata:       MessageMetadata{},
		CreatedAt:      time.Now(),
	}
}

// ConversationStats represents statistics about a conversation
type ConversationStats struct {
	ConversationID    string    `json:"conversation_id"`
	MessageCount      int       `json:"message_count"`
	UserMessages      int       `json:"user_messages"`
	AssistantMessages int       `json:"assistant_messages"`
	TokensUsed        int       `json:"tokens_used"`
	CreatedAt         time.Time `json:"created_at"`
	LastActive        time.Time `json:"last_active"`
	IsArchived        bool      `json:"is_archived"`
}
