package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/eval-prompt/internal/domain"
)

// Deprecated: WorkItem is stored in .evals/calls/*/calls.jsonl, not in database.

// EvalWorkItemRepository provides repository operations for EvalWorkItem entities.
type EvalWorkItemRepository struct {
	client *Client
}

// NewEvalWorkItemRepository creates a new EvalWorkItemRepository.
func NewEvalWorkItemRepository(client *Client) *EvalWorkItemRepository {
	return &EvalWorkItemRepository{client: client}
}

// Create creates a new eval work item in the database.
func (r *EvalWorkItemRepository) Create(ctx context.Context, w *domain.EvalWorkItem) error {
	if r.client == nil || r.client.db == nil {
		return nil
	}
	query := `
		INSERT INTO eval_work_items (id, execution_id, eval_case_id, run_number, status,
			prompt_hash, prompt_text, response, model, temperature,
			tokens_in, tokens_out, duration_ms, error, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.client.db.ExecContext(ctx, query,
		w.ID, w.ExecutionID, w.CaseID, w.RunNumber, string(w.Status),
		w.PromptHash, w.PromptText, w.Response, w.Model, w.Temperature,
		w.TokensIn, w.TokensOut, w.DurationMs, w.Error, w.CreatedAt, w.CompletedAt,
	)
	return err
}

// GetByID retrieves an eval work item by its ID.
func (r *EvalWorkItemRepository) GetByID(ctx context.Context, id string) (*domain.EvalWorkItem, error) {
	if r.client == nil || r.client.db == nil {
		return nil, nil
	}
	query := `
		SELECT id, execution_id, eval_case_id, run_number, status,
			prompt_hash, prompt_text, response, model, temperature,
			tokens_in, tokens_out, duration_ms, error, created_at, completed_at
		FROM eval_work_items WHERE id = ?
	`
	row := r.client.db.QueryRowContext(ctx, query, id)
	return r.scanWorkItem(row)
}

// GetByExecutionID retrieves all work items for an execution.
func (r *EvalWorkItemRepository) GetByExecutionID(ctx context.Context, executionID string) ([]*domain.EvalWorkItem, error) {
	if r.client == nil || r.client.db == nil {
		return nil, nil
	}
	query := `
		SELECT id, execution_id, eval_case_id, run_number, status,
			prompt_hash, prompt_text, response, model, temperature,
			tokens_in, tokens_out, duration_ms, error, created_at, completed_at
		FROM eval_work_items WHERE execution_id = ?
		ORDER BY created_at ASC
	`
	rows, err := r.client.db.QueryContext(ctx, query, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.EvalWorkItem
	for rows.Next() {
		w, err := r.scanWorkItemRows(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, w)
	}
	return results, rows.Err()
}

// GetPendingByExecutionID retrieves pending work items for an execution (for worker pool scheduling).
func (r *EvalWorkItemRepository) GetPendingByExecutionID(ctx context.Context, executionID string) ([]*domain.EvalWorkItem, error) {
	if r.client == nil || r.client.db == nil {
		return nil, nil
	}
	query := `
		SELECT id, execution_id, eval_case_id, run_number, status,
			prompt_hash, prompt_text, response, model, temperature,
			tokens_in, tokens_out, duration_ms, error, created_at, completed_at
		FROM eval_work_items WHERE execution_id = ? AND status = 'pending'
		ORDER BY created_at ASC
		LIMIT 1
	`
	rows, err := r.client.db.QueryContext(ctx, query, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.EvalWorkItem
	for rows.Next() {
		w, err := r.scanWorkItemRows(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, w)
	}
	return results, rows.Err()
}

// UpdateStatus updates the status of an eval work item.
func (r *EvalWorkItemRepository) UpdateStatus(ctx context.Context, id string, status domain.WorkItemStatus) error {
	if r.client == nil || r.client.db == nil {
		return nil
	}
	query := "UPDATE eval_work_items SET status = ? WHERE id = ?"
	_, err := r.client.db.ExecContext(ctx, query, string(status), id)
	return err
}

// UpdateResult updates the result fields of a work item.
func (r *EvalWorkItemRepository) UpdateResult(ctx context.Context, id string, status domain.WorkItemStatus, response string, tokensIn, tokensOut, durationMs int, errorMsg string, completedAt time.Time) error {
	if r.client == nil || r.client.db == nil {
		return nil
	}
	query := `
		UPDATE eval_work_items
		SET status = ?, response = ?, tokens_in = ?, tokens_out = ?,
			duration_ms = ?, error = ?, completed_at = ?
		WHERE id = ?
	`
	_, err := r.client.db.ExecContext(ctx, query,
		string(status), response, tokensIn, tokensOut, durationMs, errorMsg, completedAt, id)
	return err
}

// CountByExecutionID returns counts of work items by status for an execution.
func (r *EvalWorkItemRepository) CountByExecutionID(ctx context.Context, executionID string) (total, pending, running, completed, failed int, err error) {
	if r.client == nil || r.client.db == nil {
		return 0, 0, 0, 0, 0, nil
	}
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) as running,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
		FROM eval_work_items WHERE execution_id = ?
	`
	row := r.client.db.QueryRowContext(ctx, query, executionID)
	err = row.Scan(&total, &pending, &running, &completed, &failed)
	return
}

