// Package service implements L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/eval-prompt/plugins/llm"
)

// llmInvokerAdapter wraps a llm.Interface to satisfy service.LLMInvoker.
type llmInvokerAdapter struct {
	invoker llm.Interface
}

// Invoke implements service.LLMInvoker by delegating to the wrapped llm.Interface.
func (a *llmInvokerAdapter) Invoke(ctx context.Context, prompt string, model string, temperature float64) (*llm.LLMResponse, error) {
	return a.invoker.Invoke(ctx, prompt, model, temperature)
}

// InvokeWithSchema implements service.LLMInvoker.
func (a *llmInvokerAdapter) InvokeWithSchema(ctx context.Context, prompt string, schema []byte) ([]byte, error) {
	return a.invoker.InvokeWithSchema(ctx, prompt, schema)
}

// SemanticService provides LLM-based semantic analysis capabilities.
type SemanticService struct {
	llmInvoker LLMInvoker
	model      string
}

// NewSemanticService creates a new SemanticService.
func NewSemanticService(invoker llm.Interface, model string) *SemanticService {
	// Wrap the plugin's llm.Interface to satisfy service.LLMInvoker
	wrapped := &llmInvokerAdapter{invoker: invoker}
	return &SemanticService{
		llmInvoker: wrapped,
		model:      model,
	}
}

// Ensure SemanticService implements SemanticAnalyzer.
var _ SemanticAnalyzer = (*SemanticService)(nil)

// AnalyzeContent performs semantic analysis on prompt content.
func (s *SemanticService) AnalyzeContent(ctx context.Context, req AnalyzeContentRequest) (*AnalyzeContentResult, error) {
	if s.llmInvoker == nil {
		return nil, fmt.Errorf("LLM invoker not configured")
	}

	// Build analysis prompt
	prompt := fmt.Sprintf(`Analyze the following prompt content and provide structured feedback.

Content:
%s

Description: %s
Business Line: %s

Respond with a JSON object containing:
- triggers: array of trigger patterns with pattern, examples, and confidence (0-1)
- issues: array of issues with severity (high/medium/low), location, problem, and suggestion
- score: object with overall, clarity, and completeness scores (0-100)

Respond with ONLY the JSON object.`, req.Content, req.Description, req.BizLine)

	// Use configured model, error if not set
	model := s.model
	if model == "" {
		return nil, fmt.Errorf("model not configured: set default_model in LLM config")
	}

	resp, err := s.llmInvoker.Invoke(ctx, prompt, model, 0.3)
	if err != nil {
		return nil, fmt.Errorf("LLM invocation failed: %w", err)
	}

	// Parse JSON response
	var result AnalyzeContentResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		// Return basic result if parsing fails
		return &AnalyzeContentResult{
			Triggers: []TriggerEntry{},
			Issues:   []ContentIssue{},
			Score:    ContentScore{Overall: 50, Clarity: 50, Completeness: 50},
		}, nil
	}

	return &result, nil
}

// ExplainDiff explains the semantic differences between two versions of content.
func (s *SemanticService) ExplainDiff(ctx context.Context, req ExplainDiffRequest) (*ExplainDiffResult, error) {
	if s.llmInvoker == nil {
		return nil, fmt.Errorf("LLM invoker not configured")
	}

	// Build diff explanation prompt
	prompt := fmt.Sprintf(`Explain the semantic differences between two versions of a prompt.

Old Version (%s):
%s

New Version (%s):
%s

Respond with a JSON object containing:
- summary: brief summary of what changed
- changes: array of changes with type, location, description, and significance (high/medium/low)
- impact: assessment of how these changes might affect prompt behavior

Respond with ONLY the JSON object.`, req.OldVersion, req.OldContent, req.NewVersion, req.NewContent)

	// Use configured model, error if not set
	model := s.model
	if model == "" {
		return nil, fmt.Errorf("model not configured: set default_model in LLM config")
	}

	resp, err := s.llmInvoker.Invoke(ctx, prompt, model, 0.3)
	if err != nil {
		return nil, fmt.Errorf("LLM invocation failed: %w", err)
	}

	// Parse JSON response
	var result ExplainDiffResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		// Return basic result if parsing fails
		return &ExplainDiffResult{
			Summary: "Failed to parse LLM response",
			Changes: []SemanticChange{},
			Impact:  "Unknown",
		}, nil
	}

	return &result, nil
}
