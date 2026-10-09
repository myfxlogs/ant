package risk

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func TestMarginPreCheck_Allow(t *testing.T) {
	r := &MarginPreCheck{MaxMarginRatio: decimal.NewFromFloat(0.80)}
	state := defaultState()
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestMarginPreCheck_Block(t *testing.T) {
	r := &MarginPreCheck{MaxMarginRatio: decimal.NewFromFloat(0.80)}
	state := &AccountState{
		Equity:         decimal.NewFromInt(100),
		UsedMargin:     decimal.NewFromInt(80),
		SymbolLeverage: 100,
		ContractSize:   decimal.NewFromInt(100000),
	}
	result := r.Check(context.Background(), intentBuy("1.0"), state)
	if result.Allowed {
		t.Error("expected blocked for excessive margin")
	}
}

// Market orders without a broker margin capability retain the legacy price-unknown path.

// Market orders without a broker margin capability retain the legacy price-unknown path.
func TestMarginPreCheck_MarketOrderPriceZero_Skips(t *testing.T) {
	r := &MarginPreCheck{MaxMarginRatio: decimal.NewFromFloat(0.80)}
	state := &AccountState{
		Equity:         decimal.NewFromInt(100),
		UsedMargin:     decimal.NewFromInt(80),
		SymbolLeverage: 100,
		ContractSize:   decimal.NewFromInt(100000),
	}
	intent := intentBuy("1.0")
	intent.Price = "0" // market order — no price resolved
	result := r.Check(context.Background(), intent, state)
	if !result.Allowed {
		t.Errorf("expected skip (allowed) for market order with price=0, got: %s", result.Reason)
	}
}

// RISK-MARGIN1: when the caller resolves a market order price from TickBroker,
// the margin rule evaluates correctly and can block excessive margin.

// RISK-MARGIN1: when the caller resolves a market order price from TickBroker,
// the margin rule evaluates correctly and can block excessive margin.
func TestMarginPreCheck_MarketOrderPriceResolved_Blocks(t *testing.T) {
	r := &MarginPreCheck{MaxMarginRatio: decimal.NewFromFloat(0.80)}
	state := &AccountState{
		Equity:         decimal.NewFromInt(100),
		UsedMargin:     decimal.NewFromInt(80),
		SymbolLeverage: 100,
		ContractSize:   decimal.NewFromInt(100000),
	}
	intent := intentBuy("1.0")
	intent.Price = "1.08500" // price resolved from TickBroker mid-price
	result := r.Check(context.Background(), intent, state)
	if result.Allowed {
		t.Error("expected blocked for excessive margin with resolved market price")
	}
}

// MARGIN-GATE adversarial: BTCUSDm (crypto-like, CS=1), USDJPY (USD-base, CS=100000),
// EURUSD (FX, CS=100000) and fail-closed on missing ContractSize.

// MARGIN-GATE adversarial: BTCUSDm (crypto-like, CS=1), USDJPY (USD-base, CS=100000),
// EURUSD (FX, CS=100000) and fail-closed on missing ContractSize.
func TestMarginPreCheck_MT4DefersToBroker(t *testing.T) {
	r := &MarginPreCheck{MaxMarginRatio: decimal.NewFromFloat(0.80)}
	state := &AccountState{Platform: "mt4", Equity: decimal.NewFromInt(1), UsedMargin: decimal.NewFromInt(1)}
	intent := &antv1.OrderIntent{Symbol: "EURUSD", Side: "buy", Volume: "100", Price: "1.1", Type: "buy"}
	result := r.Check(context.Background(), intent, state)
	if !result.Allowed || !strings.Contains(result.Reason, "broker remains authoritative") {
		t.Fatalf("MT4 without RequiredMargin must defer to broker, got allow=%v reason=%q", result.Allowed, result.Reason)
	}
}

func TestMarginPreCheck_UsesBrokerRequiredMargin(t *testing.T) {
	r := &MarginPreCheck{MaxMarginRatio: decimal.NewFromFloat(0.80)}
	state := &AccountState{
		Equity: decimal.NewFromInt(100), UsedMargin: decimal.NewFromInt(50),
		BrokerMarginAvailable: true, RequiredMarginKnown: true, RequiredMargin: decimal.NewFromInt(20),
	}
	intent := &antv1.OrderIntent{Symbol: "EURUSD", Side: "buy", Volume: "1", Price: "0", Type: "buy"}
	if result := r.Check(context.Background(), intent, state); !result.Allowed {
		t.Fatalf("broker required margin should allow 90%% total margin, got: %s", result.Reason)
	}
	state.RequiredMarginKnown = false
	if result := r.Check(context.Background(), intent, state); result.Allowed {
		t.Fatal("missing broker required margin must fail closed")
	}
}

func TestMarginPreCheck_AdversarialSymbols(t *testing.T) {
	r := &MarginPreCheck{MaxMarginRatio: decimal.NewFromFloat(0.80)}

	// BTCUSDm 0.01 lot @ 63300, CS=1, lev=100 → required ≈ 6.33
	btc := &antv1.OrderIntent{Symbol: "BTCUSDm", Side: "buy", Volume: "0.01", Price: "63300", Type: "buy"}
	btcState := &AccountState{Equity: decimal.NewFromInt(10000), UsedMargin: decimal.Zero, SymbolLeverage: 100, ContractSize: decimal.NewFromInt(1)}
	if res := r.Check(context.Background(), btc, btcState); !res.Allowed {
		t.Errorf("BTCUSDm 0.01 lot should be allowed, got: %s", res.Reason)
	}

	// USDJPY 0.01 lot @ 150.0, CS=100000, lev=10 (USD-base, fx_rate=1) → required ≈ 100
	usdjpy := &antv1.OrderIntent{Symbol: "USDJPY", Side: "buy", Volume: "0.01", Price: "150.0", Type: "buy"}
	usdjpyState := &AccountState{Equity: decimal.NewFromInt(10000), UsedMargin: decimal.Zero, SymbolLeverage: 10, ContractSize: decimal.NewFromInt(100000)}
	if res := r.Check(context.Background(), usdjpy, usdjpyState); !res.Allowed {
		t.Errorf("USDJPY 0.01 lot should be allowed, got: %s", res.Reason)
	}

	// EURUSD 0.01 lot @ 1.0850, CS=100000, lev=100 → required ≈ 10.85
	eur := &antv1.OrderIntent{Symbol: "EURUSD", Side: "buy", Volume: "0.01", Price: "1.0850", Type: "buy"}
	eurState := &AccountState{Equity: decimal.NewFromInt(10000), UsedMargin: decimal.Zero, SymbolLeverage: 100, ContractSize: decimal.NewFromInt(100000)}
	if res := r.Check(context.Background(), eur, eurState); !res.Allowed {
		t.Errorf("EURUSD 0.01 lot should be allowed, got: %s", res.Reason)
	}

	// Missing ContractSize → fail-closed with explicit reason.
	missing := &antv1.OrderIntent{Symbol: "EURUSD", Side: "buy", Volume: "0.01", Price: "1.0850", Type: "buy"}
	missingState := &AccountState{Equity: decimal.NewFromInt(10000), UsedMargin: decimal.Zero, SymbolLeverage: 100}
	res := r.Check(context.Background(), missing, missingState)
	if res.Allowed {
		t.Error("expected blocked when ContractSize is missing")
	}
	if !strings.Contains(res.Reason, "contract size unknown for symbol EURUSD") {
		t.Errorf("expected contract size unknown reason, got: %s", res.Reason)
	}
}

// ── Gate: Kill-Switch ─────────────────────────────────────────────────
