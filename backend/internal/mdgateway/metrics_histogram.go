package mdgateway

import "sync/atomic"

// histogram is a simple bucket-based histogram (replaces Prometheus client
// until full Prometheus client_golang integration in M10.3-2 proper).
type histogram struct {
	buckets []float64
	counts  []atomic.Int64
	sum     atomic.Int64 // nanoseconds
}

func newHistogram(buckets []float64) *histogram {
	return &histogram{buckets: buckets, counts: make([]atomic.Int64, len(buckets))}
}

func (h *histogram) observe(seconds float64) {
	h.sum.Add(int64(seconds * 1e9))
	for i, b := range h.buckets {
		if seconds <= b {
			h.counts[i].Add(1)
			return
		}
	}
}

// percentile returns the bucket upper bound at or above the given percentile (0-100).
// Returns 0 if no observations recorded.
func (h *histogram) percentile(p float64) float64 {
	var total int64
	for i := range h.counts {
		total += h.counts[i].Load()
	}
	if total == 0 {
		return 0
	}
	threshold := int64(float64(total) * p / 100.0)
	var cum int64
	for i, b := range h.buckets {
		cum += h.counts[i].Load()
		if cum >= threshold {
			return b
		}
	}
	return h.buckets[len(h.buckets)-1]
}

// --- E2e latency ---

// e2eLatency records tick end-to-end latency buckets (seconds).
// Observed in pg_writer.go after successful flush.
var e2eLatency = newHistogram([]float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5})

// ObserveE2eLatency records a latency observation.
func ObserveE2eLatency(secs float64) {
	e2eLatency.observe(secs)
}

// E2eLatencyP99 returns the P99 end-to-end latency in seconds.
func E2eLatencyP99() float64 { return e2eLatency.percentile(99) }

// E2eLatencyCount returns the total number of latency observations.
func E2eLatencyCount() int64 {
	var total int64
	for i := range e2eLatency.counts {
		total += e2eLatency.counts[i].Load()
	}
	return total
}
