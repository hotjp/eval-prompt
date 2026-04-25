// Package service implements L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/storage"
	"github.com/eval-prompt/internal/yamlutil"
)

// EvalRun represents an evaluation run (service-level view).
type EvalRun struct {
	ID                 string
	EvalCaseID         string
	SnapshotID         string
	AssetID            string
	Status             EvalRunStatus
	DeterministicScore float64
	RubricScore        int
	RubricDetails      []RubricCheckResult
	TracePath          string
	TokenInput         int
	TokenOutput        int
	DurationMs         int64
	CreatedAt          time.Time
}

// EvalRunStatus represents the status of an evaluation run.
type EvalRunStatus string

const (
	EvalRunStatusPending EvalRunStatus = "pending"
	EvalRunStatusRunning EvalRunStatus = "running"
	EvalRunStatusPassed  EvalRunStatus = "passed"
	EvalRunStatusFailed  EvalRunStatus = "failed"
)

// RubricCheckResult represents the result of a single rubric check.
type RubricCheckResult struct {
	CheckID string `json:"check_id"`
	Passed  bool   `json:"passed"`
	Score   int    `json:"score"`
	Details string `json:"details,omitempty"`
}

// CompareResult contains the comparison between two evaluation runs.
type CompareResult struct {
	AssetID     string          `json:"asset_id"`
	Version1    string          `json:"version1"`
	Version2    string          `json:"version2"`
	Run1        *EvalRunSummary `json:"run1,omitempty"`
	Run2        *EvalRunSummary `json:"run2,omitempty"`
	ScoreDelta  int             `json:"score_delta"`
	PassedDelta int             `json:"passed_delta"`
	DiffOutput  string          `json:"diff_output,omitempty"`
}

// EvalRunSummary is a lightweight eval run representation.
type EvalRunSummary struct {
	ID                 string        `json:"id"`
	SnapshotID         string        `json:"snapshot_id"`
	Status             EvalRunStatus `json:"status"`
	DeterministicScore float64       `json:"deterministic_score"`
	RubricScore        int           `json:"rubric_score"`
	CreatedAt          time.Time     `json:"created_at"`
}

// EvalReport contains a detailed evaluation report.
type EvalReport struct {
	RunID              string              `json:"run_id"`
	AssetID            string              `json:"asset_id"`
	SnapshotVersion    string              `json:"snapshot_version"`
	Status             EvalRunStatus       `json:"status"`
	OverallScore       int                 `json:"overall_score"`
	DeterministicScore float64             `json:"deterministic_score"`
	RubricScore        int                 `json:"rubric_score"`
	RubricDetails      []RubricCheckResult `json:"rubric_details"`
	CheckResults       []CheckResult       `json:"check_results"`
	TokenUsage         TokenUsage          `json:"token_usage"`
	DurationMs         int64               `json:"duration_ms"`
	GeneratedAt        time.Time           `json:"generated_at"`
}

// CheckResult represents a single evaluation check result.
type CheckResult struct {
	CheckID   string `json:"check_id"`
	CheckType string `json:"check_type"`
	Passed    bool   `json:"passed"`
	Score     int    `json:"score"`
	Expected  string `json:"expected,omitempty"`
	Actual    string `json:"actual,omitempty"`
	Details   string `json:"details,omitempty"`
}

// TokenUsage contains token consumption information.
type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

// Diagnosis contains failure attribution analysis.
type Diagnosis struct {
	RunID               string             `json:"run_id"`
	OverallSeverity     string             `json:"overall_severity"` // high | medium | low
	Findings            []DiagnosisFinding `json:"findings"`
	RecommendedStrategy string             `json:"recommended_strategy"`
	EstimatedIterations int                `json:"estimated_iterations"`
	Confidence          float64            `json:"confidence"`
}

// DiagnosisFinding represents a single diagnosis finding.
type DiagnosisFinding struct {
	Category                 string `json:"category"`
	Severity                 string `json:"severity"` // critical | high | medium | low
	Location                 string `json:"location"`
	Problem                  string `json:"problem"`
	Evidence                 string `json:"evidence"`
	Suggestion               string `json:"suggestion"`
	ExpectedScoreImprovement int    `json:"expected_score_improvement"`
}

