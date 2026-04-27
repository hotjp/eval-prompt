// Package gateway implements L5-Gateway layer: TLS termination, protocol adaptation,
// middleware, request routing, and static resource serving.
package gateway

import (
	"log/slog"
	"net/http"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/internal/gateway/handlers"
	"github.com/eval-prompt/internal/gateway/middleware"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/plugins/llm"

	_ "github.com/eval-prompt/docs" // swagger docs
	httpSwagger "github.com/swaggo/http-swagger"
)

// RouterConfig contains configuration for the router.
type RouterConfig struct {
	TriggerService    service.TriggerServicer
	EvalService       service.EvalServiceer
	IndexService      service.AssetIndexer
	FileManager       service.AssetFileManager
	Logger            *slog.Logger
	Metrics           *middleware.MetricsCollector
	CORSOrigins       []string
	AdminConfig       *config.Config
	RestartFunc       func() // Function to call for graceful restart
	StorageClient    handlers.StorageChecker
	LLMInvoker        *handlers.LLMCheckerAdapter
	LLMInterface     llm.Interface
	LLMDefaultModel  string
	ConfigManager     service.ConfigManager
	GitBridge         service.GitBridger
	SemanticAnalyzer  service.SemanticAnalyzer
	ExecutionStore    *service.ExecutionFileStore
	CallStore         *service.LLMCallFileStore
	// Pre-created handlers (optional — if provided, router uses them; otherwise creates its own)
	AdminHandler      *handlers.AdminHandler
	LLMConfigHandler  *handlers.LLMConfigHandler
	TaxonomyHandler   *handlers.TaxonomyHandler
	ImportHandler     *handlers.ImportHandler
}

