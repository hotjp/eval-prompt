package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLabel_Validate(t *testing.T) {
	validLabel := &Label{
		ID:         ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		AssetID:    ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"},
		Name:       "prod",
		SnapshotID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FCC"},
	}

	tests := []struct {
		name    string
		label   *Label
		wantErr bool
		errCode string
	}{
		{
			name:    "valid label",
			label:   validLabel,
			wantErr: false,
		},
		{
			name: "empty ID",
			label: &Label{
				ID:         ID{},
				AssetID:    ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Name:       "prod",
				SnapshotID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FCC"},
			},
			wantErr: true,
			errCode: "L2201",
		},
		{
			name: "empty AssetID",
			label: &Label{
				ID:         ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				AssetID:    ID{},
				Name:       "prod",
				SnapshotID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FCC"},
			},
			wantErr: true,
		},
		{
			name: "empty Name",
			label: &Label{
				ID:         ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				AssetID:    ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Name:       "",
				SnapshotID: ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FCC"},
			},
			wantErr: true,
			errCode: "L2240",
		},
		{
			name: "empty SnapshotID",
			label: &Label{
				ID:         ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				AssetID:    ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Name:       "prod",
				SnapshotID: ID{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.label.Validate()
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

func TestLabel_NameHelpers(t *testing.T) {
	tests := []struct {
		name     string
		label    *Label
		isProd   bool
		isDev    bool
		isStaging bool
	}{
		{
			name:     "prod label",
			label:    &Label{Name: LabelNameProd},
			isProd:   true,
			isDev:    false,
			isStaging: false,
		},
		{
			name:     "dev label",
			label:    &Label{Name: LabelNameDev},
			isProd:   false,
			isDev:    true,
			isStaging: false,
		},
		{
			name:     "staging label",
			label:    &Label{Name: LabelNameStaging},
			isProd:   false,
			isDev:    false,
			isStaging: true,
		},
		{
			name:     "other label",
			label:    &Label{Name: "other"},
			isProd:   false,
			isDev:    false,
			isStaging: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isProd, tt.label.IsProd())
			require.Equal(t, tt.isDev, tt.label.IsDev())
			require.Equal(t, tt.isStaging, tt.label.IsStaging())
		})
	}
}

func TestNewLabel(t *testing.T) {
	assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	snapshotID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"}

	label := NewLabel(assetID, snapshotID, "prod")

	require.NotEmpty(t, label.ID.String())
	require.Equal(t, assetID, label.AssetID)
	require.Equal(t, snapshotID, label.SnapshotID)
	require.Equal(t, "prod", label.Name)
}
