// Package service implements L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"
	"fmt"
	"strings"
)

// MatchedPrompt represents a matched prompt with relevance score.
type MatchedPrompt struct {
	AssetID     string            `json:"asset_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Content     string            `json:"content"`
	Variables   []string          `json:"variables,omitempty"`
	Relevance   float64           `json:"relevance"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// TriggerService handles prompt matching and variable injection.
type TriggerService struct {
	indexer    AssetIndexer
	gitBridger GitBridger
}

// NewTriggerService creates a new TriggerService.
func NewTriggerService(indexer AssetIndexer, gitBridger GitBridger) *TriggerService {
	return &TriggerService{
		indexer:    indexer,
		gitBridger: gitBridger,
	}
}

// Ensure TriggerService implements TriggerServicer.
var _ TriggerServicer = (*TriggerService)(nil)

// TriggerServicer is the interface for trigger operations.
type TriggerServicer interface {
	// MatchTrigger matches input against available prompts.
	MatchTrigger(ctx context.Context, input string, top int) ([]*MatchedPrompt, error)

	// ValidateAntiPatterns validates that the prompt doesn't contain anti-patterns.
	ValidateAntiPatterns(ctx context.Context, prompt string) error

	// InjectVariables injects variables into a prompt template.
	InjectVariables(ctx context.Context, prompt string, vars map[string]string) (string, error)
}

// DefaultAntiPatterns are the default anti-patterns to reject.
var DefaultAntiPatterns = []string{
	"generate code",
	"write new feature",
	"refactor entire",
	"delete all",
	"drop table",
	"rm -rf",
}

// MatchTrigger matches input against available prompts.
func (s *TriggerService) MatchTrigger(ctx context.Context, input string, top int) ([]*MatchedPrompt, error) {
	if s.indexer == nil {
		return nil, fmt.Errorf("indexer not configured")
	}

	if top <= 0 {
		top = 5
	}

	// Search for matching assets
	results, err := s.indexer.Search(ctx, input, SearchFilters{})
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	// Convert to MatchedPrompt (limited to top results)
	var matches []*MatchedPrompt
	for i := 0; i < len(results) && i < top; i++ {
		r := results[i]
		matches = append(matches, &MatchedPrompt{
			AssetID:     r.ID,
			Name:        r.Name,
			Description: r.Description,
			Relevance:   calculateRelevance(input, r.Name, r.Description),
		})
	}

	return matches, nil
}

// calculateRelevance calculates a simple relevance score based on keyword matching.
func calculateRelevance(input, name, description string) float64 {
	input = strings.ToLower(input)
	name = strings.ToLower(name)
	description = strings.ToLower(description)

	// Count matching words
	score := 0.0
	inputWords := strings.Fields(input)

	for _, word := range inputWords {
		if len(word) < 3 {
			continue
		}
		if strings.Contains(name, word) {
			score += 2.0
		}
		if strings.Contains(description, word) {
			score += 1.0
		}
	}

	// Normalize to 0-1 range
	maxScore := float64(len(inputWords) * 3)
	if maxScore == 0 {
		return 0
	}
	return score / maxScore
}

// ValidateAntiPatterns validates that the prompt doesn't contain anti-patterns.
func (s *TriggerService) ValidateAntiPatterns(ctx context.Context, prompt string) error {
	promptLower := strings.ToLower(prompt)

	for _, pattern := range DefaultAntiPatterns {
		if strings.Contains(promptLower, pattern) {
			return fmt.Errorf("prompt contains anti-pattern: %s", pattern)
		}
	}

	return nil
}

// InjectVariables injects variables into a prompt template.
func (s *TriggerService) InjectVariables(ctx context.Context, prompt string, vars map[string]string) (string, error) {
	result := prompt

	for key, value := range vars {
		// Support {{var}} and ${var} syntax
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)

		placeholder = fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result, nil
}
