// Package runner executes Go strategies implementing the sdk.Strategy interface.
//
// It provides concrete implementations of sdk.Context, sdk.Broker,
// and sdk.IndicatorSet for use by strategy execution.
package runner

import (
	"context"
	"sync"

	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
)

// Config holds the parameters for running a strategy.
type Config struct {
	AccountID           string
	Symbol              string
	Timeframe           string
	Params              map[string]string // user-configured parameter overrides
	Mode                string           // "live" | "paper" | "backtest"
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
	running  bool
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

// UpdateLiveState sets the live account state from the parent process.
// Used by the live harness to pass equity/balance/positions without RPC.
func (r *Runner) UpdateLiveState(balance, equity string, positions []sdk.Position) {
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()
	r.ctx.liveBalance = balance
	r.ctx.liveEquity = equity
	r.ctx.livePositions = positions
}

// Init calls the strategy's OnInit.
func (r *Runner) Init(ctx context.Context) error {
	if r.strategy == nil {
		return nil
	}
	return r.strategy.OnInit(r.ctx)
}

// OnBar calls the strategy's OnBar for a new bar.
func (r *Runner) OnBar(ctx context.Context, bars sdk.BarSeries, timeframe string) (*sdk.Signal, error) {
	if r.strategy == nil {
		return nil, nil
	}
	r.mu.Lock()
	r.ctx.setBars(bars)
	r.mu.Unlock()
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
	return ts.OnTimer(r.ctx)
}

// UpdateTickState sets the latest bid/ask from a tick event.
func (r *Runner) UpdateTickState(bid, ask decimal.Decimal) {
	r.ctx.setTick(bid, ask)
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
	return r.strategy.OnDeinit(r.ctx, reason)
}

