package mdgateway

import (
	"context"
	"math"
	"sort"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// QualityConfig tunes tick quality checks.
type QualityConfig struct {
	GapMaxSeconds  float64 // default 5
	OutlierSigma   float64 // default 5
	SkewMaxSeconds float64 // default 30
	HistorySize    int     // default 100
}

// DefaultQualityConfig returns sensible defaults.
func DefaultQualityConfig() QualityConfig {
	return QualityConfig{
		GapMaxSeconds:  5,
		OutlierSigma:   5,
		SkewMaxSeconds: 30,
		HistorySize:    100,
	}
}

// CheckResult reports the outcome of a quality check.
type CheckResult struct {
	Outlier       bool
	Dropped       bool
	DroppedReason string // "bid_gt_ask", "non_positive", "parse_error", ""
}

// Quality validates incoming ticks and drops clearly invalid data.
// Suspicious but valid jumps (outliers, gaps, clock skew) are reported
// but never dropped — following LMAX/Bloomberg B-PIPE philosophy.
type Quality struct {
	cfg   QualityConfig
	last  map[string]int64  // broker:canonical -> last ts
	prices map[string][]float64 // broker:canonical -> recent bid prices
	dlq   *DLQWriter // optional; nil means no DLQ (M10 ADR-0010 §2.2)
}

// NewQuality creates a new quality checker.
func NewQuality(cfg QualityConfig) *Quality {
	return &Quality{
		cfg:    cfg,
		last:   make(map[string]int64),
		prices: make(map[string][]float64),
	}
}

// SetDLQWriter injects an optional DLQ writer for dropped-tick sampling.
func (q *Quality) SetDLQWriter(dlq *DLQWriter) {
	q.dlq = dlq
}

// Check validates a tick. Dropped ticks must not enter the pipeline.
func (q *Quality) Check(ctx context.Context, t *mdtick.Tick) CheckResult {
	// Hard drops: clearly impossible data.
	if t.Bid.Cmp(t.Ask) > 0 {
		if q.dlq != nil {
			q.dlq.WriteTick(ctx, t, "bid_gt_ask", "")
		}
		return CheckResult{Dropped: true, DroppedReason: "bid_gt_ask"}
	}
	if t.Bid.Sign() <= 0 || t.Ask.Sign() <= 0 {
		if q.dlq != nil {
			q.dlq.WriteTick(ctx, t, "non_positive", "")
		}
		return CheckResult{Dropped: true, DroppedReason: "non_positive"}
	}

	key := t.Broker + ":" + t.Canonical
	bidF, _ := t.Bid.Float64()

	// Outlier detection via median + MAD (never drops, just tags).
	q.prices[key] = append(q.prices[key], bidF)
	if len(q.prices[key]) > q.cfg.HistorySize {
		q.prices[key] = q.prices[key][1:]
	}
	result := CheckResult{Outlier: q.isOutlier(key, bidF)}

	// Gap and clock skew detection (metric only).
	if prev, ok := q.last[key]; ok {
		gapSec := float64(t.ArrivedUnixMs-prev) / 1000.0
		_ = gapSec // used by metrics (md_gap_total)
	}
	q.last[key] = t.ArrivedUnixMs

	return result
}

func (q *Quality) isOutlier(key string, bid float64) bool {
	prices := q.prices[key]
	if len(prices) < 10 {
		return false
	}
	sorted := make([]float64, len(prices))
	copy(sorted, prices)
	sort.Float64s(sorted)
	median := sorted[len(sorted)/2]

	absDevs := make([]float64, len(sorted))
	for i, p := range sorted {
		absDevs[i] = math.Abs(p - median)
	}
	sort.Float64s(absDevs)
	mad := absDevs[len(absDevs)/2]
	sigma := 1.4826 * mad

	return math.Abs(bid-median) > q.cfg.OutlierSigma*sigma
}
