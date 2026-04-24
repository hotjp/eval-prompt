// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/eval-prompt/internal/service"
)

// AssetHandler handles asset CRUD API endpoints.
type AssetHandler struct {
	indexer service.AssetIndexer
	logger  *slog.Logger
}

// NewAssetHandler creates a new AssetHandler.
func NewAssetHandler(indexer service.AssetIndexer, logger *slog.Logger) *AssetHandler {
	return &AssetHandler{
		indexer: indexer,
		logger:  logger,
	}
}

// AssetResponse represents the API response for an asset.
type AssetResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	BizLine     string                 `json:"biz_line,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	State       string                 `json:"state,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Snapshots   []SnapshotResponse     `json:"snapshots,omitempty"`
	CreatedAt   time.Time              `json:"created_at,omitempty"`
	UpdatedAt   time.Time              `json:"updated_at,omitempty"`
}

// SnapshotResponse represents a snapshot in API response.
type SnapshotResponse struct {
	Version    string    `json:"version"`
	CommitHash string    `json:"commit_hash,omitempty"`
	Author     string    `json:"author,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	EvalScore  *float64  `json:"eval_score,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

// ListAssets handles GET /api/v1/assets.
func (h *AssetHandler) ListAssets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bizLine := r.URL.Query().Get("biz_line")
	tag := r.URL.Query().Get("tag")
	state := r.URL.Query().Get("state")

	filters := service.SearchFilters{
		BizLine: bizLine,
		State:  state,
	}
	if tag != "" {
		filters.Tags = []string{tag}
	}

	results, err := h.indexer.Search(ctx, "", filters)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to list assets: %v", err)
		return
	}

	assets := make([]AssetResponse, len(results))
	for i, r := range results {
		assets[i] = AssetResponse{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			BizLine:     r.BizLine,
			Tags:        r.Tags,
			State:       r.State,
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"assets": assets,
		"total":  len(assets),
	})
}

// GetAsset handles GET /api/v1/assets/{id}.
func (h *AssetHandler) GetAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	detail, err := h.indexer.GetByID(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", id)
		return
	}

	// Convert to response format
	snapshots := make([]SnapshotResponse, len(detail.Snapshots))
	for i, s := range detail.Snapshots {
		snapshots[i] = SnapshotResponse{
			Version:    s.Version,
			CommitHash: s.CommitHash,
			Author:    s.Author,
			Reason:    s.Reason,
			EvalScore: s.EvalScore,
			CreatedAt: s.CreatedAt,
		}
	}

	labels := make(map[string]string)
	for _, l := range detail.Labels {
		labels[l.Name] = l.SnapshotID
	}

	resp := AssetResponse{
		ID:          detail.ID,
		Name:        detail.Name,
		Description: detail.Description,
		BizLine:     detail.BizLine,
		Tags:        detail.Tags,
		State:       detail.State,
		Snapshots:   snapshots,
		Labels:      labels,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// CreateAssetRequest represents the request body for creating an asset.
type CreateAssetRequest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	BizLine     string   `json:"biz_line,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Content     string   `json:"content,omitempty"`
}

// CreateAsset handles POST /api/v1/assets.
func (h *AssetHandler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.ID == "" || req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "id and name are required")
		return
	}

	asset := service.Asset{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		BizLine:     req.BizLine,
		Tags:        req.Tags,
		State:       "created",
	}

	if err := h.indexer.Save(ctx, asset); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to create asset: %v", err)
		return
	}

	h.logger.Info("asset created", "asset_id", req.ID, "layer", "L5")

	h.writeJSON(w, http.StatusCreated, map[string]any{
		"id":      req.ID,
		"message": "asset created successfully",
	})
}

// UpdateAssetRequest represents the request body for updating an asset.
type UpdateAssetRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	BizLine     *string  `json:"biz_line,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	State       *string  `json:"state,omitempty"`
}

// UpdateAsset handles PUT /api/v1/assets/{id}.
func (h *AssetHandler) UpdateAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req UpdateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	// Get existing asset
	detail, err := h.indexer.GetByID(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", id)
		return
	}

	// Apply updates
	if req.Name != nil {
		detail.Name = *req.Name
	}
	if req.Description != nil {
		detail.Description = *req.Description
	}
	if req.BizLine != nil {
		detail.BizLine = *req.BizLine
	}
	if req.Tags != nil {
		detail.Tags = req.Tags
	}
	if req.State != nil {
		detail.State = *req.State
	}

	asset := service.Asset{
		ID:          detail.ID,
		Name:        detail.Name,
		Description: detail.Description,
		BizLine:     detail.BizLine,
		Tags:        detail.Tags,
		State:       detail.State,
	}

	if err := h.indexer.Save(ctx, asset); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to update asset: %v", err)
		return
	}

	h.logger.Info("asset updated", "asset_id", id, "layer", "L5")

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"message": "asset updated successfully",
	})
}

// DeleteAsset handles DELETE /api/v1/assets/{id}.
func (h *AssetHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.indexer.Delete(ctx, id); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to delete asset: %v", err)
		return
	}

	h.logger.Info("asset deleted", "asset_id", id, "layer", "L5")

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"message": "asset deleted successfully",
	})
}

// writeJSON writes a JSON response.
func (h *AssetHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (h *AssetHandler) writeError(w http.ResponseWriter, status int, format string, args ...any) {
	h.logger.Error(fmt.Sprintf(format, args...), "layer", "L5", "status", status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": fmt.Sprintf(format, args...),
	})
}
