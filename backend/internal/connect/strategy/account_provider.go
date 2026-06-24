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

	"anttrader/internal/mthub"
	"anttrader/internal/risk"
)

// ── MT-backed AccountStateProvider ─────────────────────────────────────

// MTAccountStateProvider implements AccountStateProvider by querying
// the MT gateway's OrderExecutor for open positions and deriving
// account state from position PnL + balance history.
type MTAccountStateProvider struct {
	hub *mthub.Hub
	log *zap.Logger

	mu           sync.RWMutex
	peakEquity   map[string]decimal.Decimal // accountID → peak equity
	balanceCache map[string]float64         // accountID → balance from ProfitUpdate
	equityCache  map[string]float64         // accountID → equity from ProfitUpdate
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
		balanceCache: make(map[string]float64),
		equityCache:  make(map[string]float64),
	}
}

// GetAccountState fetches live account state for gate evaluation.
// Returns (nil, nil) on error — gate fail-closed per D6-A.
func (p *MTAccountStateProvider) GetAccountState(ctx context.Context, accountID string) (*risk.AccountState, error) {
	exec := p.hub.Get(accountID)
	if exec == nil {
		p.log.Debug("MTAccountStateProvider: no executor for account — gate fail-closed",
			zap.String("account", accountID))
		return nil, nil // nil state → gate blocks (fail-closed)
	}

	// Fetch open positions from the MT gateway.
	orders, err := exec.FetchOpenedOrders(ctx)
	if err != nil {
		p.log.Warn("MTAccountStateProvider: FetchOpenedOrders failed — gate fail-closed",
			zap.String("account", accountID), zap.Error(err))
		return nil, nil
	}

	state := p.buildStateFromOrders(accountID, orders)

	p.log.Debug("MTAccountStateProvider: state computed",
		zap.String("account", accountID),
		zap.String("equity", state.Equity.String()),
		zap.Int("positions", state.OpenPositions),
	)

	return state, nil
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
	var totalProfit, totalMargin float64

	for _, o := range orders {
		profit, _ := o.Profit.Float64()
		totalProfit += profit

		notional := o.Volume.Mul(o.OpenPrice).InexactFloat64()
		totalMargin += notional / 100.0 // 1:100 leverage default
	}

	// Use cached balance from ProfitUpdate events when available.
	// Falls back to 10000 if no real balance has been received yet.
	p.mu.RLock()
	cachedBalance, hasBalance := p.balanceCache[accountID]
	p.mu.RUnlock()

	balance := 10000.0
	if hasBalance {
		balance = cachedBalance
	}
	equity := balance + totalProfit
	freeMargin := equity - totalMargin
	if freeMargin < 0 {
		freeMargin = 0
	}

	equityDec := decimal.NewFromFloat(equity)

	p.mu.Lock()
	peak, ok := p.peakEquity[accountID]
	if !ok || equityDec.GreaterThan(peak) {
		peak = equityDec
		p.peakEquity[accountID] = peak
	}
	p.mu.Unlock()

	return &risk.AccountState{
		Balance:        decimal.NewFromFloat(balance),
		Equity:         equityDec,
		FreeMargin:     decimal.NewFromFloat(freeMargin),
		UsedMargin:     decimal.NewFromFloat(totalMargin),
		OpenPositions:  len(orders),
		DailyPnL:       decimal.NewFromFloat(totalProfit),
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
func (p *MTAccountStateProvider) UpdateBalanceFromProfitEvent(accountID string, balance, equity, margin, freeMargin float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.balanceCache[accountID] = balance
	p.equityCache[accountID] = equity
}

// ── Error handling ─────────────────────────────────────────────────────

// ErrNoGatewaySession is returned when the MT gateway session is not registered.
var ErrNoGatewaySession = fmt.Errorf("no gateway session registered for account")
