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
// Supports both text-only messages and tool-calling messages (OpenAI tool_use protocol).
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`  // assistant messages with tool calls
	ToolCallID string     `json:"tool_call_id,omitempty"` // tool result messages
	Name       string     `json:"name,omitempty"`         // tool result messages (tool name)
}

// ToolCall represents a single tool call in an assistant message (OpenAI format).
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction holds the function name and JSON-encoded arguments.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON-encoded string
}

// ToolDefinition describes a tool available to the LLM (OpenAI function calling format).
type ToolDefinition struct {
	Type     string         `json:"type"` // "function"
	Function ToolDefFunction `json:"function"`
}

// ToolDefFunction describes a function's name, description, and JSON Schema parameters.
type ToolDefFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ChatCompletionRequest mirrors the OpenAI /v1/chat/completions request shape.
type ChatCompletionRequest struct {
	Model       string           `json:"model"`
	Messages    []ChatMessage    `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	Stream      bool             `json:"stream"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	ToolChoice  string           `json:"tool_choice,omitempty"` // "auto", "none", or specific tool
}
// ChatCompletionResponse mirrors the OpenAI /v1/chat/completions response shape (non-streaming).
type ChatCompletionResponse struct {
	Choices []struct {
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason,omitempty"`
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
	Content      string
	Done         bool
	FinishReason string            // "stop", "tool_calls", "length", etc.
	ToolCalls    []StreamToolCall  // accumulated tool calls from streaming deltas
}

// StreamToolCall is a tool call delta as it arrives in a streaming response.
// OpenAI streams tool calls incrementally: first chunk has index+id+function.name,
// subsequent chunks append to function.arguments. The stream parser accumulates these.
type StreamToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"` // "function"
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
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

const defaultMaxTokens = 8192

// doChatRequest builds the HTTP request body and creates an authenticated request.
// tools may be nil when the caller does not need tool calling.
// maxTokens 0 means use defaultMaxTokens.
func doChatRequest(model string, messages []ChatMessage, tools []ToolDefinition, stream bool, endpoint, secret string, maxTokens int) (*http.Request, error) {
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	reqBody := ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: 0.3,
		Stream:      stream,
		Tools:       tools,
	}
	if len(tools) > 0 {
		reqBody.ToolChoice = "auto"
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
) (string, error) {
	result, err := s.ChatCompletionWithUsage(ctx, userID, messages)
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
) (*ChatResult, error) {
	// Pre-check wallet balance before making any API call.
	if s.walletChecker != nil {
		if err := s.walletChecker(ctx, userID); err != nil {
			return nil, err
		}
	}

	providers, err := s.resolveAllChatProviders(ctx, userID)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, p := range providers {
		result, _, usage, err := s.tryChatCompletion(ctx, p, messages, nil)
		if err == nil {
				// Bill before returning — user must pay to receive the result.
				if s.postCallBiller != nil && usage != nil {
					if billErr := s.postCallBiller(ctx, userID, p.providerID, p.model, usage.PromptTokens, usage.CompletionTokens); billErr != nil {
						return nil, billErr
					}
				}
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
// Retries once on transient network errors AND once on transient HTTP statuses
// (429/5xx) before giving up — so a single-provider user isn't immediately failed.
func (s *Service) tryChatCompletion(ctx context.Context, p chatProvider, messages []ChatMessage, tools []ToolDefinition) (string, []ToolCall, *ChatUsage, error) {
	endpoint := chatEndpoint(p.providerID, p.baseURL)
	httpReq, err := doChatRequest(p.model, messages, tools, false, endpoint, p.secret, p.maxTokens)
	if err != nil {
			return "", nil, nil, err
	}
	httpReq = httpReq.WithContext(ctx)
	client := &http.Client{Timeout: chatTimeout}

	const maxAttempts = 2
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second) // 0s, 1s backoff
		}
		resp, doErr := client.Do(httpReq)
		if doErr != nil {
			if !isTransientChatErr(doErr) || attempt == maxAttempts-1 {
				if isTransientChatErr(doErr) {
					s.recordProviderFailure(ctx, p.userID, p.providerID)
				}
					return "", nil, nil, &failoverErr{msg: fmt.Sprintf("chat completion http: %v", doErr), transient: isTransientChatErr(doErr)}
			}
			continue
		}
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var cr ChatCompletionResponse
			if err := json.Unmarshal(bodyBytes, &cr); err != nil {
					return "", nil, nil, fmt.Errorf("decode chat response: %w", err)
			}
			if cr.Error != nil {
				s.recordProviderFailure(ctx, p.userID, p.providerID)
					return "", nil, nil, &failoverErr{msg: fmt.Sprintf("chat completion api error: %s", cr.Error.Message), transient: true}
			}
			if len(cr.Choices) == 0 {
				return "", nil, nil, fmt.Errorf("chat completion: empty choices")
			}
			s.recordProviderSuccess(ctx, p.userID, p.providerID)
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
			return strings.TrimSpace(cr.Choices[0].Message.Content), cr.Choices[0].Message.ToolCalls, cr.Usage, nil
		}

		// Non-2xx: parse error, decide whether to retry or fail.
		ae := readAPIErrorBodyFromBytes(bodyBytes)
		transient := isFailoverStatus(resp.StatusCode)
		if resp.StatusCode == 400 && isAuthErrorBody(ae.Raw) {
			transient = false
		}
		if !transient || attempt == maxAttempts-1 {
			msg := fmt.Sprintf("[%s] chat completion: status %d", ae.Type, resp.StatusCode)
			if ae.Message != "" {
				msg += " (" + ae.Message + ")"
			} else if ae.Raw != "" {
				msg += " (" + ae.Raw + ")"
			}
			if transient {
				s.recordProviderFailure(ctx, p.userID, p.providerID)
			}
			return "", nil, nil, &failoverErr{msg: msg, transient: transient}
		}
		// Transient status — retry after backoff.
	}
	return "", nil, nil, fmt.Errorf("chat completion: exhausted retries")
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
