package memory

import (
	"time"
)

// Memory types matching database constraints
const (
	TypeWorking         Type = "working"
	TypeShortTerm       Type = "short_term"
	TypeLongTerm        Type = "long_term"
	TypeTool            Type = "tool"
	TypePersonalization Type = "personalization"

	// Legacy aliases for backward compatibility
	TypeEpisodic   = TypeLongTerm
	TypeSemantic   = TypeLongTerm
	TypeProcedural = TypeTool
)

// Importance levels
const (
	ImportanceLow    float64 = 0.0
	ImportanceMedium float64 = 0.5
	ImportanceHigh   float64 = 1.0
)

// Default values
const (
	DefaultSearchLimit    = 100
	DefaultMaxWorkingSize = 100
)

// Type represents the type of memory entry
type Type string

// Entry represents a memory entry with typed metadata
type Entry struct {
	ID          string
	Type        Type
	UserID      string
	SessionID   *string
	Content     string
	Importance  float64
	AccessCount int
	LastAccess  time.Time
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Version     int32

	// Typed metadata
	Metadata EntryMetadata

	// Legacy support
	Context map[string]interface{} `json:"-"` // Deprecated: Use Metadata
}

// EntryMetadata holds structured metadata for memory entries
type EntryMetadata struct {
	Category     string   `json:"category,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Source       string   `json:"source,omitempty"`
	Confidence   float64  `json:"confidence,omitempty"`
	Related      []string `json:"related,omitempty"`      // Related memory IDs
	Dependencies []string `json:"dependencies,omitempty"` // Dependency IDs

	// AI-specific metadata
	Model          string                 `json:"model,omitempty"`
	Temperature    float64                `json:"temperature,omitempty"`
	PromptTokens   int                    `json:"prompt_tokens,omitempty"`
	ResponseTokens int                    `json:"response_tokens,omitempty"`
	TotalTokens    int                    `json:"total_tokens,omitempty"`
	Extra          map[string]interface{} `json:"extra,omitempty"`
}

// SearchCriteria defines parameters for searching memory entries
type SearchCriteria struct {
	UserID        string
	Types         []Type
	Query         string
	ImportanceMin float64
	Limit         int
	Offset        int
	Tags          []string
	Category      string
	TimeRange     *TimeRange
}

// TimeRange defines a time range for filtering
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// MemoryStats provides statistics about memory usage
type MemoryStats struct {
	TotalEntries   int64
	EntriesByType  map[Type]int64
	TotalSize      int64
	AverageAccess  float64
	LastAccessTime time.Time
	CreationRate   float64 // Entries per hour
	AccessRate     float64 // Accesses per hour

	// HTTP-specific fields
	TotalNodes        int32            `json:"total_nodes"`
	NodesByType       map[string]int32 `json:"nodes_by_type"`
	AverageImportance float64          `json:"average_importance"`
	TotalConnections  int32            `json:"total_connections"`
	LastUpdated       time.Time        `json:"last_updated"`
}

// WorkingMemoryEntry represents an entry in working memory
type WorkingMemoryEntry struct {
	Entry
	Priority    float64   // For eviction
	LastUpdated time.Time // For LRU
}

// SortOption defines how to sort search results
type SortOption string

const (
	SortByImportance SortOption = "importance"
	SortByRecency    SortOption = "recency"
	SortByAccess     SortOption = "access"
)

// SearchOptions provides advanced search options
type SearchOptions struct {
	SortBy         SortOption
	Descending     bool
	IncludeExpired bool
	MinConfidence  float64
}
