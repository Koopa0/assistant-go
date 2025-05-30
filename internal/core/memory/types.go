package memory

import "time"

// Common types shared across memory subsystems

// MemoryType defines types of memory for consolidation
type MemoryType string

const (
	MemoryTypeWorking    MemoryType = "working"
	MemoryTypeEpisodic   MemoryType = "episodic"
	MemoryTypeSemantic   MemoryType = "semantic"
	MemoryTypeProcedural MemoryType = "procedural"
)

// FeedbackType defines types of feedback for learning
type FeedbackType string

const (
	FeedbackTypePositive      FeedbackType = "positive"
	FeedbackTypeNegative      FeedbackType = "negative"
	FeedbackTypeCorrection    FeedbackType = "correction"
	FeedbackTypeReinforcement FeedbackType = "reinforcement"
	FeedbackTypeAssociation   FeedbackType = "association"
)

// TimeRange represents a time period for querying
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// LearningStrategy defines learning approaches
type LearningStrategy string

const (
	LearningByExample     LearningStrategy = "example"
	LearningByExplanation LearningStrategy = "explanation"
	LearningByPractice    LearningStrategy = "practice"
	LearningByObservation LearningStrategy = "observation"
	LearningByFeedback    LearningStrategy = "feedback"
)

// QueryType defines types of memory queries
type QueryType string

const (
	QueryExact       QueryType = "exact"
	QuerySimilar     QueryType = "similar"
	QueryRelated     QueryType = "related"
	QueryTemporal    QueryType = "temporal"
	QueryContextual  QueryType = "contextual"
	QueryTypeConcept QueryType = "concept"
	QueryHowTo       QueryType = "how_to"
)

// ConditionType defines types of conditions
type ConditionType string

const (
	ConditionTypeInvariant ConditionType = "invariant"
	ConditionTypePrecond   ConditionType = "precondition"
	ConditionTypePostcond  ConditionType = "postcondition"
)

// SortCriteria defines how to sort memory query results
type SortCriteria string

const (
	SortByPriority   SortCriteria = "priority"
	SortByActivation SortCriteria = "activation"
	SortByRecency    SortCriteria = "recency"
	SortByRelevance  SortCriteria = "relevance"
	SortByFrequency  SortCriteria = "frequency"
	SortByImportance SortCriteria = "importance"
	SortByVividness  SortCriteria = "vividness"
	SortByRetrieval  SortCriteria = "retrieval"
)

// Common utility functions

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
