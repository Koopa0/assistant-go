package assistant

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

// EnvironmentDetector detects environment and workspace information
type EnvironmentDetector struct {
	startTime time.Time
}

// NewEnvironmentDetector creates a new environment detector
func NewEnvironmentDetector() *EnvironmentDetector {
	return &EnvironmentDetector{
		startTime: time.Now(),
	}
}

// GetVersion retrieves the build version from build info
func (d *EnvironmentDetector) GetVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	// Fallback to environment variable or default
	if version := os.Getenv("APP_VERSION"); version != "" {
		return version
	}
	return "1.0.0"
}

// GetUptime returns the time since the process started
func (d *EnvironmentDetector) GetUptime() time.Duration {
	return time.Since(d.startTime)
}

// DetectFramework detects the framework from go.mod
func (d *EnvironmentDetector) DetectFramework(projectPath string) string {
	goModPath := filepath.Join(projectPath, "go.mod")
	file, err := os.Open(goModPath)
	if err != nil {
		return "standard_library"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	frameworks := map[string]string{
		"github.com/gin-gonic/gin":   "gin",
		"github.com/labstack/echo":   "echo",
		"github.com/gofiber/fiber":   "fiber",
		"github.com/gorilla/mux":     "gorilla",
		"github.com/go-chi/chi":      "chi",
		"github.com/tmc/langchaingo": "langchain",
		"github.com/jackc/pgx":       "pgx",
	}

	detectedFrameworks := []string{}
	for scanner.Scan() {
		line := scanner.Text()
		for pkg, framework := range frameworks {
			if strings.Contains(line, pkg) {
				detectedFrameworks = append(detectedFrameworks, framework)
			}
		}
	}

	if len(detectedFrameworks) > 0 {
		return strings.Join(detectedFrameworks, ",")
	}
	return "standard_library"
}

// GetProjectPath gets the project path from environment or working directory
func (d *EnvironmentDetector) GetProjectPath() string {
	// Check environment variable first
	if path := os.Getenv("PROJECT_PATH"); path != "" {
		return path
	}

	// Try to get from working directory
	if wd, err := os.Getwd(); err == nil {
		return wd
	}

	return ""
}

// DetectGitInfo detects git repository information
func (d *EnvironmentDetector) DetectGitInfo(projectPath string) (repo string, branch string) {
	// Get repository name
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = projectPath
	if output, err := cmd.Output(); err == nil {
		url := strings.TrimSpace(string(output))
		// Extract repo name from URL
		parts := strings.Split(url, "/")
		if len(parts) > 0 {
			repo = strings.TrimSuffix(parts[len(parts)-1], ".git")
		}
	}

	// Get current branch
	cmd = exec.Command("git", "branch", "--show-current")
	cmd.Dir = projectPath
	if output, err := cmd.Output(); err == nil {
		branch = strings.TrimSpace(string(output))
	}

	if repo == "" {
		repo = "unknown"
	}
	if branch == "" {
		branch = "unknown"
	}

	return repo, branch
}

// DetectAvailableTools detects which tools are available
func (d *EnvironmentDetector) DetectAvailableTools() []string {
	tools := []string{}

	// Check for common development tools
	toolChecks := map[string]string{
		"go":      "go_analyzer",
		"docker":  "docker",
		"kubectl": "kubernetes",
		"git":     "git",
		"make":    "make",
		"psql":    "postgres",
	}

	for cmd, toolName := range toolChecks {
		if _, err := exec.LookPath(cmd); err == nil {
			tools = append(tools, toolName)
		}
	}

	return tools
}

// DetectWorkspace performs comprehensive workspace detection
func (d *EnvironmentDetector) DetectWorkspace() *WorkspaceContext {
	projectPath := d.GetProjectPath()
	repo, branch := d.DetectGitInfo(projectPath)
	framework := d.DetectFramework(projectPath)
	tools := d.DetectAvailableTools()

	// Detect project type and structure
	projectType := "go_project"
	structureType := "monorepo"

	// Check for common config files
	configFiles := []string{}
	potentialConfigs := []string{
		"go.mod", "go.sum",
		"Makefile", "Dockerfile", "docker-compose.yml",
		".env", ".env.example",
		"configs/development.yaml", "configs/production.yaml",
	}

	for _, config := range potentialConfigs {
		if _, err := os.Stat(filepath.Join(projectPath, config)); err == nil {
			configFiles = append(configFiles, config)
		}
	}

	// Detect main dependencies from go.mod
	dependencies := d.detectDependencies(projectPath)

	return &WorkspaceContext{
		ProjectType:        projectType,
		Languages:          []string{"go"},
		Framework:          framework,
		Dependencies:       dependencies,
		StructureType:      structureType,
		ConfigFiles:        configFiles,
		DocumentationStyle: "godoc",
		Metadata: map[string]string{
			"project_path":    projectPath,
			"git_repository":  repo,
			"git_branch":      branch,
			"tools_available": strings.Join(tools, ","),
			"detected_at":     time.Now().Format(time.RFC3339),
		},
	}
}

// detectDependencies extracts main dependencies from go.mod
func (d *EnvironmentDetector) detectDependencies(projectPath string) []string {
	goModPath := filepath.Join(projectPath, "go.mod")
	file, err := os.Open(goModPath)
	if err != nil {
		return []string{}
	}
	defer file.Close()

	deps := []string{}
	scanner := bufio.NewScanner(file)
	inRequire := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "require (" {
			inRequire = true
			continue
		}

		if inRequire && line == ")" {
			break
		}

		if inRequire && line != "" && !strings.HasPrefix(line, "//") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				dep := parts[0]
				// Only include major dependencies
				if strings.Contains(dep, "github.com") || strings.Contains(dep, "golang.org") {
					// Extract package name
					depParts := strings.Split(dep, "/")
					if len(depParts) >= 3 {
						deps = append(deps, fmt.Sprintf("%s/%s", depParts[1], depParts[2]))
					}
				}
			}
		}
	}

	// Limit to top 10 dependencies
	if len(deps) > 10 {
		deps = deps[:10]
	}

	return deps
}
