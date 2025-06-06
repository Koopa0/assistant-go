package toolsapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"context"
	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
	"github.com/koopa0/assistant-go/internal/tools"
)

// AssistantToolAdapter adapts Assistant to implement tools.ToolExecutor interface
type AssistantToolAdapter struct {
	assistant *assistant.Assistant
}

// GetAvailableTools implements tools.ToolExecutor
func (a *AssistantToolAdapter) GetAvailableTools() []tools.RegistryToolInfo {
	assistantTools := a.assistant.GetAvailableTools()
	toolInfos := make([]tools.RegistryToolInfo, 0, len(assistantTools))

	for _, tool := range assistantTools {
		toolInfos = append(toolInfos, tools.RegistryToolInfo{
			Name:        tool.Name,
			Description: tool.Description,
			Category:    tool.Category,
			Version:     tool.Version,
			Author:      tool.Author,
			IsEnabled:   tool.IsEnabled,
		})
	}

	return toolInfos
}

// ExecuteTool implements tools.ToolExecutor
func (a *AssistantToolAdapter) ExecuteTool(ctx context.Context, req *tools.ToolExecutionRequest) (*tools.ToolResult, error) {
	// Convert to assistant request format
	assistantReq := &assistant.ToolExecutionRequest{
		ToolName: req.ToolName,
		Input:    req.Input,
		Config:   req.Config,
	}

	return a.assistant.ExecuteTool(ctx, assistantReq)
}

// GetDB implements tools.ToolExecutor
func (a *AssistantToolAdapter) GetDB() tools.DatabaseQueries {
	return a.assistant.GetDB()
}

// HTTPHandler handles HTTP requests for tool operations
type HTTPHandler struct {
	service *tools.Service
	logger  *slog.Logger
}

// NewHTTPHandler creates a new HTTP handler for tools
func NewHTTPHandler(assistant *assistant.Assistant, queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *HTTPHandler {
	adapter := &AssistantToolAdapter{assistant: assistant}
	return &HTTPHandler{
		service: tools.NewService(adapter, queries, logger, metrics),
		logger:  observability.ServerLogger(logger, "tools_http"),
	}
}

// RegisterRoutes registers all tool API routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	// Enhanced tool information and execution (v2 API)
	mux.HandleFunc("GET /api/v2/tools", h.GetTools)
	mux.HandleFunc("POST /api/v2/tools/{id}/execute", h.ExecuteToolV2)
	mux.HandleFunc("GET /api/v2/tools/executions/{id}", h.GetExecutionStatus)

	// Legacy tool API (v1)
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

	// Backward compatibility routes
	mux.HandleFunc("GET /tools", h.GetTools)
	mux.HandleFunc("POST /tools/{id}/execute", h.ExecuteToolV2)
	mux.HandleFunc("GET /tools/executions/{id}", h.GetExecutionStatus)
}

// GetTools returns enhanced tool information with usage statistics (v2 API)
func (h *HTTPHandler) GetTools(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	var category *string
	if categoryStr := r.URL.Query().Get("category"); categoryStr != "" {
		category = &categoryStr
	}

	var status *string
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status = &statusStr
	}

	// Get tools list
	tools, err := h.service.GetTools(ctx, category, status)
	if err != nil {
		h.logger.Error("Failed to get tools", slog.Any("error", err))
		h.writeErrorResponse(w, "取得工具列表失敗", http.StatusInternalServerError)
		return
	}

	// Calculate category and status statistics
	categoryStats := make(map[string]int)
	statusStats := make(map[string]int)
	for _, tool := range tools {
		categoryStats[tool.Category]++
		statusStats[tool.Status]++
	}

	// Build response
	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"tools": tools,
			"total": len(tools),
			"stats": map[string]interface{}{
				"by_category": categoryStats,
				"by_status":   statusStats,
			},
		},
		"message": "工具列表取得成功",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// ExecuteToolV2 executes a tool using the enhanced service (v2 API)
