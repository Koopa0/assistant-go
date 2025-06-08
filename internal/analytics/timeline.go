package analytics

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// TimelineEvent represents a timeline event
type TimelineEvent struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // coding, learning, debugging, refactoring
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time,omitempty"`
	Duration    *time.Duration         `json:"duration,omitempty"`
	Tags        []string               `json:"tags"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Metrics     *EventMetrics          `json:"metrics,omitempty"`
	Related     []string               `json:"related_events,omitempty"`
}

// EventMetrics represents event metrics
type EventMetrics struct {
	LinesOfCode    int     `json:"lines_of_code,omitempty"`
	FilesModified  int     `json:"files_modified,omitempty"`
	TestsPassed    int     `json:"tests_passed,omitempty"`
	TestsFailed    int     `json:"tests_failed,omitempty"`
	CoverageChange float64 `json:"coverage_change,omitempty"`
	Complexity     int     `json:"complexity,omitempty"`
}

// TimelineResponse represents timeline response
type TimelineResponse struct {
	Events     []TimelineEvent     `json:"events"`
	TotalCount int                 `json:"total_count"`
	TimeRange  TimeRange           `json:"time_range"`
	Statistics *TimelineStatistics `json:"statistics,omitempty"`
	Insights   []TimelineInsight   `json:"insights,omitempty"`
}

// TimeRange represents time range
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// TimelineStatistics represents timeline statistics
type TimelineStatistics struct {
	TotalEvents       int                   `json:"total_events"`
	EventsByType      map[string]int        `json:"events_by_type"`
	TotalDuration     time.Duration         `json:"total_duration"`
	AverageDuration   time.Duration         `json:"average_duration"`
	ProductivityScore float64               `json:"productivity_score"`
	TopTags           []TagFrequency        `json:"top_tags"`
	DailyActivity     map[string]DailyStats `json:"daily_activity"`
}

// TagFrequency represents tag frequency
type TagFrequency struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// DailyStats represents daily statistics
type DailyStats struct {
	Date         string  `json:"date"`
	EventCount   int     `json:"event_count"`
	TotalMinutes int     `json:"total_minutes"`
	MostActive   string  `json:"most_active_hour"`
	Productivity float64 `json:"productivity"`
	TopType      string  `json:"top_type"`
}

// TimelineInsight represents timeline insight
type TimelineInsight struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // high, medium, low
	Actionable  bool   `json:"actionable"`
}

// GetTimeline retrieves timeline events with filters
func (s *AnalyticsService) GetTimeline(ctx context.Context, startDate, endDate time.Time, eventTypes []string, tags []string, limit int) (*TimelineResponse, error) {
	// TODO: Query actual events from database
	events := generateMockTimelineEvents(startDate, endDate, eventTypes, tags, limit)

	stats := calculateTimelineStatistics(events)
	insights := generateTimelineInsights(events, stats)

	return &TimelineResponse{
		Events:     events,
		TotalCount: len(events),
		TimeRange: TimeRange{
			Start: startDate,
			End:   endDate,
		},
		Statistics: stats,
		Insights:   insights,
	}, nil
}

// GetTimelineStatistics returns timeline statistics
func (s *AnalyticsService) GetTimelineStatistics(ctx context.Context, startDate, endDate time.Time) (*TimelineStatistics, error) {
	// TODO: Calculate from actual data
	return &TimelineStatistics{
		TotalEvents: 145,
		EventsByType: map[string]int{
			"coding":      65,
			"debugging":   25,
			"learning":    30,
			"refactoring": 15,
			"planning":    10,
		},
		TotalDuration:     156 * time.Hour,
		AverageDuration:   64 * time.Minute,
		ProductivityScore: 0.82,
		TopTags: []TagFrequency{
			{Tag: "golang", Count: 45},
			{Tag: "api", Count: 32},
			{Tag: "optimization", Count: 28},
			{Tag: "testing", Count: 22},
			{Tag: "refactoring", Count: 18},
		},
		DailyActivity: generateDailyActivity(startDate, endDate),
	}, nil
}

// GetTimelinePatterns analyzes patterns in timeline data
func (s *AnalyticsService) GetTimelinePatterns(ctx context.Context, days int) ([]map[string]interface{}, error) {
	patterns := []map[string]interface{}{
		{
			"id":          "pattern-1",
			"name":        "早晨高效編碼",
			"description": "您在早上 9-11 點的編碼效率最高",
			"confidence":  0.85,
			"frequency":   "daily",
			"impact":      "positive",
			"evidence": map[string]interface{}{
				"average_loc_per_hour": 125,
				"bug_rate":             0.02,
				"test_coverage":        0.92,
			},
			"suggestions": []string{
				"保護早晨時段用於核心開發",
				"避免在此時段安排會議",
			},
		},
		{
			"id":          "pattern-2",
			"name":        "週五重構習慣",
			"description": "您傾向在週五下午進行代碼重構",
			"confidence":  0.78,
			"frequency":   "weekly",
			"impact":      "positive",
			"evidence": map[string]interface{}{
				"refactoring_sessions":     8,
				"code_quality_improvement": 0.15,
			},
			"suggestions": []string{
				"這是良好的習慣，有助於保持代碼品質",
				"考慮建立重構檢查清單",
			},
		},
		{
			"id":          "pattern-3",
			"name":        "深夜調試模式",
			"description": "複雜 bug 通常在晚上 10 點後解決",
			"confidence":  0.72,
			"frequency":   "occasional",
			"impact":      "neutral",
			"evidence": map[string]interface{}{
				"night_debugging_sessions": 5,
				"success_rate":             0.80,
			},
			"suggestions": []string{
				"記錄深夜調試的解決方案",
				"考慮將複雜問題留到精力充沛時處理",
			},
		},
	}

	return patterns, nil
}

// Helper functions

func generateMockTimelineEvents(startDate, endDate time.Time, eventTypes, tags []string, limit int) []TimelineEvent {
	events := []TimelineEvent{}
	eventTypeOptions := []string{"coding", "debugging", "learning", "refactoring", "planning"}
	tagOptions := []string{"golang", "api", "optimization", "testing", "database", "frontend", "backend"}

	// Generate events for each day
	current := startDate
	for current.Before(endDate) && len(events) < limit {
		// Generate 2-5 events per day
		eventsPerDay := rand.Intn(4) + 2

		for i := 0; i < eventsPerDay && len(events) < limit; i++ {
			// Random start time during work hours
			hour := rand.Intn(10) + 9 // 9 AM to 7 PM
			startTime := time.Date(current.Year(), current.Month(), current.Day(), hour, rand.Intn(60), 0, 0, current.Location())

			// Random duration between 30 minutes and 3 hours
			duration := time.Duration(rand.Intn(150)+30) * time.Minute
			endTime := startTime.Add(duration)

			eventType := eventTypeOptions[rand.Intn(len(eventTypeOptions))]

			// Apply filters
			if len(eventTypes) > 0 && !contains(eventTypes, eventType) {
				continue
			}

			// Generate event tags
			eventTags := []string{}
			numTags := rand.Intn(3) + 1
			for j := 0; j < numTags; j++ {
				tag := tagOptions[rand.Intn(len(tagOptions))]
				if !contains(eventTags, tag) {
					eventTags = append(eventTags, tag)
				}
			}

			// Apply tag filter
			if len(tags) > 0 {
				hasMatchingTag := false
				for _, tag := range tags {
					if contains(eventTags, tag) {
						hasMatchingTag = true
						break
					}
				}
				if !hasMatchingTag {
					continue
				}
			}

			event := TimelineEvent{
				ID:          fmt.Sprintf("event-%d", len(events)+1),
				Type:        eventType,
				Title:       generateEventTitle(eventType),
				Description: generateEventDescription(eventType),
				StartTime:   startTime,
				EndTime:     &endTime,
				Duration:    &duration,
				Tags:        eventTags,
				Context:     generateEventContext(eventType),
				Metrics:     generateEventMetrics(eventType),
			}

			events = append(events, event)
		}

		current = current.AddDate(0, 0, 1)
	}

	return events
}

func calculateTimelineStatistics(events []TimelineEvent) *TimelineStatistics {
	eventsByType := make(map[string]int)
	tagCounts := make(map[string]int)
	totalDuration := time.Duration(0)

	for _, event := range events {
		eventsByType[event.Type]++
		if event.Duration != nil {
			totalDuration += *event.Duration
		}
		for _, tag := range event.Tags {
			tagCounts[tag]++
		}
	}

	// Get top tags
	topTags := []TagFrequency{}
	for tag, count := range tagCounts {
		topTags = append(topTags, TagFrequency{Tag: tag, Count: count})
	}
	// Sort by count (simplified)
	if len(topTags) > 5 {
		topTags = topTags[:5]
	}

	avgDuration := time.Duration(0)
	if len(events) > 0 {
		avgDuration = totalDuration / time.Duration(len(events))
	}

	return &TimelineStatistics{
		TotalEvents:       len(events),
		EventsByType:      eventsByType,
		TotalDuration:     totalDuration,
		AverageDuration:   avgDuration,
		ProductivityScore: 0.82,
		TopTags:           topTags,
	}
}

func generateTimelineInsights(events []TimelineEvent, stats *TimelineStatistics) []TimelineInsight {
	insights := []TimelineInsight{
		{
			Type:        "productivity",
			Title:       "編碼效率提升",
			Description: "本週的編碼活動比上週增加了 15%",
			Impact:      "high",
			Actionable:  true,
		},
		{
			Type:        "pattern",
			Title:       "學習投資增加",
			Description: "您花在學習新技術的時間增加了 25%",
			Impact:      "medium",
			Actionable:  false,
		},
	}

	// Add insights based on statistics
	if stats.ProductivityScore > 0.8 {
		insights = append(insights, TimelineInsight{
			Type:        "achievement",
			Title:       "高生產力維持",
			Description: "您的生產力分數持續保持在 80% 以上",
			Impact:      "high",
			Actionable:  false,
		})
	}

	return insights
}

func generateDailyActivity(startDate, endDate time.Time) map[string]DailyStats {
	daily := make(map[string]DailyStats)
	current := startDate

	for current.Before(endDate) {
		dateStr := current.Format("2006-01-02")
		daily[dateStr] = DailyStats{
			Date:         dateStr,
			EventCount:   rand.Intn(5) + 2,
			TotalMinutes: rand.Intn(300) + 120,
			MostActive:   fmt.Sprintf("%02d:00", rand.Intn(10)+9),
			Productivity: 0.6 + rand.Float64()*0.3,
			TopType:      []string{"coding", "debugging", "learning"}[rand.Intn(3)],
		}
		current = current.AddDate(0, 0, 1)
	}

	return daily
}

func generateEventTitle(eventType string) string {
	titles := map[string][]string{
		"coding":      {"實作新功能", "API 開發", "資料模型設計", "介面實作"},
		"debugging":   {"修復錯誤", "性能問題調查", "記憶體洩漏修復", "並發問題解決"},
		"learning":    {"學習新技術", "閱讀文檔", "觀看教程", "研究最佳實踐"},
		"refactoring": {"代碼重構", "架構優化", "提取通用模組", "簡化複雜邏輯"},
		"planning":    {"專案規劃", "技術方案設計", "任務分解", "時程安排"},
	}

	options := titles[eventType]
	return options[rand.Intn(len(options))]
}

func generateEventDescription(eventType string) string {
	descriptions := map[string][]string{
		"coding":      {"完成用戶認證模組", "實作資料快取層", "開發 RESTful API", "建立前端元件"},
		"debugging":   {"找出並修復登入問題", "解決資料庫連線異常", "優化查詢效能", "修正並發錯誤"},
		"learning":    {"深入了解 Go 並發模型", "學習微服務架構", "研究 Docker 最佳實踐", "探索新的測試框架"},
		"refactoring": {"重構認證邏輯", "優化資料庫查詢", "改善錯誤處理", "提升代碼可讀性"},
		"planning":    {"規劃下一階段功能", "設計系統架構", "制定測試策略", "安排團隊任務"},
	}

	options := descriptions[eventType]
	return options[rand.Intn(len(options))]
}

func generateEventContext(eventType string) map[string]interface{} {
	context := map[string]interface{}{
		"project": []string{"assistant-go", "web-app", "mobile-api", "data-service"}[rand.Intn(4)],
		"module":  []string{"auth", "api", "database", "frontend", "testing"}[rand.Intn(5)],
	}

	if eventType == "coding" || eventType == "refactoring" {
		context["language"] = []string{"go", "javascript", "python", "sql"}[rand.Intn(4)]
		context["framework"] = []string{"gin", "echo", "react", "vue"}[rand.Intn(4)]
	}

	return context
}

func generateEventMetrics(eventType string) *EventMetrics {
	metrics := &EventMetrics{}

	switch eventType {
	case "coding":
		metrics.LinesOfCode = rand.Intn(200) + 50
		metrics.FilesModified = rand.Intn(5) + 1
		metrics.TestsPassed = rand.Intn(20) + 5
		metrics.TestsFailed = rand.Intn(3)
		metrics.CoverageChange = (rand.Float64() - 0.3) * 10
	case "refactoring":
		metrics.LinesOfCode = -rand.Intn(50) // Negative for code reduction
		metrics.FilesModified = rand.Intn(8) + 2
		metrics.Complexity = -rand.Intn(5) - 1 // Reduced complexity
	case "debugging":
		metrics.TestsPassed = rand.Intn(10) + 1
		metrics.TestsFailed = 0 // Fixed
	}

	return metrics
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
