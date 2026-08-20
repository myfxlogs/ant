// Package runner executes Go strategies implementing the sdk.Strategy interface.
//
// It provides concrete implementations of sdk.Context, sdk.Broker,
// and sdk.IndicatorSet for use by strategy execution.
package runner

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// Config holds the parameters for running a strategy.
type Config struct {
	AccountID           string
	Symbol              string
	Timeframe           string
	Params              map[string]string // user-configured parameter overrides
	Mode                string            // "live" | "paper" | "backtest"
	DataSourceAccountID string
}

// Runner executes a strategy lifecycle against live or historical data.
type Runner struct {
	cfg    Config
	broker *brokerImpl
	ctx    *contextImpl
	ind    *indicatorSet

	mu       sync.Mutex
	strategy sdk.Strategy

	// barRev advances once per OnBar call, enabling SeriesCache to detect
	// rolling-window content changes at constant Len() (LIVE-INDICATOR-1).
	// OnTick/OnTrade/OnTimer/OnBookEvent do NOT advance it — the bar window
	// is unchanged within a bar. Atomic for race-detector cleanliness under
	// the single-owner event loop (no mutex needed on the hot path).
	barRev atomic.Uint64
}

// New creates a new Runner.
func New(cfg Config) *Runner {
	r := &Runner{cfg: cfg}
	r.broker = &brokerImpl{runner: r}
	r.ind = &indicatorSet{runner: r}
	r.ctx = &contextImpl{
		runner:    r,
		params:    cfg.Params,
		symbol:    cfg.Symbol,
		timeframe: cfg.Timeframe,
	}
	return r
}

// SetStrategy sets the strategy to execute.
func (r *Runner) SetStrategy(s sdk.Strategy) {
	r.strategy = s
}

// barRevision returns the current bar revision counter. Each OnBar call
// advances it by 1; other event handlers do not. Used by runnerBarSource
// to implement indicators.RevisionedBarSource for SeriesCache invalidation.
func (r *Runner) barRevision() uint64 {
	return r.barRev.Load()
}

// UpdateLiveState sets the live account state from the parent process.
// Used by the live harness to pass equity/balance/margin/free_margin/positions without RPC.
func (r *Runner) UpdateLiveState(balance, equity, margin, freeMargin string, positions []sdk.Position) {
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()
	r.ctx.liveBalance = balance
	r.ctx.liveEquity = equity
	r.ctx.liveMargin = margin
	r.ctx.liveFreeMargin = freeMargin
	r.ctx.livePositions = positions
}

// UpdateExtraBars sets the extra symbol bar windows for multi-symbol strategies.
func (r *Runner) UpdateExtraBars(extra map[string][]sdk.Bar) {
	r.ctx.setExtraBars(extra)
}

// Init calls the strategy's OnInit.
func (r *Runner) Init(ctx context.Context) error {
	if r.strategy == nil {
		return nil
	}
	r.ctx.setGoContext(ctx)
	return r.strategy.OnInit(r.ctx)
}

// OnBar calls the strategy's OnBar for a new bar.
// Advances barRev so SeriesCache invalidates on rolling-window content changes.
func (r *Runner) OnBar(ctx context.Context, bars sdk.BarSeries, timeframe string) (*sdk.Signal, error) {
	if r.strategy == nil {
		return nil, nil
	}
	r.ctx.setGoContext(ctx)
	r.broker.setContext(ctx)
	r.mu.Lock()
	r.ctx.setBars(bars)
	r.mu.Unlock()
	r.barRev.Add(1)
	return r.strategy.OnBar(r.ctx, timeframe)
}

// OnTick calls the strategy's OnTick if it implements TickStrategy.
func (r *Runner) OnTick(ctx context.Context, bid, ask decimal.Decimal) (*sdk.Signal, error) {
	if r.strategy == nil {
		return nil, nil
	}
	ts, ok := r.strategy.(sdk.TickStrategy)
	if !ok {
		return nil, nil
	}
	r.ctx.setGoContext(ctx)
	r.broker.setContext(ctx)
	r.ctx.setTick(bid, ask)
	return ts.OnTick(r.ctx, bid, ask)
}

