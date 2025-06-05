# Learning System

## Overview

The Learning System is the adaptive intelligence core of the Assistant that enables continuous improvement through experience. It implements machine learning algorithms, pattern recognition, preference learning, and behavioral adaptation to provide increasingly personalized and effective assistance over time.

## Architecture

```
internal/core/learning/
├── learner.go           # Core learning algorithms
├── pattern_detector.go  # Pattern recognition system
├── preference_learner.go # User preference learning
├── model_manager.go     # ML model management
└── training_pipeline.go # Training and evaluation pipelines
```

## Key Features

### 🧠 **Adaptive Learning**
- **Online Learning**: Continuous improvement from interactions
- **Transfer Learning**: Knowledge transfer between domains
- **Meta-Learning**: Learning how to learn better
- **Reinforcement Learning**: Optimization through feedback

### 📊 **Pattern Recognition**
- **Behavioral Patterns**: User interaction patterns
- **Code Patterns**: Development pattern identification
- **Error Patterns**: Common mistake detection
- **Usage Patterns**: Feature usage analytics

### 🎯 **Personalization**
- **Preference Learning**: Individual user preferences
- **Style Adaptation**: Communication style matching
- **Context Learning**: Situational awareness
- **Skill Modeling**: User expertise estimation

## Core Components

### Learning System Interface

```go
type LearningSystem interface {
    // Core learning operations
    LearnFromInteraction(interaction Interaction) error
    LearnFromFeedback(feedback Feedback) error
    
    // Pattern recognition
    DetectPatterns(data []DataPoint) ([]Pattern, error)
    PredictNext(context Context) (Prediction, error)
    
    // Model management
    TrainModel(dataset Dataset) (*Model, error)
    UpdateModel(model *Model, newData []DataPoint) error
    EvaluateModel(model *Model, testData Dataset) (*Evaluation, error)
    
    // Preference learning
    LearnPreferences(userID string, interactions []Interaction) error
    GetPreferences(userID string) (*UserPreferences, error)
}

type Pattern struct {
    ID          string
    Type        PatternType
    Confidence  float64
    Occurrences int
    Description string
    Features    map[string]interface{}
}

type Prediction struct {
    Value       interface{}
    Confidence  float64
    Reasoning   string
    Alternatives []Alternative
}
```

### Online Learning Implementation

```go
type OnlineLearner struct {
    models      map[string]*IncrementalModel
    buffer      *ExperienceBuffer
    optimizer   *AdaptiveOptimizer
    metrics     *LearningMetrics
}

func (ol *OnlineLearner) LearnFromInteraction(interaction Interaction) error {
    // Extract features
    features := ol.extractFeatures(interaction)
    
    // Update relevant models
    for modelID, model := range ol.models {
        if model.IsRelevant(features) {
            // Incremental update
            loss := model.Update(features, interaction.Outcome)
            
            // Track performance
            ol.metrics.RecordUpdate(modelID, loss)
            
            // Adaptive learning rate
            if ol.shouldAdjustLearningRate(modelID) {
                ol.optimizer.AdjustLearningRate(model)
            }
        }
    }
    
    // Store in experience buffer for batch learning
    ol.buffer.Add(interaction)
    
    // Trigger batch learning if buffer is full
    if ol.buffer.IsFull() {
        go ol.batchLearn()
    }
    
    return nil
}

type IncrementalModel struct {
    id           string
    algorithm    Algorithm
    parameters   map[string]float64
    learningRate float64
    momentum     float64
}

func (m *IncrementalModel) Update(features Features, outcome Outcome) float64 {
    // Forward pass
    prediction := m.predict(features)
    
    // Calculate loss
    loss := m.calculateLoss(prediction, outcome)
    
    // Backward pass (gradient descent)
    gradients := m.calculateGradients(features, prediction, outcome)
    
    // Update parameters
    for param, grad := range gradients {
        // Apply momentum
        m.momentum = 0.9 * m.momentum + m.learningRate * grad
        m.parameters[param] -= m.momentum
    }
    
    return loss
}
```

