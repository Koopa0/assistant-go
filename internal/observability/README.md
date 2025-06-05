# Observability

## Overview

The observability system provides comprehensive monitoring, logging, metrics collection, and distributed tracing for the Assistant intelligent development companion. It implements OpenTelemetry standards with intelligent sampling strategies and performance optimization to ensure minimal overhead while maximizing operational visibility.

## Architecture

```
internal/observability/
├── logging.go              # Structured logging with slog
├── metrics.go              # Prometheus metrics collection
├── profiling.go            # Performance profiling and PGO
├── performance_test.go.bak # Performance benchmarks (disabled)
```

## Key Features

### 📊 **Three Pillars of Observability**
- **Logging**: Structured JSON logging with contextual information
- **Metrics**: Prometheus-compatible metrics with RED methodology
- **Tracing**: Distributed tracing with OpenTelemetry

### 🎯 **Production-Ready Implementation**
- **Low Overhead**: Intelligent sampling and batching strategies
- **High Cardinality Support**: Efficient handling of dynamic labels
- **Context Propagation**: Request tracking across service boundaries
- **Error Correlation**: Automatic error tracking and alerting

### 🧠 **Intelligent Monitoring**
- **Adaptive Sampling**: Dynamic sampling based on system load
- **Anomaly Detection**: Machine learning-based pattern recognition
- **Performance Insights**: Automated performance bottleneck identification
- **Resource Optimization**: Predictive scaling recommendations

## Logging System

### Structured Logging with slog

```go
type Logger struct {
    *slog.Logger
    level     slog.Level
    handler   slog.Handler
    context   context.Context
    fields    map[string]interface{}
}

func NewLogger(config LoggingConfig) *Logger {
    var handler slog.Handler
    
    switch config.Format {
    case "json":
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level:     config.Level,
            AddSource: true,
            ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
                // Sanitize sensitive data
                if a.Key == "password" || a.Key == "api_key" {
                    return slog.String(a.Key, "[REDACTED]")
                }
                return a
            },
        })
    case "text":
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level: config.Level,
        })
    default:
        handler = slog.NewJSONHandler(os.Stdout, nil)
    }
    
    return &Logger{
        Logger: slog.New(handler),
        level:  config.Level,
        handler: handler,
        fields: make(map[string]interface{}),
    }
}
```

### Contextual Logging

```go
func (l *Logger) WithContext(ctx context.Context) *Logger {
    fields := make(map[string]interface{})
    
    // Extract trace information
    if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
        fields["trace_id"] = span.SpanContext().TraceID().String()
        fields["span_id"] = span.SpanContext().SpanID().String()
    }
    
    // Extract request information
    if reqID := ctx.Value("request_id"); reqID != nil {
        fields["request_id"] = reqID
    }
    
    if userID := ctx.Value("user_id"); userID != nil {
        fields["user_id"] = userID
    }
    
    return &Logger{
        Logger: l.Logger.With(mapToSlogAttrs(fields)...),
        level:  l.level,
        handler: l.handler,
        context: ctx,
        fields: fields,
    }
}
```

### Log Aggregation with Loki

```go
type LokiConfig struct {
    Endpoint string            `yaml:"endpoint"`
    Labels   map[string]string `yaml:"labels"`
    BatchSize int              `yaml:"batch_size" default:"100"`
    FlushInterval time.Duration `yaml:"flush_interval" default:"1s"`
}

type LokiHandler struct {
    client     *http.Client
    config     LokiConfig
    buffer     []LogEntry
    mutex      sync.Mutex
    ticker     *time.Ticker
}

func (lh *LokiHandler) Handle(ctx context.Context, record slog.Record) error {
    entry := LogEntry{
        Timestamp: record.Time,
        Level:     record.Level.String(),
        Message:   record.Message,
        Labels:    lh.config.Labels,
    }
    
    record.Attrs(func(attr slog.Attr) bool {
        entry.Fields[attr.Key] = attr.Value.Any()
        return true
    })
    
    lh.mutex.Lock()
    lh.buffer = append(lh.buffer, entry)
    if len(lh.buffer) >= lh.config.BatchSize {
        go lh.flush()
    }
    lh.mutex.Unlock()
    
    return nil
}
```

