package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// PromptInput prompts for user input and returns the entered text
func PromptInput(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// PromptPassword prompts for password input with hidden characters
func PromptPassword(prompt string) (string, error) {
	fmt.Print(prompt)

	// Disable echo for password input
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}

	// Print newline after password input
	fmt.Println()

	return string(password), nil
}
