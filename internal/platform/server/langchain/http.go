package langchain

import (
	"encoding/json"
	"net/http"

	"log/slog"

	"github.com/koopa0/assistant-go/internal/core/agent"
	"github.com/koopa0/assistant-go/internal/langchain"
	"github.com/koopa0/assistant-go/internal/langchain/chains"
	"github.com/koopa0/assistant-go/internal/langchain/memory"
	"github.com/koopa0/assistant-go/internal/platform/observability"
)

// HTTPHandler handles HTTP requests for LangChain endpoints
type HTTPHandler struct {
	service *LangChainService
	logger  *slog.Logger
}

// NewHTTPHandler creates a new HTTP handler for LangChain endpoints
func NewHTTPHandler(service *LangChainService, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		service: service,
		logger:  observability.ServerLogger(logger, "langchain_http"),
	}
}

// RegisterRoutes registers all LangChain API routes
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	// Agent routes
	mux.HandleFunc("GET /api/langchain/agents", h.GetAvailableAgents)
	mux.HandleFunc("POST /api/langchain/agents/{type}/execute", h.ExecuteAgent)

	// Chain routes
	mux.HandleFunc("GET /api/langchain/chains", h.GetAvailableChains)
	mux.HandleFunc("POST /api/langchain/chains/{type}/execute", h.ExecuteChain)

	// Memory routes
	mux.HandleFunc("POST /api/langchain/memory", h.StoreMemory)
	mux.HandleFunc("POST /api/langchain/memory/search", h.SearchMemory)
	mux.HandleFunc("GET /api/langchain/memory/stats/{userID}", h.GetMemoryStats)

	// Service routes
	mux.HandleFunc("GET /api/langchain/providers", h.GetLLMProviders)
	mux.HandleFunc("GET /api/langchain/health", h.HealthCheck)
}

// Agent API handlers

// GetAvailableAgents returns available agent types
func (h *HTTPHandler) GetAvailableAgents(w http.ResponseWriter, r *http.Request) {
	agents := h.service.GetAvailableAgents()

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"agents": agents,
			"count":  len(agents),
		},
	}

	h.writeJSON(w, http.StatusOK, response)
}

// ExecuteAgentRequest represents the request body for agent execution
type ExecuteAgentRequest struct {
	UserID   string                 `json:"user_id"`
	Query    string                 `json:"query"`
	MaxSteps int                    `json:"max_steps,omitempty"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// ExecuteAgent executes an agent
func (h *HTTPHandler) ExecuteAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract agent type from URL path
	agentTypeStr := r.PathValue("type")
	if agentTypeStr == "" {
		h.writeError(w, http.StatusBadRequest, "agent type is required")
		return
	}

	// Parse agent type
	agentType := agent.AgentType(agentTypeStr)

	// Parse request body
	var req ExecuteAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.UserID == "" {
		h.writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.Query == "" {
		h.writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	// Set defaults
	if req.MaxSteps <= 0 {
		req.MaxSteps = 5
	}
	if req.Context == nil {
		req.Context = make(map[string]interface{})
	}

	// Create agent execution request
	agentRequest := &langchain.AgentExecutionRequest{
		UserID: req.UserID,
		Request: &agent.Request{
			Query:    req.Query,
			MaxSteps: req.MaxSteps,
			Context:  req.Context,
		},
	}

	// Execute agent
	response, err := h.service.ExecuteAgent(ctx, agentType, agentRequest)
	if err != nil {
		h.logger.Error("Agent execution failed", slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "Agent execution failed")
		return
	}

	apiResponse := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"agent_type":     agentType,
			"response":       response.Result,
			"steps":          response.Steps,
			"execution_time": response.ExecutionTime,
			"tokens_used":    response.TokensUsed,
			"success":        true,
			"error_message":  "",
			"metadata":       response.Metadata,
		},
	}

	h.writeJSON(w, http.StatusOK, apiResponse)
}

// Chain API handlers

// GetAvailableChains returns available chain types
func (h *HTTPHandler) GetAvailableChains(w http.ResponseWriter, r *http.Request) {
	chains := h.service.GetAvailableChains()

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"chains": chains,
			"count":  len(chains),
		},
	}

	h.writeJSON(w, http.StatusOK, response)
}

// ExecuteChainRequest represents the request body for chain execution
type ExecuteChainRequest struct {
	UserID  string                 `json:"user_id"`
	Input   string                 `json:"input"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// ExecuteChain executes a chain
func (h *HTTPHandler) ExecuteChain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract chain type from URL path
	chainTypeStr := r.PathValue("type")
	if chainTypeStr == "" {
		h.writeError(w, http.StatusBadRequest, "chain type is required")
		return
	}

	// Parse chain type
	chainType := chains.ChainType(chainTypeStr)

	// Parse request body
	var req ExecuteChainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.UserID == "" {
		h.writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.Input == "" {
		h.writeError(w, http.StatusBadRequest, "input is required")
		return
	}

	// Set defaults
	if req.Context == nil {
		req.Context = make(map[string]interface{})
	}

	// Create chain execution request
	chainRequest := &langchain.ChainExecutionRequest{
		UserID: req.UserID,
		ChainRequest: &chains.ChainRequest{
			Input:   req.Input,
			Context: req.Context,
		},
	}

	// Execute chain
	response, err := h.service.ExecuteChain(ctx, chainType, chainRequest)
	if err != nil {
		h.logger.Error("Chain execution failed", slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "Chain execution failed")
		return
	}

	apiResponse := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"chain_type":     chainType,
			"output":         response.Output,
			"steps":          response.Steps,
			"execution_time": response.ExecutionTime,
			"tokens_used":    response.TokensUsed,
			"success":        response.Success,
			"error_message":  response.Error,
			"metadata":       response.Metadata,
		},
	}

	h.writeJSON(w, http.StatusOK, apiResponse)
}

