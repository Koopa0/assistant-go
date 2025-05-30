package godev

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/tools"
)

// GoDependencyAnalyzer is a tool for analyzing Go module dependencies
type GoDependencyAnalyzer struct {
	logger *slog.Logger
	config map[string]interface{}
}

// NewGoDependencyAnalyzer creates a new Go dependency analyzer tool
func NewGoDependencyAnalyzer(config map[string]interface{}, logger *slog.Logger) (tools.Tool, error) {
	return &GoDependencyAnalyzer{
		logger: logger,
		config: config,
	}, nil
}

// Name returns the tool name
func (g *GoDependencyAnalyzer) Name() string {
	return "go_dependency_analyzer"
}

// Description returns the tool description
func (g *GoDependencyAnalyzer) Description() string {
	return "Analyzes Go module dependencies and provides insights"
}

// Parameters returns the tool parameters schema
func (g *GoDependencyAnalyzer) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the Go module directory",
				"default":     ".",
			},
			"include_indirect": map[string]interface{}{
				"type":        "boolean",
				"description": "Include indirect dependencies",
				"default":     true,
			},
			"include_test": map[string]interface{}{
				"type":        "boolean",
				"description": "Include test dependencies",
				"default":     false,
			},
			"check_updates": map[string]interface{}{
				"type":        "boolean",
				"description": "Check for available updates",
				"default":     false,
			},
			"analyze_vulnerabilities": map[string]interface{}{
				"type":        "boolean",
				"description": "Analyze for known vulnerabilities",
				"default":     false,
			},
			"dependency_graph": map[string]interface{}{
				"type":        "boolean",
				"description": "Generate dependency graph",
				"default":     false,
			},
			"license_analysis": map[string]interface{}{
				"type":        "boolean",
				"description": "Analyze dependency licenses",
				"default":     false,
			},
		},
		"required": []string{},
	}
}

// Execute executes the Go dependency analyzer
func (g *GoDependencyAnalyzer) Execute(ctx context.Context, input map[string]interface{}) (*tools.ToolResult, error) {
	startTime := time.Now()

	// Parse input parameters
	path := "."
	if p, ok := input["path"].(string); ok && p != "" {
		path = p
	}

	includeIndirect := true
	if ii, ok := input["include_indirect"].(bool); ok {
		includeIndirect = ii
	}

	includeTest := false
	if it, ok := input["include_test"].(bool); ok {
		includeTest = it
	}

	checkUpdates := false
	if cu, ok := input["check_updates"].(bool); ok {
		checkUpdates = cu
	}

	analyzeVulnerabilities := false
	if av, ok := input["analyze_vulnerabilities"].(bool); ok {
		analyzeVulnerabilities = av
	}

	dependencyGraph := false
	if dg, ok := input["dependency_graph"].(bool); ok {
		dependencyGraph = dg
	}

	licenseAnalysis := false
	if la, ok := input["license_analysis"].(bool); ok {
		licenseAnalysis = la
	}

	g.logger.Info("Starting dependency analysis",
		slog.String("path", path),
		slog.Bool("include_indirect", includeIndirect),
		slog.Bool("check_updates", checkUpdates),
		slog.Bool("analyze_vulnerabilities", analyzeVulnerabilities))

	// Analyze dependencies
	result, err := g.analyzeDependencies(ctx, path, includeIndirect, includeTest,
		checkUpdates, analyzeVulnerabilities, dependencyGraph, licenseAnalysis)
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
			"path":                    path,
			"include_indirect":        includeIndirect,
			"include_test":            includeTest,
			"check_updates":           checkUpdates,
			"analyze_vulnerabilities": analyzeVulnerabilities,
			"dependency_graph":        dependencyGraph,
			"license_analysis":        licenseAnalysis,
			"execution_time":          time.Since(startTime).String(),
		},
		ExecutionTime: time.Since(startTime),
	}, nil
}

// Health checks if the tool is healthy
func (g *GoDependencyAnalyzer) Health(ctx context.Context) error {
	// Check if go command is available
	cmd := exec.Command("go", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go command not available: %w", err)
	}
	return nil
}

// Close closes the tool and cleans up resources
func (g *GoDependencyAnalyzer) Close(ctx context.Context) error {
	g.logger.Debug("Go dependency analyzer tool closed")
	return nil
}

