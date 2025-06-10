package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/server/middleware"
	"github.com/koopa0/assistant-go/internal/user"
)

// requestIDMiddleware adds a unique request ID to each request
func (s *Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)

		// Add request ID to context
		ctx := context.WithValue(r.Context(), observability.RequestIDKey, requestID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Get request ID from context
		requestID := ""
		if id := r.Context().Value(observability.RequestIDKey); id != nil {
			requestID = id.(string)
		}

		// Create request logger
		logger := observability.RequestLogger(s.logger, r.Method, r.URL.Path, requestID)

		logger.Info("Request started",
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.String("referer", r.Referer()))

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log completion
		duration := time.Since(start)
		logger.Info("Request completed",
			slog.Int("status_code", wrapped.statusCode),
			slog.Duration("duration", duration),
			slog.Int64("response_size", wrapped.bytesWritten))
	})
}

// corsMiddleware 處理 CORS headers - 生產環境必須配置允許的來源
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 從配置獲取允許的來源，預設只允許本地開發
		allowedOrigins := s.config.Security.AllowedOrigins
		if len(allowedOrigins) == 0 {
			// 開發環境預設配置
			if s.config.Address == ":8080" || s.config.Address == "localhost:8080" {
				allowedOrigins = []string{"http://localhost:3000", "http://localhost:8080"}
			} else {
				// 生產環境必須明確配置
				s.logger.Warn("CORS: 未配置允許的來源，拒絕所有跨域請求")
				allowedOrigins = []string{} // 不允許任何跨域
			}
		}

		// 檢查請求來源是否在允許清單中
		origin := r.Header.Get("Origin")
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		// 只有在允許的情況下設定 CORS headers
		if allowed {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// 處理預檢請求
		if r.Method == http.MethodOptions {
			if allowed {
				w.WriteHeader(http.StatusNoContent)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware recovers from panics and returns a 500 error
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Get request ID from context
				requestID := ""
				if id := r.Context().Value(observability.RequestIDKey); id != nil {
					requestID = id.(string)
				}

				// Log the panic
				s.logger.Error("Panic recovered",
					slog.String("request_id", requestID),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Any("panic", err),
					slog.String("stack", string(debug.Stack())))

				// Return 500 error
				if !isResponseWritten(w) {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// rateLimitMiddleware applies rate limiting to requests
func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	// Initialize rate limiter if not already done
	if s.rateLimiter == nil {
		s.initRateLimiter()
	}

	// Skip rate limiting if not configured
	if s.rateLimiter == nil {
		return next
	}

	return s.rateLimiter.Handler(next)
}

// responseWriter wraps http.ResponseWriter to capture status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	statusCode    int
	bytesWritten  int64
	headerWritten bool
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.headerWritten {
		rw.statusCode = statusCode
		rw.headerWritten = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

// Write captures the number of bytes written
func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.headerWritten {
		rw.WriteHeader(http.StatusOK)
	}

	n, err := rw.ResponseWriter.Write(data)
	rw.bytesWritten += int64(n)
	return n, err
}

// isResponseWritten checks if the response has been written
func isResponseWritten(w http.ResponseWriter) bool {
	if rw, ok := w.(*responseWriter); ok {
		return rw.headerWritten
	}
	return false
}

// parseJSONRequest parses a JSON request body into the provided struct
func (s *Server) parseJSONRequest(r *http.Request, v interface{}) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf("content-type must be application/json")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return nil
}

// writeJSONResponse writes a JSON response
func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("Failed to encode JSON response", slog.Any("error", err))
	}
}

// writeErrorResponse writes an error response
func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	s.writeJSONResponse(w, statusCode, map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// initRateLimiter initializes the rate limiter based on configuration
func (s *Server) initRateLimiter() {
	// Check if rate limiting is enabled in config
	if !s.config.RateLimit.Enabled {
		s.logger.Info("Rate limiting is disabled")
		return
	}

	// Default configuration
	rateLimitConfig := middleware.RateLimitConfig{
		RequestsPerSecond: 10, // Default: 10 requests per second
		BurstSize:         20, // Default: burst of 20
		UseIPBased:        true,
		EndpointLimits:    make(map[string]middleware.EndpointLimit),
	}

	// Override with config values if available
	if s.config.RateLimit.RequestsPerSecond > 0 {
		rateLimitConfig.RequestsPerSecond = s.config.RateLimit.RequestsPerSecond
	}
	if s.config.RateLimit.BurstSize > 0 {
		rateLimitConfig.BurstSize = s.config.RateLimit.BurstSize
	}

	// Configure endpoint-specific limits
	// High-cost endpoints
	rateLimitConfig.EndpointLimits["/api/v1/chat"] = middleware.EndpointLimit{
		RequestsPerMinute: 30,
		BurstSize:         5,
	}
	rateLimitConfig.EndpointLimits["/api/v1/langchain"] = middleware.EndpointLimit{
		RequestsPerMinute: 20,
		BurstSize:         3,
	}
	rateLimitConfig.EndpointLimits["/api/v1/tools"] = middleware.EndpointLimit{
		RequestsPerMinute: 60,
		BurstSize:         10,
	}

	// Initialize the rate limiter
	s.rateLimiter = middleware.NewRateLimitMiddleware(rateLimitConfig, s.logger)

	s.logger.Info("Rate limiter initialized",
		slog.Int("requests_per_second", rateLimitConfig.RequestsPerSecond),
		slog.Int("burst_size", rateLimitConfig.BurstSize),
		slog.Int("endpoint_limits", len(rateLimitConfig.EndpointLimits)))
}

// authMiddleware validates JWT tokens and adds user context
func (s *Server) authMiddleware(authService user.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for public endpoints
			publicPaths := []string{
				"/api/health",
				"/api/status",
				"/api/v1/auth/login",
				"/api/v1/auth/register",
				"/api/v1/auth/refresh",
				"/", // Root API info endpoint
			}

			for _, path := range publicPaths {
				if r.URL.Path == path {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				s.writeErrorResponse(w, http.StatusUnauthorized, "Missing authorization header")
				return
			}

			// Check Bearer prefix
			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				s.writeErrorResponse(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			token := strings.TrimPrefix(authHeader, bearerPrefix)

			// Validate token
			userID, err := authService.ValidateToken(token)
			if err != nil {
				s.logger.Warn("Invalid token", slog.Any("error", err))
				s.writeErrorResponse(w, http.StatusUnauthorized, "Invalid token")
				return
			}

			// Add user ID to request context
			ctx := context.WithValue(r.Context(), "user_id", userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
