package memory

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/koopa0/assistant-go/internal/core/events"
)

// EpisodicMemory stores experiences and events with temporal context
type EpisodicMemory struct {
	episodes        map[string]*Episode
	temporalIndex   *TemporalIndex
	contextIndex    *ContextualIndex
	emotionalIndex  *EmotionalIndex
	retrievalEngine *RetrievalEngine
	encoder         *EpisodeEncoder
	consolidator    *EpisodeConsolidator
	maxEpisodes     int
	eventBus        *events.EventBus
	logger          *slog.Logger
	mu              sync.RWMutex
}

// Episode represents a memory of an experience or event
type Episode struct {
	ID                 string
	Type               EpisodeType
	When               time.Time
	Where              LocationContext
	What               EventDetails
	Who                []Actor
	Why                CausalContext
	How                ProcessContext
	EmotionalTone      EmotionalContext
	Importance         float64
	Vividness          float64
	Coherence          float64
	Associations       []Association
	RetrievalCount     int
	LastRetrieved      *time.Time
	ConsolidationLevel int
	Metadata           map[string]interface{}
}

// EpisodeType defines types of episodes
type EpisodeType string

const (
	EpisodeTypeEvent        EpisodeType = "event"
	EpisodeTypeConversation EpisodeType = "conversation"
	EpisodeTypeTask         EpisodeType = "task"
	EpisodeTypeLearning     EpisodeType = "learning"
	EpisodeTypeError        EpisodeType = "error"
	EpisodeTypeSuccess      EpisodeType = "success"
	EpisodeTypeDiscovery    EpisodeType = "discovery"
)

// LocationContext represents where an episode occurred
type LocationContext struct {
	Physical     string   // File path, directory, etc.
	Logical      string   // Function, class, module, etc.
	Conceptual   string   // Problem space, domain, etc.
	Hierarchical []string // Breadcrumb trail
}

// EventDetails represents what happened in an episode
type EventDetails struct {
	Action     string
	Objects    []string
	Outcome    string
	Duration   time.Duration
	Complexity float64
	Uniqueness float64
}

// Actor represents who was involved in an episode
type Actor struct {
	ID      string
	Type    ActorType
	Role    string
	Actions []string
}

// ActorType defines types of actors in episodes
type ActorType string

const (
	ActorTypeUser   ActorType = "user"
	ActorTypeAgent  ActorType = "agent"
	ActorTypeTool   ActorType = "tool"
	ActorTypeSystem ActorType = "system"
)

// CausalContext represents why something happened
type CausalContext struct {
	Goal        string
	Motivation  string
	Triggers    []string
	Constraints []string
	Intentions  []string
}

// ProcessContext represents how something happened
type ProcessContext struct {
	Steps       []ProcessStep
	Strategy    string
	Techniques  []string
	Resources   []string
	Challenges  []string
	Adaptations []string
}

// ProcessStep represents a step in a process
type ProcessStep struct {
	Order    int
	Action   string
	Duration time.Duration
	Success  bool
	Details  string
}

// EmotionalContext represents the emotional tone of an episode
type EmotionalContext struct {
	Valence   float64 // Positive/negative
	Arousal   float64 // High/low energy
	Dominance float64 // Control level
	Emotions  []Emotion
	Sentiment float64
}

// Emotion represents an emotional component
type Emotion struct {
	Type      EmotionType
	Intensity float64
	Target    string
}

// EmotionType defines types of emotions
type EmotionType string

const (
	EmotionSatisfaction EmotionType = "satisfaction"
	EmotionFrustration  EmotionType = "frustration"
	EmotionCuriosity    EmotionType = "curiosity"
	EmotionConfidence   EmotionType = "confidence"
	EmotionConfusion    EmotionType = "confusion"
	EmotionSurprise     EmotionType = "surprise"
)

// Association represents connections between episodes
type Association struct {
	TargetID      string
	Type          AssociationType
	Strength      float64
	Context       string
	Bidirectional bool
}

// AssociationType defines types of associations
type AssociationType string

const (
	AssociationCausal     AssociationType = "causal"
	AssociationTemporal   AssociationType = "temporal"
	AssociationSimilarity AssociationType = "similarity"
	AssociationContrast   AssociationType = "contrast"
	AssociationPart       AssociationType = "part"
	AssociationWhole      AssociationType = "whole"
)

