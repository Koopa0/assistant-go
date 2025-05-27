package cli

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/observability"
)

// CLI represents the command-line interface
type CLI struct {
	config    config.CLIConfig
	assistant *assistant.Assistant
	logger    *slog.Logger
	scanner   *bufio.Scanner
}

// New creates a new CLI instance
func New(cfg config.CLIConfig, assistant *assistant.Assistant, logger *slog.Logger) (*CLI, error) {
	if assistant == nil {
		return nil, fmt.Errorf("assistant is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	return &CLI{
		config:    cfg,
		assistant: assistant,
		logger:    observability.CLILogger(logger, "interactive"),
		scanner:   bufio.NewScanner(os.Stdin),
	}, nil
}

// Run starts the interactive CLI
func (c *CLI) Run(ctx context.Context) error {
	c.logger.Info("Starting interactive CLI")

	// Print welcome message
	c.printWelcome()

	// Main interaction loop
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			c.logger.Info("CLI interrupted by context cancellation")
			return ctx.Err()
		default:
		}

		// Print prompt
		c.printPrompt()

		// Read user input
		if !c.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(c.scanner.Text())
		if input == "" {
			continue
		}

		// Handle special commands
		if c.handleSpecialCommand(ctx, input) {
			continue
		}

		// Process query with assistant
		if err := c.processQuery(ctx, input); err != nil {
			c.printError(fmt.Sprintf("Error: %v", err))
		}
	}

	// Check for scanner errors
	if err := c.scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	c.printGoodbye()
	return nil
}

// printWelcome prints the welcome message
func (c *CLI) printWelcome() {
	if c.config.EnableColors {
		fmt.Printf("\033[36m") // Cyan color
		fmt.Println("╔══════════════════════════════════════════════════════════════╗")
		fmt.Println("║                        GoAssistant                           ║")
		fmt.Println("║                AI-powered development assistant              ║")
		fmt.Println("╚══════════════════════════════════════════════════════════════╝")
		fmt.Printf("\033[32m") // Green color
		fmt.Println("\nWelcome! I'm here to help with your development tasks.")
		fmt.Println("Type 'help' for available commands or 'exit' to quit.")
		fmt.Printf("\033[0m") // Reset color
	} else {
		fmt.Println("GoAssistant - AI-powered development assistant")
		fmt.Println("Welcome! I'm here to help with your development tasks.")
		fmt.Println("Type 'help' for available commands or 'exit' to quit.")
	}
	fmt.Println()
}

// printPrompt prints the command prompt
func (c *CLI) printPrompt() {
	if c.config.EnableColors {
		fmt.Printf("\033[34m%s\033[0m", c.config.PromptTemplate)
	} else {
		fmt.Print(c.config.PromptTemplate)
	}
}

// printGoodbye prints the goodbye message
func (c *CLI) printGoodbye() {
	if c.config.EnableColors {
		fmt.Printf("\033[32m")
		fmt.Println("\nGoodbye! Thanks for using GoAssistant.")
		fmt.Printf("\033[0m")
	} else {
		fmt.Println("\nGoodbye! Thanks for using GoAssistant.")
	}
}

// printError prints an error message
func (c *CLI) printError(message string) {
	if c.config.EnableColors {
		fmt.Printf("\033[31m%s\033[0m\n", message)
	} else {
		fmt.Println(message)
	}
}

// printSuccess prints a success message
func (c *CLI) printSuccess(message string) {
	if c.config.EnableColors {
		fmt.Printf("\033[32m%s\033[0m\n", message)
	} else {
		fmt.Println(message)
	}
}

// printInfo prints an info message
func (c *CLI) printInfo(message string) {
	if c.config.EnableColors {
		fmt.Printf("\033[36m%s\033[0m\n", message)
	} else {
		fmt.Println(message)
	}
}

// handleSpecialCommand handles special CLI commands
func (c *CLI) handleSpecialCommand(ctx context.Context, input string) bool {
	switch strings.ToLower(input) {
	case "exit", "quit", "q":
		return true // This will break the main loop

	case "help", "h":
		c.printHelp()
		return true

	case "clear", "cls":
		c.clearScreen()
		return true

	case "status":
		c.printStatus(ctx)
		return true

	case "tools":
		c.printTools()
		return true

	case "health":
		c.printHealth(ctx)
		return true

	case "version":
		c.printVersion()
		return true

	default:
		return false
	}
}

// printHelp prints the help message
func (c *CLI) printHelp() {
	c.printInfo("Available commands:")
	fmt.Println("  help, h      - Show this help message")
	fmt.Println("  status       - Show assistant status")
	fmt.Println("  tools        - List available tools")
	fmt.Println("  health       - Check system health")
	fmt.Println("  version      - Show version information")
	fmt.Println("  clear, cls   - Clear the screen")
	fmt.Println("  exit, quit   - Exit the assistant")
	fmt.Println()
	fmt.Println("You can also ask any question or request assistance with development tasks.")
	fmt.Println()
}

// clearScreen clears the terminal screen
func (c *CLI) clearScreen() {
	fmt.Print("\033[2J\033[H")
}

// printStatus prints the assistant status
func (c *CLI) printStatus(ctx context.Context) {
	stats, err := c.assistant.Stats(ctx)
	if err != nil {
		c.printError(fmt.Sprintf("Failed to get status: %v", err))
		return
	}

	c.printInfo("Assistant Status:")
	fmt.Printf("  Database connections: %v\n", stats["database"])
	fmt.Printf("  Available tools: %v\n", stats["tools"])
	fmt.Printf("  Processor stats: %v\n", stats["processor"])
	fmt.Println()
}

// printTools prints the available tools
func (c *CLI) printTools() {
	tools := c.assistant.GetAvailableTools()

	c.printInfo(fmt.Sprintf("Available Tools (%d):", len(tools)))
	if len(tools) == 0 {
		fmt.Println("  No tools currently available")
	} else {
		for _, tool := range tools {
			status := "enabled"
			if !tool.IsEnabled {
				status = "disabled"
			}
			fmt.Printf("  %-15s - %s (%s)\n", tool.Name, tool.Description, status)
		}
	}
	fmt.Println()
}

// printHealth prints the system health status
func (c *CLI) printHealth(ctx context.Context) {
	if err := c.assistant.Health(ctx); err != nil {
		c.printError(fmt.Sprintf("Health check failed: %v", err))
		return
	}

	c.printSuccess("System is healthy ✓")
	fmt.Println()
}

// printVersion prints version information
func (c *CLI) printVersion() {
	c.printInfo("GoAssistant v0.1.0")
	fmt.Println("  Built with Go 1.24+")
	fmt.Println("  Architecture: Modular monolith")
	fmt.Println("  Database: PostgreSQL with pgvector")
	fmt.Println()
}

// processQuery processes a user query
func (c *CLI) processQuery(ctx context.Context, query string) error {
	c.logger.Debug("Processing query", slog.String("query", query))

	// Show processing indicator
	if c.config.EnableColors {
		fmt.Printf("\033[33mProcessing...\033[0m\n")
	} else {
		fmt.Println("Processing...")
	}

	// Process the query
	response, err := c.assistant.ProcessQuery(ctx, query)
	if err != nil {
		return err
	}

	// Print the response
	fmt.Println()
	if c.config.EnableColors {
		fmt.Printf("\033[32m")
		fmt.Println("Assistant:")
		fmt.Printf("\033[0m")
	} else {
		fmt.Println("Assistant:")
	}

	fmt.Println(response)
	fmt.Println()

	return nil
}
