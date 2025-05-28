package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/web/i18n"
	"github.com/koopa0/assistant-go/internal/web/templates"
)

// Handlers contains all HTTP handlers for the web interface
type Handlers struct {
	assistant *assistant.Assistant
	logger    *slog.Logger
}

// New creates a new Handlers instance
func New(assistant *assistant.Assistant, logger *slog.Logger) *Handlers {
	return &Handlers{
		assistant: assistant,
		logger:    logger,
	}
}

// getLanguageFromRequest extracts the language preference from the request
func (h *Handlers) getLanguageFromRequest(r *http.Request) string {
	// Check cookie first
	if cookie, err := r.Cookie("lang"); err == nil {
		if _, exists := i18n.SupportedLanguages[cookie.Value]; exists {
			return cookie.Value
		}
	}

	// Check Accept-Language header
	acceptLang := r.Header.Get("Accept-Language")
	if len(acceptLang) >= 2 {
		lang := acceptLang[:2]
		if _, exists := i18n.SupportedLanguages[lang]; exists {
			return lang
		}
		// Check for zh-TW specifically
		if len(acceptLang) >= 5 && acceptLang[:5] == "zh-TW" {
			return "zh-TW"
		}
	}

	// Default to English
	return "en"
}

// getThemeFromRequest extracts the theme preference from the request
func (h *Handlers) getThemeFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie("theme"); err == nil {
		if cookie.Value == "dark" || cookie.Value == "light" {
			return cookie.Value
		}
	}
	return "light" // Default theme
}

