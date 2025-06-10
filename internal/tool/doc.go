// Package tool provides the framework for defining, managing, and executing
// tools (also referred to as skills or capabilities) that the AI assistant
// can leverage. It includes a central 'Registry' for discovering and
// invoking tools, and defines the 'Tool' interface that all specific tool
// implementations must adhere to.
//
// Sub-packages (e.g., 'godev', 'docker') contain concrete tool implementations.
// This package enables the assistant to extend its functionality by interacting
// with external systems or performing specialized tasks.
package tool
