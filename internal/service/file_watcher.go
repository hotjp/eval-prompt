// Package service provides L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ImportEvent represents an event from the file watcher.
type ImportEvent struct {
	Type      string   `json:"type"` // "add", "error"
	AssetID   string   `json:"asset_id,omitempty"`
	Path      string   `json:"path,omitempty"`
	Timestamp string   `json:"timestamp"`
	Errors    []string `json:"errors,omitempty"`
}

// ImportStatus represents the current state of the import directory.
type ImportStatus struct {
	Importing    bool   `json:"importing"`
	PendingCount int    `json:"pending_count"`
	ImportPath   string `json:"import_path"`
}

// FileWatcher monitors the .import/ directory for new asset folders.
type FileWatcher struct {
	mu         sync.RWMutex
	watcher    *fsnotify.Watcher
	fileMgr    AssetFileManager
	indexer    AssetIndexer
	logger     *slog.Logger
	importPath string
	stopChan   chan struct{}
	doneChan   chan struct{}

	// SSE clients
	clients    map[string]chan ImportEvent
	clientsMu  sync.RWMutex

	// Import state — protected by mu
	importing bool
}

// NewFileWatcher creates a new FileWatcher.
func NewFileWatcher(fileMgr AssetFileManager, indexer AssetIndexer, logger *slog.Logger, importPath string) *FileWatcher {
	if logger == nil {
		logger = slog.Default()
	}
	return &FileWatcher{
		fileMgr:    fileMgr,
		indexer:    indexer,
		logger:     logger,
		importPath: importPath,
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
		clients:    make(map[string]chan ImportEvent),
	}
}

// Start begins watching the import directory.
func (w *FileWatcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	w.watcher = watcher

	// Watch the import directory
	if err := watcher.Add(w.importPath); err != nil {
		return fmt.Errorf("watch import dir: %w", err)
	}

	w.logger.Info("file watcher started", "path", w.importPath, "layer", "L4")

	// Process events in goroutine
	go w.processEvents(ctx)

	return nil
}

// processEvents handles file system events.
func (w *FileWatcher) processEvents(ctx context.Context) {
	defer close(w.doneChan)

	for {
		select {
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(ctx, event)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("watcher error", "error", err, "layer", "L4")
			w.broadcastEvent(ImportEvent{
				Type:      "error",
				Path:      w.importPath,
				Timestamp: time.Now().Format(time.RFC3339),
				Errors:    []string{err.Error()},
			})
		}
	}
}

// handleEvent processes a single file system event.
func (w *FileWatcher) handleEvent(ctx context.Context, event fsnotify.Event) {
	// Only care about new directories in import path
	if event.Op != fsnotify.Create {
		return
	}

	// Check if it's a directory under import path
	if !strings.HasPrefix(event.Name, w.importPath) {
		return
	}
	info, err := os.Stat(event.Name)
	if err != nil || !info.IsDir() {
		return
	}

	// Get the directory name
	dirName := filepath.Base(event.Name)
	if dirName == "" || dirName == "." {
		return
	}

	w.logger.Info("detected new import folder", "path", event.Name, "layer", "L4")
	w.runImport(ctx, event.Name, "fsnotify")
}

// GetStatus returns the current import status.
func (w *FileWatcher) GetStatus() ImportStatus {
	w.mu.RLock()
	importing := w.importing
	w.mu.RUnlock()

	return ImportStatus{
		Importing:    importing,
		PendingCount: w.countPending(),
		ImportPath:   w.importPath,
	}
}

// countPending counts files and directories waiting in the import path.
func (w *FileWatcher) countPending() int {
	entries, err := os.ReadDir(w.importPath)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}
		count++
	}
	return count
}

