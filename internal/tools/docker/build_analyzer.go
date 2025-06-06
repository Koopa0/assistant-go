package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

// BuildAnalyzer analyzes Docker builds for performance and efficiency
type BuildAnalyzer struct {
	logger *slog.Logger
}

// NewBuildAnalyzer creates a new build analyzer
func NewBuildAnalyzer(logger *slog.Logger) *BuildAnalyzer {
	return &BuildAnalyzer{
		logger: logger,
	}
}

// BuildAnalysisResult represents the analysis of a Docker build
type BuildAnalysisResult struct {
	TotalBuildTime   time.Duration     `json:"total_build_time"`
	Stages           []StageAnalysis   `json:"stages"`
	CacheUtilization CacheAnalysis     `json:"cache_utilization"`
	SizeAnalysis     ImageSizeAnalysis `json:"size_analysis"`
	Recommendations  []string          `json:"recommendations"`
}

// StageAnalysis represents analysis of a single build stage
type StageAnalysis struct {
	Name         string        `json:"name"`
	Duration     time.Duration `json:"duration"`
	CacheHit     bool          `json:"cache_hit"`
	LayerSize    int64         `json:"layer_size"`
	Instructions []string      `json:"instructions"`
}

// CacheAnalysis represents Docker build cache analysis
type CacheAnalysis struct {
	CacheHits   int     `json:"cache_hits"`
	CacheMisses int     `json:"cache_misses"`
	CacheRatio  float64 `json:"cache_ratio"`
}

// ImageSizeAnalysis represents size analysis of the built image
type ImageSizeAnalysis struct {
	TotalSize  int64       `json:"total_size"`
	LayerSizes []LayerSize `json:"layer_sizes"`
	BaseSize   int64       `json:"base_size"`
	AddedSize  int64       `json:"added_size"`
}

// LayerSize represents the size of a single layer
type LayerSize struct {
	ID      string `json:"id"`
	Size    int64  `json:"size"`
	Command string `json:"command"`
}

// AnalyzeBuild performs a build and analyzes its performance
func (a *BuildAnalyzer) AnalyzeBuild(dockerfilePath string) (*BuildAnalysisResult, error) {
	ctx := context.Background()
	startTime := time.Now()

	// Run build with BuildKit for better output
	buildCmd := exec.CommandContext(ctx, "docker", "build",
		"--progress=plain",
		"--no-cache", // Force rebuild to get accurate timing
		"-f", dockerfilePath,
		"-t", fmt.Sprintf("analyze-build-%d", time.Now().Unix()),
		".")

	output, err := buildCmd.CombinedOutput()
	if err != nil {
		// Even if build fails, we can still analyze what we have
		a.logger.Warn("Build failed, analyzing partial results",
			slog.String("error", err.Error()))
	}

	buildTime := time.Since(startTime)

	// Parse build output
	result := &BuildAnalysisResult{
		TotalBuildTime:   buildTime,
		Stages:           a.parseStages(string(output)),
		CacheUtilization: a.analyzeCacheUsage(string(output)),
		Recommendations:  []string{},
	}

	// If build succeeded, analyze the image
	if err == nil {
		imageName := fmt.Sprintf("analyze-build-%d", startTime.Unix())
		result.SizeAnalysis = a.analyzeImageSize(ctx, imageName)

		// Clean up the test image
		exec.CommandContext(ctx, "docker", "rmi", imageName).Run()
	}

	// Generate recommendations
	a.generateRecommendations(result)

	return result, nil
}

// parseStages parses build stages from Docker build output
func (a *BuildAnalyzer) parseStages(output string) []StageAnalysis {
	var stages []StageAnalysis
	lines := strings.Split(output, "\n")

	currentStage := &StageAnalysis{
		Instructions: []string{},
	}

	for _, line := range lines {
		// Look for stage markers
		if strings.Contains(line, "Step ") && strings.Contains(line, ":") {
			// Parse step information
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				instruction := strings.TrimSpace(parts[1])
				currentStage.Instructions = append(currentStage.Instructions, instruction)
			}
		}

		// Look for timing information
		if strings.Contains(line, "CACHED") {
			currentStage.CacheHit = true
		}

		// Look for stage completion
		if strings.Contains(line, "naming to") || strings.Contains(line, "Successfully built") {
			if len(currentStage.Instructions) > 0 {
				stages = append(stages, *currentStage)
				currentStage = &StageAnalysis{
					Instructions: []string{},
				}
			}
		}
	}

	// Add last stage if any
	if len(currentStage.Instructions) > 0 {
		stages = append(stages, *currentStage)
	}

	return stages
}

