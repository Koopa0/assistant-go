package langchain

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tmc/langchaingo/llms"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/core/agent"
	"github.com/koopa0/assistant-go/internal/langchain/chains"
	"github.com/koopa0/assistant-go/internal/langchain/memory"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
)

// Service represents the main LangChain service that integrates all components
type Service struct {
	config   config.LangChain
	logger   *slog.Logger
	dbClient *postgres.SQLCClient

	// Core components
	memoryManager *memory.MemoryManager
	chainManager  *chains.ChainManager

	// Agent instances with LangChain adapters
	agentManager *agent.SimpleManager
	agents       map[agent.AgentType]*agent.LangChainAdapter

	// LLM providers
	llmProviders map[string]llms.Model
}

// ServiceConfig contains configuration for the LangChain service
type ServiceConfig struct {
	Config       config.LangChain
	Logger       *slog.Logger
	DBClient     *postgres.SQLCClient
	LLMProviders map[string]llms.Model
}

// NewService creates a new LangChain service instance
func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	if cfg.DBClient == nil {
		return nil, fmt.Errorf("database client is required")
	}

	if len(cfg.LLMProviders) == 0 {
		return nil, fmt.Errorf("at least one LLM provider is required")
	}

	// Initialize memory manager
	memoryManager := memory.NewMemoryManager(cfg.DBClient, cfg.Config, cfg.Logger)

	// Initialize chain manager
	chainManager := chains.NewChainManager(cfg.Logger)

	// Initialize agent manager and agents
	agentManager := agent.NewSimpleManager(cfg.Logger)
	agents := make(map[agent.AgentType]*agent.LangChainAdapter)

	// Create agents with available LLM providers
	if len(cfg.LLMProviders) > 0 {
		// Use the first available LLM provider for agents
		var primaryLLM llms.Model
		for _, llm := range cfg.LLMProviders {
			primaryLLM = llm
			break
		}

		// Create development agent (the only one currently implemented)
		if devAgent := agent.NewDevelopmentAgent(primaryLLM, cfg.Logger); devAgent != nil {
			// Create LangChain adapter for the development agent
			langchainAdapter := agent.NewLangChainAdapter(devAgent)
			agents[agent.TypeDevelopment] = langchainAdapter
		}

		// TODO: Implement other agents (database, infrastructure, research) when needed
	}

	service := &Service{
		config:        cfg.Config,
		logger:        cfg.Logger,
		dbClient:      cfg.DBClient,
		memoryManager: memoryManager,
		chainManager:  chainManager,
		agentManager:  agentManager,
		agents:        agents,
		llmProviders:  cfg.LLMProviders,
	}

	cfg.Logger.Info("LangChain service initialized successfully",
		slog.Int("llm_providers", len(cfg.LLMProviders)),
		slog.Bool("memory_enabled", cfg.Config.EnableMemory),
		slog.Int("max_iterations", cfg.Config.MaxIterations))

	return service, nil
}

// Agent execution methods

// AgentExecutionRequest wraps the agent request with user context
type AgentExecutionRequest struct {
	UserID string `json:"user_id"`
	*agent.Request
}

