package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"anttrader/internal/ai"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

// maxAIChatAttempts is initial request plus retries (see docs/remediation agent reliability).
const maxAIChatAttempts = 3

// ChatWithRetry runs provider.Chat with retries (exported for connect / other packages).
func ChatWithRetry(ctx context.Context, provider ai.AIProvider, messages []ai.Message) (*ai.Response, error) {
	return chatWithRetry(ctx, provider, messages)
}

func chatWithRetry(ctx context.Context, provider ai.AIProvider, messages []ai.Message) (*ai.Response, error) {
	if provider == nil {
		return nil, errors.New("ai provider is nil")
	}
	var last error
	start := time.Now()
	logger.Info("ai chat start",
		zap.String("provider", provider.GetProviderName()),
		zap.String("model", provider.GetModelName()),
		zap.Int("messages", len(messages)))
	for attempt := 0; attempt < maxAIChatAttempts; attempt++ {
		if attempt > 0 {
			if !isRetryableAIError(last) {
				break
			}
			base := time.Duration(200*(1<<uint(attempt-1))) * time.Millisecond
			if base > 2*time.Second {
				base = 2 * time.Second
			}
			jitter := time.Duration(time.Now().UnixNano() % int64(100*time.Millisecond))
			select {
			case <-time.After(base + jitter):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		resp, err := provider.Chat(ctx, messages)
		if err == nil {
			logger.Info("ai chat success",
				zap.String("provider", provider.GetProviderName()),
				zap.String("model", provider.GetModelName()),
				zap.Duration("duration", time.Since(start)),
				zap.Int("completion_tokens", resp.Usage.CompletionTokens),
				zap.Int("total_tokens", resp.Usage.TotalTokens))
			return resp, nil
		}
		last = err
		if !isRetryableAIError(err) {
			break
		}
	}
	logger.Warn("ai chat failed",
		zap.String("provider", provider.GetProviderName()),
		zap.String("model", provider.GetModelName()),
		zap.Duration("duration", time.Since(start)),
		zap.Error(last))
	return nil, last
}

// StreamChatWithRetry consumes StreamChat deltas with retries; emit is called per non-empty delta.
func StreamChatWithRetry(ctx context.Context, provider ai.AIProvider, messages []ai.Message, emit func(string)) (string, ai.Usage, error) {
	return streamChatWithRetry(ctx, provider, messages, emit)
}

func streamChatWithRetry(ctx context.Context, provider ai.AIProvider, messages []ai.Message, emit func(string)) (string, ai.Usage, error) {
	if provider == nil {
		return "", ai.Usage{}, errors.New("ai provider is nil")
	}
	var last error
	start := time.Now()
	for attempt := 0; attempt < maxAIChatAttempts; attempt++ {
		if attempt > 0 {
			if !isRetryableAIError(last) {
				break
			}
			base := time.Duration(200*(1<<uint(attempt-1))) * time.Millisecond
			if base > 2*time.Second {
				base = 2 * time.Second
			}
			jitter := time.Duration(time.Now().UnixNano() % int64(100*time.Millisecond))
			select {
			case <-time.After(base + jitter):
			case <-ctx.Done():
				return "", ai.Usage{}, ctx.Err()
			}
		}
		ch, err := provider.StreamChat(ctx, messages)
		if err != nil {
			last = err
			if !isRetryableAIError(err) {
				break
			}
			continue
		}
		var b strings.Builder
		var fatal error
		done := false
		var firstNonEmpty time.Time
		for chunk := range ch {
			if chunk.Error != nil {
				fatal = chunk.Error
				break
			}
			if chunk.Content != "" {
				if firstNonEmpty.IsZero() {
					firstNonEmpty = time.Now()
				}
				b.WriteString(chunk.Content)
				if emit != nil {
					emit(chunk.Content)
				}
			}
			if chunk.Done {
				done = true
				break
			}
		}
		if fatal == nil && (done || strings.TrimSpace(b.String()) != "") {
			fields := []zap.Field{
				zap.String("provider", provider.GetProviderName()),
				zap.String("model", provider.GetModelName()),
				zap.Int("attempt", attempt+1),
				zap.Duration("duration", time.Since(start)),
				zap.Int("out_chars", b.Len()),
			}
			if !firstNonEmpty.IsZero() {
				fields = append(fields, zap.Duration("time_to_first_chunk", firstNonEmpty.Sub(start)))
			}
			logger.Info("ai stream success", fields...)
			return strings.TrimSpace(b.String()), ai.Usage{}, nil
		}
		if fatal == nil {
			fatal = errors.New("stream ended unexpectedly")
		}
		last = fatal
		if !isRetryableAIError(last) {
			break
		}
	}
	if last == nil {
		last = errors.New("ai stream failed")
	}
	return "", ai.Usage{}, last
}

func streamChatWithFallback(ctx context.Context, providers []ai.AIProvider, messages []ai.Message, emit func(string)) (string, ai.AIProvider, ai.Usage, error) {
	var last error
	tried := 0
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		if tried >= 2 {
			break
		}
		tried++
		text, usage, err := streamChatWithRetry(ctx, provider, messages, emit)
		if err == nil {
			return text, provider, usage, nil
		}
		last = err
		if !isRetryableAIError(err) {
			break
		}
	}
	if last == nil {
		last = errors.New("no ai provider available")
	}
	return "", nil, ai.Usage{}, last
}

func chatWithFallback(ctx context.Context, providers []ai.AIProvider, messages []ai.Message) (*ai.Response, ai.AIProvider, error) {
	var last error
	tried := 0
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		if tried >= 2 {
			break
		}
		tried++
		resp, err := chatWithRetry(ctx, provider, messages)
		if err == nil {
			return resp, provider, nil
		}
		last = err
		if !isRetryableAIError(err) {
			break
		}
	}
	if last == nil {
		last = errors.New("no ai provider available")
	}
	return nil, nil, last
}

func isRetryableAIError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return true
	}
	msg := strings.ToLower(err.Error())
	retryTerms := []string{
		"timeout", "deadline exceeded", "connection reset", "connection refused",
		"temporary", "eof", "status 429", "status 502", "status 503", "status 504",
		"status 524", "524", "too many requests", "bad gateway", "service unavailable", "gateway timeout",
	}
	for _, term := range retryTerms {
		if strings.Contains(msg, term) {
			return true
		}
	}
	stopTerms := []string{"status 400", "status 401", "status 403", "invalid api", "unauthorized", "forbidden", "model not found"}
	for _, term := range stopTerms {
		if strings.Contains(msg, term) {
			return false
		}
	}
	return false
}

func providerLabel(provider ai.AIProvider) string {
	if provider == nil {
		return "unknown"
	}
	return fmt.Sprintf("%s/%s", provider.GetProviderName(), provider.GetModelName())
}
