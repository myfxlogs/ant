// account_provider.go — Real AccountStateProvider backed by MT gateway (T3.2b).
//
// Queries the MT4/MT5 gateway for live account data (balance, equity, margin,
// open positions).  Tracks peak equity in-memory for drawdown rule evaluation.
//
// Returns nil state on error → gate fail-closed per D6-A.

package strategy

import (
	"context"
	"fmt"
	"sync"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/mthub"
	"alphaforge/internal/risk"
)

// ── MT-backed AccountStateProvider ─────────────────────────────────────

// MTAccountStateProvider implements AccountStateProvider by querying
// the MT gateway's OrderExecutor for open positions and deriving
// account state from position PnL + balance history.
type MTAccountStateProvider struct {
	hub      *mthub.Hub
	log      *zap.Logger
	posCache *PositionCache // push-based position snapshots (no polling)

	mu           sync.RWMutex
	peakEquity   map[string]decimal.Decimal // accountID → peak equity
	balanceCache map[string]decimal.Decimal // accountID → balance from ProfitUpdate
	equityCache  map[string]decimal.Decimal // accountID → equity from ProfitUpdate
}

// NewMTAccountStateProvider creates a provider backed by the MT gateway Hub.
func NewMTAccountStateProvider(hub *mthub.Hub, log *zap.Logger) *MTAccountStateProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &MTAccountStateProvider{
		hub:          hub,
		log:          log,
		peakEquity:   make(map[string]decimal.Decimal),
		balanceCache: make(map[string]decimal.Decimal),
		equityCache:  make(map[string]decimal.Decimal),
	}
}

// SetPositionCache injects the push-based position cache.
func (p *MTAccountStateProvider) SetPositionCache(pc *PositionCache) { p.posCache = pc }

// GetAccountState fetches live account state for gate evaluation.
// Uses push-based PositionCache when available (no polling).
// Returns (nil, nil) on error — gate fail-closed per D6-A.
func (p *MTAccountStateProvider) GetAccountState(ctx context.Context, accountID string) (*risk.AccountState, error) {
	// Push-first: read from PositionCache if available (no RPC).
	if p.posCache != nil {
		snap := p.posCache.GetSnapshot(accountID)
		if snap != nil {
			state := p.buildStateFromSnapshot(accountID, snap)
			p.log.Debug("MTAccountStateProvider: state from PositionCache",
				zap.String("account", accountID),
				zap.String("equity", state.Equity.String()),
				zap.Int("positions", state.OpenPositions),
			)
			return state, nil
		}
		// No snapshot yet — fall through to legacy path.
	}

	// Legacy fallback: poll FetchOpenedOrders (only when no PositionCache wired).
	exec := p.hub.Get(accountID)
	if exec == nil {
		p.log.Debug("MTAccountStateProvider: no executor for account — gate fail-closed",
			zap.String("account", accountID))
		return nil, nil // nil state → gate blocks (fail-closed)
	}

	orders, err := exec.FetchOpenedOrders(ctx)
	if err != nil {
		p.log.Warn("MTAccountStateProvider: FetchOpenedOrders failed — gate fail-closed",
			zap.String("account", accountID), zap.Error(err))
		return nil, nil
	}

	state := p.buildStateFromOrders(accountID, orders)
	p.log.Debug("MTAccountStateProvider: state computed (legacy poll)",
		zap.String("account", accountID),
		zap.String("equity", state.Equity.String()),
		zap.Int("positions", state.OpenPositions),
	)
	return state, nil
}

