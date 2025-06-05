package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/types"
)

// This file demonstrates proper dependency injection patterns for Go testing
// Following the principle: "accept interfaces, return structs"

// Simple interfaces for dependency injection
type ToolExecutor interface {
	Execute(ctx context.Context, toolName string, input types.ToolInput) (*types.ToolOutput, error)
}

type StatsRecorder interface {
	Record(toolName string, duration time.Duration, success bool)
}

// Simple service that depends on interfaces
type ExampleToolService struct {
	executor ToolExecutor
	recorder StatsRecorder
}

// NewExampleToolService accepts interfaces, returns struct
func NewExampleToolService(executor ToolExecutor, recorder StatsRecorder) *ExampleToolService {
	return &ExampleToolService{
		executor: executor,
		recorder: recorder,
	}
}

func (s *ExampleToolService) ProcessTool(ctx context.Context, toolName string, input types.ToolInput) (*types.ToolOutput, error) {
	start := time.Now()

	result, err := s.executor.Execute(ctx, toolName, input)

	duration := time.Since(start)
	success := err == nil && result != nil && result.Success
	s.recorder.Record(toolName, duration, success)

	return result, err
}

// Test doubles (fakes, not mocks) - following Go best practices

type FakeExecutor struct {
	result *types.ToolOutput
	err    error
}

func (f *FakeExecutor) Execute(ctx context.Context, toolName string, input types.ToolInput) (*types.ToolOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &types.ToolOutput{Success: true, Duration: 100 * time.Millisecond}, nil
}

type FakeRecorder struct {
	records []Record
}

type Record struct {
	ToolName string
	Duration time.Duration
	Success  bool
}

func (f *FakeRecorder) Record(toolName string, duration time.Duration, success bool) {
	f.records = append(f.records, Record{
		ToolName: toolName,
		Duration: duration,
		Success:  success,
	})
}

// Tests demonstrating proper dependency injection

func TestDependencyInjection_Principles(t *testing.T) {
	t.Run("interface_segregation", func(t *testing.T) {
		// Each interface has a single responsibility
		// ToolExecutor only executes tools
		// StatsRecorder only records statistics

		// This compiles only if our fakes implement the interfaces
		var _ ToolExecutor = (*FakeExecutor)(nil)
		var _ StatsRecorder = (*FakeRecorder)(nil)
	})

	t.Run("behavior_testing", func(t *testing.T) {
		// Arrange - set up specific behavior
		executor := &FakeExecutor{
			result: &types.ToolOutput{
				Success:  true,
				Message:  "test completed",
				Duration: 150 * time.Millisecond,
			},
		}
		recorder := &FakeRecorder{}

		service := NewExampleToolService(executor, recorder)

		// Act
		input := types.ToolInput{Action: "test"}
		result, err := service.ProcessTool(context.Background(), "test_tool", input)

		// Assert behavior, not implementation
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !result.Success {
			t.Error("Expected successful result")
		}

		// Verify interaction with recorder
		if len(recorder.records) != 1 {
			t.Fatalf("Expected 1 record, got %d", len(recorder.records))
		}

		record := recorder.records[0]
		if record.ToolName != "test_tool" {
			t.Errorf("Expected tool name 'test_tool', got %s", record.ToolName)
		}
		if !record.Success {
			t.Error("Expected successful record")
		}
		if record.Duration <= 0 {
			t.Error("Expected positive duration")
		}
	})

	t.Run("error_handling", func(t *testing.T) {
		// Arrange error scenario
		executor := &FakeExecutor{
			err: errors.New("execution failed"),
		}
		recorder := &FakeRecorder{}

		service := NewExampleToolService(executor, recorder)

		// Act
		input := types.ToolInput{Action: "test"}
		result, err := service.ProcessTool(context.Background(), "failing_tool", input)

		// Assert
		if err == nil {
			t.Error("Expected error but got none")
		}
		if result != nil {
			t.Error("Expected nil result on error")
		}

		// Verify error was recorded
		if len(recorder.records) != 1 {
			t.Fatalf("Expected 1 record, got %d", len(recorder.records))
		}

		record := recorder.records[0]
		if record.Success {
			t.Error("Expected failed record")
		}
	})

	t.Run("isolation_and_determinism", func(t *testing.T) {
		// Each test is isolated and deterministic
		executor := &FakeExecutor{}
		recorder := &FakeRecorder{}

		service := NewExampleToolService(executor, recorder)

		// Multiple calls should be consistent
		for i := 0; i < 3; i++ {
			input := types.ToolInput{Action: "consistent"}
			result, err := service.ProcessTool(context.Background(), "test_tool", input)

			if err != nil {
				t.Errorf("Call %d failed: %v", i, err)
			}
			if !result.Success {
				t.Errorf("Call %d was not successful", i)
			}
		}

		// Verify all calls were recorded
		if len(recorder.records) != 3 {
			t.Errorf("Expected 3 records, got %d", len(recorder.records))
		}
	})
}

// Benchmarks with dependency injection
func BenchmarkDependencyInjection(b *testing.B) {
	executor := &FakeExecutor{}
	recorder := &FakeRecorder{}
	service := NewExampleToolService(executor, recorder)

	input := types.ToolInput{Action: "benchmark"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.ProcessTool(ctx, "bench_tool", input)
	}
}

// Example of testing with different configurations
func TestDependencyInjection_Configurations(t *testing.T) {
	tests := []struct {
		name     string
		executor ToolExecutor
		recorder StatsRecorder
		wantErr  bool
	}{
		{
			name:     "successful_execution",
			executor: &FakeExecutor{},
			recorder: &FakeRecorder{},
			wantErr:  false,
		},
		{
			name:     "execution_failure",
			executor: &FakeExecutor{err: errors.New("failed")},
			recorder: &FakeRecorder{},
			wantErr:  true,
		},
		{
			name: "custom_result",
			executor: &FakeExecutor{
				result: &types.ToolOutput{
					Success: true,
					Message: "custom output",
				},
			},
			recorder: &FakeRecorder{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewExampleToolService(tt.executor, tt.recorder)

			input := types.ToolInput{Action: "test"}
			result, err := service.ProcessTool(context.Background(), "test_tool", input)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.wantErr && result == nil {
				t.Error("Expected result but got nil")
			}

			// Verify recording happened regardless of success/failure
			fakeRecorder := tt.recorder.(*FakeRecorder)
			if len(fakeRecorder.records) != 1 {
				t.Errorf("Expected 1 record, got %d", len(fakeRecorder.records))
			}
		})
	}
}

/*
Key principles demonstrated:

1. **Accept interfaces, return structs**: NewExampleToolService accepts interfaces
2. **Small, focused interfaces**: ToolExecutor and StatsRecorder have single responsibilities
3. **Test doubles over mocks**: FakeExecutor and FakeRecorder are simple fakes
4. **Behavior testing**: Tests verify what the system does, not how it does it
5. **Isolation**: Each test is independent and deterministic
6. **Configuration through composition**: Different test scenarios use different implementations

Benefits:
- Tests are fast (no real dependencies)
- Tests are reliable (deterministic)
- Tests are maintainable (clear intent)
- Code is more modular (loose coupling)
- Easy to add new implementations
*/
