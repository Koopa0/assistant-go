# LangChain Chains - 鏈式處理系統

本包實現了靈活的鏈式處理系統，允許組合多個 AI 操作以構建複雜的工作流程
支援順序執行、並行處理、條件分支和增強檢索生成 (RAG) 等多種執行模式

## 🎯 核心概念

### 鏈 (Chain)
一個鏈代表一系列有組織的處理步驟，能夠：
- 接收結構化輸入
- 執行一系列轉換操作
- 產生期望的輸出
- 追蹤執行過程和性能

### 步驟 (Step)
鏈執行過程中的單個處理單元：
- 包含輸入、處理邏輯和輸出
- 可以是 LLM 調用、工具執行或數據轉換
- 支援條件判斷和動態路由

### 上下文 (Context)
在鏈執行過程中傳遞的狀態信息：
- 儲存中間結果
- 維護執行歷史
- 提供步驟間的數據共享

## 🔗 鏈類型

### 1. 順序鏈 (Sequential Chain)

**用途**: 按順序執行一系列步驟，每步的輸出成為下一步的輸入

```go
type SequentialChain struct {
    *BaseChain
    steps []ChainStep
}
```

#### 使用範例

```go
// 創建順序鏈：代碼審查流程
chain := chains.NewSequentialChain(llm, config, logger)

// 定義處理步驟
steps := []chains.ChainStep{
    {
        Name:        "extract_code",
        Description: "從請求中提取代碼",
        Processor:   extractCodeProcessor,
    },
    {
        Name:        "analyze_syntax", 
        Description: "分析語法和結構",
        Processor:   syntaxAnalyzer,
    },
    {
        Name:        "check_best_practices",
        Description: "檢查最佳實踐",
        Processor:   bestPracticeChecker,
    },
    {
        Name:        "generate_report",
        Description: "生成審查報告", 
        Processor:   reportGenerator,
    },
}

chain.SetSteps(steps)

// 執行鏈
request := &chains.ChainRequest{
    Input: "請審查這段 Go 代碼...",
    Context: map[string]interface{}{
        "code": sourceCode,
        "standards": []string{"effective_go", "code_review_comments"},
    },
}

response, err := chain.Execute(ctx, request)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("審查報告: %s\n", response.Output)
```

#### 執行流程
```
輸入代碼 → 提取代碼 → 語法分析 → 最佳實踐檢查 → 生成報告 → 最終輸出
```

### 2. 並行鏈 (Parallel Chain)

**用途**: 同時執行多個獨立的處理分支，最後合併結果

```go
type ParallelChain struct {
    *BaseChain
    branches []ChainBranch
    merger   ResultMerger
}
```

#### 使用範例

```go
// 創建並行鏈：多角度分析
chain := chains.NewParallelChain(llm, config, logger)

// 定義並行分支
branches := []chains.ChainBranch{
    {
        Name:        "performance_analysis",
        Description: "效能分析",
        Chain:       performanceChain,
    },
    {
        Name:        "security_analysis", 
        Description: "安全分析",
        Chain:       securityChain,
    },
    {
        Name:        "maintainability_analysis",
        Description: "可維護性分析",
        Chain:       maintainabilityChain,
    },
}

chain.SetBranches(branches)

// 設定結果合併器
chain.SetMerger(comprehensiveAnalysisMerger)

// 執行並行分析
request := &chains.ChainRequest{
    Input: "分析這個微服務架構",
    Context: map[string]interface{}{
        "service_code": serviceCode,
        "architecture_docs": archDocs,
    },
}

response, err := chain.Execute(ctx, request)
```

#### 執行流程
```
            ┌─ 效能分析 ─┐
輸入 ───────┼─ 安全分析 ─┼─ 合併結果 ─→ 輸出 
            └─ 維護分析 ─┘
```

### 3. 條件鏈 (Conditional Chain)

**用途**: 根據條件動態選擇執行路徑

```go
type ConditionalChain struct {
    *BaseChain
    conditions []Condition
    routes     map[string]Chain
    fallback   Chain
}
```

#### 使用範例

