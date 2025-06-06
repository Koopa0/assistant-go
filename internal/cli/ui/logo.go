package ui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// Logo returns the ASCII art logo for the application
func Logo() string {
	logo := `
    ___              _      __              __ 
   /   |  __________(_)____/ /_____ _____  / /_
  / /| | / ___/ ___/ / ___/ __/ __ ` + "`" + `/ __ \/ __/
 / ___ |(__  |__  ) (__  ) /_/ /_/ / / / / /_  
/_/  |_/____/____/_/____/\__/\__,_/_/ /_/\__/  
                                                
`
	return logo
}

// ColoredLogo returns the logo with color
func ColoredLogo() string {
	lines := strings.Split(Logo(), "\n")
	colored := make([]string, len(lines))

	cyan := color.New(color.FgCyan, color.Bold)
	blue := color.New(color.FgBlue)

	for i, line := range lines {
		if i < 3 {
			colored[i] = cyan.Sprint(line)
		} else {
			colored[i] = blue.Sprint(line)
		}
	}

	return strings.Join(colored, "\n")
}

// WelcomeMessage returns the welcome message
func WelcomeMessage(version string) string {
	title := color.New(color.FgYellow, color.Bold).Sprint("AI-Powered Development Assistant")
	ver := color.New(color.FgGreen).Sprintf("Version %s", version)

	return fmt.Sprintf("\n%s\n%s\n", title, ver)
}
