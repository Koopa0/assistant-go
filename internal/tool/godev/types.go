package godev

import (
	"time"
)

// ProjectType represents the type of Go project
type ProjectType string

const (
	ProjectTypeCLI          ProjectType = "cli"
	ProjectTypeWebService   ProjectType = "web_service"
	ProjectTypeMicroservice ProjectType = "microservice"
	ProjectTypeLibrary      ProjectType = "library"
	ProjectTypeMonorepo     ProjectType = "monorepo"
	ProjectTypeUnknown      ProjectType = "unknown"
)

// WorkspaceInfo contains comprehensive information about a Go workspace
type WorkspaceInfo struct {
	// Basic project information
	RootPath    string      `json:"root_path"`
	ModulePath  string      `json:"module_path"`
	ProjectType ProjectType `json:"project_type"`
	GoVersion   string      `json:"go_version"`

	// Project structure
	Packages     []PackageInfo     `json:"packages"`
	Dependencies []DependencyInfo  `json:"dependencies"`
	TestCoverage *TestCoverageInfo `json:"test_coverage,omitempty"`

	// Build information
	BuildInfo *BuildInfo `json:"build_info,omitempty"`

	// Git information
	GitInfo *GitInfo `json:"git_info,omitempty"`

	// Analysis metadata
	AnalyzedAt time.Time `json:"analyzed_at"`
	Version    string    `json:"version"`
}

// PackageInfo represents information about a Go package
type PackageInfo struct {
	Name         string          `json:"name"`
	Path         string          `json:"path"`
	ImportPath   string          `json:"import_path"`
	IsMain       bool            `json:"is_main"`
	LineCount    int             `json:"line_count"`
	FileCount    int             `json:"file_count"`
	TestFiles    []string        `json:"test_files"`
	Functions    []FunctionInfo  `json:"functions"`
	Structs      []StructInfo    `json:"structs"`
	Interfaces   []InterfaceInfo `json:"interfaces"`
	Imports      []string        `json:"imports"`
	ExternalDeps []string        `json:"external_deps"`
}

// DependencyInfo represents a Go module dependency
type DependencyInfo struct {
	ModulePath  string    `json:"module_path"`
	Version     string    `json:"version"`
	IsIndirect  bool      `json:"is_indirect"`
	IsDev       bool      `json:"is_dev"`
	Size        int64     `json:"size,omitempty"`
	LastUpdated time.Time `json:"last_updated,omitempty"`
	License     string    `json:"license,omitempty"`
	Description string    `json:"description,omitempty"`
}

// FunctionInfo represents information about a Go function
type FunctionInfo struct {
	Name       string   `json:"name"`
	Package    string   `json:"package"`
	IsExported bool     `json:"is_exported"`
	LineStart  int      `json:"line_start"`
	LineEnd    int      `json:"line_end"`
	Complexity int      `json:"complexity"`
	Parameters []string `json:"parameters"`
	Returns    []string `json:"returns"`
	IsTest     bool     `json:"is_test"`
	IsBench    bool     `json:"is_bench"`
}

// StructInfo represents information about a Go struct
type StructInfo struct {
	Name       string       `json:"name"`
	Package    string       `json:"package"`
	IsExported bool         `json:"is_exported"`
	Fields     []FieldInfo  `json:"fields"`
	Methods    []MethodInfo `json:"methods"`
	Tags       []string     `json:"tags"`
}

// FieldInfo represents a struct field
type FieldInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Tag        string `json:"tag"`
	IsExported bool   `json:"is_exported"`
}

// MethodInfo represents a struct method
type MethodInfo struct {
	Name       string   `json:"name"`
	Receiver   string   `json:"receiver"`
	IsExported bool     `json:"is_exported"`
	Parameters []string `json:"parameters"`
	Returns    []string `json:"returns"`
}

// InterfaceInfo represents information about a Go interface
type InterfaceInfo struct {
	Name       string       `json:"name"`
	Package    string       `json:"package"`
	IsExported bool         `json:"is_exported"`
	Methods    []MethodInfo `json:"methods"`
}

