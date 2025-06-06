package cli

import (
	"context"
	"fmt"

	"github.com/koopa0/assistant-go/internal/cli/ui"
)

// processQueryStream processes a query with real streaming output
func (c *CLI) processQueryStream(ctx context.Context, query string) error {
	// Display header
	fmt.Println()
	ui.Header.Println("Assistant Response:")
	fmt.Println(ui.Divider())
	fmt.Println()

	// Get streaming response from assistant
	streamResp, err := c.assistant.ProcessQueryStream(ctx, query)
	if err != nil {
		ui.Error.Printf("Failed to start streaming: %v\n", err)
		return err
	}

	// Track if we've received any content
	hasContent := false

	// Process the stream
	for {
		select {
		case text, ok := <-streamResp.TextChan:
			if !ok {
				// Channel closed, ensure we have a newline
				if hasContent {
					fmt.Println()
				}
				return nil
			}
			// Print text chunk immediately
			fmt.Print(text)
			hasContent = true

		case event := <-streamResp.EventChan:
			// Handle events
			switch event.Type {
			case "complete":
				// Show execution time if enabled
				if c.config.ShowExecutionTime {
					if execTime, ok := event.Data["execution_time"].(string); ok {
						fmt.Printf("\n\n%s Execution time: %s", ui.Muted.Sprint("⏱"), execTime)
					}
				}
				// Show token usage if enabled
				if c.config.ShowTokenUsage {
					if tokens, ok := event.Data["tokens_used"].(int); ok {
						fmt.Printf("\n%s Tokens used: %d", ui.Muted.Sprint("🔤"), tokens)
					}
				}
			case "metadata":
				// Could handle other metadata events here
			}

		case err := <-streamResp.ErrorChan:
			ui.Error.Printf("\nStreaming error: %v\n", err)
			return err

		case <-streamResp.Done:
			// Streaming complete
			if hasContent {
				fmt.Println()
			}
			return nil

		case <-ctx.Done():
			// Context cancelled
			return ctx.Err()
		}
	}
}
