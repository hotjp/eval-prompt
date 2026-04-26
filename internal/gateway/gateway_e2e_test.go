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

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/gateway/handlers"
	"github.com/eval-prompt/internal/gateway/middleware"
	svc "github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/plugins/search"
	"github.com/stretchr/testify/require"
)

// mockEvalServiceer is a mock implementation of EvalServiceer for E2E testing.
type mockEvalServiceer struct {
	RunEvalFunc         func(ctx context.Context, req *svc.RunEvalRequest) (*domain.EvalExecution, error)
	GetEvalRunFunc      func(ctx context.Context, runID string) (*svc.EvalRun, error)
	ListEvalRunsFunc    func(ctx context.Context, assetID string) ([]*svc.EvalRun, error)
	CompareEvalFunc     func(ctx context.Context, assetID string, v1, v2 string) (*svc.CompareResult, error)
	GenerateReportFunc  func(ctx context.Context, runID string) (*svc.EvalReport, error)
	DiagnoseEvalFunc    func(ctx context.Context, runID string) (*svc.Diagnosis, error)
	ListEvalCasesFunc   func(ctx context.Context, assetID string) ([]*domain.EvalCase, error)
	GetExecutionFunc    func(ctx context.Context, executionID string) (*domain.EvalExecution, error)
	CancelExecutionFunc func(ctx context.Context, executionID string) error
	ListExecutionsFunc  func(ctx context.Context, offset, limit int) ([]*domain.EvalExecution, int, error)
}

func (m *mockEvalServiceer) RunEval(ctx context.Context, req *svc.RunEvalRequest) (*domain.EvalExecution, error) {
	if m.RunEvalFunc != nil {
		return m.RunEvalFunc(ctx, req)
	}
	return &domain.EvalExecution{
		ID:     "execution-e2e-001",
		Status: domain.ExecutionStatusRunning,
	}, nil
}

func (m *mockEvalServiceer) GetEvalRun(ctx context.Context, runID string) (*svc.EvalRun, error) {
	if m.GetEvalRunFunc != nil {
		return m.GetEvalRunFunc(ctx, runID)
	}
	return &svc.EvalRun{
		ID:                 runID,
		Status:             svc.EvalRunStatusPassed,
		DeterministicScore: 85,
		CreatedAt:          time.Now(),
	}, nil
}

func (m *mockEvalServiceer) ListEvalRuns(ctx context.Context, assetID string) ([]*svc.EvalRun, error) {
	if m.ListEvalRunsFunc != nil {
		return m.ListEvalRunsFunc(ctx, assetID)
	}
	return []*svc.EvalRun{
		{ID: "run-1", AssetID: assetID, Status: svc.EvalRunStatusPassed, DeterministicScore: 85},
		{ID: "run-2", AssetID: assetID, Status: svc.EvalRunStatusFailed, DeterministicScore: 60},
	}, nil
}

func (m *mockEvalServiceer) CompareEval(ctx context.Context, assetID string, v1, v2 string) (*svc.CompareResult, error) {
	if m.CompareEvalFunc != nil {
		return m.CompareEvalFunc(ctx, assetID, v1, v2)
	}
	return &svc.CompareResult{
		AssetID:    assetID,
		Version1:   v1,
		Version2:   v2,
		ScoreDelta: 15,
	}, nil
}

