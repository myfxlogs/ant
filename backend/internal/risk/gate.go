// Package risk implements the Risk Gate protocol (T3.2 / T0.3).
//
// The gate is the single, non-bypassable money-safety boundary.  Every
// OrderIntent — from both SimBroker and LiveBroker — passes through it
// before reaching any broker.  Rules are evaluated in order; the first
// BLOCK stops the pipeline.
//
// Architecture:
//
//	Strategy → broker.order_send()
//	              │
//	              ▼
//	         Gate.Evaluate(intent, state) → RiskDecision
//	              │
//	         ┌────┴────┐
//	         ALLOW     DENY → audit log + metric
//	         ▼
//	       broker
//
// Reuses risksvc.HardLimit (kill-switch, margin floor, KYC) and
// risksvc.Engine (max position, daily loss, drawdown, session, margin,
// symbol whitelist) where semantics align.  Adds rules unique to EA
// execution: max lot size, leverage cap, order frequency, duplicate
// protection.
package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	antv1 "anttrader/gen/proto/ant/v1"
)

// ── Account state snapshot ────────────────────────────────────────────

// AccountState carries the account context needed for rule evaluation.
// All monetary fields are decimal.Decimal per CLAUDE.md.
type AccountState struct {
	Balance        decimal.Decimal
	Equity         decimal.Decimal
	FreeMargin     decimal.Decimal
	UsedMargin     decimal.Decimal
	OpenPositions  int
	DailyPnL       decimal.Decimal
	PeakEquity     decimal.Decimal
	SymbolLeverage int
}

// ── Rule interface ────────────────────────────────────────────────────

// Rule is a single risk check.  Name must be unique.
type Rule interface {
	Name() string
	Check(ctx context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult
}

// RuleResult is the outcome of a single rule check.
type RuleResult struct {
	Allowed         bool
	Reason          string
	AdjustedVolume  decimal.Decimal // zero = no adjustment
}

// ── Gate ──────────────────────────────────────────────────────────────

// Gate evaluates OrderIntent against all registered rules.
// Safe for concurrent use.
type Gate struct {
	mu    sync.RWMutex
	rules []Rule

	// killSwitch is a function that returns true when the global kill-switch is active.
	killSwitch func() bool

	// autotradeEnabled is a function that returns the per-user autotrade setting.
	autotradeEnabled func(userID string) bool
}

// NewGate creates a Gate with the given rules.
func NewGate(rules ...Rule) *Gate {
	g := &Gate{
		rules:            rules,
		killSwitch:       func() bool { return false },
		autotradeEnabled: func(string) bool { return true },
	}
	return g
}

// SetKillSwitch sets the kill-switch function (may be toggled at runtime).
func (g *Gate) SetKillSwitch(fn func() bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.killSwitch = fn
}

// SetAutotradeEnabled sets the per-user autotrade function.
func (g *Gate) SetAutotradeEnabled(fn func(userID string) bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.autotradeEnabled = fn
}

// Evaluate runs all rules against an OrderIntent.  Returns the RiskDecision
// proto — ready to be returned to the caller or logged for audit.
//
// D6-A / T3.2b fail-closed: if AccountState is nil or has negative equity
// (sentinel for "provider not connected"), equity-dependent rules (R3, R4a,
// R4b, R9) will deny.  This ensures live trading is blocked until real
// account data is available — never fall through with defaults.
func (g *Gate) Evaluate(ctx context.Context, intent *antv1.OrderIntent, state *AccountState) *antv1.RiskDecision {
	g.mu.RLock()
	rules := g.rules
	killSwitch := g.killSwitch
	autotrade := g.autotradeEnabled
	g.mu.RUnlock()

	// R10: Global kill-switch (evaluated first — overrides everything).
	if killSwitch != nil && killSwitch() && intent.GetSource() == antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE {
		return &antv1.RiskDecision{
			Allow:   false,
			Reason:  "global kill-switch active — all live orders blocked",
			RuleHit: "kill_switch",
		}
	}

	// R11: Per-user autotrade switch (evaluated second — coarse-grained gate).
	if autotrade != nil && !autotrade(intent.GetUserId()) && intent.GetSource() == antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE {
		return &antv1.RiskDecision{
			Allow:   false,
			Reason:  "autotrade disabled for this user",
			RuleHit: "autotrade_disabled",
		}
	}

	// D6-A / T3.2b fail-closed: if AccountState is nil or has negative equity
	// (sentinel = provider not connected), equity-dependent rules cannot evaluate.
	// Block live orders until real account data is available.
	if intent.GetSource() == antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE {
		if state == nil {
			return &antv1.RiskDecision{
				Allow:   false,
				Reason:  "account state not available — equity rules fail-closed per T3.2b",
				RuleHit: "account_state_missing",
			}
		}
		if state.Equity.IsNegative() {
			return &antv1.RiskDecision{
				Allow:   false,
				Reason:  "account state provider not connected (equity sentinel) — fail-closed per T3.2b",
				RuleHit: "account_state_missing",
			}
		}
	}

	for _, rule := range rules {
		result := rule.Check(ctx, intent, state)
		if !result.Allowed {
			decision := &antv1.RiskDecision{
				Allow:   false,
				Reason:  result.Reason,
				RuleHit: rule.Name(),
			}
			if result.AdjustedVolume.IsPositive() {
				vol := result.AdjustedVolume.String()
				decision.AdjustedVolume = &vol
			}
			return decision
		}
	}

	return &antv1.RiskDecision{Allow: true}
}

// AddRule appends a rule to the gate (for dynamic configuration).
func (g *Gate) AddRule(rule Rule) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rules = append(g.rules, rule)
}

