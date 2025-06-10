package tool

import (
	"context"
)

// RegistryService defines the interface for interacting with the tool registry.
type RegistryService interface {
	Execute(ctx context.Context, name string, input *ToolInput, config *ToolConfig) (*ToolResult, error)
	IsRegistered(name string) bool
	ListTools() []ToolInfo
	GetToolInfo(name string) (*ToolInfo, error)
	Health(ctx context.Context) error
	Stats(ctx context.Context) (*RegistryStats, error) // Using the typed stats struct

	Register(name string, factory ToolFactory) error // Added
	Close(ctx context.Context) error                 // Added
    // GetTool(name string, config *ToolConfig) (Tool, error) // Keeping this commented as direct execution is often preferred
}

// Ensure that ToolInput, ToolConfig, ToolResult, ToolInfo, and RegistryStats types
// are accessible to this package or define them here if they are not.
// Given they are in the same `tool` package, they should be accessible.
// ToolFactory is also assumed to be accessible from within the tool package.
