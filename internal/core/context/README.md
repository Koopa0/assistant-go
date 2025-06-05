# Context Engine

## Overview

The Context Engine provides multi-dimensional context awareness for the Assistant intelligent development companion. It maintains a living understanding of the development environment, user preferences, temporal relationships, and semantic connections. This system enables the Assistant to provide contextually relevant responses and adapt to changing development scenarios.

## Architecture

```
internal/core/context/
├── engine.go        # Core context engine orchestration
├── workspace.go     # Workspace and project context
├── temporal.go      # Temporal context and history
├── semantic.go      # Semantic understanding and relationships
├── personal.go      # Personal preferences and patterns
└── context_test.go  # Comprehensive test suite
```

## Key Features

### 🌐 **Multi-Dimensional Context**
- **Workspace Context**: Project structure, dependencies, configurations
- **Temporal Context**: Development timeline, change history, patterns
- **Semantic Context**: Code relationships, knowledge graphs, concepts
- **Personal Context**: User preferences, habits, development style

### 🧠 **Intelligent Processing**
- **Context Enrichment**: Automatic context expansion and inference
- **Relevance Scoring**: Dynamic importance weighting
- **Context Fusion**: Intelligent merging of multiple context sources
- **Adaptive Learning**: Continuous improvement from usage patterns

### 🎯 **Production Features**
- **Real-time Updates**: Live context tracking and updates
- **Efficient Storage**: Optimized context persistence
- **Privacy Protection**: Secure handling of personal information
- **Performance**: Low-latency context retrieval

## Core Components

### Context Engine Interface

```go
type ContextEngine interface {
    // Context enrichment
    EnrichRequest(ctx context.Context, request Request) (*EnrichedRequest, error)
    EnrichIntent(ctx context.Context, intent Intent) (*EnrichedIntent, error)
    
    // Context retrieval
    GetContext(ctx context.Context, scope ContextScope) (*Context, error)
    GetRelevantContext(ctx context.Context, query Query) (*RelevantContext, error)
    
    // Context updates
    UpdateContext(ctx context.Context, update ContextUpdate) error
    Subscribe(scope ContextScope, handler UpdateHandler) (Subscription, error)
    
    // Learning and adaptation
    LearnFromInteraction(interaction Interaction) error
    AdaptToPattern(pattern Pattern) error
}
```

### Context Types

```go
type Context struct {
    ID          string                 `json:"id"`
    Type        ContextType           `json:"type"`
    Scope       ContextScope          `json:"scope"`
    Data        map[string]interface{} `json:"data"`
    Metadata    ContextMetadata       `json:"metadata"`
    Relevance   float64               `json:"relevance"`
    Timestamp   time.Time             `json:"timestamp"`
    TTL         time.Duration         `json:"ttl"`
}

type ContextType string
const (
    ContextTypeWorkspace ContextType = "workspace"
    ContextTypeTemporal  ContextType = "temporal"
    ContextTypeSemantic  ContextType = "semantic"
    ContextTypePersonal  ContextType = "personal"
)

type ContextScope struct {
    Level     ScopeLevel   `json:"level"`
    Entity    string       `json:"entity"`
    Filters   []Filter     `json:"filters"`
    TimeRange *TimeRange   `json:"time_range,omitempty"`
}
```

## Context Engine Implementation

### Core Engine

```go
type Engine struct {
    // Context providers
    workspace  WorkspaceContext
    temporal   TemporalContext
    semantic   SemanticContext
    personal   PersonalContext
    
    // Infrastructure
    storage    ContextStorage
    cache      ContextCache
    indexer    ContextIndexer
    
    // Learning
    learner    ContextLearner
    predictor  ContextPredictor
    
    // Configuration
    config     EngineConfig
    logger     *slog.Logger
    metrics    *EngineMetrics
    
    // Subscription management
    subscribers map[string][]UpdateHandler
    mutex       sync.RWMutex
}

func NewEngine(config EngineConfig) (*Engine, error) {
    engine := &Engine{
        config:      config,
        logger:      slog.With("component", "context_engine"),
        metrics:     NewEngineMetrics(),
        subscribers: make(map[string][]UpdateHandler),
    }
    
    // Initialize context providers
    engine.workspace = NewWorkspaceContext(config.Workspace)
    engine.temporal = NewTemporalContext(config.Temporal)
    engine.semantic = NewSemanticContext(config.Semantic)
    engine.personal = NewPersonalContext(config.Personal)
    
    // Initialize infrastructure
    engine.storage = NewContextStorage(config.Storage)
    engine.cache = NewContextCache(config.Cache)
    engine.indexer = NewContextIndexer(config.Indexer)
    
    // Initialize learning systems
    engine.learner = NewContextLearner(config.Learning)
    engine.predictor = NewContextPredictor(config.Prediction)
    
    return engine, nil
}
```

