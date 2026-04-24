package storage

import (
	"context"
	"testing"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent/enttest"
)

func TestEvalRunRepository_Create(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalRunRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	evalCaseRepo := NewEvalCaseRepository(&Client{ent: client})
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

	evalCase := &domain.EvalCase{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAX"),
		AssetID:        asset.ID,
		Name:           "Test Case",
		Prompt:         "Test prompt",
		ShouldTrigger:  true,
		ExpectedOutput: "Expected",
		Rubric: domain.Rubric{
			MaxScore: 100,
			Checks:   []domain.RubricCheck{},
		},
	}
	err = evalCaseRepo.Create(ctx, evalCase)
	if err != nil {
		t.Fatalf("failed to create eval case: %v", err)
	}

	evalRun := &domain.EvalRun{
		ID:                 domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAY"),
		EvalCaseID:         evalCase.ID,
		SnapshotID:         snapshot.ID,
		Status:             domain.EvalRunStatusPassed,
		DeterministicScore: 0.95,
		RubricScore:        85,
		TracePath:          "/traces/run1.jsonl",
		TokenInput:         100,
		TokenOutput:        200,
		DurationMs:         1500,
	}

	err = repo.Create(ctx, evalRun)
	if err != nil {
		t.Fatalf("failed to create eval run: %v", err)
	}

	got, err := repo.GetByID(ctx, evalRun.ID.String())
	if err != nil {
		t.Fatalf("failed to get eval run: %v", err)
	}
	if got.Status != domain.EvalRunStatusPassed {
		t.Errorf("expected status %q, got %q", domain.EvalRunStatusPassed, got.Status)
	}
	if got.RubricScore != 85 {
		t.Errorf("expected rubric score 85, got %d", got.RubricScore)
	}
}

func TestEvalRunRepository_GetByEvalCaseID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalRunRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	evalCaseRepo := NewEvalCaseRepository(&Client{ent: client})
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

	evalCase := &domain.EvalCase{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAX"),
		AssetID:        asset.ID,
		Name:           "Test Case",
		Prompt:         "Test",
		ShouldTrigger:  true,
		ExpectedOutput: "Expected",
		Rubric: domain.Rubric{
			MaxScore: 100,
			Checks:   []domain.RubricCheck{},
		},
	}
	err = evalCaseRepo.Create(ctx, evalCase)
	if err != nil {
		t.Fatalf("failed to create eval case: %v", err)
	}

	// Create multiple eval runs for same case
	for i := 0; i < 3; i++ {
		evalRun := &domain.EvalRun{
			ID:                 domain.NewAutoID(),
			EvalCaseID:         evalCase.ID,
			SnapshotID:         snapshot.ID,
			Status:             domain.EvalRunStatusPassed,
			DeterministicScore: 0.9,
			RubricScore:        80 + i*5,
		}
		err = repo.Create(ctx, evalRun)
		if err != nil {
			t.Fatalf("failed to create eval run %d: %v", i, err)
		}
	}

	runs, err := repo.GetByEvalCaseID(ctx, evalCase.ID.String())
	if err != nil {
		t.Fatalf("failed to get eval runs by case ID: %v", err)
	}
	if len(runs) != 3 {
		t.Errorf("expected 3 eval runs, got %d", len(runs))
	}
}

func TestEvalRunRepository_GetBySnapshotID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalRunRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	evalCaseRepo := NewEvalCaseRepository(&Client{ent: client})
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

	evalCase := &domain.EvalCase{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAX"),
		AssetID:        asset.ID,
		Name:           "Test Case",
		Prompt:         "Test",
		ShouldTrigger:  true,
		ExpectedOutput: "Expected",
		Rubric: domain.Rubric{
			MaxScore: 100,
			Checks:   []domain.RubricCheck{},
		},
	}
	err = evalCaseRepo.Create(ctx, evalCase)
	if err != nil {
		t.Fatalf("failed to create eval case: %v", err)
	}

	evalRun := &domain.EvalRun{
		ID:                 domain.NewAutoID(),
		EvalCaseID:         evalCase.ID,
		SnapshotID:         snapshot.ID,
		Status:             domain.EvalRunStatusPassed,
		DeterministicScore: 0.9,
		RubricScore:        85,
	}
	err = repo.Create(ctx, evalRun)
	if err != nil {
		t.Fatalf("failed to create eval run: %v", err)
	}

	runs, err := repo.GetBySnapshotID(ctx, snapshot.ID.String())
	if err != nil {
		t.Fatalf("failed to get eval runs by snapshot ID: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("expected 1 eval run, got %d", len(runs))
	}
}

