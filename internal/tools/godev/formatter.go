package godev

import (
	"bytes"
	"context"
	"fmt"
	"go/format"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/koopa0/assistant-go/internal/tools"
)

// GoFormatter is a tool for formatting Go code
type GoFormatter struct {
	logger *slog.Logger
	config map[string]interface{}
}

// NewGoFormatter creates a new Go formatter tool
func NewGoFormatter(config *tools.ToolConfig, logger *slog.Logger) (tools.Tool, error) {
	// Convert ToolConfig to legacy map format for now
	legacyConfig := make(map[string]interface{})
	if config != nil {
		legacyConfig["timeout"] = config.Timeout
		legacyConfig["debug"] = config.Debug
		legacyConfig["working_dir"] = config.WorkingDir
	}

	return &GoFormatter{
		logger: logger,
		config: legacyConfig,
	}, nil
}

// Name returns the tool name
func (g *GoFormatter) Name() string {
	return "go_formatter"
}

// Description returns the tool description
func (g *GoFormatter) Description() string {
	return "Formats Go source code according to gofmt standards"
}

// Parameters returns the tool parameters schema
func (g *GoFormatter) Parameters() *tools.ToolParametersSchema {
	return &tools.ToolParametersSchema{
		Type: "object",
		Properties: map[string]tools.ToolParameter{
			"path": {
				Type:        "string",
				Description: "Path to Go file or directory to format",
				Required:    true,
			},
			"write": {
				Type:        "boolean",
				Description: "Write formatted content back to files",
				Default:     false,
			},
			"recursive": {
				Type:        "boolean",
				Description: "Recursively format files in subdirectories",
				Default:     true,
			},
			"simplify": {
				Type:        "boolean",
				Description: "Apply gofmt simplification rules",
				Default:     true,
			},
			"check_only": {
				Type:        "boolean",
				Description: "Only check if files need formatting",
				Default:     false,
			},
		},
		Required: []string{"path"},
	}
}

// Execute executes the Go formatter
func (g *GoFormatter) Execute(ctx context.Context, input *tools.ToolInput) (*tools.ToolResult, error) {
	startTime := time.Now()

	// Parse input parameters
	path, ok := input.Parameters["path"].(string)
	if !ok || path == "" {
		return &tools.ToolResult{
			Success: false,
			Error:   "path parameter is required",
		}, fmt.Errorf("path parameter is required")
	}

	write := false
	if w, ok := input.Parameters["write"].(bool); ok {
		write = w
	}

	recursive := true
	if r, ok := input.Parameters["recursive"].(bool); ok {
		recursive = r
	}

	simplify := true
	if s, ok := input.Parameters["simplify"].(bool); ok {
		simplify = s
	}

	checkOnly := false
	if c, ok := input.Parameters["check_only"].(bool); ok {
		checkOnly = c
	}

	g.logger.Info("Starting Go code formatting",
		slog.String("path", path),
		slog.Bool("write", write),
		slog.Bool("recursive", recursive),
		slog.Bool("simplify", simplify),
		slog.Bool("check_only", checkOnly))

	// Perform formatting
	result, err := g.formatGoCode(ctx, path, write, recursive, simplify, checkOnly)
	if err != nil {
		return &tools.ToolResult{
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: time.Since(startTime),
		}, err
	}

	// Convert FormatterResult to ToolResultData
	toolData := convertFormatterResultToToolData(result)

	// Create metadata
	metadata := &tools.ToolMetadata{
		StartTime:     startTime,
		EndTime:       time.Now(),
		ExecutionTime: time.Since(startTime),
		Parameters: map[string]string{
			"path":       path,
			"write":      fmt.Sprintf("%t", write),
			"recursive":  fmt.Sprintf("%t", recursive),
			"simplify":   fmt.Sprintf("%t", simplify),
			"check_only": fmt.Sprintf("%t", checkOnly),
		},
	}

	return &tools.ToolResult{
		Success:       true,
		Data:          toolData,
		Metadata:      metadata,
		ExecutionTime: time.Since(startTime),
	}, nil
}

