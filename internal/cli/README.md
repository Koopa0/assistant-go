# Command Line Interface (CLI)

## Overview

The CLI package provides an intuitive and powerful command-line interface for the Assistant intelligent development companion. It features an interactive mode with rich terminal UI, command history, auto-completion, and seamless integration with all Assistant capabilities. The CLI serves as the primary user interface for developers who prefer terminal-based workflows.

## Architecture

```
internal/cli/
├── cli.go          # Main CLI entry point and initialization
├── commands.go     # Command definitions and handlers
├── commands/       # Individual command implementations
├── interactive/    # Interactive mode components
└── ui/             # Terminal UI components
    ├── colors.go   # Color scheme and styling
    ├── format.go   # Output formatting utilities
    ├── logo.go     # ASCII art and branding
    ├── prompt.go   # Interactive prompt handling
    └── table.go    # Table rendering for structured data
```

## Key Features

### 🎨 **Rich Terminal UI**
- **Syntax Highlighting**: Code and output highlighting
- **Tables and Lists**: Structured data presentation
- **Progress Indicators**: Real-time operation feedback
- **Color Themes**: Customizable color schemes

### 🚀 **Interactive Features**
- **Command History**: Persistent across sessions
- **Auto-completion**: Context-aware suggestions
- **Multi-line Input**: Complex query support
- **Live Validation**: Real-time input validation

### 🔧 **Command System**
- **Hierarchical Commands**: Organized command structure
- **Flexible Arguments**: Positional and flag-based args
- **Pipeline Support**: Unix-style command chaining
- **Scripting Mode**: Batch command execution

## Core Components

### CLI Interface

```go
type CLI struct {
    // Core components
    app        *cli.App
    assistant  *assistant.Assistant
    config     *config.Config
    
    // UI components
    ui         *UI
    prompt     *Prompt
    formatter  *Formatter
    
    // Interactive mode
    interactive *InteractiveMode
    history     *History
    completer   *Completer
    
    // State
    context    context.Context
    logger     *slog.Logger
}

type Command interface {
    Name() string
    Description() string
    Usage() string
    Flags() []Flag
    Execute(ctx context.Context, args []string) error
    SubCommands() []Command
}
```

### Command Structure

```go
type CommandRegistry struct {
    commands map[string]Command
    aliases  map[string]string
    
    // Command categories
    categories map[string][]Command
}

func (cr *CommandRegistry) Register(cmd Command) error {
    if _, exists := cr.commands[cmd.Name()]; exists {
        return fmt.Errorf("command %s already registered", cmd.Name())
    }
    
    cr.commands[cmd.Name()] = cmd
    
    // Register in category
    category := cmd.Category()
    cr.categories[category] = append(cr.categories[category], cmd)
    
    // Register aliases
    for _, alias := range cmd.Aliases() {
        cr.aliases[alias] = cmd.Name()
    }
    
    return nil
}
```

## Command Implementation

### Base Command

```go
type BaseCommand struct {
    name        string
    description string
    usage       string
    flags       []Flag
    handler     CommandHandler
    subCommands []Command
}

func (bc *BaseCommand) Execute(ctx context.Context, args []string) error {
    // Parse flags
    flagSet := flag.NewFlagSet(bc.name, flag.ContinueOnError)
    bc.registerFlags(flagSet)
    
    if err := flagSet.Parse(args); err != nil {
        return fmt.Errorf("parsing flags: %w", err)
    }
    
    // Extract positional arguments
    positionalArgs := flagSet.Args()
    
    // Execute handler
    return bc.handler(ctx, &CommandContext{
        Command: bc,
        Args:    positionalArgs,
        Flags:   bc.parseFlags(flagSet),
    })
}
```

### Built-in Commands

#### Chat Command

