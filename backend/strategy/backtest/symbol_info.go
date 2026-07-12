package backtest

import (
	"strings"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// DeriveSymbolInfoFromBars infers SymbolDigits, SymbolPoint, and Spread
// from the first bar's close price when broker SymbolInfo is unavailable.
// This is shared between executeVMBacktest and AgentGateway.
func DeriveSymbolInfoFromBars(cfg *Config, bars []sdk.Bar) {
	if len(bars) == 0 {
		return
	}
	closeStr := bars[0].Close.String()
	dotIdx := strings.Index(closeStr, ".")
	digits := int32(0)
	if dotIdx >= 0 {
		digits = int32(len(closeStr) - dotIdx - 1)
	}
	if digits > 8 {
		digits = 8
	}
	cfg.SymbolDigits = digits
	point := decimal.NewFromInt(1)
	for i := int32(0); i < digits; i++ {
		point = point.Div(decimal.NewFromInt(10))
	}
	cfg.SymbolPoint = point
	cfg.Spread = point.Mul(decimal.NewFromInt(10))
}
