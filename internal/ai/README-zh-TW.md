# 🧠 AI Package - 人工智慧服務整合套件

## 📋 概述

AI 套件是 Assistant 系統的核心智慧引擎，負責整合多個 AI 提供者（Claude、Gemini），提供統一的介面來處理自然語言理解、程式碼分析、智慧建議等功能。這個套件實現了提供者抽象、錯誤處理、串流支援、嵌入向量等關鍵功能。

## 🏗️ 架構設計

### 核心介面

```go
// Service 定義了 AI 服務的統一介面
type Service interface {
    // Chat 發送聊天請求並返回完整回應
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    
    // ChatStream 發送聊天請求並返回串流回應
    ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
    
    // GenerateEmbedding 生成文字的向量嵌入
    GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
    
    // GetCapabilities 返回當前 AI 提供者的能力
    GetCapabilities() Capabilities
}
```

### 提供者架構

```go
// 支援的 AI 提供者
type Provider string

const (
    ProviderClaude Provider = "claude"  // Anthropic Claude
    ProviderGemini Provider = "gemini"  // Google Gemini
)

// 提供者管理
type ProviderManager struct {
    primary   Provider           // 主要提供者
    fallback  Provider           // 備用提供者
    providers map[Provider]Service // 提供者實例
    health    map[Provider]bool    // 健康狀態
}
```

## 🔧 核心功能

### 1. 多提供者支援 (Multi-Provider Support)

#### Claude 整合
```go
// Claude 客戶端配置
type ClaudeClient struct {
    apiKey     string
    model      string           // claude-3-opus, claude-3-sonnet 等
    maxTokens  int
    httpClient *http.Client
}

// 特性支援
- 超長上下文窗口（200K tokens）
- 優秀的程式碼理解能力
- 支援系統提示詞
- 即時串流回應
```

#### Gemini 整合
```go
// Gemini 客戶端配置
type GeminiClient struct {
    apiKey        string
    model         string         // gemini-pro, gemini-pro-vision 等
    generationConfig *genai.GenerationConfig
    client        *genai.Client
}

// 特性支援
- 多模態支援（文字、圖片）
- 函數調用能力
- 高速回應
- 成本效益
```

### 2. 智慧路由 (Intelligent Routing)

```go
// 根據請求特性選擇最佳提供者
func (m *ProviderManager) RouteRequest(req ChatRequest) Provider {
    // 考慮因素：
    // 1. 請求複雜度
    if req.RequiresDeepAnalysis() {
        return ProviderClaude // Claude 更擅長深度分析
    }
    
    // 2. 成本考量
    if req.Priority == PriorityLow {
        return ProviderGemini // Gemini 成本更低
    }
    
    // 3. 可用性
    if !m.health[m.primary] {
        return m.fallback // 使用備用提供者
    }
    
    // 4. 特殊功能需求
    if req.RequiresVision() {
        return ProviderGemini // Gemini 支援視覺
    }
    
    return m.primary
}
```

### 3. 串流處理 (Stream Processing)

```go
// 統一的串流事件
type StreamEvent struct {
    Type      StreamEventType
    Content   string          // 文字內容
    Delta     string          // 增量內容
    Error     error           // 錯誤資訊
    Metadata  map[string]any  // 元資料
    Timestamp time.Time
}

// 串流處理器
type StreamProcessor struct {
    // 緩衝管理
    buffer     *bytes.Buffer
    bufferSize int
    
    // 錯誤恢復
    retryCount int
    maxRetries int
    
    // 進度追蹤
    tokenCount int
    startTime  time.Time
}
```

### 4. 嵌入向量服務 (Embeddings Service)

```go
// 嵌入向量服務介面
type EmbeddingService interface {
    // 生成單一文字的嵌入向量
    GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
    
    // 批量生成嵌入向量
    GenerateEmbeddings(ctx context.Context, texts []string) ([][]float64, error)
    
    // 計算向量相似度
    CosineSimilarity(a, b []float64) float64
}

// 實現細節
type embeddingService struct {
    provider   string          // 使用的嵌入模型
    dimension  int            // 向量維度
    cache      *lru.Cache     // LRU 快取
    batchSize  int            // 批次大小
}
```

