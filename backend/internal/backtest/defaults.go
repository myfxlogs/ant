// Package backtest provides shared defaults and validation for backtest run creation,
// used by both the strategy API handlers and the marketplace backtest flow.
package backtest

import (
	"anttrader/internal/repository"
)

// Defaults for backtest parameters.
const (
	DefaultCommission     = 0.001
	DefaultLeverage       = 1
	DefaultInitialCapital = 10000
	DefaultMode           = "KLINE_RANGE"
	DefaultTradeDirection = "both"
)

// ApplyDefaults fills in safe defaults for a BacktestRun before insertion.
func ApplyDefaults(run *repository.BacktestRun) {
	if run.Mode == "" {
		run.Mode = DefaultMode
	}
	if run.Commission == nil || *run.Commission <= 0 {
		v := DefaultCommission
		run.Commission = &v
	}
	if run.Commission != nil && (*run.Commission < 0 || *run.Commission > 10) {
		v := DefaultCommission
		run.Commission = &v
	}
	if run.Slippage != nil && (*run.Slippage < 0 || *run.Slippage > 10) {
		v := 0.0
		run.Slippage = &v
	}
	if run.Leverage == nil || *run.Leverage <= 0 {
		v := float64(DefaultLeverage)
		run.Leverage = &v
	}
	if run.Leverage != nil && *run.Leverage < 1 {
		v := float64(DefaultLeverage)
		run.Leverage = &v
	}
	if run.TradeDirection == nil || *run.TradeDirection == "" {
		v := DefaultTradeDirection
		run.TradeDirection = &v
	}
	if run.StrictMode == nil {
		t := true
		run.StrictMode = &t
	}
	if run.ExtraSymbols == nil {
		run.ExtraSymbols = []string{}
	}
	if run.InitialCapital == nil || *run.InitialCapital <= 0 {
		v := float64(DefaultInitialCapital)
		run.InitialCapital = &v
	}
}
