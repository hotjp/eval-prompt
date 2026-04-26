// Package service implements L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/eval-prompt/internal/domain"
)

// RunContext is the shared context across all workers, containing cancellation signal.
type RunContext struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu          sync.RWMutex
	status      domain.ExecutionStatus
	completed   int
	failed      int
	cancelled   int
	total       int
	startedAt   *time.Time
	completedAt *time.Time

	cancelCh chan struct{} // Closed when cancelled
}

// NewRunContext creates a new RunContext.
func NewRunContext() *RunContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &RunContext{
		ctx:      ctx,
		cancel:   cancel,
		cancelCh: make(chan struct{}, 1), // buffered to avoid potential deadlock
		status:   domain.ExecutionStatusPending,
	}
}

// Context returns the underlying context.
func (rc *RunContext) Context() context.Context {
	return rc.ctx
}

// IsCancelled returns true if the execution was cancelled.
func (rc *RunContext) IsCancelled() bool {
	select {
	case <-rc.cancelCh:
		return true
	default:
		return false
	}
}

// Cancel cancels the execution.
func (rc *RunContext) Cancel() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.cancel()
	close(rc.cancelCh)
	rc.status = domain.ExecutionStatusCancelled
}

// Status returns the current execution status.
func (rc *RunContext) Status() domain.ExecutionStatus {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.status
}

// UpdateProgress updates the completed, failed, and cancelled counters.
func (rc *RunContext) UpdateProgress(completed, failed, cancelled int) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.completed = completed
	rc.failed = failed
	rc.cancelled = cancelled

	total := rc.completed + rc.failed + rc.cancelled
	if total >= rc.total {
		if rc.cancelled > 0 && rc.completed == 0 && rc.failed == 0 {
			rc.status = domain.ExecutionStatusCancelled
			now := time.Now()
			rc.completedAt = &now
		} else if failed == 0 {
			rc.status = domain.ExecutionStatusCompleted
			now := time.Now()
			rc.completedAt = &now
		} else if completed > 0 {
			rc.status = domain.ExecutionStatusPartialFailure
			now := time.Now()
			rc.completedAt = &now
		} else {
			rc.status = domain.ExecutionStatusFailed
			now := time.Now()
			rc.completedAt = &now
		}
	}
}

// SetRunning marks the execution as running.
func (rc *RunContext) SetRunning() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.status = domain.ExecutionStatusRunning
	now := time.Now()
	rc.startedAt = &now
}

// SetTotal sets the total number of work items.
func (rc *RunContext) SetTotal(total int) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.total = total
}

// Progress returns the current progress (completed, failed, cancelled, total).
func (rc *RunContext) Progress() (completed, failed, cancelled, total int) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.completed, rc.failed, rc.cancelled, rc.total
}

// RunResult contains the result of a single work item execution.
type RunResult struct {
	WorkItem            *domain.EvalWorkItem
	Status              RunStatus
	DeterministicScore float64
	RubricScore         int
	RubricDetails       []RubricCheckResult
	Error               string
}

// RunStatus represents the status of a run result.
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning  RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCancelled RunStatus = "cancelled"
)

// Worker is the execution unit that processes work items.
type Worker struct {
	id     int
	coord  *Coordinator
	llm    LLMInvoker
	runner EvalRunner
}

// Coordinator manages the execution of eval work items using a worker pool.
type Coordinator struct {
	execution    *domain.EvalExecution
	workers      int
	runCtx       *RunContext
	results      chan *RunResult
	wg           sync.WaitGroup
	workItemRepo WorkItemRepo
	runRepo      RunRepo
	caseRepo     EvalCaseGetter
	llm          LLMInvoker
	runner       EvalRunner
	executionRepo ExecutionUpdater
}

// ExecutionUpdater is the interface for updating execution progress.
type ExecutionUpdater interface {
	UpdateProgress(ctx context.Context, id string, completedRuns, failedRuns, cancelledRuns int) error
}

