package storage

import (
	"context"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent"
	"github.com/eval-prompt/internal/storage/ent/modeladaptation"
)

// ModelAdaptationRepository provides repository operations for ModelAdaptation entities.
type ModelAdaptationRepository struct {
	client *Client
}

// NewModelAdaptationRepository creates a new ModelAdaptationRepository.
func NewModelAdaptationRepository(client *Client) *ModelAdaptationRepository {
	return &ModelAdaptationRepository{client: client}
}

// Create creates a new model adaptation in the database.
func (r *ModelAdaptationRepository) Create(ctx context.Context, m *domain.ModelAdaptation) error {
	_, err := r.client.ent.ModelAdaptation.Create().
		SetID(m.ID.String()).
		SetPromptID(m.PromptID.String()).
		SetSourceModel(m.SourceModel).
		SetTargetModel(m.TargetModel).
		SetAdaptedContent(m.AdaptedContent).
		SetParamAdjustments(m.ParamAdjustments).
		SetFormatChanges(m.FormatChanges).
		SetEvalScore(m.EvalScore).
		SetEvalRunID(m.EvalRunID.String()).
		SetCreatedAt(m.CreatedAt).
		Save(ctx)
	return err
}

// GetByID retrieves a model adaptation by its ID.
func (r *ModelAdaptationRepository) GetByID(ctx context.Context, id string) (*domain.ModelAdaptation, error) {
	entAdapt, err := r.client.ent.ModelAdaptation.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.toDomainModelAdaptation(entAdapt), nil
}

// GetByPromptID retrieves all model adaptations for a prompt.
func (r *ModelAdaptationRepository) GetByPromptID(ctx context.Context, promptID string) ([]*domain.ModelAdaptation, error) {
	entAdapts, err := r.client.ent.ModelAdaptation.Query().
		Where(modeladaptation.PromptIDEQ(promptID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	adapts := make([]*domain.ModelAdaptation, len(entAdapts))
	for i, entAdapt := range entAdapts {
		adapts[i] = r.toDomainModelAdaptation(entAdapt)
	}
	return adapts, nil
}

// GetByTargetModel retrieves all model adaptations for a target model.
func (r *ModelAdaptationRepository) GetByTargetModel(ctx context.Context, targetModel string) ([]*domain.ModelAdaptation, error) {
	entAdapts, err := r.client.ent.ModelAdaptation.Query().
		Where(modeladaptation.TargetModelEQ(targetModel)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	adapts := make([]*domain.ModelAdaptation, len(entAdapts))
	for i, entAdapt := range entAdapts {
		adapts[i] = r.toDomainModelAdaptation(entAdapt)
	}
	return adapts, nil
}

// Update updates a model adaptation.
func (r *ModelAdaptationRepository) Update(ctx context.Context, m *domain.ModelAdaptation) error {
	_, err := r.client.ent.ModelAdaptation.UpdateOneID(m.ID.String()).
		SetAdaptedContent(m.AdaptedContent).
		SetParamAdjustments(m.ParamAdjustments).
		SetFormatChanges(m.FormatChanges).
		SetEvalScore(m.EvalScore).
		SetEvalRunID(m.EvalRunID.String()).
		Save(ctx)
	return err
}

// Delete deletes a model adaptation by its ID.
func (r *ModelAdaptationRepository) Delete(ctx context.Context, id string) error {
	return r.client.ent.ModelAdaptation.DeleteOneID(id).Exec(ctx)
}

// toDomainModelAdaptation converts an ent ModelAdaptation to a domain ModelAdaptation.
func (r *ModelAdaptationRepository) toDomainModelAdaptation(e *ent.ModelAdaptation) *domain.ModelAdaptation {
	promptID := domain.ID{}
	if e.PromptID != "" {
		promptID = domain.MustNewID(e.PromptID)
	}

	evalRunID := domain.ID{}
	if e.EvalRunID != "" {
		evalRunID = domain.MustNewID(e.EvalRunID)
	}

	paramAdjustments := make(map[string]float64)
	if e.ParamAdjustments != nil {
		paramAdjustments = e.ParamAdjustments
	}

	formatChanges := make([]string, len(e.FormatChanges))
	if e.FormatChanges != nil {
		formatChanges = e.FormatChanges
	}

	return &domain.ModelAdaptation{
		ID:               domain.MustNewID(e.ID),
		PromptID:         promptID,
		SourceModel:      e.SourceModel,
		TargetModel:      e.TargetModel,
		AdaptedContent:   e.AdaptedContent,
		ParamAdjustments: paramAdjustments,
		FormatChanges:    formatChanges,
		EvalScore:        e.EvalScore,
		EvalRunID:        evalRunID,
		CreatedAt:        e.CreatedAt,
	}
}
