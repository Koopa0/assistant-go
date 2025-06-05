// Package middleware provides HTTP middleware for the Assistant API server.
package middleware

import (
	"encoding/json"
	"net/http"
	"time"
)

// StandardResponse represents the standard API response format
type StandardResponse struct {
	Success    bool            `json:"success"`
	Data       interface{}     `json:"data,omitempty"`
	Error      string          `json:"error,omitempty"`
	Message    string          `json:"message,omitempty"`
	Details    interface{}     `json:"details,omitempty"`
	Pagination *PaginationInfo `json:"pagination,omitempty"`
	Timestamp  string          `json:"timestamp"`
}

// PaginationInfo represents pagination information
type PaginationInfo struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// ResponseWriter wraps http.ResponseWriter to capture status code
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// NewResponseWriter creates a new ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.written = true
	rw.ResponseWriter.WriteHeader(code)
}

// Write implements the http.ResponseWriter interface
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// WriteSuccess writes a successful response
func WriteSuccess(w http.ResponseWriter, data interface{}, message string) {
	response := StandardResponse{
		Success:   true,
		Data:      data,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// WriteSuccessWithPagination writes a successful response with pagination
func WriteSuccessWithPagination(w http.ResponseWriter, data interface{}, page, limit, total int) {
	totalPages := total / limit
	if total%limit > 0 {
		totalPages++
	}

	response := StandardResponse{
		Success: true,
		Data:    data,
		Pagination: &PaginationInfo{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, code string, message string, statusCode int, details ...interface{}) {
	response := StandardResponse{
		Success:   false,
		Error:     code,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if len(details) > 0 {
		response.Details = details[0]
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// StandardResponseMiddleware ensures all responses follow the standard format
func StandardResponseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap the response writer
		rw := NewResponseWriter(w)

		// Set common headers
		rw.Header().Set("X-Content-Type-Options", "nosniff")
		rw.Header().Set("X-Frame-Options", "DENY")
		rw.Header().Set("X-XSS-Protection", "1; mode=block")

		// Call the next handler
		next.ServeHTTP(rw, r)
	})
}

// ErrorCode constants
const (
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeNotFound           = "NOT_FOUND"
	CodeInvalidRequest     = "INVALID_REQUEST"
	CodeRateLimited        = "RATE_LIMITED"
	CodeServerError        = "SERVER_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// GetErrorStatusCode returns the appropriate HTTP status code for an error code
func GetErrorStatusCode(code string) int {
	switch code {
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeInvalidRequest:
		return http.StatusBadRequest
	case CodeRateLimited:
		return http.StatusTooManyRequests
	case CodeServerError:
		return http.StatusInternalServerError
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