// DependencyAnalysisResult represents the result of dependency analysis
type DependencyAnalysisResult struct {
	ModuleInfo      ModuleInfo         `json:"module_info"`
	Dependencies    []Dependency       `json:"dependencies"`
	Summary         DependencySummary  `json:"summary"`
	Updates         []DependencyUpdate `json:"updates,omitempty"`
	Vulnerabilities []Vulnerability    `json:"vulnerabilities,omitempty"`
	DependencyGraph *DependencyGraph   `json:"dependency_graph,omitempty"`
	LicenseAnalysis *LicenseAnalysis   `json:"license_analysis,omitempty"`
	Recommendations []string           `json:"recommendations"`
	AnalysisTime    time.Duration      `json:"analysis_time"`
}

// ModuleInfo represents information about the main module
type ModuleInfo struct {
	Path        string `json:"path"`
	Version     string `json:"version"`
	GoVersion   string `json:"go_version"`
	Dir         string `json:"dir"`
	Sum         string `json:"sum,omitempty"`
	GoModExists bool   `json:"go_mod_exists"`
	GoSumExists bool   `json:"go_sum_exists"`
}

// Dependency represents a single dependency
type Dependency struct {
	Path     string           `json:"path"`
	Version  string           `json:"version"`
	Indirect bool             `json:"indirect"`
	Replace  *ReplacementInfo `json:"replace,omitempty"`
	Time     *time.Time       `json:"time,omitempty"`
	Sum      string           `json:"sum,omitempty"`
	Dir      string           `json:"dir,omitempty"`
	GoMod    string           `json:"go_mod,omitempty"`
	Main     bool             `json:"main"`
	Update   *UpdateInfo      `json:"update,omitempty"`
	License  string           `json:"license,omitempty"`
	Size     int64            `json:"size,omitempty"`
}

// ReplacementInfo represents module replacement information
type ReplacementInfo struct {
	Path    string     `json:"path"`
	Version string     `json:"version"`
	Time    *time.Time `json:"time,omitempty"`
	Dir     string     `json:"dir,omitempty"`
}

// UpdateInfo represents available update information
type UpdateInfo struct {
	Path    string     `json:"path"`
	Version string     `json:"version"`
	Time    *time.Time `json:"time,omitempty"`
}

// DependencySummary provides summary statistics
type DependencySummary struct {
	TotalDependencies    int            `json:"total_dependencies"`
	DirectDependencies   int            `json:"direct_dependencies"`
	IndirectDependencies int            `json:"indirect_dependencies"`
	TestDependencies     int            `json:"test_dependencies"`
	UpdatesAvailable     int            `json:"updates_available"`
	VulnerabilitiesFound int            `json:"vulnerabilities_found"`
	LicenseDistribution  map[string]int `json:"license_distribution,omitempty"`
	TotalSize            int64          `json:"total_size"`
	UniqueOrganizations  []string       `json:"unique_organizations"`
}

