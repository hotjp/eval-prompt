// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/eval-prompt/internal/service"
)

// MCPHandler handles MCP (Model Context Protocol) SSE endpoints.
type MCPHandler struct {
	triggerService service.TriggerServicer
	evalService    service.EvalServiceer
	indexer        service.AssetIndexer

	// SSE clients
	clients    map[string]chan string
	clientsMux sync.RWMutex
	slog       *slog.Logger
}

// NewMCPHandler creates a new MCPHandler.
func NewMCPHandler(triggerService service.TriggerServicer, evalService service.EvalServiceer, indexer service.AssetIndexer, logger *slog.Logger) *MCPHandler {
	return &MCPHandler{
		triggerService: triggerService,
		evalService:    evalService,
		indexer:        indexer,
		clients:        make(map[string]chan string),
		slog:           logger,
	}
}

// MCPRequest represents a JSON-RPC request.
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
	ID      any            `json:"id,omitempty"`
}

// MCPResponse represents a JSON-RPC response.
type MCPResponse struct {
	JSONRPC string `json:"jsonrpc"`
	Result  any    `json:"result,omitempty"`
	Error   *MCPError `json:"error,omitempty"`
	ID      any    `json:"id,omitempty"`
}

// MCPError represents a JSON-RPC error.
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HandleSSE handles GET /mcp/v1/sse - SSE connection for event streaming.
func (h *MCPHandler) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	clientID := fmt.Sprintf("client-%d", len(h.clients))
	clientChan := make(chan string, 100)
	h.clientsMux.Lock()
	h.clients[clientID] = clientChan
	h.clientsMux.Unlock()

	// Send initial connection event
	fmt.Fprintf(w, "data: {\"event\":\"connected\",\"client_id\":\"%s\"}\n\n", clientID)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Stream events until client disconnects
	clientGone := r.Context().Done()
	for {
		select {
		case <-clientGone:
			h.clientsMux.Lock()
			delete(h.clients, clientID)
			h.clientsMux.Unlock()
			return
		case data, ok := <-clientChan:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// HandlePOST handles POST /mcp/v1 - JSON-RPC requests.
func (h *MCPHandler) HandlePOST(w http.ResponseWriter, r *http.Request) {
	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, -32700, "Parse error")
		return
	}

	ctx := r.Context()
	var result any
	var err error

	switch req.Method {
	case "prompts/list":
		result, err = h.handlePromptsList(ctx, req.Params)
	case "prompts/get":
		result, err = h.handlePromptsGet(ctx, req.Params)
	case "prompts/eval":
		result, err = h.handlePromptsEval(ctx, req.Params)
	default:
		h.writeError(w, -32601, fmt.Sprintf("Method not found: %s", req.Method))
		return
	}

	if err != nil {
		h.writeError(w, -32603, err.Error())
		return
	}

	resp := MCPResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *MCPHandler) handlePromptsList(ctx context.Context, params map[string]any) (any, error) {
	// Extract filters
	filters := service.SearchFilters{}
	if bizLine, ok := params["biz_line"].(string); ok {
		filters.BizLine = bizLine
	}
	if tag, ok := params["tag"].(string); ok {
		filters.Tags = []string{tag}
	}

	results, err := h.indexer.Search(ctx, "", filters)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	// Convert to MCP format
	prompts := make([]map[string]any, len(results))
	for i, r := range results {
		prompts[i] = map[string]any{
			"id":          r.ID,
			"name":        r.Name,
			"description": r.Description,
			"biz_line":    r.BizLine,
			"tags":        r.Tags,
		}
	}
	return map[string]any{"prompts": prompts}, nil
}

func (h *MCPHandler) handlePromptsGet(ctx context.Context, params map[string]any) (any, error) {
	id, ok := params["id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("id is required")
	}

	variables, _ := params["variables"].(map[string]string)
	label, _ := params["label"].(string)

	// Get asset detail
	detail, err := h.indexer.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("asset not found: %s", id)
	}

	// Build result
	result := map[string]any{
		"content":     "", // Would load from Git/file
		"description": detail.Description,
		"biz_line":    detail.BizLine,
		"tags":        detail.Tags,
	}

	// Apply variable injection if provided
	if len(variables) > 0 && result["content"] != "" {
		if triggerSvc, ok := h.triggerService.(service.TriggerServicer); ok {
			content, _ := triggerSvc.InjectVariables(ctx, result["content"].(string), variables)
			result["content"] = content
		}
	}

	_ = label // Would use label to determine which snapshot
	return result, nil
}

func (h *MCPHandler) handlePromptsEval(ctx context.Context, params map[string]any) (any, error) {
	id, ok := params["id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("id is required")
	}
	snapshotVersion, _ := params["snapshot_version"].(string)
	caseID, _ := params["case_id"].(string)

	// Run eval
	run, err := h.evalService.RunEval(ctx, id, snapshotVersion, []string{caseID})
	if err != nil {
		return nil, fmt.Errorf("eval run: %w", err)
	}

	return map[string]any{
		"run_id": run.ID,
		"status": run.Status,
	}, nil
}

func (h *MCPHandler) writeError(w http.ResponseWriter, code int, message string) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		Error: &MCPError{
			Code:    code,
			Message: message,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC always returns 200
	json.NewEncoder(w).Encode(resp)
}
