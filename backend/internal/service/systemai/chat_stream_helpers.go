// chat_stream_helpers.go — Stream helpers, billing, and fallback extracted from chat_stream.go.
package systemai

import (
	"context"
	"fmt"
	"sort"

	"go.uber.org/zap"
)

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

// capMaxTokens limits the max_tokens sent to the AI provider so that
// output generation cannot exceed the user's remaining token quota.
// remainingTokens = -1 means unlimited (no cap).
func capMaxTokens(current, remainingTokens int) int {
	if remainingTokens < 0 {
		return current // unlimited
	}
	// Reserve a small floor so the provider doesn't return a degenerate response.
	// If remaining is very small (e.g. 50), still allow at least 100 tokens
	// so the model can produce a short "quota exceeded" style message.
	cap := remainingTokens
	if cap < 100 {
		cap = 100
	}
	if current <= 0 || cap < current {
		return cap
	}
	return current
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
	// Track session-scoped usage so subsequent rounds in the same agent loop
	// see the cumulative total and can enforce quota mid-session.
	if sc := sessionCounterFromCtx(ctx); sc != nil {
		sc.AddTokens(inTokens + outTokens)
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
