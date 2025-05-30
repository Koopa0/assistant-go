package godev

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/tools"
)

// GoTester is a tool for running Go tests
type GoTester struct {
	logger *slog.Logger
	config map[string]interface{}
}

// NewGoTester creates a new Go tester tool
func NewGoTester(config map[string]interface{}, logger *slog.Logger) (tools.Tool, error) {
	return &GoTester{
		logger: logger,
		config: config,
	}, nil
}

// Name returns the tool name
func (g *GoTester) Name() string {
	return "go_tester"
}

// Description returns the tool description
func (g *GoTester) Description() string {
	return "Runs Go tests and provides coverage reports"
}

// Parameters returns the tool parameters schema
func (g *GoTester) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to package or directory to test",
				"default":     "./...",
			},
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Test name pattern to run (regex)",
				"default":     "",
			},
			"coverage": map[string]interface{}{
				"type":        "boolean",
				"description": "Generate coverage report",
				"default":     true,
			},
			"coverage_profile": map[string]interface{}{
				"type":        "string",
				"description": "Coverage profile file path",
				"default":     "coverage.out",
			},
			"verbose": map[string]interface{}{
				"type":        "boolean",
				"description": "Run tests in verbose mode",
				"default":     false,
			},
			"short": map[string]interface{}{
				"type":        "boolean",
				"description": "Run only short tests",
				"default":     false,
			},
			"timeout": map[string]interface{}{
				"type":        "string",
				"description": "Test timeout duration (e.g., 30s, 5m)",
				"default":     "10m",
			},
			"parallel": map[string]interface{}{
				"type":        "integer",
				"description": "Number of test binaries to run in parallel",
				"default":     0,
			},
			"benchmark": map[string]interface{}{
				"type":        "boolean",
				"description": "Run benchmarks",
				"default":     false,
			},
		},
		"required": []string{},
	}
}

// Execute executes the Go tester
func (g *GoTester) Execute(ctx context.Context, input map[string]interface{}) (*tools.ToolResult, error) {
	startTime := time.Now()

	// Parse input parameters
	path := "./..."
	if p, ok := input["path"].(string); ok && p != "" {
		path = p
	}

	pattern := ""
	if p, ok := input["pattern"].(string); ok {
		pattern = p
	}

	coverage := true
	if c, ok := input["coverage"].(bool); ok {
		coverage = c
	}

	coverageProfile := "coverage.out"
	if cp, ok := input["coverage_profile"].(string); ok && cp != "" {
		coverageProfile = cp
	}

	verbose := false
	if v, ok := input["verbose"].(bool); ok {
		verbose = v
	}

	short := false
	if s, ok := input["short"].(bool); ok {
		short = s
	}

	timeout := "10m"
	if t, ok := input["timeout"].(string); ok && t != "" {
		timeout = t
	}

	parallel := 0
	if p, ok := input["parallel"].(float64); ok {
		parallel = int(p)
	}

	benchmark := false
	if b, ok := input["benchmark"].(bool); ok {
		benchmark = b
	}

	g.logger.Info("Starting Go test execution",
		slog.String("path", path),
		slog.String("pattern", pattern),
		slog.Bool("coverage", coverage),
		slog.Bool("verbose", verbose),
		slog.Bool("benchmark", benchmark))

	// Run tests
	result, err := g.runTests(ctx, path, pattern, coverage, coverageProfile, verbose, short, timeout, parallel, benchmark)
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
			"path":             path,
			"pattern":          pattern,
			"coverage":         coverage,
			"coverage_profile": coverageProfile,
			"verbose":          verbose,
			"short":            short,
			"timeout":          timeout,
			"parallel":         parallel,
			"benchmark":        benchmark,
			"execution_time":   time.Since(startTime).String(),
		},
		ExecutionTime: time.Since(startTime),
	}, nil
}

// Health checks if the tool is healthy
func (g *GoTester) Health(ctx context.Context) error {
	// Check if go command is available
	cmd := exec.Command("go", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go command not available: %w", err)
	}
	return nil
}

// Close closes the tool and cleans up resources
func (g *GoTester) Close(ctx context.Context) error {
	g.logger.Debug("Go tester tool closed")
	return nil
}

// TestResult represents the result of test execution
type TestResult struct {
	Summary       TestSummary       `json:"summary"`
	Packages      []PackageResult   `json:"packages"`
	Coverage      *CoverageReport   `json:"coverage,omitempty"`
	Benchmarks    []BenchmarkResult `json:"benchmarks,omitempty"`
	FailedTests   []TestCase        `json:"failed_tests"`
	Output        string            `json:"output"`
	ExecutionTime time.Duration     `json:"execution_time"`
}

// TestSummary provides a summary of test execution
type TestSummary struct {
	TotalPackages   int           `json:"total_packages"`
	TotalTests      int           `json:"total_tests"`
	PassedTests     int           `json:"passed_tests"`
	FailedTests     int           `json:"failed_tests"`
	SkippedTests    int           `json:"skipped_tests"`
	TotalBenchmarks int           `json:"total_benchmarks"`
	ExecutionTime   time.Duration `json:"execution_time"`
	Success         bool          `json:"success"`
}

