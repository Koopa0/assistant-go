package tools

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// EnhancedHTTPHandler 處理工具系統的 HTTP 請求
type EnhancedHTTPHandler struct {
	service *EnhancedToolService
	logger  *slog.Logger
}

// NewEnhancedHTTPHandler 建立新的增強 HTTP 處理器
func NewEnhancedHTTPHandler(service *EnhancedToolService, logger *slog.Logger) *EnhancedHTTPHandler {
	return &EnhancedHTTPHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterEnhancedRoutes 註冊增強工具路由
func (h *EnhancedHTTPHandler) RegisterEnhancedRoutes(mux *http.ServeMux) {
	// 工具系統 API
	mux.HandleFunc("GET /tools", h.handleGetTools)
	mux.HandleFunc("POST /tools/{id}/execute", h.handleExecuteTool)
	mux.HandleFunc("GET /tools/executions/{id}", h.handleGetExecutionStatus)

	// API v1 相容性
	mux.HandleFunc("GET /api/v1/tools", h.handleGetTools)
	mux.HandleFunc("POST /api/v1/tools/{id}/execute", h.handleExecuteTool)
	mux.HandleFunc("GET /api/v1/tools/executions/{id}", h.handleGetExecutionStatus)
}

// handleGetTools 處理取得工具列表的請求
func (h *EnhancedHTTPHandler) handleGetTools(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 解析查詢參數
	var category *string
	if categoryStr := r.URL.Query().Get("category"); categoryStr != "" {
		category = &categoryStr
	}

	var status *string
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status = &statusStr
	}

	// 取得工具列表
	tools, err := h.service.GetTools(ctx, category, status)
	if err != nil {
		h.logger.Error("Failed to get tools", slog.Any("error", err))
		h.writeErrorResponse(w, "取得工具列表失敗", http.StatusInternalServerError)
		return
	}

	// 計算分類統計
	categoryStats := make(map[string]int)
	statusStats := make(map[string]int)
	for _, tool := range tools {
		categoryStats[tool.Category]++
		statusStats[tool.Status]++
	}

	// 回應
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

// handleExecuteTool 處理執行工具的請求
func (h *EnhancedHTTPHandler) handleExecuteTool(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 取得工具 ID
	toolID := r.PathValue("id")
	if toolID == "" {
		h.writeErrorResponse(w, "工具 ID 是必需的", http.StatusBadRequest)
		return
	}

	// 解析請求體
	var req ToolExecutionRequest
	if err := h.parseJSONRequest(r, &req); err != nil {
		h.logger.Warn("Invalid tool execution request", slog.Any("error", err))
		h.writeErrorResponse(w, "無效的請求格式", http.StatusBadRequest)
		return
	}

	// 驗證輸入參數
	if req.Input == nil {
		h.writeErrorResponse(w, "工具輸入參數是必需的", http.StatusBadRequest)
		return
	}

	// 執行工具
	result, err := h.service.ExecuteTool(ctx, toolID, req)
	if err != nil {
		h.logger.Error("Failed to execute tool",
			slog.String("tool", toolID),
			slog.Any("error", err))
		h.writeErrorResponse(w, "工具執行失敗", http.StatusInternalServerError)
		return
	}

	// 決定回應狀態碼
	statusCode := http.StatusOK
	if result.Status == "failed" {
		statusCode = http.StatusBadRequest
	} else if req.Async {
		statusCode = http.StatusAccepted
	}

	// 回應
	response := map[string]interface{}{
		"success": result.Status != "failed",
		"data":    result,
		"message": h.getExecutionMessage(result.Status),
	}

	h.writeJSONResponse(w, statusCode, response)
}

// handleGetExecutionStatus 處理取得執行狀態的請求
func (h *EnhancedHTTPHandler) handleGetExecutionStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 取得執行 ID
	executionID := r.PathValue("id")
	if executionID == "" {
		h.writeErrorResponse(w, "執行 ID 是必需的", http.StatusBadRequest)
		return
	}

	// 取得執行狀態
	status, err := h.service.GetToolExecutionStatus(ctx, executionID)
	if err != nil {
		h.logger.Error("Failed to get execution status",
			slog.String("execution_id", executionID),
			slog.Any("error", err))
		h.writeErrorResponse(w, "取得執行狀態失敗", http.StatusInternalServerError)
		return
	}

	// 回應
	response := map[string]interface{}{
		"success": true,
		"data":    status,
		"message": "執行狀態取得成功",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// 輔助方法

// getExecutionMessage 取得執行訊息
func (h *EnhancedHTTPHandler) getExecutionMessage(status string) string {
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

// parseJSONRequest 解析 JSON 請求
func (h *EnhancedHTTPHandler) parseJSONRequest(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}

// writeJSONResponse 寫入 JSON 回應
func (h *EnhancedHTTPHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", slog.Any("error", err))
	}
}

// writeErrorResponse 寫入錯誤回應
func (h *EnhancedHTTPHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
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
