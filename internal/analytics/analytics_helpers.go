package analytics

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// getCurrentStats gets current statistics for the dashboard
func (s *AnalyticsService) getCurrentStats(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	stats := map[string]interface{}{}

	// Get active conversations count
	conversations, err := s.db.GetRecentConversations(ctx, sqlc.GetRecentConversationsParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Limit:  100,
	})
	if err == nil {
		// Count all recent conversations as active
		stats["active_conversations"] = len(conversations)
	} else {
		stats["active_conversations"] = 0
	}

	// Get tasks completed today
	today := time.Now().Truncate(24 * time.Hour)
	events, err := s.db.GetLearningEvents(ctx, sqlc.GetLearningEventsParams{
		Column1:   pgtype.UUID{Bytes: userID, Valid: true},
		Column2:   nil,
		SessionID: pgtype.Text{Valid: false},
		CreatedAt: today,
		Limit:     1000,
		Offset:    0,
	})

	tasksCompleted := 0
	if err == nil {
		for _, event := range events {
			if event.Outcome.Valid && event.Outcome.String == "success" {
				tasksCompleted++
			}
		}
	}
	stats["tasks_completed"] = tasksCompleted

	// Calculate code quality score based on success rate
	totalTasks := len(events)
	qualityScore := 0.5 // Default
	if totalTasks > 0 {
		qualityScore = float64(tasksCompleted) / float64(totalTasks)
	}
	stats["code_quality_score"] = math.Round(qualityScore*100) / 100

	// Get knowledge nodes count - simplified for now
	// TODO: Implement when knowledge node queries are available
	stats["knowledge_nodes"] = 42 // Placeholder

	// Calculate learning progress (simplified)
	learningProgress := 0.5
	patterns, err := s.db.GetLearnedPatterns(ctx, sqlc.GetLearnedPatternsParams{
		Column1:    pgtype.UUID{Bytes: userID, Valid: true},
		Column2:    nil,
		Confidence: 0, // Get all patterns regardless of confidence
		Limit:      100,
		Offset:     0,
	})
	if err == nil && len(patterns) > 0 {
		totalConfidence := 0.0
		for _, pattern := range patterns {
			totalConfidence += pattern.Confidence
		}
		learningProgress = totalConfidence / float64(len(patterns))
	}
	stats["learning_progress"] = math.Round(learningProgress*100) / 100

	// Collaboration score (simplified)
	stats["collaboration_score"] = 0.75 // Placeholder for now

	return stats, nil
}

// getRecentActivities gets recent user activities
func (s *AnalyticsService) getRecentActivities(ctx context.Context, userID uuid.UUID) ([]map[string]interface{}, error) {
	activities := []map[string]interface{}{}

	// Get recent learning events
	events, err := s.db.GetLearningEvents(ctx, sqlc.GetLearningEventsParams{
		Column1:   pgtype.UUID{Bytes: userID, Valid: true},
		Column2:   nil,
		SessionID: pgtype.Text{Valid: false},
		CreatedAt: time.Now().Add(-24 * time.Hour),
		Limit:     10,
		Offset:    0,
	})

	if err != nil {
		return activities, err
	}

	// Convert events to activities
	for _, event := range events {
		activity := map[string]interface{}{
			"type":      event.EventType,
			"timestamp": event.CreatedAt,
		}

		// Determine title based on event type
		switch event.EventType {
		case "code_completion":
			activity["title"] = "完成代碼編寫"
			activity["impact"] = "medium"
		case "refactoring":
			activity["title"] = "重構代碼"
			activity["impact"] = "high"
		case "debugging":
			activity["title"] = "調試問題"
			activity["impact"] = "high"
		case "query_response":
			activity["title"] = "回答查詢"
			activity["impact"] = "low"
		case "tool_usage":
			activity["title"] = "使用工具"
			activity["impact"] = "medium"
		default:
			activity["title"] = "其他活動"
			activity["impact"] = "low"
		}

		activities = append(activities, activity)

		if len(activities) >= 5 { // Limit to 5 recent activities
			break
		}
	}

	return activities, nil
}

// getPerformanceMetrics gets performance metrics
func (s *AnalyticsService) getPerformanceMetrics(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	metrics := map[string]interface{}{}

	// Get execution statistics for today
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	trends, err := s.db.GetToolExecutionTrends(ctx, sqlc.GetToolExecutionTrendsParams{
		UserID:      pgtype.UUID{Bytes: userID, Valid: true},
		StartedAt:   pgtype.Timestamptz{Time: today, Valid: true},
		StartedAt_2: pgtype.Timestamptz{Time: tomorrow, Valid: true},
	})

	totalExecutions := int32(0)
	totalSuccesses := int32(0)
	avgResponseTime := 125.0 // Default

	if err == nil {
		for _, trend := range trends {
			totalExecutions += trend.Executions
			totalSuccesses += trend.Successes
		}
	}

	// Calculate success rate
	successRate := 0.95 // Default
	if totalExecutions > 0 {
		successRate = float64(totalSuccesses) / float64(totalExecutions)
	}

	metrics["response_time_ms"] = avgResponseTime
	metrics["success_rate"] = math.Round(successRate*100) / 100
	metrics["api_calls_today"] = totalExecutions
	metrics["cache_hit_rate"] = 0.76 // Placeholder for now

	return metrics, nil
}

// generateDashboardRecommendations generates personalized recommendations
func (s *AnalyticsService) generateDashboardRecommendations(ctx context.Context, userID uuid.UUID, stats, metrics map[string]interface{}) []map[string]interface{} {
	recommendations := []map[string]interface{}{}

	// Productivity recommendation based on activity patterns
	activityMetrics, _, _ := s.GetActivityAnalytics(ctx, 7) // Last 7 days
	if activityMetrics != nil {
		// Check morning activity
		morningActivity := false
		for _, hour := range activityMetrics.PeakHours {
			if hour >= 6 && hour <= 11 {
				morningActivity = true
				break
			}
		}

		if !morningActivity {
			recommendations = append(recommendations, map[string]interface{}{
				"type":        "productivity",
				"title":       "建議在早上安排深度工作",
				"description": "根據您的活動模式，早上時段利用率較低",
				"priority":    "medium",
			})
		}
	}

	// Learning recommendation based on recent activities
	if qualityScore, ok := stats["code_quality_score"].(float64); ok && qualityScore < 0.8 {
		recommendations = append(recommendations, map[string]interface{}{
			"type":        "learning",
			"title":       "提升代碼品質",
			"description": "最近的成功率略低，建議加強測試和代碼審查",
			"priority":    "high",
		})
	}

	// Health recommendation based on continuous work
	if tasksCompleted, ok := stats["tasks_completed"].(int); ok && tasksCompleted > 20 {
		recommendations = append(recommendations, map[string]interface{}{
			"type":        "health",
			"title":       "休息時間提醒",
			"description": "今天已完成大量任務，記得適當休息",
			"priority":    "low",
		})
	}

	// Knowledge recommendation
	if knowledgeNodes, ok := stats["knowledge_nodes"].(int); ok && knowledgeNodes < 50 {
		recommendations = append(recommendations, map[string]interface{}{
			"type":        "knowledge",
			"title":       "擴展知識圖譜",
			"description": "增加更多知識節點可以提升系統智能",
			"priority":    "medium",
		})
	}

	return recommendations
}
