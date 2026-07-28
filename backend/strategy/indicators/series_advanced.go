package indicators

import "math"

// ── SAR series (Parabolic SAR, recursive) ──────────────────────────

type sarSeries struct {
	floatSeries
	accel   float64
	maxVal  float64
	isUp    bool
	sar     float64
	ep      float64
	af      float64
	hasPrev bool
	// Ring buffer for last 2 bars' high/low (for SAR clamping)
	ringHigh [2]float64
	ringLow  [2]float64
	ringCnt  int
	ringIdx  int
}

func newSARSeries(step, maximum float64) *sarSeries {
	return &sarSeries{accel: step, maxVal: maximum, floatSeries: floatSeries{maxLen: seriesMaxLen}}
}

func (s *sarSeries) update(high, low float64) {
	if !s.hasPrev {
		// First bar: initialize state, no clamping needed
		s.ringHigh[0] = high
		s.ringLow[0] = low
		s.ringIdx = 1
		s.ringCnt = 1
		s.isUp = true
		s.sar = low
		s.ep = high
		s.af = s.accel
		s.hasPrev = true
		s.append(s.sar)
		return
	}

	// Read previous bars' high/low BEFORE overwriting ring
	// ringIdx points to next slot to write = oldest of stored bars
	// prev1 = most recent stored = ring[(ringIdx-1+2)%2]
	// prev2 = oldest stored = ring[ringIdx]
	// Use sentinels so missing prev2 is a no-op (matching stateless i+2>=n skip)
	prev1High, prev1Low := -math.MaxFloat64, math.MaxFloat64
	prev2High, prev2Low := -math.MaxFloat64, math.MaxFloat64
	if s.ringCnt >= 1 {
		p1 := (s.ringIdx - 1 + 2) % 2
		prev1High = s.ringHigh[p1]
		prev1Low = s.ringLow[p1]
	}
	if s.ringCnt >= 2 {
		p2 := s.ringIdx
		prev2High = s.ringHigh[p2]
		prev2Low = s.ringLow[p2]
	}

	// Now store current bar in ring
	s.ringHigh[s.ringIdx] = high
	s.ringLow[s.ringIdx] = low
	s.ringIdx = (s.ringIdx + 1) % 2
	if s.ringCnt < 2 {
		s.ringCnt++
	}

	if s.isUp {
		newSAR := s.sar + s.af*(s.ep-s.sar)
		if newSAR > prev1Low {
			newSAR = prev1Low
		}
		if newSAR > prev2Low {
			newSAR = prev2Low
		}
		if low < newSAR {
			s.isUp = false
			s.sar = s.ep
			s.ep = low
			s.af = s.accel
		} else {
			s.sar = newSAR
			if high > s.ep {
				s.ep = high
				if s.af+s.accel <= s.maxVal {
					s.af += s.accel
				} else {
					s.af = s.maxVal
				}
			}
		}
	} else {
		newSAR := s.sar + s.af*(s.ep-s.sar)
		if newSAR < prev1High {
			newSAR = prev1High
		}
		if newSAR < prev2High {
			newSAR = prev2High
		}
		if high > newSAR {
			s.isUp = true
			s.sar = s.ep
			s.ep = high
			s.af = s.accel
		} else {
			s.sar = newSAR
			if low < s.ep {
				s.ep = low
				if s.af+s.accel <= s.maxVal {
					s.af += s.accel
				} else {
					s.af = s.maxVal
				}
			}
		}
	}
	s.append(s.sar)
}

// ── Force series (incremental, MA-smoothed) ────────────────────────

type forceSeries struct {
	floatSeries
	period    int
	prevClose float64
	hasPrev   bool
	ma        forceMA
}

// forceMA is the MA smoother used by forceSeries.
type forceMA interface {
	update(value float64)
	at(shift int) float64
}

func newForceSeries(period int, method string) *forceSeries {
	var ma forceMA
	switch method {
	case "EMA", "ema":
		ma = newEMASeries(period)
	case "SMMA", "smma":
		ma = &emaSeries{period: period, alpha: 1.0 / float64(period), floatSeries: floatSeries{maxLen: seriesMaxLen}}
	case "LWMA", "lwma":
		ma = newLWMASeries(period)
	default:
		ma = newSMASeries(period)
	}
	return &forceSeries{period: period, ma: ma, floatSeries: floatSeries{maxLen: seriesMaxLen}}
}

func (s *forceSeries) update(close, volume float64) {
	if !s.hasPrev {
		s.prevClose = close
		s.hasPrev = true
		s.append(0)
		s.ma.update(0)
		return
	}
	force := (close - s.prevClose) * volume
	s.prevClose = close
	s.append(force)
	s.ma.update(force)
}

func (s *forceSeries) at(shift int) float64 {
	return s.ma.at(shift)
}

// ── AMA series (incremental, Adaptive Moving Average) ──────────────

