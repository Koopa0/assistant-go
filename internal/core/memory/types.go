package memory

import (
	"time"
)

// Type identifies the memory type
type Type string

const (
	TypeWorking    Type = "working"    // Short-term, task-focused
	TypeEpisodic   Type = "episodic"   // Experience-based
	TypeSemantic   Type = "semantic"   // Fact-based knowledge
	TypeProcedural Type = "procedural" // How-to knowledge
)

// Entry represents a memory entry
// Value type - immutable and safe to copy
type Entry struct {
	ID          string                 `json:"id"`
	Type        Type                   `json:"type"`
	UserID      string                 `json:"user_id"`
	Content     string                 `json:"content"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Embedding   []float64              `json:"embedding,omitempty"`
	Importance  float64                `json:"importance"` // 0.0 to 1.0
	AccessCount int                    `json:"access_count"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

// SearchCriteria defines search parameters
type SearchCriteria struct {
	UserID        string     `json:"user_id"`
	Types         []Type     `json:"types,omitempty"`
	Query         string     `json:"query,omitempty"`
	Embedding     []float64  `json:"embedding,omitempty"`
	SimilarityMin float64    `json:"similarity_min,omitempty"`
	ImportanceMin float64    `json:"importance_min,omitempty"`
	Limit         int        `json:"limit,omitempty"`
	Offset        int        `json:"offset,omitempty"`
	TimeFrom      *time.Time `json:"time_from,omitempty"`
	TimeTo        *time.Time `json:"time_to,omitempty"`
}

// RelationType defines the type of relationship between memory entries
type RelationType string

const (
	RelationTypeCause    RelationType = "cause"    // A causes B
	RelationTypeSequence RelationType = "sequence" // A happens before B
	RelationTypeSimilar  RelationType = "similar"  // A is similar to B
	RelationTypeContains RelationType = "contains" // A contains B
	RelationTypeRelated  RelationType = "related"  // A is related to B
)

// Relation represents a relationship between memory entries
type Relation struct {
	ID       string                 `json:"id"`
	FromID   string                 `json:"from_id"`
	ToID     string                 `json:"to_id"`
	Type     RelationType           `json:"type"`
	Weight   float64                `json:"weight"` // 0.0 to 1.0
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Created  time.Time              `json:"created"`
}
