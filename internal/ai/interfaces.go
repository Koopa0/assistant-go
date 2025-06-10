package ai

import (
	"context"

	// Assuming prompt.PromptContext is located here. Adjust if necessary.
	"github.com/koopa0/assistant-go/internal/ai/prompt"

	// Import types from subpackages of ai
	"github.com/koopa0/assistant-go/internal/ai/claude"
	"github.com/koopa0/assistant-go/internal/ai/gemini"
)

// AIService defines the interface for interacting with the AI service layer.
// Types like GenerateRequest, GenerateResponse, EmbeddingResponse, UsageStats
// are defined in 'types.go' within this 'ai' package.
// Types GenerateStreamRequest, StreamResponse are defined in 'stream.go' within this 'ai' package.
type AIService interface {
	GenerateResponse(ctx context.Context, request *GenerateRequest, providerName ...string) (*GenerateResponse, error)
	GenerateResponseStream(ctx context.Context, request *GenerateStreamRequest, providerName ...string) (*StreamResponse, error)
	GenerateEmbedding(ctx context.Context, text string, providerName ...string) (*EmbeddingResponse, error)
	ProcessEnhancedQuery(ctx context.Context, userQuery string, promptCtx *prompt.PromptContext, providerName ...string) (*EnhancedQueryResponse, error)

	GetAvailableProviders() []string
	GetDefaultProvider() string
	// SetDefaultProvider(name string) error // Decided against including this for now, can be added if a clear use case from consumers emerges.

	Health(ctx context.Context) error
	GetUsageStats(ctx context.Context) (map[string]*UsageStats, error)
	Close(ctx context.Context) error

	GetPromptService() *prompt.PromptService // Included as it's a public method on ai.Service
}

// ClaudeProviderClient defines the interface for a Claude AI provider client.
type ClaudeProviderClient interface {
	GenerateResponse(ctx context.Context, request *claude.GenerateRequest) (*claude.GenerateResponse, error)
	GenerateResponseStream(ctx context.Context, request *claude.GenerateRequest) (*claude.StreamingResponse, error)
	GenerateEmbedding(ctx context.Context, text string) (*claude.EmbeddingResponse, error) // Implemented to return an error by claude.Client
	Health(ctx context.Context) error
	GetUsage(ctx context.Context) (*claude.UsageStats, error)
	Close(ctx context.Context) error
	Name() string
}

// GeminiProviderClient defines the interface for a Gemini AI provider client.
type GeminiProviderClient interface {
	GenerateResponse(ctx context.Context, request *gemini.GenerateRequest) (*gemini.GenerateResponse, error)
	// GenerateResponseStream is not included as the current gemini.Client does not support it directly.
	// ai.Service simulates streaming for Gemini.
	GenerateEmbedding(ctx context.Context, text string) (*gemini.EmbeddingResponse, error)
	Health(ctx context.Context) error
	GetUsage(ctx context.Context) (*gemini.UsageStats, error)
	Close(ctx context.Context) error
	Name() string
}

// Note: The types like ai.GenerateRequest, ai.GenerateResponse, ai.EmbeddingResponse, ai.UsageStats,
// ai.EnhancedQueryResponse are expected to be defined in types.go within the ai package.
// Types ai.GenerateStreamRequest, ai.StreamResponse are expected to be defined in stream.go within the ai package.
// Types claude.GenerateRequest, claude.GenerateResponse, claude.EmbeddingResponse, claude.UsageStats, claude.StreamingResponse
// are expected to be in the 'claude' subpackage.
// Types gemini.GenerateRequest, gemini.GenerateResponse, gemini.EmbeddingResponse, gemini.UsageStats
// are expected to be in the 'gemini' subpackage.
// Type prompt.PromptContext is expected to be in the 'prompt' subpackage.
