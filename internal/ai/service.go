package ai

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
)

// Service provides a unified AI service using direct dependencies
// Replaces the Factory/Manager anti-pattern with simple composition
type Service struct {
	providers       map[string]Provider
	defaultProvider string
	logger          *slog.Logger
}

// NewService creates a new AI service with direct dependencies
// Following Go's "accept interfaces, return structs" principle
func NewService(cfg *config.Config, logger *slog.Logger) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	svc := &Service{
		providers:       make(map[string]Provider),
		defaultProvider: cfg.AI.DefaultProvider,
		logger:          logger,
	}

	// Initialize providers using the existing factory
	factory := NewFactory(logger)
	RegisterProviders(factory)

	// Initialize Claude if configured
	if cfg.AI.Claude.APIKey != "" {
		claudeConfig := ProviderConfig{
			APIKey:      cfg.AI.Claude.APIKey,
			BaseURL:     cfg.AI.Claude.BaseURL,
			Model:       cfg.AI.Claude.Model,
			MaxTokens:   cfg.AI.Claude.MaxTokens,
			Temperature: cfg.AI.Claude.Temperature,
			Timeout:     30 * time.Second,
		}

		claudeProvider, err := factory.CreateProvider("claude", claudeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Claude provider: %w", err)
		}
		svc.providers["claude"] = claudeProvider
		logger.Info("Claude provider initialized")
	}

	// Initialize Gemini if configured
	if cfg.AI.Gemini.APIKey != "" {
		geminiConfig := ProviderConfig{
			APIKey:      cfg.AI.Gemini.APIKey,
			BaseURL:     cfg.AI.Gemini.BaseURL,
			Model:       cfg.AI.Gemini.Model,
			MaxTokens:   cfg.AI.Gemini.MaxTokens,
			Temperature: cfg.AI.Gemini.Temperature,
			Timeout:     30 * time.Second,
		}

		geminiProvider, err := factory.CreateProvider("gemini", geminiConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Gemini provider: %w", err)
		}
		svc.providers["gemini"] = geminiProvider
		logger.Info("Gemini provider initialized")
	}

	// Validate at least one provider is available
	if len(svc.providers) == 0 {
		return nil, fmt.Errorf("no AI providers configured")
	}

	// Validate default provider is available
	if !svc.isProviderAvailable(svc.defaultProvider) {
		return nil, fmt.Errorf("default provider %s is not available", svc.defaultProvider)
	}

	logger.Info("AI service initialized",
		slog.String("default_provider", svc.defaultProvider),
		slog.Int("provider_count", len(svc.providers)),
		slog.Any("available_providers", svc.GetAvailableProviders()))

	return svc, nil
}

// GenerateResponse generates a response using the specified or default provider
func (s *Service) GenerateResponse(ctx context.Context, request *GenerateRequest, providerName ...string) (*GenerateResponse, error) {
	provider := s.defaultProvider
	if len(providerName) > 0 && providerName[0] != "" {
		provider = providerName[0]
	}

	providerInstance, exists := s.providers[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not available", provider)
	}

	return providerInstance.GenerateResponse(ctx, request)
}

// GenerateEmbedding generates embeddings using the specified or default provider
func (s *Service) GenerateEmbedding(ctx context.Context, text string, providerName ...string) (*EmbeddingResponse, error) {
	provider := s.defaultProvider
	if len(providerName) > 0 && providerName[0] != "" {
		provider = providerName[0]
	}

	providerInstance, exists := s.providers[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not available", provider)
	}

	return providerInstance.GenerateEmbedding(ctx, text)
}

// GetAvailableProviders returns a list of available providers
func (s *Service) GetAvailableProviders() []string {
	var providers []string
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}

// GetDefaultProvider returns the default provider name
func (s *Service) GetDefaultProvider() string {
	return s.defaultProvider
}

// SetDefaultProvider sets the default provider
func (s *Service) SetDefaultProvider(name string) error {
	if !s.isProviderAvailable(name) {
		return fmt.Errorf("provider %s is not available", name)
	}

	s.defaultProvider = name
	s.logger.Info("Default provider changed", slog.String("provider", name))
	return nil
}

// Health checks the health of all available providers
func (s *Service) Health(ctx context.Context) error {
	for name, provider := range s.providers {
		if err := provider.Health(ctx); err != nil {
			return fmt.Errorf("%s health check failed: %w", name, err)
		}
	}
	return nil
}

// GetUsageStats returns usage statistics for all providers
func (s *Service) GetUsageStats(ctx context.Context) (map[string]*UsageStats, error) {
	stats := make(map[string]*UsageStats)

	for name, provider := range s.providers {
		providerStats, err := provider.GetUsage(ctx)
		if err != nil {
			s.logger.Warn("Failed to get usage stats for provider",
				slog.String("provider", name),
				slog.Any("error", err))
			continue
		}
		stats[name] = providerStats
	}

	return stats, nil
}

// Close closes all provider connections
func (s *Service) Close(ctx context.Context) error {
	var lastErr error

	for name, provider := range s.providers {
		if err := provider.Close(ctx); err != nil {
			s.logger.Error("Failed to close provider",
				slog.String("provider", name),
				slog.Any("error", err))
			lastErr = err
		}
	}

	return lastErr
}

// Private helper methods

func (s *Service) isProviderAvailable(provider string) bool {
	_, exists := s.providers[provider]
	return exists
}
