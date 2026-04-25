package storage

import (
	"context"
	"testing"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent/enttest"
	"github.com/stretchr/testify/require"
)

// AssetRepository tests using testify/require

func TestAssetRepository_Create_Require(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset Require",
		Description: "A test asset with require",
		AssetType:     "test",
		Tags:        []string{"test", "require"},
		ContentHash: "hashabc",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}

	err := repo.Create(ctx, asset)
	require.NoError(t, err, "Create should not return error")

	got, err := repo.GetByID(ctx, asset.ID.String())
	require.NoError(t, err, "GetByID should not return error")
	require.Equal(t, asset.Name, got.Name, "Name should match")
	require.Equal(t, asset.Description, got.Description, "Description should match")
	require.Equal(t, asset.ContentHash, got.ContentHash, "ContentHash should match")
	require.Equal(t, asset.State, got.State, "State should match")
}

func TestAssetRepository_Update_Require(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Original Name",
		Description: "Original description",
		AssetType:     "test",
		Tags:        []string{"test"},
		ContentHash: "hash123",
		FilePath:    "/prompts/update.md",
		State:       domain.AssetStateCreated,
	}

	err := repo.Create(ctx, asset)
	require.NoError(t, err, "Create should not return error")

	asset.Name = "Updated Name"
	asset.Description = "Updated description"
	asset.Tags = []string{"test", "updated"}

	err = repo.Update(ctx, asset)
	require.NoError(t, err, "Update should not return error")

	got, err := repo.GetByID(ctx, asset.ID.String())
	require.NoError(t, err, "GetByID should not return error")
	require.Equal(t, "Updated Name", got.Name, "Name should be updated")
	require.Equal(t, "Updated description", got.Description, "Description should be updated")
}

func TestAssetRepository_Delete_Require(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Delete Test Require",
		Description: "Test delete with require",
		AssetType:     "test",
		Tags:        []string{"test"},
		ContentHash: "hash123",
		FilePath:    "/prompts/delete.md",
		State:       domain.AssetStateCreated,
	}

	err := repo.Create(ctx, asset)
	require.NoError(t, err, "Create should not return error")

	err = repo.Delete(ctx, asset.ID.String())
	require.NoError(t, err, "Delete should not return error")

	_, err = repo.GetByID(ctx, asset.ID.String())
	require.Error(t, err, "GetByID should return error after delete")
}

func TestAssetRepository_List_Require(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	// Create multiple assets
	for i := 0; i < 3; i++ {
		asset := &domain.Asset{
			ID:          domain.NewAutoID(),
			Name:        "Asset",
			Description: "Test",
			AssetType:     "test",
			Tags:        []string{"test"},
			ContentHash: "hash",
			FilePath:    "/prompts/asset.md",
			State:       domain.AssetStateCreated,
		}
		err := repo.Create(ctx, asset)
		require.NoError(t, err, "Create should not return error")
	}

	assets, total, err := repo.List(ctx, "", 0, 10)
	require.NoError(t, err, "List should not return error")
	require.Equal(t, 3, total, "Total should be 3")
	require.Len(t, assets, 3, "Should return 3 assets")
}

func TestAssetRepository_ListByState_Require(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	// Create assets with different states
	states := []domain.State{domain.AssetStateCreated, domain.AssetStateEvaluated, domain.AssetStateCreated}
	for i, state := range states {
		asset := &domain.Asset{
			ID:          domain.NewAutoID(),
			Name:        "Asset",
			Description: "Test",
			AssetType:     "test",
			Tags:        []string{"test"},
			ContentHash: "hash",
			FilePath:    "/prompts/asset.md",
			State:       state,
		}
		err := repo.Create(ctx, asset)
		require.NoError(t, err, "Create should not return error for asset %d", i)
	}

	assets, total, err := repo.ListByState(ctx, "", domain.AssetStateCreated, 0, 10)
	require.NoError(t, err, "ListByState should not return error")
	require.Equal(t, 2, total, "Total Created should be 2")
	require.Len(t, assets, 2, "Should return 2 assets in Created state")
}

func TestAssetRepository_UpdateState_Require(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "State Test Require",
		Description: "Test state update with require",
		AssetType:     "test",
		Tags:        []string{"test"},
		ContentHash: "hash",
		FilePath:    "/prompts/state.md",
		State:       domain.AssetStateCreated,
	}

	err := repo.Create(ctx, asset)
	require.NoError(t, err, "Create should not return error")

	err = repo.UpdateState(ctx, asset.ID.String(), domain.AssetStateEvaluated)
	require.NoError(t, err, "UpdateState should not return error")

	got, err := repo.GetByID(ctx, asset.ID.String())
	require.NoError(t, err, "GetByID should not return error")
	require.Equal(t, domain.AssetStateEvaluated, got.State, "State should be Updated to Evaluated")
}

// SnapshotRepository tests - deprecated but verify no-op behavior

