package backtest

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// Engine runs a backtest of a strategy on historical data.
type Engine struct {
	config   Config
	broker   *SimBroker
	strategy sdk.Strategy

	equity []EquityPoint
	trades []Trade
	bars   []sdk.Bar
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

// Broker returns the internal SimBroker for inspection (mainly for testing).
func (e *Engine) Broker() *SimBroker {
	return e.broker
}

// Run executes the backtest and returns the result.
func (e *Engine) Run(ctx context.Context) (*Result, error) {
	startedAt := time.Now()

	// Build context
	btCtx := &backtestContext{
		broker: e.broker,
		symbol: e.config.Symbol,
		tf:     e.config.Timeframe,
		bars:   e.bars,
		params: e.config.Params,
		point:  e.config.SymbolPoint,
		digits: e.config.SymbolDigits,
	}
	// Initialize multi-symbol bar data
	if len(e.config.ExtraSymbolBars) > 0 {
		btCtx.extraBars = make(map[string][]sdk.Bar, len(e.config.ExtraSymbolBars))
		btCtx.extraBarIndex = make(map[string]int, len(e.config.ExtraSymbolBars))
		for sym, symBars := range e.config.ExtraSymbolBars {
			btCtx.extraBars[sym] = symBars
			btCtx.extraBarIndex[sym] = -1
		}
	}
	btCtx.ind = &btIndicators{bars: e.bars, barIdx: &btCtx.barIndex}

	// Validate bar ordering — unsorted bars produce silent garbage.
	for i := 1; i < len(e.bars); i++ {
		if e.bars[i].Timestamp < e.bars[i-1].Timestamp {
			return nil, fmt.Errorf("bars are not chronologically ordered at index %d: %d < %d",
				i, e.bars[i].Timestamp, e.bars[i-1].Timestamp)
		}
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
		e.broker.SetBarPrice(bar.Close)

		// Update context with current bar slice
		btCtx.barIndex = i
		btCtx.currentBar = bar

		// Advance extra symbol bar indices to the bar closest to (but not after) the current bar's timestamp.
		// This ensures no future data leakage — secondary symbols only see bars up to the current time.
		currentTs := bar.Timestamp
		for sym, symBars := range btCtx.extraBars {
			idx := btCtx.extraBarIndex[sym]
			for idx+1 < len(symBars) && symBars[idx+1].Timestamp <= currentTs {
				idx++
			}
			btCtx.extraBarIndex[sym] = idx
		}

		// Check pending orders for fills
		e.checkPendingOrders(bar)

		// Check stop loss / take profit
		e.checkSLTP(bar)

		// Call strategy — check for TickStrategy interface first
		var sig *sdk.Signal
		var err error
		useTick := false
		if _, ok := e.strategy.(sdk.TickStrategy); ok {
			if tc, ok2 := e.strategy.(sdk.TickCapable); ok2 {
				useTick = tc.HasOnTick()
			} else {
				useTick = true // implements TickStrategy but not TickCapable — assume OnTick
			}
		}
		if useTick {
			ts := e.strategy.(sdk.TickStrategy)
			spread := e.broker.config.Spread
			if spread.IsZero() {
				spread = e.broker.config.Slippage
			}
			bid := bar.Close
			ask := bar.Close.Add(spread)
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
		closePrice := bar.Close
		contractSize := e.config.ContractSize
		if contractSize.IsZero() {
			contractSize = decimal.NewFromInt(100000)
		}
		for _, pos := range e.broker.positions {
			floating := closePrice.Sub(pos.Price).Mul(pos.Volume).Mul(contractSize)
			if pos.Side == sdk.SideSell {
				floating = floating.Neg()
			}
			equity = equity.Add(floating)
		}
		e.equity = append(e.equity, EquityPoint{
			Time:   time.UnixMilli(bar.Timestamp),
			Equity: equity,
			Bar:    i,
		})
	}

	// OnDeinit
	if err := e.strategy.OnDeinit(btCtx, "backtest_complete"); err != nil {
		return nil, err
	}

	// Merge trades from broker (strategy-closed) and engine (SL/TP-closed)
	allTrades := append(e.trades, e.broker.Trades()...)

	// Calculate metrics (uses existing antv1.BacktestMetrics)
	metrics := CalculateMetrics(e.config.InitialCapital, e.equity, allTrades)

	return &Result{
		Config:     e.config,
		Metrics:    metrics,
		Equity:     e.equity,
		Trades:     allTrades,
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
		price := bar.Close
		if sig.Price.IsPositive() {
			price = sig.Price
		}
		_, _ = e.broker.OrderSend(sdk.OrderRequest{
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
		_, _ = e.broker.PositionClose(sig.OrderTicket, decimal.Zero)
	case sdk.ActionCancel:
		_, _ = e.broker.OrderDelete(sig.OrderTicket)
	case sdk.ActionCloseAll:
		for _, p := range e.broker.Positions(sig.Magic) {
			_, _ = e.broker.PositionClose(p.Ticket, decimal.Zero)
		}
	}
}

func (e *Engine) checkPendingOrders(bar sdk.Bar) {
	for i := 0; i < len(e.broker.pending); i++ {
		ord := e.broker.pending[i]
		high := bar.High
		low := bar.Low

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
	high := bar.High
	low := bar.Low
	close := bar.Close

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
			// Apply swap based on actual time held (not per-bar=1-day)
			heldDuration := time.UnixMilli(bar.Timestamp).Sub(pos.OpenTime)
			days := int64(heldDuration.Hours() / 24)
			if days < 0 {
				days = 0
			}
			e.broker.applySwap(pos, int(days))
			contractSize := e.broker.config.ContractSize
			if contractSize.IsZero() {
				contractSize = decimal.NewFromInt(100000)
			}
			pos.Profit = pos.ClosePrice.Sub(pos.Price).Mul(pos.Volume).Mul(contractSize)
			if pos.Side == sdk.SideSell {
				pos.Profit = pos.Profit.Neg()
			}
			pos.State = OrderClosed
			pos.CloseTime = time.UnixMilli(bar.Timestamp)
			e.broker.history = append(e.broker.history, pos)
			e.broker.recordDeal(pos, pos.Volume, pos.Profit, pos.CloseTime)
			e.broker.recordTrade(pos)
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

	// Multi-symbol support: extra symbol bar data and their current bar indices.
	extraBars     map[string][]sdk.Bar
	extraBarIndex map[string]int
}

func (c *backtestContext) Bars() sdk.BarSeries { return sdk.BarsToSlice(c.bars[:c.barIndex+1]) }

func (c *backtestContext) BarsTF(tf string) sdk.BarSeries {
	if tf == "" || tf == c.tf {
		return c.Bars()
	}
	// Aggregate only bars up to current barIndex — no future data leakage.
	// MT4 semantics: shift=0 on higher TF returns the bar containing the current
	// lower-TF bar, with OHLCV accumulated only from bars seen so far.
	visible := c.bars[:c.barIndex+1]
	aggregated := aggregateBars(visible, tf)
	return sdk.BarsToSlice(aggregated)
}

func (c *backtestContext) Symbol() string    { return c.symbol }
func (c *backtestContext) Timeframe() string { return c.tf }
func (c *backtestContext) Point() decimal.Decimal {
	if !c.point.IsZero() {
		return c.point
	}
	return decimal.NewFromFloat(0.00001)
}
func (c *backtestContext) Pip() decimal.Decimal { return c.Point().Mul(decimal.NewFromInt(10)) }
func (c *backtestContext) Digits() int32 {
	if c.digits > 0 {
		return c.digits
	}
	return 5
}
func (c *backtestContext) Ask() decimal.Decimal {
	spread := c.broker.config.Spread
	if spread.IsZero() {
		spread = c.broker.config.Slippage // fallback: use slippage as spread if spread not set
	}
	return c.currentBar.Close.Add(spread)
}
func (c *backtestContext) Bid() decimal.Decimal { return c.currentBar.Close }
func (c *backtestContext) Spread() decimal.Decimal {
	if !c.broker.config.Spread.IsZero() {
		return c.broker.config.Spread
	}
	return c.broker.config.Slippage
}
func (c *backtestContext) Account() sdk.AccountInfo     { return c.broker.Account() }
func (c *backtestContext) Mode() sdk.AccountMode        { return sdk.ModeHedging }
func (c *backtestContext) Broker() sdk.Broker           { return c.broker }
func (c *backtestContext) Indicators() sdk.IndicatorSet { return c.ind }
func (c *backtestContext) SetTimer(int)                 {}
func (c *backtestContext) KillTimer()                   {}
func (c *backtestContext) Log(string)                   {}
func (c *backtestContext) ServerTime() int64            { return c.currentBar.Timestamp }
func (c *backtestContext) GoContext() context.Context   { return context.Background() }

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

// BarsForSymbol moved to bars.go.
// btIndicators implementations moved to indicators_decimal.go.
