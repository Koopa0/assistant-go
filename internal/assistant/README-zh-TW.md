# 🤖 Assistant Package - 智慧助理核心套件

## 📋 概述

Assistant 套件是整個智慧開發助理系統的核心，負責協調 AI 服務、處理使用者互動，並管理整個對話生命週期。這個套件實現了智慧助理的主要邏輯，包括即時串流回應、上下文管理、錯誤處理等關鍵功能。

## 🏗️ 架構設計

### 核心組件

```go
// Assistant 是系統的主要入口點
type Assistant struct {
    ai           AIService          // AI 服務介面（支援 Claude、Gemini）
    toolRegistry *tools.Registry    // 工具註冊表
    processor    *Processor         // 訊息處理器
    stream       *StreamProcessor   // 串流處理器
    ctx          context.Context    // 上下文管理
    logger       *slog.Logger       // 結構化日誌
    config       *Config           // 配置管理
}
```

### 主要介面

```go
// Service 定義了助理服務的核心功能
type Service interface {
    // ProcessQuery 處理使用者查詢並返回回應
    ProcessQuery(ctx context.Context, req QueryRequest) (*QueryResponse, error)
    
    // ProcessStream 處理串流查詢，支援即時回應
    ProcessStream(ctx context.Context, req QueryRequest) (<-chan StreamChunk, error)
    
    // GetCapabilities 返回助理的能力描述
    GetCapabilities() Capabilities
}
```

## 🔧 核心功能

### 1. 查詢處理 (Query Processing)

Assistant 提供兩種查詢處理模式：

#### 同步模式
```go
// 適用於簡單查詢或不需要即時回饋的場景
response, err := assistant.ProcessQuery(ctx, QueryRequest{
    Query:   "解釋這段程式碼的功能",
    Context: map[string]interface{}{
        "code": codeSnippet,
        "language": "go",
    },
})
```

#### 串流模式
```go
// 適用於長回應或需要即時回饋的場景
stream, err := assistant.ProcessStream(ctx, QueryRequest{
    Query: "幫我重構這個複雜的函數",
})

for chunk := range stream {
    // 即時處理每個回應片段
    fmt.Print(chunk.Content)
}
```

### 2. 工具整合 (Tool Integration)

Assistant 可以動態調用各種開發工具：

```go
// 工具執行流程
1. 解析使用者意圖
2. 選擇適當的工具
3. 準備工具參數
4. 執行工具
5. 處理工具結果
6. 生成最終回應
```

支援的工具類型：
- **Go 開發工具**：程式碼分析、測試生成、重構建議
- **Docker 工具**：映像優化、容器管理
- **PostgreSQL 工具**：查詢優化、架構分析
- **基礎設施工具**：K8s 配置、CI/CD 設定

### 3. 上下文管理 (Context Management)

Assistant 維護豐富的上下文資訊：

```go
type Context struct {
    // 使用者上下文
    UserID       string
    Preferences  UserPreferences
    
    // 專案上下文  
    ProjectType  string
    Language     string
    Frameworks   []string
    
    // 對話上下文
    History      []Message
    CurrentTopic string
    
    // 系統上下文
    Timestamp    time.Time
    RequestID    string
}
```

### 4. 錯誤處理 (Error Handling)

採用分層錯誤處理策略：

```go
// 錯誤類型
type ErrorType string

const (
    ErrorTypeValidation   ErrorType = "validation"    // 輸入驗證錯誤
    ErrorTypeAI          ErrorType = "ai"            // AI 服務錯誤
    ErrorTypeTool        ErrorType = "tool"          // 工具執行錯誤
    ErrorTypeRateLimit   ErrorType = "rate_limit"   // 速率限制錯誤
    ErrorTypeInternal    ErrorType = "internal"     // 內部錯誤
)

// 智慧錯誤處理
func (a *Assistant) handleError(err error) error {
    switch e := err.(type) {
    case *AIError:
        // 嘗試使用備用 AI 提供者
        return a.fallbackToAlternativeAI()
    case *ToolError:
        // 提供替代建議
        return a.suggestAlternativeTool(e)
    default:
        // 記錄並包裝錯誤
        return fmt.Errorf("assistant error: %w", err)
    }
}
```

## 📊 進階功能

### 1. 處理器架構 (Processor Architecture)

Processor 負責訊息的預處理和後處理：

```go
type Processor struct {
    validators   []Validator    // 輸入驗證器
    enrichers    []Enricher     // 上下文豐富器
    transformers []Transformer  // 回應轉換器
    filters      []Filter       // 內容過濾器
}

// 處理流程
func (p *Processor) Process(msg Message) (ProcessedMessage, error) {
    // 1. 驗證輸入
    if err := p.validate(msg); err != nil {
        return ProcessedMessage{}, err
    }
    
    // 2. 豐富上下文
    enriched := p.enrich(msg)
    
    // 3. 執行主要處理
    result := p.execute(enriched)
    
    // 4. 轉換回應
    transformed := p.transform(result)
    
    // 5. 過濾敏感內容
    return p.filter(transformed), nil
}
```

