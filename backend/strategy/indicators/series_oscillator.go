package indicators

import "math"

// ── Wilder ATR series (incremental) ────────────────────────────────

type atrSeries struct {
	floatSeries
	period    int
	seedSum   float64
	seedCount int
	atr       float64
	seeded    bool
	prevClose float64
	hasPrev   bool
}

func newATRSeries(period int) *atrSeries {
	return &atrSeries{period: period, floatSeries: floatSeries{maxLen: seriesMaxLen}}
}

func (s *atrSeries) update(high, low, close float64) {
	tr := high - low
	if s.hasPrev {
		tr2 := high - s.prevClose
		if tr2 < 0 {
			tr2 = -tr2
		}
		tr3 := s.prevClose - low
		if tr3 < 0 {
			tr3 = -tr3
		}
		if tr2 > tr {
			tr = tr2
		}
		if tr3 > tr {
			tr = tr3
		}
	}
	s.prevClose = close
	s.hasPrev = true

	if !s.seeded {
		s.seedSum += tr
		s.seedCount++
		if s.seedCount == s.period {
			s.atr = s.seedSum / float64(s.period)
			s.append(s.atr)
			s.seeded = true
			return
		}
		s.append(0)
		return
	}
	s.atr = (s.atr*float64(s.period-1) + tr) / float64(s.period)
	s.append(s.atr)
}

// ── Wilder ADX series (incremental) ────────────────────────────────

type adxSeries struct {
	floatSeries
	period int
	// Smoothed +DM, -DM, TR (Wilder)
	smoothPlusDM  float64
	smoothMinusDM float64
	smoothTR      float64
	// Seeding state
	seedPlusDM  float64
	seedMinusDM float64
	seedTR      float64
	seedCount   int
	seeded      bool
	// ADX state
	lastDX     float64
	adx        float64
	adxSeeded  bool
	adxSeedSum float64
	adxSeedCnt int
	// Previous bar data
	prevHigh, prevLow, prevClose float64
	hasPrev                      bool
}

func newADXSeries(period int) *adxSeries {
	return &adxSeries{period: period, floatSeries: floatSeries{maxLen: seriesMaxLen}}
}

func (s *adxSeries) update(high, low, close float64) {
	if !s.hasPrev {
		s.prevHigh = high
		s.prevLow = low
		s.prevClose = close
		s.hasPrev = true
		return
	}

	// Compute +DM, -DM, TR
	plusDM := high - s.prevHigh
	minusDM := s.prevLow - low
	if plusDM < 0 {
		plusDM = 0
	}
	if minusDM < 0 {
		minusDM = 0
	}
	if plusDM > minusDM {
		minusDM = 0
	} else {
		plusDM = 0
	}

	tr1 := high - low
	tr2 := high - s.prevClose
	if tr2 < 0 {
		tr2 = -tr2
	}
	tr3 := s.prevClose - low
	if tr3 < 0 {
		tr3 = -tr3
	}
	tr := tr1
	if tr2 > tr {
		tr = tr2
	}
	if tr3 > tr {
		tr = tr3
	}

	s.prevHigh = high
	s.prevLow = low
	s.prevClose = close

	// Seed phase: accumulate first `period` values
	if !s.seeded {
		s.seedPlusDM += plusDM
		s.seedMinusDM += minusDM
		s.seedTR += tr
		s.seedCount++
		if s.seedCount == s.period {
			s.smoothPlusDM = s.seedPlusDM
			s.smoothMinusDM = s.seedMinusDM
			s.smoothTR = s.seedTR
			s.seeded = true
			// Don't compute DX here — DX is only computed after Wilder smoothing
		}
		// ADX not ready yet during seed phase
		s.append(0)
		return
	}

	// Wilder smoothing for +DM, -DM, TR
	s.smoothTR = s.smoothTR - s.smoothTR/float64(s.period) + tr
	s.smoothPlusDM = s.smoothPlusDM - s.smoothPlusDM/float64(s.period) + plusDM
	s.smoothMinusDM = s.smoothMinusDM - s.smoothMinusDM/float64(s.period) + minusDM

	s.computeDX()

	// ADX seeding: need `period` DX values
	if !s.adxSeeded {
		s.adxSeedSum += s.lastDX
		s.adxSeedCnt++
		if s.adxSeedCnt == s.period {
			s.adx = s.adxSeedSum / float64(s.period)
			s.adxSeeded = true
			s.append(s.adx)
			return
		}
		s.append(s.lastDX)
		return
	}

	// Wilder smoothing for ADX
	lastDX := s.lastDX
	s.adx = (s.adx*float64(s.period-1) + lastDX) / float64(s.period)
	s.append(s.adx)
}

func (s *adxSeries) computeDX() {
	if s.smoothTR == 0 {
		s.lastDX = 0
		return
	}
	plusDI := 100 * s.smoothPlusDM / s.smoothTR
	minusDI := 100 * s.smoothMinusDM / s.smoothTR
	diSum := plusDI + minusDI
	if diSum == 0 {
		s.lastDX = 0
		return
	}
	s.lastDX = 100 * math.Abs(plusDI-minusDI) / diSum
}

// ── MACD series (incremental) ──────────────────────────────────────

type macdSeries struct {
	floatSeries // MACD line values
	fast        *emaSeries
	slow        *emaSeries
	signal      *emaSeries // EMA of MACD values
	signalVals  *floatSeries
}

func newMACDSeries(fast, slow, signalPeriod int) *macdSeries {
	return &macdSeries{
		fast:       newEMASeries(fast),
		slow:       newEMASeries(slow),
		signal:     newEMASeries(signalPeriod),
		signalVals: &floatSeries{maxLen: seriesMaxLen},
	}
}

func (s *macdSeries) update(close float64) {
	s.fast.update(close)
	s.slow.update(close)

	// MACD line = EMA(fast) - EMA(slow)
	// Only available when both EMAs are seeded
	fastVal := s.fast.values[len(s.fast.values)-1]
	slowVal := s.slow.values[len(s.slow.values)-1]
	macdVal := fastVal - slowVal
	s.append(macdVal)

	// Signal = EMA of MACD values (only when MACD is meaningful, i.e., both EMAs seeded)
	if s.fast.seedCount >= s.fast.period && s.slow.seedCount >= s.slow.period {
		s.signal.update(macdVal)
		s.signalVals.append(s.signal.values[len(s.signal.values)-1])
	} else {
		s.signalVals.append(0)
	}
}

func (s *macdSeries) macdAt(shift int) float64 {
	return s.at(shift)
}

func (s *macdSeries) signalAt(shift int) float64 {
	return s.signalVals.at(shift)
}
