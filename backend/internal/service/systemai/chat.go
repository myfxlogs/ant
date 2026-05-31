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

// ChatCompletionStream sends a streaming chat completion request (stream: true)
// and calls onChunk for each delta as it arrives from the provider's SSE stream.
// It returns when the stream ends or an unrecoverable error occurs.
// The caller is responsible for context cancellation — cancelling ctx aborts the
// underlying HTTP request and causes the next read to fail.
func (s *Service) ChatCompletionStream(ctx context.Context, userID uuid.UUID, systemPrompt, userMessage, modelHint string, onChunk func(chunk ChatStreamChunk) error) error {
	_, model, baseURL, secret, err := s.resolveChatProvider(ctx, userID, modelHint)
	if err != nil {
		return err
	}

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	reqBody := ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   4096,
		Temperature: 0.3,
		Stream:      true,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal chat request: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create chat request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	authHeader(httpReq, secret)
	httpReq.Header.Set("Cache-Control", "no-cache")

	// streaming has unbounded duration — no client timeout
	client := &http.Client{Timeout: 0}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("chat completion stream http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chat completion stream: status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	// Most SSE chunks are small; bump the initial buffer so we don't
	// reallocate on every line.
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
			continue // skip unparseable keepalive/comment lines
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

// ChatCompletion sends a chat completion request to the user's configured LLM provider.
// It picks the first provider that has a configured API secret and a known model,
// falling back to the given modelHint if the provider has no model preference set.
func (s *Service) ChatCompletion(ctx context.Context, userID uuid.UUID, systemPrompt, userMessage, modelHint string) (string, error) {
	_, model, baseURL, secret, err := s.resolveChatProvider(ctx, userID, modelHint)
	if err != nil {
		return "", err
	}

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userMessage},
	}

	reqBody := ChatCompletionRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   4096,
		Temperature: 0.3,
		Stream:      false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal chat request: %w", err)
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create chat request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	authHeader(httpReq, secret)

	client := &http.Client{Timeout: chatTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("chat completion http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// Sanitize: do NOT include raw response body in the error – it may
		// contain the API key echoed by the provider.
		return "", fmt.Errorf("chat completion: status %d", resp.StatusCode)
	}

	var cr ChatCompletionResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&cr); err != nil {
		return "", fmt.Errorf("decode chat response: %w", err)
	}
	if cr.Error != nil {
		return "", fmt.Errorf("chat completion api error: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("chat completion returned no choices")
	}
	return strings.TrimSpace(cr.Choices[0].Message.Content), nil
}

// resolveChatProvider picks a provider, model, base URL, and secret for the given user.
// It iterates configured providers and picks the first one that has a secret set.
func (s *Service) resolveChatProvider(ctx context.Context, userID uuid.UUID, modelHint string) (providerID, model, baseURL, secret string, err error) {
	rows, err := s.List(ctx, userID)
	if err != nil {
		return "", "", "", "", fmt.Errorf("list AI providers: %w", err)
	}

	for _, row := range rows {
		if row == nil || !row.Enabled {
			continue
		}
		sec, secErr := s.GetSecret(ctx, userID, row.ProviderID)
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
	return "", "", "", "", fmt.Errorf("no configured AI provider with a valid API key and model")
}