```go
type ChatCommand struct {
    BaseCommand
    assistant *assistant.Assistant
    ui        *UI
}

func NewChatCommand(assistant *assistant.Assistant, ui *UI) *ChatCommand {
    return &ChatCommand{
        BaseCommand: BaseCommand{
            name:        "chat",
            description: "Start an interactive chat session",
            usage:       "chat [flags]",
            flags: []Flag{
                {Name: "model", Type: FlagTypeString, Default: "claude-3", 
                 Description: "AI model to use"},
                {Name: "stream", Type: FlagTypeBool, Default: true,
                 Description: "Enable streaming responses"},
                {Name: "context", Type: FlagTypeString,
                 Description: "Context file or directory"},
            },
        },
        assistant: assistant,
        ui:        ui,
    }
}

func (cc *ChatCommand) Execute(ctx context.Context, args []string) error {
    // Parse configuration
    config := cc.parseConfig(args)
    
    // Initialize chat session
    session, err := cc.assistant.NewChatSession(ctx, config)
    if err != nil {
        return fmt.Errorf("creating chat session: %w", err)
    }
    defer session.Close()
    
    // Enter interactive chat loop
    cc.ui.PrintLogo()
    cc.ui.PrintInfo("Starting chat session... (type 'exit' to quit)")
    
    for {
        // Get user input
        input, err := cc.ui.Prompt.ReadMultiline("> ")
        if err != nil {
            if err == io.EOF {
                break
            }
            return fmt.Errorf("reading input: %w", err)
        }
        
        if strings.TrimSpace(input) == "exit" {
            break
        }
        
        // Process with assistant
        response, err := session.SendMessage(ctx, input)
        if err != nil {
            cc.ui.PrintError("Error: %v", err)
            continue
        }
        
        // Display response
        if config.Stream {
            cc.streamResponse(response)
        } else {
            cc.ui.PrintResponse(response)
        }
    }
    
    return nil
}
```

#### Analyze Command

```go
type AnalyzeCommand struct {
    BaseCommand
    analyzer *CodeAnalyzer
    ui       *UI
}

func (ac *AnalyzeCommand) Execute(ctx context.Context, args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("no files or directories specified")
    }
    
    // Analyze each target
    for _, target := range args {
        ac.ui.PrintInfo("Analyzing %s...", target)
        
        result, err := ac.analyzer.Analyze(ctx, target)
        if err != nil {
            ac.ui.PrintError("Failed to analyze %s: %v", target, err)
            continue
        }
        
        // Display results
        ac.displayResults(result)
    }
    
    return nil
}

func (ac *AnalyzeCommand) displayResults(result *AnalysisResult) {
    // Summary table
    table := ac.ui.NewTable()
    table.SetHeaders("Metric", "Value")
    table.AddRow("Files", fmt.Sprintf("%d", result.FileCount))
    table.AddRow("Lines of Code", fmt.Sprintf("%d", result.LinesOfCode))
    table.AddRow("Complexity", fmt.Sprintf("%.2f", result.Complexity))
    table.AddRow("Test Coverage", fmt.Sprintf("%.1f%%", result.Coverage*100))
    table.Render()
    
    // Issues
    if len(result.Issues) > 0 {
        ac.ui.PrintWarning("\nIssues found:")
        for _, issue := range result.Issues {
            ac.ui.PrintIssue(issue)
        }
    }
    
    // Suggestions
    if len(result.Suggestions) > 0 {
        ac.ui.PrintInfo("\nSuggestions:")
        for _, suggestion := range result.Suggestions {
            ac.ui.PrintSuggestion(suggestion)
        }
    }
}
```

## Interactive Mode

### Interactive Shell

```go
type InteractiveMode struct {
    cli       *CLI
    reader    *readline.Instance
    history   *History
    completer *Completer
    
    // State
    session   *Session
    variables map[string]interface{}
}

func (im *InteractiveMode) Start(ctx context.Context) error {
    // Initialize readline
    config := &readline.Config{
        Prompt:          im.getPrompt(),
        HistoryFile:     im.history.GetPath(),
        AutoComplete:    im.completer,
        InterruptPrompt: "^C",
        EOFPrompt:       "exit",
        
        HistorySearchFold:   true,
        FuncFilterInputRune: im.filterInput,
    }
    
    reader, err := readline.NewEx(config)
    if err != nil {
        return fmt.Errorf("initializing readline: %w", err)
    }
    im.reader = reader
    defer reader.Close()
    
    // Display welcome message
    im.displayWelcome()
    
    // Main loop
    for {
        line, err := im.reader.Readline()
        if err != nil {
            if err == readline.ErrInterrupt {
                continue
            }
            if err == io.EOF {
                break
            }
            return fmt.Errorf("reading line: %w", err)
        }
        
        // Process command
        if err := im.processCommand(ctx, line); err != nil {
            im.cli.ui.PrintError("Error: %v", err)
        }
    }
    
    return nil
}

func (im *InteractiveMode) processCommand(ctx context.Context, line string) error {
    // Parse command line
    args, err := shellquote.Split(line)
    if err != nil {
        return fmt.Errorf("parsing command: %w", err)
    }
    
    if len(args) == 0 {
        return nil
    }
    
    // Handle special commands
    switch args[0] {
    case "help":
        return im.showHelp(args[1:])
    case "history":
        return im.showHistory()
    case "clear":
        return im.clearScreen()
    case "set":
        return im.setVariable(args[1:])
    case "exit", "quit":
        return io.EOF
    }
    
    // Execute regular command
    cmd, exists := im.cli.commands[args[0]]
    if !exists {
        return fmt.Errorf("unknown command: %s", args[0])
    }
    
    return cmd.Execute(ctx, args[1:])
}
```

