package context

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SemanticContext handles semantic understanding and intent analysis
type SemanticContext struct {
	intentClassifier *IntentClassifier
	entityExtractor  *EntityExtractor
	conceptMap       *ConceptMap
	domainKnowledge  *DomainKnowledge
	logger           *slog.Logger
	mu               sync.RWMutex
}

// IntentClassifier classifies user intents from queries
type IntentClassifier struct {
	patterns map[IntentType][]Pattern
	weights  map[IntentType]float64
	mu       sync.RWMutex
}

// IntentType defines types of user intentions
type IntentType string

const (
	IntentQuery    IntentType = "query"    // Information seeking
	IntentCommand  IntentType = "command"  // Action execution
	IntentDebug    IntentType = "debug"    // Problem solving
	IntentExplain  IntentType = "explain"  // Understanding request
	IntentOptimize IntentType = "optimize" // Improvement request
	IntentCreate   IntentType = "create"   // Creation request
	IntentModify   IntentType = "modify"   // Modification request
	IntentAnalyze  IntentType = "analyze"  // Analysis request
	IntentLearn    IntentType = "learn"    // Learning request
	IntentHelp     IntentType = "help"     // Help request
	IntentUnknown  IntentType = "unknown"  // Cannot classify
)

// Pattern represents a linguistic pattern for intent classification
type Pattern struct {
	Regex      *regexp.Regexp
	Keywords   []string
	Context    []string
	Confidence float64
}

// EntityExtractor extracts entities from text
type EntityExtractor struct {
	entityTypes map[EntityType]*EntityPattern
	mu          sync.RWMutex
}

// EntityType defines types of entities that can be extracted
type EntityType string

const (
	EntityFile       EntityType = "file"
	EntityDirectory  EntityType = "directory"
	EntityFunction   EntityType = "function"
	EntityClass      EntityType = "class"
	EntityVariable   EntityType = "variable"
	EntityDatabase   EntityType = "database"
	EntityTable      EntityType = "table"
	EntityService    EntityType = "service"
	EntityTechnology EntityType = "technology"
	EntityCommand    EntityType = "command"
	EntityError      EntityType = "error"
	EntityURL        EntityType = "url"
	EntityTime       EntityType = "time"
	EntityNumber     EntityType = "number"
)

// EntityPattern defines patterns for entity extraction
type EntityPattern struct {
	Patterns []string
	Regex    []*regexp.Regexp
	Context  []string
}

// Entity represents an extracted entity
type Entity struct {
	Type       EntityType
	Value      string
	Context    string
	Confidence float64
	Position   int
	Metadata   map[string]interface{}
}

// ConceptMap manages relationships between concepts
type ConceptMap struct {
	concepts  map[string]*Concept
	relations map[string][]Relation
	domains   map[string]*Domain
	mu        sync.RWMutex
}

// Concept represents a semantic concept
type Concept struct {
	ID         string
	Name       string
	Type       ConceptType
	Domain     string
	Definition string
	Synonyms   []string
	Related    []string
	Confidence float64
	LastSeen   time.Time
	UsageCount int
}

// ConceptType defines types of concepts
type ConceptType string

const (
	ConceptTechnical  ConceptType = "technical"
	ConceptFunctional ConceptType = "functional"
	ConceptDomain     ConceptType = "domain"
	ConceptAction     ConceptType = "action"
	ConceptObject     ConceptType = "object"
)

// Relation represents a relationship between concepts
type Relation struct {
	From       string
	To         string
	Type       RelationType
	Strength   float64
	Confidence float64
}

// RelationType defines types of concept relations
type RelationType string

const (
	RelationIsA        RelationType = "is_a"
	RelationPartOf     RelationType = "part_of"
	RelationUsedBy     RelationType = "used_by"
	RelationDependsOn  RelationType = "depends_on"
	RelationSimilarTo  RelationType = "similar_to"
	RelationOppositeOf RelationType = "opposite_of"
)

// Domain represents a knowledge domain
type Domain struct {
	ID        string
	Name      string
	Concepts  []string
	Patterns  []DomainPattern
	Expertise float64
}

// DomainPattern represents patterns specific to a domain
type DomainPattern struct {
	Pattern  string
	Weight   float64
	Context  []string
	Examples []string
}