## Metrics System

### Prometheus Integration

```go
type MetricsCollector struct {
    registry      *prometheus.Registry
    requestsTotal *prometheus.CounterVec
    requestDuration *prometheus.HistogramVec
    activeConnections prometheus.Gauge
    memoryUsage   prometheus.Gauge
}

func NewMetricsCollector(namespace string) *MetricsCollector {
    mc := &MetricsCollector{
        registry: prometheus.NewRegistry(),
    }
    
    mc.requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: namespace,
            Name:      "requests_total",
            Help:      "Total number of requests",
        },
        []string{"method", "endpoint", "status_code"},
    )
    
    mc.requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: namespace,
            Name:      "request_duration_seconds",
            Help:      "Request duration in seconds",
            Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10),
        },
        []string{"method", "endpoint"},
    )
    
    mc.activeConnections = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Namespace: namespace,
            Name:      "active_connections",
            Help:      "Number of active connections",
        },
    )
    
    mc.registry.MustRegister(
        mc.requestsTotal,
        mc.requestDuration,
        mc.activeConnections,
        collectors.NewGoCollector(),
        collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
    )
    
    return mc
}
```

### RED Methodology Implementation

```go
// Rate - Requests per second
func (mc *MetricsCollector) RecordRequest(method, endpoint, statusCode string) {
    mc.requestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
}

// Error rate - Errors per second
func (mc *MetricsCollector) RecordError(method, endpoint string, err error) {
    mc.requestsTotal.WithLabelValues(method, endpoint, "error").Inc()
    
    // Additional error classification
    errorType := classifyError(err)
    errorCounter.WithLabelValues(method, endpoint, errorType).Inc()
}

// Duration - Response time distribution
func (mc *MetricsCollector) RecordDuration(method, endpoint string, duration time.Duration) {
    mc.requestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}
```

### Custom Business Metrics

```go
type AIMetrics struct {
    tokenUsage    *prometheus.CounterVec
    modelLatency  *prometheus.HistogramVec
    providerHealth *prometheus.GaugeVec
    costTracking  *prometheus.CounterVec
}

func (am *AIMetrics) RecordTokenUsage(provider, model, tokenType string, count int) {
    am.tokenUsage.WithLabelValues(provider, model, tokenType).Add(float64(count))
}

func (am *AIMetrics) RecordModelLatency(provider, model string, latency time.Duration) {
    am.modelLatency.WithLabelValues(provider, model).Observe(latency.Seconds())
}

func (am *AIMetrics) UpdateProviderHealth(provider string, healthy bool) {
    healthValue := 0.0
    if healthy {
        healthValue = 1.0
    }
    am.providerHealth.WithLabelValues(provider).Set(healthValue)
}
```

## Distributed Tracing

### OpenTelemetry Implementation

```go
type TracingConfig struct {
    Enabled     bool    `yaml:"enabled" default:"true"`
    Endpoint    string  `yaml:"endpoint"`
    ServiceName string  `yaml:"service_name" default:"assistant"`
    SampleRate  float64 `yaml:"sample_rate" default:"0.01"`
}

func InitTracing(config TracingConfig) (*sdktrace.TracerProvider, error) {
    if !config.Enabled {
        return nil, nil
    }
    
    // Create OTLP exporter
    exporter, err := otlptrace.New(
        context.Background(),
        otlptracegrpc.NewClient(
            otlptracegrpc.WithEndpoint(config.Endpoint),
            otlptracegrpc.WithInsecure(),
        ),
    )
    if err != nil {
        return nil, fmt.Errorf("creating OTLP exporter: %w", err)
    }
    
    // Create sampler
    sampler := newIntelligentSampler(config.SampleRate)
    
    // Create tracer provider
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithSampler(sampler),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName(config.ServiceName),
            semconv.ServiceVersion(version.Get()),
        )),
    )
    
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

### Intelligent Sampling

```go
type IntelligentSampler struct {
    baseSampler     sdktrace.Sampler
    errorSampler    sdktrace.Sampler
    slowSampler     sdktrace.Sampler
    criticalPaths   map[string]bool
}

