# LangChain Agents - AI 代理系統

本包實現了專業化的 AI 代理系統，每個代理都專精於特定領域的任務執行。代理能夠使用工具、管理記憶、執行複雜的多步驟推理，並提供詳細的執行追蹤。

## 🎯 核心概念

### 代理 (Agent)
專業化的 AI 助手，能夠：
- 理解自然語言查詢
- 分解複雜任務為多個步驟
- 選擇合適的工具執行任務
- 記住過往互動歷史
- 提供詳細的執行推理

### 能力 (Capability)
每個代理擁有的專業技能集合，定義了：
- 能力名稱和描述
- 輸入參數規格
- 執行邏輯

### 步驟 (Step)
代理執行過程中的單個操作單元，包含：
- 動作描述
- 使用的工具
- 推理過程
- 執行結果

## 🏗️ 代理架構

```
BaseAgent (基礎代理)
├── DevelopmentAgent    (開發代理)
├── DatabaseAgent       (資料庫代理) 
├── InfrastructureAgent (基礎設施代理)
└── ResearchAgent       (研究代理)
```

## 🤖 可用代理

### 1. 開發代理 (DevelopmentAgent)

**專業領域**: Go 開發、程式碼分析、效能優化

#### 核心能力

```go
// 程式碼分析
{
    Name: "code_analysis",
    Description: "分析 Go 程式碼結構、依賴和模式",
    Parameters: {
        "file_path":     "string",
        "analysis_type": "string (structure|dependencies|patterns|all)",
    },
}

// 程式碼生成
{
    Name: "code_generation", 
    Description: "基於規格生成 Go 程式碼",
    Parameters: {
        "specification": "string",
        "code_type":     "string (function|struct|interface|package)",
        "style_guide":   "string",
    },
}

// 效能分析
{
    Name: "performance_analysis",
    Description: "分析程式碼效能並提供優化建議", 
    Parameters: {
        "code":           "string",
        "analysis_depth": "string (basic|detailed|comprehensive)",
    },
}
```

#### 使用範例

```go
// 創建開發代理
agent := agents.NewDevelopmentAgent(llm, config, logger)

// 程式碼分析請求
request := &agents.AgentRequest{
    Query: "分析這段 Go 程式碼的結構和潛在問題",
    Context: map[string]interface{}{
        "code": `
            func ProcessData(data []string) error {
                for i := 0; i < len(data); i++ {
                    if data[i] == "" {
                        return errors.New("empty data")
                    }
                    // 處理邏輯...
                }
                return nil
            }
        `,
    },
    MaxSteps: 5,
}

response, err := agent.Execute(ctx, request)
if err != nil {
    log.Fatal(err)
}

// 查看分析結果
fmt.Printf("分析結果: %s\n", response.Result)
for _, step := range response.Steps {
    fmt.Printf("步驟 %d: %s\n", step.StepNumber, step.Reasoning)
}
```

#### 輸出範例
```
步驟 1: 提取程式碼內容進行 AST 分析
步驟 2: 執行結構性分析，發現 1 個函數，0 個結構體
步驟 3: 生成分析報告，識別效能優化機會

分析結果:
## 程式碼結構分析

### 發現的問題
1. **效能問題**: 使用 for 迴圈而非 range，效率較低
2. **錯誤處理**: 使用 errors.New 而非 fmt.Errorf，缺乏上下文

### 建議改進
```go
func ProcessData(data []string) error {
    for i, item := range data {
        if item == "" {
            return fmt.Errorf("empty data at index %d", i)
        }
        // 處理邏輯...
    }
    return nil
}
```
```

### 2. 資料庫代理 (DatabaseAgent)

**專業領域**: SQL 查詢、資料庫優化、架構設計

#### 核心能力

```go
// SQL 生成
{
    Name: "sql_generation",
    Description: "從自然語言描述生成 SQL 查詢",
    Parameters: {
        "description": "string",
        "table_names": "[]string", 
        "query_type":  "string (SELECT|INSERT|UPDATE|DELETE)",
        "complexity":  "string (simple|medium|complex)",
    },
}

// 查詢優化
{
    Name: "query_optimization",
    Description: "分析並優化 SQL 查詢效能",
    Parameters: {
        "query":             "string",
        "optimization_type": "string (performance|readability|maintainability)",
        "include_explain":   "bool",
    },
}

// 架構探索
{
    Name: "schema_exploration", 
    Description: "探索資料庫架構和關係",
    Parameters: {
        "exploration_type": "string (tables|relationships|indexes|constraints)",
        "table_pattern":    "string",
        "include_data":     "bool",
    },
}
```

