package collaboration

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/platform/server/handlers"
	"github.com/koopa0/assistant-go/internal/platform/server/middleware"
)

// HTTPHandler handles HTTP requests for collaboration
type HTTPHandler struct {
	*handlers.Handler
	service *CollaborationService
}

// NewHTTPHandler creates a new HTTP handler for collaboration
func NewHTTPHandler(service *CollaborationService) *HTTPHandler {
	// TODO: Accept logger as parameter
	return &HTTPHandler{
		Handler: handlers.NewHandler(nil), // Logger should be passed in
		service: service,
	}
}

// RegisterRoutes registers all collaboration routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	// Agent collaboration
	mux.HandleFunc("GET /api/v1/collaboration/agents", h.HandleListAgents)
	mux.HandleFunc("POST /api/v1/collaboration/agents/session", h.HandleCreateAgentSession)
	mux.HandleFunc("GET /api/v1/collaboration/agents/session/{sessionId}", h.HandleGetAgentSession)
	mux.HandleFunc("POST /api/v1/collaboration/agents/task", h.HandleAssignAgentTask)

	// Knowledge sharing
	mux.HandleFunc("POST /api/v1/collaboration/knowledge/share", h.HandleShareKnowledge)
	mux.HandleFunc("GET /api/v1/collaboration/knowledge/shared", h.HandleGetSharedKnowledge)

	// Team collaboration
	mux.HandleFunc("GET /api/v1/collaboration/sessions", h.HandleListCollaborationSessions)
	mux.HandleFunc("GET /api/v1/collaboration/sessions/{sessionId}/timeline", h.HandleGetSessionTimeline)
	mux.HandleFunc("GET /api/v1/collaboration/effectiveness", h.HandleCollaborationEffectiveness)

	// Collaboration recommendations
	mux.HandleFunc("GET /api/v1/collaboration/recommendations", h.HandleCollaborationRecommendations)
}

// HandleListAgents lists available agents
func (h *HTTPHandler) HandleListAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	agentType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")

	agents, err := h.service.ListAgents(ctx, agentType, status)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法取得代理列表", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"agents": agents,
		"count":  len(agents),
		"filters": map[string]string{
			"type":   agentType,
			"status": status,
		},
	}

	middleware.WriteSuccess(w, response, "Agents listed successfully")
}