// OnTrade calls the strategy's OnTrade if it implements TradeStrategy.
func (r *Runner) OnTrade(ctx context.Context, event sdk.TradeEvent) (*sdk.Signal, error) {
	if r.strategy == nil {
		return nil, nil
	}
	ts, ok := r.strategy.(sdk.TradeStrategy)
	if !ok {
		return nil, nil
	}
	r.ctx.setGoContext(ctx)
	r.broker.setContext(ctx)
	return ts.OnTrade(r.ctx, event)
}

// OnTimerTick calls the strategy's OnTimer if it implements TimerStrategy.
func (r *Runner) OnTimerTick(ctx context.Context) (*sdk.Signal, error) {
	if r.strategy == nil {
		return nil, nil
	}
	ts, ok := r.strategy.(sdk.TimerStrategy)
	if !ok {
		return nil, nil
	}
	r.ctx.setGoContext(ctx)
	r.broker.setContext(ctx)
	return ts.OnTimer(r.ctx)
}

// OnTradeTransaction calls the strategy's OnTradeTransaction if it implements TradeTransactionStrategy.
func (r *Runner) OnTradeTransaction(ctx context.Context) (*sdk.Signal, error) {
	if r.strategy == nil {
		return nil, nil
	}
	ts, ok := r.strategy.(sdk.TradeTransactionStrategy)
	if !ok {
		return nil, nil
	}
	r.ctx.setGoContext(ctx)
	r.broker.setContext(ctx)
	return ts.OnTradeTransaction(r.ctx)
}

// OnBookEvent calls the strategy's OnBookEvent if it implements BookEventStrategy.
func (r *Runner) OnBookEvent(ctx context.Context) (*sdk.Signal, error) {
	if r.strategy == nil {
		return nil, nil
	}
	ts, ok := r.strategy.(sdk.BookEventStrategy)
	if !ok {
		return nil, nil
	}
	r.ctx.setGoContext(ctx)
	r.broker.setContext(ctx)
	return ts.OnBookEvent(r.ctx)
}

// HasOnTradeTransaction returns true if the underlying strategy implements TradeTransactionStrategy.
func (r *Runner) HasOnTradeTransaction() bool {
	if r.strategy == nil {
		return false
	}
	_, ok := r.strategy.(sdk.TradeTransactionStrategy)
	return ok
}

// HasOnBookEvent returns true if the underlying strategy implements BookEventStrategy.
func (r *Runner) HasOnBookEvent() bool {
	if r.strategy == nil {
		return false
	}
	_, ok := r.strategy.(sdk.BookEventStrategy)
	return ok
}

// UpdateTickState sets the latest bid/ask from a tick event.
func (r *Runner) UpdateTickState(bid, ask decimal.Decimal) {
	r.ctx.setTick(bid, ask)
}

// UpdateSymbolInfo sets the live symbol info from the parent process.
// Used by the live harness to pass Point/Digits/ContractSize/StopsLevel without RPC.
func (r *Runner) UpdateSymbolInfo(point string, digits int32, contractSize, stopsLevel string) {
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()
	r.ctx.livePoint = point
	r.ctx.liveDigits = digits
	r.ctx.liveContractSize = contractSize
	r.ctx.liveStopsLevel = stopsLevel
}

// OrderExecutor wraps the broker's trading interface.
// Used by backtest.SimBroker and live-trading adapter.
type OrderExecutor interface {
	PlaceOrder(ctx context.Context, symbol string, side sdk.PositionSide,
		orderType sdk.OrderType, volume, price, sl, tp decimal.Decimal,
		comment string, magic int32) (int64, error)
	CloseOrder(ctx context.Context, ticket int64, volume decimal.Decimal) error
	ModifyOrder(ctx context.Context, ticket int64, sl, tp decimal.Decimal) error
	CancelOrder(ctx context.Context, ticket int64) error
	OpenedOrders(ctx context.Context) ([]sdk.Position, error)
	PendingOrders(ctx context.Context) ([]sdk.PendingOrder, error)
	Account() sdk.AccountInfo
	SymbolInfo(symbol string) (sdk.SymbolInfo, error)
}

// Deinit calls the strategy's OnDeinit.
func (r *Runner) Deinit(ctx context.Context, reason string) error {
	if r.strategy == nil {
		return nil
	}
	r.ctx.setGoContext(ctx)
	return r.strategy.OnDeinit(r.ctx, reason)
}
