package chains

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tmc/langchaingo/llms"

	"github.com/koopa0/assistant-go/internal/config"
)

// ChainType represents the type of chain
type ChainType string

const (
	ChainTypeSequential  ChainType = "sequential"
	ChainTypeParallel    ChainType = "parallel"
	ChainTypeConditional ChainType = "conditional"
	ChainTypeRAG         ChainType = "rag"
)

// ChainRequest represents a request to execute a chain
type ChainRequest struct {
	Input       string                 `json:"input"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	MaxSteps    int                    `json:"max_steps,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ChainResponse represents the response from chain execution
type ChainResponse struct {
	Output        string                 `json:"output"`
	Steps         []ChainStep            `json:"steps"`
	ExecutionTime time.Duration          `json:"execution_time"`
	TokensUsed    int                    `json:"tokens_used"`
	Success       bool                   `json:"success"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// ChainStep represents a single step in chain execution
type ChainStep struct {
	StepNumber int                    `json:"step_number"`
	StepType   string                 `json:"step_type"`
	Input      string                 `json:"input"`
	Output     string                 `json:"output"`
	Duration   time.Duration          `json:"duration"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// BaseChain provides common functionality for all chain types
type BaseChain struct {
	chainType ChainType
	llm       llms.Model
	config    config.LangChain
	logger    *slog.Logger
	steps     []ChainStepDefinition
}

// ChainStepDefinition defines a step in the chain
type ChainStepDefinition struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Prompt       string                 `json:"prompt"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	Condition    string                 `json:"condition,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
}

// NewBaseChain creates a new base chain
func NewBaseChain(chainType ChainType, llm llms.Model, config config.LangChain, logger *slog.Logger) *BaseChain {
	return &BaseChain{
		chainType: chainType,
		llm:       llm,
		config:    config,
		logger:    logger,
		steps:     make([]ChainStepDefinition, 0),
	}
}

// GetType returns the chain type
func (c *BaseChain) GetType() ChainType {
	return c.chainType
}

// AddStep adds a step to the chain
func (c *BaseChain) AddStep(step ChainStepDefinition) {
	c.steps = append(c.steps, step)
	c.logger.Debug("Added step to chain",
		slog.String("chain_type", string(c.chainType)),
		slog.String("step_name", step.Name),
		slog.String("step_type", step.Type))
}

// Execute executes the chain (to be overridden by specific chain types)
func (c *BaseChain) Execute(ctx context.Context, request *ChainRequest) (*ChainResponse, error) {
	startTime := time.Now()

	// Validate request first
	if err := c.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	c.logger.Info("Executing chain",
		slog.String("chain_type", string(c.chainType)),
		slog.String("input", request.Input),
		slog.Int("steps", len(c.steps)))

	// Initialize response
	response := &ChainResponse{
		Steps:    make([]ChainStep, 0),
		Success:  false,
		Metadata: make(map[string]interface{}),
	}

	// Execute steps (default implementation - to be overridden)
	output, steps, err := c.executeSteps(ctx, request)
	if err != nil {
		response.Error = err.Error()
		response.ExecutionTime = time.Since(startTime)
		return response, err
	}

	// Build successful response
	response.Output = output
	response.Steps = steps
	response.ExecutionTime = time.Since(startTime)
	response.Success = true
	response.Metadata["chain_type"] = string(c.chainType)
	response.Metadata["steps_executed"] = len(steps)

	c.logger.Info("Chain execution completed",
		slog.String("chain_type", string(c.chainType)),
		slog.Int("steps", len(steps)),
		slog.Duration("execution_time", response.ExecutionTime),
		slog.Bool("success", response.Success))

	return response, nil
}

// executeSteps executes the chain steps (to be overridden by specific chain types)
func (c *BaseChain) executeSteps(ctx context.Context, request *ChainRequest) (string, []ChainStep, error) {
	// Default implementation - just call LLM directly
	steps := make([]ChainStep, 0)

	stepStart := time.Now()
	result, err := c.llm.Call(ctx, request.Input)
	if err != nil {
		return "", nil, fmt.Errorf("LLM call failed: %w", err)
	}

	step := ChainStep{
		StepNumber: 1,
		StepType:   "llm_call",
		Input:      request.Input,
		Output:     result,
		Duration:   time.Since(stepStart),
		Success:    true,
		Metadata:   make(map[string]interface{}),
	}

	steps = append(steps, step)
	return result, steps, nil
}

// validateRequest validates a chain request
func (c *BaseChain) validateRequest(request *ChainRequest) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if request.Input == "" {
		return fmt.Errorf("input cannot be empty")
	}

	if request.MaxSteps < 0 {
		return fmt.Errorf("max_steps cannot be negative")
	}

	if request.Temperature < 0 || request.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	return nil
}

