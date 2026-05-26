package dataquality

import "time"

// Monitor combines gap, staleness, and cross-source detectors into a unified quality report.
type Monitor struct {
	GapDetector     *GapDetector
	StalenessScorer *StalenessScorer
	CrossValidator  *CrossSourceValidator
}

// NewMonitor creates a data quality monitor with default detectors.
func NewMonitor() *Monitor {
	return &Monitor{
		GapDetector:     NewGapDetector(60),
		StalenessScorer: NewStalenessScorer(300, 900),
		CrossValidator:  NewCrossSourceValidator(0.01),
	}
}

// Report builds a QualityReport from all detectors at the current time.
func (m *Monitor) Report(symbol string) *QualityReport {
	now := time.Now()
	report := &QualityReport{
		Symbol:      symbol,
		Timestamp:   now,
		Warnings:    make([]string, 0),
	}

	// Gap stats
	report.HasGaps = m.GapDetector.HasGaps()
	if report.HasGaps {
		gapCount, maxSec, totalSec := m.GapDetector.Stats()
		report.GapCount = gapCount
		report.MaxGapSeconds = maxSec
		report.TotalGapSeconds = totalSec
	}

	// Staleness
	report.SecondsSinceLastTick = m.StalenessScorer.SinceLastTick(now)
	report.StalenessScore = m.StalenessScorer.Score(now)
	report.IsStale = m.StalenessScorer.IsStale(now)
	report.IsDead = m.StalenessScorer.IsDead(now)

	// Cross-source
	valid, maxDev, srcCount := m.CrossValidator.Validate()
	report.CrossSourceValid = valid
	report.MaxSourceDeviation = maxDev
	report.SourceCount = srcCount

	// Build warnings
	if report.HasGaps {
		report.Warnings = append(report.Warnings, "data gaps detected")
	}
	if report.IsStale {
		report.Warnings = append(report.Warnings, "data is stale")
	}
	if report.IsDead {
		report.Warnings = append(report.Warnings, "data feed appears dead")
	}
	if !valid && srcCount >= 2 {
		report.Warnings = append(report.Warnings, "cross-source price deviation exceeds threshold")
	}

	// Composite quality score
	score := 1.0
	if report.IsDead {
		score -= 0.5
	} else if report.IsStale {
		score -= 0.25
	}
	if report.HasGaps {
		score -= 0.15
	}
	if !valid && srcCount >= 2 {
		score -= 0.10
	}
	if score < 0 {
		score = 0
	}
	report.QualityScore = score

	// Status
	switch {
	case report.IsDead || score < 0.3:
		report.Status = "critical"
	case !valid || report.IsStale || report.HasGaps:
		report.Status = "degraded"
	default:
		report.Status = "healthy"
	}

	return report
}
