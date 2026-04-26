// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/plugins/llm"
)

// EvalHandler handles evaluation API endpoints.
type EvalHandler struct {
	evalService       service.EvalServiceer
	indexer           service.AssetIndexer
	logger            *slog.Logger
	semanticAnalyzer  service.SemanticAnalyzer
	llmInvoker       llm.Interface
	defaultModel     string
}

// NewEvalHandler creates a new EvalHandler.
func NewEvalHandler(evalService service.EvalServiceer, indexer service.AssetIndexer, logger *slog.Logger) *EvalHandler {
	return &EvalHandler{
		evalService: evalService,
		indexer:     indexer,
		logger:      logger,
	}
}

// SetSemanticAnalyzer sets the semantic analyzer for diff operations.
func (h *EvalHandler) SetSemanticAnalyzer(sa service.SemanticAnalyzer) {
	h.semanticAnalyzer = sa
}

// SetLLMInvoker sets the LLM invoker and default model for rewrite operations.
func (h *EvalHandler) SetLLMInvoker(invoker llm.Interface, defaultModel string) {
	h.llmInvoker = invoker
	h.defaultModel = defaultModel
}

// RunEvalRequest represents the request body for running an eval.
type RunEvalRequest struct {
	AssetID         string   `json:"asset_id"`
	SnapshotVersion string   `json:"snapshot_version"`
	EvalCaseIDs     []string `json:"eval_case_ids,omitempty"`
	Mode            string   `json:"mode,omitempty"`            // single, batch, matrix
	RunsPerCase     int      `json:"runs_per_case,omitempty"`  // for matrix mode
	Concurrency     int      `json:"concurrency,omitempty"`     // number of workers
	Model           string   `json:"model,omitempty"`           // override model
	Temperature     float64  `json:"temperature,omitempty"`     // override temperature
}

