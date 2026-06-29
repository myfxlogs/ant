package backtest

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
)

// Engine runs a backtest of a strategy on historical data.
type Engine struct {
	config   Config
	broker   *SimBroker
	strategy sdk.Strategy

	equity  []EquityPoint
	trades  []Trade
	bars    []sdk.Bar
}

// New creates a backtest engine.
func New(cfg Config, strategy sdk.Strategy, bars []sdk.Bar) *Engine {
	return &Engine{
		config:   cfg,
		broker:   NewSimBroker(cfg),
		strategy: strategy,
		bars:     bars,
	}
}

// Run executes the backtest and returns the result.
func (e *Engine) Run(ctx context.Context) (*Result, error) {
	startedAt := time.Now()

	// Build context
	btCtx := &backtestContext{
		broker:  e.broker,
		symbol:  e.config.Symbol,
		tf:      e.config.Timeframe,
		bars:    e.bars,
		ind:     &btIndicators{bars: e.bars},
		params:  e.config.Params,
		point:   e.config.SymbolPoint,
		digits:  e.config.SymbolDigits,
	}

	// OnInit
	if err := e.strategy.OnInit(btCtx); err != nil {
		return nil, err
	}

	// Walk through bars
	for i := 1; i < len(e.bars); i++ {
		if err := ctx.Err(); err != nil {
			break
		}
		bar := e.bars[i]
		e.broker.SetBar(i)
		e.broker.SetBarTime(time.UnixMilli(bar.Timestamp))

		// Update context with current bar slice
		btCtx.barIndex = i
		btCtx.currentBar = bar

		// Check pending orders for fills
		e.checkPendingOrders(bar)

		// Check stop loss / take profit
		e.checkSLTP(bar)

		// Call strategy — check for TickStrategy interface first
		var sig *sdk.Signal
		var err error
		if ts, ok := e.strategy.(sdk.TickStrategy); ok {
			bid := bar.Close
			ask := bar.Close
			sig, err = ts.OnTick(btCtx, bid, ask)
		} else {
			sig, err = e.strategy.OnBar(btCtx, e.config.Timeframe)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "backtest: OnBar error at bar %d: %v\n", i, err)
			continue
		}
		if sig != nil {
			e.dispatchSignal(sig, bar)
		}

		// Update equity with floating P&L from open positions
		equity := e.broker.balance
		closePrice := decimal.NewFromFloat(bar.Close.InexactFloat64())
		for _, pos := range e.broker.positions {
			floating := closePrice.Sub(pos.Price).Mul(pos.Volume)
			if pos.Side == sdk.SideSell {
				floating = floating.Neg()
			}
			equity = equity.Add(floating).Sub(pos.Commission).Sub(pos.Swap)
		}
		e.equity = append(e.equity, EquityPoint{
			Time:   time.UnixMilli(bar.Timestamp),
			Equity: equity,
			Bar:    i,
		})
	}

	// OnDeinit
	e.strategy.OnDeinit(btCtx, "backtest_complete")

	// Calculate metrics (uses existing antv1.BacktestMetrics)
	metrics := CalculateMetrics(e.config.InitialCapital, e.equity, e.trades)

	return &Result{
		Config:     e.config,
		Metrics:    metrics,
		Equity:     e.equity,
		Trades:     e.trades,
		StartedAt:  startedAt,
		FinishedAt: time.Now(),
	}, nil
}

func (e *Engine) dispatchSignal(sig *sdk.Signal, bar sdk.Bar) {
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
		price := decimal.NewFromFloat(bar.Close.InexactFloat64())
		if sig.Price.IsPositive() {
			price = sig.Price
		}
		e.broker.OrderSend(sdk.OrderRequest{
			Symbol:     sig.Symbol,
			Side:       side,
			Type:       ot,
			Volume:     sig.Volume,
			Price:      price,
			StopLoss:   sig.StopLoss,
			TakeProfit: sig.TakeProfit,
			Comment:    sig.Comment,
			Magic:      sig.Magic,
		})
	case sdk.ActionClose:
		e.broker.PositionClose(sig.OrderTicket, decimal.Zero)
	case sdk.ActionCancel:
		e.broker.OrderDelete(sig.OrderTicket)
	case sdk.ActionCloseAll:
		for _, p := range e.broker.Positions(sig.Magic) {
			e.broker.PositionClose(p.Ticket, decimal.Zero)
		}
	}
}

