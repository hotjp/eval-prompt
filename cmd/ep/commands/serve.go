package commands

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/internal/gateway"
	"github.com/eval-prompt/internal/gateway/handlers"
	"github.com/eval-prompt/internal/gateway/middleware"
	"github.com/eval-prompt/internal/i18n"
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

		// Kill any existing ep instance on this port
		addr := fmt.Sprintf("%s:%d", host, port)
		if err := killPort(port); err != nil {
			logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
			logger.Warn("failed to kill existing process", "port", port, "error", err)
		}

		fmt.Printf("启动服务: http://%s\n", addr)

		// Initialize services
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

		// Load configuration
		cfg, err := config.Load("")
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize i18n (EP_LANG > system LANG > config > en-US)
		if err := i18n.Init(); err != nil {
			logger.Warn("failed to initialize i18n", "error", err)
		} else {
			// Apply config language if no EP_LANG override
			i18n.SetLangIfNotEnv(cfg.Lang)
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
		var defaultModel string
		if len(llmProviders) > 0 {
			// First, look for a provider marked as default
			for _, providerConfig := range cfg.Plugins.LLM {
				if providerConfig.Default && providerConfig.Provider != "" {
					if p, ok := llmProviders[providerConfig.Name]; ok {
						llmInvoker = p
						defaultModel = providerConfig.DefaultModel
						logger.Info("using default LLM provider", "name", providerConfig.Name, "model", defaultModel)
						break
					}
				}
			}
			// If no default found, use first available
			if defaultModel == "" {
				for name, p := range llmProviders {
					llmInvoker = p
					// Find the corresponding config to get default model
					for _, providerConfig := range cfg.Plugins.LLM {
						if providerConfig.Name == name && providerConfig.Provider != "" {
							defaultModel = providerConfig.DefaultModel
							break
						}
					}
					logger.Info("using first available LLM provider as default", "name", name, "model", defaultModel)
					break
				}
			}
			// If defaultModel still not set, error
			if defaultModel == "" {
				logger.Error("default_model not configured: set default_model in LLM config or set default=true on a provider")
			}
		} else {
			logger.Info("no LLM providers configured")
		}

		// Create plugin instances
		indexer := search.Default()
		cwd, _ := os.Getwd()
		// Determine repo path: use config's RepoPath if set, otherwise fall back to cwd
		repoPath := cwd
		if cfg.PromptAssets.RepoPath != "" {
			if absPath, err := filepath.Abs(cfg.PromptAssets.RepoPath); err == nil {
				if _, statErr := os.Stat(absPath); statErr == nil {
					repoPath = absPath
					logger.Info("using repo path from config", "path", repoPath)
				}
			}
		}
		indexer.SetPersistDir(filepath.Join(cwd, ".eval-prompt"))
		if err := indexer.Load(); err != nil {
			logger.Warn("failed to load persisted index", "error", err)
		}
		gitBridge := gitbridge.NewBridge()
		if err := gitBridge.Open(repoPath); err != nil {
			logger.Warn("failed to open git repo", "error", err)
		}
		indexer.SetGitBridge(gitBridge)

		// Reconcile index with filesystem on startup
		report, err := indexer.Reconcile(cmd.Context())
		if err != nil {
			logger.Warn("reconcile failed", "error", err)
		} else if report.Added > 0 || report.Updated > 0 || report.Deleted > 0 {
			logger.Info("reconcile completed", "added", report.Added, "updated", report.Updated, "deleted", report.Deleted)
		}

		// Create trigger service
		triggerService := service.NewTriggerService(indexer, gitBridge)

		// Create eval execution and call stores (filesystem-based, not SQLite)
		evalsBaseDir := filepath.Join(cwd, ".evals")
		executionStore := service.NewExecutionFileStore(filepath.Join(evalsBaseDir, "executions"))
		callStore := service.NewLLMCallFileStore(filepath.Join(evalsBaseDir, "calls"))

		// Create eval service with stores
		evalService := service.NewEvalService().
			WithExecutionStore(executionStore).
			WithCallStore(callStore).
			WithEvalsDir(filepath.Join(cwd, "evals"))

		// Create semantic service only if LLM is properly configured with a model
		var semanticService *service.SemanticService
		if defaultModel != "" {
			semanticService = service.NewSemanticService(llmInvoker, defaultModel)
			triggerService = triggerService.WithSemanticAnalyzer(semanticService, defaultModel)
			evalService = evalService.WithSemanticAnalyzer(semanticService)
		} else {
			logger.Info("LLM not configured, semantic features disabled")
		}

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
		llmChecker.SetDefaultModel(defaultModel)

		// Create config manager and register config change handlers
		configManager := service.NewInMemoryConfigManager()

		// Create all handlers here so they can be registered with ConfigManager before routing
		adminHandler := handlers.NewAdminHandler(logger, cfg, RequestRestart, indexer, gitBridge, configManager)
		llmConfigHandler := handlers.NewLLMConfigHandler(&cfg.Plugins.LLM, logger, "config/llm.yaml", "config.yaml", &llmChecker, configManager)
		taxonomyHandler := handlers.NewTaxonomyHandler(taxonomy, logger, "config/taxonomy.yaml", configManager)

		// Register config change handlers
		configManager.Register("repo", adminHandler.HandleRepoChange)
		configManager.Register("llm", llmConfigHandler.HandleLLMChange)
		configManager.Register("taxonomy", taxonomyHandler.HandleTaxonomyChange)

		// Create router with dependency injection — pass pre-created handlers
		router := gateway.NewRouter(gateway.RouterConfig{
			TriggerService:   triggerService,
			EvalService:     evalService,
			IndexService:    indexer,
			FileManager:     indexer,
			Logger:          logger,
			Metrics:         middleware.NewMetricsCollector(),
			CORSOrigins:     []string{"http://localhost:8080", "http://127.0.0.1:8080"},
			AdminConfig:     cfg,
			RestartFunc:     RequestRestart,
			StorageClient:   storageChecker,
			LLMInvoker:     llmChecker,
			LLMInterface:   llmInvoker,
			LLMDefaultModel: defaultModel,
			ConfigManager:  configManager,
			GitBridge:      gitBridge,
			SemanticAnalyzer: semanticService,
			ExecutionStore:  executionStore,
			CallStore:       callStore,
			AdminHandler:    adminHandler,
			LLMConfigHandler: llmConfigHandler,
			TaxonomyHandler: taxonomyHandler,
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

		// Wait a bit for server to start, then check if it's running
		time.Sleep(500 * time.Millisecond)

		// Check if server is actually listening
		if lsofOutput, err := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port)).Output(); err == nil {
			if !strings.Contains(string(lsofOutput), "LISTEN") {
				fmt.Printf("\nError: Failed to start server on port %d\n", port)
				fmt.Printf("Another process may be using this port.\n")
				fmt.Printf("Kill it with: pkill -f 'ep serve' || true\n\n")
				return nil
			}
		}

		// Print startup info
		fmt.Printf("\n服务已启动: http://%s\n", addr)
		fmt.Printf("API 端点: http://%s/mcp/v1\n", addr)
		fmt.Printf("SSE 端点: http://%s/mcp/v1/sse\n\n", addr)

		if !noBrowser {
			url := fmt.Sprintf("http://%s", addr)
			fmt.Printf("正在打开浏览器: %s\n", url)
			exec.Command("open", url).Start()
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

// killPort kills any process listening on the given port
func killPort(port int) error {
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port))
	output, err := cmd.Output()
	if err != nil {
		// No process on this port
		return nil
	}
	pids := strings.TrimSpace(string(output))
	if pids == "" {
		return nil
	}
	// Kill each PID
	for _, pid := range strings.Split(pids, "\n") {
		if pid == "" {
			continue
		}
		killCmd := exec.Command("kill", pid)
		if err := killCmd.Run(); err != nil {
			return fmt.Errorf("failed to kill process %s: %w", pid, err)
		}
	}
	return nil
}

func init() {
	serveCmd.Flags().Int("port", 18880, "服务端口")
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
