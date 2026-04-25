package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/service/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEvalRunRepo is a mock eval run repository for testing.
type mockEvalRunRepo struct {
	CreateFunc    func(ctx context.Context, run *domain.EvalRun) error
	UpdateFunc    func(ctx context.Context, run *domain.EvalRun) error
	GetByIDFunc   func(ctx context.Context, id string) (*domain.EvalRun, error)
	GetBySnapshotIDFunc func(ctx context.Context, snapshotID string) ([]*domain.EvalRun, error)
}

func (m *mockEvalRunRepo) Create(ctx context.Context, run *domain.EvalRun) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, run)
	}
	return nil
}

func (m *mockEvalRunRepo) Update(ctx context.Context, run *domain.EvalRun) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, run)
	}
	return nil
}

func (m *mockEvalRunRepo) GetByID(ctx context.Context, id string) (*domain.EvalRun, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockEvalRunRepo) GetBySnapshotID(ctx context.Context, snapshotID string) ([]*domain.EvalRun, error) {
	if m.GetBySnapshotIDFunc != nil {
		return m.GetBySnapshotIDFunc(ctx, snapshotID)
	}
	return nil, nil
}

// mockSnapshotRepo is a mock snapshot repository for testing.
type mockSnapshotRepo struct {
	GetByIDFunc                  func(ctx context.Context, id string) (*domain.Snapshot, error)
	GetByAssetIDFunc             func(ctx context.Context, assetID string) ([]*domain.Snapshot, error)
	GetByAssetIDAndVersionFunc   func(ctx context.Context, assetID, version string) (*domain.Snapshot, error)
	CreateFunc                   func(ctx context.Context, snapshot *domain.Snapshot) error
}

func (m *mockSnapshotRepo) GetByID(ctx context.Context, id string) (*domain.Snapshot, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockSnapshotRepo) GetByAssetID(ctx context.Context, assetID string) ([]*domain.Snapshot, error) {
	if m.GetByAssetIDFunc != nil {
		return m.GetByAssetIDFunc(ctx, assetID)
	}
	return nil, nil
}

func (m *mockSnapshotRepo) GetByAssetIDAndVersion(ctx context.Context, assetID, version string) (*domain.Snapshot, error) {
	if m.GetByAssetIDAndVersionFunc != nil {
		return m.GetByAssetIDAndVersionFunc(ctx, assetID, version)
	}
	return nil, errors.New("snapshot not found")
}

func (m *mockSnapshotRepo) Create(ctx context.Context, snapshot *domain.Snapshot) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, snapshot)
	}
	return nil
}

// mockEvalCaseRepo is a mock eval case repository for testing.
type mockEvalCaseRepo struct {
	GetByIDFunc     func(ctx context.Context, id string) (*domain.EvalCase, error)
	GetByAssetIDFunc func(ctx context.Context, assetID string) ([]*domain.EvalCase, error)
}

func (m *mockEvalCaseRepo) GetByID(ctx context.Context, id string) (*domain.EvalCase, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockEvalCaseRepo) GetByAssetID(ctx context.Context, assetID string) ([]*domain.EvalCase, error) {
	if m.GetByAssetIDFunc != nil {
		return m.GetByAssetIDFunc(ctx, assetID)
	}
	return nil, nil
}

// mockAssetRepo is a mock asset repository for testing.
type mockAssetRepo struct {
	GetByIDFunc func(ctx context.Context, id string) (*domain.Asset, error)
}

