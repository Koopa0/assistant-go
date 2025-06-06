package system

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPHandler handles HTTP requests for system monitoring
type HTTPHandler struct {
	service *SystemService
}

// NewHTTPHandler creates a new HTTP handler for system
func NewHTTPHandler(service *SystemService) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

// RegisterRoutes registers all system routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /system/status", h.HandleGetStatus)
	mux.HandleFunc("GET /system/activities", h.HandleGetActivities)
	mux.HandleFunc("GET /system/metrics", h.HandleGetMetrics)
	mux.HandleFunc("GET /system/health", h.HandleHealthCheck)
	mux.HandleFunc("GET /system/version", h.HandleGetVersion)
	mux.HandleFunc("GET /system/performance", h.HandleGetPerformance)
}

// HandleGetStatus returns comprehensive system status
func (h *HTTPHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status, err := h.service.GetStatus(ctx)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得系統狀態", http.StatusInternalServerError)
		return
	}

	h.writeSuccess(w, status, "系統狀態正常")
}

// HandleGetActivities returns system activities with pagination
func (h *HTTPHandler) HandleGetActivities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	activityType := r.URL.Query().Get("type")
	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")
	page := h.parseInt(r.URL.Query().Get("page"), 1)
	limit := h.parseInt(r.URL.Query().Get("limit"), 20)

	activities, total, err := h.service.GetActivities(ctx, activityType, startDate, endDate, page, limit)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得系統活動", http.StatusInternalServerError)
		return
	}

	h.writeSuccessWithPagination(w, activities, page, limit, total)
}

// HandleGetMetrics returns detailed system metrics
func (h *HTTPHandler) HandleGetMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metrics, err := h.service.GetMetrics(ctx)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得系統指標", http.StatusInternalServerError)
		return
	}

	h.writeSuccess(w, metrics, "系統指標")
}

// HandleHealthCheck provides a simple health check endpoint
func (h *HTTPHandler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check assistant health
	if err := h.service.CheckHealth(ctx); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "unhealthy",
			"error":     err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// HandleGetVersion returns version information
func (h *HTTPHandler) HandleGetVersion(w http.ResponseWriter, r *http.Request) {
	version := h.service.GetVersion()
	h.writeSuccess(w, version, "版本資訊")
}

// HandleGetPerformance returns performance metrics
func (h *HTTPHandler) HandleGetPerformance(w http.ResponseWriter, r *http.Request) {
	timeRange := r.URL.Query().Get("range")
	performance := h.service.GetPerformance(timeRange)
	h.writeSuccess(w, performance, "效能指標")
}

// Helper methods

func (h *HTTPHandler) parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	var value int
	if _, err := fmt.Sscanf(s, "%d", &value); err != nil {
		return defaultValue
	}
	if value <= 0 {
		return defaultValue
	}
	return value
}

// Response helpers

func (h *HTTPHandler) writeSuccess(w http.ResponseWriter, data interface{}, message string) {
	response := map[string]interface{}{
		"success":   true,
		"data":      data,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) writeSuccessWithPagination(w http.ResponseWriter, data interface{}, page, limit, total int) {
	response := map[string]interface{}{
		"success": true,
		"data":    data,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + limit - 1) / limit,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, code, message string, status int) {
	response := map[string]interface{}{
		"success":   false,
		"error":     code,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}
