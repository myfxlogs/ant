// Package risksvc provides the PositionSizer interface (M10-BASE-C2).
//
// PositionSizer converts a "should trade" signal into a "how much to trade" lot size.
// Unlike HardLimit (binary deny), Sizer adjusts position size progressively
// (vol-target, Kelly-fraction, risk-parity) — it never returns zero unless
// the risk budget is exhausted.

package risksvc

import (
	"context"

	"github.com/shopspring/decimal"
)

// SizerRequest contains all inputs needed to compute a position size.
type SizerRequest struct {
	Symbol       string
	Price        decimal.Decimal
	ATR          decimal.Decimal // Average True Range over lookback period
	AnnualVol    float64         // annualized volatility (e.g. 0.20 = 20%)
	ContractSize decimal.Decimal // contract multiplier (e.g. 100000 for standard forex lot)
	HoldingDays  float64         // expected holding period in days

	// Account context
	AccountID  string
	Balance    decimal.Decimal
	Equity     decimal.Decimal
	FreeMargin decimal.Decimal
}

// SizerResult is the output of a PositionSizer.
type SizerResult struct {
	Lots     decimal.Decimal // computed position size in lots
	RiskUsed float64         // fraction of risk budget consumed
	Method   string          // sizer name
}

// PositionSizer computes the optimal position size for a trade.
type PositionSizer interface {
	// Name returns the sizer identifier.
	Name() string

	// Size computes position lots from the request.
	Size(ctx context.Context, req *SizerRequest) (*SizerResult, error)
}
