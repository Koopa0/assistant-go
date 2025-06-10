package godev

import (
	"context"
)

// DetectorService defines the interface for workspace detection and analysis.
// It is used by GoDevTool to perform the actual analysis of a Go workspace.
type DetectorService interface {
	DetectWorkspace(ctx context.Context, startPath string, options *AnalysisOptions) (*AnalysisResult, error)
}

// Note: AnalysisOptions and AnalysisResult types are expected to be defined
// in other .go files within the godev package (e.g., types.go or workspace.go).
