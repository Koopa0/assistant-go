# Tools 工具系統

這個包實現了 Assistant 的智能工具生態系統，提供可組合、自適應的工具，能夠理解意圖並智能協作。

## 架構概述

工具系統採用語義框架設計，每個工具都具備智能理解和協作能力：

```
tools/
├── base.go           # 工具基礎介面和類型
├── registry.go       # 工具註冊和管理
├── pipeline.go       # 工具管道執行
├── godev/           # Go 開發工具集
├── docker/          # Docker 管理工具
├── k8s/             # Kubernetes 工具
├── postgres/        # PostgreSQL 工具
├── cloudflare/      # Cloudflare 集成工具
└── search/          # 搜索和檢索工具
```

## 設計理念

### 🧠 語義理解
工具不僅僅執行命令，更能理解用戶意圖：
- **意圖分析**: 理解用戶想要達成的目標
- **上下文感知**: 基於當前項目狀態調整行為
- **智能建議**: 主動提供相關工具和操作建議

### 🤝 協作能力
工具間能夠智能協作形成工作流：
- **能力聲明**: 每個工具明確聲明其功能和限制
- **協作協議**: 標準化的工具間通信介面
- **動態組合**: 根據任務需求動態組合工具鏈

### 📈 自適應學習
工具系統能夠學習和改進：
- **使用模式學習**: 記住用戶的工作流偏好
- **成功率追踪**: 監控工具執行成功率和用戶滿意度
- **自動優化**: 基於使用數據自動調整參數

## 核心介面

### Tool 介面

```go
// Tool 定義了工具的基本行為
type Tool interface {
    // 傳統執行方法
    Execute(ctx context.Context, input ToolInput) (*ToolResult, error)
    
    // 語義理解方法
    UnderstandCapabilities() []Capability
    EstimateRelevance(intent Intent) float64
    SuggestCollaborations(intent Intent) []ToolCollaboration
    
    // 學習和適應方法
    LearnFromUsage(usage Usage) error
    AdaptToContext(ctx ToolContext) error
    
    // 元數據
    Name() string
    Description() string
    Version() string
}
```

### 語義工具基礎

```go
// SemanticTool 提供語義理解的基礎實現
type SemanticTool struct {
    name         string
    capabilities []Capability
    confidence   ConfidenceEstimator
    memory      *ToolMemory
    
    // 協作能力
    collaborators map[string]CollaborationWeight
    
    // 學習系統
    usageHistory []Usage
    successRate  float64
}

func (t *SemanticTool) EstimateRelevance(intent Intent) float64 {
    // 分析意圖與工具能力的匹配度
    relevance := 0.0
    
    for _, capability := range t.capabilities {
        if capability.Matches(intent) {
            relevance += capability.Weight
        }
    }
    
    // 基於歷史成功率調整
    relevance *= t.successRate
    
    return math.Min(relevance, 1.0)
}
```

## 工具註冊系統

### Registry 管理器

```go
// Registry 管理所有可用工具
type Registry struct {
    tools       map[string]ToolFactory
    instances   map[string]Tool
    usage       map[string]*UsageStats
    
    // 智能特性
    intentMatcher *IntentMatcher
    collaborationGraph *CollaborationGraph
    learningEngine *LearningEngine
}

// 註冊工具工廠
func (r *Registry) Register(name string, factory ToolFactory) error {
    if r.tools[name] != nil {
        return fmt.Errorf("tool %s already registered", name)
    }
    
    r.tools[name] = factory
    r.logger.Debug("Tool factory registered", slog.String("tool", name))
    
    return nil
}

// 智能工具選擇
func (r *Registry) SelectTools(intent Intent) ([]Tool, error) {
    candidates := make([]ToolCandidate, 0)
    
    // 評估所有工具的相關性
    for name, tool := range r.instances {
        relevance := tool.EstimateRelevance(intent)
        if relevance > 0.1 { // 相關性閾值
            candidates = append(candidates, ToolCandidate{
                Tool:      tool,
                Relevance: relevance,
            })
        }
    }
    
    // 根據相關性排序
    sort.Slice(candidates, func(i, j int) bool {
        return candidates[i].Relevance > candidates[j].Relevance
    })
    
    // 選擇最相關的工具
    selected := make([]Tool, 0)
    for _, candidate := range candidates {
        if len(selected) >= intent.MaxTools {
            break
        }
        selected = append(selected, candidate.Tool)
    }
    
    return selected, nil
}
```

## Go 開發工具集

### GoAnalyzer - 代碼分析工具

