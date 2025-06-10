# Assistant API Documentation

**Version**: v1.0.0  
**Last Updated**: 2025-01-07  
**API Base URL**: `http://localhost:8100/api/v1`

## 📖 Overview

The Assistant API provides a RESTful interface for interacting with the intelligent development companion. All endpoints follow REST conventions and return JSON responses.

## 🔐 Authentication

The API supports two authentication methods:

### 1. JWT Bearer Token
```bash
curl -H "Authorization: Bearer <jwt-token>" \
  http://localhost:8100/api/v1/conversations
```

### 2. API Key
```bash
curl -H "X-API-Key: <api-key>" \
  http://localhost:8100/api/v1/conversations
```

## 📡 API Endpoints

### Health Check

#### `GET /health`
Check server health status.

**Response**:
```json
{
  "status": "healthy",
  "version": "v1.0.0",
  "timestamp": "2025-01-07T10:00:00Z"
}
```

---

### Conversations

#### `POST /api/v1/conversations`
Create a new conversation.

**Request Body**:
```json
{
  "title": "Code Review Session",
  "metadata": {
    "project": "assistant-go",
    "type": "code-review"
  }
}
```

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Code Review Session",
  "created_at": "2025-01-07T10:00:00Z",
  "updated_at": "2025-01-07T10:00:00Z",
  "metadata": {
    "project": "assistant-go",
    "type": "code-review"
  }
}
```

#### `GET /api/v1/conversations`
List all conversations with pagination.

**Query Parameters**:
- `limit` (int): Number of results per page (default: 20, max: 100)
- `offset` (int): Number of results to skip (default: 0)
- `sort` (string): Sort field (created_at, updated_at)
- `order` (string): Sort order (asc, desc)

**Response**:
```json
{
  "conversations": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Code Review Session",
      "created_at": "2025-01-07T10:00:00Z",
      "updated_at": "2025-01-07T10:00:00Z",
      "message_count": 15
    }
  ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

#### `GET /api/v1/conversations/{id}`
Get a specific conversation.

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "Code Review Session",
  "created_at": "2025-01-07T10:00:00Z",
  "updated_at": "2025-01-07T10:00:00Z",
  "messages": [
    {
      "id": "msg-123",
      "role": "user",
      "content": "Review this Go code",
      "created_at": "2025-01-07T10:00:00Z"
    }
  ],
  "metadata": {
    "project": "assistant-go",
    "type": "code-review"
  }
}
```

#### `DELETE /api/v1/conversations/{id}`
Delete a conversation.

**Response**:
```json
{
  "message": "Conversation deleted successfully"
}
```

---

### Chat

#### `POST /api/v1/chat`
Send a message to the assistant.

**Request Body**:
```json
{
  "conversation_id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "How do I implement error handling in Go?",
  "context": {
    "file_path": "/src/main.go",
    "line_number": 42
  },
  "tools": ["godev", "postgres"],
  "streaming": false
}
```

**Response (Non-streaming)**:
```json
{
  "message_id": "msg-124",
  "conversation_id": "550e8400-e29b-41d4-a716-446655440000",
  "content": "In Go, error handling follows the explicit approach...",
  "role": "assistant",
  "created_at": "2025-01-07T10:00:01Z",
  "tokens_used": {
    "prompt": 150,
    "completion": 250,
    "total": 400
  },
  "tools_used": ["godev"]
}
```

#### `POST /api/v1/chat/stream`
Send a message with Server-Sent Events streaming.

**Request**: Same as `/api/v1/chat` with `streaming: true`

**Response**: Server-Sent Events stream
```
event: message
data: {"delta": "In Go, ", "message_id": "msg-124"}

event: message
data: {"delta": "error handling ", "message_id": "msg-124"}

