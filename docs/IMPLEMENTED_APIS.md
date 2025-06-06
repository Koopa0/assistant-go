# 已實現的 API 端點清單

本文檔詳細列出了 Assistant 專案中已經實現的所有 API 端點，包括其功能描述、請求格式、回應格式和使用範例。

**最後更新**: 2025-06-06
**API 版本**: v2
**基礎路徑**: http://localhost:8100

## 🎯 實現狀態總覽

### 核心功能實現狀態

- ✅ **HTTP Server**: 完全實現 (使用 Go 1.24 的新路由模式)
- ✅ **WebSocket**: 完全實現 (即時通訊)
- ✅ **AI Integration**: 90% 完成 (Claude, Gemini + 增強 Prompt)
- ✅ **記憶系統**: 完全實現 (Working Memory + Database)
- ✅ **對話系統**: 完全實現 (9個端點)
- ✅ **工具系統**: 80% 完成 (Go開發工具、Docker、PostgreSQL)
- ⏳ **使用者系統**: 50% 完成 (基礎認證實現)
- ⏳ **分析系統**: 20% 完成 (基礎架構)
- ⏳ **協作系統**: 10% 完成 (基礎架構)

## 📡 WebSocket API

### 即時通訊連接

**端點**: `ws://localhost:8100/ws`

**實現位置**: `/internal/platform/server/websocket/`

**功能特性**:

- JWT 令牌驗證 (支援查詢參數和 Authorization 標頭)
- 使用者身份識別和會話管理
- 即時訊息傳送和接收
- 心跳檢測 (Ping/Pong)
- 斷線自動重連支援

**連接範例**:

```javascript
// 使用 JWT token 連接
const ws = new WebSocket("ws://localhost:8100/ws?token=YOUR_JWT_TOKEN");

// 或使用 Authorization header
const ws = new WebSocket("ws://localhost:8100/ws", {
  headers: {
    Authorization: "Bearer YOUR_JWT_TOKEN",
  },
});

ws.onopen = function () {
  console.log("WebSocket 連接已建立");

  // 發送訊息
  ws.send(
    JSON.stringify({
      type: "message",
      content: "Hello, Assistant!",
    }),
  );
};

ws.onmessage = function (event) {
  const message = JSON.parse(event.data);
  console.log("收到訊息:", message);
};
```

## 🧠 AI 系統 API

### 1. 處理查詢 (主要端點)

**端點**: `POST /api/query`

**功能**: 使用增強的 Prompt 系統處理使用者查詢

**請求格式**:

```json
{
  "query": "分析這段 Go 代碼的性能問題",
  "conversation_id": "conv_123", // 可選
  "context": {
    // 可選
    "project_type": "microservice",
    "files": ["main.go", "handler.go"]
  }
}
```

**回應格式**:

```json
{
  "success": true,
  "data": {
    "response": "根據分析，我發現以下性能問題...",
    "conversation_id": "conv_123",
    "message_id": "msg_456",
    "tools_used": ["godev", "performance_analyzer"],
    "execution_time": "2.5s",
    "tokens_used": 1500,
    "provider": "claude",
    "model": "claude-3-sonnet-20240229"
  }
}
```

### 2. 增強查詢處理

**端點**: `POST /api/v2/query`

**功能**: 使用專門的 Prompt 模板處理不同類型的任務

**支援的任務類型**:

- `code_analysis` - 代碼分析
- `refactoring` - 重構建議
- `performance` - 性能優化
- `architecture` - 架構審查
- `test_generation` - 測試生成
- `error_diagnosis` - 錯誤診斷
- `workspace_analysis` - 工作區分析

## 💬 對話系統 API

### 1. 取得對話列表

**端點**: `GET /conversations`

**查詢參數**:

- `search` - 搜尋關鍵字
- `category` - 分類篩選
- `status` - 狀態篩選 (active, archived)
- `sortBy` - 排序欄位 (created, updated)
- `sortOrder` - 排序方向 (asc, desc)
- `page` - 頁碼 (預設: 1)
- `limit` - 每頁數量 (預設: 20)

**回應格式**:

