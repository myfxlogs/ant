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
	// Pre-check wallet balance before making any API call.
	// Industry standard for streaming micro-billing: allow a small negative balance
	// (wallet CHECK constraint allows -$0.10). Future calls are blocked until positive.
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

func accumulateToolCallDeltas(toolCalls []StreamToolCall, acc map[int]*StreamToolCall) {
	for _, tc := range toolCalls {
		entry, ok := acc[tc.Index]
		if !ok {
			entry = &StreamToolCall{Index: tc.Index}
			acc[tc.Index] = entry
		}
		if tc.ID != "" {
			entry.ID = tc.ID
		}
		if tc.Type != "" {
			entry.Type = tc.Type
		}
		if tc.Function.Name != "" {
			entry.Function.Name = tc.Function.Name
		}
		entry.Function.Arguments += tc.Function.Arguments
	}
}

func finalizeToolCalls(finishReason string, acc map[int]*StreamToolCall) []StreamToolCall {
	if finishReason == "" || len(acc) == 0 {
		return nil
	}
	idxs := make([]int, 0, len(acc))
	for idx := range acc {
		idxs = append(idxs, idx)
	}
	sort.Ints(idxs)
	out := make([]StreamToolCall, 0, len(idxs))
	for _, idx := range idxs {
		out = append(out, *acc[idx])
	}
	return out
}

func (s *Service) billStreamPostCall(ctx context.Context, p chatProvider, usage *ChatUsage, messages []ChatMessage, streamedChars int) error {
	if s.postCallBiller == nil {
		return fmt.Errorf("postCallBiller not configured — billing infrastructure missing")
	}
	var inTokens, outTokens int
	if usage != nil {
		inTokens, outTokens = usage.PromptTokens, usage.CompletionTokens
	} else {
		inTokens, _ = estimateTokens(messages, "")
		outTokens = streamedChars / 4
		if outTokens < 1 {
			outTokens = 1
		}
	}
	feature := aiFeatureFromCtx(ctx)
	if billErr := s.postCallBiller(ctx, p.userID, p.providerID, p.model, feature, inTokens, outTokens); billErr != nil {
		s.log.Error("chat stream billing failed",
			zap.String("userID", p.userID.String()),
			zap.String("provider", p.providerID),
			zap.Int("inTokens", inTokens),
			zap.Int("outTokens", outTokens),
			zap.Error(billErr))
		return billErr
	}
	s.log.Info("chat stream billed",
		zap.String("userID", p.userID.String()),
		zap.String("provider", p.providerID),
		zap.Int("inTokens", inTokens),
		zap.Int("outTokens", outTokens))
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
	// Bill after successful fallback — content already delivered via onChunk.
	if s.postCallBiller == nil {
		s.log.Error("FALLBACK BILLING FAILED — postCallBiller not configured, content delivered without payment",
			zap.String("userID", p.userID.String()), zap.String("provider", p.providerID))
	} else {
		var inTokens, outTokens int
		if usage != nil {
			inTokens, outTokens = usage.PromptTokens, usage.CompletionTokens
		} else {
			inTokens, outTokens = estimateTokens(messages, result)
		}
		feature := aiFeatureFromCtx(ctx)
		if billErr := s.postCallBiller(ctx, p.userID, p.providerID, p.model, feature, inTokens, outTokens); billErr != nil {
			s.log.Error("FALLBACK BILLING FAILED — content delivered without payment",
				zap.String("userID", p.userID.String()), zap.String("provider", p.providerID), zap.Error(billErr))
		}
	}
	return nil
}
