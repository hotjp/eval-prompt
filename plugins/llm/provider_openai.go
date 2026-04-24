// Package llm provides LLM provider implementations.
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

// OpenAIProvider implements Provider for OpenAI-compatible APIs.
type OpenAIProvider struct {
	APIKey   string
	Endpoint string // defaults to https://api.openai.com/v1
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(apiKey, endpoint string) *OpenAIProvider {
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		APIKey:   apiKey,
		Endpoint: endpoint,
	}
}

// Name implements Provider.
func (p *OpenAIProvider) Name() string { return "openai" }

type openaiChatRequest struct {
	Model       string  `json:"model"`
	Messages    []msg   `json:"messages"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
}

type msg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiChatResponse struct {
	ID      string `json:"id"`
	Choices []choice `json:"choices"`
	Usage   usage   `json:"usage"`
	Model   string `json:"model"`
}

type choice struct {
	Message     msg    `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Invoke implements Provider.
func (p *OpenAIProvider) Invoke(ctx context.Context, prompt string, model string, temperature float64) (*LLMResponse, error) {
	body := openaiChatRequest{
		Model: model,
		Messages: []msg{
			{Role: "user", Content: prompt},
		},
		Temperature: temperature,
	}
	return p.doChat(ctx, body, model)
}

// InvokeWithSchema implements Provider.
func (p *OpenAIProvider) InvokeWithSchema(ctx context.Context, prompt string, schema json.RawMessage) (json.RawMessage, error) {
	// Use response_format for structured output (OpenAI beta)
	body := map[string]any{
		"model": "gpt-4o", // TODO: make configurable
		"messages": []msg{
			{Role: "user", Content: prompt},
		},
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": schema,
		},
	}
	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (p *OpenAIProvider) doChat(ctx context.Context, body openaiChatRequest, model string) (*LLMResponse, error) {
	raw, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	var chatResp openaiChatResponse
	if err := json.Unmarshal(raw, &chatResp); err != nil {
		return nil, fmt.Errorf("openai: unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, errors.New("openai: no choices in response")
	}

	return &LLMResponse{
		Content:    chatResp.Choices[0].Message.Content,
		Model:      chatResp.Model,
		TokensIn:   chatResp.Usage.PromptTokens,
		TokensOut:  chatResp.Usage.CompletionTokens,
		StopReason: chatResp.Choices[0].FinishReason,
		RawResponse: raw,
	}, nil
}

func (p *OpenAIProvider) doRequest(ctx context.Context, body any) (json.RawMessage, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp openaiErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("openai: %s: %s", errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("openai: status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
