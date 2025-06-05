package memory

import (
	"context"
	"time"
)

// Memory represents the core memory interface for the system
// Small, focused interface following Go principles
type Memory interface {
	// Store saves an entry to memory
	Store(ctx context.Context, entry Entry) error
	// Retrieve gets an entry by ID
	Retrieve(ctx context.Context, id string) (*Entry, error)
	// Search finds entries matching criteria
	Search(ctx context.Context, criteria SearchCriteria) ([]Entry, error)
	// Delete removes an entry by ID
	Delete(ctx context.Context, id string) error
}

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
	Content       string     `json:"content,omitempty"`
	Embedding     []float64  `json:"embedding,omitempty"`
	MinImportance float64    `json:"min_importance"`
	Since         *time.Time `json:"since,omitempty"`
	Limit         int        `json:"limit"`
}

// MemoryGraph represents relationships between memories
type MemoryGraph interface {
	// AddRelation creates a relationship between two memory entries
	AddRelation(ctx context.Context, relation Relation) error
	// GetRelated finds memories related to a given memory ID
	GetRelated(ctx context.Context, memoryID string, opts RelationOptions) ([]RelatedMemory, error)
	// FindPath discovers connection paths between two memories
	FindPath(ctx context.Context, from, to string, maxDepth int) ([]Path, error)
	// GetCluster finds clusters of related memories
	GetCluster(ctx context.Context, memoryID string, opts ClusterOptions) (*MemoryCluster, error)
}

// RelationType defines the type of relationship between memories
type RelationType string

const (
	// Causal relationships
	RelationCauses   RelationType = "causes"
	RelationResultOf RelationType = "result_of"
	RelationEnables  RelationType = "enables"
	RelationPrevents RelationType = "prevents"

	// Temporal relationships
	RelationBefore   RelationType = "before"
	RelationAfter    RelationType = "after"
	RelationDuring   RelationType = "during"
	RelationOverlaps RelationType = "overlaps"

	// Semantic relationships
	RelationSimilar  RelationType = "similar"
	RelationOpposite RelationType = "opposite"
	RelationContains RelationType = "contains"
	RelationPartOf   RelationType = "part_of"

	// Episodic relationships
	RelationTriggers   RelationType = "triggers"
	RelationReferences RelationType = "references"
	RelationContext    RelationType = "context"
	RelationExample    RelationType = "example"
)

