package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/service/mocks"
	"github.com/stretchr/testify/require"
)

func newTestTriggerHandler() (*TriggerHandler, *mock.MockTriggerService) {
	mockTrigger := &mock.MockTriggerService{
		MatchTriggerFunc: func(ctx context.Context, input string, top int) ([]*service.MatchedPrompt, error) {
			return []*service.MatchedPrompt{
				{AssetID: "common/test", Name: "Test", Description: "Test prompt", Content: "test content", Relevance: 0.95},
			}, nil
		},
		ValidateAntiPatternsFunc: func(ctx context.Context, prompt string) error {
			return nil // valid prompt
		},
		InjectVariablesFunc: func(ctx context.Context, prompt string, vars map[string]string) (string, error) {
			return "injected: " + prompt, nil
		},
	}
	logger := slog.Default()
	return NewTriggerHandler(mockTrigger, logger), mockTrigger
}

func TestTriggerHandler_MatchTrigger(t *testing.T) {
	handler, _ := newTestTriggerHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/trigger/match", handler.MatchTrigger)

	body := `{"input": "test prompt", "top": 5}`
	req := httptest.NewRequest("POST", "/api/v1/trigger/match", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MatchTriggerResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, 1, resp.Total)
	require.Len(t, resp.Matches, 1)
}

func TestTriggerHandler_MatchTrigger_DefaultTop(t *testing.T) {
	handler, _ := newTestTriggerHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/trigger/match", handler.MatchTrigger)

	body := `{"input": "test prompt"}`
	req := httptest.NewRequest("POST", "/api/v1/trigger/match", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestTriggerHandler_MatchTrigger_MissingInput(t *testing.T) {
	handler, _ := newTestTriggerHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/trigger/match", handler.MatchTrigger)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/trigger/match", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTriggerHandler_ValidateAntiPatterns(t *testing.T) {
	handler, _ := newTestTriggerHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/trigger/validate", handler.ValidateAntiPatterns)

	body := `{"prompt": "valid prompt"}`
	req := httptest.NewRequest("POST", "/api/v1/trigger/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp ValidateAntiPatternsResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.True(t, resp.Valid)
}

func TestTriggerHandler_ValidateAntiPatterns_Invalid(t *testing.T) {
	handler, mockTrigger := newTestTriggerHandler()
	mockTrigger.ValidateAntiPatternsFunc = func(ctx context.Context, prompt string) error {
		return &domain.DomainError{Code: domain.ErrorCode{Layer: domain.Layer4, Sequence: 1}, Message: "anti-pattern detected"}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/trigger/validate", handler.ValidateAntiPatterns)

	body := `{"prompt": "delete all data"}`
	req := httptest.NewRequest("POST", "/api/v1/trigger/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp ValidateAntiPatternsResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.False(t, resp.Valid)
}

func TestTriggerHandler_InjectVariables(t *testing.T) {
	handler, _ := newTestTriggerHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/trigger/inject", handler.InjectVariables)

	body := `{"prompt": "Hello {{name}}", "variables": {"name": "World"}}`
	req := httptest.NewRequest("POST", "/api/v1/trigger/inject", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp InjectVariablesResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "injected: Hello {{name}}", resp.Result)
}

func TestTriggerHandler_InjectVariables_MissingPrompt(t *testing.T) {
	handler, _ := newTestTriggerHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/trigger/inject", handler.InjectVariables)

	body := `{"variables": {"name": "World"}}`
	req := httptest.NewRequest("POST", "/api/v1/trigger/inject", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTriggerHandler_InjectVariables_MissingVariables(t *testing.T) {
	handler, _ := newTestTriggerHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/trigger/inject", handler.InjectVariables)

	body := `{"prompt": "Hello"}`
	req := httptest.NewRequest("POST", "/api/v1/trigger/inject", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTriggerHandler_GetAntiPatterns(t *testing.T) {
	handler, _ := newTestTriggerHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/trigger/anti-patterns", handler.GetAntiPatterns)

	req := httptest.NewRequest("GET", "/api/v1/trigger/anti-patterns", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotNil(t, resp["anti_patterns"])
}

// DomainError for testing
type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}