package godev_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/koopa0/assistant-go/internal/tool"
	"github.com/koopa0/assistant-go/internal/tool/godev"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoDevTool_Integration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	detector := godev.NewWorkspaceDetector(logger)
	gdTool := godev.NewGoDevTool(detector, logger)

	ctx := context.Background()

	t.Run("AnalyzeWorkspace", func(t *testing.T) {
		params := map[string]interface{}{
			"command": "analyze_workspace",
			"path":    ".", // Current directory
		}

		input := &tool.ToolInput{
			Parameters: params,
		}

		result, err := gdTool.Execute(ctx, input)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Data)

		// Check result contains expected information
		if result.Data != nil && result.Data.Output != nil {
			outputJSON, err := json.Marshal(result.Data.Output)
			require.NoError(t, err)
			outputStr := string(outputJSON)
			assert.Contains(t, outputStr, "workspace")
		}
	})

	t.Run("AnalyzeComplexity", func(t *testing.T) {
		// Create test Go file
		testCode := `package main

import "fmt"

func main() {
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			fmt.Println("even")
		} else {
			fmt.Println("odd")
		}
	}
}
`
		tempFile := "/tmp/test_complexity.go"
		err := os.WriteFile(tempFile, []byte(testCode), 0644)
		require.NoError(t, err)
		defer os.Remove(tempFile)

		params := map[string]interface{}{
			"command": "analyze_complexity",
			"path":    tempFile,
		}

		input := &tool.ToolInput{
			Parameters: params,
		}

		result, err := gdTool.Execute(ctx, input)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Data)

		// Check complexity result
		assert.NotNil(t, result.Data)
		outputJSON, err := json.Marshal(result.Data.Output)
		require.NoError(t, err)
		resultStr := string(outputJSON)
		assert.Contains(t, resultStr, "Complexity:")
		assert.Contains(t, resultStr, "main:")
	})

	t.Run("AnalyzeDependencies", func(t *testing.T) {
		params := map[string]interface{}{
			"command": "analyze_dependencies",
			"path":    ".",
		}

		input := &tool.ToolInput{
			Parameters: params,
		}

		result, err := gdTool.Execute(ctx, input)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Data)

		// Check dependencies result
		assert.NotNil(t, result.Data)
		outputJSON, err := json.Marshal(result.Data.Output)
		require.NoError(t, err)
		resultStr := string(outputJSON)
		assert.Contains(t, resultStr, "Direct Dependencies:")
	})
}

func TestGoDevTool_ErrorHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	detector := godev.NewWorkspaceDetector(logger)
	gdTool := godev.NewGoDevTool(detector, logger)

	ctx := context.Background()

	t.Run("InvalidCommand", func(t *testing.T) {
		params := map[string]interface{}{
			"command": "invalid_command",
		}

		input := &tool.ToolInput{
			Parameters: params,
		}

		result, err := gdTool.Execute(ctx, input)
		require.NoError(t, err) // Should not return error, but result.Success should be false
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "unsupported command")
	})

	t.Run("MissingPath", func(t *testing.T) {
		params := map[string]interface{}{
			"command": "analyze_workspace",
			// Missing path
		}

		input := &tool.ToolInput{
			Parameters: params,
		}

		result, err := gdTool.Execute(ctx, input)
		require.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "path")
	})
}
