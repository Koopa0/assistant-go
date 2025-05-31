package memory

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/koopa0/assistant-go/internal/core/events"
)

// SemanticMemory stores knowledge, concepts, and relationships
type SemanticMemory struct {
	concepts        map[string]*Concept
	relationships   map[string]*Relationship
	schemas         map[string]*Schema
	ontology        *Ontology
	knowledgeGraph  *KnowledgeGraph
	reasoningEngine *ReasoningEngine
	learningEngine  *ConceptLearningEngine
	eventBus        *events.EventBus
	logger          *slog.Logger
	mu              sync.RWMutex
}

// Concept represents a unit of semantic knowledge
type Concept struct {
	ID              string
	Name            string
	Type            ConceptType
	Definition      string
	Properties      map[string]Property
	Examples        []Example
	Abstractions    []string // Parent concepts
	Specializations []string // Child concepts
	RelatedConcepts []string
	Confidence      float64
	UsageCount      int
	LastAccessed    time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Source          KnowledgeSource
	Embeddings      map[string][]float32 // Different embedding spaces
	Metadata        map[string]interface{}
}

// ConceptType defines types of concepts
type ConceptType string

const (
	ConceptTypeEntity   ConceptType = "entity"
	ConceptTypeAction   ConceptType = "action"
	ConceptTypeProperty ConceptType = "property"
	ConceptTypeRelation ConceptType = "relation"
	ConceptTypeCategory ConceptType = "category"
	ConceptTypeAbstract ConceptType = "abstract"
	ConceptTypeConcrete ConceptType = "concrete"
)

// Property represents a property of a concept
type Property struct {
	Name        string
	Type        PropertyType
	Value       interface{}
	Confidence  float64
	Source      string
	Constraints []Constraint
}

// PropertyType defines types of properties
type PropertyType string

const (
	PropertyTypeIntrinsic  PropertyType = "intrinsic"
	PropertyTypeExtrinsic  PropertyType = "extrinsic"
	PropertyTypeFunctional PropertyType = "functional"
	PropertyTypeCausal     PropertyType = "causal"
)

// Constraint represents a constraint on a property
type Constraint struct {
	Type  ConstraintType
	Value interface{}
}

// ConstraintType defines types of constraints
type ConstraintType string

const (
	ConstraintTypeRange     ConstraintType = "range"
	ConstraintTypePattern   ConstraintType = "pattern"
	ConstraintTypeReference ConstraintType = "reference"
	ConstraintTypeUnique    ConstraintType = "unique"
)

// Example represents an example of a concept
type Example struct {
	ID          string
	Description string
	Context     string
	Positive    bool // Positive or negative example
	Confidence  float64
}

// KnowledgeSource represents where knowledge came from
type KnowledgeSource struct {
	Type      SourceType
	Reference string
	Timestamp time.Time
	Trust     float64
}

// SourceType defines types of knowledge sources
type SourceType string

const (
	SourceTypeLearned  SourceType = "learned"
	SourceTypeInferred SourceType = "inferred"
	SourceTypeTaught   SourceType = "taught"
	SourceTypeObserved SourceType = "observed"
	SourceTypeDeduced  SourceType = "deduced"
)

