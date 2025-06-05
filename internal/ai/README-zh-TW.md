# AI 提供者管理系統

## 概述

AI 提供者管理系統為與多個 AI 服務提供者（包括 Claude (Anthropic) 和 Gemini (Google)）互動提供統一介面。此套件實作了工廠模式，具備完善的錯誤處理、速率限制和可觀測性功能。

## 架構

```
internal/ai/
├── provider.go          # 提供者介面和類型
├── factory.go          # 提供者工廠實作
├── init.go             # 初始化和配置
├── claude/
│   └── client.go       # Claude API 客戶端實作
├── gemini/
│   └── client.go       # Gemini API 客戶端實作
└── embeddings/
    ├── service.go      # 嵌入向量生成服務
    └── service_test.go # 完整測試套件
```

## 主要功能

### 🤖 **多提供者支援**
- **Claude 整合**：完整的 Anthropic Claude API 支援，包含串流功能
- **Gemini 整合**：Google Gemini API 與進階模型配置
- **統一介面**：跨所有提供者的一致 API
- **提供者選擇**：基於可用性和成本的動態提供者選擇

### 🛡️ **生產就緒功能**
- **速率限制**：智慧速率限制與指數退避
- **錯誤處理**：包含上下文的完整錯誤包裝
- **健康檢查**：提供者健康監控和故障轉移
- **Token 追蹤**：使用量監控和成本優化
- **重試邏輯**：可配置的重試策略與斷路器

### 📊 **可觀測性**
- **指標收集**：請求延遲、token 使用量、錯誤率
- **結構化日誌**：詳細的請求/回應日誌與上下文
- **效能追蹤**：回應時間分析和優化
- **成本監控**：Token 使用量和 API 成本追蹤

## 核心元件

### 提供者介面

```go
type Provider interface {
    // 帶上下文的聊天完成
    ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    
    // 串流聊天完成
    ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error)
    
    // 提供者元資料和功能
    GetCapabilities() Capabilities
    GetName() string
    GetHealth() HealthStatus
    
    // 資源管理
    Close() error
}
```

### 工廠模式

```go
type Factory struct {
    providers map[string]Provider
    config    *Config
    metrics   *Metrics
    logger    *slog.Logger
}

func NewFactory(config *Config) (*Factory, error)
func (f *Factory) GetProvider(name string) (Provider, error)
func (f *Factory) GetBestProvider(criteria Criteria) (Provider, error)
```

### 提供者類型

- **Claude 提供者**：Anthropic Claude API 整合
- **Gemini 提供者**：Google Gemini API 整合
- **Mock 提供者**：測試和開發支援

## 配置

### 環境變數

```bash
# Claude 配置
CLAUDE_API_KEY=your_claude_api_key
CLAUDE_MODEL=claude-3-sonnet-20240229
CLAUDE_MAX_TOKENS=4096

# Gemini 配置
GEMINI_API_KEY=your_gemini_api_key
GEMINI_MODEL=gemini-pro
GEMINI_TEMPERATURE=0.7

# 速率限制
AI_RATE_LIMIT_PER_MINUTE=60
AI_RATE_LIMIT_BURST=10
AI_REQUEST_TIMEOUT=30s
```

### YAML 配置

```yaml
ai:
  providers:
    claude:
      enabled: true
      model: "claude-3-sonnet-20240229"
      max_tokens: 4096
      temperature: 0.0
      rate_limit:
        requests_per_minute: 60
        burst: 10
    gemini:
      enabled: true
      model: "gemini-pro"
      temperature: 0.7
      safety_settings:
        harassment: "BLOCK_MEDIUM_AND_ABOVE"
        hate_speech: "BLOCK_MEDIUM_AND_ABOVE"
  retry:
    max_attempts: 3
    initial_delay: "1s"
    max_delay: "30s"
    multiplier: 2.0
```

## 使用範例

### 基本聊天完成

```go
// 初始化工廠
factory, err := ai.NewFactory(config)
if err != nil {
    return fmt.Errorf("建立 AI 工廠失敗: %w", err)
}

// 取得提供者
provider, err := factory.GetProvider("claude")
if err != nil {
    return fmt.Errorf("取得提供者失敗: %w", err)
}

// 聊天完成
response, err := provider.ChatCompletion(ctx, ai.ChatRequest{
    Messages: []ai.Message{
        {Role: "user", Content: "解釋量子計算"},
    },
    MaxTokens:   1000,
    Temperature: 0.7,
})
if err != nil {
    return fmt.Errorf("聊天完成失敗: %w", err)
}

fmt.Println(response.Content)
```

### 串流聊天

```go
// 串流完成
stream, err := provider.ChatCompletionStream(ctx, request)
if err != nil {
    return fmt.Errorf("啟動串流失敗: %w", err)
}

for response := range stream {
    if response.Error != nil {
        log.Printf("串流錯誤: %v", response.Error)
        continue
    }
    fmt.Print(response.Delta) // 印出增量內容
}
```

### 提供者選擇

```go
// 根據條件取得最佳提供者
provider, err := factory.GetBestProvider(ai.Criteria{
    ModelType:     ai.ModelTypeChat,
    MaxLatency:    time.Second * 5,
    MaxCostPerToken: 0.001,
    RequiredFeatures: []string{"streaming", "function_calling"},
})
```

### 嵌入向量生成

