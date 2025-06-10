# 🛠️ Tools Package - 智慧工具系統套件

## 📋 概述

Tools 套件是 Assistant 的工具生態系統核心，提供統一的工具註冊、發現、執行和管理機制。這個套件不僅整合了各種開發工具（Go、Docker、PostgreSQL、Kubernetes），還實現了工具間的智慧協作、語義理解和自適應執行。

## 🏗️ 架構設計

### 核心介面

```go
// Tool 定義了所有工具必須實現的介面
type Tool interface {
    // 基本資訊
    Name() string
    Description() string
    Category() Category
    
    // 執行功能
    Execute(ctx context.Context, params Parameters) (Result, error)
    Validate(params Parameters) error
    
    // 智慧功能
    CanExecute() bool
    EstimateDuration(params Parameters) time.Duration
    GetRequirements() []Requirement
}

// SemanticTool 擴展了基本工具介面，支援語義理解
type SemanticTool interface {
    Tool
    
    // 語義理解
    UnderstandIntent(intent string) (ToolIntent, error)
    SuggestParameters(intent ToolIntent) Parameters
    ExplainResult(result Result) string
    
    // 協作能力
    CanCollaborateWith(other Tool) bool
    SuggestCollaborations(goal string) []ToolChain
}
```

### 工具註冊表

```go
// Registry 管理所有可用工具
type Registry struct {
    tools      map[string]Tool      // 工具實例
    categories map[Category][]Tool  // 分類索引
    graph      *DependencyGraph     // 依賴關係圖
    discovery  *DiscoveryService    // 工具發現服務
    
    mu         sync.RWMutex         // 並發控制
    logger     *slog.Logger         // 結構化日誌
}

// 工具分類
type Category string

const (
    CategoryDevelopment    Category = "development"     // 開發工具
    CategoryDatabase      Category = "database"        // 資料庫工具
    CategoryInfrastructure Category = "infrastructure" // 基礎設施工具
    CategoryAnalysis      Category = "analysis"        // 分析工具
    CategoryIntegration   Category = "integration"     // 整合工具
)
```

## 🔧 核心功能

### 1. Go 開發工具 (GoDev Tool)

完整的 Go 開發工具鏈支援：

```go
type GoDevTool struct {
    workspace   *Workspace          // 工作空間管理
    analyzer    *Analyzer           // 程式碼分析器
    builder     *Builder            // 建置系統
    tester      *Tester             // 測試框架
    refactorer  *Refactorer         // 重構工具
}

// 支援的操作
const (
    ActionAnalyze    = "analyze"     // 分析程式碼品質
    ActionBuild      = "build"       // 建置專案
    ActionTest       = "test"        // 執行測試
    ActionFormat     = "format"      // 格式化程式碼
    ActionLint       = "lint"        // 程式碼檢查
    ActionRefactor   = "refactor"    // 重構建議
    ActionGenerate   = "generate"    // 生成程式碼
)

// 使用範例
result, err := goTool.Execute(ctx, Parameters{
    "action": ActionAnalyze,
    "path":   "./internal/assistant",
    "options": map[string]interface{}{
        "include_tests": true,
        "check_coverage": true,
        "suggest_improvements": true,
    },
})

// 結果包含
type AnalysisResult struct {
    CodeQuality    QualityMetrics   // 程式碼品質指標
    TestCoverage   Coverage         // 測試覆蓋率
    Complexity     []Complexity     // 複雜度分析
    Suggestions    []Suggestion     // 改進建議
    Dependencies   []Dependency     // 依賴分析
}
```

### 2. Docker 工具 (Docker Tool)

容器化開發和部署支援：

```go
type DockerTool struct {
    client      *docker.Client      // Docker 客戶端
    analyzer    *ImageAnalyzer      // 映像分析器
    optimizer   *Optimizer          // 優化器
    builder     *BuildEngine        // 建置引擎
}

// 功能特性
- Dockerfile 分析與優化
- 多階段建置優化
- 映像大小優化
- 安全掃描
- 層級快取優化
- 建置效能分析

// 智慧優化範例
result, err := dockerTool.Execute(ctx, Parameters{
    "action": "optimize",
    "dockerfile": "./Dockerfile",
    "target_size": "minimal",
    "security_scan": true,
})

// 優化結果
optimized := result.(*DockerOptimizationResult)
fmt.Printf("優化前: %s, 優化後: %s (減少 %.1f%%)\n", 
    optimized.OriginalSize, 
    optimized.OptimizedSize,
    optimized.SizeReduction)
```

### 3. PostgreSQL 工具 (Postgres Tool)

資料庫開發和優化工具：

