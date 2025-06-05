# Assistant 智能助手核心

這個包實現了 Assistant 的核心功能，提供智能開發伴侶的主要交互介面和處理邏輯。

## 架構概述

Assistant 核心採用分層架構，結合智能代理和語義理解：

```
assistant/
├── assistant.go          # 主要助手介面
├── processor.go          # 查詢處理器
├── context.go            # 上下文管理
├── errors.go             # 錯誤定義和處理
├── assistant_test.go     # 單元測試
└── *_fuzzing_test.go     # 模糊測試（Go 1.18+）
```

## 設計理念

### 🧠 智能理解
Assistant 不僅執行命令，更能理解開發意圖：
- **語義分析**: 理解查詢背後的真實需求
- **上下文感知**: 基於當前項目狀態調整回應
- **學習適應**: 從用戶交互中學習偏好和模式

### 🤝 協作智能
多個智能代理協作提供全面支援：
- **開發代理**: 程式碼理解、重構建議、最佳實踐
- **資料庫代理**: 查詢優化、架構分析、性能建議
- **基礎設施代理**: 部署管理、資源優化、故障診斷

### 📚 持續學習
系統隨使用而進化：
- **模式識別**: 自動識別開發模式和工作流
- **偏好學習**: 記住用戶的工具和方法偏好
- **知識累積**: 建立個人化的開發知識庫

## 核心介面

### Assistant 主介面

```go
// Assistant 提供智能開發伴侶的主要功能
type Assistant interface {
    // 處理用戶查詢並返回智能回應
    Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error)
    
    // 獲取對話歷史和上下文
    GetConversation(ctx context.Context, id string) (*Conversation, error)
    
    // 創建新的對話會話
    CreateConversation(ctx context.Context, userID string, title string) (*Conversation, error)
    
    // 獲取可用工具列表
    GetAvailableTools(ctx context.Context) ([]ToolInfo, error)
    
    // 執行特定工具
    ExecuteTool(ctx context.Context, toolName string, input ToolInput) (*ToolResult, error)
    
    // 獲取學習統計
    GetLearningStats(ctx context.Context, userID string) (*LearningStats, error)
    
    // 關閉助手並清理資源
    Close(ctx context.Context) error
}
```

### 查詢請求和回應

```go
// QueryRequest 表示用戶查詢請求
type QueryRequest struct {
    Query          string    `json:"query" validate:"required,min=1"`
    ConversationID *string   `json:"conversation_id,omitempty"`
    UserID         string    `json:"user_id" validate:"required"`
    Context        *Context  `json:"context,omitempty"`
    Preferences    *UserPreferences `json:"preferences,omitempty"`
}

// QueryResponse 表示助手的智能回應
type QueryResponse struct {
    ID             string           `json:"id"`
    Response       string           `json:"response"`
    ConversationID string           `json:"conversation_id"`
    ToolsUsed      []ToolExecution  `json:"tools_used,omitempty"`
    Suggestions    []Suggestion     `json:"suggestions,omitempty"`
    Confidence     float64          `json:"confidence"`
    ProcessingTime time.Duration    `json:"processing_time"`
    LearningData   *LearningData    `json:"learning_data,omitempty"`
}
```

## 處理器架構

### QueryProcessor 智能處理

```go
// QueryProcessor 處理查詢並協調各種智能代理
type QueryProcessor struct {
    // AI 提供者
    aiProvider    ai.Provider
    
    // 智能代理管理
    agentManager  *agents.Manager
    
    // 工具註冊表
    toolRegistry  *tools.Registry
    
    // 記憶系統
    memorySystem  *memory.System
    
    // 學習引擎
    learningEngine *learning.Engine
    
    // 上下文引擎
    contextEngine *context.Engine
}

func (p *QueryProcessor) ProcessQuery(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
    // 1. 豐富上下文訊息
    enrichedContext := p.contextEngine.EnrichContext(req.Context)
    
    // 2. 分析查詢意圖
    intent, err := p.analyzeIntent(req.Query, enrichedContext)
    if err != nil {
        return nil, fmt.Errorf("intent analysis failed: %w", err)
    }
    
    // 3. 選擇相關代理和工具
    agents := p.agentManager.SelectAgents(intent)
    tools := p.toolRegistry.SelectTools(intent)
    
    // 4. 執行協作處理
    result, err := p.executeCollaborative(ctx, intent, agents, tools)
    if err != nil {
        return nil, fmt.Errorf("collaborative execution failed: %w", err)
    }
    
    // 5. 學習和記憶存儲
    p.learningEngine.LearnFromInteraction(req, result)
    p.memorySystem.StoreEpisode(req, result)
    
    return result, nil
}
```

### 意圖分析

