// Package docker provides Docker integration tools for the Assistant.
// It includes functionality for container management, Dockerfile optimization,
// and multi-stage build analysis.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/tool"
)

// DockerTool implements the Tool interface for Docker operations
type DockerTool struct {
	logger *slog.Logger
}

// NewDockerTool creates a new Docker tool instance
func NewDockerTool(logger *slog.Logger) *DockerTool {
	return &DockerTool{
		logger: logger,
	}
}

// Name returns the tool name
func (t *DockerTool) Name() string {
	return "docker"
}

// Description returns the tool description
func (t *DockerTool) Description() string {
	return "Docker container management, Dockerfile optimization, and build analysis"
}

// Parameters returns the tool parameter schema
func (t *DockerTool) Parameters() *tool.ToolParametersSchema {
	return &tool.ToolParametersSchema{
		Type: "object",
		Properties: map[string]tool.ParameterProperty{
			"action": {
				Type:        tool.ParameterTypeString,
				Description: "The Docker action to perform",
				Enum: []string{
					"list_containers",
					"list_images",
					"analyze_dockerfile",
					"optimize_dockerfile",
					"inspect_container",
					"container_logs",
					"build_analyze",
				},
			},
			"dockerfile_path": {
				Type:        tool.ParameterTypeString,
				Description: "Path to the Dockerfile",
			},
			"container_id": {
				Type:        tool.ParameterTypeString,
				Description: "Container ID or name",
			},
			"image_name": {
				Type:        tool.ParameterTypeString,
				Description: "Docker image name",
			},
			"options": {
				Type:        tool.ParameterTypeObject,
				Description: "Additional options for the action",
			},
		},
		Required: []string{"action"},
	}
}

// Execute runs the Docker tool with the given parameters
func (t *DockerTool) Execute(ctx context.Context, input *tool.ToolInput) (*tool.ToolResult, error) {
	startTime := time.Now()

	// Extract parameters from input
	params := input.Parameters
	if params == nil {
		params = make(map[string]interface{})
	}

	action, ok := params["action"].(string)
	if !ok {
		return &tool.ToolResult{
			Success: false,
			Error:   "action parameter is required",
		}, nil
	}

	t.logger.Info("Executing Docker action",
		slog.String("action", action))

	var result interface{}
	var err error

	switch action {
	case "list_containers":
		result, err = t.listContainers(ctx, params)
	case "list_images":
		result, err = t.listImages(ctx, params)
	case "analyze_dockerfile":
		result, err = t.analyzeDockerfile(ctx, params)
	case "optimize_dockerfile":
		result, err = t.optimizeDockerfile(ctx, params)
	case "inspect_container":
		result, err = t.inspectContainer(ctx, params)
	case "container_logs":
		result, err = t.getContainerLogs(ctx, params)
	case "build_analyze":
		result, err = t.analyzeBuild(ctx, params)
	default:
		return &tool.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown action: %s", action),
		}, nil
	}

	if err != nil {
		return &tool.ToolResult{
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: time.Since(startTime),
		}, nil
	}

	// Convert result to map[string]interface{} for output
	var outputMap map[string]interface{}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return &tool.ToolResult{
			Success:       false,
			Error:         fmt.Sprintf("failed to marshal result: %v", err),
			ExecutionTime: time.Since(startTime),
		}, nil
	}

	if err := json.Unmarshal(resultJSON, &outputMap); err != nil {
		// If we can't unmarshal to map, return the result directly
		outputMap = map[string]interface{}{
			"result": result,
		}
	}

	return &tool.ToolResult{
		Success: true,
		Data: &tool.ToolResultData{
			Output: outputMap,
		},
		ExecutionTime: time.Since(startTime),
	}, nil
}

// listContainers lists all Docker containers
func (t *DockerTool) listContainers(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	options := extractOptions(params)

	args := []string{"ps", "--format", "json"}
	if options["all"] == "true" {
		args = append(args, "-a")
	}

	output, err := t.runDockerCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Parse JSON output
	var containers []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var container map[string]interface{}
		if err := json.Unmarshal([]byte(line), &container); err != nil {
			t.logger.Warn("Failed to parse container JSON", slog.String("line", line))
			continue
		}
		containers = append(containers, container)
	}

	return map[string]interface{}{
		"containers": containers,
		"count":      len(containers),
	}, nil
}

