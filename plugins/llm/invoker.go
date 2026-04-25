// Package llm provides LLM provider implementations for the LLMInvoker interface.
package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/eval-prompt/internal/config"
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

	// Ping performs a lightweight health check using pingPath.
	Ping(ctx context.Context) error
}

// NewProvider creates a new LLM provider based on the given config.
func NewProvider(cfg config.LLMProviderConfig) (Interface, error) {
	switch cfg.Provider {
	case "openai":
		return NewOpenAIProvider(cfg.APIKey, cfg.Endpoint, cfg.PingPath), nil
	case "claude":
		return NewClaudeProvider(cfg.APIKey, cfg.Endpoint, cfg.PingPath)
	case "ollama":
		return NewOllamaProvider(cfg.Endpoint, cfg.PingPath)
	default:
		return nil, fmt.Errorf("llm: unknown provider type: %s", cfg.Provider)
	}
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

// Ping implements Interface. Noop always returns error.
func (n *NoopInvoker) Ping(_ context.Context) error {
	return errNoop
}

// Ensure NoopInvoker implements the service.LLMInvoker interface.
var _ Interface = (*NoopInvoker)(nil)

// Interface is the LLMInvoker interface alias for the plugin layer.
type Interface interface {
	Invoke(ctx context.Context, prompt string, model string, temperature float64) (*LLMResponse, error)
	InvokeWithSchema(ctx context.Context, prompt string, schema json.RawMessage) (json.RawMessage, error)
	// Ping performs a lightweight health check. Returns nil if healthy, error otherwise.
	// If PingPath is empty, returns nil (skip check).
	Ping(ctx context.Context) error
}

// pingTCP checks TCP connectivity to the host of the given URL.
func pingTCP(ctx context.Context, endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("ping: parse endpoint: %w", err)
	}

	host := u.Host
	if !strings.Contains(host, ":") {
		// Add default port based on scheme
		if u.Scheme == "https" {
			host = net.JoinHostPort(host, "443")
		} else {
			host = net.JoinHostPort(host, "80")
		}
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", host)
	if err != nil {
		return fmt.Errorf("ping: TCP connect to %s: %w", host, err)
	}
	conn.Close()
	return nil
}
