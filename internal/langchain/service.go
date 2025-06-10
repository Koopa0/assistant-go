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
	// client holds an instance that implements ClientService, providing access
	// to the underlying Langchain client's capabilities like LLM model and memory.
	client  ClientService // Changed type
	logger  *slog.Logger
	queries *sqlc.Queries
	manager *agent.Manager
}

// NewService creates a new LangChain service.
// The client parameter now accepts any type that implements the ClientService interface,
// allowing for more flexible Langchain client implementations or mocks.
func NewService(client ClientService, logger *slog.Logger, queries *sqlc.Queries) *Service { // Changed client type
	var llmModel llms.Model // Renamed for clarity from llm to llmModel
	if client != nil {
		llmModel = client.LLM() // Use the new LLM() method from the interface
	}

	service := &Service{
		client:  client,
		logger:  logger,
		queries: queries,
		// Ensure agent.NewManager can handle a nil llmModel if client is nil or client.LLM() is nil
		manager: agent.NewManager(llmModel, logger),
	}
	// service.logger.Info("Langchain service initialized", slog.Bool("client_available", client != nil)) // Logger might not be set yet if client is nil and causes NewService to return early in other versions
	if logger != nil {
		logger.Info("Langchain service initialized", slog.Bool("client_available", client != nil))
	}
	return service
}

// ExecutePrompt executes a simple prompt
func (s *Service) ExecutePrompt(ctx context.Context, prompt string) (string, error) {
	if s.client == nil || s.client.LLM() == nil { // Check client and its LLM
		return "", fmt.Errorf("LLM client or model not initialized in Langchain service")
	}

	response, err := llms.GenerateFromSinglePrompt(ctx, s.client.LLM(), prompt) // Use client.LLM()
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
