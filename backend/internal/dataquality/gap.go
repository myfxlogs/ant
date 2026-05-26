package dataquality

import "time"

// GapDetector tracks inter-tick intervals and flags gaps exceeding a threshold.
type GapDetector struct {
	MaxGapSeconds float64 // gap threshold in seconds (default 60s for forex, 300s for crypto)
	lastTickAt    int64   // unix nano of last tick
	gapCount      int
	maxGapNs      int64
	totalGapNs    int64
}

// NewGapDetector creates a gap detector with the given threshold.
func NewGapDetector(maxGapSeconds float64) *GapDetector {
	if maxGapSeconds <= 0 {
		maxGapSeconds = 60 // 1 min default
	}
	return &GapDetector{MaxGapSeconds: maxGapSeconds}
}

// Observe records a new tick timestamp. Returns true if a gap was detected.
func (d *GapDetector) Observe(t time.Time) bool {
	ts := t.UnixNano()
	if d.lastTickAt == 0 {
		d.lastTickAt = ts
		return false
	}

	intervalNs := ts - d.lastTickAt
	if intervalNs < 0 {
		intervalNs = 0 // clock skew, treat as no gap
	}
	d.lastTickAt = ts

	intervalSec := float64(intervalNs) / 1e9
	if intervalSec > d.MaxGapSeconds {
		d.gapCount++
		d.totalGapNs += intervalNs
		if intervalNs > d.maxGapNs {
			d.maxGapNs = intervalNs
		}
		return true
	}
	return false
}

// Stats returns the current gap statistics.
func (d *GapDetector) Stats() (count int, maxSec, totalSec float64) {
	return d.gapCount, float64(d.maxGapNs) / 1e9, float64(d.totalGapNs) / 1e9
}

// HasGaps returns true if any gaps have been detected.
func (d *GapDetector) HasGaps() bool { return d.gapCount > 0 }

// Reset clears gap counters.

func (d *GapDetector) Reset() {
	d.gapCount = 0
	d.maxGapNs = 0
	d.totalGapNs = 0
}
