package i18n

import (
	"fmt"
	"time"
)

// SupportedLanguages defines the languages supported by the application
var SupportedLanguages = map[string]string{
	"en":    "English",
	"zh-TW": "繁體中文",
}

// Translations contains all translation strings
var Translations = map[string]map[string]string{
	"en": {
		// Navigation
		"nav.home":           "Home",
		"nav.chat":           "Chat",
		"nav.tools":          "Tools",
		"nav.development":    "Development",
		"nav.database":       "Database",
		"nav.infrastructure": "Infrastructure",
		"nav.settings":       "Settings",

		// Home page
		"home.greeting_personal":      "Welcome back",
		"home.greeting_generic":       "Welcome to GoAssistant",
		"home.subtitle":               "Your AI-powered development companion",
		"home.input_placeholder":      "How can I help you today?",
		"home.suggestion_performance": "Help me analyze Go code performance",
		"home.suggestion_database":    "Optimize my PostgreSQL queries",
		"home.suggestion_kubernetes":  "Debug Kubernetes deployment issues",
		"home.suggestion_testing":     "Write unit tests for my Go functions",
		"home.recent_chats":           "Recent Conversations",

		// Dashboard
		"dashboard.welcome":           "Welcome to GoAssistant",
		"dashboard.quick_stats":       "Quick Stats",
		"dashboard.recent_chats":      "Recent Conversations",
		"dashboard.active_agents":     "Active Agents",
		"dashboard.recent_activities": "Recent Activities",

		// Chat Interface
		"chat.new_conversation":   "New Conversation",
		"chat.type_message":       "Type your message...",
		"chat.send":               "Send",
		"chat.thinking":           "Thinking...",
		"chat.error":              "An error occurred",
		"chat.agent_online":       "Online",
		"chat.agent_offline":      "Offline",
		"chat.agent_processing":   "Processing",
		"chat.start_conversation": "Start a conversation with AI",
		"chat.select_topic":       "Select a topic or type your question",

		// Tools
		"tools.overview":     "Tools Overview",
		"tools.subtitle":     "Monitor and manage your AI development tools",
		"tools.active":       "Active",
		"tools.idle":         "Idle",
		"tools.error":        "Error",
		"tools.maintenance":  "Maintenance",
		"tools.configure":    "Configure",
		"tools.logs":         "Logs",
		"tools.test":         "Test",
		"tools.last_used":    "Last used",
		"tools.usage_today":  "Usage today",
		"tools.success_rate": "Success rate",

		// Development Assistant
		"dev.title":            "Go Development Assistant",
		"dev.code_editor":      "Code Editor",
		"dev.analysis_results": "Analysis Results",
		"dev.ast_analysis":     "AST Analysis",
		"dev.profiling":        "Profiling",
		"dev.trace":            "Trace",
		"dev.debug":            "Debug",

		// Database Manager
		"db.title":           "Database Manager",
		"db.schema_explorer": "Schema Explorer",
		"db.query_editor":    "Query Editor",
		"db.results":         "Results",
		"db.tables":          "Tables",
		"db.views":           "Views",
		"db.execute":         "Execute",

		// Infrastructure Monitor
		"infra.title":            "Infrastructure Monitor",
		"infra.cluster_overview": "Cluster Overview",
		"infra.nodes":            "Nodes",
		"infra.pods":             "Pods",
		"infra.cpu_usage":        "CPU Usage",
		"infra.deployments":      "Deployments",
		"infra.services":         "Services",

		// Settings
		"settings.title":         "Settings",
		"settings.general":       "General",
		"settings.language":      "Language",
		"settings.theme":         "Theme",
		"settings.dark_mode":     "Dark Mode",
		"settings.ai_providers":  "AI Providers",
		"settings.tools":         "Tools",
		"settings.appearance":    "Appearance",
		"settings.api_keys":      "API Keys",
		"settings.database":      "Database",
		"settings.notifications": "Notifications",
		"settings.privacy":       "Privacy & Security",

		// Common Actions
		"action.save":       "Save",
		"action.cancel":     "Cancel",
		"action.delete":     "Delete",
		"action.edit":       "Edit",
		"action.search":     "Search",
		"action.filter":     "Filter",
		"action.export":     "Export",
		"action.import":     "Import",
		"action.refresh":    "Refresh",
		"action.retry":      "Retry",
		"action.learn_more": "Learn More",
		"action.menu":       "Menu",
		"action.send":       "Send",

		// Status Messages
		"status.loading":     "Loading...",
		"status.processing":  "Processing...",
		"status.please_wait": "Please wait",
		"status.success":     "Success",
		"status.error":       "Error",
		"status.completed":   "Completed",

		// Error Messages
		"error.title":        "Something went wrong",
		"error.network":      "Network error. Please check your connection.",
		"error.server":       "Server error. Please try again later.",
		"error.validation":   "Please check your input and try again.",
		"error.not_found":    "The requested resource was not found.",
		"error.unauthorized": "You don't have permission to access this resource.",
		"error.rate_limit":   "Too many requests. Please wait a moment.",

		// Form Validation
		"validation.required":   "This field is required",
		"validation.email":      "Please enter a valid email address",
		"validation.min_length": "Must be at least %d characters",
		"validation.max_length": "Must be less than %d characters",

		// Success Messages
		"success.saved":   "Successfully saved",
		"success.deleted": "Successfully deleted",
		"success.updated": "Successfully updated",
		"success.created": "Successfully created",

		// Time Formats
		"time.just_now":    "Just now",
		"time.minutes_ago": "%d minutes ago",
		"time.hours_ago":   "%d hours ago",
		"time.days_ago":    "%d days ago",

		// Chat specific
		"chat.total_conversations": "Total conversations: %d",
		"action.view_all":          "View All",
		"chat.conversations":       "Conversations",
		"chat.select_agent":        "Select an agent",
		"chat.attach_file":         "Attach file",
	},
	"zh-TW": {
		// Navigation
		"nav.home":           "首頁",
		"nav.chat":           "對話",
		"nav.tools":          "工具",
		"nav.development":    "開發助手",
		"nav.database":       "資料庫",
		"nav.infrastructure": "基礎設施",
		"nav.settings":       "設定",

		// Home page
		"home.greeting_personal":      "歡迎回來",
		"home.greeting_generic":       "歡迎使用 GoAssistant",
		"home.subtitle":               "您的 AI 開發夥伴",
		"home.input_placeholder":      "今天我能為您做什麼？",
		"home.suggestion_performance": "協助分析 Go 程式碼效能",
		"home.suggestion_database":    "優化 PostgreSQL 查詢",
		"home.suggestion_kubernetes":  "除錯 Kubernetes 部署問題",
		"home.suggestion_testing":     "為 Go 函數撰寫單元測試",
		"home.recent_chats":           "最近對話",

		// Dashboard
		"dashboard.welcome":           "歡迎使用 GoAssistant",
		"dashboard.quick_stats":       "快速統計",
		"dashboard.recent_chats":      "最近對話",
		"dashboard.active_agents":     "活躍代理",
		"dashboard.recent_activities": "最近活動",

		// Chat Interface
		"chat.new_conversation":   "新對話",
		"chat.type_message":       "輸入您的訊息...",
		"chat.send":               "傳送",
		"chat.thinking":           "思考中...",
		"chat.error":              "發生錯誤",
		"chat.agent_online":       "線上",
		"chat.agent_offline":      "離線",
		"chat.agent_processing":   "處理中",
		"chat.start_conversation": "開始與 AI 對話",
		"chat.select_topic":       "選擇主題或輸入您的問題",

		// Tools
		"tools.overview":     "工具概覽",
		"tools.subtitle":     "監控和管理您的 AI 開發工具",
		"tools.active":       "活躍",
		"tools.idle":         "閒置",
		"tools.error":        "錯誤",
		"tools.maintenance":  "維護中",
		"tools.configure":    "設定",
		"tools.logs":         "日誌",
		"tools.test":         "測試",
		"tools.last_used":    "最後使用",
		"tools.usage_today":  "今日使用",
		"tools.success_rate": "成功率",

		// Development Assistant
		"dev.title":            "Go 開發助手",
		"dev.code_editor":      "程式碼編輯器",
		"dev.analysis_results": "分析結果",
		"dev.ast_analysis":     "AST 分析",
		"dev.profiling":        "效能分析",
		"dev.trace":            "追蹤",
		"dev.debug":            "除錯",

		// Database Manager
		"db.title":           "資料庫管理器",
		"db.schema_explorer": "結構探索器",
		"db.query_editor":    "查詢編輯器",
		"db.results":         "結果",
		"db.tables":          "資料表",
		"db.views":           "檢視",
		"db.execute":         "執行",

		// Infrastructure Monitor
		"infra.title":            "基礎設施監控",
		"infra.cluster_overview": "叢集概覽",
		"infra.nodes":            "節點",
		"infra.pods":             "Pod",
		"infra.cpu_usage":        "CPU 使用率",
		"infra.deployments":      "部署",
		"infra.services":         "服務",

		// Settings
		"settings.title":         "設定",
		"settings.general":       "一般",
		"settings.language":      "語言",
		"settings.theme":         "主題",
		"settings.dark_mode":     "深色模式",
		"settings.ai_providers":  "AI 提供者",
		"settings.tools":         "工具",
		"settings.appearance":    "外觀",
		"settings.api_keys":      "API 金鑰",
		"settings.database":      "資料庫",
		"settings.notifications": "通知",
		"settings.privacy":       "隱私與安全",

		// Common Actions
		"action.save":       "儲存",
		"action.cancel":     "取消",
		"action.delete":     "刪除",
		"action.edit":       "編輯",
		"action.search":     "搜尋",
		"action.filter":     "篩選",
		"action.export":     "匯出",
		"action.import":     "匯入",
		"action.refresh":    "重新整理",
		"action.retry":      "重試",
		"action.learn_more": "了解更多",
		"action.menu":       "選單",
		"action.send":       "傳送",

		// Status Messages
		"status.loading":     "載入中...",
		"status.processing":  "處理中...",
		"status.please_wait": "請稍候",
		"status.success":     "成功",
		"status.error":       "錯誤",
		"status.completed":   "已完成",

		// Error Messages
		"error.title":        "發生錯誤",
		"error.network":      "網路錯誤，請檢查您的連線。",
		"error.server":       "伺服器錯誤，請稍後再試。",
		"error.validation":   "請檢查您的輸入並重試。",
		"error.not_found":    "找不到請求的資源。",
		"error.unauthorized": "您沒有權限存取此資源。",
		"error.rate_limit":   "請求過於頻繁，請稍候再試。",

		// Form Validation
		"validation.required":   "此欄位為必填",
		"validation.email":      "請輸入有效的電子郵件地址",
		"validation.min_length": "至少需要 %d 個字元",
		"validation.max_length": "不能超過 %d 個字元",

		// Success Messages
		"success.saved":   "儲存成功",
		"success.deleted": "刪除成功",
		"success.updated": "更新成功",
		"success.created": "建立成功",

		// Time Formats
		"time.just_now":    "剛剛",
		"time.minutes_ago": "%d 分鐘前",
		"time.hours_ago":   "%d 小時前",
		"time.days_ago":    "%d 天前",

		// Chat specific
		"chat.total_conversations": "總對話數: %d",
		"action.view_all":          "查看全部",
		"chat.conversations":       "對話",
		"chat.select_agent":        "選擇代理",
		"chat.attach_file":         "附加檔案",
	},
}