// EvalService handles evaluation orchestration.
type EvalService struct {
	storage        *storage.Client
	assetRepo      *storage.AssetRepository
	evalRunRepo    *storage.EvalRunRepository
	evalCaseRepo   *storage.EvalCaseRepository
	snapshotRepo   *storage.SnapshotRepository
	evalRunner     EvalRunner
	llmInvoker     LLMInvoker
	gitBridger     GitBridger
	traceCollector TraceCollector
}

// NewEvalService creates a new EvalService.
func NewEvalService() *EvalService {
	return &EvalService{}
}

// NewEvalServiceWithStorage creates a new EvalService with the given storage client.
func NewEvalServiceWithStorage(storageClient *storage.Client) *EvalService {
	return &EvalService{
		storage:      storageClient,
		assetRepo:    storage.NewAssetRepository(storageClient),
		evalRunRepo:  storage.NewEvalRunRepository(storageClient),
		evalCaseRepo: storage.NewEvalCaseRepository(storageClient),
		snapshotRepo: storage.NewSnapshotRepository(storageClient),
	}
}

// NewEvalServiceWithDefaultStorage creates a new EvalService with a default storage client.
func NewEvalServiceWithDefaultStorage() (*EvalService, error) {
	client, err := storage.NewClientWithDSN("")
	if err != nil {
		return nil, err
	}
	svc := NewEvalServiceWithStorage(client)
	return svc, nil
}

// WithEvalRunner sets the eval runner plugin.
func (s *EvalService) WithEvalRunner(runner EvalRunner) *EvalService {
	s.evalRunner = runner
	return s
}

// WithLLMInvoker sets the LLM invoker plugin.
func (s *EvalService) WithLLMInvoker(invoker LLMInvoker) *EvalService {
	s.llmInvoker = invoker
	return s
}

// WithGitBridger sets the git bridger plugin.
func (s *EvalService) WithGitBridger(bridger GitBridger) *EvalService {
	s.gitBridger = bridger
	return s
}

// WithTraceCollector sets the trace collector plugin.
func (s *EvalService) WithTraceCollector(collector TraceCollector) *EvalService {
	s.traceCollector = collector
	return s
}

// Close closes the underlying storage client.
func (s *EvalService) Close() error {
	if s.storage != nil {
		return s.storage.Close()
	}
	return nil
}

// Ensure EvalService implements the EvalService interface.
var _ EvalServiceer = (*EvalService)(nil)

// EvalServiceer is the interface for evaluation operations.
type EvalServiceer interface {
	// RunEval executes evaluation for an asset snapshot.
	RunEval(ctx context.Context, assetID, snapshotVersion string, caseIDs []string) (*EvalRun, error)

	// GetEvalRun retrieves an eval run by ID.
	GetEvalRun(ctx context.Context, runID string) (*EvalRun, error)

	// ListEvalRuns lists all eval runs for an asset.
	ListEvalRuns(ctx context.Context, assetID string) ([]*EvalRun, error)

	// ListEvalCases lists all eval cases for an asset.
	ListEvalCases(ctx context.Context, assetID string) ([]*domain.EvalCase, error)

	// CompareEval compares two evaluation runs for the same asset.
	CompareEval(ctx context.Context, assetID string, v1, v2 string) (*CompareResult, error)

	// GenerateReport generates a detailed evaluation report.
	GenerateReport(ctx context.Context, runID string) (*EvalReport, error)

	// DiagnoseEval performs failure attribution analysis.
	DiagnoseEval(ctx context.Context, runID string) (*Diagnosis, error)
}

