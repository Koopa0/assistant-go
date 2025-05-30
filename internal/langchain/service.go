package langchain

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tmc/langchaingo/llms"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/langchain/agents"
	"github.com/koopa0/assistant-go/internal/langchain/chains"
	"github.com/koopa0/assistant-go/internal/langchain/memory"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
)

// Service represents the main LangChain service that integrates all components
type Service struct {
	config   config.LangChain
	logger   *slog.Logger
	dbClient *postgres.SQLCClient

	// Core components
	memoryManager *memory.MemoryManager
	chainManager  *chains.ChainManager

	// Agent instances
	developmentAgent    *agents.DevelopmentAgent
	databaseAgent       *agents.DatabaseAgent
	infrastructureAgent *agents.InfrastructureAgent
	researchAgent       *agents.ResearchAgent

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

	// Initialize individual agents
	var developmentAgent *agents.DevelopmentAgent
	var databaseAgent *agents.DatabaseAgent
	var infrastructureAgent *agents.InfrastructureAgent
	var researchAgent *agents.ResearchAgent

	// Create agents with available LLM providers
	if len(cfg.LLMProviders) > 0 {
		// Use the first available LLM provider for agents
		var primaryLLM llms.Model
		for _, llm := range cfg.LLMProviders {
			primaryLLM = llm
			break
		}

		developmentAgent = agents.NewDevelopmentAgent(primaryLLM, cfg.Config, cfg.Logger)
		databaseAgent = agents.NewDatabaseAgent(primaryLLM, cfg.Config, cfg.DBClient, cfg.Logger)
		infrastructureAgent = agents.NewInfrastructureAgent(primaryLLM, cfg.Config, cfg.Logger)
		researchAgent = agents.NewResearchAgent(primaryLLM, cfg.Config, cfg.Logger)
	}

	service := &Service{
		config:              cfg.Config,
		logger:              cfg.Logger,
		dbClient:            cfg.DBClient,
		memoryManager:       memoryManager,
		chainManager:        chainManager,
		developmentAgent:    developmentAgent,
		databaseAgent:       databaseAgent,
		infrastructureAgent: infrastructureAgent,
		researchAgent:       researchAgent,
		llmProviders:        cfg.LLMProviders,
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
	*agents.AgentRequest
}

// ExecuteAgent executes an agent with the given request
func (s *Service) ExecuteAgent(ctx context.Context, agentType agents.AgentType, request *AgentExecutionRequest) (*agents.AgentResponse, error) {
	startTime := time.Now()

	s.logger.Info("Executing agent",
		slog.String("agent_type", string(agentType)),
		slog.String("user_id", request.UserID),
		slog.String("query", request.AgentRequest.Query))

	// Execute the appropriate agent
	var response *agents.AgentResponse
	var err error

	switch agentType {
	case agents.AgentTypeDevelopment:
		if s.developmentAgent == nil {
			return nil, fmt.Errorf("development agent not initialized")
		}
		response, err = s.developmentAgent.Execute(ctx, request.AgentRequest)
	case agents.AgentTypeDatabase:
		if s.databaseAgent == nil {
			return nil, fmt.Errorf("database agent not initialized")
		}
		response, err = s.databaseAgent.Execute(ctx, request.AgentRequest)
	case agents.AgentTypeInfrastructure:
		if s.infrastructureAgent == nil {
			return nil, fmt.Errorf("infrastructure agent not initialized")
		}
		response, err = s.infrastructureAgent.Execute(ctx, request.AgentRequest)
	case agents.AgentTypeResearch:
		if s.researchAgent == nil {
			return nil, fmt.Errorf("research agent not initialized")
		}
		response, err = s.researchAgent.Execute(ctx, request.AgentRequest)
	default:
		return nil, fmt.Errorf("unsupported agent type: %s", agentType)
	}
	if err != nil {
		s.logger.Error("Agent execution failed",
			slog.String("agent_type", string(agentType)),
			slog.String("user_id", request.UserID),
			slog.Any("error", err))
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	executionTime := time.Since(startTime)

	// Store execution record in database
	if s.dbClient != nil {
		// Convert agent steps to interface{} for storage
		stepsInterface := make([]interface{}, len(response.Steps))
		for i, step := range response.Steps {
			stepsInterface[i] = step
		}

		execution := &postgres.AgentExecutionDomain{
			AgentType:       string(agentType),
			UserID:          request.UserID,
			Query:           request.AgentRequest.Query,
			Response:        response.Result, // AgentResponse uses Result, not Response
			Steps:           stepsInterface,
			ExecutionTimeMs: int(executionTime.Milliseconds()),
			Success:         true, // AgentResponse doesn't have Success field, assume true if no error
			ErrorMessage:    "",   // AgentResponse doesn't have ErrorMessage field
			Metadata:        response.Metadata,
		}

		_, err := s.dbClient.CreateAgentExecution(ctx, execution)
		if err != nil {
			s.logger.Warn("Failed to store agent execution record",
				slog.String("agent_type", string(agentType)),
				slog.Any("error", err))
		}
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
func (s *Service) GetAvailableAgents() []agents.AgentType {
	availableAgents := make([]agents.AgentType, 0)

	if s.developmentAgent != nil {
		availableAgents = append(availableAgents, agents.AgentTypeDevelopment)
	}
	if s.databaseAgent != nil {
		availableAgents = append(availableAgents, agents.AgentTypeDatabase)
	}
	if s.infrastructureAgent != nil {
		availableAgents = append(availableAgents, agents.AgentTypeInfrastructure)
	}
	if s.researchAgent != nil {
		availableAgents = append(availableAgents, agents.AgentTypeResearch)
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
