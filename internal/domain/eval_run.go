package domain

import (
	"time"
)

// EvalRunStatus represents the status of an evaluation run.
type EvalRunStatus string

const (
	EvalRunStatusPending  EvalRunStatus = "pending"
	EvalRunStatusRunning EvalRunStatus = "running"
	EvalRunStatusPassed  EvalRunStatus = "passed"
	EvalRunStatusFailed  EvalRunStatus = "failed"
)

// EvalRun represents a single evaluation execution.
type EvalRun struct {
	ID                 ID
	EvalCaseID        ID
	SnapshotID        ID
	Status            EvalRunStatus
	DeterministicScore float64
	RubricScore       int
	RubricDetails     []RubricCheckResult
	TracePath         string
	TokenInput        int
	TokenOutput       int
	DurationMs        int64
	CreatedAt         time.Time
}

// Validate validates the eval run.
func (e *EvalRun) Validate() error {
	if e.ID.IsEmpty() {
		return ErrInvalidID(e.ID.String())
	}
	if e.EvalCaseID.IsEmpty() {
		return NewDomainError(ErrEvalCaseNotFound, "eval_case_id is required")
	}
	if e.SnapshotID.IsEmpty() {
		return NewDomainError(ErrSnapshotNotFound, "snapshot_id is required")
	}
	return nil
}

// IsPassed returns true if the eval run passed.
func (e *EvalRun) IsPassed() bool {
	return e.Status == EvalRunStatusPassed
}

// IsFailed returns true if the eval run failed.
func (e *EvalRun) IsFailed() bool {
	return e.Status == EvalRunStatusFailed
}

// TotalScore returns the weighted total score.
func (e *EvalRun) TotalScore() int {
	return e.RubricScore
}

// NewEvalRun creates a new EvalRun.
func NewEvalRun(evalCaseID, snapshotID ID) *EvalRun {
	return &EvalRun{
		ID:          NewAutoID(),
		EvalCaseID: evalCaseID,
		SnapshotID: snapshotID,
		Status:     EvalRunStatusPending,
		CreatedAt: time.Now(),
	}
}

// Start marks the eval run as running.
func (e *EvalRun) Start() {
	e.Status = EvalRunStatusRunning
}

// Complete marks the eval run as completed with the given score.
func (e *EvalRun) Complete(deterministicScore float64, rubricScore int, passed bool) {
	e.DeterministicScore = deterministicScore
	e.RubricScore = rubricScore
	if passed {
		e.Status = EvalRunStatusPassed
	} else {
		e.Status = EvalRunStatusFailed
	}
}

// Fail marks the eval run as failed.
func (e *EvalRun) Fail() {
	e.Status = EvalRunStatusFailed
}

// EvalRunSummary is a lightweight representation of an eval run.
type EvalRunSummary struct {
	ID                 ID
	EvalCaseID        ID
	SnapshotID        ID
	Status            EvalRunStatus
	DeterministicScore float64
	RubricScore       int
	CreatedAt         time.Time
}