// Relationship represents a relationship between concepts
type Relationship struct {
	ID            string
	Type          RelationType
	Source        string // Concept ID
	Target        string // Concept ID
	Properties    map[string]interface{}
	Strength      float64
	Confidence    float64
	Bidirectional bool
	Context       []string
	Evidence      []Evidence
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// RelationType defines types of relationships
type RelationType string

const (
	RelationTypeIsA          RelationType = "is_a"
	RelationTypePartOf       RelationType = "part_of"
	RelationTypeHasProperty  RelationType = "has_property"
	RelationTypeCauses       RelationType = "causes"
	RelationTypePrerequisite RelationType = "prerequisite"
	RelationTypeSimilarTo    RelationType = "similar_to"
	RelationTypeOppositeTo   RelationType = "opposite_to"
	RelationTypeUsedFor      RelationType = "used_for"
	RelationTypeImplements   RelationType = "implements"
	RelationTypeDependsOn    RelationType = "depends_on"
)

// Evidence represents evidence for a relationship
type Evidence struct {
	Type      EvidenceType
	Source    string
	Strength  float64
	Timestamp time.Time
}

// EvidenceType defines types of evidence
type EvidenceType string

const (
	EvidenceTypeObserved    EvidenceType = "observed"
	EvidenceTypeInferred    EvidenceType = "inferred"
	EvidenceTypeStatistical EvidenceType = "statistical"
	EvidenceTypeExpert      EvidenceType = "expert"
)

// Schema represents a knowledge schema or framework
type Schema struct {
	ID          string
	Name        string
	Type        SchemaType
	Structure   SchemaStructure
	Slots       []SchemaSlot
	Constraints []SchemaConstraint
	Instances   []string // Concept IDs that match this schema
	Confidence  float64
	UsageCount  int
}

// SchemaType defines types of schemas
type SchemaType string

const (
	SchemaTypeFrame     SchemaType = "frame"
	SchemaTypeScript    SchemaType = "script"
	SchemaTypePrototype SchemaType = "prototype"
	SchemaTypeTemplate  SchemaType = "template"
)

// SchemaStructure defines the structure of a schema
type SchemaStructure struct {
	RequiredSlots []string
	OptionalSlots []string
	DefaultValues map[string]interface{}
	Inheritance   []string // Parent schemas
}

// SchemaSlot represents a slot in a schema
type SchemaSlot struct {
	Name        string
	Type        string
	Constraints []Constraint
	Default     interface{}
	Required    bool
}

// SchemaConstraint represents a constraint on a schema
type SchemaConstraint struct {
	Type       string
	Expression string
	Message    string
}

// Ontology represents the overall knowledge structure
type Ontology struct {
	RootConcepts     []string
	Taxonomies       map[string]*Taxonomy
	DomainOntologies map[string]*DomainOntology
	Rules            []InferenceRule
	mu               sync.RWMutex
}

// Taxonomy represents a hierarchical classification
type Taxonomy struct {
	ID       string
	Name     string
	Root     string
	Levels   []TaxonomyLevel
	Concepts map[string]string // Concept ID -> Level
}

// TaxonomyLevel represents a level in a taxonomy
type TaxonomyLevel struct {
	Name  string
	Depth int
	Count int
}

// DomainOntology represents domain-specific knowledge
type DomainOntology struct {
	Domain     string
	Concepts   []string
	Relations  []string
	Axioms     []Axiom
	Vocabulary map[string]string
}

// Axiom represents a logical axiom
type Axiom struct {
	ID         string
	Type       AxiomType
	Expression string
	Domain     string
}

// AxiomType defines types of axioms
type AxiomType string

const (
	AxiomTypeEquivalence  AxiomType = "equivalence"
	AxiomTypeSubsumption  AxiomType = "subsumption"
	AxiomTypeDisjointness AxiomType = "disjointness"
	AxiomTypeInverse      AxiomType = "inverse"
)

// InferenceRule represents a rule for inference
type InferenceRule struct {
	ID         string
	Name       string
	Type       RuleType
	Conditions []RuleCondition
	Conclusion RuleConclusion
	Confidence float64
	Priority   int
}

// RuleType defines types of inference rules
type RuleType string

const (
	RuleTypeDeductive  RuleType = "deductive"
	RuleTypeInductive  RuleType = "inductive"
	RuleTypeAbductive  RuleType = "abductive"
	RuleTypeAnalogical RuleType = "analogical"
)

// RuleCondition represents a condition in a rule
type RuleCondition struct {
	Type      string
	Subject   string
	Predicate string
	Object    string
}

// RuleConclusion represents the conclusion of a rule
type RuleConclusion struct {
	Type       string
	Subject    string
	Predicate  string
	Object     string
	Confidence float64
}

// KnowledgeGraph represents knowledge as a graph
type KnowledgeGraph struct {
	nodes      map[string]*GraphNode
	edges      map[string][]*GraphEdge
	index      *GraphIndex
	embeddings *GraphEmbeddings
	mu         sync.RWMutex
}

// GraphNode represents a node in the knowledge graph
type GraphNode struct {
	ID         string
	Type       string
	Properties map[string]interface{}
	Embedding  []float32
}

// GraphEdge represents an edge in the knowledge graph
type GraphEdge struct {
	ID         string
	Source     string
	Target     string
	Type       string
	Weight     float64
	Properties map[string]interface{}
}

// GraphIndex provides fast graph lookups
type GraphIndex struct {
	nodesByType map[string][]*GraphNode
	edgesByType map[string][]*GraphEdge
	neighbors   map[string][]string
}

// GraphEmbeddings manages graph embeddings
type GraphEmbeddings struct {
	nodeEmbeddings map[string][]float32
	edgeEmbeddings map[string][]float32
	dimension      int
}

// ReasoningEngine performs reasoning over semantic memory
type ReasoningEngine struct {
	strategies map[string]ReasoningStrategy
	cache      *ReasoningCache
	mu         sync.RWMutex
}

// ReasoningStrategy defines a reasoning approach
type ReasoningStrategy interface {
	Reason(query ReasoningQuery, memory *SemanticMemory) (*ReasoningResult, error)
	GetName() string
}

// ReasoningQuery represents a reasoning query
type ReasoningQuery struct {
	Type        ReasoningType
	Subject     string
	Predicate   string
	Object      string
	Context     []string
	Constraints map[string]interface{}
}

// ReasoningType defines types of reasoning
type ReasoningType string

const (
	ReasoningTypeInference   ReasoningType = "inference"
	ReasoningTypeAnalogy     ReasoningType = "analogy"
	ReasoningTypeAbstraction ReasoningType = "abstraction"
	ReasoningTypeComposition ReasoningType = "composition"
)

// ReasoningResult represents the result of reasoning
type ReasoningResult struct {
	Conclusions []Conclusion
	Confidence  float64
	Evidence    []Evidence
	Reasoning   []ReasoningStep
}

// Conclusion represents a reasoning conclusion
type Conclusion struct {
	Statement  string
	Confidence float64
	Support    []string
}

// ReasoningStep represents a step in reasoning
type ReasoningStep struct {
	Step       int
	Operation  string
	Input      []string
	Output     string
	Confidence float64
}

// ReasoningCache caches reasoning results
type ReasoningCache struct {
	results map[string]*ReasoningResult
	ttl     time.Duration
	mu      sync.RWMutex
}

// ConceptLearningEngine learns new concepts
type ConceptLearningEngine struct {
	strategies  []LearningStrategyInterface
	threshold   float64
	minExamples int
	mu          sync.RWMutex
}

// LearningStrategyInterface defines how to learn concepts
type LearningStrategyInterface interface {
	Learn(examples []Example, context map[string]interface{}) (*Concept, error)
	Update(concept *Concept, newExamples []Example) error
}

// SemanticQuery represents a query to semantic memory
type SemanticQuery struct {
	Type        SemanticQueryType
	Concept     string
	Relation    string
	Properties  map[string]interface{}
	Constraints []QueryConstraint
	Limit       int
}

// SemanticQueryType defines types of semantic queries
type SemanticQueryType string

const (
	SemanticQueryTypeConcept    SemanticQueryType = "concept"
	SemanticQueryTypeRelation   SemanticQueryType = "relation"
	SemanticQueryTypeProperty   SemanticQueryType = "property"
	SemanticQueryTypePath       SemanticQueryType = "path"
	SemanticQueryTypeInference  SemanticQueryType = "inference"
	SemanticQueryTypeSimilarity SemanticQueryType = "similarity"
)

// QueryConstraint represents a constraint on a query
type QueryConstraint struct {
	Type  string
	Field string
	Value interface{}
}

// NewSemanticMemory creates a new semantic memory
func NewSemanticMemory(eventBus *events.EventBus, logger *slog.Logger) (*SemanticMemory, error) {
	ontology := &Ontology{
		RootConcepts:     make([]string, 0),
		Taxonomies:       make(map[string]*Taxonomy),
		DomainOntologies: make(map[string]*DomainOntology),
		Rules:            make([]InferenceRule, 0),
	}

	knowledgeGraph := &KnowledgeGraph{
		nodes: make(map[string]*GraphNode),
		edges: make(map[string][]*GraphEdge),
		index: &GraphIndex{
			nodesByType: make(map[string][]*GraphNode),
			edgesByType: make(map[string][]*GraphEdge),
			neighbors:   make(map[string][]string),
		},
		embeddings: &GraphEmbeddings{
			nodeEmbeddings: make(map[string][]float32),
			edgeEmbeddings: make(map[string][]float32),
			dimension:      128,
		},
	}

	reasoningEngine := &ReasoningEngine{
		strategies: make(map[string]ReasoningStrategy),
		cache: &ReasoningCache{
			results: make(map[string]*ReasoningResult),
			ttl:     1 * time.Hour,
		},
	}

	learningEngine := &ConceptLearningEngine{
		strategies:  make([]LearningStrategyInterface, 0),
		threshold:   0.7,
		minExamples: 3,
	}

	sm := &SemanticMemory{
		concepts:        make(map[string]*Concept),
		relationships:   make(map[string]*Relationship),
		schemas:         make(map[string]*Schema),
		ontology:        ontology,
		knowledgeGraph:  knowledgeGraph,
		reasoningEngine: reasoningEngine,
		learningEngine:  learningEngine,
		eventBus:        eventBus,
		logger:          logger,
	}

	// Initialize reasoning strategies
	sm.initializeReasoningStrategies()

	// Start background processes
	go sm.runMaintenanceLoop()
	go sm.runLearningLoop()

	return sm, nil
}

// StoreConcept adds or updates a concept
func (sm *SemanticMemory) StoreConcept(ctx context.Context, concept *Concept) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Update timestamps
	now := time.Now()
	if concept.CreatedAt.IsZero() {
		concept.CreatedAt = now
	}
	concept.UpdatedAt = now
	concept.LastAccessed = now

	// Store concept
	sm.concepts[concept.ID] = concept

	// Update knowledge graph
	sm.updateKnowledgeGraph(concept)

	// Update ontology
	sm.updateOntology(concept)

	sm.logger.Info("Stored concept",
		slog.String("concept_id", concept.ID),
		slog.String("name", concept.Name),
		slog.String("type", string(concept.Type)))

	// Publish event
	if sm.eventBus != nil {
		event := events.Event{
			Type:   events.EventKnowledgeUpdate,
			Source: "semantic_memory",
			Data: map[string]interface{}{
				"action":     "store_concept",
				"concept_id": concept.ID,
				"type":       concept.Type,
			},
		}
		sm.eventBus.Publish(ctx, event)
	}

	return nil
}

