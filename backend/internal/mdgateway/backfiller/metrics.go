package backfiller

import "sync/atomic"

// Metrics for the backfiller (spec/18 §7).
type Metrics struct {
	started      atomic.Int64
	barsIngested atomic.Int64
	errors       atomic.Int64
	durationNs   atomic.Int64 // cumulative backfill duration in nanoseconds
	durationCnt  atomic.Int64
}

// Started returns the count of backfill runs started.
func (m *Metrics) Started() int64 { return m.started.Load() }

// BarsIngested returns the count of bars successfully ingested.
func (m *Metrics) BarsIngested() int64 { return m.barsIngested.Load() }

// Errors returns the count of backfill errors.
func (m *Metrics) Errors() int64 { return m.errors.Load() }

// DurationSumSeconds returns the accumulated backfill duration in seconds.
func (m *Metrics) DurationSumSeconds() float64 {
	return float64(m.durationNs.Load()) / 1e9
}