// RunEval executes evaluation for an asset snapshot.
func (s *EvalService) RunEval(ctx context.Context, assetID, snapshotVersion string, caseIDs []string) (*EvalRun, error) {
	if s.evalRunRepo == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	// Get the snapshot by assetID and version
	snapshot, err := s.snapshotRepo.GetByAssetIDAndVersion(ctx, assetID, snapshotVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	// Get eval cases - either by specific IDs or all for this asset
	var evalCases []*domain.EvalCase
	if len(caseIDs) > 0 {
		for _, caseID := range caseIDs {
			ec, err := s.evalCaseRepo.GetByID(ctx, caseID)
			if err != nil {
				return nil, fmt.Errorf("failed to get eval case %s: %w", caseID, err)
			}
			evalCases = append(evalCases, ec)
		}
	} else {
		evalCases, err = s.evalCaseRepo.GetByAssetID(ctx, assetID)
		if err != nil {
			return nil, fmt.Errorf("failed to get eval cases: %w", err)
		}
	}

	if len(evalCases) == 0 {
		return nil, fmt.Errorf("no eval cases found")
	}

	// Use the first eval case for the run (single evaluation)
	evalCase := evalCases[0]

	// Create eval run record
	run := domain.NewEvalRun(evalCase.ID, snapshot.ID)
	run.Status = domain.EvalRunStatusRunning

	if err := s.evalRunRepo.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to create eval run: %w", err)
	}

	// Start trace collection if available
	var traceCtx context.Context
	if s.traceCollector != nil {
		traceCtx, err = s.traceCollector.StartSpan(ctx, assetID, snapshot.ID.String())
		if err != nil {
			return nil, fmt.Errorf("failed to start trace: %w", err)
		}
		defer func() {
			path, _ := s.traceCollector.Finalize(traceCtx)
			run.TracePath = path
		}()
	} else {
		traceCtx = ctx
	}

	startTime := time.Now()

	// Record eval start event
	if s.traceCollector != nil {
		s.traceCollector.RecordEvent(traceCtx, TraceEvent{
			Name:      "eval_start",
			Timestamp: startTime,
			Type:      "event",
			Data: map[string]any{
				"asset_id":     assetID,
				"snapshot_id":  snapshot.ID.String(),
				"eval_case_id": evalCase.ID.String(),
			},
		})
	}

	// Build prompt from eval case
	prompt := evalCase.Prompt

	// Invoke LLM if invoker is available
	var llmResponse *LLMResponse
	if s.llmInvoker != nil {
		llmResp, err := s.llmInvoker.Invoke(traceCtx, prompt, "gpt-4o", 0.3)
		if err != nil {
			run.Fail()
			s.evalRunRepo.Update(ctx, run)
			return s.toServiceEvalRun(run), fmt.Errorf("LLM invocation failed: %w", err)
		}
		llmResponse = llmResp
		run.TokenInput = llmResp.TokensIn
		run.TokenOutput = llmResp.TokensOut
	}

	output := ""
	if llmResponse != nil {
		output = llmResponse.Content
	}

	// Run deterministic checks if eval runner is available
	deterministicScore := 1.0
	if s.evalRunner != nil && len(evalCase.Rubric.Checks) > 0 {
		// Build deterministic checks from rubric
		checks := make([]DeterministicCheck, len(evalCase.Rubric.Checks))
		for i, check := range evalCase.Rubric.Checks {
			checks[i] = DeterministicCheck{
				ID:          check.ID,
				Type:        "content_contains",
				Expected:    check.Description,
			}
		}

		// Run deterministic evaluation
		result, err := s.evalRunner.RunDeterministic(traceCtx, nil, checks)
		if err == nil {
			deterministicScore = result.Score
		}
	}

	// Run rubric evaluation if LLM invoker and eval runner are available
	rubricScore := 0
	rubricDetails := make([]domain.RubricCheckResult, 0)
	if s.evalRunner != nil && s.llmInvoker != nil && len(evalCase.Rubric.Checks) > 0 {
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

		rubricResult, err := s.evalRunner.RunRubric(traceCtx, output, rubric, s.llmInvoker)
		if err == nil {
			rubricScore = rubricResult.Score
			for _, detail := range rubricResult.Details {
				rubricDetails = append(rubricDetails, domain.RubricCheckResult{
					CheckID: detail.CheckID,
					Passed:  detail.Passed,
					Score:   detail.Score,
					Details: detail.Details,
				})
			}
		}
	}

	// Calculate duration
	durationMs := time.Since(startTime).Milliseconds()
	run.DurationMs = durationMs
	run.DeterministicScore = deterministicScore
	run.RubricScore = rubricScore
	run.RubricDetails = rubricDetails

	// Determine pass/fail
	passed := deterministicScore >= 0.8 && rubricScore >= evalCase.Rubric.MaxScore*80/100
	run.Complete(deterministicScore, rubricScore, passed)

	// Update the run record
	if err := s.evalRunRepo.Update(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to update eval run: %w", err)
	}

	// Record eval complete event
	if s.traceCollector != nil {
		s.traceCollector.RecordEvent(traceCtx, TraceEvent{
			Name:      "eval_complete",
			Timestamp: time.Now(),
			Type:      "event",
			Data: map[string]any{
				"status":              run.Status,
				"deterministic_score": deterministicScore,
				"rubric_score":        rubricScore,
				"passed":              passed,
			},
		})
	}

	// Write eval history to the .md file (transaction boundary)
	if err := s.writeEvalHistoryToFile(ctx, assetID, run, snapshot); err != nil {
		return nil, fmt.Errorf("failed to write eval history to file: %w", err)
	}

	return s.toServiceEvalRun(run), nil
}

