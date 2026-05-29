package mdgateway

import (
	"context"
	"math"
	"sort"
	"sync"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// QualityConfig tunes tick quality checks.
type QualityConfig struct {
	GapMaxSeconds    float64 // default 5
	OutlierSigma     float64 // default 5
	SkewMaxSeconds   float64 // default 30
	MaxClockSkewMs   int64   // max NTP deviation before drop (default 5000ms, M10-BASE-F3)
	HistorySize      int     // default 100
	MaxSpreadZscore  float64 // spread anomaly threshold (default 3.0, M10-BASE-F5)
	MaxTickRateZscore float64 // quote stuffing threshold (default 4.0, M10-BASE-F4)
}

// DefaultQualityConfig returns sensible defaults.
func DefaultQualityConfig() QualityConfig {
	return QualityConfig{
		GapMaxSeconds:     5,
		OutlierSigma:      5,
		SkewMaxSeconds:    30,
		MaxClockSkewMs:    5000,
		HistorySize:       100,
		MaxSpreadZscore:   3.0,
		MaxTickRateZscore: 4.0,
	}
}

// CheckResult reports the outcome of a quality check.
type CheckResult struct {
	Outlier       bool
	Dropped       bool
	DroppedReason string // "bid_gt_ask", "non_positive", "parse_error", "clock_skew", ""
	SpreadBps     float64 // spread in basis points (M10-BASE-F5)
	ClockSkewMs   int64   // NTP clock deviation (M10-BASE-F3)
}

// Quality validates incoming ticks and drops clearly invalid data.
// Suspicious but valid jumps (outliers, gaps, clock skew) are reported
// but never dropped — following LMAX/Bloomberg B-PIPE philosophy.
type Quality struct {
	cfg    QualityConfig
	mu     sync.Mutex
	last   map[string]int64       // broker:canonical -> last ts
	prices map[string][]float64   // broker:canonical -> recent bid prices
	dlq    *DLQWriter             // optional; nil means no DLQ (M10 ADR-0010 §2.2)
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

	// M10-BASE-F3: NTP clock skew hard drop (abs > 5000ms).
	skewMs := t.ArrivedUnixMs - t.TsUnixMs
	if abs64(skewMs) > q.cfg.MaxClockSkewMs {
		RecordClockSkewDropped()
		if q.dlq != nil {
			q.dlq.WriteTick(ctx, t, "clock_skew", "")
		}
		return CheckResult{Dropped: true, DroppedReason: "clock_skew", ClockSkewMs: skewMs}
	}
	RecordClockSkew(skewMs, int64(q.cfg.SkewMaxSeconds*1000))

	key := t.Broker + ":" + t.Canonical
	bidF, _ := t.Bid.Float64()
	askF, _ := t.Ask.Float64()

	// Compute spread in basis points (M10-BASE-F5).
	var spreadBps float64
	if bidF > 0 {
		spreadBps = (askF - bidF) / bidF * 10000
	}

	q.mu.Lock()
	// Outlier detection via median + MAD (never drops, just tags).
	q.prices[key] = append(q.prices[key], bidF)
	if len(q.prices[key]) > q.cfg.HistorySize {
		q.prices[key] = q.prices[key][1:]
	}
	result := CheckResult{
		Outlier:    q.isOutlier(key, bidF),
		SpreadBps:  spreadBps,
		ClockSkewMs: skewMs,
	}

	// Gap detection (metric only).
	if prev, ok := q.last[key]; ok {
		gapMs := t.ArrivedUnixMs - prev
		RecordGap(gapMs, int64(q.cfg.GapMaxSeconds*1000))
	}
	q.last[key] = t.ArrivedUnixMs
	q.mu.Unlock()

	return result
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
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

// trackSpread stores a spread observation for Z-score tracking (M10-BASE-F5).
func (q *Quality) trackSpread(key string, spreadBps float64) {
	q.mu.Lock()
	sk := "spread:" + key
	q.prices[sk] = append(q.prices[sk], spreadBps)
	if len(q.prices[sk]) > q.cfg.HistorySize {
		q.prices[sk] = q.prices[sk][1:]
	}
	q.mu.Unlock()
}

// trackTickRate stores a tick rate observation for Z-score tracking (M10-BASE-F4).
func (q *Quality) trackTickRate(key string, ratePerSec float64) {
	q.mu.Lock()
	tk := "tickrate:" + key
	q.prices[tk] = append(q.prices[tk], ratePerSec)
	if len(q.prices[tk]) > q.cfg.HistorySize {
		q.prices[tk] = q.prices[tk][1:]
	}
	q.mu.Unlock()
}

// SpreadZscore returns the Z-score of the current spread against the symbol's history.
func (q *Quality) SpreadZscore(key string, spreadBps float64) float64 {
	q.mu.Lock()
	sk := "spread:" + key
	vals := q.prices[sk]
	q.mu.Unlock()
	if len(vals) < 10 {
		return 0
	}
	return zscore(vals, spreadBps)
}

// TickRateZscore returns the Z-score of the current tick rate against the symbol's history.
func (q *Quality) TickRateZscore(key string, ratePerSec float64) float64 {
	q.mu.Lock()
	tk := "tickrate:" + key
	vals := q.prices[tk]
	q.mu.Unlock()
	if len(vals) < 10 {
		return 0
	}
	return zscore(vals, ratePerSec)
}

func zscore(history []float64, value float64) float64 {
	var sum, sumSq float64
	for _, v := range history {
		sum += v
		sumSq += v * v
	}
	n := float64(len(history))
	mean := sum / n
	variance := sumSq/n - mean*mean
	if variance <= 0 {
		// Zero variance: if value equals mean, z=0; otherwise treat as anomaly.
		if math.Abs(value-mean) < 1e-9 {
			return 0
		}
		return 10.0 // extreme deviation from constant distribution
	}
	return (value - mean) / math.Sqrt(variance)
}
