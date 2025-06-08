package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// RateLimitType represents the type of rate limit
type RateLimitType string

const (
	RateLimitTypeRequests RateLimitType = "requests"
	RateLimitTypeTokens   RateLimitType = "tokens"
	RateLimitTypeBytes    RateLimitType = "bytes"
)

// RateLimit represents a rate limit configuration
type RateLimit struct {
	Type     RateLimitType `json:"type"`
	Limit    int64         `json:"limit"`
	Window   time.Duration `json:"window"`
	BurstMax int64         `json:"burst_max,omitempty"`
}

// Usage represents current usage for a rate limit
type Usage struct {
	Count     int64     `json:"count"`
	ResetTime time.Time `json:"reset_time"`
	LastUsed  time.Time `json:"last_used"`
}

// RateLimiter manages rate limiting for different resources
type RateLimiter struct {
	limits map[string]*RateLimit
	usage  map[string]*Usage
	mutex  sync.RWMutex
	logger *slog.Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(logger *slog.Logger) *RateLimiter {
	return &RateLimiter{
		limits: make(map[string]*RateLimit),
		usage:  make(map[string]*Usage),
		logger: logger,
	}
}

// AddLimit adds a rate limit for a specific key
func (rl *RateLimiter) AddLimit(key string, limit *RateLimit) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.limits[key] = limit
	rl.logger.Debug("Rate limit added",
		slog.String("key", key),
		slog.String("type", string(limit.Type)),
		slog.Int64("limit", limit.Limit),
		slog.Duration("window", limit.Window))
}

// CheckLimit checks if a request is within rate limits
func (rl *RateLimiter) CheckLimit(ctx context.Context, key string, cost int64) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	limit, exists := rl.limits[key]
	if !exists {
		// No limit configured, allow request
		return nil
	}

	now := time.Now()
	usage, exists := rl.usage[key]

	if !exists || now.After(usage.ResetTime) {
		// First request or window has reset
		rl.usage[key] = &Usage{
			Count:     cost,
			ResetTime: now.Add(limit.Window),
			LastUsed:  now,
		}
		rl.logger.Debug("Rate limit initialized",
			slog.String("key", key),
			slog.Int64("cost", cost),
			slog.Time("reset_time", rl.usage[key].ResetTime))
		return nil
	}

	// Check if adding this cost would exceed the limit
	if usage.Count+cost > limit.Limit {
		remaining := limit.Limit - usage.Count
		if remaining < 0 {
			remaining = 0
		}

		rl.logger.Warn("Rate limit exceeded",
			slog.String("key", key),
			slog.Int64("current", usage.Count),
			slog.Int64("cost", cost),
			slog.Int64("limit", limit.Limit),
			slog.Int64("remaining", remaining),
			slog.Time("reset_time", usage.ResetTime))

		return &RateLimitError{
			Key:        key,
			Type:       limit.Type,
			Current:    usage.Count,
			Limit:      limit.Limit,
			Cost:       cost,
			ResetTime:  usage.ResetTime,
			RetryAfter: time.Until(usage.ResetTime),
		}
	}

	// Update usage
	usage.Count += cost
	usage.LastUsed = now

	rl.logger.Debug("Rate limit checked",
		slog.String("key", key),
		slog.Int64("cost", cost),
		slog.Int64("current", usage.Count),
		slog.Int64("limit", limit.Limit),
		slog.Int64("remaining", limit.Limit-usage.Count))

	return nil
}

// GetUsage returns current usage for a key
func (rl *RateLimiter) GetUsage(key string) *Usage {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	usage, exists := rl.usage[key]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	return &Usage{
		Count:     usage.Count,
		ResetTime: usage.ResetTime,
		LastUsed:  usage.LastUsed,
	}
}

// GetAllUsage returns usage for all keys
func (rl *RateLimiter) GetAllUsage() map[string]*Usage {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	result := make(map[string]*Usage)
	for key, usage := range rl.usage {
		result[key] = &Usage{
			Count:     usage.Count,
			ResetTime: usage.ResetTime,
			LastUsed:  usage.LastUsed,
		}
	}

	return result
}

// Reset resets usage for a specific key
func (rl *RateLimiter) Reset(key string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	delete(rl.usage, key)
	rl.logger.Debug("Rate limit reset", slog.String("key", key))
}

// ResetAll resets all usage counters
func (rl *RateLimiter) ResetAll() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.usage = make(map[string]*Usage)
	rl.logger.Debug("All rate limits reset")
}

// Cleanup removes expired usage entries
func (rl *RateLimiter) Cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	cleaned := 0

	for key, usage := range rl.usage {
		if now.After(usage.ResetTime.Add(time.Hour)) { // Keep for 1 hour after reset
			delete(rl.usage, key)
			cleaned++
		}
	}

	if cleaned > 0 {
		rl.logger.Debug("Rate limit cleanup completed",
			slog.Int("cleaned", cleaned))
	}
}

