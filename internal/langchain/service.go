// Package langchain provides integration with the LangChain framework for Go.
// It includes agents for specialized domains (development, database, infrastructure),
// chains for complex workflows, document processing, and vector store integration for RAG.
package langchain

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/koopa0/assistant-go/internal/langchain/agent"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
	"github.com/tmc/langchaingo/llms"
)

// Service provides LangChain integration services
type Service struct {
	client  *LangChainClient
	logger  *slog.Logger
	queries *sqlc.Queries
	manager *agent.Manager
}

// NewService creates a new LangChain service
func NewService(client *LangChainClient, logger *slog.Logger, queries *sqlc.Queries) *Service {
	var llm llms.Model
	if client != nil {
		llm = client.llm
	}

	service := &Service{
		client:  client,
		logger:  logger,
		queries: queries,
		manager: agent.NewManager(llm, logger),
	}

	return service
}

// ExecutePrompt executes a simple prompt
func (s *Service) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	if s.client == nil || s.client.llm == nil {
		return "", fmt.Errorf("LLM client not initialized")
	}

	response, err := llms.GenerateFromSinglePrompt(ctx, s.client.llm, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	return response, nil
}

// ExecuteAgent executes an agent request
func (s *Service) ExecuteAgent(ctx context.Context, agentType agent.AgentType, request *agent.Request) (*agent.Response, error) {
	if s.manager == nil {
		return nil, fmt.Errorf("agent manager not initialized")
	}

	return s.manager.ExecuteWithAgent(ctx, agentType, request)
}

// GetAgentTypes returns available agent types
func (s *Service) GetAgentTypes() []agent.AgentType {
	if s.manager == nil {
		return []agent.AgentType{}
	}
	return s.manager.GetAvailableAgents()
}

// AgentExecutionRequest wraps the agent request with user context
type AgentExecutionRequest struct {
	UserID string `json:"user_id"`
	*agent.Request
}
