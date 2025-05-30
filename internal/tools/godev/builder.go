package godev

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/tools"
)

// GoBuilder is a tool for building Go applications
type GoBuilder struct {
	logger *slog.Logger
	config map[string]interface{}
}

// NewGoBuilder creates a new Go builder tool
func NewGoBuilder(config map[string]interface{}, logger *slog.Logger) (tools.Tool, error) {
	return &GoBuilder{
		logger: logger,
		config: config,
	}, nil
}

// Name returns the tool name
func (g *GoBuilder) Name() string {
	return "go_builder"
}

// Description returns the tool description
func (g *GoBuilder) Description() string {
	return "Builds Go applications with various configurations"
}

// Parameters returns the tool parameters schema
func (g *GoBuilder) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the main package to build",
				"default":     ".",
			},
			"output": map[string]interface{}{
				"type":        "string",
				"description": "Output binary path/name",
				"default":     "",
			},
			"target_os": map[string]interface{}{
				"type":        "string",
				"description": "Target operating system (GOOS)",
				"default":     runtime.GOOS,
			},
			"target_arch": map[string]interface{}{
				"type":        "string",
				"description": "Target architecture (GOARCH)",
				"default":     runtime.GOARCH,
			},
			"build_mode": map[string]interface{}{
				"type":        "string",
				"description": "Build mode (exe, c-archive, c-shared, default, shared, pie)",
				"default":     "default",
			},
			"ldflags": map[string]interface{}{
				"type":        "string",
				"description": "Linker flags (e.g., -s -w for smaller binaries)",
				"default":     "",
			},
			"tags": map[string]interface{}{
				"type":        "array",
				"description": "Build tags to include",
				"items": map[string]interface{}{
					"type": "string",
				},
				"default": []string{},
			},
			"trimpath": map[string]interface{}{
				"type":        "boolean",
				"description": "Remove file system paths from binary",
				"default":     false,
			},
			"race": map[string]interface{}{
				"type":        "boolean",
				"description": "Enable race detector",
				"default":     false,
			},
			"mod": map[string]interface{}{
				"type":        "string",
				"description": "Module download mode (readonly, vendor, mod)",
				"default":     "",
			},
			"clean": map[string]interface{}{
				"type":        "boolean",
				"description": "Clean build cache before building",
				"default":     false,
			},
			"install": map[string]interface{}{
				"type":        "boolean",
				"description": "Install the binary to GOPATH/bin",
				"default":     false,
			},
			"static": map[string]interface{}{
				"type":        "boolean",
				"description": "Build static binary (CGO_ENABLED=0)",
				"default":     false,
			},
		},
		"required": []string{},
	}
}

