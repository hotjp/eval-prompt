// Package mock provides mock implementations for testing service interfaces.
package mock

import (
	"context"
	"time"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/service"
)

// MockAssetIndexer is a mock implementation of service.AssetIndexer.
type MockAssetIndexer struct {
	MockAssetFileManager // embeds AssetFileManager methods (GetFrontmatter, UpdateFrontmatter, WriteContent, GetBody)
	SearchFunc          func(ctx context.Context, query string, filters service.SearchFilters) ([]service.AssetSummary, error)
	GetByIDFunc         func(ctx context.Context, id string) (*service.AssetDetail, error)
	SaveFunc            func(ctx context.Context, asset service.Asset) error
	DeleteFunc          func(ctx context.Context, id string) error
	CreatePlaceholderFunc func(ctx context.Context, id, name, bizLine string, tags []string) error
	GetFileContentFunc  func(ctx context.Context, id string) (string, error)
	SaveFileContentFunc func(ctx context.Context, id, content, commitMsg string) (string, error)
}

func (m *MockAssetIndexer) Search(ctx context.Context, query string, filters service.SearchFilters) ([]service.AssetSummary, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, filters)
	}
	return nil, nil
}

func (m *MockAssetIndexer) GetByID(ctx context.Context, id string) (*service.AssetDetail, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	// Return a valid default asset for testing
	return &service.AssetDetail{
		ID:          id,
		Name:        "Test Asset",
		Description: "Test description",
		BizLine:     "ai",
		Tags:        []string{"test"},
		State:       "created",
		Snapshots:   []service.SnapshotSummary{},
		Labels:      []service.LabelInfo{},
	}, nil
}

func (m *MockAssetIndexer) Save(ctx context.Context, asset service.Asset) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, asset)
	}
	return nil
}

func (m *MockAssetIndexer) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockAssetIndexer) Reconcile(ctx context.Context) (service.ReconcileReport, error) {
	return service.ReconcileReport{}, nil
}

func (m *MockAssetIndexer) CreatePlaceholder(ctx context.Context, id, name, bizLine string, tags []string) error {
	if m.CreatePlaceholderFunc != nil {
		return m.CreatePlaceholderFunc(ctx, id, name, bizLine, tags)
	}
	return nil
}

func (m *MockAssetIndexer) GetFileContent(ctx context.Context, id string) (string, error) {
	if m.GetFileContentFunc != nil {
		return m.GetFileContentFunc(ctx, id)
	}
	return "", nil
}

func (m *MockAssetIndexer) SaveFileContent(ctx context.Context, id, content, commitMsg string) (string, error) {
	if m.SaveFileContentFunc != nil {
		return m.SaveFileContentFunc(ctx, id, content, commitMsg)
	}
	return "mock-commit-hash", nil
}

// MockAssetFileManager is a mock implementation of service.AssetFileManager.
type MockAssetFileManager struct {
	GetFrontmatterFunc    func(ctx context.Context, id string) (*domain.FrontMatter, error)
	UpdateFrontmatterFunc func(ctx context.Context, id string, updater func(*domain.FrontMatter) error, commitMsg string) (string, error)
	WriteContentFunc      func(ctx context.Context, id string, updater func(*domain.FrontMatter) error, newBody string, commitMsg string) (string, error)
	GetBodyFunc           func(ctx context.Context, id string) (string, error)
}

func (m *MockAssetFileManager) GetFrontmatter(ctx context.Context, id string) (*domain.FrontMatter, error) {
	if m.GetFrontmatterFunc != nil {
		return m.GetFrontmatterFunc(ctx, id)
	}
	return &domain.FrontMatter{ID: id, Name: "Test Asset"}, nil
}

func (m *MockAssetFileManager) UpdateFrontmatter(ctx context.Context, id string, updater func(*domain.FrontMatter) error, commitMsg string) (string, error) {
	if m.UpdateFrontmatterFunc != nil {
		return m.UpdateFrontmatterFunc(ctx, id, updater, commitMsg)
	}
	return "mock-commit-hash", nil
}