func (m *mockEvalServiceer) GenerateReport(ctx context.Context, runID string) (*svc.EvalReport, error) {
	if m.GenerateReportFunc != nil {
		return m.GenerateReportFunc(ctx, runID)
	}
	return &svc.EvalReport{
		RunID:        runID,
		Status:       svc.EvalRunStatusPassed,
		OverallScore: 85,
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

func (m *mockEvalServiceer) ListEvalCases(ctx context.Context, assetID string) ([]*domain.EvalCase, error) {
	if m.ListEvalCasesFunc != nil {
		return m.ListEvalCasesFunc(ctx, assetID)
	}
	return []*domain.EvalCase{
		{ID: domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAV"), AssetID: domain.MustNewID("01ARZ3NDEKTSV4RRFFQ69G5FAW"), Name: "Test Case 1"},
	}, nil
}

func (m *mockEvalServiceer) GetExecution(ctx context.Context, executionID string) (*domain.EvalExecution, error) {
	if m.GetExecutionFunc != nil {
		return m.GetExecutionFunc(ctx, executionID)
	}
	return &domain.EvalExecution{
		ID:     executionID,
		Status: domain.ExecutionStatusRunning,
	}, nil
}

func (m *mockEvalServiceer) CancelExecution(ctx context.Context, executionID string) error {
	if m.CancelExecutionFunc != nil {
		return m.CancelExecutionFunc(ctx, executionID)
	}
	return nil
}

func (m *mockEvalServiceer) ListExecutions(ctx context.Context, offset, limit int) ([]*domain.EvalExecution, int, error) {
	if m.ListExecutionsFunc != nil {
		return m.ListExecutionsFunc(ctx, offset, limit)
	}
	return []*domain.EvalExecution{}, 0, nil
}

// setupTestRouter creates a test router with real implementations for search and mock for eval.
func setupTestRouter(t *testing.T) http.Handler {
	logger := slog.Default()
	indexer := search.NewIndexer()
	mockEval := &mockEvalServiceer{}

	assetHandler := handlers.NewAssetHandler(indexer, indexer, logger, nil)
	evalHandler := handlers.NewEvalHandler(mockEval, indexer, logger)

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
	ctx := context.Background()
	indexer := search.NewIndexer()
	logger := slog.Default()
	assetHandler := handlers.NewAssetHandler(indexer, indexer, logger, nil)
	evalHandler := handlers.NewEvalHandler(&mockEvalServiceer{}, indexer, logger)

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

	assetID := "e2e-test-asset"
	assetName := "E2E Test Asset"
	assetDescription := "Test asset for E2E workflow"
	bizLine := "ml"
	_ = []string{"e2e", "test"} // tags declared for potential future use

	// Step 1: POST /api/v1/assets - Create asset
	t.Run("Step1_CreateAsset", func(t *testing.T) {
		body := `{
			"id": "` + assetID + `",
			"name": "` + assetName + `",
			"description": "` + assetDescription + `",
			"asset_type": "` + bizLine + `",
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
		require.Equal(t, bizLine, resp.AssetType)
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
		require.NotEmpty(t, resp.ExecutionID)
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
		newAssetID := "e2e-new-asset"
		body := `{
			"id": "` + newAssetID + `",
			"name": "New Test Asset",
			"description": "Testing new asset workflow",
			"asset_type": "data"
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
	assetHandler := handlers.NewAssetHandler(indexer, indexer, logger, nil)

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

		// Delete returns 404 when asset not found
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}

// TestRESTAPI_E2E_Filters tests filtering capabilities in list endpoint.
func TestRESTAPI_E2E_Filters(t *testing.T) {
	logger := slog.Default()
	indexer := search.NewIndexer()
	assetHandler := handlers.NewAssetHandler(indexer, indexer, logger, nil)

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
		body := `{"id":"` + a.id + `","name":"` + a.name + `","asset_type":"` + a.bizLine + `","tags":[` + strings.Join(func() []string {
			var tags []string
			for _, t := range a.tags {
				tags = append(tags, `"`+t+`"`)
			}
			return tags
		}(), ",") + `]}`
		req := httptest.NewRequest("POST", "/api/v1/assets", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		testMux.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code)
	}

	t.Run("FilterByAssetType", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets?asset_type=ml", nil)
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]any
		json.Unmarshal(rec.Body.Bytes(), &resp)
		// Should return 2 ml assets (asset1 and asset3)
		require.Equal(t, float64(2), resp["total"], "should have 2 ml assets")
	})

	t.Run("FilterByState", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/assets?state=created", nil)
		rec := httptest.NewRecorder()

		testMux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp map[string]any
		json.Unmarshal(rec.Body.Bytes(), &resp)
		// All assets are created with state "created" by the handler
		require.Equal(t, float64(3), resp["total"], "should have 3 created assets (all assets have state=created)")
	})
}

// TestRESTAPI_E2E_EvalReports tests eval report generation.
func TestRESTAPI_E2E_EvalReports(t *testing.T) {
	logger := slog.Default()
	mockEval := &mockEvalServiceer{}
	mockIndexer := search.NewIndexer()
	evalHandler := handlers.NewEvalHandler(mockEval, mockIndexer, logger)

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

// setupE2ERouter creates a test router with middleware chain for E2E testing
func setupE2ERouter() (*http.ServeMux, *mockEvalServiceer, *e2eMockAssetIndexer, *e2eMockTriggerService) {
	mux := http.NewServeMux()
	logger := slog.Default()

	mockEval := &mockEvalServiceer{
		RunEvalFunc: func(ctx context.Context, req *svc.RunEvalRequest) (*domain.EvalExecution, error) {
			return &domain.EvalExecution{
				ID:     "execution-e2e-001",
				Status: domain.ExecutionStatusRunning,
			}, nil
		},
		GetEvalRunFunc: func(ctx context.Context, runID string) (*svc.EvalRun, error) {
			return &svc.EvalRun{
				ID:        runID,
				AssetID:   "common/test",
				Status:    svc.EvalRunStatusPassed,
				CreatedAt: time.Now(),
			}, nil
		},
		CompareEvalFunc: func(ctx context.Context, assetID string, v1, v2 string) (*svc.CompareResult, error) {
			return &svc.CompareResult{
				AssetID:    assetID,
				Version1:   v1,
				Version2:   v2,
				ScoreDelta: 10,
			}, nil
		},
	}

	mockIndexer := &e2eMockAssetIndexer{
		GetByIDFunc: func(ctx context.Context, id string) (*svc.AssetDetail, error) {
			return &svc.AssetDetail{
				ID:          id,
				Name:        "Test Asset",
				Description: "A test asset",
				AssetType:     "ai",
				Tags:        []string{"test"},
				State:       "created",
				Snapshots:   []svc.SnapshotSummary{},
				Labels:      []svc.LabelInfo{},
			}, nil
		},
		SearchFunc: func(ctx context.Context, query string, filters svc.SearchFilters) ([]svc.AssetSummary, error) {
			return []svc.AssetSummary{
				{ID: "common/test", Name: "Test Asset", Description: "A test asset", AssetType: "ai", Tags: []string{"test"}, State: "created"},
			}, nil
		},
		SaveFunc: func(ctx context.Context, asset svc.Asset) error {
			return nil
		},
		DeleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}

	mockTrigger := &e2eMockTriggerService{
		MatchTriggerFunc: func(ctx context.Context, input string, top int) ([]*svc.MatchedPrompt, error) {
			return []*svc.MatchedPrompt{
				{AssetID: "common/test", Name: "Test", Description: "Test prompt", Relevance: 0.95},
			}, nil
		},
		InjectVariablesFunc: func(ctx context.Context, prompt string, vars map[string]string) (string, error) {
			return "injected: " + prompt, nil
		},
	}

	// Create handlers
	mcpHandler := handlers.NewMCPHandler(mockTrigger, mockIndexer, logger)
	assetHandler := handlers.NewAssetHandler(mockIndexer, mockIndexer, logger, nil)
	evalHandler := handlers.NewEvalHandler(mockEval, mockIndexer, logger)

	// Build middleware chain
	chain := func(h http.Handler) http.Handler {
		h = middleware.Recover(logger)(h)
		h = middleware.RequestID()(h)
		h = middleware.Metrics(middleware.NewMetricsCollector())(h)
		h = middleware.Logging(logger)(h)
		h = middleware.CORS([]string{"*"})(h)
		return h
	}

	// Register MCP endpoints
	mux.Handle("GET /mcp/v1/sse", chain(http.HandlerFunc(mcpHandler.HandleSSE)))
	mux.Handle("POST /mcp/v1", chain(http.HandlerFunc(mcpHandler.HandlePOST)))

	// Register REST API endpoints
	mux.HandleFunc("POST /api/v1/assets", assetHandler.CreateAsset)
	mux.HandleFunc("GET /api/v1/assets", assetHandler.ListAssets)
	mux.HandleFunc("GET /api/v1/assets/{id}", assetHandler.GetAsset)
	mux.HandleFunc("PUT /api/v1/assets/{id}", assetHandler.UpdateAsset)
	mux.HandleFunc("DELETE /api/v1/assets/{id}", assetHandler.DeleteAsset)

	mux.HandleFunc("POST /api/v1/evals/run", evalHandler.RunEval)
	mux.HandleFunc("GET /api/v1/evals/{id}", evalHandler.GetEvalRun)
	mux.HandleFunc("POST /api/v1/evals/compare", evalHandler.CompareEval)
	mux.HandleFunc("GET /api/v1/evals", evalHandler.ListEvalRuns)

	// Health check endpoints
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return mux, mockEval, mockIndexer, mockTrigger
}

// e2eMockAssetIndexer is a mock implementation of AssetIndexer for E2E testing.
type e2eMockAssetIndexer struct {
	SearchFunc  func(ctx context.Context, query string, filters svc.SearchFilters) ([]svc.AssetSummary, error)
	GetByIDFunc func(ctx context.Context, id string) (*svc.AssetDetail, error)
	SaveFunc    func(ctx context.Context, asset svc.Asset) error
	DeleteFunc  func(ctx context.Context, id string) error
}

func (m *e2eMockAssetIndexer) Reconcile(ctx context.Context) (svc.ReconcileReport, error) {
	return svc.ReconcileReport{}, nil
}

func (m *e2eMockAssetIndexer) Search(ctx context.Context, query string, filters svc.SearchFilters) ([]svc.AssetSummary, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, filters)
	}
	return nil, nil
}

func (m *e2eMockAssetIndexer) GetByID(ctx context.Context, id string) (*svc.AssetDetail, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return &svc.AssetDetail{ID: id, Name: "Test Asset", State: "created"}, nil
}

func (m *e2eMockAssetIndexer) Save(ctx context.Context, asset svc.Asset) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, asset)
	}
	return nil
}

