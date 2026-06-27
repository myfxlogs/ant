package backtest

import (
	"context"
	"fmt"
	"math"
	"os"
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

		// Update context with current bar slice
		btCtx.barIndex = i
		btCtx.currentBar = bar

		// Check pending orders for fills
		e.checkPendingOrders(bar)

		// Check stop loss / take profit
		e.checkSLTP(bar)

		// Call strategy
		sig, err := e.strategy.OnBar(btCtx, e.config.Timeframe)
		if err != nil {
			fmt.Fprintf(os.Stderr, "backtest: OnBar error at bar %d: %v\n", i, err)
			continue
		}
		if sig != nil {
			e.dispatchSignal(sig, bar)
		}

		// Update equity
		equity := e.broker.equity
		for _, pos := range e.broker.positions {
			equity = equity.Add(pos.Profit)
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
			pos.CloseTime = time.Now()
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
}

func (c *backtestContext) Bars() sdk.BarSeries          { return &btBarSeries{bars: c.bars[:c.barIndex+1]} }
func (c *backtestContext) BarsTF(_ string) sdk.BarSeries { return c.Bars() }
func (c *backtestContext) Symbol() string                { return c.symbol }
func (c *backtestContext) Timeframe() string             { return c.tf }
func (c *backtestContext) Point() decimal.Decimal        { return decimal.NewFromFloat(0.00001) }
func (c *backtestContext) Pip() decimal.Decimal          { return decimal.NewFromFloat(0.0001) }
func (c *backtestContext) Digits() int32                 { return 5 }
func (c *backtestContext) Ask() decimal.Decimal          { return c.currentBar.Close }
func (c *backtestContext) Bid() decimal.Decimal          { return c.currentBar.Close }
func (c *backtestContext) Account() sdk.AccountInfo      { return c.broker.Account() }
func (c *backtestContext) Mode() sdk.AccountMode         { return sdk.ModeHedging }
func (c *backtestContext) Broker() sdk.Broker            { return c.broker }
func (c *backtestContext) Indicators() sdk.IndicatorSet  { return c.ind }
func (c *backtestContext) SetTimer(int)                  {}
func (c *backtestContext) KillTimer()                    {}
func (c *backtestContext) Log(string)                    {}
func (c *backtestContext) ServerTime() int64             { return 0 }

func (c *backtestContext) Param(name string, defaultVal interface{}) interface{}         { return defaultVal }
func (c *backtestContext) ParamDecimal(name string, d decimal.Decimal) decimal.Decimal   { return d }
func (c *backtestContext) ParamInt(name string, d int) int                               { return d }
func (c *backtestContext) ParamString(name, d string) string                             { return d }
func (c *backtestContext) ParamBool(name string, d bool) bool                            { return d }

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

// ── Indicators (stub for backtesting) ──────────────────────────────

type btIndicators struct{ bars []sdk.Bar }

func (i *btIndicators) EMA(period, shift int) float64 {
	if len(i.bars) < period+shift {
		return 0
	}
	alpha := 2.0 / float64(period+1)
	var ema float64
	// Seed with SMA
	for j := period + shift - 1; j >= shift; j-- {
		p, _ := i.bars[j].Close.Float64()
		if j == period+shift-1 {
			ema = p
		} else {
			ema = p*alpha + ema*(1-alpha)
		}
	}
	return ema
}
func (i *btIndicators) MA(period, shift int, method string) float64 { return i.EMA(period, shift) }
func (i *btIndicators) RSI(period, shift int) float64 {
	if len(i.bars) < period+shift+1 {
		return 0
	}
	var avgGain, avgLoss float64
	for j := shift + 1; j <= shift+period; j++ {
		curr, _ := i.bars[j-1].Close.Float64()
		prev, _ := i.bars[j].Close.Float64()
		diff := curr - prev
		if diff > 0 {
			avgGain += diff
		} else {
			avgLoss -= diff
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func (i *btIndicators) MACD(fast, slow, signal, shift int) float64 {
	return i.EMA(fast, shift) - i.EMA(slow, shift)
}

func (i *btIndicators) MACDSignal(fast, slow, signal, shift int) float64 {
	// Simplified: EMA of MACD line at the given shift
	macd := i.MACD(fast, slow, signal, shift)
	return macd // first-order approximation
}

func (i *btIndicators) ATR(period, shift int) float64 {
	if len(i.bars) < period+shift+1 {
		return 0
	}
	var sumTR float64
	for j := shift; j < shift+period; j++ {
		high, _ := i.bars[j].High.Float64()
		low, _ := i.bars[j].Low.Float64()
		prevClose, _ := i.bars[j+1].Close.Float64()
		tr := high - low
		if d := high - prevClose; d > tr {
			tr = d
		} else if -d > tr {
			tr = -d
		}
		if d := prevClose - low; d > tr {
			tr = d
		} else if -d > tr {
			tr = -d
		}
		sumTR += tr
	}
	return sumTR / float64(period)
}

func (i *btIndicators) Bollinger(period int, deviation float64, shift int) (float64, float64, float64) {
	if len(i.bars) < period+shift {
		return 0, 0, 0
	}
	var sum float64
	for j := shift; j < shift+period; j++ {
		c, _ := i.bars[j].Close.Float64()
		sum += c
	}
	middle := sum / float64(period)
	var variance float64
	for j := shift; j < shift+period; j++ {
		c, _ := i.bars[j].Close.Float64()
		d := c - middle
		variance += d * d
	}
	std := 0.0
	if period > 1 {
		std = math.Sqrt(variance / float64(period-1))
	}
	return middle + deviation*std, middle, middle - deviation*std
}

func (i *btIndicators) Momentum(period, shift int) float64 {
	if len(i.bars) < period+shift {
		return 0
	}
	curr, _ := i.bars[shift].Close.Float64()
	prev, _ := i.bars[shift+period].Close.Float64()
	return curr - prev
}

func (i *btIndicators) StdDev(period, shift int) float64 {
	if len(i.bars) < period+shift {
		return 0
	}
	var sum float64
	for j := shift; j < shift+period; j++ {
		c, _ := i.bars[j].Close.Float64()
		sum += c
	}
	mean := sum / float64(period)
	var variance float64
	for j := shift; j < shift+period; j++ {
		c, _ := i.bars[j].Close.Float64()
		d := c - mean
		variance += d * d
	}
	if period > 1 {
		return math.Sqrt(variance / float64(period-1))
	}
	return 0
}

// Remaining indicators: implemented with direct bar slice access.
// btIndicators.bars is chronological (oldest first), shift=0 is latest bar.
func (i *btIndicators) Stochastic(kPeriod, dPeriod, slowing, shift int) (float64, float64) {
	n := len(i.bars)
	if n < kPeriod+shift {
		return 50, 50
	}
	// K = (Close - LowestLow) / (HighestHigh - LowestLow) * 100
	idx := n - 1 - shift
	var highestHigh, lowestLow float64
	for j := idx; j > idx-kPeriod && j >= 0; j-- {
		h, _ := i.bars[j].High.Float64()
		l, _ := i.bars[j].Low.Float64()
		if j == idx {
			highestHigh = h
			lowestLow = l
		} else {
			if h > highestHigh {
				highestHigh = h
			}
			if l < lowestLow {
				lowestLow = l
			}
		}
	}
	currentClose, _ := i.bars[idx].Close.Float64()
	k := 50.0
	if highestHigh != lowestLow {
		k = (currentClose - lowestLow) / (highestHigh - lowestLow) * 100
	}
	// D = SMA of K over dPeriod
	var kSum float64
	for d := 0; d < dPeriod; d++ {
		di := idx - d
		if di < 0 || di-kPeriod+1 < 0 {
			break
		}
		var hh, ll float64
		for j := di; j > di-kPeriod && j >= 0; j-- {
			h, _ := i.bars[j].High.Float64()
			l, _ := i.bars[j].Low.Float64()
			if j == di {
				hh = h
				ll = l
			} else {
				if h > hh {
					hh = h
				}
				if l < ll {
					ll = l
				}
			}
		}
		cc, _ := i.bars[di].Close.Float64()
		if hh == ll {
			kSum += 50
		} else {
			kSum += (cc - ll) / (hh - ll) * 100
		}
	}
	d := kSum / float64(dPeriod)
	return k, d
}

func (i *btIndicators) CCI(period, shift int) float64 {
	n := len(i.bars)
	if n < period+shift {
		return 0
	}
	idx := n - 1 - shift
	var sumTP float64
	for j := idx; j > idx-period && j >= 0; j-- {
		h, _ := i.bars[j].High.Float64()
		l, _ := i.bars[j].Low.Float64()
		c, _ := i.bars[j].Close.Float64()
		sumTP += (h + l + c) / 3
	}
	meanTP := sumTP / float64(period)
	var meanDev float64
	for j := idx; j > idx-period && j >= 0; j-- {
		h, _ := i.bars[j].High.Float64()
		l, _ := i.bars[j].Low.Float64()
		c, _ := i.bars[j].Close.Float64()
		tp := (h + l + c) / 3
		dev := tp - meanTP
		if dev < 0 {
			dev = -dev
		}
		meanDev += dev
	}
	meanDev /= float64(period)
	if meanDev == 0 {
		return 0
	}
	h, _ := i.bars[idx].High.Float64()
	l, _ := i.bars[idx].Low.Float64()
	c, _ := i.bars[idx].Close.Float64()
	currentTP := (h + l + c) / 3
	return (currentTP - meanTP) / (0.015 * meanDev)
}

func (i *btIndicators) ADX(period, shift int) float64 {
	n := len(i.bars)
	if n < period*2+shift {
		return 0
	}
	idx := n - 1 - shift
	var sumDX float64
	for j := idx; j > idx-period && j >= 1; j-- {
		high1, _ := i.bars[j].High.Float64()
		high2, _ := i.bars[j-1].High.Float64()
		low1, _ := i.bars[j].Low.Float64()
		low2, _ := i.bars[j-1].Low.Float64()
		prevClose, _ := i.bars[j-1].Close.Float64()
		plusDM := high1 - high2
		minusDM := low2 - low1
		if plusDM < 0 {
			plusDM = 0
		}
		if minusDM < 0 {
			minusDM = 0
		}
		if plusDM > minusDM {
			minusDM = 0
		} else {
			plusDM = 0
		}
		tr := high1 - low1
		if d := high1 - prevClose; d > tr {
			tr = d
		}
		if d := prevClose - low1; d > tr {
			tr = d
		}
		if tr == 0 {
			continue
		}
		sumDX += (plusDM - minusDM) / tr * 100
	}
	if sumDX < 0 {
		sumDX = -sumDX
	}
	return sumDX / float64(period)
}

func (i *btIndicators) MFI(period, shift int) float64 {
	n := len(i.bars)
	if n < period+shift+1 {
		return 0
	}
	idx := n - 1 - shift
	var posFlow, negFlow float64
	for j := idx; j > idx-period && j >= 1; j-- {
		h, _ := i.bars[j].High.Float64()
		l, _ := i.bars[j].Low.Float64()
		c, _ := i.bars[j].Close.Float64()
		prevC, _ := i.bars[j-1].Close.Float64()
		tp := (h + l + c) / 3
		prevTP := (h + l + prevC) / 3
		flow := tp * float64(i.bars[j].Volume)
		if tp > prevTP {
			posFlow += flow
		} else if tp < prevTP {
			negFlow += flow
		}
	}
	if negFlow == 0 {
		return 100
	}
	return 100 - 100/(1+posFlow/negFlow)
}

func (i *btIndicators) OBV(shift int) float64 {
	n := len(i.bars)
	if n < shift+2 {
		return 0
	}
	var obv float64
	for j := n - 1; j > shift; j-- {
		curr, _ := i.bars[j-1].Close.Float64()
		prev, _ := i.bars[j].Close.Float64()
		vol := float64(i.bars[j-1].Volume)
		if curr > prev {
			obv += vol
		} else if curr < prev {
			obv -= vol
		}
	}
	return obv
}

func (i *btIndicators) SAR(step, maximum float64, shift int) float64 {
	n := len(i.bars)
	if n < shift+2 {
		return 0
	}
	idx := n - 1 - shift
	high, _ := i.bars[idx].High.Float64()
	low, _ := i.bars[idx].Low.Float64()
	prevHigh, _ := i.bars[idx-1].High.Float64()
	prevLow, _ := i.bars[idx-1].Low.Float64()
	ep := prevHigh
	sar := prevLow
	if sar > low {
		sar = low
	}
	accel := step
	if accel > maximum {
		accel = maximum
	}
	sar = sar + accel*(ep-sar)
	if sar > low {
		sar = low
	}
	_ = high
	return sar
}

func (i *btIndicators) WPR(period, shift int) float64 {
	n := len(i.bars)
	if n < period+shift {
		return 0
	}
	idx := n - 1 - shift
	var highestHigh, lowestLow float64
	for j := idx; j > idx-period && j >= 0; j-- {
		h, _ := i.bars[j].High.Float64()
		l, _ := i.bars[j].Low.Float64()
		if j == idx {
			highestHigh = h
			lowestLow = l
		} else {
			if h > highestHigh {
				highestHigh = h
			}
			if l < lowestLow {
				lowestLow = l
			}
		}
	}
	currentClose, _ := i.bars[idx].Close.Float64()
	if highestHigh == lowestLow {
		return -50
	}
	return (highestHigh - currentClose) / (highestHigh - lowestLow) * -100
}

func (i *btIndicators) ICustom(name string, params []float64, buffer, shift int) float64 {
	return 0
}
