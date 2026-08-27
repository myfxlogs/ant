package strategy

import (
	"context"
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// vm_live_session_test.go — FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP adversarial
// proof (S10).
//
// Root cause: VMLiveSession is an in-process implementation, but the Session
// interface used []byte (proto-marshaled) signatures. proto3 marshal/unmarshal
// collapses an empty repeated slice `[]*T{}` to `nil` on round-trip, making
// "no open positions" (empty slice) indistinguishable from "data missing"
// (nil). This caused rejectNilRepeatedInLive to reject valid empty-position
// accounts in live mode.
//
// Fix: Session interface passes *antv1.ExecuteLiveRequest /
// *antv1.ExecuteLiveResponse pointers directly — no proto marshal/unmarshal,
// so Go's empty-slice semantics are preserved (empty stays empty, never nil).

// TestVMLiveSession_NilPositionsSurviveRoundTrip verifies that an empty
// (non-nil) Positions slice sent through SendEvent arrives at the handler
// as a non-nil empty slice and the response succeeds — the strategy is NOT
// rejected for "missing positions" in live mode.
//
// Adversarial proof (two-part mutation restoring the original bug):
//
//  1. Re-introduce proto round-trip in VMLiveSession.SendEvent:
//       b, _ := proto.Marshal(req)
//       var req2 antv1.ExecuteLiveRequest
//       proto.Unmarshal(b, &req2)
//       return s.dispatch(ctx, &req2), nil
//     → req2.TickContext.Positions collapses from `[]*LivePosition{}` to nil.
//
//  2. Re-introduce the nil-positions guard in vmHandleTick:
//       if tctx.Mode == modeLive && tctx.Positions == nil {
//           return &antv1.ExecuteLiveResponse{Success: false, Error: "live mode requires positions (nil = data missing)"}
//       }
//
// With both mutations: empty slice → nil (via proto) → rejected (via guard)
// → resp.Success == false → test RED.
// Restore the fix (no round-trip): empty slice stays non-nil → guard passes
// → resp.Success == true → test GREEN.
//
// The fix is the root-cause elimination: because the slice is passed by
// pointer (no marshal), re-introducing the nil guard ALONE does NOT break
// the test — empty positions stay non-nil and pass the guard. Only restoring
// the proto round-trip (which collapses empty→nil) re-creates the bug.
func TestVMLiveSession_NilPositionsSurviveRoundTrip(t *testing.T) {
	// Minimal MQL strategy that compiles and runs OnTick without referencing
	// positions — we only care that the session accepts the event.
	const code = `int OnInit() { return 0; } void OnTick() {}`
	sess, err := NewVMLiveSession(code)
	if err != nil {
		t.Fatalf("NewVMLiveSession failed: %v", err)
	}
	t.Cleanup(func() { _ = sess.Close() })

	// Start with a valid bar_context (required for initialization).
	startReq := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext: &antv1.LiveStrategyContext{
			Symbol:    "EURUSD",
			Timeframe: "M5",
			Mode:      modeLive,
			Close:     []string{"1.0"},
			Open:      []string{"1.0"},
			High:      []string{"1.0"},
			Low:       []string{"1.0"},
			Volume:    []string{"10"},
			BarTimesMs: []int64{1},
			// Empty (non-nil) positions — the exact case proto3 collapses to nil.
			Positions:     []*antv1.LivePosition{},
			PendingOrders: []*antv1.LivePendingOrder{},
		},
	}
	if _, err := sess.Start(context.Background(), startReq); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Send a tick event with empty (non-nil) positions in live mode.
	// This is the case that was misrejected before the fix.
	tickReq := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_TICK,
		TickContext: &antv1.TickContext{
			Symbol:    "EURUSD",
			Timeframe: "M5",
			Mode:      modeLive,
			Bid:       "1.0",
			Ask:       "1.0001",
			// Empty (non-nil) positions — proto3 would collapse this to nil.
			Positions:     []*antv1.LivePosition{},
			PendingOrders: []*antv1.LivePendingOrder{},
		},
	}

	// Sanity: the test's request has a non-nil empty Positions slice.
	if tickReq.GetTickContext().Positions == nil {
		t.Fatal("test setup error: Positions should be non-nil empty slice")
	}

	resp, err := sess.SendEvent(context.Background(), tickReq)
	if err != nil {
		t.Fatalf("SendEvent failed: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatalf("SendEvent with empty positions should succeed (proto3 nil==empty collapse eliminated), got error: %s", resp.GetError())
	}

	// Verify the request the test constructed still has a non-nil empty slice.
	// With the fix (pointer pass-through), the session never mutates or copies
	// the caller's request, so the slice identity is preserved. A proto
	// round-trip would have operated on an internal copy, leaving the caller's
	// slice untouched — so this assertion alone is not adversarial; the
	// adversarial strength comes from the resp.Success assertion combined with
	// the documented two-part mutation above.
	if tickReq.GetTickContext().Positions == nil {
		t.Fatal("caller's request Positions slice was mutated to nil — fix must pass the pointer without copying")
	}
	if len(tickReq.GetTickContext().Positions) != 0 {
		t.Errorf("caller's Positions len = %d, want 0 (empty slice preserved)", len(tickReq.GetTickContext().Positions))
	}
}
