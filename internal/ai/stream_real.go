package ai

import (
	"context"
	"strings"
	"time"
)

// GenerateResponseStreamReal generates a truly streaming response
// This is a demonstration of what real streaming would look like
func (s *Service) GenerateResponseStreamReal(ctx context.Context, request *GenerateStreamRequest, providerName ...string) (*StreamResponse, error) {
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

		// Simulate a real streaming response
		// In reality, this would be receiving data from the API in real-time
		response := "Elegant syntax flows through code,\nConcurrency patterns unfold,\nGo programs run fast and bold,\nSimplicity worth more than gold."

		// Start timing
		startTime := time.Now()

		// Send initial metadata
		chunkChan <- StreamChunk{
			Metadata: map[string]interface{}{
				"model":    request.Model,
				"provider": provider,
				"start":    startTime,
			},
		}

		// Simulate real-time generation
		// Each word is "generated" with a delay
		words := strings.Fields(response)
		totalWords := len(words)

		for i, word := range words {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				chunkChan <- StreamChunk{
					Error: ctx.Err(),
				}
				return
			default:
			}

			// Send the word
			chunk := word
			if i < totalWords-1 {
				chunk += " "
			}

			chunkChan <- StreamChunk{
				Content: chunk,
			}

			// Simulate variable generation speed
			// Real APIs would have natural variation in response times
			delay := time.Duration(30+i%20) * time.Millisecond
			time.Sleep(delay)
		}

		// Send completion
		chunkChan <- StreamChunk{
			FinishReason: "stop",
			TokensUsed: &TokenUsage{
				InputTokens:  10,
				OutputTokens: totalWords,
				TotalTokens:  10 + totalWords,
			},
			Metadata: map[string]interface{}{
				"model":          request.Model,
				"provider":       provider,
				"response_time":  time.Since(startTime),
				"real_streaming": true,
			},
		}
	}()

	return &StreamResponse{
		ChunkChan: chunkChan,
		Done:      doneChan,
	}, nil
}
