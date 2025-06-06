package learning

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// HTTPHandler handles HTTP requests for learning system
type HTTPHandler struct {
	service *LearningService
}

// NewHTTPHandler creates a new HTTP handler for learning
func NewHTTPHandler(service *LearningService) *HTTPHandler {
	return &HTTPHandler{
		service: service,
	}
}

// RegisterRoutes registers all learning system routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	// Pattern recognition
	mux.HandleFunc("GET /api/v1/learning/patterns", h.HandleListPatterns)
	mux.HandleFunc("GET /api/v1/learning/patterns/{patternId}", h.HandleGetPattern)
	mux.HandleFunc("POST /api/v1/learning/patterns/detect", h.HandleDetectPatterns)

	// Preference learning
	mux.HandleFunc("GET /api/v1/learning/preferences", h.HandleListPreferences)
	mux.HandleFunc("PUT /api/v1/learning/preferences", h.HandleUpdatePreferences)
	mux.HandleFunc("POST /api/v1/learning/preferences/predict", h.HandlePredictPreference)

	// Learning events
	mux.HandleFunc("POST /api/v1/learning/events", h.HandleCreateLearningEvent)
	mux.HandleFunc("GET /api/v1/learning/events", h.HandleListLearningEvents)

	// Feedback and reinforcement
	mux.HandleFunc("POST /api/v1/learning/feedback", h.HandleProvideFeedback)
	mux.HandleFunc("GET /api/v1/learning/reinforcement", h.HandleGetReinforcement)

	// Learning report
	mux.HandleFunc("GET /api/v1/learning/report", h.HandleLearningReport)
}

// HandleListPatterns lists identified patterns
func (h *HTTPHandler) HandleListPatterns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	patternType := r.URL.Query().Get("type")
	minConfidence, _ := strconv.ParseFloat(r.URL.Query().Get("min_confidence"), 64)
	if minConfidence == 0 {
		minConfidence = 0.7
	}

	patterns, err := h.service.ListPatterns(ctx, patternType, minConfidence)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得模式列表", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"patterns": patterns,
		"count":    len(patterns),
		"filters": map[string]interface{}{
			"type":           patternType,
			"min_confidence": minConfidence,
		},
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleGetPattern retrieves a specific pattern
func (h *HTTPHandler) HandleGetPattern(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	patternID := r.PathValue("patternId")

	if patternID == "" {
		h.writeError(w, "INVALID_REQUEST", "模式 ID 為必填", http.StatusBadRequest)
		return
	}

	pattern, err := h.service.GetPattern(ctx, patternID)
	if err != nil {
		h.writeError(w, "NOT_FOUND", "找不到此模式", http.StatusNotFound)
		return
	}

	h.writeJSON(w, http.StatusOK, pattern)
}

// HandleDetectPatterns detects patterns in provided data
func (h *HTTPHandler) HandleDetectPatterns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Data  map[string]interface{} `json:"data"`
		Scope string                 `json:"scope"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	if len(req.Data) == 0 {
		h.writeError(w, "INVALID_REQUEST", "數據為必填", http.StatusBadRequest)
		return
	}

	patterns, err := h.service.DetectPatterns(ctx, req.Data, req.Scope)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "模式檢測失敗", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"detected_patterns": patterns,
		"count":             len(patterns),
		"analysis_scope":    req.Scope,
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleListPreferences lists learned preferences
func (h *HTTPHandler) HandleListPreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	category := r.URL.Query().Get("category")

	preferences, err := h.service.ListPreferences(ctx, category)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得偏好設定", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"preferences": preferences,
		"count":       len(preferences),
		"category":    category,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleUpdatePreferences updates preferences
func (h *HTTPHandler) HandleUpdatePreferences(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdatePreferences(ctx, updates); err != nil {
		h.writeError(w, "SERVER_ERROR", "無法更新偏好設定", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message": "偏好設定更新成功",
		"updated": len(updates),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandlePredictPreference predicts a preference value
func (h *HTTPHandler) HandlePredictPreference(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Category string `json:"category"`
		Context  string `json:"context"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	if req.Category == "" || req.Context == "" {
		h.writeError(w, "INVALID_REQUEST", "類別和上下文為必填", http.StatusBadRequest)
		return
	}

	prediction, err := h.service.PredictPreference(ctx, req.Category, req.Context)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "預測失敗", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, prediction)
}

// HandleCreateLearningEvent creates a learning event
func (h *HTTPHandler) HandleCreateLearningEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var event LearningEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if event.Type == "" || event.Category == "" {
		h.writeError(w, "INVALID_REQUEST", "類型和類別為必填", http.StatusBadRequest)
		return
	}

	createdEvent, err := h.service.CreateLearningEvent(ctx, event)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法創建學習事件", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusCreated, createdEvent)
}

// HandleListLearningEvents lists learning events
func (h *HTTPHandler) HandleListLearningEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	eventType := r.URL.Query().Get("type")
	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if limit <= 0 {
		limit = 50
	}

	// Parse times
	var startTime, endTime time.Time
	if startTimeStr != "" {
		startTime, _ = time.Parse(time.RFC3339, startTimeStr)
	} else {
		startTime = time.Now().Add(-7 * 24 * time.Hour)
	}
	if endTimeStr != "" {
		endTime, _ = time.Parse(time.RFC3339, endTimeStr)
	} else {
		endTime = time.Now()
	}

	events, err := h.service.ListLearningEvents(ctx, eventType, startTime, endTime, limit)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得學習事件", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"events":     events,
		"count":      len(events),
		"time_range": map[string]string{"start": startTime.Format(time.RFC3339), "end": endTime.Format(time.RFC3339)},
		"event_type": eventType,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleProvideFeedback handles user feedback
func (h *HTTPHandler) HandleProvideFeedback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var feedback map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		h.writeError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	if err := h.service.ProvideFeedback(ctx, feedback); err != nil {
		h.writeError(w, "SERVER_ERROR", "無法處理反饋", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"message":   "反饋已接收並處理",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HandleGetReinforcement gets reinforcement learning data
func (h *HTTPHandler) HandleGetReinforcement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	action := r.URL.Query().Get("action")
	context := r.URL.Query().Get("context")

	if action == "" || context == "" {
		h.writeError(w, "INVALID_REQUEST", "動作和上下文為必填", http.StatusBadRequest)
		return
	}

	reinforcement, err := h.service.GetReinforcement(ctx, action, context)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法取得強化學習數據", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, reinforcement)
}

// HandleLearningReport generates a learning report
func (h *HTTPHandler) HandleLearningReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "week"
	}

	report, err := h.service.GetLearningReport(ctx, period)
	if err != nil {
		h.writeError(w, "SERVER_ERROR", "無法生成學習報告", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, report)
}

// Helper methods

func (h *HTTPHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, code, message string, status int) {
	response := map[string]interface{}{
		"success":   false,
		"error":     code,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	h.writeJSON(w, status, response)
}