#### 使用範例

```go
// SQL 生成
request := &agents.AgentRequest{
    Query: "生成一個查詢來找出過去 30 天內最活躍的用戶",
    Context: map[string]interface{}{
        "tables": []string{"users", "user_activities", "sessions"},
    },
    MaxSteps: 3,
}

response, err := databaseAgent.Execute(ctx, request)
```

#### 輸出範例
```sql
-- 生成的優化 SQL 查詢
SELECT 
    u.id,
    u.username,
    COUNT(ua.id) as activity_count,
    MAX(ua.created_at) as last_activity
FROM users u
JOIN user_activities ua ON u.id = ua.user_id
WHERE ua.created_at >= NOW() - INTERVAL '30 days'
GROUP BY u.id, u.username
ORDER BY activity_count DESC
LIMIT 10;

-- 查詢說明：
-- 1. 聯接用戶和活動表
-- 2. 過濾過去 30 天的活動  
-- 3. 按用戶分組並計算活動次數
-- 4. 依活動次數降序排列
```

### 3. 基礎設施代理 (InfrastructureAgent)

**專業領域**: Kubernetes、Docker、系統監控、故障排除

#### 核心能力

```go
// Kubernetes 管理
{
    Name: "kubernetes_management",
    Description: "管理 Kubernetes 叢集、Pod、服務和部署",
    Parameters: {
        "action":    "string (list|describe|create|delete|scale|logs)",
        "resource":  "string (pods|services|deployments|nodes)",
        "namespace": "string",
        "name":      "string",
    },
}

// Docker 管理  
{
    Name: "docker_management",
    Description: "管理 Docker 容器、映像和網路",
    Parameters: {
        "action":   "string (list|run|stop|remove|logs|inspect)",
        "resource": "string (containers|images|networks|volumes)",
        "name":     "string",
        "image":    "string",
        "ports":    "[]string",
    },
}

// 日誌分析
{
    Name: "log_analysis",
    Description: "分析應用程式和系統日誌以發現問題和模式",
    Parameters: {
        "log_source":   "string (kubernetes|docker|system|application)",
        "time_range":   "string",
        "log_level":    "string (error|warn|info|debug)",
        "search_terms": "[]string",
    },
}
```

#### 使用範例

```go
// Kubernetes 故障排除
request := &agents.AgentRequest{
    Query: "我的 Web 應用 Pod 一直重啟，幫我診斷問題",
    Context: map[string]interface{}{
        "namespace": "production",
        "app_name":  "web-app",
    },
    MaxSteps: 8,
}

response, err := infraAgent.Execute(ctx, request)
```

#### 輸出範例
```
步驟 1: 解析 Kubernetes 命令 - 檢查 Pod 狀態
步驟 2: 執行 kubectl get pods - 發現 3 個 Pod 處於 CrashLoopBackOff 狀態
步驟 3: 檢查 Pod 日誌 - 發現記憶體不足錯誤

診斷結果:
## Pod 重啟問題分析

### 發現的問題
1. **記憶體限制過低**: Pod 記憶體限制設定為 128Mi，但應用需要至少 256Mi
2. **資源配置不當**: 沒有設定 requests，導致調度問題
3. **健康檢查過於嚴格**: livenessProbe 超時時間過短

### 建議解決方案
```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
    
livenessProbe:
  timeoutSeconds: 30
  periodSeconds: 10
```
```

### 4. 研究代理 (ResearchAgent)

**專業領域**: 資訊收集、事實查核、報告生成

#### 核心能力