// T translates a key to the specified language
func T(key, lang string) string {
	if translations, exists := Translations[lang]; exists {
		if translation, exists := translations[key]; exists {
			return translation
		}
	}

	// Fallback to English
	if translations, exists := Translations["en"]; exists {
		if translation, exists := translations[key]; exists {
			return translation
		}
	}

	// Return key if no translation found
	return key
}

// Tf translates a key with formatting to the specified language
func Tf(key, lang string, args ...interface{}) string {
	translation := T(key, lang)
	return fmt.Sprintf(translation, args...)
}

// FormatDateTime formats time according to language preferences
func FormatDateTime(t time.Time, lang string) string {
	switch lang {
	case "zh-TW":
		return t.Format("2006年1月2日 15:04")
	default:
		return t.Format("Jan 2, 2006 3:04 PM")
	}
}

// FormatDate formats date according to language preferences
func FormatDate(t time.Time, lang string) string {
	switch lang {
	case "zh-TW":
		return t.Format("2006年1月2日")
	default:
		return t.Format("Jan 2, 2006")
	}
}

// FormatTime formats time according to language preferences
func FormatTime(t time.Time, lang string) string {
	switch lang {
	case "zh-TW":
		return t.Format("15:04")
	default:
		return t.Format("3:04 PM")
	}
}
