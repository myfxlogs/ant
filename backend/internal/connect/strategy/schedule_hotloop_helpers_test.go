// schedule_hotloop_test.go — Adversarial tests for SCHEDULE-HOTLOOP-1.
//
// Each test verifies a specific aspect of the timer occurrence pre-consume fix.
// Adversarial proofs: deleting the key line in the implementation makes each
// test go RED. Tests use a fake repo with call-order tracking and an injectable
// now() — no real sleep, no time.Now drift.

package strategy

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
)

type fakeScheduleRepo struct {
	mu sync.Mutex

	schedules map[uuid.UUID]*model.StrategySchedule

	// Call order log: records the sequence of repo calls for ordering assertions.
	callLog []string

	// Configurable failures.
	getDueErr      error
	updateNextErr  error
	clearNextErr   error
	clearEventErr  error
	getEarliestErr error
	getActiveErr   error

	// Atomic call counters (read by tests while Start goroutine runs).
	getDueCount     atomic.Int64
	updateNextCount atomic.Int64
	clearNextCount  atomic.Int64
	updateLastCount atomic.Int64
	clearEventCount atomic.Int64
}

func newFakeScheduleRepo() *fakeScheduleRepo {
	return &fakeScheduleRepo{
		schedules: make(map[uuid.UUID]*model.StrategySchedule),
	}
}

func (f *fakeScheduleRepo) logCall(name string) {
	f.mu.Lock()
	f.callLog = append(f.callLog, name)
	f.mu.Unlock()
}

func (f *fakeScheduleRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.schedules[id]; ok {
		return s, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeScheduleRepo) GetActiveSchedules(ctx context.Context) ([]*model.StrategySchedule, error) {
	f.logCall("GetActiveSchedules")
	if f.getActiveErr != nil {
		return nil, f.getActiveErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var result []*model.StrategySchedule
	for _, s := range f.schedules {
		if s.IsActive {
			result = append(result, s)
		}
	}
	return result, nil
}

func (f *fakeScheduleRepo) GetDueSchedules(ctx context.Context, now time.Time) ([]*model.StrategySchedule, error) {
	f.mu.Lock()
	f.getDueCount.Add(1)
	f.mu.Unlock()
	f.logCall("GetDueSchedules")
	if f.getDueErr != nil {
		return nil, f.getDueErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var result []*model.StrategySchedule
	for _, s := range f.schedules {
		if !s.IsActive || s.NextRunAt == nil {
			continue
		}
		if s.ScheduleType != model.ScheduleTypeInterval && s.ScheduleType != model.ScheduleTypeCron {
			continue
		}
		if !s.NextRunAt.After(now) {
			result = append(result, s)
		}
	}
	return result, nil
}

func (f *fakeScheduleRepo) GetEarliestNextRunAt(ctx context.Context) (time.Time, error) {
	if f.getEarliestErr != nil {
		return time.Time{}, f.getEarliestErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var earliest *time.Time
	for _, s := range f.schedules {
		if !s.IsActive || s.NextRunAt == nil {
			continue
		}
		if s.ScheduleType != model.ScheduleTypeInterval && s.ScheduleType != model.ScheduleTypeCron {
			continue
		}
		if earliest == nil || s.NextRunAt.Before(*earliest) {
			t := *s.NextRunAt
			earliest = &t
		}
	}
	if earliest == nil {
		return time.Time{}, nil
	}
	return *earliest, nil
}

func (f *fakeScheduleRepo) UpdateLastRun(ctx context.Context, id uuid.UUID, runErr error) error {
	f.mu.Lock()
	f.updateLastCount.Add(1)
	f.mu.Unlock()
	f.logCall("UpdateLastRun")
	return nil
}

func (f *fakeScheduleRepo) UpdateNextRunAt(ctx context.Context, id uuid.UUID, next time.Time) error {
	f.mu.Lock()
	f.updateNextCount.Add(1)
	f.mu.Unlock()
	f.logCall("UpdateNextRunAt")
	if f.updateNextErr != nil {
		return f.updateNextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.schedules[id]; ok {
		s.NextRunAt = &next
	}
	return nil
}

func (f *fakeScheduleRepo) ClearNextRunAt(ctx context.Context, id uuid.UUID) error {
	f.mu.Lock()
	f.clearNextCount.Add(1)
	f.mu.Unlock()
	f.logCall("ClearNextRunAt")
	if f.clearNextErr != nil {
		return f.clearNextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.schedules[id]; ok {
		s.NextRunAt = nil
	}
	return nil
}

func (f *fakeScheduleRepo) ClearEventNextRunAt(ctx context.Context) (int, error) {
	f.logCall("ClearEventNextRunAt")
	if f.clearEventErr != nil {
		return 0, f.clearEventErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, s := range f.schedules {
		if s.ScheduleType == model.ScheduleTypeEvent && s.NextRunAt != nil {
			s.NextRunAt = nil
			count++
		}
	}
	f.clearEventCount.Store(int64(count))
	return count, nil
}

func (f *fakeScheduleRepo) getCallLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]string, len(f.callLog))
	copy(cp, f.callLog)
	return cp
}

func (f *fakeScheduleRepo) getNextRunAt(id uuid.UUID) *time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.schedules[id]; ok {
		return s.NextRunAt
	}
	return nil
}

// --- helpers ---

// makeEngine creates a ScheduleEngine with a fake repo and injectable now.

// makeEngine creates a ScheduleEngine with a fake repo and injectable now.
func makeEngine(repo *fakeScheduleRepo, now time.Time, autoTradeFn func(uuid.UUID) bool) *ScheduleEngine {
	e := &ScheduleEngine{
		repo:                repo,
		templateReader:      &mockTemplateReader{},
		activeRuns:          make(map[uuid.UUID]*runHandle),
		notifyCh:            make(chan struct{}, 1),
		log:                 zap.NewNop(),
		now:                 func() time.Time { return now },
		autoTradeCache:      make(map[uuid.UUID]autoTradeEntry),
		autoTradeGeneration: make(map[uuid.UUID]uint64),
	}
	if autoTradeFn != nil {
		e.autoTradeEnabled = autoTradeFn
	}
	return e
}

// drainNotify consumes any pending notify signal.

// drainNotify consumes any pending notify signal.
func drainNotify(ch chan struct{}) {
	select {
	case <-ch:
	default:
	}
}

// --- Tests ---

// 1. AutoTradeDisabledConsumesDue: autoTrade=false schedule must still advance
// next_run_at > now, and dispatch=0. Adversarial: delete pre-advance → next_run_at
// stays in the past → RED (GetDueSchedules keeps returning it).

// makeIntervalScheduleProto creates an interval schedule with proto-encoded config.
func makeIntervalScheduleProto(userID uuid.UUID, nextRunAt time.Time, intervalMs int64) *model.StrategySchedule {
	s := &model.StrategySchedule{
		ID:           uuid.New(),
		UserID:       userID,
		TemplateID:   uuid.New(),
		AccountID:    uuid.New(),
		Symbol:       "EURUSD",
		Timeframe:    "1h",
		ScheduleType: model.ScheduleTypeInterval,
		IsActive:     true,
		NextRunAt:    &nextRunAt,
	}
	cfg, _ := proto.Marshal(&antv1.ScheduleConfig{IntervalMs: intervalMs})
	s.ScheduleConfig = cfg
	return s
}