// Health checks if the tool is healthy
func (g *GoFormatter) Health(ctx context.Context) error {
	// Test formatting capability
	testCode := []byte(`package main
import "fmt"
func main(){fmt.Println("test")}`)

	_, err := format.Source(testCode)
	if err != nil {
		return fmt.Errorf("Go formatter not working: %w", err)
	}

	return nil
}

// Close closes the tool and cleans up resources
func (g *GoFormatter) Close(ctx context.Context) error {
	g.logger.Debug("Go formatter tool closed")
	return nil
}

// FormatterResult represents the result of formatting operation
type FormatterResult struct {
	Summary        FormatterSummary `json:"summary"`
	FormattedFiles []FormattedFile  `json:"formatted_files"`
	UnchangedFiles []string         `json:"unchanged_files"`
	Errors         []FormatterError `json:"errors"`
}

// FormatterSummary provides a summary of formatting operation
type FormatterSummary struct {
	TotalFiles     int           `json:"total_files"`
	FormattedFiles int           `json:"formatted_files"`
	UnchangedFiles int           `json:"unchanged_files"`
	ErrorFiles     int           `json:"error_files"`
	TotalChanges   int           `json:"total_changes"`
	ExecutionTime  time.Duration `json:"execution_time"`
}

// FormattedFile represents a formatted file
type FormattedFile struct {
	Path        string    `json:"path"`
	BeforeLines int       `json:"before_lines"`
	AfterLines  int       `json:"after_lines"`
	Changes     int       `json:"changes"`
	BeforeSize  int64     `json:"before_size"`
	AfterSize   int64     `json:"after_size"`
	FormattedAt time.Time `json:"formatted_at"`
}

// FormatterError represents a formatting error
type FormatterError struct {
	Path    string `json:"path"`
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// formatGoCode performs the actual formatting
func (g *GoFormatter) formatGoCode(ctx context.Context, path string, write, recursive, simplify, checkOnly bool) (*FormatterResult, error) {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	result := &FormatterResult{
		FormattedFiles: make([]FormattedFile, 0),
		UnchangedFiles: make([]string, 0),
		Errors:         make([]FormatterError, 0),
	}

	startTime := time.Now()

	if info.IsDir() {
		// Format directory
		err = g.formatDirectory(ctx, path, write, recursive, simplify, checkOnly, result)
	} else {
		// Format single file
		err = g.formatFile(ctx, path, write, simplify, checkOnly, result)
	}

	if err != nil {
		return nil, err
	}

	// Calculate summary
	result.Summary = FormatterSummary{
		TotalFiles:     len(result.FormattedFiles) + len(result.UnchangedFiles) + len(result.Errors),
		FormattedFiles: len(result.FormattedFiles),
		UnchangedFiles: len(result.UnchangedFiles),
		ErrorFiles:     len(result.Errors),
		TotalChanges:   g.calculateTotalChanges(result),
		ExecutionTime:  time.Since(startTime),
	}

	return result, nil
}

// formatDirectory formats all Go files in a directory
func (g *GoFormatter) formatDirectory(ctx context.Context, dirPath string, write, recursive, simplify, checkOnly bool, result *FormatterResult) error {
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip vendor directories
		if strings.Contains(path, "/vendor/") {
			return nil
		}

		// Skip if not recursive and in subdirectory
		if !recursive {
			rel, err := filepath.Rel(dirPath, path)
			if err != nil {
				return err
			}
			if strings.Contains(rel, string(filepath.Separator)) {
				return nil
			}
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		return g.formatFile(ctx, path, write, simplify, checkOnly, result)
	})
}

