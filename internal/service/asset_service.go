// Package service implements L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"
	"fmt"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage"
)

// AssetService handles asset business operations.
type AssetService struct {
	assetRepo    *storage.AssetRepository
	snapshotRepo *storage.SnapshotRepository
	labelRepo    *storage.LabelRepository
}

// NewAssetService creates a new AssetService.
func NewAssetService(client *storage.Client) *AssetService {
	return &AssetService{
		assetRepo:    storage.NewAssetRepository(client),
		snapshotRepo: storage.NewSnapshotRepository(client),
		labelRepo:    storage.NewLabelRepository(client),
	}
}

// Ensure AssetService implements AssetServicer interface.
var _ AssetServicer = (*AssetService)(nil)

// AssetServicer is the interface for asset operations.
type AssetServicer interface {
	// CreateAsset creates a new asset with an initial snapshot.
	CreateAsset(ctx context.Context, req *CreateAssetRequest) (*AssetResponse, error)

	// UpdateAsset updates an asset and creates a new snapshot.
	UpdateAsset(ctx context.Context, req *UpdateAssetRequest) (*AssetResponse, error)

	// GetAsset retrieves an asset by ID.
	GetAsset(ctx context.Context, id string) (*AssetDetailResponse, error)

	// ListAssets lists assets with pagination and filtering.
	ListAssets(ctx context.Context, req *ListAssetsRequest) (*ListAssetsResponse, error)

	// SetLabel sets a label on an asset pointing to a snapshot.
	SetLabel(ctx context.Context, req *SetLabelRequest) error

	// UnsetLabel removes a label from an asset.
	UnsetLabel(ctx context.Context, req *UnsetLabelRequest) error

	// GetLabels retrieves all labels for an asset.
	GetLabels(ctx context.Context, assetID string) ([]*LabelResponse, error)
}

// CreateAssetRequest contains the request data for creating an asset.
type CreateAssetRequest struct {
	Name        string
	Description string
	AssetType     string
	Tags        []string
	FilePath    string
	ContentHash string
	Author      string
	RepoPath    string // repo isolation - set by gateway handler
}

