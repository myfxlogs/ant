// Package runner executes Go strategies implementing the sdk.Strategy interface.
//
// It provides concrete implementations of sdk.Context, sdk.Broker,
// and sdk.IndicatorSet, backed by the existing BarSource and OrderExecutor
// infrastructure.
package runner

import (
	"context"
	"sync"
	"time"

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

// Init calls the strategy's OnInit.
func (r *Runner) Init(ctx context.Context) error {
	if r.strategy == nil {
		return nil
	}
	return r.strategy.OnInit(r.ctx)
}

// OnBar calls the strategy's OnBar for a new bar.
// Returns the signal if any, or nil.
func (r *Runner) OnBar(ctx context.Context, bars sdk.BarSeries, timeframe string) (*sdk.Signal, error) {
	if r.strategy == nil {
		return nil, nil
	}
	r.mu.Lock()
	r.ctx.setBars(bars)
	r.mu.Unlock()
	return r.strategy.OnBar(r.ctx, timeframe)
}

// Deinit calls the strategy's OnDeinit.
func (r *Runner) Deinit(ctx context.Context, reason string) error {
	if r.strategy == nil {
		return nil
	}
	return r.strategy.OnDeinit(r.ctx, reason)
}

// ── Live runner ──────────────────────────────────────────────────

// LiveRunner subscribes to bar updates and runs a strategy in real-time.
type LiveRunner struct {
	*Runner
	barSource BarSource
	executor  OrderExecutor
}

// BarSource provides historical and live bar data.
type BarSource interface {
	Fetch(ctx context.Context, symbol, timeframe string, from, to *time.Time) ([]sdk.Bar, error)
	Subscribe(accountID string) (<-chan BarUpdate, func())
}

// BarUpdate is a single bar update from a live source.
type BarUpdate struct {
	Symbol    string
	Period    string
	OpenTime  int64
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    int64
	Closed    bool
}

// OrderExecutor wraps the broker's trading interface.
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

// NewLiveRunner creates a Runner for live trading.
func NewLiveRunner(cfg Config, barSource BarSource, executor OrderExecutor) *LiveRunner {
	r := New(cfg)
	lr := &LiveRunner{Runner: r, barSource: barSource, executor: executor}
	r.broker.executor = executor
	return lr
}

// Run starts the live strategy loop.
func (lr *LiveRunner) Run(ctx context.Context) error {
	if err := lr.Init(ctx); err != nil {
		return err
	}

	// Backfill historical bars
	bars, err := lr.barSource.Fetch(ctx, lr.cfg.Symbol, lr.cfg.Timeframe, nil, nil)
	if err != nil {
		return err
	}
	barSeries := sdk.BarsToSlice(bars)

	// Process initial bars to warm up indicators (no signals)
	_ = barSeries

	// Subscribe to live bar updates
	ch, cancel := lr.barSource.Subscribe(lr.cfg.DataSourceAccountID)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return lr.Deinit(ctx, "user_stop")
		case update, ok := <-ch:
			if !ok {
				return lr.Deinit(ctx, "user_stop")
			}
			if update.Symbol != lr.cfg.Symbol || update.Period != lr.cfg.Timeframe {
				continue
			}
			if !update.Closed {
				continue // only process closed bars
			}
			bars = append(bars, sdk.Bar{
				Open:      update.Open,
				High:      update.High,
				Low:       update.Low,
				Close:     update.Close,
				Volume:    update.Volume,
				Timestamp: update.OpenTime,
			})
			// Keep a sliding window of 500 bars
			if len(bars) > 500 {
				bars = bars[len(bars)-500:]
			}
			barSeries = sdk.BarsToSlice(bars)
			sig, err := lr.OnBar(ctx, barSeries, lr.cfg.Timeframe)
			if err != nil {
				return err
			}
			if sig != nil {
				lr.dispatchSignal(sig)
			}
		}
	}
}

func (lr *LiveRunner) dispatchSignal(sig *sdk.Signal) {
	if lr.executor == nil {
		return
	}
	switch sig.Action {
	case sdk.ActionBuy, sdk.ActionSell, sdk.ActionBuyLimit, sdk.ActionSellLimit,
		sdk.ActionBuyStop, sdk.ActionSellStop:
		side := sdk.SideBuy
		ot := sdk.OrderMarket
		if sig.Action == sdk.ActionSell {
			side = sdk.SideSell
		}
		if sig.Action == sdk.ActionBuyLimit || sig.Action == sdk.ActionSellLimit {
			ot = sdk.OrderLimit
		}
		if sig.Action == sdk.ActionBuyStop || sig.Action == sdk.ActionSellStop {
			ot = sdk.OrderStop
		}
		lr.executor.PlaceOrder(context.Background(), sig.Symbol, side, ot,
			sig.Volume, sig.Price, sig.StopLoss, sig.TakeProfit,
			sig.Comment, sig.Magic)
	case sdk.ActionClose:
		lr.executor.CloseOrder(context.Background(), sig.OrderTicket, decimal.Zero)
	case sdk.ActionCancel:
		lr.executor.CancelOrder(context.Background(), sig.OrderTicket)
	case sdk.ActionCloseAll:
		positions, _ := lr.executor.OpenedOrders(context.Background())
		for _, p := range positions {
			if sig.Magic == 0 || p.Magic == sig.Magic {
				lr.executor.CloseOrder(context.Background(), p.Ticket, decimal.Zero)
			}
		}
	case sdk.ActionCancelAll:
		orders, _ := lr.executor.PendingOrders(context.Background())
		for _, o := range orders {
			if sig.Magic == 0 || o.Magic == sig.Magic {
				lr.executor.CancelOrder(context.Background(), o.Ticket)
			}
		}
	}
}