// WorkItemRepo is the interface for work item persistence.
type WorkItemRepo interface {
	Create(ctx context.Context, item *domain.EvalWorkItem) error
	UpdateStatus(ctx context.Context, id string, status domain.WorkItemStatus) error
	UpdateResult(ctx context.Context, id string, status domain.WorkItemStatus, response string, tokensIn, tokensOut, durationMs int, errorMsg string, completedAt time.Time) error
	GetByExecutionID(ctx context.Context, executionID string) ([]*domain.EvalWorkItem, error)
	GetPendingByExecutionID(ctx context.Context, executionID string) ([]*domain.EvalWorkItem, error)
	CountByExecutionID(ctx context.Context, executionID string) (total, pending, running, completed, failed int, err error)
}

// RunRepo is the interface for run persistence (for idempotency checks).
type RunRepo interface {
	Create(ctx context.Context, run *domain.EvalRun) error
	Update(ctx context.Context, run *domain.EvalRun) error
	GetByID(ctx context.Context, id string) (*domain.EvalRun, error)
}

// EvalCaseGetter is the interface for eval case access.
type EvalCaseGetter interface {
	GetByID(ctx context.Context, id string) (*domain.EvalCase, error)
	GetByAssetID(ctx context.Context, assetID string) ([]*domain.EvalCase, error)
}

// NewCoordinator creates a new Coordinator.
func NewCoordinator(
	execution *domain.EvalExecution,
	workers int,
	workItemRepo WorkItemRepo,
	runRepo RunRepo,
	caseRepo EvalCaseGetter,
	llm LLMInvoker,
	runner EvalRunner,
	executionRepo ExecutionUpdater,
) *Coordinator {
	return &Coordinator{
		execution:     execution,
		workers:       workers,
		runCtx:        NewRunContext(),
		results:       make(chan *RunResult, execution.TotalRuns),
		workItemRepo:  workItemRepo,
		runRepo:       runRepo,
		caseRepo:      caseRepo,
		llm:           llm,
		runner:        runner,
		executionRepo: executionRepo,
	}
}

// Execute starts the worker pool and distributes work.
func (c *Coordinator) Execute(ctx context.Context) error {
	c.runCtx.SetRunning()
	c.runCtx.SetTotal(c.execution.TotalRuns)

	// Create work channel with buffer
	workCh := make(chan *domain.EvalWorkItem, c.execution.TotalRuns)

	// Start worker pool
	for i := 0; i < c.workers; i++ {
		w := &Worker{
			id:     i,
			coord:  c,
			llm:    c.llm,
			runner: c.runner,
		}
		c.wg.Add(1)
		go func(w *Worker) {
			defer c.wg.Done()
			w.runLoop(ctx, workCh)
		}(w)
	}

	// Produce work items
	go func() {
		for _, caseID := range c.execution.CaseIDs {
			for runNum := 1; runNum <= c.execution.RunsPerCase; runNum++ {
				item := &domain.EvalWorkItem{
					ID:          domain.NewULID(),
					ExecutionID: c.execution.ID,
					CaseID:     caseID,
					RunNumber:  runNum,
					Status:     domain.WorkItemStatusPending,
					Model:      c.execution.Model,
					Temperature: c.execution.Temperature,
					CreatedAt:  time.Now(),
				}

				// Save to persistence
				if c.workItemRepo != nil {
					if err := c.workItemRepo.Create(ctx, item); err != nil {
						slog.Warn("failed to persist work item", "error", err, "work_item_id", item.ID)
					}
				}

				workCh <- item
			}
		}
		close(workCh)
	}()

	// Collect results
	go c.collectResults()

	// Wait for all workers to finish
	c.wg.Wait()
	close(c.results)
	c.finalize()

	return nil
}

// Cancel cancels the execution.
func (c *Coordinator) Cancel() {
	c.runCtx.Cancel()
}