// listImages lists all Docker images
func (t *DockerTool) listImages(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	output, err := t.runDockerCommand(ctx, "images", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	// Parse JSON output
	var images []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var image map[string]interface{}
		if err := json.Unmarshal([]byte(line), &image); err != nil {
			t.logger.Warn("Failed to parse image JSON", slog.String("line", line))
			continue
		}
		images = append(images, image)
	}

	return map[string]interface{}{
		"images": images,
		"count":  len(images),
	}, nil
}

// analyzeDockerfile analyzes a Dockerfile for best practices
func (t *DockerTool) analyzeDockerfile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	dockerfilePath, ok := params["dockerfile_path"].(string)
	if !ok {
		dockerfilePath = "Dockerfile"
	}

	analyzer := NewDockerfileAnalyzer(t.logger)
	result, err := analyzer.AnalyzeFile(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze Dockerfile: %w", err)
	}

	return result, nil
}

// optimizeDockerfile suggests optimizations for a Dockerfile
func (t *DockerTool) optimizeDockerfile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	dockerfilePath, ok := params["dockerfile_path"].(string)
	if !ok {
		dockerfilePath = "Dockerfile"
	}

	optimizer := NewDockerfileOptimizer(t.logger)
	result, err := optimizer.OptimizeFile(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to optimize Dockerfile: %w", err)
	}

	return result, nil
}

// inspectContainer inspects a Docker container
func (t *DockerTool) inspectContainer(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	containerID, ok := params["container_id"].(string)
	if !ok {
		return nil, fmt.Errorf("container_id parameter is required")
	}

	output, err := t.runDockerCommand(ctx, "inspect", containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil, fmt.Errorf("failed to parse inspect output: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("container not found: %s", containerID)
	}

	return result[0], nil
}

// getContainerLogs retrieves logs from a Docker container
func (t *DockerTool) getContainerLogs(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	containerID, ok := params["container_id"].(string)
	if !ok {
		return nil, fmt.Errorf("container_id parameter is required")
	}

	options := extractOptions(params)
	args := []string{"logs"}

	if tail, ok := options["tail"].(string); ok {
		args = append(args, "--tail", tail)
	}
	if options["timestamps"] == "true" {
		args = append(args, "--timestamps")
	}

	args = append(args, containerID)

	output, err := t.runDockerCommand(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}

	return map[string]interface{}{
		"container_id": containerID,
		"logs":         output,
	}, nil
}

// analyzeBuild analyzes a Docker build for performance and efficiency
func (t *DockerTool) analyzeBuild(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	dockerfilePath, ok := params["dockerfile_path"].(string)
	if !ok {
		dockerfilePath = "Dockerfile"
	}

	analyzer := NewBuildAnalyzer(t.logger)
	result, err := analyzer.AnalyzeBuild(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze build: %w", err)
	}

	return result, nil
}

// runDockerCommand executes a Docker command and returns its output
func (t *DockerTool) runDockerCommand(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)

	t.logger.Debug("Running Docker command",
		slog.String("command", fmt.Sprintf("docker %s", strings.Join(args, " "))))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker command failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// extractOptions extracts options from parameters
func extractOptions(params map[string]interface{}) map[string]interface{} {
	if options, ok := params["options"].(map[string]interface{}); ok {
		return options
	}
	return make(map[string]interface{})
}

// Health checks if the Docker tool is healthy
func (t *DockerTool) Health(ctx context.Context) error {
	// Check if Docker is available
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker health check failed: %w\nOutput: %s", err, string(output))
	}

	// Try to parse the version output
	var versionInfo map[string]interface{}
	if err := json.Unmarshal(output, &versionInfo); err != nil {
		// Docker is running but output format might be different
		t.logger.Warn("Failed to parse Docker version output", slog.String("output", string(output)))
	}

	t.logger.Debug("Docker health check passed")
	return nil
}

// Close closes the Docker tool and cleans up resources
func (t *DockerTool) Close(ctx context.Context) error {
	// Docker tool doesn't maintain persistent connections
	// Nothing to clean up
	t.logger.Debug("Docker tool closed")
	return nil
}