// NewRouter creates a new HTTP router with all middleware and handlers registered.
func NewRouter(cfg RouterConfig) *http.ServeMux {
	mux := http.NewServeMux()
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	metrics := cfg.Metrics
	if metrics == nil {
		metrics = middleware.NewMetricsCollector()
	}

	// Create handler instances — use pre-created handlers if provided
	mcpHandler := handlers.NewMCPHandler(cfg.TriggerService, cfg.IndexService, logger)
	assetHandler := handlers.NewAssetHandler(cfg.IndexService, cfg.FileManager, logger, cfg.AdminConfig)
	if cfg.SemanticAnalyzer != nil {
		assetHandler = assetHandler.WithSemanticAnalyzer(cfg.SemanticAnalyzer, "")
	}
	if cfg.GitBridge != nil {
		assetHandler = assetHandler.WithGitBridge(cfg.GitBridge)
	}
	evalHandler := handlers.NewEvalHandler(cfg.EvalService, cfg.IndexService, logger)
	if cfg.SemanticAnalyzer != nil {
		evalHandler.SetSemanticAnalyzer(cfg.SemanticAnalyzer)
	}
	if cfg.LLMInterface != nil {
		evalHandler.SetLLMInvoker(cfg.LLMInterface, cfg.LLMDefaultModel)
	}
	triggerHandler := handlers.NewTriggerHandler(cfg.TriggerService, logger)
	readyHandler := handlers.NewReadyHandler(cfg.StorageClient, cfg.LLMInvoker, logger)

	adminHandler := cfg.AdminHandler
	if adminHandler == nil {
		adminHandler = handlers.NewAdminHandler(logger, cfg.AdminConfig, cfg.RestartFunc, cfg.IndexService, cfg.GitBridge, cfg.ConfigManager)
	}
	llmConfigHandler := cfg.LLMConfigHandler
	if llmConfigHandler == nil {
		llmConfigHandler = handlers.NewLLMConfigHandler(nil, logger, "", "", nil, cfg.ConfigManager)
	}
	taxonomyHandler := cfg.TaxonomyHandler
	if taxonomyHandler == nil {
		taxonomyHandler = handlers.NewTaxonomyHandler(nil, logger, "", cfg.ConfigManager)
	}
	executionHandler := handlers.NewExecutionHandler(cfg.ExecutionStore, logger)
	callHandler := handlers.NewCallHandler(cfg.CallStore, logger)
	importHandler := cfg.ImportHandler
	if importHandler == nil {
		importHandler = handlers.NewImportHandler(nil, logger)
	}

	// Build middleware chain
	chain := func(h http.Handler) http.Handler {
		h = middleware.Recover(logger)(h)
		h = middleware.RequestID()(h)
		h = middleware.Metrics(metrics)(h)
		h = middleware.Logging(logger)(h)
		h = middleware.CORS(cfg.CORSOrigins)(h)
		return h
	}

	// Register routes
	mux.Handle("GET /mcp/v1/sse", chain(http.HandlerFunc(mcpHandler.HandleSSE)))
	mux.Handle("POST /mcp/v1", chain(http.HandlerFunc(mcpHandler.HandlePOST)))

	// Asset API routes
	mux.HandleFunc("GET /api/v1/assets", assetHandler.ListAssets)
	mux.HandleFunc("POST /api/v1/assets", assetHandler.CreateAsset)
	mux.HandleFunc("GET /api/v1/assets/{id}", assetHandler.GetAsset)
	mux.HandleFunc("PUT /api/v1/assets/{id}", assetHandler.UpdateAsset)
	mux.HandleFunc("DELETE /api/v1/assets/{id}", assetHandler.DeleteAsset)
	mux.HandleFunc("GET /api/v1/assets/{id}/content", assetHandler.GetAssetContent)
	mux.HandleFunc("PUT /api/v1/assets/{id}/content", assetHandler.SaveAssetContent)
	mux.HandleFunc("GET /api/v1/assets/{id}/files", assetHandler.GetAssetFiles)
	mux.HandleFunc("POST /api/v1/assets/{id}/archive", assetHandler.ArchiveAsset)
	mux.HandleFunc("POST /api/v1/assets/{id}/restore", assetHandler.RestoreAsset)
	mux.HandleFunc("POST /api/v1/assets/{id}/commit", assetHandler.CommitAsset)
	mux.HandleFunc("POST /api/v1/assets/commit", assetHandler.CommitBatchAssets)
	mux.HandleFunc("POST /api/v1/assets/batch/tag", assetHandler.BatchTagAssets)
	mux.HandleFunc("GET /api/v1/assets/{id}/history", assetHandler.AssetHistory)
	mux.HandleFunc("GET /api/v1/assets/{id}/diff", assetHandler.AssetDiff)

	// Eval API routes
	mux.HandleFunc("GET /api/v1/evals", evalHandler.ListEvalRuns)
	mux.HandleFunc("POST /api/v1/evals/run", evalHandler.RunEval)
	mux.HandleFunc("POST /api/v1/evals/execute", evalHandler.ExecuteEval)
	mux.HandleFunc("GET /api/v1/evals/{id}", evalHandler.GetEvalRun)
	mux.HandleFunc("GET /api/v1/evals/{id}/diagnose", evalHandler.DiagnoseEval)
	mux.HandleFunc("GET /api/v1/evals/{id}/report", evalHandler.GetEvalReport)
	mux.HandleFunc("POST /api/v1/evals/compare", evalHandler.CompareEval)
	mux.HandleFunc("POST /api/v1/eval/diff", evalHandler.DiffEval)

	// Execution API routes (called by frontend)
	mux.HandleFunc("GET /api/v1/executions", executionHandler.ListExecutions)
	mux.HandleFunc("GET /api/v1/executions/{id}", executionHandler.GetExecution)
	mux.HandleFunc("POST /api/v1/executions/{id}/cancel", evalHandler.CancelExecution)
	mux.HandleFunc("GET /api/v1/executions/{id}/calls", callHandler.ListCallsByExecution)

	// Rewrite API
	mux.HandleFunc("POST /api/v1/rewrite", evalHandler.Rewrite)

	// Chat API
	mux.HandleFunc("POST /api/v1/chat", evalHandler.Chat)

	// Import SSE route
	mux.HandleFunc("GET /api/v1/import/events", importHandler.HandleSSE)

	// Trigger API routes
	mux.HandleFunc("POST /api/v1/trigger/match", triggerHandler.MatchTrigger)
	mux.HandleFunc("POST /api/v1/trigger/validate", triggerHandler.ValidateAntiPatterns)
	mux.HandleFunc("POST /api/v1/trigger/inject", triggerHandler.InjectVariables)
	mux.HandleFunc("GET /api/v1/trigger/anti-patterns", triggerHandler.GetAntiPatterns)

	// Taxonomy API routes
	mux.HandleFunc("GET /api/v1/taxonomy", taxonomyHandler.GetTaxonomy)
	mux.HandleFunc("PUT /api/v1/taxonomy/asset_types", taxonomyHandler.UpdateAssetTypes)
	mux.HandleFunc("PUT /api/v1/taxonomy/tags", taxonomyHandler.UpdateTags)

	// LLM Config API routes
	mux.HandleFunc("GET /api/v1/llm-config", llmConfigHandler.GetLLMConfig)
	mux.HandleFunc("PUT /api/v1/llm-config", llmConfigHandler.UpdateLLMConfig)
	mux.HandleFunc("POST /api/v1/llm-config/test-by-name", llmConfigHandler.TestByName)

	// Admin API routes
	mux.HandleFunc("GET /api/v1/admin/status", adminHandler.GetStatus)
	mux.HandleFunc("GET /api/v1/admin/git-info", adminHandler.GetGitInfo)
	mux.HandleFunc("GET /api/v1/admin/repo-config", adminHandler.GetRepoConfig)
	mux.HandleFunc("PUT /api/v1/admin/repo-config", adminHandler.UpdateRepoConfig)
	mux.HandleFunc("GET /api/v1/admin/repo-list", adminHandler.GetRepoList)
	mux.HandleFunc("PUT /api/v1/admin/repo-switch", adminHandler.PutRepoSwitch)
	mux.HandleFunc("GET /api/v1/admin/first-use", adminHandler.GetFirstUse)
	mux.HandleFunc("GET /api/v1/admin/repo-status", adminHandler.GetRepoStatus)
	mux.HandleFunc("POST /api/v1/admin/reload", adminHandler.ReloadConfig)
	mux.HandleFunc("POST /api/v1/admin/restart", adminHandler.Restart)
	mux.HandleFunc("POST /api/v1/admin/reconcile", adminHandler.Reconcile)
	mux.HandleFunc("POST /api/v1/admin/git-pull", adminHandler.GitPull)
	mux.HandleFunc("POST /api/v1/admin/open-folder", adminHandler.OpenFolder)
	mux.HandleFunc("PUT /api/v1/admin/config", adminHandler.SaveConfig)

	// Health check endpoints
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("GET /readyz", readyHandler.Readyz)

	// Static files (SPA) - must be last to not override API routes
	RegisterStaticRoutes(mux)

	// Swagger UI
	mux.Handle("/swagger/", httpSwagger.Handler())

	return mux
}