### Pattern Detection System

```go
type PatternDetector struct {
    algorithms  []PatternAlgorithm
    patterns    *PatternStore
    threshold   float64
}

func (pd *PatternDetector) DetectPatterns(data []DataPoint) ([]Pattern, error) {
    var detectedPatterns []Pattern
    
    // Apply multiple algorithms
    for _, algorithm := range pd.algorithms {
        patterns := algorithm.Detect(data)
        
        // Filter by confidence threshold
        for _, pattern := range patterns {
            if pattern.Confidence >= pd.threshold {
                detectedPatterns = append(detectedPatterns, pattern)
            }
        }
    }
    
    // Merge similar patterns
    merged := pd.mergePatterns(detectedPatterns)
    
    // Store significant patterns
    for _, pattern := range merged {
        if pattern.Occurrences >= pd.minOccurrences {
            pd.patterns.Store(pattern)
        }
    }
    
    return merged, nil
}

// Sequential Pattern Mining
type SequentialPatternMiner struct {
    minSupport float64
    maxGap     int
}

func (spm *SequentialPatternMiner) Detect(data []DataPoint) []Pattern {
    // Build sequence database
    sequences := spm.buildSequences(data)
    
    // Find frequent sequences using PrefixSpan
    frequent := spm.prefixSpan(sequences, nil, spm.minSupport)
    
    // Convert to patterns
    var patterns []Pattern
    for _, seq := range frequent {
        pattern := Pattern{
            Type:        PatternTypeSequential,
            Description: fmt.Sprintf("Sequence: %v", seq.Items),
            Confidence:  seq.Support,
            Occurrences: seq.Count,
            Features: map[string]interface{}{
                "sequence": seq.Items,
                "length":   len(seq.Items),
                "support":  seq.Support,
            },
        }
        patterns = append(patterns, pattern)
    }
    
    return patterns
}
```

### Preference Learning

```go
type PreferenceLearner struct {
    userModels  map[string]*UserModel
    clustering  *UserClustering
    recommender *RecommendationEngine
}

type UserModel struct {
    UserID       string
    Preferences  map[string]float64
    Behavior     BehaviorProfile
    Expertise    map[string]float64
    LastUpdated  time.Time
}

func (pl *PreferenceLearner) LearnPreferences(userID string, interactions []Interaction) error {
    model, exists := pl.userModels[userID]
    if !exists {
        model = pl.initializeUserModel(userID)
        pl.userModels[userID] = model
    }
    
    // Extract preference signals
    for _, interaction := range interactions {
        pl.updatePreferences(model, interaction)
        pl.updateBehavior(model, interaction)
        pl.updateExpertise(model, interaction)
    }
    
    // Update user cluster
    cluster := pl.clustering.AssignCluster(model)
    model.ClusterID = cluster.ID
    
    // Learn from similar users
    similarUsers := pl.clustering.GetSimilarUsers(userID, 10)
    pl.transferLearn(model, similarUsers)
    
    model.LastUpdated = time.Now()
    return nil
}

func (pl *PreferenceLearner) updatePreferences(model *UserModel, interaction Interaction) {
    // Extract preference features
    features := pl.extractPreferenceFeatures(interaction)
    
    // Update preference weights using gradient descent
    for feature, value := range features {
        current := model.Preferences[feature]
        
        // Calculate update based on implicit feedback
        feedback := pl.extractImplicitFeedback(interaction)
        update := pl.learningRate * feedback * value
        
        // Apply update with decay
        model.Preferences[feature] = current*pl.decay + update
    }
}
```

### Reinforcement Learning

