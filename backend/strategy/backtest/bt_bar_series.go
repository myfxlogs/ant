package backtest

import (
	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// btBarSeries wraps an sdk.BarSeries and returns Volume=1 for shift=0,
// matching MT4 "Open prices only" bar-based backtest semantics.
//
// In MT4's "Open prices only" mode, each bar is opened with Volume=1
// (representing the first tick of a new bar), and the EA's OnTick is
// called once per bar. The common MQL4 new-bar guard `if(Volume[0]>1) return;`
// relies on this: it passes on the first tick (Volume=1) and skips on
// subsequent ticks (Volume>1).
//
// Our bar-based backtest engine also calls OnTick once per bar, so it is
// semantically equivalent to "Open prices only". Without this override,
// Volume[0] returns the bar's full tick volume (hundreds to thousands),
// causing `Volume[0]>1` to always be true → the EA never trades.
//
// Historical bars (shift>0) keep their actual completed volume.
type btBarSeries struct {
	inner sdk.BarSeries
}

func wrapBTBarSeries(inner sdk.BarSeries) sdk.BarSeries {
	if inner == nil {
		return nil
	}
	return &btBarSeries{inner: inner}
}

func (b *btBarSeries) Open(shift int) decimal.Decimal { return b.inner.Open(shift) }
func (b *btBarSeries) High(shift int) decimal.Decimal { return b.inner.High(shift) }
func (b *btBarSeries) Low(shift int) decimal.Decimal  { return b.inner.Low(shift) }
func (b *btBarSeries) Close(shift int) decimal.Decimal { return b.inner.Close(shift) }
func (b *btBarSeries) Time(shift int) int64            { return b.inner.Time(shift) }
func (b *btBarSeries) Len() int                         { return b.inner.Len() }
func (b *btBarSeries) Timeframe() string                { return b.inner.Timeframe() }
func (b *btBarSeries) Symbol() string                   { return b.inner.Symbol() }

func (b *btBarSeries) Volume(shift int) int64 {
	if shift == 0 {
		// MT4 "Open prices only": current bar just opened, Volume=1.
		return 1
	}
	return b.inner.Volume(shift)
}

// Slice returns the inner Slice wrapped, so the Volume(0)=1 semantics
// propagate to sliced views (e.g. indicator seeding windows).
func (b *btBarSeries) Slice(n int) sdk.BarSeries {
	return wrapBTBarSeries(b.inner.Slice(n))
}
