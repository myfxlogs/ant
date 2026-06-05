package systemai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"github.com/google/uuid"
)

const chatTimeout = 60 * time.Second
const secretCacheTTL = 30 * time.Minute
// ChatMessage is a single message in a chat completion request.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest mirrors the OpenAI /v1/chat/completions request shape.
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
}
// ChatCompletionResponse mirrors the OpenAI /v1/chat/completions response shape (non-streaming).
type ChatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// ChatStreamChunk represents a single delta from a streaming chat completion.
type ChatStreamChunk struct {
	Content string
	Done    bool
}
// buildChatMessages builds messages with system + history + user.
func BuildChatMessages(systemPrompt, userMessage string, history []ChatMessage) []ChatMessage {
	msgs := make([]ChatMessage, 0, 2+len(history))
	msgs = append(msgs, ChatMessage{Role: "system", Content: systemPrompt})
	for _, h := range history {
		msgs = append(msgs, h)
	}
	msgs = append(msgs, ChatMessage{Role: "user", Content: userMessage})
	return msgs
}

// getCachedSecret returns a decrypted secret from the in-memory cache,
func (s *Service) getCachedSecret(ctx context.Context, userID uuid.UUID, providerID string) (string, error) {
	key := userID.String() + "|" + providerID
	if v, ok := s.secretCache.Load(key); ok {
		entry := v.(secretCacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.secret, nil
		}
	}
	secret, err := s.GetSecret(ctx, userID, providerID)
	if err != nil {
		return "", err
	}
	s.secretCache.Store(key, secretCacheEntry{secret: secret, expiresAt: time.Now().Add(secretCacheTTL)})
	return secret, nil
}

// doChatRequest builds the HTTP request body and creates an authenticated request.
func doChatRequest(model string, messages []ChatMessage, stream bool, endpoint, secret string) (*http.Request, error) {
	reqBody := ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   4096,
		Temperature: 0.3,
		Stream:      stream,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create chat request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	authHeader(httpReq, secret)
	if stream {
		httpReq.Header.Set("Accept", "text/event-stream")
		httpReq.Header.Set("Cache-Control", "no-cache")
	}
	return httpReq, nil
}

func (s *Service) ChatCompletionStream(
	ctx context.Context,
	userID uuid.UUID,
	messages []ChatMessage,
	modelHint string,
	onChunk func(chunk ChatStreamChunk) error,
) error {
	providers, err := s.resolveAllChatProviders(ctx, userID, modelHint)
	if err != nil {
		return err
	}

	var lastErr error
	for _, p := range providers {
		err := s.tryChatCompletionStream(ctx, p, messages, onChunk)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isFailoverErr(err) {
			return err
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("AI 未配置")
}

func (s *Service) tryChatCompletionStream(ctx context.Context, p chatProvider, messages []ChatMessage, onChunk func(chunk ChatStreamChunk) error) error {
	endpoint := chatEndpoint(p.providerID, p.baseURL)
	httpReq, err := doChatRequest(p.model, messages, true, endpoint, p.secret)
	if err != nil {
		return err
	}
	httpReq = httpReq.WithContext(ctx)
	client := &http.Client{Timeout: 0} 
	resp, err := client.Do(httpReq)
	if err != nil {
		return &failoverErr{msg: fmt.Sprintf("chat completion stream http: %v", err), transient: isTransientChatErr(err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &failoverErr{msg: fmt.Sprintf("chat completion stream: status %d", resp.StatusCode), transient: isFailoverStatus(resp.StatusCode)}
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var streamResp struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue
		}
		if len(streamResp.Choices) == 0 {
			continue
		}
		c := streamResp.Choices[0]
		done := c.FinishReason != nil && *c.FinishReason != "" && *c.FinishReason != "null"
		if err := onChunk(ChatStreamChunk{Content: c.Delta.Content, Done: done}); err != nil {
			return err
		}
		if done {
			break
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("read stream: %w", err)
	}
	return nil
}


func (s *Service) ChatCompletion(
	ctx context.Context,
	userID uuid.UUID,
	messages []ChatMessage,
	modelHint string,
) (string, error) {
	providers, err := s.resolveAllChatProviders(ctx, userID, modelHint)
	if err != nil {
		return "", err
	}

	var lastErr error
	for _, p := range providers {
		result, err := s.tryChatCompletion(ctx, p, messages)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !isFailoverErr(err) {
			return "", err // non-transient: don't try more providers
		}
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("AI 未配置：请在 workspace 中点击 ⚙ 进入 AI Settings")
}

// tryChatCompletion attempts a single chat completion against one provider.
func (s *Service) tryChatCompletion(ctx context.Context, p chatProvider, messages []ChatMessage) (string, error) {
	endpoint := chatEndpoint(p.providerID, p.baseURL)
	httpReq, err := doChatRequest(p.model, messages, false, endpoint, p.secret)
	if err != nil {
		return "", err
	}
	httpReq = httpReq.WithContext(ctx)

	client := &http.Client{Timeout: chatTimeout}
	var resp *http.Response
	var doErr error
	for attempt := 0; attempt <= 1; attempt++ {
		if attempt > 0 {
			time.Sleep(1 * time.Second)
		}
		resp, doErr = client.Do(httpReq)
		if doErr == nil {
			break
		}
		if !isTransientChatErr(doErr) {
			break
		}
	}
	if doErr != nil {
		return "", &failoverErr{msg: fmt.Sprintf("chat completion http: %v", doErr), transient: isTransientChatErr(doErr)}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", &failoverErr{msg: fmt.Sprintf("chat completion: status %d", resp.StatusCode), transient: isFailoverStatus(resp.StatusCode)}
	}

	var cr ChatCompletionResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&cr); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}
	if cr.Error != nil {
		return "", &failoverErr{msg: fmt.Sprintf("chat completion api error: %s", cr.Error.Message), transient: true}
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("chat completion returned no choices")
	}
	return strings.TrimSpace(cr.Choices[0].Message.Content), nil
}

// It prefers the provider marked as primary for "chat", then falls back to any enabled provider.
func (s *Service) resolveChatProvider(ctx context.Context, userID uuid.UUID, modelHint string) (providerID, model, baseURL, secret string, err error) {
	rows, err := s.List(ctx, userID)
	if err != nil {
		return "", "", "", "", fmt.Errorf("list AI providers: %w", err)
	}
	for _, preferPrimary := range []bool{true, false} {
		for _, row := range rows {
			if row == nil || !row.Enabled {
				continue
			}
			if preferPrimary && !hasPrimaryChat(row.PrimaryFor) {
				continue
			}
			sec, secErr := s.getCachedSecret(ctx, userID, row.ProviderID)
			if secErr != nil || sec == "" {
				continue
			}
			base := strings.TrimRight(strings.TrimSpace(row.BaseURL), "/")
			if base == "" {
				continue
			}
			m := strings.TrimSpace(row.DefaultModel)
			if m == "" {
				m = modelHint
			}
			if m == "" {
				continue
			}
			return row.ProviderID, m, base, sec, nil
		}
	}
	return "", "", "", "", fmt.Errorf("AI 未配置：请在 workspace 中点击 ⚙ 进入 AI Settings，选择一个厂商（如 DeepSeek）填写 API Key 和模型名称后启用")
}

func hasPrimaryChat(primaryFor []string) bool {
	for _, p := range primaryFor {
		if p == "chat" {
			return true
		}
	}
	return false
}
func isTransientChatErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "reset by peer")
}
