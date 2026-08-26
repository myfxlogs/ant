package strategy

import (
	"context"
	"strings"
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
	"alphaforge/strategy/runner"
	"alphaforge/tools/mql2go"
)

// newTestRunner creates a runner with a simple MQL strategy for testing.
func newTestRunner(t *testing.T) *runner.Runner {
	t.Helper()
	strategy, err := mql2go.CompileMQL(`int OnInit() { return 0; } void OnBar() {}`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	r := runner.New(runner.Config{Symbol: "EURUSD", Timeframe: "M5", Mode: "paper"})
	r.SetStrategy(strategy)
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
	return r
}

// ── VM-TRADE-CONTEXT-6 round 5: live financial fields required,
// buildTradeContext enum fail-closed, ExecuteLive server-side truth ──

// TestVMHandleBar_LiveModeEmptyFinancialRejected verifies that in live mode,
// empty Balance/Equity/Margin/FreeMargin are rejected (authoritative broker
// data must be present). Paper mode allows empty.
func TestVMHandleBar_LiveModeEmptyFinancialRejected(t *testing.T) {
	r := newTestRunner(t)
	bctx := &antv1.LiveStrategyContext{
		Open:       []string{"1"},
		High:       []string{"2"},
		Low:        []string{"0.5"},
		Close:      []string{"1.5"},
		Volume:     []string{"100"},
		BarTimesMs: []int64{100},
		Symbol:     "EURUSD",
		Timeframe:  "M5",
		Mode:       "live",
		// Financial fields intentionally empty
	}
	resp := vmHandleBar(context.Background(), r, bctx)
	if resp.Success {
		t.Fatal("vmHandleBar should reject live mode with empty financial fields, got Success=true")
	}
	if !strings.Contains(resp.Error, "required in live mode") {
		t.Fatalf("error should mention 'required in live mode', got: %s", resp.Error)
	}
}

// TestVMHandleBar_PaperModeEmptyFinancialAccepted verifies that paper mode
// accepts empty financial fields (simulation may not have real account data).
func TestVMHandleBar_PaperModeEmptyFinancialAccepted(t *testing.T) {
	r := newTestRunner(t)
	bctx := &antv1.LiveStrategyContext{
		Open:       []string{"1"},
		High:       []string{"2"},
		Low:        []string{"0.5"},
		Close:      []string{"1.5"},
		Volume:     []string{"100"},
		BarTimesMs: []int64{100},
		Symbol:     "EURUSD",
		Timeframe:  "M5",
		Mode:       "paper",
		// Financial fields intentionally empty — paper mode allows this
	}
	resp := vmHandleBar(context.Background(), r, bctx)
	if !resp.Success {
		t.Fatalf("vmHandleBar should accept paper mode with empty financial fields, got error: %s", resp.Error)
	}
}

// TestBuildTradeContext_UnknownSideRejected verifies that an unknown broker
// side is rejected, not silently normalized to "buy".
func TestBuildTradeContext_UnknownSideRejected(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	cfg := LiveStrategyConfig{AccountID: "acct-1", Mode: "paper", Symbol: "EURUSD", Timeframe: "M5"}
	// Create a trade event with an unknown side
	evt := &mthub.BrokerTradeEvent{
		Ticket:    123,
		Symbol:    "EURUSD",
		Side:      "unknown_side",
		EventType: mthub.BrokerTradeFilled,
	}
	_, err := srv.buildTradeContext(context.Background(), cfg, evt)
	if err == nil {
		t.Fatal("buildTradeContext should reject unknown side, not default to buy")
	}
	if !strings.Contains(err.Error(), "unknown broker side") {
		t.Fatalf("error should mention 'unknown broker side', got: %s", err.Error())
	}
}

// TestBuildTradeContext_UnknownEventTypeRejected verifies that an unknown
// broker trade event type is rejected, not silently normalized to "fill".
func TestBuildTradeContext_UnknownEventTypeRejected(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	cfg := LiveStrategyConfig{AccountID: "acct-1", Mode: "paper", Symbol: "EURUSD", Timeframe: "M5"}
	evt := &mthub.BrokerTradeEvent{
		Ticket:    123,
		Symbol:    "EURUSD",
		Side:      "buy",
		EventType: mthub.BrokerTradeEventType(99), // unknown event type
	}
	_, err := srv.buildTradeContext(context.Background(), cfg, evt)
	if err == nil {
		t.Fatal("buildTradeContext should reject unknown event type, not default to fill")
	}
	if !strings.Contains(err.Error(), "unknown broker trade event type") {
		t.Fatalf("error should mention 'unknown broker trade event type', got: %s", err.Error())
	}
}
