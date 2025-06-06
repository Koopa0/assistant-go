// Package system provides system monitoring and management services for the Assistant API server.
package system

import (
	"context"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// SystemService handles system monitoring and management logic
type SystemService struct {
	assistant *assistant.Assistant
	queries   *sqlc.Queries
	logger    *slog.Logger
	metrics   *observability.Metrics
	startTime time.Time

	// Statistics tracking
	activeUsers     int64
	requestsPerMin  int64
	totalRequests   int64
	totalErrors     int64
	lastMinuteReset time.Time
}

// NewSystemService creates a new system service
func NewSystemService(assistant *assistant.Assistant, queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *SystemService {
	return &SystemService{
		assistant:       assistant,
		queries:         queries,
		logger:          observability.ServerLogger(logger, "system_service"),
		metrics:         metrics,
		startTime:       time.Now(),
		lastMinuteReset: time.Now(),
	}
}

// SystemStatus represents the system health and status
type SystemStatus struct {
	Health            string                 `json:"health"`
	Uptime            int64                  `json:"uptime"`
	ActiveUsers       int64                  `json:"activeUsers"`
	CPUUsage          float64                `json:"cpuUsage"`
	MemoryUsage       float64                `json:"memoryUsage"`
	ResponseTime      int64                  `json:"responseTime"`
	RequestsPerMinute int64                  `json:"requestsPerMinute"`
	ErrorRate         float64                `json:"errorRate"`
	Services          map[string]string      `json:"services"`
	Version           map[string]interface{} `json:"version"`
}

// SystemActivity represents a system activity event
type SystemActivity struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Timestamp   time.Time              `json:"timestamp"`
	UserID      string                 `json:"userId,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// GetStatus returns comprehensive system status
func (s *SystemService) GetStatus(ctx context.Context) (*SystemStatus, error) {
	// Track request
	atomic.AddInt64(&s.totalRequests, 1)
	s.updateRequestsPerMinute()

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Calculate metrics
	uptime := int64(time.Since(s.startTime).Seconds())
	cpuUsage := s.calculateCPUUsage()
	memoryUsage := float64(memStats.Alloc) / float64(memStats.Sys) * 100
	errorRate := s.calculateErrorRate()

	// Check services health
	services := s.checkServices(ctx)

	// Determine overall health
	health := "healthy"
	if services["database"] != "healthy" {
		health = "degraded"
	}
	if errorRate > 0.1 { // >10% error rate
		health = "degraded"
	}

	return &SystemStatus{
		Health:            health,
		Uptime:            uptime,
		ActiveUsers:       atomic.LoadInt64(&s.activeUsers),
		CPUUsage:          cpuUsage,
		MemoryUsage:       memoryUsage,
		ResponseTime:      25, // Mock value
		RequestsPerMinute: atomic.LoadInt64(&s.requestsPerMin),
		ErrorRate:         errorRate,
		Services:          services,
		Version: map[string]interface{}{
			"api":       "v1.0.0",
			"buildDate": "2024-01-01",
			"goVersion": runtime.Version(),
		},
	}, nil
}

// GetActivities returns system activities with filtering
func (s *SystemService) GetActivities(ctx context.Context, activityType, startDate, endDate string, page, limit int) ([]SystemActivity, int, error) {
	activities := s.generateSampleActivities(activityType, startDate, endDate, page, limit)
	total := 100 // Mock total
	return activities, total, nil
}

// GetMetrics returns detailed system metrics
func (s *SystemService) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metrics := map[string]interface{}{
		"memory": map[string]interface{}{
			"allocated":    memStats.Alloc,
			"totalAlloc":   memStats.TotalAlloc,
			"system":       memStats.Sys,
			"gcCount":      memStats.NumGC,
			"heapInUse":    memStats.HeapInuse,
			"heapReleased": memStats.HeapReleased,
		},
		"goroutines": runtime.NumGoroutine(),
		"cpu": map[string]interface{}{
			"numCPU":     runtime.NumCPU(),
			"goMaxProcs": runtime.GOMAXPROCS(0),
			"usage":      s.calculateCPUUsage(),
		},
		"requests": map[string]interface{}{
			"total":          atomic.LoadInt64(&s.totalRequests),
			"perMinute":      atomic.LoadInt64(&s.requestsPerMin),
			"errorRate":      s.calculateErrorRate(),
			"averageLatency": s.calculateAverageLatency(),
		},
		"database": s.getDatabaseMetrics(ctx),
	}

	return metrics, nil
}

// CheckHealth performs a health check
func (s *SystemService) CheckHealth(ctx context.Context) error {
	return s.assistant.Health(ctx)
}

// GetVersion returns version information
func (s *SystemService) GetVersion() map[string]interface{} {
	return map[string]interface{}{
		"version":    "1.0.0",
		"buildDate":  "2024-01-01",
		"gitCommit":  "abc123def456",
		"goVersion":  runtime.Version(),
		"apiVersion": "v1",
		"features": map[string]bool{
			"websocket":      true,
			"export":         true,
			"memory":         true,
			"authentication": true,
			"multiAgent":     true,
			"analytics":      true,
		},
		"supported": map[string]interface{}{
			"languages": []string{"zh-TW", "en-US"},
			"models":    []string{"claude-3-opus", "claude-3-sonnet", "gemini-pro"},
			"formats":   []string{"json", "csv", "pdf"},
		},
	}
}

// GetPerformance returns performance metrics
func (s *SystemService) GetPerformance(timeRange string) map[string]interface{} {
	if timeRange == "" {
		timeRange = "1h"
	}

	return map[string]interface{}{
		"responseTime": map[string]interface{}{
			"p50":     12,
			"p90":     45,
			"p95":     78,
			"p99":     156,
			"average": 25,
		},
		"throughput": map[string]interface{}{
			"requestsPerSecond": 150,
			"bytesPerSecond":    1024 * 1024 * 2, // 2MB/s
			"concurrent":        50,
		},
		"errors": map[string]interface{}{
			"rate":  0.02,
			"total": atomic.LoadInt64(&s.totalErrors),
			"types": map[string]int{
				"4xx":     45,
				"5xx":     12,
				"timeout": 8,
			},
		},
		"saturation": map[string]interface{}{
			"cpuUtilization":    s.calculateCPUUsage(),
			"memoryUtilization": s.calculateMemoryUsage(),
			"queueDepth":        15,
			"connectionPool":    0.65,
		},
		"timeRange": timeRange,
	}
}

// TrackError increments the error counter
func (s *SystemService) TrackError() {
	atomic.AddInt64(&s.totalErrors, 1)
}

// SetActiveUsers updates the active user count
func (s *SystemService) SetActiveUsers(count int64) {
	atomic.StoreInt64(&s.activeUsers, count)
}

// Helper methods

func (s *SystemService) checkServices(ctx context.Context) map[string]string {
	services := make(map[string]string)

	// Check database
	if err := s.assistant.Health(ctx); err != nil {
		services["database"] = "unhealthy"
	} else {
		services["database"] = "healthy"
	}

	// Check cache (mock)
	services["cache"] = "healthy"

	// Check queue (mock)
	services["queue"] = "degraded"

	// Check external services (mock)
	services["ai_provider"] = "healthy"
	services["storage"] = "healthy"

	return services
}

func (s *SystemService) calculateCPUUsage() float64 {
	// Mock CPU usage calculation
	// In real implementation, use system monitoring tools
	return 45.2
}

func (s *SystemService) calculateMemoryUsage() float64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return float64(memStats.Alloc) / float64(memStats.Sys) * 100
}

func (s *SystemService) calculateErrorRate() float64 {
	total := atomic.LoadInt64(&s.totalRequests)
	if total == 0 {
		return 0
	}
	errors := atomic.LoadInt64(&s.totalErrors)
	return float64(errors) / float64(total)
}

func (s *SystemService) calculateAverageLatency() float64 {
	// Mock average latency
	return 25.5
}

func (s *SystemService) updateRequestsPerMinute() {
	now := time.Now()
	if now.Sub(s.lastMinuteReset) > time.Minute {
		atomic.StoreInt64(&s.requestsPerMin, 1)
		s.lastMinuteReset = now
	} else {
		atomic.AddInt64(&s.requestsPerMin, 1)
	}
}

func (s *SystemService) getDatabaseMetrics(ctx context.Context) map[string]interface{} {
	stats, err := s.assistant.Stats(ctx)
	if err != nil {
		return map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	}

	if stats.Database != nil {
		return map[string]interface{}{
			"status":            stats.Database.Status,
			"totalConnections":  stats.Database.TotalConns,
			"activeConnections": stats.Database.AcquiredConns,
			"idleConnections":   stats.Database.IdleConns,
			"maxConnections":    stats.Database.MaxConns,
		}
	}

	return map[string]interface{}{
		"status": "unknown",
	}
}

func (s *SystemService) generateSampleActivities(activityType, startDate, endDate string, page, limit int) []SystemActivity {
	activities := []SystemActivity{
		{
			ID:          "activity_001",
			Type:        "query",
			Description: "使用者查詢: 如何優化 Go 程式碼效能",
			Timestamp:   time.Now().Add(-5 * time.Minute),
			UserID:      "user_123",
			Metadata: map[string]interface{}{
				"conversationId": "conv_456",
				"responseTime":   1250,
				"toolsUsed":      []string{"code_analyzer"},
			},
		},
		{
			ID:          "activity_002",
			Type:        "toolExecution",
			Description: "執行工具: 程式碼分析器",
			Timestamp:   time.Now().Add(-10 * time.Minute),
			UserID:      "user_456",
			Metadata: map[string]interface{}{
				"toolId":      "tool_001",
				"duration":    2345,
				"linesOfCode": 1500,
			},
		},
		{
			ID:          "activity_003",
			Type:        "error",
			Description: "API 速率限制觸發",
			Timestamp:   time.Now().Add(-30 * time.Minute),
			UserID:      "user_789",
			Metadata: map[string]interface{}{
				"errorCode": "RATE_LIMITED",
				"endpoint":  "/api/v1/tools/execute",
				"limit":     60,
			},
		},
	}

	// Filter by type if specified
	if activityType != "" {
		filtered := []SystemActivity{}
		for _, activity := range activities {
			if activity.Type == activityType {
				filtered = append(filtered, activity)
			}
		}
		activities = filtered
	}

	// Simple pagination
	start := (page - 1) * limit
	end := start + limit
	if start >= len(activities) {
		return []SystemActivity{}
	}
	if end > len(activities) {
		end = len(activities)
	}

	return activities[start:end]
}
