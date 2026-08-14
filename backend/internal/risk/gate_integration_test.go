// D6-A integration tests: gate wiring, fail-closed, non-bypassable.
package risk

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// ── D6-A: Fail-closed on nil state ────────────────────────────────────

func TestGateFailClosedNilState(t *testing.T) {
	g := newTestGate()

	intent := &antv1.OrderIntent{
		UserId:    "user-1",
		AccountId: "acct-1",
		Symbol:    "EURUSD",
		Side:      "buy",
		Volume:    "0.01",
		Type:      "buy",
		Source:    antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
	}

	// nil state → blocked.
	decision := g.Evaluate(context.Background(), intent, nil)
	if decision.GetAllow() {
		t.Error("nil AccountState should block live orders (fail-closed)")
	}
	if decision.GetRuleHit() != "account_state_missing" {
		t.Errorf("rule_hit = %q, want account_state_missing", decision.GetRuleHit())
	}
}

func TestGateFailClosedNegativeEquity(t *testing.T) {
	g := newTestGate()

	intent := &antv1.OrderIntent{
		UserId:    "user-1",
		AccountId: "acct-1",
		Symbol:    "EURUSD",
		Side:      "buy",
		Volume:    "0.01",
		Type:      "buy",
		Source:    antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
	}

	// Negative equity = provider not connected sentinel → blocked.
	state := &AccountState{
		Equity: decimal.NewFromInt(-1),
	}
	decision := g.Evaluate(context.Background(), intent, state)
	if decision.GetAllow() {
		t.Error("negative equity sentinel should block live orders (fail-closed)")
	}
}

func TestGateAllowsSimWithNilState(t *testing.T) {
	g := newTestGate()

	intent := &antv1.OrderIntent{
		UserId:    "user-1",
		AccountId: "acct-1",
		Symbol:    "EURUSD",
		Side:      "buy",
		Volume:    "0.01",
		Type:      "buy",
		Source:    antv1.OrderIntentSource_ORDER_INTENT_SOURCE_SIM,
	}

	// SIM orders should NOT be blocked by nil state (backtest doesn't need live account data).
	decision := g.Evaluate(context.Background(), intent, nil)
	if !decision.GetAllow() {
		t.Errorf("nil state should NOT block sim orders: %s", decision.GetReason())
	}
}

// ── D6-A: Gate blocks oversized orders ────────────────────────────────

func TestGateIntegrationBlocksOversized(t *testing.T) {
	g := newTestGate()

	state := &AccountState{
		Balance:        decimal.NewFromInt(100000),
		Equity:         decimal.NewFromInt(100000),
		FreeMargin:     decimal.NewFromInt(95000),
		UsedMargin:     decimal.NewFromInt(5000),
		SymbolLeverage: 100,
	}

	// 100 lots exceeds max_lot_size (10) → blocked.
	intent := &antv1.OrderIntent{
		UserId:    "user-1",
		AccountId: "acct-1",
		Symbol:    "EURUSD",
		Side:      "buy",
		Volume:    "100.0",
		Type:      "buy",
		Price:     "1.08500",
		Source:    antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
	}

	decision := g.Evaluate(context.Background(), intent, state)
	if decision.GetAllow() {
		t.Error("100 lots should be blocked by max_lot_size")
	}
	if decision.GetRuleHit() != "max_lot_size" {
		t.Errorf("rule_hit = %q, want max_lot_size", decision.GetRuleHit())
	}
}

// ── D6-A: Kill-switch blocks everything live ──────────────────────────

func TestGateIntegrationKillSwitchLiveOnly(t *testing.T) {
	g := newTestGate()
	var ks atomic.Bool
	ks.Store(true)
	g.SetKillSwitch(func() bool { return ks.Load() })

	state := &AccountState{
		Balance:        decimal.NewFromInt(100000),
		Equity:         decimal.NewFromInt(100000),
		SymbolLeverage: 100,
	}

	liveIntent := &antv1.OrderIntent{
		UserId: "user-1", AccountId: "acct-1", Symbol: "EURUSD",
		Side: "buy", Volume: "0.01", Type: "buy",
		Source: antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
	}
	simIntent := &antv1.OrderIntent{
		UserId: "user-1", AccountId: "acct-1", Symbol: "EURUSD",
		Side: "buy", Volume: "0.01", Type: "buy",
		Source: antv1.OrderIntentSource_ORDER_INTENT_SOURCE_SIM,
	}

	liveDecision := g.Evaluate(context.Background(), liveIntent, state)
	simDecision := g.Evaluate(context.Background(), simIntent, state)

	if liveDecision.GetAllow() {
		t.Error("kill-switch should block live orders")
	}
	if !simDecision.GetAllow() {
		t.Error("kill-switch should NOT block sim orders")
	}
}

// ── D6-A: Audit trail for all decisions ───────────────────────────────

