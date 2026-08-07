// schedule_engine.go — Schedule execution engine: timer-driven loop that dispatches
// due strategy schedules to RunLiveStrategy.
//
// Architecture (push-first, zero-polling):
//   Timer + notifyCh: recomputeTimer() finds the earliest next_run_at, sets a time.Timer.
//   Timer fires → GetDueSchedules → dispatch() → go runOne().
//   External events (create, toggle, runOne completion) → notifyCh → recomputeTimer resets.
//   Active runs tracked in activeRuns map[scheduleID]*runHandle (Paper Trading pattern).

package strategy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
)

type runHandle struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// ScheduleRepo abstracts the schedule repository methods used by ScheduleEngine.
// *repository.StrategyScheduleRepository satisfies this interface implicitly.
type ScheduleRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error)
	GetActiveSchedules(ctx context.Context) ([]*model.StrategySchedule, error)
	GetDueSchedules(ctx context.Context, now time.Time) ([]*model.StrategySchedule, error)
	GetEarliestNextRunAt(ctx context.Context) (time.Time, error)
	UpdateLastRun(ctx context.Context, id uuid.UUID, runErr error) error
	UpdateNextRunAt(ctx context.Context, id uuid.UUID, next time.Time) error
}

type ScheduleEngine struct {
	repo             ScheduleRepo
	templateReader   TemplateCodeReader
	runner           *StrategyExecutionServer
	autoTradeEnabled func(userID uuid.UUID) bool                               // nil = all enabled
	entitlementCheck func(ctx context.Context, userID, strategyID string) bool // nil = skip
	activeRuns       map[uuid.UUID]*runHandle
	notifyCh         chan struct{} // 1-buffered, external events → recomputeTimer
	mu               sync.Mutex
	log              *zap.Logger

	// D6: TTL cache for autoTradeEnabled results to avoid PG query per schedule check.
	autoTradeCache   map[uuid.UUID]autoTradeEntry
	autoTradeCacheMu sync.Mutex
}

type autoTradeEntry struct {
	enabled  bool
	expireAt time.Time
}

const autoTradeCacheTTL = 30 * time.Second

const (
	backoffDelay = 30 * time.Second
	idleDelay    = 24 * time.Hour
	dbTimeout    = 5 * time.Second
)

type TemplateCodeReader interface {
	GetTemplate(ctx context.Context, id, userID uuid.UUID) (*service.TemplateRow, error)
}

func NewScheduleEngine(
	repo ScheduleRepo,
	templateReader TemplateCodeReader,
	runner *StrategyExecutionServer,
	autoTradeFn func(userID uuid.UUID) bool,
	entitlementFn func(ctx context.Context, userID, strategyID string) bool,
	log *zap.Logger,
) *ScheduleEngine {
	return &ScheduleEngine{
		repo:             repo,
		templateReader:   templateReader,
		runner:           runner,
		autoTradeEnabled: autoTradeFn,
		entitlementCheck: entitlementFn,
		activeRuns:       make(map[uuid.UUID]*runHandle),
		notifyCh:         make(chan struct{}, 1),
		log:              log,
		autoTradeCache:   make(map[uuid.UUID]autoTradeEntry),
	}
}

// Start begins the main timer loop. Blocks until ctx is cancelled.
func (e *ScheduleEngine) Start(ctx context.Context) error {
	e.log.Info("schedule engine starting")
	e.reconcileOnStartup(ctx)

	var cur *time.Timer
	defer func() {
		if cur != nil && !cur.Stop() {
			<-cur.C // drain
		}
	}()

	for {
		timer, _, err := e.recomputeTimer(ctx)
		if err != nil {
			e.log.Error("recompute timer failed", zap.Error(err))
			timer = time.NewTimer(backoffDelay)
		}
		if cur != nil {
			if !cur.Stop() {
				select {
				case <-cur.C:
				default:
				}
			}
		}
		cur = timer

		select {
		case <-timer.C:
			e.executeLoop(ctx)
		case <-e.notifyCh:
		case <-ctx.Done():
			e.log.Info("schedule engine shutting down")
			e.Stop()
			return ctx.Err()
		}
	}
}

func (e *ScheduleEngine) Notify() {
	select {
	case e.notifyCh <- struct{}{}:
	default:
	}
}

// --- internal helpers ---

func (e *ScheduleEngine) isRunning(id uuid.UUID) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := e.activeRuns[id]
	return ok
}