```go
type PostgresTool struct {
    analyzer     *QueryAnalyzer      // 查詢分析器
    optimizer    *QueryOptimizer     // 查詢優化器
    indexAdvisor *IndexAdvisor       // 索引建議器
    migrator     *MigrationEngine    // 遷移引擎
    profiler     *Profiler           // 效能分析器
}

// 核心功能
- SQL 查詢分析與優化
- 執行計劃解析
- 索引建議
- 查詢效能預測
- 遷移生成與驗證
- 資料庫健康檢查

// 查詢優化範例
result, err := pgTool.Execute(ctx, Parameters{
    "action": "optimize_query",
    "query": `
        SELECT u.*, o.* 
        FROM users u 
        JOIN orders o ON u.id = o.user_id 
        WHERE u.status = 'active'
    `,
    "schema": schemaInfo,
})

// 優化建議
optimization := result.(*QueryOptimization)
fmt.Println("建議索引:", optimization.SuggestedIndexes)
fmt.Println("改寫查詢:", optimization.OptimizedQuery)
fmt.Println("預期改進:", optimization.ExpectedImprovement)
```

### 4. 工具鏈組合 (Tool Chaining)

實現複雜工作流程的工具組合：

```go
// ToolChain 定義工具執行鏈
type ToolChain struct {
    name    string
    steps   []ToolStep
    flow    FlowControl
    context *ChainContext
}

// 工具步驟
type ToolStep struct {
    Tool       Tool
    Parameters Parameters
    Condition  Condition           // 執行條件
    OnSuccess  []ToolStep          // 成功後執行
    OnFailure  []ToolStep          // 失敗後執行
    Transform  ResultTransformer   // 結果轉換
}

// 範例：完整的部署流程
deployChain := NewToolChain("full_deployment").
    AddStep(goTool, Parameters{"action": "test"}).
    AddStep(goTool, Parameters{"action": "build"}).
    AddStep(dockerTool, Parameters{"action": "build_image"}).
    AddStep(dockerTool, Parameters{"action": "security_scan"}).
    OnSuccess(
        AddStep(k8sTool, Parameters{"action": "deploy"}),
    ).
    OnFailure(
        AddStep(notifyTool, Parameters{"action": "alert_failure"}),
    )

result := deployChain.Execute(ctx)
```

### 5. 工具發現與推薦 (Tool Discovery)

智慧工具發現和推薦系統：

```go
// DiscoveryService 提供工具發現功能
type DiscoveryService struct {
    registry    *Registry
    matcher     *IntentMatcher      // 意圖匹配器
    recommender *Recommender        // 推薦引擎
    learner     *UsageLearner       // 使用學習器
}

// 根據意圖推薦工具
func (d *DiscoveryService) RecommendTools(intent string) []ToolRecommendation {
    // 1. 解析使用者意圖
    parsed := d.matcher.ParseIntent(intent)
    
    // 2. 匹配相關工具
    candidates := d.findCandidateTools(parsed)
    
    // 3. 根據歷史使用排序
    ranked := d.rankByUsagePattern(candidates)
    
    // 4. 生成推薦
    return d.generateRecommendations(ranked)
}

// 使用範例
recommendations := discovery.RecommendTools("我想優化我的 Docker 映像大小")
// 返回:
// 1. DockerTool (信心度: 95%) - 參數建議: {action: "optimize"}
// 2. DockerTool (信心度: 80%) - 參數建議: {action: "analyze"}
// 3. BuildTool  (信心度: 60%) - 參數建議: {action: "multi-stage"}
```

## 📊 進階功能

### 1. 智慧參數推導

```go
// ParameterInference 推導工具參數
type ParameterInference struct {
    contextAnalyzer *ContextAnalyzer
    historyMatcher  *HistoryMatcher
    defaultProvider *DefaultProvider
}

// 從上下文推導參數
func (p *ParameterInference) InferParameters(
    tool Tool,
    context Context,
) Parameters {
    params := make(Parameters)
    
    // 1. 從上下文提取相關資訊
    contextParams := p.contextAnalyzer.Extract(context)
    
    // 2. 匹配歷史使用模式
    historyParams := p.historyMatcher.Match(tool, context)
    
    // 3. 填充預設值
    defaultParams := p.defaultProvider.GetDefaults(tool)
    
    // 4. 智慧合併
    return p.merge(contextParams, historyParams, defaultParams)
}
```

### 2. 執行結果解釋

```go
// ResultInterpreter 解釋執行結果
type ResultInterpreter struct {
    templates  map[string]Template
    formatter  *ResultFormatter
    summarizer *Summarizer
}

// 生成人類可讀的結果解釋
func (r *ResultInterpreter) Interpret(
    tool Tool,
    result Result,
    context Context,
) string {
    // 1. 選擇適當的模板
    template := r.selectTemplate(tool, result)
    
    // 2. 格式化關鍵資訊
    formatted := r.formatter.Format(result)
    
    // 3. 生成摘要
    summary := r.summarizer.Summarize(result, context)
    
    // 4. 組合最終解釋
    return template.Render(map[string]interface{}{
        "summary":    summary,
        "details":    formatted,
        "suggestions": r.generateSuggestions(result),
    })
}
```