```go
// 資訊收集
{
    Name: "information_gathering",
    Description: "從多個來源收集特定主題的資訊",
    Parameters: {
        "topic":      "string",
        "sources":    "[]string (web|academic|documentation|internal)",
        "depth":      "string (surface|detailed|comprehensive)",
        "time_range": "string",
        "language":   "string",
    },
}

// 事實查核
{
    Name: "fact_checking",
    Description: "對照可靠來源驗證事實和聲明",
    Parameters: {
        "claims":      "[]string",
        "sources":     "[]string", 
        "confidence":  "string (low|medium|high)",
        "cross_check": "bool",
    },
}

// 報告生成
{
    Name: "report_generation",
    Description: "從收集的資訊生成綜合性報告",
    Parameters: {
        "report_type":       "string (summary|analysis|comparison|recommendation)",
        "format":            "string (markdown|html|pdf|json)",
        "sections":          "[]string",
        "citations":         "bool",
        "executive_summary": "bool",
    },
}
```

#### 使用範例

```go
// 技術研究
request := &agents.AgentRequest{
    Query: "研究 Go 1.22 的新功能和改進",
    Context: map[string]interface{}{
        "depth": "comprehensive",
        "include_examples": true,
    },
    MaxSteps: 6,
}

response, err := researchAgent.Execute(ctx, request)
```

#### 輸出範例
```markdown
# Go 1.22 新功能研究報告

## 執行摘要
Go 1.22 版本帶來了重要的語言改進和效能優化，主要包括 for range 語意變更、切片改進和增強的標準庫功能。

## 主要新功能

### 1. For Range 語意改進
- **變更**: 迴圈變數現在在每次迭代時重新創建
- **影響**: 解決了 Go 開發者常遇到的閉包陷阱
- **範例**:
```go
// Go 1.22 之前需要額外變數
// Go 1.22+ 可以直接使用
for i, v := range items {
    go func() {
        fmt.Println(i, v) // 現在會印出正確的值
    }()
}
```

### 2. 切片改進
- **功能**: 新增 slices.Concat() 函數
- **用途**: 高效率連接多個切片
- **效能**: 比手動 append 快 15-20%

## 來源資訊
1. Go 官方發布說明 - [golang.org/doc/go1.22](https://golang.org/doc/go1.22)
2. Go 部落格文章 - 語言改進詳解
3. 社群回饋 - Reddit /r/golang 討論

*報告生成時間: 2024-01-15*
*資料收集來源: 3 個官方來源, 2 個社群來源*
```

## 🔧 自定義代理

### 創建新代理類型

```go
// 自定義代理結構
type SecurityAgent struct {
    *agents.BaseAgent
    scannerTool   *SecurityScanner
    reportTool    *VulnReporter
}

// 創建安全代理
func NewSecurityAgent(llm llms.Model, config config.LangChain, logger *slog.Logger) *SecurityAgent {
    base := agents.NewBaseAgent("security", llm, config, logger)
    
    agent := &SecurityAgent{
        BaseAgent:   base,
        scannerTool: NewSecurityScanner(),
        reportTool:  NewVulnReporter(),
    }
    
    // 定義安全代理能力
    agent.initializeCapabilities()
    
    return agent
}

// 初始化能力
func (s *SecurityAgent) initializeCapabilities() {
    capabilities := []agents.AgentCapability{
        {
            Name:        "vulnerability_scan",
            Description: "掃描程式碼和基礎設施的安全漏洞",
            Parameters: map[string]interface{}{
                "target_type": "string (code|infrastructure|network)",
                "scan_depth":  "string (quick|thorough|comprehensive)",
                "report_format": "string (json|html|pdf)",
            },
        },
        {
            Name:        "security_audit",
            Description: "執行安全稽核和合規性檢查",
            Parameters: map[string]interface{}{
                "audit_framework": "string (owasp|nist|iso27001)",
                "scope":           "[]string",
                "include_remediation": "bool",
            },
        },
    }
    
    for _, capability := range capabilities {
        s.AddCapability(capability)
    }
}

// 實作執行邏輯
func (s *SecurityAgent) executeSteps(ctx context.Context, request *agents.AgentRequest, maxSteps int) (string, []agents.AgentStep, error) {
    // 分析請求類型
    taskType, err := s.analyzeSecurityTask(request.Query)
    if err != nil {
        return "", nil, err
    }
    
    // 執行對應的安全任務
    switch taskType {
    case "vulnerability_scan":
        return s.executeVulnerabilityScan(ctx, request, maxSteps)
    case "security_audit":
        return s.executeSecurityAudit(ctx, request, maxSteps)
    default:
        return s.BaseAgent.Execute(ctx, request)
    }
}
```