func (m *e2eMockAssetIndexer) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *e2eMockAssetIndexer) CreatePlaceholder(ctx context.Context, id, name, bizLine string, tags []string, category string) error {
	return nil
}

func (m *e2eMockAssetIndexer) GetFileContent(ctx context.Context, id string) (string, error) {
	return "", nil
}

func (m *e2eMockAssetIndexer) SaveFileContent(ctx context.Context, id, content, commitMsg string) (string, error) {
	return "mock-commit-hash", nil
}

func (m *e2eMockAssetIndexer) GetFrontmatter(ctx context.Context, id string) (*domain.FrontMatter, error) {
	return &domain.FrontMatter{ID: id, Name: "Test Asset"}, nil
}

func (m *e2eMockAssetIndexer) UpdateFrontmatter(ctx context.Context, id string, updater func(*domain.FrontMatter) error, commitMsg string) (string, error) {
	return "mock-commit-hash", nil
}

func (m *e2eMockAssetIndexer) WriteContent(ctx context.Context, id string, updater func(*domain.FrontMatter) error, newBody string, commitMsg string) (string, error) {
	return "mock-commit-hash", nil
}

func (m *e2eMockAssetIndexer) GetBody(ctx context.Context, id string) (string, error) {
	return "# Test Content", nil
}