### 3. 工具健康監控

```go
// HealthMonitor 監控工具健康狀態
type HealthMonitor struct {
    checkers map[string]HealthChecker
    metrics  *MetricsCollector
    alerter  *Alerter
}

// 健康檢查
func (h *HealthMonitor) CheckHealth(tool Tool) HealthStatus {
    status := HealthStatus{
        Tool:      tool.Name(),
        Timestamp: time.Now(),
    }
    
    // 1. 基本可用性檢查
    status.Available = tool.CanExecute()
    
    // 2. 依賴檢查
    status.Dependencies = h.checkDependencies(tool)
    
    // 3. 效能指標
    status.Performance = h.metrics.GetMetrics(tool.Name())
    
    // 4. 資源使用
    status.Resources = h.checkResources(tool)
    
    return status
}
```

## 🔍 使用範例

### 完整工作流程
```go
// 1. 創建工具註冊表
registry := tools.NewRegistry()

// 2. 註冊工具
registry.Register(tools.NewGoDevTool())
registry.Register(tools.NewDockerTool())
registry.Register(tools.NewPostgresTool())

// 3. 創建工具服務
service := tools.NewService(registry)

// 4. 執行單一工具
result, err := service.ExecuteTool(ctx, "godev", tools.Parameters{
    "action": "analyze",
    "path":   "./",
})

// 5. 執行工具鏈
chain := service.CreateChain("build_and_deploy").
    AddStep("godev", tools.Parameters{"action": "test"}).
    AddStep("godev", tools.Parameters{"action": "build"}).
    AddStep("docker", tools.Parameters{"action": "build"})

chainResult := chain.Execute(ctx)

// 6. 智慧工具推薦
recommendations := service.Recommend("幫我優化資料庫查詢效能")
for _, rec := range recommendations {
    fmt.Printf("推薦: %s (信心度: %.0f%%)\n", 
        rec.Tool.Name(), rec.Confidence*100)
}
```

## 🧪 測試策略

### 工具測試框架
```go
func TestTool_Integration(t *testing.T) {
    // 設置測試環境
    env := setupTestEnvironment(t)
    tool := NewTestTool()
    
    // 測試基本執行
    t.Run("basic execution", func(t *testing.T) {
        result, err := tool.Execute(ctx, testParams)
        require.NoError(t, err)
        assert.NotNil(t, result)
    })
    
    // 測試錯誤處理
    t.Run("error handling", func(t *testing.T) {
        _, err := tool.Execute(ctx, invalidParams)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "invalid parameters")
    })
    
    // 測試效能
    t.Run("performance", func(t *testing.T) {
        start := time.Now()
        tool.Execute(ctx, largeParams)
        elapsed := time.Since(start)
        assert.Less(t, elapsed, 5*time.Second)
    })
}
```

## 🔧 配置選項

```yaml
tools:
  # 全域配置
  global:
    timeout: 30s
    max_concurrent: 10
    enable_caching: true
    cache_ttl: 1h
    
  # Go 開發工具配置
  godev:
    workspace: "./workspace"
    go_version: "1.24"
    enable_mod_cache: true
    parallel_builds: 4
    
  # Docker 工具配置
  docker:
    socket: "/var/run/docker.sock"
    registry: "docker.io"
    build_kit: true
    max_image_size: "500MB"
    
  # PostgreSQL 工具配置
  postgres:
    connection_pool_size: 10
    analyze_timeout: 5s
    explain_analyze: true
    suggest_indexes: true
    
  # 工具鏈配置
  chains:
    max_steps: 20
    enable_rollback: true
    parallel_execution: true
    failure_strategy: "stop" # stop, continue, rollback
```

## 📈 效能優化

1. **並行執行**
   - 工具鏈步驟並行化
   - 批次操作優化
   - 非阻塞執行模式

2. **快取策略**
   - 結果快取
   - 參數快取
   - 依賴快取

3. **資源管理**
   - 連線池管理
   - 記憶體限制
   - CPU 配額控制

## 🚀 未來規劃

1. **更多工具支援**
   - Rust 開發工具
   - Python 工具鏈
   - 前端框架工具
   - 雲端服務工具

2. **智慧增強**
   - 機器學習優化建議
   - 自動工具組合
   - 預測性工具推薦

3. **生態系統**
   - 工具市場
   - 社群貢獻工具
   - 工具開發 SDK

## 📚 相關文件

- [Assistant Package](../assistant/README-zh-TW.md) - 助理核心
- [LangChain Integration](../langchain/README-zh-TW.md) - LangChain 整合
- [Tool Development Guide](./DEVELOPMENT.md) - 工具開發指南
- [主要架構文件](../../CLAUDE-ARCHITECTURE.md) - 系統架構指南