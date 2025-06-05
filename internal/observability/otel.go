package observability

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// OTelSetup configures OpenTelemetry following golang_guide.md best practices
type OTelSetup struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	logger         *slog.Logger
}

// SetupOTelSDK initializes OpenTelemetry SDK following golang_guide.md recommendations
func SetupOTelSDK(ctx context.Context, serviceName, serviceVersion string, logger *slog.Logger) (*OTelSetup, func(context.Context) error, error) {
	// Create resource with semantic conventions (golang_guide.md pattern)
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			semconv.ServiceInstanceID(fmt.Sprintf("%s-%d", serviceName, time.Now().Unix())),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Setup tracing (Stable - production ready according to golang_guide.md)
	tracerProvider, err := newTracerProvider(ctx, res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create tracer provider: %w", err)
	}
	otel.SetTracerProvider(tracerProvider)

	// Setup metrics (Beta - production ready according to golang_guide.md)
	meterProvider, err := newMeterProvider(res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create meter provider: %w", err)
	}
	otel.SetMeterProvider(meterProvider)

	setup := &OTelSetup{
		tracerProvider: tracerProvider,
		meterProvider:  meterProvider,
		logger:         logger,
	}

	// Return shutdown function
	shutdown := func(shutdownCtx context.Context) error {
		var err error

		// Shutdown tracer provider
		if shutdownErr := tracerProvider.Shutdown(shutdownCtx); shutdownErr != nil {
			err = fmt.Errorf("failed to shutdown tracer provider: %w", shutdownErr)
			logger.Error("Failed to shutdown tracer provider", slog.Any("error", shutdownErr))
		}

		// Shutdown meter provider
		if shutdownErr := meterProvider.Shutdown(shutdownCtx); shutdownErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; failed to shutdown meter provider: %w", err, shutdownErr)
			} else {
				err = fmt.Errorf("failed to shutdown meter provider: %w", shutdownErr)
			}
			logger.Error("Failed to shutdown meter provider", slog.Any("error", shutdownErr))
		}

		return err
	}

	return setup, shutdown, nil
}

// newTracerProvider creates a tracer provider with production-ready sampling (golang_guide.md)
func newTracerProvider(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	// OTLP HTTP exporter for traces
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("http://localhost:4318"), // Jaeger default
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Production sampling strategy (golang_guide.md best practices)
	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(0.01),                            // 1% base sampling for performance
		sdktrace.WithRemoteParentSampled(sdktrace.AlwaysSample()),   // Always sample if parent sampled
		sdktrace.WithRemoteParentNotSampled(sdktrace.NeverSample()), // Never sample if parent not sampled
	)

	// Batch span processor for efficient export
	bsp := sdktrace.NewBatchSpanProcessor(
		traceExporter,
		sdktrace.WithBatchTimeout(5*time.Second), // Export every 5 seconds
		sdktrace.WithMaxExportBatchSize(512),     // Batch up to 512 spans
		sdktrace.WithMaxQueueSize(2048),          // Queue up to 2048 spans
	)

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	return tracerProvider, nil
}

// newMeterProvider creates a meter provider with Prometheus export (golang_guide.md)
func newMeterProvider(res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	// Prometheus exporter for metrics
	prometheusExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(prometheusExporter),
		// Periodic reader for OTLP metrics export (optional)
		// sdkmetric.WithReader(sdkmetric.NewPeriodicReader(otlpMetricExporter, sdkmetric.WithInterval(30*time.Second))),
	)

	return meterProvider, nil
}

// TraceMiddleware adds tracing to HTTP handlers following golang_guide.md patterns
func TraceMiddleware(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Start span with semantic conventions
			ctx, span := tracer.Start(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(r.Method),
					semconv.URLFull(r.URL.String()),
					semconv.HTTPRouteKey.String(r.URL.Path),
					semconv.URLScheme(r.URL.Scheme),
					semconv.UserAgentOriginal(r.Header.Get("User-Agent")),
				),
			)
			defer span.End()

			// Add span context to request context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DatabaseTracer provides database operation tracing
type DatabaseTracer struct {
	tracer trace.Tracer
}

// NewDatabaseTracer creates a new database tracer
func NewDatabaseTracer(serviceName string) *DatabaseTracer {
	return &DatabaseTracer{
		tracer: otel.Tracer(fmt.Sprintf("%s-db", serviceName)),
	}
}

// TraceQuery traces a database query following golang_guide.md database tracing patterns
func (dt *DatabaseTracer) TraceQuery(ctx context.Context, operation, table, query string) (context.Context, trace.Span) {
	return dt.tracer.Start(ctx, fmt.Sprintf("db.%s.%s", operation, table),
		trace.WithAttributes(
			semconv.DBOperationKey.String(operation),
			semconv.DBSQLTableKey.String(table),
			semconv.DBStatementKey.String(query),
			semconv.DBSystemKey.String("postgresql"),
		),
	)
}

// AITracer provides AI operation tracing
type AITracer struct {
	tracer trace.Tracer
}

