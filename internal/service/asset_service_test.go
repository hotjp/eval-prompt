package service

import (
	"context"
	"testing"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage"
	"github.com/eval-prompt/internal/storage/ent/enttest"
)

// newStorageClientForTest creates a storage client with in-memory SQLite for testing.
func newStorageClientForTest(t *testing.T) *storage.Client {
	t.Helper()
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	return storage.NewClientForTest(client)
}

func TestAssetService_CreateAsset(t *testing.T) {
	svc := NewAssetService(newStorageClientForTest(t))
	ctx := context.Background()

	req := &CreateAssetRequest{
		Name:        "Test Asset",
		Description: "A test asset",
		BizLine:     "test",
		Tags:        []string{"test", "unit"},
		FilePath:    "/prompts/test.md",
		ContentHash: "abc123",
		Author:      "tester",
	}

	resp, err := svc.CreateAsset(ctx, req)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	if resp.Name != req.Name {
		t.Errorf("expected name %q, got %q", req.Name, resp.Name)
	}
	if resp.State != string(domain.AssetStateCreated) {
		t.Errorf("expected state %q, got %q", domain.AssetStateCreated, resp.State)
	}
	if resp.Snapshot == nil {
		t.Fatal("expected snapshot to be created")
	}
	if resp.Snapshot.Version != "v0.0.0" {
		t.Errorf("expected initial version v0.0.0, got %s", resp.Snapshot.Version)
	}
}