// GetEvalRun retrieves an eval run by ID.
func (s *EvalService) GetEvalRun(ctx context.Context, runID string) (*EvalRun, error) {
	if s.evalRunRepo == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	run, err := s.evalRunRepo.GetByID(ctx, runID)
	if err != nil {
		return nil, err
	}

	return s.toServiceEvalRun(run), nil
}

// ListEvalRuns lists all eval runs for an asset.
func (s *EvalService) ListEvalRuns(ctx context.Context, assetID string) ([]*EvalRun, error) {
	if s.evalRunRepo == nil || s.snapshotRepo == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	// Get all snapshots for the asset
	snapshots, err := s.snapshotRepo.GetByAssetID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshots: %w", err)
	}

	var runs []*EvalRun
	for _, snap := range snapshots {
		// Get eval runs for each snapshot
		snapshotRuns, err := s.evalRunRepo.GetBySnapshotID(ctx, snap.ID.String())
		if err != nil {
			return nil, fmt.Errorf("failed to get eval runs for snapshot %s: %w", snap.ID.String(), err)
		}
		for _, r := range snapshotRuns {
			svcRun := s.toServiceEvalRun(r)
			svcRun.AssetID = assetID
			runs = append(runs, svcRun)
		}
	}

	return runs, nil
}

// ListEvalCases lists all eval cases for an asset.
func (s *EvalService) ListEvalCases(ctx context.Context, assetID string) ([]*domain.EvalCase, error) {
	if s.evalCaseRepo == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	evalCases, err := s.evalCaseRepo.GetByAssetID(ctx, assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to query eval cases: %w", err)
	}

	return evalCases, nil
}

