// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/yamlutil"
)

// AssetHandler handles asset CRUD API endpoints.
type AssetHandler struct {
	indexer     service.AssetIndexer
	fileManager service.AssetFileManager
	logger      *slog.Logger
}

// NewAssetHandler creates a new AssetHandler.
func NewAssetHandler(indexer service.AssetIndexer, fileManager service.AssetFileManager, logger *slog.Logger) *AssetHandler {
	return &AssetHandler{
		indexer:     indexer,
		fileManager: fileManager,
		logger:      logger,
	}
}

// AssetResponse represents the API response for an asset.
type AssetResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	BizLine     string             `json:"biz_line,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
	State       string             `json:"state,omitempty"`
	Labels      map[string]string  `json:"labels,omitempty"`
	Snapshots   []SnapshotResponse `json:"snapshots,omitempty"`
	CreatedAt   time.Time          `json:"created_at,omitempty"`
	UpdatedAt   time.Time          `json:"updated_at,omitempty"`
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
//
//	@Summary List assets
//	@Description Get all assets with optional filtering
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param biz_line query string false "Business line filter"
//	@Param tag query string false "Tag filter"
//	@Param state query string false "State filter"
//	@Success 200 {object} map[string]interface{}
//	@Router /api/v1/assets [get]
func (h *AssetHandler) ListAssets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	bizLine := r.URL.Query().Get("biz_line")
	tag := r.URL.Query().Get("tag")
	state := r.URL.Query().Get("state")

	filters := service.SearchFilters{
		BizLine: bizLine,
		State:   state,
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
//
//	@Summary Get asset by ID
//	@Description Get a single asset by its ID
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param id path string true "Asset ID"
//	@Success 200 {object} AssetResponse
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Router /api/v1/assets/{id} [get]
func (h *AssetHandler) GetAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	detail, err := h.indexer.GetByID(ctx, id)
	if err != nil || detail == nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", id)
		return
	}

	// Convert to response format
	snapshots := make([]SnapshotResponse, len(detail.Snapshots))
	for i, s := range detail.Snapshots {
		snapshots[i] = SnapshotResponse{
			Version:    s.Version,
			CommitHash: s.CommitHash,
			Author:     s.Author,
			Reason:     s.Reason,
			EvalScore:  s.EvalScore,
			CreatedAt:  s.CreatedAt,
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
//
//	@Summary Create asset
//	@Description Create a new asset
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param request body CreateAssetRequest true "Asset creation request"
//	@Success 201 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/assets [post]
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

	// Create placeholder file and commit to Git (best effort — non-fatal if git unavailable)
	if err := h.indexer.CreatePlaceholder(ctx, req.ID, req.Name, req.BizLine, req.Tags); err != nil {
		// Log but don't fail — placeholder is a courtesy for Git users
		h.logger.Warn("failed to create placeholder file", "asset_id", req.ID, "error", err, "layer", "L5")
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
//
//	@Summary Update asset
//	@Description Update an existing asset
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param id path string true "Asset ID"
//	@Param request body UpdateAssetRequest true "Asset update request"
//	@Success 200 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/assets/{id} [put]
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

// SaveContentRequest represents the request body for saving file content.
type SaveContentRequest struct {
	Content       string `json:"content"`
	CommitMessage string `json:"commit_message,omitempty"`
	ContentHash   string `json:"content_hash,omitempty"` // for conflict detection
}

// GetAssetContent handles GET /api/v1/assets/{id}/content.
// Returns only the markdown body (after frontmatter), with frontmatter stripped.
func (h *AssetHandler) GetAssetContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	fullContent, err := h.indexer.GetFileContent(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "content not found: %s", err)
		return
	}

	// Strip frontmatter — find the second ---
	lines := strings.Split(fullContent, "\n")
	frontmatterEnd := -1
	inFrontmatter := false
	for i, line := range lines {
		if i == 0 && strings.HasPrefix(line, "---") {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.HasPrefix(line, "---") {
			frontmatterEnd = i
			break
		}
	}

	var body string
	var contentHash string
	var updatedAt string
	if frontmatterEnd >= 0 {
		// Parse frontmatter to get content_hash and updated_at
		frontmatterBlock := strings.Join(lines[1:frontmatterEnd], "\n")
		fullFrontmatter := "---\n" + frontmatterBlock + "\n---"
		fm, _, _ := yamlutil.ParseFrontMatter(fullFrontmatter)
		if fm != nil {
			contentHash = fm.ContentHash
			if !fm.UpdatedAt.IsZero() {
				updatedAt = fm.UpdatedAt.Format(time.RFC3339)
				w.Header().Set("Last-Modified", updatedAt)
			}
		}
		body = strings.TrimSpace(strings.Join(lines[frontmatterEnd+1:], "\n"))
	} else {
		// No frontmatter found, return as-is
		body = fullContent
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":           id,
		"content":      body,
		"content_hash":  contentHash,
		"updated_at":   updatedAt,
	})
}

// SaveAssetContent handles PUT /api/v1/assets/{id}/content.
// Reads existing frontmatter, replaces the body, writes back full file.
func (h *AssetHandler) SaveAssetContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req SaveContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.Content == "" {
		h.writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	commitMsg := req.CommitMessage
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("Update prompt %s", id)
	}

	// Conflict detection: check content_hash before writing
	if req.ContentHash != "" {
		fm, err := h.fileManager.GetFrontmatter(ctx, id)
		if err == nil && fm.ContentHash != "" && fm.ContentHash != req.ContentHash {
			h.writeError(w, http.StatusConflict, "content has been modified by another session")
			return
		}
	}

	// Compute new hash
	hashed := sha256.Sum256([]byte(req.Content))
	newHash := hex.EncodeToString(hashed[:8])
	now := time.Now()

	// WriteContent replaces the body with req.Content and updates frontmatter
	hash, err := h.fileManager.WriteContent(ctx, id, func(fm *domain.FrontMatter) error {
		fm.ContentHash = newHash
		fm.UpdatedAt = now
		return nil
	}, req.Content, commitMsg)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to save content: %v", err)
		return
	}

	h.logger.Info("asset content saved", "asset_id", id, "commit", hash, "layer", "L5")

	// Preference-Applied: return=representation
	w.Header().Set("Preference-Applied", "return=representation")
	w.Header().Set("Last-Modified", now.Format(time.RFC3339))

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":           id,
		"content":      req.Content,
		"commit":       hash,
		"content_hash": newHash,
		"updated_at":   now.Format(time.RFC3339),
		"message":      "content saved successfully",
	})
}

// ArchiveAsset handles POST /api/v1/assets/{id}/archive.
//
//	@Summary Archive asset
//	@Description Archive an asset (soft delete)
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param id path string true "Asset ID"
//	@Success 200 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/assets/{id}/archive [post]
func (h *AssetHandler) ArchiveAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Update frontmatter state to archived
	hash, err := h.fileManager.UpdateFrontmatter(ctx, id, func(fm *domain.FrontMatter) error {
		fm.State = "archived"
		fm.UpdatedAt = time.Now()
		return nil
	}, fmt.Sprintf("Archive asset %s", id))
	if err != nil {
		h.writeError(w, http.StatusNotFound, "asset file not found: %s", err)
		return
	}

	// Update in-memory index so UI reflects change immediately
	fm, err := h.fileManager.GetFrontmatter(ctx, id)
	if err == nil {
		asset := service.Asset{
			ID:          fm.ID,
			Name:        fm.Name,
			Description: fm.Description,
			BizLine:     fm.BizLine,
			Tags:        fm.Tags,
			ContentHash: fm.ContentHash,
			State:       fm.State,
		}
		if err := h.indexer.Save(ctx, asset); err != nil {
			h.logger.Warn("failed to update index after archive", "asset_id", id, "error", err, "layer", "L5")
		}
	}

	h.logger.Info("asset archived", "asset_id", id, "commit", hash, "layer", "L5")

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"state":   "archived",
		"message": "asset archived successfully",
	})
}

// RestoreAsset handles POST /api/v1/assets/{id}/restore.
//
//	@Summary Restore asset
//	@Description Restore an archived asset to active state
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param id path string true "Asset ID"
//	@Success 200 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/assets/{id}/restore [post]
func (h *AssetHandler) RestoreAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Update frontmatter state to active
	hash, err := h.fileManager.UpdateFrontmatter(ctx, id, func(fm *domain.FrontMatter) error {
		fm.State = "active"
		fm.UpdatedAt = time.Now()
		return nil
	}, fmt.Sprintf("Restore asset %s", id))
	if err != nil {
		h.writeError(w, http.StatusNotFound, "asset file not found: %s", err)
		return
	}

	// Update in-memory index so UI reflects change immediately
	fm, err := h.fileManager.GetFrontmatter(ctx, id)
	if err == nil {
		asset := service.Asset{
			ID:          fm.ID,
			Name:        fm.Name,
			Description: fm.Description,
			BizLine:     fm.BizLine,
			Tags:        fm.Tags,
			ContentHash: fm.ContentHash,
			State:       fm.State,
		}
		if err := h.indexer.Save(ctx, asset); err != nil {
			h.logger.Warn("failed to update index after restore", "asset_id", id, "error", err, "layer", "L5")
		}
	}

	h.logger.Info("asset restored", "asset_id", id, "commit", hash, "layer", "L5")

	h.writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"state":   "active",
		"message": "asset restored successfully",
	})
}

// DeleteAsset handles DELETE /api/v1/assets/{id}.
//
//	@Summary Delete asset
//	@Description Delete an asset by ID
//	@Tags assets
//	@Accept json
//	@Produce json
//	@Param id path string true "Asset ID"
//	@Success 200 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/assets/{id} [delete]
func (h *AssetHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Check if asset exists first
	if _, err := h.indexer.GetByID(ctx, id); err != nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", id)
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
