package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	svc "github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/service/mocks"
	"github.com/stretchr/testify/require"
)

func newTestAssetHandler() (*AssetHandler, *mock.MockAssetIndexer) {
	mockIndexer := &mock.MockAssetIndexer{
		SearchFunc: func(ctx context.Context, query string, filters svc.SearchFilters) ([]svc.AssetSummary, error) {
			return []svc.AssetSummary{
				{ID: "common/test", Name: "Test Asset", Description: "A test asset", AssetType: "ai", Tags: []string{"test"}, State: "created"},
			}, nil
		},
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
		SaveFunc: func(ctx context.Context, asset svc.Asset) error {
			return nil
		},
		DeleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}
	logger := slog.Default()
	return NewAssetHandler(mockIndexer, mockIndexer, logger, nil), mockIndexer
}

func TestAssetHandler_ListAssets(t *testing.T) {
	handler, _ := newTestAssetHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/assets", handler.ListAssets)

	req := httptest.NewRequest("GET", "/api/v1/assets", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(1), resp["total"])
}

func TestAssetHandler_GetAsset(t *testing.T) {
	handler, _ := newTestAssetHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/assets/{id}", handler.GetAsset)

	req := httptest.NewRequest("GET", "/api/v1/assets/commontest", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp AssetResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "commontest", resp.ID)
}

func TestAssetHandler_GetAsset_NotFound(t *testing.T) {
	handler, mockIndexer := newTestAssetHandler()
	mockIndexer.GetByIDFunc = func(ctx context.Context, id string) (*svc.AssetDetail, error) {
		return nil, nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/assets/{id}", handler.GetAsset)

	req := httptest.NewRequest("GET", "/api/v1/assets/nonexistent", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAssetHandler_CreateAsset(t *testing.T) {
	handler, _ := newTestAssetHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/assets", handler.CreateAsset)

	body := `{"id": "common/newasset", "name": "New Asset", "description": "A new asset", "asset_type": "ai"}`
	req := httptest.NewRequest("POST", "/api/v1/assets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "common/newasset", resp["id"])
}

func TestAssetHandler_CreateAsset_MissingFields(t *testing.T) {
	handler, _ := newTestAssetHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/assets", handler.CreateAsset)

	body := `{"id": "common/newasset"}`
	req := httptest.NewRequest("POST", "/api/v1/assets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAssetHandler_UpdateAsset(t *testing.T) {
	handler, mockIndexer := newTestAssetHandler()
	updated := false
	mockIndexer.GetByIDFunc = func(ctx context.Context, id string) (*svc.AssetDetail, error) {
		return &svc.AssetDetail{
			ID:          id,
			Name:        "Original Name",
			Description: "Original desc",
			AssetType:     "ai",
			Tags:        []string{"test"},
			State:       "created",
			Snapshots:   []svc.SnapshotSummary{},
			Labels:      []svc.LabelInfo{},
		}, nil
	}
	mockIndexer.SaveFunc = func(ctx context.Context, asset svc.Asset) error {
		updated = true
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/assets/{id}", handler.UpdateAsset)

	body := `{"name": "Updated Name"}`
	req := httptest.NewRequest("PUT", "/api/v1/assets/commontest", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, updated)
}

func TestAssetHandler_DeleteAsset(t *testing.T) {
	handler, mockIndexer := newTestAssetHandler()
	deleted := false
	mockIndexer.DeleteFunc = func(ctx context.Context, id string) error {
		deleted = true
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/assets/{id}", handler.DeleteAsset)

	req := httptest.NewRequest("DELETE", "/api/v1/assets/commontest", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, deleted)
}