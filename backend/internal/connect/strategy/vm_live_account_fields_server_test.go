// vm_live_account_fields_server_test.go — LIVE-ACCOUNT-FIELDS-1 server-side
// injection tests (2026-09-18).
//
// injectAccountTruth resolves leverage/currency/account-mode from one
// mt_accounts row (accountIdentityLookup): live mode fail-closes on lookup
// errors and on missing identity (leverage<=0 / empty currency); paper mode
// tolerates both. MT5 margin mode stays "" (AccMethod adapter not wired —
// VM consumers fail-closed).
//
// Adversarial proofs (M3/M4): delete the live completeness block → T4 RED;
// make accountModeForMTType return "" unconditionally → T3's mt4 assertion RED.
package strategy

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// newIdentityTestServer wires the full stub family plus a programmable
// identity lookup (mirrors the vm_trade_context6_batch2_test.go stub family).
func newIdentityTestServer(t *testing.T, ident *AccountIdentity, identErr error) *StrategyExecutionServer {
	t.Helper()
	srv := NewStrategyExecutionServer(nil, zap.NewNop())
	srv.posCache = NewPositionCache(nil)
	snap := buildTestSnapshot()
	srv.posCache.PutSnapshot(snap, snap.CapturedAt)
	srv.SetAccountLoginLookup(func(_ context.Context, _ string) (int64, error) {
		return 12345, nil
	})
	srv.SetBrokerCompanyLookup(func(_ context.Context, _ string) string {
		return "Exness"
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
		return false, nil
	})
	srv.SetAccountIdentityLookup(func(_ context.Context, _ string) (*AccountIdentity, error) {
		return ident, identErr
	})
	return srv
}

func identityBars() []liveBar {
	return []liveBar{{open: "1.0", high: "1.1", low: "0.9", close: "1.05", volume: "100", openTime: 1}}
}

// TestInjectAccountTruth_IdentityInjected — T3: live mode with a resolvable
// mt4 identity injects leverage/currency and derives margin mode from
// platform semantics.
//
// Adversarial (M4): accountModeForMTType returning "" unconditionally →
// AccountMode == "" → the mt4 assertion REDs.
func TestInjectAccountTruth_IdentityInjected(t *testing.T) {
	srv := newIdentityTestServer(t, &AccountIdentity{Leverage: 200, Currency: "USD", MTType: "mt4"}, nil)
	cfg := LiveStrategyConfig{AccountID: "acct-1", Symbol: "EURUSD", Timeframe: "M15", Mode: "live"}

	lctx, err := srv.buildLiveContext(context.Background(), cfg, identityBars(), nil)
	if err != nil {
		t.Fatalf("buildLiveContext failed: %v", err)
	}
	if lctx.Leverage != 200 {
		t.Errorf("Leverage = %d, want 200", lctx.Leverage)
	}
	if lctx.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", lctx.Currency)
	}
	if lctx.AccountMode != "hedging" {
		t.Errorf("AccountMode = %q, want hedging (MT4 is hedging-only platform semantics)", lctx.AccountMode)
	}
}

// TestInjectAccountTruth_MT5ModeUnknown — T3 (mt5 sub-case): MT5 margin mode
// is not surfaced by the adapter — AccountMode stays "" (no misjudgment).
func TestInjectAccountTruth_MT5ModeUnknown(t *testing.T) {
	srv := newIdentityTestServer(t, &AccountIdentity{Leverage: 200, Currency: "USD", MTType: "mt5"}, nil)
	cfg := LiveStrategyConfig{AccountID: "acct-1", Symbol: "EURUSD", Timeframe: "M15", Mode: "live"}

	lctx, err := srv.buildLiveContext(context.Background(), cfg, identityBars(), nil)
	if err != nil {
		t.Fatalf("buildLiveContext failed: %v", err)
	}
	if lctx.Leverage != 200 || lctx.Currency != "USD" {
		t.Fatalf("Leverage/Currency = %d/%q, want 200/USD", lctx.Leverage, lctx.Currency)
	}
	if lctx.AccountMode != "" {
		t.Fatalf("AccountMode = %q, want \"\" (MT5 margin mode unknown → VM fail-closed)", lctx.AccountMode)
	}
}

// TestInjectAccountTruth_LiveIdentityFailClosed — T4: live mode fails closed
// on missing leverage, missing currency, and lookup errors.
//
// Adversarial (M3): delete the live completeness block → all sub-cases
// succeed with fabricated zeros → RED.
func TestInjectAccountTruth_LiveIdentityFailClosed(t *testing.T) {
	t.Run("leverage zero errors", func(t *testing.T) {
		srv := newIdentityTestServer(t, &AccountIdentity{Leverage: 0, Currency: "USD", MTType: "mt4"}, nil)
		cfg := LiveStrategyConfig{AccountID: "acct-1", Symbol: "EURUSD", Timeframe: "M15", Mode: "live"}
		_, err := srv.buildLiveContext(context.Background(), cfg, identityBars(), nil)
		if err == nil || !strings.Contains(err.Error(), "leverage") {
			t.Fatalf("err = %v, want it to mention 'leverage' (leverage 0 = authoritative row never synced)", err)
		}
	})

	t.Run("empty currency errors", func(t *testing.T) {
		srv := newIdentityTestServer(t, &AccountIdentity{Leverage: 200, Currency: "", MTType: "mt4"}, nil)
		cfg := LiveStrategyConfig{AccountID: "acct-1", Symbol: "EURUSD", Timeframe: "M15", Mode: "live"}
		_, err := srv.buildLiveContext(context.Background(), cfg, identityBars(), nil)
		if err == nil || !strings.Contains(err.Error(), "currency") {
			t.Fatalf("err = %v, want it to mention 'currency'", err)
		}
	})

	t.Run("lookup error errors", func(t *testing.T) {
		srv := newIdentityTestServer(t, nil, errors.New("db down"))
		cfg := LiveStrategyConfig{AccountID: "acct-1", Symbol: "EURUSD", Timeframe: "M15", Mode: "live"}
		_, err := srv.buildLiveContext(context.Background(), cfg, identityBars(), nil)
		if err == nil || !strings.Contains(err.Error(), "account identity lookup failed") {
			t.Fatalf("err = %v, want it to contain 'account identity lookup failed'", err)
		}
	})
}

// TestInjectAccountTruth_PaperTolerates — T5: paper mode tolerates lookup
// errors and missing identity (fail-open for simulation), fields stay zero.
func TestInjectAccountTruth_PaperTolerates(t *testing.T) {
	srv := newIdentityTestServer(t, nil, errors.New("db down"))
	cfg := LiveStrategyConfig{AccountID: "acct-1", Symbol: "EURUSD", Timeframe: "M15", Mode: "paper"}

	lctx, err := srv.buildLiveContext(context.Background(), cfg, identityBars(), nil)
	if err != nil {
		t.Fatalf("buildLiveContext failed: %v, want nil (paper tolerates lookup errors)", err)
	}
	if lctx.Leverage != 0 || lctx.Currency != "" || lctx.AccountMode != "" {
		t.Fatalf("identity = %d/%q/%q, want zero values (paper fail-open)", lctx.Leverage, lctx.Currency, lctx.AccountMode)
	}
}
