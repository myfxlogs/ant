package runner

import (
	"anttrader/strategy/indicators"
	"anttrader/strategy/sdk"
	"github.com/shopspring/decimal"
)

// runnerBarSource adapts sdk.BarSeries to indicators.BarSource.
type runnerBarSource struct {
	bars sdk.BarSeries
}

func (b *runnerBarSource) Len() int {
	if b.bars == nil {
		return 0
	}
	return b.bars.Len()
}
func (b *runnerBarSource) Open(i int) decimal.Decimal {
	if b.bars == nil {
		return decimal.Zero
	}
	return b.bars.Open(i)
}
func (b *runnerBarSource) High(i int) decimal.Decimal {
	if b.bars == nil {
		return decimal.Zero
	}
	return b.bars.High(i)
}
func (b *runnerBarSource) Low(i int) decimal.Decimal {
	if b.bars == nil {
		return decimal.Zero
	}
	return b.bars.Low(i)
}
func (b *runnerBarSource) Close(i int) decimal.Decimal {
	if b.bars == nil {
		return decimal.Zero
	}
	return b.bars.Close(i)
}
func (b *runnerBarSource) Volume(i int) int64 {
	if b.bars == nil {
		return 0
	}
	return b.bars.Volume(i)
}

func (is *indicatorSet) barSource() indicators.BarSource {
	return &runnerBarSource{bars: is.bars()}
}
