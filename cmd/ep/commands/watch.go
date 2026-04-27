package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/plugins/gitbridge"
	"github.com/eval-prompt/plugins/search"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch .import/ directory for new assets",
	Long:  "Monitor the .import/ directory and automatically import new asset folders",
	RunE:  runWatch,
}

func init() {
	assetCmd.AddCommand(watchCmd)
	watchCmd.Flags().String("import-dir", ".import", "Directory to watch for new assets")
}

func runWatch(cmd *cobra.Command, args []string) error {
	importDir, _ := cmd.Flags().GetString("import-dir")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Resolve import directory path
	importPath, err := filepath.Abs(importDir)
	if err != nil {
		return fmt.Errorf("invalid import path: %w", err)
	}

	// Check if import directory exists
	if _, err := os.Stat(importPath); os.IsNotExist(err) {
		// Create import directory if it doesn't exist
		if err := os.MkdirAll(importPath, 0755); err != nil {
			return fmt.Errorf("create import directory: %w", err)
		}
		logger.Info("created import directory", "path", importPath)
	}

	// Initialize indexer
	indexer := search.NewIndexer()
	indexer.SetPersistDir(filepath.Join(".eval-prompt"))

	// Initialize git bridge
	gitBridge := gitbridge.NewBridge()
	cwd, _ := os.Getwd()
	if err := gitBridge.Open(cwd); err != nil {
		logger.Warn("failed to open git repo", "error", err)
	}
	indexer.SetGitBridge(gitBridge)

	// Load persisted index
	if err := indexer.Load(); err != nil {
		logger.Warn("failed to load persisted index", "error", err)
	}

	// Create file watcher
	fileWatcher := service.NewFileWatcher(indexer, indexer, logger, importPath)

	// Start watching
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := fileWatcher.Start(ctx); err != nil {
		return fmt.Errorf("start file watcher: %w", err)
	}

	logger.Info("watching for new imports", "path", importPath)
	fmt.Printf("Watching %s for new assets...\n", importPath)
	fmt.Println("Drop folders into .import/ to import them automatically")
	fmt.Println("Press Ctrl+C to stop")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("received signal, stopping", "signal", sig.String())
	}

	// Stop file watcher
	if err := fileWatcher.Stop(); err != nil {
		logger.Error("failed to stop file watcher", "error", err)
	}

	fmt.Println("Stopped watching")
	return nil
}
