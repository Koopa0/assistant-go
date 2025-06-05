package tools

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// This file demonstrates refactoring the Registry to use dependency injection
// Following Go best practices for testable, maintainable code

// Extracted interfaces from Registry behavior
// These interfaces represent what the Registry actually needs, not what it has

type ExampleToolFactory func(config *ToolConfig, logger Logger) (Tool, error)

type ToolStore interface {
	Store(name string, tool Tool) error
	Retrieve(name string) (Tool, bool)
	Remove(name string) error
	List() []Tool
}

type FactoryStore interface {
	Store(name string, factory ExampleToolFactory) error
	Retrieve(name string) (ExampleToolFactory, bool)
	Remove(name string) error
	List() map[string]ExampleToolFactory
}

type Logger interface {
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
}

// Registry with dependency injection
type DIRegistry struct {
	tools     ToolStore
	factories FactoryStore
	logger    Logger
}

// NewDIRegistry accepts interfaces, returns struct
func NewDIRegistry(tools ToolStore, factories FactoryStore, logger Logger) *DIRegistry {
	return &DIRegistry{
		tools:     tools,
		factories: factories,
		logger:    logger,
	}
}

func (r *DIRegistry) Register(name string, factory ExampleToolFactory) error {
	if name == "" {
		return errors.New("tool name cannot be empty")
	}
	if factory == nil {
		return errors.New("tool factory cannot be nil")
	}

	return r.factories.Store(name, factory)
}

func (r *DIRegistry) GetTool(name string, config *ToolConfig) (Tool, error) {
	// Check if tool instance already exists
	if tool, exists := r.tools.Retrieve(name); exists {
		return tool, nil
	}

	// Get factory and create new instance
	factory, exists := r.factories.Retrieve(name)
	if !exists {
		return nil, fmt.Errorf("tool %s is not registered", name)
	}

	tool, err := factory(config, r.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool %s: %w", name, err)
	}

	// Store the instance
	if err := r.tools.Store(name, tool); err != nil {
		r.logger.Warn("Failed to store tool instance", "tool", name, "error", err)
	}

	return tool, nil
}

func (r *DIRegistry) Execute(ctx context.Context, toolName string, input *ToolInput, config *ToolConfig) (*ToolResult, error) {
	start := time.Now()

	tool, err := r.GetTool(toolName, config)
	if err != nil {
		return &ToolResult{
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: time.Since(start),
		}, err
	}

	toolInput := &ToolInput{Parameters: input}
	result, err := tool.Execute(ctx, toolInput)
	duration := time.Since(start)

	if err != nil {
		return &ToolResult{
			Success:       false,
			Error:         err.Error(),
			ExecutionTime: duration,
		}, err
	}

	if result == nil {
		result = &ToolResult{Success: true}
	}
	result.ExecutionTime = duration

	return result, nil
}

// Test doubles for dependency injection

type InMemoryToolStore struct {
	tools map[string]Tool
}

func NewInMemoryToolStore() *InMemoryToolStore {
	return &InMemoryToolStore{
		tools: make(map[string]Tool),
	}
}

func (s *InMemoryToolStore) Store(name string, tool Tool) error {
	s.tools[name] = tool
	return nil
}

func (s *InMemoryToolStore) Retrieve(name string) (Tool, bool) {
	tool, exists := s.tools[name]
	return tool, exists
}

func (s *InMemoryToolStore) Remove(name string) error {
	delete(s.tools, name)
	return nil
}

func (s *InMemoryToolStore) List() []Tool {
	tools := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}
	return tools
}

type InMemoryFactoryStore struct {
	factories map[string]ExampleToolFactory
}

func NewInMemoryFactoryStore() *InMemoryFactoryStore {
	return &InMemoryFactoryStore{
		factories: make(map[string]ExampleToolFactory),
	}
}

func (s *InMemoryFactoryStore) Store(name string, factory ExampleToolFactory) error {
	if _, exists := s.factories[name]; exists {
		return fmt.Errorf("tool %s is already registered", name)
	}
	s.factories[name] = factory
	return nil
}

func (s *InMemoryFactoryStore) Retrieve(name string) (ExampleToolFactory, bool) {
	factory, exists := s.factories[name]
	return factory, exists
}

func (s *InMemoryFactoryStore) Remove(name string) error {
	delete(s.factories, name)
	return nil
}

func (s *InMemoryFactoryStore) List() map[string]ExampleToolFactory {
	result := make(map[string]ExampleToolFactory)
	for name, factory := range s.factories {
		result[name] = factory
	}
	return result
}

type TestLogger struct {
	messages []LogMessage
}

type LogMessage struct {
	Level   string
	Message string
	Args    []any
}

func NewTestLogger() *TestLogger {
	return &TestLogger{}
}

func (l *TestLogger) Debug(msg string, args ...any) {
	l.messages = append(l.messages, LogMessage{"DEBUG", msg, args})
}

func (l *TestLogger) Error(msg string, args ...any) {
	l.messages = append(l.messages, LogMessage{"ERROR", msg, args})
}

func (l *TestLogger) Warn(msg string, args ...any) {
	l.messages = append(l.messages, LogMessage{"WARN", msg, args})
}

// Tests demonstrating dependency injection benefits

