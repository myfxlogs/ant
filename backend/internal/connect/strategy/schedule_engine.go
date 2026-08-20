// schedule_engine.go — Schedule execution engine: timer-driven loop that dispatches
// due strategy schedules to RunLiveStrategy.
//
// Architecture (push-first, zero-polling):
//   Timer + notifyCh: recomputeTimer() finds the earliest next_run_at, sets a time.Timer.
//   Timer fires → executeLoop → pre-consume due (advance next_run_at) → dispatch() → go runOne().
//   External events (create, toggle, runOne completion) → notifyCh → recomputeTimer resets.
//   Active runs tracked in activeRuns map[scheduleID]*runHandle (Paper Trading pattern).
//
// SCHEDULE-HOTLOOP-1 fix: timer occurrence is a fact that must be persistently
// consumed BEFORE any skip/deny/dispatch branch. Each due schedule has its
// next_run_at advanced to strictly > now before isRunning/autoTrade/entitlement
// checks. This prevents the 0-delay hot loop where autoTrade=false schedules
// never advance next_run_at → GetEarliestNextRunAt returns past time → timer 0.

package strategy

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/model"
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
	ClearNextRunAt(ctx context.Context, id uuid.UUID) error
	ClearEventNextRunAt(ctx context.Context) (int, error)
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
	lifecycleCtx     context.Context // set by Start(); used as runCtx parent so runs outlive handler ctx

	// now returns the current time. Injected for deterministic tests; defaults to time.Now.
	now func() time.Time

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
		now:              time.Now,
		autoTradeCache:   make(map[uuid.UUID]autoTradeEntry),
	}
}

// Start begins the main timer loop. Blocks until ctx is cancelled.
// On executeLoop error (DB failure), enters a context-aware backoff timer
// that can be preempted by Notify or ctx cancellation (SCHEDULE-HOTLOOP-1).
func (e *ScheduleEngine) Start(ctx context.Context) error {
	e.lifecycleCtx = ctx
	e.log.Info("schedule engine starting")
	e.reconcileOnStartup(ctx)

	for {
		if err := e.runTimerCycle(ctx); err != nil {
			return err
		}
	}
}

// runTimerCycle executes one timer→executeLoop→backoff cycle.
// Returns ctx.Err() when the engine should stop; nil to continue.
func (e *ScheduleEngine) runTimerCycle(ctx context.Context) error {
	timer, _, err := e.recomputeTimer(ctx)
	if err != nil {
		e.log.Error("recompute timer failed", zap.Error(err))
		timer = time.NewTimer(backoffDelay)
	}
	defer timer.Stop()

	select {
	case <-timer.C:
		if loopErr := e.executeLoop(ctx); loopErr != nil {
			e.log.Error("execute loop failed, backing off", zap.Error(loopErr))
			return e.backoff(ctx)
		}
		return nil
	case <-e.notifyCh:
		// External event → recompute timer next cycle.
		return nil
	case <-ctx.Done():
		e.log.Info("schedule engine shutting down")
		e.Stop()
		return ctx.Err()
	}
}

// backoff waits for backoffDelay, preemptable by Notify or ctx cancellation.
// The timer is always Stop+drained to avoid leakage (SCHEDULE-HOTLOOP-1).
func (e *ScheduleEngine) backoff(ctx context.Context) error {
	t := time.NewTimer(backoffDelay)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-e.notifyCh:
		return nil
	case <-ctx.Done():
		e.log.Info("schedule engine shutting down during backoff")
		e.Stop()
		return ctx.Err()
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

// nowTime returns the injected clock, defaulting to time.Now.
func (e *ScheduleEngine) nowTime() time.Time {
	if e.now != nil {
		return e.now()
	}
	return time.Now()
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

// InvalidateAutoTradeCache removes the cached autoTradeEnabled entry for a user.
// Called by external write paths (ToggleAutoTrade, UpdateGlobalSettings) via a
// callback so the autotrading package does not import strategy (no import cycle).
// After invalidation, the next isAutoTradeEnabled call re-queries the source.
// Callers should also call Notify() to recompute the timer immediately.
func (e *ScheduleEngine) InvalidateAutoTradeCache(userID uuid.UUID) {
	e.autoTradeCacheMu.Lock()
	delete(e.autoTradeCache, userID)
	e.autoTradeCacheMu.Unlock()
}