func TestGateAuditTrail(t *testing.T) {
	g := newTestGate()
	state := &AccountState{
		Balance:        decimal.NewFromInt(100000),
		Equity:         decimal.NewFromInt(100000),
		FreeMargin:     decimal.NewFromInt(95000),
		SymbolLeverage: 100,
	}

	// Allowed order.
	intent := &antv1.OrderIntent{
		UserId: "user-1", AccountId: "acct-1", Symbol: "EURUSD",
		Side: "buy", Volume: "0.01", Type: "buy",
		Price:  "1.08500",
		Source: antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
	}

	decision := g.Evaluate(context.Background(), intent, state)
	entry := &AuditEntry{
		Intent:        intent,
		Decision:      decision,
		EvaluatedAtMs: 1719000000000,
	}

	audit := entry.String()
	if audit == "" {
		t.Error("audit entry string is empty")
	}
	// Verify it contains key information.
	if decision.GetAllow() {
		if !contains(audit, "ALLOW") {
			t.Errorf("audit entry missing ALLOW: %s", audit)
		}
	}
}

// ── D6-A: SimBroker offline gate batch pre-check ──────────────────────

func TestGateOfflineBatchPreCheck(t *testing.T) {
	g := newTestGate()
	state := &AccountState{
		Balance:        decimal.NewFromInt(100000),
		Equity:         decimal.NewFromInt(100000),
		FreeMargin:     decimal.NewFromInt(95000),
		SymbolLeverage: 100,
		ContractSize:   decimal.NewFromInt(100000),
	}

	// Simulate a batch of backtest intents for offline pre-check.
	intents := []*antv1.OrderIntent{
		{UserId: "u1", AccountId: "a1", Symbol: "EURUSD", Side: "buy",
			Volume: "0.10", Type: "buy", Price: "1.08500",
			Source: antv1.OrderIntentSource_ORDER_INTENT_SOURCE_SIM},
		{UserId: "u1", AccountId: "a1", Symbol: "EURUSD", Side: "sell",
			Volume: "0.10", Type: "sell", Price: "1.09000",
			Source: antv1.OrderIntentSource_ORDER_INTENT_SOURCE_SIM},
		{UserId: "u1", AccountId: "a1", Symbol: "EURUSD", Side: "buy",
			Volume: "100.0", Type: "buy", Price: "1.10000", // ← should be blocked
			Source: antv1.OrderIntentSource_ORDER_INTENT_SOURCE_SIM},
	}

	results := make([]*antv1.RiskDecision, len(intents))
	allowed := 0
	blocked := 0

	for i, intent := range intents {
		results[i] = g.Evaluate(context.Background(), intent, state)
		if results[i].GetAllow() {
			allowed++
		} else {
			blocked++
		}
	}

	if allowed != 2 {
		t.Errorf("expected 2 allowed, got %d", allowed)
	}
	if blocked != 1 {
		t.Errorf("expected 1 blocked, got %d", blocked)
	}
	// The blocked one should be max_lot_size.
	if results[2].GetRuleHit() != "max_lot_size" {
		t.Errorf("3rd intent rule_hit = %q, want max_lot_size", results[2].GetRuleHit())
	}
}

// ── D6-A: All 11 rule names ───────────────────────────────────────────

func TestGateAllRuleNames(t *testing.T) {
	g := newTestGate()
	names := g.Rules()

	expected := []string{
		"max_lot_size",
		"max_position_count",
		"max_exposure",
		"daily_loss",
		"drawdown",
		"symbol_whitelist",
		"leverage_cap",
		"order_frequency",
		"duplicate_protection",
		"margin_pre_check",
	}

	if len(names) != len(expected) {
		t.Errorf("expected %d rules, got %d: %v", len(expected), len(names), names)
	}

	for i, name := range names {
		if name != expected[i] {
			t.Errorf("rule[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

// ── D6-A: OrderIntent field mapping ───────────────────────────────────

func TestOrderIntentFieldMapping(t *testing.T) {
	intent := &antv1.OrderIntent{
		UserId:    "user-1",
		AccountId: "acct-1",
		Symbol:    "EURUSD",
		Side:      "buy",
		Volume:    "0.10",
		Type:      "buy_limit",
		Price:     "1.08000",
		Sl:        "1.07500",
		Tp:        "1.09500",
		Magic:     42,
		Source:    antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
	}

	g := newTestGate()
	state := &AccountState{
		Balance:        decimal.NewFromInt(100000),
		Equity:         decimal.NewFromInt(100000),
		FreeMargin:     decimal.NewFromInt(95000),
		SymbolLeverage: 100,
		ContractSize:   decimal.NewFromInt(100000),
	}

	decision := g.Evaluate(context.Background(), intent, state)
	if !decision.GetAllow() {
		t.Errorf("small limit order should be allowed: %s", decision.GetReason())
	}
}

// ── Helpers ────────────────────────────────────────────────────────────

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ── D6-A: Concurrent gate evaluation does not deadlock ─────────────────

func TestGateIntegrationConcurrentBatch(t *testing.T) {
	g := newTestGate()
	state := &AccountState{
		Balance:        decimal.NewFromInt(100000),
		Equity:         decimal.NewFromInt(100000),
		SymbolLeverage: 100,
		ContractSize:   decimal.NewFromInt(100000),
	}

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				intent := &antv1.OrderIntent{
					UserId: "user-1", AccountId: "acct-1", Symbol: "EURUSD",
					Side: "buy", Volume: "0.01", Type: "buy",
					Source: antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
				}
				g.Evaluate(context.Background(), intent, state)
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
