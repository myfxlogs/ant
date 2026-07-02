package indicators

import (
	"github.com/shopspring/decimal"
)

// ── Moving average query methods ───────────────────────────────────

func (c *SeriesCache) EMA(period, shift int) float64 {
	s, ok := c.ema[period]
	if !ok {
		s = newEMASeries(period)
		c.ema[period] = s
		c.rebuild(s.update, period)
	}
	return s.at(shift)
}

func (c *SeriesCache) SMMA(period, shift int) float64 {
	s, ok := c.smma[period]
	if !ok {
		// SMMA = Wilder smoothing, alpha = 1/period (not 2/(period+1))
		s = &emaSeries{period: period, alpha: 1.0 / float64(period)}
		c.smma[period] = s
		c.rebuildSMMA(s, period)
	}
	return s.at(shift)
}

func (c *SeriesCache) SMA(period, shift int) float64 {
	s, ok := c.sma[period]
	if !ok {
		s = newSMASeries(period)
		c.sma[period] = s
		c.rebuild(s.update, period)
	}
	return s.at(shift)
}

func (c *SeriesCache) LWMA(period, shift int) float64 {
	s, ok := c.lwma[period]
	if !ok {
		s = newLWMASeries(period)
		c.lwma[period] = s
		c.rebuild(s.update, period)
	}
	return s.at(shift)
}

// DEMA = 2*EMA - EMA(EMA), cached incrementally
func (c *SeriesCache) DEMA(period, shift int) float64 {
	s, ok := c.dema[period]
	if !ok {
		s = newDEMASeries(period)
		c.dema[period] = s
		c.rebuildDEMA(s)
	}
	return s.at(shift)
}

// TEMA = 3*EMA - 3*EMA(EMA) + EMA(EMA(EMA)), cached incrementally
func (c *SeriesCache) TEMA(period, shift int) float64 {
	s, ok := c.tema[period]
	if !ok {
		s = newTEMASeries(period)
		c.tema[period] = s
		c.rebuildTEMA(s)
	}
	return s.at(shift)
}

// TriX delegates to stateless (needs EMA3 percentage change, not a MA value)
func (c *SeriesCache) TriX(period, shift int) float64 {
	v, _ := TriX(c.src, period, shift, 1).Float64()
	return v
}

// Alligator returns jaw, teeth, lips using SMMA with shifts.
func (c *SeriesCache) Alligator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, shift int) (jaw, teeth, lips float64) {
	maFunc := c.SMMA
	switch method {
	case "SMA", "sma":
		maFunc = c.SMA
	case "EMA", "ema":
		maFunc = c.EMA
	case "SMMA", "smma", "":
		maFunc = c.SMMA
	case "LWMA", "lwma":
		maFunc = c.LWMA
	default:
		maFunc = c.SMMA
	}
	jaw = maFunc(jawPeriod, shift+jawShift)
	teeth = maFunc(teethPeriod, shift+teethShift)
	lips = maFunc(lipsPeriod, shift+lipsShift)
	return
}

// Gator = |jaw-teeth| (upper), |teeth-lips| (lower)
func (c *SeriesCache) Gator(jawP, jawS, teethP, teethS, lipsP, lipsS int, method string, shift int) (upper, lower float64) {
	jaw, teeth, lips := c.Alligator(jawP, jawS, teethP, teethS, lipsP, lipsS, method, shift)
	upper = jaw - teeth
	if upper < 0 {
		upper = -upper
	}
	lower = teeth - lips
	if lower < 0 {
		lower = -lower
	}
	return
}

// Envelopes with cached MA
func (c *SeriesCache) Envelopes(period int, deviation float64, method string, shift int) (upper, lower float64) {
	var mid float64
	switch method {
	case "EMA", "ema":
		mid = c.EMA(period, shift)
	case "SMMA", "smma":
		mid = c.SMMA(period, shift)
	case "LWMA", "lwma":
		mid = c.LWMA(period, shift)
	default:
		mid = c.SMA(period, shift)
	}
	band := mid * deviation / 100.0
	return mid + band, mid - band
}

// ── Rebuild helpers for MA series ──────────────────────────────────

func (c *SeriesCache) rebuildSMMA(s *emaSeries, _ int) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		close, _ := c.src.Close(i).Float64()
		s.update(close)
	}
}

func (c *SeriesCache) rebuildDEMA(s *demaSeries) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		close, _ := c.src.Close(i).Float64()
		s.update(close)
	}
}

func (c *SeriesCache) rebuildTEMA(s *temaSeries) {
	n := c.src.Len()
	for i := n - 1; i >= 0; i-- {
		close, _ := c.src.Close(i).Float64()
		s.update(close)
	}
}

// ── Decimal wrappers for MA indicators ─────────────────────────────

func (c *SeriesCache) EMADecimal(period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.EMA(period, shift))
}

func (c *SeriesCache) SMMADecimal(period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.SMMA(period, shift))
}

func (c *SeriesCache) SMADecimal(period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.SMA(period, shift))
}

func (c *SeriesCache) LWMADecimal(period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.LWMA(period, shift))
}

func (c *SeriesCache) DEMADecimal(period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.DEMA(period, shift))
}

func (c *SeriesCache) TEMADecimal(period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.TEMA(period, shift))
}

func (c *SeriesCache) TriXDecimal(period, shift int) decimal.Decimal {
	return decimal.NewFromFloat(c.TriX(period, shift))
}

func (c *SeriesCache) AlligatorDecimal(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift int, method string, shift int) (jaw, teeth, lips decimal.Decimal) {
	j, t, l := c.Alligator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, shift)
	return decimal.NewFromFloat(j), decimal.NewFromFloat(t), decimal.NewFromFloat(l)
}

func (c *SeriesCache) GatorDecimal(jawP, jawS, teethP, teethS, lipsP, lipsS int, method string, shift int) (upper, lower decimal.Decimal) {
	u, l := c.Gator(jawP, jawS, teethP, teethS, lipsP, lipsS, method, shift)
	return decimal.NewFromFloat(u), decimal.NewFromFloat(l)
}

func (c *SeriesCache) EnvelopesDecimal(period int, deviation decimal.Decimal, method string, shift int) (upper, lower decimal.Decimal) {
	dev, _ := deviation.Float64()
	u, l := c.Envelopes(period, dev, method, shift)
	return decimal.NewFromFloat(u), decimal.NewFromFloat(l)
}
