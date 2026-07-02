package indicators

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// ── Oscillator query methods ───────────────────────────────────────

func (c *SeriesCache) RSI(period, shift int) float64 {
	s, ok := c.rsi[period]
	if !ok {
		s = newRSISeries(period)
		c.rsi[period] = s
		c.rebuildRSI(s, period)
	}
	return s.at(shift)
}

// RSIWithPrice returns RSI with appliedPrice. Uses cache for PRICE_CLOSE,
// delegates to stateless for other applied prices.
func (c *SeriesCache) RSIWithPrice(period, shift, appliedPrice int) float64 {
	if appliedPrice <= 1 {
		return c.RSI(period, shift)
	}
	v, _ := RSI(c.src, period, shift, appliedPrice).Float64()
	return v
}

func (c *SeriesCache) ATR(period, shift int) float64 {
	s, ok := c.atr[period]
	if !ok {
		s = newATRSeries(period)
		c.atr[period] = s
		c.rebuildATR(s, period)
	}
	return s.at(shift)
}

func (c *SeriesCache) ADX(period, shift int) float64 {
	s, ok := c.adx[period]
	if !ok {
		s = newADXSeries(period)
		c.adx[period] = s
		c.rebuildADX(s, period)
	}
	return s.at(shift)
}

func (c *SeriesCache) MACDLine(fast, slow, signalPeriod, shift int) float64 {
	key := fmt.Sprintf("%d,%d,%d", fast, slow, signalPeriod)
	s, ok := c.macd[key]
	if !ok {
		s = newMACDSeries(fast, slow, signalPeriod)
		c.macd[key] = s
		c.rebuildMACD(s, fast, slow)
	}
	return s.macdAt(shift)
}

// MACDLineWithPrice returns MACD line with appliedPrice.
func (c *SeriesCache) MACDLineWithPrice(fast, slow, signalPeriod, appliedPrice, shift int) float64 {
	if appliedPrice <= 1 {
		return c.MACDLine(fast, slow, signalPeriod, shift)
	}
	v, _ := MACD(c.src, fast, slow, shift, appliedPrice).Float64()
	return v
}

func (c *SeriesCache) MACDSignal(fast, slow, signalPeriod, shift int) float64 {
	key := fmt.Sprintf("%d,%d,%d", fast, slow, signalPeriod)
	s, ok := c.macd[key]
	if !ok {
		s = newMACDSeries(fast, slow, signalPeriod)
		c.macd[key] = s
		c.rebuildMACD(s, fast, slow)
	}
	return s.signalAt(shift)
}

// MACDSignalWithPrice returns MACD signal line with appliedPrice.
func (c *SeriesCache) MACDSignalWithPrice(fast, slow, signalPeriod, appliedPrice, shift int) float64 {
	if appliedPrice <= 1 {
		return c.MACDSignal(fast, slow, signalPeriod, shift)
	}
	v, _ := MACDSignal(c.src, fast, slow, signalPeriod, shift, appliedPrice).Float64()
	return v
}

// OsMA = MACD line - Signal line
func (c *SeriesCache) OsMA(fast, slow, signalPeriod, shift int) float64 {
	return c.MACDLine(fast, slow, signalPeriod, shift) - c.MACDSignal(fast, slow, signalPeriod, shift)
}

// OsMAWithPrice returns OsMA with appliedPrice.
func (c *SeriesCache) OsMAWithPrice(fast, slow, signalPeriod, appliedPrice, shift int) float64 {
	if appliedPrice <= 1 {
		return c.OsMA(fast, slow, signalPeriod, shift)
	}
	v, _ := OsMA(c.src, fast, slow, signalPeriod, appliedPrice, shift).Float64()
	return v
}

// BearsPower = Low(shift) - EMA(period, shift)
func (c *SeriesCache) BearsPower(period, shift int) float64 {
	if c.src.Len() < period+shift {
		return 0
	}
	low, _ := c.src.Low(shift).Float64()
	return low - c.EMA(period, shift)
}

// BullsPower = High(shift) - EMA(period, shift)
func (c *SeriesCache) BullsPower(period, shift int) float64 {
	if c.src.Len() < period+shift {
		return 0
	}
	high, _ := c.src.High(shift).Float64()
	return high - c.EMA(period, shift)
}

// SAR = Parabolic SAR (recursive state)
func (c *SeriesCache) SAR(step, maximum float64, shift int) float64 {
	key := fmt.Sprintf("%v,%v", step, maximum)
	s, ok := c.sar[key]
	if !ok {
		s = newSARSeries(step, maximum)
		c.sar[key] = s
		c.rebuildSAR(s)
	}
	return s.at(shift)
}