// buildStateFromSnapshot derives AccountState from a push-based PositionSnapshot.
func (p *MTAccountStateProvider) buildStateFromSnapshot(accountID string, snap *mthub.PositionSnapshot) *risk.AccountState {
	totalProfit := decimal.Zero
	totalMargin := decimal.Zero
	defaultLeverage := decimal.NewFromInt(100)

	for _, pos := range snap.Positions {
		totalProfit = totalProfit.Add(pos.Profit)
		notional := pos.Volume.Mul(pos.OpenPrice)
		totalMargin = totalMargin.Add(notional.Div(defaultLeverage))
	}

	balance := snap.Balance
	if balance.IsZero() {
		p.mu.RLock()
		cachedBalance, hasBalance := p.balanceCache[accountID]
		p.mu.RUnlock()
		if hasBalance {
			balance = cachedBalance
		} else {
			balance = decimal.NewFromInt(10000)
		}
	}

	equity := snap.Equity
	if equity.IsZero() {
		equity = balance.Add(totalProfit)
	}
	freeMargin := equity.Sub(totalMargin)
	if freeMargin.LessThan(decimal.Zero) {
		freeMargin = decimal.Zero
	}

	p.mu.Lock()
	peak, ok := p.peakEquity[accountID]
	if !ok || equity.GreaterThan(peak) {
		peak = equity
		p.peakEquity[accountID] = peak
	}
	p.mu.Unlock()

	return &risk.AccountState{
		Balance:        balance,
		Equity:         equity,
		FreeMargin:     freeMargin,
		UsedMargin:     totalMargin,
		OpenPositions:  len(snap.Positions),
		DailyPnL:       totalProfit,
		PeakEquity:     peak,
		SymbolLeverage: 100,
	}
}

// ── State computation ──────────────────────────────────────────────────

// buildStateFromOrders derives AccountState from open MT orders.
//
// MT gateways stream ProfitUpdate events (Balance/Equity/Margin/FreeMargin
// per tick) via the AccountProfitBroker.  In the absence of a live
// subscription, we approximate from open positions:
//   - Balance: default 10000 (overridden when ProfitUpdate subscription is active)
//   - Equity: balance + sum(position.Profit)
//   - Margin: sum(notional / leverage), approximate
//   - Daily PnL: sum(position.Profit) — reset at broker day rollover
//   - PeakEquity: tracked in-memory (high-water mark since provider start)
func (p *MTAccountStateProvider) buildStateFromOrders(accountID string, orders []*mthub.OrderRecord) *risk.AccountState {
	totalProfit := decimal.Zero
	totalMargin := decimal.Zero
	defaultLeverage := decimal.NewFromInt(100)

	for _, o := range orders {
		totalProfit = totalProfit.Add(o.Profit)
		notional := o.Volume.Mul(o.OpenPrice)
		totalMargin = totalMargin.Add(notional.Div(defaultLeverage))
	}

	// Use cached balance from ProfitUpdate events when available.
	// Falls back to 10000 if no real balance has been received yet.
	p.mu.RLock()
	cachedBalance, hasBalance := p.balanceCache[accountID]
	p.mu.RUnlock()

	balance := decimal.NewFromInt(10000)
	if hasBalance {
		balance = cachedBalance
	}
	equity := balance.Add(totalProfit)
	freeMargin := equity.Sub(totalMargin)
	if freeMargin.LessThan(decimal.Zero) {
		freeMargin = decimal.Zero
	}

	p.mu.Lock()
	peak, ok := p.peakEquity[accountID]
	if !ok || equity.GreaterThan(peak) {
		peak = equity
		p.peakEquity[accountID] = peak
	}
	p.mu.Unlock()

	return &risk.AccountState{
		Balance:        balance,
		Equity:         equity,
		FreeMargin:     freeMargin,
		UsedMargin:     totalMargin,
		OpenPositions:  len(orders),
		DailyPnL:       totalProfit,
		PeakEquity:     peak,
		SymbolLeverage: 100,
	}
}

// ── Peak equity management ─────────────────────────────────────────────

// GetPeakEquity returns the tracked peak equity for an account.
func (p *MTAccountStateProvider) GetPeakEquity(accountID string) decimal.Decimal {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.peakEquity[accountID]
}

// ResetPeakEquity resets peak tracking for an account (call at daily rollover).
func (p *MTAccountStateProvider) ResetPeakEquity(accountID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.peakEquity, accountID)
}

// UpdateBalanceFromProfitEvent updates the cached balance from a real
// MT ProfitUpdate event.
func (p *MTAccountStateProvider) UpdateBalanceFromProfitEvent(accountID string, balance, equity, margin, freeMargin decimal.Decimal) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.balanceCache[accountID] = balance
	p.equityCache[accountID] = equity
}

// ── Error handling ─────────────────────────────────────────────────────

// ErrNoGatewaySession is returned when the MT gateway session is not registered.
var ErrNoGatewaySession = fmt.Errorf("no gateway session registered for account")
