package strategy

import (
	"context"
	"strings"
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"

	"google.golang.org/protobuf/proto"
)

// ── VM-API-TRUTH-3 behavior tests (返工第三阶段) ────────────────────

// TestBuildLiveContext_InjectsIsDemo verifies that buildLiveContext fills
// IsDemo from the authoritative accountIsDemoLookup.
// VM-API-TRUTH-3: previously IsDemo was hardcoded true in the VM builtin,
// even for real accounts.
func TestBuildLiveContext_InjectsIsDemo(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil // real account
	})
	srv.SetAccountConnectedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
	    return false, nil // not an investor
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "paper", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("buildLiveContext: %v", err)
	}
	if lctx.IsDemo {
		t.Errorf("IsDemo=true, want false (real account from lookup)")
	}
}

// TestBuildLiveContext_IsDemoDemoAccount verifies that a demo account
// lookup correctly sets IsDemo=true.
func TestBuildLiveContext_IsDemoDemoAccount(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil // demo account
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "paper", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("buildLiveContext: %v", err)
	}
	if !lctx.IsDemo {
		t.Errorf("IsDemo=false, want true (demo account from lookup)")
	}
}

// TestBuildLiveContext_LiveModeIsConnectedFromLookup verifies that in live
// mode, IsConnected comes from accountConnectedLookup, not hardcoded true.
// VM-API-TRUTH-3 返工: mode-aware semantics, not shared constant.
func TestBuildLiveContext_LiveModeIsConnectedFromLookup(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "ACME", nil
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
	})
	srv.SetAccountConnectedLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil // account is disconnected!
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
	    return false, nil // not an investor
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("buildLiveContext: %v", err)
	}
	if lctx.IsConnected {
		t.Errorf("IsConnected=true, want false (account is disconnected from lookup)")
	}
}

// TestBuildLiveContext_LiveModeIsTradeAllowedFromLookup verifies that in
// live mode, IsTradeAllowed comes from accountTradeAllowedLookup, not
// hardcoded true. VM-API-TRUTH-3 返工: not a constant.
func TestBuildLiveContext_LiveModeIsTradeAllowedFromLookup(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "ACME", nil
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
	})
	srv.SetAccountConnectedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil // trading is NOT allowed!
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
	    return false, nil // not an investor
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("buildLiveContext: %v", err)
	}
	if lctx.IsTradeAllowed {
		t.Errorf("IsTradeAllowed=true, want false (trade not allowed from lookup)")
	}
}

// TestBuildLiveContext_LiveModeMissingConnectedLookupFailClosed verifies
// that in live mode, missing accountConnectedLookup fails closed.
func TestBuildLiveContext_LiveModeMissingConnectedLookupFailClosed(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "ACME", nil
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
	})
	// accountConnectedLookup NOT configured
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
	    return false, nil // not an investor
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("live mode with missing accountConnectedLookup should fail closed")
	}
}

// TestBuildLiveContext_LiveModeMissingTradeAllowedLookupFailClosed verifies
// that in live mode, missing accountTradeAllowedLookup fails closed.
func TestBuildLiveContext_LiveModeMissingTradeAllowedLookupFailClosed(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "ACME", nil
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
	})
	srv.SetAccountConnectedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	// accountTradeAllowedLookup NOT configured
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("live mode with missing accountTradeAllowedLookup should fail closed")
	}
}

