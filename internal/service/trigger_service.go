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
	indexer          AssetIndexer
	gitBridger      GitBridger
	semanticAnalyzer SemanticAnalyzer // nil if not configured
	model            string            // configured default model
}

// NewTriggerService creates a new TriggerService.
func NewTriggerService(indexer AssetIndexer, gitBridger GitBridger) *TriggerService {
	return &TriggerService{
		indexer:    indexer,
		gitBridger: gitBridger,
	}
}

// WithSemanticAnalyzer sets the semantic analyzer and model for the TriggerService.
func (s *TriggerService) WithSemanticAnalyzer(sa SemanticAnalyzer, model string) *TriggerService {
	s.semanticAnalyzer = sa
	s.model = model
	return s
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
// Flow:
//  1. Try frontmatter Triggers regex matching (O(N)遍历所有asset)
//  2. If results < top: use keyword Search as fallback
//  3. If SemanticAnalyzer is configured, use it for enhanced matching (future)
func (s *TriggerService) MatchTrigger(ctx context.Context, input string, top int) ([]*MatchedPrompt, error) {
	if s.indexer == nil {
		return nil, fmt.Errorf("indexer not configured")
	}

	if top <= 0 {
		top = 5
	}

	var matches []*MatchedPrompt

	// Step 1: Try frontmatter Triggers regex matching
	triggerMatches := s.matchByTriggers(ctx, input)
	matches = append(matches, triggerMatches...)

	// Step 2: Fallback to keyword Search if needed
	if len(matches) < top {
		searchResults, err := s.indexer.Search(ctx, input, SearchFilters{})
		if err != nil {
			return nil, fmt.Errorf("search: %w", err)
		}

		for i := 0; i < len(searchResults) && len(matches) < top; i++ {
			r := searchResults[i]
			// Skip if already matched by triggers
			if containsAssetID(matches, r.ID) {
				continue
			}
			matches = append(matches, &MatchedPrompt{
				AssetID:     r.ID,
				Name:        r.Name,
				Description: r.Description,
				Relevance:   calculateRelevance(input, r.Name, r.Description),
			})
		}
	}

	// Step 3: SemanticAnalyzer (future - not implemented yet)
	// if s.semanticAnalyzer != nil && len(matches) < top {
	//     // Use semantic analysis for enhanced matching
	// }

	// Limit to top results
	if len(matches) > top {
		matches = matches[:top]
	}

	return matches, nil
}

// matchByTriggers matches input against frontmatter trigger patterns.
func (s *TriggerService) matchByTriggers(ctx context.Context, input string) []*MatchedPrompt {
	var matches []*MatchedPrompt

	// For MVP: match by name/description using simple keyword matching
	// Future: read frontmatter YAML to get TriggerEntry patterns
	inputLower := strings.ToLower(input)

	results, err := s.indexer.Search(ctx, input, SearchFilters{})
	if err != nil {
		return matches
	}

	for _, r := range results {
		// Simple regex-like matching on name and description
		nameLower := strings.ToLower(r.Name)
		descLower := strings.ToLower(r.Description)

		// Check if input keywords appear in name or description
		inputWords := strings.Fields(inputLower)
		relevance := 0.0
		for _, word := range inputWords {
			if len(word) < 2 {
				continue
			}
			// Simple substring match as a stand-in for regex pattern matching
			if strings.Contains(nameLower, word) {
				relevance += 2.0
			}
			if strings.Contains(descLower, word) {
				relevance += 1.0
			}
		}

		// Normalize relevance
		if relevance > 0 {
			maxScore := float64(len(inputWords) * 3)
			if maxScore > 0 {
				relevance = relevance / maxScore
			}
			matches = append(matches, &MatchedPrompt{
				AssetID:     r.ID,
				Name:        r.Name,
				Description: r.Description,
				Relevance:   relevance,
			})
		}
	}

	return matches
}

// containsAssetID checks if a match already contains the given asset ID.
func containsAssetID(matches []*MatchedPrompt, id string) bool {
	for _, m := range matches {
		if m.AssetID == id {
			return true
		}
	}
	return false
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
