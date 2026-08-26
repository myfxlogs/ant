package backtest

import (
	"context"
	"fmt"
	"os"
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

	btCtx := &backtestContext{
		broker: e.broker,
		symbol: e.config.Symbol,
		tf:     e.config.Timeframe,
		bars:   e.bars,
		params: e.config.Params,
		point:  e.config.SymbolPoint,
		digits: e.config.SymbolDigits,
	}
	if len(e.config.ExtraSymbolBars) > 0 {
		btCtx.extraBars = make(map[string][]sdk.Bar, len(e.config.ExtraSymbolBars))
		btCtx.extraBarIndex = make(map[string]int, len(e.config.ExtraSymbolBars))
		for sym, symBars := range e.config.ExtraSymbolBars {
			btCtx.extraBars[sym] = symBars
			btCtx.extraBarIndex[sym] = -1
		}
	}
	btCtx.ind = &btIndicators{bars: e.bars, barIdx: &btCtx.barIndex}

	for i := 1; i < len(e.bars); i++ {
		if e.bars[i].Timestamp < e.bars[i-1].Timestamp {
			return nil, fmt.Errorf("bars are not chronologically ordered at index %d: %d < %d",
				i, e.bars[i].Timestamp, e.bars[i-1].Timestamp)
		}
	}

	if err := e.strategy.OnInit(btCtx); err != nil {
		return nil, err
	}

	var pendingSignals []sdk.Signal

	for i := 1; i < len(e.bars); i++ {
		if err := ctx.Err(); err != nil {
			break
		}
		bar := e.bars[i]
		e.broker.SetBar(i)
		e.broker.SetBarTime(time.UnixMilli(bar.Timestamp))
		e.broker.SetBarPrice(bar.Close)

		btCtx.barIndex = i
		btCtx.currentBar = bar

		// Execute delayed signals from previous bar (next_bar_open mode).
		// SetBarPrice(bar.Open) so PositionClose uses the open price, not close.
		if len(pendingSignals) > 0 {
			e.broker.SetBarPrice(bar.Open)
			for _, sig := range pendingSignals {
				e.dispatchSignal(&sig, bar)
			}
			e.broker.SetBarPrice(bar.Close)
			pendingSignals = nil
		}

		e.advanceExtraBars(btCtx, bar.Timestamp)
		e.checkPendingOrders(bar)
		if e.broker.config.SimulationMode == "OHLC_PATH" {
			e.checkSLTPPath(bar)
		} else {
			e.checkSLTP(bar)
		}

		sig, err := e.runStrategySignal(btCtx, bar)
		if err != nil {
			// VM-RUNTIME-FAILCLOSED-1: fail-closed — stop backtest on strategy error.
			return nil, fmt.Errorf("backtest: strategy event failed at bar %d: %w", i, err)
		}
		if sig != nil {
			if e.config.SignalTiming == "same_bar_close" {
				e.dispatchSignal(sig, bar)
			} else {
				pendingSignals = append(pendingSignals, *sig)
			}
		}

		equity := e.computeEquity(bar)
		e.checkMarginCall(equity, bar)
		e.equity = append(e.equity, EquityPoint{
			Time:   time.UnixMilli(bar.Timestamp),
			Equity: equity,
			Bar:    i,
		})
	}

	if err := e.strategy.OnDeinit(btCtx, "backtest_complete"); err != nil {
		return nil, err
	}

	allTrades := append(e.trades, e.broker.Trades()...)
	metrics := CalculateMetrics(e.config.InitialCapital, e.equity, allTrades)

	return &Result{
		Config:       e.config,
		Metrics:      metrics,
		Equity:       e.equity,
		Trades:       allTrades,
		FinalBalance: e.broker.balance,
		StartedAt:    startedAt,
		FinishedAt:   time.Now(),
		Logs:         btCtx.logs,
	}, nil
}

