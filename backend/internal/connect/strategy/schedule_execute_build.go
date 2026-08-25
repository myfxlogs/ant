// schedule_execute_build.go — buildLiveRun and runOne extracted from schedule_execute.go.
package strategy

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/repository"
)

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