// TemporalIndex indexes episodes by time
type TemporalIndex struct {
	timeline     map[int64][]*Episode  // Unix timestamp -> episodes
	dayIndex     map[string][]*Episode // YYYY-MM-DD -> episodes
	recentWindow []*Episode
	windowSize   int
	mu           sync.RWMutex
}

// ContextualIndex indexes episodes by context
type ContextualIndex struct {
	locationIndex map[string][]*Episode
	actorIndex    map[string][]*Episode
	goalIndex     map[string][]*Episode
	tagIndex      map[string][]*Episode
	mu            sync.RWMutex
}

// EmotionalIndex indexes episodes by emotional content
type EmotionalIndex struct {
	valenceBuckets map[int][]*Episode // -10 to +10
	emotionIndex   map[EmotionType][]*Episode
	mu             sync.RWMutex
}

// RetrievalEngine handles memory retrieval
type RetrievalEngine struct {
	strategies    map[string]RetrievalStrategy
	cueProcessors []CueProcessor
	scorers       []RelevanceScorer
	mu            sync.RWMutex
}

// RetrievalStrategy defines how to retrieve memories
type RetrievalStrategy interface {
	Retrieve(cue RetrievalCue, episodes map[string]*Episode) []*Episode
	GetName() string
}

// RetrievalCue represents a cue for memory retrieval
type RetrievalCue struct {
	Type         CueType
	Content      string
	Context      map[string]interface{}
	TimeRange    *TimeRange
	Associations []string
	Emotional    *EmotionalContext
}

// CueType defines types of retrieval cues
type CueType string

const (
	CueTypeFree        CueType = "free"
	CueTypeContextual  CueType = "contextual"
	CueTypeTemporal    CueType = "temporal"
	CueTypeEmotional   CueType = "emotional"
	CueTypeAssociative CueType = "associative"
)

// TimeRange is defined in types.go

// CueProcessor processes retrieval cues
type CueProcessor interface {
	Process(cue RetrievalCue) RetrievalCue
}

// RelevanceScorer scores episode relevance
type RelevanceScorer interface {
	Score(episode *Episode, cue RetrievalCue) float64
}

// EpisodeEncoder encodes episodes for storage
type EpisodeEncoder struct {
	encoders map[string]Encoder
	mu       sync.RWMutex
}

// Encoder encodes episode data
type Encoder interface {
	Encode(episode *Episode) ([]byte, error)
	Decode(data []byte) (*Episode, error)
}

// EpisodeConsolidator consolidates episodes
type EpisodeConsolidator struct {
	threshold  float64
	interval   time.Duration
	maxLevel   int
	strategies []ConsolidationStrategy
	mu         sync.Mutex
}

// ConsolidationStrategy defines how to consolidate episodes
type ConsolidationStrategy interface {
	ShouldConsolidate(episode *Episode) bool
	Consolidate(episode *Episode) error
}

// EpisodicMemoryQuery defines query parameters
type EpisodicMemoryQuery struct {
	Cue           RetrievalCue
	Limit         int
	MinImportance float64
	MinVividness  float64
	SortBy        EpisodeSortCriteria
}

// EpisodeSortCriteria defines how to sort episodes
type EpisodeSortCriteria string

const (
	EpisodeSortByRecency    EpisodeSortCriteria = "recency"
	EpisodeSortByImportance EpisodeSortCriteria = "importance"
	EpisodeSortByVividness  EpisodeSortCriteria = "vividness"
	EpisodeSortByRelevance  EpisodeSortCriteria = "relevance"
	EpisodeSortByRetrieval  EpisodeSortCriteria = "retrieval"
)

