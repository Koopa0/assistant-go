package assistant

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"log/slog"
)

// StreamProcessor handles streaming responses from the assistant
type StreamProcessor struct {
	processor     *Processor
	logger        *slog.Logger
	bufferSize    int
	flushInterval time.Duration
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(processor *Processor, logger *slog.Logger) *StreamProcessor {
	return &StreamProcessor{
		processor:     processor,
		logger:        logger,
		bufferSize:    256,
		flushInterval: 50 * time.Millisecond,
	}
}

// StreamResponse represents a streamed response chunk
type StreamResponse struct {
	Chunk    string                 `json:"chunk"`
	Finished bool                   `json:"finished"`
	Error    error                  `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ProcessStream processes a query and streams the response using real streaming
func (sp *StreamProcessor) ProcessStream(ctx context.Context, request *QueryRequest) (<-chan StreamResponse, error) {
	// Validate request
	if err := sp.processor.validateRequest(request); err != nil {
		return nil, err
	}

	// Create response channel
	responseChan := make(chan StreamResponse, 100)

	// Start processing in goroutine
	go func() {
		defer close(responseChan)

		// Send initial metadata
		responseChan <- StreamResponse{
			Metadata: map[string]interface{}{
				"conversation_id": request.ConversationID,
				"started_at":      time.Now(),
			},
		}

		// Use the new ProcessStream method for real streaming
		streamChunks, err := sp.processor.ProcessStream(ctx, request)
		if err != nil {
			responseChan <- StreamResponse{
				Error:    err,
				Finished: true,
			}
			return
		}

		// Convert processor chunks to stream responses
		for chunk := range streamChunks {
			switch chunk.Type {
			case "content":
				// Send content chunks
				responseChan <- StreamResponse{
					Chunk: chunk.Content,
				}
			case "error":
				// Send error
				responseChan <- StreamResponse{
					Error:    chunk.Error,
					Finished: true,
				}
				return
			case "complete":
				// Send completion with metadata
				responseChan <- StreamResponse{
					Finished: true,
					Metadata: chunk.Metadata,
				}
			}
		}
	}()

	return responseChan, nil
}

// streamText breaks text into chunks and streams them
func (sp *StreamProcessor) streamText(text string, out chan<- StreamResponse) {
	// Use a scanner to read the text word by word
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Split(bufio.ScanWords)

	buffer := make([]string, 0, 10)
	lastFlush := time.Now()

	for scanner.Scan() {
		word := scanner.Text()
		buffer = append(buffer, word)

		// Flush based on buffer size or time
		shouldFlush := len(buffer) >= 5 ||
			time.Since(lastFlush) > sp.flushInterval ||
			strings.ContainsAny(word, ".!?") // Flush on sentence endings

		if shouldFlush && len(buffer) > 0 {
			chunk := strings.Join(buffer, " ")
			if !strings.HasSuffix(chunk, ".") && !strings.HasSuffix(chunk, "!") && !strings.HasSuffix(chunk, "?") {
				chunk += " "
			}

			out <- StreamResponse{
				Chunk: chunk,
			}

			buffer = buffer[:0]
			lastFlush = time.Now()

			// Small delay to simulate natural streaming
			time.Sleep(20 * time.Millisecond)
		}
	}

	// Flush remaining buffer
	if len(buffer) > 0 {
		out <- StreamResponse{
			Chunk: strings.Join(buffer, " "),
		}
	}
}

// ProcessWithPipe processes a query using io.Pipe for streaming
func (sp *StreamProcessor) ProcessWithPipe(ctx context.Context, request *QueryRequest) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		// Process the request
		response, err := sp.processor.Process(ctx, request)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		// Write response in chunks
		writer := bufio.NewWriter(pw)
		sp.writeInChunks(response.Response, writer)
		writer.Flush()
	}()

	return pr, nil
}

// writeInChunks writes text in chunks to a writer
func (sp *StreamProcessor) writeInChunks(text string, w io.Writer) {
	words := strings.Fields(text)
	buffer := make([]string, 0, 5)

	for i, word := range words {
		buffer = append(buffer, word)

		// Write chunk every 5 words or at punctuation
		if len(buffer) >= 5 ||
			strings.ContainsAny(word, ".!?") ||
			i == len(words)-1 {

			chunk := strings.Join(buffer, " ")
			if i < len(words)-1 {
				chunk += " "
			}

			fmt.Fprint(w, chunk)
			buffer = buffer[:0]

			// Flush if it's a buffered writer
			if bw, ok := w.(*bufio.Writer); ok {
				bw.Flush()
			}

			time.Sleep(30 * time.Millisecond)
		}
	}
}

// StreamingWriter implements io.Writer that streams data through a channel
type StreamingWriter struct {
	ch     chan<- []byte
	buffer []byte
	mu     sync.Mutex
}

// NewStreamingWriter creates a new streaming writer
func NewStreamingWriter(ch chan<- []byte) *StreamingWriter {
	return &StreamingWriter{
		ch:     ch,
		buffer: make([]byte, 0, 1024),
	}
}

// Write implements io.Writer
func (sw *StreamingWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	// Send data directly to channel
	data := make([]byte, len(p))
	copy(data, p)

	select {
	case sw.ch <- data:
		return len(p), nil
	default:
		return 0, fmt.Errorf("channel blocked")
	}
}

// ProcessQueryStreamEnhanced provides enhanced streaming with multiple options
func (a *Assistant) ProcessQueryStreamEnhanced(ctx context.Context, request *QueryRequest) (*StreamingResponse, error) {
	if request == nil {
		return nil, NewAssistantInvalidInputError("request is required", request)
	}

	// Create channels
	textChan := make(chan string, 100)
	eventChan := make(chan StreamEvent, 10)
	errorChan := make(chan error, 1)
	doneChan := make(chan struct{})

	// Create streaming response with read-only channels
	resp := &StreamingResponse{
		TextChan:  textChan,
		EventChan: eventChan,
		ErrorChan: errorChan,
		Done:      doneChan,
	}

	// Create stream processor
	streamProc := NewStreamProcessor(a.processor, a.logger)

	go func() {
		defer close(doneChan)
		defer close(textChan)
		defer close(eventChan)

		// Send start event
		eventChan <- StreamEvent{
			Type:      "start",
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"conversation_id": request.ConversationID,
			},
		}

		// Get stream channel
		streamChan, err := streamProc.ProcessStream(ctx, request)
		if err != nil {
			errorChan <- err
			return
		}

		// Process stream
		for chunk := range streamChan {
			if chunk.Error != nil {
				errorChan <- chunk.Error
				return
			}

			if chunk.Chunk != "" {
				textChan <- chunk.Chunk
			}

			if chunk.Metadata != nil {
				eventChan <- StreamEvent{
					Type:      "metadata",
					Timestamp: time.Now(),
					Data:      chunk.Metadata,
				}
			}

			if chunk.Finished {
				eventChan <- StreamEvent{
					Type:      "complete",
					Timestamp: time.Now(),
					Data:      chunk.Metadata,
				}
				break
			}
		}
	}()

	return resp, nil
}

// StreamingResponse represents a streaming response
type StreamingResponse struct {
	TextChan  <-chan string
	EventChan <-chan StreamEvent
	ErrorChan <-chan error
	Done      <-chan struct{}
}

// StreamEvent represents an event in the stream
type StreamEvent struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// ProcessWithCustomWriter processes and writes to a custom writer
func (sp *StreamProcessor) ProcessWithCustomWriter(ctx context.Context, request *QueryRequest, writer io.Writer) error {
	// Create a pipe
	pr, pw := io.Pipe()

	// Error channel
	errChan := make(chan error, 1)

	// Start copying in background
	go func() {
		_, err := io.Copy(writer, pr)
		errChan <- err
	}()

	// Process and write
	go func() {
		defer pw.Close()

		response, err := sp.processor.Process(ctx, request)
		if err != nil {
			pw.CloseWithError(err)
			return
		}

		// Write response with streaming effect
		sp.writeInChunks(response.Response, pw)
	}()

	// Wait for completion
	return <-errChan
}

// InteractiveStreamProcessor handles interactive streaming sessions
type InteractiveStreamProcessor struct {
	assistant *Assistant
	logger    *slog.Logger
}

// NewInteractiveStreamProcessor creates a new interactive stream processor
func NewInteractiveStreamProcessor(assistant *Assistant, logger *slog.Logger) *InteractiveStreamProcessor {
	return &InteractiveStreamProcessor{
		assistant: assistant,
		logger:    logger,
	}
}

// StartSession starts an interactive streaming session
func (isp *InteractiveStreamProcessor) StartSession(ctx context.Context, input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	writer := bufio.NewWriter(output)

	// Create or get conversation
	var conversationID *string

	for scanner.Scan() {
		query := scanner.Text()
		if query == "" {
			continue
		}

		// Create request
		request := &QueryRequest{
			Query:          query,
			ConversationID: conversationID,
		}

		// Get streaming response
		resp, err := isp.assistant.ProcessQueryStreamEnhanced(ctx, request)
		if err != nil {
			fmt.Fprintf(writer, "Error: %v\n", err)
			writer.Flush()
			continue
		}

		// Process stream
		for {
			select {
			case text, ok := <-resp.TextChan:
				if !ok {
					goto done
				}
				fmt.Fprint(writer, text)
				writer.Flush()

			case event := <-resp.EventChan:
				if event.Type == "metadata" {
					if id, ok := event.Data["conversation_id"].(string); ok && conversationID == nil {
						conversationID = &id
					}
				}

			case err := <-resp.ErrorChan:
				fmt.Fprintf(writer, "\nError: %v\n", err)
				writer.Flush()
				goto done

			case <-resp.Done:
				goto done
			}
		}
	done:
		fmt.Fprintln(writer) // New line after response
		writer.Flush()
	}

	return scanner.Err()
}