```go
// IntentAnalyzer 分析用戶查詢的真實意圖
type IntentAnalyzer struct {
    nlpModel     *nlp.Model
    patternMatcher *patterns.Matcher
    contextAnalyzer *context.Analyzer
}

func (ia *IntentAnalyzer) AnalyzeIntent(query string, context *Context) (*Intent, error) {
    // 自然語言處理
    nlpResult := ia.nlpModel.Process(query)
    
    // 模式匹配
    patterns := ia.patternMatcher.FindPatterns(query)
    
    // 上下文分析
    contextClues := ia.contextAnalyzer.ExtractClues(context)
    
    // 綜合分析
    intent := &Intent{
        Primary:     ia.determinePrimaryIntent(nlpResult, patterns),
        Secondary:   ia.determineSecondaryIntents(nlpResult, patterns),
        Confidence:  ia.calculateConfidence(nlpResult, patterns, contextClues),
        Context:     contextClues,
        Entities:    nlpResult.Entities,
        Keywords:    nlpResult.Keywords,
    }
    
    return intent, nil
}
```

## 上下文管理

### Context 智能上下文

```go
// Context 表示當前開發環境的智能上下文
type Context struct {
    // 工作空間訊息
    Workspace *WorkspaceContext `json:"workspace,omitempty"`
    
    // 當前項目狀態
    Project *ProjectContext `json:"project,omitempty"`
    
    // 用戶狀態
    User *UserContext `json:"user,omitempty"`
    
    // 環境訊息
    Environment *EnvironmentContext `json:"environment,omitempty"`
    
    // 時間上下文
    Temporal *TemporalContext `json:"temporal,omitempty"`
}

// WorkspaceContext 工作空間上下文
type WorkspaceContext struct {
    RootPath        string              `json:"root_path"`
    OpenFiles       []string            `json:"open_files"`
    ModifiedFiles   []string            `json:"modified_files"`
    GitStatus       *GitStatus          `json:"git_status,omitempty"`
    BuildStatus     *BuildStatus        `json:"build_status,omitempty"`
    TestResults     *TestResults        `json:"test_results,omitempty"`
    Dependencies    []Dependency        `json:"dependencies"`
    Configuration   map[string]any      `json:"configuration"`
}

// ProjectContext 項目上下文
type ProjectContext struct {
    Name            string              `json:"name"`
    Type            string              `json:"type"`
    Language        string              `json:"language"`
    Framework       string              `json:"framework,omitempty"`
    Architecture    *ArchitectureInfo   `json:"architecture,omitempty"`
    Patterns        []PatternUsage      `json:"patterns"`
    Conventions     *CodingConventions  `json:"conventions,omitempty"`
}
```

### 上下文引擎

```go
// ContextEngine 管理和豐富上下文訊息
type ContextEngine struct {
    workspaceAnalyzer *workspace.Analyzer
    projectAnalyzer   *project.Analyzer
    gitAnalyzer       *git.Analyzer
    environmentDetector *env.Detector
}

func (ce *ContextEngine) EnrichContext(baseContext *Context) *Context {
    if baseContext == nil {
        baseContext = &Context{}
    }
    
    // 豐富工作空間訊息
    if baseContext.Workspace == nil {
        baseContext.Workspace = ce.workspaceAnalyzer.AnalyzeWorkspace()
    }
    
    // 豐富項目訊息
    if baseContext.Project == nil {
        baseContext.Project = ce.projectAnalyzer.AnalyzeProject(baseContext.Workspace)
    }
    
    // 豐富 Git 訊息
    if baseContext.Workspace.GitStatus == nil {
        baseContext.Workspace.GitStatus = ce.gitAnalyzer.GetStatus()
    }
    
    // 豐富環境訊息
    if baseContext.Environment == nil {
        baseContext.Environment = ce.environmentDetector.Detect()
    }
    
    return baseContext
}
```

## 錯誤處理

### 智能錯誤系統

```go
// AssistantError 智能錯誤類型
type AssistantError struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Cause   error                  `json:"-"`
    Context map[string]interface{} `json:"context,omitempty"`
    
    // 智能功能
    Suggestions []Suggestion       `json:"suggestions,omitempty"`
    Recovery    *RecoveryPlan      `json:"recovery,omitempty"`
    Learning    *LearningOpportunity `json:"learning,omitempty"`
}

// Suggestion 智能建議
type Suggestion struct {
    Type        string  `json:"type"`
    Title       string  `json:"title"`
    Description string  `json:"description"`
    Action      string  `json:"action,omitempty"`
    Confidence  float64 `json:"confidence"`
}

// RecoveryPlan 恢復計劃
type RecoveryPlan struct {
    Steps       []RecoveryStep `json:"steps"`
    Automated   bool          `json:"automated"`
    Confidence  float64       `json:"confidence"`
}
```