```go
// 創建條件鏈：智能路由系統
chain := chains.NewConditionalChain(llm, config, logger)

// 定義條件和路由
conditions := []chains.Condition{
    {
        Name:      "is_code_related",
        Evaluator: isCodeRelatedEvaluator,
        Route:     "development_chain",
    },
    {
        Name:      "is_database_related",
        Evaluator: isDatabaseRelatedEvaluator, 
        Route:     "database_chain",
    },
    {
        Name:      "is_infrastructure_related",
        Evaluator: isInfraRelatedEvaluator,
        Route:     "infrastructure_chain",
    },
}

routes := map[string]chains.Chain{
    "development_chain":    devChain,
    "database_chain":       dbChain,
    "infrastructure_chain": infraChain,
}

chain.SetConditions(conditions)
chain.SetRoutes(routes)
chain.SetFallback(generalChain)

// 執行條件路由
request := &chains.ChainRequest{
    Input: "我的 Kubernetes Pod 一直重啟",
    Context: map[string]interface{}{
        "user_context": userProfile,
    },
}

response, err := chain.Execute(ctx, request)
```

#### 決策流程
```
輸入 → 條件評估 → 路由選擇 → 專門鏈執行 → 輸出
         ↓
    ┌─ 代碼相關? → 開發鏈
    ├─ DB相關?   → 資料庫鏈  
    ├─ 基礎設施? → 基礎設施鏈
    └─ 其他      → 通用鏈
```

### 4. RAG 鏈 (RAG Chain)

**用途**: 結合文檔檢索和語言生成的增強回答系統

```go
type RAGChain struct {
    *BaseChain
    vectorStore       vectorstores.VectorStore
    retriever         schema.Retriever
    embedder          embeddings.Embedder
    docProcessor      *documentloader.DocumentProcessor
    retrievalConfig   RAGRetrievalConfig
}
```

#### 核心配置

```go
type RAGRetrievalConfig struct {
    MaxDocuments        int      `json:"max_documents"`         // 最大檢索文檔數
    SimilarityThreshold float64  `json:"similarity_threshold"`  // 相似度閾值  
    ContentTypes        []string `json:"content_types"`         // 內容類型過濾
    IncludeMetadata     bool     `json:"include_metadata"`      // 包含元數據
    RetrievalStrategy   string   `json:"retrieval_strategy"`    // 檢索策略
}
```

#### 使用範例

```go
// 創建 RAG 鏈
ragChain := chains.NewRAGChain(llm, vectorStore, embedder, config, logger)

// 配置檢索參數
config := chains.RAGRetrievalConfig{
    MaxDocuments:        5,
    SimilarityThreshold: 0.8,
    ContentTypes:        []string{"documentation", "code"},
    IncludeMetadata:     true,
    RetrievalStrategy:   "similarity",
}

ragChain.SetRetrievalConfig(config)

// 執行 RAG 查詢
request := &chains.ChainRequest{
    Input: "如何在 Go 中實現高效的 JSON 處理？",
    Parameters: map[string]interface{}{
        "focus_areas": []string{"performance", "memory_usage"},
        "include_examples": true,
    },
}

response, err := ragChain.Execute(ctx, request)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("回答: %s\n", response.Output)
fmt.Printf("參考文檔: %d 個\n", len(response.Steps))
```

#### RAG 執行流程
```
用戶查詢 → 向量化 → 相似度搜尋 → 檢索文檔 → 上下文構建 → LLM生成 → 增強回答
    ↓            ↓            ↓            ↓            ↓           ↓
  嵌入查詢    文檔向量庫    排序過濾    提取內容    構建提示    生成回答
```

## 🔧 高級 RAG 功能

### 增強 RAG 鏈 (Enhanced RAG Chain)

