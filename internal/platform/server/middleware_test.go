package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/server/middleware"
	"log/slog"
)

func TestRateLimitMiddleware(t *testing.T) {
	// Create a test logger
	logger := slog.New(slog.NewTextHandler(nil, nil))

	// Create rate limit config
	rateLimitConfig := middleware.RateLimitConfig{
		RequestsPerSecond: 1,
		BurstSize:         2,
		UseIPBased:        true,
	}

	// Create rate limiter
	rateLimiter := middleware.NewRateLimitMiddleware(rateLimitConfig, logger)

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with rate limiter
	handler := rateLimiter.Handler(testHandler)

	// Test burst capacity
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i+1, rec.Code)
		}
	}

	// Test rate limit exceeded
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429 (Too Many Requests), got %d", rec.Code)
	}

	// Check rate limit headers
	if rec.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Missing X-RateLimit-Limit header")
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("Missing Retry-After header")
	}
}

func TestWithMiddleware(t *testing.T) {
	// Create test configuration
	cfg := config.ServerConfig{
		Address: ":8080",
		RateLimit: config.RateLimitConfig{
			Enabled:           true,
			RequestsPerSecond: 10,
			BurstSize:         20,
		},
	}

	// Create mock assistant (minimal implementation for testing)
	// In real tests, you would use a proper mock
	logger := slog.New(slog.NewTextHandler(nil, nil))
	metrics, err := observability.NewMetrics("test-service")
	if err != nil {
		t.Fatal(err)
	}

	// Create server
	server := &Server{
		config:  cfg,
		logger:  logger,
		metrics: metrics,
		mux:     http.NewServeMux(),
	}

	// Test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request ID is set
		if r.Context().Value(observability.RequestIDKey) == nil {
			t.Error("Request ID not found in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Apply middleware
	wrappedHandler := server.withMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Verify headers
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("Missing X-Request-ID header")
	}
}
