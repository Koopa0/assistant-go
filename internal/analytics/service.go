// Package analytics provides analytics and visualization services for the Assistant API server.
package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// AnalyticsService handles analytics and visualization logic
type AnalyticsService struct {
	db      *sqlc.Queries
	logger  *slog.Logger
	metrics *observability.Metrics
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(db *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *AnalyticsService {
	return &AnalyticsService{
		db:      db,
		logger:  observability.ServerLogger(logger, "analytics_service"),
		metrics: metrics,
	}
}

// ActivityMetrics represents activity metrics
type ActivityMetrics struct {
	Period         string             `json:"period"`
	TotalEvents    int                `json:"total_events"`
	ActiveDays     int                `json:"active_days"`
	TotalHours     float64            `json:"total_hours"`
	AveragePerDay  float64            `json:"average_per_day"`
	PeakHours      []int              `json:"peak_hours"`
	ActivityByType map[string]int     `json:"activity_by_type"`
	DailyActivity  []DailyActivity    `json:"daily_activity"`
	WeeklyPattern  map[string]float64 `json:"weekly_pattern"`
}

// DailyActivity represents daily activity data
type DailyActivity struct {
	Date       string  `json:"date"`
	EventCount int     `json:"event_count"`
	Hours      float64 `json:"hours"`
	Intensity  float64 `json:"intensity"` // 0-1
	MainType   string  `json:"main_type"`
}

// HeatmapData represents heatmap visualization data
type HeatmapData struct {
	Data       []HeatmapCell          `json:"data"`
	XLabels    []string               `json:"x_labels"`
	YLabels    []string               `json:"y_labels"`
	ColorScale ColorScale             `json:"color_scale"`
	Statistics map[string]interface{} `json:"statistics"`
}

// HeatmapCell represents a heatmap cell
type HeatmapCell struct {
	X     int     `json:"x"`
	Y     int     `json:"y"`
	Value float64 `json:"value"`
	Label string  `json:"label,omitempty"`
}

// ColorScale represents color scale configuration
type ColorScale struct {
	Min    float64  `json:"min"`
	Max    float64  `json:"max"`
	Colors []string `json:"colors"`
	Labels []string `json:"labels"`
}

// ProductivityTrend represents productivity trend data
type ProductivityTrend struct {
	Period     string                 `json:"period"`
	DataPoints []ProductivityPoint    `json:"data_points"`
	TrendLine  []float64              `json:"trend_line"`
	Forecast   []ForecastPoint        `json:"forecast"`
	Statistics map[string]interface{} `json:"statistics"`
}

// ProductivityPoint represents a productivity data point
type ProductivityPoint struct {
	Date         string  `json:"date"`
	Productivity float64 `json:"productivity"`
	Hours        float64 `json:"hours"`
	Tasks        int     `json:"tasks"`
	Quality      float64 `json:"quality"`
}

// ForecastPoint represents a forecast data point
type ForecastPoint struct {
	Date         string  `json:"date"`
	Productivity float64 `json:"productivity"`
	Confidence   float64 `json:"confidence"`
}

// GetActivityAnalytics returns activity analytics for a given period
func (s *AnalyticsService) GetActivityAnalytics(ctx context.Context, days int) (*ActivityMetrics, []string, error) {
	if days <= 0 {
		days = 30
	}

	// Get current user ID from context
	userID := ctx.Value("user_id")
	if userID == nil {
		userID = "a0000000-0000-4000-8000-000000000001" // Default user ID
		s.logger.Warn("No user ID in context, using default", slog.String("default_id", userID.(string)))
	}

	// Calculate date range
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Query learning events for activity data
	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		s.logger.Error("Invalid user ID", slog.Any("error", err))
		return nil, nil, fmt.Errorf("invalid user ID: %w", err)
	}

	events, err := s.db.GetLearningEvents(ctx, sqlc.GetLearningEventsParams{
		Column1:   pgtype.UUID{Bytes: userUUID, Valid: true},
		Column2:   nil, // Get all event types
		SessionID: pgtype.Text{Valid: false},
		CreatedAt: startDate,
		Limit:     10000,
		Offset:    0,
	})
	if err != nil {
		s.logger.Error("Failed to get learning events", slog.Any("error", err))
		// Fall back to tool executions if learning events are not available
		events = s.getFallbackActivityData(ctx, userID.(string), startDate, endDate)
	}

	// Process events into daily activities
	dailyMap := make(map[string]*DailyActivity)
	activityByType := make(map[string]int)
	hourlyActivity := make(map[int]int)
	totalEvents := 0
	totalHours := float64(0)

	for _, event := range events {
		date := event.CreatedAt.Format("2006-01-02")
		hour := event.CreatedAt.Hour()

		// Update daily activity
		if _, exists := dailyMap[date]; !exists {
			dailyMap[date] = &DailyActivity{
				Date:       date,
				EventCount: 0,
				Hours:      0,
				Intensity:  0,
				MainType:   "",
			}
		}

		dailyMap[date].EventCount++
		if event.DurationMs.Valid && event.DurationMs.Int32 > 0 {
			dailyMap[date].Hours += float64(event.DurationMs.Int32) / 3600000 // Convert ms to hours
		}

		// Track activity type
		eventType := event.EventType
		activityByType[eventType]++

		// Track hourly pattern
		hourlyActivity[hour]++

		totalEvents++
		if event.DurationMs.Valid && event.DurationMs.Int32 > 0 {
			totalHours += float64(event.DurationMs.Int32) / 3600000
		}
	}

	// Convert map to slice and calculate intensity
	dailyActivities := []DailyActivity{}
	activeDays := 0
	weekdayActivity := make(map[string][]float64)

	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		weekday := strings.ToLower(date.Weekday().String())

		if activity, exists := dailyMap[dateStr]; exists && activity.EventCount > 0 {
			// Determine main type for the day
			maxType := ""
			maxCount := 0
			for eventType, count := range activityByType {
				if count > maxCount {
					maxType = eventType
					maxCount = count
				}
			}
			activity.MainType = maxType
			activity.Intensity = math.Min(activity.Hours/8, 1.0)
			activity.Hours = math.Round(activity.Hours*10) / 10

			dailyActivities = append(dailyActivities, *activity)
			activeDays++

			// Track weekly pattern
			if _, exists := weekdayActivity[weekday]; !exists {
				weekdayActivity[weekday] = []float64{}
			}
			weekdayActivity[weekday] = append(weekdayActivity[weekday], activity.Hours)
		} else {
			// Add empty day
			dailyActivities = append(dailyActivities, DailyActivity{
				Date:       dateStr,
				EventCount: 0,
				Hours:      0,
				Intensity:  0,
				MainType:   "",
			})
		}
	}

	// Calculate weekly pattern averages
	weeklyPattern := make(map[string]float64)
	weekdays := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	maxWeekdayHours := 0.0

	for _, weekday := range weekdays {
		if hours, exists := weekdayActivity[weekday]; exists && len(hours) > 0 {
			sum := 0.0
			for _, h := range hours {
				sum += h
			}
			avg := sum / float64(len(hours))
			weeklyPattern[weekday] = avg
			if avg > maxWeekdayHours {
				maxWeekdayHours = avg
			}
		} else {
			weeklyPattern[weekday] = 0
		}
	}

	// Normalize weekly pattern to 0-1 scale
	if maxWeekdayHours > 0 {
		for weekday := range weeklyPattern {
			weeklyPattern[weekday] = weeklyPattern[weekday] / maxWeekdayHours
		}
	}

	// Find peak hours
	peakHours := []int{}
	hourThreshold := totalEvents / (24 * days) * 2 // Above average threshold
	for hour, count := range hourlyActivity {
		if count > hourThreshold {
			peakHours = append(peakHours, hour)
		}
	}
	sort.Ints(peakHours)

	// Calculate averages
	avgPerDay := 0.0
	if activeDays > 0 {
		avgPerDay = math.Round(float64(totalEvents)/float64(activeDays)*10) / 10
	}

	metrics := &ActivityMetrics{
		Period:         fmt.Sprintf("%d days", days),
		TotalEvents:    totalEvents,
		ActiveDays:     activeDays,
		TotalHours:     math.Round(totalHours*10) / 10,
		AveragePerDay:  avgPerDay,
		PeakHours:      peakHours,
		ActivityByType: activityByType,
		DailyActivity:  dailyActivities,
		WeeklyPattern:  weeklyPattern,
	}

	// Generate insights based on actual data
	insights := s.generateActivityInsights(metrics, weeklyPattern, peakHours)

	return metrics, insights, nil
}

