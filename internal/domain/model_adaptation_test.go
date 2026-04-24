package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelAdaptation_Validate(t *testing.T) {
	validAdaptation := &ModelAdaptation{
		ID:             ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		PromptID:       ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"},
		SourceModel:    "gpt-4",
		TargetModel:    "claude-3",
		AdaptedContent: "adapted content",
	}

	tests := []struct {
		name    string
		ma      *ModelAdaptation
		wantErr bool
	}{
		{
			name:    "valid adaptation",
			ma:      validAdaptation,
			wantErr: false,
		},
		{
			name: "empty ID",
			ma: &ModelAdaptation{
				ID:             ID{},
				PromptID:       ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				SourceModel:    "gpt-4",
				TargetModel:    "claude-3",
				AdaptedContent: "content",
			},
			wantErr: true,
		},
		{
			name: "empty PromptID",
			ma: &ModelAdaptation{
				ID:             ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				PromptID:       ID{},
				SourceModel:    "gpt-4",
				TargetModel:    "claude-3",
				AdaptedContent: "content",
			},
			wantErr: true,
		},
		{
			name: "empty SourceModel",
			ma: &ModelAdaptation{
				ID:             ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				PromptID:       ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				SourceModel:    "",
				TargetModel:    "claude-3",
				AdaptedContent: "content",
			},
			wantErr: true,
		},
		{
			name: "empty TargetModel",
			ma: &ModelAdaptation{
				ID:             ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				PromptID:       ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				SourceModel:    "gpt-4",
				TargetModel:    "",
				AdaptedContent: "content",
			},
			wantErr: true,
		},
		{
			name: "empty AdaptedContent",
			ma: &ModelAdaptation{
				ID:             ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				PromptID:       ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				SourceModel:    "gpt-4",
				TargetModel:    "claude-3",
				AdaptedContent: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ma.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewModelAdaptation(t *testing.T) {
	promptID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}

	ma := NewModelAdaptation(promptID, "gpt-4", "claude-3", "adapted content")

	require.NotEmpty(t, ma.ID.String())
	require.Equal(t, promptID, ma.PromptID)
	require.Equal(t, "gpt-4", ma.SourceModel)
	require.Equal(t, "claude-3", ma.TargetModel)
	require.Equal(t, "adapted content", ma.AdaptedContent)
	require.NotNil(t, ma.ParamAdjustments)
	require.NotNil(t, ma.FormatChanges)
}
