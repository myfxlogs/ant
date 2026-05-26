// Package dataquality provides data quality monitoring for market data feeds (M10-BASE-F4/F5/F6/F7).
//
// Monitors four dimensions:
//   - GapDetector: detects missing data intervals per symbol
//   - StalenessScorer: computes freshness scores from last update timestamps
//   - CrossSourceValidator: compares prices across multiple data sources
//   - QualityReport: aggregates all dimensions into a per-symbol quality report

package dataquality

import "time"

// QualityReport summarizes data quality for one symbol.
type QualityReport struct {
	Symbol      string    `json:"symbol"`
	Timestamp   time.Time `json:"timestamp"`

	// Gap metrics
	HasGaps        bool    `json:"has_gaps"`
	GapCount       int     `json:"gap_count"`        // gaps in current window
	MaxGapSeconds  float64 `json:"max_gap_seconds"`  // longest gap
	TotalGapSeconds float64 `json:"total_gap_seconds"` // cumulative gap time

	// Staleness
	SecondsSinceLastTick float64 `json:"seconds_since_last_tick"`
	StalenessScore       float64 `json:"staleness_score"`   // 0-1, 1=fresh
	IsStale              bool    `json:"is_stale"`          // > threshold
	IsDead               bool    `json:"is_dead"`           // > critical threshold

	// Cross-source
	CrossSourceValid   bool    `json:"cross_source_valid"`
	MaxSourceDeviation float64 `json:"max_source_deviation"` // max price diff between sources
	SourceCount        int     `json:"source_count"`

	// Overall
	QualityScore   float64  `json:"quality_score"`    // 0-1 composite
	Status         string   `json:"status"`           // "healthy", "degraded", "critical"
	Warnings       []string `json:"warnings"`
}

// DefaultQualityReport returns a neutral quality report.
func DefaultQualityReport(symbol string) *QualityReport {
	return &QualityReport{
		Symbol:      symbol,
		Timestamp:   time.Now(),
		Status:      "healthy",
		QualityScore: 1.0,
	}
}
