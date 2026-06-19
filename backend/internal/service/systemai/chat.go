package systemai

import (
	"bytes"
	"context"
	"encoding/json"
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
	Usage *ChatUsage `json:"usage,omitempty"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// ChatUsage captures token consumption from an LLM response.
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
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
		MaxTokens:   8192,
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



// ChatResult bundles the response text with token usage.
type ChatResult struct {
	Content string
	Usage   *ChatUsage
}

func (s *Service) ChatCompletion(
	ctx context.Context,
	userID uuid.UUID,
	messages []ChatMessage,
	modelHint string,
) (string, error) {
	result, err := s.ChatCompletionWithUsage(ctx, userID, messages, modelHint)
	if err != nil {
		return "", err
	}
	return result.Content, nil
}

// ChatCompletionWithUsage is like ChatCompletion but also returns token usage.
func (s *Service) ChatCompletionWithUsage(
	ctx context.Context,
	userID uuid.UUID,
	messages []ChatMessage,
	modelHint string,
) (*ChatResult, error) {
	// Pre-check wallet balance before making any API call.
	if s.walletChecker != nil {
		if err := s.walletChecker(ctx, userID); err != nil {
			return nil, err
		}
	}

	providers, err := s.resolveAllChatProviders(ctx, userID, modelHint)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, p := range providers {
		result, usage, err := s.tryChatCompletion(ctx, p, messages)
		if err == nil {
			return &ChatResult{Content: result, Usage: usage}, nil
		}
		lastErr = err
		if !isFailoverErr(err) {
			return nil, err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("AI 未配置：请在 workspace 中点击 ⚙ 进入 AI Settings")
}

// tryChatCompletion attempts a single chat completion against one provider.
// Returns content and usage.
func (s *Service) tryChatCompletion(ctx context.Context, p chatProvider, messages []ChatMessage) (string, *ChatUsage, error) {
	endpoint := chatEndpoint(p.providerID, p.baseURL)
	httpReq, err := doChatRequest(p.model, messages, false, endpoint, p.secret)
	if err != nil {
		return "", nil, err
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
		if isTransientChatErr(doErr) {
		recordProviderFailure(p.userID, p.providerID)
	}
	return "", nil, &failoverErr{msg: fmt.Sprintf("chat completion http: %v", doErr), transient: isTransientChatErr(doErr)}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		errBody := readAPIErrorBody(resp)
		transient := isFailoverStatus(resp.StatusCode)
		// 400 with model-not-found is safe to failover; auth errors are not.
		if resp.StatusCode == 400 && isAuthErrorBody(errBody) {
			transient = false
		}
		msg := fmt.Sprintf("chat completion: status %d", resp.StatusCode)
		if errBody != "" {
			msg += " (" + errBody + ")"
		}
		if transient {
			recordProviderFailure(p.userID, p.providerID)
		}
		return "", nil, &failoverErr{msg: msg, transient: transient}
	}

	var cr ChatCompletionResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&cr); err != nil {
		return "", nil, fmt.Errorf("decode chat response: %w", err)
	}
	if cr.Error != nil {
		recordProviderFailure(p.userID, p.providerID)
	return "", nil, &failoverErr{msg: fmt.Sprintf("chat completion api error: %s", cr.Error.Message), transient: true}
	}
	if len(cr.Choices) == 0 {
		return "", nil, fmt.Errorf("chat completion returned no choices")
	}
	recordProviderSuccess(p.userID, p.providerID)
	// Record token usage if a recorder is configured.
	if s.tokenRecorder != nil && cr.Usage != nil {
		feature := "chat"
		if v := ctx.Value(aiFeatureKey{}); v != nil {
			feature = v.(string)
		}
		s.tokenRecorder(ctx, TokenRecord{
			UserID: p.userID, ProviderID: p.providerID, Model: p.model,
			Feature: feature, InputTokens: cr.Usage.PromptTokens, OutputTokens: cr.Usage.CompletionTokens,
		})
	}
	return strings.TrimSpace(cr.Choices[0].Message.Content), cr.Usage, nil
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
