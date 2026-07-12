package backtest

import (
	"alphaforge/strategy/sdk"
)

// BarsForSymbol returns bar series for a specific symbol and timeframe in multi-symbol strategies.
// Only bars up to the current primary bar's timestamp are visible (no future data leakage).
// timeframe="" means the primary timeframe.
func (c *backtestContext) BarsForSymbol(symbol, timeframe string) sdk.BarSeries {
	if symbol == c.symbol || symbol == "" {
		if timeframe == "" || timeframe == c.tf {
			return c.Bars()
		}
		return c.BarsTF(timeframe)
	}
	if c.extraBars == nil {
		return sdk.BarsToSlice(nil)
	}
	bars, ok := c.extraBars[symbol]
	if !ok {
		return sdk.BarsToSlice(nil)
	}
	idx := c.extraBarIndex[symbol]
	if idx < 0 || idx >= len(bars) {
		return sdk.BarsToSlice(nil)
	}
	visible := bars[:idx+1]
	if timeframe != "" && timeframe != c.tf {
		visible = aggregateBars(visible, timeframe)
	}
	return sdk.BarsToSlice(visible)
}