// StoreRelationship adds a relationship between concepts
func (sm *SemanticMemory) StoreRelationship(ctx context.Context, relationship *Relationship) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Verify concepts exist
	if _, exists := sm.concepts[relationship.Source]; !exists {
		return fmt.Errorf("source concept not found: %s", relationship.Source)
	}
	if _, exists := sm.concepts[relationship.Target]; !exists {
		return fmt.Errorf("target concept not found: %s", relationship.Target)
	}

	// Update timestamps
	now := time.Now()
	if relationship.CreatedAt.IsZero() {
		relationship.CreatedAt = now
	}
	relationship.UpdatedAt = now

	// Store relationship
	sm.relationships[relationship.ID] = relationship

	// Update knowledge graph
	sm.addGraphEdge(relationship)

	// Update concept connections
	sourceConcept := sm.concepts[relationship.Source]
	targetConcept := sm.concepts[relationship.Target]

	if !contains(sourceConcept.RelatedConcepts, relationship.Target) {
		sourceConcept.RelatedConcepts = append(sourceConcept.RelatedConcepts, relationship.Target)
	}

	if relationship.Bidirectional && !contains(targetConcept.RelatedConcepts, relationship.Source) {
		targetConcept.RelatedConcepts = append(targetConcept.RelatedConcepts, relationship.Source)
	}

	sm.logger.Info("Stored relationship",
		slog.String("relationship_id", relationship.ID),
		slog.String("type", string(relationship.Type)),
		slog.String("source", relationship.Source),
		slog.String("target", relationship.Target))

	return nil
}

