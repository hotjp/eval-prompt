// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/eval-prompt/internal/service"
)

// ImportStatusProvider provides a lightweight snapshot of the import queue.
type ImportStatusProvider interface {
	GetStatus() service.ImportStatus
}

// ReadyHandler handles readiness check endpoints.
type ReadyHandler struct {
	storage       StorageChecker
	llm           LLMChecker
	importChecker ImportStatusProvider
	logger        *slog.Logger

	// LLM check cache to avoid hammering the LLM provider on every readyz call.
	// Only applies to the actual LLM ping; model_config check is always cheap.
	llmCache struct {
		healthy   bool
		message   string
		expiresAt time.Time
	}
	llmCacheTTL time.Duration
}

// StorageChecker is an interface for checking database readiness.
type StorageChecker interface {
	Ping(ctx context.Context) error
}

// LLMChecker is an interface for checking LLM provider readiness.
type LLMChecker interface {
	Ping(ctx context.Context) error
	DefaultModel() string
}

// NewReadyHandler creates a new ReadyHandler.
func NewReadyHandler(storage StorageChecker, llm LLMChecker, importChecker ImportStatusProvider, logger *slog.Logger) *ReadyHandler {
	return &ReadyHandler{
		storage:       storage,
		llm:           llm,
		importChecker: importChecker,
		logger:        logger,
		llmCacheTTL:   30 * time.Minute,
	}
}

// CheckResponse represents the readiness check response.
type CheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ReadyResponse represents the full readiness response.
type ReadyResponse struct {
	Status  string                  `json:"status"`
	Checks  map[string]CheckResult   `json:"checks"`
}

// Readyz handles GET /readyz.
func (h *ReadyHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]CheckResult)
	overallStatus := "ok"
	httpStatus := http.StatusOK

	// Check 1: Database (must pass)
	if h.storage != nil {
		dbCtx, dbCancel := context.WithTimeout(ctx, 2*time.Second)
		defer dbCancel()
		if err := h.storage.Ping(dbCtx); err != nil {
			h.logger.Error("database check failed", "error", err, "layer", "L5")
			checks["database"] = CheckResult{
				Status:  "error",
				Message: err.Error(),
			}
			overallStatus = "error"
			httpStatus = http.StatusServiceUnavailable
		} else {
			checks["database"] = CheckResult{Status: "ok"}
		}
	} else {
		checks["database"] = CheckResult{Status: "ok"}
	}

	// Check 2: LLM Provider (optional, degraded on failure) with 30s cache
	if h.llm != nil {
		now := time.Now()
		if now.Before(h.llmCache.expiresAt) {
			// Use cached result
			if h.llmCache.healthy {
				checks["llm"] = CheckResult{Status: "ok"}
			} else {
				checks["llm"] = CheckResult{Status: "degraded", Message: h.llmCache.message}
				if overallStatus == "ok" {
					overallStatus = "degraded"
				}
			}
		} else {
			// Cache miss or expired: actually ping the LLM
			llmCtx, llmCancel := context.WithTimeout(ctx, 5*time.Second)
			err := h.llm.Ping(llmCtx)
			llmCancel()
			if err != nil {
				h.logger.Warn("LLM check failed, marking as degraded", "error", err, "layer", "L5")
				h.llmCache.healthy = false
				h.llmCache.message = err.Error()
				checks["llm"] = CheckResult{
					Status:  "degraded",
					Message: err.Error(),
				}
				if overallStatus == "ok" {
					overallStatus = "degraded"
				}
			} else {
				h.llmCache.healthy = true
				h.llmCache.message = ""
				checks["llm"] = CheckResult{Status: "ok"}
			}
			h.llmCache.expiresAt = now.Add(h.llmCacheTTL)
		}

		// Check 3: Model config (warn if default model not set)
		if h.llm.DefaultModel() == "" {
			checks["model_config"] = CheckResult{
				Status:  "degraded",
				Message: "default model not configured: set default_model in LLM config or set default=true on a provider",
			}
			if overallStatus == "ok" {
				overallStatus = "degraded"
			}
		} else {
			checks["model_config"] = CheckResult{
				Status:  "ok",
				Message: h.llm.DefaultModel(),
			}
		}
	}

	// Check 4: Import queue (informational — does not affect overall status)
	if h.importChecker != nil {
		st := h.importChecker.GetStatus()
		if st.Importing {
			checks["import"] = CheckResult{
				Status:  "importing",
				Message: fmt.Sprintf("import in progress (%d pending)", st.PendingCount),
			}
		} else if st.PendingCount > 0 {
			checks["import"] = CheckResult{
				Status:  "pending",
				Message: fmt.Sprintf("%d folder(s) waiting in .import/", st.PendingCount),
			}
		} else {
			checks["import"] = CheckResult{Status: "ok"}
		}
	}

	resp := ReadyResponse{
		Status: overallStatus,
		Checks: checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(resp)
}