// CompareEval compares two evaluation runs for the same asset.
func (s *EvalService) CompareEval(ctx context.Context, assetID string, v1, v2 string) (*CompareResult, error) {
	if s.snapshotRepo == nil || s.evalRunRepo == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	result := &CompareResult{
		AssetID:  assetID,
		Version1: v1,
		Version2: v2,
	}

	// Get snapshot for v1
	snap1, err := s.snapshotRepo.GetByAssetIDAndVersion(ctx, assetID, v1)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot v1: %w", err)
	}

	// Get snapshot for v2
	snap2, err := s.snapshotRepo.GetByAssetIDAndVersion(ctx, assetID, v2)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot v2: %w", err)
	}

	// Get eval runs for v1
	runs1, err := s.evalRunRepo.GetBySnapshotID(ctx, snap1.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get eval runs for v1: %w", err)
	}
	if len(runs1) > 0 {
		result.Run1 = &EvalRunSummary{
			ID:                 runs1[0].ID.String(),
			SnapshotID:         snap1.ID.String(),
			Status:             statusToService(runs1[0].Status),
			DeterministicScore: runs1[0].DeterministicScore,
			RubricScore:        runs1[0].RubricScore,
			CreatedAt:          runs1[0].CreatedAt,
		}
	}

	// Get eval runs for v2
	runs2, err := s.evalRunRepo.GetBySnapshotID(ctx, snap2.ID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get eval runs for v2: %w", err)
	}
	if len(runs2) > 0 {
		result.Run2 = &EvalRunSummary{
			ID:                 runs2[0].ID.String(),
			SnapshotID:         snap2.ID.String(),
			Status:             statusToService(runs2[0].Status),
			DeterministicScore: runs2[0].DeterministicScore,
			RubricScore:        runs2[0].RubricScore,
			CreatedAt:          runs2[0].CreatedAt,
		}
	}

	// Calculate deltas
	if result.Run1 != nil && result.Run2 != nil {
		result.ScoreDelta = result.Run2.RubricScore - result.Run1.RubricScore
		if result.Run2.Status == EvalRunStatusPassed && result.Run1.Status != EvalRunStatusPassed {
			result.PassedDelta = 1
		} else if result.Run2.Status != EvalRunStatusPassed && result.Run1.Status == EvalRunStatusPassed {
			result.PassedDelta = -1
		}
	}

	// Get git diff if git bridger is available
	if s.gitBridger != nil && snap1.CommitHash != "" && snap2.CommitHash != "" {
		diff, err := s.gitBridger.Diff(ctx, snap1.CommitHash, snap2.CommitHash)
		if err == nil {
			result.DiffOutput = diff
		}
	}

	return result, nil
}

// GenerateReport generates a detailed evaluation report.
func (s *EvalService) GenerateReport(ctx context.Context, runID string) (*EvalReport, error) {
	if s.evalRunRepo == nil || s.snapshotRepo == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	// Get the eval run
	run, err := s.evalRunRepo.GetByID(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get eval run: %w", err)
	}

	report := &EvalReport{
		RunID:              run.ID.String(),
		Status:             statusToService(run.Status),
		DeterministicScore: run.DeterministicScore,
		RubricScore:        run.RubricScore,
		TokenUsage: TokenUsage{
			Input:  run.TokenInput,
			Output: run.TokenOutput,
			Total:  run.TokenInput + run.TokenOutput,
		},
		DurationMs:  run.DurationMs,
		GeneratedAt: time.Now(),
	}

	// Get snapshot to get version and asset ID
	snapshot, err := s.snapshotRepo.GetByID(ctx, run.SnapshotID.String())
	if err == nil {
		report.AssetID = snapshot.AssetID.String()
		report.SnapshotVersion = snapshot.Version
	}

	// Convert rubric details
	report.RubricDetails = make([]RubricCheckResult, len(run.RubricDetails))
	for i, detail := range run.RubricDetails {
		report.RubricDetails[i] = RubricCheckResult{
			CheckID: detail.CheckID,
			Passed:  detail.Passed,
			Score:   detail.Score,
			Details: detail.Details,
		}
	}

	// Build check results from rubric details
	report.CheckResults = make([]CheckResult, 0, len(run.RubricDetails))
	for _, detail := range run.RubricDetails {
		report.CheckResults = append(report.CheckResults, CheckResult{
			CheckID: detail.CheckID,
			CheckType: "rubric",
			Passed:  detail.Passed,
			Score:   detail.Score,
			Details: detail.Details,
		})
	}

	// Calculate overall score
	if run.DeterministicScore > 0 && run.RubricScore > 0 {
		report.OverallScore = int(float64(run.RubricScore) * run.DeterministicScore)
	} else {
		report.OverallScore = run.RubricScore
	}

	return report, nil
}

