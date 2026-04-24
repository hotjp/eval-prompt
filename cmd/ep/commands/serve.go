package commands

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/eval-prompt/internal/gateway"
	"github.com/eval-prompt/internal/gateway/middleware"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/plugins/gitbridge"
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

		// Create plugin instances
		indexer := search.NewIndexer()
		gitBridge := gitbridge.NewBridge()

		// Create trigger service
		triggerService := service.NewTriggerService(indexer, gitBridge)

		// Create eval service (TODO: with real implementation)
		evalService := service.NewEvalService()

		// Create router with dependency injection
		router := gateway.NewRouter(gateway.RouterConfig{
			TriggerService: triggerService,
			EvalService:    evalService,
			IndexService:   indexer,
			Logger:         logger,
			Metrics:        middleware.NewMetricsCollector(),
			CORSOrigins:    []string{"http://localhost:8080", "http://127.0.0.1:8080"},
		})

		// Create HTTP server
		server := &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

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
			// Browser opening would require exec.Command("open", url)
		}

		// Wait for context cancellation
		ctx := context.Background()
		<-ctx.Done()

		// Shutdown gracefully
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)

		return nil
	},
}

func init() {
	serveCmd.Flags().Int("port", 8080, "服务端口")
	serveCmd.Flags().String("host", "127.0.0.1", "监听地址")
	serveCmd.Flags().Bool("no-browser", false, "不自动打开浏览器")
}
