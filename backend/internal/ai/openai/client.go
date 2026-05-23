package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"anttrader/internal/ai"
)

const (
	defaultBaseURL = "https://api.openai.com"
	defaultTimeout = 300 * time.Second
	providerName   = "custom"
)

type Client struct {
	apiKey      string
	model       string
	baseURL     string
	httpClient  *http.Client
	temperature *float64 // nil => omit field, use provider default
	maxTokens   *int     // nil => omit field, use provider default
}

// SetSamplingParams configures temperature/max_tokens. Pass nil to fall back
// to the provider's default (the field is omitted from the request body).
func (c *Client) SetSamplingParams(temperature *float64, maxTokens *int) {
	if c == nil {
		return
	}
	c.temperature = temperature
	c.maxTokens = maxTokens
}

// SetTimeout overrides the per-request HTTP timeout.
func (c *Client) SetTimeout(d time.Duration) {
	if c == nil || d <= 0 || c.httpClient == nil {
		return
	}
	c.httpClient.Timeout = d
}

func NewClient(apiKey, model, baseURL string) *Client {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		apiKey:  strings.TrimSpace(apiKey),
		model:   strings.TrimSpace(model),
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

type chatRequest struct {
	Model       string       `json:"model"`
	Messages    []ai.Message `json:"messages"`
	Stream      bool         `json:"stream"`
	Temperature *float64     `json:"temperature,omitempty"`
	MaxTokens   *int         `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage ai.Usage `json:"usage"`
}

type streamResponse struct {
	Choices []struct {
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func (c *Client) ValidateConfig() error {
	if strings.TrimSpace(c.apiKey) == "" {
		return ai.ErrInvalidAPIKey
	}
	if strings.TrimSpace(c.model) == "" {
		return fmt.Errorf("model name is required")
	}
	if strings.TrimSpace(c.baseURL) == "" {
		return fmt.Errorf("base_url is required")
	}
	// Validate base_url format
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base_url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("base_url must start with http:// or https://")
	}
	if strings.HasSuffix(strings.ToLower(u.Path), "/chat/completions") || strings.HasSuffix(strings.ToLower(u.Path), "/chat/completions/") {
		return fmt.Errorf("base_url should not end with /chat/completions")
	}
	// Model name basic validation
	if len(c.model) > 80 {
		return fmt.Errorf("model name too long")
	}
	// 允许的字符集来源于真实 provider 的 model id 实践：
	//   openai 等用 [a-zA-Z0-9._-]
	//   anthropic / siliconflow 用 ":" 分变体
	//   openrouter / dashscope / 自建网关 用 "/" 路由命名空间，例如 "kimi/kimi-k2.6"
	for _, ch := range c.model {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') ||
			ch == '.' || ch == '_' || ch == '-' || ch == ':' || ch == '/' {
			continue
		}
		return fmt.Errorf("invalid model name")
	}
	return nil
}

func (c *Client) GetModelName() string {
	return c.model
}

func (c *Client) GetProviderName() string {
	return providerName
}

func (c *Client) Chat(ctx context.Context, messages []ai.Message) (*ai.Response, error) {
	if err := c.ValidateConfig(); err != nil {
		return nil, err
	}

	endpoint := c.baseURL + "/chat/completions"
	reqBody := chatRequest{Model: c.model, Messages: messages, Stream: false, Temperature: c.temperature, MaxTokens: c.maxTokens}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var out chatResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	content := out.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("%w: empty content", ai.ErrInvalidResponse)
	}

	return &ai.Response{
		Content: content,
		Usage:   out.Usage,
	}, nil
}

func (c *Client) StreamChat(ctx context.Context, messages []ai.Message) (<-chan ai.StreamChunk, error) {
	if err := c.ValidateConfig(); err != nil {
		return nil, err
	}

	endpoint := c.baseURL + "/chat/completions"
	reqBody := chatRequest{Model: c.model, Messages: messages, Stream: true}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	chunkChan := make(chan ai.StreamChunk, 100)
	go func() {
		defer close(chunkChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				chunkChan <- ai.StreamChunk{Done: true}
				return
			}

			var sresp streamResponse
			if err := json.Unmarshal([]byte(data), &sresp); err != nil {
				chunkChan <- ai.StreamChunk{Error: fmt.Errorf("failed to parse stream data: %v", err)}
				return
			}
			if len(sresp.Choices) == 0 {
				continue
			}

			delta := sresp.Choices[0].Delta
			finish := sresp.Choices[0].FinishReason
			done := finish != nil && *finish == "stop"

			chunkChan <- ai.StreamChunk{Content: delta.Content, Done: done}
			if done {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			chunkChan <- ai.StreamChunk{Error: fmt.Errorf("stream read error: %v", err)}
		}
	}()

	return chunkChan, nil
}