func (m *e2eMockAssetIndexer) ReInit(ctx context.Context, path string) error {
	return nil
}

func (m *e2eMockAssetIndexer) CommitFile(ctx context.Context, id string, commitMsg string) (string, error) {
	return "mock-commit-hash", nil
}

func (m *e2eMockAssetIndexer) CommitFiles(ctx context.Context, ids []string, commitMsg string) (map[string]string, error) {
	results := make(map[string]string)
	for _, id := range ids {
		results[id] = "mock-commit-hash"
	}
	return results, nil
}

func (m *e2eMockAssetIndexer) WriteFileOnly(ctx context.Context, id string, updater func(*domain.FrontMatter) error, newBody string) error {
	return nil
}

func (m *e2eMockAssetIndexer) UpdateFrontmatterFileOnly(ctx context.Context, id string, updater func(*domain.FrontMatter) error) error {
	return nil
}

// e2eMockTriggerService is a mock implementation of TriggerServicer for E2E testing.
type e2eMockTriggerService struct {
	MatchTriggerFunc         func(ctx context.Context, input string, top int) ([]*svc.MatchedPrompt, error)
	ValidateAntiPatternsFunc func(ctx context.Context, prompt string) error
	InjectVariablesFunc      func(ctx context.Context, prompt string, vars map[string]string) (string, error)
}

func (m *e2eMockTriggerService) MatchTrigger(ctx context.Context, input string, top int) ([]*svc.MatchedPrompt, error) {
	if m.MatchTriggerFunc != nil {
		return m.MatchTriggerFunc(ctx, input, top)
	}
	return nil, nil
}

func (m *e2eMockTriggerService) ValidateAntiPatterns(ctx context.Context, prompt string) error {
	if m.ValidateAntiPatternsFunc != nil {
		return m.ValidateAntiPatternsFunc(ctx, prompt)
	}
	return nil
}

