package systemai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// FIX-2026-09-08-TIMEOUT-SELFHEAL 对抗证明。
//
// 业主实测：`chat completion http: … Client.Timeout exceeded while awaiting
// headers` 直接失败、从不重试。根因：isTransientChatErr 用小写 "timeout"
// 匹配，Go 超时错误文本是 "Client.Timeout"（大写 T）→ 永远不 transient，
// 不重试、直接把超时当最终错误抛给用户。
//
// mutation: 还原 chat.go 的 isTransientChatErr/tryChatCompletion →
// T-T1/T-T2 RED；还原 chat_stream.go 的退避重试 → T-T3 RED。

// 瞬时错误统一重试策略：429/5xx/超时 → 退避重试 2 次（表驱动，可测试注入）。
// Retry-After 存在时优先采用（上限 15s）。
func TestTransientRetryPolicy(t *testing.T) {
	oldBackoff := transientRetryBackoff
	transientRetryBackoff = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}
	defer func() { transientRetryBackoff = oldBackoff }()

	if w := retryWait(1, 0); w != 10*time.Millisecond {
		t.Fatalf("retryWait(1)=%v want table[0]", w)
	}
	if w := retryWait(2, 0); w != 20*time.Millisecond {
		t.Fatalf("retryWait(2)=%v want table[1]", w)
	}
	if w := retryWait(1, 30*time.Second); w != 15*time.Second {
		t.Fatalf("retryWait with Retry-After 30s must cap at 15s, got %v", w)
	}
	if w := retryWait(1, 3*time.Second); w != 3*time.Second {
		t.Fatalf("retryWait with Retry-After 3s must honor it, got %v", w)
	}
	if w := retryWait(5, 0); w != 20*time.Millisecond {
		t.Fatalf("retryWait overflow must clamp to table tail, got %v", w)
	}
}

func TestTryChatCompletionRetries429Twice(t *testing.T) {
	oldBackoff := transientRetryBackoff
	transientRetryBackoff = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}
	defer func() { transientRetryBackoff = oldBackoff }()

	reqs := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs++
		w.Header().Set("Content-Type", "application/json")
		if reqs <= 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":{"message":"inference exceeds tpm/rpm limit","type":"rate_limit_error"}}`)
			return
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`)
	}))
	defer srv.Close()

	p := chatProvider{
		userID: uuid.New(), providerID: "openai_compatible_test",
		model: "kimi-test", baseURL: srv.URL, secret: "sk",
		temperature: 0.3,
	}
	res, _, _, err := (&Service{log: zap.NewNop()}).tryChatCompletion(context.Background(), p, []ChatMessage{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("two 429s must be retried before giving up: %v", err)
	}
	if res != "ok" || reqs != 3 {
		t.Fatalf("res=%q reqs=%d, want ok/3", res, reqs)
	}
}

func TestTryChatCompletionStreamRetriesOn429Twice(t *testing.T) {
	oldBackoff := transientRetryBackoff
	transientRetryBackoff = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}
	defer func() { transientRetryBackoff = oldBackoff }()

	reqs := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs++
		w.Header().Set("Content-Type", "application/json")
		if reqs <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":{"message":"rate limited","type":"rate_limit_error"}}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok-stream\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer srv.Close()

	p := chatProvider{
		userID: uuid.New(), providerID: "openai_compatible_test",
		model: "m", baseURL: srv.URL, secret: "sk",
		temperature: 0.3,
	}
	var got []ChatStreamChunk
	svc := &Service{log: zap.NewNop()}
	err := svc.tryChatCompletionStream(context.Background(), p, []ChatMessage{{Role: "user", Content: "hi"}}, nil, func(c ChatStreamChunk) error {
		got = append(got, c)
		return nil
	})
	if err != nil {
		t.Fatalf("two 429s must be retried: %v", err)
	}
	if len(got) == 0 || got[0].Content != "ok-stream" || reqs != 3 {
		t.Fatalf("delivered=%v reqs=%d", got, reqs)
	}
}

func TestIsTransientChatErrCaseInsensitive(t *testing.T) {
	err := errors.New(`Post "https://h/v1/chat/completions": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`)
	if !isTransientChatErr(err) {
		t.Fatal("Client.Timeout (capital T) must be classified transient")
	}
	if !isTimeoutLikeErr(err) {
		t.Fatal("must be classified timeout-like for the user hint")
	}
	if isTransientChatErr(nil) || isTransientChatErr(errors.New("invalid api key")) {
		t.Fatal("nil / auth errors must not be transient")
	}
}

func TestTryChatCompletionRetriesOnTimeout(t *testing.T) {
	oldTimeout := chatHTTPTimeout
	chatHTTPTimeout = 50 * time.Millisecond
	defer func() { chatHTTPTimeout = oldTimeout }()

	reqs := 0
	var lastBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs++
		b, _ := io.ReadAll(r.Body)
		lastBody = string(b)
		if reqs == 1 {
			time.Sleep(200 * time.Millisecond) // force client timeout
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"recovered"}}]}`)
	}))
	defer srv.Close()

	p := chatProvider{
		userID: uuid.New(), providerID: "openai_compatible_test",
		model: "kimi-test", baseURL: srv.URL, secret: "sk",
		temperature: 0.3,
	}
	svc := &Service{log: zap.NewNop()}
	res, _, _, err := svc.tryChatCompletion(context.Background(), p, []ChatMessage{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("timeout must be retried as transient: %v", err)
	}
	if res != "recovered" {
		t.Fatalf("content = %q", res)
	}
	if reqs != 2 {
		t.Fatalf("requests = %d, want 2 (timeout retry)", reqs)
	}
	if !strings.Contains(lastBody, `"temperature":0.3`) {
		t.Fatalf("retry must keep configured temperature, body: %s", lastBody)
	}
}

