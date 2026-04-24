package storage

import (
	"context"
	"testing"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent/enttest"
)

func TestAuditLogRepository_Create(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAuditLogRepository(&Client{ent: client})
	ctx := context.Background()

	log := &domain.AuditLog{
		ID:        domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Operation: "AssetCreated",
		AssetID:   domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		UserID:    domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAX"),
		Details:   map[string]any{"key": "value"},
	}

	err := repo.Create(ctx, log)
	if err != nil {
		t.Fatalf("failed to create audit log: %v", err)
	}

	got, err := repo.GetByID(ctx, log.ID.String())
	if err != nil {
		t.Fatalf("failed to get audit log: %v", err)
	}
	if got.Operation != log.Operation {
		t.Errorf("expected operation %q, got %q", log.Operation, got.Operation)
	}
}

func TestAuditLogRepository_GetByAssetID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAuditLogRepository(&Client{ent: client})
	ctx := context.Background()

	assetID := domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW")

	// Create multiple logs for same asset
	for i := 0; i < 3; i++ {
		log := &domain.AuditLog{
			ID:        domain.NewAutoID(),
			Operation: "AssetUpdated",
			AssetID:   assetID,
			UserID:    domain.NewAutoID(),
			Details:   map[string]any{"version": i},
		}
		err := repo.Create(ctx, log)
		if err != nil {
			t.Fatalf("failed to create audit log %d: %v", i, err)
		}
	}

	logs, err := repo.GetByAssetID(ctx, assetID.String())
	if err != nil {
		t.Fatalf("failed to get audit logs by asset ID: %v", err)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 audit logs, got %d", len(logs))
	}
}

func TestAuditLogRepository_List(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAuditLogRepository(&Client{ent: client})
	ctx := context.Background()

	// Create multiple logs
	for i := 0; i < 5; i++ {
		log := &domain.AuditLog{
			ID:        domain.NewAutoID(),
			Operation: "TestOp",
			AssetID:   domain.NewAutoID(),
			UserID:    domain.NewAutoID(),
			Details:   map[string]any{},
		}
		err := repo.Create(ctx, log)
		if err != nil {
			t.Fatalf("failed to create audit log %d: %v", i, err)
		}
	}

	logs, total, err := repo.List(ctx, 0, 10)
	if err != nil {
		t.Fatalf("failed to list audit logs: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(logs) != 5 {
		t.Errorf("expected 5 audit logs, got %d", len(logs))
	}
}

func TestAuditLogRepository_Delete(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewAuditLogRepository(&Client{ent: client})
	ctx := context.Background()

	log := &domain.AuditLog{
		ID:        domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Operation: "AssetDeleted",
		AssetID:   domain.NewAutoID(),
		UserID:    domain.NewAutoID(),
		Details:   map[string]any{},
	}
	err := repo.Create(ctx, log)
	if err != nil {
		t.Fatalf("failed to create audit log: %v", err)
	}

	err = repo.Delete(ctx, log.ID.String())
	if err != nil {
		t.Fatalf("failed to delete audit log: %v", err)
	}

	_, err = repo.GetByID(ctx, log.ID.String())
	if err == nil {
		t.Error("expected error when getting deleted audit log")
	}
}