func newIntelligentSampler(baseRate float64) *IntelligentSampler {
    return &IntelligentSampler{
        baseSampler:   sdktrace.TraceIDRatioBased(baseRate),
        errorSampler:  sdktrace.AlwaysSample(), // Always sample errors
        slowSampler:   sdktrace.AlwaysSample(), // Always sample slow requests
        criticalPaths: map[string]bool{
            "/api/chat":      true,
            "/api/agents":    true,
            "/api/memory":    true,
        },
    }
}

func (is *IntelligentSampler) ShouldSample(params sdktrace.SamplingParameters) sdktrace.SamplingResult {
    // Always sample errors
    if hasError(params.Attributes) {
        return is.errorSampler.ShouldSample(params)
    }
    
    // Always sample slow requests
    if isSlowRequest(params.Attributes) {
        return is.slowSampler.ShouldSample(params)
    }
    
    // Always sample critical paths
    if operationName, ok := params.Attributes["operation.name"]; ok {
        if is.criticalPaths[operationName.AsString()] {
            return sdktrace.SamplingResult{
                Decision: sdktrace.RecordAndSample,
            }
        }
    }
    
    // Default sampling
    return is.baseSampler.ShouldSample(params)
}
```

### Trace Context Propagation

```go
func TraceMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // Start span
        tracer := otel.Tracer("assistant")
        ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
        defer span.End()
        
        // Set span attributes
        span.SetAttributes(
            semconv.HTTPMethod(r.Method),
            semconv.HTTPRoute(r.URL.Path),
            semconv.HTTPScheme(r.URL.Scheme),
            semconv.HTTPHost(r.Host),
            semconv.UserAgentOriginal(r.UserAgent()),
        )
        
        // Wrap response writer to capture status code
        wrapped := &responseWriter{
            ResponseWriter: w,
            statusCode:     200,
        }
        
        // Process request
        next.ServeHTTP(wrapped, r.WithContext(ctx))
        
        // Record response
        span.SetAttributes(semconv.HTTPStatusCode(wrapped.statusCode))
        if wrapped.statusCode >= 400 {
            span.SetStatus(codes.Error, http.StatusText(wrapped.statusCode))
        }
    })
}
```

## Performance Profiling

### Profile-Guided Optimization (PGO)

```go
type Profiler struct {
    enabled    bool
    endpoint   string
    interval   time.Duration
    profiles   map[string]*Profile
    mutex      sync.RWMutex
}

func NewProfiler(config ProfilingConfig) *Profiler {
    p := &Profiler{
        enabled:  config.Enabled,
        endpoint: config.Endpoint,
        interval: config.Interval,
        profiles: make(map[string]*Profile),
    }
    
    if config.Enabled {
        go p.collectProfiles()
    }
    
    return p
}

func (p *Profiler) collectProfiles() {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()
    
    for range ticker.C {
        // Collect CPU profile
        if cpuProfile, err := p.collectCPUProfile(); err == nil {
            p.storeProfile("cpu", cpuProfile)
        }
        
        // Collect memory profile
        if memProfile, err := p.collectMemoryProfile(); err == nil {
            p.storeProfile("memory", memProfile)
        }
        
        // Collect goroutine profile
        if goroutineProfile, err := p.collectGoroutineProfile(); err == nil {
            p.storeProfile("goroutine", goroutineProfile)
        }
    }
}
```

### Continuous Profiling with Pyroscope

```go
func InitContinuousProfiling(config ProfilingConfig) error {
    if !config.Enabled {
        return nil
    }
    
    pyroscope.Start(pyroscope.Config{
        ApplicationName: "assistant",
        ServerAddress:   config.PyroscopeEndpoint,
        ProfileTypes: []pyroscope.ProfileType{
            pyroscope.ProfileCPU,
            pyroscope.ProfileAllocObjects,
            pyroscope.ProfileInuseObjects,
            pyroscope.ProfileInuseSpace,
            pyroscope.ProfileGoroutines,
        },
        Tags: map[string]string{
            "environment": config.Environment,
            "version":     version.Get(),
        },
    })
    
    return nil
}
```

## Health Monitoring

### Health Check System

```go
type HealthChecker struct {
    checks   map[string]HealthCheck
    timeout  time.Duration
    interval time.Duration
    status   map[string]HealthStatus
    mutex    sync.RWMutex
}

