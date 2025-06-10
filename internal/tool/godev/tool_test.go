package godev

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/koopa0/assistant-go/internal/tool"
)

func TestGoDevTool_Basic(t *testing.T) {
	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during tests
	}))

	// Create detector and tool
	detector := NewWorkspaceDetector(logger)
	gdTool := NewGoDevTool(detector, logger)

	// Test basic properties
	if gdTool.Name() != "godev" {
		t.Errorf("Expected name 'godev', got '%s'", gdTool.Name())
	}

	if gdTool.Description() == "" {
		t.Error("Description should not be empty")
	}

	// Test parameters schema
	params := gdTool.Parameters()
	if params.Type != "object" {
		t.Errorf("Expected type 'object', got '%s'", params.Type)
	}

	// Check required parameters
	found := false
	for _, req := range params.Required {
		if req == "action" {
			found = true
			break
		}
	}
	if !found {
		t.Error("'action' should be a required parameter")
	}
}

func TestGoDevTool_DetectCurrentProject(t *testing.T) {
	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during tests
	}))

	// Create detector and tool
	detector := NewWorkspaceDetector(logger)
	gdTool := NewGoDevTool(detector, logger)

	// Test detection on current project (should have go.mod)
	goInput := GoDevInput{
		Action: "detect",
		Path:   "../../../", // Go to project root
	}
	paramsJSON, _ := json.Marshal(goInput)
	input := &tool.ToolInput{
		Parameters: map[string]interface{}{},
	}
	_ = json.Unmarshal(paramsJSON, &input.Parameters)

	result, err := gdTool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Detection failed: %s", result.Error)
	}

	if result.Data == nil {
		t.Error("Result data should not be nil")
	}

	if result.Data != nil {
		t.Logf("Detection result: %s", result.Data.Output)
	}
}

func TestGoDevTool_InvalidAction(t *testing.T) {
	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during tests
	}))

	// Create detector and tool
	detector := NewWorkspaceDetector(logger)
	gdTool := NewGoDevTool(detector, logger)

	// Test with invalid action
	goInput := GoDevInput{
		Action: "invalid_action",
		Path:   ".",
	}
	paramsJSON, _ := json.Marshal(goInput)
	input := &tool.ToolInput{
		Parameters: map[string]interface{}{},
	}
	_ = json.Unmarshal(paramsJSON, &input.Parameters)

	result, err := gdTool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for invalid action")
	}

	if result.Error == "" {
		t.Error("Expected error message for invalid action")
	}
}

func TestGoDevTool_JSONInput(t *testing.T) {
	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during tests
	}))

	// Create detector and tool
	detector := NewWorkspaceDetector(logger)
	gdTool := NewGoDevTool(detector, logger)

	// Test JSON input
	jsonInput := `{"action": "detect", "path": "../../../"}`

	result, err := gdTool.ExecuteWithJSON(context.Background(), jsonInput)
	if err != nil {
		t.Fatalf("ExecuteWithJSON failed: %v", err)
	}

	if !result.Success {
		t.Errorf("JSON execution failed: %s", result.Error)
	}
}

func TestGoDevTool_InvalidJSON(t *testing.T) {
	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during tests
	}))

	// Create detector and tool
	detector := NewWorkspaceDetector(logger)
	gdTool := NewGoDevTool(detector, logger)

	// Test invalid JSON
	invalidJSON := `{"action": "detect", "path": }`

	result, err := gdTool.ExecuteWithJSON(context.Background(), invalidJSON)
	if err != nil {
		t.Fatalf("ExecuteWithJSON failed: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for invalid JSON")
	}
}

func TestGoDevTool_SampleUsage(t *testing.T) {
	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during tests
	}))

	// Create detector and tool
	detector := NewWorkspaceDetector(logger)
	gdTool := NewGoDevTool(detector, logger)

	// Test that sample usage examples are valid JSON
	samples := gdTool.GetSampleUsage()
	if len(samples) == 0 {
		t.Error("Should have sample usage examples")
	}

	for i, sample := range samples {
		var input GoDevInput
		if err := json.Unmarshal([]byte(sample), &input); err != nil {
			t.Errorf("Sample %d is invalid JSON: %v", i, err)
		}
	}
}

// Benchmark the detection performance
func BenchmarkGoDevTool_Detect(b *testing.B) {
	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Quiet during benchmarks
	}))

	// Create detector and tool
	detector := NewWorkspaceDetector(logger)
	gdTool := NewGoDevTool(detector, logger)

	// Benchmark input
	goInput := GoDevInput{
		Action: "detect",
		Path:   "../../../", // Go to project root
	}
	paramsJSON, _ := json.Marshal(goInput)
	input := &tool.ToolInput{
		Parameters: map[string]interface{}{},
	}
	_ = json.Unmarshal(paramsJSON, &input.Parameters)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gdTool.Execute(context.Background(), input)
		if err != nil {
			b.Fatalf("Execute failed: %v", err)
		}
	}
}
