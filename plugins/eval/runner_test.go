package eval

import (
	"context"
	"errors"
	"testing"

	"github.com/eval-prompt/plugins/llm"
	"github.com/eval-prompt/internal/service"
	"github.com/stretchr/testify/require"
)

// mockLLMInvoker is a mock implementation of LLMInvoker for testing.
type mockLLMInvoker struct {
	responseContent string
	responseErr     error
}

func (m *mockLLMInvoker) Invoke(ctx context.Context, prompt string, model string, temperature float64) (*llm.LLMResponse, error) {
	if m.responseErr != nil {
		return nil, m.responseErr
	}
	return &llm.LLMResponse{
		Content: m.responseContent,
		Model:   model,
	}, nil
}

func (m *mockLLMInvoker) InvokeWithSchema(ctx context.Context, prompt string, schema []byte) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func TestRunner_RunDeterministic(t *testing.T) {
	tests := []struct {
		name      string
		trace     []service.TraceEvent
		checks    []service.DeterministicCheck
		wantPass  bool
		wantScore float64
		wantFail  int
	}{
		{
			name: "all checks pass",
			trace: []service.TraceEvent{
				{Type: "command_executed", Data: map[string]any{"command": "ls -la"}},
				{Type: "file_created", Data: map[string]any{"path": "/sandbox/output.txt"}},
				{Type: "llm_output", Data: map[string]any{"content": `{"result": "success"}`}},
			},
			checks: []service.DeterministicCheck{
				{ID: "c1", Type: "command_executed", Expected: "ls"},
				{ID: "c2", Type: "file_exists", Path: "/sandbox/output.txt"},
				{ID: "c3", Type: "json_valid"},
			},
			wantPass:  true,
			wantScore: 1.0,
			wantFail:  0,
		},
		{
			name: "some checks fail",
			trace: []service.TraceEvent{
				{Type: "command_executed", Data: map[string]any{"command": "ls -la"}},
				{Type: "file_created", Data: map[string]any{"path": "/sandbox/output.txt"}},
			},
			checks: []service.DeterministicCheck{
				{ID: "c1", Type: "command_executed", Expected: "ls"},
				{ID: "c2", Type: "command_executed", Expected: "git"},
				{ID: "c3", Type: "file_exists", Path: "/sandbox/other.txt"},
			},
			wantPass:  false,
			wantScore: 1.0 / 3.0,
			wantFail:  2,
		},
		{
			name:      "no checks",
			trace:     []service.TraceEvent{},
			checks:    []service.DeterministicCheck{},
			wantPass:  true,
			wantScore: 1.0,
			wantFail:  0,
		},
		{
			name: "unknown check type fails",
			trace: []service.TraceEvent{
				{Type: "command_executed", Data: map[string]any{"command": "ls"}},
			},
			checks: []service.DeterministicCheck{
				{ID: "unknown", Type: "unknown_type"},
			},
			wantPass:  false,
			wantScore: 0.0,
			wantFail:  1,
		},
		{
			name: "command not found fails check",
			trace: []service.TraceEvent{
				{Type: "command_executed", Data: map[string]any{"command": "pwd"}},
			},
			checks: []service.DeterministicCheck{
				{ID: "c1", Type: "command_executed", Expected: "ls"},
			},
			wantPass:  false,
			wantScore: 0.0,
			wantFail:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner()
			result, err := runner.RunDeterministic(context.Background(), tt.trace, tt.checks)
			require.NoError(t, err)

			require.Equal(t, tt.wantPass, result.Passed, "Passed mismatch")
			require.InDelta(t, tt.wantScore, result.Score, 0.001, "Score mismatch")
			require.Len(t, result.Failed, tt.wantFail, "Failed count mismatch")
		})
	}
}

func TestRunner_RunDeterministic_EmptyChecks(t *testing.T) {
	runner := NewRunner()
	result, err := runner.RunDeterministic(context.Background(), []service.TraceEvent{}, []service.DeterministicCheck{})
	require.NoError(t, err)
	require.True(t, result.Passed)
	require.Equal(t, 1.0, result.Score)
}