```json
{
  "success": true,
  "data": [
    {
      "id": "conv_123",
      "title": "Go 性能優化討論",
      "category": "development",
      "tags": ["golang", "performance"],
      "created_at": "2025-01-06T10:00:00Z",
      "updated_at": "2025-01-06T11:00:00Z",
      "message_count": 15,
      "last_message": "已完成性能分析報告",
      "status": "active"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "total_pages": 8
  }
}
```

### 2. 建立新對話

**端點**: `POST /conversations`

**請求格式**:

```json
{
  "title": "Docker 最佳實踐",
  "initialMessage": "請教我如何優化 Go 應用的 Dockerfile",
  "metadata": {
    "project": "assistant-go",
    "priority": "high"
  }
}
```

### 3. 取得對話詳情

**端點**: `GET /conversations/{conversationId}`

**功能**: 取得完整對話歷史，包含所有訊息

### 4. 發送訊息

**端點**: `POST /conversations/{conversationId}/messages`

**請求格式**:

```json
{
  "content": "請分析這段代碼的複雜度",
  "attachments": [
    {
      "type": "code",
      "filename": "analyzer.go",
      "content": "package main..."
    }
  ]
}
```

### 5. 更新對話

**端點**: `PUT /conversations/{conversationId}`

**功能**: 更新對話標題、分類、標籤等

### 6. 刪除對話

**端點**: `DELETE /conversations/{conversationId}`

### 7. 封存對話

**端點**: `POST /conversations/{conversationId}/archive`

### 8. 取消封存

**端點**: `POST /conversations/{conversationId}/unarchive`

### 9. 匯出對話

**端點**: `GET /conversations/{conversationId}/export?format={format}`

**支援格式**: `json`, `txt`

## 🛠️ 工具系統 API

### 1. 取得工具列表

**端點**: `GET /api/tools`

**當前可用工具**:

```json
{
  "success": true,
  "data": [
    {
      "name": "godev",
      "description": "Go development tools including workspace analysis, AST parsing, and code metrics",
      "category": "development",
      "version": "1.0.0",
      "capabilities": [
        "workspace_detection",
        "ast_analysis",
        "complexity_calculation",
        "dependency_analysis",
        "test_coverage"
      ]
    },
    {
      "name": "docker",
      "description": "Docker management tools for Go projects",
      "category": "devops",
      "version": "1.0.0",
      "capabilities": [
        "dockerfile_analysis",
        "container_management",
        "build_optimization",
        "security_scanning"
      ]
    },
    {
      "name": "postgres",
      "description": "PostgreSQL tools for database management",
      "category": "database",
      "version": "1.0.0",
      "capabilities": [
        "query_analysis",
        "migration_generation",
        "schema_analysis",
        "index_advisor",
        "performance_check"
      ]
    }
  ]
}
```

### 2. 執行工具

**端點**: `POST /api/tools/{toolName}/execute`

**Go 開發工具範例**:

```json
{
  "command": "analyze_workspace",
  "parameters": {
    "path": "/path/to/project",
    "deep": true
  }
}
```

**Docker 工具範例**:

```json
{
  "command": "analyze_dockerfile",
  "parameters": {
    "dockerfile_path": "./Dockerfile",
    "optimize": true
  }
}
```

**PostgreSQL 工具範例**:

```json
{
  "command": "analyze_query",
  "parameters": {
    "query": "SELECT * FROM users WHERE created_at > NOW() - INTERVAL '7 days'",
    "suggest_indexes": true
  }
}
```

### 3. 取得執行狀態

**端點**: `GET /api/tools/executions/{executionId}`

## 📊 記憶系統 API

### 1. 儲存記憶

**端點**: `POST /api/memory`

**請求格式**:

```json
{
  "key": "project_context",
  "value": {
    "project": "assistant-go",
    "type": "microservice",
    "technologies": ["go", "postgresql", "docker"]
  },
  "ttl": 3600 // 可選，秒為單位
}
```

### 2. 取得記憶

**端點**: `GET /api/memory/{key}`

### 3. 搜尋記憶

**端點**: `GET /api/memory/search?q={query}`

### 4. 刪除記憶

**端點**: `DELETE /api/memory/{key}`

## 👤 使用者系統 API

### 1. 使用者註冊

**端點**: `POST /api/auth/register`

**請求格式**:

```json
{
  "email": "user@example.com",
  "password": "secure_password",
  "name": "User Name"
}
```

### 2. 使用者登入

**端點**: `POST /api/auth/login`

**請求格式**:

