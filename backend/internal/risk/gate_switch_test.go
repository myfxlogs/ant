package risk

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

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

// RISK-DEDUP-KEY-1 (gate layer): two same-param orders with distinct comments
// are distinct intents (grid/pyramid strategies place same-price orders with
// different comments); an identical retry (same comment) must still block.
// Adversarial: remove Comment from the key → first case fails → RED.