func (m *MockAssetFileManager) WriteContent(ctx context.Context, id string, updater func(*domain.FrontMatter) error, newBody string, commitMsg string) (string, error) {
	if m.WriteContentFunc != nil {
		return m.WriteContentFunc(ctx, id, updater, newBody, commitMsg)
	}
	return "mock-commit-hash", nil
}

func (m *MockAssetFileManager) GetBody(ctx context.Context, id string) (string, error) {
	if m.GetBodyFunc != nil {
		return m.GetBodyFunc(ctx, id)
	}
	return "# Test Content", nil
}

// MockTriggerService is a mock implementation of service.TriggerServicer.
type MockTriggerService struct {
	MatchTriggerFunc         func(ctx context.Context, input string, top int) ([]*service.MatchedPrompt, error)
	ValidateAntiPatternsFunc func(ctx context.Context, prompt string) error
	InjectVariablesFunc      func(ctx context.Context, prompt string, vars map[string]string) (string, error)
}

func (m *MockTriggerService) MatchTrigger(ctx context.Context, input string, top int) ([]*service.MatchedPrompt, error) {
	if m.MatchTriggerFunc != nil {
		return m.MatchTriggerFunc(ctx, input, top)
	}
	return nil, nil
}

func (m *MockTriggerService) ValidateAntiPatterns(ctx context.Context, prompt string) error {
	if m.ValidateAntiPatternsFunc != nil {
		return m.ValidateAntiPatternsFunc(ctx, prompt)
	}
	return nil
}

func (m *MockTriggerService) InjectVariables(ctx context.Context, prompt string, vars map[string]string) (string, error) {
	if m.InjectVariablesFunc != nil {
		return m.InjectVariablesFunc(ctx, prompt, vars)
	}
	return "", nil
}

// MockEvalService is a mock implementation of service.EvalServiceer.
type MockEvalService struct {
	RunEvalFunc       func(ctx context.Context, assetID, snapshotVersion string, caseIDs []string) (*service.EvalRun, error)
	GetEvalRunFunc    func(ctx context.Context, runID string) (*service.EvalRun, error)
	ListEvalRunsFunc  func(ctx context.Context, assetID string) ([]*service.EvalRun, error)
	ListEvalCasesFunc func(ctx context.Context, assetID string) ([]*domain.EvalCase, error)
	CompareEvalFunc   func(ctx context.Context, assetID string, v1, v2 string) (*service.CompareResult, error)
	GenerateReportFunc func(ctx context.Context, runID string) (*service.EvalReport, error)
	DiagnoseEvalFunc  func(ctx context.Context, runID string) (*service.Diagnosis, error)
}