event: done
data: {"message_id": "msg-124", "tokens_used": {"total": 400}}
```

---

### Analytics

#### `GET /api/v1/analytics/activity`
Get activity analytics.

**Query Parameters**:
- `days` (int): Number of days to analyze (default: 30)

**Response**:
```json
{
  "period": "30 days",
  "total_events": 1250,
  "active_days": 28,
  "total_hours": 145.5,
  "average_per_day": 41.7,
  "peak_hours": [14, 15, 16],
  "activity_by_type": {
    "code_completion": 450,
    "refactoring": 230,
    "debugging": 180,
    "query_response": 390
  },
  "daily_activity": [
    {
      "date": "2025-01-07",
      "event_count": 45,
      "hours": 5.2,
      "intensity": 0.65,
      "main_type": "code_completion"
    }
  ],
  "weekly_pattern": {
    "monday": 0.95,
    "tuesday": 0.88,
    "wednesday": 0.92,
    "thursday": 0.87,
    "friday": 0.75,
    "saturday": 0.20,
    "sunday": 0.15
  }
}
```

#### `GET /api/v1/analytics/heatmap`
Get activity heatmap data.

**Response**:
```json
{
  "data": [
    {"x": 0, "y": 14, "value": 0.85},
    {"x": 1, "y": 14, "value": 0.78}
  ],
  "x_labels": ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"],
  "y_labels": ["00:00", "01:00", "02:00", "..."],
  "color_scale": {
    "min": 0,
    "max": 1,
    "colors": ["#f0f0f0", "#ffeda0", "#feb24c", "#fc4e2a", "#e31a1c"]
  },
  "statistics": {
    "most_active_time": "Wed 15:00",
    "least_active_time": "Sun 03:00",
    "weekly_coverage": 0.42,
    "consistency": 0.78
  }
}
```

#### `GET /api/v1/analytics/productivity`
Get productivity trends.

**Query Parameters**:
- `period` (string): Analysis period (week, month, quarter)

**Response**:
```json
{
  "period": "month",
  "data_points": [
    {
      "date": "2025-01-07",
      "productivity": 0.82,
      "hours": 6.5,
      "tasks": 42,
      "quality": 0.91
    }
  ],
  "trend_line": [0.75, 0.77, 0.79, 0.81, 0.82],
  "forecast": [
    {
      "date": "2025-01-14",
      "productivity": 0.84,
      "confidence": 0.85
    }
  ],
  "statistics": {
    "average_productivity": 0.79,
    "growth_rate": 0.12,
    "consistency": 0.83,
    "peak_performance": 0.92,
    "improvement_areas": ["morning_focus", "task_prioritization"]
  }
}
```

---

### Knowledge Graph

#### `GET /api/v1/knowledge/graph`
Get the knowledge graph.

**Query Parameters**:
- `node_type` (string): Filter by node type

**Response**:
```json
{
  "nodes": [
    {
      "id": "node-001",
      "name": "error-handling",
      "display_name": "Error Handling",
      "type": "concept",
      "importance": 0.92,
      "position": {"x": 100, "y": 200}
    }
  ],
  "edges": [
    {
      "id": "edge-001",
      "source": "node-001",
      "target": "node-002",
      "type": "related_to",
      "weight": 0.85
    }
  ],
  "statistics": {
    "total_nodes": 142,
    "total_edges": 256,
    "components": 3
  }
}
```

#### `POST /api/v1/knowledge/nodes`
Create a knowledge node.

**Request Body**:
```json
{
  "name": "concurrency-patterns",
  "display_name": "Concurrency Patterns",
  "description": "Common Go concurrency patterns",
  "type": "pattern",
  "properties": {
    "difficulty": "intermediate",
    "usefulness": "high"
  },
  "tags": ["go", "concurrency", "patterns"]
}
```

#### `GET /api/v1/knowledge/search`
Search knowledge nodes.

**Query Parameters**:
- `query` (string): Search query
- `limit` (int): Max results (default: 10)

---

### Learning

#### `GET /api/v1/learning/patterns`
List learned patterns.

**Query Parameters**:
- `type` (string): Pattern type filter
- `min_confidence` (float): Minimum confidence (0-1)

**Response**:
```json
{
  "patterns": [
    {
      "id": "pattern-001",
      "name": "error-wrapping",
      "type": "code-pattern",
      "description": "Consistent error wrapping with context",
      "confidence": 0.92,
      "frequency": 156,
      "examples": [
        "return fmt.Errorf(\"failed to connect: %w\", err)"
      ],
      "last_seen": "2025-01-07T09:30:00Z"
    }
  ]
}
```

#### `POST /api/v1/learning/events`
Record a learning event.

**Request Body**:
```json
{
  "event_type": "code_completion",
  "context": {
    "language": "go",
    "framework": "echo",
    "file_type": "handler"
  },
  "observation": {
    "action": "suggested_error_handling",
    "accepted": true,
    "modification": "added_context"
  },
  "metadata": {
    "session_id": "sess-123",
    "confidence": 0.85
  }
}
```

#### `GET /api/v1/learning/preferences`
Get user preferences.

**Response**:
```json
{
  "preferences": [
    {
      "key": "testing.framework",
      "value": "testify",
      "confidence": 0.95,
      "evidence_count": 23,
      "category": "testing"
    }
  ]
}
```

---

### Collaboration

#### `GET /api/v1/collaboration/agents`
List available agents.

**Query Parameters**:
- `type` (string): Agent type filter
- `status` (string): Status filter (active, inactive)

**Response**:
```json
{
  "agents": [
    {
      "id": "agent-dev",
      "name": "development",
      "display_name": "Development Specialist",
      "description": "Expert in code analysis and optimization",
      "type": "specialist",
      "capabilities": ["code-analysis", "refactoring", "testing"],
      "status": "active",
      "expertise_level": 0.92
    }
  ]
}
```

#### `GET /api/v1/collaboration/sessions/{id}`
Get collaboration session details.

**Response**:
```json
{
  "id": "session-001",
  "created_at": "2025-01-07T08:00:00Z",
  "status": "active",
  "goal": "Optimize database queries",
  "lead_agent": "agent-db",
  "participating_agents": ["agent-dev", "agent-perf"],
  "progress": 0.65,
  "plan": {
    "steps": [
      {
        "id": "step-001",
        "description": "Analyze current queries",
        "status": "completed",
        "assigned_to": "agent-db"
      }
    ]
  }
}
```

---

### Tools

#### `GET /api/v1/tools`
List available tools.

**Response**:
```json
{
  "tools": [
    {
      "name": "godev",
      "category": "development",
      "description": "Go development tools",
      "version": "1.0.0",
      "capabilities": [
        "workspace-analysis",
        "ast-parsing",
        "complexity-calculation",
        "dependency-analysis"
      ],
      "status": "active"
    }
  ]
}
```

#### `POST /api/v1/tools/{name}/execute`
Execute a tool.

**Request Body**:
```json
{
  "action": "analyze-complexity",
  "parameters": {
    "file_path": "/src/main.go",
    "threshold": 10
  }
}
```

**Response**:
```json
{
  "tool": "godev",
  "action": "analyze-complexity",
  "status": "success",
  "result": {
    "file": "/src/main.go",
    "functions": [
      {
        "name": "processData",
        "complexity": 12,
        "line": 45,
        "recommendation": "Consider breaking into smaller functions"
      }
    ],
    "average_complexity": 6.5
  },
  "execution_time_ms": 125
}
```

---

## 🔄 WebSocket API

### Connection
```javascript
const ws = new WebSocket('ws://localhost:8100/ws');
```

### Authentication
```javascript
ws.send(JSON.stringify({
  type: 'auth',
  token: 'your-jwt-token'
}));
```

### Message Types

#### Chat Message
```javascript
// Send
ws.send(JSON.stringify({
  type: 'chat',
  conversation_id: 'conv-123',
  message: 'Explain this code'
}));

