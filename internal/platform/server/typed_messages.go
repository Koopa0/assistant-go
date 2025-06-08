package server

import (
	"encoding/json"
	"fmt"
	"time"
)

// TypedWebSocketMessage provides type-safe WebSocket message handling
type TypedWebSocketMessage struct {
	Type      string    `json:"type"`
	ID        string    `json:"id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	// Payload is unmarshaled based on Type
}

// WebSocket message types
const (
	WSTypeChat      = "chat"
	WSTypeTool      = "tool"
	WSTypeError     = "error"
	WSTypeStatus    = "status"
	WSTypeHeartbeat = "heartbeat"
)

// ChatPayload represents a chat message payload
type ChatPayload struct {
	ConversationID string  `json:"conversation_id,omitempty"`
	Message        string  `json:"message"`
	Model          string  `json:"model,omitempty"`
	Temperature    float64 `json:"temperature,omitempty"`
	MaxTokens      int     `json:"max_tokens,omitempty"`
}

// ToolPayload represents a tool execution payload
type ToolPayload struct {
	ToolName   string                 `json:"tool_name"`
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters"` // Tool-specific params
}

// StatusPayload represents a status update payload
type StatusPayload struct {
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
	Progress    int    `json:"progress,omitempty"` // 0-100
	IsCompleted bool   `json:"is_completed"`
}

// ErrorPayload represents an error payload
type ErrorPayload struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// HeartbeatPayload represents a heartbeat payload
type HeartbeatPayload struct {
	Sequence int64 `json:"sequence"`
}

// ParseWebSocketMessage parses a raw WebSocket message into typed message
func ParseWebSocketMessage(data []byte) (*TypedWebSocketMessage, interface{}, error) {
	var base TypedWebSocketMessage
	if err := json.Unmarshal(data, &base); err != nil {
		return nil, nil, fmt.Errorf("failed to parse base message: %w", err)
	}

	// Extract raw payload
	var raw struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("failed to extract payload: %w", err)
	}

	// Parse payload based on type
	var payload interface{}
	var err error

	switch base.Type {
	case WSTypeChat:
		var p ChatPayload
		err = json.Unmarshal(raw.Payload, &p)
		payload = p
	case WSTypeTool:
		var p ToolPayload
		err = json.Unmarshal(raw.Payload, &p)
		payload = p
	case WSTypeError:
		var p ErrorPayload
		err = json.Unmarshal(raw.Payload, &p)
		payload = p
	case WSTypeStatus:
		var p StatusPayload
		err = json.Unmarshal(raw.Payload, &p)
		payload = p
	case WSTypeHeartbeat:
		var p HeartbeatPayload
		err = json.Unmarshal(raw.Payload, &p)
		payload = p
	default:
		// Unknown type, keep as raw
		payload = raw.Payload
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse payload for type %s: %w", base.Type, err)
	}

	return &base, payload, nil
}

// CreateWebSocketMessage creates a typed WebSocket message
func CreateWebSocketMessage(msgType string, id string, payload interface{}) (*WebSocketMessage, error) {
	return &WebSocketMessage{
		Type:      msgType,
		ID:        id,
		Payload:   payload,
		Timestamp: time.Now(),
	}, nil
}

// TypedStreamResponse provides type-safe streaming response
type TypedStreamResponse struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	// Content varies by type
}

// Stream response types
const (
	StreamTypeContent = "content"
	StreamTypeError   = "error"
	StreamTypeDone    = "done"
	StreamTypeDebug   = "debug"
	StreamTypeUsage   = "usage"
)

// StreamContent represents content in a stream
type StreamContent struct {
	Text  string `json:"text"`
	Delta string `json:"delta,omitempty"` // For incremental updates
}

// StreamUsage represents token usage information
type StreamUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamDebug represents debug information
type StreamDebug struct {
	Message   string            `json:"message"`
	Data      map[string]string `json:"data,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// CreateStreamResponse creates a typed stream response
func CreateStreamResponse(id string, respType string, content interface{}) (*StreamResponse, error) {
	resp := &StreamResponse{
		ID:        id,
		Type:      respType,
		Timestamp: time.Now(),
	}

	switch respType {
	case StreamTypeContent:
		if c, ok := content.(StreamContent); ok {
			resp.Content = c.Text
			resp.Metadata = map[string]interface{}{
				"delta": c.Delta,
			}
		} else if text, ok := content.(string); ok {
			resp.Content = text
		} else {
			return nil, fmt.Errorf("invalid content type for stream content")
		}

	case StreamTypeError:
		if err, ok := content.(*ErrorDetail); ok {
			resp.Error = err
		} else if errStr, ok := content.(string); ok {
			resp.Error = &ErrorDetail{
				Code:    "stream_error",
				Message: errStr,
			}
		} else {
			return nil, fmt.Errorf("invalid content type for stream error")
		}

	case StreamTypeUsage:
		resp.Metadata = content

	case StreamTypeDebug:
		resp.Metadata = content

	case StreamTypeDone:
		// No content needed

	default:
		return nil, fmt.Errorf("unknown stream type: %s", respType)
	}

	return resp, nil
}

// TypedErrorDetails provides structured error details
type TypedErrorDetails struct {
	RequestID  string `json:"request_id,omitempty"`
	UserID     string `json:"user_id,omitempty"`
	Resource   string `json:"resource,omitempty"`
	Operation  string `json:"operation,omitempty"`
	RetryAfter int    `json:"retry_after,omitempty"` // Seconds
	HelpURL    string `json:"help_url,omitempty"`
}

// CreateErrorResponse creates a typed error response
func CreateErrorResponse(code, message string, details *TypedErrorDetails, req *RequestInfo) *ErrorResponse {
	detailsMap := make(map[string]interface{})

	if details != nil {
		if details.RequestID != "" {
			detailsMap["request_id"] = details.RequestID
		}
		if details.UserID != "" {
			detailsMap["user_id"] = details.UserID
		}
		if details.Resource != "" {
			detailsMap["resource"] = details.Resource
		}
		if details.Operation != "" {
			detailsMap["operation"] = details.Operation
		}
		if details.RetryAfter > 0 {
			detailsMap["retry_after"] = details.RetryAfter
		}
		if details.HelpURL != "" {
			detailsMap["help_url"] = details.HelpURL
		}
	}

	return &ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: detailsMap,
		},
		Request: *req,
	}
}