### 擴展現有代理

```go
// 為開發代理添加新能力
func ExtendDevelopmentAgent(agent *agents.DevelopmentAgent) {
    // 添加程式碼品質檢查能力
    agent.AddCapability(agents.AgentCapability{
        Name:        "code_quality_check",
        Description: "執行程式碼品質和最佳實踐檢查",
        Parameters: map[string]interface{}{
            "standards":    "[]string (effective_go|code_review_comments)",
            "severity":     "string (info|warning|error)",
            "auto_fix":     "bool",
        },
    })
    
    // 添加依賴分析能力
    agent.AddCapability(agents.AgentCapability{
        Name:        "dependency_analysis", 
        Description: "分析專案依賴和潛在安全風險",
        Parameters: map[string]interface{}{
            "include_indirect": "bool",
            "check_vulnerabilities": "bool",
            "suggest_updates": "bool",
        },
    })
}
```

## 🔄 代理協作

### 代理鏈式協作

```go
// 多代理協作範例：從開發到部署
func DeploymentWorkflow(ctx context.Context, agents *AgentRegistry) error {
    // 1. 開發代理：程式碼審查
    devRequest := &agents.AgentRequest{
        Query: "審查程式碼並確保品質符合標準",
        Context: map[string]interface{}{
            "repository": "github.com/example/app",
            "branch": "feature/new-api",
        },
    }
    
    devResult, err := agents.Development.Execute(ctx, devRequest)
    if err != nil {
        return fmt.Errorf("程式碼審查失敗: %w", err)
    }
    
    // 2. 資料庫代理：檢查 Migration
    dbRequest := &agents.AgentRequest{
        Query: "檢查資料庫 migration 的安全性和效能",
        Context: map[string]interface{}{
            "migrations": devResult.Metadata["migrations"],
        },
    }
    
    dbResult, err := agents.Database.Execute(ctx, dbRequest) 
    if err != nil {
        return fmt.Errorf("資料庫檢查失敗: %w", err)
    }
    
    // 3. 基礎設施代理：執行部署
    infraRequest := &agents.AgentRequest{
        Query: "部署應用到生產環境",
        Context: map[string]interface{}{
            "image_tag": devResult.Metadata["build_tag"],
            "db_ready": dbResult.Metadata["migration_status"],
        },
    }
    
    _, err = agents.Infrastructure.Execute(ctx, infraRequest)
    if err != nil {
        return fmt.Errorf("部署失敗: %w", err)
    }
    
    return nil
}
```

### 代理記憶共享

```go
// 跨代理記憶共享
type SharedMemoryManager struct {
    agents map[string]agents.Agent
    memory memory.SharedMemory
}

func (sm *SharedMemoryManager) ShareContext(fromAgent, toAgent string, context map[string]interface{}) error {
    // 儲存共享上下文
    memoryEntry := &memory.MemoryEntry{
        Type:    memory.MemoryTypeShared,
        Content: fmt.Sprintf("從 %s 到 %s 的上下文共享", fromAgent, toAgent),
        Context: context,
        Metadata: map[string]interface{}{
            "source_agent": fromAgent,
            "target_agent": toAgent,
            "shared_at":    time.Now(),
        },
    }
    
    return sm.memory.Store(context.Background(), memoryEntry)
}
```

## 📊 代理監控和分析

### 執行統計

```go
// 獲取代理效能統計
type AgentStats struct {
    TotalExecutions    int           `json:"total_executions"`
    SuccessRate        float64       `json:"success_rate"`
    AverageSteps       float64       `json:"average_steps"`
    AverageExecutionTime time.Duration `json:"average_execution_time"`
    ToolUsageStats     map[string]int `json:"tool_usage_stats"`
    CapabilityStats    map[string]int `json:"capability_stats"`
}

func GetAgentStats(ctx context.Context, agentType agents.AgentType, timeRange time.Duration) (*AgentStats, error) {
    // 從資料庫查詢統計資料
    stats := &AgentStats{
        TotalExecutions: 150,
        SuccessRate:     0.94,
        AverageSteps:    3.2,
        AverageExecutionTime: 2500 * time.Millisecond,
        ToolUsageStats: map[string]int{
            "godev":  45,
            "search": 38, 
            "postgres": 22,
        },
        CapabilityStats: map[string]int{
            "code_analysis":        67,
            "performance_analysis": 28,
            "code_generation":      35,
        },
    }
    
    return stats, nil
}
```

