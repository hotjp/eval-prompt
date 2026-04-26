// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/eval-prompt/internal/service"
)

// CallHandler handles LLM call API endpoints.
type CallHandler struct {
	callStore *service.LLMCallFileStore
	logger    *slog.Logger
}

// NewCallHandler creates a new CallHandler.
func NewCallHandler(callStore *service.LLMCallFileStore, logger *slog.Logger) *CallHandler {
	return &CallHandler{
		callStore: callStore,
		logger:    logger,
	}
}

// ListCallsByExecution handles GET /api/v1/executions/{id}/calls.
//
//	@Summary List calls for an execution
//	@Description Get all LLM calls for a given execution
//	@Tags calls
//	@Accept json
//	@Produce json
//	@Param id path string true "Execution ID"
//	@Success 200 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/executions/{id}/calls [get]
func (h *CallHandler) ListCallsByExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	executionID := r.PathValue("id")
	if executionID == "" {
		h.writeError(w, http.StatusBadRequest, "execution id is required")
		return
	}

	calls, err := h.callStore.ListByExecution(ctx, executionID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to list calls: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"execution_id": executionID,
		"calls":        calls,
		"total":        len(calls),
	})
}

// writeJSON writes a JSON response.
func (h *CallHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (h *CallHandler) writeError(w http.ResponseWriter, status int, format string, args ...any) {
	h.logger.Error(fmt.Sprintf(format, args...), "layer", "L5", "status", status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": fmt.Sprintf(format, args...),
	})
}
