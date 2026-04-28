// Package handlers contains HTTP handlers for the gateway layer.
//
// This file implements the /api/v1/evals/orchestrate endpoint which coordinates
// evaluation plugins across multiple test cases with parallelism, confidence intervals,
// baseline comparisons, and ELO rating updates.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/i18n"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/service/eval"
	"github.com/eval-prompt/internal/service/eval/plugins/bertscore"
	"github.com/eval-prompt/internal/service/eval/plugins/beliefrevision"
	"github.com/eval-prompt/internal/service/eval/plugins/constraint"
	"github.com/eval-prompt/internal/service/eval/plugins/factscore"
	"github.com/eval-prompt/internal/service/eval/plugins/geval"
	"github.com/eval-prompt/internal/service/eval/plugins/selfcheck"
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
	orchestrator      *eval.Orchestrator
	embedder         eval.Embedder
	pluginsRegistered bool
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

// SetEmbedder sets the embedder for BERTScore and other embedding-based evaluations.
func (h *EvalHandler) SetEmbedder(embedder eval.Embedder) {
	h.embedder = embedder
}

// llmAdapter adapts llm.Interface to eval.LLMInvoker and LLMInvokerForEmbed.
type llmAdapter struct {
	invoker llm.Interface
	model   string
}

func (a *llmAdapter) Invoke(ctx context.Context, prompt string, model string, temperature float64) (*eval.LLMResponse, error) {
	resp, err := a.invoker.Invoke(ctx, prompt, model, temperature)
	if err != nil {
		return nil, err
	}
	return &eval.LLMResponse{
		Content:    resp.Content,
		Model:     resp.Model,
		TokensIn:  resp.TokensIn,
		TokensOut: resp.TokensOut,
		StopReason: resp.StopReason,
	}, nil
}

func (a *llmAdapter) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	return a.invoker.Embed(ctx, texts)
}

// RegisterPlugins registers all eval plugins with the global registry.
// This must be called after SetLLMInvoker.
func (h *EvalHandler) RegisterPlugins() {
	if h.pluginsRegistered {
		return
	}
	if h.llmInvoker == nil {
		h.logger.Warn("cannot register plugins: LLM invoker not set")
		return
	}

	// Create adapter to adapt llm.Interface to eval.LLMInvoker
	adapter := &llmAdapter{invoker: h.llmInvoker, model: h.defaultModel}
	judge := eval.NewLLMJudge(adapter, h.defaultModel)

	// Create embedder from LLM invoker if it supports embeddings
	if h.embedder == nil {
		// Try to create embedder from LLM invoker
		if emb, ok := h.llmInvoker.(interface{ Embed(ctx context.Context, texts []string) ([][]float64, error) }); ok {
			h.embedder = eval.NewLLMEmbedder(emb, "openai", "text-embedding-3-small", 1536)
			h.logger.Info("created embedder from LLM invoker")
		}
	}

	// Register BERTScore plugin if embedder is available
	if h.embedder != nil {
		bertscorePlugin := bertscore.NewPlugin(h.embedder)
		eval.Register(bertscorePlugin)
		h.logger.Info("registered bertscore plugin")
	}

	// Register G-Eval plugin with temperature for sampling
	gevalJudge := eval.NewLLMJudgeWithTemp(adapter, h.defaultModel, 0.7)
	gevalPlugin := geval.NewPlugin(gevalJudge)
	eval.Register(gevalPlugin)
	h.logger.Info("registered geval plugin")

	// Register BeliefRevision plugin
	beliefPlugin := beliefrevision.NewPlugin(judge)
	eval.Register(beliefPlugin)
	h.logger.Info("registered beliefrevision plugin")

	// Register Constraint plugin
	constraintPlugin := constraint.NewPlugin(judge)
	eval.Register(constraintPlugin)
	h.logger.Info("registered constraint plugin")

	// Register FACTScore plugin
	factscorePlugin := factscore.NewPlugin(judge)
	eval.Register(factscorePlugin)
	h.logger.Info("registered factscore plugin")

	// Register SelfCheckGPT plugin
	selfcheckPlugin := selfcheck.NewPlugin(judge)
	eval.Register(selfcheckPlugin)
	h.logger.Info("registered selfcheck plugin")

	h.pluginsRegistered = true
}

