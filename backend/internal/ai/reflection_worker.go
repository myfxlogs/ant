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
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/repository"
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
	store       repository.MarketDataStore
	log         *zap.Logger
	stopCh      chan struct{}
}

func NewReflectionWorker(cal *CalibrationService, store repository.MarketDataStore, log *zap.Logger) *ReflectionWorker {
	return &ReflectionWorker{
		calibration: cal,
		store:       store,
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

// fetchActualReturn queries the MarketDataStore for the 7-day price return after predicted_at.
func (w *ReflectionWorker) fetchActualReturn(ctx context.Context, symbol string, predictedAt time.Time) (float64, error) {
	return w.store.FetchActualReturn(ctx, symbol, predictedAt)
}

