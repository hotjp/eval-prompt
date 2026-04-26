// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/plugins/llm"
	"github.com/eval-prompt/internal/service"
)

// LLMConfigHandler handles LLM provider configuration API endpoints.
type LLMConfigHandler struct {
	cfg            *[]config.LLMProviderConfig
	logger         *slog.Logger
	mainConfigPath string
	mu             sync.RWMutex
	llmChecker    **LLMCheckerAdapter
	configManager service.ConfigManager
}

// NewLLMConfigHandler creates a new LLMConfigHandler.
func NewLLMConfigHandler(cfg *[]config.LLMProviderConfig, logger *slog.Logger, filePath, mainConfigPath string, llmChecker **LLMCheckerAdapter, configManager service.ConfigManager) *LLMConfigHandler {
	return &LLMConfigHandler{
		cfg:            cfg,
		logger:         logger,
		mainConfigPath: mainConfigPath,
		llmChecker:     llmChecker,
		configManager:  configManager,
	}
}

func (h *LLMConfigHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *LLMConfigHandler) writeError(w http.ResponseWriter, status int, format string, args ...any) {
	h.logger.Error(fmt.Sprintf(format, args...), "layer", "L5")
	h.writeJSON(w, status, map[string]string{"error": fmt.Sprintf(format, args...)})
}

// GetLLMConfig returns the current LLM provider configs.
func (h *LLMConfigHandler) GetLLMConfig(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	resp := make([]LLMConfigResp, len(*h.cfg))
	for i, c := range *h.cfg {
		resp[i] = LLMConfigResp{
			Name:         c.Name,
			Provider:     c.Provider,
			APIKey:       maskAPIKey(c.APIKey),
			Endpoint:     c.Endpoint,
			DefaultModel: c.DefaultModel,
		}
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// UpdateLLMConfig updates the LLM provider configs.
func (h *LLMConfigHandler) UpdateLLMConfig(w http.ResponseWriter, r *http.Request) {
	var configs []LLMConfigReq
	if err := json.NewDecoder(r.Body).Decode(&configs); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	nameSeen := make(map[string]bool)
	for _, c := range configs {
		if c.Name == "" {
			h.writeError(w, http.StatusBadRequest, "name is required for each LLM provider")
			return
		}
		if nameSeen[c.Name] {
			h.writeError(w, http.StatusBadRequest, "duplicate provider name: %s", c.Name)
			return
		}
		nameSeen[c.Name] = true
	}

	*h.cfg = make([]config.LLMProviderConfig, len(configs))
	for i, c := range configs {
		(*h.cfg)[i] = config.LLMProviderConfig{
			Name:         c.Name,
			Provider:     c.Provider,
			APIKey:       c.APIKey,
			Endpoint:     c.Endpoint,
			DefaultModel: c.DefaultModel,
			Default:      c.Default,
		}
	}

	if err := config.SaveLLMConfigToMain(h.mainConfigPath, *h.cfg); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to save LLM config: %v", err)
		return
	}

	if h.configManager != nil {
		h.configManager.Notify(r.Context(), "llm", []string{"providers"})
	}

	h.logger.Info("LLM config updated", "count", len(configs), "layer", "L5")
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// TestByName tests an LLM provider by name with a test message.
func (h *LLMConfigHandler) TestByName(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Find the config by name
	h.mu.RLock()
	var providerCfg *config.LLMProviderConfig
	for i := range *h.cfg {
		if (*h.cfg)[i].Name == req.Name {
			providerCfg = &(*h.cfg)[i]
			break
		}
	}
	h.mu.RUnlock()

	if providerCfg == nil {
		h.writeError(w, http.StatusNotFound, "provider not found: %s", req.Name)
		return
	}

	// Create a provider for this config
	provider, err := llm.NewProvider(*providerCfg)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to create provider: %v", err)
		return
	}

	// Use a default test message if none provided
	testMessage := req.Message
	if testMessage == "" {
		testMessage = "Hello, please respond with 'OK' if you can read this message."
	}

	// Invoke the provider
	ctx := r.Context()
	resp, err := provider.Invoke(ctx, testMessage, providerCfg.DefaultModel, 0.3)
	if err != nil {
		h.writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"content": resp.Content,
	})
}

// HandleLLMChange handles LLM configuration change notifications.
func (h *LLMConfigHandler) HandleLLMChange(ctx context.Context, domain string, changed []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	var defaultCfg *config.LLMProviderConfig
	for i := range *h.cfg {
		if (*h.cfg)[i].Default {
			defaultCfg = &(*h.cfg)[i]
			break
		}
	}
	if defaultCfg == nil && len(*h.cfg) > 0 {
		defaultCfg = &(*h.cfg)[0]
	}

	h.logger.Info("LLM config change handler triggered", "changed", changed, "default", defaultCfg.Name)
}

// maskAPIKey masks an API key for security.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// LLMConfigResp is the API response for LLM config.
type LLMConfigResp struct {
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	APIKey       string `json:"api_key"`
	Endpoint     string `json:"endpoint,omitempty"`
	DefaultModel string `json:"default_model"`
}

// LLMConfigReq is the API request for LLM config.
type LLMConfigReq struct {
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	APIKey       string `json:"api_key"`
	Endpoint     string `json:"endpoint,omitempty"`
	DefaultModel string `json:"default_model"`
	Default     bool   `json:"default"`
}