// Memory API handlers

// StoreMemoryRequest represents the request body for storing memory
type StoreMemoryRequest struct {
	UserID     string                 `json:"user_id"`
	Type       string                 `json:"type"`
	SessionID  string                 `json:"session_id,omitempty"`
	Content    string                 `json:"content"`
	Importance float64                `json:"importance,omitempty"`
	ExpiresAt  *string                `json:"expires_at,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// StoreMemory stores a memory entry
func (h *HTTPHandler) StoreMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req StoreMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.UserID == "" {
		h.writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.Content == "" {
		h.writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	// Set defaults
	if req.Importance <= 0 {
		req.Importance = 0.5
	}
	if req.Metadata == nil {
		req.Metadata = make(map[string]interface{})
	}

	// Create memory entry
	entry := &memory.MemoryEntry{
		UserID:     req.UserID,
		Type:       memory.MemoryType(req.Type),
		SessionID:  req.SessionID,
		Content:    req.Content,
		Importance: req.Importance,
		Metadata:   req.Metadata,
	}

	// Store memory
	err := h.service.StoreMemory(ctx, entry)
	if err != nil {
		h.logger.Error("Failed to store memory", slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "Failed to store memory")
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Memory stored successfully",
	}

	h.writeJSON(w, http.StatusCreated, response)
}

// SearchMemoryRequest represents the request body for memory search
type SearchMemoryRequest struct {
	UserID    string                 `json:"user_id"`
	Query     string                 `json:"query,omitempty"`
	Types     []string               `json:"types,omitempty"`
	Limit     int                    `json:"limit,omitempty"`
	Threshold float64                `json:"threshold,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SearchMemory searches for memories
func (h *HTTPHandler) SearchMemory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SearchMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.UserID == "" {
		h.writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Threshold <= 0 {
		req.Threshold = 0.7
	}

	// Convert string types to MemoryType
	var memoryTypes []memory.MemoryType
	for _, typeStr := range req.Types {
		memoryTypes = append(memoryTypes, memory.MemoryType(typeStr))
	}

	// Create memory query
	query := &memory.MemoryQuery{
		UserID:     req.UserID,
		Content:    req.Query,
		Types:      memoryTypes,
		Limit:      req.Limit,
		Similarity: req.Threshold,
	}

	// Search memories
	results, err := h.service.SearchMemory(ctx, query)
	if err != nil {
		h.logger.Error("Memory search failed", slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "Memory search failed")
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"results": results,
			"count":   len(results),
		},
	}

	h.writeJSON(w, http.StatusOK, response)
}

// GetMemoryStats returns memory usage statistics for a user
func (h *HTTPHandler) GetMemoryStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract userID from URL path
	userID := r.PathValue("userID")
	if userID == "" {
		h.writeError(w, http.StatusBadRequest, "userID is required")
		return
	}

	// Get memory stats
	stats, err := h.service.GetMemoryStats(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get memory stats", slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "Failed to get memory stats")
		return
	}

	response := map[string]interface{}{
		"success": true,
		"data":    stats,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Service API handlers

// GetLLMProviders returns available LLM providers
func (h *HTTPHandler) GetLLMProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.service.GetLLMProviders()

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"providers": providers,
			"count":     len(providers),
		},
	}

	h.writeJSON(w, http.StatusOK, response)
}

// HealthCheck performs a health check on the LangChain service
func (h *HTTPHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := h.service.HealthCheck(ctx)
	if err != nil {
		h.logger.Error("Health check failed", slog.Any("error", err))
		h.writeError(w, http.StatusServiceUnavailable, "Health check failed")
		return
	}

	response := map[string]interface{}{
		"success": true,
		"status":  "healthy",
		"message": "LangChain service is operational",
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Helper methods

// writeJSON writes a JSON response
func (h *HTTPHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to write JSON response", slog.Any("error", err))
	}
}

// writeError writes an error response
func (h *HTTPHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"message": message,
			"code":    statusCode,
		},
	}

	h.writeJSON(w, statusCode, response)
}