```go
type ReinforcementLearner struct {
    policy      *Policy
    valueFunc   *ValueFunction
    experience  *ReplayBuffer
    epsilon     float64 // Exploration rate
}

func (rl *ReinforcementLearner) Learn(state State, action Action, reward float64, nextState State) {
    // Store experience
    rl.experience.Add(Experience{
        State:     state,
        Action:    action,
        Reward:    reward,
        NextState: nextState,
    })
    
    // Sample batch for learning
    batch := rl.experience.Sample(rl.batchSize)
    
    // Update value function (Q-learning)
    for _, exp := range batch {
        // Current Q value
        currentQ := rl.valueFunc.Get(exp.State, exp.Action)
        
        // Best next Q value
        nextActions := rl.policy.GetActions(exp.NextState)
        maxNextQ := float64(0)
        for _, nextAction := range nextActions {
            nextQ := rl.valueFunc.Get(exp.NextState, nextAction)
            if nextQ > maxNextQ {
                maxNextQ = nextQ
            }
        }
        
        // TD target
        target := exp.Reward + rl.gamma*maxNextQ
        
        // Update Q value
        loss := target - currentQ
        rl.valueFunc.Update(exp.State, exp.Action, target)
        
        // Update policy
        rl.policy.Update(exp.State, loss)
    }
    
    // Decay exploration
    rl.epsilon = math.Max(rl.minEpsilon, rl.epsilon*rl.epsilonDecay)
}

func (rl *ReinforcementLearner) SelectAction(state State) Action {
    // Epsilon-greedy exploration
    if rand.Float64() < rl.epsilon {
        // Explore: random action
        actions := rl.policy.GetActions(state)
        return actions[rand.Intn(len(actions))]
    }
    
    // Exploit: best action according to policy
    return rl.policy.GetBestAction(state)
}
```

### Model Management

```go
type ModelManager struct {
    models       map[string]*Model
    storage      ModelStorage
    evaluator    *ModelEvaluator
    versioning   *ModelVersioning
}

type Model struct {
    ID           string
    Name         string
    Type         ModelType
    Version      string
    Parameters   interface{}
    Metrics      ModelMetrics
    TrainedAt    time.Time
    LastUsed     time.Time
}

func (mm *ModelManager) TrainModel(config TrainingConfig) (*Model, error) {
    // Prepare dataset
    dataset, err := mm.prepareDataset(config.DataSource)
    if err != nil {
        return nil, fmt.Errorf("preparing dataset: %w", err)
    }
    
    // Split data
    trainData, valData, testData := dataset.Split(0.7, 0.15, 0.15)
    
    // Initialize model
    model := mm.initializeModel(config.ModelType, config.Hyperparameters)
    
    // Training loop
    trainer := NewTrainer(model, config.TrainingParams)
    
    for epoch := 0; epoch < config.Epochs; epoch++ {
        // Train on batches
        trainLoss := trainer.TrainEpoch(trainData)
        
        // Validate
        valLoss := trainer.Validate(valData)
        
        // Early stopping
        if trainer.ShouldStop(valLoss) {
            break
        }
        
        // Log progress
        mm.logProgress(epoch, trainLoss, valLoss)
    }
    
    // Final evaluation
    metrics := mm.evaluator.Evaluate(model, testData)
    model.Metrics = metrics
    
    // Version and store
    version := mm.versioning.CreateVersion(model)
    model.Version = version
    
    if err := mm.storage.Save(model); err != nil {
        return nil, fmt.Errorf("saving model: %w", err)
    }
    
    return model, nil
}
```

### Active Learning

```go
type ActiveLearner struct {
    model          *Model
    strategy       SelectionStrategy
    labelBudget    int
    unlabeledPool  []DataPoint
}

func (al *ActiveLearner) SelectForLabeling() []DataPoint {
    // Score all unlabeled points
    scores := make([]float64, len(al.unlabeledPool))
    
    for i, point := range al.unlabeledPool {
        scores[i] = al.strategy.Score(al.model, point)
    }
    
    // Select top scoring points within budget
    selected := al.selectTopK(al.unlabeledPool, scores, al.labelBudget)
    
    return selected
}

// Uncertainty Sampling Strategy
type UncertaintySampling struct{}

func (us *UncertaintySampling) Score(model *Model, point DataPoint) float64 {
    // Get prediction probabilities
    probs := model.PredictProba(point)
    
    // Calculate entropy
    entropy := float64(0)
    for _, p := range probs {
        if p > 0 {
            entropy -= p * math.Log(p)
        }
    }
    
    return entropy
}
```