// RunEvalResponse represents the response for running an eval.
type RunEvalResponse struct {
	ExecutionID string `json:"execution_id"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
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

	// Convert handler request to service request
	svcReq := &service.RunEvalRequest{
		AssetID:         req.AssetID,
		SnapshotVersion: req.SnapshotVersion,
		EvalCaseIDs:     req.EvalCaseIDs,
		Mode:            domain.ExecutionMode(req.Mode),
		RunsPerCase:     req.RunsPerCase,
		Concurrency:     req.Concurrency,
		Model:           req.Model,
		Temperature:     req.Temperature,
	}

	execution, err := h.evalService.RunEval(ctx, svcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "eval run failed: %v", err)
		return
	}

	// Refresh the in-memory index so the new eval results are visible immediately
	if _, err := h.indexer.Reconcile(ctx); err != nil {
		h.logger.Warn("failed to reconcile index after eval", "asset_id", req.AssetID, "error", err)
		// Non-fatal: eval still succeeded, index will be stale
	}

	h.logger.Info("eval execution started", "asset_id", req.AssetID, "execution_id", execution.ID, "layer", "L5")

	h.writeJSON(w, http.StatusAccepted, RunEvalResponse{
		ExecutionID: execution.ID,
		Status:     string(execution.Status),
		Message:    "eval execution started",
	})
}

// ExecuteEvalRequest represents the request body for executing an eval.
type ExecuteEvalRequest struct {
	AssetID     string   `json:"asset_id"`
	CaseIDs     []string `json:"case_ids,omitempty"`
	Mode        string   `json:"mode,omitempty"`
	RunsPerCase int      `json:"runs_per_case,omitempty"`
	Concurrency int      `json:"concurrency,omitempty"`
	Model       string   `json:"model,omitempty"`
	Temperature float64  `json:"temperature,omitempty"`
}

// ExecuteEvalResponse represents the response for executing an eval.
type ExecuteEvalResponse struct {
	ExecutionID string `json:"execution_id"`
	Status      string `json:"status"`
}

// ExecuteEval handles POST /api/v1/evals/execute.
func (h *EvalHandler) ExecuteEval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ExecuteEvalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.AssetID == "" {
		h.writeError(w, http.StatusBadRequest, "asset_id is required")
		return
	}

	// Convert handler request to service request
	svcReq := &service.RunEvalRequest{
		AssetID:     req.AssetID,
		EvalCaseIDs: req.CaseIDs,
		Mode:        domain.ExecutionMode(req.Mode),
		RunsPerCase: req.RunsPerCase,
		Concurrency: req.Concurrency,
		Model:       req.Model,
		Temperature: req.Temperature,
	}

	execution, err := h.evalService.RunEval(ctx, svcReq)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "eval execution failed: %v", err)
		return
	}

	h.logger.Info("eval execution started", "asset_id", req.AssetID, "execution_id", execution.ID, "layer", "L5")

	h.writeJSON(w, http.StatusAccepted, ExecuteEvalResponse{
		ExecutionID: execution.ID,
		Status:      string(execution.Status),
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

// GetExecution handles GET /api/v1/evals/executions/{id}.
//
//	@Summary Get eval execution by ID
//	@Description Get a single eval execution by its ID
//	@Tags evals
//	@Accept json
//	@Produce json
//	@Param id path string true "Eval Execution ID"
//	@Success 200 {object} interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 404 {object} map[string]interface{}
//	@Router /api/v1/evals/executions/{id} [get]
func (h *EvalHandler) GetExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	execution, err := h.evalService.GetExecution(ctx, id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "eval execution not found: %s", id)
		return
	}

	h.writeJSON(w, http.StatusOK, execution)
}

// CancelExecution handles POST /api/v1/evals/executions/{id}/cancel.
//
//	@Summary Cancel eval execution
//	@Description Cancel a running eval execution
//	@Tags evals
//	@Accept json
//	@Produce json
//	@Param id path string true "Eval Execution ID"
//	@Success 200 {object} map[string]interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/evals/executions/{id}/cancel [post]
func (h *EvalHandler) CancelExecution(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.evalService.CancelExecution(ctx, id); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to cancel eval execution: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"execution_id": id,
		"status":       "cancelled",
		"message":       "eval execution cancelled",
	})
}

// DiffEvalRequest represents the request body for diff evaluation.
type DiffEvalRequest struct {
	OldContent string `json:"old_content"`
	NewContent string `json:"new_content"`
	OldVersion string `json:"old_version"`
	NewVersion string `json:"new_version"`
}

// DiffEval handles POST /api/v1/eval/diff.
//
//	@Summary Explain diff between versions
//	@Description Use semantic analysis to explain the differences between two versions of content
//	@Tags evals
//	@Accept json
//	@Produce json
//	@Param request body DiffEvalRequest true "Diff request"
//	@Success 200 {object} interface{}
//	@Failure 400 {object} map[string]interface{}
//	@Failure 503 {object} map[string]interface{}
//	@Failure 500 {object} map[string]interface{}
//	@Router /api/v1/eval/diff [post]
func (h *EvalHandler) DiffEval(w http.ResponseWriter, r *http.Request) {
	if h.semanticAnalyzer == nil {
		h.writeError(w, http.StatusServiceUnavailable, "semantic analyzer not configured")
		return
	}

	var req DiffEvalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request: %v", err)
		return
	}

	result, err := h.semanticAnalyzer.ExplainDiff(r.Context(), service.ExplainDiffRequest{
		OldContent: req.OldContent,
		NewContent: req.NewContent,
		OldVersion: req.OldVersion,
		NewVersion: req.NewVersion,
	})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "explain diff failed: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, result)
}

// RewriteRequest represents a rewrite request.
type RewriteRequest struct {
	Content         string `json:"content"`
	Instruction     string `json:"instruction"`
	DisableThinking bool   `json:"disable_thinking"`
}

// Rewrite handles POST /api/v1/rewrite.
func (h *EvalHandler) Rewrite(w http.ResponseWriter, r *http.Request) {
	if h.llmInvoker == nil {
		h.writeError(w, http.StatusServiceUnavailable, "LLM not configured")
		return
	}
	if h.defaultModel == "" {
		h.writeError(w, http.StatusServiceUnavailable, "default model not configured: set default_model in LLM config or set default=true on a provider")
		return
	}

	var req RewriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request: %v", err)
		return
	}

	if req.Content == "" {
		h.writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	// Build rewrite prompt - instruct LLM to return ONLY the rewritten text
	prompt := fmt.Sprintf(`Rewrite the following text according to the instruction.
Do NOT include any thinking, reasoning, or <thinking> tags in your response.
Do NOT include any markdown formatting, code blocks, or explanations.
Output ONLY the rewritten text directly.

Instruction: %s

Original text:
%s

Rewritten text:`, req.Instruction, req.Content)

	resp, err := h.llmInvoker.InvokeWithOptions(r.Context(), prompt, h.defaultModel, 0.3, llm.InvokeOptions{DisableThinking: req.DisableThinking})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "rewrite failed: %v", err)
		return
	}

	// Clean the response: remove <think> tags, markdown code blocks, and trim
	rewritten := cleanRewriteResponse(resp.Content)

	h.writeJSON(w, http.StatusOK, map[string]string{"rewritten": rewritten})
}

// cleanRewriteResponse removes think tags, markdown formatting, and trims whitespace.
func cleanRewriteResponse(s string) string {
	// Remove think tags using simple string replacement
	// (Go regex has issues matching Chinese chars between think tags)
	s = strings.ReplaceAll(s, "<think>", "")
	s = strings.ReplaceAll(s, "</think>", "")
	s = strings.ReplaceAll(s, "</think>", "")


	// Remove markdown code block markers
	s = strings.ReplaceAll(s, "```", "")

	// Remove bold/italic markers
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "_", "")

	// Remove headers but preserve content
	s = strings.ReplaceAll(s, "# ", "")
	s = strings.ReplaceAll(s, "## ", "")
	s = strings.ReplaceAll(s, "### ", "")

	// Remove leading dashes in markdown lists
	s = strings.ReplaceAll(s, "- ", "")

	// Trim whitespace but preserve newlines
	s = strings.TrimSpace(s)

	return s
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