// SetOrchestrator sets the evaluation orchestrator for multi-plugin orchestration.
func (h *EvalHandler) SetOrchestrator(orchestrator *eval.Orchestrator) {
	h.orchestrator = orchestrator
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
		Message:    i18n.T(i18n.MsgEvalExecutionStarted, nil),
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

// OrchestrateRequest represents the request body for orchestrating evals.
type OrchestrateRequest struct {
	AssetID          string   `json:"asset_id"`
	Plugins          []string `json:"plugins"`
	InjectionStrategy string   `json:"injection_strategy"`
	Parallelism      int      `json:"parallelism"`
	ConfidenceLevel  float64  `json:"confidence_level,omitempty"`
	BaselineID       string   `json:"baseline_id,omitempty"`
}

// OrchestrateResponse represents the response for orchestrating evals.
type OrchestrateResponse struct {
	OverallScore       float64                `json:"overall_score"`
	PluginResults      map[string]PluginResult `json:"plugin_results"`
	ConfidenceInterval *ConfidenceInterval    `json:"confidence_interval"`
	BaselineComparison *BaselineResult        `json:"baseline_comparison,omitempty"`
	ELOResult          *ELORatingResult       `json:"elo_result,omitempty"`
	Summary            string                 `json:"summary"`
}

// PluginResult represents a single plugin's execution result.
type PluginResult struct {
	PluginName        string           `json:"plugin_name"`
	Score             float64          `json:"score"`
	ConfidenceInterval *ConfidenceInterval `json:"confidence_interval,omitempty"`
	WorkItemResults   []WorkItemResult `json:"work_item_results"`
}

// WorkItemResult represents the result of a single work item.
type WorkItemResult struct {
	WorkItemID  string         `json:"work_item_id"`
	Score       float64        `json:"score"`
	Details     map[string]any `json:"details,omitempty"`
	DurationMs  int64          `json:"duration_ms"`
}

// ConfidenceInterval represents a statistical confidence interval.
type ConfidenceInterval struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

// BaselineResult contains the comparison against baseline.
type BaselineResult struct {
	BaselineID           string  `json:"baseline_id"`
	ScoreDelta           float64 `json:"score_delta"`
	EffectSize           float64 `json:"effect_size"`
	EffectInterpretation string  `json:"effect_interpretation"`
	TStat                float64 `json:"t_stat"`
	PValue               float64 `json:"p_value"`
	IsSignificant        bool    `json:"is_significant"`
}

// ELORatingResult contains ELO rating update result.
type ELORatingResult struct {
	NewRating       float64 `json:"new_rating"`
	PreviousRating  float64 `json:"previous_rating"`
	Outcome         float64 `json:"outcome"`
}

// Orchestrate handles POST /api/v1/evals/orchestrate.
//
//	@Summary Orchestrate multi-plugin evaluation
//	@Description Run multiple evaluation plugins with parallelism, confidence intervals, and ELO ratings
//	@Tags evals
//	@Accept json
//	@Produce json
//	@Param request body OrchestrateRequest true "Orchestration request"
//	@Success 200 {object} OrchestrateResponse
//	@Failure 400 {object} map[string]interface{}
//	@Failure 503 {object} map[string]interface{}
//	@Router /api/v1/evals/orchestrate [post]
func (h *EvalHandler) Orchestrate(w http.ResponseWriter, r *http.Request) {
	// Register plugins before orchestrator runs
	h.RegisterPlugins()

	ctx := r.Context()

	var req OrchestrateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body: %v", err)
		return
	}

	if req.AssetID == "" {
		h.writeError(w, http.StatusBadRequest, "asset_id is required")
		return
	}

	if len(req.Plugins) == 0 {
		h.writeError(w, http.StatusBadRequest, "at least one plugin is required")
		return
	}

	// Get asset and test cases from indexer
	asset, err := h.indexer.GetByID(ctx, req.AssetID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "asset not found: %s", req.AssetID)
		return
	}

	// Convert domain test cases to eval.TestCase
	testCases := make([]*eval.TestCase, 0, len(asset.TestCases))
	for _, tc := range asset.TestCases {
		var inputStr string
		if tc.Input != nil {
			inputStr = fmt.Sprintf("%v", tc.Input)
		}
		var expectedStr string
		if tc.Expected != nil {
			expectedStr = tc.Expected.Content
		}
		testCases = append(testCases, &eval.TestCase{
			ID:       tc.ID,
			Prompt:   tc.Name, // Name is used as the prompt text
			Input:    inputStr,
			Expected: expectedStr,
		})
	}

	// Build eval config
	parallelism := req.Parallelism
	if parallelism <= 0 {
		parallelism = 4
	}
	confidenceLevel := req.ConfidenceLevel
	if confidenceLevel <= 0 {
		confidenceLevel = 0.95
	}

	config := eval.EvalConfig{
		Plugins:           req.Plugins,
		InjectionStrategy: req.InjectionStrategy,
		Parallelism:       parallelism,
		StatsConfig: eval.StatsConfig{
			ConfidenceLevel:    confidenceLevel,
			BootstrapIterations: 1000,
			BaselineID:         req.BaselineID,
		},
	}

	// Create orchestrator with config
	orchestrator := eval.NewOrchestrator(nil, config, h.logger)

	// Run orchestration
	result, err := orchestrator.Run(ctx, req.AssetID, "", testCases, nil)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "orchestration failed: %v", err)
		return
	}

	// Convert to response format
	response := h.convertOrchestratorResult(result)

	h.logger.Info("orchestration completed", "asset_id", req.AssetID, "overall_score", result.OverallScore, "layer", "L5")

	h.writeJSON(w, http.StatusOK, response)
}