// GetConcept retrieves a concept by ID
func (sm *SemanticMemory) GetConcept(ctx context.Context, conceptID string) (*Concept, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	concept, exists := sm.concepts[conceptID]
	if !exists {
		return nil, fmt.Errorf("concept not found: %s", conceptID)
	}

	// Update access info
	concept.LastAccessed = time.Now()
	concept.UsageCount++

	return concept, nil
}

// Query searches semantic memory
func (sm *SemanticMemory) Query(ctx context.Context, query SemanticQuery) ([]*Concept, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var results []*Concept

	switch query.Type {
	case SemanticQueryTypeConcept:
		results = sm.queryByConcept(query)
	case SemanticQueryTypeRelation:
		results = sm.queryByRelation(query)
	case SemanticQueryTypeProperty:
		results = sm.queryByProperty(query)
	case SemanticQueryTypeSimilarity:
		results = sm.queryBySimilarity(query)
	default:
		return nil, fmt.Errorf("unsupported query type: %s", query.Type)
	}

	// Apply constraints
	results = sm.applyQueryConstraints(results, query.Constraints)

	// Apply limit
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	sm.logger.Debug("Queried semantic memory",
		slog.String("type", string(query.Type)),
		slog.Int("results", len(results)))

	return results, nil
}

// Reason performs reasoning over semantic memory
func (sm *SemanticMemory) Reason(ctx context.Context, query ReasoningQuery) (*ReasoningResult, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%v", query)
	if cached := sm.reasoningEngine.cache.get(cacheKey); cached != nil {
		return cached, nil
	}

	// Get reasoning strategy
	strategy, exists := sm.reasoningEngine.strategies[string(query.Type)]
	if !exists {
		return nil, fmt.Errorf("reasoning strategy not found: %s", query.Type)
	}

	// Perform reasoning
	result, err := strategy.Reason(query, sm)
	if err != nil {
		return nil, err
	}

	// Cache result
	sm.reasoningEngine.cache.set(cacheKey, result)

	sm.logger.Info("Performed reasoning",
		slog.String("type", string(query.Type)),
		slog.Int("conclusions", len(result.Conclusions)))

	return result, nil
}