func (h *HTTPHandler) ExecuteToolV2(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tool ID
	toolID := r.PathValue("id")
	if toolID == "" {
		h.writeErrorResponse(w, "工具 ID 是必需的", http.StatusBadRequest)
		return
	}

	// Parse request body
	var req struct {
		Input  map[string]interface{} `json:"input"`
		Config map[string]interface{} `json:"config,omitempty"`
		Async  bool                   `json:"async,omitempty"`
	}
	if err := h.parseJSONRequest(r, &req); err != nil {
		h.logger.Warn("Invalid tool execution request", slog.Any("error", err))
		h.writeErrorResponse(w, "無效的請求格式", http.StatusBadRequest)
		return
	}

	// Validate input parameters
	if req.Input == nil {
		req.Input = make(map[string]interface{})
	}

	// Convert to tools package request type
	toolsReq := tools.ToolExecutionRequest{
		ToolName: toolID,
		Input:    req.Input,
		Config:   req.Config,
		Async:    req.Async,
	}

	// Execute tool
	result, err := h.service.ExecuteToolEnhanced(ctx, toolID, toolsReq)
	if err != nil {
		h.logger.Error("Failed to execute tool",
			slog.String("tool", toolID),
			slog.Any("error", err))
		h.writeErrorResponse(w, "工具執行失敗", http.StatusInternalServerError)
		return
	}

	// Determine response status code
	statusCode := http.StatusOK
	if result.Status == "failed" {
		statusCode = http.StatusBadRequest
	} else if req.Async {
		statusCode = http.StatusAccepted
	}

	// Build response
	response := map[string]interface{}{
		"success": result.Status != "failed",
		"data":    result,
		"message": h.getExecutionMessage(result.Status),
	}

	h.writeJSONResponse(w, statusCode, response)
}

// GetExecutionStatus gets the status of a tool execution (v2 API)
func (h *HTTPHandler) GetExecutionStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get execution ID
	executionID := r.PathValue("id")
	if executionID == "" {
		h.writeErrorResponse(w, "執行 ID 是必需的", http.StatusBadRequest)
		return
	}

	// Get execution status
	status, err := h.service.GetToolExecutionStatus(ctx, executionID)
	if err != nil {
		h.logger.Error("Failed to get execution status",
			slog.String("execution_id", executionID),
			slog.Any("error", err))
		h.writeErrorResponse(w, "取得執行狀態失敗", http.StatusInternalServerError)
		return
	}

	// Build response
	response := map[string]interface{}{
		"success": true,
		"data":    status,
		"message": "執行狀態取得成功",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetEnhancedTools returns enhanced tool information with usage statistics (legacy v1 API)
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

// ExecuteTool executes a tool and records the execution (legacy v1 API)
func (h *HTTPHandler) ExecuteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from auth context (optional)
	userID, _ := ctx.Value(observability.UserIDKey).(string)

	var req tools.ExecuteToolRequest
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

// Placeholder methods for other endpoints
func (h *HTTPHandler) GetToolUsageStats(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusNotImplemented, map[string]interface{}{
		"success": false,
		"message": "Not implemented yet",
	})
}

func (h *HTTPHandler) GetToolUsageHistory(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusNotImplemented, map[string]interface{}{
		"success": false,
		"message": "Not implemented yet",
	})
}

func (h *HTTPHandler) GetToolHistory(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusNotImplemented, map[string]interface{}{
		"success": false,
		"message": "Not implemented yet",
	})
}

func (h *HTTPHandler) ToggleFavoriteTool(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusNotImplemented, map[string]interface{}{
		"success": false,
		"message": "Not implemented yet",
	})
}

func (h *HTTPHandler) RemoveFavoriteTool(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusNotImplemented, map[string]interface{}{
		"success": false,
		"message": "Not implemented yet",
	})
}

func (h *HTTPHandler) GetToolAnalytics(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusNotImplemented, map[string]interface{}{
		"success": false,
		"message": "Not implemented yet",
	})
}

func (h *HTTPHandler) GetToolDetailedAnalytics(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusNotImplemented, map[string]interface{}{
		"success": false,
		"message": "Not implemented yet",
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

// writeJSONResponse writes JSON response for v2 API
func (h *HTTPHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", slog.Any("error", err))
	}
}

// writeErrorResponse writes error response for v2 API
func (h *HTTPHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"message": message,
			"code":    statusCode,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	h.writeJSONResponse(w, statusCode, response)
}

// parseJSONRequest parses JSON request
func (h *HTTPHandler) parseJSONRequest(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}

// getExecutionMessage gets execution message
func (h *HTTPHandler) getExecutionMessage(status string) string {
	messages := map[string]string{
		"completed": "工具執行成功",
		"failed":    "工具執行失敗",
		"running":   "工具正在執行中",
		"pending":   "工具執行已排隊",
		"cancelled": "工具執行已取消",
	}

	if message, exists := messages[status]; exists {
		return message
	}
	return "工具執行狀態未知"
}
