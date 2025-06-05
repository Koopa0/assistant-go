# AI Provider Management

## Overview

The AI provider management system provides a unified interface for interacting with multiple AI service providers including Claude (Anthropic) and Gemini (Google). This package implements a factory pattern with comprehensive error handling, rate limiting, and observability features.

## Architecture

```
internal/ai/
├── provider.go          # Provider interface and types
├── factory.go          # Provider factory implementation
├── init.go             # Initialization and configuration
├── claude/
│   └── client.go       # Claude API client implementation
├── gemini/
│   └── client.go       # Gemini API client implementation
└── embeddings/
    ├── service.go      # Embedding generation service
    └── service_test.go # Comprehensive test suite
```

## Key Features

### 🤖 **Multi-Provider Support**
- **Claude Integration**: Complete Anthropic Claude API support with streaming
- **Gemini Integration**: Google Gemini API with advanced model configurations
- **Unified Interface**: Consistent API across all providers
- **Provider Selection**: Dynamic provider selection based on availability and cost

### 🛡️ **Production-Ready Features**
- **Rate Limiting**: Intelligent rate limiting with exponential backoff
- **Error Handling**: Comprehensive error wrapping with context
- **Health Checks**: Provider health monitoring and failover
- **Token Tracking**: Usage monitoring and cost optimization
- **Retry Logic**: Configurable retry policies with circuit breaker

### 📊 **Observability**
- **Metrics Collection**: Request latency, token usage, error rates
- **Structured Logging**: Detailed request/response logging with context
- **Performance Tracking**: Response time analysis and optimization
- **Cost Monitoring**: Token usage and API cost tracking

## Core Components

### Provider Interface

```go
type Provider interface {
    // Chat completion with context
    ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    
    // Streaming chat completion
    ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error)
    
    // Provider metadata and capabilities
    GetCapabilities() Capabilities
    GetName() string
    GetHealth() HealthStatus
    
    // Resource management
    Close() error
}
```

### Factory Pattern

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

### Provider Types

- **Claude Provider**: Anthropic Claude API integration
- **Gemini Provider**: Google Gemini API integration
- **Mock Provider**: Testing and development support

## Configuration

### Environment Variables

```bash
# Claude Configuration
CLAUDE_API_KEY=your_claude_api_key
CLAUDE_MODEL=claude-3-sonnet-20240229
CLAUDE_MAX_TOKENS=4096

# Gemini Configuration
GEMINI_API_KEY=your_gemini_api_key
GEMINI_MODEL=gemini-pro
GEMINI_TEMPERATURE=0.7

# Rate Limiting
AI_RATE_LIMIT_PER_MINUTE=60
AI_RATE_LIMIT_BURST=10
AI_REQUEST_TIMEOUT=30s
```

### YAML Configuration

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

## Usage Examples

### Basic Chat Completion

```go
// Initialize factory
factory, err := ai.NewFactory(config)
if err != nil {
    return fmt.Errorf("failed to create AI factory: %w", err)
}

// Get provider
provider, err := factory.GetProvider("claude")
if err != nil {
    return fmt.Errorf("failed to get provider: %w", err)
}

// Chat completion
response, err := provider.ChatCompletion(ctx, ai.ChatRequest{
    Messages: []ai.Message{
        {Role: "user", Content: "Explain quantum computing"},
    },
    MaxTokens:   1000,
    Temperature: 0.7,
})
if err != nil {
    return fmt.Errorf("chat completion failed: %w", err)
}

fmt.Println(response.Content)
```

### Streaming Chat

```go
// Streaming completion
stream, err := provider.ChatCompletionStream(ctx, request)
if err != nil {
    return fmt.Errorf("failed to start stream: %w", err)
}

for response := range stream {
    if response.Error != nil {
        log.Printf("Stream error: %v", response.Error)
        continue
    }
    fmt.Print(response.Delta) // Print incremental content
}
```

### Provider Selection

```go
// Get best provider based on criteria
provider, err := factory.GetBestProvider(ai.Criteria{
    ModelType:     ai.ModelTypeChat,
    MaxLatency:    time.Second * 5,
    MaxCostPerToken: 0.001,
    RequiredFeatures: []string{"streaming", "function_calling"},
})
```

### Embeddings Generation

```go
// Initialize embedding service
embeddingService, err := embeddings.NewService(config, factory)
if err != nil {
    return fmt.Errorf("failed to create embedding service: %w", err)
}

// Generate embeddings
vectors, err := embeddingService.GenerateEmbeddings(ctx, []string{
    "Machine learning is transforming software development",
    "Go is an excellent language for building scalable systems",
})
if err != nil {
    return fmt.Errorf("failed to generate embeddings: %w", err)
}

// Store in vector database
err = embeddingService.Store(ctx, vectors)
```

## Error Handling

### Error Types

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

### Retry Strategy

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

## Testing

### Test Coverage

- **Unit Tests**: Provider implementations, factory logic
- **Integration Tests**: Real API interactions (with test keys)
- **Performance Tests**: Latency and throughput benchmarks
- **Error Tests**: Failure scenarios and recovery

### Mock Provider

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

## Performance Optimization

### Caching Strategy

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

### Connection Pooling

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

## Monitoring and Metrics

### Key Metrics

- **Request Latency**: P50, P95, P99 response times
- **Token Usage**: Input/output tokens, cost tracking
- **Error Rate**: Error frequency by type and provider
- **Provider Health**: Availability and performance scores

### Prometheus Metrics

```go
var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "ai_request_duration_seconds",
            Help: "AI request duration in seconds",
        },
        []string{"provider", "model", "operation"},
    )
    
    tokenUsage = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ai_tokens_total",
            Help: "Total AI tokens consumed",
        },
        []string{"provider", "model", "type"},
    )
)
```

## Security

### API Key Management

- **Environment Variables**: Secure key storage
- **Key Rotation**: Support for graceful key updates
- **Access Control**: Provider-specific permissions
- **Audit Logging**: Request/response tracking

### Data Privacy

- **Request Sanitization**: Remove sensitive data before logging
- **Response Filtering**: Mask or redact sensitive outputs
- **Local Processing**: Keep sensitive data local when possible
- **Compliance**: GDPR, SOX, and industry standard compliance

## Future Enhancements

### Planned Features

1. **Additional Providers**: OpenAI GPT-4, Azure OpenAI, AWS Bedrock
2. **Function Calling**: Structured function/tool calling support
3. **Multi-Modal**: Image and audio processing capabilities
4. **Fine-Tuning**: Custom model training and deployment
5. **Edge Deployment**: Local model inference support

### Integration Roadmap

- **LangChain Integration**: Enhanced chain and agent support
- **Vector Database**: Improved embedding and retrieval
- **Model Context Protocol**: MCP support when available
- **Kubernetes Operator**: Scalable deployment management

## Contributing

When contributing to the AI provider system:

1. **Follow Interfaces**: Implement the Provider interface completely
2. **Add Tests**: Include unit, integration, and performance tests
3. **Handle Errors**: Proper error wrapping and retry logic
4. **Monitor Resources**: Track token usage and API costs
5. **Document Changes**: Update README and code documentation

## Related Documentation

- [Assistant Core](../assistant/README.md) - Main orchestration system
- [Embeddings Service](embeddings/README.md) - Vector generation and search
- [Configuration](../config/README.md) - System configuration management
- [Observability](../observability/README.md) - Monitoring and logging