package http

import (
	"encoding/json"
	"net/http"

	"log/slog"

	"github.com/koopa0/assistant-go/internal/langchain"
	"github.com/koopa0/assistant-go/internal/langchain/agent"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/server/handlers"
)

// Handler handles HTTP requests for LangChain endpoints
type Handler struct {
	*handlers.Handler
	service *langchain.Service
}

// NewHandler creates a new HTTP handler for LangChain endpoints
func NewHandler(service *langchain.Service, logger *slog.Logger) *Handler {
	loggerWithName := observability.ServerLogger(logger, "langchain_http")
	return &Handler{
		Handler: handlers.NewHandler(loggerWithName),
		service: service,
	}
}

// GetAvailableAgents returns list of available agents
func (h *Handler) GetAvailableAgents(w http.ResponseWriter, r *http.Request) {
	agents := h.getAvailableAgents()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": agents,
	})
}

// getAvailableAgents returns available agent types as strings
func (h *Handler) getAvailableAgents() []string {
	if h.service == nil {
		return []string{}
	}
	agentTypes := h.service.GetAgentTypes()
	result := make([]string, len(agentTypes))
	for i, agentType := range agentTypes {
		result[i] = string(agentType)
	}
	return result
}

// ExecuteAgentRequest represents a request to execute an agent
type ExecuteAgentRequest struct {
	UserID      string                 `json:"user_id"`
	Query       string                 `json:"query"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Tools       []string               `json:"tools,omitempty"`
	MaxSteps    int                    `json:"max_steps,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
}

// ExecuteAgent executes a specific agent
func (h *Handler) ExecuteAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract agent type from URL path
	agentTypeStr := r.PathValue("type")
	if agentTypeStr == "" {
		h.WriteBadRequest(w, "agent type is required")
		return
	}

	// Parse agent type
	agentType := agent.AgentType(agentTypeStr)

	// Parse request body
	var req ExecuteAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteBadRequest(w, "Invalid request body")
		return
	}

	// Validate request
	if req.UserID == "" {
		h.WriteBadRequest(w, "user_id is required")
		return
	}
	if req.Query == "" {
		h.WriteBadRequest(w, "query is required")
		return
	}

	// Set defaults
	if req.MaxSteps <= 0 {
		req.MaxSteps = 5
	}
	if req.Context == nil {
		req.Context = make(map[string]interface{})
	}
	if req.Temperature <= 0 {
		req.Temperature = 0.7
	}

	// Create agent execution request
	agentRequest := &agent.Request{
		Query:       req.Query,
		MaxSteps:    req.MaxSteps,
		Context:     req.Context,
		Tools:       req.Tools,
		Temperature: req.Temperature,
	}

	// Execute agent
	response, err := h.service.ExecuteAgent(ctx, agentType, agentRequest)
	if err != nil {
		h.LogError(r, "langchain.execute_agent", err)
		h.WriteInternalError(w, err)
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        response.Success,
		"result":         response.Result,
		"confidence":     response.Confidence,
		"execution_time": response.ExecutionTime.Milliseconds(),
		"steps":          response.Steps,
	})
}

// ExecutePromptRequest represents a request to execute a prompt
type ExecutePromptRequest struct {
	Prompt string `json:"prompt"`
}

// ExecutePrompt executes a simple prompt
func (h *Handler) ExecutePrompt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req ExecutePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteBadRequest(w, "Invalid request body")
		return
	}

	// Validate request
	if req.Prompt == "" {
		h.WriteBadRequest(w, "prompt is required")
		return
	}

	// Execute prompt
	response, err := h.service.ExecutePrompt(ctx, req.Prompt)
	if err != nil {
		h.LogError(r, "langchain.execute_prompt", err)
		h.WriteInternalError(w, err)
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"result":  response,
	})
}