// LearnConcept learns a new concept from examples
func (sm *SemanticMemory) LearnConcept(ctx context.Context, examples []Example, context map[string]interface{}) (*Concept, error) {
	if len(examples) < sm.learningEngine.minExamples {
		return nil, fmt.Errorf("insufficient examples: need at least %d, got %d",
			sm.learningEngine.minExamples, len(examples))
	}

	var bestConcept *Concept
	var bestConfidence float64

	// Try each learning strategy
	for _, strategy := range sm.learningEngine.strategies {
		concept, err := strategy.Learn(examples, context)
		if err != nil {
			sm.logger.Warn("Learning strategy failed",
				slog.Any("error", err))
			continue
		}

		if concept.Confidence > bestConfidence {
			bestConcept = concept
			bestConfidence = concept.Confidence
		}
	}

	if bestConcept == nil || bestConfidence < sm.learningEngine.threshold {
		return nil, fmt.Errorf("failed to learn concept with sufficient confidence")
	}

	// Store the learned concept
	if err := sm.StoreConcept(ctx, bestConcept); err != nil {
		return nil, err
	}

	sm.logger.Info("Learned new concept",
		slog.String("concept_id", bestConcept.ID),
		slog.String("name", bestConcept.Name),
		slog.Float64("confidence", bestConcept.Confidence))

	return bestConcept, nil
}

// GetRelationships gets all relationships for a concept
func (sm *SemanticMemory) GetRelationships(ctx context.Context, conceptID string) ([]*Relationship, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var relationships []*Relationship

	for _, rel := range sm.relationships {
		if rel.Source == conceptID || (rel.Target == conceptID && rel.Bidirectional) {
			relationships = append(relationships, rel)
		}
	}

	return relationships, nil
}

// FindPath finds a path between two concepts
func (sm *SemanticMemory) FindPath(ctx context.Context, sourceID, targetID string, maxDepth int) ([]string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Verify concepts exist
	if _, exists := sm.concepts[sourceID]; !exists {
		return nil, fmt.Errorf("source concept not found: %s", sourceID)
	}
	if _, exists := sm.concepts[targetID]; !exists {
		return nil, fmt.Errorf("target concept not found: %s", targetID)
	}

	// BFS to find shortest path
	queue := [][]string{{sourceID}}
	visited := make(map[string]bool)
	visited[sourceID] = true

	for len(queue) > 0 && len(queue[0]) <= maxDepth {
		path := queue[0]
		queue = queue[1:]

		current := path[len(path)-1]
		if current == targetID {
			return path, nil
		}

		// Get neighbors
		neighbors := sm.knowledgeGraph.index.neighbors[current]
		for _, neighbor := range neighbors {
			if !visited[neighbor] {
				visited[neighbor] = true
				newPath := make([]string, len(path)+1)
				copy(newPath, path)
				newPath[len(path)] = neighbor
				queue = append(queue, newPath)
			}
		}
	}

	return nil, fmt.Errorf("no path found between %s and %s within %d steps",
		sourceID, targetID, maxDepth)
}

