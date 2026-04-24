package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAuditLog_Validate(t *testing.T) {
	validLog := &AuditLog{
		ID:        ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		Operation: "create",
		AssetID:   ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"},
		UserID:    ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FCC"},
	}

	tests := []struct {
		name    string
		log     *AuditLog
		wantErr bool
	}{
		{
			name:    "valid audit log",
			log:     validLog,
			wantErr: false,
		},
		{
			name: "empty Operation",
			log: &AuditLog{
				ID:        ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
				Operation: "",
				AssetID:   ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"},
				UserID:    ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FCC"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.log.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewAuditLog(t *testing.T) {
	assetID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FAV"}
	userID := ID{value: "01ARZ3NDEKTSV4RRFFQ69G5FBB"}
	details := map[string]any{"key": "value"}

	log := NewAuditLog("create", assetID, userID, details)

	require.NotEmpty(t, log.ID.String())
	require.Equal(t, "create", log.Operation)
	require.Equal(t, assetID, log.AssetID)
	require.Equal(t, userID, log.UserID)
	require.Equal(t, details, log.Details)
}
