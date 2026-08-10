package backtest

import (
	"context"
	"strconv"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

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