// GetSimilarConcepts finds concepts similar to the given one
func (sm *SemanticMemory) GetSimilarConcepts(ctx context.Context, conceptID string, limit int) ([]*Concept, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	concept, exists := sm.concepts[conceptID]
	if !exists {
		return nil, fmt.Errorf("concept not found: %s", conceptID)
	}

	// Get concept embedding
	embedding, exists := concept.Embeddings["default"]
	if !exists || len(embedding) == 0 {
		return nil, fmt.Errorf("concept has no embedding")
	}

	// Calculate similarities
	type scoredConcept struct {
		concept    *Concept
		similarity float64
	}

	scored := make([]scoredConcept, 0)

	for id, other := range sm.concepts {
		if id == conceptID {
			continue
		}

		otherEmbedding, exists := other.Embeddings["default"]
		if !exists || len(otherEmbedding) == 0 {
			continue
		}

		similarity := cosineSimilarity(embedding, otherEmbedding)
		scored = append(scored, scoredConcept{
			concept:    other,
			similarity: similarity,
		})
	}

	// Sort by similarity
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].similarity > scored[j].similarity
	})

	// Extract concepts
	results := make([]*Concept, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		results = append(results, scored[i].concept)
	}

	return results, nil
}

// GetMetrics returns semantic memory metrics
func (sm *SemanticMemory) GetMetrics() SemanticMemoryMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics := SemanticMemoryMetrics{
		TotalConcepts:      len(sm.concepts),
		TotalRelationships: len(sm.relationships),
		TotalSchemas:       len(sm.schemas),
	}

	// Count by type
	metrics.ConceptsByType = make(map[ConceptType]int)
	totalConfidence := 0.0
	totalUsage := 0

	for _, concept := range sm.concepts {
		metrics.ConceptsByType[concept.Type]++
		totalConfidence += concept.Confidence
		totalUsage += concept.UsageCount
	}

	if len(sm.concepts) > 0 {
		metrics.AverageConfidence = totalConfidence / float64(len(sm.concepts))
		metrics.AverageUsage = float64(totalUsage) / float64(len(sm.concepts))
	}

	// Relationship stats
	metrics.RelationshipsByType = make(map[RelationType]int)
	for _, rel := range sm.relationships {
		metrics.RelationshipsByType[rel.Type]++
	}

	// Graph stats
	sm.knowledgeGraph.mu.RLock()
	metrics.GraphNodes = len(sm.knowledgeGraph.nodes)
	metrics.GraphEdges = 0
	for _, edges := range sm.knowledgeGraph.edges {
		metrics.GraphEdges += len(edges)
	}
	sm.knowledgeGraph.mu.RUnlock()

	return metrics
}

// SemanticMemoryMetrics contains metrics about semantic memory
type SemanticMemoryMetrics struct {
	TotalConcepts       int
	TotalRelationships  int
	TotalSchemas        int
	ConceptsByType      map[ConceptType]int
	RelationshipsByType map[RelationType]int
	AverageConfidence   float64
	AverageUsage        float64
	GraphNodes          int
	GraphEdges          int
}

// Helper methods

func (sm *SemanticMemory) updateKnowledgeGraph(concept *Concept) {
	sm.knowledgeGraph.mu.Lock()
	defer sm.knowledgeGraph.mu.Unlock()

	// Create or update node
	node := &GraphNode{
		ID:   concept.ID,
		Type: string(concept.Type),
		Properties: map[string]interface{}{
			"name":       concept.Name,
			"confidence": concept.Confidence,
		},
	}

	// Add embedding if available
	if embedding, exists := concept.Embeddings["default"]; exists {
		node.Embedding = embedding
		sm.knowledgeGraph.embeddings.nodeEmbeddings[concept.ID] = embedding
	}

	sm.knowledgeGraph.nodes[concept.ID] = node

	// Update index
	sm.knowledgeGraph.index.nodesByType[node.Type] = append(
		sm.knowledgeGraph.index.nodesByType[node.Type], node)
}

func (sm *SemanticMemory) addGraphEdge(relationship *Relationship) {
	sm.knowledgeGraph.mu.Lock()
	defer sm.knowledgeGraph.mu.Unlock()

	edge := &GraphEdge{
		ID:     relationship.ID,
		Source: relationship.Source,
		Target: relationship.Target,
		Type:   string(relationship.Type),
		Weight: relationship.Strength,
		Properties: map[string]interface{}{
			"confidence": relationship.Confidence,
		},
	}

	// Add edge
	sm.knowledgeGraph.edges[relationship.Source] = append(
		sm.knowledgeGraph.edges[relationship.Source], edge)

	// Update neighbors index
	sm.knowledgeGraph.index.neighbors[relationship.Source] = append(
		sm.knowledgeGraph.index.neighbors[relationship.Source], relationship.Target)

	if relationship.Bidirectional {
		sm.knowledgeGraph.index.neighbors[relationship.Target] = append(
			sm.knowledgeGraph.index.neighbors[relationship.Target], relationship.Source)
	}

	// Update edge type index
	sm.knowledgeGraph.index.edgesByType[edge.Type] = append(
		sm.knowledgeGraph.index.edgesByType[edge.Type], edge)
}

