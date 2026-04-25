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

	_ "github.com/eval-prompt/docs" // swagger docs
	httpSwagger "github.com/swaggo/http-swagger"
)

// RouterConfig contains configuration for the router.
type RouterConfig struct {
	TriggerService   service.TriggerServicer
	EvalService      service.EvalServiceer
	IndexService     service.AssetIndexer
	FileManager      service.AssetFileManager
	Logger          *slog.Logger
	Metrics         *middleware.MetricsCollector
	CORSOrigins     []string
	TaxonomyConfig  *config.TaxonomyConfig
	TaxonomyFilePath string
	AdminConfig     *config.Config
	RestartFunc     func() // Function to call for graceful restart
	StorageClient   handlers.StorageChecker
	LLMInvoker      handlers.LLMChecker
	LLMConfig       *[]config.LLMProviderConfig
	LLMConfigPath   string
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

	// Create handler instances
	mcpHandler := handlers.NewMCPHandler(cfg.TriggerService, cfg.EvalService, cfg.IndexService, logger)
	assetHandler := handlers.NewAssetHandler(cfg.IndexService, cfg.FileManager, logger)
	evalHandler := handlers.NewEvalHandler(cfg.EvalService, cfg.IndexService, logger)
	triggerHandler := handlers.NewTriggerHandler(cfg.TriggerService, logger)
	taxonomyHandler := handlers.NewTaxonomyHandler(cfg.TaxonomyConfig, logger, cfg.TaxonomyFilePath)
	adminHandler := handlers.NewAdminHandler(logger, cfg.AdminConfig, cfg.RestartFunc)
	readyHandler := handlers.NewReadyHandler(cfg.StorageClient, cfg.LLMInvoker, logger)
	llmConfigHandler := handlers.NewLLMConfigHandler(cfg.LLMConfig, logger, cfg.LLMConfigPath)

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
	mux.HandleFunc("POST /api/v1/assets/{id}/archive", assetHandler.ArchiveAsset)
	mux.HandleFunc("POST /api/v1/assets/{id}/restore", assetHandler.RestoreAsset)

	// Eval API routes
	mux.HandleFunc("GET /api/v1/evals", evalHandler.ListEvalRuns)
	mux.HandleFunc("POST /api/v1/evals/run", evalHandler.RunEval)
	mux.HandleFunc("GET /api/v1/evals/{id}", evalHandler.GetEvalRun)
	mux.HandleFunc("GET /api/v1/evals/{id}/diagnose", evalHandler.DiagnoseEval)
	mux.HandleFunc("GET /api/v1/evals/{id}/report", evalHandler.GetEvalReport)
	mux.HandleFunc("POST /api/v1/evals/compare", evalHandler.CompareEval)

	// Trigger API routes
	mux.HandleFunc("POST /api/v1/trigger/match", triggerHandler.MatchTrigger)
	mux.HandleFunc("POST /api/v1/trigger/validate", triggerHandler.ValidateAntiPatterns)
	mux.HandleFunc("POST /api/v1/trigger/inject", triggerHandler.InjectVariables)
	mux.HandleFunc("GET /api/v1/trigger/anti-patterns", triggerHandler.GetAntiPatterns)

	// Taxonomy API routes
	mux.HandleFunc("GET /api/v1/taxonomy", taxonomyHandler.GetTaxonomy)
	mux.HandleFunc("PUT /api/v1/taxonomy/biz_lines", taxonomyHandler.UpdateBizLines)
	mux.HandleFunc("PUT /api/v1/taxonomy/tags", taxonomyHandler.UpdateTags)

	// LLM Config API routes
	mux.HandleFunc("GET /api/v1/llm-config", llmConfigHandler.GetLLMConfig)
	mux.HandleFunc("PUT /api/v1/llm-config", llmConfigHandler.UpdateLLMConfig)

	// Admin API routes
	mux.HandleFunc("GET /api/v1/admin/status", adminHandler.GetStatus)
	mux.HandleFunc("GET /api/v1/admin/git-info", adminHandler.GetGitInfo)
	mux.HandleFunc("GET /api/v1/admin/repo-config", adminHandler.GetRepoConfig)
	mux.HandleFunc("PUT /api/v1/admin/repo-config", adminHandler.UpdateRepoConfig)
	mux.HandleFunc("POST /api/v1/admin/reload", adminHandler.ReloadConfig)
	mux.HandleFunc("POST /api/v1/admin/restart", adminHandler.Restart)

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
