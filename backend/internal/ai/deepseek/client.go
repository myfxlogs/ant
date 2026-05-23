package deepseek

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
	// APIEndpoint DeepSeek API地址
	APIEndpoint = "https://api.deepseek.com/v1/chat/completions"
	// ModelName DeepSeek模型名称
	ModelName = "deepseek-chat"
	// ProviderName 提供商名称
	ProviderName = "deepseek"
	// DefaultTimeout 默认超时时间
	DefaultTimeout = 300 * time.Second
)

// Client DeepSeek客户端
type Client struct {
	apiKey      string
	httpClient  *http.Client
	model       string
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

// NewClient 创建DeepSeek客户端
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		model: ModelName,
	}
}

// NewClientWithModel 创建指定模型的DeepSeek客户端
func NewClientWithModel(apiKey, model string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		model: model,
	}
}

// ChatRequest 请求结构
type ChatRequest struct {
	Model       string       `json:"model"`
	Messages    []ai.Message `json:"messages"`
	Stream      bool         `json:"stream"`
	Temperature *float64     `json:"temperature,omitempty"`
	MaxTokens   *int         `json:"max_tokens,omitempty"`
}

// ChatResponse 响应结构
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage ai.Usage `json:"usage"`
}

// StreamResponse 流式响应结构
type StreamResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// Chat 同步对话
func (c *Client) Chat(ctx context.Context, messages []ai.Message) (*ai.Response, error) {
	// 构建请求
	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Stream:      false,
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", APIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 检查响应内容
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	content := chatResp.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("%w: empty content", ai.ErrInvalidResponse)
	}

	return &ai.Response{
		Content: content,
		Usage:   chatResp.Usage,
	}, nil
}

// StreamChat 流式对话
func (c *Client) StreamChat(ctx context.Context, messages []ai.Message) (<-chan ai.StreamChunk, error) {
	// 构建请求
	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Stream:      true,
		Temperature: c.temperature,
		MaxTokens:   c.maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", APIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 创建流式响应通道
	chunkChan := make(chan ai.StreamChunk, 100)

	// 启动goroutine处理流式响应
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

			// 检查是否结束
			if data == "[DONE]" {
				chunkChan <- ai.StreamChunk{
					Done: true,
				}
				return
			}

			// 解析JSON
			var streamResp StreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				chunkChan <- ai.StreamChunk{
					Error: fmt.Errorf("failed to unmarshal stream response: %w", err),
				}
				return
			}

			// 检查响应内容
			if len(streamResp.Choices) == 0 {
				continue
			}

			// 发送内容块
			content := streamResp.Choices[0].Delta.Content
			if content != "" {
				chunkChan <- ai.StreamChunk{
					Content: content,
					Done:    false,
				}
			}

			// 检查是否完成
			if streamResp.Choices[0].FinishReason != nil {
				chunkChan <- ai.StreamChunk{
					Done: true,
				}
				return
			}
		}

		// 检查扫描错误
		if err := scanner.Err(); err != nil {
			chunkChan <- ai.StreamChunk{
				Error: fmt.Errorf("scanner error: %w", err),
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
		return fmt.Errorf("model name cannot be empty")
	}
	return nil
}

// GetModelName 获取模型名称
func (c *Client) GetModelName() string {
	return c.model
}

// GetProviderName 获取提供商名称
func (c *Client) GetProviderName() string {
	return ProviderName
}
