package zhipu

import (
	"bufio"
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
	// API地址
	apiURL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	// 默认模型
	defaultModel = "glm-4-flash"
	// 提供商名称
	providerName = "zhipu"
	// HTTP超时时间
	defaultTimeout = 300 * time.Second
)

// zhipuThinking 对应智谱「深度思考」开关。智谱多系列默认会把大量 completion 预算
// 用在 reasoning_tokens 上，易导致可见正文为空；本客户端对所有模型强制关闭思考。
type zhipuThinking struct {
	Type string `json:"type"` // disabled | enabled
}

func thinkingDisabled() *zhipuThinking {
	return &zhipuThinking{Type: "disabled"}
}

// Client 智谱AI客户端
type Client struct {
	apiKey      string
	model       string
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

// NewClient 创建智谱AI客户端
func NewClient(apiKey string, model string) *Client {
	if model == "" {
		model = defaultModel
	}

	return &Client{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// NewClientWithHTTPClient 使用自定义HTTP客户端创建智谱AI客户端
func NewClientWithHTTPClient(apiKey string, model string, httpClient *http.Client) *Client {
	if model == "" {
		model = defaultModel
	}

	return &Client{
		apiKey:     apiKey,
		model:      model,
		httpClient: httpClient,
	}
}

// Chat 同步对话
func (c *Client) Chat(ctx context.Context, messages []ai.Message) (*ai.Response, error) {
	req := &chatRequest{
		Model:       c.model,
		Messages:    messages,
		Stream:      false,
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
		Thinking:    thinkingDisabled(),
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status code %d, body: %s", ai.ErrRequestFailed, resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response: %v", ai.ErrInvalidResponse, err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("%w: no choices in response", ai.ErrInvalidResponse)
	}
	content := chatResp.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("%w: empty content", ai.ErrInvalidResponse)
	}

	return &ai.Response{
		Content: content,
		Usage: ai.Usage{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		},
	}, nil
}

// StreamChat 流式对话
func (c *Client) StreamChat(ctx context.Context, messages []ai.Message) (<-chan ai.StreamChunk, error) {
	req := &chatRequest{
		Model:       c.model,
		Messages:    messages,
		Stream:      true,
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
		Thinking:    thinkingDisabled(),
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("%w: status code %d, body: %s", ai.ErrRequestFailed, resp.StatusCode, string(body))
	}

	chunkChan := make(chan ai.StreamChunk, 100)

	go func() {
		defer close(chunkChan)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// SSE格式: data: {...}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			// 流结束标记
			if data == "[DONE]" {
				chunkChan <- ai.StreamChunk{
					Content: "",
					Done:    true,
				}
				return
			}

			var streamResp streamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				chunkChan <- ai.StreamChunk{
					Content: "",
					Done:    false,
					Error:   fmt.Errorf("%w: failed to parse stream data: %v", ai.ErrInvalidResponse, err),
				}
				return
			}

			if len(streamResp.Choices) == 0 {
				continue
			}

			delta := streamResp.Choices[0].Delta
			finishReason := streamResp.Choices[0].FinishReason

			chunkChan <- ai.StreamChunk{
				Content: delta.Content,
				Done:    finishReason == "stop",
			}

			if finishReason == "stop" {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			chunkChan <- ai.StreamChunk{
				Content: "",
				Done:    false,
				Error:   fmt.Errorf("stream read error: %v", err),
			}
		}
	}()

	return chunkChan, nil
}

// ValidateConfig 验证配置
func (c *Client) ValidateConfig() error {
	if c.apiKey == "" {
		return ai.ErrInvalidAPIKey
	}
	if c.model == "" {
		return fmt.Errorf("model name is required")
	}
	return nil
}

// GetModelName 获取模型名称
func (c *Client) GetModelName() string {
	return c.model
}

// GetProviderName 获取提供商名称
func (c *Client) GetProviderName() string {
	return providerName
}

// doRequest 执行HTTP请求
func (c *Client) doRequest(ctx context.Context, req *chatRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	return c.httpClient.Do(httpReq)
}

// chatRequest 聊天请求结构
type chatRequest struct {
	Model       string          `json:"model"`
	Messages    []ai.Message    `json:"messages"`
	Stream      bool            `json:"stream"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	Thinking    *zhipuThinking  `json:"thinking,omitempty"`
}

// chatResponse 聊天响应结构
type chatResponse struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int        `json:"index"`
		Message      ai.Message `json:"message"`
		FinishReason string     `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// streamResponse 流式响应结构
type streamResponse struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}
