package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
)

// PromptConfig represents configuration for the interactive prompt
type PromptConfig struct {
	Prompt       string
	HistoryFile  string
	MaxHistory   int
	AutoComplete readline.AutoCompleter
	VimMode      bool
	MultiLine    bool
	PromptColor  *color.Color
	InputColor   *color.Color
}

// DefaultPromptConfig returns default prompt configuration
func DefaultPromptConfig() *PromptConfig {
	return &PromptConfig{
		Prompt:      "assistant> ",
		HistoryFile: ".assistant_history",
		MaxHistory:  1000,
		VimMode:     false,
		MultiLine:   false,
		PromptColor: PromptSymbol,
		InputColor:  UserInput,
	}
}

// Prompt represents an interactive prompt
type Prompt struct {
	rl     *readline.Instance
	config *PromptConfig
}

// NewPrompt creates a new interactive prompt
func NewPrompt(config *PromptConfig) (*Prompt, error) {
	if config == nil {
		config = DefaultPromptConfig()
	}

	// Configure readline
	rlConfig := &readline.Config{
		Prompt:          config.PromptColor.Sprint(config.Prompt),
		HistoryFile:     config.HistoryFile,
		HistoryLimit:    config.MaxHistory,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		VimMode:         config.VimMode,
	}

	if config.AutoComplete != nil {
		rlConfig.AutoComplete = config.AutoComplete
	}

	rl, err := readline.NewEx(rlConfig)
	if err != nil {
		return nil, err
	}

	return &Prompt{
		rl:     rl,
		config: config,
	}, nil
}

// ReadLine reads a line from the prompt
func (p *Prompt) ReadLine() (string, error) {
	line, err := p.rl.Readline()
	if err != nil {
		return "", err
	}

	// Trim whitespace
	line = strings.TrimSpace(line)

	return line, nil
}

// SetPrompt updates the prompt text
func (p *Prompt) SetPrompt(prompt string) {
	p.rl.SetPrompt(p.config.PromptColor.Sprint(prompt))
}

// Close closes the prompt
func (p *Prompt) Close() error {
	return p.rl.Close()
}

// Confirm shows a yes/no confirmation prompt
func Confirm(message string, defaultYes bool) bool {
	defaultStr := " [y/N] "
	if defaultYes {
		defaultStr = " [Y/n] "
	}

	fmt.Print(Warning.Sprint(message) + defaultStr)

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "" {
		return defaultYes
	}

	return response == "y" || response == "yes"
}

// SelectOption shows a selection prompt
func SelectOption(message string, options []string) (int, error) {
	fmt.Println(Info.Sprint(message))

	for i, option := range options {
		fmt.Printf("  %s %s\n",
			Label.Sprintf("[%d]", i+1),
			option)
	}

	fmt.Print(PromptSymbol.Sprint("> "))

	var choice int
	_, err := fmt.Scanf("%d", &choice)
	if err != nil {
		return -1, err
	}

	if choice < 1 || choice > len(options) {
		return -1, fmt.Errorf("invalid choice: %d", choice)
	}

	return choice - 1, nil
}

// InputText shows a text input prompt
func InputText(prompt string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s %s: ",
			Info.Sprint(prompt),
			Muted.Sprintf("[%s]", defaultValue))
	} else {
		fmt.Printf("%s: ", Info.Sprint(prompt))
	}

	var input string
	fmt.Scanln(&input)

	if input == "" && defaultValue != "" {
		return defaultValue
	}

	return input
}

// ShowProgress shows a progress indicator
func ShowProgress(message string) func() {
	done := make(chan bool)

	go func() {
		chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0

		for {
			select {
			case <-done:
				fmt.Printf("\r%s %s\n", Success.Sprint("✓"), message)
				return
			default:
				fmt.Printf("\r%s %s", Info.Sprint(chars[i]), message)
				i = (i + 1) % len(chars)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	return func() {
		done <- true
		time.Sleep(100 * time.Millisecond) // Give time for final print
	}
}
