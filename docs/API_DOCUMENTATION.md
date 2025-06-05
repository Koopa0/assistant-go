# Assistant API 完整文檔

本文檔提供 Assistant 智慧開發助手的完整 API 參考。所有 API 均遵循 RESTful 設計原則，並使用統一的回應格式。

## 目錄

1. [基礎資訊](#基礎資訊)
2. [認證與授權](#認證與授權)
3. [系統狀態 API](#系統狀態-api)
4. [對話管理 API](#對話管理-api)
5. [工具管理 API](#工具管理-api)
6. [記憶系統 API](#記憶系統-api)
7. [使用者管理 API](#使用者管理-api)
8. [時間軸 API](#時間軸-api)
9. [知識圖譜 API](#知識圖譜-api)
10. [學習系統 API](#學習系統-api)
11. [洞察分析 API](#洞察分析-api)
12. [協作系統 API](#協作系統-api)
13. [分析儀表板 API](#分析儀表板-api)
14. [WebSocket 即時通訊](#websocket-即時通訊)
15. [資料匯出 API](#資料匯出-api)

## 基礎資訊

### API 基礎網址
```
http://localhost:8080/api
```

### 統一回應格式

**成功回應**
```json
{
  "success": true,
  "data": {},
  "message": "操作成功",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

**分頁回應**
```json
{
  "success": true,
  "data": [],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "totalPages": 5
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

**錯誤回應**
```json
{
  "success": false,
  "error": "ERROR_CODE",
  "message": "錯誤訊息",
  "details": {},
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### 通用請求標頭
```http
Content-Type: application/json
Accept: application/json
Accept-Language: zh-TW
Authorization: Bearer {access_token}
```

### 錯誤代碼
| 代碼 | HTTP 狀態碼 | 說明 |
|------|------------|------|
| UNAUTHORIZED | 401 | 未認證 |
| FORBIDDEN | 403 | 無權限 |
| NOT_FOUND | 404 | 資源不存在 |
| INVALID_REQUEST | 400 | 請求無效 |
| RATE_LIMITED | 429 | 請求過於頻繁 |
| SERVER_ERROR | 500 | 伺服器錯誤 |
| SERVICE_UNAVAILABLE | 503 | 服務暫時不可用 |

## 認證與授權

### 登入
**POST** `/auth/login`

**請求範例**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**回應範例**
```json
{
  "success": true,
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresIn": 3600,
    "user": {
      "id": "user_123",
      "email": "user@example.com",
      "name": "使用者名稱",
      "avatar": "https://example.com/avatar.jpg",
      "role": "user",
      "preferences": {
        "language": "zh-TW",
        "theme": "light"
      }
    }
  },
  "message": "登入成功",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### 註冊
**POST** `/auth/register`

**請求範例**
```json
{
  "email": "newuser@example.com",
  "password": "securePassword123",
  "name": "新使用者"
}
```

### 重新整理權杖
**POST** `/auth/refresh`

**請求標頭**
```http
Authorization: Bearer {refresh_token}
```

### 登出
**POST** `/auth/logout`

**請求標頭**
```http
Authorization: Bearer {access_token}
```

## 系統狀態 API

### 取得系統狀態
**GET** `/system/status`

**回應範例**
```json
{
  "success": true,
  "data": {
    "health": "healthy",
    "uptime": 3456789,
    "activeUsers": 234,
    "cpuUsage": 45.2,
    "memoryUsage": 67.8,
    "responseTime": 120,
    "requestsPerMinute": 150,
    "errorRate": 0.02,
    "services": {
      "database": "healthy",
      "cache": "healthy",
      "queue": "degraded"
    },
    "version": {
      "api": "v1.0.0",
      "buildDate": "2024-01-01",
      "goVersion": "go1.21.5"
    }
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### 取得系統活動
**GET** `/system/activities`

**查詢參數**
- `type`: 活動類型 (query, toolExecution, error)
- `startDate`: 開始日期 (ISO 8601)
- `endDate`: 結束日期 (ISO 8601)
- `page`: 頁碼 (預設 1)
- `limit`: 每頁數量 (預設 20)

### 取得系統指標
**GET** `/system/metrics`

### 健康檢查
**GET** `/system/health`

### 版本資訊
**GET** `/system/version`

## 對話管理 API

### 取得對話列表
**GET** `/conversations`

**查詢參數**
- `search`: 搜尋關鍵字
- `category`: 分類篩選 (Frontend, Backend, Database, General)
- `status`: 狀態篩選 (active, archived)
- `sortBy`: 排序欄位 (lastMessage, created)
- `sortOrder`: 排序方向 (asc, desc)
- `page`: 頁碼
- `limit`: 每頁數量

**回應範例**
```json
{
  "success": true,
  "data": [
    {
      "id": "conv_123",
      "title": "Angular 效能優化討論",
      "lastMessage": "我已經幫您優化了元件的變更偵測策略...",
      "timestamp": "2024-01-01T00:00:00Z",
      "messageCount": 42,
      "category": "Frontend",
      "status": "active",
      "participants": ["user_123", "assistant"],
      "tags": ["Angular", "效能優化", "前端開發"],
      "metadata": {
        "language": "Go",
        "toolsUsed": ["程式碼分析器", "效能監測器"],
        "satisfaction": 4.5
      }
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "totalPages": 5
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### 取得單一對話
**GET** `/conversations/{conversationId}`

### 建立新對話
**POST** `/conversations`

**請求範例**
```json
{
  "title": "新對話標題",
  "initialMessage": "你好，我想了解如何使用 Golang 建立微服務",
  "metadata": {
    "language": "Go",
    "context": "microservices"
  }
}
```

### 發送訊息
**POST** `/conversations/{conversationId}/messages`

**請求範例**
```json
{
  "content": "請幫我生成一個簡單的 HTTP 伺服器範例",
  "attachments": [
    {
      "type": "code",
      "filename": "main.go",
      "content": "package main..."
    }
  ]
}
```

### 封存對話
**POST** `/conversations/{conversationId}/archive`

### 取消封存對話
**POST** `/conversations/{conversationId}/unarchive`

### 匯出對話
**GET** `/conversations/{conversationId}/export`

**查詢參數**
- `format`: 匯出格式 (json, txt, pdf)

## 工具管理 API

### 取得增強型工具資訊
**GET** `/api/tools/enhanced`

**回應範例**
```json
{
  "success": true,
  "data": [
    {
      "id": "go_analyzer",
      "name": "Go 程式碼分析器",
      "description": "分析 Go 程式碼結構和品質",
      "category": "Development",
      "version": "1.0.0",
      "usage": 156,
      "last_used": "2024-01-15T09:30:00Z",
      "is_favorite": true,
      "average_rating": 4.8,
      "execution_count": 245,
      "success_rate": 96.7,
      "average_execution_time_ms": 1250
    }
  ],
  "message": "取得工具資訊成功",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 執行工具
**POST** `/api/tools/execute`

**請求範例**
```json
{
  "tool_name": "go_analyzer",
  "input": {
    "code": "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}",
    "analysis_type": "quality"
  },
  "config": {
    "include_suggestions": true,
    "severity_level": "medium"
  }
}
```

### 取得工具使用統計
**GET** `/api/tools/usage/stats`

**回應範例**
```json
{
  "success": true,
  "data": [
    {
      "tool_name": "go_analyzer",
      "total_executions": 245,
      "success_count": 237,
      "failure_count": 8,
      "success_rate": 96.7,
      "average_execution_time_ms": 1250,
      "last_used": "2024-01-15T09:30:00Z"
    }
  ],
  "message": "取得工具使用統計成功",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 取得工具使用歷史
**GET** `/api/tools/usage/history`

**查詢參數**
- `tool_name`: 工具名稱篩選 (可選)
- `page`: 頁碼 (預設 1)
- `limit`: 每頁數量 (預設 20)

### 取得特定工具歷史
**GET** `/api/tools/{toolName}/history`

### 切換工具收藏狀態
**POST** `/api/tools/{toolName}/favorite`

**請求範例**
```json
{
  "is_favorite": true
}
```

### 移除工具收藏
**DELETE** `/api/tools/{toolName}/favorite`

### 取得工具分析
**GET** `/api/tools/analytics`

### 取得工具詳細分析
**GET** `/api/tools/{toolName}/analytics`

## 記憶系統 API

### 儲存記憶
**POST** `/api/memory/store`

**請求範例**
```json
{
  "type": "episodic",
  "content": "完成了 Go 微服務的 JWT 認證實現",
  "context": {
    "project": "microservice-auth",
    "technology": "Go",
    "outcome": "success"
  },
  "importance": 0.8
}
```

### 搜尋記憶
**GET** `/api/memory/search`

**查詢參數**
- `query`: 搜尋關鍵字
- `type`: 記憶類型 (episodic, semantic, procedural)
- `limit`: 結果數量

## 使用者管理 API

### 取得使用者個人資料
**GET** `/api/users/profile`

**請求標頭**
```http
Authorization: Bearer {access_token}
```

**回應範例**
```json
{
  "success": true,
  "data": {
    "id": "user_123",
    "username": "john_doe",
    "email": "john@example.com",
    "full_name": "John Doe",
    "avatar_url": "https://example.com/avatar.jpg",
    "preferences": {
      "language": "zh-TW",
      "theme": "light",
      "defaultProgrammingLanguage": "Go",
      "emailNotifications": true,
      "timezone": "Asia/Taipei"
    },
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  },
  "message": "取得使用者資料成功",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 更新個人資料
**PUT** `/api/users/profile`

**請求範例**
```json
{
  "full_name": "新的全名",
  "avatar_url": "https://example.com/new-avatar.jpg"
}
```

### 更新偏好設定
**PUT** `/api/users/preferences`

**請求範例**
```json
{
  "preferences": {
    "language": "zh-TW",
    "theme": "dark",
    "defaultProgrammingLanguage": "TypeScript",
    "emailNotifications": false,
    "timezone": "Asia/Tokyo"
  }
}
```

### 變更密碼
**PUT** `/api/users/password`

**請求範例**
```json
{
  "current_password": "舊密碼",
  "new_password": "新密碼"
}
```

### 取得使用者統計
**GET** `/api/users/statistics`

**回應範例**
```json
{
  "success": true,
  "data": {
    "id": "user_123",
    "username": "john_doe",
    "email": "john@example.com",
    "full_name": "John Doe",
    "created_at": "2024-01-01T00:00:00Z",
    "total_conversations": 45,
    "total_tools_used": 12,
    "total_tokens_used": 125000,
    "total_cost_cents": 2500
  },
  "message": "取得統計資料成功",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 取得活動摘要
**GET** `/api/users/activity`

**回應範例**
```json
{
  "success": true,
  "data": {
    "id": "user_123",
    "conversations_count": 45,
    "messages_count": 382,
    "tool_executions_count": 127,
    "last_activity_at": "2024-01-15T09:45:00Z",
    "days_since_signup": 14
  },
  "message": "取得活動摘要成功",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 取得使用者設定
**GET** `/api/users/settings`

**回應範例**
```json
{
  "success": true,
  "data": {
    "language": "zh-TW",
    "theme": "light",
    "default_programming_language": "Go",
    "email_notifications": true,
    "timezone": "Asia/Taipei"
  },
  "message": "取得設定成功",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 新增收藏工具
**POST** `/api/users/tools/{toolId}/favorite`

**路徑參數**
- `toolId`: 工具 ID

### 移除收藏工具
**DELETE** `/api/users/tools/{toolId}/favorite`

**路徑參數**
- `toolId`: 工具 ID

### 註冊新使用者 (公開端點)
**POST** `/api/users/register`

**請求範例**
```json
{
  "username": "new_user",
  "email": "newuser@example.com",
  "password": "securePassword123",
  "full_name": "新使用者",
  "preferences": {
    "language": "zh-TW",
    "theme": "light"
  }
}
```

### 停用帳戶
**DELETE** `/api/users/account`

**請求標頭**
```http
Authorization: Bearer {access_token}
```

## 時間軸 API

### 取得開發事件
**GET** `/api/v1/timeline/events`

**查詢參數**
- `startDate`: 開始日期
- `endDate`: 結束日期
- `type`: 事件類型 (coding, learning, debugging, refactoring)
- `tags`: 標籤篩選

**回應範例**
```json
{
  "success": true,
  "data": [
    {
      "id": "event_123",
      "type": "coding",
      "title": "實現使用者認證系統",
      "description": "完成 JWT 認證和授權機制",
      "startTime": "2024-01-01T09:00:00Z",
      "endTime": "2024-01-01T12:00:00Z",
      "duration": "3h",
      "tags": ["authentication", "security", "backend"],
      "metrics": {
        "linesOfCode": 500,
        "filesModified": 8,
        "testsAdded": 12
      }
    }
  ],
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### 取得開發統計
**GET** `/api/v1/timeline/statistics`

### 取得時間軸洞察
**GET** `/api/v1/timeline/insights`

### 取得開發模式
**GET** `/api/v1/timeline/patterns`

## 知識圖譜 API

### 取得知識節點
**GET** `/api/v1/knowledge/graph/nodes`

**查詢參數**
- `type`: 節點類型 (concept, technology, pattern, problem, solution)
- `search`: 搜尋關鍵字
- `tags`: 標籤篩選

### 建立知識節點
**POST** `/api/v1/knowledge/graph/nodes`

**請求範例**
```json
{
  "type": "technology",
  "name": "GraphQL",
  "description": "現代 API 查詢語言",
  "properties": {
    "category": "API",
    "difficulty": "medium",
    "useCase": ["API 開發", "資料查詢"]
  },
  "tags": ["api", "query", "graphql"]
}
```

### 視覺化知識圖譜
**GET** `/api/v1/knowledge/graph/visualize`

**查詢參數**
- `layout`: 佈局類型 (force-directed, hierarchical, circular)
- `depth`: 顯示深度
- `nodeId`: 中心節點 ID

### 知識路徑查詢
**GET** `/api/v1/knowledge/graph/path`

**查詢參數**
- `from`: 起始節點 ID
- `to`: 目標節點 ID

## 學習系統 API

### 記錄學習事件
**POST** `/api/v1/learning/events`

**請求範例**
```json
{
  "type": "skill_acquired",
  "skill": "Docker 容器化",
  "level": "intermediate",
  "context": {
    "project": "微服務部署",
    "duration": "2 weeks"
  }
}
```

### 取得學習模式
**GET** `/api/v1/learning/patterns`

### 取得學習偏好
**GET** `/api/v1/learning/preferences`

### 取得學習報告
**GET** `/api/v1/learning/report`

**查詢參數**
- `period`: 時間範圍 (week, month, quarter, year)

## 洞察分析 API

### 取得洞察摘要
**GET** `/api/v1/insights/summary`

**查詢參數**
- `period`: 時間範圍

**回應範例**
```json
{
  "success": true,
  "data": {
    "period": "month",
    "overallHealth": 0.85,
    "keyMetrics": {
      "productivity": 0.78,
      "codeQuality": 0.92,
      "learningProgress": 0.65
    },
    "topInsights": [
      "您在下午 2-5 點的生產力最高",
      "Go 程式碼品質持續提升",
      "建議加強前端框架學習"
    ],
    "achievements": [
      {
        "id": "ach_001",
        "title": "程式碼品質大師",
        "description": "連續 30 天保持高品質程式碼",
        "earnedAt": "2024-01-15T00:00:00Z"
      }
    ],
    "improvementAreas": [
      "測試覆蓋率",
      "文檔撰寫"
    ]
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### 取得開發洞察
**GET** `/api/v1/insights/development`

### 取得生產力分析
**GET** `/api/v1/insights/productivity`

### 取得程式碼品質分析
**GET** `/api/v1/insights/code-quality`

### 取得技能差距分析
**GET** `/api/v1/insights/skill-gaps`

### 取得預測分析
**GET** `/api/v1/insights/predictions`

### 取得個人化建議
**GET** `/api/v1/insights/recommendations`

## 協作系統 API

### 取得可用代理
**GET** `/api/v1/collaboration/agents`

**回應範例**
```json
{
  "success": true,
  "data": [
    {
      "id": "agent_001",
      "name": "程式碼專家",
      "type": "development",
      "capabilities": ["code_review", "refactoring", "optimization"],
      "expertiseDomains": ["Go", "Python", "JavaScript"],
      "currentWorkload": 0.3,
      "status": "available",
      "performanceScore": 0.95
    }
  ],
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### 建立協作會話
**POST** `/api/v1/collaboration/sessions`

**請求範例**
```json
{
  "name": "微服務架構設計",
  "description": "設計高可用的微服務系統",
  "requiredCapabilities": ["architecture", "database", "devops"],
  "estimatedDuration": "2h"
}
```

### 分配任務
**POST** `/api/v1/collaboration/sessions/{sessionId}/tasks`

### 取得協作效率
**GET** `/api/v1/collaboration/effectiveness`

## 分析儀表板 API

### 取得儀表板概覽
**GET** `/api/v1/analytics/overview`

### 取得活動熱圖
**GET** `/api/v1/analytics/activity-heatmap`

**查詢參數**
- `period`: 時間範圍 (week, month)
- `timezone`: 時區 (預設 UTC)

**回應範例**
```json
{
  "success": true,
  "data": {
    "period": "week",
    "timezone": "Asia/Taipei",
    "data": [
      {
        "day": "Monday",
        "hour": 9,
        "value": 85,
        "activities": ["coding", "debugging"]
      }
    ],
    "peakHours": [14, 15, 16],
    "totalHours": 42
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### 取得技能分布
**GET** `/api/v1/analytics/skill-distribution`

### 取得程式碼指標
**GET** `/api/v1/analytics/code-metrics`

### 取得專案完成預測
**GET** `/api/v1/analytics/project-completion`

### 取得疲勞預測
**GET** `/api/v1/analytics/burnout-prediction`

### 取得儀表板資料
**GET** `/api/v1/analytics/dashboard`

## WebSocket 即時通訊

### 連線端點
```
ws://localhost:8080/v1/ws?token={access_token}
```

### 訊息格式

**客戶端發送 - 聊天訊息**
```json
{
  "type": "message",
  "data": {
    "conversationId": "conv_123",
    "content": "請幫我分析這段程式碼"
  }
}
```

**伺服器推送 - 聊天回應**
```json
{
  "type": "message",
  "data": {
    "conversationId": "conv_123",
    "messageId": "msg_456",
    "role": "assistant",
    "content": "正在分析您的程式碼...",
    "timestamp": "2024-01-01T00:00:00Z",
    "metadata": {
      "provider": "claude",
      "model": "claude-3-opus",
      "tokensUsed": 150
    }
  }
}
```

**伺服器推送 - 工具執行進度**
```json
{
  "type": "toolProgress",
  "data": {
    "executionId": "exec_789",
    "toolId": "tool_001",
    "progress": 75,
    "status": "running",
    "message": "正在生成程式碼..."
  }
}
```

**伺服器推送 - 系統通知**
```json
{
  "type": "notification",
  "data": {
    "level": "info",
    "title": "系統更新",
    "message": "新功能已上線",
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

### 訂閱頻道
```json
{
  "type": "subscribe",
  "data": {
    "channel": "user_123_notifications"
  }
}
```

### 心跳機制
**客戶端 Ping**
```json
{
  "type": "ping"
}
```

**伺服器 Pong**
```json
{
  "type": "pong",
  "data": {
    "timestamp": "2024-01-01T00:00:00Z"
  }
}
```

## 資料匯出 API

### 匯出資料
**POST** `/export`

**請求範例**
```json
{
  "type": "conversations",
  "format": "json",
  "dateRange": {
    "start": "2024-01-01T00:00:00Z",
    "end": "2024-12-31T23:59:59Z"
  },
  "includeMetadata": true
}
```

**支援的匯出類型**
- `conversations`: 對話紀錄
- `tools`: 工具使用紀錄
- `insights`: 洞察報告
- `knowledge`: 知識圖譜

**支援的格式**
- `json`: JSON 格式
- `csv`: CSV 格式
- `pdf`: PDF 報告

## 速率限制

API 使用滑動視窗速率限制：
- 一般使用者：每分鐘 60 次請求
- 付費使用者：每分鐘 300 次請求
- 企業使用者：每分鐘 1000 次請求

速率限制資訊會在回應標頭中返回：
```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1640995200
```

## SDK 支援

### JavaScript/TypeScript
```typescript
import { AssistantClient } from '@assistant/sdk';

const client = new AssistantClient({
  apiKey: 'your-api-key',
  baseUrl: 'http://localhost:8080/api'
});

// 發送訊息
const response = await client.conversations.sendMessage('conv_123', {
  content: '請幫我優化這段程式碼'
});
```

### Go
```go
import "github.com/koopa0/assistant-go/sdk"

client := sdk.NewClient(
    sdk.WithAPIKey("your-api-key"),
    sdk.WithBaseURL("http://localhost:8080/api"),
)

// 發送訊息
response, err := client.Conversations.SendMessage(ctx, "conv_123", &sdk.MessageRequest{
    Content: "請幫我優化這段程式碼",
})
```

## 注意事項

1. 所有時間戳記都使用 ISO 8601 格式 (UTC)
2. 分頁從第 1 頁開始
3. 檔案上傳使用 multipart/form-data
4. 大型回應支援 gzip 壓縮
5. 所有 API 支援 CORS
6. 建議使用 HTTPS 在生產環境
7. API 版本會在 URL 中明確標示 (v1, v2 等)

## 聯絡與支援

如有任何問題或建議，請透過以下方式聯絡：
- GitHub Issues: [github.com/koopa0/assistant-go/issues](https://github.com/koopa0/assistant-go/issues)
- Email: support@assistant.example.com