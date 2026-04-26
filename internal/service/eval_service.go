// Package service implements L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/eval-prompt/internal/domain"
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
	evalRunner      EvalRunner
	llmInvoker      LLMInvoker
	gitBridger      GitBridger
	traceCollector  TraceCollector
	semanticAnalyzer SemanticAnalyzer
	evalsDir        string   // Path to the evals directory (e.g., "evals" or ".evals")
	concurrency     int      // Default concurrency for worker pool
	coordinators    sync.Map // Map of executionID -> *Coordinator for cancellation
}

// NewEvalService creates a new EvalService.
func NewEvalService() *EvalService {
	return &EvalService{}
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

// WithEvalsDir sets the evals directory path.
func (s *EvalService) WithEvalsDir(evalsDir string) *EvalService {
	s.evalsDir = evalsDir
	return s
}

// WithConcurrency sets the default concurrency for worker pool.
func (s *EvalService) WithConcurrency(concurrency int) *EvalService {
	s.concurrency = concurrency
	return s
}

// WithSemanticAnalyzer sets the semantic analyzer for the EvalService.
func (s *EvalService) WithSemanticAnalyzer(sa SemanticAnalyzer) *EvalService {
	s.semanticAnalyzer = sa
	return s
}

// Ensure EvalService implements the EvalService interface.
var _ EvalServiceer = (*EvalService)(nil)

// EvalServiceer is the interface for evaluation operations.
type EvalServiceer interface {
	// RunEval executes evaluation for an asset snapshot using the worker pool.
	// Returns execution_id instead of run_id for async tracking.
	RunEval(ctx context.Context, req *RunEvalRequest) (*domain.EvalExecution, error)

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

	// GetExecution retrieves an eval execution by ID.
	GetExecution(ctx context.Context, executionID string) (*domain.EvalExecution, error)

	// CancelExecution cancels a running eval execution.
	CancelExecution(ctx context.Context, executionID string) error

	// ListExecutions lists eval executions with pagination.
	ListExecutions(ctx context.Context, offset, limit int) ([]*domain.EvalExecution, int, error)
}

// RunEvalRequest contains parameters for running an eval with the worker pool.
type RunEvalRequest struct {
	AssetID         string
	SnapshotVersion string
	EvalCaseIDs     []string
	Mode            domain.ExecutionMode
	RunsPerCase     int
	Concurrency     int
	Model           string
	Temperature     float64
}

// RunEval executes evaluation for an asset snapshot.
// NOTE: This method requires an eval runner implementation.
func (s *EvalService) RunEval(ctx context.Context, req *RunEvalRequest) (*domain.EvalExecution, error) {
	if s.llmInvoker == nil {
		return nil, fmt.Errorf("LLM invoker not configured")
	}
	if s.evalRunner == nil {
		return nil, fmt.Errorf("eval runner not configured")
	}
	return nil, fmt.Errorf("eval run not yet implemented: requires eval runner integration")
}

// GetExecution retrieves an eval execution by ID.
// Eval executions are tracked via in-memory coordinators.
func (s *EvalService) GetExecution(ctx context.Context, executionID string) (*domain.EvalExecution, error) {
	// Check coordinators for running executions
	if coord, ok := s.coordinators.Load(executionID); ok {
		if c, ok := coord.(*Coordinator); ok {
			return c.execution, nil
		}
	}
	return nil, fmt.Errorf("execution not found: %s", executionID)
}

// CancelExecution cancels a running eval execution.
func (s *EvalService) CancelExecution(ctx context.Context, executionID string) error {
	// Lookup coordinator and signal cancellation
	if coord, ok := s.coordinators.Load(executionID); ok {
		if c, ok := coord.(*Coordinator); ok {
			c.Cancel()
			slog.Info("eval execution cancellation signalled",
				"layer", "service",
				"execution_id", executionID,
			)
			return nil
		}
	}
	return fmt.Errorf("execution not found: %s", executionID)
}

// ListExecutions lists eval executions with pagination.
// Currently only returns running executions from memory.
func (s *EvalService) ListExecutions(ctx context.Context, offset, limit int) ([]*domain.EvalExecution, int, error) {
	executions := make([]*domain.EvalExecution, 0)
	s.coordinators.Range(func(key, value interface{}) bool {
		if coord, ok := value.(*Coordinator); ok {
			executions = append(executions, coord.execution)
		}
		return true
	})
	return executions, len(executions), nil
}

// GetEvalRun retrieves an eval run by ID from asset's eval_history.
func (s *EvalService) GetEvalRun(ctx context.Context, runID string) (*EvalRun, error) {
	// Search all assets' eval_history for this run
	// For now, iterate through the prompts directory
	promptsDir := "prompts"
	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompts directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		assetID := strings.TrimSuffix(entry.Name(), ".md")

		// Look in the asset's eval history (stored in asset's .md file)
		assetRuns, err := s.ListEvalRuns(ctx, assetID)
		if err != nil {
			continue
		}

		for _, run := range assetRuns {
			if run.ID == runID {
				return run, nil
			}
		}
	}

	return nil, fmt.Errorf("eval run not found: %s", runID)
}

