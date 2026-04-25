package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelStat_Update(t *testing.T) {
	t.Run("first update sets initial values", func(t *testing.T) {
		s := &ModelStat{}
		s.Update(80.0)
		require.Equal(t, 1, s.Count)
		require.Equal(t, 80.0, s.Mean)
		require.Equal(t, 80.0, s.Min)
		require.Equal(t, 80.0, s.Max)
		require.NotEmpty(t, s.LastRun)
	})

	t.Run("second update adjusts mean", func(t *testing.T) {
		s := &ModelStat{}
		s.Update(80.0)
		s.Update(60.0)
		require.Equal(t, 2, s.Count)
		require.Equal(t, 70.0, s.Mean)
		require.Equal(t, 60.0, s.Min)
		require.Equal(t, 80.0, s.Max)
	})

	t.Run("updates track min and max correctly", func(t *testing.T) {
		s := &ModelStat{}
		s.Update(50.0)
		s.Update(90.0)
		s.Update(70.0)
		require.Equal(t, 3, s.Count)
		require.Equal(t, 70.0, s.Mean)
		require.Equal(t, 50.0, s.Min)
		require.Equal(t, 90.0, s.Max)
	})
}

func TestModelStat_StdDev(t *testing.T) {
	t.Run("zero count returns zero", func(t *testing.T) {
		s := &ModelStat{}
		require.Equal(t, 0.0, s.StdDev())
	})

	t.Run("single value returns zero", func(t *testing.T) {
		s := &ModelStat{Count: 1, Mean: 80.0}
		require.Equal(t, 0.0, s.StdDev())
	})

	t.Run("two values compute stddev", func(t *testing.T) {
		s := &ModelStat{Count: 2, Mean: 75.0, M2: 200.0}
		require.InDelta(t, 14.14, s.StdDev(), 0.01)
	})
}

func TestFrontMatter_Validate(t *testing.T) {
	validFM := &FrontMatter{
		ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Name:        "Test Asset",
		ContentHash: "hash123",
		State:       "created",
	}

	tests := []struct {
		name    string
		fm      *FrontMatter
		wantErr bool
		errCode string
	}{
		{
			name:    "valid front matter",
			fm:      validFM,
			wantErr: false,
		},
		{
			name: "empty ID",
			fm: &FrontMatter{
				ID:          "",
				Name:        "Test Asset",
				ContentHash: "hash123",
			},
			wantErr: true,
		},
		{
			name: "invalid ULID in ID",
			fm: &FrontMatter{
				ID:          "not-a-ulid",
				Name:        "Test Asset",
				ContentHash: "hash123",
			},
			wantErr: true,
		},
		{
			name: "empty Name",
			fm: &FrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "",
				ContentHash: "hash123",
			},
			wantErr: true,
			errCode: "L2209",
		},
		{
			name: "empty ContentHash",
			fm: &FrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "Test Asset",
				ContentHash: "",
			},
			wantErr: true,
			errCode: "L2211",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fm.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errCode != "" {
					require.Contains(t, err.Error(), tt.errCode)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFrontMatter_HasEvalHistory(t *testing.T) {
	t.Run("has eval history", func(t *testing.T) {
		fm := &FrontMatter{
			EvalHistory: []EvalHistoryEntry{{Score: 80}},
		}
		require.True(t, fm.HasEvalHistory())
	})

	t.Run("empty eval history", func(t *testing.T) {
		fm := &FrontMatter{EvalHistory: []EvalHistoryEntry{}}
		require.False(t, fm.HasEvalHistory())
	})

	t.Run("nil eval history", func(t *testing.T) {
		fm := &FrontMatter{}
		require.False(t, fm.HasEvalHistory())
	})
}

func TestFrontMatter_HasLabels(t *testing.T) {
	t.Run("has labels", func(t *testing.T) {
		fm := &FrontMatter{Labels: []LabelEntry{{Name: "prod"}}}
		require.True(t, fm.HasLabels())
	})

	t.Run("empty labels", func(t *testing.T) {
		fm := &FrontMatter{Labels: []LabelEntry{}}
		require.False(t, fm.HasLabels())
	})

	t.Run("nil labels", func(t *testing.T) {
		fm := &FrontMatter{}
		require.False(t, fm.HasLabels())
	})
}

func TestEvalPromptFrontMatter_Validate(t *testing.T) {
	validFM := &EvalPromptFrontMatter{
		ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Name:        "Eval Prompt",
		ContentHash: "hash123",
		State:       "created",
		Model:       "gpt-4",
	}

	tests := []struct {
		name    string
		fm      *EvalPromptFrontMatter
		wantErr bool
		errCode string
	}{
		{
			name:    "valid eval prompt front matter",
			fm:      validFM,
			wantErr: false,
		},
		{
			name: "empty ID",
			fm: &EvalPromptFrontMatter{
				ID:          "",
				Name:        "Eval Prompt",
				ContentHash: "hash123",
			},
			wantErr: true,
		},
		{
			name: "invalid ULID in ID",
			fm: &EvalPromptFrontMatter{
				ID:          "not-a-ulid",
				Name:        "Eval Prompt",
				ContentHash: "hash123",
			},
			wantErr: true,
		},
		{
			name: "empty Name",
			fm: &EvalPromptFrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "",
				ContentHash: "hash123",
			},
			wantErr: true,
			errCode: "L2209",
		},
		{
			name: "empty ContentHash",
			fm: &EvalPromptFrontMatter{
				ID:          "01ARZ3NDEKTSV4RRFFQ69G5FAV",
				Name:        "Eval Prompt",
				ContentHash: "",
			},
			wantErr: true,
			errCode: "L2211",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fm.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errCode != "" {
					require.Contains(t, err.Error(), tt.errCode)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEvalPromptFrontMatter_HasEvalCaseIDs(t *testing.T) {
	t.Run("has eval case IDs", func(t *testing.T) {
		fm := &EvalPromptFrontMatter{EvalCaseIDs: []string{"id1", "id2"}}
		require.True(t, fm.HasEvalCaseIDs())
	})

	t.Run("empty eval case IDs", func(t *testing.T) {
		fm := &EvalPromptFrontMatter{EvalCaseIDs: []string{}}
		require.False(t, fm.HasEvalCaseIDs())
	})

	t.Run("nil eval case IDs", func(t *testing.T) {
		fm := &EvalPromptFrontMatter{}
		require.False(t, fm.HasEvalCaseIDs())
	})
}
