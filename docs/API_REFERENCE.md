# GoAssistant API Reference

Complete documentation for GoAssistant's APIs, commands, and operations.

## Table of Contents

- [Overview](#overview)
- [Usage Modes](#usage-modes)
- [REST API](#rest-api)
- [CLI Commands](#cli-commands)
- [Configuration](#configuration)
- [Examples](#examples)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)

## Overview

GoAssistant provides three primary interaction modes:

1. **REST API** - HTTP endpoints for integration
2. **CLI Interactive** - Terminal-based interactive mode
3. **CLI Direct** - Single command execution

All modes support the same core functionality with different interfaces.

## Usage Modes

### 1. REST API Server Mode

Start the HTTP API server:

```bash
./bin/assistant serve
# or
./bin/assistant server
```

**Default Configuration:**

- Port: `:8080`
- Base URL: `http://localhost:8080`
- Content-Type: `application/json`

### 2. CLI Interactive Mode

Start interactive terminal session:

```bash
./bin/assistant cli
# or
./bin/assistant interactive
```

### 3. CLI Direct Mode

Execute single command:

```bash
./bin/assistant ask "your question here"
```

## REST API

### Base URL

```
http://localhost:8080/api
```

### Authentication

Currently in demo mode - no authentication required.
Future versions will support API keys and OAuth.

### Content Type

All API requests and responses use `application/json`.

---

## API Endpoints

### Health & Status

#### GET /api/health

Check service health status.

**Response:**

```json
{
  "status": "healthy",
  "timestamp": "2025-05-29T09:36:04Z"
}
```

**Status Codes:**

- `200` - Service healthy
- `503` - Service unhealthy

#### GET /api/status

Get detailed system status and statistics.

**Response:**

```json
{
  "status": "running",
  "timestamp": "2025-05-29T09:36:04Z",
  "stats": {
    "total_queries": 42,
    "average_response_time": "1.2s",
    "active_conversations": 3,
    "memory_usage": "128MB",
    "uptime": "2h30m"
  }
}
```

### Query & Conversation

#### POST /api/query

Send a query to the assistant.

**Request Body:**

```json
{
  "query": "Explain Go channels",
  "conversation_id": "optional-conv-id",
  "user_id": "optional-user-id",
  "provider": "claude", // optional: claude, gemini
  "model": "claude-3-haiku-20240307", // optional
  "tools": ["search", "golang"], // optional: specific tools to use
  "context": {
    // optional: additional context
    "project_type": "go",
    "file_path": "/path/to/file.go"
  }
}
```

**Response:**

```json
{
  "response": "Go channels are a fundamental concurrency primitive...",
  "conversation_id": "conv_abc123",
  "message_id": "msg_def456",
  "provider": "claude",
  "model": "claude-3-haiku-20240307",
  "tokens_used": 245,
  "execution_time": 1234567890,
  "context": {
    "conversation_message_count": 3,
    "processing_steps": ["validation", "context", "ai_generation", "storage"]
  },
  "suggestions": [
    // optional: follow-up suggestions
    "Learn about channel patterns",
    "See channel examples"
  ],
  "tools_used": [
    // optional: tools that were executed
    {
      "name": "golang_docs",
      "execution_time": 234,
      "success": true
    }
  ]
}
```

#### GET /api/conversations

List user conversations.

**Query Parameters:**

- `user_id` (string, optional) - Filter by user
- `limit` (int, optional, default: 50) - Max results
- `offset` (int, optional, default: 0) - Pagination offset

**Response:**

```json
{
  "conversations": [
    {
      "id": "conv_abc123",
      "title": "Go Programming Help",
      "created_at": "2025-05-29T09:00:00Z",
      "updated_at": "2025-05-29T09:15:00Z",
      "message_count": 5,
      "user_id": "user_xyz789"
    }
  ],
  "total": 1,
  "limit": 50,
  "offset": 0
}
```

#### GET /api/conversations/{id}

Get specific conversation details.

**Response:**

```json
{
  "id": "conv_abc123",
  "title": "Go Programming Help",
  "created_at": "2025-05-29T09:00:00Z",
  "updated_at": "2025-05-29T09:15:00Z",
  "user_id": "user_xyz789",
  "messages": [
    {
      "id": "msg_def456",
      "role": "user",
      "content": "Explain Go channels",
      "timestamp": "2025-05-29T09:00:00Z"
    },
    {
      "id": "msg_ghi789",
      "role": "assistant",
      "content": "Go channels are...",
      "timestamp": "2025-05-29T09:00:05Z",
      "metadata": {
        "provider": "claude",
        "model": "claude-3-haiku-20240307",
        "tokens_used": 245
      }
    }
  ]
}
```

#### DELETE /api/conversations/{id}

Delete a conversation.

**Response:**

- `204` - Successfully deleted
- `404` - Conversation not found

### Tools & Capabilities

#### GET /api/tools

List available tools and their capabilities.

**Response:**

```json
{
  "tools": [
    {
      "name": "golang",
      "description": "Go language development tools",
      "version": "1.0.0",
      "capabilities": ["code_analysis", "testing", "documentation"],
      "parameters": {
        "file_path": "string",
        "analysis_type": "enum[syntax,complexity,performance]"
      }
    },
    {
      "name": "docker",
      "description": "Docker container management",
      "version": "1.0.0",
      "capabilities": [
        "container_management",
        "image_building",
        "compose_operations"
      ]
    }
  ],
  "total": 2
}
```

#### GET /api/tools/{name}

Get detailed information about a specific tool.

**Response:**

```json
{
  "name": "golang",
  "description": "Go language development tools",
  "version": "1.0.0",
  "status": "available",
  "capabilities": ["code_analysis", "testing", "documentation", "benchmarking"],
  "parameters": {
    "file_path": {
      "type": "string",
      "required": true,
      "description": "Path to Go source file"
    },
    "analysis_type": {
      "type": "enum",
      "values": ["syntax", "complexity", "performance"],
      "default": "syntax",
      "description": "Type of analysis to perform"
    }
  },
  "examples": [
    {
      "description": "Analyze Go file syntax",
      "request": {
        "tool": "golang",
        "action": "analyze",
        "params": {
          "file_path": "./main.go",
          "analysis_type": "syntax"
        }
      }
    }
  ]
}
```

### Memory & Context

#### GET /api/memory/stats

Get memory system statistics.

**Response:**

```json
{
  "working_memory": {
    "total_items": 15,
    "capacity": 100,
    "utilization_rate": 0.15,
    "average_activation": 0.7
  },
  "episodic_memory": {
    "total_episodes": 42,
    "capacity": 1000,
    "utilization_rate": 0.042,
    "average_importance": 0.6
  },
  "semantic_memory": {
    "total_concepts": 128,
    "total_relationships": 256,
    "graph_nodes": 128,
    "graph_edges": 256
  },
  "procedural_memory": {
    "total_procedures": 8,
    "total_skills": 12,
    "total_workflows": 3,
    "average_success_rate": 0.85
  }
}
```

#### POST /api/memory/consolidate

Trigger memory consolidation across all memory types.

**Response:**

```json
{
  "status": "initiated",
  "consolidation_id": "cons_abc123",
  "estimated_completion": "2025-05-29T09:40:00Z"
}
```

---

## CLI Commands

### Interactive Mode Commands

When in CLI mode (`./bin/assistant cli`), the following commands are available:

#### Core Commands

- `help`, `?` - Show help message
- `exit`, `quit` - Exit the assistant
- `clear`, `cls` - Clear the screen
- `status` - Show system status
- `history` - Show command history

#### Tool Commands

- `tools` - List available tools
- `tools <name>` - Get tool details
- `sql <query>` - Execute SQL query (when database connected)
- `k8s <command>` - Execute Kubernetes command
- `docker <command>` - Execute Docker command

#### Memory Commands

- `memory stats` - Show memory statistics
- `memory consolidate` - Trigger memory consolidation
- `memory search <query>` - Search across all memory types

#### Configuration Commands

- `theme <dark|light>` - Change color theme
- `config show` - Show current configuration
- `config set <key> <value>` - Set configuration value

#### Conversation Commands

- `conversations` - List recent conversations
- `conversation <id>` - Show conversation details
- `conversation delete <id>` - Delete conversation

### Direct Mode

```bash
# Basic query
./bin/assistant ask "your question"

# With specific provider
CLAUDE_MODEL=claude-3-sonnet-20240229 ./bin/assistant ask "complex question"

# With context
./bin/assistant ask "analyze this Go code" --file ./main.go

# With tools
./bin/assistant ask "check Docker status" --tools docker
```

---

## Configuration

### Environment Variables

```bash
# Application
APP_MODE=development               # development, production
LOG_LEVEL=info                    # debug, info, warn, error
LOG_FORMAT=text                   # text, json

# Demo Mode
ASSISTANT_DEMO_MODE=true          # true/false - skip database

# Database
DATABASE_URL=postgres://...       # PostgreSQL connection string

# Server
SERVER_ADDRESS=:8080              # Server bind address
SERVER_READ_TIMEOUT=30s           # Request read timeout
SERVER_WRITE_TIMEOUT=30s          # Response write timeout

# AI Providers
CLAUDE_API_KEY=sk-ant-api03-...   # Claude API key
CLAUDE_MODEL=claude-3-haiku-20240307  # Default Claude model
GEMINI_API_KEY=...                # Gemini API key
GEMINI_MODEL=gemini-pro           # Default Gemini model

# Tools
SEARXNG_URL=http://localhost:8888 # SearXNG search endpoint
KUBECONFIG=/path/to/kubeconfig    # Kubernetes config file
DOCKER_HOST=unix:///var/run/docker.sock  # Docker daemon

# Cloudflare (optional)
CLOUDFLARE_API_TOKEN=...          # Cloudflare API token
CLOUDFLARE_ACCOUNT_ID=...         # Account ID
CLOUDFLARE_ZONE_ID=...            # Zone ID

# CLI
CLI_ENABLE_COLORS=true            # Enable colored output
CLI_HISTORY_SIZE=1000             # Command history size
```

### YAML Configuration

Alternative to environment variables, place in `configs/development.yaml`:

```yaml
mode: development
log_level: info
log_format: text

server:
  address: ":8080"
  read_timeout: "30s"
  write_timeout: "30s"
  enable_tls: false

database:
  url: "postgres://user:pass@localhost/assistant"
  max_connections: 10
  connection_timeout: "10s"

ai:
  default_provider: "claude"
  providers:
    claude:
      api_key: "${CLAUDE_API_KEY}"
      model: "claude-3-haiku-20240307"
      max_tokens: 4096
      temperature: 0.7
    gemini:
      api_key: "${GEMINI_API_KEY}"
      model: "gemini-pro"

tools:
  searxng:
    url: "http://localhost:8888"
  kubernetes:
    config_path: "${KUBECONFIG}"
  docker:
    host: "unix:///var/run/docker.sock"

cli:
  enable_colors: true
  history_size: 1000
```

---

## Examples

### Basic Query via API

```bash
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Write a Go HTTP server that handles JSON requests"
  }'
```

### Query with Context

```bash
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Optimize this code for performance",
    "context": {
      "project_type": "go",
      "file_path": "./main.go",
      "current_code": "package main\n\nfunc main() {...}"
    },
    "tools": ["golang", "profiler"]
  }'
```

### Conversation Management

```bash
# Start conversation
CONV_ID=$(curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{"query": "Help me build a REST API"}' \
  | jq -r '.conversation_id')

# Continue conversation
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d "{
    \"query\": \"Add authentication to the API\",
    \"conversation_id\": \"$CONV_ID\"
  }"

# Get conversation history
curl http://localhost:8080/api/conversations/$CONV_ID
```

### Tool Integration

```bash
# List available tools
curl http://localhost:8080/api/tools

# Get tool details
curl http://localhost:8080/api/tools/golang

# Use specific tools in query
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Analyze my Docker setup and suggest improvements",
    "tools": ["docker", "security_scanner"],
    "context": {
      "docker_file": "./Dockerfile",
      "compose_file": "./docker-compose.yml"
    }
  }'
```

### Memory Operations

```bash
# Get memory statistics
curl http://localhost:8080/api/memory/stats

# Trigger consolidation
curl -X POST http://localhost:8080/api/memory/consolidate

# Search memory
curl -X POST http://localhost:8080/api/memory/search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Docker best practices",
    "memory_types": ["semantic", "procedural"],
    "limit": 10
  }'
```

---

## Error Handling

### HTTP Status Codes

- `200` - Success
- `400` - Bad Request (invalid input)
- `401` - Unauthorized (invalid API key)
- `404` - Not Found (resource doesn't exist)
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error
- `503` - Service Unavailable (system unhealthy)

### Error Response Format

```json
{
  "error": {
    "code": "INVALID_INPUT",
    "message": "Query cannot be empty",
    "details": {
      "field": "query",
      "received": "",
      "expected": "non-empty string"
    },
    "request_id": "req_abc123"
  }
}
```

### Common Error Codes

- `INVALID_INPUT` - Request validation failed
- `RATE_LIMITED` - Too many requests
- `UNAUTHORIZED` - Authentication failed
- `TIMEOUT` - Request timed out
- `PROVIDER_ERROR` - AI provider error
- `TOOL_ERROR` - Tool execution failed
- `CONTEXT_NOT_FOUND` - Conversation/context not found

---

## Rate Limiting

### Default Limits

- **API Queries**: 100 requests per minute per IP
- **Tool Executions**: 20 per minute per IP
- **Memory Operations**: 10 per minute per IP

### Rate Limit Headers

Response includes rate limit information:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640995200
```

### Handling Rate Limits

When rate limited (429 status), implement exponential backoff:

```javascript
async function queryWithRetry(query, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await fetch("/api/query", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query }),
      });

      if (response.status === 429) {
        const delay = Math.pow(2, i) * 1000; // Exponential backoff
        await new Promise((resolve) => setTimeout(resolve, delay));
        continue;
      }

      return await response.json();
    } catch (error) {
      if (i === maxRetries - 1) throw error;
    }
  }
}
```

---

## SDK and Libraries

### JavaScript/Node.js

```javascript
const { GoAssistant } = require("@koopa0/goassistant-js");

const assistant = new GoAssistant({
  baseURL: "http://localhost:8080",
  apiKey: "your-api-key", // when authentication is implemented
});

// Simple query
const response = await assistant.query("Explain Go interfaces");

// Query with context
const response = await assistant.query("Optimize this code", {
  context: {
    language: "go",
    file_path: "./main.go",
  },
  tools: ["golang", "profiler"],
});

// Conversation management
const conversation = await assistant.startConversation();
await conversation.send("Help me build a REST API");
await conversation.send("Add authentication");
const history = await conversation.getHistory();
```

### Python

```python
from goassistant import GoAssistant

assistant = GoAssistant(base_url='http://localhost:8080')

# Simple query
response = assistant.query('Explain Go interfaces')

# Query with context
response = assistant.query(
    'Optimize this code',
    context={'language': 'go', 'file_path': './main.go'},
    tools=['golang', 'profiler']
)

# Conversation management
conversation = assistant.start_conversation()
conversation.send('Help me build a REST API')
conversation.send('Add authentication')
history = conversation.get_history()
```

### Go

```go
package main

import (
    "github.com/koopa0/goassistant-go"
)

func main() {
    client := goassistant.NewClient("http://localhost:8080")

    // Simple query
    response, err := client.Query(ctx, &goassistant.QueryRequest{
        Query: "Explain Go interfaces",
    })

    // Query with context
    response, err := client.Query(ctx, &goassistant.QueryRequest{
        Query: "Optimize this code",
        Context: map[string]interface{}{
            "language": "go",
            "file_path": "./main.go",
        },
        Tools: []string{"golang", "profiler"},
    })

    // Conversation management
    conv, err := client.StartConversation(ctx)
    err = conv.Send(ctx, "Help me build a REST API")
    err = conv.Send(ctx, "Add authentication")
    history, err := conv.GetHistory(ctx)
}
```

---

## WebSocket API (Coming Soon)

Real-time communication for streaming responses and live collaboration.

```javascript
const ws = new WebSocket("ws://localhost:8080/ws");

ws.send(
  JSON.stringify({
    type: "query",
    data: { query: "Explain Go channels step by step" },
  }),
);

ws.onmessage = (event) => {
  const { type, data } = JSON.parse(event.data);
  if (type === "response_chunk") {
    // Stream response as it's generated
    console.log(data.content);
  }
};
```

---

For more examples and advanced usage, see the [examples](../examples/) directory.