// Force = MA-smoothed (Close - prevClose) * Volume
func (c *SeriesCache) Force(period int, method string, shift int) float64 {
	key := fmt.Sprintf("%d,%s", period, method)
	s, ok := c.force[key]
	if !ok {
		s = newForceSeries(period, method)
		c.force[key] = s
		c.rebuildForce(s)
	}
	return s.at(shift)
}

// AMA = Adaptive Moving Average (recursive state with ER/SC)
func (c *SeriesCache) AMA(period, fastP, slowP, shift int) float64 {
	key := fmt.Sprintf("%d,%d,%d", period, fastP, slowP)
	s, ok := c.ama[key]
	if !ok {
		s = newAMASeries(period, fastP, slowP)
		c.ama[key] = s
		c.rebuildAMA(s)
	}
	return s.at(shift)
}

// VIDyA = Variable Index Dynamic Average (recursive, delegates to stateless)
func (c *SeriesCache) VIDyA(cmoP, cmoShift, maP, maShift, shift int) float64 {
	v, _ := VIDyA(c.src, cmoP, cmoShift, maP, maShift, shift, 1).Float64()
	return v
}

// ── Rebuild helpers for oscillator series ──────────────────────────

func (c *SeriesCache) rebuildRSI(s *rsiSeries, _ int) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		close, _ := c.src.Close(i).Float64()
		s.update(close)
	}
}

func (c *SeriesCache) rebuildATR(s *atrSeries, _ int) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		high, _ := c.src.High(i).Float64()
		low, _ := c.src.Low(i).Float64()
		close, _ := c.src.Close(i).Float64()
		s.update(high, low, close)
	}
}

func (c *SeriesCache) rebuildADX(s *adxSeries, _ int) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		high, _ := c.src.High(i).Float64()
		low, _ := c.src.Low(i).Float64()
		close, _ := c.src.Close(i).Float64()
		s.update(high, low, close)
	}
}

func (c *SeriesCache) rebuildMACD(s *macdSeries, _, _ int) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		close, _ := c.src.Close(i).Float64()
		s.update(close)
	}
}

func (c *SeriesCache) rebuildSAR(s *sarSeries) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		high, _ := c.src.High(i).Float64()
		low, _ := c.src.Low(i).Float64()
		s.update(high, low)
	}
}

func (c *SeriesCache) rebuildForce(s *forceSeries) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		close, _ := c.src.Close(i).Float64()
		vol := float64(c.src.Volume(i))
		s.update(close, vol)
	}
}

func (c *SeriesCache) rebuildAMA(s *amaSeries) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		close, _ := c.src.Close(i).Float64()
		s.update(close)
	}
}

// ── Decimal wrappers for oscillator indicators ─────────────────────

func (c *SeriesCache) RSIDecimal(period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.RSI(period, shift))
}

func (c *SeriesCache) ATRDecimal(period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.ATR(period, shift))
}

func (c *SeriesCache) ADXDecimal(period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.ADX(period, shift))
}

func (c *SeriesCache) MACDLineDecimal(fast, slow, signalPeriod, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.MACDLine(fast, slow, signalPeriod, shift))
}

func (c *SeriesCache) MACDSignalDecimal(fast, slow, signalPeriod, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.MACDSignal(fast, slow, signalPeriod, shift))
}

func (c *SeriesCache) OsMADecimal(fast, slow, signalPeriod, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.OsMA(fast, slow, signalPeriod, shift))
}

func (c *SeriesCache) BearsPowerDecimal(period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.BearsPower(period, shift))
}

func (c *SeriesCache) BullsPowerDecimal(period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.BullsPower(period, shift))
}

func (c *SeriesCache) SARDecimal(step, maximum decimal.Decimal, shift int) decimal.Decimal {
	s, _ := step.Float64()
	m, _ := maximum.Float64()
	return decimal.NewFromFloat(c.SAR(s, m, shift))
}

func (c *SeriesCache) ForceDecimal(period int, method string, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.Force(period, method, shift))
}

func (c *SeriesCache) AMADecimal(period, fastP, slowP, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.AMA(period, fastP, slowP, shift))
}

func (c *SeriesCache) VIDyADecimal(cmoP, cmoShift, maP, maShift, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.VIDyA(cmoP, cmoShift, maP, maShift, shift))
}