// NewEpisodicMemory creates a new episodic memory
func NewEpisodicMemory(maxEpisodes int, eventBus *events.EventBus, logger *slog.Logger) (*EpisodicMemory, error) {
	temporalIndex := &TemporalIndex{
		timeline:     make(map[int64][]*Episode),
		dayIndex:     make(map[string][]*Episode),
		recentWindow: make([]*Episode, 0),
		windowSize:   100,
	}

	contextualIndex := &ContextualIndex{
		locationIndex: make(map[string][]*Episode),
		actorIndex:    make(map[string][]*Episode),
		goalIndex:     make(map[string][]*Episode),
		tagIndex:      make(map[string][]*Episode),
	}

	emotionalIndex := &EmotionalIndex{
		valenceBuckets: make(map[int][]*Episode),
		emotionIndex:   make(map[EmotionType][]*Episode),
	}

	retrievalEngine := &RetrievalEngine{
		strategies:    make(map[string]RetrievalStrategy),
		cueProcessors: make([]CueProcessor, 0),
		scorers:       make([]RelevanceScorer, 0),
	}

	encoder := &EpisodeEncoder{
		encoders: make(map[string]Encoder),
	}

	consolidator := &EpisodeConsolidator{
		threshold:  0.7,
		interval:   24 * time.Hour,
		maxLevel:   3,
		strategies: make([]ConsolidationStrategy, 0),
	}

	em := &EpisodicMemory{
		episodes:        make(map[string]*Episode),
		temporalIndex:   temporalIndex,
		contextIndex:    contextualIndex,
		emotionalIndex:  emotionalIndex,
		retrievalEngine: retrievalEngine,
		encoder:         encoder,
		consolidator:    consolidator,
		maxEpisodes:     maxEpisodes,
		eventBus:        eventBus,
		logger:          logger,
	}

	// Initialize retrieval strategies
	em.initializeRetrievalStrategies()

	// Start background processes
	go em.runMaintenanceLoop()
	go em.runConsolidationLoop()

	return em, nil
}

// Store adds a new episode to memory
func (em *EpisodicMemory) Store(ctx context.Context, episode *Episode) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	// Check capacity
	if len(em.episodes) >= em.maxEpisodes {
		if err := em.evictOldest(); err != nil {
			return fmt.Errorf("failed to evict episode: %w", err)
		}
	}

	// Store episode
	em.episodes[episode.ID] = episode

	// Update indices
	em.updateIndices(episode)

	em.logger.Info("Stored episode",
		slog.String("episode_id", episode.ID),
		slog.String("type", string(episode.Type)),
		slog.Float64("importance", episode.Importance))

	// Publish event
	if em.eventBus != nil {
		event := events.Event{
			Type:   events.EventCustom,
			Source: "episodic_memory",
			Data: map[string]interface{}{
				"action":     "store",
				"episode_id": episode.ID,
				"type":       episode.Type,
			},
		}
		em.eventBus.Publish(ctx, event)
	}

	return nil
}

// Retrieve gets episodes based on a retrieval cue
func (em *EpisodicMemory) Retrieve(ctx context.Context, cue RetrievalCue) ([]*Episode, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// Process cue through processors
	processedCue := em.processCue(cue)

	// Get retrieval strategy
	strategy, exists := em.retrievalEngine.strategies[string(processedCue.Type)]
	if !exists {
		strategy = em.retrievalEngine.strategies["default"]
	}

	// Retrieve episodes
	episodes := strategy.Retrieve(processedCue, em.episodes)

	// Score and rank episodes
	scoredEpisodes := em.scoreEpisodes(episodes, processedCue)

	// Sort by relevance
	sort.Slice(scoredEpisodes, func(i, j int) bool {
		return scoredEpisodes[i].score > scoredEpisodes[j].score
	})

	// Extract episodes
	result := make([]*Episode, 0, len(scoredEpisodes))
	for _, se := range scoredEpisodes {
		result = append(result, se.episode)

		// Update retrieval count
		se.episode.RetrievalCount++
		now := time.Now()
		se.episode.LastRetrieved = &now
	}

	em.logger.Debug("Retrieved episodes",
		slog.Int("count", len(result)),
		slog.String("cue_type", string(cue.Type)))

	return result, nil
}

// Query searches episodic memory with specific criteria
func (em *EpisodicMemory) Query(ctx context.Context, query EpisodicMemoryQuery) ([]*Episode, error) {
	// First retrieve based on cue
	episodes, err := em.Retrieve(ctx, query.Cue)
	if err != nil {
		return nil, err
	}

	// Apply filters
	filtered := make([]*Episode, 0)
	for _, episode := range episodes {
		if episode.Importance >= query.MinImportance &&
			episode.Vividness >= query.MinVividness {
			filtered = append(filtered, episode)
		}
	}

	// Sort results
	em.sortEpisodes(filtered, query.SortBy)

	// Apply limit
	if query.Limit > 0 && len(filtered) > query.Limit {
		filtered = filtered[:query.Limit]
	}

	return filtered, nil
}

