package tools

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/koopa0/assistant-go/internal/observability"
)

// HTTPHandler handles HTTP requests for tool operations
type HTTPHandler struct {
	service *ToolService
	logger  *slog.Logger
}

// NewHTTPHandler creates a new HTTP handler for tools
func NewHTTPHandler(service *ToolService, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  observability.ServerLogger(logger, "tools_http"),
	}
}

// RegisterRoutes registers all tool API routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	// Enhanced tool information and execution
	mux.HandleFunc("GET /api/tools/enhanced", h.GetEnhancedTools)
	mux.HandleFunc("POST /api/tools/execute", h.ExecuteTool)

	// Tool usage and history
	mux.HandleFunc("GET /api/tools/usage/stats", h.GetToolUsageStats)
	mux.HandleFunc("GET /api/tools/usage/history", h.GetToolUsageHistory)
	mux.HandleFunc("GET /api/tools/{toolName}/history", h.GetToolHistory)

	// Tool favorites (requires authentication)
	mux.HandleFunc("POST /api/tools/{toolName}/favorite", h.ToggleFavoriteTool)
	mux.HandleFunc("DELETE /api/tools/{toolName}/favorite", h.RemoveFavoriteTool)

	// Tool analytics and insights
	mux.HandleFunc("GET /api/tools/analytics", h.GetToolAnalytics)
	mux.HandleFunc("GET /api/tools/{toolName}/analytics", h.GetToolDetailedAnalytics)
}

// GetEnhancedTools returns enhanced tool information with usage statistics
func (h *HTTPHandler) GetEnhancedTools(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from auth context (optional for enhanced data)
	userID, _ := ctx.Value(observability.UserIDKey).(string)

	tools, err := h.service.GetEnhancedTools(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get enhanced tools",
			slog.String("user_id", userID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "取得工具資訊失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    tools,
		"message": "取得工具資訊成功",
	})
}

// ExecuteTool executes a tool and records the execution
func (h *HTTPHandler) ExecuteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from auth context (optional)
	userID, _ := ctx.Value(observability.UserIDKey).(string)

	var req ExecuteToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "請求格式錯誤")
		return
	}

	// Validate request
	if req.ToolName == "" {
		h.writeError(w, http.StatusBadRequest, "工具名稱是必需的")
		return
	}

	if req.Input == nil {
		req.Input = make(map[string]interface{})
	}

	result, err := h.service.ExecuteTool(ctx, userID, &req)
	if err != nil {
		h.logger.Error("Failed to execute tool",
			slog.String("tool_name", req.ToolName),
			slog.String("user_id", userID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "工具執行失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
		"message": "工具執行成功",
	})
}

// GetToolUsageStats returns tool usage statistics
func (h *HTTPHandler) GetToolUsageStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from auth context
	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	stats, err := h.service.GetToolUsageStats(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get tool usage stats",
			slog.String("user_id", userID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "取得工具使用統計失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    stats,
		"message": "取得工具使用統計成功",
	})
}

// GetToolUsageHistory returns tool usage history
func (h *HTTPHandler) GetToolUsageHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from auth context
	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	// Parse pagination parameters
	limit, offset := h.parsePagination(r)

	// Get tool name filter (optional)
	toolName := r.URL.Query().Get("tool_name")

	history, err := h.service.GetToolUsageHistory(ctx, userID, toolName, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get tool usage history",
			slog.String("user_id", userID),
			slog.String("tool_name", toolName),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "取得工具使用歷史失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    history,
		"message": "取得工具使用歷史成功",
	})
}

// GetToolHistory returns history for a specific tool
func (h *HTTPHandler) GetToolHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from auth context
	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	toolName := r.PathValue("toolName")
	if toolName == "" {
		h.writeError(w, http.StatusBadRequest, "工具名稱是必需的")
		return
	}

	// Parse pagination parameters
	limit, offset := h.parsePagination(r)

	history, err := h.service.GetToolUsageHistory(ctx, userID, toolName, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get tool history",
			slog.String("user_id", userID),
			slog.String("tool_name", toolName),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "取得工具歷史失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    history,
		"message": "取得工具歷史成功",
	})
}

