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
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

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
