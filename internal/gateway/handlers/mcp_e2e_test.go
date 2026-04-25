package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/service/mocks"
	"github.com/stretchr/testify/require"
)

// mockSharedState holds state shared across the E2E test workflow
type mockSharedState struct {
	mu          sync.Mutex
	listResults []service.AssetSummary
	assetID     string
	content     string
	evalRunID   string
	sseEvents   []string
}

func newMockSharedState() *mockSharedState {
	return &mockSharedState{
		listResults: []service.AssetSummary{
			{ID: "common/test-prompt", Name: "Test Prompt", Description: "A test prompt for E2E", BizLine: "ai", Tags: []string{"e2e", "test"}},
			{ID: "common/another", Name: "Another Prompt", Description: "Another test prompt", BizLine: "ai", Tags: []string{"test"}},
		},
	}
}

// setupE2EMocks creates coordinated mocks for the full E2E workflow
func setupE2EMocks(state *mockSharedState) (*MCPHandler, *mockSharedState) {
	mockTrigger := &mocks.MockTriggerService{
		MatchTriggerFunc: func(ctx context.Context, input string, top int) ([]*service.MatchedPrompt, error) {
			state.mu.Lock()
			defer state.mu.Unlock()
			var matches []*service.MatchedPrompt
			for _, r := range state.listResults {
				matches = append(matches, &service.MatchedPrompt{
					AssetID: r.ID, Name: r.Name, Description: r.Description, Relevance: 0.95,
				})
			}
			return matches, nil
		},
		InjectVariablesFunc: func(ctx context.Context, prompt string, vars map[string]string) (string, error) {
			state.mu.Lock()
			defer state.mu.Unlock()
			state.content = "injected content"
			for k, v := range vars {
				state.content += fmt.Sprintf(" [%s=%s]", k, v)
			}
			return state.content, nil
		},
	}

	mockEval := &mocks.MockEvalService{
		RunEvalFunc: func(ctx context.Context, assetID, snapshotVersion string, caseIDs []string) (*service.EvalRun, error) {
			state.mu.Lock()
			defer state.mu.Unlock()
			state.assetID = assetID
			state.evalRunID = fmt.Sprintf("run-%d", time.Now().UnixNano())
			return &service.EvalRun{
				ID:        state.evalRunID,
				Status:    service.EvalRunStatusRunning,
				CreatedAt: time.Now(),
			}, nil
		},
		GetEvalRunFunc: func(ctx context.Context, runID string) (*service.EvalRun, error) {
			state.mu.Lock()
			defer state.mu.Unlock()
			return &service.EvalRun{
				ID:        runID,
				Status:    service.EvalRunStatusPassed,
				CreatedAt: time.Now(),
			}, nil
		},
		ListEvalRunsFunc: func(ctx context.Context, assetID string) ([]*service.EvalRun, error) {
			return []*service.EvalRun{}, nil
		},
		ListEvalCasesFunc: func(ctx context.Context, assetID string) ([]*service.EvalCase, error) {
			return []*service.EvalCase{}, nil
		},
		CompareEvalFunc: func(ctx context.Context, assetID string, v1, v2 string) (*service.CompareResult, error) {
			return &service.CompareResult{}, nil
		},
		GenerateReportFunc: func(ctx context.Context, runID string) (*service.EvalReport, error) {
			return &service.EvalReport{}, nil
		},
		DiagnoseEvalFunc: func(ctx context.Context, runID string) (*service.Diagnosis, error) {
			return &service.Diagnosis{}, nil
		},
	}

	mockIndexer := &mock.MockAssetIndexer{
		SearchFunc: func(ctx context.Context, query string, filters service.SearchFilters) ([]service.AssetSummary, error) {
			state.mu.Lock()
			defer state.mu.Unlock()
			return state.listResults, nil
		},
		GetByIDFunc: func(ctx context.Context, id string) (*service.AssetDetail, error) {
			state.mu.Lock()
			defer state.mu.Unlock()
			return &service.AssetDetail{
				ID:          id,
				Name:        "Test Prompt",
				Description: "A test prompt for E2E testing",
				BizLine:     "ai",
				Tags:        []string{"e2e", "test"},
				State:       "created",
				Snapshots: []service.SnapshotSummary{
					{ID: "snap-1", Version: "v1.0.0", Author: "test"},
				},
				Labels: []service.LabelInfo{
					{Name: "prod", SnapshotID: "snap-1"},
				},
			}, nil
		},
		SaveFunc: func(ctx context.Context, asset service.Asset) error {
			return nil
		},
		DeleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
		ReconcileFunc: func(ctx context.Context) (service.ReconcileReport, error) {
			return service.ReconcileReport{}, nil
		},
	}

	logger := slog.Default()
	handler := NewMCPHandler(mockTrigger, mockEval, mockIndexer, logger)
	return handler, state
}

