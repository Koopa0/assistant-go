package system

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/server/handlers"
)

// Handler handles HTTP requests for system monitoring
type Handler struct {
	*handlers.Handler
	service *Service
}

// NewHandler creates a new HTTP handler for system
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	loggerWithName := observability.ServerLogger(logger, "system_http")
	return &Handler{
		Handler: handlers.NewHandler(loggerWithName),
		service: service,
	}
}

// RegisterRoutes registers all system routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /system/status", h.HandleGetStatus)
	mux.HandleFunc("GET /system/activities", h.HandleGetActivities)
	mux.HandleFunc("GET /system/metrics", h.HandleGetMetrics)
	mux.HandleFunc("GET /system/health", h.HandleHealthCheck)
	mux.HandleFunc("GET /system/version", h.HandleGetVersion)
	mux.HandleFunc("GET /system/performance", h.HandleGetPerformance)
}

// HandleGetStatus returns comprehensive system status
func (h *Handler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	status, err := h.service.GetStatus(ctx)
	if err != nil {
		h.WriteInternalError(w, err)
		return
	}

	h.writeSuccess(w, status, "System status is normal")
}

// HandleGetActivities returns system activities with pagination
func (h *Handler) HandleGetActivities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	activityType := r.URL.Query().Get("type")
	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")
	page := h.parseInt(r.URL.Query().Get("page"), 1)
	limit := h.parseInt(r.URL.Query().Get("limit"), 20)

	activities, total, err := h.service.GetActivities(ctx, activityType, startDate, endDate, page, limit)
	if err != nil {
		h.WriteInternalError(w, err)
		return
	}

	h.writeSuccessWithPagination(w, activities, page, limit, total)
}

// HandleGetMetrics returns detailed system metrics
func (h *Handler) HandleGetMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metrics, err := h.service.GetMetrics(ctx)
	if err != nil {
		h.WriteInternalError(w, err)
		return
	}

	h.writeSuccess(w, metrics, "System metrics")
}

// HandleHealthCheck provides a simple health check endpoint
func (h *Handler) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
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
func (h *Handler) HandleGetVersion(w http.ResponseWriter, r *http.Request) {
	version := h.service.GetVersion()
	h.writeSuccess(w, version, "Version information")
}

// HandleGetPerformance returns performance metrics
func (h *Handler) HandleGetPerformance(w http.ResponseWriter, r *http.Request) {
	timeRange := r.URL.Query().Get("range")
	performance := h.service.GetPerformance(timeRange)
	h.writeSuccess(w, performance, "Performance metrics")
}

// Helper methods

func (h *Handler) parseInt(s string, defaultValue int) int {
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

func (h *Handler) writeSuccess(w http.ResponseWriter, data interface{}, message string) {
	response := map[string]interface{}{
		"success":   true,
		"data":      data,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) writeSuccessWithPagination(w http.ResponseWriter, data interface{}, page, limit, total int) {
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