func (e *Engine) advanceExtraBars(btCtx *backtestContext, currentTs int64) {
	for sym, symBars := range btCtx.extraBars {
		idx := btCtx.extraBarIndex[sym]
		for idx+1 < len(symBars) && symBars[idx+1].Timestamp <= currentTs {
			idx++
		}
		btCtx.extraBarIndex[sym] = idx
	}
}

func (e *Engine) runStrategySignal(btCtx *backtestContext, bar sdk.Bar) (*sdk.Signal, error) {
	useTick := false
	if _, ok := e.strategy.(sdk.TickStrategy); ok {
		if tc, ok2 := e.strategy.(sdk.TickCapable); ok2 {
			useTick = tc.HasOnTick()
		} else {
			useTick = true
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
		return ts.OnTick(btCtx, bid, ask)
	}
	return e.strategy.OnBar(btCtx, e.config.Timeframe)
}

func (e *Engine) computeEquity(bar sdk.Bar) decimal.Decimal {
	equity := e.broker.balance
	contractSize := e.config.ContractSize
	if contractSize.IsZero() {
		contractSize = decimal.NewFromInt(100000)
	}
	for _, pos := range e.broker.positions {
		floating := bar.Close.Sub(pos.Price).Mul(pos.Volume).Mul(contractSize)
		if pos.Side == sdk.SideSell {
			floating = floating.Neg()
		}
		equity = equity.Add(floating)
	}
	return equity
}

// checkMarginCall force-closes all positions when equity drops below
// the margin call threshold. The threshold is the ratio of equity to
// used margin; when equity/margin < MarginCallLevel, all positions are closed.
func (e *Engine) checkMarginCall(equity decimal.Decimal, bar sdk.Bar) {
	if e.config.MarginCallLevel.IsZero() {
		return
	}
	contractSize := e.config.ContractSize
	if contractSize.IsZero() {
		contractSize = decimal.NewFromInt(100000)
	}
	usedMargin := decimal.Zero
	for _, pos := range e.broker.positions {
		notional := pos.Volume.Mul(contractSize).Mul(pos.Price)
		margin := notional.Div(decimal.NewFromInt(int64(e.config.Leverage)))
		usedMargin = usedMargin.Add(margin)
	}
	if usedMargin.IsZero() {
		return
	}
	ratio := equity.Div(usedMargin)
	if ratio.LessThan(e.config.MarginCallLevel) {
		for i := len(e.broker.positions) - 1; i >= 0; i-- {
			pos := e.broker.positions[i]
			pos.ClosePrice = bar.Close
			e.broker.PositionClose(pos.Ticket, pos.Volume)
		}
	}
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
		if !e.broker.currentPrice.IsZero() {
			price = e.broker.currentPrice
		}
		if sig.Price.IsPositive() {
			price = sig.Price
		}
		if _, err := e.broker.OrderSend(sdk.OrderRequest{
			Symbol:     sig.Symbol,
			Side:       side,
			Type:       ot,
			Volume:     sig.Volume,
			Price:      price,
			StopLoss:   sig.StopLoss,
			TakeProfit: sig.TakeProfit,
			Comment:    sig.Comment,
			Magic:      sig.Magic,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "backtest: OrderSend error at bar %d: %v\n", e.broker.currentBar, err)
		}
	case sdk.ActionClose:
		if _, err := e.broker.PositionClose(sig.OrderTicket, decimal.Zero); err != nil {
			fmt.Fprintf(os.Stderr, "backtest: PositionClose error at bar %d: %v\n", e.broker.currentBar, err)
		}
	case sdk.ActionCancel:
		if _, err := e.broker.OrderDelete(sig.OrderTicket); err != nil {
			fmt.Fprintf(os.Stderr, "backtest: OrderDelete error at bar %d: %v\n", e.broker.currentBar, err)
		}
	case sdk.ActionCloseAll:
		for _, p := range e.broker.Positions(sig.Magic) {
			if _, err := e.broker.PositionClose(p.Ticket, decimal.Zero); err != nil {
				fmt.Fprintf(os.Stderr, "backtest: PositionClose error at bar %d: %v\n", e.broker.currentBar, err)
			}
		}
	case sdk.ActionCancelAll:
		for _, o := range e.broker.Orders(sig.Magic) {
			if _, err := e.broker.OrderDelete(o.Ticket); err != nil {
				fmt.Fprintf(os.Stderr, "backtest: OrderDelete error at bar %d: %v\n", e.broker.currentBar, err)
			}
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
			// Deferred commission and margin check at fill time (real MT4 semantics:
			// pending orders incur commission and margin only when they fill).
			contractSize := e.broker.config.ContractSize
			if contractSize.IsZero() {
				contractSize = decimal.NewFromInt(100000)
			}
			notional := ord.Volume.Mul(contractSize).Mul(ord.Price)
			margin := notional.Div(decimal.NewFromInt(int64(e.broker.config.Leverage)))
			equityWithFloating := e.broker.Account().Equity
			if equityWithFloating.LessThan(margin) {
				// Insufficient margin at fill time — cancel the pending order.
				ord.State = OrderCancelled
				e.broker.history = append(e.broker.history, ord)
				e.broker.pending = append(e.broker.pending[:i], e.broker.pending[i+1:]...)
				i--
				continue
			}
			e.broker.applyCommission(ord)
			ord.State = OrderOpen
			e.broker.positions = append(e.broker.positions, ord)
			e.broker.pending = append(e.broker.pending[:i], e.broker.pending[i+1:]...)
			i--
		}
	}
}

