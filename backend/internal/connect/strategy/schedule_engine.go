// schedule_engine.go — Schedule execution engine: timer-driven loop that dispatches
// due strategy schedules to RunLiveStrategy.
//
// Architecture:
//   Timer + notifyCh (push-first)
//     recomputeTimer() finds the earliest next_run_at, sets a time.Timer.
//     When Timer fires → GetDueSchedules → dispatch() → go runOne().
//     When an external event occurs (new schedule, toggle, runOne completes)
//     → notifyCh → recomputeTimer() resets the timer.
//   Active runs are tracked in activeRuns map[scheduleID]*runHandle to prevent
//     duplicate execution and support StopSchedule().

package strategy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/internal/repository"
)

// runHandle tracks a single running strategy execution.
type runHandle struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup // waits for the goroutine to exit
}

// ScheduleEngine polls due schedules and dispatches them to RunLiveStrategy.
type ScheduleEngine struct {
	repo             *repository.StrategyScheduleRepository
	templateRepo     *repository.AIStrategyTemplatesRepository
	runner           *PythonStrategyServer
	autoTradeEnabled func(userID uuid.UUID) bool // nil = all enabled
	activeRuns       map[uuid.UUID]*runHandle
	notifyCh         chan struct{} // 1-buffered, external events → recomputeTimer
	mu               sync.Mutex
	log              *zap.Logger
}

// NewScheduleEngine creates a new schedule engine.
func NewScheduleEngine(
	repo *repository.StrategyScheduleRepository,
	templateRepo *repository.AIStrategyTemplatesRepository,
	runner *PythonStrategyServer,
	autoTradeFn func(userID uuid.UUID) bool,
	log *zap.Logger,
) *ScheduleEngine {
	return &ScheduleEngine{
		repo:             repo,
		templateRepo:     templateRepo,
		runner:           runner,
		autoTradeEnabled: autoTradeFn,
		activeRuns:       make(map[uuid.UUID]*runHandle),
		notifyCh:         make(chan struct{}, 1),
		log:              log,
	}
}

// Start begins the main timer loop. Blocks until ctx is cancelled.
// Callers should run this in a goroutine.
func (e *ScheduleEngine) Start(ctx context.Context) error {
	e.log.Info("schedule engine starting")

	// First pass: ensure all active schedules have a valid next_run_at.
	if err := e.reconcileOnStartup(ctx); err != nil {
		e.log.Warn("schedule engine startup reconcile failed", zap.Error(err))
		// Continue anyway — the timer loop will pick up due schedules.
	}

	var currentTimer *time.Timer
	defer func() {
		if currentTimer != nil {
			currentTimer.Stop()
		}
	}()

	for {
		timer, earliest, err := e.recomputeTimer(ctx)
		if err != nil {
			e.log.Error("schedule engine recompute timer failed", zap.Error(err))
			// Backoff before retrying.
			timer = time.NewTimer(30 * time.Second)
		}

		// Stop previous timer before replacing.
		if currentTimer != nil {
			currentTimer.Stop()
		}
		currentTimer = timer

		select {
		case <-timer.C:
			e.log.Debug("timer fired, checking due schedules",
				zap.Time("earliest", earliest))
			e.executeLoop(ctx)

		case <-e.notifyCh:
			e.log.Debug("notify received, recomputing timer")
			// Timer will be recomputed on next loop iteration.

		case <-ctx.Done():
			e.log.Info("schedule engine shutting down")
			e.Stop()
			return ctx.Err()
		}
	}
}

// Notify signals the engine to recompute its timer. Safe to call from any goroutine.
func (e *ScheduleEngine) Notify() {
	select {
	case e.notifyCh <- struct{}{}:
	default:
		// Channel full — a recompute is already pending.
	}
}

// reconcileOnStartup loads all active schedules and fills in missing next_run_at values.
func (e *ScheduleEngine) reconcileOnStartup(ctx context.Context) error {
	schedules, err := e.repo.GetActiveSchedules(ctx)
	if err != nil {
		return err
	}
	var updated int
	for _, s := range schedules {
		if s.NextRunAt != nil && !s.NextRunAt.IsZero() {
			continue
		}
		next, err := s.ComputeNextRunAt()
		if err != nil || next.IsZero() {
			continue
		}
		if err := e.repo.UpdateNextRunAt(ctx, s.ID, next); err != nil {
			e.log.Warn("reconcile: failed to update next_run_at",
				zap.String("schedule_id", s.ID.String()), zap.Error(err))
			continue
		}
		updated++
	}
	if updated > 0 {
		e.log.Info("reconciled missing next_run_at", zap.Int("count", updated))
	}
	return nil
}

// recomputeTimer returns a Timer that fires at the earliest next_run_at.
// If there are no active schedules with next_run_at, returns a long-wait timer.
func (e *ScheduleEngine) recomputeTimer(ctx context.Context) (*time.Timer, time.Time, error) {
	earliest, err := e.repo.GetEarliestNextRunAt(ctx)
	if err != nil {
		return time.NewTimer(30 * time.Second), time.Time{}, err
	}

	if earliest.IsZero() {
		// No active schedules — wait a long time (will be woken by Notify).
		timer := time.NewTimer(24 * time.Hour)
		e.log.Debug("no active schedules, waiting for notify")
		return timer, time.Time{}, nil
	}

	duration := time.Until(earliest)
	if duration < 0 {
		duration = 0 // fire immediately
	}
	e.log.Debug("timer set",
		zap.Time("earliest", earliest),
		zap.Duration("duration", duration))
	return time.NewTimer(duration), earliest, nil
}

