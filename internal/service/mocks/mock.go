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
	SearchFunc  func(ctx context.Context, query string, filters service.SearchFilters) ([]service.AssetSummary, error)
	GetByIDFunc func(ctx context.Context, id string) (*service.AssetDetail, error)
	SaveFunc    func(ctx context.Context, asset service.Asset) error
	DeleteFunc  func(ctx context.Context, id string) error
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

// MockTriggerService is a mock implementation of service.TriggerServicer.
type MockTriggerService struct {
	MatchTriggerFunc       func(ctx context.Context, input string, top int) ([]*service.MatchedPrompt, error)
	ValidateAntiPatternsFunc func(ctx context.Context, prompt string) error
	InjectVariablesFunc    func(ctx context.Context, prompt string, vars map[string]string) (string, error)
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
	RunEvalFunc        func(ctx context.Context, assetID, snapshotVersion string, caseIDs []string) (*service.EvalRun, error)
	GetEvalRunFunc     func(ctx context.Context, runID string) (*service.EvalRun, error)
	ListEvalRunsFunc   func(ctx context.Context, assetID string) ([]*service.EvalRun, error)
	ListEvalCasesFunc  func(ctx context.Context, assetID string) ([]*domain.EvalCase, error)
	CompareEvalFunc    func(ctx context.Context, assetID string, v1, v2 string) (*service.CompareResult, error)
	GenerateReportFunc  func(ctx context.Context, runID string) (*service.EvalReport, error)
	DiagnoseEvalFunc   func(ctx context.Context, runID string) (*service.Diagnosis, error)
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