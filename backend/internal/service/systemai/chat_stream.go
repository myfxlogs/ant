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
	"go.uber.org/zap"
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
	// Pre-check wallet balance and quota before making any API call.
	// remainingTokens is used to cap max_tokens so the AI provider limits
	// output generation to the user's remaining quota.
	remainingTokens := -1
	if s.walletChecker != nil {
		rt, err := s.walletChecker(ctx, userID)
		if err != nil {
			return err
		}
		remainingTokens = rt
	}

	// Subtract session-in-flight tokens (from prior rounds in the same agent loop).
	if sc := sessionCounterFromCtx(ctx); sc != nil && remainingTokens >= 0 {
		used := sc.Total()
		remainingTokens -= used
		if remainingTokens < 0 {
			remainingTokens = 0
		}
	}

	// Block if no tokens remain and no wallet balance to fall back on.
	if remainingTokens == 0 {
		return ErrInsufficientBalance
	}

	providers, err := s.resolveAllChatProviders(ctx, userID)
	if err != nil {
		return err
	}

	var lastErr error
	for _, p := range providers {
		// Cap max_tokens to remaining quota so the provider limits output length.
		p.maxTokens = capMaxTokens(p.maxTokens, remainingTokens)
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
	resp, err := s.doStreamHTTPRequest(ctx, p, messages, tools)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := s.handleStreamResponse(resp, p, messages, onChunk); err != nil {
		return err
	}
	return nil
}

func (s *Service) doStreamHTTPRequest(ctx context.Context, p chatProvider, messages []ChatMessage, tools []ToolDefinition) (*http.Response, error) {
	endpoint := chatEndpoint(p.providerID, p.baseURL)
	httpReq, err := doChatRequest(ctx, p.model, messages, tools, true, endpoint, p.secret, p.maxTokens)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 0}
	resp, err := client.Do(httpReq)
	if err != nil {
		if isTransientChatErr(err) {
			s.recordProviderFailure(ctx, p.userID, p.providerID)
		}
		return nil, &failoverErr{msg: fmt.Sprintf("chat completion stream http: %v", err), transient: isTransientChatErr(err)}
	}
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
		_ = resp.Body.Close()
		if resp.StatusCode == 400 && !isAuthErrorBody(ae.Raw) {
			return nil, s.fallbackNonStream(ctx, p, messages, tools, nil)
		}
		if transient {
			s.recordProviderFailure(ctx, p.userID, p.providerID)
		}
		return nil, &failoverErr{msg: msg, transient: transient}
	}
	return resp, nil
}

func (s *Service) handleStreamResponse(resp *http.Response, p chatProvider, messages []ChatMessage, onChunk func(chunk ChatStreamChunk) error) error {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	var streamUsage *ChatUsage
	toolCallAcc := make(map[int]*StreamToolCall)
	totalContentLen := 0
	totalReasoningLen := 0
	lastFinishReason := ""

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
					Content          string           `json:"content"`
					ReasoningContent string           `json:"reasoning_content"`
					ToolCalls        []StreamToolCall `json:"tool_calls"`
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
		if c.Delta.ReasoningContent != "" {
			totalReasoningLen += len(c.Delta.ReasoningContent)
		}

		accumulateToolCallDeltas(c.Delta.ToolCalls, toolCallAcc)

		finishReason := ""
		if c.FinishReason != nil && *c.FinishReason != "" && *c.FinishReason != "null" {
			finishReason = *c.FinishReason
		}

		finalToolCalls := finalizeToolCalls(finishReason, toolCallAcc)

		if err := onChunk(ChatStreamChunk{
			Content:      c.Delta.Content,
			Reasoning:    c.Delta.ReasoningContent,
			Done:         finishReason != "",
			FinishReason: finishReason,
			ToolCalls:    finalToolCalls,
		}); err != nil {
			return err
		}

		if finishReason != "" {
			lastFinishReason = finishReason
			break
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("read stream: %w", err)
	}
	if totalContentLen == 0 && totalReasoningLen == 0 && len(toolCallAcc) == 0 {
		return &failoverErr{msg: fmt.Sprintf("[%s] chat stream empty (finish_reason=%q)", p.providerID, lastFinishReason), transient: true}
	}
	if billErr := s.billStreamPostCall(context.Background(), p, streamUsage, messages, totalContentLen); billErr != nil {
		// Content already streamed to user — cannot undo delivery.
		// Log as CRITICAL: ops must investigate and manually charge.
		s.log.Error("STREAM BILLING FAILED — content delivered without payment",
			zap.String("userID", p.userID.String()),
			zap.String("provider", p.providerID),
			zap.Error(billErr))
	}
	return nil
}