type amaSeries struct {
	floatSeries
	period  int
	fastSC  float64
	slowSC  float64
	ring    []float64 // close ring buffer, size period+1
	ringIdx int
	ringCnt int
	seedSum float64
	seedCnt int
	amaVal  float64
	hasAMA  bool
}

func newAMASeries(period, fastP, slowP int) *amaSeries {
	return &amaSeries{
		period:      period,
		fastSC:      2.0 / float64(fastP+1),
		slowSC:      2.0 / float64(slowP+1),
		ring:        make([]float64, period+1),
		floatSeries: floatSeries{maxLen: seriesMaxLen},
	}
}

func (s *amaSeries) update(close float64) {
	// Store close in ring
	s.ring[s.ringIdx] = close
	s.ringIdx = (s.ringIdx + 1) % (s.period + 1)
	if s.ringCnt < s.period+1 {
		s.ringCnt++
	}

	// Seed phase: accumulate SMA
	if s.seedCnt < s.period {
		s.seedSum += close
		s.seedCnt++
		if s.seedCnt == s.period {
			s.amaVal = s.seedSum / float64(s.period)
			s.hasAMA = true
			s.append(s.amaVal)
			return
		}
		s.append(0)
		return
	}

	// We have period+1 closes in ring. Compute ER and SC.
	// ringIdx points to next write slot = oldest.
	// close[0] = current bar (newest), close[period] = oldest
	// newest = ring[(ringIdx-1+period+1)%(period+1)]
	// oldest = ring[ringIdx]
	newestIdx := (s.ringIdx - 1 + s.period + 1) % (s.period + 1)
	oldestIdx := s.ringIdx
	newest := s.ring[newestIdx]
	oldest := s.ring[oldestIdx]

	change := math.Abs(newest - oldest)
	var volatility float64
	for i := 0; i < s.period; i++ {
		idx1 := (s.ringIdx + i) % (s.period + 1)
		idx2 := (s.ringIdx + i + 1) % (s.period + 1)
		volatility += math.Abs(s.ring[idx1] - s.ring[idx2])
	}
	er := 1.0
	if volatility > 0 {
		er = change / volatility
	}
	tmp := er*(s.fastSC-s.slowSC) + s.slowSC
	sc := tmp * tmp
	s.amaVal = newest + sc*(newest-s.amaVal)
	s.append(s.amaVal)
}

// ── DEMA series (incremental, nested EMA) ──────────────────────────

// demaSeries computes DEMA = 2*EMA1 - EMA2 where EMA2 = EMA(EMA1).
// EMA1 is the embedded emaSeries. EMA2 feeds on EMA1's output,
// skipping the seed-phase zeros (first `period` bars produce 0).
type demaSeries struct {
	ema1 *emaSeries // EMA of close prices
	ema2 *emaSeries // EMA of EMA1 values (skips seed zeros)
}

func newDEMASeries(period int) *demaSeries {
	return &demaSeries{
		ema1: newEMASeries(period),
		ema2: newEMASeries(period),
	}
}

func (s *demaSeries) update(close float64) {
	s.ema1.update(close)
	// Feed EMA1's latest value to EMA2, but skip seed-phase zeros.
	// EMA1 produces zeros for the first period-1 bars, then the SMA seed.
	// We only feed valid (post-seed) values to EMA2.
	n := s.ema1.len()
	if n <= s.ema1.period {
		// EMA1 hasn't completed seeding yet; EMA2 gets no input.
		return
	}
	// The value at index n-1 is the latest valid EMA1 output.
	ema1Val := s.ema1.values[n-1]
	s.ema2.update(ema1Val)
}

func (s *demaSeries) at(shift int) float64 {
	e1 := s.ema1.at(shift)
	e2 := s.ema2.at(shift)
	return 2*e1 - e2
}

// ── TEMA series (incremental, triple nested EMA) ───────────────────

// temaSeries computes TEMA = 3*EMA1 - 3*EMA2 + EMA3
// where EMA2 = EMA(EMA1) and EMA3 = EMA(EMA2).
type temaSeries struct {
	ema1 *emaSeries
	ema2 *emaSeries
	ema3 *emaSeries
}

func newTEMASeries(period int) *temaSeries {
	return &temaSeries{
		ema1: newEMASeries(period),
		ema2: newEMASeries(period),
		ema3: newEMASeries(period),
	}
}

func (s *temaSeries) update(close float64) {
	s.ema1.update(close)
	n1 := s.ema1.len()
	if n1 <= s.ema1.period {
		return
	}
	ema1Val := s.ema1.values[n1-1]
	s.ema2.update(ema1Val)
	n2 := s.ema2.len()
	if n2 <= s.ema2.period {
		return
	}
	ema2Val := s.ema2.values[n2-1]
	s.ema3.update(ema2Val)
}

func (s *temaSeries) at(shift int) float64 {
	e1 := s.ema1.at(shift)
	e2 := s.ema2.at(shift)
	e3 := s.ema3.at(shift)
	return 3*e1 - 3*e2 + e3
}
