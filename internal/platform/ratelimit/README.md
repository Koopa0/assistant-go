# Rate Limiting

## Overview

The Rate Limiting package provides a flexible and efficient rate limiting system for the Assistant intelligent development companion. It implements multiple algorithms including token bucket, sliding window, and distributed rate limiting for protecting API endpoints, managing resource usage, and ensuring fair access to system resources.

## Architecture

```
internal/ratelimit/
└── limiter.go  # Core rate limiter implementations
```

## Key Features

### 🚦 **Multiple Algorithms**
- **Token Bucket**: Smooth rate limiting with burst capacity
- **Sliding Window**: Precise rate limiting over time windows
- **Fixed Window**: Simple time-based rate limiting
- **Leaky Bucket**: Constant rate processing

### 🌍 **Distributed Support**
- **Redis Backend**: Shared state across instances
- **Consistent Limiting**: Cluster-wide rate enforcement
- **Fault Tolerance**: Fallback to local limiting
- **High Performance**: Optimized for low latency

### 🎯 **Flexible Configuration**
- **Per-User Limits**: Individual user rate limits
- **Per-Endpoint Limits**: API endpoint specific limits
- **Dynamic Adjustment**: Runtime limit modifications
- **Custom Keys**: Flexible rate limit key generation

## Core Components

### Rate Limiter Interface

```go
type RateLimiter interface {
    // Check if request is allowed
    Allow(ctx context.Context, key string) (bool, error)
    
    // Check with custom cost
    AllowN(ctx context.Context, key string, n int) (bool, error)
    
    // Get current limit info
    GetInfo(key string) (*LimitInfo, error)
    
    // Reset limit for key
    Reset(key string) error
    
    // Close and cleanup
    Close() error
}

type LimitInfo struct {
    Limit      int64     // Total limit
    Remaining  int64     // Remaining in current window
    ResetAt    time.Time // When the limit resets
    RetryAfter int64     // Seconds until next allowed request
}

type RateLimiterConfig struct {
    // Algorithm selection
    Algorithm  Algorithm
    
    // Basic settings
    Rate       int           // Requests per interval
    Interval   time.Duration // Time interval
    Burst      int           // Burst capacity
    
    // Storage backend
    Backend    Backend
    RedisURL   string
    
    // Options
    KeyPrefix  string
    DefaultTTL time.Duration
}
```

### Token Bucket Implementation

```go
type TokenBucketLimiter struct {
    rate     float64
    burst    int
    mu       sync.Mutex
    backends map[string]*tokenBucket
    store    Store
    config   TokenBucketConfig
}

type tokenBucket struct {
    tokens    float64
    lastTime  time.Time
    mu        sync.Mutex
}

func NewTokenBucketLimiter(config TokenBucketConfig) *TokenBucketLimiter {
    return &TokenBucketLimiter{
        rate:     float64(config.Rate) / config.Interval.Seconds(),
        burst:    config.Burst,
        backends: make(map[string]*tokenBucket),
        store:    config.Store,
        config:   config,
    }
}

func (l *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
    return l.AllowN(ctx, key, 1)
}

func (l *TokenBucketLimiter) AllowN(ctx context.Context, key string, n int) (bool, error) {
    // Try distributed limiter first
    if l.store != nil {
        allowed, err := l.allowDistributed(ctx, key, n)
        if err == nil {
            return allowed, nil
        }
        // Fall back to local on error
        l.logError("Distributed limiter failed, using local", err)
    }
    
    // Local rate limiting
    return l.allowLocal(key, n), nil
}

func (l *TokenBucketLimiter) allowLocal(key string, n int) bool {
    l.mu.Lock()
    bucket, exists := l.backends[key]
    if !exists {
        bucket = &tokenBucket{
            tokens:   float64(l.burst),
            lastTime: time.Now(),
        }
        l.backends[key] = bucket
    }
    l.mu.Unlock()
    
    bucket.mu.Lock()
    defer bucket.mu.Unlock()
    
    now := time.Now()
    elapsed := now.Sub(bucket.lastTime).Seconds()
    bucket.lastTime = now
    
    // Add tokens based on elapsed time
    bucket.tokens += elapsed * l.rate
    if bucket.tokens > float64(l.burst) {
        bucket.tokens = float64(l.burst)
    }
    
    // Check if we have enough tokens
    if bucket.tokens >= float64(n) {
        bucket.tokens -= float64(n)
        return true
    }
    
    return false
}

func (l *TokenBucketLimiter) allowDistributed(ctx context.Context, key string, n int) (bool, error) {
    script := `
        local key = KEYS[1]
        local rate = tonumber(ARGV[1])
        local burst = tonumber(ARGV[2])
        local now = tonumber(ARGV[3])
        local requested = tonumber(ARGV[4])
        
        local info = redis.call('HMGET', key, 'tokens', 'last_time')
        local tokens = tonumber(info[1]) or burst
        local last_time = tonumber(info[2]) or now
        
        local elapsed = math.max(0, now - last_time)
        tokens = math.min(burst, tokens + elapsed * rate)
        
        if tokens >= requested then
            tokens = tokens - requested
            redis.call('HMSET', key, 'tokens', tokens, 'last_time', now)
            redis.call('EXPIRE', key, 3600)
            return 1
        else
            redis.call('HMSET', key, 'tokens', tokens, 'last_time', now)
            redis.call('EXPIRE', key, 3600)
            return 0
        end
    `
    
    result, err := l.store.Eval(ctx, script, []string{key}, 
        l.rate, l.burst, time.Now().Unix(), n)
    if err != nil {
        return false, err
    }
    
    return result.(int64) == 1, nil
}
```

