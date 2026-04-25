package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eval-prompt/internal/gateway/handlers"
	svc "github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/plugins/search"
	"github.com/stretchr/testify/require"
)

// mockEvalServiceer is a mock implementation of EvalServiceer for E2E testing.
type mockEvalServiceer struct {
	RunEvalFunc         func(ctx context.Context, assetID, snapshotVersion string, caseIDs []string) (*svc.EvalRun, error)
	GetEvalRunFunc      func(ctx context.Context, runID string) (*svc.EvalRun, error)
	ListEvalRunsFunc    func(ctx context.Context, assetID string) ([]*svc.EvalRun, error)
	CompareEvalFunc     func(ctx context.Context, assetID string, v1, v2 string) (*svc.CompareResult, error)
	GenerateReportFunc  func(ctx context.Context, runID string) (*svc.EvalReport, error)
	DiagnoseEvalFunc    func(ctx context.Context, runID string) (*svc.Diagnosis, error)
	ListEvalCasesFunc   func(ctx context.Context, assetID string) ([]*svc.EvalCase, error)
}

func (m *mockEvalServiceer) RunEval(ctx context.Context, assetID, snapshotVersion string, caseIDs []string) (*svc.EvalRun, error) {
	if m.RunEvalFunc != nil {
		return m.RunEvalFunc(ctx, assetID, snapshotVersion, caseIDs)
	}
	return &svc.EvalRun{
		ID:        "run-e2e-001",
		AssetID:   assetID,
		Status:    svc.EvalRunStatusRunning,
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockEvalServiceer) GetEvalRun(ctx context.Context, runID string) (*svc.EvalRun, error) {
	if m.GetEvalRunFunc != nil {
		return m.GetEvalRunFunc(ctx, runID)
	}
	return &svc.EvalRun{
		ID:        runID,
		Status:    svc.EvalRunStatusPassed,
		Score:     85,
		CreatedAt: time.Now(),
	}, nil
}

func (m *mockEvalServiceer) ListEvalRuns(ctx context.Context, assetID string) ([]*svc.EvalRun, error) {
	if m.ListEvalRunsFunc != nil {
		return m.ListEvalRunsFunc(ctx, assetID)
	}
	return []*svc.EvalRun{
		{ID: "run-1", AssetID: assetID, Status: svc.EvalRunStatusPassed, Score: 85},
		{ID: "run-2", AssetID: assetID, Status: svc.EvalRunStatusFailed, Score: 60},
	}, nil
}

func (m *mockEvalServiceer) CompareEval(ctx context.Context, assetID string, v1, v2 string) (*svc.CompareResult, error) {
	if m.CompareEvalFunc != nil {
		return m.CompareEvalFunc(ctx, assetID, v1, v2)
	}
	return &svc.CompareResult{
		AssetID:    assetID,
		Version1:   v1,
		Version2:    v2,
		ScoreDelta: 15,
		Version1Score: 75,
		Version2Score: 90,
	}, nil
}

func (m *mockEvalServiceer) GenerateReport(ctx context.Context, runID string) (*svc.EvalReport, error) {
	if m.GenerateReportFunc != nil {
		return m.GenerateReportFunc(ctx, runID)
	}
	return &svc.EvalReport{
		RunID:         runID,
		Status:        svc.EvalRunStatusPassed,
		OverallScore:  85,
		Summary:       "Test passed",
		Recommendations: []string{"Keep up the good work"},
	}, nil
}

func (m *mockEvalServiceer) DiagnoseEval(ctx context.Context, runID string) (*svc.Diagnosis, error) {
	if m.DiagnoseEvalFunc != nil {
		return m.DiagnoseEvalFunc(ctx, runID)
	}
	return &svc.Diagnosis{
		RunID:               runID,
		OverallSeverity:     "low",
		Findings:            []svc.DiagnosisFinding{},
		RecommendedStrategy: "none needed",
	}, nil
}

func (m *mockEvalServiceer) ListEvalCases(ctx context.Context, assetID string) ([]*svc.EvalCase, error) {
	if m.ListEvalCasesFunc != nil {
		return m.ListEvalCasesFunc(ctx, assetID)
	}
	return []*svc.EvalCase{
		{ID: "case-1", AssetID: assetID, Name: "Test Case 1"},
	}, nil
}

// setupTestRouter creates a test router with real implementations for search and mock for eval.
func setupTestRouter(t *testing.T) http.Handler {
	logger := slog.Default()
	indexer := search.NewIndexer()
	mockEval := &mockEvalServiceer{}

	assetHandler := handlers.NewAssetHandler(indexer, logger)
	evalHandler := handlers.NewEvalHandler(mockEval, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/assets", assetHandler.CreateAsset)
	mux.HandleFunc("GET /api/v1/assets", assetHandler.ListAssets)
	mux.HandleFunc("GET /api/v1/assets/{id}", assetHandler.GetAsset)
	mux.HandleFunc("PUT /api/v1/assets/{id}", assetHandler.UpdateAsset)
	mux.HandleFunc("DELETE /api/v1/assets/{id}", assetHandler.DeleteAsset)
	mux.HandleFunc("POST /api/v1/evals/run", evalHandler.RunEval)
	mux.HandleFunc("GET /api/v1/evals/{id}", evalHandler.GetEvalRun)
	mux.HandleFunc("POST /api/v1/evals/compare", evalHandler.CompareEval)
	mux.HandleFunc("GET /api/v1/evals/{id}/report", evalHandler.GetEvalReport)

	return mux
}

// TestRESTAPI_E2E_FullWorkflow tests the complete REST API workflow.
func TestRESTAPI_E2E_FullWorkflow(t *testing.T) {
	mux := setupTestRouter(t)
	ctx := context.Background()
	indexer := search.NewIndexer()
	logger := slog.Default()
	assetHandler := handlers.NewAssetHandler(indexer, logger)
	evalHandler := handlers.NewEvalHandler(&mockEvalServiceer{}, logger)

	// Register routes with the mux
	testMux := http.NewServeMux()
	testMux.HandleFunc("POST /api/v1/assets", assetHandler.CreateAsset)
	testMux.HandleFunc("GET /api/v1/assets", assetHandler.ListAssets)
	testMux.HandleFunc("GET /api/v1/assets/{id}", assetHandler.GetAsset)
	testMux.HandleFunc("PUT /api/v1/assets/{id}", assetHandler.UpdateAsset)
	testMux.HandleFunc("DELETE /api/v1/assets/{id}", assetHandler.DeleteAsset)
	testMux.HandleFunc("POST /api/v1/evals/run", evalHandler.RunEval)
	testMux.HandleFunc("GET /api/v1/evals/{id}", evalHandler.GetEvalRun)
	testMux.HandleFunc("POST /api/v1/evals/compare", evalHandler.CompareEval)
	testMux.HandleFunc("GET /api/v1/evals/{id}/report", evalHandler.GetEvalReport)

	assetID := "e2e/test-asset"
	assetName := "E2E Test Asset"
	assetDescription := "Test asset for E2E workflow"
	bizLine := "ml"
	tags := []string{"e2e", "test"}

	// Step 1: POST /api/v1/assets - Create asset
	t.Run("Step1_CreateAsset", func(t *testing.T) {
		body := `{
			"id": "` + assetID + `",
			"name": "` + assetName + `",
			"description": "` + assetDescription + `",
			"biz_line": "` + bizLine + `",
			"tags": ["e2e", "test"]
		}`
		req := httptest.NewRequest("POST", "/api/v1/assets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code, "Create asset should return 201")

		var resp map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, assetID, resp["id"])
		require.Equal(t, "asset created successfully", resp["message"])
	})

	// Step 2: GET /api/v1/assets - List assets and verify created asset
	t.Run("Step2_ListAssets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets", nil)
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "List assets should return 200")

		var resp map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)

		assets, ok := resp["assets"].([]any)
		require.True(t, ok, "assets should be an array")
		require.NotEmpty(t, assets, "should have at least one asset")

		total, ok := resp["total"].(float64)
		require.True(t, ok)
		require.GreaterOrEqual(t, int(total), 1, "should have at least 1 asset")
	})

	// Step 3: GET /api/v1/assets/{id} - Get specific asset
	t.Run("Step3_GetAsset", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets/"+assetID, nil)
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "Get asset should return 200")

		var resp handlers.AssetResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, assetID, resp.ID)
		require.Equal(t, assetName, resp.Name)
		require.Equal(t, assetDescription, resp.Description)
		require.Equal(t, bizLine, resp.BizLine)
	})

	// Step 4: PUT /api/v1/assets/{id} - Update asset
	t.Run("Step4_UpdateAsset", func(t *testing.T) {
		newName := "E2E Updated Asset"
		newDescription := "Updated description"
		body := `{
			"name": "` + newName + `",
			"description": "` + newDescription + `"
		}`
		req := httptest.NewRequest("PUT", "/api/v1/assets/"+assetID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "Update asset should return 200")

		var resp map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, assetID, resp["id"])
		require.Equal(t, "asset updated successfully", resp["message"])

		// Verify update was applied
		getReq := httptest.NewRequest("GET", "/api/v1/assets/"+assetID, nil)
		getRec := httptest.NewRecorder()
		testMux.ServeHTTP(getRec, getReq)

		var updated handlers.AssetResponse
		err = json.Unmarshal(getRec.Body.Bytes(), &updated)
		require.NoError(t, err)
		require.Equal(t, newName, updated.Name)
		require.Equal(t, newDescription, updated.Description)
	})

	// Step 5: POST /api/v1/evals/run - Run eval
	t.Run("Step5_RunEval", func(t *testing.T) {
		body := `{
			"asset_id": "` + assetID + `",
			"snapshot_version": "v1",
			"eval_case_ids": ["case-1"]
		}`
		req := httptest.NewRequest("POST", "/api/v1/evals/run", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusAccepted, rec.Code, "Run eval should return 202")

		var resp handlers.RunEvalResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.NotEmpty(t, resp.RunID)
		require.Equal(t, "running", resp.Status)
	})

	// Step 6: GET /api/v1/evals/{id} - Get eval run
	t.Run("Step6_GetEvalRun", func(t *testing.T) {
		runID := "run-e2e-001"
		req := httptest.NewRequest("GET", "/api/v1/evals/"+runID, nil)
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "Get eval run should return 200")

		var resp svc.EvalRun
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, runID, resp.ID)
	})

	// Step 7: POST /api/v1/evals/compare - Compare evals
	t.Run("Step7_CompareEval", func(t *testing.T) {
		body := `{
			"asset_id": "` + assetID + `",
			"version1": "v1",
			"version2": "v2"
		}`
		req := httptest.NewRequest("POST", "/api/v1/evals/compare", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "Compare eval should return 200")

		var resp svc.CompareResult
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, assetID, resp.AssetID)
		require.Equal(t, "v1", resp.Version1)
		require.Equal(t, "v2", resp.Version2)
		require.NotZero(t, resp.ScoreDelta)
	})

	// Step 8: DELETE /api/v1/assets/{id} - Delete asset
	t.Run("Step8_DeleteAsset", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/assets/"+assetID, nil)
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "Delete asset should return 200")

		var resp map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, assetID, resp["id"])
		require.Equal(t, "asset deleted successfully", resp["message"])

		// Verify asset is deleted
		getReq := httptest.NewRequest("GET", "/api/v1/assets/"+assetID, nil)
		getRec := httptest.NewRecorder()
		testMux.ServeHTTP(getRec, getReq)
		require.Equal(t, http.StatusNotFound, getRec.Code, "Deleted asset should return 404")
	})

	// Verify we can run the full workflow again with a new asset
	t.Run("FullWorkflow_NewAsset", func(t *testing.T) {
		newAssetID := "e2e/new-asset"
		body := `{
			"id": "` + newAssetID + `",
			"name": "New Test Asset",
			"description": "Testing new asset workflow",
			"biz_line": "data"
		}`

		// Create
		req := httptest.NewRequest("POST", "/api/v1/assets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		testMux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code)

		// Read
		req = httptest.NewRequest("GET", "/api/v1/assets/"+newAssetID, nil)
		rec = httptest.NewRecorder()
		testMux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		// Delete
		req = httptest.NewRequest("DELETE", "/api/v1/assets/"+newAssetID, nil)
		rec = httptest.NewRecorder()
		testMux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	// Suppress unused variable warning
	_ = ctx
}

// TestRESTAPI_E2E_ErrorHandling tests error handling in the REST API workflow.
func TestRESTAPI_E2E_ErrorHandling(t *testing.T) {
	logger := slog.Default()
	indexer := search.NewIndexer()
	assetHandler := handlers.NewAssetHandler(indexer, logger)

	testMux := http.NewServeMux()
	testMux.HandleFunc("POST /api/v1/assets", assetHandler.CreateAsset)
	testMux.HandleFunc("GET /api/v1/assets/{id}", assetHandler.GetAsset)
	testMux.HandleFunc("PUT /api/v1/assets/{id}", assetHandler.UpdateAsset)
	testMux.HandleFunc("DELETE /api/v1/assets/{id}", assetHandler.DeleteAsset)

	t.Run("CreateAsset_MissingID", func(t *testing.T) {
		body := `{"name": "Test"}`
		req := httptest.NewRequest("POST", "/api/v1/assets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("CreateAsset_MissingName", func(t *testing.T) {
		body := `{"id": "test/asset"}`
		req := httptest.NewRequest("POST", "/api/v1/assets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("CreateAsset_InvalidJSON", func(t *testing.T) {
		body := `{invalid json}`
		req := httptest.NewRequest("POST", "/api/v1/assets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("GetAsset_NotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets/nonexistent/id", nil)
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("UpdateAsset_NotFound", func(t *testing.T) {
		body := `{"name": "Updated Name"}`
		req := httptest.NewRequest("PUT", "/api/v1/assets/nonexistent/id", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("DeleteAsset_NotFound", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/v1/assets/nonexistent/id", nil)
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		// Delete should not error for non-existent (based on current implementation)
		// If the implementation changes to return error, this would be StatusNotFound
		require.Equal(t, http.StatusOK, rec.Code)
	})
}

// TestRESTAPI_E2E_Filters tests filtering capabilities in list endpoint.
func TestRESTAPI_E2E_Filters(t *testing.T) {
	logger := slog.Default()
	indexer := search.NewIndexer()
	assetHandler := handlers.NewAssetHandler(indexer, logger)

	testMux := http.NewServeMux()
	testMux.HandleFunc("POST /api/v1/assets", assetHandler.CreateAsset)
	testMux.HandleFunc("GET /api/v1/assets", assetHandler.ListAssets)

	// Create test assets
	assets := []struct {
		id      string
		name    string
		bizLine string
		tags    []string
		state   string
	}{
		{"test/asset1", "Asset One", "ml", []string{"ai", "test"}, "created"},
		{"test/asset2", "Asset Two", "data", []string{"data", "test"}, "evaluated"},
		{"test/asset3", "Asset Three", "ml", []string{"ai", "prod"}, "evaluated"},
	}

	for _, a := range assets {
		body := `{"id":"` + a.id + `","name":"` + a.name + `","biz_line":"` + a.bizLine + `","tags":[` + strings.Join(func() []string {
			var tags []string
			for _, t := range a.tags {
				tags = append(tags, `"`+t+`"`)
			}
			return tags
		}()) + `]}`
		req := httptest.NewRequest("POST", "/api/v1/assets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		testMux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code)
	}

	t.Run("FilterByBizLine", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets?biz_line=ml", nil)
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]any
		json.Unmarshal(rec.Body.Bytes(), &resp)
		// Should return 2 ml assets (asset1 and asset3)
		require.Equal(t, float64(2), resp["total"], "should have 2 ml assets")
	})

	t.Run("FilterByState", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets?state=evaluated", nil)
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]any
		json.Unmarshal(rec.Body.Bytes(), &resp)
		// Should return 2 evaluated assets (asset2 and asset3)
		require.Equal(t, float64(2), resp["total"], "should have 2 evaluated assets")
	})
}

// TestRESTAPI_E2E_EvalReports tests eval report generation.
func TestRESTAPI_E2E_EvalReports(t *testing.T) {
	logger := slog.Default()
	mockEval := &mockEvalServiceer{}
	evalHandler := handlers.NewEvalHandler(mockEval, logger)

	testMux := http.NewServeMux()
	testMux.HandleFunc("GET /api/v1/evals/{id}/report", evalHandler.GetEvalReport)

	t.Run("GetEvalReport_Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/evals/run-123/report", nil)
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var resp svc.EvalReport
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, "run-123", resp.RunID)
		require.Equal(t, 85, resp.OverallScore)
	})
}
