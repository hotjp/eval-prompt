// Package constraint provides the Constraint evaluation plugin.
//
// Constraint evaluates whether a model's output satisfies all specified
// constraint conditions. Constraints are provided in the EvalInput.Metadata
// and can include:
//
//   - Keyword requirements: specific words/phrases that must appear
//   - Format requirements: regex patterns the output must match
//   - Negation requirements: words/phrases that must NOT appear
//   - Length constraints: min/max length requirements
//
// The plugin uses a combination of regex/keyword matching and LLM-based
// validation to verify constraint satisfaction.
package constraint

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/eval-prompt/internal/service/eval"
)

// Plugin implements the EvalPlugin interface for Constraint evaluation.
type Plugin struct {
	judge eval.Judge
}

// NewPlugin creates a new Constraint plugin with the given judge.
func NewPlugin(judge eval.Judge) *Plugin {
	return &Plugin{judge: judge}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "constraint"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "Evaluates whether a model's output satisfies all specified constraint conditions"
}

// RequiredCapabilities returns the capabilities required to run this plugin.
func (p *Plugin) RequiredCapabilities() []string {
	return []string{"llm"}
}

// Evaluate runs the Constraint evaluation.
func (p *Plugin) Evaluate(ctx context.Context, input eval.EvalInput) eval.EvalResult {
	result := eval.EvalResult{
		Score:   0.0,
		Dimensions: []eval.Dimension{},
		Details: make(map[string]any),
		Metadata: make(map[string]string),
	}

	constraints := p.extractConstraints(input.Metadata)
	candidate := input.Candidate

	if len(constraints) == 0 {
		result.Details["error"] = "no constraints found in metadata"
		result.Metadata["status"] = "no_constraints"
		return result
	}

	result.Details["constraint_count"] = len(constraints)

	// Evaluate each constraint
	constraintResults := make([]constraintResult, 0, len(constraints))
	totalScore := 0.0

	for _, constraint := range constraints {
		cr := p.evaluateConstraint(ctx, candidate, constraint)
		constraintResults = append(constraintResults, cr)
		totalScore += cr.Score
	}

	result.Details["constraint_results"] = constraintResults

	// Calculate overall score (average of all constraint scores)
	if len(constraints) > 0 {
		result.Score = totalScore / float64(len(constraints))
	}

	// Calculate dimension scores for each constraint type
	result.Dimensions = p.calculateDimensions(constraintResults)

	// Calculate confidence interval
	ci := p.calculateConfidenceInterval(constraintResults)
	result.ConfidenceInterval = &ci

	result.Metadata["constraints_evaluated"] = string(rune(len(constraints)))
	result.Metadata["all_satisfied"] = boolToString(result.Score >= 1.0)

	return result
}

// constraintResult holds the result of evaluating a single constraint.
type constraintResult struct {
	Constraint string  `json:"constraint"`
	Type       string  `json:"type"`
	Satisfied  bool    `json:"satisfied"`
	Score      float64 `json:"score"`
	Details    string  `json:"details"`
}

// constraint represents a single constraint specification.
type constraint struct {
	Type     string   // "keyword", "regex", "negation", "length_min", "length_max"
	Pattern  string   // the pattern to match
	Required bool     // whether this constraint must be satisfied
	Weight   float64  // weight for scoring (default 1.0)
}

// extractConstraints extracts constraint specifications from metadata.
func (p *Plugin) extractConstraints(metadata map[string]any) []constraint {
	constraints := []constraint{}

	// Extract from "constraints" field (slice of constraint maps)
	if constraintsRaw, ok := metadata["constraints"].([]any); ok {
		for _, c := range constraintsRaw {
			if cm, ok := c.(map[string]any); ok {
				constraints = append(constraints, p.parseConstraint(cm))
			}
		}
	}

	// Extract from "keywords" field (simple string array)
	if keywords, ok := metadata["keywords"].([]any); ok {
		for _, kw := range keywords {
			if ks, ok := kw.(string); ok {
				constraints = append(constraints, constraint{
					Type:    "keyword",
					Pattern: ks,
					Required: true,
					Weight:  1.0,
				})
			}
		}
	}

	// Extract from "must_contain" field
	if mustContain, ok := metadata["must_contain"].([]any); ok {
		for _, mc := range mustContain {
			if mcs, ok := mc.(string); ok {
				constraints = append(constraints, constraint{
					Type:    "keyword",
					Pattern: mcs,
					Required: true,
					Weight:  1.0,
				})
			}
		}
	}

	// Extract from "must_not_contain" field
	if mustNotContain, ok := metadata["must_not_contain"].([]any); ok {
		for _, mnc := range mustNotContain {
			if mncs, ok := mnc.(string); ok {
				constraints = append(constraints, constraint{
					Type:    "negation",
					Pattern: mncs,
					Required: true,
					Weight:  1.0,
				})
			}
		}
	}

	// Extract from "regex_patterns" field
	if patterns, ok := metadata["regex_patterns"].([]any); ok {
		for _, pat := range patterns {
			if ps, ok := pat.(string); ok {
				constraints = append(constraints, constraint{
					Type:    "regex",
					Pattern: ps,
					Required: true,
					Weight:  1.0,
				})
			}
		}
	}

	// Extract length constraints
	if minLen, ok := metadata["min_length"].(float64); ok {
		constraints = append(constraints, constraint{
			Type:    "length_min",
			Pattern: strconv.Itoa(int(minLen)),
			Required: false,
			Weight:  0.5,
		})
	}

	if maxLen, ok := metadata["max_length"].(float64); ok {
		constraints = append(constraints, constraint{
			Type:    "length_max",
			Pattern: strconv.Itoa(int(maxLen)),
			Required: false,
			Weight:  0.5,
		})
	}

	return constraints
}