### Sliding Window Implementation

```go
type SlidingWindowLimiter struct {
    windowSize time.Duration
    limit      int
    store      Store
    local      map[string]*window
    mu         sync.RWMutex
}

type window struct {
    counts []int
    index  int
    total  int
    mu     sync.Mutex
}

func NewSlidingWindowLimiter(limit int, windowSize time.Duration, store Store) *SlidingWindowLimiter {
    return &SlidingWindowLimiter{
        windowSize: windowSize,
        limit:      limit,
        store:      store,
        local:      make(map[string]*window),
    }
}

func (l *SlidingWindowLimiter) Allow(ctx context.Context, key string) (bool, error) {
    if l.store != nil {
        return l.allowDistributed(ctx, key)
    }
    return l.allowLocal(key), nil
}

func (l *SlidingWindowLimiter) allowLocal(key string) bool {
    l.mu.RLock()
    w, exists := l.local[key]
    l.mu.RUnlock()
    
    if !exists {
        l.mu.Lock()
        w = &window{
            counts: make([]int, 60), // 60 buckets for minute granularity
        }
        l.local[key] = w
        l.mu.Unlock()
    }
    
    w.mu.Lock()
    defer w.mu.Unlock()
    
    now := time.Now()
    currentBucket := now.Second()
    
    // Reset old buckets
    if currentBucket != w.index {
        if currentBucket < w.index {
            // Wrapped around
            for i := w.index + 1; i < 60; i++ {
                w.total -= w.counts[i]
                w.counts[i] = 0
            }
            for i := 0; i <= currentBucket; i++ {
                w.total -= w.counts[i]
                w.counts[i] = 0
            }
        } else {
            // Normal progression
            for i := w.index + 1; i <= currentBucket; i++ {
                w.total -= w.counts[i]
                w.counts[i] = 0
            }
        }
        w.index = currentBucket
    }
    
    if w.total >= l.limit {
        return false
    }
    
    w.counts[currentBucket]++
    w.total++
    return true
}

func (l *SlidingWindowLimiter) allowDistributed(ctx context.Context, key string) (bool, error) {
    now := time.Now()
    windowStart := now.Add(-l.windowSize)
    
    pipe := l.store.Pipeline()
    
    // Remove old entries
    pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", windowStart.UnixNano()))
    
    // Count current window
    pipe.ZCard(ctx, key)
    
    // Add current request if under limit
    pipe.ZAdd(ctx, key, &redis.Z{
        Score:  float64(now.UnixNano()),
        Member: now.UnixNano(),
    })
    
    // Set expiry
    pipe.Expire(ctx, key, l.windowSize+time.Minute)
    
    results, err := pipe.Exec(ctx)
    if err != nil {
        return false, err
    }
    
    count := results[1].(*redis.IntCmd).Val()
    return count < int64(l.limit), nil
}
```