// TestMCP_E2E_Workflow tests the complete MCP protocol workflow
func TestMCP_E2E_Workflow(t *testing.T) {
	state := newMockSharedState()
	handler, state := setupE2EMocks(state)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)
	mux.HandleFunc("GET /mcp/v1/sse", handler.HandleSSE)

	// Step 1: prompts/list
	t.Run("Step1_PromptsList", func(t *testing.T) {
		body := `{"jsonrpc": "2.0", "method": "prompts/list", "params": {"biz_line": "ai"}, "id": 1}`
		req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var resp MCPResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, "2.0", resp.JSONRPC)
		require.Nil(t, resp.Error)

		result, ok := resp.Result.(map[string]any)
		require.True(t, ok, "result should be a map")

		prompts, ok := result["prompts"].([]any)
		require.True(t, ok, "prompts should be an array")
		require.Len(t, prompts, 2, "should return 2 prompts")

		// Verify prompt structure
		firstPrompt, ok := prompts[0].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "common/test-prompt", firstPrompt["id"])
		require.Equal(t, "Test Prompt", firstPrompt["name"])
	})

	// Step 2: prompts/get with variables
	t.Run("Step2_PromptsGetWithVariables", func(t *testing.T) {
		body := `{"jsonrpc": "2.0", "method": "prompts/get", "params": {"id": "common/test-prompt", "variables": {"name": "Alice", "env": "prod"}}, "id": 2}`
		req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var resp MCPResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, "2.0", resp.JSONRPC)
		require.Nil(t, resp.Error)

		result, ok := resp.Result.(map[string]any)
		require.True(t, ok)

		// Verify content was injected with variables
		state.mu.Lock()
		content := state.content
		state.mu.Unlock()
		require.Equal(t, "injected content [name=Alice] [env=prod]", content)

		// Verify response structure
		require.Equal(t, "A test prompt for E2E testing", result["description"])
		require.Equal(t, "ai", result["biz_line"])
	})

	// Step 3: prompts/eval
	t.Run("Step3_PromptsEval", func(t *testing.T) {
		body := `{"jsonrpc": "2.0", "method": "prompts/eval", "params": {"id": "common/test-prompt", "snapshot_version": "v1.0.0", "case_id": "case-1"}, "id": 3}`
		req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var resp MCPResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Equal(t, "2.0", resp.JSONRPC)
		require.Nil(t, resp.Error)

		result, ok := resp.Result.(map[string]any)
		require.True(t, ok)

		runID, ok := result["run_id"].(string)
		require.True(t, ok)
		require.NotEmpty(t, runID)
		require.Contains(t, runID, "run-")

		status, ok := result["status"].(string)
		require.True(t, ok)
		require.Equal(t, "running", status)

		// Verify eval was triggered with correct asset ID
		state.mu.Lock()
		assetID := state.assetID
		state.mu.Unlock()
		require.Equal(t, "common/test-prompt", assetID)
	})

	// Step 4: SSE connection
	t.Run("Step4_SSEConnection", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/mcp/v1/sse", nil)
		rec := httptest.NewRecorder()

		// Start SSE connection in goroutine
		go mux.ServeHTTP(rec, req)

		// Give SSE time to establish connection and send initial event
		time.Sleep(50 * time.Millisecond)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Header().Get("Content-Type"), "text/event-stream")
		require.Contains(t, rec.Header().Get("Cache-Control"), "no-cache")

		// Read the body and check for initial connected event
		bodyStr := rec.Body.String()
		require.Contains(t, bodyStr, "connected")
	})
}