func (e *Engine) checkPendingOrders(bar sdk.Bar) {
	for i := 0; i < len(e.broker.pending); i++ {
		ord := e.broker.pending[i]
		high := decimal.NewFromFloat(bar.High.InexactFloat64())
		low := decimal.NewFromFloat(bar.Low.InexactFloat64())

		filled := false
		if ord.OrderType == sdk.OrderLimit {
			if ord.Side == sdk.SideBuy && low.LessThanOrEqual(ord.Price) {
				filled = true
			}
			if ord.Side == sdk.SideSell && high.GreaterThanOrEqual(ord.Price) {
				filled = true
			}
		}
		if ord.OrderType == sdk.OrderStop {
			if ord.Side == sdk.SideBuy && high.GreaterThanOrEqual(ord.Price) {
				filled = true
			}
			if ord.Side == sdk.SideSell && low.LessThanOrEqual(ord.Price) {
				filled = true
			}
		}

		if filled {
			ord.State = OrderOpen
			e.broker.positions = append(e.broker.positions, ord)
			e.broker.pending = append(e.broker.pending[:i], e.broker.pending[i+1:]...)
			i--
		}
	}
}

func (e *Engine) checkSLTP(bar sdk.Bar) {
	high := decimal.NewFromFloat(bar.High.InexactFloat64())
	low := decimal.NewFromFloat(bar.Low.InexactFloat64())
	close := decimal.NewFromFloat(bar.Close.InexactFloat64())

	for i := 0; i < len(e.broker.positions); i++ {
		pos := e.broker.positions[i]
		closed := false

		if pos.Side == sdk.SideBuy {
			if pos.TakeProfit.IsPositive() && high.GreaterThanOrEqual(pos.TakeProfit) {
				pos.ClosePrice = pos.TakeProfit
				closed = true
			}
			if pos.StopLoss.IsPositive() && low.LessThanOrEqual(pos.StopLoss) {
				pos.ClosePrice = pos.StopLoss
				closed = true
			}
		} else {
			if pos.TakeProfit.IsPositive() && low.LessThanOrEqual(pos.TakeProfit) {
				pos.ClosePrice = pos.TakeProfit
				closed = true
			}
			if pos.StopLoss.IsPositive() && high.GreaterThanOrEqual(pos.StopLoss) {
				pos.ClosePrice = pos.StopLoss
				closed = true
			}
		}

		if closed {
			if pos.ClosePrice.IsZero() {
				pos.ClosePrice = close
			}
			// Apply swap for overnight positions (simplified: 1 day per bar)
			e.broker.applySwap(pos, 1)
			pos.Profit = pos.ClosePrice.Sub(pos.Price).Mul(pos.Volume)
			if pos.Side == sdk.SideSell {
				pos.Profit = pos.Profit.Neg()
			}
			pos.State = OrderClosed
			pos.CloseTime = time.UnixMilli(bar.Timestamp)
			e.broker.history = append(e.broker.history, pos)
			e.trades = append(e.trades, Trade{
				Symbol:     pos.Symbol,
				Side:       pos.Side,
				EntryTime:  pos.OpenTime,
				ExitTime:   pos.CloseTime,
				EntryPrice: pos.Price,
				ExitPrice:  pos.ClosePrice,
				Volume:     pos.Volume,
				Profit:     pos.Profit,
				Commission: pos.Commission,
				Comment:    pos.Comment,
			})
			e.broker.equity = e.broker.equity.Add(pos.Profit)
			e.broker.balance = e.broker.balance.Add(pos.Profit)
			e.broker.positions = append(e.broker.positions[:i], e.broker.positions[i+1:]...)
			i--
		}
	}
}

// ── Backtest context (implements sdk.Context) ──────────────────────

type backtestContext struct {
	broker     *SimBroker
	symbol     string
	tf         string
	bars       []sdk.Bar
	barIndex   int
	currentBar sdk.Bar
	ind        *btIndicators
	params     map[string]string
	point      decimal.Decimal
	digits     int32
}

