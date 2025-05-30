package godev

import (
	"github.com/koopa0/assistant-go/internal/tools"
)

// RegisterGoTools registers all Go development tools
func RegisterGoTools(registry *tools.Registry) error {
	// Register Go Analyzer
	if err := registry.Register("go_analyzer", NewGoAnalyzer); err != nil {
		return err
	}

	// Register Go Formatter
	if err := registry.Register("go_formatter", NewGoFormatter); err != nil {
		return err
	}

	// Register Go Tester (placeholder for future implementation)
	if err := registry.Register("go_tester", NewGoTester); err != nil {
		return err
	}

	// Register Go Builder
	if err := registry.Register("go_builder", NewGoBuilder); err != nil {
		return err
	}

	// Register Go Dependency Analyzer
	if err := registry.Register("go_dependency_analyzer", NewGoDependencyAnalyzer); err != nil {
		return err
	}

	return nil
}
