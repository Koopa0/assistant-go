# Streaming Functionality Documentation

## Overview

The Assistant Go application implements a streaming architecture for AI responses across CLI, API, and WebSocket interfaces. **Important Note**: Currently, the streaming is **simulated** - the system fetches the complete response from the AI provider first, then breaks it into chunks for streaming display.

## Architecture

### Core Components

1. **AI Service Streaming** (`/internal/ai/stream.go`)
   - `GenerateResponseStream`: Main method for streaming AI responses
   - `StreamChunk`: Individual streaming data units
   - `StreamResponse`: Container for streaming channels

2. **Processor Streaming** (`/internal/assistant/processor.go`)
   - `ProcessStream`: Handles the complete streaming pipeline
   - Manages conversation context and message storage
   - Integrates with AI service for real-time responses

3. **Stream Processor** (`/internal/assistant/stream_processor.go`)
   - Adapts processor chunks to stream responses
   - Provides multiple streaming interfaces (channels, io.Pipe)
   - Supports interactive streaming sessions

## Current Implementation Status

### What Works
✅ Complete streaming architecture and pipeline
✅ CLI streaming output with word-by-word display
✅ WebSocket streaming endpoint at `/ws/stream`
✅ Channel-based communication between components
✅ Proper error handling and metadata propagation

### What's Simulated
⚠️ **AI Response Streaming**: The system currently:
1. Sends the complete query to Claude/Gemini API
2. Waits for the full response
3. Breaks the response into word chunks
4. Sends chunks with 20ms delays to simulate streaming

This means there's a delay before streaming starts (while waiting for the AI response), then the "streaming" is just a display effect.

## Implementation Details

### Streaming Flow

1. **Request Initiation**
   ```go
   streamResp, err := assistant.ProcessQueryStream(ctx, query)
   ```

2. **Channel-Based Streaming**
   - `TextChan`: Receives text chunks as they're generated
   - `EventChan`: Receives metadata and status events
   - `ErrorChan`: Receives any errors during streaming
   - `Done`: Signals completion of the stream

3. **Chunk Types**
   - `start`: Initial chunk with correlation ID and timestamp
   - `content`: Text content chunks
   - `error`: Error notifications
   - `complete`: Final chunk with metadata (tokens, execution time)

### CLI Integration

The CLI provides real-time streaming output with:
- Immediate display of response chunks
- Optional execution time display
- Optional token usage display
- Proper error handling and formatting

Example usage:
```bash
assistant ask "Explain Go channels"
# Response will stream word by word in real-time
```

### API/WebSocket Integration

WebSocket streaming provides structured messages:
```json
{
  "type": "chunk",
  "content": "This is a streaming ",
  "timestamp": 1733573456789
}
```

Message types:
- `start`: Stream initialization
- `chunk`: Content chunks
- `event`: Metadata events
- `error`: Error messages
- `complete`: Stream completion with final metadata

### Configuration

Enable streaming in your configuration:
```yaml
cli:
  enable_streaming: true
  stream_buffer_size: 1024
  show_execution_time: true
  show_token_usage: true
```

## Usage Examples

### CLI Streaming
```go
// CLI automatically uses streaming when enabled
assistant ask "Generate a Go function"
```

### Programmatic Usage
```go
// Get streaming response
streamResp, err := assistant.ProcessQueryStream(ctx, query)
if err != nil {
    return err
}

// Process stream
for {
    select {
    case text := <-streamResp.TextChan:
        fmt.Print(text)
    case event := <-streamResp.EventChan:
        // Handle events
    case err := <-streamResp.ErrorChan:
        return err
    case <-streamResp.Done:
        return nil
    }
}
```

### WebSocket Client
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/stream');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  switch(data.type) {
    case 'chunk':
      // Append content to display
      appendToOutput(data.content);
      break;
    case 'complete':
      // Show final metadata
      showMetadata(data.metadata);
      break;
    case 'error':
      // Handle error
      showError(data.error);
      break;
  }
};

// Send query
ws.send(JSON.stringify({query: "Your question here"}));
```

## Performance Considerations

- **Chunking Strategy**: Text is broken into chunks at word boundaries and punctuation
- **Buffer Size**: Default 100 chunks buffer to prevent blocking
- **Simulated Delay**: 20ms between chunks for natural streaming effect
- **Context Handling**: Full conversation context is maintained during streaming

## Error Handling

Streaming errors are propagated through the error channel and include:
- Validation errors
- AI provider errors
- Database errors
- Context timeout errors

## Future Enhancements

1. **Native Provider Streaming**: When Claude/Gemini SDKs support streaming
2. **Adaptive Chunking**: Dynamic chunk sizes based on network conditions
3. **Stream Resumption**: Resume interrupted streams
4. **Compression**: Optional compression for large responses
5. **Progress Indicators**: Token count progress during generation

## Testing

Test streaming functionality:
```bash
# Test CLI streaming
go run examples/streaming_demo.go

# Test with mock API key
CLAUDE_API_KEY=test-key go run cmd/assistant/main.go ask "Hello"
```

## Troubleshooting

### No Streaming Output
- Verify `enable_streaming: true` in configuration
- Check CLI is using `processQueryStream` method
- Ensure AI provider is configured correctly

### Incomplete Responses
- Check for errors in error channel
- Verify conversation is properly stored
- Ensure adequate timeout settings

### Performance Issues
- Adjust `stream_buffer_size` for your use case
- Consider reducing chunk frequency
- Monitor memory usage with large responses