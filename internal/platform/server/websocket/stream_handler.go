package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
	"github.com/koopa0/assistant-go/internal/assistant"
)

// StreamMessage represents a WebSocket message for streaming
type StreamMessage struct {
	Type      string                 `json:"type"`
	Content   string                 `json:"content,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// StreamHandler handles streaming responses over WebSocket
type StreamHandler struct {
	assistant *assistant.Assistant
	logger    *slog.Logger
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(assistant *assistant.Assistant, logger *slog.Logger) *StreamHandler {
	return &StreamHandler{
		assistant: assistant,
		logger:    logger,
	}
}

// HandleStream processes a query and streams the response over WebSocket
func (sh *StreamHandler) HandleStream(ctx context.Context, conn *websocket.Conn, query string) error {
	// Create streaming request
	request := &assistant.QueryRequest{
		Query: query,
	}

	// Get streaming response
	streamResp, err := sh.assistant.ProcessQueryStreamEnhanced(ctx, request)
	if err != nil {
		return sh.sendError(conn, fmt.Sprintf("Failed to start streaming: %v", err))
	}

	// Send start message
	if err := sh.sendMessage(conn, &StreamMessage{
		Type:      "start",
		Timestamp: nowMillis(),
	}); err != nil {
		return err
	}

	// Process the stream
	for {
		select {
		case text, ok := <-streamResp.TextChan:
			if !ok {
				// Stream ended
				return sh.sendMessage(conn, &StreamMessage{
					Type:      "end",
					Timestamp: nowMillis(),
				})
			}

			// Send text chunk
			if err := sh.sendMessage(conn, &StreamMessage{
				Type:      "chunk",
				Content:   text,
				Timestamp: nowMillis(),
			}); err != nil {
				return err
			}

		case event := <-streamResp.EventChan:
			// Send event
			if err := sh.sendMessage(conn, &StreamMessage{
				Type:      "event",
				Metadata:  event.Data,
				Timestamp: nowMillis(),
			}); err != nil {
				return err
			}

		case err := <-streamResp.ErrorChan:
			// Send error
			return sh.sendError(conn, err.Error())

		case <-streamResp.Done:
			// Complete
			return sh.sendMessage(conn, &StreamMessage{
				Type:      "complete",
				Timestamp: nowMillis(),
			})

		case <-ctx.Done():
			// Context cancelled
			return sh.sendError(conn, "Request cancelled")
		}
	}
}

// sendMessage sends a message over WebSocket
func (sh *StreamHandler) sendMessage(conn *websocket.Conn, msg *StreamMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		sh.logger.Error("Failed to marshal message", slog.Any("error", err))
		return err
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		sh.logger.Error("Failed to send message", slog.Any("error", err))
		return err
	}

	return nil
}

// sendError sends an error message
func (sh *StreamHandler) sendError(conn *websocket.Conn, errMsg string) error {
	return sh.sendMessage(conn, &StreamMessage{
		Type:      "error",
		Error:     errMsg,
		Timestamp: nowMillis(),
	})
}

// nowMillis returns current time in milliseconds
func nowMillis() int64 {
	return timeNow().UnixMilli()
}

// For testing
var timeNow = func() time.Time { return time.Now() }