// NewAITracer creates a new AI tracer
func NewAITracer(serviceName string) *AITracer {
	return &AITracer{
		tracer: otel.Tracer(fmt.Sprintf("%s-ai", serviceName)),
	}
}

// TraceAIRequest traces an AI provider request
func (at *AITracer) TraceAIRequest(ctx context.Context, provider, model, operation string) (context.Context, trace.Span) {
	return at.tracer.Start(ctx, fmt.Sprintf("ai.%s.%s", provider, operation),
		trace.WithAttributes(
			semconv.ServiceNameKey.String(provider),
			semconv.ServiceVersionKey.String(model),
			semconv.CodeFunctionKey.String(operation),
		),
	)
}

// ToolTracer provides tool execution tracing
type ToolTracer struct {
	tracer trace.Tracer
}

// NewToolTracer creates a new tool tracer
func NewToolTracer(serviceName string) *ToolTracer {
	return &ToolTracer{
		tracer: otel.Tracer(fmt.Sprintf("%s-tools", serviceName)),
	}
}

// TraceToolExecution traces a tool execution
func (tt *ToolTracer) TraceToolExecution(ctx context.Context, toolName, operation string) (context.Context, trace.Span) {
	return tt.tracer.Start(ctx, fmt.Sprintf("tool.%s.%s", toolName, operation),
		trace.WithAttributes(
			semconv.CodeFunctionKey.String(operation),
			semconv.CodeNamespaceKey.String(toolName),
		),
	)
}

// Metrics provides structured metrics collection following RED method (golang_guide.md)
type Metrics struct {
	// Request metrics (RED method)
	requestsTotal   metric.Int64Counter
	requestDuration metric.Float64Histogram
	requestErrors   metric.Int64Counter

	// Custom application metrics
	activeConnections metric.Int64UpDownCounter
	cacheOperations   metric.Int64Counter
	memoryUsage       metric.Float64Gauge

	meter metric.Meter
}

// NewMetrics creates a new metrics collection
func NewMetrics(serviceName string) (*Metrics, error) {
	meter := otel.Meter(serviceName)

	requestsTotal, err := meter.Int64Counter(
		"requests_total",
		metric.WithDescription("Total number of requests processed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create requests_total counter: %w", err)
	}

	requestDuration, err := meter.Float64Histogram(
		"request_duration_seconds",
		metric.WithDescription("Duration of HTTP requests"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request_duration histogram: %w", err)
	}

	requestErrors, err := meter.Int64Counter(
		"request_errors_total",
		metric.WithDescription("Total number of request errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request_errors counter: %w", err)
	}

	activeConnections, err := meter.Int64UpDownCounter(
		"active_connections",
		metric.WithDescription("Number of active connections"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active_connections counter: %w", err)
	}

	cacheOperations, err := meter.Int64Counter(
		"cache_operations_total",
		metric.WithDescription("Total number of cache operations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache_operations counter: %w", err)
	}

	memoryUsage, err := meter.Float64Gauge(
		"memory_usage_bytes",
		metric.WithDescription("Current memory usage in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory_usage gauge: %w", err)
	}

	return &Metrics{
		requestsTotal:     requestsTotal,
		requestDuration:   requestDuration,
		requestErrors:     requestErrors,
		activeConnections: activeConnections,
		cacheOperations:   cacheOperations,
		memoryUsage:       memoryUsage,
		meter:             meter,
	}, nil
}

// RecordRequest records a request with the RED method pattern
func (m *Metrics) RecordRequest(ctx context.Context, method, path string, duration time.Duration, statusCode int) {
	// Rate - increment request counter
	m.requestsTotal.Add(ctx, 1, metric.WithAttributes(
		semconv.HTTPRequestMethodKey.String(method),
		semconv.HTTPRouteKey.String(path),
		semconv.HTTPResponseStatusCodeKey.Int(statusCode),
	))

	// Duration - record request duration
	m.requestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		semconv.HTTPRequestMethodKey.String(method),
		semconv.HTTPRouteKey.String(path),
	))

	// Errors - increment error counter if needed
	if statusCode >= 400 {
		m.requestErrors.Add(ctx, 1, metric.WithAttributes(
			semconv.HTTPRequestMethodKey.String(method),
			semconv.HTTPRouteKey.String(path),
			semconv.HTTPResponseStatusCodeKey.Int(statusCode),
		))
	}
}

// RecordCacheOperation records cache operations
func (m *Metrics) RecordCacheOperation(ctx context.Context, operation, result string) {
	m.cacheOperations.Add(ctx, 1, metric.WithAttributes(
		semconv.CodeFunctionKey.String(operation),
		semconv.HTTPResponseStatusCodeKey.String(result), // hit/miss/error
	))
}

// UpdateActiveConnections updates the active connections gauge
func (m *Metrics) UpdateActiveConnections(ctx context.Context, delta int64) {
	m.activeConnections.Add(ctx, delta)
}

// UpdateMemoryUsage updates the memory usage gauge
func (m *Metrics) UpdateMemoryUsage(ctx context.Context, bytes float64) {
	m.memoryUsage.Record(ctx, bytes)
}