// PackageResult represents test results for a package
type PackageResult struct {
	Name          string        `json:"name"`
	Tests         []TestCase    `json:"tests"`
	Coverage      float64       `json:"coverage"`
	ExecutionTime time.Duration `json:"execution_time"`
	Success       bool          `json:"success"`
	Output        string        `json:"output"`
}

// TestCase represents a single test case
type TestCase struct {
	Name          string        `json:"name"`
	Package       string        `json:"package"`
	Status        TestStatus    `json:"status"`
	ExecutionTime time.Duration `json:"execution_time"`
	Output        string        `json:"output,omitempty"`
	Error         string        `json:"error,omitempty"`
}

// TestStatus represents the status of a test
type TestStatus string

const (
	TestStatusPass    TestStatus = "pass"
	TestStatusFail    TestStatus = "fail"
	TestStatusSkip    TestStatus = "skip"
	TestStatusTimeout TestStatus = "timeout"
)

// CoverageReport represents test coverage information
type CoverageReport struct {
	TotalCoverage   float64            `json:"total_coverage"`
	PackageCoverage map[string]float64 `json:"package_coverage"`
	FileCoverage    []FileCoverage     `json:"file_coverage"`
	UncoveredLines  []UncoveredLine    `json:"uncovered_lines"`
}

// FileCoverage represents coverage for a single file
type FileCoverage struct {
	File       string  `json:"file"`
	Coverage   float64 `json:"coverage"`
	Statements int     `json:"statements"`
	Covered    int     `json:"covered"`
	Uncovered  int     `json:"uncovered"`
}

// UncoveredLine represents an uncovered line in the code
type UncoveredLine struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Code   string `json:"code,omitempty"`
}

// BenchmarkResult represents a benchmark result
type BenchmarkResult struct {
	Name          string             `json:"name"`
	Iterations    int                `json:"iterations"`
	NsPerOp       float64            `json:"ns_per_op"`
	BytesPerOp    int64              `json:"bytes_per_op"`
	AllocsPerOp   int64              `json:"allocs_per_op"`
	MBPerSec      float64            `json:"mb_per_sec,omitempty"`
	CustomMetrics map[string]float64 `json:"custom_metrics,omitempty"`
}

// runTests executes the tests
func (g *GoTester) runTests(ctx context.Context, path, pattern string, coverage bool, coverageProfile string,
	verbose, short bool, timeout string, parallel int, benchmark bool) (*TestResult, error) {

	result := &TestResult{
		Packages:    make([]PackageResult, 0),
		FailedTests: make([]TestCase, 0),
		Benchmarks:  make([]BenchmarkResult, 0),
	}

	startTime := time.Now()

	// Build test command
	args := []string{"test", path}

	// Add test flags
	if pattern != "" {
		args = append(args, "-run", pattern)
	}

	if coverage {
		args = append(args, "-cover")
		if coverageProfile != "" {
			args = append(args, "-coverprofile="+coverageProfile)
		}
	}

	if verbose {
		args = append(args, "-v")
	}

	if short {
		args = append(args, "-short")
	}

	if timeout != "" {
		args = append(args, "-timeout", timeout)
	}

	if parallel > 0 {
		args = append(args, fmt.Sprintf("-parallel=%d", parallel))
	}

	if benchmark {
		args = append(args, "-bench=.")
	}

	// Add JSON output for parsing
	args = append(args, "-json")

	// Execute test command
	cmd := exec.CommandContext(ctx, "go", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	g.logger.Debug("Executing go test command",
		slog.String("command", strings.Join(cmd.Args, " ")))

	err := cmd.Run()

	// Parse JSON output
	if stdout.Len() > 0 {
		if parseErr := g.parseTestOutput(&stdout, result); parseErr != nil {
			g.logger.Warn("Failed to parse test output", slog.Any("error", parseErr))
		}
	}

	// Capture raw output
	result.Output = stdout.String()
	if stderr.Len() > 0 {
		result.Output += "\n\nSTDERR:\n" + stderr.String()
	}

	// Process coverage if enabled
	if coverage && coverageProfile != "" {
		if _, statErr := os.Stat(coverageProfile); statErr == nil {
			coverageReport, coverErr := g.processCoverage(coverageProfile)
			if coverErr != nil {
				g.logger.Warn("Failed to process coverage", slog.Any("error", coverErr))
			} else {
				result.Coverage = coverageReport
			}
		}
	}

	// Calculate summary
	result.ExecutionTime = time.Since(startTime)
	g.calculateSummary(result)

	// Check if tests failed
	if err != nil && result.Summary.FailedTests > 0 {
		// This is expected when tests fail
		g.logger.Info("Tests completed with failures",
			slog.Int("failed", result.Summary.FailedTests),
			slog.Int("passed", result.Summary.PassedTests))
	} else if err != nil {
		// Unexpected error
		return result, fmt.Errorf("test execution failed: %w", err)
	}

	return result, nil
}

// parseTestOutput parses JSON test output
func (g *GoTester) parseTestOutput(output *bytes.Buffer, result *TestResult) error {
	scanner := bufio.NewScanner(output)
	packageMap := make(map[string]*PackageResult)

	for scanner.Scan() {
		line := scanner.Text()

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // Skip non-JSON lines
		}

		action, _ := event["Action"].(string)
		pkg, _ := event["Package"].(string)
		test, _ := event["Test"].(string)

		// Get or create package result
		if pkg != "" && packageMap[pkg] == nil {
			packageMap[pkg] = &PackageResult{
				Name:  pkg,
				Tests: make([]TestCase, 0),
			}
		}

		switch action {
		case "pass", "fail", "skip":
			if test != "" && pkg != "" {
				testCase := TestCase{
					Name:    test,
					Package: pkg,
					Status:  g.actionToStatus(action),
				}

				if elapsed, ok := event["Elapsed"].(float64); ok {
					testCase.ExecutionTime = time.Duration(elapsed * float64(time.Second))
				}

				if output, ok := event["Output"].(string); ok {
					testCase.Output = output
				}

				packageMap[pkg].Tests = append(packageMap[pkg].Tests, testCase)

				if action == "fail" {
					result.FailedTests = append(result.FailedTests, testCase)
				}
			}

		case "output":
			if output, ok := event["Output"].(string); ok && pkg != "" {
				if packageMap[pkg] != nil {
					packageMap[pkg].Output += output
				}
			}

		case "bench":
			if test != "" {
				bench := g.parseBenchmarkOutput(event)
				if bench != nil {
					result.Benchmarks = append(result.Benchmarks, *bench)
				}
			}
		}
	}

	// Convert map to slice
	for _, pkg := range packageMap {
		result.Packages = append(result.Packages, *pkg)
	}

	return scanner.Err()
}

