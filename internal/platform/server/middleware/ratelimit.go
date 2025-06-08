package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/platform/ratelimit"
)

// RateLimitMiddleware creates a middleware for HTTP rate limiting
type RateLimitMiddleware struct {
	limiter *ratelimit.RateLimiter
	logger  *slog.Logger
	config  RateLimitConfig
}

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// Global rate limit for all requests
	RequestsPerSecond int
	BurstSize         int

	// Per-endpoint rate limits
	EndpointLimits map[string]EndpointLimit

	// Whether to use IP-based or user-based limiting
	UseIPBased bool

	// Custom key extractor function
	KeyExtractor func(r *http.Request) string
}

// EndpointLimit defines rate limit for a specific endpoint
type EndpointLimit struct {
	RequestsPerMinute int
	BurstSize         int
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(config RateLimitConfig, logger *slog.Logger) *RateLimitMiddleware {
	limiter := ratelimit.NewRateLimiter(logger)

	// Configure global rate limit
	if config.RequestsPerSecond > 0 {
		limiter.AddLimit("global", &ratelimit.RateLimit{
			Type:     ratelimit.RateLimitTypeRequests,
			Limit:    int64(config.RequestsPerSecond),
			Window:   time.Second,
			BurstMax: int64(config.BurstSize),
		})
	}

	// Configure per-endpoint limits
	for endpoint, limit := range config.EndpointLimits {
		limiter.AddLimit(endpoint, &ratelimit.RateLimit{
			Type:     ratelimit.RateLimitTypeRequests,
			Limit:    int64(limit.RequestsPerMinute),
			Window:   time.Minute,
			BurstMax: int64(limit.BurstSize),
		})
	}

	return &RateLimitMiddleware{
		limiter: limiter,
		logger:  logger,
		config:  config,
	}
}

// Handler returns the HTTP middleware handler
func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract rate limit key
		key := m.extractKey(r)

		// Check global rate limit
		if err := m.limiter.CheckLimit(ctx, "global", 1); err != nil {
			m.handleRateLimitError(w, err)
			return
		}

		// Check user/IP specific rate limit
		userKey := key + ":requests"
		if err := m.limiter.CheckLimit(ctx, userKey, 1); err != nil {
			m.handleRateLimitError(w, err)
			return
		}

		// Check endpoint-specific rate limit
		endpoint := m.normalizeEndpoint(r.URL.Path)
		if _, hasLimit := m.config.EndpointLimits[endpoint]; hasLimit {
			if err := m.limiter.CheckLimit(ctx, endpoint, 1); err != nil {
				m.handleRateLimitError(w, err)
				return
			}
		}

		// Add rate limit headers
		m.addRateLimitHeaders(w, key)

		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

// extractKey extracts the rate limit key from the request
func (m *RateLimitMiddleware) extractKey(r *http.Request) string {
	// Use custom key extractor if provided
	if m.config.KeyExtractor != nil {
		return m.config.KeyExtractor(r)
	}

	// Try to get user ID from context (set by auth middleware)
	if userID := r.Context().Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok && id != "" {
			return "user:" + id
		}
	}

	// Fall back to IP-based limiting if configured
	if m.config.UseIPBased {
		return "ip:" + m.getClientIP(r)
	}

	// Default to a generic key
	return "anonymous"
}

// getClientIP extracts the client IP from the request
func (m *RateLimitMiddleware) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}

// normalizeEndpoint normalizes the endpoint path for rate limiting
func (m *RateLimitMiddleware) normalizeEndpoint(path string) string {
	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	// Normalize API version paths
	if strings.HasPrefix(path, "/api/v1/") {
		// Extract the main endpoint
		parts := strings.Split(path, "/")
		if len(parts) >= 4 {
			return "/api/v1/" + parts[3]
		}
	}

	return path
}

// handleRateLimitError handles rate limit errors
func (m *RateLimitMiddleware) handleRateLimitError(w http.ResponseWriter, err error) {
	if rlErr, ok := err.(*ratelimit.RateLimitError); ok {
		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rlErr.Limit))
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", rlErr.ResetTime.Unix()))
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(rlErr.RetryAfter.Seconds())))

		// Return 429 Too Many Requests
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)

		response := map[string]interface{}{
			"error":       "rate_limit_exceeded",
			"message":     "Too many requests. Please retry after some time.",
			"retry_after": int(rlErr.RetryAfter.Seconds()),
			"reset_time":  rlErr.ResetTime.Unix(),
		}

		json.NewEncoder(w).Encode(response)

		m.logger.Warn("Rate limit exceeded",
			slog.String("key", rlErr.Key),
			slog.String("type", string(rlErr.Type)),
			slog.Int64("limit", rlErr.Limit),
			slog.Duration("retry_after", rlErr.RetryAfter))

		return
	}

	// Generic error response
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   "internal_server_error",
		"message": "An unexpected error occurred",
	})
}