// Associate creates associations between episodes
func (em *EpisodicMemory) Associate(ctx context.Context, sourceID, targetID string, associationType AssociationType, strength float64) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	source, exists := em.episodes[sourceID]
	if !exists {
		return fmt.Errorf("source episode not found: %s", sourceID)
	}

	_, exists = em.episodes[targetID]
	if !exists {
		return fmt.Errorf("target episode not found: %s", targetID)
	}

	// Create association
	association := Association{
		TargetID:      targetID,
		Type:          associationType,
		Strength:      strength,
		Context:       fmt.Sprintf("%s->%s", sourceID, targetID),
		Bidirectional: false,
	}

	source.Associations = append(source.Associations, association)

	em.logger.Debug("Created episode association",
		slog.String("source", sourceID),
		slog.String("target", targetID),
		slog.String("type", string(associationType)))

	return nil
}

// Consolidate triggers episode consolidation
func (em *EpisodicMemory) Consolidate(ctx context.Context, episodeID string) error {
	em.mu.Lock()
	defer em.mu.Unlock()

	episode, exists := em.episodes[episodeID]
	if !exists {
		return fmt.Errorf("episode not found: %s", episodeID)
	}

	// Check if should consolidate
	shouldConsolidate := false
	for _, strategy := range em.consolidator.strategies {
		if strategy.ShouldConsolidate(episode) {
			shouldConsolidate = true
			break
		}
	}

	if !shouldConsolidate {
		return nil
	}

	// Apply consolidation
	for _, strategy := range em.consolidator.strategies {
		if err := strategy.Consolidate(episode); err != nil {
			em.logger.Warn("Consolidation strategy failed",
				slog.String("episode_id", episodeID),
				slog.Any("error", err))
		}
	}

	episode.ConsolidationLevel++

	em.logger.Info("Consolidated episode",
		slog.String("episode_id", episodeID),
		slog.Int("level", episode.ConsolidationLevel))

	return nil
}

// GetTimelineView returns episodes in chronological order
func (em *EpisodicMemory) GetTimelineView(ctx context.Context, timeRange TimeRange, limit int) ([]*Episode, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var timeline []*Episode

	// Get episodes in time range
	for _, episode := range em.episodes {
		if episode.When.After(timeRange.Start) && episode.When.Before(timeRange.End) {
			timeline = append(timeline, episode)
		}
	}

	// Sort by time
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].When.Before(timeline[j].When)
	})

	// Apply limit
	if limit > 0 && len(timeline) > limit {
		timeline = timeline[:limit]
	}

	return timeline, nil
}

// GetEmotionalView returns episodes grouped by emotional content
func (em *EpisodicMemory) GetEmotionalView(ctx context.Context, emotionType EmotionType) ([]*Episode, error) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	em.emotionalIndex.mu.RLock()
	episodes := em.emotionalIndex.emotionIndex[emotionType]
	em.emotionalIndex.mu.RUnlock()

	result := make([]*Episode, len(episodes))
	copy(result, episodes)

	return result, nil
}

// GetMetrics returns episodic memory metrics
func (em *EpisodicMemory) GetMetrics() EpisodicMemoryMetrics {
	em.mu.RLock()
	defer em.mu.RUnlock()

	metrics := EpisodicMemoryMetrics{
		TotalEpisodes:   len(em.episodes),
		Capacity:        em.maxEpisodes,
		UtilizationRate: float64(len(em.episodes)) / float64(em.maxEpisodes),
	}

	// Count by type
	metrics.EpisodesByType = make(map[EpisodeType]int)
	totalImportance := 0.0
	totalVividness := 0.0
	totalRetrieval := 0

	for _, episode := range em.episodes {
		metrics.EpisodesByType[episode.Type]++
		totalImportance += episode.Importance
		totalVividness += episode.Vividness
		totalRetrieval += episode.RetrievalCount
	}

	if len(em.episodes) > 0 {
		metrics.AverageImportance = totalImportance / float64(len(em.episodes))
		metrics.AverageVividness = totalVividness / float64(len(em.episodes))
		metrics.AverageRetrieval = float64(totalRetrieval) / float64(len(em.episodes))
	}

	// Consolidation stats
	consolidationLevels := make(map[int]int)
	for _, episode := range em.episodes {
		consolidationLevels[episode.ConsolidationLevel]++
	}
	metrics.ConsolidationLevels = consolidationLevels

	return metrics
}