// actionToStatus converts test action to status
func (g *GoTester) actionToStatus(action string) TestStatus {
	switch action {
	case "pass":
		return TestStatusPass
	case "fail":
		return TestStatusFail
	case "skip":
		return TestStatusSkip
	default:
		return TestStatusFail
	}
}

// parseBenchmarkOutput parses benchmark results
func (g *GoTester) parseBenchmarkOutput(event map[string]interface{}) *BenchmarkResult {
	bench := &BenchmarkResult{
		CustomMetrics: make(map[string]float64),
	}

	if name, ok := event["Test"].(string); ok {
		bench.Name = name
	}

	// Parse benchmark metrics
	if n, ok := event["N"].(float64); ok {
		bench.Iterations = int(n)
	}

	if nsPerOp, ok := event["NsPerOp"].(float64); ok {
		bench.NsPerOp = nsPerOp
	}

	if bytesPerOp, ok := event["BytesPerOp"].(float64); ok {
		bench.BytesPerOp = int64(bytesPerOp)
	}

	if allocsPerOp, ok := event["AllocsPerOp"].(float64); ok {
		bench.AllocsPerOp = int64(allocsPerOp)
	}

	if mbPerSec, ok := event["MBPerSec"].(float64); ok {
		bench.MBPerSec = mbPerSec
	}

	return bench
}

// processCoverage processes coverage profile
func (g *GoTester) processCoverage(profilePath string) (*CoverageReport, error) {
	report := &CoverageReport{
		PackageCoverage: make(map[string]float64),
		FileCoverage:    make([]FileCoverage, 0),
		UncoveredLines:  make([]UncoveredLine, 0),
	}

	// Run go tool cover to get coverage percentage
	cmd := exec.Command("go", "tool", "cover", "-func="+profilePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run coverage tool: %w", err)
	}

	// Parse coverage output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 3 {
			if strings.HasSuffix(parts[len(parts)-1], "%") {
				coverageStr := strings.TrimSuffix(parts[len(parts)-1], "%")
				coverage, err := strconv.ParseFloat(coverageStr, 64)
				if err == nil {
					if parts[1] == "(statements)" {
						// Total coverage
						report.TotalCoverage = coverage
					} else {
						// Function or file coverage
						filePath := parts[0]
						report.FileCoverage = append(report.FileCoverage, FileCoverage{
							File:     filePath,
							Coverage: coverage,
						})
					}
				}
			}
		}
	}

	return report, nil
}

// calculateSummary calculates test summary
func (g *GoTester) calculateSummary(result *TestResult) {
	summary := TestSummary{
		ExecutionTime: result.ExecutionTime,
		Success:       true,
	}

	for _, pkg := range result.Packages {
		summary.TotalPackages++
		for _, test := range pkg.Tests {
			summary.TotalTests++
			switch test.Status {
			case TestStatusPass:
				summary.PassedTests++
			case TestStatusFail:
				summary.FailedTests++
				summary.Success = false
			case TestStatusSkip:
				summary.SkippedTests++
			}
		}
	}

	summary.TotalBenchmarks = len(result.Benchmarks)
	result.Summary = summary
}
