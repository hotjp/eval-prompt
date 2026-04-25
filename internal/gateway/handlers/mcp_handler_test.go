package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/service/mocks"
	"github.com/eval-prompt/internal/domain"
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
		RunEvalFunc: func(ctx context.Context, req *service.RunEvalRequest) (*domain.EvalExecution, error) {
			return &domain.EvalExecution{
				ID:     "execution-123",
				Status: domain.ExecutionStatusRunning,
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
				AssetType:     "ai",
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

// E2E MCP Workflow Tests

func TestMCPHandler_E2E_Workflow(t *testing.T) {
	// This test chains: prompts/list → prompts/get (with variables) → prompts/eval
	handler, _, _, _ := newTestMCPHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	// Step 1: prompts/list to get available prompts
	listBody := `{"jsonrpc": "2.0", "method": "prompts/list", "params": {"asset_type": "ai"}, "id": 1}`
	listReq := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(listBody))
	listReq.Header.Set("Content-Type", "application/json")
	listRec := httptest.NewRecorder()

	mux.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)

	var listResp MCPResponse
	err := json.Unmarshal(listRec.Body.Bytes(), &listResp)
	require.NoError(t, err)
	require.Nil(t, listResp.Error)
	require.NotNil(t, listResp.Result)

	// Extract prompt ID from list result
	resultMap := listResp.Result.(map[string]interface{})
	prompts := resultMap["prompts"].([]interface{})
	require.NotEmpty(t, prompts)
	prompt := prompts[0].(map[string]interface{})
	promptID := prompt["id"].(string)
	require.Equal(t, "common/test", promptID)

	// Step 2: prompts/get with variables
	getBody := fmt.Sprintf(`{"jsonrpc": "2.0", "method": "prompts/get", "params": {"id": "%s", "variables": {"name": "world"}}, "id": 2}`, promptID)
	getReq := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(getBody))
	getReq.Header.Set("Content-Type", "application/json")
	getRec := httptest.NewRecorder()

	mux.ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code)

	var getResp MCPResponse
	err = json.Unmarshal(getRec.Body.Bytes(), &getResp)
	require.NoError(t, err)
	require.Nil(t, getResp.Error)

	// Step 3: prompts/eval
	evalBody := fmt.Sprintf(`{"jsonrpc": "2.0", "method": "prompts/eval", "params": {"id": "%s", "snapshot_version": "v1"}, "id": 3}`, promptID)
	evalReq := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(evalBody))
	evalReq.Header.Set("Content-Type", "application/json")
	evalRec := httptest.NewRecorder()

	mux.ServeHTTP(evalRec, evalReq)
	require.Equal(t, http.StatusOK, evalRec.Code)

	var evalResp MCPResponse
	err = json.Unmarshal(evalRec.Body.Bytes(), &evalResp)
	require.NoError(t, err)
	require.Nil(t, evalResp.Error)
	require.NotNil(t, evalResp.Result)

	// Verify eval result contains execution_id and status
	evalResult := evalResp.Result.(map[string]interface{})
	require.Contains(t, evalResult, "execution_id")
	require.Contains(t, evalResult, "status")
}

