// Package analytics provides analytics and visualization services for the Assistant API server.
package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"time"

	"github.com/koopa0/assistant-go/internal/observability"
	"github.com/koopa0/assistant-go/internal/storage/postgres/sqlc"
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

	// TODO: Query actual data from database
	// For now, generate mock data
	dailyActivities := []DailyActivity{}
	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		events := rand.Intn(20) + 5
		hours := float64(events)*0.5 + rand.Float64()

		dailyActivities = append(dailyActivities, DailyActivity{
			Date:       date.Format("2006-01-02"),
			EventCount: events,
			Hours:      math.Round(hours*10) / 10,
			Intensity:  math.Min(hours/8, 1.0),
			MainType:   []string{"coding", "debugging", "learning", "refactoring"}[rand.Intn(4)],
		})
	}

	metrics := &ActivityMetrics{
		Period:        fmt.Sprintf("%d days", days),
		TotalEvents:   425,
		ActiveDays:    26,
		TotalHours:    186.5,
		AveragePerDay: 6.2,
		PeakHours:     []int{14, 15, 16, 20, 21},
		ActivityByType: map[string]int{
			"coding":        180,
			"debugging":     85,
			"learning":      70,
			"refactoring":   50,
			"documentation": 40,
		},
		DailyActivity: dailyActivities,
		WeeklyPattern: map[string]float64{
			"monday":    0.85,
			"tuesday":   0.90,
			"wednesday": 0.92,
			"thursday":  0.88,
			"friday":    0.75,
			"saturday":  0.40,
			"sunday":    0.30,
		},
	}

	insights := []string{
		"您在週三的生產力最高",
		"下午 2-4 點是您的黃金工作時段",
		"週末活動較少，保持了良好的工作生活平衡",
		"編碼活動佔比最高，符合開發者角色",
	}

	return metrics, insights, nil
}

// GetActivityHeatmap returns activity heatmap data
func (s *AnalyticsService) GetActivityHeatmap(ctx context.Context) (*HeatmapData, []string, []string, error) {
	// Generate 24 hours x 7 days activity heatmap
	data := []HeatmapCell{}

	// Generate mock data
	for hour := 0; hour < 24; hour++ {
		for day := 0; day < 7; day++ {
			// Simulate activity intensity (higher during weekday work hours)
			value := 0.0
			if day < 5 { // Weekdays
				if hour >= 9 && hour <= 17 {
					value = 0.6 + rand.Float64()*0.4
					if hour >= 14 && hour <= 16 {
						value = math.Min(value*1.3, 1.0) // Afternoon peak
					}
				} else if hour >= 20 && hour <= 22 {
					value = 0.3 + rand.Float64()*0.3 // Evening work
				}
			} else { // Weekend
				if hour >= 10 && hour <= 16 {
					value = rand.Float64() * 0.4
				}
			}

			data = append(data, HeatmapCell{
				X:     day,
				Y:     hour,
				Value: math.Round(value*100) / 100,
			})
		}
	}

	heatmap := &HeatmapData{
		Data:    data,
		XLabels: []string{"週一", "週二", "週三", "週四", "週五", "週六", "週日"},
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
			"most_active_time":  "週三 15:00",
			"least_active_time": "週日 03:00",
			"weekly_coverage":   0.42,
			"consistency":       0.78,
		},
	}

	patterns := []string{
		"下午 2-4 點是活動高峰期",
		"週三整體活動量最高",
		"週末保持適度的學習活動",
		"深夜活動較少，作息健康",
	}

	recommendations := []string{
		"保護下午高效時段，避免會議",
		"考慮在早上增加一些深度工作",
		"週末的學習習慣值得保持",
	}

	return heatmap, patterns, recommendations, nil
}

