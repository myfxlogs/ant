package risk

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	antv1 "anttrader/gen/proto/ant/v1"
)

// ── Test helpers ───────────────────────────────────────────────────────

func intentBuy(vol string) *antv1.OrderIntent {
	return &antv1.OrderIntent{
		UserId:    "user-1",
		AccountId: "acct-1",
		Symbol:    "EURUSD",
		Side:      "buy",
		Volume:    vol,
		Type:      "market",
		Price:     "1.08500",
		Magic:     1,
		Source:    antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
	}
}

func intentSell(vol string) *antv1.OrderIntent {
	i := intentBuy(vol)
	i.Side = "sell"
	return i
}

func intentSim(vol string) *antv1.OrderIntent {
	i := intentBuy(vol)
	i.Source = antv1.OrderIntentSource_ORDER_INTENT_SOURCE_SIM
	return i
}

func defaultState() *AccountState {
	return &AccountState{
		Balance:        decimal.NewFromInt(10000),
		Equity:         decimal.NewFromInt(10050),
		FreeMargin:     decimal.NewFromInt(9550),
		UsedMargin:     decimal.NewFromInt(500),
		OpenPositions:  1,
		DailyPnL:       decimal.NewFromInt(50),
		PeakEquity:     decimal.NewFromInt(10100),
		SymbolLeverage: 100,
	}
}

// ── R1: Max Lot Size ──────────────────────────────────────────────────