### 錯誤處理最佳實踐

```go
// 智能錯誤處理示例
func (p *QueryProcessor) handleError(err error, context *Context) *AssistantError {
    assistantErr := &AssistantError{
        Code:    determineErrorCode(err),
        Message: err.Error(),
        Cause:   err,
        Context: make(map[string]interface{}),
    }
    
    // 基於上下文生成建議
    assistantErr.Suggestions = p.generateSuggestions(err, context)
    
    // 生成恢復計劃
    assistantErr.Recovery = p.generateRecoveryPlan(err, context)
    
    // 識別學習機會
    assistantErr.Learning = p.identifyLearningOpportunity(err, context)
    
    return assistantErr
}

func (p *QueryProcessor) generateSuggestions(err error, context *Context) []Suggestion {
    suggestions := make([]Suggestion, 0)
    
    // 基於錯誤類型生成建議
    switch {
    case errors.Is(err, ErrToolNotFound):
        suggestions = append(suggestions, Suggestion{
            Type:        "tool_install",
            Title:       "安裝缺失的工具",
            Description: "所需的工具似乎未安裝或不可用",
            Action:      "make setup",
            Confidence:  0.8,
        })
    case errors.Is(err, ErrInvalidInput):
        suggestions = append(suggestions, Suggestion{
            Type:        "input_validation",
            Title:       "檢查輸入格式",
            Description: "輸入格式可能不正確，請檢查文檔",
            Confidence:  0.9,
        })
    }
    
    return suggestions
}
```

## 學習系統集成

### 學習數據收集

```go
// LearningData 學習數據結構
type LearningData struct {
    UserID         string                 `json:"user_id"`
    SessionID      string                 `json:"session_id"`
    Query          string                 `json:"query"`
    Intent         *Intent                `json:"intent"`
    Response       string                 `json:"response"`
    ToolsUsed      []ToolExecution        `json:"tools_used"`
    Success        bool                   `json:"success"`
    UserFeedback   *UserFeedback          `json:"user_feedback,omitempty"`
    Context        *Context               `json:"context"`
    ProcessingTime time.Duration          `json:"processing_time"`
    Confidence     float64                `json:"confidence"`
}

// UserFeedback 用戶反饋
type UserFeedback struct {
    Rating      int    `json:"rating"`      // 1-5 星評分
    Helpful     bool   `json:"helpful"`     // 是否有幫助
    Accurate    bool   `json:"accurate"`    // 是否準確
    Comments    string `json:"comments,omitempty"`
    Improvements []string `json:"improvements,omitempty"`
}

func (a *assistant) collectLearningData(req *QueryRequest, resp *QueryResponse, feedback *UserFeedback) {
    learningData := &LearningData{
        UserID:         req.UserID,
        SessionID:      req.ConversationID,
        Query:          req.Query,
        Response:       resp.Response,
        ToolsUsed:      resp.ToolsUsed,
        Success:        resp.Confidence > 0.7,
        UserFeedback:   feedback,
        Context:        req.Context,
        ProcessingTime: resp.ProcessingTime,
        Confidence:     resp.Confidence,
    }
    
    // 異步存儲學習數據
    go a.learningEngine.Store(learningData)
}
```

## 性能優化

### 並發處理

```go
// ConcurrentProcessor 並發查詢處理器
type ConcurrentProcessor struct {
    *QueryProcessor
    
    // 並發控制
    semaphore    chan struct{}
    workerPool   *WorkerPool
    requestQueue chan *ProcessingRequest
}

type ProcessingRequest struct {
    Request  *QueryRequest
    Response chan *ProcessingResult
    Context  context.Context
}

type ProcessingResult struct {
    Response *QueryResponse
    Error    error
}

func (cp *ConcurrentProcessor) ProcessQueryConcurrent(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
    // 創建處理請求
    processingReq := &ProcessingRequest{
        Request:  req,
        Response: make(chan *ProcessingResult, 1),
        Context:  ctx,
    }
    
    // 提交到隊列
    select {
    case cp.requestQueue <- processingReq:
    case <-ctx.Done():
        return nil, ctx.Err()
    }
    
    // 等待結果
    select {
    case result := <-processingReq.Response:
        return result.Response, result.Error
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

### 緩存策略

```go
// CacheManager 智能緩存管理器
type CacheManager struct {
    queryCache    *cache.LRU[string, *QueryResponse]
    contextCache  *cache.LRU[string, *Context]
    intentCache   *cache.LRU[string, *Intent]
    
    // 緩存統計
    stats *CacheStats
}

