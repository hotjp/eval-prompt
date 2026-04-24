package domain

import (
	"fmt"
	"time"
)

// EvalCase represents a test case for evaluating a prompt asset.
type EvalCase struct {
	ID             ID
	AssetID        ID
	Name           string
	Prompt         string
	ShouldTrigger  bool
	ExpectedOutput string
	Rubric         Rubric
	CreatedAt      time.Time
	Version        int64
}

// Rubric defines the evaluation rubric structure.
type Rubric struct {
	MaxScore int           `json:"max_score"`
	Checks   []RubricCheck `json:"checks"`
}

// RubricCheck defines a single check in the rubric.
type RubricCheck struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Weight      int    `json:"weight"`
}

// Validate validates the eval case entity.
func (e *EvalCase) Validate() error {
	if e.ID.IsEmpty() {
		return ErrInvalidID(e.ID.String())
	}
	if e.AssetID.IsEmpty() {
		return NewDomainError(ErrEvalCaseNotFound, "asset_id is required")
	}
	if err := ValidateLength("name", e.Name, 1, 128); err != nil {
		return err
	}
	if e.Prompt == "" {
		return NewDomainError(ErrEvalCaseNotFound, "prompt is required")
	}
	return nil
}

// NewEvalCase creates a new EvalCase with the given parameters.
func NewEvalCase(assetID ID, name, prompt string, shouldTrigger bool, expectedOutput string, rubric Rubric) *EvalCase {
	return &EvalCase{
		ID:             NewAutoID(),
		AssetID:        assetID,
		Name:           name,
		Prompt:         prompt,
		ShouldTrigger:  shouldTrigger,
		ExpectedOutput: expectedOutput,
		Rubric:         rubric,
		CreatedAt:      time.Now(),
		Version:        0,
	}
}

// NewEvalCaseWithID creates a new EvalCase with a specific ID.
func NewEvalCaseWithID(id, assetID ID, name, prompt string, shouldTrigger bool, expectedOutput string, rubric Rubric) *EvalCase {
	return &EvalCase{
		ID:             id,
		AssetID:        assetID,
		Name:           name,
		Prompt:         prompt,
		ShouldTrigger:  shouldTrigger,
		ExpectedOutput: expectedOutput,
		Rubric:         rubric,
		CreatedAt:      time.Now(),
		Version:        0,
	}
}

// TotalRubricWeight calculates the total weight of all rubric checks.
func (e *EvalCase) TotalRubricWeight() int {
	total := 0
	for _, check := range e.Rubric.Checks {
		total += check.Weight
	}
	return total
}

// RubricWeightMap returns a map of check ID to weight for quick lookup.
func (e *EvalCase) RubricWeightMap() map[string]int {
	weightMap := make(map[string]int)
	for _, check := range e.Rubric.Checks {
		weightMap[check.ID] = check.Weight
	}
	return weightMap
}

// EvalCaseSummary is a lightweight representation of an eval case for listing.
type EvalCaseSummary struct {
	ID            ID
	AssetID       ID
	Name          string
	ShouldTrigger bool
	CreatedAt     time.Time
}

// CalculateScore calculates the weighted score based on rubric check results.
func CalculateScore(rubric Rubric, results []RubricCheckResult) int {
	if len(results) == 0 {
		return 0
	}

	totalWeight := 0
	weightedScore := 0

	for _, check := range rubric.Checks {
		totalWeight += check.Weight
		for _, result := range results {
			if result.CheckID == check.ID {
				if result.Passed {
					weightedScore += check.Weight
				}
				break
			}
		}
	}

	if totalWeight == 0 {
		return 0
	}

	// Scale to max score
	return (weightedScore * rubric.MaxScore) / totalWeight
}

// RubricCheckResult represents the result of a single rubric check.
type RubricCheckResult struct {
	CheckID string `json:"check_id"`
	Passed  bool   `json:"passed"`
	Score   int    `json:"score"`
	Details string `json:"details,omitempty"`
}

// NewRubricCheckResult creates a new rubric check result.
func NewRubricCheckResult(checkID string, passed bool, score int, details string) RubricCheckResult {
	return RubricCheckResult{
		CheckID: checkID,
		Passed:  passed,
		Score:   score,
		Details: details,
	}
}

// ValidateResults validates that the results match the rubric checks.
func ValidateResults(rubric Rubric, results []RubricCheckResult) error {
	checkIDs := make(map[string]bool)
	for _, check := range rubric.Checks {
		checkIDs[check.ID] = true
	}

	for _, result := range results {
		if !checkIDs[result.CheckID] {
			return fmt.Errorf("unknown check_id in results: %s", result.CheckID)
		}
	}
	return nil
}
