package strategy

import (
	"context"
	"strings"
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// vm_audit_2026_08_27_batch2_test.go — VM-AUDIT-2026-08-27 批次 2 对抗测试 (BUG-5).
//
// Tests verify:
//   - VM-AUDIT-2026-08-27-2 (BUG-5): VMLiveSession.dispatch default branch
//     returns an explicit error for unknown request types instead of
//     silently treating them as bar events when a BarContext is present.
//
// Adversarial proof: restore the old default branch (if bctx != nil →
// vmHandleBar) → test RED (returns Success: true) → restore → GREEN.

// TestVMLiveSession_UnknownRequestType verifies BUG-5 fix: an unknown
// RequestType (REQUEST_TYPE_UNSPECIFIED) with a non-nil BarContext must
// return Success: false with an "unknown request type" error, NOT execute
// the strategy as a bar event.
//
// The old default branch checked `if bctx != nil` and treated the request
// as a bar event — so a proto with an unknown enum value but a stale
// BarContext would silently execute the strategy. The fix removes that
// conditional and always returns an error for unknown types.
//
// Adversarial proof: restore `if bctx != nil { resp = vmHandleBar(...) }`
// → the request is treated as a bar event → response is Success: true (or
// a bar-handler error, but NOT "unknown request type") → RED. Restore the
// fix → Success: false + "unknown request type" → GREEN.
func TestVMLiveSession_UnknownRequestType(t *testing.T) {
	t.Parallel()

	// Create a VMLiveSession with a minimal MQL strategy so Start succeeds
	// and we can reach the dispatch default branch via SendEvent.
	source := `int OnInit() { return 0; }
void OnBar() {}`
	sess, err := NewVMLiveSession(source)
	if err != nil {
		t.Fatalf("NewVMLiveSession failed: %v", err)
	}

	// Start the session with a valid bar_context (required for initialization).
	startReq := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext: &antv1.LiveStrategyContext{
			Symbol:    "EURUSD",
			Timeframe: "M5",
			Mode:      "live",
		},
	}
	if _, err := sess.Start(context.Background(), startReq); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	t.Cleanup(func() { _ = sess.Close() })

	// Send an event with an unknown RequestType but a non-nil BarContext.
	// BUG-5 (old code): default branch saw bctx != nil → treated as bar →
	//   returned a bar-handler response (Success: true or bar error).
	// Fix (S3): default branch returns "unknown request type" error.
	unknownReq := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_UNSPECIFIED,
		BarContext: &antv1.LiveStrategyContext{
			Symbol:    "EURUSD",
			Timeframe: "M5",
			Mode:      "live",
		},
	}

	resp, err := sess.SendEvent(context.Background(), unknownReq)
	if err != nil {
		t.Fatalf("SendEvent failed: %v", err)
	}

	if resp.GetSuccess() {
		t.Fatal("expected Success: false for unknown request type, got Success: true (BUG-5: unknown type treated as bar event)")
	}
	if !strings.Contains(resp.GetError(), "unknown request type") {
		t.Fatalf("expected error containing 'unknown request type', got: %q", resp.GetError())
	}
}
