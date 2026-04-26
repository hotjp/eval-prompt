package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eval-prompt/internal/domain"
)

// Deprecated: Execution is stored in .evals/executions/*.json, not in database.

// EvalExecutionRepository provides repository operations for EvalExecution entities.
type EvalExecutionRepository struct {
	client *Client
}

// NewEvalExecutionRepository creates a new EvalExecutionRepository.
func NewEvalExecutionRepository(client *Client) *EvalExecutionRepository {
	return &EvalExecutionRepository{client: client}
}

// Create creates a new eval execution in the database.
func (r *EvalExecutionRepository) Create(ctx context.Context, e *domain.EvalExecution) error {
	if r.client == nil || r.client.db == nil {
		return nil
	}
	caseIDsJSON, err := json.Marshal(e.CaseIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal case_ids: %w", err)
	}

	query := `
		INSERT INTO eval_executions (id, asset_id, snapshot_id, mode, runs_per_case, case_ids,
			total_runs, completed_runs, failed_runs, status, concurrency, model, temperature,
			created_at, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.client.db.ExecContext(ctx, query,
		e.ID, e.AssetID, e.SnapshotID, string(e.Mode), e.RunsPerCase, caseIDsJSON,
		e.TotalRuns, e.CompletedRuns, e.FailedRuns, string(e.Status), e.Concurrency,
		e.Model, e.Temperature, e.CreatedAt, e.StartedAt, e.CompletedAt,
	)
	return err
}

// GetByID retrieves an eval execution by its ID.
func (r *EvalExecutionRepository) GetByID(ctx context.Context, id string) (*domain.EvalExecution, error) {
	if r.client == nil || r.client.db == nil {
		return nil, nil
	}
	query := `
		SELECT id, asset_id, snapshot_id, mode, runs_per_case, case_ids,
			total_runs, completed_runs, failed_runs, status, concurrency, model,
			temperature, created_at, started_at, completed_at
		FROM eval_executions WHERE id = ?
	`
	row := r.client.db.QueryRowContext(ctx, query, id)
	return r.scanExecution(row)
}

// GetByStatus retrieves all eval executions with the given status.
func (r *EvalExecutionRepository) GetByStatus(ctx context.Context, status domain.ExecutionStatus) ([]*domain.EvalExecution, error) {
	if r.client == nil || r.client.db == nil {
		return nil, nil
	}
	query := `
		SELECT id, asset_id, snapshot_id, mode, runs_per_case, case_ids,
			total_runs, completed_runs, failed_runs, status, concurrency, model,
			temperature, created_at, started_at, completed_at
		FROM eval_executions WHERE status = ?
		ORDER BY created_at DESC
	`
	rows, err := r.client.db.QueryContext(ctx, query, string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.EvalExecution
	for rows.Next() {
		e, err := r.scanExecutionRows(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, e)
	}
	return results, rows.Err()
}

// List retrieves eval executions with pagination.
func (r *EvalExecutionRepository) List(ctx context.Context, offset, limit int) ([]*domain.EvalExecution, int, error) {
	if r.client == nil || r.client.db == nil {
		return nil, 0, nil
	}
	countQuery := "SELECT COUNT(*) FROM eval_executions"
	var total int
	if err := r.client.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, asset_id, snapshot_id, mode, runs_per_case, case_ids,
			total_runs, completed_runs, failed_runs, status, concurrency, model,
			temperature, created_at, started_at, completed_at
		FROM eval_executions
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.client.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*domain.EvalExecution
	for rows.Next() {
		e, err := r.scanExecutionRows(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, e)
	}
	return results, total, rows.Err()
}

// UpdateStatus updates the status of an eval execution.
func (r *EvalExecutionRepository) UpdateStatus(ctx context.Context, id string, status domain.ExecutionStatus) error {
	if r.client == nil || r.client.db == nil {
		return nil
	}
	query := "UPDATE eval_executions SET status = ? WHERE id = ?"
	_, err := r.client.db.ExecContext(ctx, query, string(status), id)
	return err
}

// UpdateProgress updates the completed_runs and failed_runs counters.
// The cancelled_runs parameter is accepted for interface compatibility but
// is not persisted to the database in this implementation.
func (r *EvalExecutionRepository) UpdateProgress(ctx context.Context, id string, completedRuns, failedRuns, cancelledRuns int) error {
	if r.client == nil || r.client.db == nil {
		return nil
	}
	query := "UPDATE eval_executions SET completed_runs = ?, failed_runs = ? WHERE id = ?"
	_, err := r.client.db.ExecContext(ctx, query, completedRuns, failedRuns, id)
	return err
}

// UpdateStartedAt updates the started_at timestamp.
func (r *EvalExecutionRepository) UpdateStartedAt(ctx context.Context, id string, startedAt time.Time) error {
	if r.client == nil || r.client.db == nil {
		return nil
	}
	query := "UPDATE eval_executions SET started_at = ? WHERE id = ?"
	_, err := r.client.db.ExecContext(ctx, query, startedAt, id)
	return err
}

// UpdateCompletedAt updates the completed_at timestamp and status.
func (r *EvalExecutionRepository) UpdateCompletedAt(ctx context.Context, id string, completedAt time.Time, status domain.ExecutionStatus) error {
	if r.client == nil || r.client.db == nil {
		return nil
	}
	query := "UPDATE eval_executions SET completed_at = ?, status = ? WHERE id = ?"
	_, err := r.client.db.ExecContext(ctx, query, completedAt, string(status), id)
	return err
}

// Delete deletes an eval execution by its ID.
func (r *EvalExecutionRepository) Delete(ctx context.Context, id string) error {
	if r.client == nil || r.client.db == nil {
		return nil
	}
	query := "DELETE FROM eval_executions WHERE id = ?"
	_, err := r.client.db.ExecContext(ctx, query, id)
	return err
}

func (r *EvalExecutionRepository) scanExecution(row *sql.Row) (*domain.EvalExecution, error) {
	var e domain.EvalExecution
	var caseIDsJSON []byte
	var startedAt, completedAt sql.NullTime
	var model sql.NullString

	err := row.Scan(
		&e.ID, &e.AssetID, &e.SnapshotID, &e.Mode, &e.RunsPerCase, &caseIDsJSON,
		&e.TotalRuns, &e.CompletedRuns, &e.FailedRuns, &e.Status, &e.Concurrency,
		&model, &e.Temperature, &e.CreatedAt, &startedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}
	if caseIDsJSON != nil {
		if err := json.Unmarshal(caseIDsJSON, &e.CaseIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal case_ids: %w", err)
		}
	}
	if startedAt.Valid {
		e.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		e.CompletedAt = &completedAt.Time
	}
	if model.Valid {
		e.Model = model.String
	}
	return &e, nil
}

func (r *EvalExecutionRepository) scanExecutionRows(rows *sql.Rows) (*domain.EvalExecution, error) {
	var e domain.EvalExecution
	var caseIDsJSON []byte
	var startedAt, completedAt sql.NullTime
	var model sql.NullString

	err := rows.Scan(
		&e.ID, &e.AssetID, &e.SnapshotID, &e.Mode, &e.RunsPerCase, &caseIDsJSON,
		&e.TotalRuns, &e.CompletedRuns, &e.FailedRuns, &e.Status, &e.Concurrency,
		&model, &e.Temperature, &e.CreatedAt, &startedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}
	if caseIDsJSON != nil {
		if err := json.Unmarshal(caseIDsJSON, &e.CaseIDs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal case_ids: %w", err)
		}
	}
	if startedAt.Valid {
		e.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		e.CompletedAt = &completedAt.Time
	}
	if model.Valid {
		e.Model = model.String
	}
	return &e, nil
}