func (m *e2eMockTriggerService) InjectVariables(ctx context.Context, prompt string, vars map[string]string) (string, error) {
	if m.InjectVariablesFunc != nil {
		return m.InjectVariablesFunc(ctx, prompt, vars)
	}
	return "", nil
}

// E2E_Middleware_RequestID_NotPresent tests that X-Request-ID is generated when not provided
func TestE2E_Middleware_RequestID_NotPresent(t *testing.T) {
	mux, _, _, _ := setupE2ERouter()

	body := strings.NewReader(`{"jsonrpc": "2.0", "method": "prompts/list", "id": 1}`)
	req := httptest.NewRequest("POST", "/mcp/v1", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.NotEmpty(t, rec.Header().Get("X-Request-ID"), "X-Request-ID should be set in response")
}

// E2E_Middleware_RequestID_CustomPresent tests that custom X-Request-ID is echoed back
func TestE2E_Middleware_RequestID_CustomPresent(t *testing.T) {
	mux, _, _, _ := setupE2ERouter()

	body := strings.NewReader(`{"jsonrpc": "2.0", "method": "prompts/list", "id": 1}`)
	req := httptest.NewRequest("POST", "/mcp/v1", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "my-custom-id-12345")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, "my-custom-id-12345", rec.Header().Get("X-Request-ID"), "X-Request-ID should echo back the custom value")
}

// E2E_Middleware_CORS_OriginHeader tests that CORS Origin header is set correctly
func TestE2E_Middleware_CORS_OriginHeader(t *testing.T) {
	mux, _, _, _ := setupE2ERouter()

	body := strings.NewReader(`{"jsonrpc": "2.0", "method": "prompts/list", "id": 1}`)
	req := httptest.NewRequest("POST", "/mcp/v1", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "http://example.com", rec.Header().Get("Access-Control-Allow-Origin"), "CORS Origin header should be set")
}

// E2E_Middleware_NotFound_Path tests that nonexistent paths return proper response
func TestE2E_Middleware_NotFound_Path(t *testing.T) {
	mux, _, _, _ := setupE2ERouter()

	req := httptest.NewRequest("GET", "/nonexistent/path", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Should fall through to static handler or return 404
	require.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusNotFound,
		"Expected 200 or 404 for nonexistent path, got %d", rec.Code)
}

// E2E_REST_API_RunEval_Validation tests validation for run eval endpoint
func TestE2E_REST_API_RunEval_Validation(t *testing.T) {
	logger := slog.Default()
	mockEval := &mockEvalServiceer{}
	mockIndexer := search.NewIndexer()
	evalHandler := handlers.NewEvalHandler(mockEval, mockIndexer, logger)

	testMux := http.NewServeMux()
	testMux.HandleFunc("POST /api/v1/evals/run", evalHandler.RunEval)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "missing asset_id returns 400",
			body:       `{"snapshot_version": "v1"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty body returns 400",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/evals/run", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			testMux.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// E2E_REST_API_CompareEval_Validation tests validation for compare eval endpoint
func TestE2E_REST_API_CompareEval_Validation(t *testing.T) {
	logger := slog.Default()
	mockEval := &mockEvalServiceer{}
	mockIndexer := search.NewIndexer()
	evalHandler := handlers.NewEvalHandler(mockEval, mockIndexer, logger)

	testMux := http.NewServeMux()
	testMux.HandleFunc("POST /api/v1/evals/compare", evalHandler.CompareEval)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "missing version1 returns 400",
			body:       `{"asset_id": "common/test", "version2": "v2"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing version2 returns 400",
			body:       `{"asset_id": "common/test", "version1": "v1"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing asset_id returns 400",
			body:       `{"version1": "v1", "version2": "v2"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/evals/compare", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			testMux.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

// E2E_REST_API_HealthEndpoints tests health check endpoints
func TestE2E_REST_API_HealthEndpoints(t *testing.T) {
	mux, _, _, _ := setupE2ERouter()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "healthz returns OK",
			path:       "/healthz",
			wantStatus: http.StatusOK,
			wantBody:   "OK",
		},
		{
			name:       "readyz returns OK",
			path:       "/readyz",
			wantStatus: http.StatusOK,
			wantBody:   "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code)
			require.Contains(t, rec.Body.String(), tt.wantBody)
		})
	}
}
