package storage

import (
	"context"
	"fmt"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent"
	"github.com/eval-prompt/internal/storage/ent/label"
)

// LabelRepository provides repository operations for Label entities.
// Deprecated: Labels are stored in .md files, not in database.
type LabelRepository struct {
	client *Client
}

// NewLabelRepository creates a new LabelRepository.
func NewLabelRepository(client *Client) *LabelRepository {
	return &LabelRepository{client: client}
}

// SetLabel creates or updates a label for an asset.
// Deprecated: Labels are stored in .md files.
func (r *LabelRepository) SetLabel(ctx context.Context, assetID, snapshotID domain.ID, name string) error {
	if r.client == nil {
		return nil
	}
	// Check if label already exists
	existing, err := r.client.ent.Label.Query().
		Where(
			label.Name(name),
		).
		Limit(1).
		All(ctx)

	if err != nil {
		return err
	}

	if len(existing) > 0 {
		return nil
	}

	// Create new label
	_, err = r.client.ent.Label.Create().
		SetID(domain.NewAutoID().String()).
		SetName(name).
		Save(ctx)
	return err
}

// UnsetLabel removes a label.
func (r *LabelRepository) UnsetLabel(ctx context.Context, assetID domain.ID, name string) error {
	if r.client == nil {
		return nil
	}
	_, err := r.client.ent.Label.Delete().
		Where(
			label.Name(name),
		).
		Exec(ctx)
	return err
}

// GetLabelsByAssetID retrieves all labels for an asset.
// Deprecated: Labels are stored in .md files.
func (r *LabelRepository) GetLabelsByAssetID(ctx context.Context, assetID string) ([]*domain.Label, error) {
	if r.client == nil {
		return nil, nil
	}
	entLabels, err := r.client.ent.Label.Query().All(ctx)
	if err != nil {
		return nil, err
	}

	labels := make([]*domain.Label, 0)
	for _, entLabel := range entLabels {
		labels = append(labels, r.toDomainLabel(entLabel))
	}
	return labels, nil
}

// GetByName retrieves a label by asset ID and name.
// Deprecated: Labels are stored in .md files.
func (r *LabelRepository) GetByName(ctx context.Context, assetID string, name string) (*domain.Label, error) {
	if r.client == nil {
		return nil, nil
	}
	entLabels, err := r.client.ent.Label.Query().
		Where(
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
	if r.client == nil {
		return nil
	}
	return r.client.ent.Label.DeleteOneID(id).Exec(ctx)
}

// toDomainLabel converts an ent Label to a domain Label.
func (r *LabelRepository) toDomainLabel(e *ent.Label) *domain.Label {
	return &domain.Label{
		ID:        domain.MustNewID(e.ID),
		Name:      e.Name,
		UpdatedAt: e.UpdatedAt,
	}
}