func (sm *SemanticMemory) updateOntology(concept *Concept) {
	sm.ontology.mu.Lock()
	defer sm.ontology.mu.Unlock()

	// Add to root concepts if no abstractions
	if len(concept.Abstractions) == 0 && !contains(sm.ontology.RootConcepts, concept.ID) {
		sm.ontology.RootConcepts = append(sm.ontology.RootConcepts, concept.ID)
	}
}

func (sm *SemanticMemory) queryByConcept(query SemanticQuery) []*Concept {
	var results []*Concept

	searchTerm := strings.ToLower(query.Concept)
	for _, concept := range sm.concepts {
		if strings.Contains(strings.ToLower(concept.Name), searchTerm) ||
			strings.Contains(strings.ToLower(concept.Definition), searchTerm) {
			results = append(results, concept)
		}
	}

	return results
}

func (sm *SemanticMemory) queryByRelation(query SemanticQuery) []*Concept {
	conceptMap := make(map[string]*Concept)

	for _, rel := range sm.relationships {
		if string(rel.Type) == query.Relation {
			if source, exists := sm.concepts[rel.Source]; exists {
				conceptMap[source.ID] = source
			}
			if target, exists := sm.concepts[rel.Target]; exists {
				conceptMap[target.ID] = target
			}
		}
	}

	results := make([]*Concept, 0, len(conceptMap))
	for _, concept := range conceptMap {
		results = append(results, concept)
	}

	return results
}

func (sm *SemanticMemory) queryByProperty(query SemanticQuery) []*Concept {
	var results []*Concept

	for _, concept := range sm.concepts {
		match := true
		for key, value := range query.Properties {
			if prop, exists := concept.Properties[key]; !exists || prop.Value != value {
				match = false
				break
			}
		}
		if match {
			results = append(results, concept)
		}
	}

	return results
}

func (sm *SemanticMemory) queryBySimilarity(query SemanticQuery) []*Concept {
	if query.Concept == "" {
		return []*Concept{}
	}

	// Get similar concepts
	similar, err := sm.GetSimilarConcepts(context.Background(), query.Concept, 10)
	if err != nil {
		sm.logger.Warn("Failed to get similar concepts",
			slog.String("concept", query.Concept),
			slog.Any("error", err))
		return []*Concept{}
	}

	return similar
}

func (sm *SemanticMemory) applyQueryConstraints(concepts []*Concept, constraints []QueryConstraint) []*Concept {
	if len(constraints) == 0 {
		return concepts
	}

	filtered := make([]*Concept, 0)
	for _, concept := range concepts {
		match := true
		for _, constraint := range constraints {
			if !sm.checkConstraint(concept, constraint) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, concept)
		}
	}

	return filtered
}

func (sm *SemanticMemory) checkConstraint(concept *Concept, constraint QueryConstraint) bool {
	switch constraint.Type {
	case "confidence":
		if minConf, ok := constraint.Value.(float64); ok {
			return concept.Confidence >= minConf
		}
	case "usage":
		if minUsage, ok := constraint.Value.(int); ok {
			return concept.UsageCount >= minUsage
		}
	case "type":
		if typeStr, ok := constraint.Value.(string); ok {
			return string(concept.Type) == typeStr
		}
	}
	return true
}

func (sm *SemanticMemory) initializeReasoningStrategies() {
	sm.reasoningEngine.strategies[string(ReasoningTypeInference)] = &InferenceStrategy{}
	sm.reasoningEngine.strategies[string(ReasoningTypeAnalogy)] = &AnalogyStrategy{}
	sm.reasoningEngine.strategies[string(ReasoningTypeAbstraction)] = &AbstractionStrategy{}
}

