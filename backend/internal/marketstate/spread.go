package marketstate

import "math"

// SpreadTracker maintains a rolling window of spread observations and computes z-scores.
type SpreadTracker struct {
	window   []float64
	capacity int
	index    int
	count    int
	sum      float64
	sumSq    float64
}

// NewSpreadTracker creates a tracker with the given window size.
func NewSpreadTracker(windowSize int) *SpreadTracker {
	if windowSize <= 0 {
		windowSize = 100
	}
	return &SpreadTracker{
		window:   make([]float64, windowSize),
		capacity: windowSize,
	}
}

// Observe records a new spread observation (in pips).
func (t *SpreadTracker) Observe(spreadPips float64) {
	if t.count == t.capacity {
		old := t.window[t.index]
		t.sum -= old
		t.sumSq -= old * old
	} else {
		t.count++
	}
	t.window[t.index] = spreadPips
	t.sum += spreadPips
	t.sumSq += spreadPips * spreadPips
	t.index = (t.index + 1) % t.capacity
}

// Stats returns mean, stddev, and z-score for the given spread value.
func (t *SpreadTracker) Stats(currentSpread float64) (mean, stddev, zscore float64) {
	if t.count < 10 {
		return currentSpread, 0, 0
	}
	mean = t.sum / float64(t.count)
	variance := t.sumSq/float64(t.count) - mean*mean
	if variance < 0 {
		variance = 0
	}
	stddev = math.Sqrt(variance)
	if stddev > 0 {
		zscore = (currentSpread - mean) / stddev
	}
	return
}

// Mean returns the rolling mean spread.
func (t *SpreadTracker) Mean() float64 {
	if t.count == 0 {
		return 0
	}
	return t.sum / float64(t.count)
}

// Count returns the number of observations recorded.
func (t *SpreadTracker) Count() int { return t.count }
