package systemai

// chat_stream_scan.go — SSE stream parsing + the mid-stream stall watchdog.
// Split from chat_stream.go (EXT-BOUNDARY-WAVE2 S1) to stay under the
// file-lines budget.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

func (s *Service) handleStreamResponse(resp *http.Response, p chatProvider, messages []ChatMessage, onChunk func(chunk ChatStreamChunk) error) error {
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	// EXT-BOUNDARY-WAVE2 S1: inter-chunk stall watchdog. ResponseHeaderTimeout
	// only bounds time-to-first-byte; a mid-stream stall (connection alive,
	// no data) used to hang the scanner forever with no deadline upstream.
	// Budget reuses the per-provider timeout_seconds clamp (default 5min —
	// generous headroom for reasoning-model inter-chunk thinking).
	stallTimeout := effectiveTimeout(p.timeoutSeconds, 5*time.Minute)
	var stalled atomic.Bool
	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())
	watchStop := make(chan struct{})
	defer close(watchStop)
	go func() {
		ticker := time.NewTicker(stallTimeout / 2)
		defer ticker.Stop()
		for {
			select {
			case <-watchStop:
				return
			case <-ticker.C:
				last := time.Since(time.Unix(0, lastActivity.Load()))
				if last > stallTimeout && !stalled.Load() {
					stalled.Store(true)
					_ = resp.Body.Close() // unblock the scanner
					return
				}
			}
		}
	}()

	var streamUsage *ChatUsage
	toolCallAcc := make(map[int]*StreamToolCall)
	totalContentLen := 0
	totalReasoningLen := 0
	lastFinishReason := ""

	for scanner.Scan() {
		lastActivity.Store(time.Now().UnixNano())
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
	// EXT-BOUNDARY-WAVE2 S1: the watchdog closed the body → scanner.Err is
	// "use of closed network connection" — report the stall as the cause,
	// not the raw transport symptom.
	if stalled.Load() {
		// Vendor died mid-stream (connection alive, no data) — a legitimate
		// failover scenario: another provider can serve the request.
		return &failoverErr{
			msg:       fmt.Sprintf("[%s] chat stream stalled: no data for %s", p.providerID, stallTimeout),
			transient: true,
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
