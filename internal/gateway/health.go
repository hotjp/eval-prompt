// Package gateway provides HTTP handlers and routing.
package gateway

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/eval-prompt/plugins/llm"
)

// HealthHandler provides health check endpoints.
type HealthHandler struct {
	db       *sql.DB
	llmProv  llm.Provider
	checkDB  bool
	checkLLM bool
}

// NewHealthHandler creates a new health handler.
func NewHealthHandler(db *sql.DB, llmProv llm.Provider) *HealthHandler {
	return &HealthHandler{
		db:       db,
		llmProv:  llmProv,
		checkDB:  db != nil,
		checkLLM: llmProv != nil,
	}
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string           `json:"status"`
	Timestamp string           `json:"timestamp"`
	Checks    map[string]Check `json:"checks,omitempty"`
}

// Check represents a single health check.
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Healthz handles GET /healthz - liveness probe.
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// readyz handles GET /readyz - readiness probe.
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
		Checks:    make(map[string]Check),
	}

	allOk := true

	// Check database
	if h.checkDB {
		if h.db.PingContext(r.Context()) == nil {
			response.Checks["database"] = Check{Status: "ok"}
		} else {
			response.Checks["database"] = Check{Status: "error", Message: "ping failed"}
			allOk = false
		}
	}

	// Check LLM provider
	if h.checkLLM && h.llmProv != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Simple check - try to call the provider
		_, err := h.llmProv.Invoke(ctx, "ping", "gpt-4o-mini", 0.0)
		if err != nil {
			response.Checks["llm"] = Check{Status: "degraded", Message: err.Error()}
			// LLM failure is degraded, not fatal
		} else {
			response.Checks["llm"] = Check{Status: "ok"}
		}
	}

	if !allOk {
		response.Status = "error"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterHealthRoutes registers health check routes.
func RegisterHealthRoutes(mux *http.ServeMux, db *sql.DB, llmProv llm.Provider) {
	h := NewHealthHandler(db, llmProv)
	mux.HandleFunc("/healthz", h.Healthz)
	mux.HandleFunc("/readyz", h.Readyz)
}

// NoopHealthHandler is a noop health handler.
type NoopHealthHandler struct{}

// Healthz implements HealthHandler.
func (n *NoopHealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// Readyz implements HealthHandler.
func (n *NoopHealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