func TestSnapshotRepository_NoOpBehavior_Require(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	ctx := context.Background()

	// Create returns nil (no-op)
	snapshot := &domain.Snapshot{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		AssetID:     domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		Version:     "v1.0.0",
		ContentHash: "hash123",
		Author:      "tester",
		Reason:      "Initial",
	}

	err := snapshotRepo.Create(ctx, snapshot)
	require.NoError(t, err, "Create should be a no-op and return nil")

	// GetByID returns nil (deprecated)
	result, err := snapshotRepo.GetByID(ctx, snapshot.ID.String())
	require.NoError(t, err, "GetByID should return nil without error")
	require.Nil(t, result, "GetByID should return nil for deprecated SnapshotRepository")

	// GetByAssetID returns empty slice
	snapshots, err := snapshotRepo.GetByAssetID(ctx, snapshot.AssetID.String())
	require.NoError(t, err, "GetByAssetID should return empty slice")
	require.Len(t, snapshots, 0, "GetByAssetID should return empty slice")

	// List returns empty slice
	allSnapshots, total, err := snapshotRepo.List(ctx, 0, 10)
	require.NoError(t, err, "List should return empty slice")
	require.Len(t, allSnapshots, 0, "List should return empty slice")
	require.Equal(t, 0, total, "Total should be 0")

	// Update returns nil (no-op)
	err = snapshotRepo.Update(ctx, snapshot)
	require.NoError(t, err, "Update should be a no-op and return nil")

	// GetByAssetIDAndVersion returns nil
	result, err = snapshotRepo.GetByAssetIDAndVersion(ctx, snapshot.AssetID.String(), "v1.0.0")
	require.NoError(t, err, "GetByAssetIDAndVersion should return nil without error")
	require.Nil(t, result, "GetByAssetIDAndVersion should return nil")

	// GetByCommitHash returns nil
	result2, err := snapshotRepo.GetByCommitHash(ctx, "abc123")
	require.NoError(t, err, "GetByCommitHash should return nil without error")
	require.Nil(t, result2, "GetByCommitHash should return nil")
}

// LabelRepository tests using testify/require

func TestLabelRepository_SetLabel_Require(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	labelRepo := NewLabelRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	ctx := context.Background()

	// Create asset
	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset Require",
		Description: "Test",
		AssetType:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	require.NoError(t, err, "Create asset should not return error")

	// Create snapshot
	snapshot := &domain.Snapshot{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		AssetID:     asset.ID,
		Version:     "v1.0.0",
		ContentHash: "hash123",
		Author:      "tester",
		Reason:      "Initial",
	}
	err = snapshotRepo.Create(ctx, snapshot)
	require.NoError(t, err, "Create snapshot should not return error")

	// Set label
	err = labelRepo.SetLabel(ctx, asset.ID, snapshot.ID, "prod")
	require.NoError(t, err, "SetLabel should not return error")

	// Verify label exists
	labels, err := labelRepo.GetLabelsByAssetID(ctx, asset.ID.String())
	require.NoError(t, err, "GetLabelsByAssetID should not return error")
	require.Len(t, labels, 1, "Should have 1 label")
	require.Equal(t, "prod", labels[0].Name, "Label name should be prod")
}

func TestLabelRepository_UnsetLabel_Require(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	labelRepo := NewLabelRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	ctx := context.Background()

	// Setup
	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "Test",
		AssetType:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	require.NoError(t, err, "Create asset should not return error")

	snapshot := &domain.Snapshot{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		AssetID:     asset.ID,
		Version:     "v1.0.0",
		ContentHash: "hash123",
		Author:      "tester",
		Reason:      "Initial",
	}
	err = snapshotRepo.Create(ctx, snapshot)
	require.NoError(t, err, "Create snapshot should not return error")

	err = labelRepo.SetLabel(ctx, asset.ID, snapshot.ID, "prod")
	require.NoError(t, err, "SetLabel should not return error")

	// Unset label
	err = labelRepo.UnsetLabel(ctx, asset.ID, "prod")
	require.NoError(t, err, "UnsetLabel should not return error")

	// Verify label is gone
	labels, err := labelRepo.GetLabelsByAssetID(ctx, asset.ID.String())
	require.NoError(t, err, "GetLabelsByAssetID should not return error")
	require.Len(t, labels, 0, "Should have 0 labels after unset")
}

func TestLabelRepository_GetByName_Require(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	labelRepo := NewLabelRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	ctx := context.Background()

	// Setup
	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "Test",
		AssetType:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	require.NoError(t, err, "Create asset should not return error")

	snapshot := &domain.Snapshot{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		AssetID:     asset.ID,
		Version:     "v1.0.0",
		ContentHash: "hash123",
		Author:      "tester",
		Reason:      "Initial",
	}
	err = snapshotRepo.Create(ctx, snapshot)
	require.NoError(t, err, "Create snapshot should not return error")

	err = labelRepo.SetLabel(ctx, asset.ID, snapshot.ID, "prod")
	require.NoError(t, err, "SetLabel should not return error")

	// GetByName
	label, err := labelRepo.GetByName(ctx, asset.ID.String(), "prod")
	require.NoError(t, err, "GetByName should not return error")
	require.Equal(t, "prod", label.Name, "Label name should be prod")
}

func TestLabelRepository_Delete_Require(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	labelRepo := NewLabelRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	ctx := context.Background()

	// Setup
	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "Test",
		AssetType:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	require.NoError(t, err, "Create asset should not return error")

	snapshot := &domain.Snapshot{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		AssetID:     asset.ID,
		Version:     "v1.0.0",
		ContentHash: "hash123",
		Author:      "tester",
		Reason:      "Initial",
	}
	err = snapshotRepo.Create(ctx, snapshot)
	require.NoError(t, err, "Create snapshot should not return error")

	err = labelRepo.SetLabel(ctx, asset.ID, snapshot.ID, "prod")
	require.NoError(t, err, "SetLabel should not return error")

	labels, err := labelRepo.GetLabelsByAssetID(ctx, asset.ID.String())
	require.NoError(t, err, "GetLabelsByAssetID should not return error")
	require.Len(t, labels, 1, "Should have 1 label")

	// Delete label
	err = labelRepo.Delete(ctx, labels[0].ID.String())
	require.NoError(t, err, "Delete should not return error")

	labels, err = labelRepo.GetLabelsByAssetID(ctx, asset.ID.String())
	require.NoError(t, err, "GetLabelsByAssetID should not return error")
	require.Len(t, labels, 0, "Should have 0 labels after delete")
}
