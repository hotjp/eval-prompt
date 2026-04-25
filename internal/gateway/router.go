// Package gateway implements L5-Gateway layer: TLS termination, protocol adaptation,
// middleware, request routing, and static resource serving.
package gateway

import (
	"log/slog"
	"net/http"

	"github.com/eval-prompt/internal/gateway/handlers"
	"github.com/eval-prompt/internal/gateway/middleware"
	"github.com/eval-prompt/internal/service"

	_ "github.com/eval-prompt/docs" // swagger docs
	httpSwagger "github.com/swaggo/http-swagger"
)

// RouterConfig contains configuration for the router.
type RouterConfig struct {
	TriggerService service.TriggerServicer
	EvalService    service.EvalServiceer
	IndexService   service.AssetIndexer
	Logger         *slog.Logger
	Metrics        *middleware.MetricsCollector
	CORSOrigins    []string
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

	// Health check endpoints
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Check DB and Redis connectivity
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Static files (SPA)
	RegisterStaticRoutes(mux)

	// Swagger UI
	mux.Handle("/swagger/", httpSwagger.Handler())

	return mux
}
