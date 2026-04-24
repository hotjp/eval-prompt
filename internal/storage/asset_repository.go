package storage

import (
	"context"
	"fmt"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent"
	"github.com/eval-prompt/internal/storage/ent/asset"
)

// AssetRepository provides repository operations for Asset entities.
type AssetRepository struct {
	client *Client
}

// NewAssetRepository creates a new AssetRepository.
func NewAssetRepository(client *Client) *AssetRepository {
	return &AssetRepository{client: client}
}

// Create creates a new asset in the database.
func (r *AssetRepository) Create(ctx context.Context, a *domain.Asset) error {
	_, err := r.client.ent.Asset.Create().
		SetID(a.ID.String()).
		SetName(a.Name).
		SetDescription(a.Description).
		SetBizLine(a.BizLine).
		SetTags(a.Tags).
		SetContentHash(a.ContentHash).
		SetFilePath(a.FilePath).
		SetState(r.stateToEnt(a.State)).
		SetCreatedAt(a.CreatedAt).
		SetUpdatedAt(a.UpdatedAt).
		Save(ctx)
	return err
}

// GetByID retrieves an asset by its ID.
func (r *AssetRepository) GetByID(ctx context.Context, id string) (*domain.Asset, error) {
	entAsset, err := r.client.ent.Asset.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.toDomainAsset(entAsset), nil
}

// Update updates an existing asset.
func (r *AssetRepository) Update(ctx context.Context, a *domain.Asset) error {
	_, err := r.client.ent.Asset.UpdateOneID(a.ID.String()).
		SetName(a.Name).
		SetDescription(a.Description).
		SetBizLine(a.BizLine).
		SetTags(a.Tags).
		SetContentHash(a.ContentHash).
		SetFilePath(a.FilePath).
		SetState(r.stateToEnt(a.State)).
		SetUpdatedAt(a.UpdatedAt).
		Save(ctx)
	return err
}

// Delete deletes an asset by its ID.
func (r *AssetRepository) Delete(ctx context.Context, id string) error {
	return r.client.ent.Asset.DeleteOneID(id).Exec(ctx)
}

// List retrieves assets with pagination.
func (r *AssetRepository) List(ctx context.Context, offset, limit int) ([]*domain.Asset, int, error) {
	entAssets, err := r.client.ent.Asset.Query().
		Offset(offset).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.client.ent.Asset.Query().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	assets := make([]*domain.Asset, len(entAssets))
	for i, entAsset := range entAssets {
		assets[i] = r.toDomainAsset(entAsset)
	}
	return assets, total, nil
}

// ListByBizLine retrieves assets by business line.
func (r *AssetRepository) ListByBizLine(ctx context.Context, bizLine string, offset, limit int) ([]*domain.Asset, int, error) {
	entAssets, err := r.client.ent.Asset.Query().
		Where(asset.BizLine(bizLine)).
		Offset(offset).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.client.ent.Asset.Query().Where(asset.BizLine(bizLine)).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	assets := make([]*domain.Asset, len(entAssets))
	for i, entAsset := range entAssets {
		assets[i] = r.toDomainAsset(entAsset)
	}
	return assets, total, nil
}

// ListByState retrieves assets by state.
func (r *AssetRepository) ListByState(ctx context.Context, state domain.State, offset, limit int) ([]*domain.Asset, int, error) {
	entState := r.stateToEnt(state)
	entAssets, err := r.client.ent.Asset.Query().
		Where(asset.StateEQ(entState)).
		Offset(offset).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.client.ent.Asset.Query().Where(asset.StateEQ(entState)).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	assets := make([]*domain.Asset, len(entAssets))
	for i, entAsset := range entAssets {
		assets[i] = r.toDomainAsset(entAsset)
	}
	return assets, total, nil
}

// toEntAsset converts a domain Asset to an ent Asset.
func (r *AssetRepository) toEntAsset(a *domain.Asset) *ent.Asset {
	return &ent.Asset{
		ID:          a.ID.String(),
		Name:        a.Name,
		Description: a.Description,
		BizLine:     a.BizLine,
		Tags:        a.Tags,
		ContentHash: a.ContentHash,
		FilePath:    a.FilePath,
		State:       r.stateToEnt(a.State),
		CreatedAt:   a.CreatedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

// toDomainAsset converts an ent Asset to a domain Asset.
func (r *AssetRepository) toDomainAsset(e *ent.Asset) *domain.Asset {
	return &domain.Asset{
		ID:          domain.MustNewID(e.ID),
		Name:        e.Name,
		Description: e.Description,
		BizLine:     e.BizLine,
		Tags:        e.Tags,
		ContentHash: e.ContentHash,
		FilePath:    e.FilePath,
		State:       r.stateFromEnt(e.State),
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		Version:     0, // ent doesn't track version the same way
	}
}

// stateToEnt converts a domain state to an ent state.
func (r *AssetRepository) stateToEnt(state domain.State) asset.State {
	switch state {
	case domain.AssetStateCreated:
		return asset.StateCreated
	case domain.AssetStateEvaluating:
		return asset.StateEvaluating
	case domain.AssetStateEvaluated:
		return asset.StateEvaluated
	case domain.AssetStatePromoted:
		return asset.StatePromoted
	case domain.AssetStateArchived:
		return asset.StateArchived
	default:
		return asset.StateCreated
	}
}

// stateFromEnt converts an ent state to a domain state.
func (r *AssetRepository) stateFromEnt(state asset.State) domain.State {
	switch state {
	case asset.StateCreated:
		return domain.AssetStateCreated
	case asset.StateEvaluating:
		return domain.AssetStateEvaluating
	case asset.StateEvaluated:
		return domain.AssetStateEvaluated
	case asset.StatePromoted:
		return domain.AssetStatePromoted
	case asset.StateArchived:
		return domain.AssetStateArchived
	default:
		return domain.AssetStateCreated
	}
}

// UpdateState updates only the state of an asset.
func (r *AssetRepository) UpdateState(ctx context.Context, id string, state domain.State) error {
	_, err := r.client.ent.Asset.UpdateOneID(id).
		SetState(r.stateToEnt(state)).
		Save(ctx)
	return err
}

// GetByName retrieves an asset by its name.
func (r *AssetRepository) GetByName(ctx context.Context, name string) (*domain.Asset, error) {
	entAssets, err := r.client.ent.Asset.Query().
		Where(asset.Name(name)).
		Limit(1).
		All(ctx)
	if err != nil {
		return nil, err
	}
	if len(entAssets) == 0 {
		return nil, fmt.Errorf("asset not found: %s", name)
	}
	return r.toDomainAsset(entAssets[0]), nil
}