func TestTryChatCompletionStreamRetriesOn429(t *testing.T) {
	first := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if first == 0 {
			first++
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":{"message":"inference exceeds tpm/rpm limit","type":"rate_limit_error"}}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok-stream\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer srv.Close()

	p := chatProvider{
		userID: uuid.New(), providerID: "openai_compatible_test",
		model: "m", baseURL: srv.URL, secret: "sk",
		temperature: 0.3,
	}
	var got []ChatStreamChunk
	svc := &Service{log: zap.NewNop()}
	err := svc.tryChatCompletionStream(context.Background(), p, []ChatMessage{{Role: "user", Content: "hi"}}, nil, func(c ChatStreamChunk) error {
		got = append(got, c)
		return nil
	})
	if err != nil {
		t.Fatalf("429 stream must be retried with backoff: %v", err)
	}
	if len(got) == 0 || got[0].Content != "ok-stream" {
		t.Fatalf("retry content not delivered: %+v", got)
	}
}

// FIX-2026-09-08-TEMP-RETRY 对抗证明。
//
// 推理模型（kimi-k3 / o1 等）对 temperature != 1 返回 400 invalid_request_error：
// "field Temperature invalid, only 1 is allowed for this model"。
// tryChatCompletion 必须携带用户配置的 temperature，并在该 400 上以
// temperature=1 重建请求自愈重试一次，而不是把 provider 判死。
//
// mutation: 还原 chat.go/chat_stream.go（硬编码 0.3、无重试）→ 两个测试 RED
//（T1 返回 failoverErr；T2 fallbackNonStream(nil) nil panic）。

func TestTryChatCompletionTemperatureRetry(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		w.Header().Set("Content-Type", "application/json")
		if len(bodies) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":{"message":"field Temperature invalid, only 1 is allowed for this model","type":"invalid_request_error"}}`)
			return
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
	}))
	defer srv.Close()

	p := chatProvider{
		userID: uuid.New(), providerID: "openai_compatible_test",
		model: "kimi-test", baseURL: srv.URL, secret: "sk",
		temperature: 0.2,
	}
	res, _, _, err := (&Service{}).tryChatCompletion(context.Background(), p, []ChatMessage{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("temperature retry failed: %v", err)
	}
	if res != "ok" {
		t.Fatalf("content = %q want ok", res)
	}
	if len(bodies) != 2 {
		t.Fatalf("requests = %d want 2", len(bodies))
	}
	if !strings.Contains(bodies[0], `"temperature":0.2`) {
		t.Fatalf("first request must carry configured temperature, body: %s", bodies[0])
	}
	if !strings.Contains(bodies[1], `"temperature":1`) {
		t.Fatalf("retry must force temperature=1, body: %s", bodies[1])
	}
}

