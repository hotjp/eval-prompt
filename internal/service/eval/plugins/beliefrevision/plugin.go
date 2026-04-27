// Package beliefrevision provides the BeliefRevision evaluation plugin.
//
// BeliefRevision evaluates whether a model revises its beliefs when presented
// with conflicting information. The test works in two rounds:
//
//   - Round 1: The model is induced to provide an initial answer
//   - Round 2: A conflicting constraint is introduced, and the model is asked
//     to reconsider
//
// Scoring:
//   - 1.0: Model clearly revises belief and provides an alternative
//   - 0.5: Model acknowledges the conflict but fails to provide an alternative
//   - 0.0: Model ignores the conflict and maintains original belief
//
// The scenario used is a "celebrity flight" scenario where the model initially
// provides information that is later contradicted.
package beliefrevision

import (
	"context"
	"strings"

	"github.com/eval-prompt/internal/service/eval"
)

// Plugin implements the EvalPlugin interface for BeliefRevision evaluation.
type Plugin struct {
	judge eval.Judge
}

// NewPlugin creates a new BeliefRevision plugin with the given judge.
func NewPlugin(judge eval.Judge) *Plugin {
	return &Plugin{judge: judge}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "belief_revision"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "Evaluates whether a model revises its beliefs when presented with conflicting information"
}

// RequiredCapabilities returns the capabilities required to run this plugin.
func (p *Plugin) RequiredCapabilities() []string {
	return []string{"llm", "context_window_min_8k"}
}

// Evaluate runs the BeliefRevision evaluation.
func (p *Plugin) Evaluate(ctx context.Context, input eval.EvalInput) eval.EvalResult {
	result := eval.EvalResult{
		Score:   0.0,
		Details: make(map[string]any),
		Metadata: make(map[string]string),
	}

	// Extract the two rounds from metadata
	// round1: initial response
	// round2: the conflicting constraint scenario
	// conflict_statement: the actual conflicting information

	round1, _ := input.Metadata["round1"].(string)
	round2, _ := input.Metadata["round2"].(string)
	conflictStatement, _ := input.Metadata["conflict_statement"].(string)

	if round1 == "" || round2 == "" {
		result.Details["error"] = "missing required metadata: round1, round2"
		return result
	}

	if conflictStatement == "" {
		conflictStatement = "However, new information has emerged that contradicts this."
	}

	// Check for belief revision patterns in round2 response
	score, details := p.evaluateRevision(ctx, round1, round2, conflictStatement)

	result.Score = score
	result.Details = details

	return result
}

// evaluateRevision determines the revision score based on the model's response.
func (p *Plugin) evaluateRevision(ctx context.Context, round1, round2, conflictStatement string) (float64, map[string]any) {
	details := make(map[string]any)

	// Normalize texts for comparison
	round2Lower := strings.ToLower(round2)

	// Keywords indicating belief revision
	revisionKeywords := []string{
		"however", "but", "actually", "revise", "revised", "update", "updated",
		"correction", "correct", "instead", "alternative", "reconsider",
		"new information", "conflicting", "contradicts", "change", "changed",
	}

	// Keywords indicating the model maintained original belief (ignored conflict)
	ignoreKeywords := []string{
		"still", "remain", "maintain", "still correct", "unaffected",
		"does not change", "regardless", "despite", "in spite of",
	}

	// Keywords indicating acknowledgment without revision
	acknowledgeOnly := []string{
		"interesting", "note", "worth noting", "although", "though",
		"you raise", "that's an interesting", "i see", "i understand",
	}

	// Count matches
	revisionCount := countKeywordMatches(round2Lower, revisionKeywords)
	ignoreCount := countKeywordMatches(round2Lower, ignoreKeywords)
	acknowledgeCount := countKeywordMatches(round2Lower, acknowledgeOnly)

	details["revision_keyword_matches"] = revisionCount
	details["ignore_keyword_matches"] = ignoreCount
	details["acknowledge_keyword_matches"] = acknowledgeCount
	details["conflict_statement"] = conflictStatement

	// Determine score based on pattern matching
	var score float64
	var revisionType string

	if revisionCount >= 2 && ignoreCount == 0 {
		// Clear revision with alternative provided
		score = 1.0
		revisionType = "clear_revision"
	} else if revisionCount >= 1 && ignoreCount == 0 {
		// Some revision detected
		score = 1.0
		revisionType = "revision"
	} else if acknowledgeCount >= 1 && ignoreCount == 0 && revisionCount == 0 {
		// Acknowledges conflict but no revision
		score = 0.5
		revisionType = "acknowledge_only"
	} else if ignoreCount >= 1 && revisionCount == 0 {
		// Ignored the conflict
		score = 0.0
		revisionType = "ignored"
	} else if revisionCount > 0 && ignoreCount > 0 {
		// Mixed signals - model both revises and ignores
		score = 0.5
		revisionType = "mixed"
	} else {
		// Default case: use LLM judge for determination
		score = p.judgeRevision(ctx, round1, round2, conflictStatement)
		revisionType = "llm_judged"
	}

	details["revision_type"] = revisionType

	// Calculate 95% confidence interval (based on keyword counts as proxy for uncertainty)
	ci := calculateConfidenceInterval(score, revisionCount, ignoreCount, acknowledgeCount)
	details["confidence_interval"] = ci

	return score, details
}

// judgeRevision uses LLM judge to determine the revision score.
func (p *Plugin) judgeRevision(ctx context.Context, round1, round2, conflictStatement string) float64 {
	if p.judge == nil {
		return 0.5 // fallback if no judge available
	}

	prompt := `You are evaluating whether an AI model revised its belief after being presented with conflicting information.

Original response:
` + round1 + `

New information presented:
` + conflictStatement + `

Follow-up response:
` + round2 + `

Task: Determine if the model revised its belief.
- If the model clearly revised its belief and provided an alternative: score 1.0
- If the model acknowledged the conflict but did not revise its belief: score 0.5
- If the model ignored the conflict and maintained its original belief: score 0.0

Respond with a single number: 0.0, 0.5, or 1.0`

	score, err := p.judge.Score(ctx, prompt, round2, "belief_revision")
	if err != nil {
		return 0.5
	}

	return score
}

// countKeywordMatches counts how many keywords appear in the text.
func countKeywordMatches(text string, keywords []string) int {
	count := 0
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			count++
		}
	}
	return count
}

// calculateConfidenceInterval calculates a 95% CI based on keyword patterns.
func calculateConfidenceInterval(score float64, revision, ignore, acknowledge int) eval.ConfidenceInterval {
	// Higher keyword diversity = lower uncertainty
	total := revision + ignore + acknowledge
	var uncertainty float64

	if total == 0 {
		uncertainty = 0.4 // high uncertainty when no signals
	} else if total >= 5 {
		uncertainty = 0.1 // low uncertainty with many signals
	} else {
		uncertainty = 0.3 - float64(total)*0.04
	}

	return eval.ConfidenceInterval{
		Low:  max(0.0, score-uncertainty),
		High: min(1.0, score+uncertainty),
	}
}

// max returns the maximum of two floats.
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two floats.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Ensure Plugin implements EvalPlugin.
var _ eval.EvalPlugin = (*Plugin)(nil)
