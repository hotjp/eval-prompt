package storage

import (
	"context"
	"testing"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent/enttest"
)

func TestEvalCaseRepository_Create(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalCaseRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	// Create parent asset
	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "Test",
		BizLine:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	evalCase := &domain.EvalCase{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		AssetID:        asset.ID,
		Name:           "Test Case",
		Prompt:         "Test prompt",
		ShouldTrigger:  true,
		ExpectedOutput: "Expected output",
		Rubric: domain.Rubric{
			MaxScore: 100,
			Checks: []domain.RubricCheck{
				{ID: "check1", Description: "Check 1", Weight: 50},
				{ID: "check2", Description: "Check 2", Weight: 50},
			},
		},
	}

	err = repo.Create(ctx, evalCase)
	if err != nil {
		t.Fatalf("failed to create eval case: %v", err)
	}

	got, err := repo.GetByID(ctx, evalCase.ID.String())
	if err != nil {
		t.Fatalf("failed to get eval case: %v", err)
	}
	if got.Name != evalCase.Name {
		t.Errorf("expected name %q, got %q", evalCase.Name, got.Name)
	}
	if got.Prompt != evalCase.Prompt {
		t.Errorf("expected prompt %q, got %q", evalCase.Prompt, got.Prompt)
	}
}

func TestEvalCaseRepository_GetByID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalCaseRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "Test",
		BizLine:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	evalCase := &domain.EvalCase{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		AssetID:        asset.ID,
		Name:           "Get Test",
		Prompt:         "Test prompt",
		ShouldTrigger:  true,
		ExpectedOutput: "Expected",
		Rubric: domain.Rubric{
			MaxScore: 100,
			Checks:   []domain.RubricCheck{},
		},
	}
	err = repo.Create(ctx, evalCase)
	if err != nil {
		t.Fatalf("failed to create eval case: %v", err)
	}

	got, err := repo.GetByID(ctx, evalCase.ID.String())
	if err != nil {
		t.Fatalf("failed to get eval case: %v", err)
	}
	if got.ID.String() != evalCase.ID.String() {
		t.Errorf("expected ID %q, got %q", evalCase.ID.String(), got.ID.String())
	}
}

func TestEvalCaseRepository_GetByAssetID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalCaseRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "Test",
		BizLine:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Create multiple eval cases
	for i := 0; i < 3; i++ {
		evalCase := &domain.EvalCase{
			ID:             domain.NewAutoID(),
			AssetID:        asset.ID,
			Name:           "Case",
			Prompt:         "Test",
			ShouldTrigger:  true,
			ExpectedOutput: "Expected",
			Rubric: domain.Rubric{
				MaxScore: 100,
				Checks:   []domain.RubricCheck{},
			},
		}
		err = repo.Create(ctx, evalCase)
		if err != nil {
			t.Fatalf("failed to create eval case %d: %v", i, err)
		}
	}

	cases, err := repo.GetByAssetID(ctx, asset.ID.String())
	if err != nil {
		t.Fatalf("failed to get eval cases by asset ID: %v", err)
	}
	if len(cases) != 3 {
		t.Errorf("expected 3 eval cases, got %d", len(cases))
	}
}

func TestEvalCaseRepository_List(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalCaseRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "Test",
		BizLine:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Create eval cases
	for i := 0; i < 5; i++ {
		evalCase := &domain.EvalCase{
			ID:             domain.NewAutoID(),
			AssetID:        asset.ID,
			Name:           "Case",
			Prompt:         "Test",
			ShouldTrigger:  true,
			ExpectedOutput: "Expected",
			Rubric: domain.Rubric{
				MaxScore: 100,
				Checks:   []domain.RubricCheck{},
			},
		}
		err = repo.Create(ctx, evalCase)
		if err != nil {
			t.Fatalf("failed to create eval case %d: %v", i, err)
		}
	}

	cases, total, err := repo.List(ctx, 0, 10)
	if err != nil {
		t.Fatalf("failed to list eval cases: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(cases) != 5 {
		t.Errorf("expected 5 eval cases, got %d", len(cases))
	}
}

func TestEvalCaseRepository_Update(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalCaseRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "Test",
		BizLine:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	evalCase := &domain.EvalCase{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		AssetID:        asset.ID,
		Name:           "Original Name",
		Prompt:         "Original prompt",
		ShouldTrigger:  true,
		ExpectedOutput: "Original",
		Rubric: domain.Rubric{
			MaxScore: 100,
			Checks:   []domain.RubricCheck{},
		},
	}
	err = repo.Create(ctx, evalCase)
	if err != nil {
		t.Fatalf("failed to create eval case: %v", err)
	}

	evalCase.Name = "Updated Name"
	evalCase.Prompt = "Updated prompt"
	err = repo.Update(ctx, evalCase)
	if err != nil {
		t.Fatalf("failed to update eval case: %v", err)
	}

	got, err := repo.GetByID(ctx, evalCase.ID.String())
	if err != nil {
		t.Fatalf("failed to get eval case: %v", err)
	}
	if got.Name != "Updated Name" {
		t.Errorf("expected name %q, got %q", "Updated Name", got.Name)
	}
	if got.Prompt != "Updated prompt" {
		t.Errorf("expected prompt %q, got %q", "Updated prompt", got.Prompt)
	}
}

func TestEvalCaseRepository_Delete(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalCaseRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	ctx := context.Background()

	asset := &domain.Asset{
		ID:          domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		Name:        "Test Asset",
		Description: "Test",
		BizLine:     "test",
		Tags:        []string{"test"},
		ContentHash: "abc123",
		FilePath:    "/prompts/test.md",
		State:       domain.AssetStateCreated,
	}
	err := assetRepo.Create(ctx, asset)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	evalCase := &domain.EvalCase{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		AssetID:        asset.ID,
		Name:           "Delete Test",
		Prompt:         "Test",
		ShouldTrigger:  true,
		ExpectedOutput: "Expected",
		Rubric: domain.Rubric{
			MaxScore: 100,
			Checks:   []domain.RubricCheck{},
		},
	}
	err = repo.Create(ctx, evalCase)
	if err != nil {
		t.Fatalf("failed to create eval case: %v", err)
	}

	err = repo.Delete(ctx, evalCase.ID.String())
	if err != nil {
		t.Fatalf("failed to delete eval case: %v", err)
	}

	_, err = repo.GetByID(ctx, evalCase.ID.String())
	if err == nil {
		t.Error("expected error when getting deleted eval case")
	}
}