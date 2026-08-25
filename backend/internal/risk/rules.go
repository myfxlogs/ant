// Package risk — rule implementations for the 11-rule gate (T3.2 / spec/31).
//
// Rules are evaluated in registration order; first BLOCK stops the pipeline.
// Each rule implements the Rule interface from gate.go.
package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// ── Helper: parse volume from proto string ─────────────────────────────

func parseVol(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func parsePrice(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

// contractSize returns the per-symbol contract multiplier from AccountState.
// It no longer silently defaults; the second result reports whether the value
// was explicitly populated. Missing/zero contract size must fail-closed.
func contractSize(state *AccountState) (decimal.Decimal, bool) {
	if state != nil && state.ContractSize.GreaterThan(decimal.Zero) {
		return state.ContractSize, true
	}
	return decimal.Zero, false
}

// ── R1: Max Lot Size ──────────────────────────────────────────────────

type MaxLotSize struct {
	MaxLots decimal.Decimal
}

func (r *MaxLotSize) Name() string { return "max_lot_size" }

func (r *MaxLotSize) Check(_ context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
	vol := parseVol(intent.GetVolume())
	if vol.GreaterThan(r.MaxLots) {
		return &RuleResult{
			Allowed:        false,
			Reason:         fmt.Sprintf("volume %s exceeds max lot size %s", vol, r.MaxLots),
			AdjustedVolume: r.MaxLots,
		}
	}
	return &RuleResult{Allowed: true}
}

// ── R2: Max Position Count ────────────────────────────────────────────

type MaxPositionCount struct {
	Max int
}

func (r *MaxPositionCount) Name() string { return "max_position_count" }

func (r *MaxPositionCount) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
	if state == nil {
		return &RuleResult{Allowed: true}
	}
	if state.OpenPositions >= r.Max {
		return &RuleResult{
			Allowed: false,
			Reason:  fmt.Sprintf("open positions %d >= max %d", state.OpenPositions, r.Max),
		}
	}
	return &RuleResult{Allowed: true}
}

// ── R3: Max Exposure ──────────────────────────────────────────────────

type MaxExposure struct {
	MaxRatio decimal.Decimal // fraction of balance
}

func (r *MaxExposure) Name() string { return "max_exposure" }

func (r *MaxExposure) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
	if state == nil || r.MaxRatio.IsZero() {
		return &RuleResult{Allowed: true}
	}
	vol := parseVol(intent.GetVolume())
	price := parsePrice(intent.GetPrice())
	if price.IsZero() {
		// Market order — use current_price from state (approximate).
		return &RuleResult{Allowed: true}
	}
	cs, ok := contractSize(state)
	if !ok {
		return &RuleResult{
			Allowed: false,
			Reason:  fmt.Sprintf("contract size unknown for symbol %s", intent.GetSymbol()),
		}
	}
	notional := vol.Mul(price).Mul(cs) // vol × price × contract_size
	maxAllowed := state.Balance.Mul(r.MaxRatio)
	if notional.GreaterThan(maxAllowed) {
		return &RuleResult{
			Allowed: false,
			Reason:  fmt.Sprintf("notional %s exceeds max exposure %s (%.0f%% of balance)", notional, maxAllowed, r.MaxRatio.InexactFloat64()*100),
		}
	}
	return &RuleResult{Allowed: true}
}

// ── R4a: Daily Loss Circuit Breaker ───────────────────────────────────

type DailyLossBreaker struct {
	MaxDailyLoss decimal.Decimal // zero = disabled
}

func (r *DailyLossBreaker) Name() string { return "daily_loss" }

func (r *DailyLossBreaker) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
	if r.MaxDailyLoss.IsZero() || state == nil {
		return &RuleResult{Allowed: true}
	}
	if state.DailyPnL.LessThan(r.MaxDailyLoss.Neg()) {
		return &RuleResult{
			Allowed: false,
			Reason:  fmt.Sprintf("daily loss %s exceeds limit %s", state.DailyPnL, r.MaxDailyLoss.Neg()),
		}
	}
	return &RuleResult{Allowed: true}
}

// ── R4b: Drawdown Circuit Breaker ─────────────────────────────────────

type DrawdownBreaker struct {
	MaxDrawdownPct decimal.Decimal // e.g. 0.30 = 30%
}

func (r *DrawdownBreaker) Name() string { return "drawdown" }

func (r *DrawdownBreaker) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
	if r.MaxDrawdownPct.IsZero() || state == nil || state.PeakEquity.IsZero() {
		return &RuleResult{Allowed: true}
	}
	dd := decimal.NewFromInt(1).Sub(state.Equity.Div(state.PeakEquity))
	if dd.GreaterThan(r.MaxDrawdownPct) {
		return &RuleResult{
			Allowed: false,
			Reason:  fmt.Sprintf("drawdown %.1f%% exceeds limit %.1f%%", dd.InexactFloat64()*100, r.MaxDrawdownPct.InexactFloat64()*100),
		}
	}
	return &RuleResult{Allowed: true}
}

// ── R5: Symbol Whitelist ──────────────────────────────────────────────

type SymbolWhitelist struct {
	Whitelist []string // nil or empty = all allowed
}

func (r *SymbolWhitelist) Name() string { return "symbol_whitelist" }

func (r *SymbolWhitelist) Check(_ context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
	if len(r.Whitelist) == 0 {
		return &RuleResult{Allowed: true}
	}
	symbol := intent.GetSymbol()
	for _, s := range r.Whitelist {
		if s == symbol {
			return &RuleResult{Allowed: true}
		}
	}
	return &RuleResult{
		Allowed: false,
		Reason:  fmt.Sprintf("symbol %s not in whitelist", symbol),
	}
}

// ── R6: Leverage Cap ──────────────────────────────────────────────────

type LeverageCap struct {
	MaxLeverage int
}

func (r *LeverageCap) Name() string { return "leverage_cap" }

func (r *LeverageCap) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
	if state == nil || state.SymbolLeverage == 0 {
		return &RuleResult{Allowed: true}
	}
	if state.SymbolLeverage > r.MaxLeverage {
		return &RuleResult{
			Allowed: false,
			Reason:  fmt.Sprintf("symbol leverage %d exceeds cap %d", state.SymbolLeverage, r.MaxLeverage),
		}
	}
	return &RuleResult{Allowed: true}
}

// ── R7: Order Frequency Limit ─────────────────────────────────────────

type OrderFrequencyLimit struct {
	MaxOrders  int
	Window     time.Duration
	mu         sync.Mutex
	timestamps map[string][]time.Time // per-userID
}