## 📊 進階功能

### 1. 提示詞工程 (Prompt Engineering)

```go
// 提示詞模板系統
type PromptTemplate struct {
    Name        string
    Template    string
    Variables   []string
    Constraints []string
    Examples    []Example
}

// 動態提示詞生成
func (s *promptService) GeneratePrompt(
    template PromptTemplate,
    context map[string]interface{},
) (string, error) {
    // 1. 驗證必要變數
    if err := s.validateVariables(template, context); err != nil {
        return "", err
    }
    
    // 2. 注入上下文
    prompt := s.injectContext(template.Template, context)
    
    // 3. 添加約束條件
    prompt = s.applyConstraints(prompt, template.Constraints)
    
    // 4. 添加範例（Few-shot learning）
    if len(template.Examples) > 0 {
        prompt = s.addExamples(prompt, template.Examples)
    }
    
    return prompt, nil
}
```

### 2. Token 管理 (Token Management)

```go
// Token 計數器
type TokenCounter interface {
    // 計算文字的 token 數量
    Count(text string) int
    
    // 估算回應所需的 token
    EstimateResponseTokens(prompt string) int
    
    // 截斷文字到指定 token 數
    Truncate(text string, maxTokens int) string
}

// 智慧 token 優化
type TokenOptimizer struct {
    maxContextTokens int
    reserveTokens    int  // 為回應保留的 token
    
    // 優化策略
    strategies []OptimizationStrategy
}

// 優化策略範例
func (o *TokenOptimizer) Optimize(messages []Message) []Message {
    // 1. 移除冗餘內容
    messages = o.removeRedundancy(messages)
    
    // 2. 摘要長對話
    messages = o.summarizeOldMessages(messages)
    
    // 3. 壓縮系統提示詞
    messages = o.compressSystemPrompt(messages)
    
    return messages
}
```

### 3. 錯誤處理與恢復 (Error Handling & Recovery)

```go
// AI 錯誤類型
type AIError struct {
    Type        ErrorType
    Provider    Provider
    StatusCode  int
    Message     string
    Retryable   bool
    RetryAfter  time.Duration
}

// 錯誤恢復策略
type RecoveryStrategy struct {
    // 重試策略
    MaxRetries     int
    BackoffFactor  float64
    
    // 降級策略
    FallbackProvider Provider
    SimplifyRequest  bool
    
    // 斷路器
    CircuitBreaker *CircuitBreaker
}

// 智慧錯誤處理
func (s *Service) HandleError(err error, strategy RecoveryStrategy) error {
    switch e := err.(type) {
    case *RateLimitError:
        // 等待後重試
        time.Sleep(e.RetryAfter)
        return s.retryWithBackoff(strategy)
        
    case *ContextLengthError:
        // 優化上下文後重試
        return s.retryWithOptimizedContext()
        
    case *ProviderError:
        // 切換到備用提供者
        return s.fallbackToAlternative(strategy.FallbackProvider)
        
    default:
        return fmt.Errorf("unrecoverable AI error: %w", err)
    }
}
```

## 🔍 使用範例

### 基本聊天
```go
// 創建 AI 服務
aiService := ai.NewService(
    ai.WithProvider(ai.ProviderClaude),
    ai.WithAPIKey(apiKey),
    ai.WithModel("claude-3-opus-20240229"),
)

// 發送聊天請求
response, err := aiService.Chat(ctx, ai.ChatRequest{
    Messages: []ai.Message{
        {
            Role:    ai.RoleUser,
            Content: "解釋 Go 的 interface 概念",
        },
    },
    MaxTokens:   1000,
    Temperature: 0.7,
})
```

