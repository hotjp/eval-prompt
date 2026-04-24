package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	svc "github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/service/mocks"
	"github.com/stretchr/testify/require"
)

func newTestEvalHandler() (*EvalHandler, *mock.MockEvalService) {
	mockEval := &mock.MockEvalService{
		RunEvalFunc: func(ctx context.Context, assetID, snapshotVersion string, caseIDs []string) (*svc.EvalRun, error) {
			return &svc.EvalRun{
				ID:        "run-123",
				AssetID:   assetID,
				Status:    svc.EvalRunStatusRunning,
				CreatedAt: time.Now(),
			}, nil
		},
		GetEvalRunFunc: func(ctx context.Context, runID string) (*svc.EvalRun, error) {
			return &svc.EvalRun{
				ID:        runID,
				Status:    svc.EvalRunStatusPassed,
				CreatedAt: time.Now(),
			}, nil
		},
		ListEvalRunsFunc: func(ctx context.Context, assetID string) ([]*svc.EvalRun, error) {
			return []*svc.EvalRun{
				{ID: "run-1", AssetID: assetID, Status: svc.EvalRunStatusPassed},
				{ID: "run-2", AssetID: assetID, Status: svc.EvalRunStatusFailed},
			}, nil
		},
		GenerateReportFunc: func(ctx context.Context, runID string) (*svc.EvalReport, error) {
			return &svc.EvalReport{
				RunID:        runID,
				Status:       svc.EvalRunStatusPassed,
				OverallScore:  85,
			}, nil
		},
		DiagnoseEvalFunc: func(ctx context.Context, runID string) (*svc.Diagnosis, error) {
			return &svc.Diagnosis{
				RunID:               runID,
				OverallSeverity:     "low",
				Findings:            []svc.DiagnosisFinding{},
				RecommendedStrategy: "none needed",
			}, nil
		},
		CompareEvalFunc: func(ctx context.Context, assetID string, v1, v2 string) (*svc.CompareResult, error) {
			return &svc.CompareResult{
				AssetID:    assetID,
				Version1:    v1,
				Version2:    v2,
				ScoreDelta:  10,
			}, nil
		},
	}
	logger := slog.Default()
	return NewEvalHandler(mockEval, logger), mockEval
}

func TestEvalHandler_RunEval(t *testing.T) {
	handler, _ := newTestEvalHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/evals/run", handler.RunEval)

	body := `{"asset_id": "common/test", "snapshot_version": "v1"}`
	req := httptest.NewRequest("POST", "/api/v1/evals/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusAccepted, rec.Code)

	var resp RunEvalResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "run-123", resp.RunID)
	require.Equal(t, "running", resp.Status)
}

func TestEvalHandler_RunEval_MissingAssetID(t *testing.T) {
	handler, _ := newTestEvalHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/evals/run", handler.RunEval)

	body := `{"snapshot_version": "v1"}`
	req := httptest.NewRequest("POST", "/api/v1/evals/run", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestEvalHandler_GetEvalRun(t *testing.T) {
	handler, _ := newTestEvalHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/evals/{id}", handler.GetEvalRun)

	req := httptest.NewRequest("GET", "/api/v1/evals/run-123", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestEvalHandler_GetEvalReport(t *testing.T) {
	handler, _ := newTestEvalHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/evals/{id}/report", handler.GetEvalReport)

	req := httptest.NewRequest("GET", "/api/v1/evals/run-123/report", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestEvalHandler_CompareEval(t *testing.T) {
	handler, _ := newTestEvalHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/evals/compare", handler.CompareEval)

	body := `{"asset_id": "common/test", "version1": "v1", "version2": "v2"}`
	req := httptest.NewRequest("POST", "/api/v1/evals/compare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestEvalHandler_CompareEval_MissingFields(t *testing.T) {
	handler, _ := newTestEvalHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/evals/compare", handler.CompareEval)

	body := `{"asset_id": "common/test"}`
	req := httptest.NewRequest("POST", "/api/v1/evals/compare", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestEvalHandler_DiagnoseEval(t *testing.T) {
	handler, _ := newTestEvalHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/evals/{id}/diagnose", handler.DiagnoseEval)

	req := httptest.NewRequest("GET", "/api/v1/evals/run-123/diagnose", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestEvalHandler_ListEvalRuns(t *testing.T) {
	handler, _ := newTestEvalHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/evals", handler.ListEvalRuns)

	req := httptest.NewRequest("GET", "/api/v1/evals?asset_id=common/test", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(2), resp["total"])
}

func TestEvalHandler_ListEvalRuns_MissingAssetID(t *testing.T) {
	handler, _ := newTestEvalHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/evals", handler.ListEvalRuns)

	req := httptest.NewRequest("GET", "/api/v1/evals", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}