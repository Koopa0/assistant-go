package ui

import (
	"github.com/fatih/color"
)

// Color scheme for the application
var (
	// Basic colors
	Success = color.New(color.FgGreen)
	Error   = color.New(color.FgRed, color.Bold)
	Warning = color.New(color.FgYellow)
	Info    = color.New(color.FgCyan)

	// Text styles
	Bold      = color.New(color.Bold)
	Italic    = color.New(color.Italic)
	Underline = color.New(color.Underline)

	// Component colors
	Header    = color.New(color.FgHiWhite, color.Bold)
	Subheader = color.New(color.FgHiWhite)
	Label     = color.New(color.FgYellow)
	Value     = color.New(color.FgGreen)
	Muted     = color.New(color.FgHiBlack)
	Highlight = color.New(color.FgMagenta, color.Bold)

	// Tool-specific colors
	SQLKeyword = color.New(color.FgBlue, color.Bold)
	SQLTable   = color.New(color.FgCyan)
	SQLColumn  = color.New(color.FgGreen)
	SQLValue   = color.New(color.FgYellow)

	K8sResource  = color.New(color.FgCyan, color.Bold)
	K8sNamespace = color.New(color.FgBlue)
	K8sStatus    = color.New(color.FgGreen)
	K8sError     = color.New(color.FgRed)

	DockerImage     = color.New(color.FgBlue, color.Bold)
	DockerContainer = color.New(color.FgCyan)
	DockerRunning   = color.New(color.FgGreen)
	DockerStopped   = color.New(color.FgRed)

	// Prompt colors
	PromptSymbol = color.New(color.FgMagenta, color.Bold)
	PromptText   = color.New(color.FgHiWhite)
	UserInput    = color.New(color.FgYellow)
)

// StatusColor returns appropriate color based on status
func StatusColor(status string) *color.Color {
	switch status {
	case "running", "active", "ready", "success", "healthy":
		return Success
	case "error", "failed", "unhealthy", "crashloopbackoff":
		return Error
	case "pending", "creating", "updating", "warning":
		return Warning
	default:
		return Info
	}
}