### Context Enrichment

```go
func (e *Engine) EnrichRequest(ctx context.Context, request Request) (*EnrichedRequest, error) {
    start := time.Now()
    defer func() {
        e.metrics.RecordEnrichmentTime(time.Since(start))
    }()
    
    // Create enriched request
    enriched := &EnrichedRequest{
        Original: request,
        Context:  make(map[ContextType]*Context),
    }
    
    // Gather all relevant contexts concurrently
    var wg sync.WaitGroup
    var mu sync.Mutex
    errors := make([]error, 0)
    
    // Workspace context
    wg.Add(1)
    go func() {
        defer wg.Done()
        if ctx, err := e.workspace.GetContext(request); err == nil {
            mu.Lock()
            enriched.Context[ContextTypeWorkspace] = ctx
            mu.Unlock()
        } else {
            mu.Lock()
            errors = append(errors, fmt.Errorf("workspace context: %w", err))
            mu.Unlock()
        }
    }()
    
    // Temporal context
    wg.Add(1)
    go func() {
        defer wg.Done()
        if ctx, err := e.temporal.GetRelevantHistory(request); err == nil {
            mu.Lock()
            enriched.Context[ContextTypeTemporal] = ctx
            mu.Unlock()
        } else {
            mu.Lock()
            errors = append(errors, fmt.Errorf("temporal context: %w", err))
            mu.Unlock()
        }
    }()
    
    // Semantic context
    wg.Add(1)
    go func() {
        defer wg.Done()
        if ctx, err := e.semantic.GetRelatedConcepts(request); err == nil {
            mu.Lock()
            enriched.Context[ContextTypeSemantic] = ctx
            mu.Unlock()
        } else {
            mu.Lock()
            errors = append(errors, fmt.Errorf("semantic context: %w", err))
            mu.Unlock()
        }
    }()
    
    // Personal context
    wg.Add(1)
    go func() {
        defer wg.Done()
        if ctx, err := e.personal.GetPreferences(request); err == nil {
            mu.Lock()
            enriched.Context[ContextTypePersonal] = ctx
            mu.Unlock()
        } else {
            mu.Lock()
            errors = append(errors, fmt.Errorf("personal context: %w", err))
            mu.Unlock()
        }
    }()
    
    wg.Wait()
    
    // Log non-critical errors
    for _, err := range errors {
        e.logger.Warn("Context enrichment error", slog.String("error", err.Error()))
    }
    
    // Apply context fusion
    if err := e.fuseContexts(enriched); err != nil {
        return nil, fmt.Errorf("context fusion: %w", err)
    }
    
    // Predict additional context needs
    if predicted := e.predictor.PredictNeededContext(enriched); predicted != nil {
        enriched.PredictedContext = predicted
    }
    
    return enriched, nil
}
```

### Context Fusion

```go
func (e *Engine) fuseContexts(enriched *EnrichedRequest) error {
    // Calculate relevance scores
    for contextType, context := range enriched.Context {
        relevance := e.calculateRelevance(enriched.Original, context)
        context.Relevance = relevance
        
        // Apply relevance threshold
        if relevance < e.config.MinRelevanceThreshold {
            delete(enriched.Context, contextType)
        }
    }
    
    // Detect and resolve conflicts
    conflicts := e.detectConflicts(enriched.Context)
    for _, conflict := range conflicts {
        resolution := e.resolveConflict(conflict)
        e.applyResolution(enriched, resolution)
    }
    
    // Create unified context view
    unified := e.createUnifiedView(enriched.Context)
    enriched.UnifiedContext = unified
    
    return nil
}

func (e *Engine) calculateRelevance(request Request, context *Context) float64 {
    score := 0.0
    
    // Time-based relevance (exponential decay)
    age := time.Since(context.Timestamp)
    timeRelevance := math.Exp(-age.Hours() / 24) // Half-life of 24 hours
    score += timeRelevance * 0.3
    
    // Semantic similarity
    semanticScore := e.semantic.CalculateSimilarity(request.Content, context.Data)
    score += semanticScore * 0.4
    
    // User interaction history
    interactionScore := e.personal.GetInteractionScore(request.UserID, context.ID)
    score += interactionScore * 0.3
    
    return math.Min(score, 1.0)
}
```