func TestMaxLotSize_Allow(t *testing.T) {
	r := &MaxLotSize{MaxLots: decimal.NewFromInt(10)}
	result := r.Check(context.Background(), intentBuy("5.0"), nil)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestMaxLotSize_Block(t *testing.T) {
	r := &MaxLotSize{MaxLots: decimal.NewFromInt(10)}
	result := r.Check(context.Background(), intentBuy("15.0"), nil)
	if result.Allowed {
		t.Error("expected blocked for 15 lots")
	}
}

func TestMaxLotSize_SuggestsAdjustment(t *testing.T) {
	r := &MaxLotSize{MaxLots: decimal.NewFromInt(10)}
	result := r.Check(context.Background(), intentBuy("50.0"), nil)
	if result.AdjustedVolume.IsZero() {
		t.Error("expected adjusted volume suggestion")
	}
}

// ── R2: Max Position Count ────────────────────────────────────────────

func TestMaxPositionCount_Allow(t *testing.T) {
	r := &MaxPositionCount{Max: 20}
	state := &AccountState{OpenPositions: 5}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestMaxPositionCount_Block(t *testing.T) {
	r := &MaxPositionCount{Max: 20}
	state := &AccountState{OpenPositions: 20}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if result.Allowed {
		t.Error("expected blocked when at max positions")
	}
}

// ── R3: Max Exposure ──────────────────────────────────────────────────

func TestMaxExposure_Allow(t *testing.T) {
	r := &MaxExposure{MaxRatio: decimal.NewFromFloat(0.5)}
	// 0.01 lots × 1.08500 × 100000 = 1085. 1085 < 5000 (50% of 10000) → allowed.
	state := &AccountState{Balance: decimal.NewFromInt(10000)}
	result := r.Check(context.Background(), intentBuy("0.01"), state)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestMaxExposure_Block(t *testing.T) {
	r := &MaxExposure{MaxRatio: decimal.NewFromFloat(0.5)}
	state := &AccountState{Balance: decimal.NewFromInt(10000)}
	result := r.Check(context.Background(), intentBuy("100.0"), state)
	if result.Allowed {
		t.Error("expected blocked for 100 lots")
	}
}

// ── R4a: Daily Loss Breaker ───────────────────────────────────────────

func TestDailyLoss_Allow(t *testing.T) {
	r := &DailyLossBreaker{MaxDailyLoss: decimal.NewFromInt(500)}
	state := &AccountState{DailyPnL: decimal.NewFromInt(-100)}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestDailyLoss_Block(t *testing.T) {
	r := &DailyLossBreaker{MaxDailyLoss: decimal.NewFromInt(500)}
	state := &AccountState{DailyPnL: decimal.NewFromInt(-600)}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if result.Allowed {
		t.Error("expected blocked when daily loss exceeds limit")
	}
}

func TestDailyLoss_Disabled(t *testing.T) {
	r := &DailyLossBreaker{MaxDailyLoss: decimal.Zero}
	state := &AccountState{DailyPnL: decimal.NewFromInt(-99999)}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if !result.Allowed {
		t.Errorf("expected allowed when disabled, got: %s", result.Reason)
	}
}

// ── R4b: Drawdown Breaker ─────────────────────────────────────────────

func TestDrawdown_Allow(t *testing.T) {
	r := &DrawdownBreaker{MaxDrawdownPct: decimal.NewFromFloat(0.30)}
	state := &AccountState{Equity: decimal.NewFromInt(9000), PeakEquity: decimal.NewFromInt(10000)}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if !result.Allowed {
		t.Errorf("expected allowed (10%% DD), got: %s", result.Reason)
	}
}

func TestDrawdown_Block(t *testing.T) {
	r := &DrawdownBreaker{MaxDrawdownPct: decimal.NewFromFloat(0.30)}
	state := &AccountState{Equity: decimal.NewFromInt(6000), PeakEquity: decimal.NewFromInt(10000)}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if result.Allowed {
		t.Error("expected blocked at 40% drawdown")
	}
}

// ── R5: Symbol Whitelist ──────────────────────────────────────────────

func TestSymbolWhitelist_Allow(t *testing.T) {
	r := &SymbolWhitelist{Whitelist: []string{"EURUSD", "GBPUSD"}}
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestSymbolWhitelist_Block(t *testing.T) {
	r := &SymbolWhitelist{Whitelist: []string{"GBPUSD"}}
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if result.Allowed {
		t.Error("expected blocked — EURUSD not in whitelist")
	}
}

func TestSymbolWhitelist_EmptyAllowsAll(t *testing.T) {
	r := &SymbolWhitelist{}
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if !result.Allowed {
		t.Errorf("empty whitelist should allow all")
	}
}

// ── R6: Leverage Cap ──────────────────────────────────────────────────

func TestLeverageCap_Allow(t *testing.T) {
	r := &LeverageCap{MaxLeverage: 500}
	state := &AccountState{SymbolLeverage: 100}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestLeverageCap_Block(t *testing.T) {
	r := &LeverageCap{MaxLeverage: 500}
	state := &AccountState{SymbolLeverage: 1000}
	result := r.Check(context.Background(), intentBuy("0.1"), state)
	if result.Allowed {
		t.Error("expected blocked for 1000x leverage")
	}
}

// ── R7: Order Frequency Limit ─────────────────────────────────────────

func TestOrderFrequency_Allow(t *testing.T) {
	r := &OrderFrequencyLimit{MaxOrders: 60, Window: time.Minute}
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestOrderFrequency_Block(t *testing.T) {
	r := &OrderFrequencyLimit{MaxOrders: 2, Window: time.Hour}
	r.Check(context.Background(), intentBuy("0.1"), nil)
	r.Check(context.Background(), intentBuy("0.1"), nil)
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if result.Allowed {
		t.Error("expected blocked after 2 orders")
	}
}

// ── R8: Duplicate Protection ──────────────────────────────────────────

func TestDuplicateProtection_Allow(t *testing.T) {
	r := &DuplicateProtection{DedupWindow: 5 * time.Second}
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if !result.Allowed {
		t.Errorf("expected allowed, got: %s", result.Reason)
	}
}

func TestDuplicateProtection_Block(t *testing.T) {
	r := &DuplicateProtection{DedupWindow: time.Hour}
	r.Check(context.Background(), intentBuy("0.1"), nil)
	// Same intent within dedup window.
	result := r.Check(context.Background(), intentBuy("0.1"), nil)
	if result.Allowed {
		t.Error("expected duplicate blocked")
	}
}

func TestDuplicateProtection_DifferentVolAllowed(t *testing.T) {
	r := &DuplicateProtection{DedupWindow: time.Hour}
	r.Check(context.Background(), intentBuy("0.1"), nil)
	result := r.Check(context.Background(), intentBuy("0.2"), nil)
	if !result.Allowed {
		t.Errorf("different volume should be allowed")
	}
}

// ── R9: Margin Pre-Check ──────────────────────────────────────────────

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
	}
	result := r.Check(context.Background(), intentBuy("1.0"), state)
	if result.Allowed {
		t.Error("expected blocked for excessive margin")
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
	g := NewDefaultGate()
	// 0.01 lots: small enough to pass all rules (max lot=10, max exposure=50%, margin OK, etc.)
	i := intentBuy("0.01")
	decision := g.Evaluate(context.Background(), i, defaultState())
	if !decision.GetAllow() {
		t.Errorf("default gate should allow small order (0.01 lots): %s", decision.GetReason())
	}
}

func TestGateFirstRuleBlocks(t *testing.T) {
	// Max lot size = 10 → block 100 lots.
	g := NewDefaultGate()
	decision := g.Evaluate(context.Background(), intentBuy("100.0"), defaultState())
	if decision.GetAllow() {
		t.Error("expected blocked by max_lot_size")
	}
	if decision.GetRuleHit() != "max_lot_size" {
		t.Errorf("rule_hit = %q, want max_lot_size", decision.GetRuleHit())
	}
}

func TestGateRulesInOrder(t *testing.T) {
	g := NewDefaultGate()
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
	g := NewDefaultGate()
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