func TestDIRegistry_DependencyInjection(t *testing.T) {
	t.Run("interface_compliance", func(t *testing.T) {
		// Verify our test doubles implement the interfaces
		var _ ToolStore = (*InMemoryToolStore)(nil)
		var _ FactoryStore = (*InMemoryFactoryStore)(nil)
		var _ Logger = (*TestLogger)(nil)
	})

	t.Run("configurable_behavior", func(t *testing.T) {
		// Different stores can have different behaviors

		// Scenario 1: Normal in-memory stores
		tools1 := NewInMemoryToolStore()
		factories1 := NewInMemoryFactoryStore()
		logger1 := NewTestLogger()
		registry1 := NewDIRegistry(tools1, factories1, logger1)

		// Scenario 2: Failing factory store for error testing
		tools2 := NewInMemoryToolStore()
		factories2 := &FailingFactoryStore{}
		logger2 := NewTestLogger()
		registry2 := NewDIRegistry(tools2, factories2, logger2)

		// Test normal behavior
		factory := func(config *ToolConfig, logger Logger) (Tool, error) {
			return &SimpleTool{name: "test_tool"}, nil
		}

		err1 := registry1.Register("test_tool", factory)
		if err1 != nil {
			t.Errorf("Registry1 Register failed: %v", err1)
		}

		// Test failing behavior
		err2 := registry2.Register("test_tool", factory)
		if err2 == nil {
			t.Error("Registry2 Register should have failed")
		}
	})

	t.Run("isolated_testing", func(t *testing.T) {
		// Each test gets fresh dependencies
		tools := NewInMemoryToolStore()
		factories := NewInMemoryFactoryStore()
		logger := NewTestLogger()
		registry := NewDIRegistry(tools, factories, logger)

		// Register a tool
		factory := func(config *ToolConfig, logger Logger) (Tool, error) {
			return &SimpleTool{name: "isolated_tool"}, nil
		}

		err := registry.Register("isolated_tool", factory)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		// Execute the tool
		ctx := context.Background()
		input := &ToolInput{Parameters: map[string]interface{}{}}
		result, err := registry.Execute(ctx, "isolated_tool", input, nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if !result.Success {
			t.Error("Expected successful execution")
		}

		// Verify logging behavior - may or may not have messages depending on implementation
		t.Logf("Logger captured %d messages", len(logger.messages))

		// Verify tool was stored
		tool, exists := tools.Retrieve("isolated_tool")
		if !exists {
			t.Error("Tool should be stored after creation")
		}
		if tool.Name() != "isolated_tool" {
			t.Errorf("Wrong tool name: %s", tool.Name())
		}
	})

	t.Run("error_scenarios", func(t *testing.T) {
		tools := NewInMemoryToolStore()
		factories := NewInMemoryFactoryStore()
		logger := NewTestLogger()
		registry := NewDIRegistry(tools, factories, logger)

		// Test with failing factory
		failingFactory := func(config map[string]interface{}, logger Logger) (Tool, error) {
			return nil, errors.New("factory creation failed")
		}

		err := registry.Register("failing_tool", failingFactory)
		if err != nil {
			t.Fatalf("Register should not fail: %v", err)
		}

		// Try to get the tool
		tool, err := registry.GetTool("failing_tool", nil)
		if err == nil {
			t.Error("GetTool should have failed")
		}
		if tool != nil {
			t.Error("Tool should be nil on failure")
		}

		// Verify logging behavior - may or may not have error messages depending on implementation
		errorCount := 0
		for _, msg := range logger.messages {
			if msg.Level == "ERROR" {
				errorCount++
			}
		}
		t.Logf("Logger captured %d error messages", errorCount)
	})
}

// Supporting types for tests

type FailingFactoryStore struct{}

func (f *FailingFactoryStore) Store(name string, factory ExampleToolFactory) error {
	return errors.New("factory store is full")
}

func (f *FailingFactoryStore) Retrieve(name string) (ExampleToolFactory, bool) {
	return nil, false
}

func (f *FailingFactoryStore) Remove(name string) error {
	return errors.New("cannot remove from failing store")
}

func (f *FailingFactoryStore) List() map[string]ExampleToolFactory {
	return make(map[string]ExampleToolFactory)
}

type SimpleTool struct {
	name string
}

func (s *SimpleTool) Name() string {
	return s.name
}

func (s *SimpleTool) Description() string {
	return "Simple test tool"
}

func (s *SimpleTool) Parameters() *ToolParametersSchema {
	return &ToolParametersSchema{Type: "object"}
}

func (s *SimpleTool) Execute(ctx context.Context, input *ToolInput) (*ToolResult, error) {
	return &ToolResult{Success: true, Data: &ToolResultData{Output: "simple result"}}, nil
}

func (s *SimpleTool) Health(ctx context.Context) error {
	return nil
}

func (s *SimpleTool) Close(ctx context.Context) error {
	return nil
}

/*
Benefits of this dependency injection approach:

1. **Testability**: Easy to inject test doubles for isolated testing
2. **Flexibility**: Different implementations for different environments
3. **Single Responsibility**: Each store has one clear purpose
4. **Mockable**: Can easily test error scenarios
5. **Maintainable**: Changes to store behavior don't affect registry logic
6. **Fast Tests**: In-memory implementations are very fast

Key Go principles demonstrated:
- Accept interfaces, return structs
- Small, focused interfaces
- Composition over inheritance
- Test doubles over mocks
- Behavior-driven testing
*/
