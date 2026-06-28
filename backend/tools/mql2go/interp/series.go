package interp

import (
	"anttrader/strategy/sdk"
)

// SeriesAccessor provides MQL-style inverse-indexed access to OHLCV data.
// Close[0] = current bar, Close[1] = previous bar, etc.
type SeriesAccessor struct {
	bars sdk.BarSeries
}

func (s *SeriesAccessor) Close(shift int) Value {
	if s.bars == nil || shift < 0 || shift >= s.bars.Len() {
		return NoneVal()
	}
	return DecimalVal(s.bars.Close(shift))
}

func (s *SeriesAccessor) Open(shift int) Value {
	if s.bars == nil || shift < 0 || shift >= s.bars.Len() {
		return NoneVal()
	}
	return DecimalVal(s.bars.Open(shift))
}

func (s *SeriesAccessor) High(shift int) Value {
	if s.bars == nil || shift < 0 || shift >= s.bars.Len() {
		return NoneVal()
	}
	return DecimalVal(s.bars.High(shift))
}

func (s *SeriesAccessor) Low(shift int) Value {
	if s.bars == nil || shift < 0 || shift >= s.bars.Len() {
		return NoneVal()
	}
	return DecimalVal(s.bars.Low(shift))
}

func (s *SeriesAccessor) Volume(shift int) Value {
	if s.bars == nil || shift < 0 || shift >= s.bars.Len() {
		return NoneVal()
	}
	return IntVal(int32(s.bars.Volume(shift)))
}

func (s *SeriesAccessor) Time(shift int) Value {
	if s.bars == nil || shift < 0 || shift >= s.bars.Len() {
		return NoneVal()
	}
	return Value{Kind: ValDatetime, Datetime: s.bars.Time(shift)}
}
