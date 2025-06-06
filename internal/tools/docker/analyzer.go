package docker

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// DockerfileAnalyzer analyzes Dockerfiles for best practices and issues
type DockerfileAnalyzer struct {
	logger *slog.Logger
}

// NewDockerfileAnalyzer creates a new Dockerfile analyzer
func NewDockerfileAnalyzer(logger *slog.Logger) *DockerfileAnalyzer {
	return &DockerfileAnalyzer{
		logger: logger,
	}
}

// AnalysisResult represents the result of Dockerfile analysis
type AnalysisResult struct {
	Issues        []Issue           `json:"issues"`
	Metrics       DockerfileMetrics `json:"metrics"`
	Suggestions   []string          `json:"suggestions"`
	BestPractices map[string]bool   `json:"best_practices"`
}

// Issue represents a problem found in the Dockerfile
type Issue struct {
	Line     int    `json:"line"`
	Severity string `json:"severity"` // "error", "warning", "info"
	Message  string `json:"message"`
	Rule     string `json:"rule"`
}

// DockerfileMetrics contains metrics about the Dockerfile
type DockerfileMetrics struct {
	TotalLines       int `json:"total_lines"`
	Instructions     int `json:"instructions"`
	Layers           int `json:"layers"`
	BaseImageCount   int `json:"base_image_count"`
	CopyInstructions int `json:"copy_instructions"`
	RunInstructions  int `json:"run_instructions"`
}

// AnalyzeFile analyzes a Dockerfile file
func (a *DockerfileAnalyzer) AnalyzeFile(filepath string) (*AnalysisResult, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Dockerfile: %w", err)
	}
	defer file.Close()

	result := &AnalysisResult{
		Issues:        []Issue{},
		Suggestions:   []string{},
		BestPractices: make(map[string]bool),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	hasHealthcheck := false
	hasUser := false
	hasWorkdir := false
	runCommands := []string{}

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse instruction
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		instruction := strings.ToUpper(parts[0])
		result.Metrics.Instructions++

		// Analyze each instruction
		switch instruction {
		case "FROM":
			result.Metrics.BaseImageCount++
			result.Metrics.Layers++
			a.analyzeFrom(line, lineNum, result)

		case "RUN":
			result.Metrics.RunInstructions++
			result.Metrics.Layers++
			runCommands = append(runCommands, line)
			a.analyzeRun(line, lineNum, result)

		case "COPY", "ADD":
			result.Metrics.CopyInstructions++
			result.Metrics.Layers++
			a.analyzeCopy(line, lineNum, instruction, result)

		case "USER":
			hasUser = true

		case "WORKDIR":
			hasWorkdir = true

		case "HEALTHCHECK":
			hasHealthcheck = true

		case "ENV":
			a.analyzeEnv(line, lineNum, result)

		case "EXPOSE":
			a.analyzeExpose(line, lineNum, result)
		}

	}

	result.Metrics.TotalLines = lineNum

	// Check best practices
	result.BestPractices["has_user"] = hasUser
	result.BestPractices["has_workdir"] = hasWorkdir
	result.BestPractices["has_healthcheck"] = hasHealthcheck
	result.BestPractices["minimal_layers"] = result.Metrics.Layers < 20
	result.BestPractices["efficient_caching"] = a.checkCachingEfficiency(runCommands)

	// Add suggestions based on analysis
	a.generateSuggestions(result)

	return result, scanner.Err()
}

// analyzeFrom analyzes FROM instructions
func (a *DockerfileAnalyzer) analyzeFrom(line string, lineNum int, result *AnalysisResult) {
	if strings.Contains(line, "latest") {
		result.Issues = append(result.Issues, Issue{
			Line:     lineNum,
			Severity: "warning",
			Message:  "Using 'latest' tag is not recommended for reproducible builds",
			Rule:     "no-latest-tag",
		})
	}

	// Check for official base images
	if !strings.Contains(line, "/") && !strings.Contains(line, "AS") {
		// Likely an official image, which is good
		return
	}

	// Check for Alpine Linux (good for size)
	if strings.Contains(line, "alpine") {
		result.Suggestions = append(result.Suggestions,
			"Good choice using Alpine Linux for smaller image size")
	}
}