```go
// 創建增強 RAG 鏈
enhancedRAG := chains.NewEnhancedRAGChain(llm, vectorStore, embedder, config, logger)

// 文檔攝取
err := enhancedRAG.IngestDocument(ctx, "docs/go-best-practices.md", map[string]any{
    "category": "best_practices",
    "language": "go", 
    "priority": "high",
})

// 批量攝取目錄
err = enhancedRAG.IngestDirectory(ctx, "docs/", true, []string{".md", ".txt"}, map[string]any{
    "source": "documentation",
    "version": "latest",
})

// 從 URL 攝取
err = enhancedRAG.IngestFromURL(ctx, "https://golang.org/doc/effective_go.html", map[string]any{
    "source": "official_docs",
})

// 帶來源的查詢
result, err := enhancedRAG.QueryWithSources(ctx, "Go 的錯誤處理最佳實踐", map[string]interface{}{
    "include_sources": true,
    "max_sources": 3,
})

if err != nil {
    log.Fatal(err)
}

fmt.Printf("回答: %s\n", result.Answer)
fmt.Printf("來源文檔:\n")
for _, source := range result.Sources {
    fmt.Printf("- %s (相似度: %.3f)\n", source.Title, source.Score)
}
```

### 自定義檢索策略

```go
// 混合檢索策略
type HybridRetriever struct {
    vectorRetriever   schema.Retriever
    keywordRetriever  KeywordRetriever
    reranker         Reranker
}

func (h *HybridRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
    // 向量檢索
    vectorDocs, err := h.vectorRetriever.GetRelevantDocuments(ctx, query)
    if err != nil {
        return nil, err
    }
    
    // 關鍵字檢索
    keywordDocs, err := h.keywordRetriever.Search(ctx, query)
    if err != nil {
        return nil, err
    }
    
    // 合併和重排序
    allDocs := append(vectorDocs, keywordDocs...)
    rankedDocs := h.reranker.Rerank(ctx, query, allDocs)
    
    return rankedDocs, nil
}

// 設定自定義檢索器
ragChain.SetRetriever(hybridRetriever)
```

## 🔄 鏈組合和嵌套

### 複雜工作流程

```go
// 構建複雜的嵌套鏈：完整的開發工作流程
func BuildDevelopmentWorkflow() chains.Chain {
    // 1. 代碼分析鏈
    codeAnalysisChain := chains.NewSequentialChain(llm, config, logger)
    codeAnalysisChain.SetSteps([]chains.ChainStep{
        {Name: "extract_code", Processor: codeExtractor},
        {Name: "ast_analysis", Processor: astAnalyzer},
        {Name: "complexity_check", Processor: complexityChecker},
    })
    
    // 2. 並行品質檢查鏈
    qualityCheckChain := chains.NewParallelChain(llm, config, logger)
    qualityCheckChain.SetBranches([]chains.ChainBranch{
        {Name: "performance", Chain: performanceChain},
        {Name: "security", Chain: securityChain},
        {Name: "style", Chain: styleChain},
    })
    
    // 3. 條件修復鏈
    fixChain := chains.NewConditionalChain(llm, config, logger)
    fixChain.SetConditions([]chains.Condition{
        {
            Name: "has_critical_issues",
            Evaluator: func(ctx context.Context, input map[string]interface{}) (bool, error) {
                issues := input["issues"].([]Issue)
                for _, issue := range issues {
                    if issue.Severity == "critical" {
                        return true, nil
                    }
                }
                return false, nil
            },
            Route: "auto_fix_chain",
        },
    })
    
    // 4. 主工作流程鏈
    workflowChain := chains.NewSequentialChain(llm, config, logger)
    workflowChain.SetSteps([]chains.ChainStep{
        {Name: "code_analysis", Processor: codeAnalysisChain},
        {Name: "quality_check", Processor: qualityCheckChain},
        {Name: "conditional_fix", Processor: fixChain},
        {Name: "generate_report", Processor: reportGenerator},
    })
    
    return workflowChain
}
```

### 動態鏈構建

```go
// 根據用戶需求動態構建鏈
func BuildDynamicChain(requirements []string) chains.Chain {
    var steps []chains.ChainStep
    
    for _, req := range requirements {
        switch req {
        case "code_review":
            steps = append(steps, chains.ChainStep{
                Name: "code_review",
                Processor: codeReviewProcessor,
            })
        case "performance_analysis":
            steps = append(steps, chains.ChainStep{
                Name: "performance_analysis", 
                Processor: performanceProcessor,
            })
        case "security_scan":
            steps = append(steps, chains.ChainStep{
                Name: "security_scan",
                Processor: securityProcessor,
            })
        }
    }
    
    chain := chains.NewSequentialChain(llm, config, logger)
    chain.SetSteps(steps)
    
    return chain
}

// 使用動態鏈
requirements := []string{"code_review", "security_scan"}
dynamicChain := BuildDynamicChain(requirements)

response, err := dynamicChain.Execute(ctx, request)
```

