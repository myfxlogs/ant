package indicators

// seriesMaxLen caps the number of values stored per indicator series.
// 1000 is generous: strategies rarely query shift > 500.
const seriesMaxLen = 1000

// floatSeries stores indicator values in chronological order.
// values[0] = oldest bar, values[len-1] = newest bar.
// At(shift) returns the value at `shift` bars ago from the newest.
// maxLen caps memory: when exceeded, oldest values are trimmed.
type floatSeries struct {
	values []float64
	maxLen int
}

func (s *floatSeries) append(v float64) {
	s.values = append(s.values, v)
	if s.maxLen > 0 && len(s.values) > s.maxLen {
		s.values = s.values[len(s.values)-s.maxLen:]
	}
}
func (s *floatSeries) len() int { return len(s.values) }

func (s *floatSeries) at(shift int) float64 {
	n := len(s.values)
	idx := n - 1 - shift
	if idx < 0 || idx >= n {
		return 0
	}
	return s.values[idx]
}

// ── EMA series (incremental, SMA-seeded) ───────────────────────────

type emaSeries struct {
	floatSeries
	period    int
	alpha     float64
	seedSum   float64
	seedCount int
}

func newEMASeries(period int) *emaSeries {
	return &emaSeries{period: period, alpha: 2.0 / float64(period+1), floatSeries: floatSeries{maxLen: seriesMaxLen}}
}

func (s *emaSeries) update(close float64) {
	if s.seedCount < s.period {
		s.seedSum += close
		s.seedCount++
		if s.seedCount == s.period {
			s.append(s.seedSum / float64(s.period))
			return
		}
		s.append(0)
		return
	}
	prev := s.values[len(s.values)-1]
	s.append(close*s.alpha + prev*(1-s.alpha))
}

// ── Wilder RSI series (incremental) ────────────────────────────────

type rsiSeries struct {
	floatSeries
	period    int
	avgGain   float64
	avgLoss   float64
	seedGain  float64
	seedLoss  float64
	seedCount int
	prevClose float64
	hasPrev   bool
}

func newRSISeries(period int) *rsiSeries {
	return &rsiSeries{period: period, floatSeries: floatSeries{maxLen: seriesMaxLen}}
}

func (s *rsiSeries) update(close float64) {
	if !s.hasPrev {
		s.prevClose = close
		s.append(0)
		s.hasPrev = true
		return
	}
	diff := close - s.prevClose
	gain, loss := 0.0, 0.0
	if diff > 0 {
		gain = diff
	} else {
		loss = -diff
	}
	s.prevClose = close

	if s.seedCount < s.period {
		s.seedGain += gain
		s.seedLoss += loss
		s.seedCount++
		if s.seedCount == s.period {
			s.avgGain = s.seedGain / float64(s.period)
			s.avgLoss = s.seedLoss / float64(s.period)
			s.append(rsiValue(s.avgGain, s.avgLoss))
			return
		}
		s.append(0)
		return
	}
	s.avgGain = (s.avgGain*float64(s.period-1) + gain) / float64(s.period)
	s.avgLoss = (s.avgLoss*float64(s.period-1) + loss) / float64(s.period)
	s.append(rsiValue(s.avgGain, s.avgLoss))
}

func rsiValue(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// ── SMA series (incremental, running sum) ──────────────────────────

type smaSeries struct {
	floatSeries
	period  int
	runSum  float64
	count   int
	ring    []float64
	ringIdx int
}

func newSMASeries(period int) *smaSeries {
	return &smaSeries{period: period, ring: make([]float64, period), floatSeries: floatSeries{maxLen: seriesMaxLen}}
}

func (s *smaSeries) update(close float64) {
	if s.count >= s.period {
		s.runSum -= s.ring[s.ringIdx]
	}
	s.ring[s.ringIdx] = close
	s.ringIdx = (s.ringIdx + 1) % s.period
	s.runSum += close
	s.count++
	if s.count >= s.period {
		s.append(s.runSum / float64(s.period))
	} else {
		s.append(0)
	}
}

// ── LWMA series (incremental, ring buffer) ──────────────────────────

type lwmaSeries struct {
	floatSeries
	period  int
	ring    []float64
	ringIdx int
	count   int
}

func newLWMASeries(period int) *lwmaSeries {
	return &lwmaSeries{period: period, ring: make([]float64, period), floatSeries: floatSeries{maxLen: seriesMaxLen}}
}

func (s *lwmaSeries) update(close float64) {
	s.ring[s.ringIdx] = close
	s.ringIdx = (s.ringIdx + 1) % s.period
	s.count++
	if s.count >= s.period {
		// Weights: oldest=1, newest=period. Ring is filled oldest→newest
		// in insertion order. ringIdx points to next write slot = oldest.
		var wsum float64
		w := 1.0
		for i := 0; i < s.period; i++ {
			idx := (s.ringIdx + i) % s.period
			wsum += s.ring[idx] * w
			w++
		}
		denom := float64(s.period) * float64(s.period+1) / 2.0
		s.append(wsum / denom)
	} else {
		s.append(0)
	}
}
