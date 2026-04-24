package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalRunStatus(t *testing.T) {
	tests := []struct {
		status   EvalRunStatus
		wantStr  string
	}{
		{EvalRunStatusPending, "pending"},
		{EvalRunStatusRunning, "running"},
		{EvalRunStatusPassed, "passed"},
		{EvalRunStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.wantStr, func(t *testing.T) {
			require.Equal(t, tt.wantStr, string(tt.status))
		})
	}
}

func TestEvalRun_Validate(t *testing.T) {
	validRun := &EvalRun{
		ID:         ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		EvalCaseID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"},
		SnapshotID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FCC"},
	}

	tests := []struct {
		name    string
		run     *EvalRun
		wantErr bool
	}{
		{
			name:    "valid eval run",
			run:     validRun,
			wantErr: false,
		},
		{
			name: "empty ID",
			run: &EvalRun{
				ID:         ID{},
				EvalCaseID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				SnapshotID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FCC"},
			},
			wantErr: true,
		},
		{
			name: "empty EvalCaseID",
			run: &EvalRun{
				ID:         ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				EvalCaseID: ID{},
				SnapshotID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FCC"},
			},
			wantErr: true,
		},
		{
			name: "empty SnapshotID",
			run: &EvalRun{
				ID:         ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				EvalCaseID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				SnapshotID: ID{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEvalRun_IsPassed(t *testing.T) {
	run := &EvalRun{Status: EvalRunStatusPassed}
	require.True(t, run.IsPassed())
	require.False(t, run.IsFailed())
}

func TestEvalRun_IsFailed(t *testing.T) {
	run := &EvalRun{Status: EvalRunStatusFailed}
	require.True(t, run.IsFailed())
	require.False(t, run.IsPassed())
}

func TestEvalRun_TotalScore(t *testing.T) {
	run := &EvalRun{RubricScore: 85}
	require.Equal(t, 85, run.TotalScore())
}

func TestNewEvalRun(t *testing.T) {
	evalCaseID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	snapshotID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"}

	run := NewEvalRun(evalCaseID, snapshotID)

	require.NotEmpty(t, run.ID.String())
	require.Equal(t, evalCaseID, run.EvalCaseID)
	require.Equal(t, snapshotID, run.SnapshotID)
	require.Equal(t, EvalRunStatusPending, run.Status)
}

func TestEvalRun_Start(t *testing.T) {
	run := &EvalRun{Status: EvalRunStatusPending}
	run.Start()
	require.Equal(t, EvalRunStatusRunning, run.Status)
}

func TestEvalRun_Complete(t *testing.T) {
	run := &EvalRun{Status: EvalRunStatusPending}

	t.Run("passed", func(t *testing.T) {
		run.Complete(0.95, 85, true)
		require.Equal(t, EvalRunStatusPassed, run.Status)
		require.Equal(t, 0.95, run.DeterministicScore)
		require.Equal(t, 85, run.RubricScore)
	})

	t.Run("failed", func(t *testing.T) {
		run2 := &EvalRun{Status: EvalRunStatusPending}
		run2.Complete(0.3, 30, false)
		require.Equal(t, EvalRunStatusFailed, run2.Status)
		require.Equal(t, 0.3, run2.DeterministicScore)
		require.Equal(t, 30, run2.RubricScore)
	})
}

func TestEvalRun_Fail(t *testing.T) {
	run := &EvalRun{Status: EvalRunStatusRunning}
	run.Fail()
	require.Equal(t, EvalRunStatusFailed, run.Status)
}
