package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Load config
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

	// Use mock database for testing
	db := postgres.NewMockClient(logger)

	// Create assistant
	ctx := context.Background()
	asst, err := assistant.New(ctx, cfg, db, logger)
	if err != nil {
		log.Fatalf("Failed to create assistant: %v", err)
	}

	// Check available tools
	tools := asst.GetAvailableTools()
	fmt.Printf("Available tools: %d\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}

	// Test godev tool
	fmt.Println("\nTesting godev tool...")
	request := &assistant.ToolExecutionRequest{
		ToolName: "godev",
		Input: map[string]interface{}{
			"action": "detect",
			"path":   ".",
		},
	}

	result, err := asst.ExecuteTool(ctx, request)
	if err != nil {
		log.Fatalf("Failed to execute godev tool: %v", err)
	}

	fmt.Printf("Tool execution result: Success=%v\n", result.Success)
	if !result.Success {
		fmt.Printf("Error: %s\n", result.Error)
	}
}
