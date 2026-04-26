package domain

import (
	"errors"
	"time"
)

// ExecutionMode represents the eval execution mode.
type ExecutionMode string

const (
	ModeSingle ExecutionMode = "single"
	ModeBatch  ExecutionMode = "batch"
	ModeMatrix ExecutionMode = "matrix"
)

// ExecutionStatus represents the status of an eval execution.
type ExecutionStatus string

const (
	ExecutionStatusPending        ExecutionStatus = "pending"
	ExecutionStatusRunning        ExecutionStatus = "running"
	ExecutionStatusCompleted      ExecutionStatus = "completed"
	ExecutionStatusPartialFailure ExecutionStatus = "partial_failure"
	ExecutionStatusFailed         ExecutionStatus = "failed"
	ExecutionStatusCancelled      ExecutionStatus = "cancelled"
)

// EvalExecution represents an eval execution batch.
// An EvalExecution tracks the execution of one or more test cases against an asset,
// supporting various execution modes (single, batch, matrix) with configurable
// concurrency and model parameters.
type EvalExecution struct {
	ID             string         `json:"id"`
	AssetID        string         `json:"asset_id"`
	SnapshotID     string         `json:"snapshot_id"`
	Mode           ExecutionMode  `json:"mode"`
	RunsPerCase    int            `json:"runs_per_case"`
	CaseIDs        []string       `json:"case_ids"`
	TotalRuns      int            `json:"total_runs"`
	CompletedRuns  int            `json:"completed_runs"`
	FailedRuns     int            `json:"failed_runs"`
	CancelledRuns  int            `json:"cancelled_runs"`
	Status         ExecutionStatus `json:"status"`
	Concurrency    int            `json:"concurrency"`
	Model          string         `json:"model"`
	Temperature    float64        `json:"temperature"`
	CreatedAt      time.Time      `json:"created_at"`
	StartedAt      *time.Time     `json:"started_at,omitempty"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
}

// Validate validates the eval execution structure.
func (e *EvalExecution) Validate() error {
	if e.ID == "" {
		return errors.New("execution ID is required")
	}
	if e.AssetID == "" {
		return errors.New("asset ID is required")
	}
	if len(e.CaseIDs) == 0 {
		return errors.New("case IDs cannot be empty")
	}
	return nil
}

// WorkItemStatus represents the status of an eval work item.
type WorkItemStatus string

const (
	WorkItemStatusPending   WorkItemStatus = "pending"
	WorkItemStatusRunning   WorkItemStatus = "running"
	WorkItemStatusCompleted WorkItemStatus = "completed"
	WorkItemStatusFailed    WorkItemStatus = "failed"
	WorkItemStatusCancelled WorkItemStatus = "cancelled"
)

// EvalWorkItem represents a single work item in an eval execution.
type EvalWorkItem struct {
	ID           string         `json:"id"`
	ExecutionID  string         `json:"execution_id"`
	CaseID       string         `json:"case_id"`
	RunNumber    int            `json:"run_number"`
	Status       WorkItemStatus `json:"status"`
	PromptHash   string         `json:"prompt_hash,omitempty"`
	PromptText   string         `json:"prompt_text,omitempty"`
	Response     string         `json:"response,omitempty"`
	Model        string         `json:"model,omitempty"`
	Temperature  float64       `json:"temperature"`
	TokensIn     int            `json:"tokens_in"`
	TokensOut    int            `json:"tokens_out"`
	DurationMs   int            `json:"duration_ms"`
	Error        string         `json:"error,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
}
