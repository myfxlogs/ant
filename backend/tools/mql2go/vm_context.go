package mql2go

import (
	"context"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// noopContext is the zero-value sdk.Context injected by NewVM so that
// vm.ctx is never nil (QS-2.3). Every method returns the same zero value
// the former `vm.ctx == nil` guard branches produced, so guard removal is
// observationally identical for all equivalent sites.
type noopContext struct{}

// emptyBars is a shared immutable empty series. barSlice is read-only
// (Slice allocates a new one), so a single instance is safe to share.
var emptyBars = sdk.BarsToSlice(nil)

// ── Parameters ───────────────────────────────────────────────────────

func (noopContext) Param(_ string, defaultVal interface{}) interface{} {
	return defaultVal
}
func (noopContext) ParamDecimal(_ string, defaultVal decimal.Decimal) decimal.Decimal {
	return defaultVal
}
func (noopContext) ParamInt(_ string, defaultVal int) int    { return defaultVal }
func (noopContext) ParamString(_, defaultVal string) string  { return defaultVal }
func (noopContext) ParamBool(_ string, defaultVal bool) bool { return defaultVal }

// ── Market data ──────────────────────────────────────────────────────

func (noopContext) Bars() sdk.BarSeries                     { return emptyBars }
func (noopContext) BarsTF(string) sdk.BarSeries             { return emptyBars }
func (noopContext) BarsForSymbol(_, _ string) sdk.BarSeries { return emptyBars }
func (noopContext) Symbol() string                          { return "" }
func (noopContext) Timeframe() string                       { return "" }
func (noopContext) Point() decimal.Decimal                  { return decimal.Zero }
func (noopContext) Pip() decimal.Decimal                    { return decimal.Zero }
func (noopContext) Digits() int32                           { return 0 }
func (noopContext) Ask() decimal.Decimal                    { return decimal.Zero }
func (noopContext) Bid() decimal.Decimal                    { return decimal.Zero }
func (noopContext) Spread() decimal.Decimal                 { return decimal.Zero }

// ── Account ──────────────────────────────────────────────────────────

func (noopContext) Account() sdk.AccountInfo { return sdk.AccountInfo{} }
func (noopContext) Mode() sdk.AccountMode    { return "" }

// ── Services ─────────────────────────────────────────────────────────

// Broker returns nil — `vm.ctx.Broker() == nil` checks remain meaningful.
func (noopContext) Broker() sdk.Broker           { return nil }
func (noopContext) Indicators() sdk.IndicatorSet { return noopIndicatorSet{} }

// ── Lifecycle ────────────────────────────────────────────────────────

func (noopContext) SetTimer(int)               {}
func (noopContext) KillTimer()                 {}
func (noopContext) Log(string)                 {}
func (noopContext) ServerTime() int64          { return 0 }
func (noopContext) GoContext() context.Context { return context.Background() }

// noopIndicatorSet returns decimal.Zero for every indicator — the same
// value the former `vm.ctx == nil` early-returns produced.
type noopIndicatorSet struct{}

func (noopIndicatorSet) MA(_, _ int, _ string, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) EMA(_, _ int) decimal.Decimal { return decimal.Zero }
func (noopIndicatorSet) RSI(_, _, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) MACD(_, _, _, _, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) MACDSignal(_, _, _, _, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) ATR(_, _ int) decimal.Decimal { return decimal.Zero }
func (noopIndicatorSet) Bollinger(_ int, _ decimal.Decimal, _, _ int) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero, decimal.Zero
}
func (noopIndicatorSet) Stochastic(_, _, _, _ int) (decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero
}
func (noopIndicatorSet) CCI(_, _, _ int) decimal.Decimal { return decimal.Zero }
func (noopIndicatorSet) ADX(_, _ int) decimal.Decimal    { return decimal.Zero }
func (noopIndicatorSet) MFI(_, _ int) decimal.Decimal    { return decimal.Zero }
func (noopIndicatorSet) OBV(_, _ int) decimal.Decimal    { return decimal.Zero }
func (noopIndicatorSet) SAR(_, _ decimal.Decimal, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) StdDev(_, _ int, _ string, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) WPR(_, _ int) decimal.Decimal { return decimal.Zero }
func (noopIndicatorSet) Momentum(_, _, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) ICustom(_ string, _ []decimal.Decimal, _, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) Alligator(_, _, _, _, _, _ int, _ string, _, _ int) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero, decimal.Zero
}
func (noopIndicatorSet) Ichimoku(_, _, _, _ int) (decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
}
func (noopIndicatorSet) Envelopes(_ int, _ decimal.Decimal, _ string, _, _ int) (decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero
}
func (noopIndicatorSet) DeMarker(_, _ int) decimal.Decimal { return decimal.Zero }
func (noopIndicatorSet) OsMA(_, _, _, _, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) RVI(_, _ int) decimal.Decimal { return decimal.Zero }
func (noopIndicatorSet) Force(_ int, _ string, _, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) Fractals(_ int) (decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero
}
func (noopIndicatorSet) Gator(_, _, _, _, _, _ int, _ string, _, _ int) (decimal.Decimal, decimal.Decimal) {
	return decimal.Zero, decimal.Zero
}
func (noopIndicatorSet) AC(_ int) decimal.Decimal { return decimal.Zero }
func (noopIndicatorSet) AD(_ int) decimal.Decimal { return decimal.Zero }
func (noopIndicatorSet) AO(_ int) decimal.Decimal { return decimal.Zero }
func (noopIndicatorSet) BearsPower(_, _, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) BullsPower(_, _, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) BWMFI(_ int) decimal.Decimal { return decimal.Zero }
func (noopIndicatorSet) AMA(_, _, _, _, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) DEMA(_, _, _ int) decimal.Decimal  { return decimal.Zero }
func (noopIndicatorSet) TEMA(_, _, _ int) decimal.Decimal  { return decimal.Zero }
func (noopIndicatorSet) FrAMA(_, _, _ int) decimal.Decimal { return decimal.Zero }
func (noopIndicatorSet) VIDyA(_, _, _, _, _, _ int) decimal.Decimal {
	return decimal.Zero
}
func (noopIndicatorSet) TriX(_, _, _ int) decimal.Decimal    { return decimal.Zero }
func (noopIndicatorSet) ADXWilder(_, _ int) decimal.Decimal  { return decimal.Zero }
func (noopIndicatorSet) Chaikin(_, _, _ int) decimal.Decimal { return decimal.Zero }
func (noopIndicatorSet) Volumes(_ int) decimal.Decimal       { return decimal.Zero }