```json
{
  "email": "user@example.com",
  "password": "secure_password"
}
```

**回應格式**:

```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 3600,
    "user": {
      "id": "user_123",
      "email": "user@example.com",
      "name": "User Name"
    }
  }
}
```

### 3. 重新整理令牌

**端點**: `POST /api/auth/refresh`

### 4. 登出

**端點**: `POST /api/auth/logout`

## 📈 系統監控 API

### 1. 健康檢查

**端點**: `GET /health`

**回應格式**:

```json
{
  "status": "healthy",
  "timestamp": "2025-01-06T10:00:00Z",
  "version": "0.1.0",
  "uptime": "72h15m30s",
  "services": {
    "database": "connected",
    "ai_claude": "healthy",
    "ai_gemini": "healthy",
    "memory": "healthy",
    "tools": "healthy"
  },
  "metrics": {
    "total_requests": 15420,
    "active_connections": 23,
    "memory_usage_mb": 128,
    "cpu_usage_percent": 15
  }
}
```

### 2. 系統指標

**端點**: `GET /api/metrics`

**功能**: 提供 Prometheus 格式的系統指標

## 🔒 認證和授權

### JWT 令牌系統

所有受保護的 API 端點都需要 JWT 令牌認證：

**請求標頭**:

```
Authorization: Bearer <jwt_token>
```

**令牌格式**:

```json
{
  "user_id": "user_123",
  "email": "user@example.com",
  "role": "user",
  "permissions": ["read", "write", "execute"],
  "exp": 1704538800,
  "iat": 1704535200
}
```

### 權限等級

- `admin` - 完整系統權限
- `user` - 標準使用者權限
- `guest` - 有限的唯讀權限

## ⚡ 效能和限制

### API 限制

| 資源           | 限制             |
| -------------- | ---------------- |
| 請求大小       | 最大 10MB        |
| 檔案上傳       | 最大 50MB        |
| WebSocket 連接 | 每使用者 5 個    |
| API 請求頻率   | 100 請求/分鐘    |
| 工具執行時間   | 最大 10 分鐘     |
| 對話長度       | 最多 1000 則訊息 |
| 記憶體儲存     | 每使用者 100MB   |

### 回應時間目標

- 簡單查詢: < 500ms
- 代碼分析: < 3s
- 工具執行: < 30s
- WebSocket 延遲: < 100ms

## 🚧 開發中的功能

### 即將推出 (Q1 2025)

1. **Kubernetes 工具集成**

   - K8s 資源管理
   - Manifest 分析和優化
   - 部署自動化

2. **增強的 Git 整合**

   - 版本控制操作
   - Commit 分析
   - Pull Request 自動化

3. **進階分析 API**

   - 代碼品質報告
   - 性能趨勢分析
   - 使用模式洞察

4. **協作功能**

   - 即時程式碼共享
   - 團隊工作區
   - 程式碼審查整合

5. **插件系統**
   - 第三方工具整合
   - 自定義工具開發
   - 市場生態系統

## 📝 錯誤處理規範

### 統一錯誤格式

```json
{
  "success": false,
  "error": {
    "code": "TOOL_EXECUTION_FAILED",
    "message": "工具執行失敗",
    "details": {
      "tool": "godev",
      "reason": "AST parsing error",
      "file": "main.go",
      "line": 42
    },
    "timestamp": "2025-01-06T10:00:00Z",
    "request_id": "req_abc123"
  }
}
```

### 錯誤代碼分類

- `AUTH_*` - 認證相關錯誤
- `VALIDATION_*` - 輸入驗證錯誤
- `TOOL_*` - 工具執行錯誤
- `AI_*` - AI 服務錯誤
- `DB_*` - 資料庫錯誤
- `SYSTEM_*` - 系統錯誤

## 🔧 開發者工具

### API 測試端點

**端點**: `GET /api/test`

**功能**: 提供互動式 API 測試介面

### API 文檔

**端點**: `GET /api/docs`

**功能**: 提供 OpenAPI/Swagger 文檔

### GraphQL 介面 (計劃中)

**端點**: `POST /graphql`

**功能**: 提供更靈活的查詢介面

---

**維護者**: Assistant 開發團隊
**文檔版本**: 2.0
**API 版本**: v2.0
**相容性**: v1 API 保持向後相容