// Execute executes the Go builder
func (g *GoBuilder) Execute(ctx context.Context, input map[string]interface{}) (*tools.ToolResult, error) {
	startTime := time.Now()

	// Parse input parameters
	path := "."
	if p, ok := input["path"].(string); ok && p != "" {
		path = p
	}

	output := ""
	if o, ok := input["output"].(string); ok {
		output = o
	}

	targetOS := runtime.GOOS
	if os, ok := input["target_os"].(string); ok && os != "" {
		targetOS = os
	}

	targetArch := runtime.GOARCH
	if arch, ok := input["target_arch"].(string); ok && arch != "" {
		targetArch = arch
	}

	buildMode := "default"
	if mode, ok := input["build_mode"].(string); ok && mode != "" {
		buildMode = mode
	}

	ldflags := ""
	if flags, ok := input["ldflags"].(string); ok {
		ldflags = flags
	}

	tags := []string{}
	if t, ok := input["tags"].([]interface{}); ok {
		for _, tag := range t {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	trimpath := false
	if trim, ok := input["trimpath"].(bool); ok {
		trimpath = trim
	}

	race := false
	if r, ok := input["race"].(bool); ok {
		race = r
	}

	mod := ""
	if m, ok := input["mod"].(string); ok {
		mod = m
	}

	clean := false
	if c, ok := input["clean"].(bool); ok {
		clean = c
	}

	install := false
	if i, ok := input["install"].(bool); ok {
		install = i
	}

	static := false
	if s, ok := input["static"].(bool); ok {
		static = s
	}

	g.logger.Info("Starting Go build",
		slog.String("path", path),
		slog.String("output", output),
		slog.String("target_os", targetOS),
		slog.String("target_arch", targetArch),
		slog.String("build_mode", buildMode),
		slog.Bool("static", static))

	// Clean build cache if requested
	if clean {
		if err := g.cleanBuildCache(ctx); err != nil {
			g.logger.Warn("Failed to clean build cache", slog.Any("error", err))
		}
	}

	// Build the application
	result, err := g.buildApplication(ctx, path, output, targetOS, targetArch, buildMode,
		ldflags, tags, trimpath, race, mod, install, static)
	if err != nil {
		return &tools.ToolResult{
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: time.Since(startTime),
			Data:          result,
		}, err
	}

	return &tools.ToolResult{
		Success: true,
		Data:    result,
		Metadata: map[string]interface{}{
			"path":           path,
			"output":         output,
			"target_os":      targetOS,
			"target_arch":    targetArch,
			"build_mode":     buildMode,
			"static":         static,
			"execution_time": time.Since(startTime).String(),
		},
		ExecutionTime: time.Since(startTime),
	}, nil
}

// Health checks if the tool is healthy
func (g *GoBuilder) Health(ctx context.Context) error {
	// Check if go command is available
	cmd := exec.Command("go", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go command not available: %w", err)
	}
	return nil
}

// Close closes the tool and cleans up resources
func (g *GoBuilder) Close(ctx context.Context) error {
	g.logger.Debug("Go builder tool closed")
	return nil
}

// BuildResult represents the result of a build operation
type BuildResult struct {
	Success        bool               `json:"success"`
	BinaryPath     string             `json:"binary_path"`
	BinarySize     int64              `json:"binary_size"`
	BuildTime      time.Duration      `json:"build_time"`
	GoVersion      string             `json:"go_version"`
	TargetPlatform TargetPlatform     `json:"target_platform"`
	BuildInfo      BuildInfo          `json:"build_info"`
	Dependencies   []ModuleDependency `json:"dependencies,omitempty"`
	Output         string             `json:"output"`
}

// TargetPlatform represents the target platform for the build
type TargetPlatform struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	CGOEnabled   bool   `json:"cgo_enabled"`
}

// BuildInfo contains detailed build information
type BuildInfo struct {
	BuildMode    string   `json:"build_mode"`
	BuildTags    []string `json:"build_tags"`
	LDFlags      string   `json:"ldflags"`
	TrimPath     bool     `json:"trim_path"`
	RaceDetector bool     `json:"race_detector"`
	ModMode      string   `json:"mod_mode"`
	Static       bool     `json:"static"`
}

// ModuleDependency represents a Go module dependency
type ModuleDependency struct {
	Path     string `json:"path"`
	Version  string `json:"version"`
	Replace  string `json:"replace,omitempty"`
	Indirect bool   `json:"indirect"`
}

// buildApplication builds the Go application
func (g *GoBuilder) buildApplication(ctx context.Context, path, output, targetOS, targetArch, buildMode,
	ldflags string, tags []string, trimpath, race bool, mod string, install, static bool) (*BuildResult, error) {

	result := &BuildResult{
		TargetPlatform: TargetPlatform{
			OS:           targetOS,
			Architecture: targetArch,
			CGOEnabled:   !static,
		},
		BuildInfo: BuildInfo{
			BuildMode:    buildMode,
			BuildTags:    tags,
			LDFlags:      ldflags,
			TrimPath:     trimpath,
			RaceDetector: race,
			ModMode:      mod,
			Static:       static,
		},
	}

	buildStartTime := time.Now()

	// Get Go version
	if version, err := g.getGoVersion(ctx); err == nil {
		result.GoVersion = version
	}

	// Determine output path
	if output == "" {
		// Generate default output name
		baseName := filepath.Base(path)
		if baseName == "." || baseName == "/" {
			// Try to get module name
			if modName, err := g.getModuleName(ctx, path); err == nil {
				baseName = filepath.Base(modName)
			} else {
				baseName = "main"
			}
		}

		output = baseName
		if targetOS == "windows" {
			output += ".exe"
		}
	}

	// Build command
	var cmd *exec.Cmd
	if install {
		cmd = exec.CommandContext(ctx, "go", "install")
	} else {
		cmd = exec.CommandContext(ctx, "go", "build")
		cmd.Args = append(cmd.Args, "-o", output)
	}

	// Set environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOOS=%s", targetOS),
		fmt.Sprintf("GOARCH=%s", targetArch),
	)

	if static {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=0")
	}

	// Add build flags
	if buildMode != "default" && buildMode != "" {
		cmd.Args = append(cmd.Args, "-buildmode="+buildMode)
	}

	if ldflags != "" {
		cmd.Args = append(cmd.Args, "-ldflags", ldflags)
	}

	if len(tags) > 0 {
		cmd.Args = append(cmd.Args, "-tags", strings.Join(tags, ","))
	}

	if trimpath {
		cmd.Args = append(cmd.Args, "-trimpath")
	}

	if race {
		cmd.Args = append(cmd.Args, "-race")
	}

	if mod != "" {
		cmd.Args = append(cmd.Args, "-mod="+mod)
	}

	// Add package path
	cmd.Args = append(cmd.Args, path)

	// Set working directory
	cmd.Dir = filepath.Dir(path)

	// Execute build
	g.logger.Debug("Executing go build command",
		slog.String("command", strings.Join(cmd.Args, " ")),
		slog.Any("env", []string{
			fmt.Sprintf("GOOS=%s", targetOS),
			fmt.Sprintf("GOARCH=%s", targetArch),
			fmt.Sprintf("CGO_ENABLED=%d", map[bool]int{true: 1, false: 0}[!static]),
		}))

	buildOutput, err := cmd.CombinedOutput()
	result.Output = string(buildOutput)

	if err != nil {
		return result, fmt.Errorf("build failed: %w\nOutput: %s", err, buildOutput)
	}

	result.BuildTime = time.Since(buildStartTime)
	result.Success = true

	// Get binary info
	if !install {
		result.BinaryPath = output
		if info, err := os.Stat(output); err == nil {
			result.BinarySize = info.Size()
		}
	} else {
		// For install, try to find the binary in GOPATH/bin
		if gopath := os.Getenv("GOPATH"); gopath != "" {
			binName := filepath.Base(path)
			if targetOS == "windows" {
				binName += ".exe"
			}
			result.BinaryPath = filepath.Join(gopath, "bin", binName)
			if info, err := os.Stat(result.BinaryPath); err == nil {
				result.BinarySize = info.Size()
			}
		}
	}

	// Get module dependencies
	if deps, err := g.getModuleDependencies(ctx, path); err == nil {
		result.Dependencies = deps
	}

	g.logger.Info("Build completed successfully",
		slog.String("binary_path", result.BinaryPath),
		slog.Int64("binary_size", result.BinarySize),
		slog.Duration("build_time", result.BuildTime))

	return result, nil
}