func (sm *SemanticMemory) runMaintenanceLoop() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()

		// Clean up unused concepts
		for id, concept := range sm.concepts {
			if concept.UsageCount == 0 && time.Since(concept.CreatedAt) > 30*24*time.Hour {
				delete(sm.concepts, id)
				sm.logger.Debug("Removed unused concept",
					slog.String("concept_id", id))
			}
		}

		// Clean reasoning cache
		sm.reasoningEngine.cache.clean()

		sm.mu.Unlock()
	}
}

func (sm *SemanticMemory) runLearningLoop() {
	// Placeholder for continuous learning
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// Analyze patterns and learn new concepts
		sm.analyzeAndLearn()
	}
}

func (sm *SemanticMemory) analyzeAndLearn() {
	// Placeholder for pattern analysis and learning
	sm.logger.Debug("Running learning analysis")
}

// Reasoning strategy implementations

// InferenceStrategy implements logical inference
type InferenceStrategy struct{}

func (s *InferenceStrategy) Reason(query ReasoningQuery, memory *SemanticMemory) (*ReasoningResult, error) {
	result := &ReasoningResult{
		Conclusions: make([]Conclusion, 0),
		Reasoning:   make([]ReasoningStep, 0),
	}

	// Apply inference rules
	memory.ontology.mu.RLock()
	defer memory.ontology.mu.RUnlock()

	for _, rule := range memory.ontology.Rules {
		if s.matchesConditions(query, rule.Conditions, memory) {
			conclusion := Conclusion{
				Statement:  s.formatConclusion(rule.Conclusion),
				Confidence: rule.Conclusion.Confidence,
				Support:    []string{rule.Name},
			}
			result.Conclusions = append(result.Conclusions, conclusion)

			step := ReasoningStep{
				Step:       len(result.Reasoning) + 1,
				Operation:  "inference",
				Input:      []string{query.Subject, query.Predicate, query.Object},
				Output:     conclusion.Statement,
				Confidence: conclusion.Confidence,
			}
			result.Reasoning = append(result.Reasoning, step)
		}
	}

	if len(result.Conclusions) > 0 {
		result.Confidence = result.Conclusions[0].Confidence
	}

	return result, nil
}

func (s *InferenceStrategy) GetName() string {
	return "inference"
}

func (s *InferenceStrategy) matchesConditions(query ReasoningQuery, conditions []RuleCondition, memory *SemanticMemory) bool {
	// Simplified condition matching
	for _, condition := range conditions {
		if condition.Subject != query.Subject && condition.Subject != "*" {
			return false
		}
		if condition.Predicate != query.Predicate && condition.Predicate != "*" {
			return false
		}
		if condition.Object != query.Object && condition.Object != "*" {
			return false
		}
	}
	return true
}

func (s *InferenceStrategy) formatConclusion(conclusion RuleConclusion) string {
	return fmt.Sprintf("%s %s %s", conclusion.Subject, conclusion.Predicate, conclusion.Object)
}

// AnalogyStrategy implements analogical reasoning
type AnalogyStrategy struct{}

func (s *AnalogyStrategy) Reason(query ReasoningQuery, memory *SemanticMemory) (*ReasoningResult, error) {
	// Placeholder implementation
	return &ReasoningResult{
		Conclusions: []Conclusion{
			{
				Statement:  "Analogical reasoning not yet implemented",
				Confidence: 0.0,
				Support:    []string{},
			},
		},
		Confidence: 0.0,
	}, nil
}

func (s *AnalogyStrategy) GetName() string {
	return "analogy"
}

// AbstractionStrategy implements abstraction reasoning
type AbstractionStrategy struct{}

func (s *AbstractionStrategy) Reason(query ReasoningQuery, memory *SemanticMemory) (*ReasoningResult, error) {
	// Placeholder implementation
	return &ReasoningResult{
		Conclusions: []Conclusion{
			{
				Statement:  "Abstraction reasoning not yet implemented",
				Confidence: 0.0,
				Support:    []string{},
			},
		},
		Confidence: 0.0,
	}, nil
}

func (s *AbstractionStrategy) GetName() string {
	return "abstraction"
}

// Cache methods

func (c *ReasoningCache) get(key string) *ReasoningResult {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.results[key]
}

func (c *ReasoningCache) set(key string, result *ReasoningResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results[key] = result
}

func (c *ReasoningCache) clean() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple cache cleanup - remove old entries
	c.results = make(map[string]*ReasoningResult)
}

// Utility functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Close gracefully shuts down semantic memory
func (sm *SemanticMemory) Close() error {
	sm.logger.Info("Semantic memory shut down")
	return nil
}
