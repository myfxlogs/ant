package strategy

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// ── VM-API-TRUTH-3 round 5: accountIsInvestorLookup required in live mode,
// IsTradeAllowed not derived from connected status ──

// TestBuildLiveContext_MissingInvestorLookupRejected verifies that live mode
// without accountIsInvestorLookup is rejected (investor safety gate bypassed).
func TestBuildLiveContext_MissingInvestorLookupRejected(t *testing.T) {
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
		return true, nil
	})
	// accountIsInvestorLookup intentionally NOT configured
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("buildLiveContext should reject live mode without accountIsInvestorLookup")
	}
	if !strings.Contains(err.Error(), "accountIsInvestorLookup not configured") {
		t.Fatalf("error should mention 'accountIsInvestorLookup not configured', got: %s", err.Error())
	}
}

// TestBuildLiveContext_InvestorLookupRequiredEvenWhenTradeAllowedFalse verifies
// that the investor lookup is required even when tradeAllowed is already false.
// This prevents a future change to tradeAllowed from bypassing the investor gate.
func TestBuildLiveContext_InvestorLookupRequiredEvenWhenTradeAllowedFalse(t *testing.T) {
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
		return false, nil // already false
	})
	// accountIsInvestorLookup intentionally NOT configured
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("buildLiveContext should reject even when tradeAllowed=false without investor lookup")
	}
	if !strings.Contains(err.Error(), "accountIsInvestorLookup not configured") {
		t.Fatalf("error should mention 'accountIsInvestorLookup not configured', got: %s", err.Error())
	}
}

// TestBuildLiveContext_InvestorLookupQueryErrorBlocksExecution verifies that
// a DB error in accountIsInvestorLookup blocks execution (fail-closed).
func TestBuildLiveContext_InvestorLookupQueryErrorBlocksExecution_Round5(t *testing.T) {
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
		return true, nil
	})
	srv.SetAccountIsInvestorLookup(func(_ context.Context, _ string) (bool, error) {
		return false, fmt.Errorf("simulated DB error (sentinel: INVESTOR_LOOKUP_FAIL_5F3A)")
	})
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	_, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{
		Symbol: "EURUSD", Timeframe: "M5", Mode: "live", AccountID: "acct-1",
	}, bars, nil)
	if err == nil {
		t.Fatal("buildLiveContext should fail when is_investor lookup returns DB error")
	}
	if !strings.Contains(err.Error(), "INVESTOR_LOOKUP_FAIL_5F3A") {
		t.Fatalf("error should contain sentinel, got: %s", err.Error())
	}
}

// TestBuildLiveContext_TradeAllowedNotFromConnected verifies that
// IsTradeAllowed is NOT derived from account_status == 'connected'.
// The test uses a tradeAllowedLookup that returns false even when connected=true,
// proving that connected status alone does not determine trade permission.
func TestBuildLiveContext_TradeAllowedNotFromConnected(t *testing.T) {
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
		return false, nil // not an investor
	})
	srv.SetAccountTradeAllowedLookup(func(_ context.Context, _ string) (bool, error) {
		// VM-API-TRUTH-3 round 5: trade permission is NOT derived from connected.
		// This lookup returns false even though connected=true, proving that
		// connected status alone does not determine trade permission.
		return false, nil
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
		t.Error("IsTradeAllowed=true, want false (trade permission is independent of connected status)")
	}
}