// DomainKnowledge manages domain-specific knowledge
type DomainKnowledge struct {
	domains map[string]*Domain
	mu      sync.RWMutex
}

// SemanticInfo contains semantic analysis results
type SemanticInfo struct {
	Intent      IntentType
	Entities    []Entity
	Concepts    []Concept
	Domain      string
	Confidence  float64
	Complexity  float64
	Ambiguity   float64
	Suggestions []Suggestion
}

// Suggestion represents a semantic suggestion
type Suggestion struct {
	Type       SuggestionType
	Content    string
	Confidence float64
	Reasoning  string
}

// SuggestionType defines types of suggestions
type SuggestionType string

const (
	SuggestionClarification SuggestionType = "clarification"
	SuggestionAlternative   SuggestionType = "alternative"
	SuggestionCompletion    SuggestionType = "completion"
	SuggestionCorrection    SuggestionType = "correction"
)

// SemanticState represents current semantic state
type SemanticState struct {
	RecentIntents  []IntentType
	ActiveConcepts []string
	DomainFocus    string
	LastUpdate     time.Time
}

// NewSemanticContext creates a new semantic context
func NewSemanticContext(logger *slog.Logger) (*SemanticContext, error) {
	classifier, err := NewIntentClassifier()
	if err != nil {
		return nil, fmt.Errorf("failed to create intent classifier: %w", err)
	}

	extractor, err := NewEntityExtractor()
	if err != nil {
		return nil, fmt.Errorf("failed to create entity extractor: %w", err)
	}

	conceptMap, err := NewConceptMap()
	if err != nil {
		return nil, fmt.Errorf("failed to create concept map: %w", err)
	}

	domainKnowledge, err := NewDomainKnowledge()
	if err != nil {
		return nil, fmt.Errorf("failed to create domain knowledge: %w", err)
	}

	return &SemanticContext{
		intentClassifier: classifier,
		entityExtractor:  extractor,
		conceptMap:       conceptMap,
		domainKnowledge:  domainKnowledge,
		logger:           logger,
	}, nil
}

// ExtractMeaning extracts semantic meaning from a request
func (sc *SemanticContext) ExtractMeaning(ctx context.Context, request Request) (SemanticInfo, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	query := strings.TrimSpace(request.Query)
	if query == "" {
		return SemanticInfo{
			Intent:     IntentUnknown,
			Confidence: 0.0,
		}, nil
	}

	sc.logger.Debug("Extracting semantic meaning",
		slog.String("query", query),
		slog.String("type", request.Type))

	// Classify intent
	intent, intentConfidence := sc.intentClassifier.Classify(query)

	// Extract entities
	entities := sc.entityExtractor.Extract(query)

	// Identify concepts
	concepts := sc.conceptMap.IdentifyConcepts(query)

	// Determine domain
	domain := sc.domainKnowledge.IdentifyDomain(query, entities, concepts)

	// Calculate overall confidence
	confidence := sc.calculateConfidence(intentConfidence, entities, concepts, domain)

	// Calculate complexity
	complexity := sc.calculateComplexity(query, entities, concepts)

	// Calculate ambiguity
	ambiguity := sc.calculateAmbiguity(query, intent, entities)

	// Generate suggestions
	suggestions := sc.generateSuggestions(query, intent, entities, concepts, ambiguity)

	semanticInfo := SemanticInfo{
		Intent:      intent,
		Entities:    entities,
		Concepts:    concepts,
		Domain:      domain,
		Confidence:  confidence,
		Complexity:  complexity,
		Ambiguity:   ambiguity,
		Suggestions: suggestions,
	}

	// Update usage statistics
	go sc.updateUsageStats(intent, entities, concepts, domain)

	sc.logger.Info("Semantic meaning extracted",
		slog.String("intent", string(intent)),
		slog.Float64("confidence", confidence),
		slog.String("domain", domain),
		slog.Int("entities", len(entities)),
		slog.Int("concepts", len(concepts)))

	return semanticInfo, nil
}

// ProcessUpdate processes semantic context updates
func (sc *SemanticContext) ProcessUpdate(ctx context.Context, update ContextUpdate) error {
	// Semantic context can learn from user interactions and feedback
	return nil
}