// Delete deletes an eval work item by its ID.
func (r *EvalWorkItemRepository) Delete(ctx context.Context, id string) error {
	if r.client == nil || r.client.db == nil {
		return nil
	}
	query := "DELETE FROM eval_work_items WHERE id = ?"
	_, err := r.client.db.ExecContext(ctx, query, id)
	return err
}

// DeleteByExecutionID deletes all work items for an execution.
func (r *EvalWorkItemRepository) DeleteByExecutionID(ctx context.Context, executionID string) error {
	if r.client == nil || r.client.db == nil {
		return nil
	}
	query := "DELETE FROM eval_work_items WHERE execution_id = ?"
	_, err := r.client.db.ExecContext(ctx, query, executionID)
	return err
}

func (r *EvalWorkItemRepository) scanWorkItem(row *sql.Row) (*domain.EvalWorkItem, error) {
	var w domain.EvalWorkItem
	var promptHash, promptText, response, model, errorMsg sql.NullString
	var completedAt sql.NullTime

	err := row.Scan(
		&w.ID, &w.ExecutionID, &w.CaseID, &w.RunNumber, &w.Status,
		&promptHash, &promptText, &response, &model, &w.Temperature,
		&w.TokensIn, &w.TokensOut, &w.DurationMs, &errorMsg, &w.CreatedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}
	if promptHash.Valid {
		w.PromptHash = promptHash.String
	}
	if promptText.Valid {
		w.PromptText = promptText.String
	}
	if response.Valid {
		w.Response = response.String
	}
	if model.Valid {
		w.Model = model.String
	}
	if errorMsg.Valid {
		w.Error = errorMsg.String
	}
	if completedAt.Valid {
		w.CompletedAt = &completedAt.Time
	}
	return &w, nil
}

func (r *EvalWorkItemRepository) scanWorkItemRows(rows *sql.Rows) (*domain.EvalWorkItem, error) {
	var w domain.EvalWorkItem
	var promptHash, promptText, response, model, errorMsg sql.NullString
	var completedAt sql.NullTime

	err := rows.Scan(
		&w.ID, &w.ExecutionID, &w.CaseID, &w.RunNumber, &w.Status,
		&promptHash, &promptText, &response, &model, &w.Temperature,
		&w.TokensIn, &w.TokensOut, &w.DurationMs, &errorMsg, &w.CreatedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}
	if promptHash.Valid {
		w.PromptHash = promptHash.String
	}
	if promptText.Valid {
		w.PromptText = promptText.String
	}
	if response.Valid {
		w.Response = response.String
	}
	if model.Valid {
		w.Model = model.String
	}
	if errorMsg.Valid {
		w.Error = errorMsg.String
	}
	if completedAt.Valid {
		w.CompletedAt = &completedAt.Time
	}
	return &w, nil
}