### Adaptive Rate Limiter

```go
type AdaptiveLimiter struct {
    base       RateLimiter
    monitor    *SystemMonitor
    config     AdaptiveConfig
    multiplier float64
    mu         sync.RWMutex
}

type AdaptiveConfig struct {
    BaseRate           int
    MinRate            int
    MaxRate            int
    IncreaseThreshold  float64 // CPU/Memory threshold to increase limits
    DecreaseThreshold  float64 // CPU/Memory threshold to decrease limits
    AdjustmentInterval time.Duration
}

func NewAdaptiveLimiter(config AdaptiveConfig) *AdaptiveLimiter {
    baseLimiter := NewTokenBucketLimiter(TokenBucketConfig{
        Rate:     config.BaseRate,
        Interval: time.Second,
        Burst:    config.BaseRate,
    })
    
    limiter := &AdaptiveLimiter{
        base:       baseLimiter,
        monitor:    NewSystemMonitor(),
        config:     config,
        multiplier: 1.0,
    }
    
    go limiter.adjustLoop()
    return limiter
}

func (l *AdaptiveLimiter) adjustLoop() {
    ticker := time.NewTicker(l.config.AdjustmentInterval)
    defer ticker.Stop()
    
    for range ticker.C {
        metrics := l.monitor.GetMetrics()
        l.adjust(metrics)
    }
}

func (l *AdaptiveLimiter) adjust(metrics SystemMetrics) {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    // Calculate system load
    load := math.Max(metrics.CPUUsage, metrics.MemoryUsage)
    
    if load < l.config.IncreaseThreshold && l.multiplier < 2.0 {
        // System has capacity, increase limits
        l.multiplier = math.Min(2.0, l.multiplier*1.1)
    } else if load > l.config.DecreaseThreshold && l.multiplier > 0.5 {
        // System under stress, decrease limits
        l.multiplier = math.Max(0.5, l.multiplier*0.9)
    }
    
    // Update base limiter
    newRate := int(float64(l.config.BaseRate) * l.multiplier)
    newRate = max(l.config.MinRate, min(l.config.MaxRate, newRate))
    
    l.updateBaseRate(newRate)
}

func (l *AdaptiveLimiter) Allow(ctx context.Context, key string) (bool, error) {
    return l.base.Allow(ctx, key)
}
```

## Rate Limit Strategies

### User-Based Limiting

```go
type UserRateLimiter struct {
    limiter   RateLimiter
    userTiers map[string]Tier
}

type Tier struct {
    Name      string
    RateLimit int
    Burst     int
}

var defaultTiers = map[string]Tier{
    "free": {
        Name:      "free",
        RateLimit: 10,
        Burst:     20,
    },
    "pro": {
        Name:      "pro",
        RateLimit: 100,
        Burst:     200,
    },
    "enterprise": {
        Name:      "enterprise",
        RateLimit: 1000,
        Burst:     2000,
    },
}

func (u *UserRateLimiter) AllowUser(ctx context.Context, userID string) (bool, error) {
    // Get user tier
    tier := u.getUserTier(userID)
    
    // Create tier-specific limiter key
    key := fmt.Sprintf("user:%s:tier:%s", userID, tier.Name)
    
    // Check rate limit for tier
    return u.limiter.Allow(ctx, key)
}
```

### Endpoint-Based Limiting

```go
type EndpointRateLimiter struct {
    limiters map[string]RateLimiter
    defaults RateLimiter
}

func NewEndpointRateLimiter(configs map[string]EndpointConfig) *EndpointRateLimiter {
    limiters := make(map[string]RateLimiter)
    
    for endpoint, config := range configs {
        limiters[endpoint] = NewTokenBucketLimiter(TokenBucketConfig{
            Rate:     config.Rate,
            Interval: config.Interval,
            Burst:    config.Burst,
        })
    }
    
    return &EndpointRateLimiter{
        limiters: limiters,
        defaults: NewTokenBucketLimiter(defaultConfig),
    }
}

func (e *EndpointRateLimiter) Allow(ctx context.Context, endpoint, clientID string) (bool, error) {
    limiter, exists := e.limiters[endpoint]
    if !exists {
        limiter = e.defaults
    }
    
    key := fmt.Sprintf("%s:%s", endpoint, clientID)
    return limiter.Allow(ctx, key)
}
```

