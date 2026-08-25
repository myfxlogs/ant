package runner

import (
	"context"
	"log"
	"sync"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// contextImpl implements sdk.Context.
type contextImpl struct {
	runner *Runner

	mu     sync.RWMutex
	params map[string]string
	bars   sdk.BarSeries

	symbol    string
	timeframe string
	timerSet  bool

	// goCtx is the Go context.Context for the current event dispatch.
	// Set by Runner before calling strategy methods; read by VM via GoContext().
	goCtx context.Context

	// Live state from parent process (harness mode — no RPC).
	liveBalance        string
	liveEquity         string
	liveMargin         string
	liveFreeMargin     string
	livePositions      []sdk.Position
	livePendingOrders  []sdk.PendingOrder // LIVE-MQL-ORDER-CONTEXT-1
	liveLogin          int64              // VM-TRADE-CONTEXT-3: account login from parent process
	liveCompany        string             // VM-TRADE-CONTEXT-3: broker company from parent process
	liveIsDemo         bool               // VM-API-TRUTH-3: account is demo/contest
	liveIsConnected    bool               // VM-API-TRUTH-3: host is connected to broker
	liveIsTradeAllowed bool               // VM-API-TRUTH-3: trading is permitted

	// Symbol info from parent process (harness mode — no RPC).
	livePoint        string
	liveDigits       int32
	liveContractSize string
	liveStopsLevel   string

	// Tick-level prices (harness mode — set on TICK requests).
	tickBid decimal.Decimal
	tickAsk decimal.Decimal

	// Multi-symbol bar windows for BarsForSymbol (live/paper mode).
	extraBars map[string][]sdk.Bar
}

func (c *contextImpl) setBars(b sdk.BarSeries) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bars = b
}

func (c *contextImpl) setExtraBars(extra map[string][]sdk.Bar) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.extraBars = extra
}

func (c *contextImpl) setTick(bid, ask decimal.Decimal) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tickBid = bid
	c.tickAsk = ask
}

// ── Parameters ────────────────────────────────────────────────────

func (c *contextImpl) Param(name string, defaultVal interface{}) interface{} {
	if v, ok := c.params[name]; ok {
		return v
	}
	return defaultVal
}

func (c *contextImpl) ParamDecimal(name string, defaultVal decimal.Decimal) decimal.Decimal {
	if v, ok := c.params[name]; ok {
		d, err := decimal.NewFromString(v)
		if err == nil {
			return d
		}
	}
	return defaultVal
}

func (c *contextImpl) ParamInt(name string, defaultVal int) int {
	d := c.ParamDecimal(name, decimal.NewFromInt(int64(defaultVal)))
	return int(d.IntPart())
}

func (c *contextImpl) ParamString(name, defaultVal string) string {
	if v, ok := c.params[name]; ok {
		return v
	}
	return defaultVal
}

func (c *contextImpl) ParamBool(name string, defaultVal bool) bool {
	if v, ok := c.params[name]; ok {
		return v == "true" || v == "1"
	}
	return defaultVal
}

// ── Market data ────────────────────────────────────────────────────

func (c *contextImpl) Bars() sdk.BarSeries {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.bars
}

func (c *contextImpl) BarsTF(timeframe string) sdk.BarSeries {
	// In the harness context only the primary timeframe is available.
	// Multi-timeframe requires a multi-symbol bar source (Phase B2).
	return c.Bars()
}

func (c *contextImpl) BarsForSymbol(symbol, timeframe string) sdk.BarSeries {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if symbol == c.symbol || symbol == "" {
		if timeframe == "" || timeframe == c.timeframe {
			return c.bars
		}
		return c.BarsTF(timeframe)
	}
	if c.extraBars == nil {
		return sdk.BarsToSlice(nil)
	}
	bars, ok := c.extraBars[symbol]
	if !ok || len(bars) == 0 {
		return sdk.BarsToSlice(nil)
	}
	return sdk.BarsToSlice(bars)
}

func (c *contextImpl) Symbol() string    { return c.symbol }
func (c *contextImpl) Timeframe() string { return c.timeframe }

func (c *contextImpl) Point() decimal.Decimal {
	c.mu.RLock()
	if c.livePoint != "" {
		if d, err := decimal.NewFromString(c.livePoint); err == nil {
			c.mu.RUnlock()
			return d
		}
	}
	c.mu.RUnlock()
	info, _ := c.runner.broker.SymbolInfo(c.symbol)
	return info.Point
}

func (c *contextImpl) Pip() decimal.Decimal {
	return c.Point().Mul(decimal.NewFromInt(10))
}

func (c *contextImpl) Digits() int32 {
	c.mu.RLock()
	if c.liveDigits != 0 {
		d := c.liveDigits
		c.mu.RUnlock()
		return d
	}
	c.mu.RUnlock()
	info, _ := c.runner.broker.SymbolInfo(c.symbol)
	return info.Digits
}

func (c *contextImpl) Ask() decimal.Decimal {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Tick mode: use the actual Ask from the quote stream.
	if !c.tickAsk.IsZero() {
		return c.tickAsk
	}
	if c.bars != nil && c.bars.Len() > 0 {
		return c.bars.Close(0)
	}
	return decimal.Zero
}

func (c *contextImpl) Bid() decimal.Decimal {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.tickBid.IsZero() {
		return c.tickBid
	}
	if c.bars != nil && c.bars.Len() > 0 {
		return c.bars.Close(0)
	}
	return decimal.Zero
}

func (c *contextImpl) Spread() decimal.Decimal {
	return c.Ask().Sub(c.Bid())
}

// ── Account ────────────────────────────────────────────────────────

func (c *contextImpl) Account() sdk.AccountInfo {
	return c.runner.broker.Account()
}

func (c *contextImpl) Mode() sdk.AccountMode {
	return c.runner.broker.Account().Mode
}

// ── Services ───────────────────────────────────────────────────────

func (c *contextImpl) Broker() sdk.Broker           { return c.runner.broker }
func (c *contextImpl) Indicators() sdk.IndicatorSet { return c.runner.ind }

// ── Lifecycle ──────────────────────────────────────────────────────

func (c *contextImpl) SetTimer(seconds int) { c.timerSet = true }
func (c *contextImpl) KillTimer()           { c.timerSet = false }

func (c *contextImpl) Log(msg string) {
	log.Printf("[strategy:%s] %s", c.symbol, msg)
}

func (c *contextImpl) ServerTime() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.bars != nil && c.bars.Len() > 0 {
		return c.bars.Time(0)
	}
	return 0
}

func (c *contextImpl) GoContext() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.goCtx != nil {
		return c.goCtx
	}
	return context.Background()
}

func (c *contextImpl) setGoContext(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.goCtx = ctx
}
