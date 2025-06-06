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

	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/server/middleware"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// ToolExecutor defines what the service needs to execute tools
// Consumer-defined interface following Go principles
type ToolExecutor interface {
	GetAvailableTools() []RegistryToolInfo
	ExecuteTool(ctx context.Context, req *ToolExecutionRequest) (*ToolResult, error)
	GetDB() DatabaseQueries
}

// RegistryToolInfo represents basic tool information from the registry
type RegistryToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	IsEnabled   bool   `json:"is_enabled"`
}

// DatabaseQueries defines what the service needs from the database
type DatabaseQueries interface {
	GetQueries() *sqlc.Queries
}

// ToolExecutionRequest represents a request to execute a tool
type ToolExecutionRequest struct {
	ToolName string                 `json:"tool_name"`
	Input    map[string]interface{} `json:"input"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Async    bool                   `json:"async,omitempty"`
}

// Service handles tool-related business logic with both legacy and enhanced functionality
type Service struct {
	executor ToolExecutor
	queries  *sqlc.Queries
	logger   *slog.Logger
	metrics  *observability.Metrics
}

// NewService creates a new consolidated tool service
func NewService(executor ToolExecutor, queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *Service {
	return &Service{
		executor: executor,
		queries:  queries,
		logger:   observability.ServerLogger(logger, "tools"),
		metrics:  metrics,
	}
}

// Enhanced service functionality (from EnhancedToolService)

// ToolExecutionResponse represents the response from tool execution
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

// ToolExecutionStatus represents tool execution status
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

// EnhancedToolInfo represents enhanced tool information with usage statistics
type EnhancedToolInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	DisplayName string                 `json:"display_name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Version     string                 `json:"version"`
	Status      string                 `json:"status"` // available, maintenance, deprecated
	Usage       ToolUsageInfo          `json:"usage"`
	Parameters  []ServiceToolParam     `json:"parameters"`
	Examples    []ServiceToolExample   `json:"examples"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ToolUsageInfo represents tool usage statistics
type ToolUsageInfo struct {
	TotalExecutions int32      `json:"total_executions"`
	SuccessfulRuns  int32      `json:"successful_runs"`
	FailedRuns      int32      `json:"failed_runs"`
	AverageExecTime float64    `json:"average_execution_time_ms"`
	LastUsed        *time.Time `json:"last_used,omitempty"`
	PopularityScore float64    `json:"popularity_score"`
}

// ServiceToolParam represents tool parameter definition for service
type ServiceToolParam struct {
	Name        string                      `json:"name"`
	Type        string                      `json:"type"`
	Description string                      `json:"description"`
	Required    bool                        `json:"required"`
	Default     interface{}                 `json:"default,omitempty"`
	Validation  *ServiceParameterValidation `json:"validation,omitempty"`
}

// ServiceParameterValidation represents parameter validation rules for service
type ServiceParameterValidation struct {
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Pattern *string  `json:"pattern,omitempty"`
	Options []string `json:"options,omitempty"`
}

// ServiceToolExample represents tool usage example for service
type ServiceToolExample struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Output      map[string]interface{} `json:"expected_output"`
}

// Enhanced service methods

// GetTools returns enhanced tool information with usage statistics
func (s *Service) GetTools(ctx context.Context, category *string, status *string) ([]EnhancedToolInfo, error) {
	s.logger.Debug("Getting tools",
		slog.Any("category", category),
		slog.Any("status", status))

	// Get available tools from executor
	availableTools := s.executor.GetAvailableTools()

	tools := make([]EnhancedToolInfo, 0, len(availableTools))
	for _, tool := range availableTools {
		// Get tool usage statistics
		usage, err := s.getToolUsageStats(ctx, tool.Name)
		if err != nil {
			s.logger.Warn("Failed to get tool usage stats",
				slog.String("tool", tool.Name),
				slog.Any("error", err))
			// Use default statistics
			usage = ToolUsageInfo{
				PopularityScore: 0.5,
			}
		}

		// Convert tool information
		toolInfo := EnhancedToolInfo{
			ID:          tool.Name, // Use name as ID
			Name:        tool.Name,
			DisplayName: s.getDisplayName(tool.Name),
			Description: tool.Description,
			Category:    tool.Category,
			Version:     tool.Version,
			Status:      "available",
			Usage:       usage,
			Parameters:  s.getToolParameters(tool.Name),
			Examples:    s.getToolExamples(tool.Name),
			Metadata:    make(map[string]interface{}),
		}

		// Apply filters
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

// ExecuteToolEnhanced executes a tool using enhanced request/response format
func (s *Service) ExecuteToolEnhanced(ctx context.Context, toolName string, req ToolExecutionRequest) (*ToolExecutionResponse, error) {
	s.logger.Debug("Executing tool",
		slog.String("tool", toolName),
		slog.Bool("async", req.Async))

	startTime := time.Now()

	// Build execution request
	execReq := &ToolExecutionRequest{
		ToolName: toolName,
		Input:    req.Input,
		Config:   req.Config,
	}

	// Execute tool
	result, err := s.executor.ExecuteTool(ctx, execReq)
	if err != nil {
		// Record failed execution
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

	// Calculate execution time
	executionTime := time.Since(startTime)
	executionTimeMs := int32(executionTime.Milliseconds())

	// Convert ToolResult to map[string]interface{}
	resultMap := s.convertToolResultToMap(result)

	// Record successful execution
	s.recordToolExecution(ctx, toolName, req.Input, resultMap, nil, executionTime)

	// Build response
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

// GetToolExecutionStatus gets the status of a tool execution
func (s *Service) GetToolExecutionStatus(ctx context.Context, executionID string) (*ToolExecutionStatus, error) {
	s.logger.Debug("Getting execution status", slog.String("execution_id", executionID))

	// Validate execution ID
	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID: %w", err)
	}

	// Convert to pgtype.UUID
	var pgtypeUUID pgtype.UUID
	if err := pgtypeUUID.Scan(execUUID); err != nil {
		return nil, fmt.Errorf("failed to convert UUID: %w", err)
	}

	// Get execution record from database
	execution, err := s.queries.GetToolExecution(ctx, pgtypeUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool execution: %w", err)
	}

	// Parse output data
	var result map[string]interface{}
	if execution.OutputData != nil {
		if err := json.Unmarshal(execution.OutputData, &result); err != nil {
			s.logger.Warn("Failed to parse output data",
				slog.String("execution_id", executionID))
		}
	}

	// Build status response
	status := &ToolExecutionStatus{
		ExecutionID:     execution.ID.String(),
		ToolName:        execution.ToolName,
		Status:          execution.Status,
		Result:          result,
		ExecutionTimeMs: middleware.PgtypeInt4ToInt32Ptr(execution.ExecutionTimeMs),
		StartedAt:       middleware.PgtypeTimestamptzToTime(execution.StartedAt),
		CompletedAt:     middleware.PgtypeTimestamptzToTimePtr(execution.CompletedAt),
		Metadata: map[string]interface{}{
			"message_id": execution.MessageID,
		},
	}

	// Add error message if present
	if execution.ErrorMessage.Valid {
		status.Error = &execution.ErrorMessage.String
	}

	s.logger.Debug("Retrieved execution status",
		slog.String("execution_id", executionID),
		slog.String("status", status.Status))

	return status, nil
}

// Legacy service functionality (from ToolService)

// LegacyEnhancedToolInfo represents enhanced tool information for legacy API
type LegacyEnhancedToolInfo struct {
	ToolInfo        EnhancedToolInfo `json:"tool_info"`
	Usage           int64            `json:"usage"`
	LastUsed        *time.Time       `json:"last_used,omitempty"`
	IsFavorite      bool             `json:"is_favorite"`
	AverageRating   float64          `json:"average_rating"`
	ExecutionCount  int64            `json:"execution_count"`
	SuccessRate     float64          `json:"success_rate"`
	AverageExecTime int64            `json:"average_execution_time_ms"`
}

// ToolExecution represents a tool execution record
type ToolExecution struct {
	ID              string                 `json:"id"`
	ToolName        string                 `json:"tool_name"`
	UserID          string                 `json:"user_id"`
	Status          string                 `json:"status"`
	InputData       map[string]interface{} `json:"input_data"`
	OutputData      map[string]interface{} `json:"output_data,omitempty"`
	ErrorMessage    *string                `json:"error_message,omitempty"`
	ExecutionTimeMs int                    `json:"execution_time_ms"`
	StartedAt       time.Time              `json:"started_at"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
}

