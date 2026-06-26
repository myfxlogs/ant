package sdk

import "github.com/shopspring/decimal"

// BarSeries provides MQL-style inverse-indexed access to OHLCV data.
//
// bar[0] is the current (latest) bar.
// bar[1] is the previous bar.
// bar[N] is N bars ago.
//
// This matches MQL4/MQL5 semantics exactly.
type BarSeries interface {
	// Open returns the open price at shift bars back.
	Open(shift int) decimal.Decimal

	// High returns the high price at shift bars back.
	High(shift int) decimal.Decimal

	// Low returns the low price at shift bars back.
	Low(shift int) decimal.Decimal

	// Close returns the close price at shift bars back.
	Close(shift int) decimal.Decimal

	// Volume returns the tick volume at shift bars back.
	Volume(shift int) int64

	// Time returns the bar open timestamp (unix_ms) at shift bars back.
	Time(shift int) int64

	// Len returns the total number of bars available.
	Len() int

	// Slice returns the most recent n bars as a new BarSeries.
	// Useful for passing to indicator functions that need full series.
	Slice(n int) BarSeries

	// Timeframe returns the period string of this series ("M5", "H1", etc.).
	Timeframe() string

	// Symbol returns the symbol this series belongs to.
	Symbol() string
}

// Bar represents a single complete OHLCV bar.
type Bar struct {
	Open      decimal.Decimal
	High      decimal.Decimal
	Low       decimal.Decimal
	Close     decimal.Decimal
	Volume    int64
	Timestamp int64 // unix_ms
}

// BarsToSlice converts a []Bar to a BarSeries.
// The slice is ordered chronologically (oldest first).
// BarSeries inverse-indexes it: [0] = last element = current bar.
func BarsToSlice(bars []Bar) BarSeries {
	return &barSlice{bars: bars}
}

type barSlice struct {
	bars []Bar
}

func (b *barSlice) Open(shift int) decimal.Decimal   { return b.at(shift).Open }
func (b *barSlice) High(shift int) decimal.Decimal   { return b.at(shift).High }
func (b *barSlice) Low(shift int) decimal.Decimal    { return b.at(shift).Low }
func (b *barSlice) Close(shift int) decimal.Decimal  { return b.at(shift).Close }
func (b *barSlice) Volume(shift int) int64           { return b.at(shift).Volume }
func (b *barSlice) Time(shift int) int64             { return b.at(shift).Timestamp }
func (b *barSlice) Len() int                         { return len(b.bars) }
func (b *barSlice) Slice(n int) BarSeries            { return &barSlice{bars: b.tail(n)} }
func (b *barSlice) Timeframe() string                { return "" }
func (b *barSlice) Symbol() string                   { return "" }

func (b *barSlice) at(shift int) Bar {
	idx := len(b.bars) - 1 - shift
	if idx < 0 || idx >= len(b.bars) {
		return Bar{}
	}
	return b.bars[idx]
}

func (b *barSlice) tail(n int) []Bar {
	if n >= len(b.bars) {
		return b.bars
	}
	return b.bars[len(b.bars)-n:]
}
