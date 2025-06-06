package docker

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// DockerfileOptimizer optimizes Dockerfiles for better performance and smaller size
type DockerfileOptimizer struct {
	logger *slog.Logger
}

// NewDockerfileOptimizer creates a new Dockerfile optimizer
func NewDockerfileOptimizer(logger *slog.Logger) *DockerfileOptimizer {
	return &DockerfileOptimizer{
		logger: logger,
	}
}

// OptimizationResult represents the result of Dockerfile optimization
type OptimizationResult struct {
	OriginalSize     int            `json:"original_size"`
	OptimizedSize    int            `json:"optimized_size"`
	LayersReduced    int            `json:"layers_reduced"`
	Optimizations    []Optimization `json:"optimizations"`
	OptimizedContent string         `json:"optimized_content"`
	SizeReduction    string         `json:"size_reduction"`
}

// Optimization represents a single optimization made
type Optimization struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Impact      string `json:"impact"` // "high", "medium", "low"
	LinesBefore int    `json:"lines_before"`
	LinesAfter  int    `json:"lines_after"`
}

// OptimizeFile optimizes a Dockerfile
func (o *DockerfileOptimizer) OptimizeFile(filepath string) (*OptimizationResult, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Dockerfile: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	result := &OptimizationResult{
		OriginalSize:  len(lines),
		Optimizations: []Optimization{},
	}

	// Apply optimizations
	optimizedLines := o.optimizeLines(lines, result)

	// Generate optimized content
	result.OptimizedContent = strings.Join(optimizedLines, "\n")
	result.OptimizedSize = len(optimizedLines)
	result.LayersReduced = result.OriginalSize - result.OptimizedSize

	// Calculate size reduction
	if result.OriginalSize > 0 {
		reduction := float64(result.LayersReduced) / float64(result.OriginalSize) * 100
		result.SizeReduction = fmt.Sprintf("%.1f%%", reduction)
	}

	return result, nil
}

// optimizeLines applies various optimizations to Dockerfile lines
func (o *DockerfileOptimizer) optimizeLines(lines []string, result *OptimizationResult) []string {
	var optimized []string
	var currentRunCommands []string
	var lastInstruction string
	inMultiStage := false
	stageCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines in sequences
		if trimmed == "" && i > 0 && i < len(lines)-1 {
			prev := strings.TrimSpace(lines[i-1])
			next := strings.TrimSpace(lines[i+1])
			if prev == "" || next == "" {
				continue
			}
		}

		// Handle comments
		if strings.HasPrefix(trimmed, "#") {
			optimized = append(optimized, line)
			continue
		}

		parts := strings.Fields(trimmed)
		if len(parts) == 0 {
			optimized = append(optimized, line)
			continue
		}

		instruction := strings.ToUpper(parts[0])

		switch instruction {
		case "FROM":
			// Handle multi-stage builds
			if strings.Contains(trimmed, " AS ") || strings.Contains(trimmed, " as ") {
				inMultiStage = true
				stageCount++
			}

			// Flush any pending RUN commands
			if len(currentRunCommands) > 0 {
				optimized = append(optimized, o.combineRunCommands(currentRunCommands, result))
				currentRunCommands = nil
			}

			// Optimize FROM instruction
			optimized = append(optimized, o.optimizeFrom(line, result))

		case "RUN":
			// Collect RUN commands for potential combination
			currentRunCommands = append(currentRunCommands, trimmed)

		case "COPY", "ADD":
			// Flush RUN commands before COPY/ADD
			if len(currentRunCommands) > 0 {
				optimized = append(optimized, o.combineRunCommands(currentRunCommands, result))
				currentRunCommands = nil
			}

			// Optimize COPY/ADD
			optimized = append(optimized, o.optimizeCopy(line, instruction, result))

		case "ENV":
			// Combine ENV instructions if possible
			if lastInstruction == "ENV" && len(optimized) > 0 {
				lastLine := optimized[len(optimized)-1]
				combined := o.combineEnv(lastLine, line, result)
				optimized[len(optimized)-1] = combined
			} else {
				optimized = append(optimized, line)
			}

		case "WORKDIR":
			// Optimize WORKDIR
			optimized = append(optimized, o.optimizeWorkdir(line, result))

		default:
			// Flush RUN commands before other instructions
			if len(currentRunCommands) > 0 {
				optimized = append(optimized, o.combineRunCommands(currentRunCommands, result))
				currentRunCommands = nil
			}
			optimized = append(optimized, line)
		}

		lastInstruction = instruction
	}

	// Flush any remaining RUN commands
	if len(currentRunCommands) > 0 {
		optimized = append(optimized, o.combineRunCommands(currentRunCommands, result))
	}

	// Add multi-stage build optimization suggestion
	if inMultiStage && stageCount > 1 {
		result.Optimizations = append(result.Optimizations, Optimization{
			Type:        "multi-stage",
			Description: "Multi-stage build detected - already optimized for size",
			Impact:      "high",
		})
	}

	return optimized
}

