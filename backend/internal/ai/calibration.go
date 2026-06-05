// calibration.go: confidence calibration based on historical prediction accuracy.
//
// Adapted from QuantDinger ai_calibration.py:
//   - Track predictions per confidence bucket
//   - Validate against actual outcomes after a minimum age
//   - Recalibrate thresholds to maximize accuracy
//   - Apply calibrated thresholds to future predictions
//
// This closes the feedback loop: AI predicts → outcome observed → calibration adjusts.

package ai

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Repository ──

// CalibrationRepository persists prediction and calibration data.
type CalibrationRepository struct {
	db *pgxpool.Pool
}

func NewCalibrationRepository(db *pgxpool.Pool) *CalibrationRepository {
	return &CalibrationRepository{db: db}
}

// CalibrationBucket holds accuracy stats for one confidence bucket.
type CalibrationBucket struct {
	ConfidenceBucket    int
	TotalPredictions    int
	CorrectPredictions  int
	Accuracy            float64
	CalibratedThreshold float64
}

// AIPrediction represents a single AI decision record for later validation.
type AIPrediction struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Decision        string
	RawConfidence   float64
	PredictedAt     time.Time
	Symbol          string
	ActualReturnPct *float64
	WasCorrect      *bool
	ValidatedAt     *time.Time
}

// InsertPrediction records a new AI prediction for future validation.
func (r *CalibrationRepository) InsertPrediction(ctx context.Context, p *AIPrediction) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO ai_predictions (user_id, decision, raw_confidence, predicted_at, symbol)
		 VALUES ($1, $2, $3, $4, $5)`,
		p.UserID, p.Decision, p.RawConfidence, p.PredictedAt, p.Symbol)
	return err
}

// GetUnvalidatedPredictions returns predictions older than minAge that
// haven't been validated yet.
func (r *CalibrationRepository) GetUnvalidatedPredictions(ctx context.Context, minAge time.Duration, limit int) ([]AIPrediction, error) {
	cutoff := time.Now().Add(-minAge)
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, decision, raw_confidence, predicted_at,
		        COALESCE(symbol,''), actual_return_pct, was_correct, validated_at
		   FROM ai_predictions
		  WHERE validated_at IS NULL AND predicted_at < $1
		  ORDER BY predicted_at ASC
		  LIMIT $2`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AIPrediction
	for rows.Next() {
		var p AIPrediction
		if err := rows.Scan(&p.ID, &p.UserID, &p.Decision, &p.RawConfidence,
			&p.PredictedAt, &p.Symbol, &p.ActualReturnPct, &p.WasCorrect, &p.ValidatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ValidatePrediction marks a prediction as validated with its outcome.
func (r *CalibrationRepository) ValidatePrediction(ctx context.Context, id uuid.UUID, actualReturn float64, correct bool) error {
	now := time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE ai_predictions SET actual_return_pct=$1, was_correct=$2, validated_at=$3 WHERE id=$4`,
		actualReturn, correct, now, id)
	return err
}

// UpsertCalibration updates or inserts a calibration bucket.
func (r *CalibrationRepository) UpsertCalibration(ctx context.Context, userID uuid.UUID, b CalibrationBucket) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO ai_calibrations (user_id, confidence_bucket, total_predictions, correct_predictions, accuracy, calibrated_threshold, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,NOW())
		 ON CONFLICT (user_id, confidence_bucket)
		 DO UPDATE SET total_predictions=$3, correct_predictions=$4, accuracy=$5, calibrated_threshold=$6, updated_at=NOW()`,
		userID, b.ConfidenceBucket, b.TotalPredictions, b.CorrectPredictions, b.Accuracy, b.CalibratedThreshold)
	return err
}

// GetCalibrations returns all calibration buckets for a user.
func (r *CalibrationRepository) GetCalibrations(ctx context.Context, userID uuid.UUID) ([]CalibrationBucket, error) {
	rows, err := r.db.Query(ctx,
		`SELECT confidence_bucket, total_predictions, correct_predictions, accuracy,
		        COALESCE(calibrated_threshold, 0)
		   FROM ai_calibrations
		  WHERE user_id = $1
		  ORDER BY confidence_bucket`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CalibrationBucket
	for rows.Next() {
		var b CalibrationBucket
		if err := rows.Scan(&b.ConfidenceBucket, &b.TotalPredictions, &b.CorrectPredictions, &b.Accuracy, &b.CalibratedThreshold); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// GetPredictionStats returns total and correct counts per confidence bucket
// from validated predictions. Used during recalibration.
func (r *CalibrationRepository) GetPredictionStats(ctx context.Context, userID uuid.UUID) (map[int][2]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT CAST(FLOOR(raw_confidence/10)*10 AS INT) AS bucket,
		        COUNT(*) AS total,
		        COUNT(*) FILTER (WHERE was_correct) AS correct
		   FROM ai_predictions
		  WHERE user_id = $1 AND validated_at IS NOT NULL
		  GROUP BY bucket
		  ORDER BY bucket`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int][2]int)
	for rows.Next() {
		var bucket, total, correct int
		if err := rows.Scan(&bucket, &total, &correct); err != nil {
			return nil, err
		}
		out[bucket] = [2]int{total, correct}
	}
	return out, rows.Err()
}

// ── Calibration Logic ──

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

// Recalibrate recomputes thresholds per confidence bucket to maximize accuracy.
// Algorithm from QuantDinger ai_calibration.py:
//   For each bucket, compute accuracy. If accuracy < bucket/100,
//   raise the effective threshold (require higher confidence).
func (s *CalibrationService) Recalibrate(ctx context.Context, userID uuid.UUID) error {
	stats, err := s.repo.GetPredictionStats(ctx, userID)
	if err != nil {
		return err
	}
	for bucket := 10; bucket <= 100; bucket += 10 {
		counts := stats[bucket]
		total := counts[0]
		correct := counts[1]
		accuracy := 0.0
		if total > 0 {
			accuracy = float64(correct) / float64(total)
		}
		// Calibrated threshold: raise if accuracy < nominal confidence
		nominal := float64(bucket) / 100.0
		calibrated := accuracy
		if calibrated > 1.0 {
			calibrated = 1.0
		}
		// If accuracy is significantly below nominal, push threshold up
		if nominal-accuracy > 0.15 && total >= 5 {
			calibrated = nominal + (nominal-accuracy)*0.5
			if calibrated > 1.0 {
				calibrated = 1.0
			}
		}
		if err := s.repo.UpsertCalibration(ctx, userID, CalibrationBucket{
			ConfidenceBucket:    bucket,
			TotalPredictions:    total,
			CorrectPredictions:  correct,
			Accuracy:            math.Round(accuracy*1000) / 1000,
			CalibratedThreshold: math.Round(calibrated*1000) / 1000,
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
