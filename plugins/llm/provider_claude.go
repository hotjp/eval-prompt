package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ClaudeProvider implements Provider for Anthropic Claude APIs.
type ClaudeProvider struct {
	APIKey   string
	Endpoint string // defaults to https://api.anthropic.com/v1
	PingPath string // lightweight health check path, e.g. "/v1/models"
}

// NewClaudeProvider creates a new Claude provider.
func NewClaudeProvider(apiKey, endpoint, pingPath string) (*ClaudeProvider, error) {
	if apiKey == "" {
		return nil, errors.New("claude: api key required")
	}
	if endpoint == "" {
		endpoint = "https://api.anthropic.com/v1"
	}
	if pingPath == "" {
		pingPath = "/v1/models"
	}
	return &ClaudeProvider{
		APIKey:   apiKey,
		Endpoint: endpoint,
		PingPath: pingPath,
	}, nil
}

// Name implements Provider.
func (p *ClaudeProvider) Name() string { return "claude" }

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeRequest struct {
	Model       string           `json:"model"`
	Messages    []claudeMessage  `json:"messages"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens"`
	// For structured output via JSON schema
	Beta string `json:"-"` // "json-schema-2025-03-01"
	// Disable thinking for Claude models that support it
	Thinking *thinkingBlock `json:"thinking,omitempty"`
}

type claudeResponse struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Role         string   `json:"role"`
	Content      []contentBlock `json:"content"`
	Model        string   `json:"model"`
	StopReason   string   `json:"stop_reason"`
	StopSequence string   `json:"stop_sequence"`
	Usage        claudeUsage `json:"usage"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// tool use block fields ignored for now
}

type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type claudeErrorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Invoke implements Provider.
func (p *ClaudeProvider) Invoke(ctx context.Context, prompt string, model string, temperature float64) (*LLMResponse, error) {
	reqBody := claudeRequest{
		Model: model,
		Messages: []claudeMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: temperature,
		MaxTokens:   4096, // default, should be configurable
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("claude: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint+"/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("claude: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "json-schema-2025-03-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("claude: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp claudeErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("claude: %s: %s", errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("claude: status %d: %s", resp.StatusCode, string(respBody))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, fmt.Errorf("claude: unmarshal response: %w", err)
	}

	content := ""
	for _, block := range claudeResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &LLMResponse{
		Content:    content,
		Model:      claudeResp.Model,
		TokensIn:   claudeResp.Usage.InputTokens,
		TokensOut:  claudeResp.Usage.OutputTokens,
		StopReason: claudeResp.StopReason,
		RawResponse: respBody,
	}, nil
}

// InvokeWithOptions implements Provider.
func (p *ClaudeProvider) InvokeWithOptions(ctx context.Context, prompt string, model string, temperature float64, opts InvokeOptions) (*LLMResponse, error) {
	reqBody := claudeRequest{
		Model: model,
		Messages: []claudeMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: temperature,
		MaxTokens:   4096,
	}
	if opts.DisableThinking {
		reqBody.Thinking = &thinkingBlock{Type: "disabled"}
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("claude: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint+"/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("claude: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("claude: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp claudeErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("claude: %s: %s", errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("claude: status %d: %s", resp.StatusCode, string(respBody))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, fmt.Errorf("claude: unmarshal response: %w", err)
	}

	content := ""
	for _, block := range claudeResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &LLMResponse{
		Content:    content,
		Model:      claudeResp.Model,
		TokensIn:   claudeResp.Usage.InputTokens,
		TokensOut:  claudeResp.Usage.OutputTokens,
		StopReason: claudeResp.StopReason,
		RawResponse: respBody,
	}, nil
}

// InvokeWithSchema implements Provider using Claude's JSON schema output mode.
func (p *ClaudeProvider) InvokeWithSchema(ctx context.Context, prompt string, schema json.RawMessage) (json.RawMessage, error) {
	// Claude uses a different format for structured output
	schemaMap := make(map[string]any)
	if err := json.Unmarshal(schema, &schemaMap); err != nil {
		return nil, fmt.Errorf("claude: unmarshal schema: %w", err)
	}

	reqBody := map[string]any{
		"model": "claude-sonnet-4-20250514", // TODO: make configurable
		"messages": []claudeMessage{
			{Role: "user", Content: prompt},
		},
		"max_tokens": 4096,
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": schemaMap,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("claude: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint+"/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("claude: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2025-03-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("claude: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp claudeErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("claude: %s: %s", errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("claude: status %d: %s", resp.StatusCode, string(respBody))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, fmt.Errorf("claude: unmarshal response: %w", err)
	}

	content := ""
	for _, block := range claudeResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return json.RawMessage(content), nil
}

// Embed implements Provider. Claude does not support embeddings API.
func (p *ClaudeProvider) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	return nil, errors.New("claude: embedding API is not supported")
}

// Ping implements Provider. If PingPath is set, sends GET to verify HTTP connectivity.
// If PingPath is empty, checks TCP connectivity to the endpoint host.
func (p *ClaudeProvider) Ping(ctx context.Context) error {
	if p.PingPath == "" {
		return pingTCP(ctx, p.Endpoint)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.Endpoint+p.PingPath, nil)
	if err != nil {
		return fmt.Errorf("claude: create ping request: %w", err)
	}
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("claude: ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("claude: ping status %d", resp.StatusCode)
	}
	return nil
}
