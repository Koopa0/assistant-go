package analytics

import (
	"net/http"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/platform/server/handlers"
	"github.com/koopa0/assistant-go/internal/platform/server/middleware"
	api "github.com/koopa0/assistant-go/internal/transport/http"
)

// HTTPHandler handles HTTP requests for analytics
type HTTPHandler struct {
	*handlers.Handler
	service *AnalyticsService
}

// NewHTTPHandler creates a new HTTP handler for analytics
func NewHTTPHandler(service *AnalyticsService) *HTTPHandler {
	// TODO: Accept logger as parameter
	return &HTTPHandler{
		Handler: handlers.NewHandler(nil), // Logger should be passed in
		service: service,
	}
}

// RegisterRoutes registers all analytics routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	// Activity analytics
	mux.HandleFunc("GET /api/v1/analytics/activity", h.HandleActivityAnalytics)
	mux.HandleFunc("GET /api/v1/analytics/activity/heatmap", h.HandleActivityHeatmap)
	mux.HandleFunc("GET /api/v1/analytics/productivity/trends", h.HandleProductivityTrends)

	// Skills analytics
	mux.HandleFunc("GET /api/v1/analytics/skills/distribution", h.HandleSkillsDistribution)
	mux.HandleFunc("GET /api/v1/analytics/skills/growth", h.HandleSkillsGrowth)
	mux.HandleFunc("GET /api/v1/analytics/skills/radar", h.HandleSkillsRadar)

	// Code quality analytics
	mux.HandleFunc("GET /api/v1/analytics/code/metrics", h.HandleCodeMetrics)
	mux.HandleFunc("GET /api/v1/analytics/code/evolution", h.HandleCodeEvolution)
	mux.HandleFunc("GET /api/v1/analytics/code/hotspots", h.HandleCodeHotspots)

	// Knowledge network analytics
	mux.HandleFunc("GET /api/v1/analytics/knowledge/network", h.HandleKnowledgeNetwork)
	mux.HandleFunc("GET /api/v1/analytics/knowledge/clusters", h.HandleKnowledgeClusters)

	// Predictions
	mux.HandleFunc("GET /api/v1/analytics/predictions/burnout", h.HandleBurnoutPrediction)
	mux.HandleFunc("GET /api/v1/analytics/predictions/completion", h.HandleCompletionPrediction)

	// Dashboard
	mux.HandleFunc("GET /api/v1/analytics/dashboard", h.HandleDashboardData)

	// Insights routes
	mux.HandleFunc("GET /api/v1/insights/development-patterns", h.HandleDevelopmentPatterns)
	mux.HandleFunc("GET /api/v1/insights/productivity", h.HandleProductivityInsights)
	mux.HandleFunc("GET /api/v1/insights/code-quality", h.HandleCodeQualityInsights)
	mux.HandleFunc("GET /api/v1/insights/technical-debt", h.HandleTechnicalDebt)
	mux.HandleFunc("GET /api/v1/insights/learning-effectiveness", h.HandleLearningEffectiveness)
	mux.HandleFunc("GET /api/v1/insights/skill-gaps", h.HandleSkillGaps)
	mux.HandleFunc("GET /api/v1/insights/recommendations", h.HandleRecommendations)
	mux.HandleFunc("GET /api/v1/insights/next-actions", h.HandleNextActions)
	mux.HandleFunc("GET /api/v1/insights/summary", h.HandleInsightsSummary)

	// Timeline routes
	mux.HandleFunc("GET /api/v1/timeline", h.HandleGetTimeline)
	mux.HandleFunc("GET /api/v1/timeline/events", h.HandleGetTimelineEvents)
	mux.HandleFunc("POST /api/v1/timeline/events", h.HandleCreateEvent)
	mux.HandleFunc("GET /api/v1/timeline/statistics", h.HandleGetTimelineStatistics)
	mux.HandleFunc("GET /api/v1/timeline/patterns", h.HandleGetPatterns)
}

// HandleActivityAnalytics handles activity analytics requests
func (h *HTTPHandler) HandleActivityAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	days := api.QueryParamInt(r, "days", 30)

	metrics, insights, err := h.service.GetActivityAnalytics(ctx, days)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法取得活動分析", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"metrics":  metrics,
		"insights": insights,
		"trends": map[string]interface{}{
			"activity_trend":     "stable",
			"productivity_trend": "improving",
			"consistency_score":  0.82,
			"streak_days":        12,
		},
	}

	middleware.WriteSuccess(w, response, "Activity analytics retrieved successfully")
}

// HandleActivityHeatmap handles activity heatmap requests
func (h *HTTPHandler) HandleActivityHeatmap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	heatmap, patterns, recommendations, err := h.service.GetActivityHeatmap(ctx)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法生成活動熱圖", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"heatmap":         heatmap,
		"patterns":        patterns,
		"recommendations": recommendations,
	}

	middleware.WriteSuccess(w, response, "Activity analytics retrieved successfully")
}

// HandleProductivityTrends handles productivity trends requests
func (h *HTTPHandler) HandleProductivityTrends(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	period := r.URL.Query().Get("period")
	trend, insights, err := h.service.GetProductivityTrends(ctx, period)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法取得生產力趨勢", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"trend":    trend,
		"insights": insights,
	}

	middleware.WriteSuccess(w, response, "Activity analytics retrieved successfully")
}

