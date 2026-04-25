package storage

import (
	"context"
	"testing"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent/enttest"
)

func TestAssetRepository_Create(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "A test asset",
		BizLine:     "test",
		Tags:        []string{"test", "unit"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}

	err := repo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Verify created
	got, err := repo.GetByID(ctx, asset.ID.String())
	if err != nil {
		t.Fatalf("failed to get asset: %v", err)
	}
	if got.Name != asset.Name {
		t.Errorf("expected name %q, got %q", asset.Name, got.Name)
	}
	if got.Description != asset.Description {
		t.Errorf("expected description %q, got %q", asset.Description, got.Description)
	}
	// BizLine is not stored in database in V1.1
}

func TestAssetRepository_GetByID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Get Test",
		Description: "Test get by ID",
		BizLine:     "test",
		Tags:        []string{"test"},
		ContentHash: "hash123",
		FilePath:    "/prompts/get.md",
		State:       domain.AssetStateCreated,
	}

	err := repo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	got, err := repo.GetByID(ctx, asset.ID.String())
	if err != nil {
		t.Fatalf("failed to get asset by ID: %v", err)
	}
	if got.ID.String() != asset.ID.String() {
		t.Errorf("expected ID %q, got %q", asset.ID.String(), got.ID.String())
	}
}

func TestAssetRepository_Update(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Original Name",
		Description: "Original description",
		BizLine:     "test",
		Tags:        []string{"test"},
		ContentHash: "hash123",
		FilePath:    "/prompts/update.md",
		State:       domain.AssetStateCreated,
	}

	err := repo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Update
	asset.Name = "Updated Name"
	asset.Description = "Updated description"
	err = repo.Update(ctx, asset)
	if err != nil {
		t.Fatalf("failed to update asset: %v", err)
	}

	got, err := repo.GetByID(ctx, asset.ID.String())
	if err != nil {
		t.Fatalf("failed to get updated asset: %v", err)
	}
	if got.Name != "Updated Name" {
		t.Errorf("expected name %q, got %q", "Updated Name", got.Name)
	}
}

func TestAssetRepository_Delete(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Delete Test",
		Description: "Test delete",
		BizLine:     "test",
		Tags:        []string{"test"},
		ContentHash: "hash123",
		FilePath:    "/prompts/delete.md",
		State:       domain.AssetStateCreated,
	}

	err := repo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	err = repo.Delete(ctx, asset.ID.String())
	if err != nil {
		t.Fatalf("failed to delete asset: %v", err)
	}

	_, err = repo.GetByID(ctx, asset.ID.String())
	if err == nil {
		t.Error("expected error when getting deleted asset")
	}
}

func TestAssetRepository_List(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	// Create multiple assets
	for i := 0; i < 5; i++ {
		asset := &domain.Asset{
			ID:          domain.NewAutoID(),
			Name:        "Asset",
			Description: "Test",
			BizLine:     "test",
			Tags:        []string{"test"},
			ContentHash: "hash",
			FilePath:    "/prompts/asset.md",
			State:       domain.AssetStateCreated,
		}
		err := repo.Create(ctx, asset)
		if err != nil {
			t.Fatalf("failed to create asset: %v", err)
		}
	}

	assets, total, err := repo.List(ctx, 0, 10)
	if err != nil {
		t.Fatalf("failed to list assets: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(assets) != 5 {
		t.Errorf("expected 5 assets, got %d", len(assets))
	}

	// Test pagination
	assets, total, err = repo.List(ctx, 0, 2)
	if err != nil {
		t.Fatalf("failed to list assets with limit: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(assets))
	}
}

func TestAssetRepository_ListByBizLine(t *testing.T) {
	t.Skip("biz_line is deprecated in V1.1 schema")
}

func TestAssetRepository_ListByState(t *testing.T) {
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
			BizLine:     "test",
			Tags:        []string{"test"},
			ContentHash: "hash",
			FilePath:    "/prompts/asset.md",
			State:       state,
		}
		err := repo.Create(ctx, asset)
		if err != nil {
			t.Fatalf("failed to create asset %d: %v", i, err)
		}
	}

	assets, total, err := repo.ListByState(ctx, domain.AssetStateCreated, 0, 10)
	if err != nil {
		t.Fatalf("failed to list by state: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2 for Created state, got %d", total)
	}
	if len(assets) != 2 {
		t.Errorf("expected 2 assets for Created state, got %d", len(assets))
	}
}

func TestAssetRepository_UpdateState(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "State Test",
		Description: "Test state update",
		BizLine:     "test",
		Tags:        []string{"test"},
		ContentHash: "hash",
		FilePath:    "/prompts/state.md",
		State:       domain.AssetStateCreated,
	}

	err := repo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	err = repo.UpdateState(ctx, asset.ID.String(), domain.AssetStateEvaluated)
	if err != nil {
		t.Fatalf("failed to update state: %v", err)
	}

	got, err := repo.GetByID(ctx, asset.ID.String())
	if err != nil {
		t.Fatalf("failed to get asset: %v", err)
	}
	if got.State != domain.AssetStateEvaluated {
		t.Errorf("expected state %q, got %q", domain.AssetStateEvaluated, got.State)
	}
}