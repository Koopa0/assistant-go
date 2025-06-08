package assistant

import (
	"context"
	"time"

	"github.com/koopa0/assistant-go/internal/conversation"
	"github.com/koopa0/assistant-go/internal/tool"
)

// QueryProcessor defines the interface for processing queries
// This interface handles the core query processing logic
type QueryProcessor interface {
	// ProcessQueryRequest processes a query and returns a response
	ProcessQueryRequest(ctx context.Context, req *QueryRequest) (*QueryResponse, error)
}

// ConversationManager defines the interface for managing conversations
// This interface handles all conversation-related operations
type ConversationManager interface {
	// ListConversations returns conversations for a user with pagination
	ListConversations(ctx context.Context, userID string, limit, offset int) ([]*conversation.Conversation, error)

	// GetConversation retrieves a specific conversation by ID
	GetConversation(ctx context.Context, conversationID string) (*conversation.Conversation, error)

	// GetConversationMessages retrieves all messages for a conversation
	GetConversationMessages(ctx context.Context, conversationID string) ([]*conversation.Message, error)

	// DeleteConversation removes a conversation and its messages
	DeleteConversation(ctx context.Context, conversationID string) error
}

// AssistantService combines query processing and conversation management
// This is the main interface that most consumers will use
type AssistantService interface {
	QueryProcessor
	ConversationManager
}

// HealthChecker defines the interface for health checks
type HealthChecker interface {
	// HealthCheck performs a health check and returns status
	HealthCheck(ctx context.Context) error
}

// StatsProvider defines the interface for statistics
type StatsProvider interface {
	// GetStats returns current statistics
	GetStats(ctx context.Context) (*Stats, error)
}

// ToolExecutor defines the interface for direct tool execution
// This is for cases where tools need to be executed outside query processing
type ToolExecutor interface {
	// ExecuteTool executes a specific tool with parameters
	ExecuteTool(ctx context.Context, req *ToolExecutionRequest) (*ToolExecutionResponse, error)

	// GetAvailableTools returns all available tools
	GetAvailableTools() []Tool
}

// StreamingQueryProcessor extends QueryProcessor with streaming support
type StreamingQueryProcessor interface {
	QueryProcessor

	// ProcessStreamingQuery processes a query and streams the response
	ProcessStreamingQuery(ctx context.Context, req *QueryRequest, stream chan<- string) error
}

// AssistantCore represents the core assistant interface with all capabilities
// This interface is implemented by the main Assistant struct
type AssistantCore interface {
	AssistantService
	HealthChecker
	StatsProvider
	ToolExecutor
}

// conversationWrapper wraps Assistant to implement conversation.AssistantInterface
type conversationWrapper struct {
	a *Assistant
}

// NewConversationInterface creates a wrapper that implements conversation.AssistantInterface
func (a *Assistant) AsConversationInterface() conversation.AssistantInterface {
	return &conversationWrapper{a: a}
}

func (w *conversationWrapper) ListConversations(ctx context.Context, userID string, limit, offset int) ([]*conversation.Conversation, error) {
	return w.a.ListConversations(ctx, userID, limit, offset)
}

func (w *conversationWrapper) GetConversation(ctx context.Context, conversationID string) (*conversation.Conversation, error) {
	return w.a.GetConversation(ctx, conversationID)
}

func (w *conversationWrapper) GetConversationMessages(ctx context.Context, conversationID string) ([]*conversation.Message, error) {
	return w.a.GetConversationMessages(ctx, conversationID)
}

func (w *conversationWrapper) DeleteConversation(ctx context.Context, conversationID string) error {
	return w.a.DeleteConversation(ctx, conversationID)
}

func (w *conversationWrapper) ProcessQueryRequest(ctx context.Context, req *conversation.QueryRequest) (*conversation.QueryResponse, error) {
	// Convert from conversation.QueryRequest to assistant.QueryRequest
	assistantReq := &QueryRequest{
		Query:          req.Query,
		UserID:         req.UserID,
		ConversationID: req.ConversationID,
		Context:        req.Context,
		Model:          req.Model,
	}

	// Handle optional temperature (convert *float64 to float64)
	if req.Temperature != nil {
		assistantReq.Temperature = *req.Temperature
	}

	// Handle optional max tokens (convert *int to int)
	if req.MaxTokens != nil {
		assistantReq.MaxTokens = *req.MaxTokens
	}

	// Process the request
	resp, err := w.a.ProcessQueryRequest(ctx, assistantReq)
	if err != nil {
		return nil, err
	}

	// Convert from assistant.QueryResponse to conversation.QueryResponse
	return &conversation.QueryResponse{
		Response:       resp.Response,
		ConversationID: resp.ConversationID,
		MessageID:      resp.MessageID,
		TokensUsed:     resp.TokensUsed,
		ProcessingTime: resp.ExecutionTime.Seconds(), // Convert Duration to float64 seconds
		ToolsUsed:      resp.ToolsUsed,
		Model:          resp.Model,
		Error:          "",  // assistant.QueryResponse doesn't have an error field
		Metadata:       nil, // assistant.QueryResponse doesn't have metadata
	}, nil
}

// toolWrapper wraps Assistant to implement tool.AssistantToolInterface
type toolWrapper struct {
	a *Assistant
}

// NewToolInterface creates a wrapper that implements tool.AssistantToolInterface
func (a *Assistant) AsToolInterface() tool.AssistantToolInterface {
	return &toolWrapper{a: a}
}

func (w *toolWrapper) GetAvailableTools() []tool.AssistantToolInfo {
	tools := w.a.GetAvailableTools()
	infos := make([]tool.AssistantToolInfo, 0, len(tools))

	for _, t := range tools {
		infos = append(infos, tool.AssistantToolInfo{
			Name:        t.Name,
			Description: t.Description,
			Category:    t.Category,
			Version:     t.Version,
			Author:      t.Author,
		})
	}

	return infos
}

func (w *toolWrapper) ExecuteTool(ctx context.Context, req *struct {
	ToolName string
	Input    map[string]interface{}
	Config   map[string]interface{}
}) (*struct {
	Success     bool
	Result      interface{}
	Error       string
	ToolsUsed   []string
	ElapsedTime time.Duration
	Metadata    map[string]interface{}
}, error) {
	// Convert to assistant's ToolExecutionRequest
	toolReq := &ToolExecutionRequest{
		ToolName: req.ToolName,
		Input:    req.Input,
		Config:   req.Config,
	}

	// Execute the tool
	result, err := w.a.ExecuteTool(ctx, toolReq)
	if err != nil {
		return nil, err
	}

	// Convert the result
	// Extract result data
	var resultData interface{}
	if result.Data != nil {
		resultData = result.Data.Result
	}

	// Convert metadata
	var metadata map[string]interface{}
	if result.Metadata != nil && result.Metadata.Custom != nil {
		metadata = result.Metadata.Custom
	}

	return &struct {
		Success     bool
		Result      interface{}
		Error       string
		ToolsUsed   []string
		ElapsedTime time.Duration
		Metadata    map[string]interface{}
	}{
		Success:     result.Success,
		Result:      resultData,
		Error:       result.Error,
		ToolsUsed:   []string{req.ToolName}, // Just the tool that was executed
		ElapsedTime: result.ExecutionTime,
		Metadata:    metadata,
	}, nil
}
