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
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/mthub"
	"alphaforge/internal/risk"
)

// MT-backed AccountStateProvider is sourced exclusively from the fresh broker snapshot.
type MTAccountStateProvider struct {
	log      *zap.Logger
	posCache *PositionCache

	mu         sync.RWMutex
	peakEquity map[string]decimal.Decimal
	now        func() time.Time
}

// NewMTAccountStateProvider creates a provider backed by the push-based MT snapshot cache.
func NewMTAccountStateProvider(_ *mthub.Hub, log *zap.Logger) *MTAccountStateProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &MTAccountStateProvider{
		log:        log,
		peakEquity: make(map[string]decimal.Decimal),
		now:        time.Now,
	}
}

// SetPositionCache injects the push-based authoritative snapshot cache.
func (p *MTAccountStateProvider) SetPositionCache(pc *PositionCache) { p.posCache = pc }

// GetAccountState returns nil when the broker snapshot is unavailable or stale.
func (p *MTAccountStateProvider) GetAccountState(_ context.Context, accountID string) (*risk.AccountState, error) {
	if p.posCache == nil {
		p.log.Warn("MTAccountStateProvider: position cache not configured — gate fail-closed",
			zap.String("account", accountID))
		return nil, nil
	}
	snap, ok := p.posCache.GetFreshTradingSnapshot(accountID, p.now())
	if !ok {
		p.log.Warn("MTAccountStateProvider: authoritative account snapshot missing or stale — gate fail-closed",
			zap.String("account", accountID))
		return nil, nil
	}
	if snap.Leverage <= 0 {
		p.log.Warn("MTAccountStateProvider: authoritative leverage missing — gate fail-closed",
			zap.String("account", accountID))
		return nil, nil
	}
	p.mu.Lock()
	peak := p.peakEquity[accountID]
	if peak.IsZero() || snap.Equity.GreaterThan(peak) {
		peak = snap.Equity
		p.peakEquity[accountID] = peak
	}
	p.mu.Unlock()
	return &risk.AccountState{
		Platform:       snap.Platform,
		Balance:        snap.Balance,
		Equity:         snap.Equity,
		FreeMargin:     snap.FreeMargin,
		UsedMargin:     snap.Margin,
		OpenPositions:  len(snap.Positions),
		DailyPnL:       snap.Profit,
		PeakEquity:     peak,
		SymbolLeverage: int(snap.Leverage),
	}, nil
}

// GetPeakEquity returns the locally tracked peak used only for drawdown rules.
func (p *MTAccountStateProvider) GetPeakEquity(accountID string) decimal.Decimal {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.peakEquity[accountID]
}

// ResetPeakEquity resets peak tracking for an account at a daily rollover.
func (p *MTAccountStateProvider) ResetPeakEquity(accountID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.peakEquity, accountID)
}

// ErrNoGatewaySession is returned when the MT gateway session is not registered.
var ErrNoGatewaySession = fmt.Errorf("no gateway session registered for account")
