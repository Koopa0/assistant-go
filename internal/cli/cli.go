package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/koopa0/assistant/internal/assistant"
	"github.com/koopa0/assistant/internal/cli/ui"
	"github.com/koopa0/assistant/internal/config"
)

// CLI represents the enhanced command-line interface
type CLI struct {
	config    config.CLIConfig
	assistant *assistant.Assistant
	logger    *slog.Logger
	prompt    *ui.Prompt
	version   string
}

// New creates a new enhanced CLI instance
func New(cfg config.CLIConfig, assistant *assistant.Assistant, logger *slog.Logger) (*CLI, error) {
	if assistant == nil {
		return nil, fmt.Errorf("assistant is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Create prompt with auto-completion
	promptConfig := &ui.PromptConfig{
		Prompt:       cfg.PromptTemplate,
		HistoryFile:  cfg.HistoryFile,
		MaxHistory:   cfg.MaxHistorySize,
		AutoComplete: createAutoCompleter(),
		VimMode:      false,
		MultiLine:    false,
		PromptColor:  ui.PromptSymbol,
		InputColor:   ui.UserInput,
	}

	prompt, err := ui.NewPrompt(promptConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt: %w", err)
	}

	return &CLI{
		config:    cfg,
		assistant: assistant,
		logger:    logger,
		prompt:    prompt,
		version:   "0.1.0", // TODO: Get from build flags
	}, nil
}

// Run starts the interactive CLI session
func (c *CLI) Run(ctx context.Context) error {
	// Show welcome screen
	c.showWelcome()

	// Show help hint
	ui.Info.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println()

	// Main loop
	for {
		// Read user input
		line, err := c.prompt.ReadLine()
		if err != nil {
			if err == readline.ErrInterrupt {
				if ui.Confirm("Do you want to exit?", false) {
					break
				}
				continue
			} else if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read input: %w", err)
		}

		// Skip empty lines
		if line == "" {
			continue
		}

		// Handle special commands
		if c.handleCommand(ctx, line) {
			continue
		}

		// Process query through assistant
		c.processQuery(ctx, line)
	}

	// Show goodbye message
	c.showGoodbye()

	return nil
}

// showWelcome displays the welcome screen
func (c *CLI) showWelcome() {
	// Clear screen (optional)
	if c.config.EnableColors {
		fmt.Print("\033[H\033[2J")
	}

	// Show logo
	fmt.Println(ui.ColoredLogo())

	// Show welcome message
	fmt.Println(ui.WelcomeMessage(c.version))

	// Show divider
	fmt.Println(ui.Divider())
	fmt.Println()
}

// showGoodbye displays the goodbye message
func (c *CLI) showGoodbye() {
	fmt.Println()
	fmt.Println(ui.Divider())
	ui.Success.Println("Thank you for using Assistant!")
	ui.Muted.Println("Goodbye! 👋")
}

// handleCommand handles special CLI commands
func (c *CLI) handleCommand(ctx context.Context, input string) bool {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	switch command {
	case "help", "?":
		c.showHelp()
		return true

	case "exit", "quit", "bye":
		c.prompt.Close()
		c.showGoodbye()
		os.Exit(0)

	case "clear", "cls":
		fmt.Print("\033[H\033[2J")
		return true

	case "status":
		c.showStatus(ctx)
		return true

	case "tools":
		c.showTools()
		return true

	case "history":
		c.showHistory()
		return true

	case "theme":
		if len(args) > 0 {
			c.setTheme(args[0])
		} else {
			ui.Warning.Println("Usage: theme <dark|light>")
		}
		return true

	case "sql", "postgres", "mysql":
		if len(args) > 0 {
			c.handleSQLCommand(ctx, strings.Join(args, " "))
		} else {
			ui.Warning.Println("Usage: sql <query>")
		}
		return true

	case "k8s", "kubectl":
		if len(args) > 0 {
			c.handleK8sCommand(ctx, args)
		} else {
			ui.Warning.Println("Usage: k8s <command> [args...]")
		}
		return true

	case "docker":
		if len(args) > 0 {
			c.handleDockerCommand(ctx, args)
		} else {
			ui.Warning.Println("Usage: docker <command> [args...]")
		}
		return true
	}

	return false
}

// processQuery processes a regular query through the assistant
func (c *CLI) processQuery(ctx context.Context, query string) {
	// Show progress indicator
	stop := ui.ShowProgress("Processing query...")

	// Process the query
	response, err := c.assistant.ProcessQuery(ctx, query)
	stop()

	if err != nil {
		ui.Error.Printf("Error: %v\n", err)
		return
	}

	// Display response with formatting
	fmt.Println()
	ui.Header.Println("Assistant Response:")
	fmt.Println(ui.Divider())

	// Format and display the response
	fmt.Println(c.formatResponse(response))

	fmt.Println()
}

// formatResponse formats the assistant response
func (c *CLI) formatResponse(response string) string {
	// TODO: Add markdown parsing and syntax highlighting
	// For now, just wrap long lines
	lines := strings.Split(response, "\n")
	var formatted []string

	for _, line := range lines {
		if len(line) > 80 {
			wrapped := ui.WrapText(line, 80)
			formatted = append(formatted, wrapped...)
		} else {
			formatted = append(formatted, line)
		}
	}

	return strings.Join(formatted, "\n")
}

// showHelp displays help information
func (c *CLI) showHelp() {
	fmt.Println()
	ui.Header.Println("Available Commands:")
	fmt.Println(ui.Divider())

	commands := []struct {
		cmd  string
		desc string
	}{
		{"help, ?", "Show this help message"},
		{"exit, quit", "Exit the assistant"},
		{"clear, cls", "Clear the screen"},
		{"status", "Show system status"},
		{"tools", "List available tools"},
		{"history", "Show command history"},
		{"theme <dark|light>", "Change color theme"},
		{"sql <query>", "Execute SQL query"},
		{"k8s <command>", "Execute Kubernetes command"},
		{"docker <command>", "Execute Docker command"},
	}

	for _, cmd := range commands {
		ui.Label.Printf("  %-20s", cmd.cmd)
		ui.Muted.Println(cmd.desc)
	}

	fmt.Println()
	ui.Info.Println("Or just type your question and press Enter!")
	fmt.Println()
}

// showStatus displays system status
func (c *CLI) showStatus(ctx context.Context) {
	fmt.Println()
	ui.Header.Println("System Status:")
	fmt.Println(ui.Divider())

	// Get assistant stats
	stats, err := c.assistant.Stats(ctx)
	if err != nil {
		ui.Error.Printf("Failed to get stats: %v\n", err)
		return
	}

	// Display stats
	data := make(map[string]string)

	// Database stats
	if dbStats, ok := stats["database"].(map[string]interface{}); ok {
		data["Database Connections"] = fmt.Sprintf("%v", dbStats["total_connections"])
		data["Active Connections"] = fmt.Sprintf("%v", dbStats["acquired_connections"])
	}

	// Tool stats
	if toolStats, ok := stats["tools"].(map[string]interface{}); ok {
		data["Available Tools"] = fmt.Sprintf("%v", toolStats["count"])
	}

	ui.RenderKeyValueTable("", data)
	fmt.Println()
}

// showTools displays available tools
func (c *CLI) showTools() {
	fmt.Println()
	ui.Header.Println("Available Tools:")
	fmt.Println(ui.Divider())

	tools := c.assistant.GetAvailableTools()

	if len(tools) == 0 {
		ui.Warning.Println("No tools available")
		return
	}

	headers := []string{"Name", "Category", "Description", "Version"}
	var rows [][]string

	for _, tool := range tools {
		rows = append(rows, []string{
			tool.Name,
			tool.Category,
			ui.TruncateString(tool.Description, 40),
			tool.Version,
		})
	}

	opts := ui.DefaultTableOptions()
	opts.Headers = headers
	opts.Rows = rows
	ui.RenderTable(opts)

	fmt.Println()
}

// showHistory displays command history
func (c *CLI) showHistory() {
	// TODO: Implement history display
	ui.Info.Println("Command history (last 10 commands):")
	ui.Warning.Println("History display not yet implemented")
}

// setTheme changes the color theme
func (c *CLI) setTheme(theme string) {
	switch theme {
	case "dark":
		ui.Success.Println("Switched to dark theme")
	case "light":
		ui.Success.Println("Switched to light theme")
	default:
		ui.Error.Printf("Unknown theme: %s\n", theme)
	}
}

// createAutoCompleter creates the auto-completer for the prompt
func createAutoCompleter() readline.AutoCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("help"),
		readline.PcItem("exit"),
		readline.PcItem("quit"),
		readline.PcItem("clear"),
		readline.PcItem("status"),
		readline.PcItem("tools"),
		readline.PcItem("history"),
		readline.PcItem("theme",
			readline.PcItem("dark"),
			readline.PcItem("light"),
		),
		readline.PcItem("sql"),
		readline.PcItem("k8s",
			readline.PcItem("get"),
			readline.PcItem("describe"),
			readline.PcItem("logs"),
			readline.PcItem("exec"),
		),
		readline.PcItem("docker",
			readline.PcItem("ps"),
			readline.PcItem("images"),
			readline.PcItem("logs"),
			readline.PcItem("exec"),
		),
	)
}

// Close closes the CLI
func (c *CLI) Close() error {
	if c.prompt != nil {
		return c.prompt.Close()
	}
	return nil
}