// cleanBuildCache cleans the Go build cache
func (g *GoBuilder) cleanBuildCache(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "go", "clean", "-cache")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clean build cache: %w", err)
	}
	g.logger.Debug("Build cache cleaned")
	return nil
}

// getGoVersion gets the Go version
func (g *GoBuilder) getGoVersion(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse version from output
	versionStr := strings.TrimSpace(string(output))
	parts := strings.Fields(versionStr)
	if len(parts) >= 3 {
		return parts[2], nil // e.g., "go1.21.0"
	}
	return versionStr, nil
}

// getModuleName gets the module name from go.mod
func (g *GoBuilder) getModuleName(ctx context.Context, path string) (string, error) {
	// Find go.mod file
	dir := path
	if !filepath.IsAbs(dir) {
		var err error
		dir, err = filepath.Abs(dir)
		if err != nil {
			return "", err
		}
	}

	for {
		modPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(modPath); err == nil {
			// Read module name from go.mod
			cmd := exec.CommandContext(ctx, "go", "list", "-m")
			cmd.Dir = dir
			output, err := cmd.Output()
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(output)), nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("go.mod not found")
}

// getModuleDependencies gets module dependencies
func (g *GoBuilder) getModuleDependencies(ctx context.Context, path string) ([]ModuleDependency, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json", "all")
	cmd.Dir = filepath.Dir(path)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse JSON output
	var deps []ModuleDependency
	decoder := json.NewDecoder(strings.NewReader(string(output)))

	for {
		var mod struct {
			Path    string `json:"Path"`
			Version string `json:"Version"`
			Replace *struct {
				Path    string `json:"Path"`
				Version string `json:"Version"`
			} `json:"Replace,omitempty"`
			Indirect bool `json:"Indirect"`
			Main     bool `json:"Main"`
		}

		if err := decoder.Decode(&mod); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, err
		}

		// Skip main module
		if mod.Main {
			continue
		}

		dep := ModuleDependency{
			Path:     mod.Path,
			Version:  mod.Version,
			Indirect: mod.Indirect,
		}

		if mod.Replace != nil {
			dep.Replace = fmt.Sprintf("%s@%s", mod.Replace.Path, mod.Replace.Version)
		}

		deps = append(deps, dep)
	}

	return deps, nil
}