```go
// GoAnalyzer 分析 Go 代碼並提供智能建議
type GoAnalyzer struct {
    *SemanticTool
    
    // AST 分析能力
    parser     *GoParser
    inspector  *ast.Inspector
    
    // 智能分析
    patternDetector *PatternDetector
    qualityChecker  *QualityChecker
}

func (g *GoAnalyzer) Execute(ctx context.Context, input ToolInput) (*ToolResult, error) {
    analysisInput, ok := input.(*GoAnalysisInput)
    if !ok {
        return nil, fmt.Errorf("invalid input type for GoAnalyzer")
    }
    
    // 解析代碼
    fileSet := token.NewFileSet()
    node, err := parser.ParseFile(fileSet, analysisInput.Filename, 
                                 analysisInput.Source, parser.ParseComments)
    if err != nil {
        return nil, fmt.Errorf("failed to parse Go file: %w", err)
    }
    
    // 智能分析
    result := &GoAnalysisResult{
        Issues:        g.findIssues(node),
        Suggestions:   g.generateSuggestions(node),
        Complexity:    g.calculateComplexity(node),
        TestCoverage:  g.estimateTestCoverage(node),
        Patterns:      g.detectPatterns(node),
    }
    
    return &ToolResult{
        Success:     true,
        Data:        result,
        Confidence:  g.calculateConfidence(result),
        Suggestions: g.generateNextSteps(result),
    }, nil
}

// 智能建議生成
func (g *GoAnalyzer) generateSuggestions(node ast.Node) []Suggestion {
    suggestions := make([]Suggestion, 0)
    
    // 檢查常見模式和最佳實踐
    ast.Inspect(node, func(n ast.Node) bool {
        switch x := n.(type) {
        case *ast.FuncDecl:
            // 檢查函數複雜度
            if g.calculateFuncComplexity(x) > 10 {
                suggestions = append(suggestions, Suggestion{
                    Type:        "refactoring",
                    Severity:    "medium",
                    Message:     "Function complexity is high, consider breaking it down",
                    Line:        g.getLineNumber(x.Pos()),
                    Confidence:  0.8,
                })
            }
            
            // 檢查錯誤處理
            if !g.hasProperErrorHandling(x) {
                suggestions = append(suggestions, Suggestion{
                    Type:        "error_handling",
                    Severity:    "high",
                    Message:     "Add proper error handling with context wrapping",
                    Line:        g.getLineNumber(x.Pos()),
                    Confidence:  0.9,
                })
            }
            
        case *ast.StructType:
            // 檢查結構體設計
            if g.hasPublicFieldsWithoutMethods(x) {
                suggestions = append(suggestions, Suggestion{
                    Type:        "design",
                    Severity:    "low",
                    Message:     "Consider adding methods to encapsulate struct behavior",
                    Line:        g.getLineNumber(x.Pos()),
                    Confidence:  0.6,
                })
            }
        }
        return true
    })
    
    return suggestions
}
```

### GoFormatter - 智能代碼格式化

```go
// GoFormatter 提供智能代碼格式化
type GoFormatter struct {
    *SemanticTool
    
    // 格式化配置
    style       *FormattingStyle
    preferences *UserPreferences
}

func (g *GoFormatter) Execute(ctx context.Context, input ToolInput) (*ToolResult, error) {
    formatInput, ok := input.(*GoFormatInput)
    if !ok {
        return nil, fmt.Errorf("invalid input type for GoFormatter")
    }
    
    // 智能格式化決策
    style := g.determineStyle(formatInput.Source, formatInput.Context)
    
    // 應用格式化
    formatted, err := g.formatWithStyle(formatInput.Source, style)
    if err != nil {
        return nil, fmt.Errorf("formatting failed: %w", err)
    }
    
    // 生成格式化報告
    changes := g.detectChanges(formatInput.Source, formatted)
    
    return &ToolResult{
        Success: true,
        Data: &GoFormatResult{
            FormattedCode: formatted,
            Changes:       changes,
            Style:         style,
        },
        Confidence: 1.0, // 格式化總是確定的
    }, nil
}

// 智能樣式決策
func (g *GoFormatter) determineStyle(source string, context *ProjectContext) *FormattingStyle {
    // 分析現有代碼風格
    existingStyle := g.analyzeExistingStyle(source)
    
    // 考慮項目偏好
    if context != nil && context.FormattingRules != nil {
        existingStyle = g.mergeWithProjectRules(existingStyle, context.FormattingRules)
    }
    
    // 應用用戶偏好
    if g.preferences != nil {
        existingStyle = g.applyUserPreferences(existingStyle, g.preferences)
    }
    
    return existingStyle
}
```

## 工具協作示例

### 智能代碼重構流程

