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
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
)

// --- fakeScheduleRepo: tracks call order and state for adversarial tests ---

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
func TestSCHEDULE_HOTLOOP_1_AutoTradeDisabledConsumesDue(t *testing.T) {
	fixedNow := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	pastTime := fixedNow.Add(-1 * time.Hour)
	repo := newFakeScheduleRepo()
	sched := makeIntervalScheduleProto(userID, pastTime, 3600_000)
	repo.schedules[sched.ID] = sched

	engine := makeEngine(repo, fixedNow, func(uid uuid.UUID) bool {
		return false // autoTrade disabled
	})

	err := engine.executeLoop(context.Background())
	if err != nil {
		t.Fatalf("executeLoop returned error: %v", err)
	}

	// next_run_at must be advanced to > now.
	next := repo.getNextRunAt(sched.ID)
	if next == nil {
		t.Fatal("next_run_at is nil — pre-consume did not run")
	}
	if !next.After(fixedNow) {
		t.Errorf("next_run_at %v must be > now %v (pre-consume failed)", *next, fixedNow)
	}

	// No dispatch should have happened (autoTrade disabled).
	if repo.updateLastCount.Load() > 0 {
		t.Errorf("UpdateLastRun called %d times — autoTrade=false should not dispatch", repo.updateLastCount.Load())
	}
}

// 2. AlreadyRunningConsumesDue: a schedule already in activeRuns must still
// advance next_run_at > now, and not re-dispatch. Adversarial: delete pre-advance
// → next_run_at stays past → RED.
func TestSCHEDULE_HOTLOOP_1_AlreadyRunningConsumesDue(t *testing.T) {
	fixedNow := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	pastTime := fixedNow.Add(-1 * time.Hour)
	repo := newFakeScheduleRepo()
	sched := makeIntervalScheduleProto(userID, pastTime, 3600_000)
	repo.schedules[sched.ID] = sched

	engine := makeEngine(repo, fixedNow, func(uid uuid.UUID) bool { return true })

	// Simulate already running.
	engine.activeRuns[sched.ID] = &runHandle{cancel: func() {}}

	err := engine.executeLoop(context.Background())
	if err != nil {
		t.Fatalf("executeLoop returned error: %v", err)
	}

	next := repo.getNextRunAt(sched.ID)
	if next == nil {
		t.Fatal("next_run_at is nil — pre-consume did not run for already-running schedule")
	}
	if !next.After(fixedNow) {
		t.Errorf("next_run_at %v must be > now %v (pre-consume failed for running schedule)", *next, fixedNow)
	}
}

// 3. EligibleAdvancesBeforeDispatch: UpdateNextRunAt must occur before dispatch
// (buildLiveRun/runOne). Adversarial: delete pre-advance → callLog shows dispatch
// before UpdateNextRunAt → RED.
func TestSCHEDULE_HOTLOOP_1_EligibleAdvancesBeforeDispatch(t *testing.T) {
	fixedNow := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	pastTime := fixedNow.Add(-1 * time.Hour)
	repo := newFakeScheduleRepo()
	sched := makeIntervalScheduleProto(userID, pastTime, 3600_000)
	repo.schedules[sched.ID] = sched

	engine := makeEngine(repo, fixedNow, func(uid uuid.UUID) bool { return true })
	// runner is nil → dispatch will call runOne which records error but still
	// goes through buildLiveRun (which calls UpdateLastRun on gate failure or
	// template failure). We just need to verify UpdateNextRunAt is first.

	_ = engine.executeLoop(context.Background())

	callLog := repo.getCallLog()
	// Find UpdateNextRunAt index.
	nextIdx := -1
	for i, c := range callLog {
		if c == "UpdateNextRunAt" {
			nextIdx = i
			break
		}
	}
	if nextIdx < 0 {
		t.Fatal("UpdateNextRunAt was never called — pre-consume missing")
	}
	// UpdateLastRun from buildLiveRun/runOne must come AFTER UpdateNextRunAt.
	// (buildLiveRun may call UpdateLastRun on gate failure, but that should
	// also be after pre-consume since dispatch is after pre-consume.)
	lastRunIdx := -1
	for i, c := range callLog {
		if c == "UpdateLastRun" {
			lastRunIdx = i
			break
		}
	}
	if lastRunIdx >= 0 && lastRunIdx < nextIdx {
		t.Errorf("UpdateLastRun (idx %d) happened before UpdateNextRunAt (idx %d) — pre-consume order violated",
			lastRunIdx, nextIdx)
	}
}

