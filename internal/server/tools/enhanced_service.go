package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/observability"
	"github.com/koopa0/assistant-go/internal/server/pghelpers"
	"github.com/koopa0/assistant-go/internal/storage/postgres/sqlc"
	"github.com/koopa0/assistant-go/internal/tools"
)

// EnhancedToolService 提供增強的工具管理功能
type EnhancedToolService struct {
	assistant *assistant.Assistant
	queries   *sqlc.Queries
	logger    *slog.Logger
	metrics   *observability.Metrics
}

// NewEnhancedToolService 建立新的增強工具服務
func NewEnhancedToolService(assistant *assistant.Assistant, queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *EnhancedToolService {
	return &EnhancedToolService{
		assistant: assistant,
		queries:   queries,
		logger:    logger,
		metrics:   metrics,
	}
}

// ToolInfo 工具資訊
type ToolInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	DisplayName string                 `json:"display_name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Version     string                 `json:"version"`
	Status      string                 `json:"status"` // available, maintenance, deprecated
	Usage       ToolUsageInfo          `json:"usage"`
	Parameters  []ToolParameter        `json:"parameters"`
	Examples    []ToolExample          `json:"examples"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ToolUsageInfo 工具使用統計
type ToolUsageInfo struct {
	TotalExecutions int32      `json:"total_executions"`
	SuccessfulRuns  int32      `json:"successful_runs"`
	FailedRuns      int32      `json:"failed_runs"`
	AverageExecTime float64    `json:"average_execution_time_ms"`
	LastUsed        *time.Time `json:"last_used,omitempty"`
	PopularityScore float64    `json:"popularity_score"`
}

// ToolParameter 工具參數定義
type ToolParameter struct {
	Name        string               `json:"name"`
	Type        string               `json:"type"`
	Description string               `json:"description"`
	Required    bool                 `json:"required"`
	Default     interface{}          `json:"default,omitempty"`
	Validation  *ParameterValidation `json:"validation,omitempty"`
}

// ParameterValidation 參數驗證規則
type ParameterValidation struct {
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Pattern *string  `json:"pattern,omitempty"`
	Options []string `json:"options,omitempty"`
}

// ToolExample 工具使用範例
type ToolExample struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"expected_output"`
}

// ToolExecutionRequest 工具執行請求
type ToolExecutionRequest struct {
	Input  map[string]interface{} `json:"input"`
	Config map[string]interface{} `json:"config,omitempty"`
	Async  bool                   `json:"async,omitempty"`
}

// ToolExecutionResponse 工具執行回應
type ToolExecutionResponse struct {
	ExecutionID     string                 `json:"execution_id"`
	Status          string                 `json:"status"`
	Result          map[string]interface{} `json:"result,omitempty"`
	Error           *string                `json:"error,omitempty"`
	ExecutionTimeMs *int32                 `json:"execution_time_ms,omitempty"`
	StartedAt       time.Time              `json:"started_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ToolExecutionStatus 工具執行狀態
type ToolExecutionStatus struct {
	ExecutionID     string                 `json:"execution_id"`
	ToolName        string                 `json:"tool_name"`
	Status          string                 `json:"status"`
	Progress        *int32                 `json:"progress,omitempty"` // 0-100
	Result          map[string]interface{} `json:"result,omitempty"`
	Error           *string                `json:"error,omitempty"`
	ExecutionTimeMs *int32                 `json:"execution_time_ms,omitempty"`
	StartedAt       time.Time              `json:"started_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	Logs            []string               `json:"logs,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// GetTools 取得工具列表
func (s *EnhancedToolService) GetTools(ctx context.Context, category *string, status *string) ([]ToolInfo, error) {
	s.logger.Debug("Getting tools",
		slog.Any("category", category),
		slog.Any("status", status))

	// 從 assistant 取得可用工具
	availableTools := s.assistant.GetAvailableTools()

	tools := make([]ToolInfo, 0, len(availableTools))
	for _, tool := range availableTools {
		// 取得工具使用統計
		usage, err := s.getToolUsageStats(ctx, tool.Name)
		if err != nil {
			s.logger.Warn("Failed to get tool usage stats",
				slog.String("tool", tool.Name),
				slog.Any("error", err))
			// 使用預設統計
			usage = ToolUsageInfo{
				PopularityScore: 0.5,
			}
		}

		// 轉換工具資訊
		toolInfo := ToolInfo{
			ID:          tool.Name, // 使用名稱作為 ID
			Name:        tool.Name,
			DisplayName: s.getDisplayName(tool.Name),
			Description: tool.Description,
			Category:    s.getToolCategory(tool.Name),
			Version:     "1.0.0", // TODO: 從工具中取得版本
			Status:      "available",
			Usage:       usage,
			Parameters:  s.getToolParameters(tool.Name),
			Examples:    s.getToolExamples(tool.Name),
			Metadata:    make(map[string]interface{}),
		}

		// 應用過濾器
		if category != nil && toolInfo.Category != *category {
			continue
		}
		if status != nil && toolInfo.Status != *status {
			continue
		}

		tools = append(tools, toolInfo)
	}

	s.logger.Debug("Retrieved tools", slog.Int("count", len(tools)))
	return tools, nil
}

// ExecuteTool 執行工具
func (s *EnhancedToolService) ExecuteTool(ctx context.Context, toolName string, req ToolExecutionRequest) (*ToolExecutionResponse, error) {
	s.logger.Debug("Executing tool",
		slog.String("tool", toolName),
		slog.Bool("async", req.Async))

	startTime := time.Now()

	// 建立執行請求
	execReq := &assistant.ToolExecutionRequest{
		ToolName: toolName,
		Input:    req.Input,
		Config:   req.Config,
	}

	// 執行工具
	result, err := s.assistant.ExecuteTool(ctx, execReq)
	if err != nil {
		// 記錄失敗的執行
		s.recordToolExecution(ctx, toolName, req.Input, nil, err, time.Since(startTime))

		errorMsg := err.Error()
		now := time.Now()
		return &ToolExecutionResponse{
			ExecutionID: uuid.New().String(),
			Status:      "failed",
			Error:       &errorMsg,
			StartedAt:   startTime,
			CompletedAt: &now,
		}, nil
	}

	// 計算執行時間
	executionTime := time.Since(startTime)
	executionTimeMs := int32(executionTime.Milliseconds())

	// 轉換 ToolResult 到 map[string]interface{}
	resultMap := s.convertToolResultToMap(result)

	// 記錄成功的執行
	s.recordToolExecution(ctx, toolName, req.Input, resultMap, nil, executionTime)

	// 建立回應
	now := time.Now()
	response := &ToolExecutionResponse{
		ExecutionID:     uuid.New().String(),
		Status:          "completed",
		Result:          resultMap,
		ExecutionTimeMs: &executionTimeMs,
		StartedAt:       startTime,
		CompletedAt:     &now,
		Metadata: map[string]interface{}{
			"tool_version":      "1.0.0",
			"execution_context": "api",
		},
	}

	s.logger.Debug("Tool executed successfully",
		slog.String("tool", toolName),
		slog.Int64("execution_time_ms", executionTime.Milliseconds()))

	return response, nil
}

// GetToolExecutionStatus 取得工具執行狀態
func (s *EnhancedToolService) GetToolExecutionStatus(ctx context.Context, executionID string) (*ToolExecutionStatus, error) {
	s.logger.Debug("Getting execution status", slog.String("execution_id", executionID))

	// 驗證執行 ID
	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	// 轉換為 pgtype.UUID
	var pgtypeUUID pgtype.UUID
	if err := pgtypeUUID.Scan(execUUID); err != nil {
		return nil, fmt.Errorf("failed to convert UUID: %w", err)
	}

	// 從資料庫取得執行記錄
	execution, err := s.queries.GetToolExecution(ctx, pgtypeUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool execution: %w", err)
	}

	// 解析輸出資料
	var result map[string]interface{}
	if execution.OutputData != nil {
		if err := json.Unmarshal(execution.OutputData, &result); err != nil {
			s.logger.Warn("Failed to parse output data",
				slog.String("execution_id", executionID))
		}
	}

	// 建立狀態回應
	status := &ToolExecutionStatus{
		ExecutionID:     execution.ID.String(),
		ToolName:        execution.ToolName,
		Status:          execution.Status,
		Result:          result,
		ExecutionTimeMs: pghelpers.PgtypeInt4ToInt32Ptr(execution.ExecutionTimeMs),
		StartedAt:       pghelpers.PgtypeTimestamptzToTime(execution.StartedAt),
		CompletedAt:     pghelpers.PgtypeTimestamptzToTimePtr(execution.CompletedAt),
		Metadata: map[string]interface{}{
			"message_id": execution.MessageID,
		},
	}

	// 如果有錯誤訊息
	if execution.ErrorMessage.Valid {
		status.Error = &execution.ErrorMessage.String
	}

	s.logger.Debug("Retrieved execution status",
		slog.String("execution_id", executionID),
		slog.String("status", status.Status))

	return status, nil
}

// 輔助方法

// getToolUsageStats 取得工具使用統計
func (s *EnhancedToolService) getToolUsageStats(ctx context.Context, toolName string) (ToolUsageInfo, error) {
	// 使用 SQLC 查詢取得統計資料
	stats, err := s.queries.GetToolUsageStatsByTool(ctx, sqlc.GetToolUsageStatsByToolParams{
		ToolName: toolName,
		// UserID 設為 NULL 表示查詢所有用戶
		UserID: pgtype.UUID{Valid: false},
	})
	if err != nil {
		// 如果沒有找到統計資料，返回預設值
		if err.Error() == "sql: no rows in result set" {
			return ToolUsageInfo{
				TotalExecutions: 0,
				SuccessfulRuns:  0,
				FailedRuns:      0,
				AverageExecTime: 0,
				PopularityScore: 0,
			}, nil
		}
		return ToolUsageInfo{}, fmt.Errorf("failed to get tool usage stats: %w", err)
	}

	// 計算人氣分數（基於使用次數和成功率）
	popularityScore := calculatePopularityScore(
		int32(stats.TotalExecutions),
		float64(stats.SuccessRate),
	)

	// 轉換最後使用時間
	var lastUsed *time.Time
	if lastUsedTimestamp, ok := stats.LastUsed.(pgtype.Timestamptz); ok && lastUsedTimestamp.Valid {
		lastUsed = &lastUsedTimestamp.Time
	}

	return ToolUsageInfo{
		TotalExecutions: int32(stats.TotalExecutions),
		SuccessfulRuns:  int32(stats.SuccessCount),
		FailedRuns:      int32(stats.FailureCount),
		AverageExecTime: convertToFloat64(stats.AvgExecutionTimeMs),
		LastUsed:        lastUsed,
		PopularityScore: popularityScore,
	}, nil
}

// calculatePopularityScore 計算工具的人氣分數
func calculatePopularityScore(totalExecutions int32, successRate float64) float64 {
	// 人氣分數算法：
	// 1. 使用次數的對數值（避免極端值）
	// 2. 成功率的權重
	// 3. 正規化到 0-1 範圍

	if totalExecutions == 0 {
		return 0
	}

	// 使用對數來平滑極端值
	usageScore := math.Log10(float64(totalExecutions)+1) / 4 // 假設 10000 次是最高使用量

	// 成功率影響（0-1）
	successScore := successRate / 100.0

	// 組合分數（使用量佔 60%，成功率佔 40%）
	popularity := (usageScore * 0.6) + (successScore * 0.4)

	// 確保在 0-1 範圍內
	if popularity > 1 {
		popularity = 1
	}
	if popularity < 0 {
		popularity = 0
	}

	// 四捨五入到小數點後兩位
	return math.Round(popularity*100) / 100
}

// getDisplayName 取得工具顯示名稱
func (s *EnhancedToolService) getDisplayName(toolName string) string {
	displayNames := map[string]string{
		"go_analyzer":   "Go 程式碼分析器",
		"go_formatter":  "Go 程式碼格式化器",
		"go_tester":     "Go 測試執行器",
		"postgres_tool": "PostgreSQL 工具",
		"k8s_tool":      "Kubernetes 工具",
		"docker_tool":   "Docker 工具",
	}

	if display, exists := displayNames[toolName]; exists {
		return display
	}
	return toolName
}

// getToolCategory 取得工具分類
func (s *EnhancedToolService) getToolCategory(toolName string) string {
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

// getToolParameters 取得工具參數定義
func (s *EnhancedToolService) getToolParameters(toolName string) []ToolParameter {
	// TODO: 從工具定義中動態取得參數
	// 目前返回預定義的參數
	switch toolName {
	case "go_analyzer":
		return []ToolParameter{
			{
				Name:        "code",
				Type:        "string",
				Description: "要分析的 Go 程式碼",
				Required:    true,
			},
			{
				Name:        "analysis_type",
				Type:        "string",
				Description: "分析類型",
				Required:    false,
				Default:     "full",
				Validation: &ParameterValidation{
					Options: []string{"syntax", "semantic", "full"},
				},
			},
		}
	case "postgres_tool":
		return []ToolParameter{
			{
				Name:        "query",
				Type:        "string",
				Description: "要執行的 SQL 查詢",
				Required:    true,
			},
			{
				Name:        "database",
				Type:        "string",
				Description: "資料庫名稱",
				Required:    false,
				Default:     "assistant",
			},
		}
	default:
		return []ToolParameter{}
	}
}

// getToolExamples 取得工具使用範例
func (s *EnhancedToolService) getToolExamples(toolName string) []ToolExample {
	// TODO: 從工具定義中動態取得範例
	switch toolName {
	case "go_analyzer":
		return []ToolExample{
			{
				Name:        "分析簡單函數",
				Description: "分析一個簡單的 Go 函數",
				Input: map[string]interface{}{
					"code":          "func Add(a, b int) int { return a + b }",
					"analysis_type": "full",
				},
				Output: map[string]interface{}{
					"issues":      []interface{}{},
					"complexity":  1,
					"suggestions": []string{"Add unit tests"},
				},
			},
		}
	default:
		return []ToolExample{}
	}
}

// recordToolExecution 記錄工具執行
func (s *EnhancedToolService) recordToolExecution(ctx context.Context, toolName string, input map[string]interface{}, output map[string]interface{}, execErr error, duration time.Duration) {
	now := time.Now()

	// 序列化輸入資料
	inputData, err := json.Marshal(input)
	if err != nil {
		s.logger.Warn("Failed to marshal input data", slog.Any("error", err))
		return
	}

	// 序列化輸出資料
	var outputData []byte
	if output != nil {
		outputData, err = json.Marshal(output)
		if err != nil {
			s.logger.Warn("Failed to marshal output data", slog.Any("error", err))
		}
	}

	// 確定狀態
	status := "completed"
	var errorMessage pgtype.Text
	if execErr != nil {
		status = "failed"
		errorMessage = pgtype.Text{String: execErr.Error(), Valid: true}
	}

	// 建立執行記錄
	params := sqlc.CreateToolExecutionParams{
		ToolName:        toolName,
		MessageID:       pgtype.UUID{Valid: false}, // 直接 API 調用沒有關聯訊息
		Status:          status,
		InputData:       inputData,
		OutputData:      outputData,
		ErrorMessage:    errorMessage,
		ExecutionTimeMs: pgtype.Int4{Int32: int32(duration.Milliseconds()), Valid: true},
		StartedAt:       pgtype.Timestamptz{Time: now.Add(-duration), Valid: true},
		CompletedAt:     pgtype.Timestamptz{Time: now, Valid: true},
	}

	_, err = s.queries.CreateToolExecution(ctx, params)
	if err != nil {
		s.logger.Warn("Failed to record tool execution",
			slog.String("tool", toolName),
			slog.Any("error", err))
		return
	}

	s.logger.Debug("Tool execution recorded successfully",
		slog.String("tool", toolName),
		slog.Int64("duration_ms", duration.Milliseconds()),
		slog.Bool("success", execErr == nil))
}

// convertToolResultToMap 轉換 ToolResult 到 map[string]interface{}
func (s *EnhancedToolService) convertToolResultToMap(result *tools.ToolResult) map[string]interface{} {
	if result == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "no result",
		}
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

	if result.ExecutionTime > 0 {
		resultMap["execution_time_ms"] = result.ExecutionTime.Milliseconds()
	}

	return resultMap
}

// GetAllToolsUsageStats 取得所有工具的使用統計
func (s *EnhancedToolService) GetAllToolsUsageStats(ctx context.Context, userID *uuid.UUID) ([]ToolUsageInfo, error) {
	s.logger.Debug("Getting all tools usage stats",
		slog.Any("user_id", userID))

	// 準備查詢參數
	var userUUID pgtype.UUID
	if userID != nil {
		userUUID = pghelpers.UUIDToPgtypeUUID(*userID)
	} else {
		userUUID = pgtype.UUID{Valid: false}
	}

	// 查詢統計資料
	stats, err := s.queries.GetToolUsageStats(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool usage stats: %w", err)
	}

	// 轉換結果
	results := make([]ToolUsageInfo, 0, len(stats))
	for _, stat := range stats {
		var lastUsed *time.Time
		if lastUsedTimestamp, ok := stat.LastUsed.(pgtype.Timestamptz); ok && lastUsedTimestamp.Valid {
			lastUsed = &lastUsedTimestamp.Time
		}

		popularityScore := calculatePopularityScore(
			int32(stat.TotalExecutions),
			float64(stat.SuccessRate),
		)

		results = append(results, ToolUsageInfo{
			TotalExecutions: int32(stat.TotalExecutions),
			SuccessfulRuns:  int32(stat.SuccessCount),
			FailedRuns:      int32(stat.FailureCount),
			AverageExecTime: convertToFloat64(stat.AvgExecutionTimeMs),
			LastUsed:        lastUsed,
			PopularityScore: popularityScore,
		})
	}

	s.logger.Debug("Retrieved tools usage stats",
		slog.Int("count", len(results)))

	return results, nil
}

// GetMostUsedTools 取得最常使用的工具
func (s *EnhancedToolService) GetMostUsedTools(ctx context.Context, userID uuid.UUID, limit int32) ([]struct {
	ToolName   string
	UsageCount int32
	LastUsed   *time.Time
}, error) {
	s.logger.Debug("Getting most used tools",
		slog.String("user_id", userID.String()),
		slog.Int("limit", int(limit)))

	userUUID := pghelpers.UUIDToPgtypeUUID(userID)

	// 查詢最常使用的工具
	tools, err := s.queries.GetMostUsedTools(ctx, sqlc.GetMostUsedToolsParams{
		UserID: userUUID,
		Limit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get most used tools: %w", err)
	}

	// 轉換結果
	results := make([]struct {
		ToolName   string
		UsageCount int32
		LastUsed   *time.Time
	}, 0, len(tools))

	for _, tool := range tools {
		var lastUsed *time.Time
		if lastUsedTimestamp, ok := tool.LastUsed.(pgtype.Timestamptz); ok && lastUsedTimestamp.Valid {
			lastUsed = &lastUsedTimestamp.Time
		}

		results = append(results, struct {
			ToolName   string
			UsageCount int32
			LastUsed   *time.Time
		}{
			ToolName:   tool.ToolName,
			UsageCount: int32(tool.UsageCount),
			LastUsed:   lastUsed,
		})
	}

	return results, nil
}

// GetToolExecutionTrends 取得工具執行趨勢
func (s *EnhancedToolService) GetToolExecutionTrends(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]struct {
	Date       time.Time
	ToolName   string
	Executions int32
	Successes  int32
	Failures   int32
}, error) {
	s.logger.Debug("Getting tool execution trends",
		slog.String("user_id", userID.String()),
		slog.Time("start_date", startDate),
		slog.Time("end_date", endDate))

	userUUID := pghelpers.UUIDToPgtypeUUID(userID)
	startTimestamp := pghelpers.TimeToPgtypeTimestamptz(startDate)
	endTimestamp := pghelpers.TimeToPgtypeTimestamptz(endDate)

	// 查詢執行趨勢
	trends, err := s.queries.GetToolExecutionTrends(ctx, sqlc.GetToolExecutionTrendsParams{
		UserID:      userUUID,
		StartedAt:   startTimestamp,
		StartedAt_2: endTimestamp,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get tool execution trends: %w", err)
	}

	// 轉換結果
	results := make([]struct {
		Date       time.Time
		ToolName   string
		Executions int32
		Successes  int32
		Failures   int32
	}, 0, len(trends))

	for _, trend := range trends {
		results = append(results, struct {
			Date       time.Time
			ToolName   string
			Executions int32
			Successes  int32
			Failures   int32
		}{
			Date:       trend.ExecutionDate.Time,
			ToolName:   trend.ToolName,
			Executions: int32(trend.Executions),
			Successes:  int32(trend.Successes),
			Failures:   int32(trend.Failures),
		})
	}

	return results, nil
}

// convertToFloat64 安全地將 interface{} 轉換為 float64
func convertToFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
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