func TestAssetService_CreateAsset_ValidationError(t *testing.T) {
	svc := NewAssetService(newStorageClientForTest(t))
	ctx := context.Background()

	tests := []struct {
		name string
		req  *CreateAssetRequest
	}{
		{
			name: "missing name",
			req: &CreateAssetRequest{
				Name:        "",
				FilePath:    "/prompts/test.md",
				ContentHash: "abc123",
			},
		},
		{
			name: "missing file path",
			req: &CreateAssetRequest{
				Name:        "Test",
				FilePath:    "",
				ContentHash: "abc123",
			},
		},
		{
			name: "missing content hash",
			req: &CreateAssetRequest{
				Name:        "Test",
				FilePath:    "/prompts/test.md",
				ContentHash: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateAsset(ctx, tt.req)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestAssetService_UpdateAsset(t *testing.T) {
	svc := NewAssetService(newStorageClientForTest(t))
	ctx := context.Background()

	// Create initial asset
	createResp, err := svc.CreateAsset(ctx, &CreateAssetRequest{
		Name:        "Original Name",
		Description: "Original description",
		BizLine:     "test",
		Tags:        []string{"test"},
		FilePath:    "/prompts/test.md",
		ContentHash: "originalhash",
		Author:      "tester",
	})
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Update the asset
	updateResp, err := svc.UpdateAsset(ctx, &UpdateAssetRequest{
		ID:          createResp.ID,
		Name:        "Updated Name",
		Description: "Updated description",
		Tags:        []string{"test", "updated"},
		ContentHash: "newhash",
		Author:      "tester",
		Reason:      "Updated content",
	})
	if err != nil {
		t.Fatalf("failed to update asset: %v", err)
	}

	if updateResp.Name != "Updated Name" {
		t.Errorf("expected name %q, got %q", "Updated Name", updateResp.Name)
	}
	if updateResp.Snapshot == nil {
		t.Fatal("expected new snapshot")
	}
}

func TestAssetService_UpdateAsset_UnchangedHash(t *testing.T) {
	svc := NewAssetService(newStorageClientForTest(t))
	ctx := context.Background()

	// Create asset
	createResp, err := svc.CreateAsset(ctx, &CreateAssetRequest{
		Name:        "Test",
		Description: "Test",
		BizLine:     "test",
		Tags:        []string{"test"},
		FilePath:    "/prompts/test.md",
		ContentHash: "samehash",
		Author:      "tester",
	})
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Try to update with same content hash
	_, err = svc.UpdateAsset(ctx, &UpdateAssetRequest{
		ID:          createResp.ID,
		ContentHash: "samehash",
		Author:      "tester",
		Reason:      "No change",
	})
	if err == nil {
		t.Error("expected error for unchanged content hash")
	}
}

func TestAssetService_GetAsset(t *testing.T) {
	svc := NewAssetService(newStorageClientForTest(t))
	ctx := context.Background()

	// Create asset
	createResp, err := svc.CreateAsset(ctx, &CreateAssetRequest{
		Name:        "Test Asset",
		Description: "Test description",
		BizLine:     "test",
		Tags:        []string{"test"},
		FilePath:    "/prompts/test.md",
		ContentHash: "abc123",
		Author:      "tester",
	})
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Get the asset
	detail, err := svc.GetAsset(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("failed to get asset: %v", err)
	}

	if detail.Name != "Test Asset" {
		t.Errorf("expected name %q, got %q", "Test Asset", detail.Name)
	}
	// In V1.1, snapshots are stored in files, not in the database
	if len(detail.Snapshots) != 0 {
		t.Errorf("expected 0 snapshots in V1.1 (stored in files), got %d", len(detail.Snapshots))
	}
}

func TestAssetService_GetAsset_NotFound(t *testing.T) {
	svc := NewAssetService(newStorageClientForTest(t))
	ctx := context.Background()

	_, err := svc.GetAsset(ctx, "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if err == nil {
		t.Error("expected error for nonexistent asset")
	}
}

func TestAssetService_ListAssets(t *testing.T) {
	svc := NewAssetService(newStorageClientForTest(t))
	ctx := context.Background()

	// Create multiple assets
	for i := 0; i < 5; i++ {
		_, err := svc.CreateAsset(ctx, &CreateAssetRequest{
			Name:        "Asset",
			Description: "Test",
			BizLine:     "test",
			Tags:        []string{"test"},
			FilePath:    "/prompts/asset.md",
			ContentHash: "hash",
			Author:      "tester",
		})
		if err != nil {
			t.Fatalf("failed to create asset %d: %v", i, err)
		}
	}

	resp, err := svc.ListAssets(ctx, &ListAssetsRequest{Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("failed to list assets: %v", err)
	}
	if resp.Total != 5 {
		t.Errorf("expected total 5, got %d", resp.Total)
	}
	if len(resp.Assets) != 5 {
		t.Errorf("expected 5 assets, got %d", len(resp.Assets))
	}
}

func TestAssetService_ListAssets_Pagination(t *testing.T) {
	svc := NewAssetService(newStorageClientForTest(t))
	ctx := context.Background()

	// Create assets
	for i := 0; i < 5; i++ {
		_, err := svc.CreateAsset(ctx, &CreateAssetRequest{
			Name:        "Asset",
			Description: "Test",
			BizLine:     "test",
			Tags:        []string{"test"},
			FilePath:    "/prompts/asset.md",
			ContentHash: "hash",
			Author:      "tester",
		})
		if err != nil {
			t.Fatalf("failed to create asset %d: %v", i, err)
		}
	}

	resp, err := svc.ListAssets(ctx, &ListAssetsRequest{Offset: 0, Limit: 2})
	if err != nil {
		t.Fatalf("failed to list assets: %v", err)
	}
	if resp.Total != 5 {
		t.Errorf("expected total 5, got %d", resp.Total)
	}
	if len(resp.Assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(resp.Assets))
	}
}

func TestAssetService_SetLabel(t *testing.T) {
	svc := NewAssetService(newStorageClientForTest(t))
	ctx := context.Background()

	// Create asset
	createResp, err := svc.CreateAsset(ctx, &CreateAssetRequest{
		Name:        "Test",
		Description: "Test",
		BizLine:     "test",
		Tags:        []string{"test"},
		FilePath:    "/prompts/test.md",
		ContentHash: "abc123",
		Author:      "tester",
	})
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Set label
	err = svc.SetLabel(ctx, &SetLabelRequest{
		AssetID:    createResp.ID,
		SnapshotID: createResp.Snapshot.ID,
		Name:       "prod",
	})
	if err != nil {
		t.Fatalf("failed to set label: %v", err)
	}

	// Verify label
	labels, err := svc.GetLabels(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("failed to get labels: %v", err)
	}
	if len(labels) != 1 {
		t.Errorf("expected 1 label, got %d", len(labels))
	}
	if labels[0].Name != "prod" {
		t.Errorf("expected label name %q, got %q", "prod", labels[0].Name)
	}
}

func TestAssetService_UnsetLabel(t *testing.T) {
	svc := NewAssetService(newStorageClientForTest(t))
	ctx := context.Background()

	// Create asset
	createResp, err := svc.CreateAsset(ctx, &CreateAssetRequest{
		Name:        "Test",
		Description: "Test",
		BizLine:     "test",
		Tags:        []string{"test"},
		FilePath:    "/prompts/test.md",
		ContentHash: "abc123",
		Author:      "tester",
	})
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Set label
	err = svc.SetLabel(ctx, &SetLabelRequest{
		AssetID:    createResp.ID,
		SnapshotID: createResp.Snapshot.ID,
		Name:       "prod",
	})
	if err != nil {
		t.Fatalf("failed to set label: %v", err)
	}

	// Unset label
	err = svc.UnsetLabel(ctx, &UnsetLabelRequest{
		AssetID: createResp.ID,
		Name:    "prod",
	})
	if err != nil {
		t.Fatalf("failed to unset label: %v", err)
	}

	// Verify no labels
	labels, err := svc.GetLabels(ctx, createResp.ID)
	if err != nil {
		t.Fatalf("failed to get labels: %v", err)
	}
	if len(labels) != 0 {
		t.Errorf("expected 0 labels, got %d", len(labels))
	}
}