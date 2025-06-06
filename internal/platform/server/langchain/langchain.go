package langchain

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/koopa0/assistant-go/internal/core/agent"
	"github.com/koopa0/assistant-go/internal/langchain"
	"github.com/koopa0/assistant-go/internal/langchain/chains"
	"github.com/koopa0/assistant-go/internal/langchain/memory"
	"github.com/koopa0/assistant-go/internal/platform/observability"
)

// LangChainService handles LangChain operations
type LangChainService struct {
	service *langchain.Service
	logger  *slog.Logger
}

// NewLangChainService creates a new LangChain service
func NewLangChainService(service *langchain.Service, logger *slog.Logger) *LangChainService {
	return &LangChainService{
		service: service,
		logger:  observability.ServerLogger(logger, "langchain"),
	}
}

// GetAvailableAgents returns available agent types
func (s *LangChainService) GetAvailableAgents() []string {
	if s.service == nil {
		return []string{}
	}
	agentTypes := s.service.GetAvailableAgents()
	result := make([]string, len(agentTypes))
	for i, agentType := range agentTypes {
		result[i] = string(agentType)
	}
	return result
}

// ExecuteAgent executes an agent
func (s *LangChainService) ExecuteAgent(ctx context.Context, agentType agent.AgentType, req *langchain.AgentExecutionRequest) (*agent.Response, error) {
	if s.service == nil {
		return nil, fmt.Errorf("LangChain service not available")
	}
	return s.service.ExecuteAgent(ctx, agentType, req)
}

// GetAvailableChains returns available chain types
func (s *LangChainService) GetAvailableChains() []string {
	if s.service == nil {
		return []string{}
	}
	chainTypes := s.service.GetAvailableChains()
	result := make([]string, len(chainTypes))
	for i, chainType := range chainTypes {
		result[i] = string(chainType)
	}
	return result
}

// ExecuteChain executes a chain
func (s *LangChainService) ExecuteChain(ctx context.Context, chainType chains.ChainType, req *langchain.ChainExecutionRequest) (*chains.ChainResponse, error) {
	if s.service == nil {
		return nil, fmt.Errorf("LangChain service not available")
	}
	return s.service.ExecuteChain(ctx, chainType, req)
}

// StoreMemory stores a memory entry
func (s *LangChainService) StoreMemory(ctx context.Context, entry *memory.MemoryEntry) error {
	if s.service == nil {
		return fmt.Errorf("LangChain service not available")
	}
	return s.service.StoreMemory(ctx, entry)
}

// SearchMemory searches for memories
func (s *LangChainService) SearchMemory(ctx context.Context, query *memory.MemoryQuery) ([]*memory.MemorySearchResult, error) {
	if s.service == nil {
		return nil, fmt.Errorf("LangChain service not available")
	}
	return s.service.SearchMemory(ctx, query)
}

// GetMemoryStats returns memory usage statistics for a user
func (s *LangChainService) GetMemoryStats(ctx context.Context, userID string) (*memory.MemoryStats, error) {
	if s.service == nil {
		return nil, fmt.Errorf("LangChain service not available")
	}
	return s.service.GetMemoryStats(ctx, userID)
}

// GetLLMProviders returns available LLM providers
func (s *LangChainService) GetLLMProviders() []string {
	if s.service == nil {
		return []string{}
	}
	return s.service.GetLLMProviders()
}

// HealthCheck performs a health check on the LangChain service
func (s *LangChainService) HealthCheck(ctx context.Context) error {
	if s.service == nil {
		return fmt.Errorf("LangChain service not available")
	}
	return s.service.HealthCheck(ctx)
}
