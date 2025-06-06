// Package middleware provides HTTP middleware for the Assistant API server.
package middleware

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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
		rw.ResponseWriter.Header().Set("X-Content-Type-Options", "nosniff")
		rw.ResponseWriter.Header().Set("X-Frame-Options", "DENY")
		rw.ResponseWriter.Header().Set("X-XSS-Protection", "1; mode=block")

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

// PostgreSQL type conversion utilities

// PgtypeUUIDToUUID 將 pgtype.UUID 轉換為 uuid.UUID
func PgtypeUUIDToUUID(pgu pgtype.UUID) uuid.UUID {
	if !pgu.Valid {
		return uuid.Nil
	}
	return pgu.Bytes
}

// UUIDToPgtypeUUID 將 uuid.UUID 轉換為 pgtype.UUID
func UUIDToPgtypeUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: u,
		Valid: u != uuid.Nil,
	}
}

// PgtypeTextToStringPtr 將 pgtype.Text 轉換為 *string
func PgtypeTextToStringPtr(pt pgtype.Text) *string {
	if !pt.Valid {
		return nil
	}
	return &pt.String
}

// StringToPgtypeText 將 string 轉換為 pgtype.Text
func StringToPgtypeText(s string) pgtype.Text {
	return pgtype.Text{
		String: s,
		Valid:  true,
	}
}

// StringPtrToPgtypeText 將 *string 轉換為 pgtype.Text
func StringPtrToPgtypeText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{
		String: *s,
		Valid:  true,
	}
}

// PgtypeTimestamptzToTimePtr 將 pgtype.Timestamptz 轉換為 *time.Time
func PgtypeTimestamptzToTimePtr(pt pgtype.Timestamptz) *time.Time {
	if !pt.Valid {
		return nil
	}
	return &pt.Time
}

// TimeToPgtypeTimestamptz 將 time.Time 轉換為 pgtype.Timestamptz
func TimeToPgtypeTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  t,
		Valid: !t.IsZero(),
	}
}

// TimePtrToPgtypeTimestamptz 將 *time.Time 轉換為 pgtype.Timestamptz
func TimePtrToPgtypeTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{
		Time:  *t,
		Valid: true,
	}
}

// PgtypeInt4ToInt32 將 pgtype.Int4 轉換為 int32
func PgtypeInt4ToInt32(pi pgtype.Int4) int32 {
	if !pi.Valid {
		return 0
	}
	return pi.Int32
}

// Int32ToPgtypeInt4 將 int32 轉換為 pgtype.Int4
func Int32ToPgtypeInt4(i int32) pgtype.Int4 {
	return pgtype.Int4{
		Int32: i,
		Valid: true,
	}
}

// PgtypeInt4ToInt32Ptr 將 pgtype.Int4 轉換為 *int32
func PgtypeInt4ToInt32Ptr(pi pgtype.Int4) *int32 {
	if !pi.Valid {
		return nil
	}
	return &pi.Int32
}

// PgtypeNumericToFloat64 將 pgtype.Numeric 轉換為 float64
func PgtypeNumericToFloat64(pn pgtype.Numeric) float64 {
	if !pn.Valid {
		return 0
	}

	// 嘗試轉換為 float64
	f64, err := pn.Float64Value()
	if err != nil {
		return 0
	}

	return f64.Float64
}

// Float64ToPgtypeNumeric 將 float64 轉換為 pgtype.Numeric
func Float64ToPgtypeNumeric(f float64) pgtype.Numeric {
	var pn pgtype.Numeric
	if err := pn.Scan(f); err != nil {
		return pgtype.Numeric{Valid: false}
	}
	return pn
}

// PgtypeTimestamptzToTime 將 pgtype.Timestamptz 轉換為 time.Time
func PgtypeTimestamptzToTime(pt pgtype.Timestamptz) time.Time {
	if !pt.Valid {
		return time.Time{}
	}
	return pt.Time
}
