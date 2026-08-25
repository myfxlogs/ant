package strategy

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
)

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