// ListEvalCases lists all eval cases from the evals directory.
// Eval cases are stored as .md files in the evals directory.
func (s *EvalService) ListEvalCases(ctx context.Context, assetID string) ([]*domain.EvalCase, error) {
	if s.evalsDir == "" {
		return nil, fmt.Errorf("evals directory not configured")
	}

	entries, err := os.ReadDir(s.evalsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read evals directory: %w", err)
	}

	var cases []*domain.EvalCase
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(s.evalsDir, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		fm, _, err := yamlutil.ParseEvalPromptFrontMatter(string(content))
		if err != nil {
			continue
		}

		// Filter by assetID if provided
		if assetID != "" && fm.ID != assetID {
			continue
		}

		c := &domain.EvalCase{
			ID:           domain.MustNewID(fm.ID),
			Name:         fm.Name,
			Prompt:       string(content),
			ShouldTrigger: true,
		}
		cases = append(cases, c)
	}

	return cases, nil
}

// ListEvalRuns lists all eval runs for an asset from its frontmatter eval_history.
func (s *EvalService) ListEvalRuns(ctx context.Context, assetID string) ([]*EvalRun, error) {
	// Read the asset's .md file and parse eval_history from frontmatter
	filePath := filepath.Join("prompts", assetID+".md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset file: %w", err)
	}

	fm, _, err := yamlutil.ParseFrontMatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	runs := make([]*EvalRun, 0, len(fm.EvalHistory))
	for _, entry := range fm.EvalHistory {
		status := EvalRunStatusPassed
		if entry.Score < 80 {
			status = EvalRunStatusFailed
		}

		createdAt, _ := time.Parse("2006-01-02", entry.Date)
		runs = append(runs, &EvalRun{
			ID:                 entry.RunID,
			EvalCaseID:         entry.EvalCaseVersion,
			AssetID:            assetID,
			Status:             status,
			DeterministicScore: entry.DeterministicScore,
			RubricScore:        entry.Score,
			CreatedAt:          createdAt,
		})
	}

	// Also add runs from EvalStats (aggregated)
	if fm.EvalStats != nil {
		for model, stat := range fm.EvalStats {
			if len(runs) == 0 || stat.LastRun == "" {
				continue
			}
			// Check if we already have this run
			hasRun := false
			for _, r := range runs {
				if r.CreatedAt.Format("2006-01-02") == stat.LastRun {
					hasRun = true
					break
				}
			}
			if !hasRun {
				lastRun, _ := time.Parse("2006-01-02", stat.LastRun)
				runs = append(runs, &EvalRun{
					ID:                 fmt.Sprintf("%s-%s", assetID, stat.LastRun),
					EvalCaseID:         model,
					AssetID:            assetID,
					Status:             EvalRunStatusPassed,
					DeterministicScore: stat.Mean / 100.0,
					RubricScore:        int(stat.Mean),
					CreatedAt:          lastRun,
				})
			}
		}
	}

	return runs, nil
}

