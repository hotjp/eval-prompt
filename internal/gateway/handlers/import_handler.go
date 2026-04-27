package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/eval-prompt/internal/service"
)

// ImportHandler handles import-related HTTP endpoints.
type ImportHandler struct {
	fileWatcher *service.FileWatcher
	logger     *slog.Logger
}

// NewImportHandler creates a new ImportHandler.
func NewImportHandler(fileWatcher *service.FileWatcher, logger *slog.Logger) *ImportHandler {
	return &ImportHandler{
		fileWatcher: fileWatcher,
		logger:      logger,
	}
}

// HandleSSE handles GET /api/v1/import/events - SSE connection for import event streaming.
func (h *ImportHandler) HandleSSE(w http.ResponseWriter, r *http.Request) {
	if h.fileWatcher == nil {
		h.writeError(w, http.StatusServiceUnavailable, "file watcher not enabled")
		return
	}

	// Generate client ID
	clientID := fmt.Sprintf("import-%d", time.Now().UnixNano())

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Register client
	clientChan := h.fileWatcher.RegisterClient(clientID)
	defer h.fileWatcher.UnregisterClient(clientID)

	// Send initial connection event
	fmt.Fprintf(w, "data: {\"event\":\"connected\",\"client_id\":\"%s\"}\n\n", clientID)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Stream events until client disconnects
	clientGone := r.Context().Done()
	for {
		select {
		case <-clientGone:
			h.logger.Info("SSE client disconnected", "client_id", clientID, "layer", "L5")
			return
		case event, ok := <-clientChan:
			if !ok {
				// Channel closed
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				h.logger.Error("failed to marshal import event", "error", err, "layer", "L5")
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// writeError writes an error response.
func (h *ImportHandler) writeError(w http.ResponseWriter, status int, format string, args ...any) {
	h.logger.Error(fmt.Sprintf(format, args...), "layer", "L5", "status", status)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": fmt.Sprintf(format, args...),
	})
}
