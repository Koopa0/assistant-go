package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/observability"
	"github.com/koopa0/assistant-go/internal/storage/postgres/sqlc"
	"github.com/koopa0/assistant-go/internal/tools"
)

// ToolService handles tool-related business logic
type ToolService struct {
	assistant *assistant.Assistant
	queries   *sqlc.Queries
	logger    *slog.Logger
	metrics   *observability.Metrics
}

// NewToolService creates a new tool service
func NewToolService(assistant *assistant.Assistant, queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *ToolService {
	return &ToolService{
		assistant: assistant,
		queries:   queries,
		logger:    observability.ServerLogger(logger, "tools"),
		metrics:   metrics,
	}
}

// EnhancedToolInfo represents enhanced tool information
type EnhancedToolInfo struct {
	tools.ToolInfo
	Usage           int64      `json:"usage"`
	LastUsed        *time.Time `json:"last_used,omitempty"`
	IsFavorite      bool       `json:"is_favorite"`
	AverageRating   float64    `json:"average_rating"`
	ExecutionCount  int64      `json:"execution_count"`
	SuccessRate     float64    `json:"success_rate"`
	AverageExecTime int64      `json:"average_execution_time_ms"`
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

// ExecuteToolRequest represents a tool execution request
type ExecuteToolRequest struct {
	ToolName string                 `json:"tool_name"`
	Input    map[string]interface{} `json:"input"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// GetEnhancedTools returns enhanced tool information with usage statistics
func (s *ToolService) GetEnhancedTools(ctx context.Context, userID string) ([]EnhancedToolInfo, error) {
	// Get available tools from assistant
	toolInfos := s.assistant.GetAvailableTools()

	enhancedTools := make([]EnhancedToolInfo, 0, len(toolInfos))

	for _, toolInfo := range toolInfos {
		enhanced := EnhancedToolInfo{
			ToolInfo: toolInfo,
		}

		// Get usage statistics if database is available
		if s.queries != nil {
			// TODO: Implement tool usage statistics queries
			// This would require additional SQL queries to get:
			// - Usage count
			// - Last used time
			// - Success rate
			// - Average execution time
			enhanced.Usage = 0
			enhanced.ExecutionCount = 0
			enhanced.SuccessRate = 0.0
			enhanced.AverageExecTime = 0
		}

		enhancedTools = append(enhancedTools, enhanced)
	}

	return enhancedTools, nil
}

// ExecuteTool executes a tool and records the execution
func (s *ToolService) ExecuteTool(ctx context.Context, userID string, req *ExecuteToolRequest) (*tools.ToolResult, error) {
	startTime := time.Now()

	s.logger.Info("Executing tool",
		slog.String("tool_name", req.ToolName),
		slog.String("user_id", userID))

	// Execute tool through assistant
	toolReq := &assistant.ToolExecutionRequest{
		ToolName: req.ToolName,
		Input:    req.Input,
		Config:   req.Config,
	}

	result, err := s.assistant.ExecuteTool(ctx, toolReq)
	if err != nil {
		s.logger.Error("Tool execution failed",
			slog.String("tool_name", req.ToolName),
			slog.String("user_id", userID),
			slog.Any("error", err))

		// Record failed execution if database is available
		if s.queries != nil {
			s.recordToolExecution(ctx, userID, req.ToolName, req.Input, nil, err, time.Since(startTime))
		}

		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	// Record successful execution if database is available
	if s.queries != nil {
		s.recordToolExecution(ctx, userID, req.ToolName, req.Input, result.Data, nil, time.Since(startTime))
	}

	s.logger.Info("Tool execution completed",
		slog.String("tool_name", req.ToolName),
		slog.String("user_id", userID),
		slog.Bool("success", result.Success),
		slog.Duration("execution_time", time.Since(startTime)))

	return result, nil
}

// GetToolUsageHistory returns tool usage history for a user
func (s *ToolService) GetToolUsageHistory(ctx context.Context, userID string, toolName string, limit, offset int) ([]ToolExecution, error) {
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
func (s *ToolService) GetToolUsageStats(ctx context.Context, userID string) ([]ToolUsageStats, error) {
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
func (s *ToolService) ToggleFavoriteTool(ctx context.Context, userID, toolName string, isFavorite bool) error {
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

// recordToolExecution records a tool execution in the database
func (s *ToolService) recordToolExecution(ctx context.Context, userID, toolName string, input map[string]interface{}, output interface{}, execErr error, duration time.Duration) {
	if s.queries == nil {
		return
	}

	status := "completed"
	var errorMessage *string
	var outputData interface{}

	if execErr != nil {
		status = "failed"
		errMsg := execErr.Error()
		errorMessage = &errMsg
	} else {
		outputData = output
	}

	// Marshal input and output data to JSON bytes
	inputBytes, err := json.Marshal(input)
	if err != nil {
		s.logger.Error("Failed to marshal input data", slog.Any("error", err))
		return
	}

	var outputBytes []byte
	if outputData != nil {
		outputBytes, err = json.Marshal(outputData)
		if err != nil {
			s.logger.Error("Failed to marshal output data", slog.Any("error", err))
			outputBytes = nil
		}
	}

	// Note: For now, we record tool executions without a message_id
	// In a full implementation, this would be linked to a specific message
	startedAt := time.Now().Add(-duration)
	completedAt := time.Now()

	var errorMsgField pgtype.Text
	if errorMessage != nil {
		errorMsgField = pgtype.Text{String: *errorMessage, Valid: true}
	}

	_, err = s.queries.CreateToolExecution(ctx, sqlc.CreateToolExecutionParams{
		ToolName:        toolName,
		MessageID:       pgtype.UUID{}, // No message context available
		Status:          status,
		InputData:       inputBytes,
		OutputData:      outputBytes,
		ErrorMessage:    errorMsgField,
		ExecutionTimeMs: pgtype.Int4{Int32: int32(duration.Milliseconds()), Valid: true},
		StartedAt:       pgtype.Timestamptz{Time: startedAt, Valid: true},
		CompletedAt:     pgtype.Timestamptz{Time: completedAt, Valid: status == "completed"},
	})

	if err != nil {
		s.logger.Error("Failed to record tool execution",
			slog.String("tool_name", toolName),
			slog.String("user_id", userID),
			slog.String("status", status),
			slog.Any("error", err))
		return
	}

	s.logger.Debug("Recorded tool execution",
		slog.String("tool_name", toolName),
		slog.String("user_id", userID),
		slog.String("status", status),
		slog.Duration("duration", duration))
}

// dbToolExecutionToToolExecution converts a database tool execution to our ToolExecution type
func (s *ToolService) dbToolExecutionToToolExecution(exec sqlc.ToolExecution) ToolExecution {
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