// convertOrchestratorResult converts eval.OrchestratorResult to OrchestrateResponse.
func (h *EvalHandler) convertOrchestratorResult(result *eval.OrchestratorResult) *OrchestrateResponse {
	if result == nil {
		return nil
	}

	pluginResults := make(map[string]PluginResult)
	for name, pr := range result.PluginResults {
		workItems := make([]WorkItemResult, 0, len(pr.WorkItemResults))
		for _, wi := range pr.WorkItemResults {
			workItems = append(workItems, WorkItemResult{
				WorkItemID: wi.WorkItemID,
				Score:      wi.Score,
				Details:    wi.Details,
				DurationMs: wi.DurationMs,
			})
		}

		var ci *ConfidenceInterval
		if pr.ConfidenceInterval != nil {
			ci = &ConfidenceInterval{
				Low:  pr.ConfidenceInterval.Low,
				High: pr.ConfidenceInterval.High,
			}
		}

		pluginResults[name] = PluginResult{
			PluginName:        pr.PluginName,
			Score:             pr.Score,
			ConfidenceInterval: ci,
			WorkItemResults:   workItems,
		}
	}

	var ci *ConfidenceInterval
	if result.ConfidenceInterval != nil {
		ci = &ConfidenceInterval{
			Low:  result.ConfidenceInterval.Low,
			High: result.ConfidenceInterval.High,
		}
	}

	var baseline *BaselineResult
	if result.BaselineComparison != nil {
		baseline = &BaselineResult{
			BaselineID:           result.BaselineComparison.BaselineID,
			ScoreDelta:           result.BaselineComparison.ScoreDelta,
			EffectSize:           result.BaselineComparison.EffectSize,
			EffectInterpretation: result.BaselineComparison.EffectInterpretation,
			TStat:                result.BaselineComparison.TStat,
			PValue:               result.BaselineComparison.PValue,
			IsSignificant:        result.BaselineComparison.IsSignificant,
		}
	}

	var elo *ELORatingResult
	if result.ELOResult != nil {
		elo = &ELORatingResult{
			NewRating:      result.ELOResult.NewRating,
			PreviousRating: result.ELOResult.PreviousRating,
			Outcome:        result.ELOResult.Outcome,
		}
	}

	return &OrchestrateResponse{
		OverallScore:       result.OverallScore,
		PluginResults:      pluginResults,
		ConfidenceInterval: ci,
		BaselineComparison: baseline,
		ELOResult:          elo,
		Summary:            result.Summary,
	}
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
		"message":       i18n.T(i18n.MsgEvalExecutionCancelled, nil),
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

// cleanRewriteResponse removes think tags and their content, and cleans up markdown formatting.
func cleanRewriteResponse(s string) string {
	// Remove <think>...</think> blocks (iteratively, in case of nested or consecutive)
	for {
		start := strings.Index(s, "<think>")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "</think>")
		if end == -1 {
			// Orphaned opening tag, remove from start to end of line
			s = s[:start]
			break
		}
		s = s[:start] + s[start+end+len("</think>"):]
	}

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

	return strings.TrimSpace(s)
}

// ChatRequest represents a chat request.
type ChatRequest struct {
	Prompt  string `json:"prompt"`
	Context string `json:"context"`
}

// Chat handles POST /api/v1/chat.
func (h *EvalHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if h.llmInvoker == nil {
		h.writeError(w, http.StatusServiceUnavailable, "LLM not configured")
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request: %v", err)
		return
	}

	if req.Prompt == "" {
		h.writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}

	fullPrompt := req.Prompt
	if req.Context != "" {
		fullPrompt = fmt.Sprintf("Here is the prompt content I am currently editing:\n\n```\n%s\n```\n\nPlease answer my question based on the above prompt content:\n\n%s", req.Context, req.Prompt)
	}

	resp, err := h.llmInvoker.Invoke(r.Context(), fullPrompt, h.defaultModel, 0.7)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "chat failed: %v", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"content": resp.Content})
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
