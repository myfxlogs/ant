// Package mdgateway metrics (Prometheus counters/gauges/histograms).
// This file is progressively populated by M10 cards; see spec/11 §12 and spec/15 §3.
package mdgateway

import (
	"path/filepath"
	"sync/atomic"
)

// --- M10 ADR-0009 §2.2: bar finality ---

var barSkippedFinalized atomic.Int64

// BarSkippedFinalized returns the count of bars rejected by finality check.
func BarSkippedFinalized() int64 {
	return barSkippedFinalized.Load()
}

// --- M10 ADR-0010 §2.4: new metrics ---

// E2eLatency records tick end-to-end latency buckets (seconds).
// Observed in clickhouse_writer.go after successful flush.
var e2eLatency = newHistogram([]float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5})

// SpillPendingFiles is the count of unreplayed spill JSONL files.
// Updated every 30s by spill_replay goroutine.
var spillPendingFiles atomic.Int64

// DLQSampled tracks DLQ entries written, by reason.
var dlqSampled = map[string]*atomic.Int64{
	"parse_error": {},
	"bid_gt_ask":  {},
	"non_positive":{},
	"spill_failed":{},
}

// DLQSampled returns the DLQ sample count for a reason.
func DLQSampled(reason string) int64 {
	if c, ok := dlqSampled[reason]; ok {
		return c.Load()
	}
	return 0
}

// ObserveE2eLatency records a latency observation.
func ObserveE2eLatency(secs float64) {
	e2eLatency.observe(secs)
}

// UpdateSpillPendingFiles scans the spill directory and updates the gauge.
func UpdateSpillPendingFiles(spillDir string) {
	if spillDir == "" {
		return
	}
	files, _ := filepath.Glob(spillDir + "/*.jsonl")
	spillPendingFiles.Store(int64(len(files)))
}

// SpillPendingFilesCount returns the current spill backlog count.
func SpillPendingFilesCount() int64 {
	return spillPendingFiles.Load()
}

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