// HandleDashboardData handles dashboard data requests
func (h *HTTPHandler) HandleDashboardData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	dashboard, err := h.service.GetDashboardData(ctx)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法取得儀表板數據", http.StatusInternalServerError)
		return
	}

	middleware.WriteSuccess(w, dashboard, "Dashboard data retrieved successfully")
}

// Insights handlers

// HandleDevelopmentPatterns handles development patterns insights
func (h *HTTPHandler) HandleDevelopmentPatterns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	timeRange := r.URL.Query().Get("time_range")
	patterns, summary, err := h.service.GetDevelopmentPatterns(ctx, timeRange)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法分析開發模式", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"patterns": patterns,
		"summary":  summary,
	}

	middleware.WriteSuccess(w, response, "Activity analytics retrieved successfully")
}

// HandleCodeQualityInsights handles code quality insights
func (h *HTTPHandler) HandleCodeQualityInsights(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	insights, overallHealth, err := h.service.GetCodeQualityInsights(ctx)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法取得程式碼品質分析", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"insights":       insights,
		"overall_health": overallHealth,
	}

	middleware.WriteSuccess(w, response, "Activity analytics retrieved successfully")
}

// HandleRecommendations handles personalized recommendations
func (h *HTTPHandler) HandleRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	category := r.URL.Query().Get("category")
	recommendations, err := h.service.GetRecommendations(ctx, category)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法生成建議", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"recommendations": recommendations,
		"generated_at":    time.Now().Format(time.RFC3339),
	}

	middleware.WriteSuccess(w, response, "Activity analytics retrieved successfully")
}

// Timeline handlers

// HandleGetTimeline handles timeline requests
func (h *HTTPHandler) HandleGetTimeline(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	eventTypes := strings.Split(r.URL.Query().Get("types"), ",")
	tags := strings.Split(r.URL.Query().Get("tags"), ",")
	limit := api.QueryParamInt(r, "limit", 100)

	// Parse dates
	startDate, _ := time.Parse("2006-01-02", startDateStr)
	endDate, _ := time.Parse("2006-01-02", endDateStr)
	if startDate.IsZero() {
		startDate = time.Now().AddDate(0, 0, -7) // Last 7 days
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	// Clean up empty strings
	eventTypes = api.FilterEmptyStrings(eventTypes)
	tags = api.FilterEmptyStrings(tags)

	timeline, err := h.service.GetTimeline(ctx, startDate, endDate, eventTypes, tags, limit)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法取得時間軸", http.StatusInternalServerError)
		return
	}

	middleware.WriteSuccess(w, timeline, "Timeline retrieved successfully")
}

// HandleGetTimelineStatistics handles timeline statistics requests
func (h *HTTPHandler) HandleGetTimelineStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse date range
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	startDate, _ := time.Parse("2006-01-02", startDateStr)
	endDate, _ := time.Parse("2006-01-02", endDateStr)
	if startDate.IsZero() {
		startDate = time.Now().AddDate(0, 0, -30)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	stats, err := h.service.GetTimelineStatistics(ctx, startDate, endDate)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法取得時間軸統計", http.StatusInternalServerError)
		return
	}

	middleware.WriteSuccess(w, stats, "Timeline statistics retrieved successfully")
}

// HandleGetPatterns handles timeline patterns requests
func (h *HTTPHandler) HandleGetPatterns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	days := api.QueryParamInt(r, "days", 30)

	patterns, err := h.service.GetTimelinePatterns(ctx, days)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法分析時間軸模式", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"patterns":      patterns,
		"analysis_days": days,
	}

	middleware.WriteSuccess(w, response, "Activity analytics retrieved successfully")
}

// Placeholder handlers for unimplemented routes

func (h *HTTPHandler) HandleSkillsDistribution(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Skills distribution analysis coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleSkillsGrowth(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Skills growth analysis coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleSkillsRadar(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Skills radar chart coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleCodeMetrics(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Code metrics analysis coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleCodeEvolution(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Code evolution analysis coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleCodeHotspots(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Code hotspots analysis coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleKnowledgeNetwork(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Knowledge network visualization coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleKnowledgeClusters(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Knowledge clusters analysis coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleBurnoutPrediction(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Burnout prediction analysis coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleCompletionPrediction(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Completion prediction analysis coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleProductivityInsights(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Productivity insights coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleTechnicalDebt(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Technical debt analysis coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleLearningEffectiveness(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Learning effectiveness analysis coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleSkillGaps(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Skill gaps analysis coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleNextActions(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Next actions suggestions coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleInsightsSummary(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Insights summary coming soon",
	}, "Feature coming soon")
}

func (h *HTTPHandler) HandleGetTimelineEvents(w http.ResponseWriter, r *http.Request) {
	// Redirect to main timeline endpoint
	h.HandleGetTimeline(w, r)
}

func (h *HTTPHandler) HandleCreateEvent(w http.ResponseWriter, r *http.Request) {
	middleware.WriteSuccess(w, map[string]interface{}{
		"message": "Event creation coming soon",
	}, "Feature coming soon")
}
