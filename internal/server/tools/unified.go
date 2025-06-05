// Package tools provides a unified approach to tool management
// This implementation follows Go best practices by merging handler/service patterns
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/observability"
	"github.com/koopa0/assistant-go/internal/server/pghelpers"
	"github.com/koopa0/assistant-go/internal/storage/postgres/sqlc"
	"github.com/koopa0/assistant-go/internal/tools"
	"github.com/koopa0/assistant-go/internal/types"
)

// UnifiedToolManager provides tool management with integrated HTTP handling
// This follows the principle of "discover abstractions, don't create them"
type UnifiedToolManager struct {
	assistant *assistant.Assistant
	queries   *sqlc.Queries
	logger    *slog.Logger
	metrics   *observability.Metrics
}

// New creates a new unified tool manager
func New(assistant *assistant.Assistant, queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *UnifiedToolManager {
	return &UnifiedToolManager{
		assistant: assistant,
		queries:   queries,
		logger:    logger,
		metrics:   metrics,
	}
}

// RegisterRoutes registers HTTP routes for tool operations
func (m *UnifiedToolManager) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /tools", m.handleGetTools)
	mux.HandleFunc("POST /tools/{id}/execute", m.handleExecuteTool)
	mux.HandleFunc("GET /tools/executions/{id}", m.handleGetExecutionStatus)

	// API v1 compatibility
	mux.HandleFunc("GET /api/v1/tools", m.handleGetTools)
	mux.HandleFunc("POST /api/v1/tools/{id}/execute", m.handleExecuteTool)
	mux.HandleFunc("GET /api/v1/tools/executions/{id}", m.handleGetExecutionStatus)
}

// GetTools retrieves available tools with optional filtering
func (m *UnifiedToolManager) GetTools(ctx context.Context, category, status string) ([]types.ToolInfo, error) {
	m.logger.Debug("Getting tools",
		slog.String("category", category),
		slog.String("status", status))

	availableTools := m.assistant.GetAvailableTools()
	tools := make([]types.ToolInfo, 0, len(availableTools))

	for _, tool := range availableTools {
		usage, err := m.getToolUsage(ctx, tool.Name)
		if err != nil {
			m.logger.Warn("Failed to get tool usage",
				slog.String("tool", tool.Name),
				slog.Any("error", err))
			usage = types.ToolUsage{PopularityScore: 0.5}
		}

		toolInfo := types.ToolInfo{
			ID:          tool.Name,
			Name:        tool.Name,
			DisplayName: m.getDisplayName(tool.Name),
			Description: tool.Description,
			Category:    m.getCategory(tool.Name),
			Version:     "1.0.0",
			Status:      "available",
			Usage:       usage,
		}

		// Apply filters
		if category != "" && toolInfo.Category != category {
			continue
		}
		if status != "" && toolInfo.Status != status {
			continue
		}

		tools = append(tools, toolInfo)
	}

	return tools, nil
}

// ExecuteToolWithInput executes a tool with typed input
func (m *UnifiedToolManager) ExecuteToolWithInput(ctx context.Context, toolName string, input types.ToolInput, async bool) (*types.ToolOutput, error) {
	m.logger.Debug("Executing tool",
		slog.String("tool", toolName),
		slog.Bool("async", async))

	startTime := time.Now()

	// Convert typed input to map for legacy assistant interface
	inputMap := m.convertInputToMap(input)

	// Execute the tool
	result, err := m.assistant.ExecuteTool(ctx, &assistant.ToolExecutionRequest{
		ToolName: toolName,
		Input:    inputMap,
	})

	duration := time.Since(startTime)

	// Create typed output
	output := &types.ToolOutput{
		Success:  err == nil && result != nil && result.Success,
		Duration: duration,
		Metadata: map[string]string{
			"tool_name":    toolName,
			"async":        strconv.FormatBool(async),
			"started_at":   startTime.Format(time.RFC3339),
			"completed_at": time.Now().Format(time.RFC3339),
		},
	}

	if err != nil {
		output.Error = err.Error()
		m.recordExecution(ctx, toolName, inputMap, nil, err, duration)
		return output, nil
	}

	if result != nil {
		// Convert result to typed output based on tool type
		m.convertResultToTypedOutput(result, output, input)
		resultMap := m.convertToolResultToMap(result)
		m.recordExecution(ctx, toolName, inputMap, resultMap, nil, duration)
	}

	return output, nil
}