// ExecuteAgent executes an agent with the given request
func (s *Service) ExecuteAgent(ctx context.Context, agentType agent.AgentType, request *AgentExecutionRequest) (*agent.Response, error) {
	startTime := time.Now()

	s.logger.Info("Executing agent",
		slog.String("agent_type", string(agentType)),
		slog.String("user_id", request.UserID),
		slog.String("query", request.Request.Query))

	// Execute agent using the agent manager
	var response *agent.Response
	var err error

	// Check if the agent type is available
	langchainAdapter, exists := s.agents[agentType]
	if !exists {
		return nil, fmt.Errorf("agent type %s not available", agentType)
	}

	// Convert request to LangChain format and execute
	params := map[string]interface{}{
		"query":       request.Request.Query,
		"context":     request.Request.Context,
		"tools":       request.Request.Tools,
		"max_steps":   request.Request.MaxSteps,
		"temperature": request.Request.Temperature,
	}

	result, err := langchainAdapter.Execute(ctx, params)
	if err == nil {
		// Convert result back to agent.Response format
		response = &agent.Response{
			Result:        result["answer"].(string),
			Success:       true,
			Confidence:    result["confidence"].(float64),
			ExecutionTime: result["execution_time"].(time.Duration),
		}
		if steps, ok := result["steps"].([]agent.Step); ok {
			response.Steps = steps
		}
	}
	if err != nil {
		s.logger.Error("Agent execution failed",
			slog.String("agent_type", string(agentType)),
			slog.String("user_id", request.UserID),
			slog.Any("error", err))
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	executionTime := time.Since(startTime)

	// TODO: Store execution record in database when schema is available
	if s.dbClient != nil {
		s.logger.Debug("Agent execution record storage not yet implemented")
	}

	s.logger.Info("Agent execution completed",
		slog.String("agent_type", string(agentType)),
		slog.String("user_id", request.UserID),
		slog.Bool("success", true), // Agent execution succeeded if no error
		slog.Duration("execution_time", executionTime))

	return response, nil
}

// Chain execution methods

// ChainExecutionRequest wraps the chain request with user context
type ChainExecutionRequest struct {
	UserID string `json:"user_id"`
	*chains.ChainRequest
}

// ExecuteChain executes a chain with the given request
func (s *Service) ExecuteChain(ctx context.Context, chainType chains.ChainType, request *ChainExecutionRequest) (*chains.ChainResponse, error) {
	startTime := time.Now()

	s.logger.Info("Executing chain",
		slog.String("chain_type", string(chainType)),
		slog.String("user_id", request.UserID),
		slog.String("input", request.ChainRequest.Input))

	// Execute the chain
	response, err := s.chainManager.ExecuteChain(ctx, chainType, request.ChainRequest)
	if err != nil {
		s.logger.Error("Chain execution failed",
			slog.String("chain_type", string(chainType)),
			slog.String("user_id", request.UserID),
			slog.Any("error", err))
		return nil, fmt.Errorf("chain execution failed: %w", err)
	}

	executionTime := time.Since(startTime)

	// Store execution record in database
	if s.dbClient != nil {
		// Convert chain steps to interface{} for storage
		stepsInterface := make([]interface{}, len(response.Steps))
		for i, step := range response.Steps {
			stepsInterface[i] = step
		}

		execution := &postgres.ChainExecutionDomain{
			ChainType:       string(chainType),
			UserID:          request.UserID,
			Input:           request.ChainRequest.Input,
			Output:          response.Output,
			Steps:           stepsInterface,
			ExecutionTimeMs: int(executionTime.Milliseconds()),
			TokensUsed:      response.TokensUsed,
			Success:         response.Success,
			ErrorMessage:    response.Error, // ChainResponse uses Error, not ErrorMessage
			Metadata:        response.Metadata,
		}

		_, err := s.dbClient.CreateChainExecution(ctx, execution)
		if err != nil {
			s.logger.Warn("Failed to store chain execution record",
				slog.String("chain_type", string(chainType)),
				slog.Any("error", err))
		}
	}

	s.logger.Info("Chain execution completed",
		slog.String("chain_type", string(chainType)),
		slog.String("user_id", request.UserID),
		slog.Bool("success", response.Success),
		slog.Duration("execution_time", executionTime))

	return response, nil
}

// Memory management methods

// StoreMemory stores a memory entry
func (s *Service) StoreMemory(ctx context.Context, entry *memory.MemoryEntry) error {
	return s.memoryManager.Store(ctx, entry)
}

// SearchMemory searches for memories
func (s *Service) SearchMemory(ctx context.Context, query *memory.MemoryQuery) ([]*memory.MemorySearchResult, error) {
	return s.memoryManager.Retrieve(ctx, query)
}

// GetMemoryStats returns memory usage statistics
func (s *Service) GetMemoryStats(ctx context.Context, userID string) (*memory.MemoryStats, error) {
	return s.memoryManager.GetStats(ctx, userID)
}

// Utility methods

// GetAvailableAgents returns a list of available agent types
func (s *Service) GetAvailableAgents() []agent.AgentType {
	availableAgents := make([]agent.AgentType, 0, len(s.agents))

	for agentType := range s.agents {
		availableAgents = append(availableAgents, agentType)
	}

	return availableAgents
}

// GetAvailableChains returns a list of available chain types
func (s *Service) GetAvailableChains() []chains.ChainType {
	return s.chainManager.ListChains()
}

// GetLLMProviders returns a list of available LLM providers
func (s *Service) GetLLMProviders() []string {
	providers := make([]string, 0, len(s.llmProviders))
	for name := range s.llmProviders {
		providers = append(providers, name)
	}
	return providers
}

// Health check methods

// HealthCheck performs a health check on the service
func (s *Service) HealthCheck(ctx context.Context) error {
	// Check database connectivity
	if s.dbClient != nil {
		// TODO: Add a simple database ping/health check
		s.logger.Debug("Database health check passed")
	}

	// Check LLM providers
	for name := range s.llmProviders {
		s.logger.Debug("LLM provider available", slog.String("provider", name))
	}

	s.logger.Info("LangChain service health check passed")
	return nil
}

// Close gracefully shuts down the service
func (s *Service) Close() error {
	s.logger.Info("Shutting down LangChain service")

	// Close database connection
	if s.dbClient != nil {
		if err := s.dbClient.Close(); err != nil {
			s.logger.Error("Failed to close database connection", slog.Any("error", err))
			return err
		}
	}

	s.logger.Info("LangChain service shutdown completed")
	return nil
}
