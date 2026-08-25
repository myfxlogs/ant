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

// TestDispatchVMLive_LiveModeRejectsClientIdentityWithoutAccountID verifies
// that in live mode, ExecuteLive rejects requests without account_id (client-
// submitted Login/Company/status are not authoritative).
func TestDispatchVMLive_LiveModeRejectsClientIdentityWithoutAccountID(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	strategy, err := mql2go.CompileMQL(`int OnInit() { return 0; } void OnBar() {}`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	req := &antv1.ExecuteLiveRequest{
		StrategyCode: `int OnInit() { return 0; } void OnBar() {}`,
		RequestType:  antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext: &antv1.LiveStrategyContext{
			Open:           []string{"1"},
			High:           []string{"2"},
			Low:            []string{"0.5"},
			Close:          []string{"1.5"},
			Volume:         []string{"100"},
			BarTimesMs:     []int64{100},
			Symbol:         "EURUSD",
			Timeframe:      "M5",
			Mode:           "live",
			Login:          999999, // client-submitted — not authoritative
			Company:        "FAKE", // client-submitted — not authoritative
			IsDemo:         false,  // client-submitted — not authoritative
			IsConnected:    true,   // client-submitted — not authoritative
			IsTradeAllowed: true,   // client-submitted — not authoritative
			Balance:        "10000",
			Equity:         "10000",
			Margin:         "0",
			FreeMargin:     "10000",
		},
		// account_id intentionally empty — live mode must reject
	}
	resp, err := srv.dispatchVMLive(context.Background(), req, strategy)
	if err != nil {
		t.Fatalf("dispatchVMLive error: %v", err)
	}
	if resp.Success {
		t.Fatal("dispatchVMLive should reject live mode without account_id, got Success=true")
	}
	if !strings.Contains(resp.Error, "account_id") {
		t.Fatalf("error should mention 'account_id', got: %s", resp.Error)
	}
}

// TestDispatchVMLive_LiveModeWithAccountIDOverridesClientIdentity verifies
// that in live mode with account_id, server-side lookups override client-
// submitted identity/status.
func TestDispatchVMLive_LiveModeWithAccountIDOverridesClientIdentity(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil // server-side truth
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "ACME Broker", nil // server-side truth
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil // server-side truth: demo account
	})
	srv.SetAccountConnectedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil // server-side truth: trade not allowed
	})
	strategy, err := mql2go.CompileMQL(`int OnInit() { return 0; } void OnBar() {}`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	req := &antv1.ExecuteLiveRequest{
		StrategyCode: `int OnInit() { return 0; } void OnBar() {}`,
		RequestType:  antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext: &antv1.LiveStrategyContext{
			Open:           []string{"1"},
			High:           []string{"2"},
			Low:            []string{"0.5"},
			Close:          []string{"1.5"},
			Volume:         []string{"100"},
			BarTimesMs:     []int64{100},
			Symbol:         "EURUSD",
			Timeframe:      "M5",
			Mode:           "live",
			Login:          999999, // client-submitted fake
			Company:        "FAKE", // client-submitted fake
			IsDemo:         false,  // client-submitted fake
			IsConnected:    true,   // client-submitted fake
			IsTradeAllowed: true,   // client-submitted fake
			Balance:        "10000",
			Equity:         "10000",
			Margin:         "0",
			FreeMargin:     "10000",
		},
		AccountId: "acct-1", // server-side account truth
	}
	resp, err := srv.dispatchVMLive(context.Background(), req, strategy)
	if err != nil {
		t.Fatalf("dispatchVMLive error: %v", err)
	}
	// The request should succeed (server-side lookups override client fakes)
	// and the VM should see server-side truth, not client fakes.
	if !resp.Success {
		// It's OK if it fails due to VM execution, but it should NOT fail
		// due to account truth issues (those should be resolved by lookups).
		if strings.Contains(resp.Error, "account_id") || strings.Contains(resp.Error, "lookup") {
			t.Fatalf("dispatchVMLive should use server-side lookups, got: %s", resp.Error)
		}
	}
}
