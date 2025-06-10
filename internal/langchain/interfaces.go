package langchain

import (
	"context" // Added context in case methods like Health are added later
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

// ClientService defines the interface for the Langchain client,
// exposing methods needed by the langchain.Service.
type ClientService interface {
	LLM() llms.Model          // To access the underlying LLM model.
	GetMemory() schema.Memory // Matches existing method on Client.
	Name() string             // Returns the name of the client/provider.

	// Consider adding these if langchain.Service needs them directly in the future,
	// or if other consumers would benefit from a common interface for execution via langchain.Client.
	// GenerateResponse(ctx context.Context, request *GenerateRequest) (*GenerateResponse, error)
	// Health(ctx context.Context) error
}

// Note: GenerateRequest and GenerateResponse in the commented out section above would refer to
// types that might be defined in "github.com/koopa0/assistant-go/internal/langchain" if those
// operations were to be part of this interface. Currently, they are handled by the specific
// AI provider clients (Claude, Gemini) or the top-level ai.Service.