### 2. 串流處理器 (Stream Processor)

處理即時串流回應的複雜邏輯：

```go
type StreamProcessor struct {
    bufferSize   int              // 緩衝區大小
    timeout      time.Duration    // 逾時設定
    interceptors []Interceptor    // 串流攔截器
}

// 串流處理特性
- 自動緩衝管理
- 錯誤恢復機制
- 進度追蹤
- 取消支援
- 背壓處理
```

### 3. 能力管理 (Capabilities Management)

動態管理和報告助理能力：

```go
type Capabilities struct {
    // AI 能力
    SupportedModels   []string
    MaxTokens         int
    StreamingSupport  bool
    
    // 工具能力
    AvailableTools    []ToolInfo
    ToolIntegrations  map[string]bool
    
    // 語言支援
    ProgrammingLangs  []string
    NaturalLangs      []string
    
    // 特殊功能
    Features          []Feature
}
```

## 🔍 使用範例

### 基本使用
```go
// 創建助理實例
assistant := assistant.New(
    assistant.WithAIService(aiService),
    assistant.WithTools(toolRegistry),
    assistant.WithLogger(logger),
)

// 處理查詢
response, err := assistant.ProcessQuery(ctx, QueryRequest{
    Query: "幫我優化這個 SQL 查詢",
    Context: map[string]interface{}{
        "sql": "SELECT * FROM users WHERE status = 'active'",
    },
})
```

### 進階使用
```go
// 使用串流處理複雜任務
stream, err := assistant.ProcessStream(ctx, QueryRequest{
    Query: "分析整個專案的程式碼品質並提供改進建議",
    Options: QueryOptions{
        IncludeCodeAnalysis: true,
        IncludeTestCoverage: true,
        IncludePerformance:  true,
    },
})

// 處理串流回應
for chunk := range stream {
    switch chunk.Type {
    case ChunkTypeProgress:
        fmt.Printf("進度: %s\n", chunk.Progress)
    case ChunkTypeContent:
        fmt.Print(chunk.Content)
    case ChunkTypeError:
        log.Error("串流錯誤", "error", chunk.Error)
    }
}
```

## 🧪 測試策略

### 單元測試
```go
// 測試基本功能
func TestAssistant_ProcessQuery(t *testing.T) {
    // 使用模擬 AI 服務
    mockAI := NewMockAIService()
    assistant := assistant.New(
        assistant.WithAIService(mockAI),
    )
    
    // 測試各種場景
    testCases := []struct {
        name     string
        query    QueryRequest
        expected QueryResponse
    }{
        // ... 測試案例
    }
}
```

### 整合測試
```go
// 測試完整工作流程
func TestAssistant_Integration(t *testing.T) {
    // 使用真實服務但模擬外部依賴
    assistant := setupTestAssistant(t)
    
    // 測試複雜互動
    // ... 整合測試邏輯
}
```

## 🔧 配置選項

```yaml
assistant:
  # AI 配置
  ai:
    primary_provider: claude
    fallback_provider: gemini
    max_retries: 3
    timeout: 30s
    
  # 處理器配置
  processor:
    max_context_size: 8192
    enable_caching: true
    cache_ttl: 1h
    
  # 串流配置
  streaming:
    buffer_size: 1024
    chunk_size: 256
    enable_compression: true
    
  # 安全配置
  security:
    enable_content_filtering: true
    sensitive_data_detection: true
    audit_logging: true
```

## 📈 效能考量

1. **記憶體管理**
   - 使用物件池減少 GC 壓力
   - 限制上下文大小防止記憶體溢出
   - 定期清理過期快取

2. **並發處理**
   - 使用 goroutine 池控制並發數
   - 實現優雅關閉機制
   - 避免 goroutine 洩漏

3. **錯誤恢復**
   - 實現斷路器模式
   - 自動重試機制
   - 降級策略

## 🚀 未來規劃

1. **智慧增強**
   - 實現自適應學習
   - 個性化回應風格
   - 預測性協助

2. **工具生態**
   - 支援更多開發工具
   - 外掛系統
   - 自訂工具開發框架

3. **效能優化**
   - 實現智慧快取
   - 分散式處理
   - 邊緣計算支援

## 📚 相關文件

- [AI Package README](../ai/README-zh-TW.md) - AI 服務整合
- [Tools Package README](../tools/README-zh-TW.md) - 工具系統
- [Memory Package README](../memory/README-zh-TW.md) - 記憶系統
- [主要架構文件](../../CLAUDE-ARCHITECTURE.md) - 系統架構指南