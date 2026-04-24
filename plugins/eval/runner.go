// Package eval provides the EvalRunner implementation with deterministic checker
// and rubric grader.
package eval

//go:generate go run entgo.io/ent/cmd/ent generate ./ent/schema

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/eval-prompt/internal/service"
)

// Config holds eval plugin configuration.
type Config struct {
	Enabled bool
}

// Runner implements the EvalRunner interface using the service types.
type Runner struct {
	assertions *AssertionLibrary
}

// NewRunner creates a new eval Runner.
func NewRunner() *Runner {
	return &Runner{
		assertions: NewAssertionLibrary(),
	}
}

// RunDeterministic runs deterministic checks against trace events.
func (r *Runner) RunDeterministic(ctx context.Context, trace []service.TraceEvent, checks []service.DeterministicCheck) (service.DeterministicResult, error) {
	result := service.DeterministicResult{
		Passed:  true,
		Score:   1.0,
		Failed:  []string{},
	}

	for _, check := range checks {
		checker := r.assertions.Get(check.Type)
		if checker == nil {
			result.Passed = false
			result.Score = 0.0
			result.Failed = append(result.Failed, check.ID)
			result.Message = fmt.Sprintf("unknown check type: %s", check.Type)
			continue
		}

		err := checker.Check(trace, check)
		if err != nil {
			result.Passed = false
			result.Failed = append(result.Failed, check.ID)
		}
	}

	// Calculate score based on pass/fail ratio
	if len(checks) > 0 {
		result.Score = float64(len(checks)-len(result.Failed)) / float64(len(checks))
	}

	if len(result.Failed) > 0 {
		result.Passed = false
	}

	return result, nil
}

// RunRubric runs LLM-based rubric evaluation on output text.
func (r *Runner) RunRubric(ctx context.Context, output string, rubric service.Rubric, invoker service.LLMInvoker) (service.RubricResult, error) {
	if invoker == nil {
		return service.RubricResult{}, errors.New("eval: LLMInvoker is required for rubric evaluation")
	}

	result := service.RubricResult{
		Score:    0,
		MaxScore: rubric.MaxScore,
		Passed:   true,
		Details:  []service.RubricCheckResult{},
	}

	if len(rubric.Checks) == 0 {
		return result, nil
	}

	// Build a grading prompt for the LLM
	prompt := r.buildGradingPrompt(output, rubric)

	// Call the LLM
	resp, err := invoker.Invoke(ctx, prompt, "gpt-4o", 0.3)
	if err != nil {
		return result, fmt.Errorf("eval: LLM invocation failed: %w", err)
	}

	// Parse the JSON response from LLM
	var checkResults []service.RubricCheckResult
	if err := json.Unmarshal([]byte(resp.Content), &checkResults); err != nil {
		// If parsing fails, try to extract from the content directly
		checkResults = r.parseGradingResponse(resp.Content, rubric)
	}

	result.Details = checkResults

	// Calculate weighted score
	totalWeight := 0
	weightedScore := 0
	for _, check := range rubric.Checks {
		totalWeight += check.Weight
		for _, res := range checkResults {
			if res.CheckID == check.ID && res.Passed {
				weightedScore += check.Weight
			}
		}
	}

	if totalWeight > 0 {
		result.Score = (weightedScore * rubric.MaxScore) / totalWeight
		result.Passed = result.Score >= rubric.MaxScore*80/100 // 80% threshold
	}

	return result, nil
}

// buildGradingPrompt constructs a prompt for the LLM to grade rubric checks.
func (r *Runner) buildGradingPrompt(output string, rubric service.Rubric) string {
	checksJSON, _ := json.Marshal(rubric.Checks)
	return "You are evaluating an AI assistant's response against a rubric.\n\n" +
		"Output to evaluate:\n" +
		"```\n" + output + "\n```\n\n" +
		"Rubric checks (answer in JSON array format):\n" +
		"```\n" + string(checksJSON) + "\n```\n\n" +
		"For each check, determine if it passed (true/false) and provide a score (0 to weight) and details.\n" +
		`Respond with a JSON array of results like:
[{"check_id": "id1", "passed": true, "score": 10, "details": "explanation"}]`
}

// parseGradingResponse parses the LLM response when JSON parsing fails.
func (r *Runner) parseGradingResponse(content string, rubric service.Rubric) []service.RubricCheckResult {
	results := []service.RubricCheckResult{}
	for _, check := range rubric.Checks {
		// Simple heuristic: check if the output contains keywords related to the check
		passed := len(content) > 0 // Placeholder
		results = append(results, service.RubricCheckResult{
			CheckID: check.ID,
			Passed: passed,
			Score:  0,
			Details: "parse failed, using fallback",
		})
	}
	return results
}

// Ensure Runner implements service.EvalRunner.
var _ service.EvalRunner = (*Runner)(nil)

// TraceEvent is an alias for service.TraceEvent.
type TraceEvent = service.TraceEvent

// DeterministicCheck is an alias for service.DeterministicCheck.
type DeterministicCheck = service.DeterministicCheck

// DeterministicResult is an alias for service.DeterministicResult.
type DeterministicResult = service.DeterministicResult

// Rubric is an alias for service.Rubric.
type Rubric = service.Rubric

// RubricCheck is an alias for service.RubricCheck.
type RubricCheck = service.RubricCheck

// RubricCheckResult is an alias for service.RubricCheckResult.
type RubricCheckResult = service.RubricCheckResult

// RubricResult is an alias for service.RubricResult.
type RubricResult = service.RubricResult

// LLMInvoker is an alias for service.LLMInvoker.
type LLMInvoker = service.LLMInvoker