// EpisodicMemoryMetrics contains metrics about episodic memory
type EpisodicMemoryMetrics struct {
	TotalEpisodes       int
	Capacity            int
	UtilizationRate     float64
	EpisodesByType      map[EpisodeType]int
	AverageImportance   float64
	AverageVividness    float64
	AverageRetrieval    float64
	ConsolidationLevels map[int]int
}

// Helper methods

func (em *EpisodicMemory) updateIndices(episode *Episode) {
	// Update temporal index
	em.temporalIndex.mu.Lock()
	timestamp := episode.When.Unix()
	em.temporalIndex.timeline[timestamp] = append(em.temporalIndex.timeline[timestamp], episode)

	dayKey := episode.When.Format("2006-01-02")
	em.temporalIndex.dayIndex[dayKey] = append(em.temporalIndex.dayIndex[dayKey], episode)

	// Update recent window
	em.temporalIndex.recentWindow = append(em.temporalIndex.recentWindow, episode)
	if len(em.temporalIndex.recentWindow) > em.temporalIndex.windowSize {
		em.temporalIndex.recentWindow = em.temporalIndex.recentWindow[1:]
	}
	em.temporalIndex.mu.Unlock()

	// Update contextual index
	em.contextIndex.mu.Lock()
	if episode.Where.Physical != "" {
		em.contextIndex.locationIndex[episode.Where.Physical] = append(
			em.contextIndex.locationIndex[episode.Where.Physical], episode)
	}
	for _, actor := range episode.Who {
		em.contextIndex.actorIndex[actor.ID] = append(
			em.contextIndex.actorIndex[actor.ID], episode)
	}
	if episode.Why.Goal != "" {
		em.contextIndex.goalIndex[episode.Why.Goal] = append(
			em.contextIndex.goalIndex[episode.Why.Goal], episode)
	}
	em.contextIndex.mu.Unlock()

	// Update emotional index
	em.emotionalIndex.mu.Lock()
	valenceBucket := int(episode.EmotionalTone.Valence * 10)
	if valenceBucket < -10 {
		valenceBucket = -10
	} else if valenceBucket > 10 {
		valenceBucket = 10
	}
	em.emotionalIndex.valenceBuckets[valenceBucket] = append(
		em.emotionalIndex.valenceBuckets[valenceBucket], episode)

	for _, emotion := range episode.EmotionalTone.Emotions {
		em.emotionalIndex.emotionIndex[emotion.Type] = append(
			em.emotionalIndex.emotionIndex[emotion.Type], episode)
	}
	em.emotionalIndex.mu.Unlock()
}

func (em *EpisodicMemory) evictOldest() error {
	var oldestEpisode *Episode
	var oldestID string

	// Find episode with lowest importance and oldest retrieval
	for id, episode := range em.episodes {
		if oldestEpisode == nil ||
			episode.Importance < oldestEpisode.Importance ||
			(episode.Importance == oldestEpisode.Importance &&
				episode.When.Before(oldestEpisode.When)) {
			oldestEpisode = episode
			oldestID = id
		}
	}

	if oldestID != "" {
		delete(em.episodes, oldestID)
		em.logger.Debug("Evicted oldest episode",
			slog.String("episode_id", oldestID),
			slog.Float64("importance", oldestEpisode.Importance))
	}

	return nil
}

func (em *EpisodicMemory) processCue(cue RetrievalCue) RetrievalCue {
	processedCue := cue
	for _, processor := range em.retrievalEngine.cueProcessors {
		processedCue = processor.Process(processedCue)
	}
	return processedCue
}

type scoredEpisode struct {
	episode *Episode
	score   float64
}

func (em *EpisodicMemory) scoreEpisodes(episodes []*Episode, cue RetrievalCue) []scoredEpisode {
	scored := make([]scoredEpisode, 0, len(episodes))

	for _, episode := range episodes {
		totalScore := 0.0
		scorerCount := 0

		for _, scorer := range em.retrievalEngine.scorers {
			score := scorer.Score(episode, cue)
			totalScore += score
			scorerCount++
		}

		if scorerCount > 0 {
			totalScore /= float64(scorerCount)
		}

		scored = append(scored, scoredEpisode{
			episode: episode,
			score:   totalScore,
		})
	}

	return scored
}

