// Package api provides common API utilities and shared handler functionality
// for HTTP services in the Assistant application.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// ResponseWriter provides standardized JSON response handling
type ResponseWriter struct{}

// NewResponseWriter creates a new ResponseWriter instance
func NewResponseWriter() *ResponseWriter {
	return &ResponseWriter{}
}

type Response struct {
	Success   bool   `json:"success"`
	Data      any    `json:"data"`
	Error     string `json:"error"`
	Message   string `json:"message"`
	TimeStamp string `json:"timestamp"`
}

// WriteJSON writes a JSON response with the given status code
func (rw *ResponseWriter) WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(Response{
		Success:   true,
		Data:      data,
		TimeStamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		// Log error but don't write another response
		// The caller should handle logging
		_ = err
	}
}

// WriteError writes a standardized error response
func (rw *ResponseWriter) WriteError(w http.ResponseWriter, code, message string, status int) {
	rw.WriteJSON(w, status, Response{
		Success:   false,
		Error:     code,
		Message:   message,
		TimeStamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// WriteSuccess writes a standardized success response
func (rw *ResponseWriter) WriteSuccess(w http.ResponseWriter, data any) {
	response := map[string]any{
		"success":   true,
		"data":      data,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	rw.WriteJSON(w, http.StatusOK, response)
}

// ParseJSONRequest parses a JSON request body into the given struct
func ParseJSONRequest(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// QueryParamInt parses an integer query parameter with a default value
func QueryParamInt(r *http.Request, name string, defaultValue int) int {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil || intValue <= 0 {
		return defaultValue
	}

	return intValue
}

// QueryParamString returns a query parameter string value with a default
func QueryParamString(r *http.Request, name, defaultValue string) string {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}
	return value
}

// QueryParamBool parses a boolean query parameter with a default value
func QueryParamBool(r *http.Request, name string, defaultValue bool) bool {
	value := r.URL.Query().Get(name)
	if value == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return boolValue
}

// FilterEmptyStrings removes empty strings from a slice
func FilterEmptyStrings(slice []string) []string {
	filtered := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != "" {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
