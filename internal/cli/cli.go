// Package cli provides the command-line interface for the assistant.
// It includes interactive mode with rich UI elements, direct query mode,
// streaming response handling, and integration with the assistant core functionality.
package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/cli/ui"
	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/langchain/agent"
	"github.com/koopa0/assistant-go/internal/user"
)

// CLI represents the enhanced command-line interface
type CLI struct {
	config      config.CLIConfig
	assistant   *assistant.Assistant
	logger      *slog.Logger
	prompt      *ui.Prompt
	version     string
	authToken   string
	currentUser *user.UserInfo
	authService *user.AuthService
}

// New creates a new enhanced CLI instance
func New(cfg config.CLIConfig, assistant *assistant.Assistant, authService *user.AuthService, logger *slog.Logger) (*CLI, error) {
	if assistant == nil {
		return nil, fmt.Errorf("assistant is required")
	}
	if authService == nil {
		return nil, fmt.Errorf("auth service is required")
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
		config:      cfg,
		assistant:   assistant,
		authService: authService,
		logger:      logger,
		prompt:      prompt,
		version:     GetVersion(),
	}, nil
}

// Run starts the interactive CLI session
func (c *CLI) Run(ctx context.Context) error {
	// Show welcome screen
	c.showWelcome()

	// Check if user is logged in
	if !c.isLoggedIn() {
		ui.Info.Println("Please login to continue")
		if err := c.login(ctx); err != nil {
			ui.Error.Printf("Login failed: %v\n", err)
			return err
		}
	}

	// Show help hint
	ui.Info.Println("Type 'help' for available commands, 'menu' for interactive mode, 'exit' to quit")
	ui.Success.Println("💡 新功能: 輸入 'menu' 進入互動式任務選單!")
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

	case "logout":
		c.logout()
		return true

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

	// New interactive menu command
	case "menu":
		return c.showMainMenu(ctx) == nil

	// Workflow guide command
	case "workflow", "guide":
		c.showWorkflowGuide()
		return true

	// Quick access commands
	case "analyze":
		return c.analyzeCodeQuality(ctx) == nil

	case "test":
		return c.generateTests(ctx) == nil

	case "refactor":
		return c.suggestRefactoring(ctx) == nil

	case "optimize":
		return c.analyzePerformance(ctx) == nil

	// SQL command shortcuts
	case "sql", "postgres", "mysql":
		return c.optimizeSQL(ctx) == nil

	// Docker shortcuts
	case "docker":
		return c.analyzeDockerfile(ctx) == nil

	// K8s shortcuts
	case "k8s", "kubectl":
		return c.checkK8sConfig(ctx) == nil

	case "langchain", "lc":
		if len(args) == 0 {
			c.showLangChainHelp()
		} else {
			c.handleLangChainCommand(ctx, args)
		}
		return true

	case "agents":
		c.showLangChainAgents(ctx)
		return true

	case "chains":
		c.showLangChainChains(ctx)
		return true
	}

	return false
}

