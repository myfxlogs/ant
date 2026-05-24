// Package mdgateway metrics (Prometheus counters/gauges/histograms).
// This file is progressively populated by M10 cards; see spec/11 §12 and spec/15 §3.
package mdgateway

import "sync/atomic"

// --- M10 ADR-0009 §2.2: bar finality ---

var barSkippedFinalized atomic.Int64

// BarSkippedFinalized returns the count of bars rejected by finality check.
func BarSkippedFinalized() int64 {
	return barSkippedFinalized.Load()
}
