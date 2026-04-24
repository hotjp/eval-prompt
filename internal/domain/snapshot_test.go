package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSnapshot_Validate(t *testing.T) {
	validSnapshot := &Snapshot{
		ID:          ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		AssetID:     ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"},
		Version:     "v1.0.0",
		ContentHash: "hash123",
	}

	tests := []struct {
		name      string
		snapshot  *Snapshot
		wantErr   bool
		errCode   string
	}{
		{
			name:      "valid snapshot",
			snapshot:  validSnapshot,
			wantErr:   false,
		},
		{
			name: "empty ID",
			snapshot: &Snapshot{
				ID:          ID{},
				AssetID:     ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Version:     "v1.0.0",
				ContentHash: "hash123",
			},
			wantErr: true,
			errCode: "L2201",
		},
		{
			name: "empty AssetID",
			snapshot: &Snapshot{
				ID:          ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				AssetID:     ID{},
				Version:     "v1.0.0",
				ContentHash: "hash123",
			},
			wantErr: true,
		},
		{
			name: "empty Version",
			snapshot: &Snapshot{
				ID:          ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				AssetID:     ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Version:     "",
				ContentHash: "hash123",
			},
			wantErr: true,
			errCode: "L2220",
		},
		{
			name: "empty ContentHash",
			snapshot: &Snapshot{
				ID:          ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				AssetID:     ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Version:     "v1.0.0",
				ContentHash: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.snapshot.Validate()
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

func TestNewSnapshot(t *testing.T) {
	assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}

	snapshot := NewSnapshot(assetID, "v1.0.0", "hash123", "author", "reason")

	require.NotEmpty(t, snapshot.ID.String())
	require.Equal(t, assetID, snapshot.AssetID)
	require.Equal(t, "v1.0.0", snapshot.Version)
	require.Equal(t, "hash123", snapshot.ContentHash)
	require.Equal(t, "author", snapshot.Author)
	require.Equal(t, "reason", snapshot.Reason)
	require.NotNil(t, snapshot.Metrics)
}

func TestNewSnapshotWithCommit(t *testing.T) {
	assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}

	snapshot := NewSnapshotWithCommit(assetID, "v1.0.0", "hash123", "commit123", "author", "reason")

	require.NotEmpty(t, snapshot.ID.String())
	require.Equal(t, assetID, snapshot.AssetID)
	require.Equal(t, "v1.0.0", snapshot.Version)
	require.Equal(t, "hash123", snapshot.ContentHash)
	require.Equal(t, "commit123", snapshot.CommitHash)
	require.Equal(t, "author", snapshot.Author)
}