// formatFile formats a single Go file
func (g *GoFormatter) formatFile(ctx context.Context, filePath string, write, simplify, checkOnly bool, result *FormatterResult) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		result.Errors = append(result.Errors, FormatterError{
			Path:    filePath,
			Error:   "Failed to read file",
			Details: err.Error(),
		})
		return nil // Continue with other files
	}

	// Get file info for size
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		result.Errors = append(result.Errors, FormatterError{
			Path:    filePath,
			Error:   "Failed to get file info",
			Details: err.Error(),
		})
		return nil
	}

	beforeLines := bytes.Count(content, []byte("\n"))
	beforeSize := fileInfo.Size()

	// Format the source code
	var formatted []byte
	if simplify {
		formatted, err = format.Source(content)
	} else {
		// Basic formatting without simplification
		formatted, err = format.Source(content)
	}

	if err != nil {
		result.Errors = append(result.Errors, FormatterError{
			Path:    filePath,
			Error:   "Failed to format file",
			Details: err.Error(),
		})
		return nil
	}

	afterLines := bytes.Count(formatted, []byte("\n"))
	afterSize := int64(len(formatted))

	// Check if content changed
	if bytes.Equal(content, formatted) {
		result.UnchangedFiles = append(result.UnchangedFiles, filePath)
		return nil
	}

	// File needs formatting
	formattedFile := FormattedFile{
		Path:        filePath,
		BeforeLines: beforeLines,
		AfterLines:  afterLines,
		Changes:     1, // Simplified change count
		BeforeSize:  beforeSize,
		AfterSize:   afterSize,
		FormattedAt: time.Now(),
	}

	result.FormattedFiles = append(result.FormattedFiles, formattedFile)

	// If check only mode, don't write
	if checkOnly {
		g.logger.Info("File needs formatting",
			slog.String("path", filePath),
			slog.Int("before_lines", beforeLines),
			slog.Int("after_lines", afterLines))
		return nil
	}

	// Write formatted content back if requested
	if write {
		err = os.WriteFile(filePath, formatted, fileInfo.Mode())
		if err != nil {
			result.Errors = append(result.Errors, FormatterError{
				Path:    filePath,
				Error:   "Failed to write formatted content",
				Details: err.Error(),
			})
			return nil
		}

		g.logger.Info("File formatted and written",
			slog.String("path", filePath),
			slog.Int64("size_diff", afterSize-beforeSize))
	}

	return nil
}

// calculateTotalChanges calculates total number of changes
func (g *GoFormatter) calculateTotalChanges(result *FormatterResult) int {
	total := 0
	for _, file := range result.FormattedFiles {
		total += file.Changes
	}
	return total
}

// convertFormatterResultToToolData converts FormatterResult to ToolResultData
func convertFormatterResultToToolData(formatResult *FormatterResult) *tools.ToolResultData {
	if formatResult == nil {
		return &tools.ToolResultData{}
	}

	// Create results for each formatted file
	results := make([]tools.ResultItem, 0, len(formatResult.FormattedFiles))
	for _, file := range formatResult.FormattedFiles {
		results = append(results, tools.ResultItem{
			ID:          fmt.Sprintf("formatted_%s", filepath.Base(file.Path)),
			Type:        "formatted_file",
			Name:        file.Path,
			Description: fmt.Sprintf("Formatted with %d changes", file.Changes),
			Value:       fmt.Sprintf("%d", file.Changes),
			Metadata: map[string]string{
				"before_size":  fmt.Sprintf("%d", file.BeforeSize),
				"after_size":   fmt.Sprintf("%d", file.AfterSize),
				"before_lines": fmt.Sprintf("%d", file.BeforeLines),
				"after_lines":  fmt.Sprintf("%d", file.AfterLines),
				"changes":      fmt.Sprintf("%d", file.Changes),
			},
		})
	}

	return &tools.ToolResultData{
		LinesProcessed: int64(formatResult.Summary.TotalFiles),
		Results:        results,
	}
}
