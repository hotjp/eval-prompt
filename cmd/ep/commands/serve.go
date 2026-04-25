package commands

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/internal/gateway"
	"github.com/eval-prompt/internal/gateway/handlers"
	"github.com/eval-prompt/internal/gateway/middleware"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/storage"
	"github.com/eval-prompt/plugins/gitbridge"
	"github.com/eval-prompt/plugins/llm"
	"github.com/eval-prompt/plugins/search"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动本地 HTTP 服务（包含 Web UI）",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		host, _ := cmd.Flags().GetString("host")
		noBrowser, _ := cmd.Flags().GetBool("no-browser")

		addr := fmt.Sprintf("%s:%d", host, port)
		fmt.Printf("启动服务: http://%s\n", addr)

		// Initialize services
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

		// Load configuration
		cfg, err := config.Load("")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Load taxonomy from config/taxonomy.yaml
		taxonomy, err := config.LoadTaxonomy("config/taxonomy.yaml")
		if err != nil {
			return fmt.Errorf("failed to load taxonomy: %w", err)
		}

		// Create storage client for database checks
		storageClient, err := storage.NewClient(cfg.Database)
		if err != nil {
			logger.Warn("failed to create storage client", "error", err)
			// Continue without storage - some features will be degraded
		}

		// Create LLM providers based on config
		llmProviders := make(map[string]llm.Interface)
		for _, providerConfig := range cfg.Plugins.LLM {
			if providerConfig.Provider == "" {
				continue
			}
			provider, err := llm.NewProvider(providerConfig)
			if err != nil {
				logger.Warn("failed to create LLM provider", "name", providerConfig.Name, "error", err)
				continue
			}
			llmProviders[providerConfig.Name] = provider
			logger.Info("created LLM provider", "name", providerConfig.Name, "type", providerConfig.Provider)
		}

		// Use first available provider as the default LLM invoker, or NoopInvoker if none
		var llmInvoker llm.Interface = &llm.NoopInvoker{}
		if len(llmProviders) > 0 {
			for name, p := range llmProviders {
				llmInvoker = p
				logger.Info("using LLM provider as default", "name", name)
				break
			}
		}

		// Create plugin instances
		indexer := search.Default()
		// Use current working directory as persist dir (user should run from project root)
		cwd, _ := os.Getwd()
		indexer.SetPersistDir(filepath.Join(cwd, ".eval-prompt"))
		if err := indexer.Load(); err != nil {
			logger.Warn("failed to load persisted index", "error", err)
		}
		gitBridge := gitbridge.NewBridge()
		if err := gitBridge.Open(cwd); err != nil {
			logger.Warn("failed to open git repo", "error", err)
		}
		indexer.SetGitBridge(gitBridge)

		// Create trigger service
		triggerService := service.NewTriggerService(indexer, gitBridge)

		// Create eval service (TODO: with real implementation)
		evalService := service.NewEvalService()

		// Create storage checker for readyz
		var storageChecker handlers.StorageChecker
		dbDSN := cfg.Database.DSN
		if dbDSN == "" {
			dbDSN = "eval-prompt.db"
		}
		dbDSN = fmt.Sprintf("%s?_fk=1&_journal_mode=WAL", dbDSN)
		storageChecker = handlers.NewSQLiteChecker(dbDSN)

		// Create LLM checker for readyz
		llmChecker := handlers.NewLLMCheckerAdapter(llmInvoker)

		// Create router with dependency injection
		router := gateway.NewRouter(gateway.RouterConfig{
			TriggerService:   triggerService,
			EvalService:     evalService,
			IndexService:    indexer,
			FileManager:     indexer,
			Logger:          logger,
			Metrics:         middleware.NewMetricsCollector(),
			CORSOrigins:     []string{"http://localhost:8080", "http://127.0.0.1:8080"},
			TaxonomyConfig:   taxonomy,
			TaxonomyFilePath: "config/taxonomy.yaml",
			AdminConfig:     cfg,
			RestartFunc:     RequestRestart,
			StorageClient:   storageChecker,
			LLMInvoker:     llmChecker,
			LLMConfig:      &cfg.Plugins.LLM,
			LLMConfigPath:  "config/llm.yaml",
		})

		// Create HTTP server
		server := &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		// Channel to receive shutdown signal
		shutdownChan := make(chan struct{})

		// Handle SIGTERM for graceful shutdown / restart
		go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
			sig := <-sigChan

			logger.Info("received signal, shutting down", "signal", sig.String())

			// Shutdown gracefully
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			server.Shutdown(shutdownCtx)

			// Close storage if open
			if storageClient != nil {
				storageClient.Close()
			}

			close(shutdownChan)
		}()

		// Start server in goroutine
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("server error", "error", err)
			}
		}()

		// Print startup info
		fmt.Printf("服务已启动: http://%s\n", addr)
		fmt.Printf("API 端点: http://%s/mcp/v1\n", addr)
		fmt.Printf("SSE 端点: http://%s/mcp/v1/sse\n", addr)

		if !noBrowser {
			fmt.Println("正在打开浏览器...")
		}

		// Wait for shutdown signal or restart request
		select {
		case <-shutdownChan:
			logger.Info("server stopped", "action", "shutdown")
		case <-WaitForRestart():
			logger.Info("server stopped", "action", "restart requested")
		}

		// If restart was requested, exec ourselves
		if IsRestartRequested() {
			logger.Info("restarting server...")
			execSelf()
		}

		 return nil
	},
}

func init() {
	serveCmd.Flags().Int("port", 8080, "服务端口")
	serveCmd.Flags().String("host", "127.0.0.1", "监听地址")
	serveCmd.Flags().Bool("no-browser", false, "不自动打开浏览器")
}

// execSelf replaces the current process with a new instance of the same binary
func execSelf() {
	// Get the current executable path
	exe, err := os.Executable()
	if err != nil {
		fmt.Printf("failed to get executable: %v\n", err)
		return
	}

	// Get current args (including any flags)
	args := os.Args

	// Use syscall.Exec to replace the current process
	// This replaces the Go runtime with a new instance
	syscall.Exec(exe, args, os.Environ())
}
