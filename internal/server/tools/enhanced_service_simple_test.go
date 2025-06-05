package tools

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/koopa0/assistant-go/internal/tools"
)

// Test pure functions that don't require complex dependencies

func TestCalculatePopularityScore(t *testing.T) {
	tests := []struct {
		name            string
		totalExecutions int32
		successRate     float64
		wantScore       float64
	}{
		{
			name:            "zero executions",
			totalExecutions: 0,
			successRate:     0,
			wantScore:       0,
		},
		{
			name:            "low usage high success",
			totalExecutions: 10,
			successRate:     100,
			wantScore:       0.55, // (log10(11)/4 * 0.6) + (1.0 * 0.4)
		},
		{
			name:            "high usage low success",
			totalExecutions: 1000,
			successRate:     50,
			wantScore:       0.65, // (log10(1001)/4 * 0.6) + (0.5 * 0.4)
		},
		{
			name:            "extreme usage",
			totalExecutions: 10000,
			successRate:     90,
			wantScore:       0.96, // (log10(10001)/4 * 0.6) + (0.9 * 0.4)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePopularityScore(tt.totalExecutions, tt.successRate)
			// Allow small floating point differences
			if math.Abs(got-tt.wantScore) > 0.01 {
				t.Errorf("calculatePopularityScore() = %v, want %v", got, tt.wantScore)
			}
		})
	}
}

func TestConvertToFloat64(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  float64
	}{
		{"nil", nil, 0},
		{"float64", float64(42.5), 42.5},
		{"float32", float32(42.5), 42.5},
		{"int", int(42), 42},
		{"int32", int32(42), 42},
		{"int64", int64(42), 42},
		{"string", "not a number", 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToFloat64(tt.input)
			if got != tt.want {
				t.Errorf("convertToFloat64(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertToolResultToMap(t *testing.T) {
	service := &EnhancedToolService{}

	t.Run("nil result", func(t *testing.T) {
		result := service.convertToolResultToMap(nil)
		if result["success"].(bool) != false {
			t.Error("expected success to be false for nil result")
		}
		if result["error"].(string) != "no result" {
			t.Error("expected error message for nil result")
		}
	})

	t.Run("successful result", func(t *testing.T) {
		toolResult := &tools.ToolResult{
			Success: true,
			Data: map[string]interface{}{
				"output": "test data",
			},
			ExecutionTime: 100 * time.Millisecond,
			Metadata: map[string]interface{}{
				"version": "1.0",
			},
		}

		result := service.convertToolResultToMap(toolResult)
		if result["success"].(bool) != true {
			t.Error("expected success to be true")
		}
		if result["data"].(map[string]interface{})["output"] != "test data" {
			t.Error("expected data to be preserved")
		}
		if result["execution_time_ms"].(int64) != 100 {
			t.Error("expected execution time to be 100ms")
		}
	})

	t.Run("failed result", func(t *testing.T) {
		toolResult := &tools.ToolResult{
			Success: false,
			Error:   "test error",
		}

		result := service.convertToolResultToMap(toolResult)
		if result["success"].(bool) != false {
			t.Error("expected success to be false")
		}
		if result["error"].(string) != "test error" {
			t.Error("expected error message to be preserved")
		}
	})
}

func TestToolInfo_GettersAndSetters(t *testing.T) {
	info := ToolInfo{
		ID:          uuid.New().String(),
		Name:        "test_tool",
		DisplayName: "Test Tool",
		Description: "A test tool",
		Category:    "testing",
		Version:     "1.0.0",
		Status:      "available",
	}

	if info.Name != "test_tool" {
		t.Errorf("expected name to be 'test_tool', got %s", info.Name)
	}
	if info.Category != "testing" {
		t.Errorf("expected category to be 'testing', got %s", info.Category)
	}
}

// TODO: Add integration tests with proper dependency injection
// Following Go best practices:
// - Use interfaces for dependencies in the service
// - Create test doubles (not mocks) for testing
// - Test behavior, not implementation