// Relation represents a connection between two memory entries
type Relation struct {
	ID         string       `json:"id"`
	FromID     string       `json:"from_id"`
	ToID       string       `json:"to_id"`
	Type       RelationType `json:"type"`
	Strength   float64      `json:"strength"`   // 0.0 to 1.0
	Confidence float64      `json:"confidence"` // 0.0 to 1.0
	UserID     string       `json:"user_id"`
	Context    string       `json:"context,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

// RelatedMemory represents a memory with its relationship information
type RelatedMemory struct {
	Entry    Entry    `json:"entry"`
	Relation Relation `json:"relation"`
	Distance int      `json:"distance"` // Degrees of separation
}

// Path represents a connection path between two memories
type Path struct {
	From      string     `json:"from"`
	To        string     `json:"to"`
	Relations []Relation `json:"relations"`
	Length    int        `json:"length"`
	Score     float64    `json:"score"` // Path relevance score
}

// MemoryCluster represents a cluster of related memories
type MemoryCluster struct {
	CenterID  string          `json:"center_id"`
	Members   []RelatedMemory `json:"members"`
	Cohesion  float64         `json:"cohesion"` // 0.0 to 1.0
	Diameter  int             `json:"diameter"` // Max distance between any two members
	CreatedAt time.Time       `json:"created_at"`
}

// RelationOptions configures relation queries
type RelationOptions struct {
	Types       []RelationType `json:"types,omitempty"`
	MinStrength float64        `json:"min_strength"`
	MaxDepth    int            `json:"max_depth"`
	Limit       int            `json:"limit"`
	Direction   Direction      `json:"direction"`
}

// ClusterOptions configures cluster queries
type ClusterOptions struct {
	MaxSize     int            `json:"max_size"`
	MinCohesion float64        `json:"min_cohesion"`
	Types       []RelationType `json:"types,omitempty"`
	Algorithm   string         `json:"algorithm,omitempty"`
}

// Direction defines traversal direction for graph queries
type Direction string

const (
	DirectionOut  Direction = "outbound"
	DirectionIn   Direction = "inbound"
	DirectionBoth Direction = "both"
)

// ConsolidationStrategy defines how memories are consolidated
type ConsolidationStrategy string

const (
	ConsolidationByImportance ConsolidationStrategy = "importance"
	ConsolidationByFrequency  ConsolidationStrategy = "frequency"
	ConsolidationByRecency    ConsolidationStrategy = "recency"
	ConsolidationByRelevance  ConsolidationStrategy = "relevance"
)

// ConsolidationOptions configures memory consolidation
type ConsolidationOptions struct {
	Strategy    ConsolidationStrategy `json:"strategy"`
	Threshold   float64               `json:"threshold"`
	MaxEntries  int                   `json:"max_entries"`
	PreserveAll bool                  `json:"preserve_all"`
}

// MemoryManager coordinates different memory types and operations
type MemoryManager interface {
	// Core operations
	Store(ctx context.Context, entry Entry) error
	Retrieve(ctx context.Context, id string) (*Entry, error)
	Search(ctx context.Context, criteria SearchCriteria) ([]Entry, error)
	Delete(ctx context.Context, id string) error

	// Memory type specific operations
	GetWorkingMemory() Memory
	GetEpisodicMemory() Memory
	GetSemanticMemory() Memory
	GetProceduralMemory() Memory

	// Graph operations
	GetGraph() MemoryGraph

	// Consolidation operations
	Consolidate(ctx context.Context, opts ConsolidationOptions) error

	// Statistics
	GetStats(ctx context.Context) (*MemoryStats, error)
}

// MemoryStats provides statistics about memory usage
type MemoryStats struct {
	TotalEntries     int                  `json:"total_entries"`
	EntriesByType    map[Type]int         `json:"entries_by_type"`
	TotalRelations   int                  `json:"total_relations"`
	RelationsByType  map[RelationType]int `json:"relations_by_type"`
	StorageUsed      int64                `json:"storage_used"` // bytes
	LastConsolidated time.Time            `json:"last_consolidated"`
	Performance      PerformanceStats     `json:"performance"`
}

// PerformanceStats tracks memory system performance
type PerformanceStats struct {
	AverageStoreTime    time.Duration `json:"average_store_time"`
	AverageRetrieveTime time.Duration `json:"average_retrieve_time"`
	AverageSearchTime   time.Duration `json:"average_search_time"`
	CacheHitRate        float64       `json:"cache_hit_rate"`
	ConsolidationRate   float64       `json:"consolidation_rate"`
}

// VectorSimilarity calculates cosine similarity between two vectors
func VectorSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	// Simple square root approximation
	sqrtA := sqrt(normA)
	sqrtB := sqrt(normB)

	return dotProduct / (sqrtA * sqrtB)
}

// CalculateStrength computes relation strength based on various factors
func CalculateStrength(frequency int, recency time.Time, importance float64) float64 {
	// Frequency component (0-1)
	freqScore := float64(frequency) / (float64(frequency) + 10) // Logarithmic scaling

	// Recency component (0-1)
	daysSince := time.Since(recency).Hours() / 24
	recencyScore := 1.0 / (1.0 + daysSince/30) // 30-day half-life

	// Weighted combination
	strength := (freqScore*0.4 + recencyScore*0.3 + importance*0.3)

	// Ensure bounds
	if strength > 1.0 {
		strength = 1.0
	}
	if strength < 0.0 {
		strength = 0.0
	}

	return strength
}

// CalculateCohesion computes cluster cohesion
func CalculateCohesion(relations []Relation) float64 {
	if len(relations) == 0 {
		return 0
	}

	totalStrength := 0.0
	for _, rel := range relations {
		totalStrength += rel.Strength
	}

	return totalStrength / float64(len(relations))
}

// Simple square root approximation
func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}

	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}