// parseConstraint parses a constraint from a map.
func (p *Plugin) parseConstraint(m map[string]any) constraint {
	c := constraint{
		Required: true,
		Weight:   1.0,
	}

	if t, ok := m["type"].(string); ok {
		c.Type = t
	}
	if p, ok := m["pattern"].(string); ok {
		c.Pattern = p
	}
	if r, ok := m["required"].(bool); ok {
		c.Required = r
	}
	if w, ok := m["weight"].(float64); ok {
		c.Weight = w
	}

	return c
}

// evaluateConstraint evaluates a single constraint against the candidate text.
func (p *Plugin) evaluateConstraint(ctx context.Context, candidate string, c constraint) constraintResult {
	cr := constraintResult{
		Constraint: c.Pattern,
		Type:       c.Type,
		Satisfied:  false,
		Score:      0.0,
	}

	switch c.Type {
	case "keyword":
		cr.Satisfied = strings.Contains(strings.ToLower(candidate), strings.ToLower(c.Pattern))
		cr.Details = "keyword matching"

	case "regex":
		matched, err := regexp.MatchString(c.Pattern, candidate)
		if err == nil {
			cr.Satisfied = matched
		}
		cr.Details = "regex matching"

	case "negation":
		cr.Satisfied = !strings.Contains(strings.ToLower(candidate), strings.ToLower(c.Pattern))
		cr.Details = "negation check (must NOT contain)"

	case "length_min":
		minLen := len(c.Pattern)
		cr.Satisfied = len(candidate) >= minLen
		cr.Details = "minimum length check"

	case "length_max":
		maxLen := len(c.Pattern)
		cr.Satisfied = len(candidate) <= maxLen
		cr.Details = "maximum length check"

	default:
		// For unknown types, use LLM judge for semantic validation
		if p.judge != nil {
			score, err := p.judge.Score(ctx, c.Type+": "+c.Pattern, candidate, "constraint_satisfaction")
			if err == nil {
				cr.Satisfied = score >= 0.8
				cr.Score = score
				cr.Details = "llm_judged"
				return cr
			}
		}
		cr.Details = "unknown constraint type: " + c.Type
	}

	if c.Required && !cr.Satisfied {
		cr.Score = 0.0
	} else if cr.Satisfied {
		cr.Score = c.Weight
	} else {
		cr.Score = c.Weight * 0.5 // partial credit for non-required
	}

	return cr
}

// calculateDimensions calculates dimension scores for each constraint type.
func (p *Plugin) calculateDimensions(results []constraintResult) []eval.Dimension {
	dimensions := make(map[string]dimensionAccum)

	for _, r := range results {
		if _, ok := dimensions[r.Type]; !ok {
			dimensions[r.Type] = dimensionAccum{count: 0, totalScore: 0}
		}
		da := dimensions[r.Type]
		da.count++
		da.totalScore += r.Score
		dimensions[r.Type] = da
	}

	var dims []eval.Dimension
	for t, da := range dimensions {
		score := 0.0
		if da.count > 0 {
			score = da.totalScore / float64(da.count)
		}
		dims = append(dims, eval.Dimension{
			Name:   t,
			Score:  score,
			Weight: 1.0,
		})
	}

	return dims
}

// dimensionAccum accumulates scores for a dimension.
type dimensionAccum struct {
	count      int
	totalScore float64
}

// calculateConfidenceInterval calculates a 95% confidence interval.
func (p *Plugin) calculateConfidenceInterval(results []constraintResult) eval.ConfidenceInterval {
	if len(results) == 0 {
		return eval.ConfidenceInterval{Low: 0.0, High: 1.0}
	}

	// Calculate variance based on satisfaction rate
	satisfiedCount := 0
	for _, r := range results {
		if r.Satisfied {
			satisfiedCount++
		}
	}

	satisfactionRate := float64(satisfiedCount) / float64(len(results))
	uncertainty := satisfactionRate * (1 - satisfactionRate) / float64(len(results))

	// 95% CI using standard error
	se := 1.96 * uncertainty
	low := max(0.0, satisfactionRate-se)
	high := min(1.0, satisfactionRate+se)

	return eval.ConfidenceInterval{Low: low, High: high}
}

// boolToString converts a boolean to a string.
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
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