// GetActivityHeatmap returns activity heatmap data
func (s *AnalyticsService) GetActivityHeatmap(ctx context.Context) (*HeatmapData, []string, []string, error) {
	// Get current user ID from context
	userID := ctx.Value("user_id")
	if userID == nil {
		userID = "a0000000-0000-4000-8000-000000000001" // Default user ID
		s.logger.Warn("No user ID in context, using default", slog.String("default_id", userID.(string)))
	}

	// Get data for the last 4 weeks to build a representative heatmap
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -28) // 4 weeks

	// Query learning events
	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		s.logger.Error("Invalid user ID", slog.Any("error", err))
		return nil, nil, nil, fmt.Errorf("invalid user ID: %w", err)
	}

	events, err := s.db.GetLearningEvents(ctx, sqlc.GetLearningEventsParams{
		Column1:   pgtype.UUID{Bytes: userUUID, Valid: true},
		Column2:   nil, // Get all event types
		SessionID: pgtype.Text{Valid: false},
		CreatedAt: startDate,
		Limit:     50000, // More data for heatmap
		Offset:    0,
	})
	if err != nil {
		s.logger.Error("Failed to get learning events", slog.Any("error", err))
		// Fall back to tool executions
		events = s.getFallbackActivityData(ctx, userID.(string), startDate, endDate)
	}

	// Initialize heatmap data structure (7 days x 24 hours)
	// Using a map to accumulate activity counts
	activityMap := make(map[string]float64) // key: "day-hour", value: activity intensity
	hourCounts := make(map[string]int)      // Track total events per day-hour

	// Process events
	for _, event := range events {
		weekday := int(event.CreatedAt.Weekday())
		// Convert Sunday (0) to 6 for our display
		if weekday == 0 {
			weekday = 6
		} else {
			weekday = weekday - 1 // Shift Monday to 0
		}

		hour := event.CreatedAt.Hour()
		key := fmt.Sprintf("%d-%d", weekday, hour)

		// Weight by duration if available
		weight := 1.0
		if event.DurationMs.Valid && event.DurationMs.Int32 > 0 {
			// Weight by duration (minutes)
			weight = float64(event.DurationMs.Int32) / 60000.0
		}

		activityMap[key] += weight
		hourCounts[key]++
	}

	// Find max activity for normalization
	maxActivity := 0.0
	for _, activity := range activityMap {
		if activity > maxActivity {
			maxActivity = activity
		}
	}

	// Build heatmap cells
	data := []HeatmapCell{}
	weekLabels := []string{"週一", "週二", "週三", "週四", "週五", "週六", "週日"}

	// Track statistics
	mostActiveTime := ""
	mostActiveValue := 0.0
	leastActiveTime := ""
	leastActiveValue := 999999.0
	activeHours := 0
	totalHours := 24 * 7

	for hour := 0; hour < 24; hour++ {
		for day := 0; day < 7; day++ {
			key := fmt.Sprintf("%d-%d", day, hour)
			value := 0.0

			if activity, exists := activityMap[key]; exists && maxActivity > 0 {
				// Normalize to 0-1 scale
				value = activity / maxActivity
			}

			data = append(data, HeatmapCell{
				X:     day,
				Y:     hour,
				Value: math.Round(value*100) / 100,
			})

			// Track statistics
			timeLabel := fmt.Sprintf("%s %02d:00", weekLabels[day], hour)
			if value > mostActiveValue {
				mostActiveValue = value
				mostActiveTime = timeLabel
			}
			if value > 0 && value < leastActiveValue {
				leastActiveValue = value
				leastActiveTime = timeLabel
			}
			if value > 0.1 { // Consider >10% activity as "active"
				activeHours++
			}
		}
	}

	// Calculate consistency (standard deviation of activity)
	var sum, count float64
	for _, value := range activityMap {
		if value > 0 {
			sum += value
			count++
		}
	}
	mean := sum / count

	var variance float64
	for _, value := range activityMap {
		if value > 0 {
			variance += math.Pow(value-mean, 2)
		}
	}
	stdDev := math.Sqrt(variance / count)
	consistency := 1.0 - (stdDev / mean) // Higher consistency = lower variance
	if consistency < 0 {
		consistency = 0
	}

	heatmap := &HeatmapData{
		Data:    data,
		XLabels: weekLabels,
		YLabels: []string{
			"00:00", "01:00", "02:00", "03:00", "04:00", "05:00",
			"06:00", "07:00", "08:00", "09:00", "10:00", "11:00",
			"12:00", "13:00", "14:00", "15:00", "16:00", "17:00",
			"18:00", "19:00", "20:00", "21:00", "22:00", "23:00",
		},
		ColorScale: ColorScale{
			Min:    0,
			Max:    1,
			Colors: []string{"#f0f0f0", "#ffeda0", "#feb24c", "#fc4e2a", "#e31a1c"},
			Labels: []string{"無活動", "低", "中", "高", "非常高"},
		},
		Statistics: map[string]interface{}{
			"most_active_time":  mostActiveTime,
			"least_active_time": leastActiveTime,
			"weekly_coverage":   float64(activeHours) / float64(totalHours),
			"consistency":       math.Round(consistency*100) / 100,
		},
	}

	// Generate patterns based on actual data
	patterns := s.analyzeHeatmapPatterns(activityMap, weekLabels)

	// Generate recommendations based on patterns
	recommendations := s.generateHeatmapRecommendations(activityMap, consistency)

	return heatmap, patterns, recommendations, nil
}

