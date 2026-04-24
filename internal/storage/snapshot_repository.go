package storage

import (
	"context"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent"
	"github.com/eval-prompt/internal/storage/ent/asset"
	"github.com/eval-prompt/internal/storage/ent/snapshot"
)

// SnapshotRepository provides repository operations for Snapshot entities.
type SnapshotRepository struct {
	client *Client
}

// NewSnapshotRepository creates a new SnapshotRepository.
func NewSnapshotRepository(client *Client) *SnapshotRepository {
	return &SnapshotRepository{client: client}
}

// Create creates a new snapshot in the database.
func (r *SnapshotRepository) Create(ctx context.Context, s *domain.Snapshot) error {
	_, err := r.client.ent.Snapshot.Create().
		SetID(s.ID.String()).
		SetAssetID(s.AssetID.String()).
		SetVersion(s.Version).
		SetContentHash(s.ContentHash).
		SetCommitHash(s.CommitHash).
		SetAuthor(s.Author).
		SetReason(s.Reason).
		SetModel(s.Model).
		SetTemperature(s.Temperature).
		SetMetrics(s.Metrics).
		SetCreatedAt(s.CreatedAt).
		Save(ctx)
	return err
}

// GetByID retrieves a snapshot by its ID.
func (r *SnapshotRepository) GetByID(ctx context.Context, id string) (*domain.Snapshot, error) {
	entSnapshot, err := r.client.ent.Snapshot.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.toDomainSnapshot(entSnapshot), nil
}

// GetByAssetID retrieves all snapshots for an asset.
func (r *SnapshotRepository) GetByAssetID(ctx context.Context, assetID string) ([]*domain.Snapshot, error) {
	entSnapshots, err := r.client.ent.Snapshot.Query().
		Where(snapshot.HasAssetWith(asset.IDEQ(assetID))).
		All(ctx)
	if err != nil {
		return nil, err
	}

	snapshots := make([]*domain.Snapshot, len(entSnapshots))
	for i, entSnapshot := range entSnapshots {
		snapshots[i] = r.toDomainSnapshot(entSnapshot)
	}
	return snapshots, nil
}

// GetByCommitHash retrieves a snapshot by its Git commit hash.
func (r *SnapshotRepository) GetByCommitHash(ctx context.Context, commitHash string) (*domain.Snapshot, error) {
	entSnapshots, err := r.client.ent.Snapshot.Query().
		Where(snapshot.CommitHash(commitHash)).
		Limit(1).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(entSnapshots) == 0 {
		return nil, nil
	}
	return r.toDomainSnapshot(entSnapshots[0]), nil
}

// List retrieves snapshots with pagination.
func (r *SnapshotRepository) List(ctx context.Context, offset, limit int) ([]*domain.Snapshot, int, error) {
	entSnapshots, err := r.client.ent.Snapshot.Query().
		Offset(offset).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.client.ent.Snapshot.Query().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	snapshots := make([]*domain.Snapshot, len(entSnapshots))
	for i, entSnapshot := range entSnapshots {
		snapshots[i] = r.toDomainSnapshot(entSnapshot)
	}
	return snapshots, total, nil
}

// Update updates a snapshot.
func (r *SnapshotRepository) Update(ctx context.Context, s *domain.Snapshot) error {
	_, err := r.client.ent.Snapshot.UpdateOneID(s.ID.String()).
		SetVersion(s.Version).
		SetContentHash(s.ContentHash).
		SetCommitHash(s.CommitHash).
		SetAuthor(s.Author).
		SetReason(s.Reason).
		SetModel(s.Model).
		SetTemperature(s.Temperature).
		SetMetrics(s.Metrics).
		Save(ctx)
	return err
}

// toDomainSnapshot converts an ent Snapshot to a domain Snapshot.
func (r *SnapshotRepository) toDomainSnapshot(e *ent.Snapshot) *domain.Snapshot {
	assetID := domain.ID{}
	if e.Edges.Asset != nil {
		assetID = domain.MustNewID(e.Edges.Asset.ID)
	}

	return &domain.Snapshot{
		ID:          domain.MustNewID(e.ID),
		AssetID:     assetID,
		Version:     e.Version,
		ContentHash: e.ContentHash,
		CommitHash:  e.CommitHash,
		Author:      e.Author,
		Reason:      e.Reason,
		Model:       e.Model,
		Temperature: e.Temperature,
		Metrics:     e.Metrics,
		CreatedAt:   e.CreatedAt,
	}
}
