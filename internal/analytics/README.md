# Analytics Package

The analytics package provides comprehensive analytics and visualization services for the Assistant application.

## Structure

- `service.go` - Core analytics service implementation
- `http.go` - HTTP handlers for analytics endpoints
- `analytics_helpers.go` - Helper functions for analytics calculations
- `insights.go` - Insights generation and analysis
- `timeline.go` - Timeline event tracking and analysis

## Features

### Activity Analytics
- Daily activity tracking
- Activity heatmaps
- Weekly patterns analysis
- Peak hours identification

### Productivity Analytics
- Productivity trends and forecasting
- Task completion metrics
- Quality scoring
- Growth rate analysis

### Timeline Management
- Event tracking with tags
- Pattern recognition
- Historical analysis

### Insights Generation
- Development pattern analysis
- Code quality insights
- Personalized recommendations
- Learning effectiveness metrics

## Usage

```go
// Create analytics service
service := analytics.NewAnalyticsService(db, logger, metrics)

// Create HTTP handler
handler := analytics.NewHTTPHandler(service)

// Register routes
handler.RegisterRoutes(mux)
```

## API Endpoints

### Activity Analytics
- `GET /api/v1/analytics/activity` - Get activity metrics
- `GET /api/v1/analytics/activity/heatmap` - Get activity heatmap
- `GET /api/v1/analytics/productivity/trends` - Get productivity trends

### Insights
- `GET /api/v1/insights/development-patterns` - Development patterns
- `GET /api/v1/insights/recommendations` - Personalized recommendations
- `GET /api/v1/insights/summary` - Overall insights summary

### Timeline
- `GET /api/v1/timeline` - Get timeline events
- `POST /api/v1/timeline/events` - Create timeline event
- `GET /api/v1/timeline/patterns` - Analyze timeline patterns