// TestMCP_E2E_PromptsGetMissingVariables tests prompts/get when variables are missing
func TestMCP_E2E_PromptsGetMissingVariables(t *testing.T) {
	state := newMockSharedState()
	handler, _ := setupE2EMocks(state)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	// Get without variables should still work
	body := `{"jsonrpc": "2.0", "method": "prompts/get", "params": {"id": "common/test-prompt"}, "id": 2}`
	req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MCPResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Nil(t, resp.Error)
}

// TestMCP_E2E_PromptsListWithFilters tests prompts/list with various filters
func TestMCP_E2E_PromptsListWithFilters(t *testing.T) {
	state := newMockSharedState()
	handler, _ := setupE2EMocks(state)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	tests := []struct {
		name   string
		filter string
	}{
		{"with biz_line filter", `{"biz_line": "ai"}`},
		{"with tag filter", `{"tag": "e2e"}`},
		{"with empty params", `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"jsonrpc": "2.0", "method": "prompts/list", "params": %s, "id": 1}`, tt.filter)
			req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)

			var resp MCPResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)
			require.Nil(t, resp.Error)
		})
	}
}

// TestMCP_E2E_ErrorHandling tests error cases in the workflow
func TestMCP_E2E_ErrorHandling(t *testing.T) {
	state := newMockSharedState()
	handler, _ := setupE2EMocks(state)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	t.Run("prompts/get with missing id", func(t *testing.T) {
		body := `{"jsonrpc": "2.0", "method": "prompts/get", "params": {}, "id": 1}`
		req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var resp MCPResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.NotNil(t, resp.Error)
		require.Equal(t, -32603, resp.Error.Code)
	})

	t.Run("prompts/eval with missing id", func(t *testing.T) {
		body := `{"jsonrpc": "2.0", "method": "prompts/eval", "params": {}, "id": 1}`
		req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)

		var resp MCPResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.NotNil(t, resp.Error)
		require.Equal(t, -32603, resp.Error.Code)
	})

	t.Run("invalid jsonrpc version", func(t *testing.T) {
		body := `{"jsonrpc": "1.0", "method": "prompts/list", "id": 1}`
		req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
	})
}

// TestMCP_E2E_SSEClientManagement tests SSE client registration and cleanup
func TestMCP_E2E_SSEClientManagement(t *testing.T) {
	state := newMockSharedState()
	handler, _ := setupE2EMocks(state)

	// Manually test client management through handler's internal state
	require.NotNil(t, handler.clients)
	require.Empty(t, handler.clients)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /mcp/v1/sse", handler.HandleSSE)

	// Create first SSE connection
	req1 := httptest.NewRequest("GET", "/mcp/v1/sse", nil)
	rec1 := httptest.NewRecorder()
	go mux.ServeHTTP(rec1, req1)
	time.Sleep(30 * time.Millisecond)

	handler.clientsMux.RLock()
	clientCount := len(handler.clients)
	handler.clientsMux.RUnlock()

	require.Equal(t, 1, clientCount, "should have 1 client after first connection")

	// Create second SSE connection
	req2 := httptest.NewRequest("GET", "/mcp/v1/sse", nil)
	rec2 := httptest.NewRecorder()
	go mux.ServeHTTP(rec2, req2)
	time.Sleep(30 * time.Millisecond)

	handler.clientsMux.RLock()
	clientCount = len(handler.clients)
	handler.clientsMux.RUnlock()

	require.Equal(t, 2, clientCount, "should have 2 clients after second connection")
}

// TestMCP_E2E_ConcurrentRequests tests handling of concurrent MCP requests
func TestMCP_E2E_ConcurrentRequests(t *testing.T) {
	state := newMockSharedState()
	handler, _ := setupE2EMocks(state)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	var wg sync.WaitGroup
	concurrency := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"jsonrpc": "2.0", "method": "prompts/list", "params": {}, "id": %d}`, id)
			req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)

			var resp MCPResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)
			require.Nil(t, resp.Error)
		}(i)
	}

	wg.Wait()
}