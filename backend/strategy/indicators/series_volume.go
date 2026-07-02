package indicators

// ── Generic EMA on arbitrary input (for Chaikin MFV) ───────────────

type emaInputSeries struct {
	floatSeries
	period    int
	alpha     float64
	seedSum   float64
	seedCount int
}

func newEMAInputSeries(period int) *emaInputSeries {
	return &emaInputSeries{period: period, alpha: 2.0 / float64(period+1), floatSeries: floatSeries{maxLen: seriesMaxLen}}
}

func (s *emaInputSeries) update(value float64) {
	if s.seedCount < s.period {
		s.seedSum += value
		s.seedCount++
		if s.seedCount == s.period {
			s.append(s.seedSum / float64(s.period))
			return
		}
		s.append(0)
		return
	}
	prev := s.values[len(s.values)-1]
	s.append(value*s.alpha + prev*(1-s.alpha))
}

// ── Chaikin series (EMA of MFV) ────────────────────────────────────

type chaikinSeries struct {
	fast *emaInputSeries
	slow *emaInputSeries
}

func newChaikinSeries(fastP, slowP int) *chaikinSeries {
	return &chaikinSeries{
		fast: newEMAInputSeries(fastP),
		slow: newEMAInputSeries(slowP),
	}
}

func (s *chaikinSeries) update(high, low, close, volume float64) {
	hl := high - low
	var mfv float64
	if hl != 0 {
		mfv = ((close - low) - (high - close)) / hl * volume
	}
	s.fast.update(mfv)
	s.slow.update(mfv)
}

func (s *chaikinSeries) at(shift int) float64 {
	return s.fast.at(shift) - s.slow.at(shift)
}

// ── AD series (cumulative Accumulation/Distribution) ───────────────

type adSeries struct {
	floatSeries
	total float64
}

func newADSeries() *adSeries {
	return &adSeries{floatSeries: floatSeries{maxLen: seriesMaxLen}}
}

func (s *adSeries) update(high, low, close, volume float64) {
	hl := high - low
	if hl != 0 {
		s.total += ((close - low) - (high - close)) / hl * volume
	}
	s.append(s.total)
}

// ── OBV series (cumulative On-Balance Volume) ──────────────────────

type obvSeries struct {
	floatSeries
	total     float64
	prevClose float64
	hasPrev   bool
}

func newOBVSeries() *obvSeries {
	return &obvSeries{floatSeries: floatSeries{maxLen: seriesMaxLen}}
}

func (s *obvSeries) update(close, volume float64) {
	if s.hasPrev {
		if close > s.prevClose {
			s.total += volume
		} else if close < s.prevClose {
			s.total -= volume
		}
	}
	s.prevClose = close
	s.hasPrev = true
	s.append(s.total)
}