func (m *MockEvalService) RunEval(ctx context.Context, assetID, snapshotVersion string, caseIDs []string) (*service.EvalRun, error) {
	if m.RunEvalFunc != nil {
		return m.RunEvalFunc(ctx, assetID, snapshotVersion, caseIDs)
	}
	return &service.EvalRun{
		ID:        "test-run-id",
		Status:    service.EvalRunStatusRunning,
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockEvalService) GetEvalRun(ctx context.Context, runID string) (*service.EvalRun, error) {
	if m.GetEvalRunFunc != nil {
		return m.GetEvalRunFunc(ctx, runID)
	}
	return &service.EvalRun{
		ID:        runID,
		Status:    service.EvalRunStatusPassed,
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockEvalService) ListEvalRuns(ctx context.Context, assetID string) ([]*service.EvalRun, error) {
	if m.ListEvalRunsFunc != nil {
		return m.ListEvalRunsFunc(ctx, assetID)
	}
	return nil, nil
}

func (m *MockEvalService) ListEvalCases(ctx context.Context, assetID string) ([]*domain.EvalCase, error) {
	if m.ListEvalCasesFunc != nil {
		return m.ListEvalCasesFunc(ctx, assetID)
	}
	return nil, nil
}

func (m *MockEvalService) CompareEval(ctx context.Context, assetID string, v1, v2 string) (*service.CompareResult, error) {
	if m.CompareEvalFunc != nil {
		return m.CompareEvalFunc(ctx, assetID, v1, v2)
	}
	return nil, nil
}

func (m *MockEvalService) GenerateReport(ctx context.Context, runID string) (*service.EvalReport, error) {
	if m.GenerateReportFunc != nil {
		return m.GenerateReportFunc(ctx, runID)
	}
	return nil, nil
}

func (m *MockEvalService) DiagnoseEval(ctx context.Context, runID string) (*service.Diagnosis, error) {
	if m.DiagnoseEvalFunc != nil {
		return m.DiagnoseEvalFunc(ctx, runID)
	}
	return nil, nil
}

// MockGitBridger is a mock implementation of service.GitBridger.
type MockGitBridger struct {
	InitRepoFunc       func(ctx context.Context, path string) error
	StageAndCommitFunc func(ctx context.Context, filePath, message string) (string, error)
	DiffFunc           func(ctx context.Context, commit1, commit2 string) (string, error)
	LogFunc            func(ctx context.Context, filePath string, limit int) ([]service.CommitInfo, error)
	StatusFunc         func(ctx context.Context) (added, modified, deleted []string, err error)
}

func (m *MockGitBridger) InitRepo(ctx context.Context, path string) error {
	if m.InitRepoFunc != nil {
		return m.InitRepoFunc(ctx, path)
	}
	return nil
}

func (m *MockGitBridger) StageAndCommit(ctx context.Context, filePath, message string) (string, error) {
	if m.StageAndCommitFunc != nil {
		return m.StageAndCommitFunc(ctx, filePath, message)
	}
	return "mock-commit-hash", nil
}

func (m *MockGitBridger) Diff(ctx context.Context, commit1, commit2 string) (string, error) {
	if m.DiffFunc != nil {
		return m.DiffFunc(ctx, commit1, commit2)
	}
	return "", nil
}

func (m *MockGitBridger) Log(ctx context.Context, filePath string, limit int) ([]service.CommitInfo, error) {
	if m.LogFunc != nil {
		return m.LogFunc(ctx, filePath, limit)
	}
	return nil, nil
}

func (m *MockGitBridger) Status(ctx context.Context) (added, modified, deleted []string, err error) {
	if m.StatusFunc != nil {
		return m.StatusFunc(ctx)
	}
	return nil, nil, nil, nil
}

// MockLLMInvoker is a mock implementation of service.LLMInvoker.
type MockLLMInvoker struct {
	InvokeFunc         func(ctx context.Context, prompt string, model string, temperature float64) (*service.LLMResponse, error)
	InvokeWithSchemaFunc func(ctx context.Context, prompt string, schema []byte) ([]byte, error)
}

func (m *MockLLMInvoker) Invoke(ctx context.Context, prompt string, model string, temperature float64) (*service.LLMResponse, error) {
	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, prompt, model, temperature)
	}
	return &service.LLMResponse{
		Content:    "mock response",
		Model:     model,
		TokensIn:  100,
		TokensOut: 50,
		StopReason: "stop",
	}, nil
}

func (m *MockLLMInvoker) InvokeWithSchema(ctx context.Context, prompt string, schema []byte) ([]byte, error) {
	if m.InvokeWithSchemaFunc != nil {
		return m.InvokeWithSchemaFunc(ctx, prompt, schema)
	}
	return []byte(`{"result": "ok"}`), nil
}

// MockTraceCollector is a mock implementation of service.TraceCollector.
type MockTraceCollector struct {
	StartSpanFunc   func(ctx context.Context, assetID, snapshotID string) (context.Context, error)
	RecordEventFunc func(ctx context.Context, event service.TraceEvent) error
	FinalizeFunc    func(ctx context.Context) (string, error)
}

func (m *MockTraceCollector) StartSpan(ctx context.Context, assetID, snapshotID string) (context.Context, error) {
	if m.StartSpanFunc != nil {
		return m.StartSpanFunc(ctx, assetID, snapshotID)
	}
	return ctx, nil
}

func (m *MockTraceCollector) RecordEvent(ctx context.Context, event service.TraceEvent) error {
	if m.RecordEventFunc != nil {
		return m.RecordEventFunc(ctx, event)
	}
	return nil
}