// HTTP Handlers

func (m *UnifiedToolManager) handleGetTools(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	category := r.URL.Query().Get("category")
	status := r.URL.Query().Get("status")

	tools, err := m.GetTools(ctx, category, status)
	if err != nil {
		m.writeError(w, "Failed to get tools", http.StatusInternalServerError)
		return
	}

	// Calculate statistics
	categoryStats := make(map[string]int)
	statusStats := make(map[string]int)
	for _, tool := range tools {
		categoryStats[tool.Category]++
		statusStats[tool.Status]++
	}

	data := types.ToolListResponse{
		Tools: tools,
		Total: len(tools),
		Stats: types.ToolListStats{
			ByCategory: categoryStats,
			ByStatus:   statusStats,
		},
	}

	response := types.APIResponse{
		Success: true,
		Data:    data,
	}

	m.writeJSON(w, http.StatusOK, response)
}

func (m *UnifiedToolManager) handleExecuteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	toolID := r.PathValue("id")
	if toolID == "" {
		m.writeError(w, "Tool ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Input *types.ToolInput `json:"input"`
		Async bool             `json:"async,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.Input == nil {
		m.writeError(w, "Tool input required", http.StatusBadRequest)
		return
	}

	result, err := m.ExecuteToolWithInput(ctx, toolID, *req.Input, req.Async)
	if err != nil {
		m.writeError(w, "Tool execution failed", http.StatusInternalServerError)
		return
	}

	statusCode := http.StatusOK
	if !result.Success {
		statusCode = http.StatusBadRequest
	} else if req.Async {
		statusCode = http.StatusAccepted
	}

	response := types.APIResponse{
		Success: result.Success,
		Data:    result,
	}

	m.writeJSON(w, statusCode, response)
}

func (m *UnifiedToolManager) handleGetExecutionStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	executionID := r.PathValue("id")
	if executionID == "" {
		m.writeError(w, "Execution ID required", http.StatusBadRequest)
		return
	}

	status, err := m.getExecutionStatus(ctx, executionID)
	if err != nil {
		m.writeError(w, "Failed to get execution status", http.StatusInternalServerError)
		return
	}

	response := types.APIResponse{
		Success: true,
		Data:    status,
	}

	m.writeJSON(w, http.StatusOK, response)
}

// Helper methods

func (m *UnifiedToolManager) getToolUsage(ctx context.Context, toolName string) (types.ToolUsage, error) {
	stats, err := m.queries.GetToolUsageStatsByTool(ctx, sqlc.GetToolUsageStatsByToolParams{
		ToolName: toolName,
		UserID:   pgtype.UUID{Valid: false},
	})
	if err != nil {
		return types.ToolUsage{}, err
	}

	return types.ToolUsage{
		TotalExecutions: int32(stats.TotalExecutions),
		SuccessfulRuns:  int32(stats.SuccessCount),
		FailedRuns:      int32(stats.FailureCount),
		AverageExecTime: m.convertToFloat64(stats.AvgExecutionTimeMs),
		PopularityScore: m.calculatePopularity(int32(stats.TotalExecutions), float64(stats.SuccessRate)),
	}, nil
}

func (m *UnifiedToolManager) convertInputToMap(input types.ToolInput) map[string]interface{} {
	result := map[string]interface{}{
		"action": input.Action,
		"target": input.Target,
	}

	if input.Options != nil {
		for k, v := range input.Options {
			result[k] = v
		}
	}

	if input.Code != nil {
		result["code"] = map[string]interface{}{
			"language": input.Code.Language,
			"content":  input.Code.Content,
		}
	}

	if input.File != nil {
		result["file"] = map[string]interface{}{
			"path":      input.File.Path,
			"operation": input.File.Operation,
		}
	}

	if input.Query != nil {
		result["query"] = map[string]interface{}{
			"sql":      input.Query.SQL,
			"database": input.Query.Database,
		}
	}

	if input.Command != nil {
		result["command"] = map[string]interface{}{
			"command": input.Command.Command,
			"args":    input.Command.Args,
		}
	}

	return result
}

func (m *UnifiedToolManager) convertResultToTypedOutput(result *tools.ToolResult, output *types.ToolOutput, input types.ToolInput) {
	if result.Data == nil {
		return
	}

	// Convert based on input type
	switch {
	case input.Code != nil:
		output.Code = m.convertToCodeOutput(result.Data)
	case input.File != nil:
		output.File = m.convertToFileOutput(result.Data)
	case input.Query != nil:
		output.Query = m.convertToQueryOutput(result.Data)
	case input.Command != nil:
		output.Command = m.convertToCommandOutput(result.Data)
	default:
		output.Analysis = m.convertToAnalysisOutput(result.Data)
	}
}

func (m *UnifiedToolManager) convertToCodeOutput(data interface{}) *types.CodeOutput {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	return &types.CodeOutput{
		FormattedCode: m.getString(dataMap, "formatted_code"),
		Metrics: types.CodeMetrics{
			LinesOfCode:          m.getInt(dataMap, "lines_of_code"),
			CyclomaticComplexity: m.getInt(dataMap, "complexity"),
			FunctionCount:        m.getInt(dataMap, "function_count"),
		},
	}
}

func (m *UnifiedToolManager) convertToFileOutput(data interface{}) *types.FileOutput {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	return &types.FileOutput{
		Path:     m.getString(dataMap, "path"),
		Size:     int64(m.getInt(dataMap, "size")),
		Modified: m.getBool(dataMap, "modified"),
	}
}

func (m *UnifiedToolManager) convertToQueryOutput(data interface{}) *types.QueryOutput {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	return &types.QueryOutput{
		RowsAffected: int64(m.getInt(dataMap, "rows_affected")),
		Results:      m.getResultsArray(dataMap, "results"),
	}
}

func (m *UnifiedToolManager) convertToCommandOutput(data interface{}) *types.CommandOutput {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	return &types.CommandOutput{
		ExitCode: m.getInt(dataMap, "exit_code"),
		Stdout:   m.getString(dataMap, "stdout"),
		Stderr:   m.getString(dataMap, "stderr"),
	}
}

func (m *UnifiedToolManager) convertToAnalysisOutput(data interface{}) *types.AnalysisOutput {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}

	return &types.AnalysisOutput{
		Summary:    m.getString(dataMap, "summary"),
		Details:    dataMap,
		Confidence: m.getFloat64(dataMap, "confidence"),
	}
}

// Utility methods for safe type conversion
func (m *UnifiedToolManager) getString(data map[string]interface{}, key string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (m *UnifiedToolManager) getInt(data map[string]interface{}, key string) int {
	if v, ok := data[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		}
	}
	return 0
}

func (m *UnifiedToolManager) getBool(data map[string]interface{}, key string) bool {
	if v, ok := data[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func (m *UnifiedToolManager) getFloat64(data map[string]interface{}, key string) float64 {
	if v, ok := data[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		}
	}
	return 0
}

func (m *UnifiedToolManager) getResultsArray(data map[string]interface{}, key string) []map[string]interface{} {
	if v, ok := data[key]; ok {
		if arr, ok := v.([]interface{}); ok {
			results := make([]map[string]interface{}, len(arr))
			for i, item := range arr {
				if itemMap, ok := item.(map[string]interface{}); ok {
					results[i] = itemMap
				}
			}
			return results
		}
	}
	return nil
}

func (m *UnifiedToolManager) getDisplayName(toolName string) string {
	names := map[string]string{
		"go_analyzer":   "Go Code Analyzer",
		"go_formatter":  "Go Code Formatter",
		"go_tester":     "Go Test Runner",
		"postgres_tool": "PostgreSQL Tool",
		"k8s_tool":      "Kubernetes Tool",
		"docker_tool":   "Docker Tool",
	}
	if name, exists := names[toolName]; exists {
		return name
	}
	return toolName
}

func (m *UnifiedToolManager) getCategory(toolName string) string {
	categories := map[string]string{
		"go_analyzer":   "development",
		"go_formatter":  "development",
		"go_tester":     "development",
		"postgres_tool": "database",
		"k8s_tool":      "infrastructure",
		"docker_tool":   "infrastructure",
	}
	if category, exists := categories[toolName]; exists {
		return category
	}
	return "general"
}

func (m *UnifiedToolManager) calculatePopularity(totalExecutions int32, successRate float64) float64 {
	if totalExecutions == 0 {
		return 0
	}

	usageScore := float64(totalExecutions) / 1000.0 // Normalize to 0-1
	if usageScore > 1 {
		usageScore = 1
	}

	successScore := successRate / 100.0
	return (usageScore*0.6 + successScore*0.4)
}

func (m *UnifiedToolManager) convertToFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

func (m *UnifiedToolManager) recordExecution(ctx context.Context, toolName string, input map[string]interface{}, output map[string]interface{}, execErr error, duration time.Duration) {
	inputData, _ := json.Marshal(input)
	var outputData []byte
	if output != nil {
		outputData, _ = json.Marshal(output)
	}

	status := "completed"
	var errorMessage pgtype.Text
	if execErr != nil {
		status = "failed"
		errorMessage = pgtype.Text{String: execErr.Error(), Valid: true}
	}

	now := time.Now()
	params := sqlc.CreateToolExecutionParams{
		ToolName:        toolName,
		MessageID:       pgtype.UUID{Valid: false},
		Status:          status,
		InputData:       inputData,
		OutputData:      outputData,
		ErrorMessage:    errorMessage,
		ExecutionTimeMs: pgtype.Int4{Int32: int32(duration.Milliseconds()), Valid: true},
		StartedAt:       pgtype.Timestamptz{Time: now.Add(-duration), Valid: true},
		CompletedAt:     pgtype.Timestamptz{Time: now, Valid: true},
	}

	if _, err := m.queries.CreateToolExecution(ctx, params); err != nil {
		m.logger.Warn("Failed to record execution", slog.Any("error", err))
	}
}

func (m *UnifiedToolManager) convertToolResultToMap(result *tools.ToolResult) map[string]interface{} {
	if result == nil {
		return map[string]interface{}{"success": false, "error": "no result"}
	}

	resultMap := map[string]interface{}{
		"success": result.Success,
	}

	if result.Data != nil {
		resultMap["data"] = result.Data
	}
	if result.Error != "" {
		resultMap["error"] = result.Error
	}
	if result.Metadata != nil {
		resultMap["metadata"] = result.Metadata
	}

	return resultMap
}

func (m *UnifiedToolManager) getExecutionStatus(ctx context.Context, executionID string) (interface{}, error) {
	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	var pgtypeUUID pgtype.UUID
	if err := pgtypeUUID.Scan(execUUID); err != nil {
		return nil, fmt.Errorf("failed to convert UUID: %w", err)
	}

	execution, err := m.queries.GetToolExecution(ctx, pgtypeUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool execution: %w", err)
	}

	var result map[string]interface{}
	if execution.OutputData != nil {
		json.Unmarshal(execution.OutputData, &result)
	}

	status := map[string]interface{}{
		"execution_id":      execution.ID.String(),
		"tool_name":         execution.ToolName,
		"status":            execution.Status,
		"result":            result,
		"execution_time_ms": pghelpers.PgtypeInt4ToInt32Ptr(execution.ExecutionTimeMs),
		"started_at":        pghelpers.PgtypeTimestamptzToTime(execution.StartedAt),
		"completed_at":      pghelpers.PgtypeTimestamptzToTimePtr(execution.CompletedAt),
	}

	if execution.ErrorMessage.Valid {
		status["error"] = execution.ErrorMessage.String
	}

	return status, nil
}

func (m *UnifiedToolManager) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (m *UnifiedToolManager) writeError(w http.ResponseWriter, message string, statusCode int) {
	response := types.APIResponse{
		Success: false,
		Error: &types.APIError{
			Code:    statusCode,
			Message: message,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	m.writeJSON(w, statusCode, response)
}
