package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// StreamingResponse represents a streaming response from Claude API
type StreamingResponse struct {
	reader    *bufio.Reader
	response  *http.Response
	dataChan  chan StreamEvent
	errorChan chan error
	done      chan struct{}
}

// StreamEvent represents a single event in the SSE stream
type StreamEvent struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message,omitempty"`
	Delta   *ContentDelta   `json:"delta,omitempty"`
	Usage   *StreamUsage    `json:"usage,omitempty"`
}

// ContentDelta represents incremental content in streaming
type ContentDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// StreamUsage represents token usage information in streaming
type StreamUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// GenerateResponseStream sends a streaming request to Claude API
func (c *Client) GenerateResponseStream(ctx context.Context, request *GenerateRequest) (*StreamingResponse, error) {
	// Build the API request
	apiReq := &apiRequest{
		Model:       request.Model,
		Messages:    convertToAPIMessages(request.Messages),
		MaxTokens:   request.MaxTokens,
		Temperature: request.Temperature,
		Stream:      true, // Enable streaming
	}

	if request.SystemPrompt != nil && *request.SystemPrompt != "" {
		apiReq.System = *request.SystemPrompt
	}

	// Marshal request body
	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Create streaming response
	streamResp := &StreamingResponse{
		reader:    bufio.NewReader(resp.Body),
		response:  resp,
		dataChan:  make(chan StreamEvent, 100),
		errorChan: make(chan error, 1),
		done:      make(chan struct{}),
	}

	// Start processing SSE stream
	go streamResp.processStream()

	return streamResp, nil
}

// processStream processes the SSE stream
func (s *StreamingResponse) processStream() {
	defer close(s.dataChan)
	defer close(s.done)
	defer s.response.Body.Close()

	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				s.errorChan <- fmt.Errorf("error reading stream: %w", err)
			}
			return
		}

		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse SSE format
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			// Handle [DONE] signal
			if data == "[DONE]" {
				return
			}

			// Parse JSON data
			var event StreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				s.errorChan <- fmt.Errorf("error parsing event: %w", err)
				continue
			}

			// Send event to channel
			select {
			case s.dataChan <- event:
			case <-time.After(30 * time.Second):
				s.errorChan <- fmt.Errorf("timeout sending event")
				return
			}
		}
	}
}

// Events returns the channel for receiving stream events
func (s *StreamingResponse) Events() <-chan StreamEvent {
	return s.dataChan
}

// Errors returns the channel for receiving errors
func (s *StreamingResponse) Errors() <-chan error {
	return s.errorChan
}

// Done returns the channel that's closed when streaming is complete
func (s *StreamingResponse) Done() <-chan struct{} {
	return s.done
}

// Close closes the streaming response
func (s *StreamingResponse) Close() error {
	if s.response != nil && s.response.Body != nil {
		return s.response.Body.Close()
	}
	return nil
}

// apiRequest represents the Claude API request format
type apiRequest struct {
	Model       string       `json:"model"`
	Messages    []apiMessage `json:"messages"`
	MaxTokens   int          `json:"max_tokens"`
	Temperature float64      `json:"temperature,omitempty"`
	System      string       `json:"system,omitempty"`
	Stream      bool         `json:"stream,omitempty"`
}

// apiMessage represents a message in the API format
type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// convertToAPIMessages converts our messages to API format
func convertToAPIMessages(messages []Message) []apiMessage {
	apiMessages := make([]apiMessage, len(messages))
	for i, msg := range messages {
		apiMessages[i] = apiMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return apiMessages
}