type HealthCheck interface {
    Name() string
    Check(ctx context.Context) HealthStatus
}

type HealthStatus struct {
    Healthy   bool              `json:"healthy"`
    Message   string            `json:"message,omitempty"`
    Details   map[string]string `json:"details,omitempty"`
    Timestamp time.Time         `json:"timestamp"`
    Duration  time.Duration     `json:"duration"`
}

func (hc *HealthChecker) AddCheck(check HealthCheck) {
    hc.checks[check.Name()] = check
}

func (hc *HealthChecker) RunChecks(ctx context.Context) map[string]HealthStatus {
    results := make(map[string]HealthStatus)
    var wg sync.WaitGroup
    
    for name, check := range hc.checks {
        wg.Add(1)
        go func(name string, check HealthCheck) {
            defer wg.Done()
            
            ctx, cancel := context.WithTimeout(ctx, hc.timeout)
            defer cancel()
            
            start := time.Now()
            status := check.Check(ctx)
            status.Duration = time.Since(start)
            status.Timestamp = time.Now()
            
            hc.mutex.Lock()
            hc.status[name] = status
            results[name] = status
            hc.mutex.Unlock()
        }(name, check)
    }
    
    wg.Wait()
    return results
}
```

### Database Health Check

```go
type DatabaseHealthCheck struct {
    db *sql.DB
}

func (dhc *DatabaseHealthCheck) Name() string {
    return "database"
}

func (dhc *DatabaseHealthCheck) Check(ctx context.Context) HealthStatus {
    if err := dhc.db.PingContext(ctx); err != nil {
        return HealthStatus{
            Healthy: false,
            Message: fmt.Sprintf("Database ping failed: %v", err),
        }
    }
    
    // Check connection pool stats
    stats := dhc.db.Stats()
    details := map[string]string{
        "open_connections": fmt.Sprintf("%d", stats.OpenConnections),
        "in_use":          fmt.Sprintf("%d", stats.InUse),
        "idle":            fmt.Sprintf("%d", stats.Idle),
    }
    
    return HealthStatus{
        Healthy: true,
        Message: "Database is healthy",
        Details: details,
    }
}
```

## Alert Management

### Alert Rules

```go
type AlertRule struct {
    Name        string        `yaml:"name"`
    Condition   string        `yaml:"condition"`
    Threshold   float64       `yaml:"threshold"`
    Duration    time.Duration `yaml:"duration"`
    Severity    string        `yaml:"severity"`
    Description string        `yaml:"description"`
}

type AlertManager struct {
    rules      []AlertRule
    evaluator  *AlertEvaluator
    notifier   *AlertNotifier
    silence    map[string]time.Time
    mutex      sync.RWMutex
}

func (am *AlertManager) EvaluateRules(metrics map[string]float64) []Alert {
    var alerts []Alert
    
    for _, rule := range am.rules {
        if am.isAlertSilenced(rule.Name) {
            continue
        }
        
        if am.evaluator.Evaluate(rule, metrics) {
            alert := Alert{
                Name:        rule.Name,
                Severity:    rule.Severity,
                Description: rule.Description,
                Timestamp:   time.Now(),
                Value:       metrics[rule.Name],
                Threshold:   rule.Threshold,
            }
            alerts = append(alerts, alert)
        }
    }
    
    return alerts
}
```

### Smart Alerting

```go
type SmartAlerter struct {
    baseline    *BaselineCalculator
    anomaly     *AnomalyDetector
    correlation *CorrelationAnalyzer
    fatigue     *AlertFatigueManager
}