// addRateLimitHeaders adds rate limit information headers
func (m *RateLimitMiddleware) addRateLimitHeaders(w http.ResponseWriter, key string) {
	usage := m.limiter.GetUsage(key + ":requests")
	if usage == nil {
		return
	}

	// Get the limit configuration
	// This is simplified - in production, you'd look up the actual limit
	limit := int64(100) // Default limit

	remaining := limit - usage.Count
	if remaining < 0 {
		remaining = 0
	}

	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", usage.ResetTime.Unix()))
}

// APIKeyRateLimitMiddleware provides rate limiting based on API keys
type APIKeyRateLimitMiddleware struct {
	limiter   *ratelimit.RateLimiter
	logger    *slog.Logger
	keyLimits map[string]APIKeyLimit
}

// APIKeyLimit defines rate limits for an API key
type APIKeyLimit struct {
	RequestsPerHour int64
	TokensPerDay    int64
	Tier            string // "free", "basic", "premium", "enterprise"
}

// NewAPIKeyRateLimitMiddleware creates a new API key rate limiter
func NewAPIKeyRateLimitMiddleware(logger *slog.Logger) *APIKeyRateLimitMiddleware {
	limiter := ratelimit.NewRateLimiter(logger)

	// Define default tier limits
	keyLimits := map[string]APIKeyLimit{
		"free": {
			RequestsPerHour: 100,
			TokensPerDay:    10000,
			Tier:            "free",
		},
		"basic": {
			RequestsPerHour: 1000,
			TokensPerDay:    100000,
			Tier:            "basic",
		},
		"premium": {
			RequestsPerHour: 10000,
			TokensPerDay:    1000000,
			Tier:            "premium",
		},
		"enterprise": {
			RequestsPerHour: 100000,
			TokensPerDay:    10000000,
			Tier:            "enterprise",
		},
	}

	return &APIKeyRateLimitMiddleware{
		limiter:   limiter,
		logger:    logger,
		keyLimits: keyLimits,
	}
}

// Handler returns the HTTP middleware handler for API key rate limiting
func (m *APIKeyRateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract API key from header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// Try Authorization header
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if apiKey == "" {
			// No API key provided
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "missing_api_key",
				"message": "API key is required",
			})
			return
		}

		// TODO: Look up API key tier from database
		// For now, use a default tier
		tier := "free"

		// Get limits for tier
		limits, exists := m.keyLimits[tier]
		if !exists {
			limits = m.keyLimits["free"]
		}

		// Configure rate limits for this API key
		requestKey := "apikey:" + apiKey + ":requests"
		m.limiter.AddLimit(requestKey, &ratelimit.RateLimit{
			Type:   ratelimit.RateLimitTypeRequests,
			Limit:  limits.RequestsPerHour,
			Window: time.Hour,
		})

		tokenKey := "apikey:" + apiKey + ":tokens"
		m.limiter.AddLimit(tokenKey, &ratelimit.RateLimit{
			Type:   ratelimit.RateLimitTypeTokens,
			Limit:  limits.TokensPerDay,
			Window: 24 * time.Hour,
		})

		// Check request rate limit
		if err := m.limiter.CheckLimit(r.Context(), requestKey, 1); err != nil {
			m.handleRateLimitError(w, err)
			return
		}

		// Add API key info to context
		ctx := context.WithValue(r.Context(), "api_key", apiKey)
		ctx = context.WithValue(ctx, "api_key_tier", tier)

		// Continue with request
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// handleRateLimitError handles rate limit errors for API keys
func (m *APIKeyRateLimitMiddleware) handleRateLimitError(w http.ResponseWriter, err error) {
	if rlErr, ok := err.(*ratelimit.RateLimitError); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)

		response := map[string]interface{}{
			"error":       "rate_limit_exceeded",
			"message":     "API rate limit exceeded for your key",
			"type":        string(rlErr.Type),
			"retry_after": int(rlErr.RetryAfter.Seconds()),
			"reset_time":  rlErr.ResetTime.Unix(),
		}

		json.NewEncoder(w).Encode(response)
		return
	}

	// Generic error
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   "internal_server_error",
		"message": "An unexpected error occurred",
	})
}
