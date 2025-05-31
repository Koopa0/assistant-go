package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Domain types for the postgres package
// These types represent the domain model for database operations

// PoolStats represents database connection pool statistics
// This is a typed interface for pgxpool.Stat methods
type PoolStats struct {
	// Connection metrics
	AcquireCount            int64         `json:"acquire_count"`              // Total times connection was acquired from pool
	AcquireDuration         time.Duration `json:"acquire_duration"`           // Total time spent acquiring connections
	AcquiredConns           int32         `json:"acquired_conns"`             // Current number of acquired connections
	CanceledAcquireCount    int64         `json:"canceled_acquire_count"`     // Times connection acquire was canceled
	ConstructingConns       int32         `json:"constructing_conns"`         // Current number of connections being constructed
	EmptyAcquireCount       int64         `json:"empty_acquire_count"`        // Times acquire was called on empty pool
	EmptyAcquireWaitTime    time.Duration `json:"empty_acquire_wait_time"`    // Total time waiting for connection when pool was empty
	IdleConns               int32         `json:"idle_conns"`                 // Current number of idle connections
	MaxConns                int32         `json:"max_conns"`                  // Maximum number of connections allowed
	MaxIdleDestroyCount     int64         `json:"max_idle_destroy_count"`     // Connections destroyed due to max idle time
	MaxLifetimeDestroyCount int64         `json:"max_lifetime_destroy_count"` // Connections destroyed due to max lifetime
	NewConnsCount           int64         `json:"new_conns_count"`            // Total new connections created
	TotalConns              int32         `json:"total_conns"`                // Current total number of connections
}

// NewPoolStatsFromPgxStat creates a PoolStats from pgxpool.Stat
func NewPoolStatsFromPgxStat(stat *pgxpool.Stat) *PoolStats {
	if stat == nil {
		return nil
	}

	return &PoolStats{
		AcquireCount:            stat.AcquireCount(),
		AcquireDuration:         stat.AcquireDuration(),
		AcquiredConns:           stat.AcquiredConns(),
		CanceledAcquireCount:    stat.CanceledAcquireCount(),
		ConstructingConns:       stat.ConstructingConns(),
		EmptyAcquireCount:       stat.EmptyAcquireCount(),
		EmptyAcquireWaitTime:    stat.EmptyAcquireWaitTime(),
		IdleConns:               stat.IdleConns(),
		MaxConns:                stat.MaxConns(),
		MaxIdleDestroyCount:     stat.MaxIdleDestroyCount(),
		MaxLifetimeDestroyCount: stat.MaxLifetimeDestroyCount(),
		NewConnsCount:           stat.NewConnsCount(),
		TotalConns:              stat.TotalConns(),
	}
}

// Conversation represents a conversation in the database
type Conversation struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Title     string                 `json:"title"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Messages  []*Message             `json:"messages,omitempty"`
}

// Message represents a message in a conversation
type Message struct {
	ID             string                 `json:"id"`
	ConversationID string                 `json:"conversation_id"`
	Role           string                 `json:"role"`
	Content        string                 `json:"content"`
	TokenCount     int                    `json:"token_count"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
}

// EmbeddingRecord represents a stored embedding
type EmbeddingRecord struct {
	ID          string                 `json:"id"`
	ContentType string                 `json:"content_type"`
	ContentID   string                 `json:"content_id"`
	ContentText string                 `json:"content_text"`
	Embedding   []float64              `json:"embedding"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

// EmbeddingSearchResult represents a similarity search result
type EmbeddingSearchResult struct {
	Record     *EmbeddingRecord `json:"record"`
	Similarity float64          `json:"similarity"`
	Distance   float64          `json:"distance"`
}

// LangChain domain types for the new integration

// MemoryEntryDomain represents a memory entry in the LangChain system
type MemoryEntryDomain struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	UserID      string                 `json:"user_id"`
	SessionID   string                 `json:"session_id,omitempty"`
	Content     string                 `json:"content"`
	Importance  float64                `json:"importance"`
	AccessCount int                    `json:"access_count"`
	LastAccess  time.Time              `json:"last_access"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// AgentExecutionDomain represents an agent execution record
type AgentExecutionDomain struct {
	ID              string                 `json:"id"`
	AgentType       string                 `json:"agent_type"`
	UserID          string                 `json:"user_id"`
	ConversationID  string                 `json:"conversation_id,omitempty"`
	Query           string                 `json:"query"`
	Response        string                 `json:"response"`
	Steps           []interface{}          `json:"steps"`
	ExecutionTimeMs int                    `json:"execution_time_ms"`
	Success         bool                   `json:"success"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// ChainExecutionDomain represents a chain execution record
type ChainExecutionDomain struct {
	ID              string                 `json:"id"`
	ChainType       string                 `json:"chain_type"`
	UserID          string                 `json:"user_id"`
	ConversationID  string                 `json:"conversation_id,omitempty"`
	Input           string                 `json:"input"`
	Output          string                 `json:"output"`
	Steps           []interface{}          `json:"steps"`
	ExecutionTimeMs int                    `json:"execution_time_ms"`
	TokensUsed      int                    `json:"tokens_used"`
	Success         bool                   `json:"success"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// ToolCacheDomain represents a cached tool execution result
type ToolCacheDomain struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	ToolName        string                 `json:"tool_name"`
	InputHash       string                 `json:"input_hash"`
	InputData       map[string]interface{} `json:"input_data"`
	OutputData      map[string]interface{} `json:"output_data"`
	ExecutionTimeMs int                    `json:"execution_time_ms"`
	Success         bool                   `json:"success"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	HitCount        int                    `json:"hit_count"`
	LastHit         time.Time              `json:"last_hit"`
	CreatedAt       time.Time              `json:"created_at"`
	ExpiresAt       time.Time              `json:"expires_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// UserPreferenceDomain represents a user preference setting
type UserPreferenceDomain struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	Category        string                 `json:"category"`
	PreferenceKey   string                 `json:"preference_key"`
	PreferenceValue map[string]interface{} `json:"preference_value"`
	ValueType       string                 `json:"value_type"`
	Description     string                 `json:"description,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// UserContextDomain represents user contextual information
type UserContextDomain struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	ContextType  string                 `json:"context_type"`
	ContextKey   string                 `json:"context_key"`
	ContextValue map[string]interface{} `json:"context_value"`
	Importance   float64                `json:"importance"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}