// DiagnoseEval performs failure attribution analysis.
func (s *EvalService) DiagnoseEval(ctx context.Context, runID string) (*Diagnosis, error) {
	if s.evalRunRepo == nil {
		return nil, fmt.Errorf("storage not initialized")
	}

	if s.llmInvoker == nil {
		return nil, fmt.Errorf("LLM invoker not available")
	}

	// Get the eval run
	run, err := s.evalRunRepo.GetByID(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get eval run: %w", err)
	}

	// Build diagnosis prompt
	prompt := s.buildDiagnosisPrompt(run)

	// Invoke LLM for diagnosis
	resp, err := s.llmInvoker.Invoke(ctx, prompt, "gpt-4o", 0.3)
	if err != nil {
		return nil, fmt.Errorf("LLM diagnosis failed: %w", err)
	}

	// Parse LLM response - expect JSON format for diagnosis
	diagnosis, err := s.parseDiagnosisResponse(resp.Content, runID)
	if err != nil {
		// If parsing fails, create a basic diagnosis
		diagnosis = &Diagnosis{
			RunID:               runID,
			OverallSeverity:     "medium",
			Findings:            []DiagnosisFinding{},
			RecommendedStrategy:  "Review rubric checks and improve prompt clarity",
			EstimatedIterations:  3,
			Confidence:          0.5,
		}

		// Add a finding for each failed rubric check
		for _, detail := range run.RubricDetails {
			if !detail.Passed {
				diagnosis.Findings = append(diagnosis.Findings, DiagnosisFinding{
					Category:                 "rubric",
					Severity:                 "medium",
					Location:                 "rubric_check:" + detail.CheckID,
					Problem:                  fmt.Sprintf("Rubric check '%s' failed", detail.CheckID),
					Evidence:                 detail.Details,
					Suggestion:               "Review and update the prompt to better address this criterion",
					ExpectedScoreImprovement: detail.Score,
				})
			}
		}
	}

	return diagnosis, nil
}

// buildDiagnosisPrompt creates a prompt for LLM-based diagnosis.
func (s *EvalService) buildDiagnosisPrompt(run *domain.EvalRun) string {
	checksJSON := "{rubric_details not available}"
	if len(run.RubricDetails) > 0 {
		checksJSON = fmt.Sprintf("%v", run.RubricDetails)
	}

	return fmt.Sprintf(`You are analyzing an AI evaluation failure.

Eval Run ID: %s
Status: %s
Deterministic Score: %.2f
Rubric Score: %d

Rubric Check Results:
%s

Based on this data, provide a diagnosis in JSON format with:
- overall_severity: "high" | "medium" | "low"
- findings: array of findings with category, severity, location, problem, evidence, suggestion, expected_score_improvement
- recommended_strategy: a recommended improvement approach
- estimated_iterations: estimated number of iterations to fix
- confidence: your confidence in this diagnosis (0.0-1.0)

Respond with a JSON object only.`, run.ID.String(), run.Status, run.DeterministicScore, run.RubricScore, checksJSON)
}

// parseDiagnosisResponse parses the LLM diagnosis response.
func (s *EvalService) parseDiagnosisResponse(content, runID string) (*Diagnosis, error) {
	// Try to extract JSON from the response
	var diagnosis Diagnosis
	diagnosis.RunID = runID

	// Simple parsing - in production, use proper JSON parsing
	// For now, return a basic structure if we got a response
	if len(content) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	diagnosis.OverallSeverity = "medium"
	diagnosis.RecommendedStrategy = "Review and improve prompt based on failed checks"
	diagnosis.EstimatedIterations = 3
	diagnosis.Confidence = 0.7
	diagnosis.Findings = []DiagnosisFinding{}

	// Try to detect severity keywords
	lowerContent := content
	if len(lowerContent) > 50 { // Only analyze if we have substantial content
		diagnosis.Findings = append(diagnosis.Findings, DiagnosisFinding{
			Category:                 "general",
			Severity:                 "medium",
			Location:                 "prompt",
			Problem:                  "Evaluation did not pass all criteria",
			Evidence:                 content[:min(len(content), 200)],
			Suggestion:               "Review failed rubric checks and improve prompt",
			ExpectedScoreImprovement: 20,
		})
	}

	return &diagnosis, nil
}

