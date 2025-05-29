package langchain

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/koopa0/assistant/internal/langchain"
	"github.com/koopa0/assistant/internal/langchain/agents"
	"github.com/koopa0/assistant/internal/langchain/chains"
	"github.com/koopa0/assistant/internal/langchain/memory"
)

// JSONResponse implements a JSON encoder for HTTP responses
type JSONResponse struct {
	Data       interface{} `json:"data,omitempty"`
	StatusCode int         `json:"-"`
}

// Encode implements the web.Encoder interface
func (jr JSONResponse) Encode() ([]byte, string, error) {
	data, err := json.Marshal(jr.Data)
	return data, "application/json", err
}

// HTTPStatus implements the httpStatus interface for setting status codes
func (jr JSONResponse) HTTPStatus() int {
	if jr.StatusCode == 0 {
		return http.StatusOK
	}
	return jr.StatusCode
}

// HandlerFunc represents a function that handles HTTP requests
type HandlerFunc func(ctx context.Context, r *http.Request) JSONResponse

// Handler handles LangChain API requests
type Handler struct {
	service *langchain.Service
	logger  *slog.Logger
}

// NewHandler creates a new LangChain API handler
func NewHandler(service *langchain.Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers all LangChain API routes using standard HTTP mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Agent routes
	mux.HandleFunc("GET /api/langchain/agents", h.wrapHandler(h.GetAvailableAgents))
	mux.HandleFunc("POST /api/langchain/agents/{type}/execute", h.wrapHandler(h.ExecuteAgent))

	// Chain routes
	mux.HandleFunc("GET /api/langchain/chains", h.wrapHandler(h.GetAvailableChains))
	mux.HandleFunc("POST /api/langchain/chains/{type}/execute", h.wrapHandler(h.ExecuteChain))

	// Memory routes
	mux.HandleFunc("POST /api/langchain/memory", h.wrapHandler(h.StoreMemory))
	mux.HandleFunc("POST /api/langchain/memory/search", h.wrapHandler(h.SearchMemory))
	mux.HandleFunc("GET /api/langchain/memory/stats/{userID}", h.wrapHandler(h.GetMemoryStats))

	// Service routes
	mux.HandleFunc("GET /api/langchain/providers", h.wrapHandler(h.GetLLMProviders))
	mux.HandleFunc("GET /api/langchain/health", h.wrapHandler(h.HealthCheck))
}

// wrapHandler wraps our custom handler function to work with standard HTTP handlers
func (h *Handler) wrapHandler(handlerFunc HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Call our handler
		response := handlerFunc(ctx, r)

		// Encode and write the response
		data, contentType, err := response.Encode()
		if err != nil {
			h.logger.Error("Failed to encode response", slog.Any("error", err))
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(response.HTTPStatus())

		if _, err := w.Write(data); err != nil {
			h.logger.Error("Failed to write response", slog.Any("error", err))
		}
	}
}

// Agent API handlers

// GetAvailableAgents returns available agent types
func (h *Handler) GetAvailableAgents(ctx context.Context, r *http.Request) JSONResponse {
	agents := h.service.GetAvailableAgents()

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"agents": agents,
			"count":  len(agents),
		},
	}

	return JSONResponse{Data: response, StatusCode: http.StatusOK}
}

