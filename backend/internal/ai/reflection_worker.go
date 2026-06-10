// reflection_worker.go: background validation loop for AI predictions.
//
// Adapted from QuantDinger reflection.py:
//   - Periodically validates unvalidated predictions against actual outcomes
//   - BUY correct if return > +2%, SELL if < -2%, HOLD if |return| <= 5%
//   - Triggers recalibration after each validation batch
//
// This closes the feedback loop so AI confidence improves over time.

package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	reflectionInterval = 24 * time.Hour
	reflectionMinAge   = 7 * 24 * time.Hour
	reflectionBatch    = 100
)

// ReflectionWorker periodically validates past AI predictions and
// triggers confidence recalibration.
type ReflectionWorker struct {
	calibration *CalibrationService
	ch          clickhouse.Conn
	log         *zap.Logger
	stopCh      chan struct{}
}

func NewReflectionWorker(cal *CalibrationService, ch clickhouse.Conn, log *zap.Logger) *ReflectionWorker {
	return &ReflectionWorker{
		calibration: cal,
		ch:          ch,
		log:         log,
		stopCh:      make(chan struct{}),
	}
}

func (w *ReflectionWorker) Start(ctx context.Context) {
	go func() {
		// Run once at startup, then periodically.
		w.run(ctx)
		ticker := time.NewTicker(reflectionInterval)
		defer ticker.Stop()
		for {
			select {
			case <-w.stopCh:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.run(ctx)
			}
		}
	}()
}

func (w *ReflectionWorker) Stop() { close(w.stopCh) }

func (w *ReflectionWorker) run(ctx context.Context) {
	// 1. Fetch unvalidated predictions older than min age.
	preds, err := w.calibration.repo.GetUnvalidatedPredictions(ctx, reflectionMinAge, reflectionBatch)
	if err != nil {
		w.log.Warn("reflection: fetch unvalidated predictions", zap.Error(err))
		return
	}
	if len(preds) == 0 {
		return
	}

	// 2. Validate each prediction against actual price movement.
	validated := 0
	byUser := make(map[string]struct{})
	for _, p := range preds {
		if p.Symbol == "" {
			continue
		}
		actualReturn, err := w.fetchActualReturn(ctx, p.Symbol, p.PredictedAt)
		if err != nil {
			w.log.Debug("reflection: skip prediction (no price data)",
				zap.String("symbol", p.Symbol),
				zap.Error(err))
			continue
		}
		buyT, sellT, holdT := w.calibration.GetDecisionThresholds(ctx, p.UserID)
		correct := ValidateOutcome(p.Decision, actualReturn, buyT, sellT, holdT)
		if err := w.calibration.repo.ValidatePrediction(ctx, p.ID, actualReturn, correct); err != nil {
			w.log.Warn("reflection: validate prediction", zap.Error(err))
			continue
		}
		validated++
		byUser[p.UserID.String()] = struct{}{}
	}

	// 3. Recalibrate affected users.
	for uid := range byUser {
		userID, err := uuid.Parse(uid)
		if err != nil {
			continue
		}
		if err := w.calibration.Recalibrate(ctx, userID); err != nil {
			w.log.Warn("reflection: recalibrate", zap.String("user", uid), zap.Error(err))
		}
	}

	if validated > 0 {
		w.log.Info("reflection: validated and recalibrated",
			zap.Int("predictions", validated),
			zap.Int("users", len(byUser)))
	}
}

// fetchActualReturn queries ClickHouse for the 7-day price return after predicted_at.
func (w *ReflectionWorker) fetchActualReturn(ctx context.Context, symbol string, predictedAt time.Time) (float64, error) {
	start := predictedAt
	end := predictedAt.Add(7 * 24 * time.Hour)

	var openPrice, closePrice float64
	err := w.ch.QueryRow(ctx,
		`SELECT
			COALESCE((SELECT open FROM md_bars
			  WHERE canonical = $1 AND period = '1d'
			    AND open_ts_unix_ms >= $2 AND open_ts_unix_ms < $3
			  ORDER BY open_ts_unix_ms ASC LIMIT 1), 0),
			COALESCE((SELECT close FROM md_bars
			  WHERE canonical = $1 AND period = '1d'
			    AND close_ts_unix_ms <= $4
			  ORDER BY close_ts_unix_ms DESC LIMIT 1), 0)`,
		symbol,
		start.UnixMilli(), end.UnixMilli(),
		end.UnixMilli(),
	).Scan(&openPrice, &closePrice)
	if err != nil {
		return 0, err
	}
	if openPrice <= 0 {
		return 0, fmt.Errorf("no price data for %s", symbol)
	}
	return (closePrice - openPrice) / openPrice, nil
}

