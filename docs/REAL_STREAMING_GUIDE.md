# Real Streaming Implementation Guide

## Overview

This guide explains how to implement true end-to-end streaming with Claude and Gemini APIs, using Server-Sent Events (SSE) and WebSocket protocols.

## Architecture

### 1. Claude API Native Streaming

Claude API supports streaming through Server-Sent Events. The implementation is in `/internal/ai/claude/stream.go`:

```go
// Enable streaming in the request
apiReq := &apiRequest{
    Model:    request.Model,
    Messages: messages,
    Stream:   true, // This enables SSE streaming
}

// Set headers for SSE
req.Header.Set("Accept", "text/event-stream")
req.Header.Set("Cache-Control", "no-cache")
```

The API returns events in this format:
```
event: content_block_delta
data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}

event: message_stop
data: {"type":"message_stop","usage":{"input_tokens":10,"output_tokens":50}}
```

### 2. LangChain-Go Streaming Integration

LangChain-Go provides streaming callbacks that work with both Claude and Gemini:

```go
// Create streaming handler
handler := NewStreamingHandler(chunkChan, logger)

// Add streaming callback
options := []llms.CallOption{
    llms.WithStreamingFunc(handler.HandleText),
}

// Generate with streaming
llm.GenerateContent(ctx, messages, options...)
```

### 3. HTTP Server-Sent Events (SSE)

The SSE endpoint at `/api/v1/stream` provides real-time streaming to web clients:

```go
// Set SSE headers
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")

// Send events
fmt.Fprintf(w, "event: message\n")
fmt.Fprintf(w, "data: %s\n\n", jsonData)
flusher.Flush()
```

### 4. WebSocket Streaming

WebSocket provides bidirectional streaming at `/ws/stream`:

```go
// Send streaming chunks
ws.send(JSON.stringify({
    type: "chunk",
    content: "streaming text...",
    timestamp: Date.now()
}));
```

## Implementation Steps

### Step 1: Configure AI Providers

```go
// Claude configuration
claude.New(
    claude.WithAPIKey(apiKey),
    claude.WithStreaming(true),
)

// Gemini configuration  
googleai.New(
    googleai.WithAPIKey(apiKey),
    googleai.WithStreaming(true),
)
```

### Step 2: Handle Streaming Events

```go
for {
    select {
    case event := <-streamResp.Events():
        switch event.Type {
        case "content_block_delta":
            // Send text chunk immediately
            sendChunk(event.Delta.Text)
        case "message_stop":
            // Handle completion
            sendCompletion(event.Usage)
        }
    }
}
```

### Step 3: Error Handling

```go
// Graceful error handling
defer func() {
    if r := recover(); r != nil {
        sendError("Stream interrupted")
    }
}()

// Timeout management
ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
defer cancel()
```

## Best Practices

### 1. Connection Management

- Use context for cancellation
- Implement heartbeat/ping for long connections
- Handle client disconnections gracefully
- Set appropriate timeouts

### 2. Buffer Management

```go
// Use buffered channels
chunkChan := make(chan StreamChunk, 100)

// Flush regularly
if time.Since(lastFlush) > 50*time.Millisecond {
    flusher.Flush()
}
```

### 3. Multi-Model Consistency

```go
// Abstract streaming interface
type StreamProvider interface {
    Stream(ctx context.Context, messages []Message) (<-chan Chunk, error)
}

// Implement for each provider
type ClaudeStreamer struct{}
type GeminiStreamer struct{}
```

### 4. Performance Optimization

- Minimize allocations in hot path
- Use sync.Pool for buffers
- Batch small chunks when appropriate
- Monitor goroutine leaks

## Testing Real Streaming

### 1. Test SSE Endpoint

```bash
# Test with curl
curl -N -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"query":"Tell me a story"}' \
  http://localhost:8080/api/v1/stream
```

### 2. Test with Browser

Open `examples/real_streaming_demo.html` and observe:
- Time to First Token (TTFT)
- Tokens per second
- Smooth character-by-character display

### 3. Performance Metrics

Monitor these key metrics:
- **TTFT**: Should be < 1 second
- **Throughput**: 50-100 tokens/second
- **Latency**: < 100ms between chunks
- **Memory**: Stable under load

## Troubleshooting

### No Streaming Output

1. Verify API key supports streaming
2. Check network allows SSE/WebSocket
3. Ensure headers are set correctly
4. Verify no proxy/CDN buffering

### Chunked Output

If output appears in blocks rather than smoothly:
- Check for proxy buffering
- Verify `http.Flusher` is called
- Reduce chunk aggregation time
- Check client-side buffering

### Connection Drops

- Implement reconnection logic
- Use exponential backoff
- Monitor for rate limits
- Check timeout settings

## Example Implementation

See complete examples:
- `/internal/ai/claude/stream.go` - Claude SSE implementation
- `/internal/langchain/streaming.go` - LangChain integration
- `/internal/platform/server/sse/handler.go` - HTTP SSE endpoint
- `/examples/real_streaming_demo.html` - Browser client

## Migration from Simulated Streaming

To migrate from simulated to real streaming:

1. Replace `GenerateResponse` with `GenerateResponseStream`
2. Remove artificial delays and chunking
3. Update error handling for streaming context
4. Test with real API keys
5. Monitor performance metrics

## Resources

- [Claude API Streaming Docs](https://docs.anthropic.com/claude/reference/streaming)
- [Gemini API Streaming Docs](https://ai.google.dev/tutorials/stream)
- [Server-Sent Events Spec](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [LangChain Streaming Guide](https://github.com/tmc/langchaingo#streaming)