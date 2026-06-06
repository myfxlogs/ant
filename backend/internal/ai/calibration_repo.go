// calibration_repo.go: persistence layer for AI prediction calibration.
//
// Adapted from QuantDinger ai_calibration.py:
//   - Track predictions per confidence bucket
//   - Validate against actual outcomes after a minimum age

package ai

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