// ToolUsageStats represents tool usage statistics
type ToolUsageStats struct {
	ToolName        string     `json:"tool_name"`
	TotalExecutions int64      `json:"total_executions"`
	SuccessCount    int64      `json:"success_count"`
	FailureCount    int64      `json:"failure_count"`
	SuccessRate     float64    `json:"success_rate"`
	AvgExecTimeMs   int64      `json:"average_execution_time_ms"`
	LastUsed        *time.Time `json:"last_used,omitempty"`
}

// ExecuteToolRequest represents a tool execution request for legacy API
type ExecuteToolRequest struct {
	ToolName string                 `json:"tool_name"`
	Input    map[string]interface{} `json:"input"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// GetEnhancedTools returns enhanced tool information with usage statistics (legacy)
func (s *Service) GetEnhancedTools(ctx context.Context, userID string) ([]LegacyEnhancedToolInfo, error) {
	// Get available tools from executor
	availableTools := s.executor.GetAvailableTools()

	enhancedTools := make([]LegacyEnhancedToolInfo, 0, len(availableTools))

	for _, toolInfo := range availableTools {
		// Get usage stats for this tool
		usage, err := s.getToolUsageStats(ctx, toolInfo.Name)
		if err != nil {
			s.logger.Warn("Failed to get tool usage stats",
				slog.String("tool", toolInfo.Name),
				slog.Any("error", err))
			// Use default statistics
			usage = ToolUsageInfo{
				PopularityScore: 0.5,
			}
		}

		enhanced := LegacyEnhancedToolInfo{
			ToolInfo: EnhancedToolInfo{
				ID:          toolInfo.Name,
				Name:        toolInfo.Name,
				DisplayName: s.getDisplayName(toolInfo.Name),
				Description: toolInfo.Description,
				Category:    toolInfo.Category,
				Version:     toolInfo.Version,
				Status:      "available",
				Usage:       usage,
				Parameters:  s.getToolParameters(toolInfo.Name),
				Examples:    s.getToolExamples(toolInfo.Name),
				Metadata:    make(map[string]interface{}),
			},
			Usage:           int64(usage.TotalExecutions),
			LastUsed:        usage.LastUsed,
			IsFavorite:      false, // TODO: Check user favorites
			AverageRating:   0.0,   // TODO: Implement ratings
			ExecutionCount:  int64(usage.TotalExecutions),
			SuccessRate:     usage.PopularityScore * 100, // Convert to percentage
			AverageExecTime: int64(usage.AverageExecTime),
		}

		enhancedTools = append(enhancedTools, enhanced)
	}

	return enhancedTools, nil
}

// ExecuteTool executes a tool and records the execution (legacy)
func (s *Service) ExecuteTool(ctx context.Context, userID string, req *ExecuteToolRequest) (*ToolResult, error) {
	startTime := time.Now()

	s.logger.Info("Executing tool",
		slog.String("tool_name", req.ToolName),
		slog.String("user_id", userID))

	// Execute tool through executor
	toolReq := &ToolExecutionRequest{
		ToolName: req.ToolName,
		Input:    req.Input,
		Config:   req.Config,
	}

	result, err := s.executor.ExecuteTool(ctx, toolReq)
	if err != nil {
		s.logger.Error("Tool execution failed",
			slog.String("tool_name", req.ToolName),
			slog.String("user_id", userID),
			slog.Any("error", err))

		// Record failed execution if database is available
		if s.queries != nil {
			s.recordToolExecution(ctx, req.ToolName, req.Input, nil, err, time.Since(startTime))
		}

		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	// Record successful execution if database is available
	if s.queries != nil {
		resultMap := s.convertToolResultToMap(result)
		s.recordToolExecution(ctx, req.ToolName, req.Input, resultMap, nil, time.Since(startTime))
	}

	s.logger.Info("Tool execution completed",
		slog.String("tool_name", req.ToolName),
		slog.String("user_id", userID),
		slog.Bool("success", result.Success),
		slog.Duration("execution_time", time.Since(startTime)))

	return result, nil
}

// GetToolUsageHistory returns tool usage history for a user
func (s *Service) GetToolUsageHistory(ctx context.Context, userID string, toolName string, limit, offset int) ([]ToolExecution, error) {
	if s.queries == nil {
		return []ToolExecution{}, nil
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	var toolNameFilter pgtype.Text
	if toolName != "" {
		toolNameFilter = pgtype.Text{String: toolName, Valid: true}
	}

	executions, err := s.queries.GetToolExecutionsByUser(ctx, sqlc.GetToolExecutionsByUserParams{
		UserID:   pgtype.UUID{Bytes: userUUID, Valid: true},
		ToolName: toolNameFilter,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		s.logger.Error("Failed to get tool usage history",
			slog.String("user_id", userID),
			slog.String("tool_name", toolName),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to get tool usage history: %w", err)
	}

	result := make([]ToolExecution, 0, len(executions))
	for _, exec := range executions {
		result = append(result, s.dbToolExecutionToToolExecution(*exec))
	}

	s.logger.Debug("Retrieved tool usage history",
		slog.String("user_id", userID),
		slog.String("tool_name", toolName),
		slog.Int("count", len(result)))

	return result, nil
}

// GetToolUsageStats returns usage statistics for tools
func (s *Service) GetToolUsageStats(ctx context.Context, userID string) ([]ToolUsageStats, error) {
	if s.queries == nil {
		return []ToolUsageStats{}, nil
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	stats, err := s.queries.GetToolUsageStats(ctx, pgtype.UUID{Bytes: userUUID, Valid: true})
	if err != nil {
		s.logger.Error("Failed to get tool usage stats",
			slog.String("user_id", userID),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to get tool usage stats: %w", err)
	}

	result := make([]ToolUsageStats, 0, len(stats))
	for _, stat := range stats {
		var lastUsed *time.Time
		if lastUsedTime, ok := stat.LastUsed.(time.Time); ok {
			lastUsed = &lastUsedTime
		}

		avgExecTime := int64(0)
		if avgTime, ok := stat.AvgExecutionTimeMs.(int32); ok {
			avgExecTime = int64(avgTime)
		}

		result = append(result, ToolUsageStats{
			ToolName:        stat.ToolName,
			TotalExecutions: int64(stat.TotalExecutions),
			SuccessCount:    int64(stat.SuccessCount),
			FailureCount:    int64(stat.FailureCount),
			SuccessRate:     stat.SuccessRate,
			AvgExecTimeMs:   avgExecTime,
			LastUsed:        lastUsed,
		})
	}

	s.logger.Debug("Retrieved tool usage stats",
		slog.String("user_id", userID),
		slog.Int("tool_count", len(result)))

	return result, nil
}

// ToggleFavoriteTool toggles a tool's favorite status for a user
func (s *Service) ToggleFavoriteTool(ctx context.Context, userID, toolName string, isFavorite bool) error {
	if s.queries == nil {
		return fmt.Errorf("database not available")
	}

	// Get user's current preferences
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := s.queries.GetUserByID(ctx, pgtype.UUID{Bytes: userUUID, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Parse preferences from JSONB
	var preferences map[string]interface{}
	if err := json.Unmarshal(user.Preferences, &preferences); err != nil {
		s.logger.Warn("Failed to parse user preferences", slog.Any("error", err))
		preferences = make(map[string]interface{})
	}

	// Get current favorite tools from preferences
	favoriteTools := []string{}
	if favs, ok := preferences["favoriteTools"]; ok {
		if favsSlice, ok := favs.([]interface{}); ok {
			for _, fav := range favsSlice {
				if favStr, ok := fav.(string); ok {
					favoriteTools = append(favoriteTools, favStr)
				}
			}
		}
	}

	// Update favorite tools list
	if isFavorite {
		// Add to favorites if not already present
		found := false
		for _, tool := range favoriteTools {
			if tool == toolName {
				found = true
				break
			}
		}
		if !found {
			favoriteTools = append(favoriteTools, toolName)
		}
	} else {
		// Remove from favorites
		newFavorites := []string{}
		for _, tool := range favoriteTools {
			if tool != toolName {
				newFavorites = append(newFavorites, tool)
			}
		}
		favoriteTools = newFavorites
	}

	// Update preferences
	newPrefs := make(map[string]interface{})
	for k, v := range preferences {
		newPrefs[k] = v
	}
	newPrefs["favoriteTools"] = favoriteTools

	// Marshal preferences back to JSON bytes
	prefBytes, err := json.Marshal(newPrefs)
	if err != nil {
		s.logger.Error("Failed to marshal preferences", slog.Any("error", err))
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	_, err = s.queries.UpdateUserPreferences(ctx, sqlc.UpdateUserPreferencesParams{
		ID:          pgtype.UUID{Bytes: userUUID, Valid: true},
		Preferences: prefBytes,
	})
	if err != nil {
		s.logger.Error("Failed to update favorite tools",
			slog.String("user_id", userID),
			slog.String("tool_name", toolName),
			slog.Any("error", err))
		return fmt.Errorf("failed to update favorites: %w", err)
	}

	s.logger.Info("Favorite tool updated",
		slog.String("user_id", userID),
		slog.String("tool_name", toolName),
		slog.Bool("is_favorite", isFavorite))

	return nil
}

// Helper methods

// getToolUsageStats gets tool usage statistics
func (s *Service) getToolUsageStats(ctx context.Context, toolName string) (ToolUsageInfo, error) {
	// Use SQLC query to get statistics
	stats, err := s.queries.GetToolUsageStatsByTool(ctx, sqlc.GetToolUsageStatsByToolParams{
		ToolName: toolName,
		// UserID set to NULL indicates query for all users
		UserID: pgtype.UUID{Valid: false},
	})
	if err != nil {
		// If no statistics found, return default values
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

	// Calculate popularity score (based on usage count and success rate)
	popularityScore := calculatePopularityScore(
		int32(stats.TotalExecutions),
		float64(stats.SuccessRate),
	)

	// Convert last used time
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

// calculatePopularityScore calculates tool popularity score
func calculatePopularityScore(totalExecutions int32, successRate float64) float64 {
	// Popularity score algorithm:
	// 1. Logarithmic value of usage count (avoid extreme values)
	// 2. Success rate weighting
	// 3. Normalize to 0-1 range

	if totalExecutions == 0 {
		return 0
	}

	// Use logarithm to smooth extreme values
	usageScore := math.Log10(float64(totalExecutions)+1) / 4 // Assume 10000 times is maximum usage

	// Success rate impact (0-1)
	successScore := successRate / 100.0

	// Combined score (usage 60%, success rate 40%)
	popularity := (usageScore * 0.6) + (successScore * 0.4)

	// Ensure within 0-1 range
	if popularity > 1 {
		popularity = 1
	}
	if popularity < 0 {
		popularity = 0
	}

	// Round to two decimal places
	return math.Round(popularity*100) / 100
}

// getDisplayName gets tool display name
func (s *Service) getDisplayName(toolName string) string {
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

// getToolParameters gets tool parameter definitions
func (s *Service) getToolParameters(toolName string) []ServiceToolParam {
	// TODO: Get parameters dynamically from tool definition
	// Currently return predefined parameters
	switch toolName {
	case "go_analyzer":
		return []ServiceToolParam{
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
				Validation: &ServiceParameterValidation{
					Options: []string{"syntax", "semantic", "full"},
				},
			},
		}
	case "postgres_tool":
		return []ServiceToolParam{
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
		return []ServiceToolParam{}
	}
}

// getToolExamples gets tool usage examples
func (s *Service) getToolExamples(toolName string) []ServiceToolExample {
	// TODO: Get examples dynamically from tool definition
	switch toolName {
	case "go_analyzer":
		return []ServiceToolExample{
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
		return []ServiceToolExample{}
	}
}

// recordToolExecution records a tool execution
func (s *Service) recordToolExecution(ctx context.Context, toolName string, input map[string]interface{}, output map[string]interface{}, execErr error, duration time.Duration) {
	now := time.Now()

	// Serialize input data
	inputData, err := json.Marshal(input)
	if err != nil {
		s.logger.Warn("Failed to marshal input data", slog.Any("error", err))
		return
	}

	// Serialize output data
	var outputData []byte
	if output != nil {
		outputData, err = json.Marshal(output)
		if err != nil {
			s.logger.Warn("Failed to marshal output data", slog.Any("error", err))
		}
	}

	// Determine status
	status := "completed"
	var errorMessage pgtype.Text
	if execErr != nil {
		status = "failed"
		errorMessage = pgtype.Text{String: execErr.Error(), Valid: true}
	}

	// Create execution record
	params := sqlc.CreateToolExecutionParams{
		ToolName:        toolName,
		MessageID:       pgtype.UUID{Valid: false}, // Direct API call has no associated message
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

// convertToolResultToMap converts ToolResult to map[string]interface{}
func (s *Service) convertToolResultToMap(result *ToolResult) map[string]interface{} {
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

// dbToolExecutionToToolExecution converts a database tool execution to our ToolExecution type
func (s *Service) dbToolExecutionToToolExecution(exec sqlc.ToolExecution) ToolExecution {
	var completedAt *time.Time
	if exec.CompletedAt.Valid {
		completedAt = &exec.CompletedAt.Time
	}

	var executionTimeMs int
	if exec.ExecutionTimeMs.Valid {
		executionTimeMs = int(exec.ExecutionTimeMs.Int32)
	}

	var inputData map[string]interface{}
	if exec.InputData != nil {
		// Parse JSON bytes from database
		if err := json.Unmarshal(exec.InputData, &inputData); err != nil {
			s.logger.Warn("Failed to parse input data JSON", slog.Any("error", err))
			inputData = make(map[string]interface{})
		}
	}

	var outputData map[string]interface{}
	if exec.OutputData != nil {
		// Parse JSON bytes from database
		if err := json.Unmarshal(exec.OutputData, &outputData); err != nil {
			s.logger.Warn("Failed to parse output data JSON", slog.Any("error", err))
		}
	}

	var errorMessage *string
	if exec.ErrorMessage.Valid {
		errorMessage = &exec.ErrorMessage.String
	}

	return ToolExecution{
		ID:              uuid.UUID(exec.ID.Bytes).String(),
		ToolName:        exec.ToolName,
		UserID:          "", // Not directly available in schema
		Status:          exec.Status,
		InputData:       inputData,
		OutputData:      outputData,
		ErrorMessage:    errorMessage,
		ExecutionTimeMs: executionTimeMs,
		StartedAt:       exec.StartedAt.Time,
		CompletedAt:     completedAt,
	}
}

// convertToFloat64 safely converts interface{} to float64
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
