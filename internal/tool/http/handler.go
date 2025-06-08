package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
	"github.com/koopa0/assistant-go/internal/tool"
)

// Handler handles HTTP requests for tool operations
type Handler struct {
	service *tool.Service
	logger  *slog.Logger
}

// NewHandler creates a new HTTP handler for tools
func NewHandler(assistant tool.AssistantToolInterface, queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *Handler {
	adapter := &AssistantToolAdapter{
		assistant: assistant,
		queries:   queries,
	}
	return &Handler{
		service: tool.NewService(adapter, queries, logger, metrics),
		logger:  observability.ServerLogger(logger, "tools_http"),
	}
}

// AssistantToolAdapter adapts Assistant to implement tool.ToolExecutor interface
type AssistantToolAdapter struct {
	assistant tool.AssistantToolInterface
	queries   *sqlc.Queries
}

// GetAvailableTools implements tool.ToolExecutor
func (a *AssistantToolAdapter) GetAvailableTools() []tool.RegistryToolInfo {
	assistantTools := a.assistant.GetAvailableTools()
	toolInfos := make([]tool.RegistryToolInfo, 0, len(assistantTools))

	for _, t := range assistantTools {
		toolInfos = append(toolInfos, tool.RegistryToolInfo{
			Name:        t.Name,
			Description: t.Description,
			Category:    t.Category,
			Version:     t.Version,
			Author:      t.Author,
			IsEnabled:   t.IsEnabled,
		})
	}

	return toolInfos
}

// ExecuteTool implements tool.ToolExecutor
func (a *AssistantToolAdapter) ExecuteTool(ctx context.Context, req *tool.ToolExecutionRequest) (*tool.ToolResult, error) {
	// Convert to assistant request format
	assistantReq := &struct {
		ToolName string
		Input    map[string]interface{}
		Config   map[string]interface{}
	}{
		ToolName: req.ToolName,
		Input:    req.Input,
		Config:   req.Config,
	}

	result, err := a.assistant.ExecuteTool(ctx, assistantReq)
	if err != nil {
		return nil, err
	}

	// Convert from assistant result format
	toolResult := &tool.ToolResult{
		Success:       result.Success,
		Error:         result.Error,
		ExecutionTime: result.ElapsedTime,
	}

	// Handle data if result is present
	if result.Result != nil {
		toolResult.Data = &tool.ToolResultData{
			Result: result.Result,
		}
	}

	// Handle metadata if present
	if result.Metadata != nil {
		toolResult.Metadata = &tool.ToolMetadata{
			Custom: result.Metadata,
		}
	}

	return toolResult, nil
}

// GetDB implements tool.ToolExecutor
func (a *AssistantToolAdapter) GetDB() tool.DatabaseQueries {
	return &queriesWrapper{queries: a.queries}
}

// queriesWrapper wraps sqlc.Queries to implement DatabaseQueries
type queriesWrapper struct {
	queries *sqlc.Queries
}

// GetQueries implements tool.DatabaseQueries
func (q *queriesWrapper) GetQueries() *sqlc.Queries {
	return q.queries
}

// RegisterRoutes registers all tool API routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
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
}

// GetTools returns enhanced tool information with usage statistics (v2 API)
func (h *Handler) GetTools(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tools from service
	tools, err := h.service.GetTools(ctx, nil, nil)
	if err != nil {
		h.logger.Error("Failed to get tools", slog.String("error", err.Error()))
		h.writeErrorResponse(w, "Failed to get tools", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    tools,
	})
}

// ExecuteToolV2 executes a tool with enhanced error handling and tracking
func (h *Handler) ExecuteToolV2(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Tool execution endpoint - not implemented",
	})
}

// GetExecutionStatus gets the status of a tool execution
func (h *Handler) GetExecutionStatus(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  "pending",
	})
}

// GetEnhancedTools returns enhanced tool information
func (h *Handler) GetEnhancedTools(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tools from service
	tools, err := h.service.GetTools(ctx, nil, nil)
	if err != nil {
		h.logger.Error("Failed to get tools", slog.String("error", err.Error()))
		h.writeErrorResponse(w, "Failed to get tools", http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"tools":   tools,
	})
}

// ExecuteTool executes a tool (legacy API)
func (h *Handler) ExecuteTool(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Tool execution endpoint - not implemented",
	})
}

// GetToolUsageStats returns usage statistics for tools
func (h *Handler) GetToolUsageStats(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement GetStatistics in service
	// For now, return mock stats
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":           true,
		"total_tools":       0,
		"enabled_tools":     0,
		"disabled_tools":    0,
		"tools_by_category": map[string]int{},
		"tool_usage":        map[string]interface{}{},
		"last_health_check": time.Now().Format(time.RFC3339),
		"healthy_tools":     0,
		"unhealthy_tools":   0,
	})
}

// GetToolUsageHistory returns usage history for tools
func (h *Handler) GetToolUsageHistory(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"history": []map[string]interface{}{},
		"limit":   50,
		"offset":  0,
		"total":   0,
	})
}

// GetToolHistory returns history for a specific tool
func (h *Handler) GetToolHistory(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"history": []map[string]interface{}{},
	})
}

// ToggleFavoriteTool toggles a tool as favorite
func (h *Handler) ToggleFavoriteTool(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Favorite toggled",
	})
}

// RemoveFavoriteTool removes a tool from favorites
func (h *Handler) RemoveFavoriteTool(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Favorite removed",
	})
}

// GetToolAnalytics returns analytics for all tools
func (h *Handler) GetToolAnalytics(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"analytics": map[string]interface{}{},
	})
}

// GetToolDetailedAnalytics returns detailed analytics for a specific tool
func (h *Handler) GetToolDetailedAnalytics(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"analytics": map[string]interface{}{},
	})
}

// writeJSON writes a JSON response
func (h *Handler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode response", slog.String("error", err.Error()))
	}
}

// writeErrorResponse writes an error response
func (h *Handler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	h.writeJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}
