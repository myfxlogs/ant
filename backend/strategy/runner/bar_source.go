package runner

import (
	"alphaforge/strategy/indicators"
	"alphaforge/strategy/sdk"

	"github.com/shopspring/decimal"
)

// runnerBarSource adapts sdk.BarSeries to indicators.BarSource.
// It also implements indicators.RevisionedBarSource so the SeriesCache can
// detect rolling-window content changes (live mode). The revision is advanced
// by Runner.OnBar only — tick/trade/timer events do not change the bar window.
type runnerBarSource struct {
	bars   sdk.BarSeries
	runner *Runner
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

// Revision returns the Runner's bar revision counter, which advances once per
// OnBar call. Returns 0 when the source has no runner reference (stateless
// barSource() path — never passed to SeriesCache, so revision is irrelevant).
func (b *runnerBarSource) Revision() uint64 {
	if b.runner == nil {
		return 0
	}
	return b.runner.barRevision()
}

func (is *indicatorSet) barSource() indicators.BarSource {
	return &runnerBarSource{bars: is.bars(), runner: is.runner}
}
