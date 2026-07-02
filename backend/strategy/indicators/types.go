package indicators

import (
	"github.com/shopspring/decimal"
)

// BarSource is the minimal interface needed for indicator calculations.
// Index 0 = most recent bar, increasing index = older bars.
type BarSource interface {
	Open(i int) decimal.Decimal
	High(i int) decimal.Decimal
	Low(i int) decimal.Decimal
	Close(i int) decimal.Decimal
	Volume(i int) int64
	Len() int
}

// ── Helper functions ────────────────────────────────────────────────

// selectPrice extracts the price at index i based on appliedPrice.
// MT4/MT5 ENUM_APPLIED_PRICE: 1=close, 2=open, 3=high, 4=low,
// 5=median (H+L)/2, 6=typical (H+L+C)/3, 7=weighted (H+L+2C)/4.
func selectPrice(src BarSource, appliedPrice, i int) float64 {
	switch appliedPrice {
	case 2:
		o, _ := src.Open(i).Float64()
		return o
	case 3:
		h, _ := src.High(i).Float64()
		return h
	case 4:
		l, _ := src.Low(i).Float64()
		return l
	case 5:
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		return (h + l) / 2
	case 6:
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		c, _ := src.Close(i).Float64()
		return (h + l + c) / 3
	case 7:
		h, _ := src.High(i).Float64()
		l, _ := src.Low(i).Float64()
		c, _ := src.Close(i).Float64()
		return (h + l + 2*c) / 4
	default: // 1 = PRICE_CLOSE
		c, _ := src.Close(i).Float64()
		return c
	}
}

// appliedPriceBarSource wraps a BarSource, overriding Close() to return
// the selected applied price. This lets all Close()-based indicator
// functions (ema, sma, rsiWilder, etc.) work with any appliedPrice.
type appliedPriceBarSource struct {
	inner        BarSource
	appliedPrice int
}

func (s *appliedPriceBarSource) Open(i int) decimal.Decimal { return s.inner.Open(i) }
func (s *appliedPriceBarSource) High(i int) decimal.Decimal { return s.inner.High(i) }
func (s *appliedPriceBarSource) Low(i int) decimal.Decimal  { return s.inner.Low(i) }
func (s *appliedPriceBarSource) Volume(i int) int64         { return s.inner.Volume(i) }
func (s *appliedPriceBarSource) Len() int                   { return s.inner.Len() }
func (s *appliedPriceBarSource) Close(i int) decimal.Decimal {
	return decimal.NewFromFloat(selectPrice(s.inner, s.appliedPrice, i))
}

// withAppliedPrice returns src unchanged for PRICE_CLOSE (default),
// or wraps it in appliedPriceBarSource for other applied prices.
func withAppliedPrice(src BarSource, appliedPrice int) BarSource {
	if appliedPrice <= 1 {
		return src
	}
	return &appliedPriceBarSource{inner: src, appliedPrice: appliedPrice}
}

// sliceBarSource adapts a []float64 of close prices to BarSource.
// Used internally for indicators that need to smooth a derived series (e.g. Force).
type sliceBarSource struct {
	closes []float64
}

func (s *sliceBarSource) Open(i int) decimal.Decimal { return s.Close(i) }
func (s *sliceBarSource) High(i int) decimal.Decimal { return s.Close(i) }
func (s *sliceBarSource) Low(i int) decimal.Decimal  { return s.Close(i) }
func (s *sliceBarSource) Close(i int) decimal.Decimal {
	if i < 0 || i >= len(s.closes) {
		return decimal.Zero
	}
	return decimal.NewFromFloat(s.closes[i])
}
func (s *sliceBarSource) Volume(i int) int64 { return 0 }
func (s *sliceBarSource) Len() int           { return len(s.closes) }

func highestHigh(src BarSource, period, shift int) float64 {
	if src.Len() < period+shift {
		return 0
	}
	hh, _ := src.High(shift).Float64()
	for i := shift + 1; i < shift+period; i++ {
		h, _ := src.High(i).Float64()
		if h > hh {
			hh = h
		}
	}
	return hh
}

func lowestLow(src BarSource, period, shift int) float64 {
	if src.Len() < period+shift {
		return 0
	}
	ll, _ := src.Low(shift).Float64()
	for i := shift + 1; i < shift+period; i++ {
		l, _ := src.Low(i).Float64()
		if l < ll {
			ll = l
		}
	}
	return ll
}
