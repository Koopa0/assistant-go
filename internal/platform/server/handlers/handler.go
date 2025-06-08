package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/server/middleware"
)

// Handler provides common functionality for all HTTP handlers
type Handler struct {
	logger *slog.Logger
}

// NewHandler creates a new handler with common functionality
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// GetUserID extracts the user ID from the request context
func (h *Handler) GetUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok || userID == "" {
		return "", errors.New("user not authenticated")
	}
	return userID, nil
}

// GetUserIDOrDefault extracts the user ID from the request context or returns a default
func (h *Handler) GetUserIDOrDefault(ctx context.Context, defaultID string) string {
	userID, err := h.GetUserID(ctx)
	if err != nil {
		return defaultID
	}
	return userID
}

// WriteSuccess writes a successful JSON response
func (h *Handler) WriteSuccess(w http.ResponseWriter, data interface{}, message string) {
	middleware.WriteSuccess(w, data, message)
}

// WriteSuccessWithPagination writes a successful JSON response with pagination
func (h *Handler) WriteSuccessWithPagination(w http.ResponseWriter, data interface{}, page, limit, total int) {
	middleware.WriteSuccessWithPagination(w, data, page, limit, total)
}

// WriteError writes an error response
func (h *Handler) WriteError(w http.ResponseWriter, code string, message string, statusCode int, details ...interface{}) {
	middleware.WriteError(w, code, message, statusCode, details...)
}

// WriteInternalError writes an internal server error response
func (h *Handler) WriteInternalError(w http.ResponseWriter, err error) {
	h.logger.Error("Internal server error", slog.Any("error", err))
	h.WriteError(w, middleware.CodeServerError, "Internal server error", http.StatusInternalServerError, err.Error())
}

// WriteNotFound writes a not found error response
func (h *Handler) WriteNotFound(w http.ResponseWriter, resource string) {
	message := fmt.Sprintf("%s not found", resource)
	h.WriteError(w, middleware.CodeNotFound, message, http.StatusNotFound)
}

// WriteBadRequest writes a bad request error response
func (h *Handler) WriteBadRequest(w http.ResponseWriter, message string, details ...interface{}) {
	h.WriteError(w, middleware.CodeInvalidRequest, message, http.StatusBadRequest, details...)
}

// WriteUnauthorized writes an unauthorized error response
func (h *Handler) WriteUnauthorized(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Unauthorized"
	}
	h.WriteError(w, middleware.CodeUnauthorized, message, http.StatusUnauthorized)
}

// DecodeJSON decodes JSON request body
func (h *Handler) DecodeJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return errors.New("request body is empty")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return nil
}

// ParsePagination parses pagination parameters from query string
func (h *Handler) ParsePagination(r *http.Request) (page, limit int, err error) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	// Default values
	page = 1
	limit = 20

	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return 0, 0, errors.New("invalid page parameter")
		}
	}

	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			return 0, 0, errors.New("invalid limit parameter (must be 1-100)")
		}
	}

	return page, limit, nil
}

// ValidateRequired validates that required fields are not empty
func (h *Handler) ValidateRequired(fields map[string]string) error {
	for name, value := range fields {
		if value == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	return nil
}

// GetRequestID extracts request ID from context for logging
func (h *Handler) GetRequestID(ctx context.Context) string {
	requestID, ok := ctx.Value(observability.RequestIDKey).(string)
	if !ok {
		return "unknown"
	}
	return requestID
}

// LogRequest logs incoming request with context
func (h *Handler) LogRequest(r *http.Request, operation string) {
	h.logger.Info("Handling request",
		slog.String("operation", operation),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("request_id", h.GetRequestID(r.Context())),
		slog.String("user_id", h.GetUserIDOrDefault(r.Context(), "anonymous")),
	)
}

// LogError logs error with context
func (h *Handler) LogError(r *http.Request, operation string, err error) {
	h.logger.Error("Request failed",
		slog.String("operation", operation),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("request_id", h.GetRequestID(r.Context())),
		slog.String("user_id", h.GetUserIDOrDefault(r.Context(), "anonymous")),
		slog.Any("error", err),
	)
}
