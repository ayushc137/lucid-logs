// Package llm provides a minimal OpenAI-compatible chat-completions client.
//
// This package implements:
//   - Provider presets for well-known OpenAI-compatible APIs
//   - Configurable base URL, API key, and model
//   - 60s timeout, graceful error handling
//   - No heavy dependencies — stdlib net/http only
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

// =============================================================================
// PRESETS
// =============================================================================

// Preset defines a well-known provider configuration.
type Preset struct {
	BaseURL string // OpenAI-compatible base URL (without /chat/completions)
}

// Presets maps preset keys to their base URLs.
var Presets = map[string]Preset{
	"openrouter": {BaseURL: "https://openrouter.ai/api/v1"},
	"openai":     {BaseURL: "https://api.openai.com/v1"},
	"gemini":     {BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai"},
	"ollama":     {BaseURL: "http://localhost:11434/v1"},
}

// DefaultModels provides a sensible default model per preset.
var DefaultModels = map[string]string{
	"openrouter": "deepseek/deepseek-chat",
	"openai":     "gpt-4o-mini",
	"gemini":     "gemini-2.0-flash",
	"ollama":     "llama3.1",
}

// ResolveBaseURL returns the base URL for a provider preset or a custom URL.
func ResolveBaseURL(provider, customBaseURL string) string {
	if provider == "custom" && customBaseURL != "" {
		return customBaseURL
	}
	if preset, ok := Presets[provider]; ok {
		return preset.BaseURL
	}
	return customBaseURL
}

// DefaultModelFor returns a sensible default model for a provider.
func DefaultModelFor(provider string) string {
	if m, ok := DefaultModels[provider]; ok {
		return m
	}
	return "gpt-4o-mini"
}

// =============================================================================
// CLIENT
// =============================================================================

// Client is a minimal OpenAI-compatible chat-completions client.
type Client struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

// NewClient creates a new LLM client.
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		http: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatCompletionRequest is the OpenAI-compatible request body.
type chatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// chatCompletionResponse is the OpenAI-compatible response body.
type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *APIError `json:"error,omitempty"`
}

// APIError represents an error from the LLM API.
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("llm api error: %s (type=%s, code=%s)", e.Message, e.Type, e.Code)
}

// Complete sends a chat-completions request and returns the assistant's response.
func (c *Client) Complete(ctx context.Context, messages []Message) (string, error) {
	reqBody := chatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   1024,
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Try to parse structured error, fall back to raw body
		var errResp struct {
			Error *APIError `json:"error"`
		}
		if json.Unmarshal(raw, &errResp) == nil && errResp.Error != nil {
			return "", errResp.Error
		}
		return "", fmt.Errorf("llm api returned status %d: %s", resp.StatusCode, string(raw))
	}

	var result chatCompletionResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return "", result.Error
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("llm returned no choices")
	}

	return result.Choices[0].Message.Content, nil
}
