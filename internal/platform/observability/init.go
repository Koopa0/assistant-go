package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// ObservabilityConfig configures the observability stack
type ObservabilityConfig struct {
	// Service identification
	ServiceName    string `yaml:"service_name"`
	ServiceVersion string `yaml:"service_version"`
	Environment    string `yaml:"environment"`

	// Logging configuration
	LogLevel  string `yaml:"log_level"`
	LogFormat string `yaml:"log_format"`

	// OpenTelemetry configuration
	OTelEnabled     bool    `yaml:"otel_enabled"`
	TraceEndpoint   string  `yaml:"trace_endpoint"`
	MetricsEndpoint string  `yaml:"metrics_endpoint"`
	SamplingRate    float64 `yaml:"sampling_rate"`

	// Profiling configuration
	ProfilingEnabled  bool          `yaml:"profiling_enabled"`
	ProfilingInterval time.Duration `yaml:"profiling_interval"`

	// Metrics configuration
	MetricsEnabled bool `yaml:"metrics_enabled"`

	// Performance monitoring
	PerformanceMonitoringEnabled bool               `yaml:"performance_monitoring_enabled"`
	BaselineMetrics              map[string]float64 `yaml:"baseline_metrics"`
	RegressionThresholds         map[string]float64 `yaml:"regression_thresholds"`
}

// ObservabilityStack contains all observability components
type ObservabilityStack struct {
	Logger             *slog.Logger
	OTelSetup          *OTelSetup
	OTelShutdown       func(context.Context) error
	Metrics            *Metrics
	PerformanceMonitor *PerformanceMonitor
	ProfileManager     *ProfileManager
	DatabaseTracer     *DatabaseTracer
	AITracer           *AITracer
	ToolTracer         *ToolTracer
	Config             *ObservabilityConfig
}

// Initialize sets up the complete observability stack following golang_guide.md best practices
func Initialize(config *ObservabilityConfig) (*ObservabilityStack, error) {
	// Set default values if not provided
	if config.ServiceName == "" {
		config.ServiceName = "assistant-go"
	}
	if config.ServiceVersion == "" {
		config.ServiceVersion = "1.0.0"
	}
	if config.Environment == "" {
		config.Environment = "development"
	}
	if config.LogLevel == "" {
		config.LogLevel = "info"
	}
	if config.LogFormat == "" {
		config.LogFormat = "json"
	}
	if config.SamplingRate == 0 {
		config.SamplingRate = 0.01 // 1% default sampling
	}
	if config.ProfilingInterval == 0 {
		config.ProfilingInterval = 5 * time.Minute
	}

	stack := &ObservabilityStack{
		Config: config,
	}

	// 1. Initialize structured logging (golang_guide.md priority: critical for debugging)
	logger := SetupLogging(config.LogLevel, config.LogFormat)

	// Add service context to all logs
	logger = logger.With(
		slog.String("service", config.ServiceName),
		slog.String("version", config.ServiceVersion),
		slog.String("environment", config.Environment),
	)

	stack.Logger = logger
	logger.Info("Observability stack initialization started",
		slog.String("service", config.ServiceName),
		slog.String("version", config.ServiceVersion))

	// 2. Initialize OpenTelemetry (golang_guide.md: traces stable, metrics beta - both production ready)
	if config.OTelEnabled {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		otelSetup, shutdown, err := SetupOTelSDK(ctx, config.ServiceName, config.ServiceVersion, logger)
		if err != nil {
			logger.Warn("Failed to initialize OpenTelemetry", slog.Any("error", err))
		} else {
			stack.OTelSetup = otelSetup
			stack.OTelShutdown = shutdown
			logger.Info("OpenTelemetry initialized successfully")
		}

		// Initialize tracers for different components
		stack.DatabaseTracer = NewDatabaseTracer(config.ServiceName)
		stack.AITracer = NewAITracer(config.ServiceName)
		stack.ToolTracer = NewToolTracer(config.ServiceName)
	}

	// 3. Initialize metrics (golang_guide.md: implement RED method)
	if config.MetricsEnabled {
		metrics, err := NewMetrics(config.ServiceName)
		if err != nil {
			logger.Warn("Failed to initialize metrics", slog.Any("error", err))
		} else {
			stack.Metrics = metrics
			logger.Info("Metrics system initialized successfully")
		}
	}

	// 4. Initialize performance monitoring
	if config.PerformanceMonitoringEnabled {
		stack.PerformanceMonitor = NewPerformanceMonitor(logger)

		// Set up performance baselines if provided
		if len(config.BaselineMetrics) > 0 && len(config.RegressionThresholds) > 0 {
			regression := NewPerformanceRegression(stack.PerformanceMonitor.GetCollector(), logger)
			for metric, baseline := range config.BaselineMetrics {
				if threshold, exists := config.RegressionThresholds[metric]; exists {
					regression.SetBaseline(metric, baseline, threshold)
				}
			}
		}

		logger.Info("Performance monitoring initialized successfully")
	}

	// 5. Initialize profiling (golang_guide.md: use for production debugging)
	stack.ProfileManager = NewProfileManager(logger)
	if config.ProfilingEnabled {
		stack.ProfileManager.EnableProfiling(config.ProfilingInterval)
		logger.Info("Profiling enabled", slog.Duration("interval", config.ProfilingInterval))
	}

	logger.Info("Observability stack initialized successfully",
		slog.Bool("otel_enabled", config.OTelEnabled),
		slog.Bool("metrics_enabled", config.MetricsEnabled),
		slog.Bool("profiling_enabled", config.ProfilingEnabled),
		slog.Bool("performance_monitoring", config.PerformanceMonitoringEnabled))

	return stack, nil
}