func TestEvalRunRepository_List(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalRunRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	evalCaseRepo := NewEvalCaseRepository(&Client{ent: client})
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

	evalCase := &domain.EvalCase{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAX"),
		AssetID:        asset.ID,
		Name:           "Test Case",
		Prompt:         "Test",
		ShouldTrigger:  true,
		ExpectedOutput: "Expected",
		Rubric: domain.Rubric{
			MaxScore: 100,
			Checks:   []domain.RubricCheck{},
		},
	}
	err = evalCaseRepo.Create(ctx, evalCase)
	if err != nil {
		t.Fatalf("failed to create eval case: %v", err)
	}

	// Create multiple eval runs
	for i := 0; i < 5; i++ {
		evalRun := &domain.EvalRun{
			ID:                 domain.NewAutoID(),
			EvalCaseID:         evalCase.ID,
			SnapshotID:         snapshot.ID,
			Status:             domain.EvalRunStatusPassed,
			DeterministicScore: 0.9,
			RubricScore:        80 + i,
		}
		err = repo.Create(ctx, evalRun)
		if err != nil {
			t.Fatalf("failed to create eval run %d: %v", i, err)
		}
	}

	runs, total, err := repo.List(ctx, 0, 10)
	if err != nil {
		t.Fatalf("failed to list eval runs: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(runs) != 5 {
		t.Errorf("expected 5 eval runs, got %d", len(runs))
	}
}

func TestEvalRunRepository_ListByStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalRunRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	evalCaseRepo := NewEvalCaseRepository(&Client{ent: client})
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

	evalCase := &domain.EvalCase{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAX"),
		AssetID:        asset.ID,
		Name:           "Test Case",
		Prompt:         "Test",
		ShouldTrigger:  true,
		ExpectedOutput: "Expected",
		Rubric: domain.Rubric{
			MaxScore: 100,
			Checks:   []domain.RubricCheck{},
		},
	}
	err = evalCaseRepo.Create(ctx, evalCase)
	if err != nil {
		t.Fatalf("failed to create eval case: %v", err)
	}

	// Create eval runs with different statuses
	statuses := []domain.EvalRunStatus{
		domain.EvalRunStatusPassed,
		domain.EvalRunStatusFailed,
		domain.EvalRunStatusPassed,
	}
	for i, status := range statuses {
		evalRun := &domain.EvalRun{
			ID:                 domain.NewAutoID(),
			EvalCaseID:         evalCase.ID,
			SnapshotID:         snapshot.ID,
			Status:             status,
			DeterministicScore: 0.9,
			RubricScore:        80 + i,
		}
		err = repo.Create(ctx, evalRun)
		if err != nil {
			t.Fatalf("failed to create eval run %d: %v", i, err)
		}
	}

	runs, total, err := repo.ListByStatus(ctx, domain.EvalRunStatusPassed, 0, 10)
	if err != nil {
		t.Fatalf("failed to list eval runs by status: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 passed eval runs, got %d", total)
	}
	if len(runs) != 2 {
		t.Errorf("expected 2 eval runs, got %d", len(runs))
	}
}

func TestEvalRunRepository_UpdateStatus(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalRunRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	evalCaseRepo := NewEvalCaseRepository(&Client{ent: client})
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

	evalCase := &domain.EvalCase{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAX"),
		AssetID:        asset.ID,
		Name:           "Test Case",
		Prompt:         "Test",
		ShouldTrigger:  true,
		ExpectedOutput: "Expected",
		Rubric: domain.Rubric{
			MaxScore: 100,
			Checks:   []domain.RubricCheck{},
		},
	}
	err = evalCaseRepo.Create(ctx, evalCase)
	if err != nil {
		t.Fatalf("failed to create eval case: %v", err)
	}

	evalRun := &domain.EvalRun{
		ID:                 domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAY"),
		EvalCaseID:         evalCase.ID,
		SnapshotID:         snapshot.ID,
		Status:             domain.EvalRunStatusPending,
		DeterministicScore: 0.9,
		RubricScore:        85,
	}
	err = repo.Create(ctx, evalRun)
	if err != nil {
		t.Fatalf("failed to create eval run: %v", err)
	}

	err = repo.UpdateStatus(ctx, evalRun.ID.String(), domain.EvalRunStatusPassed)
	if err != nil {
		t.Fatalf("failed to update status: %v", err)
	}

	got, err := repo.GetByID(ctx, evalRun.ID.String())
	if err != nil {
		t.Fatalf("failed to get eval run: %v", err)
	}
	if got.Status != domain.EvalRunStatusPassed {
		t.Errorf("expected status %q, got %q", domain.EvalRunStatusPassed, got.Status)
	}
}

func TestEvalRunRepository_Delete(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewEvalRunRepository(&Client{ent: client})
	assetRepo := NewAssetRepository(&Client{ent: client})
	snapshotRepo := NewSnapshotRepository(&Client{ent: client})
	evalCaseRepo := NewEvalCaseRepository(&Client{ent: client})
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

	evalCase := &domain.EvalCase{
		ID:             domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAX"),
		AssetID:        asset.ID,
		Name:           "Test Case",
		Prompt:         "Test",
		ShouldTrigger:  true,
		ExpectedOutput: "Expected",
		Rubric: domain.Rubric{
			MaxScore: 100,
			Checks:   []domain.RubricCheck{},
		},
	}
	err = evalCaseRepo.Create(ctx, evalCase)
	if err != nil {
		t.Fatalf("failed to create eval case: %v", err)
	}

	evalRun := &domain.EvalRun{
		ID:                 domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAY"),
		EvalCaseID:         evalCase.ID,
		SnapshotID:         snapshot.ID,
		Status:             domain.EvalRunStatusPassed,
		DeterministicScore: 0.9,
		RubricScore:        85,
	}
	err = repo.Create(ctx, evalRun)
	if err != nil {
		t.Fatalf("failed to create eval run: %v", err)
	}

	err = repo.Delete(ctx, evalRun.ID.String())
	if err != nil {
		t.Fatalf("failed to delete eval run: %v", err)
	}

	_, err = repo.GetByID(ctx, evalRun.ID.String())
	if err == nil {
		t.Error("expected error when getting deleted eval run")
	}
}