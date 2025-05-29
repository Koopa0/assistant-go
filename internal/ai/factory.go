package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/koopa0/assistant/internal/config"
	"github.com/koopa0/assistant/internal/observability"
	"github.com/koopa0/assistant/internal/ratelimit"
)

// ProviderConstructor is a function that creates a provider instance
type ProviderConstructor func(config ProviderConfig, logger *slog.Logger) (Provider, error)

// Factory implements ProviderFactory interface
type Factory struct {
	logger       *slog.Logger
	providers    map[string]Provider
	constructors map[string]ProviderConstructor
	mutex        sync.RWMutex
}

// NewFactory creates a new AI provider factory
func NewFactory(logger *slog.Logger) *Factory {
	return &Factory{
		logger:       logger,
		providers:    make(map[string]Provider),
		constructors: make(map[string]ProviderConstructor),
	}
}

// RegisterProvider registers a provider constructor
func (f *Factory) RegisterProvider(name string, constructor ProviderConstructor) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.constructors[name] = constructor
}

// CreateProvider creates an AI provider instance
func (f *Factory) CreateProvider(name string, config ProviderConfig) (Provider, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// Check if provider already exists
	if provider, exists := f.providers[name]; exists {
		return provider, nil
	}

	// Get constructor
	constructor, exists := f.constructors[name]
	if !exists {
		return nil, fmt.Errorf("unsupported provider: %s", name)
	}

	// Create provider
	provider, err := constructor(config, f.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s provider: %w", name, err)
	}

	// Store provider instance
	f.providers[name] = provider

	f.logger.Info("AI provider created",
		slog.String("provider", name),
		slog.String("model", config.Model))

	return provider, nil
}

// SupportedProviders returns a list of supported providers
func (f *Factory) SupportedProviders() []string {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	providers := make([]string, 0, len(f.constructors))
	for name := range f.constructors {
		providers = append(providers, name)
	}
	return providers
}

// GetProvider returns an existing provider instance
func (f *Factory) GetProvider(name string) (Provider, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	provider, exists := f.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// CloseAll closes all provider instances
func (f *Factory) CloseAll() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	var lastErr error
	for name, provider := range f.providers {
		if err := provider.Close(nil); err != nil {
			f.logger.Error("Failed to close provider",
				slog.String("provider", name),
				slog.Any("error", err))
			lastErr = err
		}
	}

	// Clear providers map
	f.providers = make(map[string]Provider)

	return lastErr
}

// Manager manages AI providers and provides a unified interface
type Manager struct {
	factory         *Factory
	config          *config.Config
	logger          *slog.Logger
	defaultProvider string
	metrics         *observability.AIMetrics
	rateLimiter     *ratelimit.AIRateLimiter
}

// NewManager creates a new AI manager
func NewManager(cfg *config.Config, logger *slog.Logger) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	factory := NewFactory(logger)

	// Initialize metrics
	metricsCollector := observability.NewMetricsCollector(logger)
	aiMetrics := observability.NewAIMetrics(metricsCollector, logger)

	// Initialize rate limiter
	aiRateLimiter := ratelimit.NewAIRateLimiter(logger)

	manager := &Manager{
		factory:         factory,
		config:          cfg,
		logger:          logger,
		defaultProvider: cfg.AI.DefaultProvider,
		metrics:         aiMetrics,
		rateLimiter:     aiRateLimiter,
	}

	// Initialize providers based on configuration
	if err := manager.initializeProviders(); err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	return manager, nil
}

