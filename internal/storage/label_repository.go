package storage

import (
	"context"
	"fmt"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent"
	"github.com/eval-prompt/internal/storage/ent/asset"
	"github.com/eval-prompt/internal/storage/ent/label"
)

// LabelRepository provides repository operations for Label entities.
type LabelRepository struct {
	client *Client
}

// NewLabelRepository creates a new LabelRepository.
func NewLabelRepository(client *Client) *LabelRepository {
	return &LabelRepository{client: client}
}

// SetLabel creates or updates a label for an asset pointing to a snapshot.
// Deprecated: Snapshot is no longer used. Labels now point directly to asset versions via file_path.
func (r *LabelRepository) SetLabel(ctx context.Context, assetID, snapshotID domain.ID, name string) error {
	// Check if label already exists
	existing, err := r.client.ent.Label.Query().
		Where(
			label.HasAssetWith(asset.IDEQ(assetID.String())),
			label.Name(name),
		).
		Limit(1).
		All(ctx)

	if err != nil {
		return err
	}

	if len(existing) > 0 {
		// Update existing label (snapshotID is no longer stored)
		return nil
	}

	// Create new label
	_, err = r.client.ent.Label.Create().
		SetID(domain.NewAutoID().String()).
		SetAssetID(assetID.String()).
		SetName(name).
		Save(ctx)
	return err
}

// UnsetLabel removes a label.
func (r *LabelRepository) UnsetLabel(ctx context.Context, assetID domain.ID, name string) error {
	_, err := r.client.ent.Label.Delete().
		Where(
			label.HasAssetWith(asset.IDEQ(assetID.String())),
			label.Name(name),
		).
		Exec(ctx)
	return err
}

// GetLabelsByAssetID retrieves all labels for an asset.
func (r *LabelRepository) GetLabelsByAssetID(ctx context.Context, assetID string) ([]*domain.Label, error) {
	entLabels, err := r.client.ent.Label.Query().
		Where(label.HasAssetWith(asset.IDEQ(assetID))).
		All(ctx)
	if err != nil {
		return nil, err
	}

	labels := make([]*domain.Label, len(entLabels))
	for i, entLabel := range entLabels {
		labels[i] = r.toDomainLabel(entLabel)
	}
	return labels, nil
}

// GetByName retrieves a label by asset ID and name.
func (r *LabelRepository) GetByName(ctx context.Context, assetID string, name string) (*domain.Label, error) {
	entLabels, err := r.client.ent.Label.Query().
		Where(
			label.HasAssetWith(asset.IDEQ(assetID)),
			label.Name(name),
		).
		Limit(1).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(entLabels) == 0 {
		return nil, fmt.Errorf("label not found: %s", name)
	}
	return r.toDomainLabel(entLabels[0]), nil
}

// Delete deletes a label.
func (r *LabelRepository) Delete(ctx context.Context, id string) error {
	return r.client.ent.Label.DeleteOneID(id).Exec(ctx)
}

// toDomainLabel converts an ent Label to a domain Label.
func (r *LabelRepository) toDomainLabel(e *ent.Label) *domain.Label {
	assetID := domain.ID{}
	if e.Edges.Asset != nil {
		assetID = domain.MustNewID(e.Edges.Asset.ID)
	}

	return &domain.Label{
		ID:         domain.MustNewID(e.ID),
		AssetID:    assetID,
		Name:       e.Name,
		UpdatedAt:  e.UpdatedAt,
	}
}
