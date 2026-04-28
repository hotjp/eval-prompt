// Package service implements L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"
	"encoding/json"
	"fmt"
)

// SemanticService provides LLM-based semantic analysis capabilities.
type SemanticService struct {
	llmInvoker LLMInvoker
	model      string
}

// NewSemanticService creates a new SemanticService.
func NewSemanticService(invoker LLMInvoker, model string) *SemanticService {
	return &SemanticService{
		llmInvoker: invoker,
		model:      model,
	}
}

// AnalyzeContent performs semantic analysis on prompt content.
func (s *SemanticService) AnalyzeContent(ctx context.Context, req AnalyzeContentRequest) (*AnalyzeContentResult, error) {
	prompt := fmt.Sprintf("Analyze the following prompt content and provide triggers, issues, and scores:\n\nContent: %s\nDescription: %s\nAssetType: %s",
		req.Content, req.Description, req.AssetType)

	resp, err := s.llmInvoker.Invoke(ctx, prompt, s.model, 0.3)
	if err != nil {
		return nil, fmt.Errorf("semantic analysis failed: %w", err)
	}

	// Parse the response as JSON
	var result AnalyzeContentResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		// If not valid JSON, return a basic result with the raw response
		result = AnalyzeContentResult{
			Score: ContentScore{Overall: 0.5},
		}
	}

	return &result, nil
}

// ExplainDiff generates a semantic explanation of differences between two prompt versions.
func (s *SemanticService) ExplainDiff(ctx context.Context, req ExplainDiffRequest) (*ExplainDiffResult, error) {
	prompt := fmt.Sprintf("Explain the semantic differences between these two versions of a prompt:\n\nVersion 1 (%s):\n%s\n\nVersion 2 (%s):\n%s",
		req.OldVersion, req.OldContent, req.NewVersion, req.NewContent)

	resp, err := s.llmInvoker.Invoke(ctx, prompt, s.model, 0.3)
	if err != nil {
		return nil, fmt.Errorf("diff explanation failed: %w", err)
	}

	var result ExplainDiffResult
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		result = ExplainDiffResult{
			Summary: resp.Content,
		}
	}

	return &result, nil
}