```go
// RefactoringPipeline 展示工具協作進行代碼重構
func (r *Registry) RefactoringPipeline(ctx context.Context, intent *RefactoringIntent) error {
    // 1. 分析代碼
    analyzer, err := r.GetTool("go_analyzer")
    if err != nil {
        return fmt.Errorf("analyzer not available: %w", err)
    }
    
    analysisResult, err := analyzer.Execute(ctx, &GoAnalysisInput{
        Filename: intent.Filename,
        Source:   intent.Source,
    })
    if err != nil {
        return fmt.Errorf("analysis failed: %w", err)
    }
    
    // 2. 基於分析結果決定重構策略
    refactoringPlan := r.createRefactoringPlan(analysisResult)
    
    // 3. 執行重構
    for _, step := range refactoringPlan.Steps {
        tool, err := r.GetTool(step.ToolName)
        if err != nil {
            return fmt.Errorf("tool %s not available: %w", step.ToolName, err)
        }
        
        result, err := tool.Execute(ctx, step.Input)
        if err != nil {
            return fmt.Errorf("refactoring step %s failed: %w", step.Name, err)
        }
        
        // 更新意圖狀態
        intent.Source = result.Data.(string)
    }
    
    // 4. 運行測試驗證
    tester, err := r.GetTool("go_tester")
    if err != nil {
        return fmt.Errorf("tester not available: %w", err)
    }
    
    testResult, err := tester.Execute(ctx, &GoTestInput{
        ProjectPath: intent.ProjectPath,
        TestPattern: "./...",
    })
    if err != nil {
        return fmt.Errorf("testing failed: %w", err)
    }
    
    // 5. 格式化最終代碼
    formatter, err := r.GetTool("go_formatter")
    if err != nil {
        return fmt.Errorf("formatter not available: %w", err)
    }
    
    _, err = formatter.Execute(ctx, &GoFormatInput{
        Source: intent.Source,
    })
    if err != nil {
        return fmt.Errorf("formatting failed: %w", err)
    }
    
    return nil
}
```

## 工具學習系統

### 使用模式學習

```go
// LearningEngine 管理工具使用學習
type LearningEngine struct {
    patterns map[string]*UsagePattern
    outcomes map[string]*OutcomeHistory
    
    // 機器學習模型（簡化版）
    predictor *SimplePredictor
}

func (le *LearningEngine) LearnFromExecution(execution *ToolExecution) {
    // 記錄使用模式
    pattern := &UsagePattern{
        ToolName:    execution.ToolName,
        Context:     execution.Context,
        Input:       execution.Input,
        UserIntent:  execution.UserIntent,
        Timestamp:   execution.Timestamp,
    }
    
    le.patterns[execution.ID] = pattern
    
    // 記錄結果
    outcome := &ExecutionOutcome{
        Success:        execution.Success,
        UserSatisfaction: execution.UserSatisfaction,
        ExecutionTime:   execution.ExecutionTime,
        ErrorType:      execution.ErrorType,
    }
    
    if le.outcomes[execution.ToolName] == nil {
        le.outcomes[execution.ToolName] = &OutcomeHistory{}
    }
    le.outcomes[execution.ToolName].Add(outcome)
    
    // 更新成功率
    le.updateSuccessRate(execution.ToolName)
}

func (le *LearningEngine) PredictBestTool(intent Intent) (string, float64) {
    // 簡化的預測邏輯
    bestTool := ""
    bestScore := 0.0
    
    for toolName, history := range le.outcomes {
        // 計算基於歷史的得分
        score := history.SuccessRate * 0.7 + history.UserSatisfaction * 0.3
        
        // 考慮上下文相似性
        contextSimilarity := le.calculateContextSimilarity(intent, toolName)
        score *= contextSimilarity
        
        if score > bestScore {
            bestScore = score
            bestTool = toolName
        }
    }
    
    return bestTool, bestScore
}
```

## 配置和定制

### 工具配置

```yaml
tools:
  registry:
    auto_load: true
    discovery_paths:
      - "./tools/custom"
      - "./tools/plugins"
  
  go_analyzer:
    enabled: true
    complexity_threshold: 10
    coverage_threshold: 0.8
    check_patterns:
      - "error_handling"
      - "naming_conventions" 
      - "package_organization"
  
  go_formatter:
    enabled: true
    style: "gofmt"
    line_length: 120
    tab_width: 4
  
  learning:
    enabled: true
    retention_days: 90
    min_executions_for_prediction: 10
```

### 自定義工具開發