// GetCurrentState returns current semantic state
func (sc *SemanticContext) GetCurrentState() SemanticState {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Return current semantic state
	return SemanticState{
		RecentIntents:  []IntentType{},
		ActiveConcepts: []string{},
		DomainFocus:    "",
		LastUpdate:     time.Now(),
	}
}

// NewIntentClassifier creates a new intent classifier
func NewIntentClassifier() (*IntentClassifier, error) {
	classifier := &IntentClassifier{
		patterns: make(map[IntentType][]Pattern),
		weights:  make(map[IntentType]float64),
	}

	// Initialize with default patterns
	classifier.initializePatterns()

	return classifier, nil
}

// Classify classifies the intent of a query
func (ic *IntentClassifier) Classify(query string) (IntentType, float64) {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	bestIntent := IntentUnknown
	bestScore := 0.0

	for intentType, patterns := range ic.patterns {
		score := ic.scoreIntent(query, patterns)
		if score > bestScore {
			bestScore = score
			bestIntent = intentType
		}
	}

	// Apply intent weights
	if weight, exists := ic.weights[bestIntent]; exists {
		bestScore *= weight
	}

	return bestIntent, bestScore
}

// initializePatterns initializes default intent patterns
func (ic *IntentClassifier) initializePatterns() {
	// Query patterns
	ic.patterns[IntentQuery] = []Pattern{
		{
			Keywords:   []string{"what", "how", "why", "when", "where", "which"},
			Confidence: 0.8,
		},
		{
			Keywords:   []string{"show", "list", "find", "search", "look"},
			Confidence: 0.7,
		},
	}

	// Command patterns
	ic.patterns[IntentCommand] = []Pattern{
		{
			Keywords:   []string{"run", "execute", "start", "stop", "restart", "deploy"},
			Confidence: 0.9,
		},
		{
			Keywords:   []string{"create", "make", "build", "generate"},
			Confidence: 0.8,
		},
	}

	// Debug patterns
	ic.patterns[IntentDebug] = []Pattern{
		{
			Keywords:   []string{"debug", "fix", "error", "issue", "problem", "bug"},
			Confidence: 0.9,
		},
		{
			Keywords:   []string{"failing", "broken", "not working", "crash"},
			Confidence: 0.8,
		},
	}

	// Explain patterns
	ic.patterns[IntentExplain] = []Pattern{
		{
			Keywords:   []string{"explain", "describe", "tell me about", "what is"},
			Confidence: 0.9,
		},
	}

	// Optimize patterns
	ic.patterns[IntentOptimize] = []Pattern{
		{
			Keywords:   []string{"optimize", "improve", "faster", "better", "performance"},
			Confidence: 0.9,
		},
	}

	// Help patterns
	ic.patterns[IntentHelp] = []Pattern{
		{
			Keywords:   []string{"help", "assist", "guide", "how to"},
			Confidence: 0.8,
		},
	}

	// Set default weights
	ic.weights[IntentQuery] = 1.0
	ic.weights[IntentCommand] = 1.1
	ic.weights[IntentDebug] = 1.2
	ic.weights[IntentExplain] = 1.0
	ic.weights[IntentOptimize] = 1.1
	ic.weights[IntentHelp] = 0.9
}

// scoreIntent calculates the score for an intent given the query
func (ic *IntentClassifier) scoreIntent(query string, patterns []Pattern) float64 {
	totalScore := 0.0

	for _, pattern := range patterns {
		score := 0.0

		// Check keywords
		for _, keyword := range pattern.Keywords {
			if strings.Contains(query, keyword) {
				score += pattern.Confidence
				break // Only count once per pattern
			}
		}

		// Check regex if present
		if pattern.Regex != nil && pattern.Regex.MatchString(query) {
			score += pattern.Confidence
		}

		totalScore += score
	}

	return totalScore
}

// NewEntityExtractor creates a new entity extractor
func NewEntityExtractor() (*EntityExtractor, error) {
	extractor := &EntityExtractor{
		entityTypes: make(map[EntityType]*EntityPattern),
	}

	extractor.initializePatterns()
	return extractor, nil
}

// Extract extracts entities from text
func (ee *EntityExtractor) Extract(text string) []Entity {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	var entities []Entity

	for entityType, pattern := range ee.entityTypes {
		for _, regex := range pattern.Regex {
			matches := regex.FindAllStringSubmatch(text, -1)
			for _, match := range matches {
				if len(match) > 1 {
					entity := Entity{
						Type:       entityType,
						Value:      match[1],
						Confidence: 0.8, // Default confidence
						Position:   strings.Index(text, match[0]),
						Metadata:   make(map[string]interface{}),
					}
					entities = append(entities, entity)
				}
			}
		}
	}

	return entities
}

