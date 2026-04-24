// Package llm provides LLM provider implementations for the LLMInvoker interface.
package llm

import (
	"context"
	"encoding/json"
	"errors"
)

// Config holds LLM plugin configuration.
type Config struct {
	Provider     string // openai | claude | ollama
	APIKey       string
	Endpoint     string // optional custom endpoint
	DefaultModel string
}

// Provider is the interface for LLM providers.
type Provider interface {
	// Invoke performs a text completion.
	Invoke(ctx context.Context, prompt string, model string, temperature float64) (*LLMResponse, error)

	// InvokeWithSchema performs a structured completion using JSON schema.
	InvokeWithSchema(ctx context.Context, prompt string, schema json.RawMessage) (json.RawMessage, error)

	// Name returns the provider name.
	Name() string
}

// LLMResponse contains the LLM output and metadata.
type LLMResponse struct {
	Content    string
	Model      string
	TokensIn   int
	TokensOut  int
	StopReason string
	RawResponse json.RawMessage
}

// NoopInvoker is a no-operation LLMInvoker for when no LLM plugin is enabled.
type NoopInvoker struct{}

var errNoop = errors.New("llm: noop invoker, enable a provider plugin")

// Invoke implements LLMInvoker.
func (n *NoopInvoker) Invoke(_ context.Context, _, _ string, _ float64) (*LLMResponse, error) {
	return nil, errNoop
}

// InvokeWithSchema implements LLMInvoker.
func (n *NoopInvoker) InvokeWithSchema(_ context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
	return nil, errNoop
}

// Ensure NoopInvoker implements the service.LLMInvoker interface.
var _ Interface = (*NoopInvoker)(nil)

// Interface is the LLMInvoker interface alias for the plugin layer.
type Interface interface {
	Invoke(ctx context.Context, prompt string, model string, temperature float64) (*LLMResponse, error)
	InvokeWithSchema(ctx context.Context, prompt string, schema json.RawMessage) (json.RawMessage, error)
}