### Composite Rate Limiting

```go
type CompositeRateLimiter struct {
    limiters []RateLimiter
    mode     CompositeMode
}

type CompositeMode int

const (
    ModeAll CompositeMode = iota // All limiters must allow
    ModeAny                      // Any limiter can allow
)

func (c *CompositeRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
    switch c.mode {
    case ModeAll:
        for _, limiter := range c.limiters {
            allowed, err := limiter.Allow(ctx, key)
            if err != nil {
                return false, err
            }
            if !allowed {
                return false, nil
            }
        }
        return true, nil
        
    case ModeAny:
        for _, limiter := range c.limiters {
            allowed, err := limiter.Allow(ctx, key)
            if err != nil {
                continue
            }
            if allowed {
                return true, nil
            }
        }
        return false, nil
        
    default:
        return false, errors.New("unknown composite mode")
    }
}
```

## Redis Store Implementation

```go
type RedisStore struct {
    client  *redis.Client
    prefix  string
    logger  *slog.Logger
}

func NewRedisStore(url, prefix string) (*RedisStore, error) {
    opts, err := redis.ParseURL(url)
    if err != nil {
        return nil, fmt.Errorf("parsing Redis URL: %w", err)
    }
    
    client := redis.NewClient(opts)
    
    // Test connection
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("connecting to Redis: %w", err)
    }
    
    return &RedisStore{
        client: client,
        prefix: prefix,
        logger: slog.With("component", "redis_store"),
    }, nil
}

func (s *RedisStore) prefixKey(key string) string {
    if s.prefix != "" {
        return fmt.Sprintf("%s:%s", s.prefix, key)
    }
    return key
}

func (s *RedisStore) Incr(ctx context.Context, key string) (int64, error) {
    return s.client.Incr(ctx, s.prefixKey(key)).Result()
}

func (s *RedisStore) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
    return s.client.IncrBy(ctx, s.prefixKey(key), value).Result()
}

func (s *RedisStore) Expire(ctx context.Context, key string, ttl time.Duration) error {
    return s.client.Expire(ctx, s.prefixKey(key), ttl).Err()
}

func (s *RedisStore) Del(ctx context.Context, keys ...string) error {
    prefixedKeys := make([]string, len(keys))
    for i, key := range keys {
        prefixedKeys[i] = s.prefixKey(key)
    }
    return s.client.Del(ctx, prefixedKeys...).Err()
}
```

## Middleware Integration

### HTTP Middleware

```go
func RateLimitMiddleware(limiter RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Generate rate limit key
            key := generateKey(r)
            
            // Check rate limit
            allowed, err := limiter.Allow(r.Context(), key)
            if err != nil {
                http.Error(w, "Internal Server Error", http.StatusInternalServerError)
                return
            }
            
            // Get limit info
            info, _ := limiter.GetInfo(key)
            
            // Set rate limit headers
            w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(info.Limit, 10))
            w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(info.Remaining, 10))
            w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(info.ResetAt.Unix(), 10))
            
            if !allowed {
                w.Header().Set("Retry-After", strconv.FormatInt(info.RetryAfter, 10))
                http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}

func generateKey(r *http.Request) string {
    // Try to get user ID from context
    if userID := r.Context().Value("user_id"); userID != nil {
        return fmt.Sprintf("user:%v", userID)
    }
    
    // Fall back to IP address
    ip := getClientIP(r)
    return fmt.Sprintf("ip:%s", ip)
}
```

## Configuration

### Rate Limiter Configuration

