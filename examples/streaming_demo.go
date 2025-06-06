package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create demo config
	cfg := &config.Config{
		Mode: "demo",
		AI: config.AIConfig{
			DefaultProvider: "claude",
			Claude: config.Claude{
				APIKey:      os.Getenv("CLAUDE_API_KEY"),
				Model:       "claude-3-sonnet-20240229",
				MaxTokens:   4096,
				Temperature: 0.7,
			},
		},
	}

	// Use mock database for demo
	db := postgres.NewMockClient(logger)

	// Create assistant
	ctx := context.Background()
	asst, err := assistant.New(ctx, cfg, db, logger)
	if err != nil {
		log.Fatalf("Failed to create assistant: %v", err)
	}

	fmt.Println("🚀 Streaming Demo - Assistant Go")
	fmt.Println("================================")
	fmt.Println("Type 'exit' to quit, or enter your question:")
	fmt.Println()

	// Setup input/output
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("You> ")
		query, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		query = strings.TrimSpace(query)
		if query == "exit" || query == "quit" {
			fmt.Println("Goodbye! 👋")
			break
		}

		if query == "" {
			continue
		}

		fmt.Print("\nAssistant> ")

		// Process with streaming
		streamResp, err := asst.ProcessQueryStream(ctx, query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Handle the stream
		for {
			select {
			case text, ok := <-streamResp.TextChan:
				if !ok {
					goto done
				}
				fmt.Print(text)

			case event := <-streamResp.EventChan:
				// Could display metadata here
				_ = event

			case err := <-streamResp.ErrorChan:
				fmt.Printf("\nError: %v\n", err)
				goto done

			case <-streamResp.Done:
				goto done
			}
		}
	done:
		fmt.Println()
	}
}

// Alternative demo using io.Pipe
func pipeDemo() {
	fmt.Println("\n--- io.Pipe Demo ---")

	// Create pipes
	pr, pw := io.Pipe()

	// Start reader in goroutine
	go func() {
		scanner := bufio.NewScanner(pr)
		scanner.Split(bufio.ScanWords)

		for scanner.Scan() {
			fmt.Print(scanner.Text() + " ")
			time.Sleep(50 * time.Millisecond) // Simulate reading delay
		}
	}()

	// Write to pipe
	writer := bufio.NewWriter(pw)
	message := "This is a streaming message that will appear word by word!"

	words := strings.Fields(message)
	for _, word := range words {
		writer.WriteString(word + " ")
		writer.Flush()
		time.Sleep(100 * time.Millisecond) // Simulate processing delay
	}

	pw.Close()
	time.Sleep(200 * time.Millisecond) // Wait for reader to finish
	fmt.Println()
}
