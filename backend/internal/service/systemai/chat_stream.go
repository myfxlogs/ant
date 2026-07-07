package systemai

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/google/uuid"
)

func (s *Service) ChatCompletionStream(
	ctx context.Context,
	userID uuid.UUID,
	messages []ChatMessage,
	onChunk func(chunk ChatStreamChunk) error,
) error {
	return s.chatCompletionStream(ctx, userID, messages, nil, onChunk)
}

// ChatCompletionStreamWithTools is like ChatCompletionStream but passes tool
// definitions to the LLM so it can request tool calls via the native tool_use
// protocol (OpenAI function calling).
func (s *Service) ChatCompletionStreamWithTools(
	ctx context.Context,
	userID uuid.UUID,
	messages []ChatMessage,
	tools []ToolDefinition,
	onChunk func(chunk ChatStreamChunk) error,
) error {
	return s.chatCompletionStream(ctx, userID, messages, tools, onChunk)
}

func (s *Service) chatCompletionStream(
	ctx context.Context,
	userID uuid.UUID,
	messages []ChatMessage,
	tools []ToolDefinition,
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
		err := s.tryChatCompletionStream(ctx, p, messages, tools, onChunk)
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

func (s *Service) tryChatCompletionStream(ctx context.Context, p chatProvider, messages []ChatMessage, tools []ToolDefinition, onChunk func(chunk ChatStreamChunk) error) error {
	endpoint := chatEndpoint(p.providerID, p.baseURL)
	httpReq, err := doChatRequest(p.model, messages, tools, true, endpoint, p.secret, p.maxTokens)
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
		if resp.StatusCode == 400 && !isAuthErrorBody(ae.Raw) {
			return s.fallbackNonStream(ctx, p, messages, tools, onChunk)
		}
		if transient {
			s.recordProviderFailure(ctx, p.userID, p.providerID)
		}
		return &failoverErr{msg: msg, transient: transient}
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	var streamUsage *ChatUsage
	// Accumulate streaming tool_call deltas keyed by index.
	toolCallAcc := make(map[int]*StreamToolCall)
	totalContentLen := 0

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
					Content   string           `json:"content"`
					ToolCalls []StreamToolCall `json:"tool_calls"`
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
		if c.Delta.Content != "" {
			totalContentLen += len(c.Delta.Content)
		}

		// Accumulate tool_call deltas (OpenAI streams them incrementally).
		for _, tc := range c.Delta.ToolCalls {
			acc, ok := toolCallAcc[tc.Index]
			if !ok {
				acc = &StreamToolCall{Index: tc.Index}
				toolCallAcc[tc.Index] = acc
			}
			if tc.ID != "" {
				acc.ID = tc.ID
			}
			if tc.Type != "" {
				acc.Type = tc.Type
			}
			if tc.Function.Name != "" {
				acc.Function.Name = tc.Function.Name
			}
			acc.Function.Arguments += tc.Function.Arguments
		}

		finishReason := ""
		if c.FinishReason != nil && *c.FinishReason != "" && *c.FinishReason != "null" {
			finishReason = *c.FinishReason
		}

		// Only emit tool calls on the final chunk (finish_reason set).
		var finalToolCalls []StreamToolCall
		if finishReason != "" && len(toolCallAcc) > 0 {
			// Sort by index for deterministic order.
			idxs := make([]int, 0, len(toolCallAcc))
			for idx := range toolCallAcc {
				idxs = append(idxs, idx)
			}
			sort.Ints(idxs)
			for _, idx := range idxs {
				finalToolCalls = append(finalToolCalls, *toolCallAcc[idx])
			}
		}

		if err := onChunk(ChatStreamChunk{
			Content:      c.Delta.Content,
			Done:         finishReason != "",
			FinishReason: finishReason,
			ToolCalls:    finalToolCalls,
		}); err != nil {
			return err
		}

		if finishReason != "" {
			break
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("read stream: %w", err)
	}
	// Empty response from provider: trigger failover to next provider.
	if totalContentLen == 0 && len(toolCallAcc) == 0 {
		return &failoverErr{msg: fmt.Sprintf("[%s] chat stream returned empty response", p.providerID), transient: true}
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
func (s *Service) fallbackNonStream(ctx context.Context, p chatProvider, messages []ChatMessage, tools []ToolDefinition, onChunk func(chunk ChatStreamChunk) error) error {
	result, toolCalls, usage, err := s.tryChatCompletion(ctx, p, messages, tools)
	if err != nil {
		return err
	}
	// Convert non-streaming ToolCalls to StreamToolCall format for the chunk.
	var streamToolCalls []StreamToolCall
	for _, tc := range toolCalls {
		stc := StreamToolCall{ID: tc.ID, Type: tc.Type}
		stc.Function.Name = tc.Function.Name
		stc.Function.Arguments = tc.Function.Arguments
		streamToolCalls = append(streamToolCalls, stc)
	}
	finishReason := "stop"
	if len(streamToolCalls) > 0 {
		finishReason = "tool_calls"
	}
	// Deliver as a single chunk (frontend stream consumers handle this).
	if err := onChunk(ChatStreamChunk{Content: result, Done: true, FinishReason: finishReason, ToolCalls: streamToolCalls}); err != nil {
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