// TriggerImport starts a manual import of all pending folders.
// It returns nil if the import was accepted (started or already running).
func (w *FileWatcher) TriggerImport(ctx context.Context) error {
	if _, err := os.Stat(w.importPath); os.IsNotExist(err) {
		return fmt.Errorf("import directory does not exist: %s", w.importPath)
	}
	go w.runImport(ctx, w.importPath, "manual")
	return nil
}

// runImport performs the actual import with concurrency protection.
// If another import is already running, this call is skipped.
func (w *FileWatcher) runImport(ctx context.Context, source string, trigger string) {
	w.mu.Lock()
	if w.importing {
		w.mu.Unlock()
		w.logger.Info("import already in progress, skipping", "trigger", trigger, "layer", "L4")
		return
	}
	w.importing = true
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.importing = false
		w.mu.Unlock()

		// If new files arrived while we were importing, trigger another pass.
		if w.countPending() > 0 {
			w.logger.Info("new files detected after import, re-triggering", "layer", "L4")
			go w.runImport(context.Background(), w.importPath, "auto-retry")
		}
	}()

	result, err := w.fileMgr.Scan(ctx, source)
	if err != nil {
		w.logger.Error("import failed", "source", source, "trigger", trigger, "error", err, "layer", "L4")
		w.broadcastEvent(ImportEvent{
			Type:      "error",
			Path:      source,
			Timestamp: time.Now().Format(time.RFC3339),
			Errors:    []string{fmt.Sprintf("scan failed: %v", err)},
		})
		return
	}

	for _, assetID := range result.CreatedAssets {
		w.broadcastEvent(ImportEvent{
			Type:      "add",
			AssetID:   assetID,
			Path:      source,
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	for _, assetID := range result.UpdatedAssets {
		w.broadcastEvent(ImportEvent{
			Type:      "update",
			AssetID:   assetID,
			Path:      source,
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	if len(result.Errors) > 0 {
		w.broadcastEvent(ImportEvent{
			Type:      "error",
			Path:      source,
			Timestamp: time.Now().Format(time.RFC3339),
			Errors:    result.Errors,
		})
	}

	w.logger.Info("import completed",
		"trigger", trigger,
		"created", len(result.CreatedAssets),
		"updated", len(result.UpdatedAssets),
		"errors", len(result.Errors),
		"layer", "L4",
	)

	// Reconcile index so new assets appear in search immediately
	if w.indexer != nil {
		if _, err := w.indexer.Reconcile(ctx); err != nil {
			w.logger.Error("reconcile after import failed", "error", err, "layer", "L4")
		}
	}
}

// Stop stops the file watcher.
func (w *FileWatcher) Stop() error {
	close(w.stopChan)
	<-w.doneChan
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}

// RegisterClient registers an SSE client to receive import events.
// Returns a channel that will be closed when the client should disconnect.
func (w *FileWatcher) RegisterClient(clientID string) chan ImportEvent {
	w.clientsMu.Lock()
	defer w.clientsMu.Unlock()

	ch := make(chan ImportEvent, 100) // Buffer for backpressure
	w.clients[clientID] = ch
	w.logger.Info("SSE client registered", "client_id", clientID, "layer", "L4")
	return ch
}

// UnregisterClient removes an SSE client.
func (w *FileWatcher) UnregisterClient(clientID string) {
	w.clientsMu.Lock()
	defer w.clientsMu.Unlock()

	if ch, ok := w.clients[clientID]; ok {
		close(ch)
		delete(w.clients, clientID)
		w.logger.Info("SSE client unregistered", "client_id", clientID, "layer", "L4")
	}
}

// broadcastEvent sends an event to all connected SSE clients.
func (w *FileWatcher) broadcastEvent(event ImportEvent) {
	w.clientsMu.RLock()
	defer w.clientsMu.RUnlock()

	for clientID, ch := range w.clients {
		select {
		case ch <- event:
		default:
			// Client buffer full, skip this event
			w.logger.Warn("SSE client buffer full, dropping event", "client_id", clientID, "layer", "L4")
		}
	}
}