// Shutdown gracefully shuts down the observability stack
func (stack *ObservabilityStack) Shutdown(ctx context.Context) error {
	var shutdownErr error

	stack.Logger.Info("Shutting down observability stack")

	// Shutdown OpenTelemetry
	if stack.OTelShutdown != nil {
		if err := stack.OTelShutdown(ctx); err != nil {
			shutdownErr = fmt.Errorf("failed to shutdown OpenTelemetry: %w", err)
			stack.Logger.Error("OpenTelemetry shutdown failed", slog.Any("error", err))
		} else {
			stack.Logger.Info("OpenTelemetry shutdown completed")
		}
	}

	// Disable profiling
	if stack.ProfileManager != nil {
		stack.ProfileManager.DisableProfiling()
		stack.Logger.Info("Profiling disabled")
	}

	stack.Logger.Info("Observability stack shutdown completed")

	return shutdownErr
}

// StartBackgroundServices starts background observability services
func (stack *ObservabilityStack) StartBackgroundServices(ctx context.Context) {
	if stack.Config.ProfilingEnabled && stack.ProfileManager != nil {
		stack.ProfileManager.StartPeriodicProfiling(ctx)
	}

	stack.Logger.Info("Background observability services started")
}

// GetHealthStatus returns the health status of observability components
func (stack *ObservabilityStack) GetHealthStatus(ctx context.Context) map[string]interface{} {
	status := map[string]interface{}{
		"service":    stack.Config.ServiceName,
		"version":    stack.Config.ServiceVersion,
		"timestamp":  time.Now().Format(time.RFC3339),
		"components": map[string]interface{}{},
	}

	components := status["components"].(map[string]interface{})

	// Logging health (always healthy if we can log)
	components["logging"] = map[string]interface{}{
		"status": "healthy",
		"level":  stack.Config.LogLevel,
		"format": stack.Config.LogFormat,
	}

	// OpenTelemetry health
	if stack.Config.OTelEnabled {
		components["opentelemetry"] = map[string]interface{}{
			"status":          "healthy",
			"tracing_enabled": stack.OTelSetup != nil,
			"sampling_rate":   stack.Config.SamplingRate,
		}
	}

	// Metrics health
	if stack.Config.MetricsEnabled && stack.Metrics != nil {
		components["metrics"] = map[string]interface{}{
			"status": "healthy",
			"type":   "opentelemetry",
		}
	}

	// Performance monitoring health
	if stack.Config.PerformanceMonitoringEnabled && stack.PerformanceMonitor != nil {
		if err := stack.PerformanceMonitor.Health(ctx); err == nil {
			components["performance_monitoring"] = map[string]interface{}{
				"status": "healthy",
				"stats":  stack.PerformanceMonitor.GetSystemStats(),
			}
		} else {
			components["performance_monitoring"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		}
	}

	// Profiling health
	if stack.ProfileManager != nil {
		components["profiling"] = map[string]interface{}{
			"status":  "healthy",
			"enabled": stack.Config.ProfilingEnabled,
			"stats":   stack.ProfileManager.GetProfileStats(),
		}
	}

	return status
}

// NewDefaultConfig creates a default observability configuration
func NewDefaultConfig() *ObservabilityConfig {
	return &ObservabilityConfig{
		ServiceName:                  "assistant-go",
		ServiceVersion:               "1.0.0",
		Environment:                  getEnvOrDefault("ENVIRONMENT", "development"),
		LogLevel:                     getEnvOrDefault("LOG_LEVEL", "info"),
		LogFormat:                    getEnvOrDefault("LOG_FORMAT", "json"),
		OTelEnabled:                  getEnvBoolOrDefault("OTEL_ENABLED", true),
		TraceEndpoint:                getEnvOrDefault("OTEL_TRACE_ENDPOINT", "http://localhost:4318"),
		MetricsEndpoint:              getEnvOrDefault("OTEL_METRICS_ENDPOINT", "http://localhost:4318"),
		SamplingRate:                 0.01, // 1% sampling for production
		ProfilingEnabled:             getEnvBoolOrDefault("PROFILING_ENABLED", false),
		ProfilingInterval:            5 * time.Minute,
		MetricsEnabled:               getEnvBoolOrDefault("METRICS_ENABLED", true),
		PerformanceMonitoringEnabled: getEnvBoolOrDefault("PERFORMANCE_MONITORING_ENABLED", true),
		BaselineMetrics:              make(map[string]float64),
		RegressionThresholds:         make(map[string]float64),
	}
}

// NewProductionConfig creates a production-optimized observability configuration
func NewProductionConfig() *ObservabilityConfig {
	config := NewDefaultConfig()
	config.Environment = "production"
	config.LogLevel = "info"
	config.LogFormat = "json"
	config.OTelEnabled = true
	config.SamplingRate = 0.01 // 1% sampling for performance
	config.ProfilingEnabled = true
	config.ProfilingInterval = 10 * time.Minute // Less frequent in production
	config.MetricsEnabled = true
	config.PerformanceMonitoringEnabled = true

	// Set up common performance baselines and thresholds
	config.BaselineMetrics = map[string]float64{
		"request_duration_p95": 1.0,   // 1 second P95 baseline
		"memory_usage_mb":      500.0, // 500MB memory baseline
		"error_rate":           0.01,  // 1% error rate baseline
	}
	config.RegressionThresholds = map[string]float64{
		"request_duration_p95": 50.0,  // 50% increase threshold
		"memory_usage_mb":      30.0,  // 30% increase threshold
		"error_rate":           100.0, // 100% increase (double) threshold
	}

	return config
}

// Helper functions for environment variable parsing
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}