func TestMCPHandler_E2E_PromptsGetWithVariables(t *testing.T) {
	// Set up mock that tracks variable injection
	mockTrigger := &mock.MockTriggerService{
		InjectVariablesFunc: func(ctx context.Context, prompt string, vars map[string]string) (string, error) {
			// Verify variables were passed correctly
			if vars["name"] != "Alice" {
				return "", fmt.Errorf("expected name=Alice, got %s", vars["name"])
			}
			return "Hello Alice! Your order is ready.", nil
		},
	}
	mockEval := &mock.MockEvalService{}
	mockIndexer := &mock.MockAssetIndexer{
		GetByIDFunc: func(ctx context.Context, id string) (*service.AssetDetail, error) {
			return &service.AssetDetail{
				ID:          id,
				Name:        "Greeting Prompt",
				Description: "A greeting prompt",
				AssetType:     "ai",
				Tags:        []string{"greeting"},
				State:       "created",
				Snapshots:   []service.SnapshotSummary{},
				Labels:      []service.LabelInfo{},
			}, nil
		},
	}
	logger := slog.Default()
	handler := NewMCPHandler(mockTrigger, mockEval, mockIndexer, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	// prompts/get with variables
	body := `{"jsonrpc": "2.0", "method": "prompts/get", "params": {"id": "common/greeting", "variables": {"name": "Alice"}}, "id": 1}`
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
	require.NotNil(t, resp.Result)
}

func TestMCPHandler_E2E_PromptsGetWithoutVariables(t *testing.T) {
	// Test that prompts/get works without variables
	mockTrigger := &mock.MockTriggerService{}
	mockEval := &mock.MockEvalService{}
	mockIndexer := &mock.MockAssetIndexer{
		GetByIDFunc: func(ctx context.Context, id string) (*service.AssetDetail, error) {
			return &service.AssetDetail{
				ID:          id,
				Name:        "Test Prompt",
				Description: "A test prompt",
				AssetType:     "ml",
				Tags:        []string{"test"},
				State:       "created",
				Snapshots:   []service.SnapshotSummary{},
				Labels:      []service.LabelInfo{},
			}, nil
		},
	}
	logger := slog.Default()
	handler := NewMCPHandler(mockTrigger, mockEval, mockIndexer, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	// prompts/get without variables
	body := `{"jsonrpc": "2.0", "method": "prompts/get", "params": {"id": "common/test"}, "id": 1}`
	req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MCPResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Nil(t, resp.Error)

	result := resp.Result.(map[string]interface{})
	require.Equal(t, "A test prompt", result["description"])
	require.Equal(t, "ml", result["asset_type"])
}

func TestMCPHandler_E2E_PromptsListWithFilters(t *testing.T) {
	// Test prompts/list with various filters
	searchedAssetType := ""
	mockIndexer := &mock.MockAssetIndexer{
		SearchFunc: func(ctx context.Context, query string, filters service.SearchFilters) ([]service.AssetSummary, error) {
			searchedAssetType = filters.AssetType
			return []service.AssetSummary{
				{ID: "ai/gpt-prompt", Name: "GPT Prompt", Description: "GPT prompt", AssetType: "ai", Tags: []string{"gpt"}},
				{ID: "ml/llm-prompt", Name: "LLM Prompt", Description: "LLM prompt", AssetType: "ml", Tags: []string{"llm"}},
			}, nil
		},
	}
	mockTrigger := &mock.MockTriggerService{}
	mockEval := &mock.MockEvalService{}
	logger := slog.Default()
	handler := NewMCPHandler(mockTrigger, mockEval, mockIndexer, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	// List with asset_type filter
	body := `{"jsonrpc": "2.0", "method": "prompts/list", "params": {"asset_type": "ai", "tag": "gpt"}, "id": 1}`
	req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "ai", searchedAssetType)

	var resp MCPResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Nil(t, resp.Error)

	result := resp.Result.(map[string]interface{})
	prompts := result["prompts"].([]interface{})
	require.Len(t, prompts, 2)
}

func TestMCPHandler_E2E_SSEWithEvalUpdates(t *testing.T) {
	// This test verifies the SSE connection can receive eval updates
	handler, _, _, _ := newTestMCPHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /mcp/v1/sse", handler.HandleSSE)
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	// Start SSE connection in background
	sseReq := httptest.NewRequest("GET", "/mcp/v1/sse", nil)
	sseRec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		mux.ServeHTTP(sseRec, sseReq)
		close(done)
	}()

	// Give SSE time to establish connection
	time.Sleep(20 * time.Millisecond)

	// Verify SSE headers
	require.Equal(t, http.StatusOK, sseRec.Code)
	require.Contains(t, sseRec.Header().Get("Content-Type"), "text/event-stream")
	require.Contains(t, sseRec.Header().Get("Cache-Control"), "no-cache")
	require.Contains(t, sseRec.Header().Get("Connection"), "keep-alive")

	// Trigger an eval which would normally send SSE update
	evalBody := `{"jsonrpc": "2.0", "method": "prompts/eval", "params": {"id": "common/test", "snapshot_version": "v1"}, "id": 1}`
	evalReq := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(evalBody))
	evalReq.Header.Set("Content-Type", "application/json")
	evalRec := httptest.NewRecorder()

	mux.ServeHTTP(evalRec, evalReq)
	require.Equal(t, http.StatusOK, evalRec.Code)
}

func TestMCPHandler_E2E_PromptsListEmptyResult(t *testing.T) {
	// Test prompts/list when no prompts match
	mockIndexer := &mock.MockAssetIndexer{
		SearchFunc: func(ctx context.Context, query string, filters service.SearchFilters) ([]service.AssetSummary, error) {
			return []service.AssetSummary{}, nil
		},
	}
	mockTrigger := &mock.MockTriggerService{}
	mockEval := &mock.MockEvalService{}
	logger := slog.Default()
	handler := NewMCPHandler(mockTrigger, mockEval, mockIndexer, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	body := `{"jsonrpc": "2.0", "method": "prompts/list", "params": {"asset_type": "nonexistent"}, "id": 1}`
	req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MCPResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Nil(t, resp.Error)

	result := resp.Result.(map[string]interface{})
	prompts := result["prompts"].([]interface{})
	require.Empty(t, prompts)
}

func TestMCPHandler_E2E_PromptsGetNotFound(t *testing.T) {
	// Test prompts/get when asset doesn't exist
	mockIndexer := &mock.MockAssetIndexer{
		GetByIDFunc: func(ctx context.Context, id string) (*service.AssetDetail, error) {
			return nil, fmt.Errorf("asset not found: %s", id)
		},
	}
	mockTrigger := &mock.MockTriggerService{}
	mockEval := &mock.MockEvalService{}
	logger := slog.Default()
	handler := NewMCPHandler(mockTrigger, mockEval, mockIndexer, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /mcp/v1", handler.HandlePOST)

	body := `{"jsonrpc": "2.0", "method": "prompts/get", "params": {"id": "nonexistent/prompt"}, "id": 1}`
	req := httptest.NewRequest("POST", "/mcp/v1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp MCPResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.NotNil(t, resp.Error)
	require.Equal(t, -32603, resp.Error.Code) // JSON-RPC internal error
}