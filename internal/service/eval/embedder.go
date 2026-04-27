package eval

import (
	"context"
	"fmt"
)

// Embedder generates text embeddings via LLM interface calls.
type Embedder interface {
	// Embed generates embeddings for the given texts.
	// Returns a 2D slice where result[i] is the embedding for texts[i].
	Embed(ctx context.Context, texts []string) ([][]float64, error)

	// Dimension returns the embedding dimension.
	Dimension() int

	// Name returns the embedder name (e.g., "openai/text-embedding-3-small").
	Name() string
}

// LLMEmbedder wraps an LLM interface to implement the Embedder interface.
type LLMEmbedder struct {
	invoker  LLMInvokerForEmbed
	provider string
	model   string
	dim     int
}

// LLMInvokerForEmbed is the interface for LLM providers that support embeddings.
type LLMInvokerForEmbed interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}

// NewLLMEmbedder creates a new embedder that delegates to an LLM interface.
func NewLLMEmbedder(invoker LLMInvokerForEmbed, provider, model string, dimension int) *LLMEmbedder {
	return &LLMEmbedder{
		invoker:  invoker,
		provider: provider,
		model:    model,
		dim:      dimension,
	}
}

// Embed implements Embedder by delegating to the LLM interface.
func (e *LLMEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if e.invoker == nil {
		return nil, fmt.Errorf("embedder: no LLM invoker configured")
	}
	return e.invoker.Embed(ctx, texts)
}

// Dimension implements Embedder.
func (e *LLMEmbedder) Dimension() int { return e.dim }

// Name implements Embedder.
func (e *LLMEmbedder) Name() string { return e.provider + "/" + e.model }