```go
// CustomTool 展示如何開發自定義工具
type CustomTool struct {
    *SemanticTool
    
    // 自定義配置
    config *CustomConfig
}

func NewCustomTool(config map[string]any, logger *slog.Logger) (Tool, error) {
    customConfig, err := parseCustomConfig(config)
    if err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    
    tool := &CustomTool{
        SemanticTool: NewSemanticTool("custom_tool", logger),
        config:       customConfig,
    }
    
    // 聲明工具能力
    tool.capabilities = []Capability{
        {Name: "custom_analysis", Weight: 1.0},
        {Name: "data_processing", Weight: 0.8},
    }
    
    return tool, nil
}

func (c *CustomTool) Execute(ctx context.Context, input ToolInput) (*ToolResult, error) {
    // 實現自定義邏輯
    return &ToolResult{
        Success: true,
        Data:    "Custom tool result",
    }, nil
}

// 註冊自定義工具
func init() {
    registry.Register("custom_tool", NewCustomTool)
}
```

## 性能優化

### 工具執行優化

```go
// 並行工具執行
func (r *Registry) ExecuteParallel(ctx context.Context, tools []Tool, input ToolInput) ([]*ToolResult, error) {
    results := make([]*ToolResult, len(tools))
    errors := make([]error, len(tools))
    
    var wg sync.WaitGroup
    for i, tool := range tools {
        wg.Add(1)
        go func(index int, t Tool) {
            defer wg.Done()
            
            result, err := t.Execute(ctx, input)
            results[index] = result
            errors[index] = err
        }(i, tool)
    }
    
    wg.Wait()
    
    // 檢查錯誤
    for i, err := range errors {
        if err != nil {
            return nil, fmt.Errorf("tool %s failed: %w", tools[i].Name(), err)
        }
    }
    
    return results, nil
}

// 工具實例池
type ToolPool struct {
    factory ToolFactory
    pool    chan Tool
    created int32
    maxSize int32
}

func (tp *ToolPool) Get() (Tool, error) {
    select {
    case tool := <-tp.pool:
        return tool, nil
    default:
        if atomic.LoadInt32(&tp.created) < tp.maxSize {
            atomic.AddInt32(&tp.created, 1)
            return tp.factory(nil, slog.Default())
        }
        // 等待可用實例
        return <-tp.pool, nil
    }
}

func (tp *ToolPool) Put(tool Tool) {
    select {
    case tp.pool <- tool:
    default:
        // 池已滿，丟棄實例
        atomic.AddInt32(&tp.created, -1)
    }
}
```

## 監控和調試

### 工具執行監控

```go
// ToolMonitor 監控工具執行
type ToolMonitor struct {
    metrics   map[string]*ToolMetrics
    listeners []ExecutionListener
}

type ToolMetrics struct {
    ExecutionCount   int64
    SuccessCount     int64
    TotalDuration    time.Duration
    AverageDuration  time.Duration
    LastExecuted     time.Time
    ErrorRate        float64
}

func (tm *ToolMonitor) RecordExecution(execution *ToolExecution) {
    metrics := tm.getOrCreateMetrics(execution.ToolName)
    
    metrics.ExecutionCount++
    if execution.Success {
        metrics.SuccessCount++
    }
    
    metrics.TotalDuration += execution.Duration
    metrics.AverageDuration = time.Duration(int64(metrics.TotalDuration) / metrics.ExecutionCount)
    metrics.LastExecuted = execution.Timestamp
    metrics.ErrorRate = 1.0 - float64(metrics.SuccessCount)/float64(metrics.ExecutionCount)
    
    // 通知監聽器
    for _, listener := range tm.listeners {
        listener.OnExecution(execution)
    }
}
```

## 最佳實踐

### 1. 工具設計原則

- **單一職責**: 每個工具專注一個特定功能
- **可組合性**: 工具應該能夠組合形成複雜工作流
- **幂等性**: 重複執行相同輸入應產生相同結果
- **錯誤透明**: 清晰的錯誤訊息和恢復建議

### 2. 性能考慮

- **惰性加載**: 工具實例按需創建
- **資源池化**: 重用昂貴的工具實例
- **並行執行**: 無依賴的工具可並行運行
- **緩存結果**: 緩存重複計算的結果

### 3. 用戶體驗

- **智能建議**: 基於上下文主動建議相關工具
- **進度反饋**: 長時間運行的工具提供進度訊息
- **錯誤指導**: 失敗時提供具體的修復建議
- **學習適應**: 根據用戶使用習慣調整工具行為

---

*工具系統是 Assistant 的核心組件，提供了智能、協作、自適應的工具生態系統。通過語義理解和機器學習，工具不僅能執行任務，更能理解用戶意圖並提供智能建議。*