// collectResults collects results from workers and updates progress.
func (c *Coordinator) collectResults() {
	for result := range c.results {
		switch result.Status {
		case RunStatusCompleted:
			c.runCtx.UpdateProgress(1, 0, 0)
		case RunStatusFailed:
			c.runCtx.UpdateProgress(0, 1, 0)
		case RunStatusCancelled:
			c.runCtx.UpdateProgress(0, 0, 1)
		default:
			c.runCtx.UpdateProgress(0, 1, 0)
		}

		// Update work item status in persistence
		if c.workItemRepo != nil && result.WorkItem != nil {
			item := result.WorkItem
			var status domain.WorkItemStatus
			switch result.Status {
			case RunStatusCompleted:
				status = domain.WorkItemStatusCompleted
			case RunStatusCancelled:
				status = domain.WorkItemStatusCancelled
			default:
				status = domain.WorkItemStatusFailed
			}
			c.workItemRepo.UpdateResult(context.Background(), item.ID, status, item.Response, item.TokensIn, item.TokensOut, item.DurationMs, item.Error, time.Now())
		}

		// Persist progress to database
		if c.executionRepo != nil {
			completed, failed, cancelled, _ := c.runCtx.Progress()
			if err := c.executionRepo.UpdateProgress(context.Background(), c.execution.ID, completed, failed, cancelled); err != nil {
				slog.Warn("failed to update execution progress",
					"layer", "service",
					"execution_id", c.execution.ID,
					"error", err,
				)
			}
		}
	}
}

// finalize performs cleanup after all workers are done.
func (c *Coordinator) finalize() {
	slog.Info("execution finalized",
		"layer", "service",
		"execution_id", c.execution.ID,
		"status", c.runCtx.Status(),
	)
}

// runLoop is the main loop for a worker.
func (w *Worker) runLoop(ctx context.Context, workCh <-chan *domain.EvalWorkItem) {
	for item := range workCh {
		// Check for cancellation
		select {
		case <-ctx.Done():
			w.markCancelled(ctx, item)
			continue
		case <-w.coord.runCtx.cancelCh:
			w.markCancelled(ctx, item)
			continue
		default:
		}

		// Process the work item
		w.processItem(ctx, item)
	}
}

