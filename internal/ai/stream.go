package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/ai/claude"
)

// StreamChunk represents a chunk of streaming response
type StreamChunk struct {
	Content      string                 `json:"content"`
	FinishReason string                 `json:"finish_reason,omitempty"`
	TokensUsed   *TokenUsage            `json:"tokens_used,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Error        error                  `json:"error,omitempty"`
}

// StreamResponse represents a streaming response
type StreamResponse struct {
	ChunkChan <-chan StreamChunk
	Done      <-chan struct{}
}

// GenerateStreamRequest represents a request for streaming generation
type GenerateStreamRequest struct {
	Messages     []Message              `json:"messages"`
	MaxTokens    int                    `json:"max_tokens,omitempty"`
	Temperature  float64                `json:"temperature,omitempty"`
	Model        string                 `json:"model,omitempty"`
	SystemPrompt *string                `json:"system_prompt,omitempty"`
	Tools        []Tool                 `json:"tools,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// StreamCallback is a callback function for streaming responses
type StreamCallback func(chunk StreamChunk) error

// GenerateResponseStream generates a streaming response
func (s *Service) GenerateResponseStream(ctx context.Context, request *GenerateStreamRequest, providerName ...string) (*StreamResponse, error) {
	provider := s.defaultProvider
	if len(providerName) > 0 && providerName[0] != "" {
		provider = providerName[0]
	}

	// Create channels for streaming
	chunkChan := make(chan StreamChunk, 100)
	doneChan := make(chan struct{})

	// Start streaming in goroutine
	go func() {
		defer close(chunkChan)
		defer close(doneChan)

		switch provider {
		case "claude":
			if s.claudeClient == nil {
				chunkChan <- StreamChunk{
					Error: NewProviderError(ErrorTypeInvalidRequest, "Claude provider not available", "claude"),
				}
				return
			}
			s.streamFromClaude(ctx, request, chunkChan)

		case "gemini":
			if s.geminiClient == nil {
				chunkChan <- StreamChunk{
					Error: NewProviderError(ErrorTypeInvalidRequest, "Gemini provider not available", "gemini"),
				}
				return
			}
			s.streamFromGemini(ctx, request, chunkChan)

		default:
			chunkChan <- StreamChunk{
				Error: NewProviderError(ErrorTypeInvalidRequest, "unknown provider: "+provider, provider),
			}
		}
	}()

	return &StreamResponse{
		ChunkChan: chunkChan,
		Done:      doneChan,
	}, nil
}

// streamFromClaude handles real streaming from Claude API
func (s *Service) streamFromClaude(ctx context.Context, request *GenerateStreamRequest, chunkChan chan<- StreamChunk) {
	// Convert to Claude request
	claudeReq := &claude.GenerateRequest{
		Messages:     convertMessagesToClaude(request.Messages),
		MaxTokens:    request.MaxTokens,
		Temperature:  request.Temperature,
		Model:        request.Model,
		SystemPrompt: request.SystemPrompt,
		Metadata:     request.Metadata,
	}

	// Start timing
	startTime := time.Now()

	// Get streaming response from Claude
	streamResp, err := s.claudeClient.GenerateResponseStream(ctx, claudeReq)
	if err != nil {
		chunkChan <- StreamChunk{Error: err}
		return
	}
	defer streamResp.Close()

	// Process the SSE stream
	var totalContent strings.Builder
	var tokensUsed TokenUsage

	for {
		select {
		case event, ok := <-streamResp.Events():
			if !ok {
				// Stream ended
				return
			}

			// Handle different event types
			switch event.Type {
			case "content_block_delta":
				// This is where the actual content comes
				if event.Delta != nil && event.Delta.Type == "text_delta" {
					chunkChan <- StreamChunk{
						Content: event.Delta.Text,
					}
					totalContent.WriteString(event.Delta.Text)
				}

			case "message_stop":
				// Stream is complete
				if event.Usage != nil {
					tokensUsed = TokenUsage{
						InputTokens:  event.Usage.InputTokens,
						OutputTokens: event.Usage.OutputTokens,
						TotalTokens:  event.Usage.InputTokens + event.Usage.OutputTokens,
					}
				}

				// Send final chunk with metadata
				chunkChan <- StreamChunk{
					FinishReason: "stop",
					TokensUsed:   &tokensUsed,
					Metadata: map[string]interface{}{
						"model":          request.Model,
						"provider":       "claude",
						"response_time":  time.Since(startTime),
						"total_content":  totalContent.String(),
						"real_streaming": true,
					},
				}
				return

			case "error":
				// Handle error events
				var errMsg string
				json.Unmarshal(event.Message, &errMsg)
				chunkChan <- StreamChunk{
					Error: fmt.Errorf("claude streaming error: %s", errMsg),
				}
				return
			}

		case err := <-streamResp.Errors():
			chunkChan <- StreamChunk{Error: err}
			return

		case <-streamResp.Done():
			return

		case <-ctx.Done():
			chunkChan <- StreamChunk{Error: ctx.Err()}
			return
		}
	}
}

// streamFromGemini handles streaming from Gemini
func (s *Service) streamFromGemini(ctx context.Context, request *GenerateStreamRequest, chunkChan chan<- StreamChunk) {
	// Similar to Claude, simulate streaming for now
	// TODO: Implement real streaming when Gemini SDK supports it

	geminiReq := &GenerateRequest{
		Messages:     request.Messages,
		MaxTokens:    request.MaxTokens,
		Temperature:  request.Temperature,
		Model:        request.Model,
		SystemPrompt: request.SystemPrompt,
		Metadata:     request.Metadata,
	}

	startTime := time.Now()

	resp, err := s.GenerateResponse(ctx, geminiReq, "gemini")
	if err != nil {
		chunkChan <- StreamChunk{Error: err}
		return
	}

	// Simulate streaming
	words := splitIntoWords(resp.Content)
	buffer := make([]string, 0, 5)

	for i, word := range words {
		buffer = append(buffer, word)

		if len(buffer) >= 5 ||
			containsPunctuation(word) ||
			i == len(words)-1 {

			chunk := joinWords(buffer)
			chunkChan <- StreamChunk{
				Content: chunk,
			}

			buffer = buffer[:0]

			select {
			case <-time.After(20 * time.Millisecond):
			case <-ctx.Done():
				return
			}
		}
	}

	// Send final chunk
	chunkChan <- StreamChunk{
		FinishReason: resp.FinishReason,
		TokensUsed:   &resp.TokensUsed,
		Metadata: map[string]interface{}{
			"model":         resp.Model,
			"provider":      resp.Provider,
			"response_time": time.Since(startTime),
			"request_id":    resp.RequestID,
		},
	}
}

// Helper functions

func splitIntoWords(text string) []string {
	return strings.Fields(text)
}

func containsPunctuation(word string) bool {
	return strings.ContainsAny(word, ".!?,;:")
}

func joinWords(words []string) string {
	result := strings.Join(words, " ")
	// Add space after joined words unless it ends with punctuation
	if len(result) > 0 && !containsPunctuation(result[len(result)-1:]) {
		result += " "
	}
	return result
}