## Workspace Context

### Implementation

```go
type WorkspaceContext struct {
    // Project information
    projects    map[string]*Project
    currentPath string
    fileWatcher *FileWatcher
    
    // Analysis components
    analyzer    *ProjectAnalyzer
    dependency  *DependencyTracker
    structure   *StructureAnalyzer
    
    // State tracking
    state       WorkspaceState
    changes     []WorkspaceChange
    mutex       sync.RWMutex
}

func (wc *WorkspaceContext) GetContext(request Request) (*Context, error) {
    wc.mutex.RLock()
    defer wc.mutex.RUnlock()
    
    // Determine relevant project
    project := wc.determineProject(request)
    if project == nil {
        return nil, ErrNoProjectContext
    }
    
    // Build workspace context
    context := &Context{
        Type:  ContextTypeWorkspace,
        Scope: ContextScope{Level: ScopeLevelProject, Entity: project.Name},
        Data: map[string]interface{}{
            "project":       project,
            "structure":     wc.structure.GetStructure(project),
            "dependencies":  wc.dependency.GetDependencies(project),
            "configuration": project.Configuration,
            "recent_changes": wc.getRecentChanges(project, 10),
        },
        Timestamp: time.Now(),
    }
    
    return context, nil
}
```

### File Watching

```go
func (wc *WorkspaceContext) watchFiles() {
    for event := range wc.fileWatcher.Events {
        switch event.Op {
        case fsnotify.Create, fsnotify.Write:
            wc.handleFileChange(event.Name)
        case fsnotify.Remove:
            wc.handleFileRemoval(event.Name)
        case fsnotify.Rename:
            wc.handleFileRename(event.Name)
        }
        
        // Notify subscribers
        wc.notifySubscribers(WorkspaceChange{
            Type:      ChangeTypeFileModified,
            Path:      event.Name,
            Timestamp: time.Now(),
        })
    }
}

func (wc *WorkspaceContext) handleFileChange(path string) {
    // Analyze file type
    fileType := wc.analyzer.DetermineFileType(path)
    
    switch fileType {
    case FileTypeSource:
        wc.updateSourceAnalysis(path)
    case FileTypeConfig:
        wc.updateConfiguration(path)
    case FileTypeDependency:
        wc.updateDependencies(path)
    }
    
    // Update project structure
    wc.structure.UpdateFile(path)
}
```

## Temporal Context

### Time-Series Management

```go
type TemporalContext struct {
    // Time series storage
    timeseries  TimeSeriesDB
    events      EventStore
    timeline    *Timeline
    
    // Pattern detection
    patterns    *PatternDetector
    anomalies   *AnomalyDetector
    trends      *TrendAnalyzer
    
    // Configuration
    retention   time.Duration
    resolution  time.Duration
}

func (tc *TemporalContext) GetRelevantHistory(request Request) (*Context, error) {
    // Determine time range
    timeRange := tc.determineTimeRange(request)
    
    // Retrieve events
    events, err := tc.events.Query(EventQuery{
        TimeRange: timeRange,
        Filters:   tc.createFilters(request),
        Limit:     100,
    })
    if err != nil {
        return nil, fmt.Errorf("querying events: %w", err)
    }
    
    // Detect patterns
    patterns := tc.patterns.DetectPatterns(events)
    
    // Analyze trends
    trends := tc.trends.AnalyzeTrends(events)
    
    // Build temporal context
    context := &Context{
        Type: ContextTypeTemporal,
        Data: map[string]interface{}{
            "events":    events,
            "patterns":  patterns,
            "trends":    trends,
            "timeline":  tc.timeline.GetSegment(timeRange),
            "frequency": tc.calculateFrequency(events),
        },
        Timestamp: time.Now(),
    }
    
    return context, nil
}
```

### Pattern Detection

