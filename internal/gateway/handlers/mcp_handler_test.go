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

	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/service/mocks"
	"github.com/stretchr/testify/require"
)

func newTestMCPHandler() (*MCPHandler, *mock.MockTriggerService, *mock.MockEvalService, *mock.MockAssetIndexer) {
	mockTrigger := &mock.MockTriggerService{
		MatchTriggerFunc: func(ctx context.Context, input string, top int) ([]*service.MatchedPrompt, error) {
			return []*service.MatchedPrompt{
				{AssetID: "common/test", Name: "Test", Description: "Test prompt", Relevance: 0.95},
			}, nil
		},
		InjectVariablesFunc: func(ctx context.Context, prompt string, vars map[string]string) (string, error) {
			return "injected: " + prompt, nil
		},
	}
	mockEval := &mock.MockEvalService{
		RunEvalFunc: func(ctx context.Context, assetID, snapshotVersion string, caseIDs []string) (*service.EvalRun, error) {
			return &service.EvalRun{
				ID:        "run-123",
				Status:    service.EvalRunStatusRunning,
				CreatedAt: time.Now(),
			}, nil
		},
	}
	mockIndexer := &mock.MockAssetIndexer{
		SearchFunc: func(ctx context.Context, query string, filters service.SearchFilters) ([]service.AssetSummary, error) {
			return []service.AssetSummary{
				{ID: "common/test", Name: "Test Asset", Description: "A test asset"},
			}, nil
		},
		GetByIDFunc: func(ctx context.Context, id string) (*service.AssetDetail, error) {
			return &service.AssetDetail{
				ID:          id,
				Name:        "Test Asset",
				Description: "A test asset",
				BizLine:     "ai",
				Tags:        []string{"test"},
				State:       "created",
				Snapshots:   []service.SnapshotSummary{},
				Labels:      []service.LabelInfo{},
			}, nil
		},
	}
	logger := slog.Default()
	return NewMCPHandler(mockTrigger, mockEval, mockIndexer, logger), mockTrigger, mockEval, mockIndexer
}

func TestMCPHandler_HandlePOST_PromptsList(t *testing.T) {
	handler, _, _, _ := newTestMCPHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	body := `{"jsonrpc": "2.0", "method": "prompts/list", "id": 1}`
	req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MCPResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "2.0", resp.JSONRPC)
	require.NotNil(t, resp.Result)
}

func TestMCPHandler_HandlePOST_PromptsGet(t *testing.T) {
	handler, _, _, _ := newTestMCPHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	body := `{"jsonrpc": "2.0", "method": "prompts/get", "params": {"id": "common/test"}, "id": 1}`
	req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MCPResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "2.0", resp.JSONRPC)
}

func TestMCPHandler_HandlePOST_PromptsEval(t *testing.T) {
	handler, _, _, _ := newTestMCPHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	body := `{"jsonrpc": "2.0", "method": "prompts/eval", "params": {"id": "common/test", "snapshot_version": "v1"}, "id": 1}`
	req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MCPResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "2.0", resp.JSONRPC)
}

func TestMCPHandler_HandlePOST_InvalidRequest(t *testing.T) {
	handler, _, _, _ := newTestMCPHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code) // JSON-RPC always returns 200 for errors

	var resp MCPResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	require.Equal(t, -32700, resp.Error.Code) // Parse error
}

func TestMCPHandler_HandlePOST_MethodNotFound(t *testing.T) {
	handler, _, _, _ := newTestMCPHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	body := `{"jsonrpc": "2.0", "method": "unknown/method", "id": 1}`
	req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MCPResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
}

func TestMCPHandler_HandleSSE(t *testing.T) {
	handler, _, _, _ := newTestMCPHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /mcp/v1/sse", handler.HandleSSE)

	req := httptest.NewRequest("GET", "/mcp/v1/sse", nil)
	rec := httptest.NewRecorder()

	// Use a goroutine since SSE blocks
	go mux.ServeHTTP(rec, req)

	// Give it a moment to write headers
	time.Sleep(10 * time.Millisecond)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Header().Get("Content-Type"), "text/event-stream")
}