// analyzeCacheUsage analyzes Docker build cache usage
func (a *BuildAnalyzer) analyzeCacheUsage(output string) CacheAnalysis {
	analysis := CacheAnalysis{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "CACHED") {
			analysis.CacheHits++
		} else if strings.Contains(line, "Step ") && !strings.Contains(line, "CACHED") {
			analysis.CacheMisses++
		}
	}

	total := analysis.CacheHits + analysis.CacheMisses
	if total > 0 {
		analysis.CacheRatio = float64(analysis.CacheHits) / float64(total)
	}

	return analysis
}

// analyzeImageSize analyzes the size of the built image
func (a *BuildAnalyzer) analyzeImageSize(ctx context.Context, imageName string) ImageSizeAnalysis {
	analysis := ImageSizeAnalysis{
		LayerSizes: []LayerSize{},
	}

	// Get image history
	historyCmd := exec.CommandContext(ctx, "docker", "history", "--no-trunc", "--format", "json", imageName)
	output, err := historyCmd.Output()
	if err != nil {
		a.logger.Error("Failed to get image history",
			slog.String("error", err.Error()))
		return analysis
	}

	// Parse each line as JSON
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Extract layer information
		size := int64(0)
		if sizeStr, ok := entry["Size"].(string); ok {
			// Parse size (e.g., "1.2MB", "500KB")
			size = parseSize(sizeStr)
		}

		command := ""
		if cmd, ok := entry["CreatedBy"].(string); ok {
			// Truncate long commands
			if len(cmd) > 100 {
				command = cmd[:100] + "..."
			} else {
				command = cmd
			}
		}

		layer := LayerSize{
			Size:    size,
			Command: command,
		}

		analysis.LayerSizes = append(analysis.LayerSizes, layer)
		analysis.TotalSize += size
	}

	// Calculate base vs added size
	if len(analysis.LayerSizes) > 0 {
		// Assume last layer is base image
		analysis.BaseSize = analysis.LayerSizes[len(analysis.LayerSizes)-1].Size
		analysis.AddedSize = analysis.TotalSize - analysis.BaseSize
	}

	return analysis
}

// generateRecommendations generates build recommendations
func (a *BuildAnalyzer) generateRecommendations(result *BuildAnalysisResult) {
	// Check build time
	if result.TotalBuildTime > 5*time.Minute {
		result.Recommendations = append(result.Recommendations,
			"Build time exceeds 5 minutes - consider optimizing slow steps or using build cache")
	}

	// Check cache utilization
	if result.CacheUtilization.CacheRatio < 0.3 {
		result.Recommendations = append(result.Recommendations,
			"Low cache utilization detected - structure Dockerfile to maximize cache reuse")
	}

	// Check image size
	if result.SizeAnalysis.TotalSize > 1024*1024*1024 { // 1GB
		result.Recommendations = append(result.Recommendations,
			"Image size exceeds 1GB - consider using multi-stage builds or smaller base images")
	}

	// Check layer count
	if len(result.SizeAnalysis.LayerSizes) > 30 {
		result.Recommendations = append(result.Recommendations,
			"High layer count detected - combine RUN commands to reduce layers")
	}

	// Check for large individual layers
	for _, layer := range result.SizeAnalysis.LayerSizes {
		if layer.Size > 200*1024*1024 { // 200MB
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("Large layer detected (%dMB) - consider splitting or optimizing: %s",
					layer.Size/(1024*1024), layer.Command))
		}
	}

	// Add Go-specific recommendations if detected
	for _, stage := range result.Stages {
		for _, inst := range stage.Instructions {
			if strings.Contains(inst, "go build") {
				result.Recommendations = append(result.Recommendations,
					"For Go builds: use vendor directory or go mod download in separate layer")
				result.Recommendations = append(result.Recommendations,
					"Consider using scratch or distroless base image for Go binaries")
				break
			}
		}
	}
}

// parseSize parses Docker size strings like "1.2MB", "500KB"
func parseSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "0B" || sizeStr == "" {
		return 0
	}

	// Simple parsing - in production, use a proper parser
	multipliers := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
	}

	for suffix, multiplier := range multipliers {
		if strings.HasSuffix(sizeStr, suffix) {
			numStr := strings.TrimSuffix(sizeStr, suffix)
			var num float64
			fmt.Sscanf(numStr, "%f", &num)
			return int64(num * float64(multiplier))
		}
	}

	return 0
}