```go
func (pd *PatternDetector) DetectPatterns(events []Event) []Pattern {
    var patterns []Pattern
    
    // Sequence patterns
    sequences := pd.findSequencePatterns(events)
    patterns = append(patterns, sequences...)
    
    // Periodic patterns
    periodic := pd.findPeriodicPatterns(events)
    patterns = append(patterns, periodic...)
    
    // Behavioral patterns
    behavioral := pd.findBehavioralPatterns(events)
    patterns = append(patterns, behavioral...)
    
    // Rank by significance
    sort.Slice(patterns, func(i, j int) bool {
        return patterns[i].Significance > patterns[j].Significance
    })
    
    return patterns
}

func (pd *PatternDetector) findSequencePatterns(events []Event) []Pattern {
    // Use suffix tree for efficient pattern matching
    tree := NewSuffixTree()
    for _, event := range events {
        tree.Insert(event.Sequence())
    }
    
    // Extract frequent subsequences
    sequences := tree.GetFrequentPatterns(pd.minSupport)
    
    var patterns []Pattern
    for _, seq := range sequences {
        pattern := Pattern{
            Type:         PatternTypeSequence,
            Description:  fmt.Sprintf("Sequence: %v", seq.Elements),
            Occurrences:  seq.Count,
            Significance: pd.calculateSignificance(seq),
        }
        patterns = append(patterns, pattern)
    }
    
    return patterns
}
```

## Semantic Context

### Knowledge Graph

```go
type SemanticContext struct {
    // Knowledge representation
    graph       *KnowledgeGraph
    embeddings  *EmbeddingStore
    ontology    *Ontology
    
    // Analysis components
    similarity  *SimilarityCalculator
    inference   *InferenceEngine
    conceptMap  *ConceptMapper
}

func (sc *SemanticContext) GetRelatedConcepts(request Request) (*Context, error) {
    // Extract concepts from request
    concepts, err := sc.conceptMap.ExtractConcepts(request.Content)
    if err != nil {
        return nil, fmt.Errorf("extracting concepts: %w", err)
    }
    
    // Find related concepts in knowledge graph
    related := make(map[string]*Concept)
    for _, concept := range concepts {
        neighbors := sc.graph.GetNeighbors(concept.ID, 2) // 2-hop neighborhood
        for _, neighbor := range neighbors {
            related[neighbor.ID] = neighbor
        }
    }
    
    // Calculate embeddings for semantic similarity
    queryEmbedding := sc.embeddings.GetEmbedding(request.Content)
    similarities := sc.calculateSimilarities(queryEmbedding, related)
    
    // Apply inference rules
    inferred := sc.inference.InferRelations(concepts, related)
    
    context := &Context{
        Type: ContextTypeSemantic,
        Data: map[string]interface{}{
            "concepts":     concepts,
            "related":      related,
            "similarities": similarities,
            "inferred":     inferred,
            "graph_view":   sc.graph.GetSubgraph(concepts),
        },
        Timestamp: time.Now(),
    }
    
    return context, nil
}
```

### Embedding-based Similarity

```go
func (sc *SemanticContext) calculateSimilarities(query []float32, concepts map[string]*Concept) map[string]float64 {
    similarities := make(map[string]float64)
    
    for id, concept := range concepts {
        embedding := sc.embeddings.GetEmbedding(concept.Description)
        similarity := sc.similarity.CosineSimilarity(query, embedding)
        similarities[id] = similarity
    }
    
    return similarities
}

type SimilarityCalculator struct {
    cache *lru.Cache
}

func (calc *SimilarityCalculator) CosineSimilarity(a, b []float32) float64 {
    if len(a) != len(b) {
        return 0.0
    }
    
    var dotProduct, normA, normB float64
    for i := range a {
        dotProduct += float64(a[i] * b[i])
        normA += float64(a[i] * a[i])
        normB += float64(b[i] * b[i])
    }
    
    if normA == 0 || normB == 0 {
        return 0.0
    }
    
    return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
```

## Personal Context

### User Preferences

```go
type PersonalContext struct {
    // User profiles
    profiles    map[string]*UserProfile
    preferences *PreferenceStore
    habits      *HabitTracker
    
    // Learning components
    learner     *PreferenceLearner
    predictor   *BehaviorPredictor
    
    // Privacy
    privacy     *PrivacyManager
    anonymizer  *DataAnonymizer
}

func (pc *PersonalContext) GetPreferences(request Request) (*Context, error) {
    // Get user profile
    profile, err := pc.getOrCreateProfile(request.UserID)
    if err != nil {
        return nil, fmt.Errorf("getting user profile: %w", err)
    }
    
    // Check privacy settings
    if !pc.privacy.AllowContextAccess(request.UserID, request.Purpose) {
        return pc.getAnonymizedContext(request)
    }
    
    // Get preferences
    preferences := pc.preferences.GetPreferences(request.UserID)
    
    // Get habits and patterns
    habits := pc.habits.GetHabits(request.UserID)
    
    // Predict behavior
    predictions := pc.predictor.PredictBehavior(profile, request)
    
    context := &Context{
        Type: ContextTypePersonal,
        Data: map[string]interface{}{
            "preferences":   preferences,
            "habits":        habits,
            "profile":       profile.PublicData(),
            "predictions":   predictions,
            "style":         profile.DevelopmentStyle,
        },
        Timestamp: time.Now(),
    }
    
    return context, nil
}
```