### 串流回應
```go
// 創建串流請求
stream, err := aiService.ChatStream(ctx, ai.ChatRequest{
    Messages: messages,
    Options: ai.StreamOptions{
        ChunkSize:      100,
        IncludeUsage:   true,
        StopOnError:    false,
    },
})

// 處理串流事件
for event := range stream {
    switch event.Type {
    case ai.StreamEventTypeContent:
        fmt.Print(event.Content)
    case ai.StreamEventTypeError:
        log.Error("串流錯誤", "error", event.Error)
    case ai.StreamEventTypeEnd:
        fmt.Printf("\n使用 tokens: %d\n", event.Metadata["usage"])
    }
}
```

### 嵌入向量生成
```go
// 創建嵌入服務
embedService := ai.NewEmbeddingService(
    ai.WithEmbeddingModel("text-embedding-ada-002"),
    ai.WithBatchSize(100),
)

// 生成向量
embeddings, err := embedService.GenerateEmbeddings(ctx, []string{
    "Go 是一個靜態類型的編譯語言",
    "Python 是一個動態類型的解釋語言",
    "Rust 注重記憶體安全",
})

// 計算相似度
similarity := embedService.CosineSimilarity(embeddings[0], embeddings[1])
fmt.Printf("Go 與 Python 的相似度: %.2f\n", similarity)
```

## 🧪 測試策略

### 單元測試
```go
func TestAIService_Chat(t *testing.T) {
    // 使用模擬客戶端
    mockClient := NewMockAIClient()
    service := ai.NewService(ai.WithClient(mockClient))
    
    // 測試正常情況
    t.Run("successful chat", func(t *testing.T) {
        mockClient.SetResponse(&ai.ChatResponse{
            Content: "測試回應",
            Usage:   ai.Usage{TotalTokens: 100},
        })
        
        response, err := service.Chat(ctx, testRequest)
        require.NoError(t, err)
        assert.Equal(t, "測試回應", response.Content)
    })
    
    // 測試錯誤處理
    t.Run("rate limit error", func(t *testing.T) {
        mockClient.SetError(&ai.RateLimitError{
            RetryAfter: 60 * time.Second,
        })
        
        _, err := service.Chat(ctx, testRequest)
        assert.ErrorIs(t, err, ai.ErrRateLimit)
    })
}
```

## 🔧 配置選項

```yaml
ai:
  # 提供者配置
  providers:
    claude:
      api_key: ${CLAUDE_API_KEY}
      model: claude-3-opus-20240229
      max_tokens: 4096
      temperature: 0.7
      
    gemini:
      api_key: ${GEMINI_API_KEY}
      model: gemini-pro
      max_tokens: 8192
      temperature: 0.8
  
  # 路由配置
  routing:
    primary_provider: claude
    fallback_provider: gemini
    health_check_interval: 30s
    
  # 串流配置
  streaming:
    buffer_size: 4096
    chunk_timeout: 100ms
    enable_compression: true
    
  # Token 管理
  tokens:
    max_context_tokens: 100000
    reserve_response_tokens: 2000
    enable_optimization: true
    
  # 嵌入向量
  embeddings:
    provider: openai
    model: text-embedding-ada-002
    cache_size: 10000
    batch_size: 100
```

## 📈 效能優化

1. **快取策略**
   - LRU 快取頻繁請求
   - 嵌入向量快取
   - 提示詞模板快取

2. **批次處理**
   - 批量嵌入向量生成
   - 請求合併優化
   - 非同步處理佇列

3. **資源管理**
   - 連線池管理
   - Token 預算控制
   - 記憶體使用優化

## 🚀 未來規劃

1. **更多 AI 提供者**
   - OpenAI GPT-4 整合
   - Llama 本地模型支援
   - 專業領域模型

2. **進階功能**
   - 多模態支援（圖片、音訊）
   - 函數調用（Function Calling）
   - 智慧對話管理

3. **優化增強**
   - 分散式推理
   - 邊緣部署支援
   - 自適應模型選擇

## 📚 相關文件

- [Assistant Package README](../assistant/README-zh-TW.md) - 助理核心套件
- [Embeddings 子套件](./embeddings/README.md) - 嵌入向量詳細說明
- [Prompts 子套件](./prompts/README.md) - 提示詞工程指南
- [主要架構文件](../../CLAUDE-ARCHITECTURE.md) - 系統架構指南