// Rules returns a copy of the registered rule names (for inspection).
func (g *Gate) Rules() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	names := make([]string, len(g.rules))
	for i, r := range g.rules {
		names[i] = r.Name()
	}
	return names
}

// ── Default gate factory ───────────────────────────────────────────────

// NewDefaultGate creates a Gate with all 11 rules from spec/31, using
// sensible defaults.  Callers can override thresholds via SetKillSwitch and
// SetAutotradeEnabled.
func NewDefaultGate() *Gate {
	return NewGate(
		&MaxLotSize{MaxLots: decimal.NewFromInt(10)},
		&MaxPositionCount{Max: 20},
		&MaxExposure{MaxRatio: decimal.NewFromFloat(0.5)}, // 50% of balance
		&DailyLossBreaker{MaxDailyLoss: decimal.Zero},       // disabled by default
		&DrawdownBreaker{MaxDrawdownPct: decimal.NewFromFloat(0.30)},
		&SymbolWhitelist{Whitelist: nil},                    // all allowed
		&LeverageCap{MaxLeverage: 500},
		&OrderFrequencyLimit{MaxOrders: 60, Window: time.Minute},
		&DuplicateProtection{DedupWindow: 5 * time.Second},
		&MarginPreCheck{MaxMarginRatio: decimal.NewFromFloat(0.80)},
	)
}

// ── Audit trail ───────────────────────────────────────────────────────

// AuditEntry pairs an OrderIntent with its RiskDecision and timestamp.
type AuditEntry struct {
	Intent        *antv1.OrderIntent
	Decision      *antv1.RiskDecision
	EvaluatedAtMs int64
}

// String returns a human-readable audit log line.
func (e *AuditEntry) String() string {
	decision := "ALLOW"
	if !e.Decision.GetAllow() {
		decision = fmt.Sprintf("DENY(%s)", e.Decision.GetRuleHit())
	}
	return fmt.Sprintf("risk_gate: ts=%d user=%s account=%s symbol=%s side=%s vol=%s type=%s %s",
		e.EvaluatedAtMs,
		e.Intent.GetUserId(),
		e.Intent.GetAccountId(),
		e.Intent.GetSymbol(),
		e.Intent.GetSide(),
		e.Intent.GetVolume(),
		e.Intent.GetType(),
		decision,
	)
}