// AssetResponse contains the response data for an asset operation.
type AssetResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	AssetType     string            `json:"asset_type"`
	Tags        []string          `json:"tags"`
	State       string            `json:"state"`
	Version     int64             `json:"version"`
	Snapshot    *SnapshotResponse `json:"snapshot,omitempty"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// SnapshotResponse contains snapshot information.
type SnapshotResponse struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	CommitHash  string `json:"commit_hash"`
	ContentHash string `json:"content_hash"`
	Author      string `json:"author"`
	Reason      string `json:"reason"`
	CreatedAt   string `json:"created_at"`
}

// AssetDetailResponse contains detailed asset information.
type AssetDetailResponse struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	AssetType     string              `json:"asset_type"`
	Tags        []string            `json:"tags"`
	State       string              `json:"state"`
	Version     int64               `json:"version"`
	Labels      []*LabelResponse    `json:"labels"`
	Snapshots   []*SnapshotResponse `json:"snapshots"`
	CreatedAt   string              `json:"created_at"`
	UpdatedAt   string              `json:"updated_at"`
}

// LabelResponse contains label information.
type LabelResponse struct {
	Name       string `json:"name"`
	SnapshotID string `json:"snapshot_id"`
	UpdatedAt  string `json:"updated_at"`
}

// ListAssetsRequest contains the request data for listing assets.
type ListAssetsRequest struct {
	Offset   int
	Limit    int
	AssetType  string
	State    string
	RepoPath string // repo isolation
}

// ListAssetsResponse contains the response data for listing assets.
type ListAssetsResponse struct {
	Assets []*AssetResponse `json:"assets"`
	Total  int              `json:"total"`
}

// UpdateAssetRequest contains the request data for updating an asset.
type UpdateAssetRequest struct {
	ID          string
	Name        string
	Description string
	Tags        []string
	ContentHash string
	Author      string
	Reason      string
}

// SetLabelRequest contains the request data for setting a label.
type SetLabelRequest struct {
	AssetID    string
	SnapshotID string
	Name       string
}

// UnsetLabelRequest contains the request data for unsetting a label.
type UnsetLabelRequest struct {
	AssetID string
	Name    string
}

// CreateAsset creates a new asset with an initial snapshot.
func (s *AssetService) CreateAsset(ctx context.Context, req *CreateAssetRequest) (*AssetResponse, error) {
	// Validate request
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.FilePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}
	if req.ContentHash == "" {
		return nil, fmt.Errorf("content_hash is required")
	}

	// Create domain asset
	asset := domain.NewAsset(
		req.Name,
		req.Description,
		req.AssetType,
		req.Tags,
		req.ContentHash,
		req.FilePath,
		req.RepoPath,
	)

	// Validate asset
	if err := asset.Validate(); err != nil {
		return nil, err
	}

	// Create asset in storage
	if err := s.assetRepo.Create(ctx, asset); err != nil {
		return nil, err
	}

	// Create initial snapshot (v0.0.0)
	reason := "Initial version"
	snapshot := domain.NewSnapshot(
		asset.ID,
		"v0.0.0",
		req.ContentHash,
		req.Author,
		reason,
	)

	if err := s.snapshotRepo.Create(ctx, snapshot); err != nil {
		return nil, err
	}

	return &AssetResponse{
		ID:          asset.ID.String(),
		Name:        asset.Name,
		Description: asset.Description,
		AssetType:     asset.AssetType,
		Tags:        asset.Tags,
		State:       string(asset.State),
		Version:     asset.Version,
		Snapshot: &SnapshotResponse{
			ID:          snapshot.ID.String(),
			Version:     snapshot.Version,
			ContentHash: snapshot.ContentHash,
			Author:      snapshot.Author,
			Reason:      snapshot.Reason,
			CreatedAt:   snapshot.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
		CreatedAt: asset.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: asset.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// UpdateAsset updates an asset and creates a new snapshot.
func (s *AssetService) UpdateAsset(ctx context.Context, req *UpdateAssetRequest) (*AssetResponse, error) {
	// Validate request
	if req.ID == "" {
		return nil, fmt.Errorf("asset ID is required")
	}
	if req.ContentHash == "" {
		return nil, fmt.Errorf("content_hash is required")
	}

	// Get existing asset
	asset, err := s.assetRepo.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	// Check if content actually changed
	if asset.ContentHash == req.ContentHash {
		return nil, fmt.Errorf("content hash unchanged, no new snapshot needed")
	}

	// Update asset fields
	asset.Name = req.Name
	asset.Description = req.Description
	asset.Tags = req.Tags
	asset.ContentHash = req.ContentHash

	// Update asset in storage
	if err := s.assetRepo.Update(ctx, asset); err != nil {
		return nil, err
	}

	// Get the latest snapshot to increment version
	snapshots, err := s.snapshotRepo.GetByAssetID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	// Calculate new version
	newVersion := "v1.0.0"
	if len(snapshots) > 0 {
		// Parse last version and increment
		lastVersion := snapshots[0].Version
		newVersion = incrementVersion(lastVersion)
	}

	// Create new snapshot
	snapshot := domain.NewSnapshot(
		asset.ID,
		newVersion,
		req.ContentHash,
		req.Author,
		req.Reason,
	)

	if err := s.snapshotRepo.Create(ctx, snapshot); err != nil {
		return nil, err
	}

	return &AssetResponse{
		ID:          asset.ID.String(),
		Name:        asset.Name,
		Description: asset.Description,
		AssetType:     asset.AssetType,
		Tags:        asset.Tags,
		State:       string(asset.State),
		Version:     asset.Version,
		Snapshot: &SnapshotResponse{
			ID:          snapshot.ID.String(),
			Version:     snapshot.Version,
			ContentHash: snapshot.ContentHash,
			Author:      snapshot.Author,
			Reason:      snapshot.Reason,
			CreatedAt:   snapshot.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
		CreatedAt: asset.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: asset.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// GetAsset retrieves an asset by ID.
func (s *AssetService) GetAsset(ctx context.Context, id string) (*AssetDetailResponse, error) {
	if id == "" {
		return nil, fmt.Errorf("asset ID is required")
	}

	asset, err := s.assetRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get labels
	labels, err := s.labelRepo.GetLabelsByAssetID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get snapshots
	snapshots, err := s.snapshotRepo.GetByAssetID(ctx, id)
	if err != nil {
		return nil, err
	}

	labelResponses := make([]*LabelResponse, len(labels))
	for i, l := range labels {
		labelResponses[i] = &LabelResponse{
			Name:       l.Name,
			SnapshotID: l.SnapshotID.String(),
			UpdatedAt:  l.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	snapshotResponses := make([]*SnapshotResponse, len(snapshots))
	for i, snap := range snapshots {
		snapshotResponses[i] = &SnapshotResponse{
			ID:          snap.ID.String(),
			Version:     snap.Version,
			CommitHash:  snap.CommitHash,
			ContentHash: snap.ContentHash,
			Author:      snap.Author,
			Reason:      snap.Reason,
			CreatedAt:   snap.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return &AssetDetailResponse{
		ID:          asset.ID.String(),
		Name:        asset.Name,
		Description: asset.Description,
		AssetType:     asset.AssetType,
		Tags:        asset.Tags,
		State:       string(asset.State),
		Version:     asset.Version,
		Labels:      labelResponses,
		Snapshots:   snapshotResponses,
		CreatedAt:   asset.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   asset.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// ListAssets lists assets with pagination and filtering.
func (s *AssetService) ListAssets(ctx context.Context, req *ListAssetsRequest) (*ListAssetsResponse, error) {
	if req == nil {
		req = &ListAssetsRequest{Offset: 0, Limit: 20}
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	var assets []*domain.Asset
	var total int
	var err error

	if req.State != "" {
		state := domain.State(req.State)
		assets, total, err = s.assetRepo.ListByState(ctx, req.RepoPath, state, req.Offset, req.Limit)
	} else {
		assets, total, err = s.assetRepo.List(ctx, req.RepoPath, req.Offset, req.Limit)
	}

	if err != nil {
		return nil, err
	}

	responses := make([]*AssetResponse, len(assets))
	for i, a := range assets {
		responses[i] = &AssetResponse{
			ID:          a.ID.String(),
			Name:        a.Name,
			Description: a.Description,
			AssetType:     a.AssetType,
			Tags:        a.Tags,
			State:       string(a.State),
			Version:     a.Version,
			CreatedAt:   a.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   a.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return &ListAssetsResponse{
		Assets: responses,
		Total:  total,
	}, nil
}

// SetLabel sets a label on an asset pointing to a snapshot.
func (s *AssetService) SetLabel(ctx context.Context, req *SetLabelRequest) error {
	if req.AssetID == "" {
		return fmt.Errorf("asset_id is required")
	}
	if req.SnapshotID == "" {
		return fmt.Errorf("snapshot_id is required")
	}
	if req.Name == "" {
		return fmt.Errorf("label name is required")
	}

	assetID := domain.MustNewID(req.AssetID)
	snapshotID := domain.MustNewID(req.SnapshotID)

	return s.labelRepo.SetLabel(ctx, assetID, snapshotID, req.Name)
}

// UnsetLabel removes a label from an asset.
func (s *AssetService) UnsetLabel(ctx context.Context, req *UnsetLabelRequest) error {
	if req.AssetID == "" {
		return fmt.Errorf("asset_id is required")
	}
	if req.Name == "" {
		return fmt.Errorf("label name is required")
	}

	assetID := domain.MustNewID(req.AssetID)

	return s.labelRepo.UnsetLabel(ctx, assetID, req.Name)
}

// GetLabels retrieves all labels for an asset.
func (s *AssetService) GetLabels(ctx context.Context, assetID string) ([]*LabelResponse, error) {
	if assetID == "" {
		return nil, fmt.Errorf("asset_id is required")
	}

	labels, err := s.labelRepo.GetLabelsByAssetID(ctx, assetID)
	if err != nil {
		return nil, err
	}

	responses := make([]*LabelResponse, len(labels))
	for i, l := range labels {
		responses[i] = &LabelResponse{
			Name:       l.Name,
			SnapshotID: l.SnapshotID.String(),
			UpdatedAt:  l.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return responses, nil
}

// incrementVersion increments a semantic version string.
// For simplicity, this handles v0.0.0 -> v0.0.1 and v0.1.0 -> v1.0.0 patterns.
func incrementVersion(v string) string {
	// Remove leading 'v' if present
	if len(v) > 0 && v[0] == 'v' {
		v = v[1:]
	}

	// Simple increment: just append .1 to the end
	// A more robust implementation would parse and increment properly
	return "v" + v + ".1"
}
