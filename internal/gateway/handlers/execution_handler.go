// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/eval-prompt/internal/service"
)

// ExecutionHandler handles execution API endpoints.
type ExecutionHandler struct {
	executionStore *service.ExecutionFileStore
	logger         *slog.Logger
}

// NewExecutionHandler creates a new ExecutionHandler.
func NewExecutionHandler(executionStore *service.ExecutionFileStore, logger *slog.Logger) *ExecutionHandler {
	return &ExecutionHandler{
		executionStore: executionStore,
		logger:         logger,
	}
}

// ListExecutions handles GET /api/v1/executions.
//
//	@Summary List all executions
//	@Description Get all eval executions with pagination
//	@Tags executions
//	@Accept json
//	@Produce json
//	@Param offset query int false "Offset for pagination" default(0)
//	@Param limit query int false "Limit for pagination" default(50)
//	@Success 200 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/executions [get]
func (h *ExecutionHandler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	offset := 0
	limit := 50

	executions, total, err := h.executionStore.List(ctx, offset, limit)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to list executions: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"executions": executions,
		"total":      total,
		"offset":     offset,
		"limit":      limit,
	})
}

// GetExecution handles GET /api/v1/executions/{id}.
//
//	@Summary Get execution by ID
//	@Description Get a single execution by its ID
//	@Tags executions
//	@Accept json
//	@Produce json
//	@Param id path string true "Execution ID"
//	@Success 200 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Router /api/v1/executions/{id} [get]
func (h *ExecutionHandler) GetExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	execution, err := h.executionStore.Get(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "execution not found: %s", id)
		return
	}

	h.writeJSON(w, http.StatusOK, execution)
}

// writeJSON writes a JSON response.
func (h *ExecutionHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (h *ExecutionHandler) writeError(w http.ResponseWriter, status int, format string, args ...any) {
	h.logger.Error(fmt.Sprintf(format, args...), "layer", "L5", "status", status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": fmt.Sprintf(format, args...),
	})
}
