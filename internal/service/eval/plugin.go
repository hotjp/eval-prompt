// Package eval provides the evaluation orchestrator and plugin system.
package eval

import (
	"context"
)

// EvalPlugin is the interface for evaluation plugins.
// Each plugin implements a specific evaluation metric (BERTScore, G-Eval, etc.).
type EvalPlugin interface {
	// Name returns the plugin name.
	Name() string

	// Description returns a human-readable description.
	Description() string

	// RequiredCapabilities returns the capabilities required to run this plugin.
	RequiredCapabilities() []string

	// Evaluate runs the evaluation on the given input.
	Evaluate(ctx context.Context, input EvalInput) EvalResult
}

// EvalInput contains the input data for evaluation.
type EvalInput struct {
	// AssetID is the asset being evaluated.
	AssetID string

	// Candidate is the text/output to evaluate.
	Candidate string

	// Reference is the reference text for comparison (optional for some metrics).
	Reference string

	// TestCase is the original test case (optional).
	TestCase *TestCase

	// Metadata contains additional context (optional).
	Metadata map[string]any
}

// TestCase represents an evaluation test case.
type TestCase struct {
	ID       string
	Input    string
	Expected string
	Prompt   string
}

// EvalResult contains the evaluation result.
type EvalResult struct {
	// Score is the main score (0.0 - 1.0).
	Score float64

	// Dimensions contain per-dimension scores.
	Dimensions []Dimension

	// ConfidenceInterval contains the 95% confidence interval.
	ConfidenceInterval *ConfidenceInterval

	// Details contains plugin-specific details.
	Details map[string]any

	// Metadata contains additional metadata.
	Metadata map[string]string
}

// Dimension represents a named evaluation dimension.
type Dimension struct {
	Name   string
	Score  float64
	Weight float64
}

// ConfidenceInterval represents a statistical confidence interval.
type ConfidenceInterval struct {
	Low  float64
	High float64
}
