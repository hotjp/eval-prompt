package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalCase_Validate(t *testing.T) {
	validCase := &EvalCase{
		ID:      ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		AssetID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"},
		Name:    "Test Case",
		Prompt:  "Test prompt",
	}

	tests := []struct {
		name    string
		ec      *EvalCase
		wantErr bool
	}{
		{
			name:    "valid eval case",
			ec:      validCase,
			wantErr: false,
		},
		{
			name: "empty ID",
			ec: &EvalCase{
				ID:      ID{},
				AssetID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Name:    "Test",
				Prompt:  "prompt",
			},
			wantErr: true,
		},
		{
			name: "empty AssetID",
			ec: &EvalCase{
				ID:      ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				AssetID: ID{},
				Name:    "Test",
				Prompt:  "prompt",
			},
			wantErr: true,
		},
		{
			name: "empty Name",
			ec: &EvalCase{
				ID:      ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				AssetID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Name:    "",
				Prompt:  "prompt",
			},
			wantErr: true,
		},
		{
			name: "empty Prompt",
			ec: &EvalCase{
				ID:      ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				AssetID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Name:    "Test",
				Prompt:  "",
			},
			wantErr: true,
		},
		{
			name: "name too long",
			ec: &EvalCase{
				ID:      ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				AssetID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Name:    string(make([]byte, 129)),
				Prompt:  "prompt",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ec.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEvalCase_TotalRubricWeight(t *testing.T) {
	ec := &EvalCase{
		Rubric: Rubric{
			MaxScore: 100,
			Checks: []RubricCheck{
				{ID: "c1", Weight: 10},
				{ID: "c2", Weight: 20},
				{ID: "c3", Weight: 30},
			},
		},
	}

	require.Equal(t, 60, ec.TotalRubricWeight())
}

func TestEvalCase_RubricWeightMap(t *testing.T) {
	ec := &EvalCase{
		Rubric: Rubric{
			Checks: []RubricCheck{
				{ID: "c1", Weight: 10},
				{ID: "c2", Weight: 20},
			},
		},
	}

	m := ec.RubricWeightMap()
	require.Equal(t, 10, m["c1"])
	require.Equal(t, 20, m["c2"])
	require.Equal(t, 0, m["c3"])
}

func TestNewEvalCase(t *testing.T) {
	assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	rubric := Rubric{MaxScore: 100, Checks: []RubricCheck{{ID: "c1", Weight: 10}}}

	ec := NewEvalCase(assetID, "Test", "prompt", true, "output", rubric)

	require.NotEmpty(t, ec.ID.String())
	require.Equal(t, assetID, ec.AssetID)
	require.Equal(t, "Test", ec.Name)
	require.Equal(t, "prompt", ec.Prompt)
	require.True(t, ec.ShouldTrigger)
	require.Equal(t, "output", ec.ExpectedOutput)
	require.Equal(t, int64(0), ec.Version)
}

func TestCalculateScore(t *testing.T) {
	rubric := Rubric{
		MaxScore: 100,
		Checks: []RubricCheck{
			{ID: "c1", Weight: 50},
			{ID: "c2", Weight: 50},
		},
	}

	t.Run("all passed", func(t *testing.T) {
		results := []RubricCheckResult{
			{CheckID: "c1", Passed: true},
			{CheckID: "c2", Passed: true},
		}
		score := CalculateScore(rubric, results)
		require.Equal(t, 100, score)
	})

	t.Run("none passed", func(t *testing.T) {
		results := []RubricCheckResult{
			{CheckID: "c1", Passed: false},
			{CheckID: "c2", Passed: false},
		}
		score := CalculateScore(rubric, results)
		require.Equal(t, 0, score)
	})

	t.Run("half passed", func(t *testing.T) {
		results := []RubricCheckResult{
			{CheckID: "c1", Passed: true},
			{CheckID: "c2", Passed: false},
		}
		score := CalculateScore(rubric, results)
		require.Equal(t, 50, score)
	})

	t.Run("empty results", func(t *testing.T) {
		score := CalculateScore(rubric, []RubricCheckResult{})
		require.Equal(t, 0, score)
	})
}

func TestValidateResults(t *testing.T) {
	rubric := Rubric{
		Checks: []RubricCheck{
			{ID: "c1", Weight: 10},
			{ID: "c2", Weight: 20},
		},
	}

	t.Run("valid results", func(t *testing.T) {
		results := []RubricCheckResult{
			{CheckID: "c1", Passed: true},
			{CheckID: "c2", Passed: false},
		}
		err := ValidateResults(rubric, results)
		require.NoError(t, err)
	})

	t.Run("unknown check_id", func(t *testing.T) {
		results := []RubricCheckResult{
			{CheckID: "c1", Passed: true},
			{CheckID: "unknown", Passed: false},
		}
		err := ValidateResults(rubric, results)
		require.Error(t, err)
	})
}

func TestNewEvalCaseWithID(t *testing.T) {
	specificID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"}
	rubric := Rubric{MaxScore: 100, Checks: []RubricCheck{{ID: "c1", Weight: 10}}}

	ec := NewEvalCaseWithID(specificID, assetID, "Test", "prompt", true, "output", rubric)

	require.Equal(t, specificID, ec.ID)
	require.Equal(t, assetID, ec.AssetID)
	require.Equal(t, "Test", ec.Name)
	require.Equal(t, "prompt", ec.Prompt)
	require.True(t, ec.ShouldTrigger)
	require.Equal(t, "output", ec.ExpectedOutput)
	require.Equal(t, int64(0), ec.Version)
}

func TestCalculateScore_EdgeCases(t *testing.T) {
	t.Run("zero total weight returns zero", func(t *testing.T) {
		rubric := Rubric{
			MaxScore: 100,
			Checks:   []RubricCheck{},
		}
		results := []RubricCheckResult{}
		score := CalculateScore(rubric, results)
		require.Equal(t, 0, score)
	})

	t.Run("zero weight checks still scale correctly", func(t *testing.T) {
		rubric := Rubric{
			MaxScore: 100,
			Checks: []RubricCheck{
				{ID: "c1", Weight: 0},
				{ID: "c2", Weight: 50},
			},
		}
		results := []RubricCheckResult{
			{CheckID: "c1", Passed: true},
			{CheckID: "c2", Passed: true},
		}
		score := CalculateScore(rubric, results)
		require.Equal(t, 100, score)
	})
}

func TestNewRubricCheckResult(t *testing.T) {
	result := NewRubricCheckResult("check1", true, 10, "details")

	require.Equal(t, "check1", result.CheckID)
	require.True(t, result.Passed)
	require.Equal(t, 10, result.Score)
	require.Equal(t, "details", result.Details)
}

func TestNewRubricCheckResult_EmptyDetails(t *testing.T) {
	result := NewRubricCheckResult("check1", false, 0, "")
	require.Equal(t, "", result.Details)
}
