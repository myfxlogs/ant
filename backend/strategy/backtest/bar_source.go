package backtest

import (
	"github.com/shopspring/decimal"

	"anttrader/strategy/indicators"
	"anttrader/strategy/sdk"
)

// btBarSource adapts []sdk.Bar to indicators.BarSource.
// Index 0 = most recent bar (matches btBarSeries convention).
type btBarSource struct {
	bars []sdk.Bar
}

func (b *btBarSource) Len() int { return len(b.bars) }

func (b *btBarSource) Open(i int) decimal.Decimal {
	idx := len(b.bars) - 1 - i
	if idx < 0 || idx >= len(b.bars) {
		return decimal.Zero
	}
	return b.bars[idx].Open
}
func (b *btBarSource) High(i int) decimal.Decimal {
	idx := len(b.bars) - 1 - i
	if idx < 0 || idx >= len(b.bars) {
		return decimal.Zero
	}
	return b.bars[idx].High
}
func (b *btBarSource) Low(i int) decimal.Decimal {
	idx := len(b.bars) - 1 - i
	if idx < 0 || idx >= len(b.bars) {
		return decimal.Zero
	}
	return b.bars[idx].Low
}
func (b *btBarSource) Close(i int) decimal.Decimal {
	idx := len(b.bars) - 1 - i
	if idx < 0 || idx >= len(b.bars) {
		return decimal.Zero
	}
	return b.bars[idx].Close
}
func (b *btBarSource) Volume(i int) int64 {
	idx := len(b.bars) - 1 - i
	if idx < 0 || idx >= len(b.bars) {
		return 0
	}
	return b.bars[idx].Volume
}

func (i *btIndicators) barSource() indicators.BarSource {
	return &btBarSource{bars: i.visibleBars()}
}
