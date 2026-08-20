// schedule_execute.go — Execute loop, dispatch, runOne, reconcile, recomputeTimer.
// Split from schedule_engine.go for file-lines compliance (SCHEDULE-HOTLOOP-1).
//
// Core invariant (SCHEDULE-HOTLOOP-1): a due timer occurrence is a fact that must
// be persistently consumed (next_run_at advanced to > now) BEFORE any skip/deny/
// dispatch branch. This prevents the 0-delay hot loop where autoTrade=false or
// already-running schedules never advance next_run_at → GetEarliestNextRunAt
// returns a past time → timer delay=0 → hundreds of due-log lines per second.

package strategy

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/repository"
)

// reconcileOnStartup ensures every active timer schedule has a next_run_at and
// cleans dirty event-schedule next_run_at values (SCHEDULE-HOTLOOP-1 event invariant).
func (e *ScheduleEngine) reconcileOnStartup(ctx context.Context) {
	// Event invariant: event schedules must have NULL next_run_at. Clear any
	// stale values left by bugs/migrations so they never enter the timer loop.
	if cleaned, err := e.repo.ClearEventNextRunAt(ctx); err != nil {
		e.log.Warn("reconcile: clear event next_run_at failed", zap.Error(err))
	} else if cleaned > 0 {
		e.log.Info("reconcile: cleared stale event next_run_at", zap.Int("count", cleaned))
	}

	schedules, err := e.repo.GetActiveSchedules(ctx)
	if err != nil {
		e.log.Warn("reconcile: failed to load active schedules", zap.Error(err))
		return
	}
	var updated int
	var eventStarted int
	for _, s := range schedules {
		if s.ScheduleType == model.ScheduleTypeEvent {
			// Event-type: launch persistent streaming session.
			if err := e.launchEventSession(ctx, s); err != nil {
				e.log.Warn("reconcile: failed to start event session",
					zap.String("schedule_id", s.ID.String()), zap.Error(err))
			} else {
				eventStarted++
			}
			continue
		}
		// Timer-type: ensure next_run_at is set.
		if s.NextRunAt != nil && !s.NextRunAt.IsZero() {
			continue
		}
		next, err := model.ComputeNextRunAtFromConfigAt(s.ScheduleType, s.ScheduleConfig, e.nowTime())
		if err != nil || next.IsZero() {
			reason := "compute next_run_at returned zero"
			if err != nil {
				reason = err.Error()
			}
			e.log.Warn("reconcile: invalid config, quarantining schedule",
				zap.String("schedule_id", s.ID.String()), zap.String("reason", reason))
			_ = e.repo.UpdateLastRun(ctx, s.ID, fmt.Errorf("invalid schedule config: %s", reason))
			_ = e.repo.ClearNextRunAt(ctx, s.ID)
			continue
		}
		if err := e.repo.UpdateNextRunAt(ctx, s.ID, next); err != nil {
			e.log.Warn("reconcile: update next_run_at failed",
				zap.String("schedule_id", s.ID.String()), zap.Error(err))
			continue
		}
		updated++
	}
	if updated > 0 {
		e.log.Info("reconciled missing next_run_at", zap.Int("count", updated))
	}
	if eventStarted > 0 {
		e.log.Info("reconciled event-type sessions", zap.Int("count", eventStarted))
	}
}

// recomputeTimer finds the earliest next_run_at among timer schedules and sets
// a time.Timer for that moment. Returns idleDelay timer if no schedule is due.
func (e *ScheduleEngine) recomputeTimer(ctx context.Context) (*time.Timer, time.Time, error) {
	earliest, err := e.repo.GetEarliestNextRunAt(ctx)
	if err != nil {
		return time.NewTimer(backoffDelay), time.Time{}, err
	}
	if earliest.IsZero() {
		return time.NewTimer(idleDelay), time.Time{}, nil
	}
	d := earliest.Sub(e.nowTime())
	if d < 0 {
		d = 0
	}
	return time.NewTimer(d), earliest, nil
}