func TestRunner_RunRubric(t *testing.T) {
	tests := []struct {
		name         string
		output       string
		rubric       service.Rubric
		mockResponse string
		mockErr      error
		wantPass     bool
		wantScore    int
		wantErr      bool
	}{
		{
			name:   "all checks pass",
			output: "The response correctly handles the request",
			rubric: service.Rubric{
				MaxScore: 100,
				Checks: []service.RubricCheck{
					{ID: "r1", Description: "Contains correct answer", Weight: 50},
					{ID: "r2", Description: "Is well formatted", Weight: 50},
				},
			},
			mockResponse: `[{"check_id": "r1", "passed": true, "score": 50, "details": "ok"}, {"check_id": "r2", "passed": true, "score": 50, "details": "ok"}]`,
			wantPass:     true,
			wantScore:    100,
		},
		{
			name:   "some checks fail - score below 80 percent",
			output: "Partial response",
			rubric: service.Rubric{
				MaxScore: 100,
				Checks: []service.RubricCheck{
					{ID: "r1", Description: "Complete answer", Weight: 80},
					{ID: "r2", Description: "Well formatted", Weight: 20},
				},
			},
			mockResponse: `[{"check_id": "r1", "passed": false, "score": 0, "details": "incomplete"}, {"check_id": "r2", "passed": true, "score": 20, "details": "ok"}]`,
			wantPass:     false,
			wantScore:    20,
		},
		{
			name:   "empty rubric checks",
			output: "Any output",
			rubric: service.Rubric{
				MaxScore: 100,
				Checks:   []service.RubricCheck{},
			},
			wantPass:  true,
			wantScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner()
			mockInvoker := &mockLLMInvoker{
				responseContent: tt.mockResponse,
				responseErr:     tt.mockErr,
			}

			result, err := runner.RunRubric(context.Background(), tt.output, tt.rubric, mockInvoker, "test-model")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, tt.wantPass, result.Passed, "Passed mismatch")
			require.Equal(t, tt.wantScore, result.Score, "Score mismatch")
		})
	}
}

func TestRunner_RunRubric_NoInvoker(t *testing.T) {
	runner := NewRunner()
	_, err := runner.RunRubric(context.Background(), "output", service.Rubric{}, nil, "test-model")
	require.Error(t, err)
	require.Contains(t, err.Error(), "LLMInvoker is required")
}

func TestRunner_RunRubric_InvokeError(t *testing.T) {
	runner := NewRunner()
	mockInvoker := &mockLLMInvoker{
		responseErr: errors.New("invoke error"),
	}

	_, err := runner.RunRubric(context.Background(), "output", service.Rubric{
		MaxScore: 100,
		Checks:   []service.RubricCheck{{ID: "c1", Weight: 10}},
	}, mockInvoker, "test-model")
	require.Error(t, err)
	require.Contains(t, err.Error(), "LLM invocation failed")
}

func TestRunner_RunRubric_InvalidJSONResponse(t *testing.T) {
	runner := NewRunner()
	mockInvoker := &mockLLMInvoker{
		responseContent: "not valid json",
	}

	rubric := service.Rubric{
		MaxScore: 100,
		Checks: []service.RubricCheck{
			{ID: "c1", Description: "test", Weight: 100},
		},
	}

	result, err := runner.RunRubric(context.Background(), "output", rubric, mockInvoker, "test-model")
	require.NoError(t, err)
	// Should use fallback parser and return results
	require.NotNil(t, result.Details)
}

func TestRunner_BuildGradingPrompt(t *testing.T) {
	runner := NewRunner()
	rubric := service.Rubric{
		MaxScore: 100,
		Checks: []service.RubricCheck{
			{ID: "c1", Description: "Check 1", Weight: 50},
			{ID: "c2", Description: "Check 2", Weight: 50},
		},
	}

	prompt := runner.buildGradingPrompt("test output", rubric)
	require.Contains(t, prompt, "test output")
	require.Contains(t, prompt, "Check 1")
	require.Contains(t, prompt, "Check 2")
}

func TestRunner_ParseGradingResponse(t *testing.T) {
	runner := NewRunner()
	rubric := service.Rubric{
		MaxScore: 100,
		Checks: []service.RubricCheck{
			{ID: "c1", Description: "Check 1", Weight: 50},
			{ID: "c2", Description: "Check 2", Weight: 50},
		},
	}

	// Test with actual JSON content
	content := `[{"check_id": "c1", "passed": true, "score": 50}]`
	results := runner.parseGradingResponse(content, rubric)
	require.Len(t, results, 2)
}

func TestRunnerTypeAliases(t *testing.T) {
	// Verify all type aliases work correctly
	_ = DeterministicResult{}
	_ = Rubric{}
	_ = RubricCheck{}
	_ = RubricCheckResult{}
	_ = RubricResult{}
	_ = LLMInvoker(nil)
}

func TestRunner_ImplementsInterface(t *testing.T) {
	runner := NewRunner()
	var _ service.EvalRunner = runner
}