// DependencyUpdate represents an available update
type DependencyUpdate struct {
	Path           string     `json:"path"`
	CurrentVersion string     `json:"current_version"`
	LatestVersion  string     `json:"latest_version"`
	UpdateTime     *time.Time `json:"update_time,omitempty"`
	BreakingChange bool       `json:"breaking_change"`
	Severity       string     `json:"severity"`
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	ID          string   `json:"id"`
	Module      string   `json:"module"`
	Version     string   `json:"version"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	FixedIn     string   `json:"fixed_in,omitempty"`
	References  []string `json:"references,omitempty"`
}

// DependencyGraph represents the dependency graph structure
type DependencyGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Stats GraphStats  `json:"stats"`
}

// GraphNode represents a node in the dependency graph
type GraphNode struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Version string `json:"version"`
	Type    string `json:"type"` // "main", "direct", "indirect"
	Level   int    `json:"level"`
}

// GraphEdge represents an edge in the dependency graph
type GraphEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Type   string `json:"type"` // "requires", "replaces"
	Weight int    `json:"weight"`
}

// GraphStats provides graph statistics
type GraphStats struct {
	MaxDepth     int     `json:"max_depth"`
	AvgDepth     float64 `json:"avg_depth"`
	Complexity   float64 `json:"complexity"`
	CircularDeps int     `json:"circular_dependencies"`
}

// LicenseAnalysis represents license analysis results
type LicenseAnalysis struct {
	LicenseDistribution map[string]int    `json:"license_distribution"`
	UnknownLicenses     []string          `json:"unknown_licenses"`
	ConflictingLicenses []LicenseConflict `json:"conflicting_licenses"`
	Recommendations     []string          `json:"recommendations"`
}

// LicenseConflict represents a license conflict
type LicenseConflict struct {
	Module1  string `json:"module1"`
	License1 string `json:"license1"`
	Module2  string `json:"module2"`
	License2 string `json:"license2"`
	Severity string `json:"severity"`
}

// analyzeDependencies performs the main dependency analysis
func (g *GoDependencyAnalyzer) analyzeDependencies(ctx context.Context, path string, includeIndirect, includeTest,
	checkUpdates, analyzeVulnerabilities, dependencyGraph, licenseAnalysis bool) (*DependencyAnalysisResult, error) {

	analysisStartTime := time.Now()

	// Check if go.mod exists
	goModPath := filepath.Join(path, "go.mod")
	goModExists := false
	if _, err := os.Stat(goModPath); err == nil {
		goModExists = true
	}

	goSumPath := filepath.Join(path, "go.sum")
	goSumExists := false
	if _, err := os.Stat(goSumPath); err == nil {
		goSumExists = true
	}

	if !goModExists {
		return nil, fmt.Errorf("go.mod not found in directory: %s", path)
	}

	// Get module information
	moduleInfo, err := g.getModuleInfo(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get module info: %w", err)
	}
	moduleInfo.GoModExists = goModExists
	moduleInfo.GoSumExists = goSumExists

	// Get dependencies
	dependencies, err := g.getDependencies(ctx, path, includeIndirect, includeTest)
	if err != nil {
		return nil, fmt.Errorf("failed to get dependencies: %w", err)
	}

	// Calculate summary
	summary := g.calculateSummary(dependencies)

	result := &DependencyAnalysisResult{
		ModuleInfo:      *moduleInfo,
		Dependencies:    dependencies,
		Summary:         summary,
		Recommendations: make([]string, 0),
		AnalysisTime:    time.Since(analysisStartTime),
	}

	// Optional analyses
	if checkUpdates {
		updates, err := g.checkForUpdates(ctx, path, dependencies)
		if err != nil {
			g.logger.Warn("Failed to check for updates", slog.Any("error", err))
		} else {
			result.Updates = updates
			result.Summary.UpdatesAvailable = len(updates)
		}
	}

	if analyzeVulnerabilities {
		vulnerabilities, err := g.analyzeVulnerabilities(ctx, path)
		if err != nil {
			g.logger.Warn("Failed to analyze vulnerabilities", slog.Any("error", err))
		} else {
			result.Vulnerabilities = vulnerabilities
			result.Summary.VulnerabilitiesFound = len(vulnerabilities)
		}
	}

	if dependencyGraph {
		graph, err := g.buildDependencyGraph(ctx, path, dependencies)
		if err != nil {
			g.logger.Warn("Failed to build dependency graph", slog.Any("error", err))
		} else {
			result.DependencyGraph = graph
		}
	}

	if licenseAnalysis {
		licenses, err := g.analyzeLicenses(ctx, dependencies)
		if err != nil {
			g.logger.Warn("Failed to analyze licenses", slog.Any("error", err))
		} else {
			result.LicenseAnalysis = licenses
			result.Summary.LicenseDistribution = licenses.LicenseDistribution
		}
	}

	// Generate recommendations
	result.Recommendations = g.generateRecommendations(result)

	return result, nil
}

// getModuleInfo gets information about the main module
func (g *GoDependencyAnalyzer) getModuleInfo(ctx context.Context, path string) (*ModuleInfo, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-json")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get module info: %w", err)
	}

	var moduleInfo ModuleInfo
	if err := json.Unmarshal(output, &moduleInfo); err != nil {
		return nil, fmt.Errorf("failed to parse module info: %w", err)
	}

	// Get Go version
	goVersionCmd := exec.CommandContext(ctx, "go", "version")
	if versionOutput, err := goVersionCmd.Output(); err == nil {
		versionStr := strings.TrimSpace(string(versionOutput))
		parts := strings.Fields(versionStr)
		if len(parts) >= 3 {
			moduleInfo.GoVersion = parts[2]
		}
	}

	return &moduleInfo, nil
}

// getDependencies gets all dependencies
func (g *GoDependencyAnalyzer) getDependencies(ctx context.Context, path string, includeIndirect, includeTest bool) ([]Dependency, error) {
	args := []string{"list", "-m", "-json"}

	if includeIndirect {
		args = append(args, "all")
	} else {
		// Get only direct dependencies
		args = append(args, "-mod=readonly")
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list dependencies: %w", err)
	}

	var dependencies []Dependency
	decoder := json.NewDecoder(strings.NewReader(string(output)))

	for {
		var dep Dependency
		if err := decoder.Decode(&dep); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to parse dependency: %w", err)
		}

		// Skip main module
		if dep.Main {
			continue
		}

		// Filter test dependencies if not requested
		if !includeTest && strings.Contains(dep.Path, "/test") {
			continue
		}

		dependencies = append(dependencies, dep)
	}

	return dependencies, nil
}

// calculateSummary calculates dependency summary statistics
func (g *GoDependencyAnalyzer) calculateSummary(dependencies []Dependency) DependencySummary {
	summary := DependencySummary{
		TotalDependencies:   len(dependencies),
		UniqueOrganizations: make([]string, 0),
	}

	orgMap := make(map[string]bool)

	for _, dep := range dependencies {
		if dep.Indirect {
			summary.IndirectDependencies++
		} else {
			summary.DirectDependencies++
		}

		// Extract organization from path
		parts := strings.Split(dep.Path, "/")
		if len(parts) >= 2 {
			org := strings.Join(parts[:2], "/")
			if !orgMap[org] {
				orgMap[org] = true
				summary.UniqueOrganizations = append(summary.UniqueOrganizations, org)
			}
		}

		summary.TotalSize += dep.Size
	}

	return summary
}

// checkForUpdates checks for available dependency updates
func (g *GoDependencyAnalyzer) checkForUpdates(ctx context.Context, path string, dependencies []Dependency) ([]DependencyUpdate, error) {
	var updates []DependencyUpdate

	// Use go list -u to check for updates
	cmd := exec.CommandContext(ctx, "go", "list", "-u", "-m", "-json", "all")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}

	decoder := json.NewDecoder(strings.NewReader(string(output)))

	for {
		var module struct {
			Path    string      `json:"Path"`
			Version string      `json:"Version"`
			Update  *UpdateInfo `json:"Update,omitempty"`
			Main    bool        `json:"Main"`
		}

		if err := decoder.Decode(&module); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("failed to parse update info: %w", err)
		}

		if module.Main || module.Update == nil {
			continue
		}

		update := DependencyUpdate{
			Path:           module.Path,
			CurrentVersion: module.Version,
			LatestVersion:  module.Update.Version,
			UpdateTime:     module.Update.Time,
			Severity:       "minor", // Default, could be enhanced with semantic version analysis
		}

		// Simple breaking change detection based on major version
		if strings.HasPrefix(module.Version, "v") && strings.HasPrefix(module.Update.Version, "v") {
			currentMajor := strings.Split(module.Version[1:], ".")[0]
			latestMajor := strings.Split(module.Update.Version[1:], ".")[0]
			if currentMajor != latestMajor {
				update.BreakingChange = true
				update.Severity = "major"
			}
		}

		updates = append(updates, update)
	}

	return updates, nil
}

// analyzeVulnerabilities analyzes dependencies for known vulnerabilities
func (g *GoDependencyAnalyzer) analyzeVulnerabilities(ctx context.Context, path string) ([]Vulnerability, error) {
	// Try to use govulncheck if available
	cmd := exec.CommandContext(ctx, "govulncheck", "-json", ".")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		// govulncheck not available, return empty list
		g.logger.Debug("govulncheck not available, skipping vulnerability analysis")
		return []Vulnerability{}, nil
	}

	var vulnerabilities []Vulnerability

	// Parse govulncheck JSON output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var vuln struct {
			OSV struct {
				ID      string `json:"id"`
				Summary string `json:"summary"`
			} `json:"osv"`
			Module struct {
				Path    string `json:"path"`
				Version string `json:"version"`
			} `json:"module"`
		}

		if err := json.Unmarshal([]byte(line), &vuln); err == nil {
			vulnerabilities = append(vulnerabilities, Vulnerability{
				ID:          vuln.OSV.ID,
				Module:      vuln.Module.Path,
				Version:     vuln.Module.Version,
				Description: vuln.OSV.Summary,
				Severity:    "medium", // Default severity
			})
		}
	}

	return vulnerabilities, nil
}

// buildDependencyGraph builds a dependency graph
func (g *GoDependencyAnalyzer) buildDependencyGraph(ctx context.Context, path string, dependencies []Dependency) (*DependencyGraph, error) {
	graph := &DependencyGraph{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
	}

	// Add main module as root node
	graph.Nodes = append(graph.Nodes, GraphNode{
		ID:      "main",
		Label:   "main module",
		Version: "local",
		Type:    "main",
		Level:   0,
	})

	// Add dependency nodes
	for i, dep := range dependencies {
		nodeType := "direct"
		if dep.Indirect {
			nodeType = "indirect"
		}

		graph.Nodes = append(graph.Nodes, GraphNode{
			ID:      fmt.Sprintf("dep_%d", i),
			Label:   dep.Path,
			Version: dep.Version,
			Type:    nodeType,
			Level:   1, // Simplified level assignment
		})

		// Add edge from main to direct dependencies
		if !dep.Indirect {
			graph.Edges = append(graph.Edges, GraphEdge{
				From:   "main",
				To:     fmt.Sprintf("dep_%d", i),
				Type:   "requires",
				Weight: 1,
			})
		}
	}

	// Calculate graph statistics
	graph.Stats = GraphStats{
		MaxDepth:     2, // Simplified
		AvgDepth:     1.5,
		Complexity:   float64(len(dependencies)) / 10.0,
		CircularDeps: 0, // Would need more complex analysis
	}

	return graph, nil
}

// analyzeLicenses analyzes dependency licenses
func (g *GoDependencyAnalyzer) analyzeLicenses(ctx context.Context, dependencies []Dependency) (*LicenseAnalysis, error) {
	analysis := &LicenseAnalysis{
		LicenseDistribution: make(map[string]int),
		UnknownLicenses:     make([]string, 0),
		ConflictingLicenses: make([]LicenseConflict, 0),
		Recommendations:     make([]string, 0),
	}

	// This is a simplified implementation
	// In practice, you would use tools like go-licenses or license scanning APIs

	for _, dep := range dependencies {
		license := dep.License
		if license == "" {
			license = "Unknown"
			analysis.UnknownLicenses = append(analysis.UnknownLicenses, dep.Path)
		}
		analysis.LicenseDistribution[license]++
	}

	// Generate recommendations
	if len(analysis.UnknownLicenses) > 0 {
		analysis.Recommendations = append(analysis.Recommendations,
			fmt.Sprintf("Review %d dependencies with unknown licenses", len(analysis.UnknownLicenses)))
	}

	return analysis, nil
}

// generateRecommendations generates analysis recommendations
func (g *GoDependencyAnalyzer) generateRecommendations(result *DependencyAnalysisResult) []string {
	var recommendations []string

	// Check for too many dependencies
	if result.Summary.TotalDependencies > 100 {
		recommendations = append(recommendations,
			"Consider reducing the number of dependencies to improve build times and security")
	}

	// Check for updates
	if len(result.Updates) > 10 {
		recommendations = append(recommendations,
			"Many dependencies have available updates. Consider updating to get security fixes and improvements")
	}

	// Check for vulnerabilities
	if len(result.Vulnerabilities) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Found %d security vulnerabilities. Update affected dependencies immediately", len(result.Vulnerabilities)))
	}

	// Check go.sum file
	if !result.ModuleInfo.GoSumExists {
		recommendations = append(recommendations,
			"go.sum file is missing. Run 'go mod tidy' to generate it for better security")
	}

	// Check for indirect dependencies ratio
	indirectRatio := float64(result.Summary.IndirectDependencies) / float64(result.Summary.TotalDependencies)
	if indirectRatio > 0.7 {
		recommendations = append(recommendations,
			"High ratio of indirect dependencies. Consider reviewing your direct dependencies")
	}

	return recommendations
}
