package mdgateway

import "sync/atomic"

// --- M10 ADR-0009 §2.2: bar finality ---

var barSkippedFinalized atomic.Int64

// BarSkippedFinalized returns the count of bars rejected by finality check.
func BarSkippedFinalized() int64 {
	return barSkippedFinalized.Load()
}

// --- Inter-tick gap metrics ---

var (
	gapSeconds  atomic.Int64 // cumulative inter-tick gap in milliseconds
	gapCount    atomic.Int64
	gapMaxMs    atomic.Int64
	gapExceeded atomic.Int64 // gaps exceeding GapMaxSeconds
)

// RecordGap records an inter-tick gap observation.
func RecordGap(gapMs int64, maxMs int64) {
	gapSeconds.Add(gapMs)
	gapCount.Add(1)
	if gapMs > gapMaxMs.Load() {
		gapMaxMs.Store(gapMs)
	}
	if maxMs > 0 && gapMs > maxMs {
		gapExceeded.Add(1)
	}
}

// GapAvgSeconds returns the average inter-tick gap in seconds.
func GapAvgSeconds() float64 {
	n := gapCount.Load()
	if n == 0 {
		return 0
	}
	return float64(gapSeconds.Load()) / float64(n) / 1000.0
}

// GapMaxSeconds returns the maximum observed gap in seconds.
func GapMaxSeconds() float64 { return float64(gapMaxMs.Load()) / 1000.0 }

// GapExceeded returns the count of gaps exceeding the threshold.
func GapExceeded() int64 { return gapExceeded.Load() }

// --- Clock skew metrics ---

var (
	skewMaxMs    atomic.Int64
	skewCount    atomic.Int64
	skewExceeded atomic.Int64 // skew exceeding SkewMaxSeconds
)

// RecordClockSkew records a clock skew observation (arrived - broker timestamp).
func RecordClockSkew(skewMs int64, maxMs int64) {
	if skewMs < 0 {
		skewMs = -skewMs
	}
	skewCount.Add(1)
	if skewMs > skewMaxMs.Load() {
		skewMaxMs.Store(skewMs)
	}
	if maxMs > 0 && skewMs > maxMs {
		skewExceeded.Add(1)
	}
}

// ClockSkewMaxSeconds returns the max observed clock skew in seconds.
func ClockSkewMaxSeconds() float64 { return float64(skewMaxMs.Load()) / 1000.0 }

// ClockSkewExceeded returns the count of skews exceeding the threshold.
func ClockSkewExceeded() int64 { return skewExceeded.Load() }

// --- Stale data metrics ---

var (
	staleAccounts atomic.Int64
	deadAccounts  atomic.Int64
)

// SetStaleAccountCount records the number of stale/dead accounts for alerting.
func SetStaleAccountCount(stale, dead int64) {
	staleAccounts.Store(stale)
	deadAccounts.Store(dead)
}

// StaleAccountCount returns the number of accounts with no ticks for >5 min.
func StaleAccountCount() int64 { return staleAccounts.Load() }

// DeadAccountCount returns the number of accounts with no ticks for >15 min.
func DeadAccountCount() int64 { return deadAccounts.Load() }

// --- M10-BASE-F4: Quote stuffing detection ---

var stuffingDetected atomic.Int64

func recordStuffingDetected() { stuffingDetected.Add(1) }

// StuffingDetectedTotal returns the count of quote stuffing detections.
func StuffingDetectedTotal() int64 { return stuffingDetected.Load() }

// --- M10-BASE-F5: Spread anomaly ---

var spreadAnomalyTotal atomic.Int64

// RecordSpreadAnomaly increments the spread anomaly counter.
func RecordSpreadAnomaly() { spreadAnomalyTotal.Add(1) }

// SpreadAnomalyTotal returns the count of spread anomaly detections.
func SpreadAnomalyTotal() int64 { return spreadAnomalyTotal.Load() }

// --- M10-BASE-F3: Clock skew drop ---

var clockSkewDropped atomic.Int64

// RecordClockSkewDropped increments the clock skew drop counter.
func RecordClockSkewDropped() { clockSkewDropped.Add(1) }

// ClockSkewDroppedTotal returns the count of ticks dropped due to clock skew.
func ClockSkewDroppedTotal() int64 { return clockSkewDropped.Load() }
