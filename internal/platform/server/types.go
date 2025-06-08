package server

import (
	"time"

	"github.com/koopa0/assistant-go/internal/assistant"
)

// StatusResponse represents the response for the /status endpoint
type StatusResponse struct {
	Status    string                    `json:"status"`
	Timestamp string                    `json:"timestamp"`
	Stats     *assistant.AssistantStats `json:"stats"`
}

// HealthResponse represents the response for the /health endpoint
type HealthResponse struct {
	Status    string    `json:"status"`
	Version   string    `json:"version,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// APIInfoResponse represents the response for the root endpoint
type APIInfoResponse struct {
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Description   string            `json:"description"`
	Endpoints     EndpointsInfo     `json:"endpoints"`
	Documentation DocumentationInfo `json:"documentation"`
	Features      []string          `json:"features"`
	Timestamp     string            `json:"timestamp"`
}

// EndpointsInfo contains information about available endpoints
type EndpointsInfo struct {
	Health string      `json:"health"`
	Status string      `json:"status"`
	V1     V1Endpoints `json:"v1"`
}

// V1Endpoints contains v1 API endpoints
type V1Endpoints struct {
	Chat          string                 `json:"chat"`
	Memory        MemoryEndpoints        `json:"memory"`
	Tools         ToolsEndpoints         `json:"tools"`
	Conversations ConversationEndpoints  `json:"conversations"`
	Analytics     AnalyticsEndpoints     `json:"analytics"`
	Knowledge     KnowledgeEndpoints     `json:"knowledge"`
	Learning      LearningEndpoints      `json:"learning"`
	Collaboration CollaborationEndpoints `json:"collaboration"`
}

// MemoryEndpoints contains memory-related endpoints
type MemoryEndpoints struct {
	Store    string `json:"store"`
	Retrieve string `json:"retrieve"`
	Search   string `json:"search"`
}

// ToolsEndpoints contains tool-related endpoints
type ToolsEndpoints struct {
	List     string `json:"list"`
	Execute  string `json:"execute"`
	Register string `json:"register"`
}

// ConversationEndpoints contains conversation-related endpoints
type ConversationEndpoints struct {
	Create string `json:"create"`
	Get    string `json:"get"`
	List   string `json:"list"`
	Delete string `json:"delete"`
}

// AnalyticsEndpoints contains analytics-related endpoints
type AnalyticsEndpoints struct {
	Activity     string `json:"activity"`
	Heatmap      string `json:"heatmap"`
	Productivity string `json:"productivity"`
	Dashboard    string `json:"dashboard"`
}

// KnowledgeEndpoints contains knowledge graph endpoints
type KnowledgeEndpoints struct {
	Graph           string `json:"graph"`
	Nodes           string `json:"nodes"`
	Search          string `json:"search"`
	Paths           string `json:"paths"`
	Clusters        string `json:"clusters"`
	Recommendations string `json:"recommendations"`
}

// LearningEndpoints contains learning-related endpoints
type LearningEndpoints struct {
	Patterns      string `json:"patterns"`
	Preferences   string `json:"preferences"`
	Events        string `json:"events"`
	Predictions   string `json:"predictions"`
	Report        string `json:"report"`
	Reinforcement string `json:"reinforcement"`
}

// CollaborationEndpoints contains collaboration endpoints
type CollaborationEndpoints struct {
	Agents    string `json:"agents"`
	Sessions  string `json:"sessions"`
	Knowledge string `json:"knowledge"`
}

// DocumentationInfo contains API documentation information
type DocumentationInfo struct {
	OpenAPI  string `json:"openapi"`
	Postman  string `json:"postman"`
	Tutorial string `json:"tutorial"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   ErrorDetail `json:"error"`
	Request RequestInfo `json:"request,omitempty"`
}

// ErrorDetail contains detailed error information
type ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// RequestInfo contains information about the failed request
type RequestInfo struct {
	ID        string `json:"id,omitempty"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Timestamp string `json:"timestamp"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	ID        string      `json:"id,omitempty"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

// StreamResponse represents a streaming response chunk
type StreamResponse struct {
	ID        string       `json:"id"`
	Type      string       `json:"type"` // "content", "error", "done"
	Content   string       `json:"content,omitempty"`
	Error     *ErrorDetail `json:"error,omitempty"`
	Metadata  interface{}  `json:"metadata,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}
