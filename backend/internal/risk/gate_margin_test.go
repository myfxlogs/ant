package risk

import (
	"context"
	"strings"
	"sync/atomic"
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

func TestGateKillSwitch_BlocksLive(t *testing.T) {
	var ks atomic.Bool
	ks.Store(true)

	g := NewGate()
	g.SetKillSwitch(func() bool { return ks.Load() })

	decision := g.Evaluate(context.Background(), intentBuy("0.1"), defaultState())
	if decision.GetAllow() {
		t.Error("kill-switch should block live orders")
	}
	if decision.GetRuleHit() != "kill_switch" {
		t.Errorf("rule_hit = %q, want kill_switch", decision.GetRuleHit())
	}
}

func TestGateKillSwitch_AllowsSim(t *testing.T) {
	var ks atomic.Bool
	ks.Store(true)

	g := NewGate()
	g.SetKillSwitch(func() bool { return ks.Load() })

	decision := g.Evaluate(context.Background(), intentSim("0.1"), defaultState())
	if !decision.GetAllow() {
		t.Errorf("kill-switch should allow sim orders: %s", decision.GetReason())
	}
}

func TestGateKillSwitch_OffAllowsLive(t *testing.T) {
	var ks atomic.Bool
	ks.Store(false)

	g := NewGate()
	g.SetKillSwitch(func() bool { return ks.Load() })

	decision := g.Evaluate(context.Background(), intentBuy("0.1"), defaultState())
	if !decision.GetAllow() {
		t.Errorf("kill-switch off should allow: %s", decision.GetReason())
	}
}

// ── Gate: Autotrade Switch ────────────────────────────────────────────

func TestGateAutotrade_Block(t *testing.T) {
	g := NewGate()
	g.SetAutotradeEnabled(func(uid string) bool { return false })

	decision := g.Evaluate(context.Background(), intentBuy("0.1"), defaultState())
	if decision.GetAllow() {
		t.Error("autotrade disabled should block")
	}
	if decision.GetRuleHit() != "autotrade_disabled" {
		t.Errorf("rule_hit = %q, want autotrade_disabled", decision.GetRuleHit())
	}
}

func TestGateAutotrade_AllowsSim(t *testing.T) {
	g := NewGate()
	g.SetAutotradeEnabled(func(uid string) bool { return false })

	decision := g.Evaluate(context.Background(), intentSim("0.1"), defaultState())
	if !decision.GetAllow() {
		t.Errorf("autotrade disabled should still allow sim: %s", decision.GetReason())
	}
}

// ── Gate: Full Pipeline ───────────────────────────────────────────────

func TestGateAllRulesPass(t *testing.T) {
	g := newTestGate()
	// 0.01 lots: small enough to pass all rules (max lot=10, max exposure=50%, margin OK, etc.)
	i := intentBuy("0.01")
	decision := g.Evaluate(context.Background(), i, defaultState())
	if !decision.GetAllow() {
		t.Errorf("default gate should allow small order (0.01 lots): %s", decision.GetReason())
	}
}

func TestGateFirstRuleBlocks(t *testing.T) {
	// Max lot size = 10 → block 100 lots.
	g := newTestGate()
	decision := g.Evaluate(context.Background(), intentBuy("100.0"), defaultState())
	if decision.GetAllow() {
		t.Error("expected blocked by max_lot_size")
	}
	if decision.GetRuleHit() != "max_lot_size" {
		t.Errorf("rule_hit = %q, want max_lot_size", decision.GetRuleHit())
	}
}

func TestGateRulesInOrder(t *testing.T) {
	g := newTestGate()
	names := g.Rules()
	if len(names) != 10 {
		t.Errorf("expected 10 rules (R1-R9 + R4a/R4b as two), got %d: %v", len(names), names)
	}
	// Verify first and last rules are in spec order.
	if names[0] != "max_lot_size" {
		t.Errorf("first rule = %q, want max_lot_size", names[0])
	}
	if names[len(names)-1] != "margin_pre_check" {
		t.Errorf("last rule = %q, want margin_pre_check", names[len(names)-1])
	}
}

func TestGateAddRule(t *testing.T) {
	g := NewGate()
	g.AddRule(&MaxLotSize{MaxLots: decimal.NewFromInt(1)})
	if len(g.Rules()) != 1 {
		t.Error("expected 1 rule after AddRule")
	}
	decision := g.Evaluate(context.Background(), intentBuy("5.0"), nil)
	if decision.GetAllow() {
		t.Error("expected blocked by added rule")
	}
}

// ── Gate: Audit Entry ─────────────────────────────────────────────────

func TestAuditEntryString(t *testing.T) {
	entry := &AuditEntry{
		Intent:        intentBuy("0.1"),
		Decision:      &antv1.RiskDecision{Allow: true},
		EvaluatedAtMs: 1719000000000,
	}
	s := entry.String()
	if s == "" {
		t.Error("audit entry string is empty")
	}
}

func TestAuditEntryDeny(t *testing.T) {
	entry := &AuditEntry{
		Intent:   intentBuy("0.1"),
		Decision: &antv1.RiskDecision{Allow: false, RuleHit: "max_lot_size"},
	}
	s := entry.String()
	if s == "" {
		t.Error("audit deny entry string is empty")
	}
}

// ── Concurrency ─────────────────────────────────────────────────────────

func TestGateConcurrent(t *testing.T) {
	g := newTestGate()
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				g.Evaluate(context.Background(), intentBuy("0.1"), defaultState())
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
