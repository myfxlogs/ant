package service

import (
	"context"
	"errors"
	"testing"

	"anttrader/internal/ai"
)

type chatStubProvider struct {
	name, model string
	calls        int
	failUntil    int
	failErr      error
	ok           *ai.Response
}

func (s *chatStubProvider) Chat(ctx context.Context, messages []ai.Message) (*ai.Response, error) {
	s.calls++
	if s.calls <= s.failUntil {
		if s.failErr != nil {
			return nil, s.failErr
		}
		return nil, errors.New("status 503 service unavailable")
	}
	if s.ok != nil {
		return s.ok, nil
	}
	return &ai.Response{Content: "ok", Usage: ai.Usage{}}, nil
}

func (s *chatStubProvider) StreamChat(ctx context.Context, messages []ai.Message) (<-chan ai.StreamChunk, error) {
	return nil, ai.ErrStreamNotSupported
}

func (s *chatStubProvider) ValidateConfig() error { return nil }

func (s *chatStubProvider) GetModelName() string { return s.model }

func (s *chatStubProvider) GetProviderName() string { return s.name }

func TestChatWithRetry_succeedsAfterRetryableFailures(t *testing.T) {
	stub := &chatStubProvider{name: "x", model: "m", failUntil: 2, failErr: errors.New("status 503")}
	ctx := context.Background()
	resp, err := chatWithRetry(ctx, stub, nil)
	if err != nil {
		t.Fatalf("chatWithRetry: %v", err)
	}
	if resp == nil || resp.Content != "ok" {
		t.Fatalf("unexpected resp: %#v", resp)
	}
	if stub.calls != 3 {
		t.Fatalf("want 3 Chat calls, got %d", stub.calls)
	}
}

func TestChatWithRetry_noRetryOnNonRetryable(t *testing.T) {
	stub := &chatStubProvider{name: "x", model: "m", failUntil: 3, failErr: errors.New("status 401 unauthorized")}
	ctx := context.Background()
	_, err := chatWithRetry(ctx, stub, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if stub.calls != 1 {
		t.Fatalf("want 1 Chat call, got %d", stub.calls)
	}
}

func TestIsRetryableAIError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "deadline", err: context.DeadlineExceeded, want: true},
		{name: "timeout text", err: errors.New("failed to send request: i/o timeout"), want: true},
		{name: "rate limit", err: errors.New("API request failed with status 429"), want: true},
		{name: "bad gateway", err: errors.New("API request failed with status 502"), want: true},
		{name: "unauthorized", err: errors.New("API request failed with status 401: unauthorized"), want: false},
		{name: "bad request", err: errors.New("API request failed with status 400"), want: false},
		{name: "model not found", err: errors.New("model not found"), want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetryableAIError(tc.err); got != tc.want {
				t.Fatalf("isRetryableAIError() = %v, want %v", got, tc.want)
			}
		})
	}
}
