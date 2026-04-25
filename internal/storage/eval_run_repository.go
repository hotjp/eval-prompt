package storage

import (
	"context"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage/ent"
	"github.com/eval-prompt/internal/storage/ent/evalcase"
	"github.com/eval-prompt/internal/storage/ent/evalrun"
	"github.com/eval-prompt/internal/storage/ent/schema"
)

// EvalRunRepository provides repository operations for EvalRun entities.
type EvalRunRepository struct {
	client *Client
}

// NewEvalRunRepository creates a new EvalRunRepository.
func NewEvalRunRepository(client *Client) *EvalRunRepository {
	return &EvalRunRepository{client: client}
}

// Create creates a new eval run in the database.
func (r *EvalRunRepository) Create(ctx context.Context, e *domain.EvalRun) error {
	entRubricDetails := make([]schema.RubricCheckResult, len(e.RubricDetails))
	for i, rd := range e.RubricDetails {
		entRubricDetails[i] = schema.RubricCheckResult{
			CheckID: rd.CheckID,
			Passed:  rd.Passed,
			Score:   rd.Score,
			Details: rd.Details,
		}
	}

	_, err := r.client.ent.EvalRun.Create().
		SetID(e.ID.String()).
		SetEvalCaseID(e.EvalCaseID.String()).
		SetStatus(r.statusToEnt(e.Status)).
		SetDeterministicScore(e.DeterministicScore).
		SetRubricScore(e.RubricScore).
		SetRubricDetails(entRubricDetails).
		SetTracePath(e.TracePath).
		SetTokenInput(e.TokenInput).
		SetTokenOutput(e.TokenOutput).
		SetDurationMs(int(e.DurationMs)).
		SetCreatedAt(e.CreatedAt).
		Save(ctx)
	return err
}

// GetByID retrieves an eval run by its ID.
func (r *EvalRunRepository) GetByID(ctx context.Context, id string) (*domain.EvalRun, error) {
	entRun, err := r.client.ent.EvalRun.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.toDomainEvalRun(entRun), nil
}

// GetByEvalCaseID retrieves all eval runs for an eval case.
func (r *EvalRunRepository) GetByEvalCaseID(ctx context.Context, evalCaseID string) ([]*domain.EvalRun, error) {
	entRuns, err := r.client.ent.EvalRun.Query().
		Where(evalrun.HasEvalCaseWith(evalcase.IDEQ(evalCaseID))).
		All(ctx)
	if err != nil {
		return nil, err
	}

	runs := make([]*domain.EvalRun, len(entRuns))
	for i, entRun := range entRuns {
		runs[i] = r.toDomainEvalRun(entRun)
	}
	return runs, nil
}

// GetBySnapshotID is deprecated and always returns empty results.
// Snapshot is no longer used - eval history is stored in .md files.
func (r *EvalRunRepository) GetBySnapshotID(ctx context.Context, snapshotID string) ([]*domain.EvalRun, error) {
	return []*domain.EvalRun{}, nil
}

// List retrieves eval runs with pagination.
func (r *EvalRunRepository) List(ctx context.Context, offset, limit int) ([]*domain.EvalRun, int, error) {
	entRuns, err := r.client.ent.EvalRun.Query().
		Offset(offset).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.client.ent.EvalRun.Query().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	runs := make([]*domain.EvalRun, len(entRuns))
	for i, entRun := range entRuns {
		runs[i] = r.toDomainEvalRun(entRun)
	}
	return runs, total, nil
}

// ListByStatus retrieves eval runs by status.
func (r *EvalRunRepository) ListByStatus(ctx context.Context, status domain.EvalRunStatus, offset, limit int) ([]*domain.EvalRun, int, error) {
	entStatus := r.statusToEnt(status)
	entRuns, err := r.client.ent.EvalRun.Query().
		Where(evalrun.StatusEQ(entStatus)).
		Offset(offset).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	total, err := r.client.ent.EvalRun.Query().Where(evalrun.StatusEQ(entStatus)).Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	runs := make([]*domain.EvalRun, len(entRuns))
	for i, entRun := range entRuns {
		runs[i] = r.toDomainEvalRun(entRun)
	}
	return runs, total, nil
}