// HandleDashboard renders the home page
func (h *Handlers) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	lang := h.getLanguageFromRequest(r)
	theme := h.getThemeFromRequest(r)

	// Get user name from session/cookie if available
	userName := "" // TODO: Get from session

	// Get recent chats from database
	recentChats := []templates.RecentChat{
		// TODO: Fetch from database
	}

	homeData := templates.HomePageData{
		AppLayoutData: templates.AppLayoutData{
			BaseLayoutData: templates.BaseLayoutData{
				Title:       i18n.T("nav.home", lang),
				Description: "GoAssistant - Your AI-powered development companion",
				Lang:        lang,
				Theme:       theme,
			},
			CurrentPage: "home",
		},
		UserName:    userName,
		RecentChats: recentChats,
	}

	component := templates.HomePage(homeData)

	if err := component.Render(r.Context(), w); err != nil {
		h.logger.Error("Failed to render home page", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// HandleChat renders the chat interface
func (h *Handlers) HandleChat(w http.ResponseWriter, r *http.Request) {
	lang := h.getLanguageFromRequest(r)
	theme := h.getThemeFromRequest(r)

	chatData := templates.ChatPageData{
		AppLayoutData: templates.AppLayoutData{
			BaseLayoutData: templates.BaseLayoutData{
				Title:       i18n.T("nav.chat", lang),
				Description: "Chat with AI assistants",
				Lang:        lang,
				Theme:       theme,
			},
			CurrentPage: "chat",
		},
		Messages: []templates.ChatMessage{
			// TODO: Load from database
		},
		RecentChats: []templates.ChatItem{
			// TODO: Load from database
		},
	}

	component := templates.ChatPage(chatData)

	if err := component.Render(r.Context(), w); err != nil {
		h.logger.Error("Failed to render chat", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// HandleAPI handles API endpoints for HTMX requests
func (h *Handlers) HandleAPI(w http.ResponseWriter, r *http.Request) {
	// Set HTMX headers
	w.Header().Set("Content-Type", "text/html")

	switch r.URL.Path {
	case "/api/activities":
		h.handleActivitiesAPI(w, r)
	case "/api/stats":
		h.handleStatsAPI(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleActivitiesAPI returns recent activities as HTML
func (h *Handlers) handleActivitiesAPI(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement activity items
	w.Write([]byte("<div>Activities coming soon</div>"))
}

// handleStatsAPI returns updated statistics
func (h *Handlers) handleStatsAPI(w http.ResponseWriter, r *http.Request) {
	// Return updated stats as JSON or HTML depending on request
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
}

// HandleTools renders the tools dashboard
func (h *Handlers) HandleTools(w http.ResponseWriter, r *http.Request) {
	lang := h.getLanguageFromRequest(r)
	theme := h.getThemeFromRequest(r)

	toolsData := templates.ToolsPageData{
		AppLayoutData: templates.AppLayoutData{
			BaseLayoutData: templates.BaseLayoutData{
				Title:       i18n.T("tools.overview", lang),
				Description: "AI development tools dashboard",
				Lang:        lang,
				Theme:       theme,
			},
			CurrentPage: "tools",
		},
		Tools: []templates.ToolCardData{
			{
				ID:          "go-dev",
				Name:        i18n.T("nav.development", lang),
				Description: "Go code analysis, profiling, and debugging tools",
				Icon:        "code",
				Status:      "active",
				StatusText:  i18n.T("tools.active", lang),
				LastUsed:    "2 minutes ago",
				UsageToday:  127,
				SuccessRate: 98.5,
				ConfigURL:   "/tools/go-dev/config",
			},
			{
				ID:          "postgres",
				Name:        i18n.T("nav.database", lang),
				Description: "PostgreSQL query optimization and schema management",
				Icon:        "storage",
				Status:      "active",
				StatusText:  i18n.T("tools.active", lang),
				LastUsed:    "15 minutes ago",
				UsageToday:  89,
				SuccessRate: 99.2,
				ConfigURL:   "/tools/postgres/config",
			},
			{
				ID:          "k8s",
				Name:        "Kubernetes",
				Description: "Cluster management and deployment automation",
				Icon:        "cloud",
				Status:      "idle",
				StatusText:  i18n.T("tools.idle", lang),
				LastUsed:    "1 hour ago",
				UsageToday:  23,
				SuccessRate: 97.8,
				ConfigURL:   "/tools/k8s/config",
			},
			{
				ID:          "docker",
				Name:        "Docker",
				Description: "Container lifecycle management and monitoring",
				Icon:        "deployed_code",
				Status:      "active",
				StatusText:  i18n.T("tools.active", lang),
				LastUsed:    "5 minutes ago",
				UsageToday:  56,
				SuccessRate: 99.8,
				ConfigURL:   "/tools/docker/config",
			},
			{
				ID:          "cloudflare",
				Name:        "Cloudflare",
				Description: "CDN, DNS, and security services management",
				Icon:        "security",
				Status:      "maintenance",
				StatusText:  i18n.T("tools.maintenance", lang),
				LastUsed:    "1 day ago",
				UsageToday:  0,
				SuccessRate: 95.3,
				ConfigURL:   "/tools/cloudflare/config",
			},
			{
				ID:          "search",
				Name:        "AI Search",
				Description: "Intelligent web search and research capabilities",
				Icon:        "search",
				Status:      "active",
				StatusText:  i18n.T("tools.active", lang),
				LastUsed:    "30 minutes ago",
				UsageToday:  42,
				SuccessRate: 96.7,
				ConfigURL:   "/tools/search/config",
			},
		},
	}

	component := templates.ToolsPage(toolsData)

	if err := component.Render(r.Context(), w); err != nil {
		h.logger.Error("Failed to render tools page", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// HandlePreferences handles user preference updates
func (h *Handlers) HandlePreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	switch r.URL.Path {
	case "/api/preferences/theme":
		h.handleThemePreference(w, r)
	case "/api/preferences/language":
		h.handleLanguagePreference(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleThemePreference updates theme preference
func (h *Handlers) handleThemePreference(w http.ResponseWriter, r *http.Request) {
	// Parse theme from request body
	// Set cookie and respond
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok"}`))
}

// handleLanguagePreference updates language preference
func (h *Handlers) handleLanguagePreference(w http.ResponseWriter, r *http.Request) {
	// Parse language from request body
	// Set cookie and respond
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "ok"}`))
}

// HandleDevelopment handles the development assistant page
func (h *Handlers) HandleDevelopment(w http.ResponseWriter, r *http.Request) {
	// Get language preference
	lang := h.getLanguageFromRequest(r)
	theme := "light" // Default theme

	developmentData := templates.DevelopmentPageData{
		AppLayoutData: templates.AppLayoutData{
			BaseLayoutData: templates.BaseLayoutData{
				Title:       "Development Assistant - GoAssistant",
				Description: "AI-powered development assistant",
				Lang:        lang,
				Theme:       theme,
			},
			CurrentPage: "development",
		},
		CodeAnalysis: templates.CodeAnalysisData{
			CurrentFile:  "main.go",
			Language:     "go",
			LinesOfCode:  156,
			Complexity:   12,
			TestCoverage: 85.5,
			Issues: []templates.DevelopmentCodeIssue{
				{
					Type:       "warning",
					Line:       45,
					Column:     12,
					Message:    "Function 'processData' has a cyclomatic complexity of 8 (threshold 5)",
					Suggestion: "Consider breaking this function into smaller, more focused functions",
				},
				{
					Type:       "error",
					Line:       89,
					Column:     23,
					Message:    "Error handling missing for database query",
					Suggestion: "Add proper error handling with context wrapping",
				},
			},
			Suggestions: []templates.CodeSuggestion{
				{
					Type:        "optimization",
					Priority:    "high",
					Title:       "Optimize Database Queries",
					Description: "Multiple N+1 queries detected in the user service. Consider using batch loading or joins.",
					CodeSnippet: "// Use batch loading\nusers, err := db.GetUsersByIDs(ctx, userIDs)",
				},
				{
					Type:        "security",
					Priority:    "medium",
					Title:       "SQL Injection Risk",
					Description: "Direct string concatenation in SQL query. Use parameterized queries instead.",
					CodeSnippet: "query := `SELECT * FROM users WHERE id = $1`\nrows, err := db.Query(query, userID)",
				},
			},
		},
		RecentProjects: []templates.ProjectItem{
			{
				ID:           "1",
				Name:         "GoAssistant",
				Path:         "/Users/koopa/go/src/github.com/koopa0/assistant-go",
				Language:     "go",
				LastModified: "2 hours ago",
				Status:       "active",
			},
			{
				ID:           "2",
				Name:         "frontend-app",
				Path:         "/Users/koopa/projects/frontend-app",
				Language:     "typescript",
				LastModified: "1 day ago",
				Status:       "idle",
			},
		},
		ActiveTools: []templates.DevelopmentTool{
			{
				ID:       "gopls",
				Name:     "Go Language Server",
				Category: "analysis",
				Status:   "running",
				LastUsed: "5 minutes ago",
			},
			{
				ID:       "go-test",
				Name:     "Go Test Runner",
				Category: "testing",
				Status:   "ready",
				LastUsed: "1 hour ago",
			},
			{
				ID:       "delve",
				Name:     "Delve Debugger",
				Category: "debugging",
				Status:   "ready",
				LastUsed: "3 hours ago",
			},
			{
				ID:       "pprof",
				Name:     "Go Profiler",
				Category: "profiling",
				Status:   "ready",
				LastUsed: "1 day ago",
			},
		},
	}

	component := templates.DevelopmentAssistantPage(developmentData)

	if err := component.Render(r.Context(), w); err != nil {
		h.logger.Error("Failed to render development page", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// HandleDatabase handles the database manager page
func (h *Handlers) HandleDatabase(w http.ResponseWriter, r *http.Request) {
	// Get language preference
	lang := h.getLanguageFromRequest(r)
	theme := "light" // Default theme

	databaseData := templates.DatabasePageData{
		AppLayoutData: templates.AppLayoutData{
			BaseLayoutData: templates.BaseLayoutData{
				Title:       "Database Manager - GoAssistant",
				Description: "SQL query editor and schema explorer",
				Lang:        lang,
				Theme:       theme,
			},
			CurrentPage: "database",
		},
		Connections: []templates.DatabaseConnectionItem{
			{
				ID:          "1",
				Name:        "Production DB",
				Type:        "postgres",
				Host:        "db.example.com",
				Port:        5432,
				Database:    "production",
				IsConnected: true,
				Status:      "connected",
			},
			{
				ID:          "2",
				Name:        "Development DB",
				Type:        "postgres",
				Host:        "localhost",
				Port:        5432,
				Database:    "development",
				IsConnected: false,
				Status:      "disconnected",
			},
		},
		ActiveConnection: &templates.DatabaseConnectionItem{
			ID:          "1",
			Name:        "Production DB",
			Type:        "postgres",
			Host:        "db.example.com",
			Port:        5432,
			Database:    "production",
			IsConnected: true,
			Status:      "connected",
		},
		Schema: templates.DatabaseSchemaData{
			Tables: []templates.TableItem{
				{
					Name:     "users",
					Schema:   "public",
					RowCount: 1523,
					Size:     "512 KB",
					Columns: []templates.ColumnItem{
						{
							Name:         "id",
							Type:         "uuid",
							IsNullable:   false,
							IsPrimaryKey: true,
							IsForeignKey: false,
							DefaultValue: "gen_random_uuid()",
						},
						{
							Name:         "email",
							Type:         "varchar(255)",
							IsNullable:   false,
							IsPrimaryKey: false,
							IsForeignKey: false,
							DefaultValue: "",
						},
						{
							Name:         "created_at",
							Type:         "timestamp",
							IsNullable:   false,
							IsPrimaryKey: false,
							IsForeignKey: false,
							DefaultValue: "now()",
						},
					},
				},
				{
					Name:     "conversations",
					Schema:   "public",
					RowCount: 4521,
					Size:     "2.3 MB",
					Columns: []templates.ColumnItem{
						{
							Name:         "id",
							Type:         "uuid",
							IsNullable:   false,
							IsPrimaryKey: true,
							IsForeignKey: false,
							DefaultValue: "gen_random_uuid()",
						},
						{
							Name:         "user_id",
							Type:         "uuid",
							IsNullable:   false,
							IsPrimaryKey: false,
							IsForeignKey: true,
							DefaultValue: "",
						},
						{
							Name:         "title",
							Type:         "text",
							IsNullable:   true,
							IsPrimaryKey: false,
							IsForeignKey: false,
							DefaultValue: "",
						},
					},
				},
			},
		},
		QueryHistory: []templates.QueryHistoryItem{
			{
				ID:           "1",
				Query:        "SELECT * FROM users WHERE created_at > NOW() - INTERVAL '7 days' ORDER BY created_at DESC",
				Timestamp:    "2 minutes ago",
				Duration:     "23ms",
				RowsAffected: 142,
				Status:       "success",
				Error:        "",
			},
			{
				ID:           "2",
				Query:        "UPDATE users SET last_login = NOW() WHERE id = '123e4567-e89b-12d3-a456-426614174000'",
				Timestamp:    "5 minutes ago",
				Duration:     "12ms",
				RowsAffected: 1,
				Status:       "success",
				Error:        "",
			},
			{
				ID:           "3",
				Query:        "SELECT COUNT(*) FROM conversations WHERE user_id IN (SELECT id FROM users WHERE plan = 'premium')",
				Timestamp:    "10 minutes ago",
				Duration:     "156ms",
				RowsAffected: 1,
				Status:       "success",
				Error:        "",
			},
			{
				ID:           "4",
				Query:        "DELETE FROM old_logs WHERE created_at < NOW() - INTERVAL '30 days'",
				Timestamp:    "15 minutes ago",
				Duration:     "542ms",
				RowsAffected: 0,
				Status:       "error",
				Error:        "relation \"old_logs\" does not exist",
			},
		},
	}

	component := templates.DatabaseManagerPage(databaseData)

	if err := component.Render(r.Context(), w); err != nil {
		h.logger.Error("Failed to render database page", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// HandleInfrastructure handles the infrastructure monitor page
func (h *Handlers) HandleInfrastructure(w http.ResponseWriter, r *http.Request) {
	// Get language preference
	lang := h.getLanguageFromRequest(r)
	theme := "light" // Default theme

	infrastructureData := templates.InfrastructurePageData{
		AppLayoutData: templates.AppLayoutData{
			BaseLayoutData: templates.BaseLayoutData{
				Title:       "Infrastructure Monitor - GoAssistant",
				Description: "Kubernetes and Docker container management",
				Lang:        lang,
				Theme:       theme,
			},
			CurrentPage: "infrastructure",
		},
		Clusters: []templates.ClusterItem{
			{
				ID:        "1",
				Name:      "production-k8s",
				Provider:  "k8s",
				Status:    "healthy",
				NodeCount: 5,
				PodCount:  47,
				Namespace: "default",
			},
			{
				ID:        "2",
				Name:      "staging-k8s",
				Provider:  "k8s",
				Status:    "warning",
				NodeCount: 3,
				PodCount:  23,
				Namespace: "default",
			},
			{
				ID:        "3",
				Name:      "local-docker",
				Provider:  "docker",
				Status:    "healthy",
				NodeCount: 1,
				PodCount:  12,
				Namespace: "N/A",
			},
		},
		ActiveCluster: &templates.ClusterItem{
			ID:        "1",
			Name:      "production-k8s",
			Provider:  "k8s",
			Status:    "healthy",
			NodeCount: 5,
			PodCount:  47,
			Namespace: "default",
		},
		Resources: templates.InfrastructureResources{
			Nodes: []templates.NodeItem{
				{
					Name:        "master-1",
					Status:      "ready",
					Role:        "master",
					Version:     "v1.28.2",
					CPUCapacity: "8 cores",
					MemCapacity: "32 GB",
					CPUUsage:    45.2,
					MemUsage:    62.8,
					PodCount:    12,
				},
				{
					Name:        "worker-1",
					Status:      "ready",
					Role:        "worker",
					Version:     "v1.28.2",
					CPUCapacity: "16 cores",
					MemCapacity: "64 GB",
					CPUUsage:    78.5,
					MemUsage:    85.3,
					PodCount:    18,
				},
				{
					Name:        "worker-2",
					Status:      "ready",
					Role:        "worker",
					Version:     "v1.28.2",
					CPUCapacity: "16 cores",
					MemCapacity: "64 GB",
					CPUUsage:    65.1,
					MemUsage:    92.0,
					PodCount:    17,
				},
			},
			Pods: []templates.PodItem{
				{
					Name:         "api-server-7d9b4c8d5-xvlzk",
					Namespace:    "default",
					Status:       "running",
					Node:         "worker-1",
					RestartCount: 0,
					Age:          "2d",
					CPUUsage:     "250m",
					MemUsage:     "512Mi",
				},
				{
					Name:         "frontend-6f7d8b9c5-abc123",
					Namespace:    "default",
					Status:       "running",
					Node:         "worker-2",
					RestartCount: 0,
					Age:          "5h",
					CPUUsage:     "100m",
					MemUsage:     "256Mi",
				},
				{
					Name:         "database-0",
					Namespace:    "default",
					Status:       "running",
					Node:         "worker-1",
					RestartCount: 1,
					Age:          "7d",
					CPUUsage:     "500m",
					MemUsage:     "2Gi",
				},
			},
			Services: []templates.ServiceItem{
				{
					Name:      "api-service",
					Namespace: "default",
					Type:      "LoadBalancer",
					ClusterIP: "10.96.0.1",
					Ports:     []string{"80:30080", "443:30443"},
					Age:       "30d",
				},
				{
					Name:      "database-service",
					Namespace: "default",
					Type:      "ClusterIP",
					ClusterIP: "10.96.0.2",
					Ports:     []string{"5432"},
					Age:       "30d",
				},
			},
			Deployments: []templates.DeploymentItem{
				{
					Name:      "api-server",
					Namespace: "default",
					Replicas:  "3/3",
					UpToDate:  3,
					Available: 3,
					Age:       "15d",
					Status:    "healthy",
				},
				{
					Name:      "frontend",
					Namespace: "default",
					Replicas:  "2/2",
					UpToDate:  2,
					Available: 2,
					Age:       "10d",
					Status:    "healthy",
				},
				{
					Name:      "worker",
					Namespace: "default",
					Replicas:  "4/5",
					UpToDate:  4,
					Available: 4,
					Age:       "20d",
					Status:    "updating",
				},
			},
		},
		Metrics: templates.InfrastructureMetrics{
			CPUUsage:    68.5,
			MemoryUsage: 78.2,
			NetworkIn:   "125 MB/s",
			NetworkOut:  "89 MB/s",
			DiskUsage:   45.8,
			ErrorRate:   0.12,
		},
	}

	component := templates.InfrastructureMonitorPage(infrastructureData)

	if err := component.Render(r.Context(), w); err != nil {
		h.logger.Error("Failed to render infrastructure page", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
