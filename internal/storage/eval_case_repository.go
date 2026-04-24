package storage

import (
	"context"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent"
	"github.com/eval-prompt/internal/storage/ent/asset"
	"github.com/eval-prompt/internal/storage/ent/evalcase"
	"github.com/eval-prompt/internal/storage/ent/schema"
)

// EvalCaseRepository provides repository operations for EvalCase entities.
type EvalCaseRepository struct {
	client *Client
}

// NewEvalCaseRepository creates a new EvalCaseRepository.
func NewEvalCaseRepository(client *Client) *EvalCaseRepository {
	return &EvalCaseRepository{client: client}
}

// Create creates a new eval case in the database.
func (r *EvalCaseRepository) Create(ctx context.Context, e *domain.EvalCase) error {
	entRubric := schema.Rubric{
		MaxScore: e.Rubric.MaxScore,
		Checks:   make([]schema.RubricCheck, len(e.Rubric.Checks)),
	}
	for i, c := range e.Rubric.Checks {
		entRubric.Checks[i] = schema.RubricCheck{
			ID:          c.ID,
			Description: c.Description,
			Weight:      c.Weight,
		}
	}

	_, err := r.client.ent.EvalCase.Create().
		SetID(e.ID.String()).
		SetAssetID(e.AssetID.String()).
		SetName(e.Name).
		SetPrompt(e.Prompt).
		SetShouldTrigger(e.ShouldTrigger).
		SetExpectedOutput(e.ExpectedOutput).
		SetRubric(entRubric).
		SetCreatedAt(e.CreatedAt).
		Save(ctx)
	return err
}

// GetByID retrieves an eval case by its ID.
func (r *EvalCaseRepository) GetByID(ctx context.Context, id string) (*domain.EvalCase, error) {
	entCase, err := r.client.ent.EvalCase.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.toDomainEvalCase(entCase), nil
}

// GetByAssetID retrieves all eval cases for an asset.
func (r *EvalCaseRepository) GetByAssetID(ctx context.Context, assetID string) ([]*domain.EvalCase, error) {
	entCases, err := r.client.ent.EvalCase.Query().
		Where(evalcase.HasAssetWith(asset.IDEQ(assetID))).
		All(ctx)
	if err != nil {
		return nil, err
	}

	cases := make([]*domain.EvalCase, len(entCases))
	for i, entCase := range entCases {
		cases[i] = r.toDomainEvalCase(entCase)
	}
	return cases, nil
}

// List retrieves eval cases with pagination.
func (r *EvalCaseRepository) List(ctx context.Context, offset, limit int) ([]*domain.EvalCase, int, error) {
	entCases, err := r.client.ent.EvalCase.Query().
		Offset(offset).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.client.ent.EvalCase.Query().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	cases := make([]*domain.EvalCase, len(entCases))
	for i, entCase := range entCases {
		cases[i] = r.toDomainEvalCase(entCase)
	}
	return cases, total, nil
}

// Update updates an existing eval case.
func (r *EvalCaseRepository) Update(ctx context.Context, e *domain.EvalCase) error {
	entRubric := schema.Rubric{
		MaxScore: e.Rubric.MaxScore,
		Checks:   make([]schema.RubricCheck, len(e.Rubric.Checks)),
	}
	for i, c := range e.Rubric.Checks {
		entRubric.Checks[i] = schema.RubricCheck{
			ID:          c.ID,
			Description: c.Description,
			Weight:      c.Weight,
		}
	}

	_, err := r.client.ent.EvalCase.UpdateOneID(e.ID.String()).
		SetName(e.Name).
		SetPrompt(e.Prompt).
		SetShouldTrigger(e.ShouldTrigger).
		SetExpectedOutput(e.ExpectedOutput).
		SetRubric(entRubric).
		Save(ctx)
	return err
}

// Delete deletes an eval case by its ID.
func (r *EvalCaseRepository) Delete(ctx context.Context, id string) error {
	return r.client.ent.EvalCase.DeleteOneID(id).Exec(ctx)
}

// toDomainEvalCase converts an ent EvalCase to a domain EvalCase.
func (r *EvalCaseRepository) toDomainEvalCase(e *ent.EvalCase) *domain.EvalCase {
	assetID := domain.ID{}
	if e.Edges.Asset != nil {
		assetID = domain.MustNewID(e.Edges.Asset.ID)
	}

	rubric := domain.Rubric{
		MaxScore: e.Rubric.MaxScore,
		Checks:   make([]domain.RubricCheck, len(e.Rubric.Checks)),
	}
	for i, c := range e.Rubric.Checks {
		rubric.Checks[i] = domain.RubricCheck{
			ID:          c.ID,
			Description: c.Description,
			Weight:      c.Weight,
		}
	}

	return &domain.EvalCase{
		ID:             domain.MustNewID(e.ID),
		AssetID:        assetID,
		Name:           e.Name,
		Prompt:         e.Prompt,
		ShouldTrigger:  e.ShouldTrigger,
		ExpectedOutput: e.ExpectedOutput,
		Rubric:         rubric,
		CreatedAt:      e.CreatedAt,
		Version:        0,
	}
}
