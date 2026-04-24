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
}

// NewClaudeProvider creates a new Claude provider.
func NewClaudeProvider(apiKey, endpoint string) (*ClaudeProvider, error) {
	if apiKey == "" {
		return nil, errors.New("claude: api key required")
	}
	if endpoint == "" {
		endpoint = "https://api.anthropic.com/v1"
	}
	return &ClaudeProvider{
		APIKey:   apiKey,
		Endpoint: endpoint,
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