### Auto-completion

```go
type Completer struct {
    cli      *CLI
    context  *CompletionContext
    providers []CompletionProvider
}

func (c *Completer) Do(line []rune, pos int) ([][]rune, int) {
    // Parse current line
    parsed := c.parseLine(string(line[:pos]))
    
    // Get completions from providers
    var suggestions []string
    for _, provider := range c.providers {
        if provider.CanComplete(parsed) {
            suggestions = append(suggestions, 
                provider.GetCompletions(parsed)...)
        }
    }
    
    // Filter by prefix
    prefix := parsed.CurrentWord
    filtered := c.filterByPrefix(suggestions, prefix)
    
    // Convert to readline format
    result := make([][]rune, len(filtered))
    for i, s := range filtered {
        result[i] = []rune(s)
    }
    
    return result, len(prefix)
}

type CommandCompletionProvider struct {
    registry *CommandRegistry
}

func (ccp *CommandCompletionProvider) GetCompletions(ctx ParsedLine) []string {
    if ctx.CommandIndex == 0 {
        // Complete command names
        return ccp.getCommandNames(ctx.CurrentWord)
    }
    
    // Get command
    cmd, exists := ccp.registry.commands[ctx.Command]
    if !exists {
        return nil
    }
    
    // Complete based on command context
    if ctx.IsFlag {
        return ccp.getFlagCompletions(cmd, ctx.CurrentWord)
    }
    
    return ccp.getArgumentCompletions(cmd, ctx)
}
```

## Terminal UI Components

### Output Formatting

```go
type Formatter struct {
    theme      *Theme
    terminal   *Terminal
    markdown   *MarkdownRenderer
    syntax     *SyntaxHighlighter
}

func (f *Formatter) FormatCode(code string, language string) string {
    // Apply syntax highlighting
    highlighted := f.syntax.Highlight(code, language)
    
    // Add line numbers if requested
    if f.theme.ShowLineNumbers {
        highlighted = f.addLineNumbers(highlighted)
    }
    
    // Apply theme colors
    return f.theme.ApplyColors(highlighted)
}

func (f *Formatter) FormatMarkdown(md string) string {
    // Parse markdown
    doc := f.markdown.Parse(md)
    
    // Render with terminal-appropriate formatting
    rendered := f.markdown.RenderTerminal(doc, f.terminal.Width())
    
    return rendered
}

func (f *Formatter) FormatTable(headers []string, rows [][]string) string {
    table := tablewriter.NewWriter(&strings.Builder{})
    table.SetHeader(headers)
    table.SetAutoWrapText(false)
    table.SetAutoFormatHeaders(true)
    table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
    table.SetAlignment(tablewriter.ALIGN_LEFT)
    table.SetCenterSeparator("")
    table.SetColumnSeparator("")
    table.SetRowSeparator("")
    table.SetHeaderLine(false)
    table.SetBorder(false)
    table.SetTablePadding("\t")
    table.SetNoWhiteSpace(true)
    
    for _, row := range rows {
        table.Append(row)
    }
    
    var buf strings.Builder
    table.Render()
    
    return buf.String()
}
```

### Progress Indicators

