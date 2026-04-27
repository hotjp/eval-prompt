package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaProvider implements Provider for Ollama local APIs.
type OllamaProvider struct {
	Endpoint string // defaults to http://localhost:11434
	PingPath string // lightweight health check path, e.g. "/api/tags"
}

// NewOllamaProvider creates a new Ollama provider.
func NewOllamaProvider(endpoint, pingPath string) (*OllamaProvider, error) {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if pingPath == "" {
		pingPath = "/api/tags"
	}
	return &OllamaProvider{
		Endpoint: endpoint,
		PingPath: pingPath,
	}, nil
}

// Name implements Provider.
func (p *OllamaProvider) Name() string { return "ollama" }

type ollamaGenerateRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature,omitempty"`
	Stream       bool   `json:"stream"`
	Format      string  `json:"format"` // "json" for structured output
}

type ollamaGenerateResponse struct {
	Model     string `json:"model"`
	Response string `json:"response"`
	DoneReason string `json:"done_reason,omitempty"`
	Context   []int  `json:"context,omitempty"`
	TotalDuration int64 `json:"total_duration,omitempty"`
	LoadDuration  int64 `json:"load_duration,omitempty"`
	PromptEvalCount int `json:"prompt_eval_count,omitempty"`
	EvalCount      int `json:"eval_count,omitempty"`
}

type ollamaErrorResponse struct {
	Error string `json:"error"`
}

// Invoke implements Provider.
func (p *OllamaProvider) Invoke(ctx context.Context, prompt string, model string, temperature float64) (*LLMResponse, error) {
	reqBody := ollamaGenerateRequest{
		Model:       model,
		Prompt:      prompt,
		Temperature: temperature,
		Stream:       false,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ollama: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second} // Ollama can be slow
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ollama: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ollamaErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("ollama: %s", errResp.Error)
		}
		return nil, fmt.Errorf("ollama: status %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("ollama: unmarshal response: %w", err)
	}

	return &LLMResponse{
		Content:    ollamaResp.Response,
		Model:      ollamaResp.Model,
		TokensIn:   ollamaResp.PromptEvalCount,
		TokensOut:  ollamaResp.EvalCount,
		StopReason: ollamaResp.DoneReason,
		RawResponse: respBody,
	}, nil
}

// InvokeWithOptions implements Provider. Ollama does not support disabling thinking via API.
func (p *OllamaProvider) InvokeWithOptions(ctx context.Context, prompt string, model string, temperature float64, opts InvokeOptions) (*LLMResponse, error) {
	// Ollama doesn't have a standard API parameter to disable thinking
	// Just delegate to regular Invoke
	return p.Invoke(ctx, prompt, model, temperature)
}

// InvokeWithSchema implements Provider using Ollama's JSON mode.
func (p *OllamaProvider) InvokeWithSchema(ctx context.Context, prompt string, schema json.RawMessage) (json.RawMessage, error) {
	// Ollama supports JSON output via format: "json"
	// It doesn't natively support JSON Schema, but we can include schema in the prompt
	reqBody := map[string]any{
		"model": "llama3", // TODO: make configurable
		"prompt": prompt,
		"stream": false,
		"format": "json", // request JSON output
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint+"/api/generate", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ollama: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ollama: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ollamaErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("ollama: %s", errResp.Error)
		}
		return nil, fmt.Errorf("ollama: status %d: %s", resp.StatusCode, string(respBody))
	}

	// Ollama returns raw JSON text in Response field
	var ollamaResp ollamaGenerateResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("ollama: unmarshal response: %w", err)
	}

	// Validate that the response is valid JSON
	if err := json.Unmarshal([]byte(ollamaResp.Response), &map[string]any{}); err != nil {
		return nil, fmt.Errorf("ollama: response is not valid JSON: %w", err)
	}

	return json.RawMessage(ollamaResp.Response), nil
}

// Embed implements Provider by calling the Ollama embeddings API.
// Ollama /api/embeddings accepts a single prompt per request, so we make parallel requests.
func (p *OllamaProvider) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	results := make([][]float64, len(texts))
	for i, text := range texts {
		reqBody := map[string]any{
			"model":  "nomic-embed-text", // default model for embeddings
			"prompt": text,
		}

		payload, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("ollama embed: marshal request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint+"/api/embeddings", bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("ollama embed: create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("ollama embed: do request: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("ollama embed: read body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("ollama embed: status %d: %s", resp.StatusCode, string(respBody))
		}

		var embedResp struct {
			Embedding  []float64   `json:"embedding"`
			Embeddings [][]float64 `json:"embeddings,omitempty"`
		}
		if err := json.Unmarshal(respBody, &embedResp); err != nil {
			return nil, fmt.Errorf("ollama embed: unmarshal response: %w", err)
		}

		if len(embedResp.Embeddings) > 0 {
			results[i] = embedResp.Embeddings[0]
		} else if len(embedResp.Embedding) > 0 {
			results[i] = embedResp.Embedding
		} else {
			return nil, fmt.Errorf("ollama embed: empty embedding for text %d", i)
		}
	}

	return results, nil
}

// Ping implements Provider. If PingPath is set, sends GET to verify HTTP connectivity.
// If PingPath is empty, checks TCP connectivity to the endpoint host.
func (p *OllamaProvider) Ping(ctx context.Context) error {
	if p.PingPath == "" {
		return pingTCP(ctx, p.Endpoint)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.Endpoint+p.PingPath, nil)
	if err != nil {
		return fmt.Errorf("ollama: create ping request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama: ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama: ping status %d", resp.StatusCode)
	}
	return nil
}