## 📊 鏈監控和分析

### 執行統計

```go
// 鏈執行統計
type ChainStats struct {
    ChainType            string        `json:"chain_type"`
    TotalExecutions      int           `json:"total_executions"`
    SuccessRate          float64       `json:"success_rate"`
    AverageExecutionTime time.Duration `json:"average_execution_time"`
    AverageSteps         float64       `json:"average_steps"`
    AverageTokensUsed    int           `json:"average_tokens_used"`
    StepStats            map[string]StepStats `json:"step_stats"`
}

type StepStats struct {
    ExecutionCount int           `json:"execution_count"`
    SuccessRate    float64       `json:"success_rate"`
    AverageTime    time.Duration `json:"average_time"`
    ErrorRate      float64       `json:"error_rate"`
}

// 獲取鏈統計資料
func GetChainStats(ctx context.Context, chainType chains.ChainType, timeRange time.Duration) (*ChainStats, error) {
    // 從資料庫查詢統計
    stats := &ChainStats{
        ChainType:            string(chainType),
        TotalExecutions:      89,
        SuccessRate:          0.95,
        AverageExecutionTime: 3200 * time.Millisecond,
        AverageSteps:         4.2,
        AverageTokensUsed:    1850,
        StepStats: map[string]StepStats{
            "retrieval": {
                ExecutionCount: 89,
                SuccessRate:    0.98,
                AverageTime:    800 * time.Millisecond,
                ErrorRate:      0.02,
            },
            "generation": {
                ExecutionCount: 87,
                SuccessRate:    0.96,
                AverageTime:    2100 * time.Millisecond, 
                ErrorRate:      0.04,
            },
        },
    }
    
    return stats, nil
}
```

### 性能優化

```go
// 鏈性能優化器
type ChainOptimizer struct {
    profiler  *ChainProfiler
    optimizer *ExecutionOptimizer
}

func (co *ChainOptimizer) OptimizeChain(chain chains.Chain) chains.Chain {
    // 分析性能瓶頸
    profile := co.profiler.ProfileChain(chain)
    
    // 識別優化機會
    optimizations := co.optimizer.IdentifyOptimizations(profile)
    
    // 應用優化
    optimizedChain := chain
    for _, opt := range optimizations {
        optimizedChain = opt.Apply(optimizedChain)
    }
    
    return optimizedChain
}

// 性能優化建議
type OptimizationSuggestion struct {
    Type        string  `json:"type"`
    Description string  `json:"description"`
    Impact      string  `json:"impact"`
    Confidence  float64 `json:"confidence"`
}

func AnalyzeChainPerformance(stats *ChainStats) []OptimizationSuggestion {
    var suggestions []OptimizationSuggestion
    
    // 檢查執行時間
    if stats.AverageExecutionTime > 5*time.Second {
        suggestions = append(suggestions, OptimizationSuggestion{
            Type:        "execution_time",
            Description: "考慮並行化某些步驟以減少執行時間",
            Impact:      "high",
            Confidence:  0.85,
        })
    }
    
    // 檢查 Token 使用
    if stats.AverageTokensUsed > 3000 {
        suggestions = append(suggestions, OptimizationSuggestion{
            Type:        "token_usage",
            Description: "優化提示詞以減少 Token 消耗",
            Impact:      "medium",
            Confidence:  0.75,
        })
    }
    
    return suggestions
}
```

## 🔧 自定義鏈開發

### 創建自定義鏈類型

