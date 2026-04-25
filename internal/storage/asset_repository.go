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
		SetTags(a.Tags).
		SetContentHash(a.ContentHash).
		SetFilePath(a.FilePath).
		SetRepoPath(a.RepoPath).
		SetState(r.stateToEnt(a.State)).
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
		SetTags(a.Tags).
		SetContentHash(a.ContentHash).
		SetFilePath(a.FilePath).
		SetRepoPath(a.RepoPath).
		SetState(r.stateToEnt(a.State)).
		Save(ctx)
	return err
}

// Delete deletes an asset by its ID.
func (r *AssetRepository) Delete(ctx context.Context, id string) error {
	return r.client.ent.Asset.DeleteOneID(id).Exec(ctx)
}

// List retrieves assets with pagination.
func (r *AssetRepository) List(ctx context.Context, repoPath string, offset, limit int) ([]*domain.Asset, int, error) {
	var query *ent.AssetQuery
	if repoPath == "" {
		query = r.client.ent.Asset.Query()
	} else {
		query = r.client.ent.Asset.Query().Where(asset.RepoPathEQ(repoPath))
	}
	entAssets, err := query.Offset(offset).Limit(limit).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	assets := make([]*domain.Asset, len(entAssets))
	for i, entAsset := range entAssets {
		assets[i] = r.toDomainAsset(entAsset)
	}
	return assets, total, nil
}

// ListByState retrieves assets by state within a repo.
func (r *AssetRepository) ListByState(ctx context.Context, repoPath string, state domain.State, offset, limit int) ([]*domain.Asset, int, error) {
	entState := r.stateToEnt(state)
	var query *ent.AssetQuery
	if repoPath == "" {
		query = r.client.ent.Asset.Query().Where(asset.StateEQ(entState))
	} else {
		query = r.client.ent.Asset.Query().Where(asset.RepoPathEQ(repoPath), asset.StateEQ(entState))
	}
	entAssets, err := query.Offset(offset).Limit(limit).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	assets := make([]*domain.Asset, len(entAssets))
	for i, entAsset := range entAssets {
		assets[i] = r.toDomainAsset(entAsset)
	}
	return assets, total, nil
}

// toDomainAsset converts an ent Asset to a domain Asset.
func (r *AssetRepository) toDomainAsset(e *ent.Asset) *domain.Asset {
	return &domain.Asset{
		ID:          domain.MustNewID(e.ID),
		Name:        e.Name,
		Description: e.Description,
		Tags:        e.Tags,
		ContentHash: e.ContentHash,
		FilePath:    e.FilePath,
		RepoPath:    e.RepoPath,
		State:       r.stateFromEnt(e.State),
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

// GetByName retrieves an asset by its name within a repo.
func (r *AssetRepository) GetByName(ctx context.Context, repoPath, name string) (*domain.Asset, error) {
	var entAssets []*ent.Asset
	var err error
	if repoPath == "" {
		entAssets, err = r.client.ent.Asset.Query().
			Where(asset.Name(name)).
			Limit(1).
			All(ctx)
	} else {
		entAssets, err = r.client.ent.Asset.Query().
			Where(asset.RepoPathEQ(repoPath), asset.Name(name)).
			Limit(1).
			All(ctx)
	}
	if err != nil {
		return nil, err
	}
	if len(entAssets) == 0 {
		return nil, fmt.Errorf("asset not found: %s", name)
	}
	return r.toDomainAsset(entAssets[0]), nil
}
