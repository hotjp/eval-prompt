// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/eval-prompt/plugins/llm"
)

// LLMCheckerAdapter wraps an LLM invoker to check readiness.
type LLMCheckerAdapter struct {
	invoker llm.Interface
}

// NewLLMCheckerAdapter creates a new LLMCheckerAdapter.
func NewLLMCheckerAdapter(invoker llm.Interface) *LLMCheckerAdapter {
	return &LLMCheckerAdapter{invoker: invoker}
}

// Ping attempts to ping the LLM provider with a lightweight health check.
func (c *LLMCheckerAdapter) Ping(ctx context.Context) error {
	if c.invoker == nil {
		return errors.New("llm: no invoker configured")
	}

	// Use lightweight ping instead of Invoke to avoid consuming tokens
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.invoker.Ping(pingCtx)
}
