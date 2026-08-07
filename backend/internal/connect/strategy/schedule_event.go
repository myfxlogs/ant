package strategy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/repository"
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
	// Entitlement gate (task 3).
	if e.entitlementCheck != nil {
		if !e.entitlementCheck(ctx, schedule.UserID.String(), schedule.TemplateID.String()) {
			e.log.Warn("launchEventSession: entitlement denied",
				zap.String("schedule_id", schedule.ID.String()),
				zap.String("user_id", schedule.UserID.String()))
			_ = e.repo.UpdateLastRun(ctx, schedule.ID, fmt.Errorf("unauthorized: no active entitlement"))
			return fmt.Errorf("unauthorized: no active entitlement")
		}
	}

	// Quota gate (task 5).
	if e.runner != nil {
		if err := e.runner.checkStrategyQuota(ctx, schedule.UserID, "live"); err != nil {
			e.log.Warn("launchEventSession: quota exceeded",
				zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
			_ = e.repo.UpdateLastRun(ctx, schedule.ID, err)
			return err
		}
	}

	// Load template code from strategy_templates (not ai_strategy_templates).
	tpl, err := e.templateReader.GetTemplate(ctx, schedule.TemplateID, schedule.UserID)
	if err != nil || tpl == nil || tpl.Code == "" {
		reason := "template code is empty"
		if err != nil {
			reason = err.Error()
		}
		e.log.Error("launchEventSession: invalid template",
			zap.String("schedule_id", schedule.ID.String()),
			zap.String("template_id", schedule.TemplateID.String()), zap.Error(err))
		_ = e.repo.UpdateLastRun(ctx, schedule.ID, fmt.Errorf("launch: %s", reason))
		return fmt.Errorf("launch: %s", reason)
	}

	strParams, _ := schedule.GetParameters()

	runCtx, cancel := context.WithCancel(ctx)
	handle := &runHandle{cancel: cancel}
	handle.wg.Add(1)

	e.mu.Lock()
	e.activeRuns[schedule.ID] = handle
	e.mu.Unlock()

	// Per-bar entitlement revalidation for marketplace strategies (task 4).
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
		Mode:             "live",
		Params:           strParams,
		EntitlementCheck: entCheck,
	}

	// Pre-create run record (RunLiveStrategy requires RunID to be set).
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
			e.log.Warn("launchEventSession: failed to create run record", zap.Error(err))
		} else {
			cfg.RunID = run.ID
		}
	}

	// Pre-register session to detect account conflicts before launching.
	// If Register returns nil, another session is already running for this account.
	// We do NOT replace it — first-runner-wins, never silently kill a session
	// that may hold open positions (spec §七-Q2).
	if e.runner != nil && e.runner.sessionRegistry != nil && cfg.RunID != uuid.Nil {
		uid, _ := uuid.Parse(cfg.UserID)
		sess := e.runner.sessionRegistry.Register(cfg.RunID, uid, cfg.AccountID, cfg.Symbol, cfg.Timeframe, cfg.Mode, cancel)
		if sess == nil {
			cancel()
			e.mu.Lock()
			delete(e.activeRuns, schedule.ID)
			e.mu.Unlock()
			errMsg := fmt.Sprintf("account %s already has a running strategy", cfg.AccountID)
			_ = e.repo.UpdateLastRun(ctx, schedule.ID, fmt.Errorf("launch: %s", errMsg))
			if e.runner.runRepo != nil {
				_ = e.runner.runRepo.UpdateStopped(context.Background(), cfg.RunID, "error", errMsg)
			}
			e.log.Warn("launchEventSession: account conflict",
				zap.String("schedule_id", schedule.ID.String()),
				zap.String("account_id", cfg.AccountID))
			return fmt.Errorf("launch: %s", errMsg)
		}
		cfg.PreRegisteredSession = sess
	}

	go func(ctx context.Context) { e.runOne(ctx, schedule, cfg, handle) }(runCtx)

	e.log.Info("launchEventSession: started",
		zap.String("schedule_id", schedule.ID.String()),
		zap.String("symbol", schedule.Symbol),
		zap.String("timeframe", schedule.Timeframe))
	return nil
}
