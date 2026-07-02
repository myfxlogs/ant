package indicators

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// ── Volume indicator query methods ─────────────────────────────────

// Chaikin = EMA(MFV, fast) - EMA(MFV, slow)
func (c *SeriesCache) Chaikin(fastP, slowP, shift int) float64 {
	key := fmt.Sprintf("%d,%d", fastP, slowP)
	s, ok := c.chaikin[key]
	if !ok {
		s = newChaikinSeries(fastP, slowP)
		c.chaikin[key] = s
		c.rebuildChaikin(s)
	}
	return s.at(shift)
}

// AD = cumulative Accumulation/Distribution
func (c *SeriesCache) AD(shift int) float64 {
	if c.ad == nil {
		c.ad = newADSeries()
		c.rebuildAD(c.ad)
	}
	return c.ad.at(shift)
}

// OBV = cumulative On-Balance Volume
func (c *SeriesCache) OBV(shift int) float64 {
	if c.obv == nil {
		c.obv = newOBVSeries()
		c.rebuildOBV(c.obv)
	}
	return c.obv.at(shift)
}

// OBVWithPrice returns OBV with appliedPrice.
func (c *SeriesCache) OBVWithPrice(appliedPrice, shift int) float64 {
	if appliedPrice <= 1 {
		return c.OBV(shift)
	}
	v, _ := OBV(c.src, shift, appliedPrice).Float64()
	return v
}

// ── Rebuild helpers for volume series ──────────────────────────────

func (c *SeriesCache) rebuildChaikin(s *chaikinSeries) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		high, _ := c.src.High(i).Float64()
		low, _ := c.src.Low(i).Float64()
		close, _ := c.src.Close(i).Float64()
		vol := float64(c.src.Volume(i))
		s.update(high, low, close, vol)
	}
}

func (c *SeriesCache) rebuildAD(s *adSeries) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		high, _ := c.src.High(i).Float64()
		low, _ := c.src.Low(i).Float64()
		close, _ := c.src.Close(i).Float64()
		vol := float64(c.src.Volume(i))
		s.update(high, low, close, vol)
	}
}

func (c *SeriesCache) rebuildOBV(s *obvSeries) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		close, _ := c.src.Close(i).Float64()
		vol := float64(c.src.Volume(i))
		s.update(close, vol)
	}
}

// ── Decimal wrappers for volume indicators ────────────────────────

func (c *SeriesCache) ChaikinDecimal(fastP, slowP, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.Chaikin(fastP, slowP, shift))
}

func (c *SeriesCache) ADDecimal(shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.AD(shift))
}

func (c *SeriesCache) OBVDecimal(shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.OBV(shift))
}
