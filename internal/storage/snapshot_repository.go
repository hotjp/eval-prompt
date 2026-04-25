package storage

import (
	"context"

	"github.com/eval-prompt/internal/domain"
)

// SnapshotRepository provides repository operations for Snapshot entities.
// Deprecated: Snapshot is no longer used. Eval history is stored in .md files.
type SnapshotRepository struct {
	client *Client
}

// NewSnapshotRepository creates a new SnapshotRepository.
// Deprecated: Snapshot is no longer used.
func NewSnapshotRepository(client *Client) *SnapshotRepository {
	return &SnapshotRepository{client: client}
}

// Create is a no-op since Snapshot is deprecated.
func (r *SnapshotRepository) Create(ctx context.Context, s *domain.Snapshot) error {
	if r.client == nil {
		return nil
	}
	return nil
}

// GetByID always returns nil since Snapshot is deprecated.
func (r *SnapshotRepository) GetByID(ctx context.Context, id string) (*domain.Snapshot, error) {
	return nil, nil
}

// GetByAssetID always returns empty slice since Snapshot is deprecated.
func (r *SnapshotRepository) GetByAssetID(ctx context.Context, assetID string) ([]*domain.Snapshot, error) {
	return []*domain.Snapshot{}, nil
}

// GetByAssetIDAndVersion always returns nil since Snapshot is deprecated.
func (r *SnapshotRepository) GetByAssetIDAndVersion(ctx context.Context, assetID, version string) (*domain.Snapshot, error) {
	return nil, nil
}

// GetByCommitHash always returns nil since Snapshot is deprecated.
func (r *SnapshotRepository) GetByCommitHash(ctx context.Context, commitHash string) (*domain.Snapshot, error) {
	return nil, nil
}

// List always returns empty slice since Snapshot is deprecated.
func (r *SnapshotRepository) List(ctx context.Context, offset, limit int) ([]*domain.Snapshot, int, error) {
	return []*domain.Snapshot{}, 0, nil
}

// Update is a no-op since Snapshot is deprecated.
func (r *SnapshotRepository) Update(ctx context.Context, s *domain.Snapshot) error {
	return nil
}
