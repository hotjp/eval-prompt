// Package authz implements L3-Authz layer: permission checks (RBAC/OpenFGA),
// rate limiting, and identity verification.
package authz

import (
	"context"
	"errors"
	"fmt"
)

// ErrEvalScoreTooLow is returned when the eval score is below the threshold.
var ErrEvalScoreTooLow = errors.New("eval score below threshold")

// DefaultEvalThreshold is the default minimum score for prod promotion.
const DefaultEvalThreshold = 80.0

// EvalGateGuard checks if a snapshot's eval score meets the threshold before allowing prod promotion.
type EvalGateGuard struct {
	snapshotGetter SnapshotGetter
	threshold      float64
}

// SnapshotGetter retrieves a snapshot by ID and returns its eval score.
type SnapshotGetter interface {
	GetSnapshotEvalScore(ctx context.Context, snapshotID string) (float64, error)
}

// NewEvalGateGuard creates a new EvalGateGuard.
func NewEvalGateGuard(getter SnapshotGetter) *EvalGateGuard {
	return &EvalGateGuard{
		snapshotGetter: getter,
		threshold:      DefaultEvalThreshold,
	}
}

// NewEvalGateGuardWithThreshold creates a new EvalGateGuard with a custom threshold.
func NewEvalGateGuardWithThreshold(getter SnapshotGetter, threshold float64) *EvalGateGuard {
	return &EvalGateGuard{
		snapshotGetter: getter,
		threshold:      threshold,
	}
}

// CheckProdPromotion checks if the snapshot meets the eval score threshold for prod promotion.
// Returns nil if promotion is allowed, ErrEvalScoreTooLow if score is below threshold.
func (g *EvalGateGuard) CheckProdPromotion(ctx context.Context, snapshotID string) error {
	score, err := g.snapshotGetter.GetSnapshotEvalScore(ctx, snapshotID)
	if err != nil {
		return fmt.Errorf("failed to get eval score: %w", err)
	}

	if score < g.threshold {
		return fmt.Errorf("%w: got %.2f, need %.2f", ErrEvalScoreTooLow, score, g.threshold)
	}

	return nil
}

// Threshold returns the configured threshold.
func (g *EvalGateGuard) Threshold() float64 {
	return g.threshold
}

// SnapshotMetricsStore is an in-memory implementation of SnapshotGetter for testing and development.
type SnapshotMetricsStore struct {
	scores map[string]float64
}

// NewSnapshotMetricsStore creates a new in-memory snapshot metrics store.
func NewSnapshotMetricsStore() *SnapshotMetricsStore {
	return &SnapshotMetricsStore{
		scores: make(map[string]float64),
	}
}

// Ensure SnapshotMetricsStore implements SnapshotGetter.
var _ SnapshotGetter = (*SnapshotMetricsStore)(nil)

// SetEvalScore sets the eval score for a snapshot.
func (s *SnapshotMetricsStore) SetEvalScore(snapshotID string, score float64) {
	s.scores[snapshotID] = score
}

// GetSnapshotEvalScore returns the eval score for a snapshot.
func (s *SnapshotMetricsStore) GetSnapshotEvalScore(ctx context.Context, snapshotID string) (float64, error) {
	score, ok := s.scores[snapshotID]
	if !ok {
		return 0, fmt.Errorf("snapshot not found: %s", snapshotID)
	}
	return score, nil
}