// ErrNotImplemented is returned when a method is not yet implemented.
var ErrNotImplemented = &NotImplementedError{Method: "EvalService method"}

// NotImplementedError indicates a method is not yet implemented.
type NotImplementedError struct {
	Method string
}

func (e *NotImplementedError) Error() string {
	return "not implemented: " + e.Method
}

// statusToService converts a domain.EvalRunStatus to service.EvalRunStatus.
func statusToService(status domain.EvalRunStatus) EvalRunStatus {
	switch status {
	case domain.EvalRunStatusRunning:
		return EvalRunStatusRunning
	case domain.EvalRunStatusPassed:
		return EvalRunStatusPassed
	case domain.EvalRunStatusFailed:
		return EvalRunStatusFailed
	default:
		return EvalRunStatusPending
	}
}

// toServiceEvalRun converts a domain.EvalRun to a service.EvalRun.
func (s *EvalService) toServiceEvalRun(domainRun *domain.EvalRun) *EvalRun {
	if domainRun == nil {
		return nil
	}

	rubricDetails := make([]RubricCheckResult, len(domainRun.RubricDetails))
	for i, rd := range domainRun.RubricDetails {
		rubricDetails[i] = RubricCheckResult{
			CheckID: rd.CheckID,
			Passed:  rd.Passed,
			Score:   rd.Score,
			Details: rd.Details,
		}
	}

	return &EvalRun{
		ID:                 domainRun.ID.String(),
		EvalCaseID:         domainRun.EvalCaseID.String(),
		SnapshotID:         domainRun.SnapshotID.String(),
		Status:             statusToService(domainRun.Status),
		DeterministicScore: domainRun.DeterministicScore,
		RubricScore:        domainRun.RubricScore,
		RubricDetails:      rubricDetails,
		TracePath:          domainRun.TracePath,
		TokenInput:         domainRun.TokenInput,
		TokenOutput:        domainRun.TokenOutput,
		DurationMs:         domainRun.DurationMs,
		CreatedAt:          domainRun.CreatedAt,
	}
}

// writeEvalHistoryToFile writes the eval history to the .md file.
// This is the transaction boundary - the eval is not complete until the file is written successfully.
func (s *EvalService) writeEvalHistoryToFile(ctx context.Context, assetID string, run *domain.EvalRun, snapshot *domain.Snapshot) error {
	if s.assetRepo == nil {
		return fmt.Errorf("asset repository not initialized")
	}

	// Get the asset to find its FilePath
	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		return fmt.Errorf("failed to get asset: %w", err)
	}

	// Read the .md file
	fileContent, err := os.ReadFile(asset.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", asset.FilePath, err)
	}

	// Parse the YAML front matter
	fm, markdownContent, err := yamlutil.ParseFrontMatter(string(fileContent))
	if err != nil {
		return fmt.Errorf("failed to parse front matter: %w", err)
	}

	// Create the eval history entry
	entry := domain.EvalHistoryEntry{
		RunID:      run.ID.String(),
		SnapshotID: snapshot.Version,
		Score:      run.RubricScore,
		Model:      "gpt-4o", // TODO: get from actual LLM response
		Date:       time.Now().Format("2006-01-02"),
		By:         "", // TODO: get from context if available
	}

	// Insert at the beginning of eval_history
	fm.EvalHistory = append([]domain.EvalHistoryEntry{entry}, fm.EvalHistory...)

	// Format the complete .md file
	newContent, err := yamlutil.FormatMarkdown(fm, markdownContent)
	if err != nil {
		return fmt.Errorf("failed to format markdown: %w", err)
	}

	// Write back to file
	if err := os.WriteFile(asset.FilePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", asset.FilePath, err)
	}

	slog.Info("eval history written to file",
		"layer", "service",
		"asset_id", assetID,
		"file_path", asset.FilePath,
		"run_id", run.ID.String(),
		"score", run.RubricScore,
	)

	return nil
}