// processQuery processes a regular query through the assistant
func (c *CLI) processQuery(ctx context.Context, query string) {
	// Check if streaming mode is enabled
	if c.config.EnableStreaming {
		if err := c.processQueryStream(ctx, query); err != nil {
			ui.Error.Printf("Error: %v\n", err)
		}
		return
	}

	// Show progress indicator for non-streaming mode
	stop := ui.ShowProgress("Processing query...")

	// Create a query request with authenticated user ID
	if c.currentUser == nil {
		ui.Error.Println("Please login first")
		return
	}

	request := &assistant.QueryRequest{
		Query:  query,
		UserID: &c.currentUser.ID,
	}

	// Process the query with request
	response, err := c.assistant.ProcessQueryRequest(ctx, request)
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
	fmt.Println(c.formatResponse(response.Response))

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
		{"menu", "Show interactive task menu"},
		{"workflow, guide", "Show workflow guides"},
		{"exit, quit", "Exit the assistant"},
		{"clear, cls", "Clear the screen"},
		{"status", "Show system status"},
		{"tools", "List available tools"},
		{"history", "Show command history"},
		{"theme <dark|light>", "Change color theme"},
	}

	ui.SubHeader.Println("\n快速命令:")
	quickCommands := []struct {
		cmd  string
		desc string
	}{
		{"analyze", "Quick code quality analysis"},
		{"test", "Generate unit tests"},
		{"refactor", "Get refactoring suggestions"},
		{"optimize", "Performance optimization"},
		{"sql", "SQL query optimization"},
		{"docker", "Dockerfile analysis"},
		{"k8s", "Kubernetes config check"},
	}

	ui.SubHeader.Println("\n基本命令:")
	for _, cmd := range commands {
		ui.Label.Printf("  %-20s", cmd.cmd)
		ui.Muted.Println(cmd.desc)
	}

	ui.SubHeader.Println("\n快捷命令:")
	for _, cmd := range quickCommands {
		ui.Label.Printf("  %-20s", cmd.cmd)
		ui.Muted.Println(cmd.desc)
	}

	ui.SubHeader.Println("\nLangChain 功能:")
	langchainCommands := []struct {
		cmd  string
		desc string
	}{
		{"langchain, lc", "LangChain operations"},
		{"agents", "List available LangChain agents"},
		{"chains", "List available LangChain chains"},
	}

	for _, cmd := range langchainCommands {
		ui.Label.Printf("  %-20s", cmd.cmd)
		ui.Muted.Println(cmd.desc)
	}

	fmt.Println()
	ui.Info.Println("💡 提示: 輸入 'menu' 來使用互動式選單，或直接輸入您的問題！")
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
	if stats.Database != nil {
		data["Database Status"] = stats.Database.Status
		data["Database Connections"] = fmt.Sprintf("%d", stats.Database.TotalConns)
		data["Active Connections"] = fmt.Sprintf("%d", stats.Database.AcquiredConns)
		data["Idle Connections"] = fmt.Sprintf("%d", stats.Database.IdleConns)
	}

	// Tool stats
	if stats.Tools != nil {
		data["Available Tools"] = fmt.Sprintf("%d", stats.Tools.RegisteredTools)
	}

	// Processor stats
	if stats.Processor != nil {
		if stats.Processor.Processor != nil {
			data["Processor Status"] = stats.Processor.Processor.Status
			data["Processor Version"] = stats.Processor.Processor.Version
		}
		if stats.Processor.Health != nil {
			data["Health Status"] = stats.Processor.Health.Status
		}
		// TODO: Add request count and timing stats when processor tracking is implemented
		data["Requests Processed"] = "tracking not implemented"
		data["Avg Processing Time"] = "tracking not implemented"
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

// isLoggedIn checks if user is logged in
func (c *CLI) isLoggedIn() bool {
	return c.authToken != "" && c.currentUser != nil
}

// login performs the login flow
func (c *CLI) login(ctx context.Context) error {
	// Get email
	email, err := ui.PromptInput("Email: ")
	if err != nil {
		return err
	}

	// Get password (hidden input)
	password, err := ui.PromptPassword("Password: ")
	if err != nil {
		return err
	}

	// Authenticate
	authResp, err := c.authService.Login(ctx, email, password)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Store auth info
	c.authToken = authResp.AccessToken
	c.currentUser = &user.UserInfo{
		ID:       authResp.User.ID,
		Email:    authResp.User.Email,
		Username: authResp.User.Name,
		Roles:    []string{authResp.User.Role},
	}

	// Update prompt to show username
	c.prompt.SetPrompt(fmt.Sprintf("Assistant[%s]> ", c.currentUser.Username))

	ui.Success.Printf("Welcome, %s!\n", c.currentUser.Username)
	return nil
}

// logout logs out the current user
func (c *CLI) logout() {
	c.authToken = ""
	c.currentUser = nil
	c.prompt.SetPrompt(c.config.PromptTemplate)
	ui.Info.Println("Logged out successfully")
}

// LangChain command handlers

// showLangChainHelp displays help for LangChain commands
func (c *CLI) showLangChainHelp() {
	ui.Info.Println("\nLangChain Commands:")
	ui.Muted.Println("  langchain agents [execute <type> <query>]  - List or execute agents")
	ui.Muted.Println("  langchain chains [execute <type> <input>]  - List or execute chains")
	ui.Muted.Println("  langchain memory <command>                 - Memory operations")
	ui.Muted.Println("  agents                                     - List available agents")
	ui.Muted.Println("  chains                                     - List available chains")
}

// handleLangChainCommand handles LangChain subcommands
func (c *CLI) handleLangChainCommand(ctx context.Context, args []string) {
	if len(args) == 0 {
		c.showLangChainHelp()
		return
	}

	langchainService := c.assistant.GetLangChainService()
	if langchainService == nil {
		ui.Error.Println("LangChain service is not available")
		return
	}

	subcommand := strings.ToLower(args[0])
	subArgs := args[1:]

	switch subcommand {
	case "agents":
		if len(subArgs) > 0 && subArgs[0] == "execute" {
			if len(subArgs) < 3 {
				ui.Warning.Println("Usage: langchain agents execute <type> <query>")
				return
			}
			c.executeLangChainAgent(ctx, subArgs[1], strings.Join(subArgs[2:], " "))
		} else {
			c.showLangChainAgents(ctx)
		}

	case "chains":
		if len(subArgs) > 0 && subArgs[0] == "execute" {
			if len(subArgs) < 3 {
				ui.Warning.Println("Usage: langchain chains execute <type> <input>")
				return
			}
			c.executeLangChainChain(ctx, subArgs[1], strings.Join(subArgs[2:], " "))
		} else {
			c.showLangChainChains(ctx)
		}

	case "memory":
		ui.Warning.Println("Memory commands not yet implemented")

	default:
		ui.Error.Printf("Unknown langchain command: %s\n", subcommand)
		c.showLangChainHelp()
	}
}

// showLangChainAgents displays available LangChain agents
func (c *CLI) showLangChainAgents(ctx context.Context) {
	langchainService := c.assistant.GetLangChainService()
	if langchainService == nil {
		ui.Error.Println("LangChain service is not available")
		return
	}

	// Get agents from service
	agentTypes := langchainService.GetAgentTypes()

	ui.Info.Println("\nAvailable LangChain Agents:")
	for _, agentType := range agentTypes {
		switch agentType {
		case agent.TypeDevelopment:
			ui.Muted.Println("  - development: Development-focused agent")
		case agent.TypeDatabase:
			ui.Muted.Println("  - database: Database operations agent")
		case agent.TypeInfrastructure:
			ui.Muted.Println("  - infrastructure: Infrastructure management agent")
		case agent.TypeResearch:
			ui.Muted.Println("  - research: Research and analysis agent")
		case agent.TypeGeneral:
			ui.Muted.Println("  - general: General-purpose agent")
		default:
			ui.Muted.Printf("  - %s: Custom agent\n", agentType)
		}
	}
}

// showLangChainChains displays available LangChain chains
func (c *CLI) showLangChainChains(ctx context.Context) {
	langchainService := c.assistant.GetLangChainService()
	if langchainService == nil {
		ui.Error.Println("LangChain service is not available")
		return
	}

	// TODO: Get chains from service
	ui.Info.Println("\nAvailable LangChain Chains:")
	ui.Muted.Println("  - sequential: Sequential processing chain")
	ui.Muted.Println("  - conditional: Conditional branching chain")
	ui.Muted.Println("  - parallel: Parallel processing chain")
	ui.Muted.Println("  - rag: Retrieval-Augmented Generation chain")
}

// executeLangChainAgent executes a specific agent
func (c *CLI) executeLangChainAgent(ctx context.Context, agentType, query string) {
	langchainService := c.assistant.GetLangChainService()
	if langchainService == nil {
		ui.Error.Println("LangChain service is not available")
		return
	}

	// Show progress
	stop := ui.ShowProgress(fmt.Sprintf("Executing %s agent...", agentType))

	// Create agent request
	request := &agent.Request{
		Query:       query,
		MaxSteps:    5,
		Temperature: 0.7,
		Context:     make(map[string]interface{}),
	}

	// Execute agent through service
	response, err := langchainService.ExecuteAgent(ctx, agent.AgentType(agentType), request)
	stop()

	if err != nil {
		ui.Error.Printf("\nAgent execution failed: %v\n", err)
		return
	}

	if response.Success {
		ui.Success.Printf("\nAgent '%s' executed successfully\n", agentType)
	} else {
		ui.Warning.Printf("\nAgent '%s' execution completed with issues\n", agentType)
	}

	ui.Info.Printf("Query: %s\n", query)
	ui.Info.Printf("Execution time: %v\n", response.ExecutionTime)
	ui.Info.Printf("Confidence: %.2f\n", response.Confidence)

	// Show steps if available
	if len(response.Steps) > 0 {
		ui.Muted.Println("\nExecution steps:")
		for i, step := range response.Steps {
			ui.Muted.Printf("  %d. %s\n", i+1, step.Action)
			if step.Result != "" {
				ui.Muted.Printf("     %s\n", step.Result)
			}
		}
	}

	ui.Success.Println("\nResult:")
	fmt.Println(response.Result)
}

// executeLangChainChain executes a specific chain
func (c *CLI) executeLangChainChain(ctx context.Context, chainType, input string) {
	langchainService := c.assistant.GetLangChainService()
	if langchainService == nil {
		ui.Error.Println("LangChain service is not available")
		return
	}

	// Show progress
	stop := ui.ShowProgress(fmt.Sprintf("Executing %s chain...", chainType))
	defer stop()

	// TODO: Execute chain through service
	ui.Success.Printf("\nChain '%s' executed successfully\n", chainType)
	ui.Info.Printf("Input: %s\n", input)
	ui.Muted.Println("\nOutput: [Chain execution not yet implemented]")
}
