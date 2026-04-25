package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/eval-prompt/internal/config"
)

// TaxonomyHandler handles taxonomy (biz_line, tag) API endpoints.
type TaxonomyHandler struct {
	cfg        *config.TaxonomyConfig
	logger     *slog.Logger
	filePath   string
	mu         sync.RWMutex
}

// NewTaxonomyHandler creates a new TaxonomyHandler.
func NewTaxonomyHandler(cfg *config.TaxonomyConfig, logger *slog.Logger, filePath string) *TaxonomyHandler {
	return &TaxonomyHandler{
		cfg:      cfg,
		logger:   logger,
		filePath: filePath,
	}
}

func (h *TaxonomyHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *TaxonomyHandler) writeError(w http.ResponseWriter, status int, format string, args ...any) {
	h.logger.Error(fmt.Sprintf(format, args...), "layer", "L5")
	h.writeJSON(w, status, map[string]string{"error": fmt.Sprintf(format, args...)})
}

// GetTaxonomy returns the current taxonomy config.
// @Summary Get taxonomy
// @Description Get biz_lines and tags taxonomy
// @Tags taxonomy
// @Produce json
// @Success 200 {object} TaxonomyResponse
// @Router /api/v1/taxonomy [get]
func (h *TaxonomyHandler) GetTaxonomy(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	resp := TaxonomyResponse{
		BizLines: make([]BizLineResp, len(h.cfg.BizLines)),
		Tags:     make([]TagResp, len(h.cfg.Tags)),
	}
	for i, b := range h.cfg.BizLines {
		resp.BizLines[i] = BizLineResp{
			Name:        b.Name,
			Description: b.Description,
			Color:       b.Color,
			BuiltIn:     b.BuiltIn,
		}
	}
	for i, t := range h.cfg.Tags {
		resp.Tags[i] = TagResp{
			Name:    t.Name,
			Color:   t.Color,
			BuiltIn: t.BuiltIn,
		}
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// UpdateBizLines updates the biz_lines taxonomy.
// Built-in items cannot be modified via API; only user-defined items are saved.
// @Summary Update biz_lines
// @Description Replace user-defined biz_lines (built-in items are preserved)
// @Tags taxonomy
// @Accept json
// @Produce json
// @Param body body []BizLineResp true "biz_lines"
// @Success 200 {object} map[string]string
// @Router /api/v1/taxonomy/biz_lines [put]
func (h *TaxonomyHandler) UpdateBizLines(w http.ResponseWriter, r *http.Request) {
	var bizLines []BizLineResp
	if err := json.NewDecoder(r.Body).Decode(&bizLines); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Separate built-in and user-defined items from current config
	var currentBuiltIn []config.BizLineConfig
	var currentUserDefined []config.BizLineConfig
	for _, b := range h.cfg.BizLines {
		if b.BuiltIn {
			currentBuiltIn = append(currentBuiltIn, b)
		} else {
			currentUserDefined = append(currentUserDefined, b)
		}
	}

	// Build new user-defined list from request (skip built-in items)
	newUserDefined := make([]config.BizLineConfig, 0, len(bizLines))
	newNames := make(map[string]bool)
	for _, b := range bizLines {
		if b.BuiltIn {
			continue // Skip built-in items from request
		}
		newUserDefined = append(newUserDefined, config.BizLineConfig{
			Name:        b.Name,
			Description: b.Description,
			Color:       b.Color,
			BuiltIn:     false,
		})
		newNames[b.Name] = true
	}

	// Remove items that were deleted by user (not in request anymore)
	keptUserDefined := make([]config.BizLineConfig, 0)
	for _, u := range currentUserDefined {
		if newNames[u.Name] {
			keptUserDefined = append(keptUserDefined, u)
		}
	}

	// Merge: keep built-in + kept user-defined + new user-defined
	h.cfg.BizLines = append(currentBuiltIn, keptUserDefined...)
	for _, n := range newUserDefined {
		if _, exists := func() (config.BizLineConfig, bool) {
			for _, u := range keptUserDefined {
				if u.Name == n.Name {
					return u, true
				}
			}
			return config.BizLineConfig{}, false
		}(); !exists {
			h.cfg.BizLines = append(h.cfg.BizLines, n)
		}
	}

	// Save only user-defined items to file (built-in come from code)
	userOnlyConfig := &config.TaxonomyConfig{
		BizLines: make([]config.BizLineConfig, 0, len(h.cfg.BizLines)),
		Tags:     h.cfg.Tags,
	}
	for _, b := range h.cfg.BizLines {
		if !b.BuiltIn {
			userOnlyConfig.BizLines = append(userOnlyConfig.BizLines, b)
		}
	}

	if err := config.SaveTaxonomy(h.filePath, userOnlyConfig); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to save taxonomy: %v", err)
		return
	}

	h.logger.Info("taxonomy biz_lines updated", "count", len(bizLines), "layer", "L5")
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// UpdateTags updates the tags taxonomy.
// Built-in items cannot be modified via API; only user-defined items are saved.
// @Summary Update tags
// @Description Replace user-defined tags (built-in items are preserved)
// @Tags taxonomy
// @Accept json
// @Produce json
// @Param body body []TagResp true "tags"
// @Success 200 {object} map[string]string
// @Router /api/v1/taxonomy/tags [put]
func (h *TaxonomyHandler) UpdateTags(w http.ResponseWriter, r *http.Request) {
	var tags []TagResp
	if err := json.NewDecoder(r.Body).Decode(&tags); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Separate built-in and user-defined items from current config
	var currentBuiltIn []config.TagConfig
	var currentUserDefined []config.TagConfig
	for _, t := range h.cfg.Tags {
		if t.BuiltIn {
			currentBuiltIn = append(currentBuiltIn, t)
		} else {
			currentUserDefined = append(currentUserDefined, t)
		}
	}

	// Build new user-defined list from request (skip built-in items)
	newUserDefined := make([]config.TagConfig, 0, len(tags))
	newNames := make(map[string]bool)
	for _, t := range tags {
		if t.BuiltIn {
			continue // Skip built-in items from request
		}
		newUserDefined = append(newUserDefined, config.TagConfig{
			Name:    t.Name,
			Color:   t.Color,
			BuiltIn: false,
		})
		newNames[t.Name] = true
	}

	// Remove items that were deleted by user (not in request anymore)
	keptUserDefined := make([]config.TagConfig, 0)
	for _, u := range currentUserDefined {
		if newNames[u.Name] {
			keptUserDefined = append(keptUserDefined, u)
		}
	}

	// Merge: keep built-in + kept user-defined + new user-defined
	h.cfg.Tags = append(currentBuiltIn, keptUserDefined...)
	for _, n := range newUserDefined {
		if _, exists := func() (config.TagConfig, bool) {
			for _, u := range keptUserDefined {
				if u.Name == n.Name {
					return u, true
				}
			}
			return config.TagConfig{}, false
		}(); !exists {
			h.cfg.Tags = append(h.cfg.Tags, n)
		}
	}

	// Save only user-defined items to file (built-in come from code)
	userOnlyConfig := &config.TaxonomyConfig{
		BizLines: h.cfg.BizLines,
		Tags:     make([]config.TagConfig, 0),
	}
	for _, t := range h.cfg.Tags {
		if !t.BuiltIn {
			userOnlyConfig.Tags = append(userOnlyConfig.Tags, t)
		}
	}

	if err := config.SaveTaxonomy(h.filePath, userOnlyConfig); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to save taxonomy: %v", err)
		return
	}

	h.logger.Info("taxonomy tags updated", "count", len(tags), "layer", "L5")
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// TaxonomyResponse is the API response for taxonomy.
type TaxonomyResponse struct {
	BizLines []BizLineResp `json:"biz_lines"`
	Tags     []TagResp     `json:"tags"`
}

// BizLineResp represents a biz_line in API responses.
type BizLineResp struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color"`
	BuiltIn     bool   `json:"built_in"`
}

// TagResp represents a tag in API responses.
type TagResp struct {
	Name    string `json:"name"`
	Color   string `json:"color"`
	BuiltIn bool   `json:"built_in"`
}