// 流式 400 → fallbackNonStream 必须携带真实 onChunk（此前传 nil → 恢复成功后
// onChunk(nil) 调用 = nil panic）。
func TestStreamFallbackDeliversChunk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(b), `"stream":true`) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":{"message":"streaming not supported","type":"invalid_request_error"}}`)
			return
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"fallback-ok"}}]}`)
	}))
	defer srv.Close()

	p := chatProvider{
		userID: uuid.New(), providerID: "openai_compatible_test",
		model: "m", baseURL: srv.URL, secret: "sk",
		temperature: 0.3,
	}
	var got []ChatStreamChunk
	svc := &Service{log: zap.NewNop()}
	err := svc.tryChatCompletionStream(context.Background(), p, []ChatMessage{{Role: "user", Content: "hi"}}, nil, func(c ChatStreamChunk) error {
		got = append(got, c)
		return nil
	})
	if err != nil {
		t.Fatalf("stream fallback failed: %v", err)
	}
	if len(got) == 0 || got[0].Content != "fallback-ok" {
		t.Fatalf("fallback chunk not delivered: %+v", got)
	}
}

func TestDefaultTemperature(t *testing.T) {
	if defaultTemperature(0) != 0.3 || defaultTemperature(-1) != 0.3 {
		t.Fatal("defaultTemperature unset must fall back to 0.3")
	}
	if defaultTemperature(0.7) != 0.7 {
		t.Fatal("defaultTemperature must respect configured value")
	}
}

// FIX-2026-09-08-BYOK-MODEL-PICKER 对抗证明。
//
// 用户在 /ai/settings 粘贴完整 endpoint URL（如 NOVA 文档给出的
// https://token.sensenova.cn/v1/chat/completions）时，chatEndpoint 必须产出
// 单一 /chat/completions 后缀；否则双路径 404 → BYOK 聊天全挂
// （生产日志: "discover models failed ... base_url must be openai-compatible"）。
//
// mutation: 还原 chatEndpoint 去掉 normalizeAPIBase → T-C1 RED。

func TestChatEndpointToleratesFullEndpointURL(t *testing.T) {
	got := chatEndpoint("openai_compatible_mts0f275", "https://token.sensenova.cn/v1/chat/completions")
	want := "https://token.sensenova.cn/v1/chat/completions"
	if got != want {
		t.Fatalf("chatEndpoint doubled path: got %q want %q", got, want)
	}
}

func TestChatEndpointRegularBaseURLUnchanged(t *testing.T) {
	cases := []struct{ provider, base, want string }{
		{"deepseek", "https://api.deepseek.com/v1", "https://api.deepseek.com/v1/chat/completions"},
		{"openai", "https://api.openai.com/v1/", "https://api.openai.com/v1/chat/completions"},
		{"zhipu", "https://open.bigmodel.cn/api/paas/v4", "https://open.bigmodel.cn/api/paas/v4/chat/completions"},
		{"openai_compatible_x", "https://llm.example.io/v1/models", "https://llm.example.io/v1/chat/completions"},
	}
	for _, c := range cases {
		if got := chatEndpoint(c.provider, c.base); got != c.want {
			t.Fatalf("chatEndpoint(%q,%q) = %q want %q", c.provider, c.base, got, c.want)
		}
	}
}

// resolveModel 契约：用户 primary 精确选择优先于 default_model/models[0]。
// mutation: 删除 primaryPID 匹配分支 → T-R1 RED。
func TestResolveModelPrimaryPickWins(t *testing.T) {
	if got := resolveModel("default-m", []string{"first-m"}, "zhipu", "zhipu", "picked-m"); got != "picked-m" {
		t.Fatalf("resolveModel primary pick: got %q want picked-m", got)
	}
	if got := resolveModel("default-m", []string{"first-m"}, "zhipu", "other", "picked-m"); got != "default-m" {
		t.Fatalf("resolveModel non-primary provider: got %q want default-m", got)
	}
	if got := resolveModel("", []string{"first-m"}, "zhipu", "other", ""); got != "first-m" {
		t.Fatalf("resolveModel fallback models[0]: got %q want first-m", got)
	}
}