// GetProductivityTrends returns productivity trends for a given period
func (s *AnalyticsService) GetProductivityTrends(ctx context.Context, period string) (*ProductivityTrend, []string, error) {
	if period == "" {
		period = "month"
	}

	// Get current user ID from context
	userID := ctx.Value("user_id")
	if userID == nil {
		userID = "a0000000-0000-4000-8000-000000000001" // Default user ID
		s.logger.Warn("No user ID in context, using default", slog.String("default_id", userID.(string)))
	}

	// Determine date range based on period
	days := 30
	if period == "week" {
		days = 7
	} else if period == "quarter" {
		days = 90
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Query execution data for productivity metrics
	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		s.logger.Error("Invalid user ID", slog.Any("error", err))
		return nil, nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get tool execution trends for productivity analysis
	trends, err := s.db.GetToolExecutionTrends(ctx, sqlc.GetToolExecutionTrendsParams{
		UserID:      pgtype.UUID{Bytes: userUUID, Valid: true},
		StartedAt:   pgtype.Timestamptz{Time: startDate, Valid: true},
		StartedAt_2: pgtype.Timestamptz{Time: endDate, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to get tool execution trends", slog.Any("error", err))
	}

	// Get learning events for more detailed productivity data
	events, err := s.db.GetLearningEvents(ctx, sqlc.GetLearningEventsParams{
		Column1:   pgtype.UUID{Bytes: userUUID, Valid: true},
		Column2:   nil, // Get all event types
		SessionID: pgtype.Text{Valid: false},
		CreatedAt: startDate,
		Limit:     50000,
		Offset:    0,
	})
	if err != nil {
		s.logger.Error("Failed to get learning events", slog.Any("error", err))
	}

	// Build daily productivity metrics
	dailyMetrics := make(map[string]*ProductivityPoint)

	// Process learning events
	for _, event := range events {
		dateStr := event.CreatedAt.Format("2006-01-02")

		if _, exists := dailyMetrics[dateStr]; !exists {
			dailyMetrics[dateStr] = &ProductivityPoint{
				Date:         dateStr,
				Productivity: 0,
				Hours:        0,
				Tasks:        0,
				Quality:      0,
			}
		}

		// Calculate hours from duration
		if event.DurationMs.Valid && event.DurationMs.Int32 > 0 {
			dailyMetrics[dateStr].Hours += float64(event.DurationMs.Int32) / 3600000
		}

		// Count tasks
		dailyMetrics[dateStr].Tasks++

		// Calculate quality based on outcome
		if event.Outcome.Valid && event.Outcome.String == "success" {
			dailyMetrics[dateStr].Quality += 1.0
		} else if event.Outcome.Valid && event.Outcome.String == "partial" {
			dailyMetrics[dateStr].Quality += 0.5
		}
	}

	// Process tool execution trends
	for _, trend := range trends {
		if trend.ExecutionDate.Valid {
			dateStr := trend.ExecutionDate.Time.Format("2006-01-02")

			if _, exists := dailyMetrics[dateStr]; !exists {
				dailyMetrics[dateStr] = &ProductivityPoint{
					Date:         dateStr,
					Productivity: 0,
					Hours:        0,
					Tasks:        int(trend.Executions),
					Quality:      0,
				}
			} else {
				dailyMetrics[dateStr].Tasks += int(trend.Executions)
			}

			// Calculate quality from success rate
			if trend.Executions > 0 {
				successRate := float64(trend.Successes) / float64(trend.Executions)
				dailyMetrics[dateStr].Quality = successRate
			}
		}
	}

	// Build data points and calculate productivity
	dataPoints := []ProductivityPoint{}
	trendLine := []float64{}

	totalProductivity := 0.0
	nonZeroDays := 0

	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")

		point := ProductivityPoint{
			Date:         dateStr,
			Productivity: 0,
			Hours:        0,
			Tasks:        0,
			Quality:      0,
		}

		if metrics, exists := dailyMetrics[dateStr]; exists {
			point = *metrics

			// Calculate productivity score (0-1 scale)
			// Productivity = (Tasks/Hours) * Quality * Scaling Factor
			if point.Hours > 0 {
				tasksPerHour := float64(point.Tasks) / point.Hours
				// Normalize tasks per hour (assume 10 tasks/hour is excellent)
				normalizedTPH := math.Min(tasksPerHour/10.0, 1.0)

				// Final productivity score
				point.Productivity = normalizedTPH * point.Quality

				totalProductivity += point.Productivity
				nonZeroDays++
			}

			// Round values
			point.Productivity = math.Round(point.Productivity*100) / 100
			point.Hours = math.Round(point.Hours*10) / 10
			point.Quality = math.Round(point.Quality*100) / 100
		}

		dataPoints = append(dataPoints, point)
		trendLine = append(trendLine, point.Productivity)
	}

	// Calculate average productivity
	avgProductivity := 0.0
	if nonZeroDays > 0 {
		avgProductivity = totalProductivity / float64(nonZeroDays)
	}

	// Generate simple forecast based on trend
	forecast := s.generateProductivityForecast(trendLine, 7)

	// Calculate growth rate
	growthRate := 0.0
	if len(trendLine) >= 7 {
		// Compare last week average to first week average
		firstWeekAvg := 0.0
		lastWeekAvg := 0.0
		weekLen := min(7, len(trendLine)/4)

		for i := 0; i < weekLen; i++ {
			firstWeekAvg += trendLine[i]
			lastWeekAvg += trendLine[len(trendLine)-weekLen+i]
		}

		if firstWeekAvg > 0 {
			growthRate = (lastWeekAvg - firstWeekAvg) / firstWeekAvg
		}
	}

	// Find improvement areas based on low productivity days
	improvementAreas := s.identifyImprovementAreas(dataPoints)

	// Calculate consistency (standard deviation of productivity)
	consistency := s.calculateConsistency(trendLine)

	// Find peak performance
	peakPerformance := 0.0
	for _, point := range dataPoints {
		if point.Productivity > peakPerformance {
			peakPerformance = point.Productivity
		}
	}

	trend := &ProductivityTrend{
		Period:     period,
		DataPoints: dataPoints,
		TrendLine:  trendLine,
		Forecast:   forecast,
		Statistics: map[string]interface{}{
			"average_productivity": math.Round(avgProductivity*100) / 100,
			"growth_rate":          math.Round(growthRate*100) / 100,
			"consistency":          math.Round(consistency*100) / 100,
			"peak_performance":     math.Round(peakPerformance*100) / 100,
			"improvement_areas":    improvementAreas,
		},
	}

	// Generate insights based on actual data
	insights := s.generateProductivityInsights(trend, avgProductivity, growthRate)

	return trend, insights, nil
}

// GetDashboardData returns comprehensive dashboard data
func (s *AnalyticsService) GetDashboardData(ctx context.Context) (map[string]interface{}, error) {
	// Get current user ID from context
	userID := ctx.Value("user_id")
	if userID == nil {
		userID = "a0000000-0000-4000-8000-000000000001" // Default user ID
		s.logger.Warn("No user ID in context, using default", slog.String("default_id", userID.(string)))
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		s.logger.Error("Invalid user ID", slog.Any("error", err))
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get current stats from various sources
	currentStats, err := s.getCurrentStats(ctx, userUUID)
	if err != nil {
		s.logger.Error("Failed to get current stats", slog.Any("error", err))
	}

	// Get recent activities
	recentActivities, err := s.getRecentActivities(ctx, userUUID)
	if err != nil {
		s.logger.Error("Failed to get recent activities", slog.Any("error", err))
	}

	// Get performance metrics
	performanceMetrics, err := s.getPerformanceMetrics(ctx, userUUID)
	if err != nil {
		s.logger.Error("Failed to get performance metrics", slog.Any("error", err))
	}

	// Generate recommendations based on data
	recommendations := s.generateDashboardRecommendations(ctx, userUUID, currentStats, performanceMetrics)
	dashboard := map[string]interface{}{
		"current_stats":       currentStats,
		"recent_activities":   recentActivities,
		"performance_metrics": performanceMetrics,
		"recommendations":     recommendations,
		"last_updated":        time.Now().Format(time.RFC3339),
	}

	return dashboard, nil
}

// generateActivityInsights generates insights based on activity metrics
func (s *AnalyticsService) generateActivityInsights(metrics *ActivityMetrics, weeklyPattern map[string]float64, peakHours []int) []string {
	insights := []string{}

	// Find most productive day
	maxDay := ""
	maxValue := 0.0
	for day, value := range weeklyPattern {
		if value > maxValue {
			maxValue = value
			maxDay = day
		}
	}
	if maxDay != "" {
		dayMap := map[string]string{
			"monday": "週一", "tuesday": "週二", "wednesday": "週三",
			"thursday": "週四", "friday": "週五", "saturday": "週六", "sunday": "週日",
		}
		insights = append(insights, fmt.Sprintf("您在%s的生產力最高", dayMap[maxDay]))
	}

	// Peak hours insight
	if len(peakHours) > 0 {
		if len(peakHours) >= 3 {
			insights = append(insights, fmt.Sprintf("下午 %d-%d 點是您的黃金工作時段", peakHours[0], peakHours[len(peakHours)-1]))
		} else {
			insights = append(insights, fmt.Sprintf("%d 點是您的活動高峰期", peakHours[0]))
		}
	}

	// Work-life balance insight
	weekendActivity := (weeklyPattern["saturday"] + weeklyPattern["sunday"]) / 2
	if weekendActivity < 0.5 {
		insights = append(insights, "週末活動較少，保持了良好的工作生活平衡")
	}

	// Main activity type insight
	maxType := ""
	maxCount := 0
	for actType, count := range metrics.ActivityByType {
		if count > maxCount {
			maxCount = count
			maxType = actType
		}
	}
	if maxType != "" {
		activityMap := map[string]string{
			"code_completion": "編碼", "refactoring": "重構", "debugging": "調試",
			"query_response": "查詢", "tool_usage": "工具使用", "error_recovery": "錯誤修復",
		}
		if displayName, exists := activityMap[maxType]; exists {
			insights = append(insights, fmt.Sprintf("%s活動佔比最高，符合您的工作模式", displayName))
		}
	}

	return insights
}

// getFallbackActivityData gets activity data from tool executions when learning events are not available
func (s *AnalyticsService) getFallbackActivityData(ctx context.Context, userID string, startDate, endDate time.Time) []*sqlc.LearningEvent {
	// Query tool execution trends as a fallback
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		s.logger.Error("Invalid user ID for fallback", slog.Any("error", err))
		return []*sqlc.LearningEvent{}
	}

	trends, err := s.db.GetToolExecutionTrends(ctx, sqlc.GetToolExecutionTrendsParams{
		UserID:      pgtype.UUID{Bytes: userUUID, Valid: true},
		StartedAt:   pgtype.Timestamptz{Time: startDate, Valid: true},
		StartedAt_2: pgtype.Timestamptz{Time: endDate, Valid: true},
	})
	if err != nil {
		s.logger.Error("Failed to get tool execution trends", slog.Any("error", err))
		return []*sqlc.LearningEvent{}
	}

	// Convert tool execution trends to learning events format
	events := []*sqlc.LearningEvent{}
	for _, trend := range trends {
		// Create synthetic events based on execution trends
		for i := int32(0); i < trend.Executions; i++ {
			eventType := "tool_usage"
			outcome := "success"
			if i < trend.Failures {
				outcome = "failure"
			}

			// Distribute events throughout the day
			var eventTime time.Time
			if trend.ExecutionDate.Valid {
				eventTime = trend.ExecutionDate.Time.Add(time.Duration(i) * time.Hour)
			} else {
				eventTime = time.Now()
			}

			eventID := uuid.New()

			events = append(events, &sqlc.LearningEvent{
				ID:        pgtype.UUID{Bytes: eventID, Valid: true},
				UserID:    pgtype.UUID{Bytes: userUUID, Valid: true},
				EventType: eventType,
				CreatedAt: eventTime,
				DurationMs: pgtype.Int4{
					Int32: 1000, // Default 1 second
					Valid: true,
				},
				Outcome: pgtype.Text{
					String: outcome,
					Valid:  true,
				},
				Context:   []byte("{}"),
				InputData: []byte("{}"),
			})
		}
	}

	return events
}

// analyzeHeatmapPatterns analyzes activity patterns from heatmap data
func (s *AnalyticsService) analyzeHeatmapPatterns(activityMap map[string]float64, weekLabels []string) []string {
	patterns := []string{}

	// Find peak hours across all days
	hourTotals := make(map[int]float64)
	for key, value := range activityMap {
		parts := strings.Split(key, "-")
		if len(parts) == 2 {
			hour, _ := strconv.Atoi(parts[1])
			hourTotals[hour] += value
		}
	}

	// Find top 3 peak hours
	type hourActivity struct {
		hour     int
		activity float64
	}
	var hours []hourActivity
	for h, a := range hourTotals {
		hours = append(hours, hourActivity{h, a})
	}
	sort.Slice(hours, func(i, j int) bool {
		return hours[i].activity > hours[j].activity
	})

	if len(hours) >= 3 {
		patterns = append(patterns, fmt.Sprintf("%d-%d 點是活動高峰期", hours[0].hour, hours[2].hour))
	}

	// Find most active day
	dayTotals := make(map[int]float64)
	for key, value := range activityMap {
		parts := strings.Split(key, "-")
		if len(parts) == 2 {
			day, _ := strconv.Atoi(parts[0])
			dayTotals[day] += value
		}
	}

	maxDay := -1
	maxDayActivity := 0.0
	for day, activity := range dayTotals {
		if activity > maxDayActivity {
			maxDayActivity = activity
			maxDay = day
		}
	}

	if maxDay >= 0 && maxDay < len(weekLabels) {
		patterns = append(patterns, fmt.Sprintf("%s整體活動量最高", weekLabels[maxDay]))
	}

	// Check weekend activity
	weekendActivity := dayTotals[5] + dayTotals[6] // Saturday + Sunday
	weekdayActivity := dayTotals[0] + dayTotals[1] + dayTotals[2] + dayTotals[3] + dayTotals[4]
	if weekendActivity > 0 && weekdayActivity > 0 {
		weekendRatio := weekendActivity / (weekdayActivity / 5) / 2
		if weekendRatio < 0.5 {
			patterns = append(patterns, "週末保持適度的學習活動")
		} else if weekendRatio > 0.8 {
			patterns = append(patterns, "週末活動量與平日相當")
		}
	}

	// Check late night activity
	lateNightActivity := hourTotals[0] + hourTotals[1] + hourTotals[2] + hourTotals[3]
	totalActivity := 0.0
	for _, a := range hourTotals {
		totalActivity += a
	}

	if totalActivity > 0 && lateNightActivity/totalActivity < 0.05 {
		patterns = append(patterns, "深夜活動較少，作息健康")
	}

	return patterns
}

// generateHeatmapRecommendations generates recommendations based on heatmap analysis
func (s *AnalyticsService) generateHeatmapRecommendations(activityMap map[string]float64, consistency float64) []string {
	recommendations := []string{}

	// Find peak hours for recommendations
	hourTotals := make(map[int]float64)
	for key, value := range activityMap {
		parts := strings.Split(key, "-")
		if len(parts) == 2 {
			hour, _ := strconv.Atoi(parts[1])
			hourTotals[hour] += value
		}
	}

	// Find afternoon peak
	afternoonPeak := false
	for h := 14; h <= 16; h++ {
		if activity, exists := hourTotals[h]; exists && activity > 0 {
			afternoonPeak = true
			break
		}
	}

	if afternoonPeak {
		recommendations = append(recommendations, "保護下午高效時段，避免會議")
	}

	// Check morning activity
	morningActivity := hourTotals[6] + hourTotals[7] + hourTotals[8] + hourTotals[9]
	afternoonActivity := hourTotals[14] + hourTotals[15] + hourTotals[16]

	if morningActivity < afternoonActivity*0.5 {
		recommendations = append(recommendations, "考慮在早上增加一些深度工作")
	}

	// Weekend recommendations
	weekendTotal := 0.0
	for day := 5; day <= 6; day++ {
		for hour := 0; hour < 24; hour++ {
			key := fmt.Sprintf("%d-%d", day, hour)
			if activity, exists := activityMap[key]; exists {
				weekendTotal += activity
			}
		}
	}

	if weekendTotal > 0 {
		recommendations = append(recommendations, "週末的學習習慣值得保持")
	}

	// Consistency recommendations
	if consistency < 0.5 {
		recommendations = append(recommendations, "建立更規律的工作節奏")
	} else if consistency > 0.8 {
		recommendations = append(recommendations, "保持當前穩定的工作模式")
	}

	return recommendations
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateProductivityForecast generates forecast points based on trend data
func (s *AnalyticsService) generateProductivityForecast(trendLine []float64, days int) []ForecastPoint {
	forecast := []ForecastPoint{}

	if len(trendLine) < 3 {
		// Not enough data for forecast
		return forecast
	}

	// Simple linear regression for trend
	n := float64(len(trendLine))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, y := range trendLine {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope and intercept
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// Generate forecast
	for i := 1; i <= days; i++ {
		date := time.Now().AddDate(0, 0, i)
		x := float64(len(trendLine) + i - 1)

		// Predicted value with some bounds
		predicted := slope*x + intercept
		predicted = math.Max(0, math.Min(1, predicted)) // Keep between 0 and 1

		// Confidence decreases with distance
		confidence := 0.95 - float64(i)*0.05
		confidence = math.Max(0.5, confidence)

		forecast = append(forecast, ForecastPoint{
			Date:         date.Format("2006-01-02"),
			Productivity: math.Round(predicted*100) / 100,
			Confidence:   math.Round(confidence*100) / 100,
		})
	}

	return forecast
}

// identifyImprovementAreas analyzes productivity data to find improvement opportunities
func (s *AnalyticsService) identifyImprovementAreas(dataPoints []ProductivityPoint) []string {
	areas := []string{}

	// Analyze morning vs afternoon productivity
	morningProductivity := 0.0
	afternoonProductivity := 0.0
	morningCount := 0
	afternoonCount := 0

	// Check for low productivity patterns
	lowProductivityDays := 0
	noActivityDays := 0

	for _, point := range dataPoints {
		if point.Tasks == 0 {
			noActivityDays++
		} else if point.Productivity < 0.5 {
			lowProductivityDays++
		}

		// This is simplified - in real implementation, you'd check actual hours
		if point.Hours > 0 && point.Hours < 6 {
			morningProductivity += point.Productivity
			morningCount++
		} else if point.Hours >= 6 {
			afternoonProductivity += point.Productivity
			afternoonCount++
		}
	}

	// Generate improvement suggestions
	if morningCount > 0 && afternoonCount > 0 {
		avgMorning := morningProductivity / float64(morningCount)
		avgAfternoon := afternoonProductivity / float64(afternoonCount)

		if avgMorning < avgAfternoon*0.7 {
			areas = append(areas, "morning_focus")
		}
	}

	if lowProductivityDays > len(dataPoints)/4 {
		areas = append(areas, "task_prioritization")
	}

	if noActivityDays > len(dataPoints)/3 {
		areas = append(areas, "consistency")
	}

	// Check for quality issues
	avgQuality := 0.0
	qualityCount := 0
	for _, point := range dataPoints {
		if point.Tasks > 0 {
			avgQuality += point.Quality
			qualityCount++
		}
	}

	if qualityCount > 0 && avgQuality/float64(qualityCount) < 0.7 {
		areas = append(areas, "quality_improvement")
	}

	return areas
}

// calculateConsistency calculates the consistency score from productivity data
func (s *AnalyticsService) calculateConsistency(trendLine []float64) float64 {
	if len(trendLine) < 2 {
		return 0
	}

	// Calculate mean
	sum := 0.0
	count := 0
	for _, v := range trendLine {
		if v > 0 {
			sum += v
			count++
		}
	}

	if count == 0 {
		return 0
	}

	mean := sum / float64(count)

	// Calculate standard deviation
	variance := 0.0
	for _, v := range trendLine {
		if v > 0 {
			variance += math.Pow(v-mean, 2)
		}
	}

	stdDev := math.Sqrt(variance / float64(count))

	// Consistency is inverse of coefficient of variation
	if mean > 0 {
		cv := stdDev / mean
		consistency := 1.0 - cv
		return math.Max(0, math.Min(1, consistency))
	}

	return 0
}

// generateProductivityInsights generates insights from productivity trends
func (s *AnalyticsService) generateProductivityInsights(trend *ProductivityTrend, avgProductivity, growthRate float64) []string {
	insights := []string{}

	// Growth trend insight
	if growthRate > 0.1 {
		insights = append(insights, "生產力呈現穩定上升趨勢")
	} else if growthRate > 0 {
		insights = append(insights, "生產力有輕微提升")
	} else if growthRate < -0.1 {
		insights = append(insights, "生產力有下降趨勢，需要關注")
	}

	// Quality insight
	totalQuality := 0.0
	qualityCount := 0
	for _, point := range trend.DataPoints {
		if point.Tasks > 0 {
			totalQuality += point.Quality
			qualityCount++
		}
	}

	if qualityCount > 0 {
		avgQuality := totalQuality / float64(qualityCount)
		if avgQuality > 0.8 {
			insights = append(insights, "品質保持在高水準")
		} else if avgQuality > 0.6 {
			insights = append(insights, "品質表現穩定")
		}
	}

	// Consistency insight
	if consistency, ok := trend.Statistics["consistency"].(float64); ok {
		if consistency > 0.7 {
			insights = append(insights, "工作節奏保持穩定")
		}
	}

	// Forecast insight
	if len(trend.Forecast) > 0 {
		lastForecast := trend.Forecast[len(trend.Forecast)-1]
		if lastForecast.Productivity > avgProductivity {
			insights = append(insights, "預測未來生產力將持續提升")
		}
	}

	return insights
}
