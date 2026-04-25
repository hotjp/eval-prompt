// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// ReadyHandler handles readiness check endpoints.
type ReadyHandler struct {
	storage   StorageChecker
	llm       LLMChecker
	logger    *slog.Logger
}

// StorageChecker is an interface for checking database readiness.
type StorageChecker interface {
	Ping(ctx context.Context) error
}

// LLMChecker is an interface for checking LLM provider readiness.
type LLMChecker interface {
	Ping(ctx context.Context) error
}

// NewReadyHandler creates a new ReadyHandler.
func NewReadyHandler(storage StorageChecker, llm LLMChecker, logger *slog.Logger) *ReadyHandler {
	return &ReadyHandler{
		storage: storage,
		llm:     llm,
		logger:  logger,
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

	// Check 2: LLM Provider (optional, degraded on failure)
	if h.llm != nil {
		llmCtx, llmCancel := context.WithTimeout(ctx, 5*time.Second)
		defer llmCancel()
		if err := h.llm.Ping(llmCtx); err != nil {
			h.logger.Warn("LLM check failed, marking as degraded", "error", err, "layer", "L5")
			checks["llm"] = CheckResult{
				Status:  "degraded",
				Message: err.Error(),
			}
			if overallStatus == "ok" {
				overallStatus = "degraded"
			}
		} else {
			checks["llm"] = CheckResult{Status: "ok"}
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
