package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

// FormatJSON formats JSON data with indentation and syntax highlighting
func FormatJSON(data interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	// TODO: Add syntax highlighting for JSON
	return string(jsonBytes), nil
}

// FormatYAML formats YAML data with syntax highlighting
func FormatYAML(data interface{}) (string, error) {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}

	// TODO: Add syntax highlighting for YAML
	return string(yamlBytes), nil
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// FormatBytes formats bytes in a human-readable way
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// TruncateString truncates a string to a maximum length
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// WrapText wraps text to a specified width
func WrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)

	var currentLine strings.Builder
	for _, word := range words {
		if currentLine.Len() > 0 && currentLine.Len()+1+len(word) > width {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
		}
		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
		}
		currentLine.WriteString(word)
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// FormatK8sAge formats Kubernetes age from a timestamp
func FormatK8sAge(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1d"
	}
	return fmt.Sprintf("%dd", days)
}

// FormatDockerSize formats Docker image/container size
func FormatDockerSize(size int64) string {
	return FormatBytes(size)
}

// FormatSQLQuery formats SQL query with basic syntax highlighting
func FormatSQLQuery(query string) string {
	// SQL keywords to highlight
	keywords := []string{
		"SELECT", "FROM", "WHERE", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER",
		"ON", "AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE", "AS",
		"ORDER", "BY", "GROUP", "HAVING", "LIMIT", "OFFSET", "UNION", "ALL",
		"INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE", "CREATE", "TABLE",
		"ALTER", "DROP", "INDEX", "PRIMARY", "KEY", "FOREIGN", "REFERENCES",
		"CONSTRAINT", "DEFAULT", "NULL", "NOT NULL", "UNIQUE", "CHECK",
		"BEGIN", "COMMIT", "ROLLBACK", "TRANSACTION",
	}

	result := query
	for _, keyword := range keywords {
		// Replace whole words only
		result = strings.ReplaceAll(result, " "+keyword+" ", " "+SQLKeyword.Sprint(keyword)+" ")
		result = strings.ReplaceAll(result, " "+strings.ToLower(keyword)+" ", " "+SQLKeyword.Sprint(strings.ToLower(keyword))+" ")

		// Handle start of line
		if strings.HasPrefix(result, keyword+" ") {
			result = SQLKeyword.Sprint(keyword) + result[len(keyword):]
		}
		if strings.HasPrefix(result, strings.ToLower(keyword)+" ") {
			result = SQLKeyword.Sprint(strings.ToLower(keyword)) + result[len(strings.ToLower(keyword)):]
		}
	}

	return result
}

// Divider returns a formatted divider line
func Divider() string {
	return strings.Repeat("─", 60)
}

// SubHeader is a secondary header color (alias for Info)
var SubHeader = Info