func (e *ScheduleEngine) reconcileOnStartup(ctx context.Context) {
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
		next, err := s.ComputeNextRunAt()
		if err != nil || next.IsZero() {
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

func (e *ScheduleEngine) recomputeTimer(ctx context.Context) (*time.Timer, time.Time, error) {
	earliest, err := e.repo.GetEarliestNextRunAt(ctx)
	if err != nil {
		return time.NewTimer(backoffDelay), time.Time{}, err
	}
	if earliest.IsZero() {
		return time.NewTimer(idleDelay), time.Time{}, nil
	}
	d := time.Until(earliest)
	if d < 0 {
		d = 0
	}
	return time.NewTimer(d), earliest, nil
}

func (e *ScheduleEngine) executeLoop(ctx context.Context) {
	due, err := e.repo.GetDueSchedules(ctx, time.Now())
	if err != nil {
		e.log.Error("get due schedules failed", zap.Error(err))
		return
	}
	if len(due) == 0 {
		return
	}
	e.log.Info("due schedules found", zap.Int("count", len(due)))

	for _, s := range due {
		if e.isRunning(s.ID) {
			continue
		}
		if !e.isAutoTradeEnabled(s.UserID) {
			continue
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		e.dispatch(ctx, s)
	}
}

// isAutoTradeEnabled checks the per-user autotrade setting with a TTL cache
// to avoid querying PG on every schedule dispatch cycle.
func (e *ScheduleEngine) isAutoTradeEnabled(userID uuid.UUID) bool {
	if e.autoTradeEnabled == nil {
		return true
	}
	now := time.Now()
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
	// Entitlement gate (task 3): verify active subscription/trial/ownership before running.
	if e.entitlementCheck != nil {
		if !e.entitlementCheck(ctx, schedule.UserID.String(), schedule.TemplateID.String()) {
			e.log.Warn("dispatch: entitlement denied",
				zap.String("schedule_id", schedule.ID.String()),
				zap.String("user_id", schedule.UserID.String()))
			_ = e.repo.UpdateLastRun(ctx, schedule.ID, fmt.Errorf("unauthorized: no active entitlement"))
			return
		}
	}
	// Quota gate (task 5): enforce live strategy limit.
	if e.runner != nil {
		if err := e.runner.checkStrategyQuota(ctx, schedule.UserID, "live"); err != nil {
			e.log.Warn("dispatch: quota exceeded",
				zap.String("schedule_id", schedule.ID.String()), zap.Error(err))
			_ = e.repo.UpdateLastRun(ctx, schedule.ID, err)
			return
		}
	}
	// Load and validate template.
	tpl, err := e.templateReader.GetTemplate(ctx, schedule.TemplateID, schedule.UserID)
	if err != nil || tpl == nil || tpl.Code == "" {
		reason := "template code is empty"
		if err != nil {
			reason = err.Error()
		}
		e.log.Error("dispatch: invalid template",
			zap.String("schedule_id", schedule.ID.String()), zap.String("template_id", schedule.TemplateID.String()), zap.Error(err))
		_ = e.repo.UpdateLastRun(ctx, schedule.ID, fmt.Errorf("dispatch: %s", reason))
		return
	}

	strParams, _ := schedule.GetParameters()

	runCtx, cancel := context.WithCancel(ctx)
	handle := &runHandle{cancel: cancel}
	handle.wg.Add(1)

	e.mu.Lock()
	e.activeRuns[schedule.ID] = handle
	e.mu.Unlock()

	// Set per-bar entitlement revalidation for marketplace strategies (task 4).
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
			e.log.Warn("dispatch: failed to create run record", zap.Error(err))
		} else {
			cfg.RunID = run.ID
		}
	}

	go func(ctx context.Context) { e.runOne(ctx, schedule, cfg, handle) }(runCtx)

	e.log.Info("dispatched", zap.String("schedule_id", schedule.ID.String()),
		zap.String("symbol", schedule.Symbol), zap.String("timeframe", schedule.Timeframe))
}

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
	if next, err := schedule.ComputeNextRunAt(); err != nil {
		e.log.Error("compute next_run_at failed", zap.Error(err))
	} else if !next.IsZero() {
		if err := e.repo.UpdateNextRunAt(recordCtx, schedule.ID, next); err != nil {
			e.log.Error("update next_run_at failed", zap.Error(err))
		}
	}

	e.Notify()

	if runErr != nil {
		e.log.Warn("run completed with error",
			zap.String("schedule_id", schedule.ID.String()), zap.Error(runErr))
	} else {
		e.log.Info("run completed", zap.String("schedule_id", schedule.ID.String()))
	}
}

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

func (e *ScheduleEngine) Stop() {
	e.mu.Lock()
	handles := make([]*runHandle, 0, len(e.activeRuns))
	for id, h := range e.activeRuns {
		handles = append(handles, h)
		e.log.Info("cancelling", zap.String("schedule_id", id.String()))
		h.cancel()
		delete(e.activeRuns, id)
	}
	e.mu.Unlock()

	for _, h := range handles {
		h.wg.Wait()
	}
	e.log.Info("all schedules stopped")
}
