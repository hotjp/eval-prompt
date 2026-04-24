package authz

import (
	"context"
	"errors"
	"testing"
)

func TestEvalGateGuard_CheckProdPromotion(t *testing.T) {
	tests := []struct {
		name       string
		snapshotID string
		scores     map[string]float64
		threshold  float64
		wantErr    bool
		errType    error
	}{
		{
			name:       "score above threshold passes",
			snapshotID: "snap-001",
			scores:     map[string]float64{"snap-001": 85.0},
			threshold:  80.0,
			wantErr:    false,
		},
		{
			name:       "score equal to threshold passes",
			snapshotID: "snap-002",
			scores:     map[string]float64{"snap-002": 80.0},
			threshold:  80.0,
			wantErr:    false,
		},
		{
			name:       "score below threshold fails",
			snapshotID: "snap-003",
			scores:     map[string]float64{"snap-003": 75.0},
			threshold:  80.0,
			wantErr:    true,
			errType:    ErrEvalScoreTooLow,
		},
		{
			name:       "snapshot not found returns error",
			snapshotID: "snap-missing",
			scores:     map[string]float64{},
			threshold:  80.0,
			wantErr:    true,
		},
		{
			name:       "custom threshold works",
			snapshotID: "snap-004",
			scores:     map[string]float64{"snap-004": 90.0},
			threshold:  95.0,
			wantErr:    true,
			errType:    ErrEvalScoreTooLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewSnapshotMetricsStore()
			for id, score := range tt.scores {
				store.SetEvalScore(id, score)
			}

			guard := NewEvalGateGuardWithThreshold(store, tt.threshold)
			err := guard.CheckProdPromotion(context.Background(), tt.snapshotID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestEvalGateGuard_Threshold(t *testing.T) {
	store := NewSnapshotMetricsStore()
	guard := NewEvalGateGuardWithThreshold(store, 85.0)

	if got := guard.Threshold(); got != 85.0 {
		t.Errorf("expected threshold 85.0, got %v", got)
	}
}

func TestSnapshotMetricsStore_SetAndGet(t *testing.T) {
	store := NewSnapshotMetricsStore()
	store.SetEvalScore("snap-001", 92.5)

	score, err := store.GetSnapshotEvalScore(context.Background(), "snap-001")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if score != 92.5 {
		t.Errorf("expected score 92.5, got %v", score)
	}
}