```go
// 初始化嵌入服務
embeddingService, err := embeddings.NewService(config, factory)
if err != nil {
    return fmt.Errorf("建立嵌入服務失敗: %w", err)
}

// 生成嵌入向量
vectors, err := embeddingService.GenerateEmbeddings(ctx, []string{
    "機器學習正在改變軟體開發",
    "Go 是建構可擴展系統的優秀語言",
})
if err != nil {
    return fmt.Errorf("生成嵌入向量失敗: %w", err)
}

// 儲存至向量資料庫
err = embeddingService.Store(ctx, vectors)
```

## 錯誤處理

### 錯誤類型

```go
type ProviderError struct {
    Provider string
    Code     ErrorCode
    Message  string
    Retryable bool
    Cause    error
}

const (
    ErrCodeRateLimit     ErrorCode = "RATE_LIMIT"
    ErrCodeInvalidRequest ErrorCode = "INVALID_REQUEST"
    ErrCodeProviderDown  ErrorCode = "PROVIDER_DOWN"
    ErrCodeTokenLimit    ErrorCode = "TOKEN_LIMIT"
    ErrCodeTimeout       ErrorCode = "TIMEOUT"
)
```

### 重試策略

```go
func (c *ClaudeClient) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    return retry.Do(ctx, func() (*ChatResponse, error) {
        return c.doRequest(ctx, req)
    }, retry.Config{
        MaxAttempts: 3,
        Delay:       time.Second,
        Multiplier:  2.0,
        ShouldRetry: func(err error) bool {
            if providerErr, ok := err.(*ProviderError); ok {
                return providerErr.Retryable
            }
            return false
        },
    })
}
```

## 測試

### 測試覆蓋率

- **單元測試**：提供者實作、工廠邏輯
- **整合測試**：真實 API 互動（使用測試金鑰）
- **效能測試**：延遲和吞吐量基準測試
- **錯誤測試**：失敗場景和復原

### Mock 提供者

```go
func TestWithMockProvider(t *testing.T) {
    factory := ai.NewFactory(&ai.Config{
        Providers: map[string]ai.ProviderConfig{
            "mock": {
                Type:    "mock",
                Enabled: true,
            },
        },
    })
    
    provider, err := factory.GetProvider("mock")
    require.NoError(t, err)
    
    response, err := provider.ChatCompletion(ctx, request)
    assert.NoError(t, err)
    assert.Contains(t, response.Content, "mock response")
}
```

## 效能優化

### 快取策略

```go
type CachedProvider struct {
    provider Provider
    cache    cache.Cache
    ttl      time.Duration
}

func (cp *CachedProvider) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    key := generateCacheKey(req)
    
    if cached, found := cp.cache.Get(key); found {
        return cached.(*ChatResponse), nil
    }
    
    response, err := cp.provider.ChatCompletion(ctx, req)
    if err != nil {
        return nil, err
    }
    
    cp.cache.Set(key, response, cp.ttl)
    return response, nil
}
```

### 連線池

```go
type ClientPool struct {
    clients chan *http.Client
    factory func() *http.Client
}

func NewClientPool(size int) *ClientPool {
    pool := &ClientPool{
        clients: make(chan *http.Client, size),
        factory: createHTTPClient,
    }
    
    for i := 0; i < size; i++ {
        pool.clients <- pool.factory()
    }
    
    return pool
}
```

## 監控和指標

### 關鍵指標

- **請求延遲**：P50、P95、P99 回應時間
- **Token 使用量**：輸入/輸出 tokens、成本追蹤
- **錯誤率**：按類型和提供者的錯誤頻率
- **提供者健康**：可用性和效能分數

### Prometheus 指標

```go
var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "ai_request_duration_seconds",
            Help: "AI 請求持續時間（秒）",
        },
        []string{"provider", "model", "operation"},
    )
    
    tokenUsage = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ai_tokens_total",
            Help: "AI tokens 總消耗量",
        },
        []string{"provider", "model", "type"},
    )
)
```

## 安全性

### API 金鑰管理

- **環境變數**：安全的金鑰儲存
- **金鑰輪換**：支援優雅的金鑰更新
- **存取控制**：提供者特定權限
- **稽核日誌**：請求/回應追蹤

### 資料隱私

- **請求清理**：記錄前移除敏感資料
- **回應過濾**：遮罩或編輯敏感輸出
- **本地處理**：盡可能保持敏感資料在本地
- **合規性**：GDPR、SOX 和產業標準合規

## 未來增強功能

### 計劃中的功能

1. **額外提供者**：OpenAI GPT-4、Azure OpenAI、AWS Bedrock
2. **函數呼叫**：結構化函數/工具呼叫支援
3. **多模態**：圖像和音訊處理能力
4. **微調**：自訂模型訓練和部署
5. **邊緣部署**：本地模型推理支援

### 整合路線圖

- **LangChain 整合**：增強的鏈和代理支援
- **向量資料庫**：改進的嵌入和檢索
- **模型上下文協議**：當可用時支援 MCP
- **Kubernetes Operator**：可擴展的部署管理

## 貢獻

為 AI 提供者系統貢獻時：

1. **遵循介面**：完整實作 Provider 介面
2. **添加測試**：包括單元、整合和效能測試
3. **處理錯誤**：適當的錯誤包裝和重試邏輯
4. **監控資源**：追蹤 token 使用量和 API 成本
5. **文件更新**：更新 README 和程式碼文件

## 相關文件

- [助理核心](../assistant/README.md) - 主要協調系統
- [嵌入服務](embeddings/README.md) - 向量生成和搜尋
- [配置](../config/README.md) - 系統配置管理
- [可觀測性](../observability/README.md) - 監控和日誌