// executeLoop queries due schedules and dispatches them.
func (e *ScheduleEngine) executeLoop(ctx context.Context) {
	due, err := e.repo.GetDueSchedules(ctx, time.Now())
	if err != nil {
		e.log.Error("failed to get due schedules", zap.Error(err))
		return
	}
	if len(due) == 0 {
		return
	}
	e.log.Info("due schedules found", zap.Int("count", len(due)))

	for _, s := range due {
		select {
		case <-ctx.Done():
			return
		default:
		}
		e.mu.Lock()
		_, running := e.activeRuns[s.ID]
		e.mu.Unlock()
		if running {
			e.log.Debug("schedule already running, skipping",
				zap.String("schedule_id", s.ID.String()))
			continue
		}

		if e.autoTradeEnabled != nil && !e.autoTradeEnabled(s.UserID) {
			e.log.Debug("auto-trade disabled for user, skipping",
				zap.String("user_id", s.UserID.String()),
				zap.String("schedule_id", s.ID.String()))
			continue
		}

		e.dispatch(ctx, s)
	}
}

// dispatch loads the template code and launches a goroutine to run the strategy.
func (e *ScheduleEngine) dispatch(ctx context.Context, schedule *model.StrategySchedule) {
	// Load template code.
	tpl, err := e.templateRepo.GetByID(ctx, schedule.TemplateID)
	if err != nil || tpl == nil {
		e.log.Error("failed to load template for schedule",
			zap.String("schedule_id", schedule.ID.String()),
			zap.String("template_id", schedule.TemplateID.String()),
			zap.Error(err))
		errMsg := "template not found"
		if err != nil {
			errMsg = err.Error()
		}
		_ = e.repo.UpdateLastRun(ctx, schedule.ID, wrapErr(errMsg))
		return
	}
	if tpl.PythonSkeleton == "" {
		e.log.Warn("template has empty code, skipping",
			zap.String("schedule_id", schedule.ID.String()))
		_ = e.repo.UpdateLastRun(ctx, schedule.ID, wrapErr("template code is empty"))
		return
	}

	params, _ := schedule.GetParameters()
	strParams := make(map[string]string, len(params))
	for k, v := range params {
		strParams[k] = fmt.Sprintf("%v", v)
	}

	cfg := LiveStrategyConfig{
		AccountID: schedule.AccountID.String(),
		Symbol:    schedule.Symbol,
		Timeframe: schedule.Timeframe,
		Code:      tpl.PythonSkeleton,
		Mode:      "live",
		Params:    strParams,
	}

	runCtx, cancel := context.WithCancel(context.Background())
	handle := &runHandle{cancel: cancel}
	handle.wg.Add(1)

	e.mu.Lock()
	e.activeRuns[schedule.ID] = handle
	e.mu.Unlock()

	go e.runOne(runCtx, schedule, cfg, handle)

	e.log.Info("schedule dispatched",
		zap.String("schedule_id", schedule.ID.String()),
		zap.String("symbol", schedule.Symbol),
		zap.String("timeframe", schedule.Timeframe))
}

// runOne executes a single strategy run and cleans up on completion.
func (e *ScheduleEngine) runOne(ctx context.Context, schedule *model.StrategySchedule, cfg LiveStrategyConfig, handle *runHandle) {
	defer handle.wg.Done()
	defer func() {
		e.mu.Lock()
		delete(e.activeRuns, schedule.ID)
		e.mu.Unlock()
		handle.cancel()
	}()

	var runErr error
	if e.runner != nil {
		runErr = e.runner.RunLiveStrategy(ctx, cfg)
	} else {
		runErr = wrapErr("strategy runner not configured")
	}

	// Record execution result.
	recordCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.repo.UpdateLastRun(recordCtx, schedule.ID, runErr); err != nil {
		e.log.Error("failed to update last_run", zap.Error(err))
	}

	// Schedule next run.
	next, err := schedule.ComputeNextRunAt()
	if err != nil {
		e.log.Error("failed to compute next_run_at", zap.Error(err))
	} else if !next.IsZero() {
		if err := e.repo.UpdateNextRunAt(recordCtx, schedule.ID, next); err != nil {
			e.log.Error("failed to update next_run_at", zap.Error(err))
		}
	}

	// Signal the timer to recompute.
	e.Notify()

	if runErr != nil {
		e.log.Warn("schedule run completed with error",
			zap.String("schedule_id", schedule.ID.String()),
			zap.Error(runErr))
	} else {
		e.log.Info("schedule run completed",
			zap.String("schedule_id", schedule.ID.String()))
	}
}

// StopSchedule cancels a running schedule execution by its ID.
// Blocks until the goroutine has exited. Safe to call from any goroutine.
func (e *ScheduleEngine) StopSchedule(id uuid.UUID) {
	e.mu.Lock()
	handle, ok := e.activeRuns[id]
	e.mu.Unlock()
	if !ok {
		return
	}
	handle.cancel()
	handle.wg.Wait()
	e.log.Info("schedule stopped", zap.String("schedule_id", id.String()))
}

// Stop cancels all running strategy executions and waits for them to exit.
func (e *ScheduleEngine) Stop() {
	e.mu.Lock()
	handles := make([]*runHandle, 0, len(e.activeRuns))
	for id, h := range e.activeRuns {
		handles = append(handles, h)
		e.log.Info("cancelling schedule", zap.String("schedule_id", id.String()))
		h.cancel()
	}
	// Clear the map so runOne defers don't race.
	for id := range e.activeRuns {
		delete(e.activeRuns, id)
	}
	e.mu.Unlock()

	for _, h := range handles {
		h.wg.Wait()
	}
	e.log.Info("all schedules stopped")
}

// wrapErr is a helper to create a recognizable error for UpdateLastRun.
func wrapErr(msg string) error {
	return &scheduleError{msg: msg}
}

type scheduleError struct{ msg string }

func (e *scheduleError) Error() string { return e.msg }