func (m *MockTraceCollector) Finalize(ctx context.Context) (string, error) {
	if m.FinalizeFunc != nil {
		return m.FinalizeFunc(ctx)
	}
	return "/tmp/trace.jsonl", nil
}

// MockEvalRunner is a mock implementation of service.EvalRunner.
type MockEvalRunner struct {
	RunDeterministicFunc func(ctx context.Context, trace []service.TraceEvent, checks []service.DeterministicCheck) (service.DeterministicResult, error)
	RunRubricFunc        func(ctx context.Context, output string, rubric service.Rubric, invoker service.LLMInvoker) (service.RubricResult, error)
}

func (m *MockEvalRunner) RunDeterministic(ctx context.Context, trace []service.TraceEvent, checks []service.DeterministicCheck) (service.DeterministicResult, error) {
	if m.RunDeterministicFunc != nil {
		return m.RunDeterministicFunc(ctx, trace, checks)
	}
	return service.DeterministicResult{
		Passed:  true,
		Score:   1.0,
		Message: "all checks passed",
	}, nil
}

func (m *MockEvalRunner) RunRubric(ctx context.Context, output string, rubric service.Rubric, invoker service.LLMInvoker) (service.RubricResult, error) {
	if m.RunRubricFunc != nil {
		return m.RunRubricFunc(ctx, output, rubric, invoker)
	}
	// Default: all checks pass with full score
	details := make([]service.RubricCheckResult, len(rubric.Checks))
	for i, c := range rubric.Checks {
		details[i] = service.RubricCheckResult{
			CheckID: c.ID,
			Passed:  true,
			Score:   c.Weight,
			Details: "passed",
		}
	}
	return service.RubricResult{
		Score:    rubric.MaxScore,
		MaxScore: rubric.MaxScore,
		Passed:   true,
		Details:  details,
		Message:  "all rubric checks passed",
	}, nil
}

// MockModelAdapter is a mock implementation of service.ModelAdapter.
type MockModelAdapter struct {
	AdaptFunc           func(ctx context.Context, prompt service.PromptContent, sourceModel, targetModel string) (service.AdaptedPrompt, error)
	RecommendParamsFunc func(ctx context.Context, targetModel string, taskType string) (service.ModelParams, error)
	EstimateScoreFunc   func(ctx context.Context, promptID string, targetModel string) (float64, error)
	GetModelProfileFunc func(ctx context.Context, model string) (service.ModelProfile, error)
}

func (m *MockModelAdapter) Adapt(ctx context.Context, prompt service.PromptContent, sourceModel, targetModel string) (service.AdaptedPrompt, error) {
	if m.AdaptFunc != nil {
		return m.AdaptFunc(ctx, prompt, sourceModel, targetModel)
	}
	return service.AdaptedPrompt{
		Content:          prompt.Instruction,
		ParamAdjustments: map[string]float64{},
		FormatChanges:    []string{},
		Warnings:        []string{},
	}, nil
}

func (m *MockModelAdapter) RecommendParams(ctx context.Context, targetModel string, taskType string) (service.ModelParams, error) {
	if m.RecommendParamsFunc != nil {
		return m.RecommendParamsFunc(ctx, targetModel, taskType)
	}
	return service.ModelParams{
		Temperature: 0.7,
		MaxTokens:   2048,
	}, nil
}

func (m *MockModelAdapter) EstimateScore(ctx context.Context, promptID string, targetModel string) (float64, error) {
	if m.EstimateScoreFunc != nil {
		return m.EstimateScoreFunc(ctx, promptID, targetModel)
	}
	return 0.85, nil
}

func (m *MockModelAdapter) GetModelProfile(ctx context.Context, model string) (service.ModelProfile, error) {
	if m.GetModelProfileFunc != nil {
		return m.GetModelProfileFunc(ctx, model)
	}
	return service.ModelProfile{
		ContextWindow:     128000,
		InstructionStyle:  "xml_preference",
		FewShotCapacity:   10,
		TemperatureCurve:  "linear",
		SystemRoleSupport: true,
		JSONReliability:   0.9,
	}, nil
}