func (em *EpisodicMemory) sortEpisodes(episodes []*Episode, criteria EpisodeSortCriteria) {
	switch criteria {
	case EpisodeSortByRecency:
		sort.Slice(episodes, func(i, j int) bool {
			return episodes[i].When.After(episodes[j].When)
		})
	case EpisodeSortByImportance:
		sort.Slice(episodes, func(i, j int) bool {
			return episodes[i].Importance > episodes[j].Importance
		})
	case EpisodeSortByVividness:
		sort.Slice(episodes, func(i, j int) bool {
			return episodes[i].Vividness > episodes[j].Vividness
		})
	case EpisodeSortByRetrieval:
		sort.Slice(episodes, func(i, j int) bool {
			return episodes[i].RetrievalCount > episodes[j].RetrievalCount
		})
	}
}

func (em *EpisodicMemory) initializeRetrievalStrategies() {
	// Add default retrieval strategy
	em.retrievalEngine.strategies["default"] = &FreeRecallStrategy{}
	em.retrievalEngine.strategies[string(CueTypeFree)] = &FreeRecallStrategy{}
	em.retrievalEngine.strategies[string(CueTypeContextual)] = &ContextualRecallStrategy{
		contextIndex: em.contextIndex,
	}
	em.retrievalEngine.strategies[string(CueTypeTemporal)] = &TemporalRecallStrategy{
		temporalIndex: em.temporalIndex,
	}
	em.retrievalEngine.strategies[string(CueTypeEmotional)] = &EmotionalRecallStrategy{
		emotionalIndex: em.emotionalIndex,
	}

	// Add default relevance scorers
	em.retrievalEngine.scorers = append(em.retrievalEngine.scorers,
		&RecencyScorer{},
		&ImportanceScorer{},
		&SimilarityScorer{},
	)
}

func (em *EpisodicMemory) runMaintenanceLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		em.mu.Lock()

		// Decay vividness over time
		for _, episode := range em.episodes {
			age := time.Since(episode.When)
			decayFactor := 1.0 - (age.Hours()/(24*365))*0.1 // 10% decay per year
			if decayFactor < 0.1 {
				decayFactor = 0.1
			}
			episode.Vividness *= decayFactor
		}

		em.mu.Unlock()
	}
}

func (em *EpisodicMemory) runConsolidationLoop() {
	ticker := time.NewTicker(em.consolidator.interval)
	defer ticker.Stop()

	for range ticker.C {
		em.mu.RLock()
		episodesToConsolidate := make([]*Episode, 0)

		for _, episode := range em.episodes {
			if episode.Importance >= em.consolidator.threshold &&
				episode.ConsolidationLevel < em.consolidator.maxLevel {
				episodesToConsolidate = append(episodesToConsolidate, episode)
			}
		}
		em.mu.RUnlock()

		// Consolidate episodes
		for _, episode := range episodesToConsolidate {
			if err := em.Consolidate(context.Background(), episode.ID); err != nil {
				em.logger.Warn("Failed to consolidate episode",
					slog.String("episode_id", episode.ID),
					slog.Any("error", err))
			}
		}
	}
}

// Retrieval strategy implementations

// FreeRecallStrategy implements free recall
type FreeRecallStrategy struct{}

func (s *FreeRecallStrategy) Retrieve(cue RetrievalCue, episodes map[string]*Episode) []*Episode {
	result := make([]*Episode, 0)
	for _, episode := range episodes {
		result = append(result, episode)
	}
	return result
}

func (s *FreeRecallStrategy) GetName() string {
	return "free_recall"
}

// ContextualRecallStrategy implements contextual recall
type ContextualRecallStrategy struct {
	contextIndex *ContextualIndex
}

