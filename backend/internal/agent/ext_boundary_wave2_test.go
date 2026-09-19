// ext_boundary_wave2_test.go — EXT-BOUNDARY-WAVE2 (2026-09-19).
//
//	S3  SMTP auth failure → explicit error; no-AUTH relay stays silent-ok
//	S4  webhook unreachable / bad URL → Abort=true (fail-closed symmetric
//	    with the >=400 abort)
//
// Adversarial (M5): restore the S3 warn-continue → S3 auth sub-case RED;
// (M2): restore S4 fail-open returns → S4 sub-cases RED.
package agent

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

// ── S4: webhook transport fail-closed ────────────────────────────────

// TestWebhook_UnreachableAborts — a webhook pointing at a closed port aborts
// the operation (gate unreachable = cannot verify = reject), symmetric with
// the >=400 status abort.
//
// Adversarial (M2): restore the fail-open HookResult{} returns → Abort=false
// → RED.
func TestWebhook_UnreachableAborts(t *testing.T) {
	// Reserve a port then close the listener → connection refused.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	e := NewHookEngine(zap.NewNop())
	res := e.execWebhook(context.Background(), HookConfig{
		Type:       HookTypeWebhook,
		WebhookURL: "http://" + addr + "/gate",
		Timeout:    2 * time.Second,
	}, &HookContext{Event: HookPreStrategySubmit})

	if !res.Abort {
		t.Fatal("Abort = false, want true (unreachable gate must fail closed)")
	}
	if !strings.Contains(res.Reason, "unreachable") {
		t.Fatalf("Reason = %q, want it to contain 'unreachable'", res.Reason)
	}
}

// TestWebhook_BadURLAborts — an unparseable URL means the gate cannot run.
func TestWebhook_BadURLAborts(t *testing.T) {
	e := NewHookEngine(zap.NewNop())
	res := e.execWebhook(context.Background(), HookConfig{
		Type:       HookTypeWebhook,
		WebhookURL: "ht tp://not a url",
		Timeout:    time.Second,
	}, &HookContext{Event: HookPreStrategySubmit})

	if !res.Abort {
		t.Fatal("Abort = false, want true (bad URL = gate cannot run)")
	}
}