// Receive
{
  type: 'chat_response',
  message_id: 'msg-125',
  content: 'This code implements...',
  timestamp: '2025-01-07T10:00:00Z'
}
```

#### Streaming Response
```javascript
// Receive multiple chunks
{
  type: 'chat_chunk',
  message_id: 'msg-125',
  delta: 'This code ',
  index: 0
}
```

---

## 🚨 Error Handling

All errors follow a consistent format:

```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Conversation not found",
    "details": {
      "resource": "conversation",
      "id": "invalid-id"
    }
  },
  "request_id": "req-12345",
  "timestamp": "2025-01-07T10:00:00Z"
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Malformed request |
| `UNAUTHORIZED` | 401 | Missing or invalid authentication |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `RESOURCE_NOT_FOUND` | 404 | Resource doesn't exist |
| `CONFLICT` | 409 | Resource conflict |
| `RATE_LIMITED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |
| `SERVICE_UNAVAILABLE` | 503 | Service temporarily unavailable |

---

## 🔒 Rate Limiting

Rate limits are applied per user/IP:

| Endpoint | Limit | Window |
|----------|-------|--------|
| `/api/v1/chat` | 100 requests | 1 minute |
| `/api/v1/chat/stream` | 50 requests | 1 minute |
| All other endpoints | 300 requests | 1 minute |

Rate limit headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704628800
```

---

## 📝 Best Practices

### 1. Use Pagination
Always use pagination for list endpoints:
```bash
GET /api/v1/conversations?limit=20&offset=40
```

### 2. Include Request IDs
For debugging, include a request ID:
```bash
curl -H "X-Request-ID: unique-id-123" ...
```

### 3. Handle Streaming Properly
For streaming responses, handle connection drops:
```javascript
const eventSource = new EventSource('/api/v1/chat/stream');
eventSource.onerror = (error) => {
  // Implement reconnection logic
};
```

### 4. Cache Responses
Use ETags for caching:
```bash
curl -H "If-None-Match: \"etag-value\"" ...
```

---

## 🔧 SDK Examples

### Go Client
```go
client := assistant.NewClient("http://localhost:8100", "api-key")

// Create conversation
conv, err := client.Conversations.Create(ctx, &assistant.CreateConversationRequest{
    Title: "New Session",
})

// Send message
resp, err := client.Chat.Send(ctx, &assistant.ChatRequest{
    ConversationID: conv.ID,
    Message: "Explain Go interfaces",
})
```

### Python Client
```python
from assistant import Client

client = Client(base_url="http://localhost:8100", api_key="api-key")

# Create conversation
conv = client.conversations.create(title="New Session")

# Send message
response = client.chat.send(
    conversation_id=conv.id,
    message="Explain Go interfaces"
)
```

### JavaScript/TypeScript Client
```typescript
import { AssistantClient } from '@assistant/client';

const client = new AssistantClient({
  baseURL: 'http://localhost:8100',
  apiKey: 'api-key'
});

// Create conversation
const conv = await client.conversations.create({
  title: 'New Session'
});

// Send message with streaming
const stream = await client.chat.stream({
  conversationId: conv.id,
  message: 'Explain Go interfaces'
});

for await (const chunk of stream) {
  console.log(chunk.delta);
}
```

---

## 📚 Additional Resources

- [OpenAPI Specification](/docs/api/openapi.yaml)
- [Postman Collection](/docs/api/postman.json)
- [API Changelog](/docs/api/CHANGELOG.md)
- [Migration Guide](/docs/api/MIGRATION.md)

---

**Need Help?** Contact support or check our [GitHub Issues](https://github.com/koopa0/assistant-go/issues)