// Update updates an existing eval run.
func (r *EvalRunRepository) Update(ctx context.Context, e *domain.EvalRun) error {
	entRubricDetails := make([]schema.RubricCheckResult, len(e.RubricDetails))
	for i, rd := range e.RubricDetails {
		entRubricDetails[i] = schema.RubricCheckResult{
			CheckID: rd.CheckID,
			Passed:  rd.Passed,
			Score:   rd.Score,
			Details: rd.Details,
		}
	}

	_, err := r.client.ent.EvalRun.UpdateOneID(e.ID.String()).
		SetStatus(r.statusToEnt(e.Status)).
		SetDeterministicScore(e.DeterministicScore).
		SetRubricScore(e.RubricScore).
		SetRubricDetails(entRubricDetails).
		SetTracePath(e.TracePath).
		SetTokenInput(e.TokenInput).
		SetTokenOutput(e.TokenOutput).
		SetDurationMs(int(e.DurationMs)).
		Save(ctx)
	return err
}

// UpdateStatus updates only the status of an eval run.
func (r *EvalRunRepository) UpdateStatus(ctx context.Context, id string, status domain.EvalRunStatus) error {
	_, err := r.client.ent.EvalRun.UpdateOneID(id).
		SetStatus(r.statusToEnt(status)).
		Save(ctx)
	return err
}

// Delete deletes an eval run by its ID.
func (r *EvalRunRepository) Delete(ctx context.Context, id string) error {
	return r.client.ent.EvalRun.DeleteOneID(id).Exec(ctx)
}

// toDomainEvalRun converts an ent EvalRun to a domain EvalRun.
func (r *EvalRunRepository) toDomainEvalRun(e *ent.EvalRun) *domain.EvalRun {
	evalCaseID := domain.ID{}
	if e.Edges.EvalCase != nil {
		evalCaseID = domain.MustNewID(e.Edges.EvalCase.ID)
	}

	rubricDetails := make([]domain.RubricCheckResult, len(e.RubricDetails))
	for i, rd := range e.RubricDetails {
		rubricDetails[i] = domain.RubricCheckResult{
			CheckID: rd.CheckID,
			Passed:  rd.Passed,
			Score:   rd.Score,
			Details: rd.Details,
		}
	}

	return &domain.EvalRun{
		ID:                 domain.MustNewID(e.ID),
		EvalCaseID:         evalCaseID,
		Status:             r.statusFromEnt(e.Status),
		DeterministicScore: e.DeterministicScore,
		RubricScore:        e.RubricScore,
		RubricDetails:      rubricDetails,
		TracePath:          e.TracePath,
		TokenInput:         e.TokenInput,
		TokenOutput:        e.TokenOutput,
		DurationMs:         int64(e.DurationMs),
		CreatedAt:          e.CreatedAt,
	}
}

// statusToEnt converts a domain EvalRunStatus to an ent evalrun.Status.
func (r *EvalRunRepository) statusToEnt(status domain.EvalRunStatus) evalrun.Status {
	switch status {
	case domain.EvalRunStatusPending:
		return evalrun.StatusPending
	case domain.EvalRunStatusRunning:
		return evalrun.StatusRunning
	case domain.EvalRunStatusPassed:
		return evalrun.StatusPassed
	case domain.EvalRunStatusFailed:
		return evalrun.StatusFailed
	default:
		return evalrun.StatusPending
	}
}

// statusFromEnt converts an ent evalrun.Status to a domain EvalRunStatus.
func (r *EvalRunRepository) statusFromEnt(status evalrun.Status) domain.EvalRunStatus {
	switch status {
	case evalrun.StatusPending:
		return domain.EvalRunStatusPending
	case evalrun.StatusRunning:
		return domain.EvalRunStatusRunning
	case evalrun.StatusPassed:
		return domain.EvalRunStatusPassed
	case evalrun.StatusFailed:
		return domain.EvalRunStatusFailed
	default:
		return domain.EvalRunStatusPending
	}
}