// analyzeRun analyzes RUN instructions
func (a *DockerfileAnalyzer) analyzeRun(line string, lineNum int, result *AnalysisResult) {
	// Check for apt-get/yum without clean
	if strings.Contains(line, "apt-get install") && !strings.Contains(line, "apt-get clean") &&
		!strings.Contains(line, "rm -rf /var/lib/apt/lists/*") {
		result.Issues = append(result.Issues, Issue{
			Line:     lineNum,
			Severity: "warning",
			Message:  "apt-get install should be followed by cleanup to reduce image size",
			Rule:     "apt-cleanup",
		})
	}

	// Check for multiple RUN commands that could be combined
	if strings.Count(line, "&&") < 1 && (strings.Contains(line, "apt-get") ||
		strings.Contains(line, "yum") || strings.Contains(line, "apk")) {
		result.Issues = append(result.Issues, Issue{
			Line:     lineNum,
			Severity: "info",
			Message:  "Consider combining multiple RUN commands to reduce layers",
			Rule:     "combine-run",
		})
	}

	// Check for sudo usage
	if strings.Contains(line, "sudo") {
		result.Issues = append(result.Issues, Issue{
			Line:     lineNum,
			Severity: "warning",
			Message:  "Avoid using sudo in Dockerfiles",
			Rule:     "no-sudo",
		})
	}
}

// analyzeCopy analyzes COPY and ADD instructions
func (a *DockerfileAnalyzer) analyzeCopy(line string, lineNum int, instruction string, result *AnalysisResult) {
	if instruction == "ADD" && !strings.Contains(line, "http") && !strings.Contains(line, ".tar") {
		result.Issues = append(result.Issues, Issue{
			Line:     lineNum,
			Severity: "info",
			Message:  "Prefer COPY over ADD for simple file copying",
			Rule:     "prefer-copy",
		})
	}

	// Check for copying entire context
	if strings.Contains(line, "COPY . .") || strings.Contains(line, "ADD . .") {
		result.Issues = append(result.Issues, Issue{
			Line:     lineNum,
			Severity: "warning",
			Message:  "Copying entire context may include unnecessary files. Consider using .dockerignore",
			Rule:     "specific-copy",
		})
	}
}

// analyzeEnv analyzes ENV instructions
func (a *DockerfileAnalyzer) analyzeEnv(line string, lineNum int, result *AnalysisResult) {
	// Check for secrets in ENV
	lowerLine := strings.ToLower(line)
	if strings.Contains(lowerLine, "password") || strings.Contains(lowerLine, "secret") ||
		strings.Contains(lowerLine, "key") || strings.Contains(lowerLine, "token") {
		result.Issues = append(result.Issues, Issue{
			Line:     lineNum,
			Severity: "error",
			Message:  "Avoid hardcoding secrets in ENV instructions",
			Rule:     "no-secrets",
		})
	}
}

// analyzeExpose analyzes EXPOSE instructions
func (a *DockerfileAnalyzer) analyzeExpose(line string, lineNum int, result *AnalysisResult) {
	// Check for privileged ports
	parts := strings.Fields(line)
	if len(parts) > 1 {
		port := parts[1]
		if strings.HasPrefix(port, "22") || port == "23" {
			result.Issues = append(result.Issues, Issue{
				Line:     lineNum,
				Severity: "warning",
				Message:  "Exposing SSH/Telnet ports may be a security risk",
				Rule:     "secure-ports",
			})
		}
	}
}

// checkCachingEfficiency checks if the Dockerfile is structured for efficient caching
func (a *DockerfileAnalyzer) checkCachingEfficiency(runCommands []string) bool {
	// Simple heuristic: package installations should come before code copying
	hasPackageInstall := false
	for _, cmd := range runCommands {
		if strings.Contains(cmd, "apt-get") || strings.Contains(cmd, "yum") ||
			strings.Contains(cmd, "apk") || strings.Contains(cmd, "npm install") ||
			strings.Contains(cmd, "pip install") {
			hasPackageInstall = true
			break
		}
	}

	// This is a simplified check - in reality, we'd need to track instruction order
	return !hasPackageInstall || len(runCommands) < 5
}

// generateSuggestions generates suggestions based on the analysis
func (a *DockerfileAnalyzer) generateSuggestions(result *AnalysisResult) {
	if !result.BestPractices["has_user"] {
		result.Suggestions = append(result.Suggestions,
			"Consider using USER instruction to run as non-root for better security")
	}

	if !result.BestPractices["has_healthcheck"] {
		result.Suggestions = append(result.Suggestions,
			"Consider adding HEALTHCHECK instruction for better container monitoring")
	}

	if result.Metrics.Layers > 30 {
		result.Suggestions = append(result.Suggestions,
			"Consider reducing the number of layers by combining RUN commands")
	}

	if result.Metrics.BaseImageCount > 1 {
		result.Suggestions = append(result.Suggestions,
			"Multi-stage build detected - good for reducing final image size")
	}

	// Add Go-specific suggestions if detected
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, "go build") {
			result.Suggestions = append(result.Suggestions,
				"For Go applications, consider using CGO_ENABLED=0 for static binaries")
			result.Suggestions = append(result.Suggestions,
				"Use -ldflags='-w -s' to reduce binary size")
			break
		}
	}
}