### Habit Learning

```go
type HabitTracker struct {
    storage     HabitStorage
    detector    *PatternDetector
    classifier  *HabitClassifier
}

func (ht *HabitTracker) LearnFromInteraction(userID string, interaction Interaction) error {
    // Extract features
    features := ht.extractFeatures(interaction)
    
    // Update habit model
    model, err := ht.storage.GetModel(userID)
    if err != nil {
        model = NewHabitModel(userID)
    }
    
    model.Update(features)
    
    // Detect new habits
    if newHabits := ht.detector.DetectNewHabits(model); len(newHabits) > 0 {
        for _, habit := range newHabits {
            ht.classifier.ClassifyHabit(habit)
            model.AddHabit(habit)
        }
    }
    
    return ht.storage.SaveModel(model)
}

type Habit struct {
    ID          string        `json:"id"`
    Type        HabitType     `json:"type"`
    Pattern     string        `json:"pattern"`
    Frequency   float64       `json:"frequency"`
    Confidence  float64       `json:"confidence"`
    TimeOfDay   []int         `json:"time_of_day"`
    Context     []string      `json:"context"`
    LastSeen    time.Time     `json:"last_seen"`
}
```

## Context Storage

### Persistence Layer

```go
type ContextStorage struct {
    db       *sql.DB
    cache    cache.Cache
    indexer  *ContextIndexer
    
    retention map[ContextType]time.Duration
}

func (cs *ContextStorage) Store(context *Context) error {
    // Serialize context
    data, err := json.Marshal(context)
    if err != nil {
        return fmt.Errorf("marshaling context: %w", err)
    }
    
    // Store in database
    _, err = cs.db.Exec(`
        INSERT INTO contexts (id, type, scope, data, relevance, timestamp, ttl)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (id) DO UPDATE
        SET data = $4, relevance = $5, timestamp = $6, ttl = $7
    `, context.ID, context.Type, context.Scope.String(), data, 
       context.Relevance, context.Timestamp, context.TTL)
    
    if err != nil {
        return fmt.Errorf("storing context: %w", err)
    }
    
    // Update cache
    cs.cache.Set(context.ID, context, context.TTL)
    
    // Update indexes
    if err := cs.indexer.Index(context); err != nil {
        cs.logger.Warn("Failed to index context", slog.String("error", err.Error()))
    }
    
    return nil
}

func (cs *ContextStorage) Query(query ContextQuery) ([]*Context, error) {
    // Check cache first
    if cached := cs.checkCache(query); len(cached) > 0 {
        return cached, nil
    }
    
    // Build SQL query
    sqlQuery, args := cs.buildQuery(query)
    
    rows, err := cs.db.Query(sqlQuery, args...)
    if err != nil {
        return nil, fmt.Errorf("querying contexts: %w", err)
    }
    defer rows.Close()
    
    var contexts []*Context
    for rows.Next() {
        context, err := cs.scanContext(rows)
        if err != nil {
            return nil, fmt.Errorf("scanning context: %w", err)
        }
        contexts = append(contexts, context)
    }
    
    return contexts, nil
}
```

## Configuration

### Context Engine Configuration

```yaml
context:
  engine:
    min_relevance_threshold: 0.3
    max_context_age: "24h"
    cache_size: 10000
    cache_ttl: "1h"
    
  workspace:
    watch_enabled: true
    watch_patterns: ["*.go", "*.yaml", "*.json", "go.mod", "go.sum"]
    ignore_patterns: ["vendor/", ".git/", "*.test"]
    analysis_depth: 3
    dependency_tracking: true
    
  temporal:
    retention_period: "90d"
    resolution: "1m"
    pattern_detection:
      min_support: 0.1
      min_confidence: 0.7
    trend_analysis:
      window_size: "7d"
      sensitivity: 0.8
      
  semantic:
    embedding_model: "sentence-transformers/all-MiniLM-L6-v2"
    embedding_dimensions: 384
    similarity_threshold: 0.7
    graph_max_depth: 3
    inference_rules: "rules/semantic_inference.yaml"
    
  personal:
    privacy_level: "balanced"
    habit_detection:
      min_occurrences: 5
      confidence_threshold: 0.8
    preference_learning:
      update_frequency: "1h"
      decay_factor: 0.95
    anonymization:
      enabled: true
      techniques: ["generalization", "suppression"]
```

