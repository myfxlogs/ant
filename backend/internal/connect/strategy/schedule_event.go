package strategy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/model"
)

// StartSchedule loads a schedule by ID and starts it.
// For event-type schedules, launches a persistent streaming session.
// For timer-type schedules, notifies the timer loop to recompute.
// This is the symmetric counterpart to StopSchedule.
func (e *ScheduleEngine) StartSchedule(ctx context.Context, id uuid.UUID) error {
	schedule, err := e.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("start schedule: %w", err)
	}
	if !schedule.IsActive {
		return fmt.Errorf("start schedule: schedule %s is not active", id)
	}
	if e.isRunning(id) {
		return nil // already running, no-op
	}

	if schedule.ScheduleType == model.ScheduleTypeEvent {
		return e.launchEventSession(ctx, schedule)
	}

	// Timer-type: just notify the timer loop to pick up the next_run_at.
	e.Notify()
	return nil
}

// launchEventSession starts a persistent RunLiveStrategy streaming session
// for an event-type schedule (kline_close / hf_quote).
//
// Push-first architecture: instead of polling the DB per bar, the strategy
// subscribes to the bar stream and reacts to each closed bar in a long-lived
// goroutine. This is the correct execution model for event-driven schedules.
//
// Gates applied before launch (in order):
//  1. Entitlement gate — verifies active subscription/trial/ownership
//  2. Quota gate — enforces live strategy limit
//  3. Account conflict — SessionRegistry rejects if account already has a running session
//
// Revoked entitlement during runtime is handled by per-bar EntitlementCheck
// in the live event loop (task 4), which self-terminates the session.
func (e *ScheduleEngine) launchEventSession(ctx context.Context, schedule *model.StrategySchedule) error {
	cfg, handle, runCtx, err := e.buildLiveRun(ctx, schedule, "launchEventSession")
	if err != nil {
		return err
	}

	// Pre-register session before launching.
	// ARCH-4: Multiple sessions per account are now allowed — position
	// attribution is via Magic Numbers, not session exclusivity.
	if e.runner != nil && e.runner.sessionRegistry != nil && cfg.RunID != uuid.Nil {
		uid, _ := uuid.Parse(cfg.UserID)
		sess := e.runner.sessionRegistry.Register(cfg.RunID, uid, cfg.AccountID, cfg.Symbol, cfg.Timeframe, cfg.Mode, schedule.ID, cfg.StrategyID, handle.cancel)
		if sess != nil {
			cfg.PreRegisteredSession = sess
		}
	}

	go func(ctx context.Context) { e.runOne(ctx, schedule, cfg, handle) }(runCtx)

	e.log.Info("launchEventSession: started",
		zap.String("schedule_id", schedule.ID.String()),
		zap.String("symbol", schedule.Symbol),
		zap.String("timeframe", schedule.Timeframe))
	return nil
}
