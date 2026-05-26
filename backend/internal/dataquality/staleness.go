package dataquality

import "time"

// StalenessScorer computes freshness scores based on time since last update.
type StalenessScorer struct {
	StaleThresholdSec    float64 // > this = stale (default 300 = 5 min)
	DeadThresholdSec     float64 // > this = dead (default 900 = 15 min)
	lastTickAt           time.Time
	updateCount          int64
}

// NewStalenessScorer creates a staleness scorer.
func NewStalenessScorer(staleSec, deadSec float64) *StalenessScorer {
	if staleSec <= 0 {
		staleSec = 300 // 5 min
	}
	if deadSec <= 0 {
		deadSec = 900 // 15 min
	}
	return &StalenessScorer{
		StaleThresholdSec: staleSec,
		DeadThresholdSec:  deadSec,
	}
}

// Observe records a new tick time.
func (s *StalenessScorer) Observe(t time.Time) {
	s.lastTickAt = t
	s.updateCount++
}

// Score computes the staleness score at the given time.
// 1.0 = fresh (< stale threshold), 0.0 = dead (> dead threshold).
func (s *StalenessScorer) Score(now time.Time) float64 {
	if s.lastTickAt.IsZero() {
		return 0
	}
	elapsed := now.Sub(s.lastTickAt).Seconds()
	if elapsed <= 0 {
		return 1.0
	}
	if elapsed >= s.DeadThresholdSec {
		return 0
	}
	if elapsed >= s.StaleThresholdSec {
		// Linear decay from 1.0 to 0.0 between stale and dead
		return 1.0 - (elapsed-s.StaleThresholdSec)/(s.DeadThresholdSec-s.StaleThresholdSec)
	}
	return 1.0
}

// IsStale returns true if the data has exceeded the stale threshold.
func (s *StalenessScorer) IsStale(now time.Time) bool {
	return s.Score(now) < 1.0
}

// IsDead returns true if the data has exceeded the dead threshold.
func (s *StalenessScorer) IsDead(now time.Time) bool {
	if s.lastTickAt.IsZero() {
		return true
	}
	return now.Sub(s.lastTickAt).Seconds() >= s.DeadThresholdSec
}

// SinceLastTick returns seconds since the last tick.
func (s *StalenessScorer) SinceLastTick(now time.Time) float64 {
	if s.lastTickAt.IsZero() {
		return -1
	}
	return now.Sub(s.lastTickAt).Seconds()
}