## Usage Examples

### Basic Context Retrieval

```go
func ExampleGetContext() {
    engine, err := context.NewEngine(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Get workspace context
    ctx, err := engine.GetContext(context.Background(), context.ContextScope{
        Level:  context.ScopeLevelProject,
        Entity: "assistant-go",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Access context data
    project := ctx.Data["project"].(*Project)
    fmt.Printf("Project: %s\n", project.Name)
    fmt.Printf("Dependencies: %d\n", len(project.Dependencies))
}
```

### Context Subscription

```go
func ExampleSubscription() {
    engine, err := context.NewEngine(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Subscribe to workspace changes
    sub, err := engine.Subscribe(context.ContextScope{
        Level: context.ScopeLevelWorkspace,
    }, func(update context.ContextUpdate) {
        fmt.Printf("Context update: %s\n", update.Type)
        
        switch update.Type {
        case context.UpdateTypeFileChanged:
            fmt.Printf("File changed: %s\n", update.Data["path"])
        case context.UpdateTypePreferenceChanged:
            fmt.Printf("Preference changed: %s\n", update.Data["preference"])
        }
    })
    
    defer sub.Unsubscribe()
}
```

### Intelligent Context Usage

```go
func ProcessWithContext(engine *context.Engine, request Request) (*Response, error) {
    // Enrich request with context
    enriched, err := engine.EnrichRequest(context.Background(), request)
    if err != nil {
        return nil, fmt.Errorf("enriching request: %w", err)
    }
    
    // Use workspace context
    if workspace := enriched.Context[context.ContextTypeWorkspace]; workspace != nil {
        project := workspace.Data["project"].(*Project)
        // Adjust response based on project type
        if project.Type == "go" {
            // Go-specific handling
        }
    }
    
    // Use temporal context
    if temporal := enriched.Context[context.ContextTypeTemporal]; temporal != nil {
        patterns := temporal.Data["patterns"].([]Pattern)
        // Adapt based on detected patterns
        for _, pattern := range patterns {
            if pattern.Type == PatternTypeFrequentError {
                // Proactively address common errors
            }
        }
    }
    
    // Use personal context
    if personal := enriched.Context[context.ContextTypePersonal]; personal != nil {
        style := personal.Data["style"].(DevelopmentStyle)
        // Adjust response style
        response.Style = style.PreferredResponseStyle
    }
    
    return response, nil
}
```

## Performance Optimization

### Context Caching

```go
type ContextCache struct {
    lru      *lru.Cache
    ttlCache *ttlcache.Cache
    bloom    *bloom.BloomFilter
}

func (cc *ContextCache) Get(key string) (*Context, bool) {
    // Check bloom filter first
    if !cc.bloom.Test([]byte(key)) {
        return nil, false
    }
    
    // Check TTL cache
    if item, exists := cc.ttlCache.Get(key); exists {
        return item.(*Context), true
    }
    
    // Check LRU cache
    if value, ok := cc.lru.Get(key); ok {
        return value.(*Context), true
    }
    
    return nil, false
}
```

### Parallel Processing

```go
func (e *Engine) GetMultipleContexts(scopes []ContextScope) ([]*Context, error) {
    results := make([]*Context, len(scopes))
    errors := make([]error, len(scopes))
    
    var wg sync.WaitGroup
    for i, scope := range scopes {
        wg.Add(1)
        go func(idx int, s ContextScope) {
            defer wg.Done()
            ctx, err := e.GetContext(context.Background(), s)
            results[idx] = ctx
            errors[idx] = err
        }(i, scope)
    }
    
    wg.Wait()
    
    // Check for errors
    for i, err := range errors {
        if err != nil {
            return nil, fmt.Errorf("getting context %d: %w", i, err)
        }
    }
    
    return results, nil
}
```

## Related Documentation

- [Agent System](../agents/README.md) - Agents that use context
- [Memory Systems](../memory/README.md) - Memory and context integration
- [AI Providers](../../ai/README.md) - AI model context usage
- [Personal Knowledge](../intelligence/README.md) - Knowledge graph integration