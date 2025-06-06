package websocket

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/koopa0/assistant-go/internal/assistant"
)

// StreamEndpoint handles WebSocket streaming connections
type StreamEndpoint struct {
	handler  *StreamHandler
	upgrader websocket.Upgrader
	logger   *slog.Logger
}

// NewStreamEndpoint creates a new streaming endpoint
func NewStreamEndpoint(assistant *assistant.Assistant, logger *slog.Logger) *StreamEndpoint {
	return &StreamEndpoint{
		handler: NewStreamHandler(assistant, logger),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking for production
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		logger: logger,
	}
}

// ServeHTTP handles WebSocket upgrade and streaming
func (e *StreamEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection to WebSocket
	conn, err := e.upgrader.Upgrade(w, r, nil)
	if err != nil {
		e.logger.Error("Failed to upgrade connection", slog.Any("error", err))
		return
	}
	defer conn.Close()

	// Set read/write deadlines
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(60 * time.Second))

	// Handle pings to keep connection alive
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Read messages and stream responses
	for {
		// Read message
		var req StreamRequest
		err := conn.ReadJSON(&req)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				e.logger.Error("WebSocket error", slog.Any("error", err))
			}
			break
		}

		// Reset read deadline
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// Handle the streaming request
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		err = e.handler.HandleStream(ctx, conn, req.Query)
		cancel()

		if err != nil {
			e.logger.Error("Stream handling error",
				slog.String("query", req.Query),
				slog.Any("error", err))
		}
	}
}

// StreamRequest represents a WebSocket streaming request
type StreamRequest struct {
	Query          string                 `json:"query"`
	ConversationID *string                `json:"conversation_id,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
}
