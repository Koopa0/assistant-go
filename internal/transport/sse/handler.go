package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/koopa0/assistant-go/internal/assistant"
)

// Handler handles Server-Sent Events streaming
type Handler struct {
	assistant *assistant.Assistant
	logger    *slog.Logger
}

// NewHandler creates a new SSE handler
func NewHandler(assistant *assistant.Assistant, logger *slog.Logger) *Handler {
	return &Handler{
		assistant: assistant,
		logger:    logger,
	}
}

// RegisterRoutes registers SSE routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/stream", h.handleStream)
	mux.HandleFunc("GET /api/v1/stream", h.handleStreamGET)
	mux.HandleFunc("GET /api/v1/stream/test", h.handleTestStream)
}

// handleStream handles SSE streaming requests
func (h *Handler) handleStream(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse request
	var request StreamRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.sendError(w, "Invalid request", err)
		return
	}

	// Create query request
	queryReq := &assistant.QueryRequest{
		Query:          request.Query,
		ConversationID: request.ConversationID,
		Provider:       request.Provider,
		Model:          request.Model,
		Context:        request.Context,
	}

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.sendError(w, "Streaming not supported", nil)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	// Get streaming response
	streamResp, err := h.assistant.ProcessQueryStreamEnhanced(ctx, queryReq)
	if err != nil {
		h.sendError(w, "Failed to start streaming", err)
		return
	}

	// Send initial event
	h.sendEvent(w, flusher, Event{
		Type: "start",
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	})

	// Process stream
	for {
		select {
		case text, ok := <-streamResp.TextChan:
			if !ok {
				// Stream ended
				h.sendEvent(w, flusher, Event{
					Type: "end",
					Data: map[string]interface{}{
						"timestamp": time.Now().Unix(),
					},
				})
				return
			}

			// Send text chunk
			h.sendEvent(w, flusher, Event{
				Type: "message",
				Data: map[string]interface{}{
					"content": text,
				},
			})

		case event := <-streamResp.EventChan:
			// Send metadata events
			if event.Type == "complete" {
				h.sendEvent(w, flusher, Event{
					Type: "complete",
					Data: event.Data,
				})
			}

		case err := <-streamResp.ErrorChan:
			h.sendError(w, "Streaming error", err)
			return

		case <-streamResp.Done:
			return

		case <-ctx.Done():
			h.sendError(w, "Request timeout", ctx.Err())
			return
		}
	}
}

// handleStreamGET handles GET requests for SSE streaming
func (h *Handler) handleStreamGET(w http.ResponseWriter, r *http.Request) {
	// Get query parameter
	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "Query parameter is required", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create request
	request := StreamRequest{
		Query: query,
	}

	// Create query request
	queryReq := &assistant.QueryRequest{
		Query:          request.Query,
		ConversationID: request.ConversationID,
		Provider:       request.Provider,
		Model:          request.Model,
		Context:        request.Context,
	}

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		h.sendError(w, "Streaming not supported", nil)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	// Get streaming response
	streamResp, err := h.assistant.ProcessQueryStreamEnhanced(ctx, queryReq)
	if err != nil {
		h.sendError(w, "Failed to start streaming", err)
		return
	}

	// Send initial event
	h.sendEvent(w, flusher, Event{
		Type: "start",
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	})

	// Process stream
	for {
		select {
		case text, ok := <-streamResp.TextChan:
			if !ok {
				// Stream ended
				h.sendEvent(w, flusher, Event{
					Type: "end",
					Data: map[string]interface{}{
						"timestamp": time.Now().Unix(),
					},
				})
				return
			}

			// Send text chunk
			h.sendEvent(w, flusher, Event{
				Type: "message",
				Data: map[string]interface{}{
					"content": text,
				},
			})

		case event := <-streamResp.EventChan:
			// Send metadata events
			if event.Type == "complete" {
				h.sendEvent(w, flusher, Event{
					Type: "complete",
					Data: event.Data,
				})
			}

		case err := <-streamResp.ErrorChan:
			h.sendError(w, "Streaming error", err)
			return

		case <-streamResp.Done:
			return

		case <-ctx.Done():
			h.sendError(w, "Request timeout", ctx.Err())
			return
		}
	}
}

// handleTestStream provides a test endpoint for SSE streaming
func (h *Handler) handleTestStream(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send test events
	for i := 0; i < 10; i++ {
		h.sendEvent(w, flusher, Event{
			Type: "message",
			Data: map[string]interface{}{
				"content": fmt.Sprintf("Test message %d ", i+1),
			},
		})
		time.Sleep(500 * time.Millisecond)
	}

	h.sendEvent(w, flusher, Event{
		Type: "end",
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
		},
	})
}

// sendEvent sends an SSE event
func (h *Handler) sendEvent(w http.ResponseWriter, flusher http.Flusher, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("Failed to marshal event", slog.Any("error", err))
		return
	}

	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "data: %s\n\n", string(data))
	flusher.Flush()
}

// sendError sends an error event
func (h *Handler) sendError(w http.ResponseWriter, message string, err error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	errorData := map[string]interface{}{
		"message": message,
	}
	if err != nil {
		errorData["error"] = err.Error()
	}

	h.sendEvent(w, flusher, Event{
		Type: "error",
		Data: errorData,
	})
}

// Event represents an SSE event
type Event struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// StreamRequest represents a streaming request
type StreamRequest struct {
	Query          string                 `json:"query"`
	ConversationID *string                `json:"conversation_id,omitempty"`
	Provider       *string                `json:"provider,omitempty"`
	Model          *string                `json:"model,omitempty"`
	Context        map[string]interface{} `json:"context,omitempty"`
}
