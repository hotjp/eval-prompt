// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/eval-prompt/internal/service"
)

// EvalHandler handles evaluation API endpoints.
type EvalHandler struct {
	evalService service.EvalServiceer
	logger      *slog.Logger
}

// NewEvalHandler creates a new EvalHandler.
func NewEvalHandler(evalService service.EvalServiceer, logger *slog.Logger) *EvalHandler {
	return &EvalHandler{
		evalService: evalService,
		logger:      logger,
	}
}

// RunEvalRequest represents the request body for running an eval.
type RunEvalRequest struct {
	AssetID         string   `json:"asset_id"`
	SnapshotVersion string   `json:"snapshot_version"`
	EvalCaseIDs     []string `json:"eval_case_ids,omitempty"`
}

// RunEvalResponse represents the response for running an eval.
type RunEvalResponse struct {
	RunID   string `json:"run_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// RunEval handles POST /api/v1/evals/run.
//
//	@Summary Run evaluation
//	@Description Start an eval run for an asset
//	@Tags evals
//	@Accept json
//	@Produce json
//	@Param request body RunEvalRequest true "Eval run request"
//	@Success 202 {object} RunEvalResponse
//	@Failure 400 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/evals/run [post]
func (h *EvalHandler) RunEval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RunEvalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.AssetID == "" {
		h.writeError(w, http.StatusBadRequest, "asset_id is required")
		return
	}

	if req.SnapshotVersion == "" {
		req.SnapshotVersion = "latest"
	}

	run, err := h.evalService.RunEval(ctx, req.AssetID, req.SnapshotVersion, req.EvalCaseIDs)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "eval run failed: %v", err)
		return
	}

	h.logger.Info("eval run started", "asset_id", req.AssetID, "run_id", run.ID, "layer", "L5")

	h.writeJSON(w, http.StatusAccepted, RunEvalResponse{
		RunID:   run.ID,
		Status:  string(run.Status),
		Message: "eval run started",
	})
}

// GetEvalRun handles GET /api/v1/evals/{id}.
//
//	@Summary Get eval run by ID
//	@Description Get a single eval run by its ID
//	@Tags evals
//	@Accept json
//	@Produce json
//	@Param id path string true "Eval Run ID"
//	@Success 200 {object} interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Router /api/v1/evals/{id} [get]
func (h *EvalHandler) GetEvalRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	run, err := h.evalService.GetEvalRun(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "eval run not found: %s", id)
		return
	}

	h.writeJSON(w, http.StatusOK, run)
}

// GetEvalReport handles GET /api/v1/evals/{id}/report.
//
//	@Summary Get eval report
//	@Description Get the report for an eval run
//	@Tags evals
//	@Accept json
//	@Produce json
//	@Param id path string true "Eval Run ID"
//	@Success 200 {object} interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/evals/{id}/report [get]
func (h *EvalHandler) GetEvalReport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	report, err := h.evalService.GenerateReport(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to generate report: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, report)
}

// CompareEvalRequest represents the request body for comparing evals.
type CompareEvalRequest struct {
	AssetID  string `json:"asset_id"`
	Version1 string `json:"version1"`
	Version2 string `json:"version2"`
}

// CompareEval handles POST /api/v1/evals/compare.
//
//	@Summary Compare eval versions
//	@Description Compare two versions of an eval
//	@Tags evals
//	@Accept json
//	@Produce json
//	@Param request body CompareEvalRequest true "Compare request"
//	@Success 200 {object} interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/evals/compare [post]
func (h *EvalHandler) CompareEval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CompareEvalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.AssetID == "" || req.Version1 == "" || req.Version2 == "" {
		h.writeError(w, http.StatusBadRequest, "asset_id, version1, and version2 are required")
		return
	}

	result, err := h.evalService.CompareEval(ctx, req.AssetID, req.Version1, req.Version2)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "compare failed: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// DiagnoseEval handles GET /api/v1/evals/{id}/diagnose.
//
//	@Summary Diagnose eval run
//	@Description Get diagnosis for an eval run
//	@Tags evals
//	@Accept json
//	@Produce json
//	@Param id path string true "Eval Run ID"
//	@Success 200 {object} interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/evals/{id}/diagnose [get]
func (h *EvalHandler) DiagnoseEval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	diagnosis, err := h.evalService.DiagnoseEval(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "diagnosis failed: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, diagnosis)
}

// ListEvalRuns handles GET /api/v1/evals.
//
//	@Summary List eval runs
//	@Description Get all eval runs for an asset
//	@Tags evals
//	@Accept json
//	@Produce json
//	@Param asset_id query string true "Asset ID"
//	@Success 200 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/evals [get]
func (h *EvalHandler) ListEvalRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	assetID := r.URL.Query().Get("asset_id")
	if assetID == "" {
		h.writeError(w, http.StatusBadRequest, "asset_id is required")
		return
	}

	runs, err := h.evalService.ListEvalRuns(ctx, assetID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to list eval runs: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"runs":  runs,
		"total": len(runs),
	})
}

// writeJSON writes a JSON response.
func (h *EvalHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (h *EvalHandler) writeError(w http.ResponseWriter, status int, format string, args ...any) {
	h.logger.Error(fmt.Sprintf(format, args...), "layer", "L5", "status", status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": fmt.Sprintf(format, args...),
	})
}