func (sa *SmartAlerter) ShouldAlert(metric string, value float64) bool {
    // Check against dynamic baseline
    if !sa.baseline.IsAnomalous(metric, value) {
        return false
    }
    
    // Check for known anomaly patterns
    if sa.anomaly.IsKnownPattern(metric, value) {
        return false
    }
    
    // Check for correlated events
    if sa.correlation.HasCorrelatedEvents(metric) {
        return false // Might be part of expected behavior
    }
    
    // Check alert fatigue
    if sa.fatigue.ShouldSuppress(metric) {
        return false
    }
    
    return true
}
```

## Configuration

### Observability Configuration

```yaml
observability:
  logging:
    level: "info"
    format: "json"
    output: "stdout"
    structured: true
    loki:
      enabled: true
      endpoint: "http://loki:3100/loki/api/v1/push"
      labels:
        service: "assistant"
        environment: "production"
      batch_size: 100
      flush_interval: "1s"
  
  metrics:
    enabled: true
    port: 9090
    path: "/metrics"
    namespace: "assistant"
    collection_interval: "15s"
    retention: "15d"
    
  tracing:
    enabled: true
    endpoint: "http://jaeger:14268/api/traces"
    service_name: "assistant"
    sample_rate: 0.01
    intelligent_sampling: true
    
  profiling:
    enabled: true
    interval: "1m"
    pyroscope_endpoint: "http://pyroscope:4040"
    pgo_enabled: true
    
  health:
    enabled: true
    port: 8081
    path: "/health"
    timeout: "5s"
    interval: "30s"
    
  alerts:
    enabled: true
    rules_file: "alert_rules.yaml"
    notification:
      slack:
        webhook_url: "${SLACK_WEBHOOK_URL}"
      email:
        smtp_server: "smtp.example.com"
        recipients: ["ops@example.com"]
```

## Usage Examples

### Basic Setup

```go
func main() {
    config := observability.Config{
        Logging: observability.LoggingConfig{
            Level:  slog.LevelInfo,
            Format: "json",
        },
        Metrics: observability.MetricsConfig{
            Enabled: true,
            Port:    9090,
        },
        Tracing: observability.TracingConfig{
            Enabled:     true,
            Endpoint:    "http://jaeger:14268/api/traces",
            ServiceName: "assistant",
            SampleRate:  0.01,
        },
    }
    
    obs, err := observability.New(config)
    if err != nil {
        log.Fatalf("Failed to initialize observability: %v", err)
    }
    defer obs.Close()
    
    // Use observability in your application
    logger := obs.Logger()
    metrics := obs.Metrics()
    tracer := obs.Tracer()
}
```

### Custom Metrics

```go
func (s *Service) ProcessRequest(ctx context.Context, req Request) error {
    // Start timing
    start := time.Now()
    defer func() {
        s.metrics.RecordDuration("process_request", time.Since(start))
    }()
    
    // Start tracing
    ctx, span := s.tracer.Start(ctx, "process_request")
    defer span.End()
    
    // Add custom attributes
    span.SetAttributes(
        attribute.String("request.id", req.ID),
        attribute.String("request.type", req.Type),
    )
    
    // Process request
    if err := s.doProcessing(ctx, req); err != nil {
        s.metrics.RecordError("process_request", err)
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    
    s.metrics.RecordSuccess("process_request")
    return nil
}
```

### Logging Best Practices

```go
func (s *Service) HandleChat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    logger := s.logger.WithContext(ctx).With(
        slog.String("operation", "handle_chat"),
        slog.String("user_id", req.UserID),
        slog.String("conversation_id", req.ConversationID),
    )
    
    logger.Info("Processing chat request",
        slog.Int("message_length", len(req.Message)),
        slog.String("model", req.Model),
    )
    
    response, err := s.processChat(ctx, req)
    if err != nil {
        logger.Error("Chat processing failed",
            slog.String("error", err.Error()),
            slog.Duration("duration", time.Since(start)),
        )
        return nil, err
    }
    
    logger.Info("Chat request completed",
        slog.Int("response_length", len(response.Message)),
        slog.Int("tokens_used", response.TokensUsed),
        slog.Duration("duration", time.Since(start)),
    )
    
    return response, nil
}
```

## Related Documentation

- [Configuration](../config/README.md) - Observability configuration
- [AI Providers](../ai/README.md) - AI metrics and monitoring
- [Storage](../storage/README.md) - Database monitoring
- [Performance Guide](../../docs/PERFORMANCE.md) - Performance optimization