// TestVMLiveSession_IsDemoEndToEnd verifies that IsDemo() in MQL OnInit
// reads the real value from the LiveStrategyContext, not a hardcoded true.
// VM-API-TRUTH-3 返工: penetrate VMLiveSession.Start → OnInit → IsDemo().
func TestVMLiveSession_IsDemoEndToEnd(t *testing.T) {
	source := `
int g_isDemo = -1;
int g_isConnected = -1;
int g_isTradeAllowed = -1;

int OnInit() {
    g_isDemo = IsDemo();
    g_isConnected = IsConnected();
    g_isTradeAllowed = IsTradeAllowed();
    return 0;
}

int OnBar() { return 0; }
`
	sess, err := NewVMLiveSession(source)
	if err != nil {
		t.Fatalf("NewVMLiveSession: %v", err)
	}
	bctx := &antv1.LiveStrategyContext{
		Close:          []string{"1.5"},
		Open:           []string{"1.0"},
		High:           []string{"2.0"},
		Low:            []string{"0.5"},
		Volume:         []string{"100"},
		BarTimesMs:     []int64{100},
		Symbol:         "EURUSD",
		Timeframe:      "M5",
		Login:          123456,
		Company:        "TestBroker",
		IsDemo:         false, // real account
		IsConnected:    true,
		IsTradeAllowed: true,
	}
	req := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:  bctx,
	}
	reqBytes, _ := proto.Marshal(req)
	respBytes, err := sess.Start(context.Background(), reqBytes)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	var resp antv1.ExecuteLiveResponse
	if err := proto.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("Start failed: %s", resp.Error)
	}
	// Read back IsDemo from VM global.
	v, ok := sess.strategy.GetGlobal("g_isDemo")
	if !ok {
		t.Fatal("global g_isDemo not found")
	}
	if v.ToInt() != 0 {
		t.Errorf("IsDemo() = %d, want 0 (false, real account)", v.ToInt())
	}
}

// TestVMLiveSession_IsDemoTrueEndToEnd verifies that IsDemo() returns true
// for a demo account. VM-API-TRUTH-3 返工: true→false bidirectional test.
func TestVMLiveSession_IsDemoTrueEndToEnd(t *testing.T) {
	source := `
int g_isDemo = -1;
int OnInit() {
    g_isDemo = IsDemo();
    return 0;
}
int OnBar() { return 0; }
`
	sess, err := NewVMLiveSession(source)
	if err != nil {
		t.Fatalf("NewVMLiveSession: %v", err)
	}
	bctx := &antv1.LiveStrategyContext{
		Close:      []string{"1.5"},
		Open:       []string{"1.0"},
		High:       []string{"2.0"},
		Low:        []string{"0.5"},
		Volume:     []string{"100"},
		BarTimesMs: []int64{100},
		Symbol:     "EURUSD",
		Timeframe:  "M5",
		Login:      123456,
		Company:    "TestBroker",
		IsDemo:     true, // demo account
	}
	req := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:  bctx,
	}
	reqBytes, _ := proto.Marshal(req)
	respBytes, err := sess.Start(context.Background(), reqBytes)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	var resp antv1.ExecuteLiveResponse
	if err := proto.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("Start failed: %s", resp.Error)
	}
	v, ok := sess.strategy.GetGlobal("g_isDemo")
	if !ok {
		t.Fatal("global g_isDemo not found")
	}
	if v.ToInt() != 1 {
		t.Errorf("IsDemo() = %d, want 1 (true, demo account)", v.ToInt())
	}
}

// TestVMLiveSession_IsTradeAllowedFalseEndToEnd verifies that
// IsTradeAllowed() returns false when the context says trade is not allowed.
// VM-API-TRUTH-3 返工: true→false bidirectional test through VMLiveSession.
func TestVMLiveSession_IsTradeAllowedFalseEndToEnd(t *testing.T) {
	source := `
int g_isTradeAllowed = -1;
int OnInit() {
    g_isTradeAllowed = IsTradeAllowed();
    return 0;
}
int OnBar() { return 0; }
`
	sess, err := NewVMLiveSession(source)
	if err != nil {
		t.Fatalf("NewVMLiveSession: %v", err)
	}
	bctx := &antv1.LiveStrategyContext{
		Close:          []string{"1.5"},
		Open:           []string{"1.0"},
		High:           []string{"2.0"},
		Low:            []string{"0.5"},
		Volume:         []string{"100"},
		BarTimesMs:     []int64{100},
		Symbol:         "EURUSD",
		Timeframe:      "M5",
		Login:          123456,
		Company:        "TestBroker",
		IsDemo:         false,
		IsConnected:    true,
		IsTradeAllowed: false, // trade NOT allowed!
	}
	req := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:  bctx,
	}
	reqBytes, _ := proto.Marshal(req)
	respBytes, err := sess.Start(context.Background(), reqBytes)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	var resp antv1.ExecuteLiveResponse
	if err := proto.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("Start failed: %s", resp.Error)
	}
	v, ok := sess.strategy.GetGlobal("g_isTradeAllowed")
	if !ok {
		t.Fatal("global g_isTradeAllowed not found")
	}
	if v.ToInt() != 0 {
		t.Errorf("IsTradeAllowed() = %d, want 0 (false, trade not allowed)", v.ToInt())
	}
}