// executeLoop processes all due timer schedules. For each due schedule, it
// advances next_run_at to strictly > now BEFORE any skip/deny/dispatch check
// (SCHEDULE-HOTLOOP-1). Returns an aggregated error if DB operations fail;
// the caller (Start) enters backoff on error.
func (e *ScheduleEngine) executeLoop(ctx context.Context) error {
	now := e.nowTime()
	due, err := e.repo.GetDueSchedules(ctx, now)
	if err != nil {
		e.log.Error("get due schedules failed", zap.Error(err))
		return fmt.Errorf("get due schedules: %w", err)
	}
	if len(due) == 0 {
		return nil
	}
	e.log.Info("due schedules found", zap.Int("count", len(due)))

	var firstErr error
	for _, s := range due {
		// Pre-consume: advance next_run_at to strictly > now before any branch.
		// This is the SCHEDULE-HOTLOOP-1 fix — without it, autoTrade=false or
		// already-running schedules never advance → 0-delay hot loop.
		next, computeErr := model.ComputeNextRunAtFromConfigAt(s.ScheduleType, s.ScheduleConfig, now)
		if computeErr != nil || next.IsZero() {
			reason := "compute next_run_at returned zero"
			if computeErr != nil {
				reason = computeErr.Error()
			}
			e.log.Warn("executeLoop: invalid config, quarantining schedule",
				zap.String("schedule_id", s.ID.String()), zap.String("reason", reason))
			_ = e.repo.UpdateLastRun(ctx, s.ID, fmt.Errorf("invalid schedule config: %s", reason))
			if clearErr := e.repo.ClearNextRunAt(ctx, s.ID); clearErr != nil {
				e.log.Error("clear next_run_at failed",
					zap.String("schedule_id", s.ID.String()), zap.Error(clearErr))
				if firstErr == nil {
					firstErr = clearErr
				}
			}
			continue
		}
		if updateErr := e.repo.UpdateNextRunAt(ctx, s.ID, next); updateErr != nil {
			e.log.Error("advance next_run_at failed, not dispatching",
				zap.String("schedule_id", s.ID.String()), zap.Error(updateErr))
			if firstErr == nil {
				firstErr = updateErr
			}
			continue // do not dispatch — occurrence not consumed
		}

		// Occurrence consumed. Now decide whether to dispatch.
		if e.isRunning(s.ID) {
			continue
		}
		if !e.isAutoTradeEnabled(s.UserID) {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		e.dispatch(ctx, s)
	}
	return firstErr
}

// isAutoTradeEnabled checks the per-user autotrade setting with a TTL cache
// to avoid querying PG on every schedule dispatch cycle.
func (e *ScheduleEngine) isAutoTradeEnabled(userID uuid.UUID) bool {
	if e.autoTradeEnabled == nil {
		return true
	}
	now := e.nowTime()
	e.autoTradeCacheMu.Lock()
	if entry, ok := e.autoTradeCache[userID]; ok && now.Before(entry.expireAt) {
		e.autoTradeCacheMu.Unlock()
		return entry.enabled
	}
	e.autoTradeCacheMu.Unlock()

	enabled := e.autoTradeEnabled(userID)

	e.autoTradeCacheMu.Lock()
	e.autoTradeCache[userID] = autoTradeEntry{enabled: enabled, expireAt: now.Add(autoTradeCacheTTL)}
	e.autoTradeCacheMu.Unlock()
	return enabled
}

func (e *ScheduleEngine) dispatch(ctx context.Context, schedule *model.StrategySchedule) {
	cfg, handle, runCtx, err := e.buildLiveRun(ctx, schedule, "dispatch")
	if err != nil {
		return
	}
	go func(ctx context.Context) { e.runOne(ctx, schedule, cfg, handle) }(runCtx)

	e.log.Info("dispatched", zap.String("schedule_id", schedule.ID.String()),
		zap.String("symbol", schedule.Symbol), zap.String("timeframe", schedule.Timeframe))
}

// buildLiveRun runs the four pre-launch gates (entitlement/quota/bound/template),
// assembles LiveStrategyConfig, pre-creates the run record, and registers the
// run handle. Denied gates call repo.UpdateLastRun and return an error; callers
// decide whether to propagate (launch) or swallow (dispatch).
// Shared by dispatch and launchEventSession so a new gate can never be added
// to one path and missed on the other (LEAKAGE-1 lesson).
func (e *ScheduleEngine) buildLiveRun(ctx context.Context, schedule *model.StrategySchedule, logPrefix string) (LiveStrategyConfig, *runHandle, context.Context, error) {
	// Entitlement gate.
	if e.entitlementCheck != nil {
		if !e.entitlementCheck(ctx, schedule.UserID.String(), schedule.TemplateID.String()) {
			e.log.Warn(logPrefix+": entitlement denied",
				zap.String("schedule_id", schedule.ID.String()),
				zap.String("user_id", schedule.UserID.String()))
			err := fmt.Errorf("unauthorized: no active entitlement")
			_ = e.repo.UpdateLastRun(ctx, schedule.ID, err)
			return LiveStrategyConfig{}, nil, nil, err
		}
	}

	// Quota gate.
	if e.runner != nil {
		if err := e.runner.checkStrategyQuota(ctx, schedule.UserID, modeLive); err != nil {
			e.log.Warn(logPrefix+": quota exceeded",
				zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
			_ = e.repo.UpdateLastRun(ctx, schedule.ID, err)
			return LiveStrategyConfig{}, nil, nil, err
		}
	}

	// Bound account gate (LEAKAGE-1).
	if e.runner != nil {
		if err := e.runner.checkBoundAccount(ctx, schedule.UserID, schedule.AccountID); err != nil {
			e.log.Warn(logPrefix+": bound account check failed",
				zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
			_ = e.repo.UpdateLastRun(ctx, schedule.ID, err)
			return LiveStrategyConfig{}, nil, nil, err
		}
	}

	// Load and validate template.
	tpl, err := e.templateReader.GetTemplate(ctx, schedule.TemplateID, schedule.UserID)
	if err != nil || tpl == nil || tpl.Code == "" {
		reason := "template code is empty"
		if err != nil {
			reason = err.Error()
		}
		e.log.Error(logPrefix+": invalid template",
			zap.String("schedule_id", schedule.ID.String()),
			zap.String("template_id", schedule.TemplateID.String()), zap.Error(err))
		err := fmt.Errorf("%s: %s", logPrefix, reason)
		_ = e.repo.UpdateLastRun(ctx, schedule.ID, err)
		return LiveStrategyConfig{}, nil, nil, err
	}

	strParams, _ := schedule.GetParameters()
	// lifecycleCtx = engine lifetime; runs survive handler ctx cancellation. nil guard for Start() race.
	runParent := e.lifecycleCtx
	if runParent == nil {
		runParent = context.Background()
	}
	runCtx, cancel := context.WithCancel(runParent)
	handle := &runHandle{cancel: cancel}
	handle.wg.Add(1)
	e.mu.Lock()
	e.activeRuns[schedule.ID] = handle
	e.mu.Unlock()

	// Per-bar entitlement revalidation for marketplace strategies.
	var entCheck func(ctx context.Context) bool
	if e.entitlementCheck != nil {
		isOwner := tpl.UserID != nil && *tpl.UserID == schedule.UserID
		if !isOwner {
			entCheck = func(ctx context.Context) bool {
				return e.entitlementCheck(ctx, schedule.UserID.String(), schedule.TemplateID.String())
			}
		}
	}

	cfg := LiveStrategyConfig{
		AccountID:        schedule.AccountID.String(),
		UserID:           schedule.UserID.String(),
		Symbol:           schedule.Symbol,
		Timeframe:        schedule.Timeframe,
		Code:             tpl.Code,
		Mode:             modeLive,
		Params:           strParams,
		ScheduleID:       schedule.ID,
		EntitlementCheck: entCheck,
		TickSeq:          new(atomic.Int64),
	}

	// Pre-create run record.
	if e.runner != nil && e.runner.runRepo != nil {
		uid, _ := uuid.Parse(cfg.UserID)
		run := &repository.StrategyRun{
			UserID:       uid,
			AccountID:    cfg.AccountID,
			Symbol:       cfg.Symbol,
			Timeframe:    cfg.Timeframe,
			Mode:         cfg.Mode,
			StrategyCode: cfg.Code,
			Status:       "running",
		}
		if err := e.runner.runRepo.Create(ctx, run); err != nil {
			e.log.Warn(logPrefix+": failed to create run record", zap.Error(err))
		} else {
			cfg.RunID = run.ID
		}
	}

	return cfg, handle, runCtx, nil
}

// runOne executes a single strategy run. It does NOT advance next_run_at —
// that was already done by executeLoop before dispatch (SCHEDULE-HOTLOOP-1).
// runOne only: runs the strategy, records last_run, clears activeRuns, Notify.
// Live runs can block forever; next_run_at must not depend on run completion.
func (e *ScheduleEngine) runOne(ctx context.Context, schedule *model.StrategySchedule, cfg LiveStrategyConfig, handle *runHandle) {
	defer handle.wg.Done()
	defer func() {
		e.mu.Lock()
		delete(e.activeRuns, schedule.ID)
		e.mu.Unlock()
		handle.cancel()
	}()

	runErr := fmt.Errorf("strategy runner not configured")
	if e.runner != nil {
		runErr = e.runner.RunLiveStrategy(ctx, cfg)
	}

	recordCtx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	if err := e.repo.UpdateLastRun(recordCtx, schedule.ID, runErr); err != nil {
		e.log.Error("update last_run failed", zap.Error(err))
	}

	e.Notify()

	if runErr != nil {
		e.log.Warn("run completed with error",
			zap.String("schedule_id", schedule.ID.String()), zap.Error(runErr))
	} else {
		e.log.Info("run completed", zap.String("schedule_id", schedule.ID.String()))
	}
}