## Configuration

### Learning System Configuration

```yaml
learning:
  # Online learning
  online:
    enabled: true
    learning_rate: 0.01
    momentum: 0.9
    decay: 0.95
    buffer_size: 10000
    batch_interval: "5m"
    
  # Pattern detection
  patterns:
    enabled: true
    algorithms:
      - sequential
      - clustering
      - anomaly
    min_confidence: 0.7
    min_occurrences: 3
    
  # Preference learning
  preferences:
    enabled: true
    learning_rate: 0.1
    decay_factor: 0.95
    clustering:
      algorithm: "kmeans"
      num_clusters: 10
      
  # Reinforcement learning
  reinforcement:
    enabled: true
    algorithm: "q_learning"
    epsilon: 0.1
    epsilon_decay: 0.995
    min_epsilon: 0.01
    gamma: 0.95
    batch_size: 32
    
  # Model management
  models:
    storage_path: "/var/lib/assistant/models"
    max_versions: 5
    auto_cleanup: true
    evaluation_metrics:
      - accuracy
      - precision
      - recall
      - f1_score
```

## Usage Examples

### Basic Learning

```go
func ExampleBasicLearning() {
    // Create learning system
    learner := learning.NewLearningSystem(config)
    
    // Learn from interaction
    interaction := learning.Interaction{
        UserID:    "user-123",
        Action:    "code_completion",
        Context:   map[string]interface{}{"language": "go"},
        Outcome:   learning.OutcomeAccepted,
        Timestamp: time.Now(),
    }
    
    if err := learner.LearnFromInteraction(interaction); err != nil {
        log.Printf("Learning failed: %v", err)
    }
    
    // Make prediction
    prediction, err := learner.PredictNext(learning.Context{
        UserID: "user-123",
        State:  map[string]interface{}{"language": "go"},
    })
    
    if err == nil {
        fmt.Printf("Predicted action: %v (confidence: %.2f)\n", 
            prediction.Value, prediction.Confidence)
    }
}
```

### Pattern Detection

```go
func ExamplePatternDetection() {
    detector := learning.NewPatternDetector(learning.PatternConfig{
        MinSupport:   0.1,
        MinConfidence: 0.7,
    })
    
    // Collect user interactions
    var interactions []learning.DataPoint
    // ... populate interactions ...
    
    // Detect patterns
    patterns, err := detector.DetectPatterns(interactions)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, pattern := range patterns {
        fmt.Printf("Pattern: %s (confidence: %.2f, occurrences: %d)\n",
            pattern.Description, pattern.Confidence, pattern.Occurrences)
    }
}
```

## Integration with Other Systems

### Memory Integration

```go
func (ls *LearningSystem) IntegrateWithMemory(memory *memory.System) {
    // Learn from episodic memories
    memory.Subscribe(memory.EventTypeEpisodeStored, func(event memory.Event) {
        episode := event.Data.(memory.Episode)
        ls.LearnFromEpisode(episode)
    })
    
    // Update semantic memory with learned concepts
    ls.OnPatternDetected(func(pattern Pattern) {
        concept := memory.Concept{
            Name:        pattern.Description,
            Type:        "learned_pattern",
            Confidence:  pattern.Confidence,
            Metadata:    pattern.Features,
        }
        memory.Semantic.StoreConcept(concept)
    })
}
```

## Related Documentation

- [Memory Systems](../memory/README.md) - Memory integration
- [Intelligence System](../intelligence/README.md) - Reasoning and decision making
- [Agent System](../agents/README.md) - Agent learning
- [Context Engine](../context/README.md) - Contextual learning