// RateLimitError represents a rate limit exceeded error
type RateLimitError struct {
	Key        string        `json:"key"`
	Type       RateLimitType `json:"type"`
	Current    int64         `json:"current"`
	Limit      int64         `json:"limit"`
	Cost       int64         `json:"cost"`
	ResetTime  time.Time     `json:"reset_time"`
	RetryAfter time.Duration `json:"retry_after"`
}

// Error implements the error interface
func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded for %s: %d/%d %s (cost: %d, retry after: %v)",
		e.Key, e.Current, e.Limit, e.Type, e.Cost, e.RetryAfter)
}

// IsRetryable returns true if the request can be retried after the reset time
func (e *RateLimitError) IsRetryable() bool {
	return true
}

// AIRateLimiter manages rate limits specifically for AI providers
type AIRateLimiter struct {
	limiter *RateLimiter
	logger  *slog.Logger
}

// NewAIRateLimiter creates a new AI rate limiter with default limits
func NewAIRateLimiter(logger *slog.Logger) *AIRateLimiter {
	limiter := NewRateLimiter(logger)

	// Add default rate limits for common AI providers
	// These can be overridden based on actual provider limits
	limiter.AddLimit("claude:requests", &RateLimit{
		Type:   RateLimitTypeRequests,
		Limit:  1000,
		Window: time.Hour,
	})

	limiter.AddLimit("claude:tokens", &RateLimit{
		Type:   RateLimitTypeTokens,
		Limit:  100000,
		Window: time.Hour,
	})

	limiter.AddLimit("gemini:requests", &RateLimit{
		Type:   RateLimitTypeRequests,
		Limit:  1500,
		Window: time.Hour,
	})

	limiter.AddLimit("gemini:tokens", &RateLimit{
		Type:   RateLimitTypeTokens,
		Limit:  150000,
		Window: time.Hour,
	})

	return &AIRateLimiter{
		limiter: limiter,
		logger:  logger,
	}
}

// CheckRequest checks if an AI request is within rate limits
func (arl *AIRateLimiter) CheckRequest(ctx context.Context, provider, model string, estimatedTokens int64) error {
	// Check request rate limit
	requestKey := fmt.Sprintf("%s:requests", provider)
	if err := arl.limiter.CheckLimit(ctx, requestKey, 1); err != nil {
		return err
	}

	// Check token rate limit if tokens are provided
	if estimatedTokens > 0 {
		tokenKey := fmt.Sprintf("%s:tokens", provider)
		if err := arl.limiter.CheckLimit(ctx, tokenKey, estimatedTokens); err != nil {
			return err
		}
	}

	arl.logger.Debug("AI rate limit check passed",
		slog.String("provider", provider),
		slog.String("model", model),
		slog.Int64("estimated_tokens", estimatedTokens))

	return nil
}

// RecordUsage records actual usage after a successful request
func (arl *AIRateLimiter) RecordUsage(ctx context.Context, provider string, actualTokens int64) {
	if actualTokens <= 0 {
		return
	}

	// The request was already counted in CheckRequest, so we only need to
	// adjust token usage if it differs from the estimate
	tokenKey := fmt.Sprintf("%s:tokens", provider)

	arl.logger.Debug("AI usage recorded",
		slog.String("provider", provider),
		slog.Int64("actual_tokens", actualTokens),
		slog.String("token_key", tokenKey))
}

// GetProviderUsage returns usage statistics for a provider
func (arl *AIRateLimiter) GetProviderUsage(provider string) map[string]*Usage {
	result := make(map[string]*Usage)

	requestKey := fmt.Sprintf("%s:requests", provider)
	if usage := arl.limiter.GetUsage(requestKey); usage != nil {
		result["requests"] = usage
	}

	tokenKey := fmt.Sprintf("%s:tokens", provider)
	if usage := arl.limiter.GetUsage(tokenKey); usage != nil {
		result["tokens"] = usage
	}

	return result
}

// UpdateLimits updates rate limits for a provider
func (arl *AIRateLimiter) UpdateLimits(provider string, requestLimit, tokenLimit int64, window time.Duration) {
	if requestLimit > 0 {
		arl.limiter.AddLimit(fmt.Sprintf("%s:requests", provider), &RateLimit{
			Type:   RateLimitTypeRequests,
			Limit:  requestLimit,
			Window: window,
		})
	}

	if tokenLimit > 0 {
		arl.limiter.AddLimit(fmt.Sprintf("%s:tokens", provider), &RateLimit{
			Type:   RateLimitTypeTokens,
			Limit:  tokenLimit,
			Window: window,
		})
	}

	arl.logger.Info("AI rate limits updated",
		slog.String("provider", provider),
		slog.Int64("request_limit", requestLimit),
		slog.Int64("token_limit", tokenLimit),
		slog.Duration("window", window))
}

// StartCleanupRoutine starts a background routine to clean up expired entries
func (arl *AIRateLimiter) StartCleanupRoutine(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				arl.logger.Debug("Rate limiter cleanup routine stopped")
				return
			case <-ticker.C:
				arl.limiter.Cleanup()
			}
		}
	}()

	arl.logger.Debug("Rate limiter cleanup routine started",
		slog.Duration("interval", interval))
}