// initializePatterns initializes entity extraction patterns
func (ee *EntityExtractor) initializePatterns() {
	// File patterns
	ee.entityTypes[EntityFile] = &EntityPattern{
		Regex: []*regexp.Regexp{
			regexp.MustCompile(`([a-zA-Z0-9_\-\.]+\.(go|js|py|java|cpp|c|rb|php))`),
			regexp.MustCompile(`file\s+([a-zA-Z0-9_\-\.\/]+)`),
		},
	}

	// Function patterns
	ee.entityTypes[EntityFunction] = &EntityPattern{
		Regex: []*regexp.Regexp{
			regexp.MustCompile(`function\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
			regexp.MustCompile(`func\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
		},
	}

	// Database patterns
	ee.entityTypes[EntityDatabase] = &EntityPattern{
		Regex: []*regexp.Regexp{
			regexp.MustCompile(`database\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
			regexp.MustCompile(`db\s+([a-zA-Z_][a-zA-Z0-9_]*)`),
		},
	}

	// Technology patterns
	ee.entityTypes[EntityTechnology] = &EntityPattern{
		Regex: []*regexp.Regexp{
			regexp.MustCompile(`\b(docker|kubernetes|k8s|postgres|mysql|redis|nginx)\b`),
		},
	}

	// Command patterns
	ee.entityTypes[EntityCommand] = &EntityPattern{
		Regex: []*regexp.Regexp{
			regexp.MustCompile(`\$\s*([a-zA-Z0-9_\-\s]+)`),
			regexp.MustCompile("`([^`]+)`"),
		},
	}
}

// NewConceptMap creates a new concept map
func NewConceptMap() (*ConceptMap, error) {
	return &ConceptMap{
		concepts:  make(map[string]*Concept),
		relations: make(map[string][]Relation),
		domains:   make(map[string]*Domain),
	}, nil
}

// IdentifyConcepts identifies concepts in text
func (cm *ConceptMap) IdentifyConcepts(text string) []Concept {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var concepts []Concept
	text = strings.ToLower(text)

	for _, concept := range cm.concepts {
		// Check if concept name or synonyms appear in text
		if strings.Contains(text, strings.ToLower(concept.Name)) {
			concepts = append(concepts, *concept)
			continue
		}

		for _, synonym := range concept.Synonyms {
			if strings.Contains(text, strings.ToLower(synonym)) {
				concepts = append(concepts, *concept)
				break
			}
		}
	}

	return concepts
}

// NewDomainKnowledge creates new domain knowledge
func NewDomainKnowledge() (*DomainKnowledge, error) {
	dk := &DomainKnowledge{
		domains: make(map[string]*Domain),
	}

	dk.initializeDomains()
	return dk, nil
}

// IdentifyDomain identifies the domain of a query
func (dk *DomainKnowledge) IdentifyDomain(query string, entities []Entity, concepts []Concept) string {
	dk.mu.RLock()
	defer dk.mu.RUnlock()

	domainScores := make(map[string]float64)

	// Score based on entities
	for _, entity := range entities {
		switch entity.Type {
		case EntityDatabase, EntityTable:
			domainScores["database"] += 0.3
		case EntityFile, EntityFunction:
			domainScores["development"] += 0.3
		case EntityService:
			domainScores["infrastructure"] += 0.3
		}
	}

	// Score based on concepts
	for _, concept := range concepts {
		if concept.Domain != "" {
			domainScores[concept.Domain] += 0.2
		}
	}

	// Find highest scoring domain
	bestDomain := "general"
	bestScore := 0.0

	for domain, score := range domainScores {
		if score > bestScore {
			bestScore = score
			bestDomain = domain
		}
	}

	return bestDomain
}

// initializeDomains initializes default domains
func (dk *DomainKnowledge) initializeDomains() {
	// Database domain
	dk.domains["database"] = &Domain{
		ID:   "database",
		Name: "Database",
		Patterns: []DomainPattern{
			{Pattern: "sql", Weight: 0.9},
			{Pattern: "query", Weight: 0.8},
			{Pattern: "table", Weight: 0.8},
		},
		Expertise: 0.8,
	}

	// Development domain
	dk.domains["development"] = &Domain{
		ID:   "development",
		Name: "Development",
		Patterns: []DomainPattern{
			{Pattern: "code", Weight: 0.9},
			{Pattern: "function", Weight: 0.8},
			{Pattern: "class", Weight: 0.8},
		},
		Expertise: 0.8,
	}

	// Infrastructure domain
	dk.domains["infrastructure"] = &Domain{
		ID:   "infrastructure",
		Name: "Infrastructure",
		Patterns: []DomainPattern{
			{Pattern: "kubernetes", Weight: 0.9},
			{Pattern: "docker", Weight: 0.9},
			{Pattern: "deploy", Weight: 0.8},
		},
		Expertise: 0.7,
	}
}

// Helper methods

func (sc *SemanticContext) calculateConfidence(intentConfidence float64, entities []Entity, concepts []Concept, domain string) float64 {
	// Base confidence from intent classification
	confidence := intentConfidence * 0.4

	// Add confidence from entity extraction
	if len(entities) > 0 {
		entityConfidence := 0.0
		for _, entity := range entities {
			entityConfidence += entity.Confidence
		}
		confidence += (entityConfidence / float64(len(entities))) * 0.3
	}

	// Add confidence from concept identification
	if len(concepts) > 0 {
		conceptConfidence := 0.0
		for _, concept := range concepts {
			conceptConfidence += concept.Confidence
		}
		confidence += (conceptConfidence / float64(len(concepts))) * 0.2
	}

	// Add confidence from domain identification
	if domain != "general" {
		confidence += 0.1
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func (sc *SemanticContext) calculateComplexity(query string, entities []Entity, concepts []Concept) float64 {
	// Simple complexity calculation based on various factors
	complexity := 0.0

	// Length factor
	complexity += float64(len(strings.Split(query, " "))) * 0.05

	// Entity factor
	complexity += float64(len(entities)) * 0.1

	// Concept factor
	complexity += float64(len(concepts)) * 0.1

	if complexity > 1.0 {
		complexity = 1.0
	}

	return complexity
}

func (sc *SemanticContext) calculateAmbiguity(query string, intent IntentType, entities []Entity) float64 {
	// Simplified ambiguity calculation
	ambiguity := 0.0

	// Check for ambiguous words
	ambiguousWords := []string{"it", "this", "that", "them", "they"}
	queryLower := strings.ToLower(query)

	for _, word := range ambiguousWords {
		if strings.Contains(queryLower, word) {
			ambiguity += 0.2
		}
	}

	// Low confidence intent increases ambiguity
	if intent == IntentUnknown {
		ambiguity += 0.3
	}

	if ambiguity > 1.0 {
		ambiguity = 1.0
	}

	return ambiguity
}

func (sc *SemanticContext) generateSuggestions(query string, intent IntentType, entities []Entity, concepts []Concept, ambiguity float64) []Suggestion {
	var suggestions []Suggestion

	// Generate clarification suggestions for high ambiguity
	if ambiguity > 0.5 {
		suggestions = append(suggestions, Suggestion{
			Type:       SuggestionClarification,
			Content:    "Could you provide more specific details about what you're trying to do?",
			Confidence: 0.8,
			Reasoning:  "High ambiguity in query",
		})
	}

	// Generate completion suggestions for partial queries
	if intent == IntentUnknown && len(entities) > 0 {
		suggestions = append(suggestions, Suggestion{
			Type:       SuggestionCompletion,
			Content:    "Did you want to perform an action on " + entities[0].Value + "?",
			Confidence: 0.6,
			Reasoning:  "Detected entities but unclear intent",
		})
	}

	return suggestions
}

func (sc *SemanticContext) updateUsageStats(intent IntentType, entities []Entity, concepts []Concept, domain string) {
	// Update usage statistics for learning
	// This would be implemented to track usage patterns and improve classification
}

// Close shuts down the semantic context
func (sc *SemanticContext) Close(ctx context.Context) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.logger.Info("Shutting down semantic context")

	// Clear data
	sc.intentClassifier = nil
	sc.entityExtractor = nil
	sc.conceptMap = nil
	sc.domainKnowledge = nil

	return nil
}
