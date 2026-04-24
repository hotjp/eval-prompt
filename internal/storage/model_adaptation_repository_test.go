package storage

import (
	"context"
	"testing"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent/enttest"
)

func TestModelAdaptationRepository_Create(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewModelAdaptationRepository(&Client{ent: client})
	ctx := context.Background()

	adaptation := &domain.ModelAdaptation{
		ID:               domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		PromptID:         domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		SourceModel:      "gpt-4",
		TargetModel:      "claude-3-opus",
		AdaptedContent:   "Adapted prompt content",
		ParamAdjustments: map[string]float64{"temperature": 0.7, "max_tokens": 2000},
		FormatChanges:    []string{"xml_format", "add_few_shot"},
		EvalScore:        0.85,
		EvalRunID:        domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAX"),
	}

	err := repo.Create(ctx, adaptation)
	if err != nil {
		t.Fatalf("failed to create model adaptation: %v", err)
	}

	got, err := repo.GetByID(ctx, adaptation.ID.String())
	if err != nil {
		t.Fatalf("failed to get model adaptation: %v", err)
	}
	if got.SourceModel != adaptation.SourceModel {
		t.Errorf("expected source model %q, got %q", adaptation.SourceModel, got.SourceModel)
	}
	if got.TargetModel != adaptation.TargetModel {
		t.Errorf("expected target model %q, got %q", adaptation.TargetModel, got.TargetModel)
	}
	if got.AdaptedContent != adaptation.AdaptedContent {
		t.Errorf("expected adapted content %q, got %q", adaptation.AdaptedContent, got.AdaptedContent)
	}
}

func TestModelAdaptationRepository_GetByPromptID(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewModelAdaptationRepository(&Client{ent: client})
	ctx := context.Background()

	promptID := domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW")

	// Create multiple adaptations for same prompt
	for i := 0; i < 3; i++ {
		adaptation := &domain.ModelAdaptation{
			ID:               domain.NewAutoID(),
			PromptID:         promptID,
			SourceModel:      "gpt-4",
			TargetModel:      "claude-3-" + string(rune('0'+i)),
			AdaptedContent:   "Content",
			ParamAdjustments: map[string]float64{},
			FormatChanges:    []string{},
			EvalScore:        0.8 + float64(i)*0.05,
			EvalRunID:        domain.NewAutoID(),
		}
		err := repo.Create(ctx, adaptation)
		if err != nil {
			t.Fatalf("failed to create adaptation %d: %v", i, err)
		}
	}

	adaptations, err := repo.GetByPromptID(ctx, promptID.String())
	if err != nil {
		t.Fatalf("failed to get adaptations by prompt ID: %v", err)
	}
	if len(adaptations) != 3 {
		t.Errorf("expected 3 adaptations, got %d", len(adaptations))
	}
}

func TestModelAdaptationRepository_GetByTargetModel(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewModelAdaptationRepository(&Client{ent: client})
	ctx := context.Background()

	targetModel := "claude-3-opus"

	// Create adaptations with different target models
	for i := 0; i < 2; i++ {
		adaptation := &domain.ModelAdaptation{
			ID:               domain.NewAutoID(),
			PromptID:         domain.NewAutoID(),
			SourceModel:      "gpt-4",
			TargetModel:      targetModel,
			AdaptedContent:   "Content",
			ParamAdjustments: map[string]float64{},
			FormatChanges:    []string{},
			EvalScore:        0.85,
			EvalRunID:        domain.NewAutoID(),
		}
		err := repo.Create(ctx, adaptation)
		if err != nil {
			t.Fatalf("failed to create adaptation %d: %v", i, err)
		}
	}

	adaptations, err := repo.GetByTargetModel(ctx, targetModel)
	if err != nil {
		t.Fatalf("failed to get adaptations by target model: %v", err)
	}
	if len(adaptations) != 2 {
		t.Errorf("expected 2 adaptations, got %d", len(adaptations))
	}
}

func TestModelAdaptationRepository_Update(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewModelAdaptationRepository(&Client{ent: client})
	ctx := context.Background()

	adaptation := &domain.ModelAdaptation{
		ID:               domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		PromptID:         domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		SourceModel:      "gpt-4",
		TargetModel:      "claude-3-opus",
		AdaptedContent:   "Original content",
		ParamAdjustments: map[string]float64{},
		FormatChanges:    []string{},
		EvalScore:        0.75,
		EvalRunID:        domain.NewAutoID(),
	}
	err := repo.Create(ctx, adaptation)
	if err != nil {
		t.Fatalf("failed to create adaptation: %v", err)
	}

	adaptation.AdaptedContent = "Updated content"
	adaptation.EvalScore = 0.92
	err = repo.Update(ctx, adaptation)
	if err != nil {
		t.Fatalf("failed to update adaptation: %v", err)
	}

	got, err := repo.GetByID(ctx, adaptation.ID.String())
	if err != nil {
		t.Fatalf("failed to get adaptation: %v", err)
	}
	if got.AdaptedContent != "Updated content" {
		t.Errorf("expected adapted content %q, got %q", "Updated content", got.AdaptedContent)
	}
	if got.EvalScore != 0.92 {
		t.Errorf("expected eval score 0.92, got %f", got.EvalScore)
	}
}

func TestModelAdaptationRepository_Delete(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file::memory:?_fk=1&_journal_mode=WAL")
	defer client.Close()

	repo := NewModelAdaptationRepository(&Client{ent: client})
	ctx := context.Background()

	adaptation := &domain.ModelAdaptation{
		ID:               domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		PromptID:         domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
		SourceModel:      "gpt-4",
		TargetModel:      "claude-3-opus",
		AdaptedContent:   "Content",
		ParamAdjustments: map[string]float64{},
		FormatChanges:    []string{},
		EvalScore:        0.85,
		EvalRunID:        domain.NewAutoID(),
	}
	err := repo.Create(ctx, adaptation)
	if err != nil {
		t.Fatalf("failed to create adaptation: %v", err)
	}

	err = repo.Delete(ctx, adaptation.ID.String())
	if err != nil {
		t.Fatalf("failed to delete adaptation: %v", err)
	}

	_, err = repo.GetByID(ctx, adaptation.ID.String())
	if err == nil {
		t.Error("expected error when getting deleted adaptation")
	}
}