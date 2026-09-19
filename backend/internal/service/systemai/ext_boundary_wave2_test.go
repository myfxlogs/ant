// ext_boundary_wave2_test.go — EXT-BOUNDARY-WAVE2 (2026-09-19).
//
// Six fail-open/silent-distortion boundary fixes, one test per item:
//
//	S1  SSE mid-stream stall watchdog → failoverErr "stalled"
//	S2  chain-monitor checkpoint stall → Error log (parameterized threshold)
//	S3  SMTP auth failure → explicit error; no-AUTH relay stays silent-ok
//	S4  webhook unreachable → Abort=true "unreachable"
//	S5  TronScan inconclusive → MANUAL_REVIEW (not auto-confirm)
//	S6  finish_reason=length → non-transient "truncated" error
//
// Adversarial proofs (M1-M5): see the dispatch — each fix deleted/restored
// flips its test RED/GREEN.
package systemai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ── S1: SSE stall watchdog ───────────────────────────────────────────

// TestChatStreamStallWatchdog — a server that stops sending mid-stream trips
// the watchdog (1s test budget via timeoutSeconds) → failoverErr "stalled".
//
// Adversarial (M1): delete the watchdog → the scanner blocks until the test
// server's own deadline → RED (hang/failoverErr mismatch).
func TestChatStreamStallWatchdog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"start\"}}]}\n\n"))
		if f, ok := w.(interface{ Flush() }); ok {
			f.Flush()
		}
		select {
		case <-r.Context().Done(): // client watchdog closed the body
		case <-time.After(30 * time.Second): // far beyond the 1s stall budget
		}
	}))
	defer srv.Close()

	p := chatProvider{
		userID: testUUID(), providerID: "stall_test", model: "m",
		baseURL: srv.URL, secret: "sk", timeoutSeconds: 1, // 1s stall budget
		temperature: 0.3,
	}
	err := (&Service{log: zap.NewNop()}).tryChatCompletionStream(context.Background(), p, []ChatMessage{{Role: "user", Content: "hi"}}, nil, func(chunk ChatStreamChunk) error {
		return nil
	})
	if err == nil {
		t.Fatal("err = nil, want failoverErr 'stalled'")
	}
	if !strings.Contains(err.Error(), "stalled") {
		t.Fatalf("err = %v, want it to contain 'stalled'", err)
	}
	if fe, ok := err.(*failoverErr); !ok || !fe.transient {
		t.Fatalf("stall must be a transient failoverErr, got %T (%v)", err, err)
	}
}

// ── S6: finish_reason=length fail-closed (non-transient) ─────────────

// TestTryChatCompletion_LengthTruncated — a length-truncated response is an
// explicit non-transient failure: no failover budget burn, truncated content
// never delivered as complete.
//
// Adversarial (M4): delete the finish_reason check → err = nil with
// truncated content → RED.
func TestTryChatCompletion_LengthTruncated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"partial co"},"finish_reason":"length"}]}`))
	}))
	defer srv.Close()

	p := chatProvider{
		userID: testUUID(), providerID: "length_test", model: "m",
		baseURL: srv.URL, secret: "sk", temperature: 0.3,
	}
	_, _, _, err := (&Service{log: zap.NewNop()}).tryChatCompletion(context.Background(), p, []ChatMessage{{Role: "user", Content: "hi"}}, nil)
	if err == nil {
		t.Fatal("err = nil, want 'truncated' error")
	}
	if !strings.Contains(err.Error(), "truncated") || !strings.Contains(err.Error(), "finish_reason") {
		t.Fatalf("err = %v, want it to contain 'truncated' and 'finish_reason'", err)
	}
	if fe, ok := err.(*failoverErr); ok && fe.transient {
		t.Fatal("length truncation must be NON-transient (same-params retry truncates again)")
	}
}

// testUUID returns a fresh random UUID for provider keys.
func testUUID() uuid.UUID { return uuid.New() }