```go
// 自定義審計鏈
type AuditChain struct {
    *chains.BaseChain
    auditRules    []AuditRule
    complianceDB  ComplianceDatabase
    reportFormat  ReportFormat
}

// 審計規則
type AuditRule struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Evaluator   func(ctx context.Context, data interface{}) (*AuditResult, error)
    Severity    string                 `json:"severity"`
    Framework   string                 `json:"framework"`
}

// 創建審計鏈
func NewAuditChain(llm llms.Model, config config.LangChain, logger *slog.Logger) *AuditChain {
    base := chains.NewBaseChain("audit", llm, config, logger)
    
    auditChain := &AuditChain{
        BaseChain:    base,
        auditRules:   loadAuditRules(),
        complianceDB: NewComplianceDB(),
        reportFormat: JSONReportFormat,
    }
    
    return auditChain
}

// 實現鏈執行邏輯
func (ac *AuditChain) Execute(ctx context.Context, request *chains.ChainRequest) (*chains.ChainResponse, error) {
    startTime := time.Now()
    steps := make([]chains.ChainStep, 0)
    
    // 步驟 1: 數據收集
    stepStart := time.Now()
    auditData, err := ac.collectAuditData(ctx, request)
    if err != nil {
        return nil, fmt.Errorf("審計數據收集失敗: %w", err)
    }
    
    step1 := chains.ChainStep{
        StepNumber:   1,
        StepType:     "data_collection",
        Input:        request.Input,
        Output:       fmt.Sprintf("收集了 %d 項審計數據", len(auditData)),
        ExecutionTime: time.Since(stepStart),
        Success:      true,
    }
    steps = append(steps, step1)
    
    // 步驟 2: 規則評估
    stepStart = time.Now()
    results := make([]AuditResult, 0)
    
    for _, rule := range ac.auditRules {
        result, err := rule.Evaluator(ctx, auditData)
        if err != nil {
            ac.logger.Warn("審計規則評估失敗",
                slog.String("rule", rule.Name),
                slog.Any("error", err))
            continue
        }
        results = append(results, *result)
    }
    
    step2 := chains.ChainStep{
        StepNumber:   2,
        StepType:     "rule_evaluation",
        Input:        fmt.Sprintf("%d 條審計規則", len(ac.auditRules)),
        Output:       fmt.Sprintf("評估完成，發現 %d 項結果", len(results)),
        ExecutionTime: time.Since(stepStart),
        Success:      true,
    }
    steps = append(steps, step2)
    
    // 步驟 3: 報告生成
    stepStart = time.Now()
    report, err := ac.generateAuditReport(ctx, results)
    if err != nil {
        return nil, fmt.Errorf("審計報告生成失敗: %w", err)
    }
    
    step3 := chains.ChainStep{
        StepNumber:   3,
        StepType:     "report_generation",
        Input:        fmt.Sprintf("%d 項審計結果", len(results)),
        Output:       "審計報告已生成",
        ExecutionTime: time.Since(stepStart),
        Success:      true,
    }
    steps = append(steps, step3)
    
    // 構建響應
    response := &chains.ChainResponse{
        Output:        report,
        Steps:         steps,
        ExecutionTime: time.Since(startTime),
        Success:       true,
        Metadata: map[string]interface{}{
            "audit_rules_count": len(ac.auditRules),
            "findings_count":    len(results),
            "compliance_score":  ac.calculateComplianceScore(results),
        },
    }
    
    return response, nil
}
```

### 鏈中間件

```go
// 鏈中間件接口
type ChainMiddleware interface {
    Before(ctx context.Context, request *chains.ChainRequest) (*chains.ChainRequest, error)
    After(ctx context.Context, response *chains.ChainResponse) (*chains.ChainResponse, error)
    OnError(ctx context.Context, err error) error
}

// 日誌中間件
type LoggingMiddleware struct {
    logger *slog.Logger
}

func (lm *LoggingMiddleware) Before(ctx context.Context, request *chains.ChainRequest) (*chains.ChainRequest, error) {
    lm.logger.Info("鏈執行開始",
        slog.String("input", request.Input),
        slog.Any("context", request.Context))
    return request, nil
}

func (lm *LoggingMiddleware) After(ctx context.Context, response *chains.ChainResponse) (*chains.ChainResponse, error) {
    lm.logger.Info("鏈執行完成",
        slog.Bool("success", response.Success),
        slog.Duration("execution_time", response.ExecutionTime),
        slog.Int("steps", len(response.Steps)))
    return response, nil
}

// 性能監控中間件
type PerformanceMiddleware struct {
    metrics MetricsCollector
}

func (pm *PerformanceMiddleware) Before(ctx context.Context, request *chains.ChainRequest) (*chains.ChainRequest, error) {
    ctx = context.WithValue(ctx, "start_time", time.Now())
    return request, nil
}

func (pm *PerformanceMiddleware) After(ctx context.Context, response *chains.ChainResponse) (*chains.ChainResponse, error) {
    startTime := ctx.Value("start_time").(time.Time)
    duration := time.Since(startTime)
    
    pm.metrics.RecordChainExecution(duration, response.Success, len(response.Steps))
    return response, nil
}

// 應用中間件
func ApplyMiddleware(chain chains.Chain, middlewares ...ChainMiddleware) chains.Chain {
    return &MiddlewareChain{
        chain:       chain,
        middlewares: middlewares,
    }
}
```