func (s *ContextualRecallStrategy) Retrieve(cue RetrievalCue, episodes map[string]*Episode) []*Episode {
	s.contextIndex.mu.RLock()
	defer s.contextIndex.mu.RUnlock()

	resultMap := make(map[string]*Episode)

	// Get episodes by context
	if location, ok := cue.Context["location"].(string); ok && location != "" {
		for _, episode := range s.contextIndex.locationIndex[location] {
			resultMap[episode.ID] = episode
		}
	}

	if actor, ok := cue.Context["actor"].(string); ok && actor != "" {
		for _, episode := range s.contextIndex.actorIndex[actor] {
			resultMap[episode.ID] = episode
		}
	}

	if goal, ok := cue.Context["goal"].(string); ok && goal != "" {
		for _, episode := range s.contextIndex.goalIndex[goal] {
			resultMap[episode.ID] = episode
		}
	}

	// Convert to slice
	result := make([]*Episode, 0, len(resultMap))
	for _, episode := range resultMap {
		result = append(result, episode)
	}

	return result
}

func (s *ContextualRecallStrategy) GetName() string {
	return "contextual_recall"
}

// TemporalRecallStrategy implements temporal recall
type TemporalRecallStrategy struct {
	temporalIndex *TemporalIndex
}

func (s *TemporalRecallStrategy) Retrieve(cue RetrievalCue, episodes map[string]*Episode) []*Episode {
	s.temporalIndex.mu.RLock()
	defer s.temporalIndex.mu.RUnlock()

	if cue.TimeRange == nil {
		// Return recent window
		result := make([]*Episode, len(s.temporalIndex.recentWindow))
		copy(result, s.temporalIndex.recentWindow)
		return result
	}

	// Get episodes in time range
	result := make([]*Episode, 0)
	for timestamp, episodeList := range s.temporalIndex.timeline {
		t := time.Unix(timestamp, 0)
		if t.After(cue.TimeRange.Start) && t.Before(cue.TimeRange.End) {
			result = append(result, episodeList...)
		}
	}

	return result
}

func (s *TemporalRecallStrategy) GetName() string {
	return "temporal_recall"
}

// EmotionalRecallStrategy implements emotional recall
type EmotionalRecallStrategy struct {
	emotionalIndex *EmotionalIndex
}

func (s *EmotionalRecallStrategy) Retrieve(cue RetrievalCue, episodes map[string]*Episode) []*Episode {
	s.emotionalIndex.mu.RLock()
	defer s.emotionalIndex.mu.RUnlock()

	if cue.Emotional == nil {
		return []*Episode{}
	}

	// Get episodes by valence
	valenceBucket := int(cue.Emotional.Valence * 10)
	if valenceBucket < -10 {
		valenceBucket = -10
	} else if valenceBucket > 10 {
		valenceBucket = 10
	}

	result := make([]*Episode, 0)

	// Get exact bucket and adjacent buckets
	for i := -1; i <= 1; i++ {
		bucket := valenceBucket + i
		if bucket >= -10 && bucket <= 10 {
			result = append(result, s.emotionalIndex.valenceBuckets[bucket]...)
		}
	}

	return result
}

func (s *EmotionalRecallStrategy) GetName() string {
	return "emotional_recall"
}

// Relevance scorer implementations

// RecencyScorer scores by recency
type RecencyScorer struct{}

func (s *RecencyScorer) Score(episode *Episode, cue RetrievalCue) float64 {
	age := time.Since(episode.When)
	// Exponential decay: more recent = higher score
	return 1.0 / (1.0 + age.Hours()/24.0)
}

// ImportanceScorer scores by importance
type ImportanceScorer struct{}

func (s *ImportanceScorer) Score(episode *Episode, cue RetrievalCue) float64 {
	return episode.Importance
}

// SimilarityScorer scores by content similarity
type SimilarityScorer struct{}

func (s *SimilarityScorer) Score(episode *Episode, cue RetrievalCue) float64 {
	// Simple keyword matching for now
	score := 0.0
	content := strings.ToLower(cue.Content)

	if strings.Contains(strings.ToLower(episode.What.Action), content) {
		score += 0.3
	}
	if strings.Contains(strings.ToLower(episode.What.Outcome), content) {
		score += 0.2
	}
	if strings.Contains(strings.ToLower(episode.Why.Goal), content) {
		score += 0.3
	}

	for _, obj := range episode.What.Objects {
		if strings.Contains(strings.ToLower(obj), content) {
			score += 0.2
			break
		}
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

// Close gracefully shuts down episodic memory
func (em *EpisodicMemory) Close() error {
	em.logger.Info("Episodic memory shut down")
	return nil
}