// TestCoverageInfo represents test coverage information
type TestCoverageInfo struct {
	TotalLines      int                `json:"total_lines"`
	CoveredLines    int                `json:"covered_lines"`
	Percentage      float64            `json:"percentage"`
	PackageCoverage map[string]float64 `json:"package_coverage"`
	LastRun         time.Time          `json:"last_run"`
}

// BuildInfo represents build information
type BuildInfo struct {
	BinaryName  string            `json:"binary_name"`
	OutputSize  int64             `json:"output_size"`
	BuildTime   time.Duration     `json:"build_time"`
	GoVersion   string            `json:"go_version"`
	Platform    string            `json:"platform"`
	Tags        []string          `json:"tags"`
	LDFlags     string            `json:"ld_flags"`
	Environment map[string]string `json:"environment"`
	LastBuilt   time.Time         `json:"last_built"`
}

// GitInfo represents Git repository information
type GitInfo struct {
	IsRepo        bool      `json:"is_repo"`
	Branch        string    `json:"branch"`
	CommitHash    string    `json:"commit_hash"`
	CommitMessage string    `json:"commit_message"`
	CommitAuthor  string    `json:"commit_author"`
	CommitDate    time.Time `json:"commit_date"`
	IsDirty       bool      `json:"is_dirty"`
	RemoteURL     string    `json:"remote_url"`
	Tags          []string  `json:"tags"`
}

// AnalysisOptions configures workspace analysis
type AnalysisOptions struct {
	IncludeTestFiles    bool `json:"include_test_files"`
	IncludeDependencies bool `json:"include_dependencies"`
	IncludeGitInfo      bool `json:"include_git_info"`
	IncludeCoverage     bool `json:"include_coverage"`
	IncludeBuildInfo    bool `json:"include_build_info"`
	MaxDepth            int  `json:"max_depth"`
	ExcludeVendor       bool `json:"exclude_vendor"`
}

// DefaultAnalysisOptions returns sensible default options
func DefaultAnalysisOptions() *AnalysisOptions {
	return &AnalysisOptions{
		IncludeTestFiles:    true,
		IncludeDependencies: true,
		IncludeGitInfo:      true,
		IncludeCoverage:     false, // Can be slow
		IncludeBuildInfo:    false, // Can be slow
		MaxDepth:            10,
		ExcludeVendor:       true,
	}
}

// AnalysisResult represents the result of workspace analysis
type AnalysisResult struct {
	Workspace   *WorkspaceInfo `json:"workspace"`
	Issues      []Issue        `json:"issues"`
	Suggestions []Suggestion   `json:"suggestions"`
	Metrics     *Metrics       `json:"metrics"`
}

// Issue represents a code issue or problem
type Issue struct {
	Type       string `json:"type"`
	Severity   string `json:"severity"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Column     int    `json:"column"`
	Message    string `json:"message"`
	Rule       string `json:"rule"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Suggestion represents an improvement suggestion
type Suggestion struct {
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Title       string `json:"title"`
	Description string `json:"description"`
	File        string `json:"file,omitempty"`
	Example     string `json:"example,omitempty"`
}

// Metrics represents code metrics
type Metrics struct {
	TotalLines      int     `json:"total_lines"`
	CodeLines       int     `json:"code_lines"`
	CommentLines    int     `json:"comment_lines"`
	BlankLines      int     `json:"blank_lines"`
	TotalFiles      int     `json:"total_files"`
	TotalPackages   int     `json:"total_packages"`
	TotalFunctions  int     `json:"total_functions"`
	TotalStructs    int     `json:"total_structs"`
	TotalInterfaces int     `json:"total_interfaces"`
	AvgComplexity   float64 `json:"avg_complexity"`
	MaxComplexity   int     `json:"max_complexity"`
	TestCoverage    float64 `json:"test_coverage"`
}
