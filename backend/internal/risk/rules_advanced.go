// rules_advanced.go — Advanced rules (frequency, duplicate, margin) extracted from rules.go.
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
