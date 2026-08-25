package strategy

import (
	"context"
	"errors"
	"strings"
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"
	"google.golang.org/protobuf/proto"
)

// ── VM-API-TRUTH-3 round 4 behavior tests ────────────────────────────
// Tests for: lookup query error vs real false, investor+connected,
// IsTradeAllowed fail-closed for investor accounts.

// TestBuildLiveContext_LookupQueryErrorBlocksExecution verifies that a
// DB query error in any lookup blocks live context creation (fail-closed),
// rather than being confused with a real false/0/"" value.
// VM-API-TRUTH-3 round 4: lookups return (value, error) — error must propagate.
func TestBuildLiveContext_LookupQueryErrorBlocksExecution(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	queryErr := errors.New("simulated DB connection failure")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "ACME", nil
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, queryErr // DB error!
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
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("buildLiveContext should fail when isDemo lookup returns DB error")
	}
	if !errors.Is(err, queryErr) {
		t.Fatalf("error should wrap the query error, got: %v", err)
	}
}

// TestBuildLiveContext_ConnectedLookupQueryErrorBlocksExecution verifies
// that a DB error in accountConnectedLookup blocks execution.
func TestBuildLiveContext_ConnectedLookupQueryErrorBlocksExecution(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	queryErr := errors.New("DB timeout")
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
		return false, queryErr // DB error, not real false!
	})
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
		t.Fatal("buildLiveContext should fail when connected lookup returns DB error")
	}
	if !errors.Is(err, queryErr) {
		t.Fatalf("error should wrap the query error, got: %v", err)
	}
}

// TestBuildLiveContext_TradeAllowedLookupQueryErrorBlocksExecution verifies
// that a DB error in accountTradeAllowedLookup blocks execution.
func TestBuildLiveContext_TradeAllowedLookupQueryErrorBlocksExecution(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	queryErr := errors.New("DB connection lost")
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
		return false, queryErr // DB error, not real false!
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
	    return false, nil // not an investor
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("buildLiveContext should fail when trade allowed lookup returns DB error")
	}
	if !errors.Is(err, queryErr) {
		t.Fatalf("error should wrap the query error, got: %v", err)
	}
}

// TestBuildLiveContext_InvestorConnectedIsTradeAllowedFalse verifies that
// an investor (read-only) account with account_status=connected gets
// IsTradeAllowed=false, even though the account is connected.
// VM-API-TRUTH-3 round 4: account_status=connected does NOT imply trade-allowed
// for investor accounts.
func TestBuildLiveContext_InvestorConnectedIsTradeAllowedFalse(t *testing.T) {
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
		return true, nil // account IS connected
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil // but it's an investor (read-only) account!
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil // connected proxy says true
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("buildLiveContext: %v", err)
	}
	if !lctx.IsConnected {
		t.Error("IsConnected=false, want true (account is connected)")
	}
	if lctx.IsTradeAllowed {
		t.Error("IsTradeAllowed=true, want false (investor/read-only account cannot trade)")
	}
}

// TestBuildLiveContext_NonInvestorConnectedIsTradeAllowedTrue verifies
// that a non-investor account with account_status=connected gets
// IsTradeAllowed=true (the normal case).
func TestBuildLiveContext_NonInvestorConnectedIsTradeAllowedTrue(t *testing.T) {
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
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil // not an investor — can trade
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("buildLiveContext: %v", err)
	}
	if !lctx.IsTradeAllowed {
		t.Error("IsTradeAllowed=false, want true (non-investor connected account can trade)")
	}
}

// TestBuildLiveContext_InvestorLookupQueryErrorBlocksExecution verifies
// that a DB error in accountIsInvestorLookup blocks execution.
func TestBuildLiveContext_InvestorLookupQueryErrorBlocksExecution(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	queryErr := errors.New("is_investor query failed")
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
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
		return false, queryErr // DB error!
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return true, nil
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("buildLiveContext should fail when is_investor lookup returns DB error")
	}
	if !errors.Is(err, queryErr) {
		t.Fatalf("error should wrap the query error, got: %v", err)
	}
}

// TestBuildLiveContext_LoginLookupQueryErrorBlocksExecution verifies
// that a DB error in accountLoginLookup blocks execution.
func TestBuildLiveContext_LoginLookupQueryErrorBlocksExecution(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	queryErr := errors.New("login query failed")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 0, queryErr // DB error, not real login=0!
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
		t.Fatal("buildLiveContext should fail when login lookup returns DB error")
	}
	if !errors.Is(err, queryErr) {
		t.Fatalf("error should wrap the query error, got: %v", err)
	}
}

// TestBuildLiveContext_BrokerCompanyQueryErrorBlocksExecution verifies
// that a DB error in brokerCompanyLookup blocks execution.
func TestBuildLiveContext_BrokerCompanyQueryErrorBlocksExecution(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	queryErr := errors.New("broker_company query failed")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "", queryErr // DB error, not real empty!
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil
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
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("buildLiveContext should fail when broker company lookup returns DB error")
	}
	if !errors.Is(err, queryErr) {
		t.Fatalf("error should wrap the query error, got: %v", err)
	}
}

// TestVMLiveSession_IsTradeAllowedFalseForInvestor verifies end-to-end
// that an investor account gets IsTradeAllowed=false in the VM.
func TestVMLiveSession_IsTradeAllowedFalseForInvestor(t *testing.T) {
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
		IsTradeAllowed: false, // investor account — trade not allowed
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
		t.Errorf("IsTradeAllowed() = %d, want 0 (false, investor account)", v.ToInt())
	}
}

// TestBuildLiveContext_RealFalseNotConfusedWithError verifies that a
// real false from a lookup (not an error) is correctly propagated as
// IsConnected=false, without blocking execution.
func TestBuildLiveContext_RealFalseNotConfusedWithError(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "ACME", nil
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil // real false, not error
	})
	srv.SetAccountConnectedLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil // real false — account is disconnected
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		return false, nil // real false — trade not allowed
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
	    return false, nil // not an investor
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("real false should not block execution, got: %v", err)
	}
	if lctx.IsConnected {
		t.Error("IsConnected=true, want false (real disconnected)")
	}
	if lctx.IsTradeAllowed {
		t.Error("IsTradeAllowed=true, want false (real not allowed)")
	}
}

// TestBuildLiveContext_PaperModeLookupErrorNonFatal verifies that in
// paper mode, lookup errors are non-fatal (fail-open for simulation).
func TestBuildLiveContext_PaperModeLookupErrorNonFatal(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, errors.New("DB error in paper mode")
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "paper", AccountID: "acct-1",
	}, bars, nil)
	if err != nil {
		t.Fatalf("paper mode should tolerate lookup errors, got: %v", err)
	}
	// IsDemo should be default false (zero value) since lookup failed.
	if lctx.IsDemo {
		t.Error("IsDemo=true, want false (default on lookup error in paper mode)")
	}
}

// TestBuildLiveContext_ErrorMessagesDistinguishQueryError verifies that
// error messages clearly indicate which lookup failed.
func TestBuildLiveContext_ErrorMessagesDistinguishQueryError(t *testing.T) {
	srv := newTestServerWithSnapshot("acct-1")
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 123456, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) (string, error) {
		return "ACME", nil
	})
	srv.SetAccountIsDemoLookup(func(_ context.Context, _ string) (bool, error) {
		return false, errors.New("pgx: no rows")
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
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("should fail with query error")
	}
	if !strings.Contains(err.Error(), "isDemo") {
		t.Fatalf("error should mention isDemo lookup, got: %s", err.Error())
	}
}