// checkBuySLTP checks SL/TP for a buy position. Returns (closed, closePrice).
// Conservative: if both SL and TP are in the bar range, SL hits first.
func checkBuySLTP(open, high, low, sl, tp decimal.Decimal) (bool, decimal.Decimal) {
	if open.IsPositive() && sl.IsPositive() && open.LessThanOrEqual(sl) {
		return true, open
	}
	if open.IsPositive() && tp.IsPositive() && open.GreaterThanOrEqual(tp) {
		return true, open
	}
	if sl.IsPositive() && low.LessThanOrEqual(sl) {
		return true, sl
	}
	if tp.IsPositive() && high.GreaterThanOrEqual(tp) {
		return true, tp
	}
	return false, decimal.Zero
}

// checkSellSLTP checks SL/TP for a sell position. Returns (closed, closePrice).
// Conservative: if both SL and TP are in the bar range, SL hits first.
func checkSellSLTP(open, high, low, sl, tp decimal.Decimal) (bool, decimal.Decimal) {
	if open.IsPositive() && sl.IsPositive() && open.GreaterThanOrEqual(sl) {
		return true, open
	}
	if open.IsPositive() && tp.IsPositive() && open.LessThanOrEqual(tp) {
		return true, open
	}
	if sl.IsPositive() && high.GreaterThanOrEqual(sl) {
		return true, sl
	}
	if tp.IsPositive() && low.LessThanOrEqual(tp) {
		return true, tp
	}
	return false, decimal.Zero
}

func (e *Engine) checkSLTP(bar sdk.Bar) {
	for i := 0; i < len(e.broker.positions); i++ {
		pos := e.broker.positions[i]
		var closed bool
		var closePrice decimal.Decimal

		if pos.Side == sdk.SideBuy {
			closed, closePrice = checkBuySLTP(bar.Open, bar.High, bar.Low, pos.StopLoss, pos.TakeProfit)
		} else {
			closed, closePrice = checkSellSLTP(bar.Open, bar.High, bar.Low, pos.StopLoss, pos.TakeProfit)
		}

		if closed {
			if closePrice.IsZero() {
				closePrice = bar.Close
			}
			pos.ClosePrice = closePrice
			heldDuration := time.UnixMilli(bar.Timestamp).Sub(pos.OpenTime)
			days := int64(heldDuration.Hours() / 24)
			if days < 0 {
				days = 0
			}
			e.broker.applySwap(pos, pos.Volume, int(days))
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