```go
type ProgressIndicator struct {
    spinner  *spinner.Spinner
    bar      *progressbar.ProgressBar
    multi    *mpb.Progress
    
    activeOps map[string]*Operation
    mutex     sync.RWMutex
}

func (pi *ProgressIndicator) StartOperation(name string, total int64) *Operation {
    pi.mutex.Lock()
    defer pi.mutex.Unlock()
    
    op := &Operation{
        Name:  name,
        Total: total,
        Start: time.Now(),
    }
    
    if total > 0 {
        // Create progress bar
        op.Bar = progressbar.New64(total)
        op.Bar.Describe(name)
    } else {
        // Create spinner for indeterminate progress
        op.Spinner = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
        op.Spinner.Suffix = " " + name
        op.Spinner.Start()
    }
    
    pi.activeOps[name] = op
    return op
}

func (op *Operation) Update(current int64, message string) {
    if op.Bar != nil {
        op.Bar.Set64(current)
        if message != "" {
            op.Bar.Describe(fmt.Sprintf("%s: %s", op.Name, message))
        }
    } else if op.Spinner != nil {
        op.Spinner.Suffix = fmt.Sprintf(" %s: %s", op.Name, message)
    }
}

func (op *Operation) Complete() {
    if op.Bar != nil {
        op.Bar.Finish()
    } else if op.Spinner != nil {
        op.Spinner.Stop()
    }
    
    duration := time.Since(op.Start)
    fmt.Printf("✓ %s completed in %v\n", op.Name, duration)
}
```

### Color Themes

```go
type Theme struct {
    Name        string
    Primary     lipgloss.Color
    Secondary   lipgloss.Color
    Success     lipgloss.Color
    Warning     lipgloss.Color
    Error       lipgloss.Color
    Info        lipgloss.Color
    
    // Syntax highlighting
    Keyword     lipgloss.Color
    String      lipgloss.Color
    Number      lipgloss.Color
    Comment     lipgloss.Color
    Function    lipgloss.Color
    Type        lipgloss.Color
}

var Themes = map[string]*Theme{
    "default": {
        Name:      "Default",
        Primary:   lipgloss.Color("#7D56F4"),
        Secondary: lipgloss.Color("#5A4A78"),
        Success:   lipgloss.Color("#04B575"),
        Warning:   lipgloss.Color("#FBBF24"),
        Error:     lipgloss.Color("#F87171"),
        Info:      lipgloss.Color("#60A5FA"),
        
        Keyword:   lipgloss.Color("#FF79C6"),
        String:    lipgloss.Color("#F1FA8C"),
        Number:    lipgloss.Color("#BD93F9"),
        Comment:   lipgloss.Color("#6272A4"),
        Function:  lipgloss.Color("#50FA7B"),
        Type:      lipgloss.Color("#8BE9FD"),
    },
    "monokai": {
        // Monokai theme colors...
    },
    "solarized": {
        // Solarized theme colors...
    },
}

func (t *Theme) ApplyTo(text string, style TextStyle) string {
    var s lipgloss.Style
    
    switch style {
    case StylePrimary:
        s = lipgloss.NewStyle().Foreground(t.Primary)
    case StyleSuccess:
        s = lipgloss.NewStyle().Foreground(t.Success).Bold(true)
    case StyleError:
        s = lipgloss.NewStyle().Foreground(t.Error).Bold(true)
    case StyleWarning:
        s = lipgloss.NewStyle().Foreground(t.Warning)
    case StyleInfo:
        s = lipgloss.NewStyle().Foreground(t.Info)
    }
    
    return s.Render(text)
}
```

## Configuration

### CLI Configuration

```yaml
cli:
  # Interactive mode settings
  interactive:
    enabled: true
    history_file: "~/.assistant/history"
    history_size: 10000
    auto_complete: true
    multi_line: true
    vi_mode: false
    
  # UI settings
  ui:
    theme: "default"
    show_timestamps: true
    show_line_numbers: true
    syntax_highlighting: true
    markdown_rendering: true
    table_borders: false
    
  # Progress indicators
  progress:
    show_spinner: true
    show_progress_bar: true
    show_eta: true
    update_interval: "100ms"
    
  # Command settings
  commands:
    timeout: "5m"
    max_output_size: "10MB"
    buffer_size: 4096
    
  # Prompt customization
  prompt:
    format: "assistant> "
    show_context: true
    show_mode: true
    multiline_prompt: "... "
    
  # Output settings
  output:
    pager: "less -R"
    editor: "${EDITOR:-vim}"
    format: "pretty" # pretty, json, yaml
    color: "auto" # auto, always, never
```

## Usage Examples

### Basic Command Usage

```bash
# Start interactive mode
assistant cli

# Execute single command
assistant chat "How do I optimize database queries?"

# Analyze code
assistant analyze ./src --format=detailed

# Pipeline commands
assistant search "TODO" | assistant analyze --type=tech-debt

# Use with context
assistant chat --context=./project "Explain this codebase"
```

### Programmatic Usage