// TestVMLiveSession_IsConnectedFalseEndToEnd verifies that
// IsConnected() returns false when the context says not connected.
func TestVMLiveSession_IsConnectedFalseEndToEnd(t *testing.T) {
	source := `
int g_isConnected = -1;
int OnInit() {
    g_isConnected = IsConnected();
    return 0;
}
int OnBar() { return 0; }
`
	sess, err := NewVMLiveSession(source)
	if err != nil {
		t.Fatalf("NewVMLiveSession: %v", err)
	}
	bctx := &antv1.LiveStrategyContext{
		Close:       []string{"1.5"},
		Open:        []string{"1.0"},
		High:        []string{"2.0"},
		Low:         []string{"0.5"},
		Volume:      []string{"100"},
		BarTimesMs:  []int64{100},
		Symbol:      "EURUSD",
		Timeframe:   "M5",
		Login:       123456,
		Company:     "TestBroker",
		IsDemo:      false,
		IsConnected: false, // NOT connected!
	}
	req := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:  bctx,
	}
	reqBytes, _ := proto.Marshal(req)
	respBytes, err := sess.Start(context.Background(), reqBytes)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	var resp antv1.ExecuteLiveResponse
	if err := proto.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("Start failed: %s", resp.Error)
	}
	v, ok := sess.strategy.GetGlobal("g_isConnected")
	if !ok {
		t.Fatal("global g_isConnected not found")
	}
	if v.ToInt() != 0 {
		t.Errorf("IsConnected() = %d, want 0 (false, not connected)", v.ToInt())
	}
}

// TestBuildLiveContext_PaperModeIsConnectedDefault verifies that in paper
// mode without lookups, IsConnected defaults to false (zero value), not
// hardcoded true. VM-API-TRUTH-3 返工: no shared constant.
func TestBuildLiveContext_PaperModeIsConnectedDefault(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "paper", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("buildLiveContext: %v", err)
	}
	if lctx.IsConnected {
		t.Errorf("IsConnected=true in paper mode without lookup, want false (zero default, not hardcoded true)")
	}
	if lctx.IsTradeAllowed {
		t.Errorf("IsTradeAllowed=true in paper mode without lookup, want false (zero default, not hardcoded true)")
	}
}

// TestBuildLiveContext_LiveModeIsDemoRealAccount verifies that a real
// account (account_type != demo) gets IsDemo=false.
// VM-API-TRUTH-3 返工: real account returns false, not fake-green default.
func TestBuildLiveContext_LiveModeIsDemoRealAccount(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "ACME", nil
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil // real account
	})
	srv.SetAccountConnectedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
	    return false, nil // not an investor
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("buildLiveContext: %v", err)
	}
	if lctx.IsDemo {
		t.Errorf("IsDemo=true for real account, want false")
	}
}

// TestBuildLiveContext_LiveModeIsDemoDemoAccount verifies that a demo
// account gets IsDemo=true.
func TestBuildLiveContext_LiveModeIsDemoDemoAccount(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "ACME", nil
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil // demo account
	})
	srv.SetAccountConnectedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
	    return false, nil // not an investor
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("buildLiveContext: %v", err)
	}
	if !lctx.IsDemo {
		t.Errorf("IsDemo=false for demo account, want true")
	}
}

// TestBuildLiveContext_LiveModeAllLookupsMissingErrorContainsAll verifies
// that the error message mentions which lookup is missing.
func TestBuildLiveContext_LiveModeAllLookupsMissingErrorContainsAll(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	// No lookups configured at all.
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("should fail closed")
	}
	if !strings.Contains(err.Error(), "accountLoginLookup") {
		t.Errorf("error should mention accountLoginLookup, got: %s", err.Error())
	}
}