// optimizeFrom optimizes FROM instructions
func (o *DockerfileOptimizer) optimizeFrom(line string, result *OptimizationResult) string {
	// Replace latest tags with specific versions
	if strings.Contains(line, ":latest") {
		result.Optimizations = append(result.Optimizations, Optimization{
			Type:        "pin-version",
			Description: "Replace 'latest' tag with specific version for reproducibility",
			Impact:      "medium",
		})
		// In a real implementation, we might look up the actual latest version
		// For now, we'll just add a comment
		return line + " # TODO: Replace 'latest' with specific version"
	}

	// Suggest Alpine variants for smaller size
	if strings.Contains(line, "ubuntu") || strings.Contains(line, "debian") {
		if !strings.Contains(line, "alpine") {
			result.Optimizations = append(result.Optimizations, Optimization{
				Type:        "base-image",
				Description: "Consider using Alpine Linux variant for smaller image size",
				Impact:      "high",
			})
		}
	}

	return line
}

// combineRunCommands combines multiple RUN commands into one
func (o *DockerfileOptimizer) combineRunCommands(commands []string, result *OptimizationResult) string {
	if len(commands) <= 1 {
		return strings.Join(commands, "\n")
	}

	// Create a combined RUN command
	combined := "RUN "
	cmdParts := []string{}

	for _, cmd := range commands {
		// Remove "RUN " prefix
		cmdContent := strings.TrimPrefix(cmd, "RUN ")
		cmdContent = strings.TrimSpace(cmdContent)

		// Add cleanup for package managers
		if strings.Contains(cmdContent, "apt-get install") {
			if !strings.Contains(cmdContent, "apt-get clean") {
				cmdContent += " && apt-get clean && rm -rf /var/lib/apt/lists/*"
			}
		} else if strings.Contains(cmdContent, "yum install") {
			if !strings.Contains(cmdContent, "yum clean") {
				cmdContent += " && yum clean all"
			}
		} else if strings.Contains(cmdContent, "apk add") {
			if !strings.Contains(cmdContent, "rm -rf /var/cache/apk/*") {
				cmdContent += " && rm -rf /var/cache/apk/*"
			}
		}

		cmdParts = append(cmdParts, cmdContent)
	}

	combined += strings.Join(cmdParts, " && \\\n    ")

	result.Optimizations = append(result.Optimizations, Optimization{
		Type:        "combine-run",
		Description: fmt.Sprintf("Combined %d RUN commands to reduce layers", len(commands)),
		Impact:      "high",
		LinesBefore: len(commands),
		LinesAfter:  1,
	})

	return combined
}

// optimizeCopy optimizes COPY and ADD instructions
func (o *DockerfileOptimizer) optimizeCopy(line, instruction string, result *OptimizationResult) string {
	// Replace ADD with COPY when appropriate
	if instruction == "ADD" && !strings.Contains(line, "http") && !strings.Contains(line, ".tar") {
		optimized := strings.Replace(line, "ADD", "COPY", 1)
		result.Optimizations = append(result.Optimizations, Optimization{
			Type:        "add-to-copy",
			Description: "Replaced ADD with COPY for simple file operations",
			Impact:      "low",
		})
		return optimized
	}

	// Add --chown flag suggestion for COPY
	if instruction == "COPY" && !strings.Contains(line, "--chown") {
		// Check if we're copying executable files
		if strings.Contains(line, ".sh") || strings.Contains(line, "bin/") {
			result.Optimizations = append(result.Optimizations, Optimization{
				Type:        "copy-chown",
				Description: "Consider using --chown flag with COPY to avoid separate RUN chown",
				Impact:      "medium",
			})
		}
	}

	return line
}

// combineEnv combines multiple ENV instructions
func (o *DockerfileOptimizer) combineEnv(prev, current string, result *OptimizationResult) string {
	// Extract ENV values
	prevParts := strings.SplitN(prev, " ", 2)
	currParts := strings.SplitN(current, " ", 2)

	if len(prevParts) < 2 || len(currParts) < 2 {
		return current
	}

	// Combine ENV instructions
	combined := fmt.Sprintf("%s \\\n    %s", prev, currParts[1])

	result.Optimizations = append(result.Optimizations, Optimization{
		Type:        "combine-env",
		Description: "Combined ENV instructions",
		Impact:      "low",
		LinesBefore: 2,
		LinesAfter:  1,
	})

	return combined
}

// optimizeWorkdir optimizes WORKDIR instructions
func (o *DockerfileOptimizer) optimizeWorkdir(line string, result *OptimizationResult) string {
	// Suggest creating directory with WORKDIR instead of RUN mkdir
	parts := strings.Fields(line)
	if len(parts) > 1 {
		dir := parts[1]
		if !strings.HasPrefix(dir, "/") {
			result.Optimizations = append(result.Optimizations, Optimization{
				Type:        "workdir-absolute",
				Description: "Use absolute paths with WORKDIR for clarity",
				Impact:      "low",
			})
		}
	}

	return line
}
