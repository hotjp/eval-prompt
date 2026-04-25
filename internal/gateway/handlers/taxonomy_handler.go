package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/internal/service"
)

// TaxonomyHandler handles taxonomy (asset_type, tag) API endpoints.
type TaxonomyHandler struct {
	cfg           *config.TaxonomyConfig
	logger        *slog.Logger
	filePath      string
	mu            sync.RWMutex
	configManager service.ConfigManager
}

// NewTaxonomyHandler creates a new TaxonomyHandler.
func NewTaxonomyHandler(cfg *config.TaxonomyConfig, logger *slog.Logger, filePath string, configManager service.ConfigManager) *TaxonomyHandler {
	return &TaxonomyHandler{
		cfg:           cfg,
		logger:        logger,
		filePath:      filePath,
		configManager: configManager,
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
func (h *TaxonomyHandler) GetTaxonomy(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	resp := TaxonomyResponse{
		AssetTypes: make([]AssetTypeResp, len(h.cfg.AssetTypes)),
		Tags:     make([]TagResp, len(h.cfg.Tags)),
	}
	for i, b := range h.cfg.AssetTypes {
		resp.AssetTypes[i] = AssetTypeResp{
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

// UpdateAssetTypes updates the asset_types taxonomy.
func (h *TaxonomyHandler) UpdateAssetTypes(w http.ResponseWriter, r *http.Request) {
	var bizLines []AssetTypeResp
	if err := json.NewDecoder(r.Body).Decode(&bizLines); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	var currentBuiltIn []config.AssetTypeConfig
	var currentUserDefined []config.AssetTypeConfig
	for _, b := range h.cfg.AssetTypes {
		if b.BuiltIn {
			currentBuiltIn = append(currentBuiltIn, b)
		} else {
			currentUserDefined = append(currentUserDefined, b)
		}
	}

	newUserDefined := make([]config.AssetTypeConfig, 0, len(bizLines))
	newNames := make(map[string]bool)
	for _, b := range bizLines {
		if b.BuiltIn {
			continue
		}
		newUserDefined = append(newUserDefined, config.AssetTypeConfig{
			Name:        b.Name,
			Description: b.Description,
			Color:       b.Color,
			BuiltIn:     false,
		})
		newNames[b.Name] = true
	}

	keptUserDefined := make([]config.AssetTypeConfig, 0)
	for _, u := range currentUserDefined {
		if newNames[u.Name] {
			keptUserDefined = append(keptUserDefined, u)
		}
	}

	h.cfg.AssetTypes = append(currentBuiltIn, keptUserDefined...)
	for _, n := range newUserDefined {
		if _, exists := func() (config.AssetTypeConfig, bool) {
			for _, u := range keptUserDefined {
				if u.Name == n.Name {
					return u, true
				}
			}
			return config.AssetTypeConfig{}, false
		}(); !exists {
			h.cfg.AssetTypes = append(h.cfg.AssetTypes, n)
		}
	}

	userOnlyConfig := &config.TaxonomyConfig{
		AssetTypes: make([]config.AssetTypeConfig, 0, len(h.cfg.AssetTypes)),
		Tags:     h.cfg.Tags,
	}
	for _, b := range h.cfg.AssetTypes {
		if !b.BuiltIn {
			userOnlyConfig.AssetTypes = append(userOnlyConfig.AssetTypes, b)
		}
	}

	if err := config.SaveTaxonomy(h.filePath, userOnlyConfig); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to save taxonomy: %v", err)
		return
	}

	if h.configManager != nil {
		h.configManager.Notify(r.Context(), "taxonomy", []string{"asset_types"})
	}

	h.logger.Info("taxonomy asset_types updated", "count", len(bizLines), "layer", "L5")
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// UpdateTags updates the tags taxonomy.
func (h *TaxonomyHandler) UpdateTags(w http.ResponseWriter, r *http.Request) {
	var tags []TagResp
	if err := json.NewDecoder(r.Body).Decode(&tags); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	var currentBuiltIn []config.TagConfig
	var currentUserDefined []config.TagConfig
	for _, t := range h.cfg.Tags {
		if t.BuiltIn {
			currentBuiltIn = append(currentBuiltIn, t)
		} else {
			currentUserDefined = append(currentUserDefined, t)
		}
	}

	newUserDefined := make([]config.TagConfig, 0, len(tags))
	newNames := make(map[string]bool)
	for _, t := range tags {
		if t.BuiltIn {
			continue
		}
		newUserDefined = append(newUserDefined, config.TagConfig{
			Name:    t.Name,
			Color:   t.Color,
			BuiltIn: false,
		})
		newNames[t.Name] = true
	}

	keptUserDefined := make([]config.TagConfig, 0)
	for _, u := range currentUserDefined {
		if newNames[u.Name] {
			keptUserDefined = append(keptUserDefined, u)
		}
	}

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

	userOnlyConfig := &config.TaxonomyConfig{
		AssetTypes: h.cfg.AssetTypes,
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

	if h.configManager != nil {
		h.configManager.Notify(r.Context(), "taxonomy", []string{"tags"})
	}

	h.logger.Info("taxonomy tags updated", "count", len(tags), "layer", "L5")
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// TaxonomyResponse is the API response for taxonomy.
type TaxonomyResponse struct {
	AssetTypes []AssetTypeResp `json:"asset_types"`
	Tags     []TagResp     `json:"tags"`
}

// AssetTypeResp represents a asset_type in API responses.
type AssetTypeResp struct {
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

// HandleTaxonomyChange is the config change handler for the taxonomy domain.
// Currently a no-op since no downstream components need to react to taxonomy changes.
func (h *TaxonomyHandler) HandleTaxonomyChange(ctx context.Context, domain string, changed []string) {
	h.logger.Info("taxonomy config change received", "domain", domain, "changed", changed)
}
