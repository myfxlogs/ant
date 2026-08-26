// Package risk — rule implementations for the 11-rule gate (T3.2 / spec/31).
//
// Rules are evaluated in registration order; first BLOCK stops the pipeline.
// Each rule implements the Rule interface from gate.go.
package risk

import (
	"context"
	"fmt"
	"strings"
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

func (r *OrderFrequencyLimit) Name() string { return "order_frequency" }

func (r *OrderFrequencyLimit) Check(_ context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	uid := intent.GetUserId()
	if r.timestamps == nil {
		r.timestamps = make(map[string][]time.Time)
	}

	now := time.Now()
	cutoff := now.Add(-r.Window)

	// Prune old timestamps for this user.
	old := r.timestamps[uid]
	recent := make([]time.Time, 0, len(old))
	for _, ts := range old {
		if ts.After(cutoff) {
			recent = append(recent, ts)
		}
	}

	if len(recent) >= r.MaxOrders {
		r.timestamps[uid] = recent
		return &RuleResult{
			Allowed: false,
			Reason:  fmt.Sprintf("order frequency limit: %d orders in %v", r.MaxOrders, r.Window),
		}
	}

	recent = append(recent, now)
	r.timestamps[uid] = recent

	// Lazy cleanup: purge inactive users when map grows large.
	if len(r.timestamps) > 1000 {
		for u, stamps := range r.timestamps {
			if len(stamps) == 0 || stamps[len(stamps)-1].Before(cutoff) {
				delete(r.timestamps, u)
			}
		}
	}

	return &RuleResult{Allowed: true}
}

// ── R8: Duplicate Order Protection ────────────────────────────────────

type DuplicateProtection struct {
	DedupWindow time.Duration
	mu          sync.Mutex
	lastOrders  map[string]time.Time // key: account|symbol|side|volume|type|price|magic
}

func (r *DuplicateProtection) Name() string { return "duplicate_protection" }

func (r *DuplicateProtection) Check(_ context.Context, intent *antv1.OrderIntent, _ *AccountState) *RuleResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	key := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d",
		intent.GetAccountId(), intent.GetSymbol(), intent.GetSide(), intent.GetVolume(),
		intent.GetType(), intent.GetPrice(), intent.GetMagic())

	if r.lastOrders == nil {
		r.lastOrders = make(map[string]time.Time)
	}

	if last, ok := r.lastOrders[key]; ok && now.Sub(last) < r.DedupWindow {
		return &RuleResult{
			Allowed: false,
			Reason:  fmt.Sprintf("duplicate order detected within %v", r.DedupWindow),
		}
	}

	r.lastOrders[key] = now

	// Cleanup old entries (lazy).
	if len(r.lastOrders) > 1000 {
		cutoff := now.Add(-r.DedupWindow * 2)
		for k, ts := range r.lastOrders {
			if ts.Before(cutoff) {
				delete(r.lastOrders, k)
			}
		}
	}

	return &RuleResult{Allowed: true}
}

// ── R9: Margin Pre-Check ──────────────────────────────────────────────

type MarginPreCheck struct {
	MaxMarginRatio decimal.Decimal // e.g. 0.80 = 80%
}

func (r *MarginPreCheck) Name() string { return "margin_pre_check" }

func (r *MarginPreCheck) Check(_ context.Context, intent *antv1.OrderIntent, state *AccountState) *RuleResult {
	if state == nil || r.MaxMarginRatio.IsZero() {
		return &RuleResult{Allowed: true}
	}
	if state.Equity.IsZero() {
		return &RuleResult{Allowed: false, Reason: "broker equity unavailable"}
	}
	vol := parseVol(intent.GetVolume())
	price := parsePrice(intent.GetPrice())
	if price.IsZero() && !state.BrokerMarginAvailable {
		return &RuleResult{Allowed: true}
	}

	if state.Platform == "mt4" && !state.BrokerMarginAvailable {
		return &RuleResult{Allowed: true, Reason: "MT4 broker required-margin RPC unavailable; broker remains authoritative"}
	}
	if state.BrokerMarginAvailable {
		if !state.RequiredMarginKnown {
			return &RuleResult{Allowed: false, Reason: "broker required margin unavailable"}
		}
		requiredMargin := state.RequiredMargin
		totalMargin := state.UsedMargin.Add(requiredMargin)
		ratio := totalMargin.Div(state.Equity)
		if ratio.GreaterThan(r.MaxMarginRatio) {
			return &RuleResult{
				Allowed: false,
				Reason: fmt.Sprintf("margin ratio %.1f%% exceeds limit %.1f%% (required=%s used=%s equity=%s)",
					ratio.InexactFloat64()*100, r.MaxMarginRatio.InexactFloat64()*100,
					requiredMargin, state.UsedMargin, state.Equity),
			}
		}
		return &RuleResult{Allowed: true}
	}
	cs, ok := contractSize(state)
	if !ok {
		return &RuleResult{
			Allowed: false,
			Reason:  fmt.Sprintf("contract size unknown for symbol %s", intent.GetSymbol()),
		}
	}
	if state.SymbolLeverage <= 0 {
		return &RuleResult{Allowed: false, Reason: "broker leverage unavailable"}
	}
	leverage := decimal.NewFromInt(int64(state.SymbolLeverage))

	// This local formula is retained only for platforms without a broker
	// required-margin capability; MT5 uses the broker RPC above.
	// USD-base symbols use fx_rate=1; other symbols use the quoted price.
	fxRate := decimal.NewFromInt(1)
	if !strings.HasPrefix(intent.GetSymbol(), "USD") {
		fxRate = price
	}
	requiredMargin := vol.Mul(cs).Mul(fxRate).Div(leverage)
	totalMargin := state.UsedMargin.Add(requiredMargin)
	ratio := totalMargin.Div(state.Equity)

	if ratio.GreaterThan(r.MaxMarginRatio) {
		return &RuleResult{
			Allowed: false,
			Reason: fmt.Sprintf("margin ratio %.1f%% exceeds limit %.1f%% (required=%s used=%s equity=%s)",
				ratio.InexactFloat64()*100, r.MaxMarginRatio.InexactFloat64()*100,
				requiredMargin, state.UsedMargin, state.Equity),
		}
	}
	return &RuleResult{Allowed: true}
}
