// Package backtest provides shared defaults and validation for backtest run creation,
// used by both the strategy API handlers and the marketplace backtest flow.
package backtest

import (
	"github.com/shopspring/decimal"

	"alphaforge/internal/repository"
)

// Defaults for backtest parameters.
var (
	DefaultCommission     = decimal.NewFromFloat(0.001)
	DefaultLeverage       = decimal.NewFromInt(1)
	DefaultInitialCapital = decimal.NewFromInt(10000)
)

const (
	DefaultMode           = "KLINE_RANGE"
	DefaultTradeDirection = "both"
)

// ApplyDefaults fills in safe defaults for a BacktestRun before insertion.
func ApplyDefaults(run *repository.BacktestRun) {
	if run.Mode == "" {
		run.Mode = DefaultMode
	}
	if run.Commission == nil || run.Commission.LessThanOrEqual(decimal.Zero) {
		run.Commission = &DefaultCommission
	}
	if run.Commission != nil && (run.Commission.LessThan(decimal.Zero) || run.Commission.GreaterThan(decimal.NewFromInt(10))) {
		run.Commission = &DefaultCommission
	}
	if run.Slippage == nil {
		zero := decimal.Zero
		run.Slippage = &zero
	}
	if run.Slippage != nil && (run.Slippage.LessThan(decimal.Zero) || run.Slippage.GreaterThan(decimal.NewFromInt(10))) {
		zero := decimal.Zero
		run.Slippage = &zero
	}
	if run.Leverage == nil || run.Leverage.LessThanOrEqual(decimal.Zero) {
		run.Leverage = &DefaultLeverage
	}
	if run.Leverage != nil && run.Leverage.LessThan(decimal.NewFromInt(1)) {
		run.Leverage = &DefaultLeverage
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
	if run.InitialCapital == nil || run.InitialCapital.LessThanOrEqual(decimal.Zero) {
		run.InitialCapital = &DefaultInitialCapital
	}
}