func (c *backtestContext) Bars() sdk.BarSeries          { return &btBarSeries{bars: c.bars[:c.barIndex+1]} }
func (c *backtestContext) BarsTF(_ string) sdk.BarSeries { return c.Bars() }
func (c *backtestContext) Symbol() string                { return c.symbol }
func (c *backtestContext) Timeframe() string             { return c.tf }
func (c *backtestContext) Point() decimal.Decimal {
	if !c.point.IsZero() {
		return c.point
	}
	return decimal.NewFromFloat(0.00001)
}
func (c *backtestContext) Pip() decimal.Decimal          { return c.Point().Mul(decimal.NewFromInt(10)) }
func (c *backtestContext) Digits() int32 {
	if c.digits > 0 {
		return c.digits
	}
	return 5
}
func (c *backtestContext) Ask() decimal.Decimal {
	spread := c.broker.config.Slippage
	return c.currentBar.Close.Add(spread)
}
func (c *backtestContext) Bid() decimal.Decimal          { return c.currentBar.Close }
func (c *backtestContext) Spread() decimal.Decimal       { return c.broker.config.Slippage }
func (c *backtestContext) Account() sdk.AccountInfo      { return c.broker.Account() }
func (c *backtestContext) Mode() sdk.AccountMode         { return sdk.ModeHedging }
func (c *backtestContext) Broker() sdk.Broker            { return c.broker }
func (c *backtestContext) Indicators() sdk.IndicatorSet  { return c.ind }
func (c *backtestContext) SetTimer(int)                  {}
func (c *backtestContext) KillTimer()                    {}
func (c *backtestContext) Log(string)                    {}
func (c *backtestContext) ServerTime() int64             { return c.currentBar.Timestamp }

func (c *backtestContext) Param(name string, defaultVal interface{}) interface{} {
	if c.params != nil {
		if v, ok := c.params[name]; ok {
			return v
		}
	}
	return defaultVal
}
func (c *backtestContext) ParamDecimal(name string, d decimal.Decimal) decimal.Decimal {
	if c.params != nil {
		if v, ok := c.params[name]; ok {
			if parsed, err := decimal.NewFromString(v); err == nil {
				return parsed
			}
		}
	}
	return d
}
func (c *backtestContext) ParamInt(name string, d int) int {
	if c.params != nil {
		if v, ok := c.params[name]; ok {
			if parsed, err := strconv.Atoi(v); err == nil {
				return parsed
			}
		}
	}
	return d
}
func (c *backtestContext) ParamString(name, d string) string {
	if c.params != nil {
		if v, ok := c.params[name]; ok {
			return v
		}
	}
	return d
}
func (c *backtestContext) ParamBool(name string, d bool) bool {
	if c.params != nil {
		if v, ok := c.params[name]; ok {
			if v == "true" || v == "1" {
				return true
			}
			return false
		}
	}
	return d
}

// ── Bar series ────────────────────────────────────────────────────

type btBarSeries struct{ bars []sdk.Bar }

func (b *btBarSeries) Open(shift int) decimal.Decimal {
	idx := len(b.bars) - 1 - shift
	if idx < 0 || idx >= len(b.bars) {
		return decimal.Zero
	}
	return b.bars[idx].Open
}
func (b *btBarSeries) High(shift int) decimal.Decimal {
	idx := len(b.bars) - 1 - shift
	if idx < 0 || idx >= len(b.bars) {
		return decimal.Zero
	}
	return b.bars[idx].High
}
func (b *btBarSeries) Low(shift int) decimal.Decimal {
	idx := len(b.bars) - 1 - shift
	if idx < 0 || idx >= len(b.bars) {
		return decimal.Zero
	}
	return b.bars[idx].Low
}
func (b *btBarSeries) Close(shift int) decimal.Decimal {
	idx := len(b.bars) - 1 - shift
	if idx < 0 || idx >= len(b.bars) {
		return decimal.Zero
	}
	return b.bars[idx].Close
}
func (b *btBarSeries) Volume(shift int) int64 {
	idx := len(b.bars) - 1 - shift
	if idx < 0 || idx >= len(b.bars) {
		return 0
	}
	return b.bars[idx].Volume
}
func (b *btBarSeries) Time(shift int) int64 {
	idx := len(b.bars) - 1 - shift
	if idx < 0 || idx >= len(b.bars) {
		return 0
	}
	return b.bars[idx].Timestamp
}
func (b *btBarSeries) Len() int                    { return len(b.bars) }
func (b *btBarSeries) Slice(n int) sdk.BarSeries {
	if n >= len(b.bars) {
		return &btBarSeries{bars: b.bars}
	}
	return &btBarSeries{bars: b.bars[len(b.bars)-n:]}
}
func (b *btBarSeries) Timeframe() string            { return "" }
func (b *btBarSeries) Symbol() string               { return "" }

// btIndicators implementations moved to indicators_decimal.go.