```go
func ExampleCLI() {
    // Create CLI instance
    cli, err := cli.New(&cli.Config{
        Assistant: assistant,
        UI: &cli.UIConfig{
            Theme: "monokai",
            Interactive: true,
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Register custom command
    cli.RegisterCommand(&MyCustomCommand{
        name: "custom",
        handler: func(ctx context.Context, args []string) error {
            fmt.Println("Custom command executed")
            return nil
        },
    })
    
    // Run CLI
    if err := cli.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}
```

### Custom Command Implementation

```go
type DeployCommand struct {
    cli.BaseCommand
    deployer *Deployer
}

func NewDeployCommand(deployer *Deployer) *DeployCommand {
    return &DeployCommand{
        BaseCommand: cli.BaseCommand{
            name:        "deploy",
            description: "Deploy application to target environment",
            usage:       "deploy [environment] [flags]",
            flags: []cli.Flag{
                {Name: "dry-run", Type: cli.FlagTypeBool, 
                 Description: "Perform dry run without actual deployment"},
                {Name: "force", Type: cli.FlagTypeBool,
                 Description: "Force deployment even with warnings"},
                {Name: "config", Type: cli.FlagTypeString,
                 Description: "Custom configuration file"},
            },
        },
        deployer: deployer,
    }
}

func (dc *DeployCommand) Execute(ctx context.Context, args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("environment not specified")
    }
    
    environment := args[0]
    flags := dc.ParsedFlags()
    
    // Show deployment plan
    plan, err := dc.deployer.CreatePlan(environment, flags)
    if err != nil {
        return fmt.Errorf("creating deployment plan: %w", err)
    }
    
    dc.ui.PrintInfo("Deployment Plan:")
    dc.ui.PrintPlan(plan)
    
    // Confirm if not dry-run
    if !flags.GetBool("dry-run") {
        confirmed, err := dc.ui.Confirm("Proceed with deployment?")
        if err != nil || !confirmed {
            return fmt.Errorf("deployment cancelled")
        }
        
        // Execute deployment
        result, err := dc.deployer.Deploy(ctx, plan)
        if err != nil {
            return fmt.Errorf("deployment failed: %w", err)
        }
        
        dc.ui.PrintSuccess("Deployment completed successfully")
        dc.ui.PrintDeploymentResult(result)
    }
    
    return nil
}
```

## Shell Integration

### Bash Completion

```bash
# ~/.bashrc or ~/.bash_profile
eval "$(assistant completion bash)"

# Custom completion function
_assistant_custom() {
    local cur prev
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    case "${prev}" in
        --model)
            COMPREPLY=( $(compgen -W "claude-3 gemini-pro gpt-4" -- ${cur}) )
            return 0
            ;;
        --context)
            COMPREPLY=( $(compgen -f -- ${cur}) )
            return 0
            ;;
    esac
}
```

### Zsh Completion

```zsh
# ~/.zshrc
eval "$(assistant completion zsh)"

# Advanced completion with descriptions
#compdef assistant

_assistant() {
    local -a commands
    commands=(
        'chat:Start an interactive chat session'
        'analyze:Analyze code for issues and improvements'
        'search:Search through codebase'
        'deploy:Deploy application to environment'
        'help:Show help information'
    )
    
    _describe 'command' commands
}
```

## Error Handling

### User-Friendly Errors

```go
type CLIError struct {
    Code       ErrorCode
    Message    string
    Details    string
    Suggestion string
    Cause      error
}

func (e *CLIError) Error() string {
    return e.Message
}

func (e *CLIError) Display(ui *UI) {
    ui.PrintError(e.Message)
    
    if e.Details != "" {
        ui.PrintDetails(e.Details)
    }
    
    if e.Suggestion != "" {
        ui.PrintSuggestion(e.Suggestion)
    }
    
    if e.Cause != nil && ui.Config.ShowErrorCause {
        ui.PrintCause(e.Cause)
    }
}

func WrapError(err error, message string) *CLIError {
    return &CLIError{
        Code:    ErrorCodeGeneral,
        Message: message,
        Cause:   err,
        Suggestion: getSuggestionForError(err),
    }
}
```

## Related Documentation

- [Assistant Core](../assistant/README.md) - Core assistant functionality
- [UI Components](ui/README.md) - Terminal UI components
- [Commands](commands/README.md) - Command implementations
- [Configuration](../config/README.md) - CLI configuration