// HandleCreateAgentSession creates a new collaboration session
func (h *HTTPHandler) HandleCreateAgentSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		SessionType    string   `json:"session_type"`
		TaskDesc       string   `json:"task_description"`
		RequiredAgents []string `json:"required_agents,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	if req.SessionType == "" || req.TaskDesc == "" {
		h.WriteError(w, "INVALID_REQUEST", "會話類型和任務描述為必填", http.StatusBadRequest)
		return
	}

	session, err := h.service.CreateAgentSession(ctx, req.SessionType, req.TaskDesc, req.RequiredAgents)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法創建協作會話", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(session)
}

// HandleGetAgentSession retrieves a collaboration session
func (h *HTTPHandler) HandleGetAgentSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionID := r.PathValue("sessionId")

	if sessionID == "" {
		h.WriteError(w, "INVALID_REQUEST", "會話 ID 為必填", http.StatusBadRequest)
		return
	}

	session, err := h.service.GetAgentSession(ctx, sessionID)
	if err != nil {
		h.WriteError(w, "NOT_FOUND", "找不到此會話", http.StatusNotFound)
		return
	}

	middleware.WriteSuccess(w, session, "Session retrieved successfully")
}

// HandleAssignAgentTask assigns a task to agents
func (h *HTTPHandler) HandleAssignAgentTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var task map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		h.WriteError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	if task["description"] == nil || task["type"] == nil {
		h.WriteError(w, "INVALID_REQUEST", "任務描述和類型為必填", http.StatusBadRequest)
		return
	}

	assignment, err := h.service.AssignAgentTask(ctx, task)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法分配任務", http.StatusInternalServerError)
		return
	}

	middleware.WriteSuccess(w, assignment, "Task assigned successfully")
}

// HandleShareKnowledge shares knowledge
func (h *HTTPHandler) HandleShareKnowledge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var knowledge SharedKnowledge
	if err := json.NewDecoder(r.Body).Decode(&knowledge); err != nil {
		h.WriteError(w, "INVALID_REQUEST", "請求格式無效", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if knowledge.Title == "" || knowledge.Content == "" || knowledge.Type == "" {
		h.WriteError(w, "INVALID_REQUEST", "標題、內容和類型為必填", http.StatusBadRequest)
		return
	}

	shared, err := h.service.ShareKnowledge(ctx, knowledge)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法分享知識", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(shared)
}

// HandleGetSharedKnowledge retrieves shared knowledge
func (h *HTTPHandler) HandleGetSharedKnowledge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	knowledgeType := r.URL.Query().Get("type")
	visibility := r.URL.Query().Get("visibility")
	tags := strings.Split(r.URL.Query().Get("tags"), ",")

	// Clean up empty strings
	cleanTags := []string{}
	for _, tag := range tags {
		if tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}

	knowledge, err := h.service.GetSharedKnowledge(ctx, knowledgeType, visibility, cleanTags)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法取得分享知識", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"knowledge": knowledge,
		"count":     len(knowledge),
		"filters": map[string]interface{}{
			"type":       knowledgeType,
			"visibility": visibility,
			"tags":       cleanTags,
		},
	}

	middleware.WriteSuccess(w, response, "Agents listed successfully")
}

// HandleListCollaborationSessions lists collaboration sessions
func (h *HTTPHandler) HandleListCollaborationSessions(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement session listing logic
	sessions := []map[string]interface{}{
		{
			"id":          "session-1",
			"type":        "development",
			"status":      "active",
			"agent_count": 3,
			"started_at":  time.Now().Add(-2 * time.Hour),
			"progress":    0.65,
		},
		{
			"id":           "session-2",
			"type":         "debugging",
			"status":       "completed",
			"agent_count":  2,
			"started_at":   time.Now().Add(-4 * time.Hour),
			"completed_at": time.Now().Add(-3 * time.Hour),
			"progress":     1.0,
		},
	}

	response := map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	}

	middleware.WriteSuccess(w, response, "Agents listed successfully")
}

// HandleGetSessionTimeline gets session timeline
func (h *HTTPHandler) HandleGetSessionTimeline(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionId")

	if sessionID == "" {
		h.WriteError(w, "INVALID_REQUEST", "會話 ID 為必填", http.StatusBadRequest)
		return
	}

	// TODO: Implement timeline retrieval logic
	timeline := map[string]interface{}{
		"session_id": sessionID,
		"events": []map[string]interface{}{
			{
				"timestamp":   time.Now().Add(-2 * time.Hour),
				"type":        "session_started",
				"agent":       "agent-1",
				"description": "協作會話開始",
			},
			{
				"timestamp":   time.Now().Add(-90 * time.Minute),
				"type":        "phase_completed",
				"agent":       "agent-1",
				"description": "完成分析階段",
			},
			{
				"timestamp":   time.Now().Add(-45 * time.Minute),
				"type":        "agent_joined",
				"agent":       "agent-2",
				"description": "資料庫專家加入會話",
			},
			{
				"timestamp":   time.Now().Add(-30 * time.Minute),
				"type":        "knowledge_shared",
				"agent":       "agent-3",
				"description": "分享架構設計知識",
			},
		},
	}

	middleware.WriteSuccess(w, timeline, "Timeline retrieved successfully")
}

// HandleCollaborationEffectiveness analyzes collaboration effectiveness
func (h *HTTPHandler) HandleCollaborationEffectiveness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "week"
	}

	effectiveness, err := h.service.GetCollaborationEffectiveness(ctx, period)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法分析協作效果", http.StatusInternalServerError)
		return
	}

	middleware.WriteSuccess(w, effectiveness, "Effectiveness analyzed successfully")
}

// HandleCollaborationRecommendations generates recommendations
func (h *HTTPHandler) HandleCollaborationRecommendations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	taskType := r.URL.Query().Get("task_type")

	recommendations, err := h.service.GetCollaborationRecommendations(ctx, taskType)
	if err != nil {
		h.WriteError(w, "SERVER_ERROR", "無法生成協作建議", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"recommendations": recommendations,
		"count":           len(recommendations),
		"task_type":       taskType,
		"generated_at":    time.Now().Format(time.RFC3339),
	}

	middleware.WriteSuccess(w, response, "Agents listed successfully")
}

// Helper methods
