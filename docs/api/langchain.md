# LangChain API Documentation

This document describes the REST API endpoints for the LangChain integration in GoAssistant. All endpoints follow a consistent JSON response format and support the blue-dominant cyberpunk theme for the web interface.

## Base URL

All LangChain API endpoints are prefixed with `/api/langchain`.

## Response Format

All API responses follow this consistent format:

```json
{
  "success": true|false,
  "data": { ... },           // Present on success
  "error": {                 // Present on error
    "message": "Error description",
    "code": 400,
    "details": "Detailed error information"
  }
}
```

## Agent Endpoints

### GET /api/langchain/agents

Returns a list of available agent types.

**Response:**
```json
{
  "success": true,
  "data": {
    "agents": ["development", "database", "infrastructure", "research"],
    "count": 4
  }
}
```

### POST /api/langchain/agents/{type}/execute

Executes an agent of the specified type.

**Parameters:**
- `type` (path): Agent type (development, database, infrastructure, research)

**Request Body:**
```json
{
  "user_id": "user-123",
  "query": "Help me debug this Python code",
  "max_steps": 5,
  "context": {
    "language": "python",
    "framework": "django"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "agent_type": "development",
    "response": "I can help you debug your Python code...",
    "steps": [
      {
        "step": 1,
        "action": "analyze_code",
        "result": "Found potential issue in line 42"
      }
    ],
    "execution_time": 1250,
    "tokens_used": 150,
    "success": true,
    "error_message": null,
    "metadata": {
      "model_used": "claude-3",
      "confidence": 0.95
    }
  }
}
```

## Chain Endpoints

### GET /api/langchain/chains

Returns a list of available chain types.

**Response:**
```json
{
  "success": true,
  "data": {
    "chains": ["sequential", "parallel", "conditional", "rag"],
    "count": 4
  }
}
```

### POST /api/langchain/chains/{type}/execute

Executes a chain of the specified type.

**Parameters:**
- `type` (path): Chain type (sequential, parallel, conditional, rag)

**Request Body:**
```json
{
  "user_id": "user-123",
  "input": "Analyze this data and generate a report",
  "context": {
    "data_source": "database",
    "format": "markdown"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "chain_type": "sequential",
    "output": "# Data Analysis Report\n\nBased on the analysis...",
    "steps": [
      {
        "step": 1,
        "chain": "data_extraction",
        "result": "Extracted 1000 records"
      },
      {
        "step": 2,
        "chain": "analysis",
        "result": "Identified 3 key trends"
      }
    ],
    "execution_time": 2500,
    "tokens_used": 300,
    "success": true,
    "error_message": null,
    "metadata": {
      "chains_executed": 2,
      "total_records": 1000
    }
  }
}
```

## Memory Endpoints

### POST /api/langchain/memory

Stores a memory entry for later retrieval.

**Request Body:**
```json
{
  "user_id": "user-123",
  "type": "short_term",
  "session_id": "session-456",
  "content": "User prefers Python over JavaScript",
  "importance": 0.8,
  "expires_at": "2024-12-31T23:59:59Z",
  "metadata": {
    "category": "preference",
    "source": "conversation"
  }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Memory stored successfully"
}
```

### POST /api/langchain/memory/search

Searches for stored memories.

**Request Body:**
```json
{
  "user_id": "user-123",
  "query": "programming preferences",
  "types": ["short_term", "long_term"],
  "limit": 10,
  "threshold": 0.7,
  "metadata": {
    "category": "preference"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "results": [
      {
        "id": "mem-789",
        "content": "User prefers Python over JavaScript",
        "similarity": 0.95,
        "importance": 0.8,
        "created_at": "2024-01-15T10:30:00Z",
        "metadata": {
          "category": "preference",
          "source": "conversation"
        }
      }
    ],
    "count": 1
  }
}
```

### GET /api/langchain/memory/stats/{userID}

Returns memory usage statistics for a user.

**Parameters:**
- `userID` (path): User identifier

**Response:**
```json
{
  "success": true,
  "data": {
    "total_entries": 150,
    "by_type": {
      "short_term": 50,
      "long_term": 75,
      "tool": 20,
      "personalization": 5
    },
    "average_importance": 0.65,
    "oldest_entry": "2024-01-01T00:00:00Z",
    "newest_entry": "2024-01-15T15:30:00Z"
  }
}
```

## Service Endpoints

### GET /api/langchain/providers

Returns available LLM providers.

**Response:**
```json
{
  "success": true,
  "data": {
    "providers": ["claude-3", "gpt-4", "gemini-pro"],
    "count": 3
  }
}
```

### GET /api/langchain/health

Performs a health check on the LangChain service.

**Response:**
```json
{
  "success": true,
  "status": "healthy",
  "message": "LangChain service is operational"
}
```

## Error Responses

All endpoints may return error responses in the following format:

```json
{
  "success": false,
  "error": {
    "message": "Invalid request body",
    "code": 400,
    "details": "JSON decode error: unexpected end of JSON input"
  }
}
```

## Common HTTP Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request parameters or body
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable

## Rate Limiting

API endpoints may be rate-limited to prevent abuse. Rate limit headers will be included in responses:

- `X-RateLimit-Limit`: Maximum requests per time window
- `X-RateLimit-Remaining`: Remaining requests in current window
- `X-RateLimit-Reset`: Time when the rate limit resets

## Authentication

All endpoints require proper authentication. Include the authentication token in the request headers:

```
Authorization: Bearer <your-token>
```

## Cyberpunk Theme Integration

The API is designed to support a blue-dominant cyberpunk themed web interface:

- **Primary Colors**: CyberBlue (#0095FF), CyberGreen (#00FF88)
- **Background**: DarkBg (#121212)
- **Response format optimized for real-time updates and streaming
- **Metadata fields support theme customization
- **Error messages designed for cyberpunk UI styling
