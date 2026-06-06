// calibration.go: confidence calibration logic based on historical prediction accuracy.
//
// Algorithm from QuantDinger ai_calibration.py:
//   1. Record predictions with raw confidence scores
//   2. Validate against actual outcomes after a minimum age
//   3. Recalibrate thresholds via grid search to maximize accuracy
//   4. Apply calibrated thresholds to future predictions
//
// This closes the feedback loop: AI predicts → outcome observed → calibration adjusts.

package ai

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
)

// CalibrationService manages confidence calibration lifecycle.
type CalibrationService struct {
	repo *CalibrationRepository
}

func NewCalibrationService(repo *CalibrationRepository) *CalibrationService {
	return &CalibrationService{repo: repo}
}

// RecordPrediction stores a new AI prediction for future validation.
func (s *CalibrationService) RecordPrediction(ctx context.Context, userID uuid.UUID, decision string, confidence float64, symbol string) error {
	return s.repo.InsertPrediction(ctx, &AIPrediction{
		ID:            uuid.New(),
		UserID:        userID,
		Decision:      decision,
		RawConfidence: confidence,
		PredictedAt:   time.Now(),
		Symbol:        symbol,
	})
}

// Recalibrate recomputes thresholds per confidence bucket via grid search
// over candidate thresholds, maximizing accuracy while maintaining coverage.
// Algorithm from QuantDinger ai_calibration.py:
//   1. Search candidate thresholds (grid: 0.10–0.30 step 0.02)
//   2. For each threshold, compute accuracy and coverage (BUY+SELL ratio)
//   3. Pick threshold maximizing accuracy, tie-break by coverage
func (s *CalibrationService) Recalibrate(ctx context.Context, userID uuid.UUID) error {
	stats, err := s.repo.GetPredictionStats(ctx, userID)
	if err != nil {
		return err
	}

	// Build flat list of (confidence, was_correct) pairs from validated predictions.
	type pred struct {
		confidence float64
		correct    bool
	}
	var all []pred
	for bucket, counts := range stats {
		total := counts[0]
		correct := counts[1]
		if total == 0 {
			continue
		}
		// Approximate: distribute predictions evenly within the bucket.
		conf := float64(bucket)/100.0 + 0.05
		for i := 0; i < correct; i++ {
			all = append(all, pred{confidence: conf, correct: true})
		}
		for i := 0; i < total-correct; i++ {
			all = append(all, pred{confidence: conf, correct: false})
		}
	}
	if len(all) < 10 {
		return nil // not enough data
	}

	// Grid search over candidate absolute thresholds.
	candidates := []float64{0.10, 0.12, 0.14, 0.16, 0.18, 0.20, 0.22, 0.25, 0.30}
	bestThreshold := 0.20
	bestAccuracy := 0.0
	bestCoverage := 0

	for _, thresh := range candidates {
		correctCnt := 0
		actionCnt := 0 // BUY + SELL count (non-HOLD)
		for _, p := range all {
			if p.confidence >= thresh {
				actionCnt++
				if p.correct {
					correctCnt++
				}
			}
		}
		if actionCnt == 0 {
			continue
		}
		acc := float64(correctCnt) / float64(actionCnt)
		coverage := actionCnt

		// Maximize accuracy; if tied, prefer higher coverage (avoids always-HOLD).
		if acc > bestAccuracy || (math.Abs(acc-bestAccuracy) < 0.01 && coverage > bestCoverage) {
			bestAccuracy = acc
			bestCoverage = coverage
			bestThreshold = thresh
		}
	}

	// Persist calibration results per bucket.
	for bucket := 10; bucket <= 100; bucket += 10 {
		counts := stats[bucket]
		total := counts[0]
		correct := counts[1]
		accuracy := 0.0
		if total > 0 {
			accuracy = float64(correct) / float64(total)
		}
		if err := s.repo.UpsertCalibration(ctx, userID, CalibrationBucket{
			ConfidenceBucket:    bucket,
			TotalPredictions:    total,
			CorrectPredictions:  correct,
			Accuracy:            math.Round(accuracy*1000) / 1000,
			CalibratedThreshold: math.Round(bestThreshold*1000) / 1000,
		}); err != nil {
			return err
		}
	}
	return nil
}

// GetCalibratedConfidence returns the calibrated confidence for a raw value.
// Falls back to raw confidence if no calibration data exists.
func (s *CalibrationService) GetCalibratedConfidence(ctx context.Context, userID uuid.UUID, rawConfidence float64) float64 {
	buckets, err := s.repo.GetCalibrations(ctx, userID)
	if err != nil || len(buckets) == 0 {
		return rawConfidence
	}
	// Find the nearest bucket and apply its calibrated threshold.
	bucket := int(math.Floor(rawConfidence/10)) * 10
	if bucket < 10 {
		bucket = 10
	}
	if bucket > 100 {
		bucket = 100
	}
	for _, b := range buckets {
		if b.ConfidenceBucket == bucket && b.TotalPredictions >= 5 {
			return b.CalibratedThreshold
		}
	}
	return rawConfidence
}

// GetDecisionThresholds returns calibrated BUY/SELL/HOLD thresholds for a user.
// Falls back to defaults (0.02 / -0.02 / 0.05) if no calibration data exists.
func (s *CalibrationService) GetDecisionThresholds(ctx context.Context, userID uuid.UUID) (buy, sell, hold float64) {
	buy, sell, hold = 0.02, -0.02, 0.05
	buckets, err := s.repo.GetCalibrations(ctx, userID)
	if err != nil || len(buckets) == 0 {
		return
	}
	// Use calibration from mid-range buckets (50-70) which represent
	// the typical confidence level for actionable decisions.
	var sumAcc float64
	var count int
	for _, b := range buckets {
		if b.ConfidenceBucket >= 50 && b.ConfidenceBucket <= 70 && b.TotalPredictions >= 5 {
			sumAcc += b.Accuracy
			count++
		}
	}
	if count < 2 {
		return
	}
	avgAcc := sumAcc / float64(count)
	// Tighten thresholds proportionally to calibration accuracy.
	// Higher accuracy → keep defaults. Lower accuracy → widen thresholds.
	scale := 1.0 + (1.0-avgAcc)*0.5
	if scale > 2.0 {
		scale = 2.0
	}
	buy = 0.02 * scale
	sell = -0.02 * scale
	hold = 0.05 * scale
	return
}

// ValidateOutcome checks whether a prediction was correct given actual return.
// Uses calibrated thresholds when available, falling back to defaults:
// BUY correct if return > +2%, SELL if < -2%, HOLD if |return| <= 5%.
// When calibrated thresholds exist, they replace the hardcoded values.
func ValidateOutcome(decision string, actualReturn float64, calibratedThresholds ...float64) bool {
	buyThreshold := 0.02
	sellThreshold := -0.02
	holdThreshold := 0.05
	if len(calibratedThresholds) >= 3 {
		buyThreshold = calibratedThresholds[0]
		sellThreshold = calibratedThresholds[1]
		holdThreshold = calibratedThresholds[2]
	}
	switch decision {
	case "BUY":
		return actualReturn > buyThreshold
	case "SELL":
		return actualReturn < sellThreshold
	case "HOLD":
		return math.Abs(actualReturn) <= holdThreshold
	default:
		return false
	}
}
