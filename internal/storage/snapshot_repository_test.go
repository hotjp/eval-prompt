package storage

import (
	"context"
	"testing"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent/enttest"
)

func TestSnapshotRepository_Create(t *testing.T) {
	t.Skip("SnapshotRepository is deprecated in V1.1 - snapshot stored in files")
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	// Create parent asset first
	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "A test asset",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	snapshot := &domain.Snapshot{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		AssetID:     asset.ID,
		Version:     "v1.0.0",
		ContentHash: "hash123",
		CommitHash:  "abc123def456",
		Author:      "tester",
		Reason:      "Initial version",
		CreatedAt:   asset.CreatedAt,
	}

	err = snapshotRepo.Create(ctx, snapshot)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	got, err := snapshotRepo.GetByID(ctx, snapshot.ID.String())
	if err != nil {
		t.Fatalf("failed to get snapshot: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil (deprecated), got %+v", got)
	}
}

func TestSnapshotRepository_GetByAssetID(t *testing.T) {
	t.Skip("SnapshotRepository is deprecated in V1.1 - snapshots stored in files")
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "A test asset",
		AssetType:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Create multiple snapshots
	for i := 0; i < 3; i++ {
		snapshot := &domain.Snapshot{
			ID:          domain.NewAutoID(),
			AssetID:     asset.ID,
			Version:     "v1.0." + string(rune('0'+i)),
			ContentHash: "hash",
			Author:      "tester",
			Reason:     "Test version",
		}
		err = snapshotRepo.Create(ctx, snapshot)
		if err != nil {
			t.Fatalf("failed to create snapshot %d: %v", i, err)
		}
	}

	snapshots, err := snapshotRepo.GetByAssetID(ctx, asset.ID.String())
	if err != nil {
		t.Fatalf("failed to get snapshots by asset ID: %v", err)
	}
	if len(snapshots) != 3 {
		t.Errorf("expected 3 snapshots, got %d", len(snapshots))
	}
}

func TestSnapshotRepository_GetByCommitHash(t *testing.T) {
	t.Skip("SnapshotRepository is deprecated in V1.1 - snapshots stored in files")
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "A test asset",
		AssetType:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	commitHash := "abc123def456789"
	snapshot := &domain.Snapshot{
		ID:          domain.NewAutoID(),
		AssetID:     asset.ID,
		Version:     "v1.0.0",
		ContentHash: "hash123",
		CommitHash:  commitHash,
		Author:      "tester",
		Reason:     "Initial",
	}
	err = snapshotRepo.Create(ctx, snapshot)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	got, err := snapshotRepo.GetByCommitHash(ctx, commitHash)
	if err != nil {
		t.Fatalf("failed to get snapshot by commit hash: %v", err)
	}
	if got == nil {
		t.Fatal("expected snapshot, got nil")
	}
	if got.CommitHash != commitHash {
		t.Errorf("expected commit hash %q, got %q", commitHash, got.CommitHash)
	}
}

func TestSnapshotRepository_List(t *testing.T) {
	t.Skip("SnapshotRepository is deprecated in V1.1 - snapshots stored in files")
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "A test asset",
		AssetType:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Create snapshots
	for i := 0; i < 5; i++ {
		snapshot := &domain.Snapshot{
			ID:          domain.NewAutoID(),
			AssetID:     asset.ID,
			Version:     "v1.0." + string(rune('0'+i)),
			ContentHash: "hash",
			Author:      "tester",
			Reason:     "Test",
		}
		err = snapshotRepo.Create(ctx, snapshot)
		if err != nil {
			t.Fatalf("failed to create snapshot %d: %v", i, err)
		}
	}

	snapshots, total, err := snapshotRepo.List(ctx, 0, 10)
	if err != nil {
		t.Fatalf("failed to list snapshots: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(snapshots) != 5 {
		t.Errorf("expected 5 snapshots, got %d", len(snapshots))
	}
}

func TestSnapshotRepository_Update(t *testing.T) {
	t.Skip("SnapshotRepository is deprecated in V1.1 - snapshots stored in files")
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "A test asset",
		AssetType:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	snapshot := &domain.Snapshot{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		AssetID:     asset.ID,
		Version:     "v1.0.0",
		ContentHash: "hash123",
		Author:      "tester",
		Reason:     "Initial",
	}
	err = snapshotRepo.Create(ctx, snapshot)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	snapshot.Reason = "Updated reason"
	err = snapshotRepo.Update(ctx, snapshot)
	if err != nil {
		t.Fatalf("failed to update snapshot: %v", err)
	}

	got, err := snapshotRepo.GetByID(ctx, snapshot.ID.String())
	if err != nil {
		t.Fatalf("failed to get snapshot: %v", err)
	}
	if got.Reason != "Updated reason" {
		t.Errorf("expected reason %q, got %q", "Updated reason", got.Reason)
	}
}