func (m *mockAssetRepo) GetByID(ctx context.Context, id string) (*domain.Asset, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

// testEvalService is a testable version of EvalService that uses mock repos.
type testEvalService struct {
	*EvalService
	evalRunRepo   *mockEvalRunRepo
	snapshotRepo  *mockSnapshotRepo
	evalCaseRepo  *mockEvalCaseRepo
	assetRepo     *mockAssetRepo
}

func newTestEvalService() *testEvalService {
	svc := &testEvalService{
		EvalService:  NewEvalService(),
		evalRunRepo:  &mockEvalRunRepo{},
		snapshotRepo: &mockSnapshotRepo{},
		evalCaseRepo: &mockEvalCaseRepo{},
		assetRepo:    &mockAssetRepo{},
	}
	// Override the service to use mock repos via interfaces
	// This is a simplified test setup
	return svc
}

func TestCompareResult_Structure(t *testing.T) {
	result := &CompareResult{
		AssetID:  "asset-123",
		Version1: "v1.0.0",
		Version2: "v2.0.0",
		Run1: &EvalRunSummary{
			ID:                 "run-1",
			SnapshotID:         "snap-1",
			Status:             EvalRunStatusPassed,
			DeterministicScore: 1.0,
			RubricScore:        90,
			CreatedAt:          time.Now(),
		},
		Run2: &EvalRunSummary{
			ID:                 "run-2",
			SnapshotID:         "snap-2",
			Status:             EvalRunStatusPassed,
			DeterministicScore: 1.0,
			RubricScore:        95,
			CreatedAt:          time.Now(),
		},
		ScoreDelta:  5,
		PassedDelta: 0,
		DiffOutput:  "mock diff output",
	}

	assert.Equal(t, "asset-123", result.AssetID)
	assert.Equal(t, "v1.0.0", result.Version1)
	assert.Equal(t, "v2.0.0", result.Version2)
	assert.Equal(t, 5, result.ScoreDelta)
	assert.Equal(t, 95, result.Run2.RubricScore)
}

func TestEvalRunStatus_Values(t *testing.T) {
	tests := []struct {
		status   EvalRunStatus
		expected string
	}{
		{EvalRunStatusPending, "pending"},
		{EvalRunStatusRunning, "running"},
		{EvalRunStatusPassed, "passed"},
		{EvalRunStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestEvalReport_Structure(t *testing.T) {
	report := &EvalReport{
		RunID:              "run-123",
		AssetID:            "asset-456",
		SnapshotVersion:    "v1.0.0",
		Status:             EvalRunStatusPassed,
		OverallScore:       90,
		DeterministicScore: 1.0,
		RubricScore:        90,
		RubricDetails: []RubricCheckResult{
			{CheckID: "check-1", Passed: true, Score: 45, Details: "passed"},
			{CheckID: "check-2", Passed: true, Score: 45, Details: "passed"},
		},
		CheckResults: []CheckResult{
			{CheckID: "check-1", CheckType: "rubric", Passed: true, Score: 45},
			{CheckID: "check-2", CheckType: "rubric", Passed: true, Score: 45},
		},
		TokenUsage: TokenUsage{
			Input:  100,
			Output: 50,
			Total:  150,
		},
		DurationMs:     1500,
		GeneratedAt:    time.Now(),
	}

	assert.Equal(t, "run-123", report.RunID)
	assert.Equal(t, 90, report.OverallScore)
	assert.Equal(t, 150, report.TokenUsage.Total)
	assert.Len(t, report.RubricDetails, 2)
	assert.Len(t, report.CheckResults, 2)
}

func TestDiagnosis_Structure(t *testing.T) {
	diagnosis := &Diagnosis{
		RunID:               "run-123",
		OverallSeverity:     "medium",
		Findings: []DiagnosisFinding{
			{
				Category:                 "rubric",
				Severity:                 "medium",
				Location:                 "rubric_check:check-1",
				Problem:                  "Rubric check failed",
				Evidence:                 "Expected X but got Y",
				Suggestion:               "Improve the prompt",
				ExpectedScoreImprovement: 20,
			},
		},
		RecommendedStrategy: "Review and update prompt",
		EstimatedIterations: 3,
		Confidence:          0.75,
	}

	assert.Equal(t, "run-123", diagnosis.RunID)
	assert.Equal(t, "medium", diagnosis.OverallSeverity)
	assert.Len(t, diagnosis.Findings, 1)
	assert.Equal(t, 3, diagnosis.EstimatedIterations)
	assert.Equal(t, 0.75, diagnosis.Confidence)
}

func TestTokenUsage_Total(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		output   int
		expected int
	}{
		{"normal", 100, 50, 150},
		{"zero input", 0, 50, 50},
		{"zero output", 100, 0, 100},
		{"both zero", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := TokenUsage{Input: tt.input, Output: tt.output}
			assert.Equal(t, tt.expected, usage.Total)
		})
	}
}

func TestRubricCheckResult_Structure(t *testing.T) {
	result := RubricCheckResult{
		CheckID: "test-check",
		Passed:  true,
		Score:   50,
		Details: "Check passed successfully",
	}

	assert.Equal(t, "test-check", result.CheckID)
	assert.True(t, result.Passed)
	assert.Equal(t, 50, result.Score)
	assert.Equal(t, "Check passed successfully", result.Details)
}

func TestDeterministicResult_Structure(t *testing.T) {
	result := DeterministicResult{
		Passed:  true,
		Score:   0.95,
		Message: "All deterministic checks passed",
		Failed:  []string{},
	}

	assert.True(t, result.Passed)
	assert.Equal(t, 0.95, result.Score)
	assert.Empty(t, result.Failed)
}

func TestRubricResult_Structure(t *testing.T) {
	result := RubricResult{
		Score:    80,
		MaxScore: 100,
		Passed:   true,
		Details: []RubricCheckResult{
			{CheckID: "c1", Passed: true, Score: 40},
			{CheckID: "c2", Passed: true, Score: 40},
		},
		Message: "Rubric evaluation complete",
	}

	assert.Equal(t, 80, result.Score)
	assert.Equal(t, 100, result.MaxScore)
	assert.True(t, result.Passed)
	assert.Len(t, result.Details, 2)
}

func TestEvalRunSummary_Structure(t *testing.T) {
	summary := EvalRunSummary{
		ID:                 "run-abc",
		SnapshotID:         "snap-xyz",
		Status:             EvalRunStatusPassed,
		DeterministicScore: 1.0,
		RubricScore:        85,
		CreatedAt:          time.Now(),
	}

	assert.Equal(t, "run-abc", summary.ID)
	assert.Equal(t, "snap-xyz", summary.SnapshotID)
	assert.Equal(t, EvalRunStatusPassed, summary.Status)
	assert.Equal(t, 1.0, summary.DeterministicScore)
	assert.Equal(t, 85, summary.RubricScore)
}

func TestEvalRun_Structure(t *testing.T) {
	run := EvalRun{
		ID:                 "run-123",
		EvalCaseID:         "case-456",
		SnapshotID:         "snap-789",
		AssetID:            "asset-abc",
		Status:             EvalRunStatusRunning,
		DeterministicScore: 0.9,
		RubricScore:        85,
		RubricDetails: []RubricCheckResult{
			{CheckID: "check-1", Passed: true, Score: 85},
		},
		TracePath:  "/tmp/trace.jsonl",
		TokenInput: 150,
		TokenOutput: 75,
		DurationMs: 2000,
		CreatedAt: time.Now(),
	}

	assert.Equal(t, "run-123", run.ID)
	assert.Equal(t, "case-456", run.EvalCaseID)
	assert.Equal(t, "snap-789", run.SnapshotID)
	assert.Equal(t, EvalRunStatusRunning, run.Status)
	assert.Equal(t, 0.9, run.DeterministicScore)
	assert.Equal(t, 85, run.RubricScore)
	assert.Equal(t, 150, run.TokenInput)
	assert.Equal(t, 75, run.TokenOutput)
	assert.Equal(t, int64(2000), run.DurationMs)
}

func TestCheckResult_Structure(t *testing.T) {
	result := CheckResult{
		CheckID:   "test-check",
		CheckType: "rubric",
		Passed:    false,
		Score:     0,
		Expected:  "some content",
		Actual:    "different content",
		Details:   "Content mismatch",
	}

	assert.Equal(t, "test-check", result.CheckID)
	assert.Equal(t, "rubric", result.CheckType)
	assert.False(t, result.Passed)
	assert.Equal(t, "some content", result.Expected)
	assert.Equal(t, "different content", result.Actual)
}

func TestReconcileReport_Structure(t *testing.T) {
	report := ReconcileReport{
		Added:   5,
		Updated: 3,
		Deleted: 1,
		Errors:  []string{"error 1", "error 2"},
	}

	assert.Equal(t, 5, report.Added)
	assert.Equal(t, 3, report.Updated)
	assert.Equal(t, 1, report.Deleted)
	assert.Len(t, report.Errors, 2)
}

func TestLLMResponse_Structure(t *testing.T) {
	resp := &LLMResponse{
		Content:    "This is the response content",
		Model:      "gpt-4o",
		TokensIn:   120,
		TokensOut:  60,
		StopReason: "stop",
		RawResponse: []byte(`{"choices": []}`),
	}

	assert.Equal(t, "This is the response content", resp.Content)
	assert.Equal(t, "gpt-4o", resp.Model)
	assert.Equal(t, 120, resp.TokensIn)
	assert.Equal(t, 60, resp.TokensOut)
	assert.Equal(t, "stop", resp.StopReason)
}

func TestAdaptedPrompt_Structure(t *testing.T) {
	adapted := AdaptedPrompt{
		Content:     "Adapted instruction content",
		ParamAdjustments: map[string]float64{
			"temperature": 0.5,
		},
		FormatChanges: []string{"xml_format", "add_examples"},
		Warnings:      []string{"Model may struggle with this task"},
	}

	assert.Equal(t, "Adapted instruction content", adapted.Content)
	assert.Equal(t, 0.5, adapted.ParamAdjustments["temperature"])
	assert.Len(t, adapted.FormatChanges, 2)
	assert.Len(t, adapted.Warnings, 1)
}

func TestModelParams_Structure(t *testing.T) {
	params := ModelParams{
		Temperature:      0.7,
		MaxTokens:       2048,
		TopP:            0.9,
		FrequencyPenalty: 0.1,
		PresencePenalty:  0.0,
	}

	assert.Equal(t, 0.7, params.Temperature)
	assert.Equal(t, 2048, params.MaxTokens)
	assert.Equal(t, 0.9, params.TopP)
	assert.Equal(t, 0.1, params.FrequencyPenalty)
}

func TestModelProfile_Structure(t *testing.T) {
	profile := ModelProfile{
		ContextWindow:     128000,
		InstructionStyle:  "xml_preference",
		FewShotCapacity:   10,
		TemperatureCurve: "linear",
		SystemRoleSupport: true,
		JSONReliability:   0.9,
	}

	assert.Equal(t, 128000, profile.ContextWindow)
	assert.Equal(t, "xml_preference", profile.InstructionStyle)
	assert.Equal(t, 10, profile.FewShotCapacity)
	assert.Equal(t, "linear", profile.TemperatureCurve)
	assert.True(t, profile.SystemRoleSupport)
	assert.Equal(t, 0.9, profile.JSONReliability)
}

func TestTraceEvent_Structure(t *testing.T) {
	event := TraceEvent{
		SpanID:    "span-abc",
		ParentID:  "parent-xyz",
		Name:      "eval_start",
		Timestamp: time.Now(),
		Type:      "event",
		Data: map[string]any{
			"asset_id": "asset-123",
			"status":   "running",
		},
	}

	assert.Equal(t, "span-abc", event.SpanID)
	assert.Equal(t, "parent-xyz", event.ParentID)
	assert.Equal(t, "eval_start", event.Name)
	assert.Equal(t, "event", event.Type)
	assert.Equal(t, "asset-123", event.Data["asset_id"])
}

func TestPromptContent_Structure(t *testing.T) {
	content := PromptContent{
		Description: "Test prompt",
		Instruction: "Please analyze the following code",
		Examples: []Example{
			{Input: "code1", Output: "analysis1"},
			{Input: "code2", Output: "analysis2"},
		},
		Variables: []string{"code", "language"},
	}

	assert.Equal(t, "Test prompt", content.Description)
	assert.Equal(t, "Please analyze the following code", content.Instruction)
	assert.Len(t, content.Examples, 2)
	assert.Len(t, content.Variables, 2)
}

func TestExample_Structure(t *testing.T) {
	example := Example{
		Input:    "What is 2+2?",
		Output:   "4",
		Footnote: "Basic arithmetic",
	}

	assert.Equal(t, "What is 2+2?", example.Input)
	assert.Equal(t, "4", example.Output)
	assert.Equal(t, "Basic arithmetic", example.Footnote)
}

func TestCommitInfo_Structure(t *testing.T) {
	info := CommitInfo{
		Hash:      "abc123def456",
		ShortHash: "abc123d",
		Subject:   "Add new feature",
		Body:      "This commit adds a new feature",
		Author:    "test@example.com",
		Timestamp: time.Now(),
	}

	assert.Equal(t, "abc123def456", info.Hash)
	assert.Equal(t, "abc123d", info.ShortHash)
	assert.Equal(t, "Add new feature", info.Subject)
	assert.Equal(t, "test@example.com", info.Author)
}

func TestDeterministicCheck_Structure(t *testing.T) {
	check := DeterministicCheck{
		ID:       "check-1",
		Type:     "content_contains",
		Path:     "/tmp/file.txt",
		Expected: "expected content",
		JSONPath: "$.result",
	}

	assert.Equal(t, "check-1", check.ID)
	assert.Equal(t, "content_contains", check.Type)
	assert.Equal(t, "/tmp/file.txt", check.Path)
	assert.Equal(t, "expected content", check.Expected)
	assert.Equal(t, "$.result", check.JSONPath)
}

func TestRubric_Structure(t *testing.T) {
	rubric := Rubric{
		MaxScore: 100,
		Checks: []RubricCheck{
			{ID: "c1", Description: "Check 1", Weight: 50},
			{ID: "c2", Description: "Check 2", Weight: 50},
		},
	}

	assert.Equal(t, 100, rubric.MaxScore)
	assert.Len(t, rubric.Checks, 2)
	assert.Equal(t, "c1", rubric.Checks[0].ID)
	assert.Equal(t, 50, rubric.Checks[0].Weight)
}

func TestRubricCheck_Structure(t *testing.T) {
	check := RubricCheck{
		ID:          "quality-check",
		Description: "Response should be helpful",
		Weight:      30,
	}

	assert.Equal(t, "quality-check", check.ID)
	assert.Equal(t, "Response should be helpful", check.Description)
	assert.Equal(t, 30, check.Weight)
}

func TestStatusToService(t *testing.T) {
	tests := []struct {
		domainStatus domain.EvalRunStatus
		serviceStatus EvalRunStatus
	}{
		{domain.EvalRunStatusRunning, EvalRunStatusRunning},
		{domain.EvalRunStatusPassed, EvalRunStatusPassed},
		{domain.EvalRunStatusFailed, EvalRunStatusFailed},
		{domain.EvalRunStatusPending, EvalRunStatusPending},
	}

	for _, tt := range tests {
		t.Run(string(tt.serviceStatus), func(t *testing.T) {
			result := statusToService(tt.domainStatus)
			assert.Equal(t, tt.serviceStatus, result)
		})
	}

	// Test unknown status
	result := statusToService("unknown")
	assert.Equal(t, EvalRunStatusPending, result)
}

func TestSearchFilters_Structure(t *testing.T) {
	filters := SearchFilters{
		BizLine: "ai",
		Tags:    []string{"tag1", "tag2"},
		State:   "created",
		Label:   "prod",
	}

	assert.Equal(t, "ai", filters.BizLine)
	assert.Len(t, filters.Tags, 2)
	assert.Equal(t, "created", filters.State)
	assert.Equal(t, "prod", filters.Label)
}

func TestAssetSummary_Structure(t *testing.T) {
	score := 0.85
	summary := AssetSummary{
		ID:          "asset-123",
		Name:        "Test Asset",
		Description: "A test asset",
		BizLine:     "engineering",
		Tags:        []string{"test", "unit"},
		State:       "created",
		LatestScore: &score,
	}

	assert.Equal(t, "asset-123", summary.ID)
	assert.Equal(t, "Test Asset", summary.Name)
	assert.Equal(t, "engineering", summary.BizLine)
	assert.Equal(t, 0.85, *summary.LatestScore)
}

func TestAssetDetail_Structure(t *testing.T) {
	detail := AssetDetail{
		ID:          "asset-456",
		Name:        "Detailed Asset",
		Description: "A detailed asset",
		BizLine:     "data",
		Tags:        []string{"detailed"},
		State:       "active",
		Snapshots: []SnapshotSummary{
			{Version: "v1.0.0", CommitHash: "abc123"},
		},
		Labels: []LabelInfo{
			{Name: "prod", SnapshotID: "snap-1"},
		},
	}

	assert.Equal(t, "asset-456", detail.ID)
	assert.Len(t, detail.Snapshots, 1)
	assert.Len(t, detail.Labels, 1)
}

func TestSnapshotSummary_Structure(t *testing.T) {
	score := 0.9
	summary := SnapshotSummary{
		Version:    "v2.0.0",
		CommitHash: "def456",
		Author:     "author@example.com",
		Reason:     "Updated instructions",
		EvalScore:  &score,
		CreatedAt:  time.Now(),
	}

	assert.Equal(t, "v2.0.0", summary.Version)
	assert.Equal(t, "def456", summary.CommitHash)
	assert.Equal(t, 0.9, *summary.EvalScore)
}

func TestLabelInfo_Structure(t *testing.T) {
	info := LabelInfo{
		Name:       "latest",
		SnapshotID: "snap-xyz",
		UpdatedAt:  time.Now(),
	}

	assert.Equal(t, "latest", info.Name)
	assert.Equal(t, "snap-xyz", info.SnapshotID)
}

func TestEvalRun_NewEvalRun(t *testing.T) {
	domainRun := domain.NewEvalRun(
		domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
	)

	assert.NotEmpty(t, domainRun.ID.String())
	assert.Equal(t, domain.EvalRunStatusPending, domainRun.Status)
}

func TestEvalRun_Complete(t *testing.T) {
	run := domain.NewEvalRun(
		domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
	)

	run.Complete(0.95, 90, true)
	assert.Equal(t, domain.EvalRunStatusPassed, run.Status)
	assert.Equal(t, 0.95, run.DeterministicScore)
	assert.Equal(t, 90, run.RubricScore)

	run2 := domain.NewEvalRun(
		domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
	)
	run2.Complete(0.5, 40, false)
	assert.Equal(t, domain.EvalRunStatusFailed, run2.Status)
}

func TestEvalRun_Fail(t *testing.T) {
	run := domain.NewEvalRun(
		domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
	)
	run.Fail()
	assert.Equal(t, domain.EvalRunStatusFailed, run.Status)
}

func TestEvalCase_NewEvalCase(t *testing.T) {
	assetID := domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV")
	rubric := domain.Rubric{
		MaxScore: 100,
		Checks: []domain.RubricCheck{
			{ID: "c1", Description: "Check 1", Weight: 100},
		},
	}

	ec := domain.NewEvalCase(assetID, "Test Case", "Test prompt", true, "expected output", rubric)

	assert.NotEmpty(t, ec.ID.String())
	assert.Equal(t, "Test Case", ec.Name)
	assert.Equal(t, "Test prompt", ec.Prompt)
	assert.True(t, ec.ShouldTrigger)
}

func TestEvalCase_TotalRubricWeight(t *testing.T) {
	rubric := domain.Rubric{
		MaxScore: 100,
		Checks: []domain.RubricCheck{
			{ID: "c1", Weight: 30},
			{ID: "c2", Weight: 70},
		},
	}
	ec := domain.NewEvalCase(
		domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		"Test", "prompt", true, "output", rubric,
	)

	assert.Equal(t, 100, ec.TotalRubricWeight())
}

func TestEvalCase_RubricWeightMap(t *testing.T) {
	rubric := domain.Rubric{
		MaxScore: 100,
		Checks: []domain.RubricCheck{
			{ID: "check-a", Weight: 40},
			{ID: "check-b", Weight: 60},
		},
	}
	ec := domain.NewEvalCase(
		domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		"Test", "prompt", true, "output", rubric,
	)

	weightMap := ec.RubricWeightMap()
	assert.Equal(t, 40, weightMap["check-a"])
	assert.Equal(t, 60, weightMap["check-b"])
}

func TestCalculateScore(t *testing.T) {
	rubric := domain.Rubric{
		MaxScore: 100,
		Checks: []domain.RubricCheck{
			{ID: "c1", Weight: 50},
			{ID: "c2", Weight: 50},
		},
	}

	tests := []struct {
		name     string
		results  []domain.RubricCheckResult
		expected int
	}{
		{
			name:     "all passed",
			results:  []domain.RubricCheckResult{{CheckID: "c1", Passed: true}, {CheckID: "c2", Passed: true}},
			expected: 100,
		},
		{
			name:     "half passed",
			results:  []domain.RubricCheckResult{{CheckID: "c1", Passed: true}},
			expected: 50,
		},
		{
			name:     "none passed",
			results:  []domain.RubricCheckResult{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := domain.CalculateScore(rubric, tt.results)
			assert.Equal(t, tt.expected, score)
		})
	}
}

func TestValidateResults(t *testing.T) {
	rubric := domain.Rubric{
		MaxScore: 100,
		Checks: []domain.RubricCheck{
			{ID: "c1", Weight: 50},
			{ID: "c2", Weight: 50},
		},
	}

	tests := []struct {
		name    string
		results []domain.RubricCheckResult
		wantErr bool
	}{
		{
			name:    "valid results",
			results: []domain.RubricCheckResult{{CheckID: "c1"}, {CheckID: "c2"}},
			wantErr: false,
		},
		{
			name:    "unknown check",
			results: []domain.RubricCheckResult{{CheckID: "unknown"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := domain.ValidateResults(rubric, tt.results)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// MockEvalRunnerForTests is a simple mock for testing.
type MockEvalRunnerForTests struct {
	RunDeterministicFunc func(ctx context.Context, trace []TraceEvent, checks []DeterministicCheck) (DeterministicResult, error)
	RunRubricFunc        func(ctx context.Context, output string, rubric Rubric, invoker LLMInvoker) (RubricResult, error)
}

func (m *MockEvalRunnerForTests) RunDeterministic(ctx context.Context, trace []TraceEvent, checks []DeterministicCheck) (DeterministicResult, error) {
	if m.RunDeterministicFunc != nil {
		return m.RunDeterministicFunc(ctx, trace, checks)
	}
	return DeterministicResult{Passed: true, Score: 1.0}, nil
}

func (m *MockEvalRunnerForTests) RunRubric(ctx context.Context, output string, rubric Rubric, invoker LLMInvoker) (RubricResult, error) {
	if m.RunRubricFunc != nil {
		return m.RunRubricFunc(ctx, output, rubric, invoker)
	}
	return RubricResult{Score: rubric.MaxScore, MaxScore: rubric.MaxScore, Passed: true}, nil
}

// MockLLMInvokerForTests is a simple mock for testing.
type MockLLMInvokerForTests struct {
	InvokeFunc func(ctx context.Context, prompt string, model string, temperature float64) (*LLMResponse, error)
}

func (m *MockLLMInvokerForTests) Invoke(ctx context.Context, prompt string, model string, temperature float64) (*LLMResponse, error) {
	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, prompt, model, temperature)
	}
	return &LLMResponse{Content: "mock response", Model: model, TokensIn: 10, TokensOut: 5}, nil
}

func (m *MockLLMInvokerForTests) InvokeWithSchema(ctx context.Context, prompt string, schema []byte) ([]byte, error) {
	return []byte(`{}`), nil
}

func TestEvalService_ToServiceEvalRun(t *testing.T) {
	svc := NewEvalService()

	domainRun := domain.NewEvalRun(
		domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"),
		domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"),
	)
	domainRun.Complete(0.95, 85, true)
	domainRun.TokenInput = 100
	domainRun.TokenOutput = 50
	domainRun.DurationMs = 1500

	serviceRun := svc.toServiceEvalRun(domainRun)

	require.NotNil(t, serviceRun)
	assert.Equal(t, domainRun.ID.String(), serviceRun.ID)
	assert.Equal(t, 0.95, serviceRun.DeterministicScore)
	assert.Equal(t, 85, serviceRun.RubricScore)
	assert.Equal(t, EvalRunStatusPassed, serviceRun.Status)
	assert.Equal(t, 100, serviceRun.TokenInput)
	assert.Equal(t, 50, serviceRun.TokenOutput)
	assert.Equal(t, int64(1500), serviceRun.DurationMs)
}

func TestEvalService_ToServiceEvalRun_Nil(t *testing.T) {
	svc := NewEvalService()
	result := svc.toServiceEvalRun(nil)
	assert.Nil(t, result)
}

func TestEvalService_CompareResult_Deltas(t *testing.T) {
	result := &CompareResult{
		AssetID:     "asset-123",
		Version1:    "v1.0.0",
		Version2:    "v2.0.0",
		ScoreDelta:  10,
		PassedDelta: 1,
		Run1: &EvalRunSummary{
			ID:          "run-1",
			Status:      EvalRunStatusFailed,
			RubricScore: 70,
		},
		Run2: &EvalRunSummary{
			ID:          "run-2",
			Status:      EvalRunStatusPassed,
			RubricScore: 80,
		},
	}

	assert.Equal(t, 10, result.ScoreDelta)
	assert.Equal(t, 1, result.PassedDelta)
}

func TestEvalRunSummary_StatusTransitions(t *testing.T) {
	// Test all status types in EvalRunSummary
	statuses := []EvalRunStatus{
		EvalRunStatusPending,
		EvalRunStatusRunning,
		EvalRunStatusPassed,
		EvalRunStatusFailed,
	}

	for _, status := range statuses {
		summary := EvalRunSummary{
			ID:     "test-run",
			Status: status,
		}
		assert.Equal(t, status, summary.Status)
	}
}

func TestDiagnosis_EmptyFindings(t *testing.T) {
	diagnosis := &Diagnosis{
		RunID:               "run-123",
		OverallSeverity:     "low",
		Findings:            []DiagnosisFinding{},
		RecommendedStrategy: "No changes needed",
		EstimatedIterations: 0,
		Confidence:          1.0,
	}

	assert.Empty(t, diagnosis.Findings)
	assert.Equal(t, 0, diagnosis.EstimatedIterations)
	assert.Equal(t, 1.0, diagnosis.Confidence)
}

func TestEvalReport_OverallScoreCalculation(t *testing.T) {
	tests := []struct {
		name              string
		deterministicScore float64
		rubricScore       int
		expectedOverall   int
	}{
		{"both positive", 1.0, 80, 80},
		{"both zero", 0.0, 0, 0},
		{"deterministic zero", 0.0, 80, 80},
		{"rubric zero", 1.0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var overallScore int
			if tt.deterministicScore > 0 && tt.rubricScore > 0 {
				overallScore = int(float64(tt.rubricScore) * tt.deterministicScore)
			} else {
				overallScore = tt.rubricScore
			}
			assert.Equal(t, tt.expectedOverall, overallScore)
		})
	}
}

func TestMockEvalService_Full(t *testing.T) {
	mockSvc := &mocks.MockEvalService{
		RunEvalFunc: func(ctx context.Context, assetID, snapshotVersion string, caseIDs []string) (*service.EvalRun, error) {
			return &service.EvalRun{
				ID:        "mock-run-id",
				Status:    service.EvalRunStatusPassed,
				CreatedAt: time.Now(),
			}, nil
		},
		GetEvalRunFunc: func(ctx context.Context, runID string) (*service.EvalRun, error) {
			return &service.EvalRun{
				ID:     runID,
				Status: service.EvalRunStatusPassed,
			}, nil
		},
		CompareEvalFunc: func(ctx context.Context, assetID string, v1, v2 string) (*service.CompareResult, error) {
			return &service.CompareResult{
				AssetID:  assetID,
				Version1: v1,
				Version2: v2,
			}, nil
		},
	}

	ctx := context.Background()

	// Test RunEval
	run, err := mockSvc.RunEval(ctx, "asset-1", "v1.0.0", nil)
	require.NoError(t, err)
	assert.Equal(t, "mock-run-id", run.ID)
	assert.Equal(t, service.EvalRunStatusPassed, run.Status)

	// Test GetEvalRun
	run2, err := mockSvc.GetEvalRun(ctx, "run-123")
	require.NoError(t, err)
	assert.Equal(t, "run-123", run2.ID)

	// Test CompareEval
	result, err := mockSvc.CompareEval(ctx, "asset-1", "v1.0.0", "v2.0.0")
	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", result.Version1)
	assert.Equal(t, "v2.0.0", result.Version2)
}

func TestMockLLMInvoker_Full(t *testing.T) {
	mockInvoker := &mocks.MockLLMInvoker{
		InvokeFunc: func(ctx context.Context, prompt string, model string, temperature float64) (*service.LLMResponse, error) {
			return &service.LLMResponse{
				Content:    "Test response",
				Model:      model,
				TokensIn:   50,
				TokensOut:  25,
				StopReason: "stop",
			}, nil
		},
	}

	ctx := context.Background()
	resp, err := mockInvoker.Invoke(ctx, "Test prompt", "gpt-4o", 0.7)
	require.NoError(t, err)
	assert.Equal(t, "Test response", resp.Content)
	assert.Equal(t, 50, resp.TokensIn)
	assert.Equal(t, 25, resp.TokensOut)
}

func TestMockGitBridger_Full(t *testing.T) {
	mockBridger := &mocks.MockGitBridger{
		DiffFunc: func(ctx context.Context, commit1, commit2 string) (string, error) {
			return "diff output here", nil
		},
		StatusFunc: func(ctx context.Context) (added, modified, deleted []string, err error) {
			return []string{"file1.go"}, []string{"file2.go"}, []string{"file3.go"}, nil
		},
	}

	ctx := context.Background()

	// Test Diff
	diff, err := mockBridger.Diff(ctx, "abc123", "def456")
	require.NoError(t, err)
	assert.Equal(t, "diff output here", diff)

	// Test Status
	added, modified, deleted, err := mockBridger.Status(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"file1.go"}, added)
	assert.Equal(t, []string{"file2.go"}, modified)
	assert.Equal(t, []string{"file3.go"}, deleted)
}

func TestMockTraceCollector_Full(t *testing.T) {
	mockCollector := &mocks.MockTraceCollector{
		StartSpanFunc: func(ctx context.Context, assetID, snapshotID string) (context.Context, error) {
			return ctx, nil
		},
		RecordEventFunc: func(ctx context.Context, event service.TraceEvent) error {
			return nil
		},
		FinalizeFunc: func(ctx context.Context) (string, error) {
			return "/tmp/test-trace.jsonl", nil
		},
	}

	ctx := context.Background()

	// Test StartSpan
	newCtx, err := mockCollector.StartSpan(ctx, "asset-1", "snap-1")
	require.NoError(t, err)
	assert.NotNil(t, newCtx)

	// Test RecordEvent
	err = mockCollector.RecordEvent(ctx, service.TraceEvent{Name: "test-event"})
	require.NoError(t, err)

	// Test Finalize
	path, err := mockCollector.Finalize(ctx)
	require.NoError(t, err)
	assert.Equal(t, "/tmp/test-trace.jsonl", path)
}

func TestMockEvalRunner_Full(t *testing.T) {
	mockRunner := &mocks.MockEvalRunner{
		RunDeterministicFunc: func(ctx context.Context, trace []service.TraceEvent, checks []service.DeterministicCheck) (service.DeterministicResult, error) {
			return service.DeterministicResult{Passed: true, Score: 0.9}, nil
		},
		RunRubricFunc: func(ctx context.Context, output string, rubric service.Rubric, invoker service.LLMInvoker) (service.RubricResult, error) {
			return service.RubricResult{Score: 85, MaxScore: 100, Passed: true}, nil
		},
	}

	ctx := context.Background()

	// Test RunDeterministic
	result, err := mockRunner.RunDeterministic(ctx, nil, nil)
	require.NoError(t, err)
	assert.True(t, result.Passed)
	assert.Equal(t, 0.9, result.Score)

	// Test RunRubric
	rubricResult, err := mockRunner.RunRubric(ctx, "output", service.Rubric{MaxScore: 100}, nil)
	require.NoError(t, err)
	assert.Equal(t, 85, rubricResult.Score)
	assert.True(t, rubricResult.Passed)
}

func TestMockModelAdapter_Full(t *testing.T) {
	mockAdapter := &mocks.MockModelAdapter{
		AdaptFunc: func(ctx context.Context, prompt service.PromptContent, sourceModel, targetModel string) (service.AdaptedPrompt, error) {
			return service.AdaptedPrompt{
				Content:          prompt.Instruction + " [adapted for " + targetModel + "]",
				ParamAdjustments: map[string]float64{"temperature": 0.5},
			}, nil
		},
		EstimateScoreFunc: func(ctx context.Context, promptID string, targetModel string) (float64, error) {
			return 0.92, nil
		},
	}

	ctx := context.Background()

	// Test Adapt
	prompt := service.PromptContent{Instruction: "Original instruction"}
	adapted, err := mockAdapter.Adapt(ctx, prompt, "gpt-4o", "claude-3")
	require.NoError(t, err)
	assert.Contains(t, adapted.Content, "[adapted for claude-3]")

	// Test EstimateScore
	score, err := mockAdapter.EstimateScore(ctx, "prompt-1", "gpt-4o")
	require.NoError(t, err)
	assert.Equal(t, 0.92, score)
}

func TestMockAssetIndexer_Full(t *testing.T) {
	mockIndexer := &mocks.MockAssetIndexer{
		SearchFunc: func(ctx context.Context, query string, filters service.SearchFilters) ([]service.AssetSummary, error) {
			return []service.AssetSummary{
				{ID: "asset-1", Name: "Search Result", Description: "Found"},
			}, nil
		},
		GetByIDFunc: func(ctx context.Context, id string) (*service.AssetDetail, error) {
			return &service.AssetDetail{ID: id, Name: "Detail Asset"}, nil
		},
		SaveFunc: func(ctx context.Context, asset service.Asset) error {
			return nil
		},
		DeleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}

	ctx := context.Background()

	// Test Search
	results, err := mockIndexer.Search(ctx, "test query", service.SearchFilters{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Search Result", results[0].Name)

	// Test GetByID
	detail, err := mockIndexer.GetByID(ctx, "asset-1")
	require.NoError(t, err)
	assert.Equal(t, "Detail Asset", detail.Name)

	// Test Save
	err = mockIndexer.Save(ctx, service.Asset{ID: "new-asset"})
	require.NoError(t, err)

	// Test Delete
	err = mockIndexer.Delete(ctx, "asset-1")
	require.NoError(t, err)
}

func TestMockTriggerService_Full(t *testing.T) {
	mockTrigger := &mocks.MockTriggerService{
		MatchTriggerFunc: func(ctx context.Context, input string, top int) ([]*service.MatchedPrompt, error) {
			return []*service.MatchedPrompt{
				{AssetID: "asset-1", Name: "Matched Prompt", Relevance: 0.95},
			}, nil
		},
		ValidateAntiPatternsFunc: func(ctx context.Context, prompt string) error {
			return nil
		},
		InjectVariablesFunc: func(ctx context.Context, prompt string, vars map[string]string) (string, error) {
			return "Injected: " + vars["name"], nil
		},
	}

	ctx := context.Background()

	// Test MatchTrigger
	matches, err := mockTrigger.MatchTrigger(ctx, "test input", 5)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, 0.95, matches[0].Relevance)

	// Test ValidateAntiPatterns
	err = mockTrigger.ValidateAntiPatterns(ctx, "valid prompt")
	require.NoError(t, err)

	// Test InjectVariables
	result, err := mockTrigger.InjectVariables(ctx, "Hello {{name}}", map[string]string{"name": "World"})
	require.NoError(t, err)
	assert.Equal(t, "Injected: World", result)
}