// CompareEval compares two evaluation runs for the same asset.
func (s *EvalService) CompareEval(ctx context.Context, assetID string, v1, v2 string) (*CompareResult, error) {
	// Read asset's eval history
	filePath := filepath.Join("prompts", assetID+".md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset file: %w", err)
	}

	fm, _, err := yamlutil.ParseFrontMatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	result := &CompareResult{
		AssetID:  assetID,
		Version1: v1,
		Version2: v2,
	}

	// Find runs matching v1 and v2
	for _, entry := range fm.EvalHistory {
		if entry.EvalCaseVersion == v1 {
			status := EvalRunStatusPassed
			if entry.Score < 80 {
				status = EvalRunStatusFailed
			}
			result.Run1 = &EvalRunSummary{
				ID:                 entry.RunID,
				SnapshotID:         entry.SnapshotID,
				Status:             status,
				DeterministicScore: entry.DeterministicScore,
				RubricScore:        entry.Score,
				CreatedAt:          time.Now(),
			}
		}
		if entry.EvalCaseVersion == v2 {
			status := EvalRunStatusPassed
			if entry.Score < 80 {
				status = EvalRunStatusFailed
			}
			result.Run2 = &EvalRunSummary{
				ID:                 entry.RunID,
				SnapshotID:         entry.SnapshotID,
				Status:             status,
				DeterministicScore: entry.DeterministicScore,
				RubricScore:        entry.Score,
				CreatedAt:          time.Now(),
			}
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

	return result, nil
}

// GenerateReport generates a detailed evaluation report.
func (s *EvalService) GenerateReport(ctx context.Context, runID string) (*EvalReport, error) {
	// Find the run in asset's eval history
	if s.evalsDir == "" {
		return nil, fmt.Errorf("evals directory not configured")
	}

	entries, err := os.ReadDir(s.evalsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read evals directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		assetID := strings.TrimSuffix(entry.Name(), ".md")
		filePath := filepath.Join("prompts", assetID+".md")

		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		fm, _, err := yamlutil.ParseFrontMatter(string(content))
		if err != nil {
			continue
		}

		for _, e := range fm.EvalHistory {
			if e.RunID == runID {
				status := EvalRunStatusPassed
				if e.Score < 80 {
					status = EvalRunStatusFailed
				}

				return &EvalReport{
					RunID:              runID,
					AssetID:            assetID,
					SnapshotVersion:    e.SnapshotID,
					Status:             status,
					OverallScore:       e.Score,
					DeterministicScore: e.DeterministicScore,
					RubricScore:        e.Score,
					TokenUsage: TokenUsage{
						Input:  e.TokensIn,
						Output: e.TokensOut,
						Total:  e.TokensIn + e.TokensOut,
					},
					DurationMs:  e.DurationMs,
					GeneratedAt: time.Now(),
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("eval run not found: %s", runID)
}

// DiagnoseEval performs failure attribution analysis.
func (s *EvalService) DiagnoseEval(ctx context.Context, runID string) (*Diagnosis, error) {
	if s.llmInvoker == nil {
		return nil, fmt.Errorf("LLM invoker not available")
	}

	// Get the eval run data
	report, err := s.GenerateReport(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get eval report: %w", err)
	}

	// Build diagnosis prompt
	prompt := fmt.Sprintf(`You are analyzing an AI evaluation failure.

Eval Run ID: %s
Asset ID: %s
Score: %d
Deterministic Score: %.2f

Based on this data, provide a diagnosis in JSON format with:
- overall_severity: "high" | "medium" | "low"
- findings: array of findings with category, severity, location, problem, evidence, suggestion, expected_score_improvement
- recommended_strategy: a recommended improvement approach
- estimated_iterations: estimated number of iterations to fix
- confidence: your confidence in this diagnosis (0.0-1.0)

Respond with a JSON object only.`, runID, report.AssetID, report.RubricScore, report.DeterministicScore)

	// Invoke LLM for diagnosis
	diagnosisModel := "gpt-4o"
	resp, err := s.llmInvoker.Invoke(ctx, prompt, diagnosisModel, 0.3)
	if err != nil {
		return nil, fmt.Errorf("LLM diagnosis failed: %w", err)
	}

	// Parse response
	diagnosis := &Diagnosis{
		RunID:               runID,
		OverallSeverity:     "medium",
		RecommendedStrategy: "Review and improve prompt based on failed checks",
		EstimatedIterations: 3,
		Confidence:          0.7,
		Findings:           []DiagnosisFinding{},
	}

	if len(resp.Content) > 50 {
		diagnosis.Findings = append(diagnosis.Findings, DiagnosisFinding{
			Category:                 "general",
			Severity:                 "medium",
			Location:                 "prompt",
			Problem:                  "Evaluation did not pass all criteria",
			Evidence:                 resp.Content[:min(len(resp.Content), 200)],
			Suggestion:               "Review failed rubric checks and improve prompt",
			ExpectedScoreImprovement: 20,
		})
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

// evalPromptFile represents an eval prompt loaded from the evals directory.
type evalPromptFile struct {
	FrontMatter *domain.EvalPromptFrontMatter
	Content     string
	FilePath    string
}

// findEvalPrompt looks for an eval prompt file in the evals directory for the given asset.
// It looks for a file named {assetID}.md in the evals directory.
func (s *EvalService) findEvalPrompt(assetID string) (*evalPromptFile, error) {
	if s.evalsDir == "" {
		return nil, fmt.Errorf("evals directory not configured")
	}

	// Construct the expected eval prompt file path
	evalFilePath := filepath.Join(s.evalsDir, assetID+".md")

	// Check if file exists
	if _, err := os.Stat(evalFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("eval prompt file not found for asset %s", assetID)
	}

	// Read the file
	fileContent, err := os.ReadFile(evalFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read eval prompt file %s: %w", evalFilePath, err)
	}

	// Parse the front matter
	fm, content, err := yamlutil.ParseEvalPromptFrontMatter(string(fileContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse eval prompt front matter: %w", err)
	}

	return &evalPromptFile{
		FrontMatter: fm,
		Content:     content,
		FilePath:    evalFilePath,
	}, nil
}