// ToggleFavoriteTool toggles a tool's favorite status
func (h *HTTPHandler) ToggleFavoriteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from auth context
	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	toolName := r.PathValue("toolName")
	if toolName == "" {
		h.writeError(w, http.StatusBadRequest, "工具名稱是必需的")
		return
	}

	// Parse request body for favorite status
	var req struct {
		IsFavorite bool `json:"is_favorite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "請求格式錯誤")
		return
	}

	err := h.service.ToggleFavoriteTool(ctx, userID, toolName, req.IsFavorite)
	if err != nil {
		h.logger.Error("Failed to toggle favorite tool",
			slog.String("user_id", userID),
			slog.String("tool_name", toolName),
			slog.Bool("is_favorite", req.IsFavorite),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "更新收藏工具失敗")
		return
	}

	var message string
	if req.IsFavorite {
		message = "工具已新增至收藏"
	} else {
		message = "工具已從收藏中移除"
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": message,
	})
}

// RemoveFavoriteTool removes a tool from favorites
func (h *HTTPHandler) RemoveFavoriteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from auth context
	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	toolName := r.PathValue("toolName")
	if toolName == "" {
		h.writeError(w, http.StatusBadRequest, "工具名稱是必需的")
		return
	}

	err := h.service.ToggleFavoriteTool(ctx, userID, toolName, false)
	if err != nil {
		h.logger.Error("Failed to remove favorite tool",
			slog.String("user_id", userID),
			slog.String("tool_name", toolName),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "移除收藏工具失敗")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "工具已從收藏中移除",
	})
}

// GetToolAnalytics returns overall tool analytics
func (h *HTTPHandler) GetToolAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from auth context
	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	stats, err := h.service.GetToolUsageStats(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get tool analytics",
			slog.String("user_id", userID),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "取得工具分析失敗")
		return
	}

	// Calculate analytics from stats
	analytics := h.calculateToolAnalytics(stats)

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    analytics,
		"message": "取得工具分析成功",
	})
}

// GetToolDetailedAnalytics returns detailed analytics for a specific tool
func (h *HTTPHandler) GetToolDetailedAnalytics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from auth context
	userID, ok := ctx.Value(observability.UserIDKey).(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "使用者未認證")
		return
	}

	toolName := r.PathValue("toolName")
	if toolName == "" {
		h.writeError(w, http.StatusBadRequest, "工具名稱是必需的")
		return
	}

	// Get tool usage history for analytics
	history, err := h.service.GetToolUsageHistory(ctx, userID, toolName, 100, 0)
	if err != nil {
		h.logger.Error("Failed to get tool detailed analytics",
			slog.String("user_id", userID),
			slog.String("tool_name", toolName),
			slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "取得工具詳細分析失敗")
		return
	}

	// Calculate detailed analytics
	analytics := h.calculateDetailedAnalytics(toolName, history)

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    analytics,
		"message": "取得工具詳細分析成功",
	})
}

// Helper methods

func (h *HTTPHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", slog.Any("error", err))
	}
}

func (h *HTTPHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	h.writeJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   http.StatusText(statusCode),
		"message": message,
	})
}

// Pagination helpers
func (h *HTTPHandler) parsePagination(r *http.Request) (limit, offset int) {
	limitStr := r.URL.Query().Get("limit")
	pageStr := r.URL.Query().Get("page")

	limit = 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	page := 1 // default
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	offset = (page - 1) * limit
	return limit, offset
}

// Analytics calculation helpers

func (h *HTTPHandler) calculateToolAnalytics(stats []ToolUsageStats) map[string]interface{} {
	if len(stats) == 0 {
		return map[string]interface{}{
			"total_tools":          0,
			"total_executions":     0,
			"average_success_rate": 0.0,
			"most_used_tool":       nil,
			"tools_by_usage":       []interface{}{},
		}
	}

	totalExecutions := int64(0)
	var mostUsedTool *ToolUsageStats
	var totalSuccessRate float64

	for i := range stats {
		totalExecutions += stats[i].TotalExecutions
		totalSuccessRate += stats[i].SuccessRate

		if mostUsedTool == nil || stats[i].TotalExecutions > mostUsedTool.TotalExecutions {
			mostUsedTool = &stats[i]
		}
	}

	averageSuccessRate := totalSuccessRate / float64(len(stats))

	return map[string]interface{}{
		"total_tools":          len(stats),
		"total_executions":     totalExecutions,
		"average_success_rate": averageSuccessRate,
		"most_used_tool":       mostUsedTool,
		"tools_by_usage":       stats,
	}
}

func (h *HTTPHandler) calculateDetailedAnalytics(toolName string, history []ToolExecution) map[string]interface{} {
	if len(history) == 0 {
		return map[string]interface{}{
			"tool_name":          toolName,
			"total_executions":   0,
			"success_count":      0,
			"failure_count":      0,
			"success_rate":       0.0,
			"avg_execution_time": 0,
			"recent_executions":  []interface{}{},
		}
	}

	successCount := 0
	totalExecTime := int64(0)

	for _, exec := range history {
		if exec.Status == "completed" {
			successCount++
		}
		totalExecTime += int64(exec.ExecutionTimeMs)
	}

	successRate := float64(successCount) / float64(len(history)) * 100
	avgExecTime := totalExecTime / int64(len(history))

	// Get recent executions (last 10)
	recentCount := 10
	if len(history) < recentCount {
		recentCount = len(history)
	}
	recent := history[:recentCount]

	return map[string]interface{}{
		"tool_name":          toolName,
		"total_executions":   len(history),
		"success_count":      successCount,
		"failure_count":      len(history) - successCount,
		"success_rate":       successRate,
		"avg_execution_time": avgExecTime,
		"recent_executions":  recent,
	}
}
