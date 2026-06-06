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
	modelHint string,
	onChunk func(chunk ChatStreamChunk) error,
) error {
	providers, err := s.resolveAllChatProviders(ctx, userID, modelHint)
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
			recordProviderFailure(p.userID, p.providerID)
		}
		return &failoverErr{msg: fmt.Sprintf("chat completion stream http: %v", err), transient: isTransientChatErr(err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if isFailoverStatus(resp.StatusCode) {
			recordProviderFailure(p.userID, p.providerID)
		}
		return &failoverErr{msg: fmt.Sprintf("chat completion stream: status %d", resp.StatusCode), transient: isFailoverStatus(resp.StatusCode)}
	}
	scanner := bufio.NewScanner(resp.Body)
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
			continue
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