### 執行追蹤

```go
// 詳細的執行追蹤
type ExecutionTrace struct {
    ExecutionID    string                 `json:"execution_id"`
    AgentType      agents.AgentType       `json:"agent_type"`
    StartTime      time.Time              `json:"start_time"`
    EndTime        time.Time              `json:"end_time"`
    Steps          []agents.AgentStep     `json:"steps"`
    ToolCalls      []ToolCall             `json:"tool_calls"`
    MemoryAccess   []MemoryAccess         `json:"memory_access"`
    LLMInteractions []LLMInteraction      `json:"llm_interactions"`
    ErrorDetails   *ErrorDetails          `json:"error_details,omitempty"`
}

type ToolCall struct {
    ToolName      string                 `json:"tool_name"`
    Input         map[string]interface{} `json:"input"`
    Output        interface{}            `json:"output"`
    ExecutionTime time.Duration          `json:"execution_time"`
    Success       bool                   `json:"success"`
}
```

## 🐛 除錯和故障排除

### 常見問題

1. **代理執行超時**
```go
// 解決方案：調整超時設定
config := config.LangChain{
    AgentTimeout: 5 * time.Minute,
    MaxSteps:     10,
}
```

2. **工具調用失敗**
```go
// 解決方案：檢查工具健康狀態
for _, tool := range agent.GetTools() {
    if err := tool.Health(ctx); err != nil {
        log.Printf("工具 %s 健康檢查失敗: %v", tool.Name(), err)
    }
}
```

3. **記憶存取錯誤**
```go
// 解決方案：驗證記憶配置
memoryStats := agent.GetMemoryStats()
if memoryStats.ErrorRate > 0.1 {
    log.Printf("記憶錯誤率過高: %.2f%%", memoryStats.ErrorRate*100)
}
```

### 除錯模式

```go
// 啟用詳細除錯
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

agent := agents.NewDevelopmentAgent(llm, config, logger)

// 啟用步驟追蹤
agent.EnableStepTracing(true)

// 啟用工具調用記錄
agent.EnableToolCallLogging(true)
```

## 📚 最佳實踐

### 1. 代理設計原則
- **單一職責**: 每個代理專注於特定領域
- **可組合性**: 代理應該能夠協作和組合
- **可擴展性**: 容易添加新能力和工具
- **可觀測性**: 提供充分的日誌和監控

### 2. 效能優化
- **並行執行**: 在可能的情況下並行執行步驟
- **快取結果**: 快取重複的工具調用結果
- **記憶管理**: 定期清理不需要的記憶項目
- **資源限制**: 設定合理的執行超時和步驟限制

### 3. 錯誤處理
- **優雅降級**: 工具失敗時提供替代方案
- **重試機制**: 對暫時性錯誤實施重試
- **錯誤分類**: 區分系統錯誤和用戶錯誤
- **詳細日誌**: 記錄足夠的上下文資訊

### 4. 安全考量
- **輸入驗證**: 驗證所有用戶輸入
- **權限控制**: 根據用戶角色限制代理能力
- **審計追蹤**: 記錄所有敏感操作
- **資料隱私**: 避免在日誌中洩露敏感資訊

## 🔗 相關資源

- [LangChain 主要文檔](../README.md)
- [Chains 鏈式處理](../chains/README.md)  
- [Memory 記憶管理](../memory/README.md)
- [Tools 工具系統](../../../tools/README.md)

## 🤝 貢獻指南

歡迎貢獻新的代理類型或改進現有代理：

1. **新代理類型**: 繼承 `BaseAgent` 並實現專業化邏輯
2. **新能力**: 為現有代理添加新的 `AgentCapability`
3. **工具整合**: 開發新工具並整合到代理中
4. **測試覆蓋**: 為新功能添加充分的測試
5. **文檔更新**: 更新相關文檔和範例

遵循專案的編碼標準和測試要求，確保向後相容性。