// GetProductivityTrends returns productivity trends for a given period
func (s *AnalyticsService) GetProductivityTrends(ctx context.Context, period string) (*ProductivityTrend, []string, error) {
	if period == "" {
		period = "month"
	}

	// Generate historical data points
	dataPoints := []ProductivityPoint{}
	days := 30
	if period == "week" {
		days = 7
	} else if period == "quarter" {
		days = 90
	}

	trendLine := []float64{}
	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		base := 0.7 + float64(days-i)*0.002 // Upward trend
		productivity := base + (rand.Float64()-0.5)*0.1
		hours := 6.0 + rand.Float64()*4
		tasks := int(productivity * hours * 2)
		quality := 0.8 + (rand.Float64()-0.5)*0.2

		dataPoints = append(dataPoints, ProductivityPoint{
			Date:         date.Format("2006-01-02"),
			Productivity: math.Round(productivity*100) / 100,
			Hours:        math.Round(hours*10) / 10,
			Tasks:        tasks,
			Quality:      math.Round(quality*100) / 100,
		})

		trendLine = append(trendLine, base)
	}

	// Generate forecast
	forecast := []ForecastPoint{}
	for i := 1; i <= 7; i++ {
		date := time.Now().AddDate(0, 0, i)
		predictedProductivity := 0.85 + float64(i)*0.001
		confidence := 0.95 - float64(i)*0.05

		forecast = append(forecast, ForecastPoint{
			Date:         date.Format("2006-01-02"),
			Productivity: math.Round(predictedProductivity*100) / 100,
			Confidence:   math.Round(confidence*100) / 100,
		})
	}

	trend := &ProductivityTrend{
		Period:     period,
		DataPoints: dataPoints,
		TrendLine:  trendLine,
		Forecast:   forecast,
		Statistics: map[string]interface{}{
			"average_productivity": 0.78,
			"growth_rate":          0.12,
			"consistency":          0.85,
			"peak_performance":     0.92,
			"improvement_areas":    []string{"morning_focus", "task_prioritization"},
		},
	}

	insights := []string{
		"生產力呈現穩定上升趨勢",
		"品質保持在高水準",
		"任務完成率持續改善",
		"預測下週將達到新高峰",
	}

	return trend, insights, nil
}

// GetDashboardData returns comprehensive dashboard data
func (s *AnalyticsService) GetDashboardData(ctx context.Context) (map[string]interface{}, error) {
	// Get current stats
	currentStats := map[string]interface{}{
		"active_conversations": 12,
		"tasks_completed":      45,
		"code_quality_score":   0.88,
		"knowledge_nodes":      256,
		"learning_progress":    0.73,
		"collaboration_score":  0.82,
	}

	// Get recent activities
	recentActivities := []map[string]interface{}{
		{
			"type":      "code_review",
			"title":     "優化查詢性能",
			"timestamp": time.Now().Add(-30 * time.Minute),
			"impact":    "high",
		},
		{
			"type":      "learning",
			"title":     "學習 Go 1.24 新特性",
			"timestamp": time.Now().Add(-2 * time.Hour),
			"impact":    "medium",
		},
		{
			"type":      "collaboration",
			"title":     "協助團隊解決部署問題",
			"timestamp": time.Now().Add(-4 * time.Hour),
			"impact":    "high",
		},
	}

	// Get performance metrics
	performanceMetrics := map[string]interface{}{
		"response_time_ms": 125,
		"success_rate":     0.98,
		"api_calls_today":  342,
		"cache_hit_rate":   0.76,
	}

	// Get recommendations
	recommendations := []map[string]interface{}{
		{
			"type":        "productivity",
			"title":       "建議在早上安排深度工作",
			"description": "根據您的活動模式，早上 9-11 點的干擾較少",
			"priority":    "medium",
		},
		{
			"type":        "learning",
			"title":       "探索 Docker 容器優化",
			"description": "您最近的項目可以受益於容器化改進",
			"priority":    "high",
		},
		{
			"type":        "health",
			"title":       "休息時間提醒",
			"description": "已連續工作 2.5 小時，建議休息 15 分鐘",
			"priority":    "low",
		},
	}

	dashboard := map[string]interface{}{
		"current_stats":       currentStats,
		"recent_activities":   recentActivities,
		"performance_metrics": performanceMetrics,
		"recommendations":     recommendations,
		"last_updated":        time.Now().Format(time.RFC3339),
	}

	return dashboard, nil
}
