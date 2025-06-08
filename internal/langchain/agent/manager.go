package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/llms"
)

// Manager manages and coordinates multiple agents
type Manager struct {
	agents map[AgentType]Agent
	llm    llms.Model
	logger *slog.Logger
	mu     sync.RWMutex
}

// NewManager creates a new agent manager
func NewManager(llm llms.Model, logger *slog.Logger) *Manager {
	m := &Manager{
		agents: make(map[AgentType]Agent),
		llm:    llm,
		logger: logger,
	}

	// Initialize default agents if LLM is available
	if llm != nil {
		m.initializeDefaultAgents()
	}

	return m
}

// initializeDefaultAgents creates the standard set of agents
func (m *Manager) initializeDefaultAgents() {
	// Create development agent
	developmentAgent := NewDevelopmentAgent(m.llm, m.logger)
	m.RegisterAgent(TypeDevelopment, developmentAgent)

	// Create database agent
	databaseAgent := NewDatabaseAgent(m.llm, m.logger)
	m.RegisterAgent(TypeDatabase, databaseAgent)

	// Create infrastructure agent
	infrastructureAgent := NewInfrastructureAgent(m.llm, m.logger)
	m.RegisterAgent(TypeInfrastructure, infrastructureAgent)

	// Create research agent
	researchAgent := NewResearchAgent(m.llm, m.logger)
	m.RegisterAgent(TypeResearch, researchAgent)

	// Create general agent (uses base agent directly)
	generalAgent := NewBaseAgent(TypeGeneral, m.llm, m.logger)
	m.RegisterAgent(TypeGeneral, generalAgent)

	m.logger.Info("Default agents initialized",
		slog.Int("agent_count", len(m.agents)))
}

// RegisterAgent registers a new agent with the manager
func (m *Manager) RegisterAgent(agentType AgentType, agent Agent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if agent == nil {
		return fmt.Errorf("agent cannot be nil")
	}

	m.agents[agentType] = agent
	m.logger.Info("Agent registered",
		slog.String("type", string(agentType)))

	return nil
}

// GetAgent retrieves an agent by type
func (m *Manager) GetAgent(agentType AgentType) (Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[agentType]
	if !exists {
		return nil, fmt.Errorf("agent type %s not found", agentType)
	}

	return agent, nil
}

// ExecuteWithAgent executes a request with a specific agent type
func (m *Manager) ExecuteWithAgent(ctx context.Context, agentType AgentType, request *Request) (*Response, error) {
	agent, err := m.GetAgent(agentType)
	if err != nil {
		return nil, err
	}

	return agent.Execute(ctx, request)
}

// GetAvailableAgents returns a list of registered agent types
func (m *Manager) GetAvailableAgents() []AgentType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	types := make([]AgentType, 0, len(m.agents))
	for agentType := range m.agents {
		types = append(types, agentType)
	}

	return types
}

// SelectBestAgent analyzes a request and selects the most appropriate agent
func (m *Manager) SelectBestAgent(request *Request) (AgentType, error) {
	// Simple heuristic-based selection
	// In a real implementation, this could use the LLM to analyze the query

	query := request.Query

	// Check for development-related keywords
	developmentKeywords := []string{"code", "implement", "function", "class", "bug", "error", "test", "refactor", "go", "golang"}
	for _, keyword := range developmentKeywords {
		if containsIgnoreCase(query, keyword) {
			return TypeDevelopment, nil
		}
	}

	// Check for database-related keywords
	databaseKeywords := []string{"sql", "query", "database", "postgres", "table", "index", "migration", "schema"}
	for _, keyword := range databaseKeywords {
		if containsIgnoreCase(query, keyword) {
			return TypeDatabase, nil
		}
	}

	// Check for infrastructure-related keywords
	infraKeywords := []string{"docker", "kubernetes", "k8s", "deploy", "container", "ci/cd", "cloud", "aws", "gcp"}
	for _, keyword := range infraKeywords {
		if containsIgnoreCase(query, keyword) {
			return TypeInfrastructure, nil
		}
	}

	// Check for research-related keywords
	researchKeywords := []string{"research", "analyze", "compare", "explain", "documentation", "best practices", "learn"}
	for _, keyword := range researchKeywords {
		if containsIgnoreCase(query, keyword) {
			return TypeResearch, nil
		}
	}

	// Default to general agent
	return TypeGeneral, nil
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