// processItem processes a single work item.
func (w *Worker) processItem(ctx context.Context, item *domain.EvalWorkItem) {
	result := &RunResult{
		WorkItem: item,
		Status:  RunStatusRunning,
	}

	// Update item status to running
	item.Status = domain.WorkItemStatusRunning
	if w.coord.workItemRepo != nil {
		w.coord.workItemRepo.UpdateStatus(ctx, item.ID, domain.WorkItemStatusRunning)
	}

	// Get the eval case
	evalCase, err := w.coord.caseRepo.GetByID(ctx, item.CaseID)
	if err != nil {
		result.Status = RunStatusFailed
		result.Error = fmt.Sprintf("failed to get eval case: %v", err)
		item.Status = domain.WorkItemStatusFailed
		item.Error = result.Error
		w.coord.results <- result
		return
	}

	// Use eval prompt if available
	prompt := evalCase.Prompt

	// Hash the prompt for idempotency check
	item.PromptText = prompt
	item.PromptHash = hashPrompt(prompt)
	item.Model = w.coord.execution.Model
	item.Temperature = w.coord.execution.Temperature

	// Check for cancellation before LLM call
	select {
	case <-ctx.Done():
		w.markCancelled(ctx, item)
		return
	case <-w.coord.runCtx.cancelCh:
		w.markCancelled(ctx, item)
		return
	default:
	}

	// Invoke LLM
	startTime := time.Now()
	llmResp, err := w.llm.Invoke(ctx, prompt, w.coord.execution.Model, w.coord.execution.Temperature)
	if err != nil {
		result.Status = RunStatusFailed
		result.Error = fmt.Sprintf("LLM invocation failed: %v", err)
		item.Status = domain.WorkItemStatusFailed
		item.Error = result.Error
		w.coord.results <- result
		return
	}

	item.Response = llmResp.Content
	item.TokensIn = llmResp.TokensIn
	item.TokensOut = llmResp.TokensOut
	item.DurationMs = int(time.Since(startTime).Milliseconds())

	// Run deterministic checks
	deterministicScore := 1.0
	if w.runner != nil && len(evalCase.Rubric.Checks) > 0 {
		checks := make([]DeterministicCheck, len(evalCase.Rubric.Checks))
		for i, check := range evalCase.Rubric.Checks {
			checks[i] = DeterministicCheck{
				ID:          check.ID,
				Type:        "content_contains",
				Expected:    check.Description,
			}
		}

		detResult, err := w.runner.RunDeterministic(ctx, nil, checks)
		if err == nil {
			deterministicScore = detResult.Score
		}
	}

	// Run rubric evaluation
	rubricScore := 0
	rubricDetails := make([]RubricCheckResult, 0)
	if w.runner != nil && w.llm != nil && len(evalCase.Rubric.Checks) > 0 {
		rubric := Rubric{
			MaxScore: evalCase.Rubric.MaxScore,
			Checks:   make([]RubricCheck, len(evalCase.Rubric.Checks)),
		}
		for i, c := range evalCase.Rubric.Checks {
			rubric.Checks[i] = RubricCheck{
				ID:          c.ID,
				Description: c.Description,
				Weight:      c.Weight,
			}
		}

		rubricResult, err := w.runner.RunRubric(ctx, item.Response, rubric, w.llm, item.Model)
		if err == nil {
			rubricScore = rubricResult.Score
			for _, detail := range rubricResult.Details {
				rubricDetails = append(rubricDetails, RubricCheckResult{
					CheckID: detail.CheckID,
					Passed:  detail.Passed,
					Score:   detail.Score,
					Details: detail.Details,
				})
			}
		}
	}

	// Create the eval run record
	run := domain.NewEvalRun(evalCase.ID, domain.MustNewID(w.coord.execution.SnapshotID))
	run.DeterministicScore = deterministicScore
	run.RubricScore = rubricScore
	run.RubricDetails = make([]domain.RubricCheckResult, len(rubricDetails))
	for i, rd := range rubricDetails {
		run.RubricDetails[i] = domain.RubricCheckResult{
			CheckID: rd.CheckID,
			Passed:  rd.Passed,
			Score:   rd.Score,
			Details: rd.Details,
		}
	}
	run.TokenInput = item.TokensIn
	run.TokenOutput = item.TokensOut
	run.DurationMs = int64(item.DurationMs)

	// Determine pass/fail
	passed := deterministicScore >= 0.8 && rubricScore >= evalCase.Rubric.MaxScore*80/100
	run.Complete(deterministicScore, rubricScore, passed)

	// Save the run
	if w.coord.runRepo != nil {
		if err := w.coord.runRepo.Create(ctx, run); err != nil {
			slog.Warn("failed to save run", "error", err)
		}
	}

	// Update work item with final status
	item.Status = domain.WorkItemStatusCompleted
	now := time.Now()
	if w.coord.workItemRepo != nil {
		w.coord.workItemRepo.UpdateResult(ctx, item.ID, domain.WorkItemStatusCompleted, item.Response, item.TokensIn, item.TokensOut, item.DurationMs, item.Error, now)
	}

	result.Status = RunStatusCompleted
	result.DeterministicScore = deterministicScore
	result.RubricScore = rubricScore
	result.RubricDetails = rubricDetails
	w.coord.results <- result
}

// markCancelled marks a work item as cancelled.
func (w *Worker) markCancelled(ctx context.Context, item *domain.EvalWorkItem) {
	item.Status = domain.WorkItemStatusCancelled
	now := time.Now()
	item.CompletedAt = &now
	item.Error = "cancelled"
	if w.coord.workItemRepo != nil {
		w.coord.workItemRepo.UpdateResult(ctx, item.ID, domain.WorkItemStatusCancelled, "", 0, 0, 0, "cancelled", now)
	}
	w.coord.results <- &RunResult{
		WorkItem: item,
		Status:  RunStatusCancelled,
		Error:   "cancelled",
	}
}

// hashPrompt creates a SHA256 hash of the prompt for idempotency checks.
func hashPrompt(prompt string) string {
	hash := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(hash[:8])
}
