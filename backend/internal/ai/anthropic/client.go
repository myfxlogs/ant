// Package anthropic implements an ai.AIProvider talking to the Anthropic
// Messages API (https://api.anthropic.com/v1/messages).
//
// The wire format differs from OpenAI Chat Completions in two key ways:
//
//  1. The "system" message must be sent as a top-level `system` field, not
//     in the `messages` array.
//  2. Authentication uses the `x-api-key` and `anthropic-version` headers
//     instead of `Authorization: Bearer ...`.
//
// We translate ai.Message lists into Anthropic's expected payload, run a
// non-streaming request, and reshape the response back into ai.Response.
// Streaming is currently surfaced as a single chunk to keep the adapter
// minimal; the engine never relies on Anthropic streaming today.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"anttrader/internal/ai"
)

const (
	defaultBaseURL   = "https://api.anthropic.com"
	defaultTimeout   = 300 * time.Second
	apiVersion       = "2023-06-01"
	defaultMaxTokens = 4096
	providerName     = "anthropic"
	defaultModel     = "claude-3-5-sonnet-latest"
)

type Client struct {
	apiKey      string
	model       string
	baseURL     string
	httpClient  *http.Client
	temperature *float64
	maxTokens   *int
}

func (c *Client) SetSamplingParams(temperature *float64, maxTokens *int) {
	if c == nil {
		return
	}
	c.temperature = temperature
	c.maxTokens = maxTokens
}

func (c *Client) SetTimeout(d time.Duration) {
	if c == nil || d <= 0 || c.httpClient == nil {
		return
	}
	c.httpClient.Timeout = d
}

func NewClient(apiKey, model, baseURL string) *Client {
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultModel
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		apiKey:     strings.TrimSpace(apiKey),
		model:      model,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}
}

type messagesRequest struct {
	Model       string    `json:"model"`
	System      string    `json:"system,omitempty"`
	Messages    []chatMsg `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature *float64  `json:"temperature,omitempty"`
}

type chatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (c *Client) Chat(ctx context.Context, messages []ai.Message) (*ai.Response, error) {
	system, msgs := splitSystem(messages)
	maxTokens := defaultMaxTokens
	if c.maxTokens != nil && *c.maxTokens > 0 {
		maxTokens = *c.maxTokens
	}
	body := messagesRequest{
		Model:       c.model,
		System:      system,
		Messages:    msgs,
		MaxTokens:   maxTokens,
		Temperature: c.temperature,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed messagesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if len(parsed.Content) == 0 {
		return nil, fmt.Errorf("%w: empty content", ai.ErrInvalidResponse)
	}
	var sb strings.Builder
	for _, item := range parsed.Content {
		if item.Type == "text" {
			sb.WriteString(item.Text)
		}
	}
	content := sb.String()
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("%w: empty text", ai.ErrInvalidResponse)
	}
	return &ai.Response{
		Content: content,
		Usage: ai.Usage{
			PromptTokens:     parsed.Usage.InputTokens,
			CompletionTokens: parsed.Usage.OutputTokens,
			TotalTokens:      parsed.Usage.InputTokens + parsed.Usage.OutputTokens,
		},
	}, nil
}

func (c *Client) StreamChat(ctx context.Context, messages []ai.Message) (<-chan ai.StreamChunk, error) {
	// Anthropic does support SSE streaming, but the rest of the codebase
	// already falls back to non-streaming when streaming is unavailable, so
	// we reuse Chat and surface a single chunk + Done. Keeping this minimal
	// avoids ~150 extra lines of SSE plumbing for a feature nothing yet uses.
	resp, err := c.Chat(ctx, messages)
	if err != nil {
		return nil, err
	}
	out := make(chan ai.StreamChunk, 2)
	out <- ai.StreamChunk{Content: resp.Content}
	out <- ai.StreamChunk{Done: true}
	close(out)
	return out, nil
}

func (c *Client) ValidateConfig() error {
	if c.apiKey == "" {
		return ai.ErrInvalidAPIKey
	}
	if c.model == "" {
		return fmt.Errorf("model name cannot be empty")
	}
	return nil
}

func (c *Client) GetModelName() string    { return c.model }
func (c *Client) GetProviderName() string { return providerName }

// splitSystem extracts the leading system message (if any) and returns the
// rest as a list of {role, content} suitable for the Anthropic Messages API.
func splitSystem(messages []ai.Message) (string, []chatMsg) {
	var system string
	out := make([]chatMsg, 0, len(messages))
	for _, m := range messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role == "system" {
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
			continue
		}
		if role != "user" && role != "assistant" {
			continue
		}
		out = append(out, chatMsg{Role: role, Content: m.Content})
	}
	return system, out
}
