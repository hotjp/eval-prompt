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
	indexer       service.AssetIndexer

	// SSE clients
	clients    map[string]chan string
	clientsMux sync.RWMutex
	slog       *slog.Logger
}

// NewMCPHandler creates a new MCPHandler.
func NewMCPHandler(triggerService service.TriggerServicer, indexer service.AssetIndexer, logger *slog.Logger) *MCPHandler {
	return &MCPHandler{
		triggerService: triggerService,
		indexer:       indexer,
		clients:       make(map[string]chan string),
		slog:          logger,
	}
}

// MCPRequest represents a JSON-RPC request.
type MCPRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
	ID      any            `json:"id,omitempty"`
}

// MCPResponse represents a JSON-RPC response.
type MCPResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	Result  any       `json:"result,omitempty"`
	Error   *MCPError `json:"error,omitempty"`
	ID      any       `json:"id,omitempty"`
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
	// MCP protocol: prompts/list params are cursor? and limit?
	cursor, _ := params["cursor"].(string)
	limit, _ := params["limit"].(int)
	if limit <= 0 {
		limit = 20 // default limit
	}
	if limit > 100 {
		limit = 100 // max limit
	}

	// Search with empty query to get all (supports filtering via SearchFilters if needed)
	results, err := h.indexer.Search(ctx, "", service.SearchFilters{})
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	// Apply cursor offset (cursor is the index to start from)
	offset := 0
	if cursor != "" {
		for i, r := range results {
			if r.ID == cursor {
				offset = i
				break
			}
		}
	}

	// Apply limit
	if offset >= len(results) {
		results = []service.AssetSummary{}
	} else {
		results = results[offset:]
		if len(results) > limit {
			results = results[:limit]
		}
	}

	// Build next cursor
	var nextCursor string
	if offset+limit < len(results) {
		nextCursor = results[offset+limit-1].ID
	}

	// Convert to MCP format: { prompts: [{ name, description }] }
	// MCP uses 'name' as the identifier for prompts/get
	prompts := make([]map[string]any, len(results))
	for i, r := range results {
		prompts[i] = map[string]any{
			"name":        r.ID, // MCP name maps to our ID
			"description": r.Description,
		}
	}

	result := map[string]any{"prompts": prompts}
	if nextCursor != "" {
		result["nextCursor"] = nextCursor
	}
	return result, nil
}

func (h *MCPHandler) handlePromptsGet(ctx context.Context, params map[string]any) (any, error) {
	// MCP protocol: prompts/get params are name and arguments
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// MCP arguments maps to our variables for template injection
	arguments, _ := params["arguments"].(map[string]string)

	// MCP name maps to our asset ID
	detail, err := h.indexer.GetByID(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("prompt not found: %s", name)
	}

	// Get asset content from file
	content, err := h.indexer.GetFileContent(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt content: %w", err)
	}

	// Apply variable injection if arguments provided
	if len(arguments) > 0 {
		content, err = h.triggerService.InjectVariables(ctx, content, arguments)
		if err != nil {
			return nil, fmt.Errorf("variable injection failed: %w", err)
		}
	}

	// MCP protocol returns: { description?, messages: [{ role, content }] }
	// We treat the prompt as a user message
	return map[string]any{
		"description": detail.Description,
		"messages": []map[string]any{
			{"role": "user", "content": content},
		},
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