// initializeProviders initializes AI providers based on configuration
func (m *Manager) initializeProviders() error {
	// Register all available providers
	RegisterProviders(m.factory)

	// Initialize Claude if API key is provided
	if m.config.AI.Claude.APIKey != "" {
		claudeConfig := ProviderConfig{
			APIKey:      m.config.AI.Claude.APIKey,
			BaseURL:     m.config.AI.Claude.BaseURL,
			Model:       m.config.AI.Claude.Model,
			MaxTokens:   m.config.AI.Claude.MaxTokens,
			Temperature: m.config.AI.Claude.Temperature,
			Timeout:     30 * time.Second, // TODO: Add timeout to config
		}

		_, err := m.factory.CreateProvider("claude", claudeConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize Claude provider: %w", err)
		}

		// TODO: LangChain integration will be implemented in a future phase
		// if m.config.Tools.LangChain.EnableMemory {
		//     if err := m.initializeLangChainProvider("claude", m.config.AI.Claude); err != nil {
		//         m.logger.Warn("Failed to initialize LangChain-Claude provider", slog.Any("error", err))
		//     } else {
		//         m.logger.Info("LangChain-Claude provider initialized")
		//     }
		// }
	}

	// Initialize Gemini if API key is provided
	if m.config.AI.Gemini.APIKey != "" {
		geminiConfig := ProviderConfig{
			APIKey:      m.config.AI.Gemini.APIKey,
			BaseURL:     m.config.AI.Gemini.BaseURL,
			Model:       m.config.AI.Gemini.Model,
			MaxTokens:   m.config.AI.Gemini.MaxTokens,
			Temperature: m.config.AI.Gemini.Temperature,
			Timeout:     30 * time.Second, // TODO: Add timeout to config
		}

		_, err := m.factory.CreateProvider("gemini", geminiConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize Gemini provider: %w", err)
		}

		// TODO: LangChain integration will be implemented in a future phase
		// if m.config.Tools.LangChain.EnableMemory {
		//     if err := m.initializeLangChainProvider("gemini", m.config.AI.Gemini); err != nil {
		//         m.logger.Warn("Failed to initialize LangChain-Gemini provider", slog.Any("error", err))
		//     } else {
		//         m.logger.Info("LangChain-Gemini provider initialized")
		//     }
		// }
	}

	// Validate that at least one provider is available
	if len(m.factory.providers) == 0 {
		return fmt.Errorf("no AI providers configured - at least one provider (Claude or Gemini) must be configured")
	}

	// Validate default provider
	if _, err := m.factory.GetProvider(m.defaultProvider); err != nil {
		return fmt.Errorf("default provider %s is not available: %w", m.defaultProvider, err)
	}

	m.logger.Info("AI providers initialized",
		slog.Int("provider_count", len(m.factory.providers)),
		slog.String("default_provider", m.defaultProvider))

	return nil
}

// TODO: LangChain integration will be implemented in a future phase
// initializeLangChainProvider initializes a LangChain-enhanced provider
// func (m *Manager) initializeLangChainProvider(provider string, aiConfig interface{}) error {
//     // Implementation will be added when LangChain integration is ready
//     return nil
// }