```yaml
rate_limit:
  # Default configuration
  default:
    algorithm: "token_bucket"
    rate: 100
    interval: "1m"
    burst: 200
    
  # Redis backend
  redis:
    enabled: true
    url: "${REDIS_URL}"
    prefix: "ratelimit"
    
  # Adaptive limiting
  adaptive:
    enabled: true
    min_rate: 50
    max_rate: 500
    increase_threshold: 0.6
    decrease_threshold: 0.8
    adjustment_interval: "30s"
    
  # Per-user tiers
  user_tiers:
    free:
      rate: 10
      burst: 20
    pro:
      rate: 100
      burst: 200
    enterprise:
      rate: 1000
      burst: 2000
      
  # Endpoint-specific limits
  endpoints:
    "/api/v1/chat":
      rate: 30
      interval: "1m"
      burst: 5
    "/api/v1/analyze":
      rate: 10
      interval: "1m"
      burst: 2
    "/api/v1/langchain/execute":
      rate: 5
      interval: "1m"
      burst: 1
```

## Monitoring and Metrics

### Rate Limit Metrics

```go
type RateLimitMetrics struct {
    allowed   *prometheus.CounterVec
    denied    *prometheus.CounterVec
    latency   *prometheus.HistogramVec
}

func NewRateLimitMetrics() *RateLimitMetrics {
    return &RateLimitMetrics{
        allowed: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "ratelimit_allowed_total",
                Help: "Total number of allowed requests",
            },
            []string{"key", "limiter"},
        ),
        denied: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "ratelimit_denied_total",
                Help: "Total number of denied requests",
            },
            []string{"key", "limiter", "reason"},
        ),
        latency: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "ratelimit_check_duration_seconds",
                Help:    "Rate limit check duration",
                Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
            },
            []string{"limiter"},
        ),
    }
}
```

## Usage Examples

### Basic Usage

```go
func ExampleBasicRateLimiter() {
    // Create token bucket limiter
    limiter := NewTokenBucketLimiter(TokenBucketConfig{
        Rate:     100,
        Interval: time.Minute,
        Burst:    10,
    })
    
    // Check if request is allowed
    allowed, err := limiter.Allow(context.Background(), "user:123")
    if err != nil {
        log.Fatal(err)
    }
    
    if !allowed {
        fmt.Println("Rate limit exceeded")
        return
    }
    
    // Process request
    fmt.Println("Request allowed")
}
```

### Advanced Usage with Redis

```go
func ExampleDistributedRateLimiter() {
    // Create Redis store
    store, err := NewRedisStore("redis://localhost:6379", "myapp")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create distributed limiter
    limiter := NewSlidingWindowLimiter(100, time.Minute, store)
    
    // Use in HTTP handler
    http.HandleFunc("/api/endpoint", RateLimitMiddleware(limiter)(
        http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Write([]byte("Hello, World!"))
        }),
    ))
}
```

## Testing

### Rate Limiter Testing

```go
func TestTokenBucketLimiter(t *testing.T) {
    limiter := NewTokenBucketLimiter(TokenBucketConfig{
        Rate:     10,
        Interval: time.Second,
        Burst:    5,
    })
    
    key := "test-key"
    
    // Should allow burst
    for i := 0; i < 5; i++ {
        allowed, err := limiter.Allow(context.Background(), key)
        assert.NoError(t, err)
        assert.True(t, allowed, "Request %d should be allowed", i+1)
    }
    
    // Should deny after burst
    allowed, err := limiter.Allow(context.Background(), key)
    assert.NoError(t, err)
    assert.False(t, allowed, "Request should be denied after burst")
    
    // Wait for refill
    time.Sleep(100 * time.Millisecond)
    
    // Should allow one more
    allowed, err = limiter.Allow(context.Background(), key)
    assert.NoError(t, err)
    assert.True(t, allowed, "Request should be allowed after refill")
}
```

### Load Testing

```go
func BenchmarkRateLimiter(b *testing.B) {
    limiter := NewTokenBucketLimiter(TokenBucketConfig{
        Rate:     1000,
        Interval: time.Second,
        Burst:    100,
    })
    
    b.RunParallel(func(pb *testing.PB) {
        key := fmt.Sprintf("bench-key-%d", rand.Int())
        for pb.Next() {
            limiter.Allow(context.Background(), key)
        }
    })
}
```

## Related Documentation

- [API](../api/README.md) - API rate limiting integration
- [Server](../server/README.md) - Server middleware
- [Configuration](../config/README.md) - Rate limit configuration
- [Monitoring](../observability/README.md) - Metrics and monitoring