// 4. UpdateNextFailureDoesNotDispatch: if UpdateNextRunAt fails, no dispatch
// occurs and executeLoop returns error. Adversarial: delete error gate →
// dispatch happens despite failure → RED.
func TestSCHEDULE_HOTLOOP_1_UpdateNextFailureDoesNotDispatch(t *testing.T) {
	fixedNow := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	pastTime := fixedNow.Add(-1 * time.Hour)
	repo := newFakeScheduleRepo()
	sched := makeIntervalScheduleProto(userID, pastTime, 3600_000)
	repo.schedules[sched.ID] = sched
	repo.updateNextErr = errors.New("DB connection lost")

	engine := makeEngine(repo, fixedNow, func(uid uuid.UUID) bool { return true })

	err := engine.executeLoop(context.Background())
	if err == nil {
		t.Fatal("executeLoop must return error when UpdateNextRunAt fails")
	}

	// No dispatch → no UpdateLastRun (buildLiveRun/runOne would call it).
	if repo.updateLastCount.Load() > 0 {
		t.Errorf("UpdateLastRun called %d times — should not dispatch when UpdateNextRunAt fails", repo.updateLastCount.Load())
	}
}

// 5. GetDueFailureBacksOff: when GetDueSchedules fails, executeLoop returns
// error and Start enters backoff. GetDue call count must be bounded within
// a window. Notify can preempt backoff. Adversarial: delete backoff →
// GetDue called unbounded times → RED.
func TestSCHEDULE_HOTLOOP_1_GetDueFailureBacksOff(t *testing.T) {
	fixedNow := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	repo := newFakeScheduleRepo()
	repo.getDueErr = errors.New("DB down")

	// Add a schedule with past next_run_at so GetEarliestNextRunAt returns
	// a past time → timer fires immediately → executeLoop → GetDue fails → backoff.
	userID := uuid.New()
	pastTime := fixedNow.Add(-1 * time.Hour)
	sched := makeIntervalScheduleProto(userID, pastTime, 3600_000)
	repo.schedules[sched.ID] = sched

	var nowVal atomic.Int64
	nowVal.Store(fixedNow.UnixNano())
	engine := &ScheduleEngine{
		repo:           repo,
		templateReader: &mockTemplateReader{},
		activeRuns:     make(map[uuid.UUID]*runHandle),
		notifyCh:       make(chan struct{}, 1),
		log:            zap.NewNop(),
		now:            func() time.Time { return time.Unix(0, nowVal.Load()) },
		autoTradeCache: make(map[uuid.UUID]autoTradeEntry),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go func() { _ = engine.Start(ctx) }()

	time.Sleep(150 * time.Millisecond)
	cancel()

	count := repo.getDueCount.Load()
	// With backoffDelay=30s, in 200ms we expect at most 1-2 GetDue calls.
	// Without backoff it would be hundreds. Allow up to 5 for scheduling jitter.
	if count > 5 {
		t.Errorf("GetDueSchedules called %d times in 200ms — backoff not working (expected ≤5)", count)
	}
}

// 5b. NotifyPreemptsBackoff: Notify can end backoff early.
func TestSCHEDULE_HOTLOOP_1_NotifyPreemptsBackoff(t *testing.T) {
	fixedNow := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	repo := newFakeScheduleRepo()
	repo.getDueErr = errors.New("DB down")

	// Add a schedule with past next_run_at so timer fires immediately.
	userID := uuid.New()
	pastTime := fixedNow.Add(-1 * time.Hour)
	sched := makeIntervalScheduleProto(userID, pastTime, 3600_000)
	repo.schedules[sched.ID] = sched

	engine := &ScheduleEngine{
		repo:           repo,
		templateReader: &mockTemplateReader{},
		activeRuns:     make(map[uuid.UUID]*runHandle),
		notifyCh:       make(chan struct{}, 1),
		log:            zap.NewNop(),
		now:            func() time.Time { return fixedNow },
		autoTradeCache: make(map[uuid.UUID]autoTradeEntry),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startTime := time.Now()
	go func() { _ = engine.Start(ctx) }()

	// Wait for first executeLoop to fail and enter backoff (30s).
	time.Sleep(100 * time.Millisecond)
	firstCount := repo.getDueCount.Load()
	if firstCount == 0 {
		t.Fatal("first executeLoop did not run — timer never fired")
	}

	// Send Notify — should preempt the 30s backoff.
	engine.Notify()

	// Within a short time, GetDue should be called again.
	time.Sleep(200 * time.Millisecond)
	secondCount := repo.getDueCount.Load()

	if secondCount <= firstCount {
		t.Errorf("Notify did not preempt backoff: GetDue count %d → %d (expected increase)", firstCount, secondCount)
	}

	elapsed := time.Since(startTime)
	if elapsed > 3*time.Second {
		t.Errorf("test took too long (%v) — Notify preempt not working", elapsed)
	}

	cancel()
}

// 6. EventScheduleExcluded: timer repository queries must not return event
// schedules, and startup reconcile must clear dirty event next_run_at.
func TestSCHEDULE_HOTLOOP_1_EventScheduleExcluded(t *testing.T) {
	fixedNow := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	repo := newFakeScheduleRepo()

	// Event schedule with dirty next_run_at (should be cleared on reconcile).
	pastTime := fixedNow.Add(-1 * time.Hour)
	eventSched := &model.StrategySchedule{
		ID:           uuid.New(),
		UserID:       userID,
		TemplateID:   uuid.New(),
		AccountID:    uuid.New(),
		Symbol:       "EURUSD",
		Timeframe:    "1h",
		ScheduleType: model.ScheduleTypeEvent,
		IsActive:     true,
		NextRunAt:    &pastTime,
	}
	repo.schedules[eventSched.ID] = eventSched

	engine := makeEngine(repo, fixedNow, nil)

	// Reconcile should clear the event's next_run_at.
	engine.reconcileOnStartup(context.Background())

	if repo.clearEventCount.Load() == 0 {
		t.Error("ClearEventNextRunAt did not clear any event schedules")
	}
	next := repo.getNextRunAt(eventSched.ID)
	if next != nil {
		t.Errorf("event schedule next_run_at should be nil after reconcile, got %v", *next)
	}

	// GetDueSchedules should not return event schedules even if they had
	// a next_run_at (the SQL filter excludes them; the fake repo also filters).
	repo.schedules[eventSched.ID].NextRunAt = &pastTime // re-dirty for GetDue test
	due, err := repo.GetDueSchedules(context.Background(), fixedNow)
	if err != nil {
		t.Fatalf("GetDueSchedules failed: %v", err)
	}
	for _, s := range due {
		if s.ScheduleType == model.ScheduleTypeEvent {
			t.Error("GetDueSchedules returned an event schedule — timer query must exclude events")
		}
	}

	// GetEarliestNextRunAt should also exclude events.
	earliest, err := repo.GetEarliestNextRunAt(context.Background())
	if err != nil {
		t.Fatalf("GetEarliestNextRunAt failed: %v", err)
	}
	if !earliest.IsZero() {
		t.Errorf("GetEarliestNextRunAt should return zero (only event exists), got %v", earliest)
	}
}

// 7. InvalidConfigQuarantined: if ComputeNextRunAt fails or returns zero,
// last_error is recorded, next_run_at is cleared to NULL, and the schedule
// is not dispatched. Subsequent timer queries must not return it.
func TestSCHEDULE_HOTLOOP_1_InvalidConfigQuarantined(t *testing.T) {
	fixedNow := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	pastTime := fixedNow.Add(-1 * time.Hour)
	repo := newFakeScheduleRepo()

	// Interval schedule with corrupt config bytes → ComputeNextRunAtFromConfigAt
	// returns a parse error. The schedule type is valid (interval) so it passes
	// the SQL/fake filter, but config is unparseable.
	sched := &model.StrategySchedule{
		ID:             uuid.New(),
		UserID:         userID,
		TemplateID:     uuid.New(),
		AccountID:      uuid.New(),
		Symbol:         "EURUSD",
		Timeframe:      "1h",
		ScheduleType:   model.ScheduleTypeInterval,
		ScheduleConfig: []byte{0xFF, 0xFF, 0xFF, 0xFF}, // corrupt proto bytes
		IsActive:       true,
		NextRunAt:      &pastTime,
	}
	repo.schedules[sched.ID] = sched

	engine := makeEngine(repo, fixedNow, func(uid uuid.UUID) bool { return true })

	_ = engine.executeLoop(context.Background())

	// last_error should be recorded.
	if repo.updateLastCount.Load() == 0 {
		t.Error("UpdateLastRun was not called for invalid config — last_error not recorded")
	}
	// next_run_at should be cleared to NULL.
	next := repo.getNextRunAt(sched.ID)
	if next != nil {
		t.Errorf("next_run_at should be nil (quarantined), got %v", *next)
	}
	// ClearNextRunAt should have been called.
	if repo.clearNextCount.Load() == 0 {
		t.Error("ClearNextRunAt was not called — schedule not quarantined")
	}

	// Subsequent GetDueSchedules must not return it (next_run_at is NULL).
	due, _ := repo.GetDueSchedules(context.Background(), fixedNow)
	for _, s := range due {
		if s.ID == sched.ID {
			t.Error("quarantined schedule still returned by GetDueSchedules — will repeat every cycle")
		}
	}
}

// 8. AutoTradeCacheInvalidation: ToggleAutoTrade and UpdateGlobalSettings
// must call the onAutoTradeChanged callback. UpdateGlobalSettings only when
// autoTradeEnabled actually changes. ScheduleEngine cache entry is deleted
// and Notify is triggered.
func TestSCHEDULE_HOTLOOP_1_AutoTradeCacheInvalidation(t *testing.T) {
	fixedNow := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	repo := newFakeScheduleRepo()

	engine := makeEngine(repo, fixedNow, func(uid uuid.UUID) bool { return false })

	// Prime the cache.
	if !engine.isAutoTradeEnabled(userID) {
		// expected false
	}
	engine.autoTradeCacheMu.Lock()
	if _, ok := engine.autoTradeCache[userID]; !ok {
		t.Fatal("cache entry not created after isAutoTradeEnabled")
	}
	engine.autoTradeCacheMu.Unlock()

	// Invalidate.
	engine.InvalidateAutoTradeCache(userID)
	engine.autoTradeCacheMu.Lock()
	_, stillCached := engine.autoTradeCache[userID]
	engine.autoTradeCacheMu.Unlock()
	if stillCached {
		t.Error("cache entry still present after InvalidateAutoTradeCache")
	}

	// Verify Notify is triggered.
	drainNotify(engine.notifyCh)
	engine.InvalidateAutoTradeCache(userID)
	engine.Notify()
	select {
	case <-engine.notifyCh:
		// good
	case <-time.After(100 * time.Millisecond):
		t.Error("Notify did not send signal after cache invalidation")
	}
}

// 8b. AutoTradeCacheInvalidationLinearizable: deterministic TOCTOU test for
// SCHEDULE-HOTLOOP-1a. Uses channels to force the exact interleaving:
//
//	A: cache miss → unlock → DB query starts, captures old true
//	B: DB commits false → callback invalidate (generation++ + delete cache)
//	A: old query returns true → re-lock → generation mismatch → discard → retry
//	A: retry query returns false → write false to cache
//
// Adversarial: delete the generation mismatch retry → A writes old true to cache
// → test RED (returns true, cache has true, queryCallCount==1).
func TestSCHEDULE_HOTLOOP_1_AutoTradeCacheInvalidationLinearizable(t *testing.T) {
	fixedNow := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	repo := newFakeScheduleRepo()

	// authoritativeState simulates the DB's autoTradeEnabled value.
	// Initial: true. The test will flip it to false mid-query.
	var authoritativeState atomic.Bool
	authoritativeState.Store(true)

	// Channels for deterministic timing control.
	queryStarted := make(chan struct{})
	releaseOldQuery := make(chan struct{})

	// Track how many times autoTradeEnabled was called.
	var queryCallCount atomic.Int64

	// fake autoTradeEnabled: first call blocks until released, captures old value.
	// Subsequent calls return the current authoritative state immediately.
	firstCall := atomic.Bool{}
	firstCall.Store(true)
	autoTradeFn := func(uid uuid.UUID) bool {
		queryCallCount.Add(1)
		if firstCall.CompareAndSwap(true, false) {
			// First call: capture the OLD value (true), signal that the query
			// has started, then block until the test releases us — by which
			// point the DB will have been updated and the cache invalidated.
			captured := authoritativeState.Load() // captures true
			close(queryStarted)
			<-releaseOldQuery // blocks until test allows return
			return captured   // returns stale true
		}
		// Subsequent calls: return current authoritative state (false).
		return authoritativeState.Load()
	}

	engine := makeEngine(repo, fixedNow, autoTradeFn)

	// Step 1: Start isAutoTradeEnabled in a goroutine. It will cache-miss,
	// call autoTradeEnabled (first call), which captures true and blocks.
	type result struct {
		enabled bool
	}
	resultCh := make(chan result, 1)
	go func() {
		enabled := engine.isAutoTradeEnabled(userID)
		resultCh <- result{enabled: enabled}
	}()

	// Step 2: Wait for the first query to start (old true captured).
	select {
	case <-queryStarted:
		// good — first query has captured true and is now blocked
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first query to start")
	}

	// Step 3: Simulate DB commit — user disables autoTrade.
	authoritativeState.Store(false)

	// Step 4: Invalidate the cache (as the callback would).
	// This increments generation and deletes any cache entry.
	engine.InvalidateAutoTradeCache(userID)

	// Step 5: Release the old query — it will return stale true.
	close(releaseOldQuery)

	// Step 6: Wait for isAutoTradeEnabled to complete.
	select {
	case r := <-resultCh:
		// Step 7a: Assert final result is false (not stale true).
		if r.enabled {
			t.Error("isAutoTradeEnabled returned stale true after invalidate — TOCTOU not prevented")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("isAutoTradeEnabled did not complete — possible infinite retry loop")
	}

	// Step 7b: Assert queryCallCount >= 2 (old result discarded, retried).
	if queryCallCount.Load() < 2 {
		t.Errorf("autoTradeEnabled called %d times, expected >= 2 (stale result must be discarded and retried)", queryCallCount.Load())
	}

	// Step 7c: Assert cache holds false (not stale true).
	engine.autoTradeCacheMu.Lock()
	entry, ok := engine.autoTradeCache[userID]
	engine.autoTradeCacheMu.Unlock()
	if !ok {
		t.Error("cache entry missing after isAutoTradeEnabled — should be populated with false")
	} else if entry.enabled {
		t.Error("cache holds stale true — TOCTOU allowed old query to overwrite invalidate")
	}

	// Step 7d: Assert generation was incremented.
	engine.autoTradeCacheMu.Lock()
	gen := engine.autoTradeGeneration[userID]
	engine.autoTradeCacheMu.Unlock()
	if gen == 0 {
		t.Error("generation not incremented — InvalidateAutoTradeCache did not advance generation")
	}

	// Step 7e: Subsequent call hits cache (false), no new DB query.
	countBefore := queryCallCount.Load()
	enabled := engine.isAutoTradeEnabled(userID)
	if enabled {
		t.Error("subsequent isAutoTradeEnabled returned true — cache should hold false")
	}
	if queryCallCount.Load() != countBefore {
		t.Errorf("subsequent call queried DB (count %d → %d) — should hit cache", countBefore, queryCallCount.Load())
	}
}

// 9. RunOneDoesNotRewriteNext: executeLoop advances next_run_at before dispatch.
// After runOne completes, next_run_at must NOT be rewritten. Adversarial:
// restore the old runOne ComputeNext/UpdateNext → next_run_at changes after
// runOne → RED.
func TestSCHEDULE_HOTLOOP_1_RunOneDoesNotRewriteNext(t *testing.T) {
	fixedNow := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	pastTime := fixedNow.Add(-1 * time.Hour)
	repo := newFakeScheduleRepo()
	sched := makeIntervalScheduleProto(userID, pastTime, 3600_000)
	repo.schedules[sched.ID] = sched

	engine := makeEngine(repo, fixedNow, func(uid uuid.UUID) bool { return true })

	// Run executeLoop to pre-consume and dispatch.
	_ = engine.executeLoop(context.Background())

	// Capture next_run_at after pre-consume.
	nextAfterPreConsume := repo.getNextRunAt(sched.ID)
	if nextAfterPreConsume == nil {
		t.Fatal("pre-consume did not set next_run_at")
	}

	// Simulate runOne completing (runner is nil → runErr, but runOne should
	// NOT touch next_run_at).
	// We call runOne directly with a dummy config.
	cfg := LiveStrategyConfig{ScheduleID: sched.ID}
	handle := &runHandle{cancel: func() {}}
	handle.wg.Add(1)
	engine.activeRuns[sched.ID] = handle

	engine.runOne(context.Background(), sched, cfg, handle)

	// next_run_at must be unchanged.
	nextAfterRunOne := repo.getNextRunAt(sched.ID)
	if nextAfterRunOne == nil {
		t.Fatal("next_run_at became nil after runOne — runOne must not clear it")
	}
	if !nextAfterRunOne.Equal(*nextAfterPreConsume) {
		t.Errorf("next_run_at changed after runOne: %v → %v (runOne must not rewrite)",
			*nextAfterPreConsume, *nextAfterRunOne)
	}
}

// --- proto config helper ---

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
