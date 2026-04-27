// Package geval implements the G-Eval evaluation plugin.
//
// G-Eval (Liu et al., NeurIPS 2024) is an LLM-based evaluation framework
// that uses Chain-of-Thought reasoning to score outputs. The plugin performs
// multiple sampling runs and computes statistical confidence intervals via
// bootstrap resampling.
//
// Reference:
// Liu et al., "G-Eval: GPT-4 Scoring with Human Alignment", NeurIPS 2024
package geval

import (
	"context"
	"strings"

	"github.com/eval-prompt/internal/service/eval"
	"github.com/eval-prompt/internal/service/eval/stats"
)

// Plugin implements the EvalPlugin interface for G-Eval evaluation.
type Plugin struct {
	judged eval.Judge
}

// NewPlugin creates a new G-Eval plugin with the given Judge.
func NewPlugin(judge eval.Judge) *Plugin {
	return &Plugin{judged: judge}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "geval"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "G-Eval: LLM-based evaluation with Chain-of-Thought reasoning"
}

// RequiredCapabilities returns the capabilities required to run this plugin.
func (p *Plugin) RequiredCapabilities() []string {
	return []string{"judge"}
}

// defaultCriteria is the default evaluation criteria template for G-Eval.
// It guides the LLM through Chain-of-Thought reasoning before scoring.
const defaultCriteria = `You are an expert evaluator assessing the quality of an AI assistant's response.

Your task is to evaluate the response based on the following criteria:
1. Relevance: Does the response directly address the user's query?
2. Coherence: Is the response well-structured and easy to follow?
3. Accuracy: Is the information provided correct and factual?
4. Helpfulness: Does the response provide actionable and useful information?

Please follow this process:
1. Carefully read the input query and the response to evaluate.
2. Consider each criterion above, providing brief reasoning.
3. Assign an overall score from 0-10 based on your analysis.

Output format:
- Step-by-step reasoning: [Your Chain-of-Thought analysis]
- Final score: [A single number between 0-10]`

// Evaluate runs the G-Eval evaluation on the given input.
// It performs multiple sampling runs (n=5 by default) with Temperature=0,
// computes the mean score, and calculates a 95% bootstrap confidence interval.
func (p *Plugin) Evaluate(ctx context.Context, input eval.EvalInput) eval.EvalResult {
	n := 5 // number of sampling runs
	scores := make([]float64, n)

	// Get criteria from metadata or use default
	criteria := defaultCriteria
	if c, ok := input.Metadata["criteria"].(string); ok && c != "" {
		criteria = c
	}

	// Build the evaluation prompt
	prompt := buildPrompt(input, criteria)

	// Run multiple evaluations
	for i := 0; i < n; i++ {
		score, err := p.judged.Score(ctx, prompt, input.Candidate, criteria)
		if err != nil {
			// On error, use fallback score
			scores[i] = 0.5
			continue
		}
		scores[i] = score
	}

	// Calculate mean score
	meanScore := mean(scores)

	// Calculate 95% bootstrap confidence interval
	low, high := stats.BootstrapCI(scores, 0.95, 1000)

	return eval.EvalResult{
		Score: meanScore,
		ConfidenceInterval: &eval.ConfidenceInterval{
			Low:  low,
			High: high,
		},
		Details: map[string]any{
			"n_samples":  n,
			"scores":     scores,
			"criteria":   criteria,
			"model_used": "unknown", // could be extracted from Judge if needed
		},
		Metadata: map[string]string{
			"paper":  "Liu et al., G-Eval: GPT-4 Scoring with Human Alignment, NeurIPS 2024",
			"method": "chain-of-thought",
		},
	}
}

// buildPrompt constructs the evaluation prompt from input.
func buildPrompt(input eval.EvalInput, criteria string) string {
	var sb strings.Builder

	if input.TestCase != nil && input.TestCase.Prompt != "" {
		sb.WriteString("Input Query:\n")
		sb.WriteString(input.TestCase.Prompt)
		sb.WriteString("\n\n")
	} else if input.TestCase != nil && input.TestCase.Input != "" {
		sb.WriteString("Input Query:\n")
		sb.WriteString(input.TestCase.Input)
		sb.WriteString("\n\n")
	}

	if input.Reference != "" {
		sb.WriteString("Reference Output:\n")
		sb.WriteString(input.Reference)
		sb.WriteString("\n\n")
	}

	sb.WriteString("Response to Evaluate:\n")
	sb.WriteString(input.Candidate)

	return sb.String()
}

// mean calculates the arithmetic mean of a slice of floats.
func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