// ExecuteAgentRequest represents the request body for agent execution
type ExecuteAgentRequest struct {
	UserID   string                 `json:"user_id"`
	Query    string                 `json:"query"`
	MaxSteps int                    `json:"max_steps,omitempty"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// ExecuteAgent executes an agent
func (h *Handler) ExecuteAgent(ctx context.Context, r *http.Request) JSONResponse {
	// Extract agent type from URL path using Go 1.22+ path parameters
	agentTypeStr := r.PathValue("type")
	if agentTypeStr == "" {
		return h.errorResponse(http.StatusBadRequest, "agent type is required", nil)
	}

	// Parse agent type
	agentType := agents.AgentType(agentTypeStr)

	// Parse request body
	var req ExecuteAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return h.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate request
	if req.UserID == "" {
		return h.errorResponse(http.StatusBadRequest, "user_id is required", nil)
	}
	if req.Query == "" {
		return h.errorResponse(http.StatusBadRequest, "query is required", nil)
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
		AgentRequest: &agents.AgentRequest{
			Query:    req.Query,
			MaxSteps: req.MaxSteps,
			Context:  req.Context,
		},
	}

	// Execute agent
	response, err := h.service.ExecuteAgent(ctx, agentType, agentRequest)
	if err != nil {
		return h.errorResponse(http.StatusInternalServerError, "Agent execution failed", err)
	}

	apiResponse := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"agent_type":     agentType,
			"response":       response.Result, // AgentResponse uses Result, not Response
			"steps":          response.Steps,
			"execution_time": response.ExecutionTime,
			"tokens_used":    response.TokensUsed,
			"success":        true, // AgentResponse doesn't have Success field, assume true if no error
			"error_message":  "",   // AgentResponse doesn't have ErrorMessage field
			"metadata":       response.Metadata,
		},
	}

	return JSONResponse{Data: apiResponse, StatusCode: http.StatusOK}
}

// Chain API handlers

// GetAvailableChains returns available chain types
func (h *Handler) GetAvailableChains(ctx context.Context, r *http.Request) JSONResponse {
	chains := h.service.GetAvailableChains()

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"chains": chains,
			"count":  len(chains),
		},
	}

	return JSONResponse{Data: response, StatusCode: http.StatusOK}
}

// ExecuteChainRequest represents the request body for chain execution
type ExecuteChainRequest struct {
	UserID  string                 `json:"user_id"`
	Input   string                 `json:"input"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// ExecuteChain executes a chain
func (h *Handler) ExecuteChain(ctx context.Context, r *http.Request) JSONResponse {
	// Extract chain type from URL path using Go 1.22+ path parameters
	chainTypeStr := r.PathValue("type")
	if chainTypeStr == "" {
		return h.errorResponse(http.StatusBadRequest, "chain type is required", nil)
	}

	// Parse chain type
	chainType := chains.ChainType(chainTypeStr)

	// Parse request body
	var req ExecuteChainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return h.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate request
	if req.UserID == "" {
		return h.errorResponse(http.StatusBadRequest, "user_id is required", nil)
	}
	if req.Input == "" {
		return h.errorResponse(http.StatusBadRequest, "input is required", nil)
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
		return h.errorResponse(http.StatusInternalServerError, "Chain execution failed", err)
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
			"error_message":  response.Error, // ChainResponse uses Error, not ErrorMessage
			"metadata":       response.Metadata,
		},
	}

	return JSONResponse{Data: apiResponse, StatusCode: http.StatusOK}
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
func (h *Handler) StoreMemory(ctx context.Context, r *http.Request) JSONResponse {
	var req StoreMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return h.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate request
	if req.UserID == "" {
		return h.errorResponse(http.StatusBadRequest, "user_id is required", nil)
	}
	if req.Content == "" {
		return h.errorResponse(http.StatusBadRequest, "content is required", nil)
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
		Type:       memory.MemoryType(req.Type), // Convert string to MemoryType
		SessionID:  req.SessionID,
		Content:    req.Content,
		Importance: req.Importance,
		Metadata:   req.Metadata,
	}

	// Store memory
	err := h.service.StoreMemory(ctx, entry)
	if err != nil {
		return h.errorResponse(http.StatusInternalServerError, "Failed to store memory", err)
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Memory stored successfully",
	}

	return JSONResponse{Data: response, StatusCode: http.StatusCreated}
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
func (h *Handler) SearchMemory(ctx context.Context, r *http.Request) JSONResponse {
	var req SearchMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return h.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate request
	if req.UserID == "" {
		return h.errorResponse(http.StatusBadRequest, "user_id is required", nil)
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
		Similarity: req.Threshold, // MemoryQuery uses Similarity, not Threshold
		// Note: MemoryQuery doesn't have Metadata field, so we skip it
	}

	// Search memories
	results, err := h.service.SearchMemory(ctx, query)
	if err != nil {
		return h.errorResponse(http.StatusInternalServerError, "Memory search failed", err)
	}

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"results": results,
			"count":   len(results),
		},
	}

	return JSONResponse{Data: response, StatusCode: http.StatusOK}
}

// GetMemoryStats returns memory usage statistics for a user
func (h *Handler) GetMemoryStats(ctx context.Context, r *http.Request) JSONResponse {
	// Extract userID from URL path using Go 1.22+ path parameters
	userID := r.PathValue("userID")
	if userID == "" {
		return h.errorResponse(http.StatusBadRequest, "userID is required", nil)
	}

	// Get memory stats
	stats, err := h.service.GetMemoryStats(ctx, userID)
	if err != nil {
		return h.errorResponse(http.StatusInternalServerError, "Failed to get memory stats", err)
	}

	response := map[string]interface{}{
		"success": true,
		"data":    stats,
	}

	return JSONResponse{Data: response, StatusCode: http.StatusOK}
}

// Service API handlers

// GetLLMProviders returns available LLM providers
func (h *Handler) GetLLMProviders(ctx context.Context, r *http.Request) JSONResponse {
	providers := h.service.GetLLMProviders()

	response := map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"providers": providers,
			"count":     len(providers),
		},
	}

	return JSONResponse{Data: response, StatusCode: http.StatusOK}
}

// HealthCheck performs a health check on the LangChain service
func (h *Handler) HealthCheck(ctx context.Context, r *http.Request) JSONResponse {
	err := h.service.HealthCheck(ctx)
	if err != nil {
		return h.errorResponse(http.StatusServiceUnavailable, "Health check failed", err)
	}

	response := map[string]interface{}{
		"success": true,
		"status":  "healthy",
		"message": "LangChain service is operational",
	}

	return JSONResponse{Data: response, StatusCode: http.StatusOK}
}

// Helper methods

// errorResponse creates an error response
func (h *Handler) errorResponse(statusCode int, message string, err error) JSONResponse {
	h.logger.Error("API error",
		slog.String("message", message),
		slog.Int("status_code", statusCode),
		slog.Any("error", err))

	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"message": message,
			"code":    statusCode,
		},
	}

	if err != nil {
		response["error"].(map[string]interface{})["details"] = err.Error()
	}

	return JSONResponse{Data: response, StatusCode: statusCode}
}