func (cm *CacheManager) GetCachedResponse(req *QueryRequest) (*QueryResponse, bool) {
    // 生成緩存鍵
    key := cm.generateCacheKey(req)
    
    // 檢查緩存
    if response, exists := cm.queryCache.Get(key); exists {
        // 檢查緩存是否仍然有效
        if cm.isCacheValid(response, req.Context) {
            cm.stats.RecordHit()
            return response, true
        }
        // 緩存失效，移除
        cm.queryCache.Remove(key)
    }
    
    cm.stats.RecordMiss()
    return nil, false
}

func (cm *CacheManager) CacheResponse(req *QueryRequest, resp *QueryResponse) {
    key := cm.generateCacheKey(req)
    
    // 只緩存高信心度的回應
    if resp.Confidence >= 0.8 {
        cm.queryCache.Set(key, resp)
    }
}
```

## 配置和定制

### Assistant 配置

```yaml
assistant:
  # 核心設置
  name: "Assistant"
  version: "1.0.0"
  
  # 處理器設置
  processor:
    max_concurrent_queries: 10
    query_timeout: 30s
    context_enrichment: true
    learning_enabled: true
  
  # 緩存設置
  cache:
    query_cache_size: 1000
    context_cache_size: 500
    intent_cache_size: 200
    cache_ttl: 1h
  
  # 學習設置
  learning:
    enabled: true
    retention_days: 90
    feedback_required: false
    pattern_detection: true
  
  # 智能代理設置
  agents:
    development:
      enabled: true
      confidence_threshold: 0.7
    database:
      enabled: true
      confidence_threshold: 0.8
    infrastructure:
      enabled: true
      confidence_threshold: 0.6
```

## 最佳實踐

### 1. 查詢設計

```go
// 良好的查詢請求設計
req := &QueryRequest{
    Query:   "Analyze the performance of this SQL query",
    UserID:  userID,
    Context: &Context{
        Workspace: currentWorkspace,
        Project:   currentProject,
    },
    Preferences: &UserPreferences{
        Verbosity:    "detailed",
        Language:     "en",
        ExpertMode:   true,
    },
}

// 處理回應
resp, err := assistant.Query(ctx, req)
if err != nil {
    // 檢查是否為 AssistantError
    if assistantErr := GetAssistantError(err); assistantErr != nil {
        // 顯示建議
        for _, suggestion := range assistantErr.Suggestions {
            fmt.Printf("建議: %s\n", suggestion.Description)
        }
    }
    return fmt.Errorf("query failed: %w", err)
}

// 處理建議
for _, suggestion := range resp.Suggestions {
    fmt.Printf("建議: %s (信心度: %.2f)\n", suggestion.Description, suggestion.Confidence)
}
```

### 2. 錯誤處理

```go
// 智能錯誤處理
func handleAssistantError(err error) {
    assistantErr := GetAssistantError(err)
    if assistantErr == nil {
        log.Printf("未預期的錯誤: %v", err)
        return
    }
    
    // 顯示用戶友好的錯誤訊息
    fmt.Printf("錯誤: %s\n", assistantErr.Message)
    
    // 顯示建議
    for _, suggestion := range assistantErr.Suggestions {
        fmt.Printf("💡 %s\n", suggestion.Description)
        if suggestion.Action != "" {
            fmt.Printf("   執行: %s\n", suggestion.Action)
        }
    }
    
    // 如果有自動恢復計劃
    if assistantErr.Recovery != nil && assistantErr.Recovery.Automated {
        fmt.Printf("🔄 嘗試自動恢復...\n")
        executeRecoveryPlan(assistantErr.Recovery)
    }
}
```

### 3. 性能監控

```go
// 性能監控中間件
func (a *assistant) withMetrics(next QueryHandler) QueryHandler {
    return func(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
        start := time.Now()
        
        // 執行查詢
        resp, err := next(ctx, req)
        
        // 記錄指標
        duration := time.Since(start)
        a.metrics.RecordQuery(req.UserID, duration, err == nil)
        
        // 記錄詳細性能數據
        if resp != nil {
            a.metrics.RecordConfidence(resp.Confidence)
            a.metrics.RecordToolUsage(resp.ToolsUsed)
        }
        
        return resp, err
    }
}
```

## 故障排除

### 常見問題

1. **查詢處理緩慢**
   - 檢查工具執行時間
   - 優化上下文分析
   - 調整並發設置

2. **回應質量不佳**
   - 增加上下文訊息
   - 調整代理選擇策略
   - 提供用戶反饋

3. **記憶體使用過高**
   - 調整緩存大小
   - 檢查上下文洩漏
   - 優化學習數據存儲

### 監控指標

- 查詢處理時間
- 回應信心度分布
- 工具使用統計
- 用戶滿意度評分
- 錯誤率和類型分布

---

*Assistant 核心模組是整個智能開發伴侶的大腦，通過語義理解、智能協作和持續學習，為開發者提供真正智能的編程助手體驗。*