// Health checks the health of the chain
func (c *BaseChain) Health(ctx context.Context) error {
	// Check LLM health
	_, err := c.llm.Call(ctx, "Health check", llms.WithMaxTokens(1))
	if err != nil {
		return fmt.Errorf("LLM health check failed: %w", err)
	}

	return nil
}

// ChainManager manages different types of chains
type ChainManager struct {
	chains map[ChainType]Chain
	logger *slog.Logger
}

// Chain interface that all chain types must implement
type Chain interface {
	GetType() ChainType
	Execute(ctx context.Context, request *ChainRequest) (*ChainResponse, error)
	Health(ctx context.Context) error
}

// NewChainManager creates a new chain manager
func NewChainManager(logger *slog.Logger) *ChainManager {
	return &ChainManager{
		chains: make(map[ChainType]Chain),
		logger: logger,
	}
}

// RegisterChain registers a chain with the manager
func (cm *ChainManager) RegisterChain(chainType ChainType, chain Chain) {
	cm.chains[chainType] = chain
	cm.logger.Debug("Chain registered",
		slog.String("chain_type", string(chainType)))
}

// GetChain returns a chain by type
func (cm *ChainManager) GetChain(chainType ChainType) (Chain, error) {
	chain, exists := cm.chains[chainType]
	if !exists {
		return nil, fmt.Errorf("chain type %s not found", chainType)
	}
	return chain, nil
}

// ExecuteChain executes a chain by type
func (cm *ChainManager) ExecuteChain(ctx context.Context, chainType ChainType, request *ChainRequest) (*ChainResponse, error) {
	chain, err := cm.GetChain(chainType)
	if err != nil {
		return nil, err
	}

	return chain.Execute(ctx, request)
}

// ListChains returns all registered chain types
func (cm *ChainManager) ListChains() []ChainType {
	types := make([]ChainType, 0, len(cm.chains))
	for chainType := range cm.chains {
		types = append(types, chainType)
	}
	return types
}

// Health checks the health of all registered chains
func (cm *ChainManager) Health(ctx context.Context) error {
	for chainType, chain := range cm.chains {
		if err := chain.Health(ctx); err != nil {
			return fmt.Errorf("chain %s health check failed: %w", chainType, err)
		}
	}
	return nil
}

// ChainFactory creates different types of chains
type ChainFactory struct {
	llm    llms.Model
	config config.LangChain
	logger *slog.Logger
}

// NewChainFactory creates a new chain factory
func NewChainFactory(llm llms.Model, config config.LangChain, logger *slog.Logger) *ChainFactory {
	return &ChainFactory{
		llm:    llm,
		config: config,
		logger: logger,
	}
}

// CreateChain creates a chain of the specified type
func (cf *ChainFactory) CreateChain(chainType ChainType) (Chain, error) {
	switch chainType {
	case ChainTypeSequential:
		return NewSequentialChain(cf.llm, cf.config, cf.logger), nil
	case ChainTypeParallel:
		return NewParallelChain(cf.llm, cf.config, cf.logger), nil
	case ChainTypeConditional:
		return NewConditionalChain(cf.llm, cf.config, cf.logger), nil
	case ChainTypeRAG:
		return NewRAGChain(cf.llm, cf.config, cf.logger), nil
	default:
		return nil, fmt.Errorf("unsupported chain type: %s", chainType)
	}
}

// CreateAllChains creates all supported chain types
func (cf *ChainFactory) CreateAllChains() (map[ChainType]Chain, error) {
	chains := make(map[ChainType]Chain)

	chainTypes := []ChainType{
		ChainTypeSequential,
		ChainTypeParallel,
		ChainTypeConditional,
		ChainTypeRAG,
	}

	for _, chainType := range chainTypes {
		chain, err := cf.CreateChain(chainType)
		if err != nil {
			return nil, fmt.Errorf("failed to create chain %s: %w", chainType, err)
		}
		chains[chainType] = chain
	}

	return chains, nil
}
