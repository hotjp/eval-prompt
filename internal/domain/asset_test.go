package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAsset_Validate(t *testing.T) {
	validAsset := &Asset{
		ID:          ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Name:        "Test Asset",
		ContentHash: "hash123",
		FilePath:    "/path/to/file",
	}

	tests := []struct {
		name    string
		asset   *Asset
		wantErr bool
		errCode string
	}{
		{
			name:    "valid asset",
			asset:   validAsset,
			wantErr: false,
		},
		{
			name: "empty ID",
			asset: &Asset{
				ID:          ID{},
				Name:        "Test Asset",
				ContentHash: "hash123",
				FilePath:    "/path/to/file",
			},
			wantErr: true,
			errCode: "L2201",
		},
		{
			name: "empty name",
			asset: &Asset{
				ID:          ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Name:        "",
				ContentHash: "hash123",
				FilePath:    "/path/to/file",
			},
			wantErr: true,
		},
		{
			name: "name too long",
			asset: &Asset{
				ID:          ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Name:        string(make([]byte, 101)),
				ContentHash: "hash123",
				FilePath:    "/path/to/file",
			},
			wantErr: true,
		},
		{
			name: "empty content hash",
			asset: &Asset{
				ID:          ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Name:        "Test Asset",
				ContentHash: "",
				FilePath:    "/path/to/file",
			},
			wantErr: true,
			errCode: "L2211",
		},
		{
			name: "empty file path",
			asset: &Asset{
				ID:          ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Name:        "Test Asset",
				ContentHash: "hash123",
				FilePath:    "",
			},
			wantErr: true,
			errCode: "L2212",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.asset.Validate()
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

func TestAsset_CanPromote(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{AssetStateCreated, false},
		{AssetStateEvaluating, false},
		{AssetStateEvaluated, true},
		{AssetStatePromoted, true},
		{AssetStateArchived, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			asset := &Asset{State: tt.state}
			require.Equal(t, tt.want, asset.CanPromote())
		})
	}
}

func TestAsset_CanEval(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{AssetStateCreated, true},
		{AssetStateEvaluating, false},
		{AssetStateEvaluated, true},
		{AssetStatePromoted, false},
		{AssetStateArchived, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			asset := &Asset{State: tt.state}
			require.Equal(t, tt.want, asset.CanEval())
		})
	}
}

func TestAsset_CanArchive(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{AssetStateCreated, true},
		{AssetStateEvaluating, false},
		{AssetStateEvaluated, true},
		{AssetStatePromoted, true},
		{AssetStateArchived, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			asset := &Asset{State: tt.state}
			require.Equal(t, tt.want, asset.CanArchive())
		})
	}
}

func TestAsset_TransitionTo(t *testing.T) {
	tests := []struct {
		name      string
		from      State
		to        State
		event     EventType
		wantErr   bool
		wantState State
	}{
		{
			name:      "CREATED to EVALUATING with EvalStarted",
			from:      AssetStateCreated,
			to:        AssetStateEvaluating,
			event:     EventEvalStarted,
			wantErr:   false,
			wantState: AssetStateEvaluating,
		},
		{
			name:      "CREATED to ARCHIVED with Archive",
			from:      AssetStateCreated,
			to:        AssetStateArchived,
			event:     EventPromptAssetArchived,
			wantErr:   false,
			wantState: AssetStateArchived,
		},
		{
			name:      "EVALUATING to EVALUATED with EvalCompleted",
			from:      AssetStateEvaluating,
			to:        AssetStateEvaluated,
			event:     EventEvalCompleted,
			wantErr:   false,
			wantState: AssetStateEvaluated,
		},
		{
			name:      "EVALUATING to CREATED with EvalFailed",
			from:      AssetStateEvaluating,
			to:        AssetStateCreated,
			event:     EventEvalFailed,
			wantErr:   false,
			wantState: AssetStateCreated,
		},
		{
			name:      "EVALUATED to PROMOTED with LabelPromoted",
			from:      AssetStateEvaluated,
			to:        AssetStatePromoted,
			event:     EventLabelPromoted,
			wantErr:   false,
			wantState: AssetStatePromoted,
		},
		{
			name:      "EVALUATED to CREATED with ContentChanged",
			from:      AssetStateEvaluated,
			to:        AssetStateCreated,
			event:     EventPromptAssetUpdated,
			wantErr:   false,
			wantState: AssetStateCreated,
		},
		{
			name:      "EVALUATED to ARCHIVED with Archive",
			from:      AssetStateEvaluated,
			to:        AssetStateArchived,
			event:     EventPromptAssetArchived,
			wantErr:   false,
			wantState: AssetStateArchived,
		},
		{
			name:      "PROMOTED to CREATED with ContentChanged",
			from:      AssetStatePromoted,
			to:        AssetStateCreated,
			event:     EventPromptAssetUpdated,
			wantErr:   false,
			wantState: AssetStateCreated,
		},
		{
			name:      "PROMOTED to ARCHIVED with Archive",
			from:      AssetStatePromoted,
			to:        AssetStateArchived,
			event:     EventPromptAssetArchived,
			wantErr:   false,
			wantState: AssetStateArchived,
		},
		{
			name:      "invalid transition CREATED to PROMOTED",
			from:      AssetStateCreated,
			to:        AssetStatePromoted,
			event:     EventLabelPromoted,
			wantErr:   true,
			wantState: AssetStateCreated,
		},
		{
			name:      "invalid transition EVALUATING to PROMOTED",
			from:      AssetStateEvaluating,
			to:        AssetStatePromoted,
			event:     EventLabelPromoted,
			wantErr:   true,
			wantState: AssetStateEvaluating,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := &Asset{State: tt.from}
			err := asset.TransitionTo(tt.to, tt.event)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantState, asset.State)
			}
		})
	}
}

func TestNewAsset(t *testing.T) {
	asset := NewAsset("Test", "desc", "biz", "category", []string{"tag1"}, "hash123", "/path", "/repo")

	require.NotEmpty(t, asset.ID.String())
	require.Equal(t, "Test", asset.Name)
	require.Equal(t, "desc", asset.Description)
	require.Equal(t, "biz", asset.AssetType)
	require.Equal(t, "category", asset.Category)
	require.Equal(t, []string{"tag1"}, asset.Tags)
	require.Equal(t, "hash123", asset.ContentHash)
	require.Equal(t, "/path", asset.FilePath)
	require.Equal(t, "/repo", asset.RepoPath)
	require.Equal(t, AssetStateCreated, asset.State)
	require.Equal(t, int64(0), asset.Version)
}

func TestNewAssetWithID(t *testing.T) {
	specificID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	asset := NewAssetWithID(specificID, "Test", "desc", "biz", "category", []string{"tag1"}, "hash123", "/path", "/repo")

	require.Equal(t, specificID, asset.ID)
	require.Equal(t, "Test", asset.Name)
	require.Equal(t, "/repo", asset.RepoPath)
	require.Equal(t, AssetStateCreated, asset.State)
}
