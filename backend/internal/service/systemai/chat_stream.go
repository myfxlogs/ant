package systemai

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (s *Service) ChatCompletionStream(
	ctx context.Context,
	userID uuid.UUID,
	messages []ChatMessage,
	onChunk func(chunk ChatStreamChunk) error,
) error {
	// Pre-check wallet balance before making any API call.
	if s.walletChecker != nil {
		if err := s.walletChecker(ctx, userID); err != nil {
			return err
		}
	}

	providers, err := s.resolveAllChatProviders(ctx, userID)
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
		if isTransientChatErr(err) {
			s.recordProviderFailure(ctx, p.userID, p.providerID)
		}
		return &failoverErr{msg: fmt.Sprintf("chat completion stream http: %v", err), transient: isTransientChatErr(err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ae := readAPIErrorBody(resp)
		transient := isFailoverStatus(resp.StatusCode)
		if resp.StatusCode == 400 && isAuthErrorBody(ae.Raw) {
			transient = false
		}
		msg := fmt.Sprintf("[%s] chat completion stream: status %d", ae.Type, resp.StatusCode)
		if ae.Message != "" {
			msg += " (" + ae.Message + ")"
		} else if ae.Raw != "" {
			msg += " (" + ae.Raw + ")"
		}
		// PRIORITY: fallbackNonStream fires BEFORE failover for 400 errors.
		//  1. 400 + streaming → fallbackNonStream (same provider, no streaming)
		//  2. fallbackNonStream fails → failover to next provider (if transient)
		//  3. non-transient → stop, return FriendlyError
		// Only auth errors skip the fallback (retrying non-streaming won't fix a bad key).
		if resp.StatusCode == 400 && !isAuthErrorBody(ae.Raw) {
			return s.fallbackNonStream(ctx, p, messages, onChunk)
		}
		if transient {
			s.recordProviderFailure(ctx, p.userID, p.providerID)
		}
		return &failoverErr{msg: msg, transient: transient}
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	var streamUsage *ChatUsage
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *ChatUsage `json:"usage,omitempty"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Usage != nil {
			streamUsage = chunk.Usage
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		c := chunk.Choices[0]
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
	// Record token usage from streaming response.
	if s.tokenRecorder != nil && streamUsage != nil {
		feature := "chat"
		if v := ctx.Value(aiFeatureKey{}); v != nil {
			feature = v.(string)
		}
		s.tokenRecorder(ctx, TokenRecord{
			UserID: p.userID, ProviderID: p.providerID, Model: p.model,
			Feature: feature, InputTokens: streamUsage.PromptTokens, OutputTokens: streamUsage.CompletionTokens,
		})
	}
	return nil
}

// fallbackNonStream retries the chat completion on the same provider with
// streaming disabled, then delivers the full content as a single chunk.
// Used when the provider returns 400 "streaming not supported".
func (s *Service) fallbackNonStream(ctx context.Context, p chatProvider, messages []ChatMessage, onChunk func(chunk ChatStreamChunk) error) error {
	result, usage, err := s.tryChatCompletion(ctx, p, messages)
	if err != nil {
		return err
	}
	// Deliver as a single chunk (frontend stream consumers handle this).
	if err := onChunk(ChatStreamChunk{Content: result, Done: true}); err != nil {
		return err
	}
	// Record token usage from the non-streaming fallback.
	if s.tokenRecorder != nil && usage != nil {
		feature := "chat"
		if v := ctx.Value(aiFeatureKey{}); v != nil {
			feature = v.(string)
		}
		s.tokenRecorder(ctx, TokenRecord{
			UserID: p.userID, ProviderID: p.providerID, Model: p.model,
			Feature: feature, InputTokens: usage.PromptTokens, OutputTokens: usage.CompletionTokens,
		})
	}
	return nil
}
