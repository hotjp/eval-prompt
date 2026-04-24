package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eval-prompt/plugins/llm"
	"github.com/stretchr/testify/require"
)

// MockLLMProvider is a mock LLM provider for testing
type MockLLMProvider struct{}

func (m *MockLLMProvider) Invoke(ctx context.Context, prompt string, model string, temperature float64) (*llm.LLMResponse, error) {
	return &llm.LLMResponse{Content: "mock response"}, nil
}

func (m *MockLLMProvider) InvokeWithSchema(ctx context.Context, prompt string, schema json.RawMessage) (json.RawMessage, error) {
	return nil, nil
}

func (m *MockLLMProvider) Name() string {
	return "mock"
}

func TestHealthz(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHealthRoutes(mux, nil, nil)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, rec.Body.String())
}

func TestReadyz_NoChecks(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHealthRoutes(mux, nil, nil)

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, rec.Body.String())
}

func TestHealthHandler_Healthz(t *testing.T) {
	h := NewHealthHandler(nil, nil)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	h.Healthz(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp HealthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "ok", resp.Status)
	require.NotEmpty(t, resp.Timestamp)
}

func TestHealthHandler_Readyz_DefaultStatus(t *testing.T) {
	h := NewHealthHandler(nil, nil)

	req := httptest.NewRequest("GET", "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Readyz(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp HealthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "ok", resp.Status)
}

func TestNoopHealthHandler(t *testing.T) {
	h := &NoopHealthHandler{}

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	h.Healthz(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp HealthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "ok", resp.Status)

	req2 := httptest.NewRequest("GET", "/readyz", nil)
	rec2 := httptest.NewRecorder()

	h.Readyz(rec2, req2)

	require.Equal(t, http.StatusOK, rec2.Code)

	var resp2 HealthResponse
	err = json.Unmarshal(rec2.Body.Bytes(), &resp2)
	require.NoError(t, err)
	require.Equal(t, "ok", resp2.Status)
}

// Verify MockLLMProvider implements llm.Provider interface
var _ llm.Provider = (*MockLLMProvider)(nil)