## 🐛 除錯和故障排除

### 常見問題

1. **鏈執行超時**
```go
// 解決方案：調整超時設定
config := config.LangChain{
    ChainTimeout: 10 * time.Minute,
    StepTimeout:  2 * time.Minute,
}
```

2. **RAG 檢索結果不相關**
```go
// 解決方案：調整相似度閾值
ragConfig := chains.RAGRetrievalConfig{
    SimilarityThreshold: 0.85, // 提高閾值
    MaxDocuments:        3,    // 減少文檔數量
}
```

3. **並行鏈性能問題**
```go
// 解決方案：限制並發數
parallelChain.SetMaxConcurrency(4)
parallelChain.SetTimeout(5 * time.Minute)
```

### 除錯工具

```go
// 鏈執行追蹤器
type ChainTracer struct {
    traceID string
    events  []TraceEvent
}

type TraceEvent struct {
    Timestamp time.Time              `json:"timestamp"`
    Type      string                 `json:"type"`
    StepName  string                 `json:"step_name"`
    Input     interface{}            `json:"input"`
    Output    interface{}            `json:"output"`
    Metadata  map[string]interface{} `json:"metadata"`
}

// 啟用鏈追蹤
func EnableChainTracing(chain chains.Chain) chains.Chain {
    tracer := &ChainTracer{
        traceID: generateTraceID(),
        events:  make([]TraceEvent, 0),
    }
    
    return &TracedChain{
        chain:  chain,
        tracer: tracer,
    }
}
```

## 📚 最佳實踐

### 1. 鏈設計原則
- **模組化**: 將複雜邏輯分解為簡單的步驟
- **可重用**: 設計可在多個場景重用的鏈組件
- **容錯性**: 實現優雅的錯誤處理和降級機制
- **可觀測性**: 提供充分的日誌和監控

### 2. 性能優化
- **並行化**: 在可能的情況下並行執行獨立步驟
- **快取**: 快取重複的計算結果和檢索
- **批處理**: 批量處理相似的操作
- **資源管理**: 合理設定超時和資源限制

### 3. RAG 優化
- **文檔品質**: 確保攝取高品質、結構化的文檔
- **分段策略**: 選擇適當的文檔分段大小和重疊
- **嵌入品質**: 使用高品質的嵌入模型
- **檢索調優**: 根據具體用例調整檢索參數

### 4. 錯誤處理
- **重試機制**: 對暫時性錯誤實施指數退避重試
- **降級策略**: 在組件失敗時提供替代方案
- **錯誤分類**: 區分系統錯誤、用戶錯誤和業務邏輯錯誤
- **上下文保存**: 在錯誤發生時保存充分的上下文信息

## 🔗 相關資源

- [LangChain 主要文檔](../README.md)
- [Agents 代理系統](../agents/README.md)
- [Memory 記憶管理](../memory/README.md)
- [VectorStore 向量儲存](../vectorstore/README.md)
- [DocumentLoader 文檔處理](../documentloader/README.md)

## 🤝 貢獻指南

歡迎貢獻新的鏈類型或改進現有鏈：

1. **新鏈類型**: 繼承 `BaseChain` 並實現特定邏輯
2. **中間件**: 開發新的鏈中間件以增強功能
3. **優化器**: 貢獻性能優化和監控工具
4. **測試**: 為新功能添加充分的測試覆蓋
5. **文檔**: 更新相關文檔和使用範例

請遵循專案的編碼標準，確保向後相容性，並提供清晰的文檔說明。