// GenerateResponse generates a response using the specified or default provider
func (m *Manager) GenerateResponse(ctx context.Context, request *GenerateRequest, providerName ...string) (*GenerateResponse, error) {
	startTime := time.Now()

	// Determine which provider to use
	provider := m.defaultProvider
	if len(providerName) > 0 && providerName[0] != "" {
		provider = providerName[0]
	}

	// Estimate token usage for rate limiting
	estimatedTokens := m.estimateTokenUsage(request)

	// Check rate limits
	if err := m.rateLimiter.CheckRequest(ctx, provider, request.Model, estimatedTokens); err != nil {
		// Record rate limit error
		m.metrics.RecordRequest(provider, request.Model, startTime, false, nil, "rate_limited")
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Get provider instance
	providerInstance, err := m.factory.GetProvider(provider)
	if err != nil {
		// Record error metrics
		m.metrics.RecordRequest(provider, request.Model, startTime, false, nil, "provider_not_found")
		return nil, fmt.Errorf("failed to get provider %s: %w", provider, err)
	}

	// Generate response
	response, err := providerInstance.GenerateResponse(ctx, request)
	if err != nil {
		// Record error metrics
		errorType := "generation_failed"
		if providerErr, ok := err.(*ProviderError); ok {
			errorType = providerErr.Type
		}
		m.metrics.RecordRequest(provider, request.Model, startTime, false, nil, errorType)
		return nil, err
	}

	// Record actual usage for rate limiting
	m.rateLimiter.RecordUsage(ctx, provider, int64(response.TokensUsed.TotalTokens))

	// Record success metrics
	tokenUsage := map[string]int{
		"input":  response.TokensUsed.InputTokens,
		"output": response.TokensUsed.OutputTokens,
		"total":  response.TokensUsed.TotalTokens,
	}
	m.metrics.RecordRequest(provider, response.Model, startTime, true, tokenUsage, "")

	return response, nil
}

// GenerateEmbedding generates embeddings using the specified or default provider
func (m *Manager) GenerateEmbedding(ctx context.Context, text string, providerName ...string) (*EmbeddingResponse, error) {
	startTime := time.Now()

	// Determine which provider to use
	provider := m.defaultProvider
	if len(providerName) > 0 && providerName[0] != "" {
		provider = providerName[0]
	}

	// Get provider instance
	providerInstance, err := m.factory.GetProvider(provider)
	if err != nil {
		// Record error metrics
		m.metrics.RecordEmbedding(provider, "", startTime, false, 0, 0, "provider_not_found")
		return nil, fmt.Errorf("failed to get provider %s: %w", provider, err)
	}

	// Generate embedding
	response, err := providerInstance.GenerateEmbedding(ctx, text)
	if err != nil {
		// Record error metrics
		errorType := "generation_failed"
		if providerErr, ok := err.(*ProviderError); ok {
			errorType = providerErr.Type
		}
		m.metrics.RecordEmbedding(provider, "", startTime, false, 0, 0, errorType)
		return nil, err
	}

	// Record success metrics
	m.metrics.RecordEmbedding(provider, response.Model, startTime, true, response.TokensUsed, len(response.Embedding), "")

	return response, nil
}

// GetProvider returns a specific provider instance
func (m *Manager) GetProvider(name string) (Provider, error) {
	return m.factory.GetProvider(name)
}

// GetAvailableProviders returns a list of available providers
func (m *Manager) GetAvailableProviders() []string {
	m.factory.mutex.RLock()
	defer m.factory.mutex.RUnlock()

	providers := make([]string, 0, len(m.factory.providers))
	for name := range m.factory.providers {
		providers = append(providers, name)
	}

	return providers
}

// GetDefaultProvider returns the default provider name
func (m *Manager) GetDefaultProvider() string {
	return m.defaultProvider
}

// SetDefaultProvider sets the default provider
func (m *Manager) SetDefaultProvider(name string) error {
	if _, err := m.factory.GetProvider(name); err != nil {
		return fmt.Errorf("provider %s is not available: %w", name, err)
	}

	m.defaultProvider = name
	m.logger.Info("Default provider changed", slog.String("provider", name))
	return nil
}

// Health checks the health of all providers
func (m *Manager) Health(ctx context.Context) error {
	m.factory.mutex.RLock()
	providers := make(map[string]Provider)
	for name, provider := range m.factory.providers {
		providers[name] = provider
	}
	m.factory.mutex.RUnlock()

	for name, provider := range providers {
		if err := provider.Health(ctx); err != nil {
			return fmt.Errorf("provider %s health check failed: %w", name, err)
		}
	}

	return nil
}

// GetUsageStats returns usage statistics for all providers
func (m *Manager) GetUsageStats(ctx context.Context) (map[string]*UsageStats, error) {
	m.factory.mutex.RLock()
	providers := make(map[string]Provider)
	for name, provider := range m.factory.providers {
		providers[name] = provider
	}
	m.factory.mutex.RUnlock()

	stats := make(map[string]*UsageStats)
	for name, provider := range providers {
		providerStats, err := provider.GetUsage(ctx)
		if err != nil {
			m.logger.Warn("Failed to get usage stats for provider",
				slog.String("provider", name),
				slog.Any("error", err))
			continue
		}
		stats[name] = providerStats
	}

	return stats, nil
}

// GetMetrics returns AI performance metrics
func (m *Manager) GetMetrics() map[string]interface{} {
	return m.metrics.GetAIStats()
}

// estimateTokenUsage estimates token usage for rate limiting
func (m *Manager) estimateTokenUsage(request *GenerateRequest) int64 {
	// Simple estimation based on message content length
	// This is a rough approximation - real implementations might use tokenizers
	totalChars := 0
	for _, msg := range request.Messages {
		totalChars += len(msg.Content)
	}

	// Rough estimation: 1 token ≈ 4 characters for English text
	estimatedInputTokens := int64(totalChars / 4)

	// Estimate output tokens based on MaxTokens or a default
	estimatedOutputTokens := int64(request.MaxTokens)
	if estimatedOutputTokens == 0 {
		estimatedOutputTokens = 1000 // Default estimate
	}

	return estimatedInputTokens + estimatedOutputTokens
}

// Close closes all providers
func (m *Manager) Close(ctx context.Context) error {
	return m.factory.CloseAll()
}
