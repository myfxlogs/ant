package strategy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

func TestLIVE_ORDER_REENTRY_1_T7_FAIL_ReadAfterWriteFails(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			return 42, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return nil, errors.New("broker unavailable")
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("T7-FAIL: barrier state=%s, want outcome_unknown", state)
	}
}

// T8-REPLAY: Retained positions with old PositionsCapturedAt must NOT be fresh,
// even if receivedAt is recent. R2: zero PositionsCapturedAt = fail-closed.
// This test reproduces the real replay scenario: an old retained snapshot
// is received NOW (new receivedAt) but has old PositionsCapturedAt.
func TestLIVE_ORDER_REENTRY_1_T8_REPLAY_StalePositionsNotFresh(t *testing.T) {
	pc := NewPositionCache(zap.NewNop())
	oldTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	// R2 test A: old PositionsCapturedAt + new receivedAt → must fail.
	snap := &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(12000),
		FinancialsAuthoritative: true, FinancialsSource: "account_summary",
		CapturedAt:             now, // financials are fresh
		PositionsAuthoritative: true,
		PositionsCapturedAt:    oldTime, // positions are stale (old captured-at)
		PositionsSource:        "order_stream",
		Positions:              []mthub.PositionSnapshotItem{{Ticket: 1}},
	}
	pc.PutSnapshot(snap, now) // received NOW — new receivedAt

	// GetFreshTradingSnapshot must fail because PositionsCapturedAt is old.
	_, ok := pc.GetFreshTradingSnapshot("acct-1", now)
	if ok {
		t.Fatal("T8-REPLAY: GetFreshTradingSnapshot returned true for stale PositionsCapturedAt + new receivedAt — R2 violated")
	}

	// GetFreshPositionSnapshot must also fail.
	_, ok = pc.GetFreshPositionSnapshot("acct-1", now)
	if ok {
		t.Fatal("T8-REPLAY: GetFreshPositionSnapshot returned true for stale PositionsCapturedAt — R2 violated")
	}

	// R2 test B: zero PositionsCapturedAt + new receivedAt → must fail (fail-closed).
	pc2 := NewPositionCache(zap.NewNop())
	snap2 := &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(12000),
		FinancialsAuthoritative: true, FinancialsSource: "account_summary",
		CapturedAt:             now,
		PositionsAuthoritative: true,
		PositionsCapturedAt:    time.Time{}, // zero — no provenance
		PositionsSource:        "",
		Positions:              []mthub.PositionSnapshotItem{{Ticket: 1}},
	}
	pc2.PutSnapshot(snap2, now)
	_, ok = pc2.GetFreshTradingSnapshot("acct-1", now)
	if ok {
		t.Fatal("T8-REPLAY: GetFreshTradingSnapshot returned true for zero PositionsCapturedAt — R2 fail-closed violated")
	}
	_, ok = pc2.GetFreshPositionSnapshot("acct-1", now)
	if ok {
		t.Fatal("T8-REPLAY: GetFreshPositionSnapshot returned true for zero PositionsCapturedAt — R2 fail-closed violated")
	}
}

// T9: Position-only update does NOT refresh financials; financial-only does NOT refresh positions.
func TestLIVE_ORDER_REENTRY_1_T9_ProvenanceSeparation(t *testing.T) {
	pc := NewPositionCache(zap.NewNop())
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Initial: both fresh.
	snap := &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(12000),
		FinancialsAuthoritative: true, FinancialsSource: "account_summary",
		CapturedAt:             baseTime,
		PositionsAuthoritative: true,
		PositionsCapturedAt:    baseTime,
		PositionsSource:        "order_stream",
		Positions:              []mthub.PositionSnapshotItem{{Ticket: 1}},
	}
	pc.PutSnapshot(snap, baseTime)

	// Financial-only refresh at baseTime+10s.
	finTime := baseTime.Add(10 * time.Second)
	finRefresh := &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(11000), Equity: decimal.NewFromInt(13000),
		FinancialsAuthoritative: true, FinancialsSource: "account_summary",
		CapturedAt: finTime,
		// PositionsAuthoritative=false → financial-only.
	}
	pc.PutSnapshot(finRefresh, finTime)

	// Financials should be fresh at finTime+1s.
	now := finTime.Add(1 * time.Second)
	finSnap, ok := pc.GetFreshFinancialSnapshot("acct-1", now)
	if !ok || !finSnap.Equity.Equal(decimal.NewFromInt(13000)) {
		t.Fatalf("T9: financial refresh should update financials: ok=%v equity=%s", ok, finSnap.Equity)
	}
	// Positions should still be stale (captured at baseTime, now is finTime+1s > 90s from baseTime? No, 11s).
	// Actually baseTime+11s is within 90s, so positions are still fresh from the initial snapshot.
	// Let's test with a time past max age for positions but fresh for financials.
	staleNow := baseTime.Add(AccountSnapshotMaxAge + time.Second)
	_, posOk := pc.GetFreshPositionSnapshot("acct-1", staleNow)
	if posOk {
		t.Fatal("T9: financial-only refresh must NOT make old positions fresh at staleNow")
	}
	// Financials should still be fresh at staleNow (finTime is baseTime+10s, staleNow is baseTime+91s → 81s < 90s).
	_, finOk := pc.GetFreshFinancialSnapshot("acct-1", staleNow)
	if !finOk {
		t.Fatal("T9: financial refresh should keep financials fresh at staleNow")
	}
}

// T10-REQUEST: dispatch constructs OrderRequest.Magic = StrategyMagic(scheduleID).
func TestLIVE_ORDER_REENTRY_1_T10_REQUEST_MagicInOrderRequest(t *testing.T) {
	var capturedMagic int32
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (int64, error) {
			capturedMagic = req.Magic
			return 1, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{Ticket: 1, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	expected := strategyMagic(cfg.ScheduleID)
	if capturedMagic != expected {
		t.Fatalf("T10-REQUEST: OrderRequest.Magic=%d, want StrategyMagic=%d", capturedMagic, expected)
	}
}

// MUTATION-CLOSE: CloseOrder timeout → outcome_unknown, barrier locked.
func TestLIVE_ORDER_REENTRY_1_MUTATION_CLOSE_TimeoutStaysLocked(t *testing.T) {
	exec := &prodMockExecutor{
		closeFn: func(ctx context.Context, ticket int64, lots decimal.Decimal) error {
			return &mthub.MutationError{Phase: mthub.PhaseBroker, Cause: errors.New("context deadline exceeded")}
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{Ticket: 123, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "close", Volume: "0", ExecutedTicket: 123}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("MUTATION-CLOSE: barrier state=%s, want outcome_unknown", state)
	}
}

// MUTATION-MODIFY: Modify goes through coordinator (not just Acquire+defer Release).
// R5: read-after-write verifies SL/TP match the requested values.
func TestLIVE_ORDER_REENTRY_1_MUTATION_MODIFY_Wiring(t *testing.T) {
	exec := &prodMockExecutor{
		modifyFn: func(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// R5: return order with SL/TP matching the requested values.
			return []*mthub.OrderRecord{{
				Ticket: 123, Canonical: "EURUSD",
				StopLoss:   decimal.NewFromFloat(1.0),
				TakeProfit: decimal.NewFromFloat(2.0),
			}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "modify", Volume: "0", ExecutedTicket: 123, StopLoss: "1.0", TakeProfit: "2.0"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("MUTATION-MODIFY: barrier state=%s, want idle (confirmed+released)", state)
	}
	if got := exec.modifyCount.Load(); got != 1 {
		t.Fatalf("MUTATION-MODIFY: ModifyOrder called %d times, want 1", got)
	}
}

// MUTATION-CANCEL: Cancel goes through coordinator (not just Acquire+defer Release).
func TestLIVE_ORDER_REENTRY_1_MUTATION_CANCEL_Wiring(t *testing.T) {
	exec := &prodMockExecutor{
		deleteFn: func(ctx context.Context, ticket int64) error {
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// Ticket 123 is absent → cancel confirmed.
			return nil, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "cancel", Volume: "0", ExecutedTicket: 123}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("MUTATION-CANCEL: barrier state=%s, want idle (confirmed+released)", state)
	}
	if got := exec.deleteCount.Load(); got != 1 {
		t.Fatalf("MUTATION-CANCEL: DeleteOrder called %d times, want 1", got)
	}
}

// R3-ADVERSARIAL: Incompatible updateType must NOT confirm a mutation.
// A "modify" event with matching ticket+magic must NOT confirm a "close" action.
func TestLIVE_ORDER_REENTRY_1_R3_IncompatibleUpdateTypeNotConfirmed(t *testing.T) {
	placeStarted := make(chan struct{})
	placeProceed := make(chan struct{})
	exec := &prodMockExecutor{
		closeFn: func(ctx context.Context, ticket int64, lots decimal.Decimal) error {
			close(placeStarted)
			<-placeProceed
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// Ticket 42 still present → close NOT confirmed by read-after-write.
			return []*mthub.OrderRecord{{Ticket: 42, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()
	magic := strategyMagic(cfg.ScheduleID)

	done := make(chan struct{})
	go func() {
		defer close(done)
		sig := &antv1.StrategySignal{SignalType: "close", Volume: "0", ExecutedTicket: 42}
		srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)
	}()

	<-placeStarted
	// Push a "modify" event with matching ticket+magic — must NOT confirm close.
	publishOrderUpdate(broker, cfg.AccountID, 42, magic, "modify")
	close(placeProceed)
	<-done

	// Barrier should be outcome_unknown (close not confirmed by incompatible event
	// or by read-after-write which shows ticket still present).
	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("R3: barrier state=%s, want outcome_unknown (incompatible updateType must not confirm)", state)
	}
}

// R5-ADVERSARIAL: Modify read-after-write must verify SL/TP actually changed.
// If the order's SL/TP don't match the requested values, it's NOT confirmed.
func TestLIVE_ORDER_REENTRY_1_R5_ModifyVerifySLTPChanged(t *testing.T) {
	exec := &prodMockExecutor{
		modifyFn: func(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			// Ticket 123 present but SL/TP are OLD values (not the requested 1.0/2.0).
			return []*mthub.OrderRecord{{
				Ticket: 123, Canonical: "EURUSD",
				StopLoss:   decimal.NewFromFloat(0.5), // old SL, not 1.0
				TakeProfit: decimal.NewFromFloat(1.5), // old TP, not 2.0
			}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "modify", Volume: "0", ExecutedTicket: 123, StopLoss: "1.0", TakeProfit: "2.0"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	// R5: modify not confirmed because SL/TP don't match → outcome_unknown.
	if state := sess.barrier.State(); state != barrierOutcomeUnknown {
		t.Fatalf("R5: barrier state=%s, want outcome_unknown (SL/TP not changed)", state)
	}
}

// R5-POSITIVE: Modify read-after-write with matching SL/TP → confirmed.
func TestLIVE_ORDER_REENTRY_1_R5_ModifyMatchingSLTPConfirmed(t *testing.T) {
	exec := &prodMockExecutor{
		modifyFn: func(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error {
			return nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{
				Ticket: 123, Canonical: "EURUSD",
				StopLoss:   decimal.NewFromFloat(1.0), // matches requested
				TakeProfit: decimal.NewFromFloat(2.0), // matches requested
			}}, nil
		},
	}
	srv, _, _ := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	sig := &antv1.StrategySignal{SignalType: "modify", Volume: "0", ExecutedTicket: 123, StopLoss: "1.0", TakeProfit: "2.0"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("R5-POSITIVE: barrier state=%s, want idle (confirmed+released)", state)
	}
}

// R4-BOUNDED: Event cache must be bounded — sending >maxEventCacheEntries
// unrelated events must not cause unbounded growth. R4 rework: assert
// len(eventCache) <= maxEventCacheEntries AND FIFO eviction (oldest evicted
// first, newest retained). The previous test only checked "no panic" which
// passed even with the eviction code deleted.
func TestLIVE_ORDER_REENTRY_1_R4_EventCacheBounded(t *testing.T) {
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")

	// Send maxEventCacheEntries + 10 unrelated events (tickets 1..26).
	total := int64(maxEventCacheEntries + 10)
	for i := int64(1); i <= total; i++ {
		b.NotifyConfirmationEvent(i, 999, "open")
	}

	// R4 rework: assert the cache is bounded to maxEventCacheEntries.
	b.mu.Lock()
	cacheLen := len(b.eventCache)
	ordLen := len(b.eventCacheOrd)
	b.mu.Unlock()
	if cacheLen > maxEventCacheEntries {
		t.Fatalf("R4: eventCache len=%d, want <= %d (bounded)", cacheLen, maxEventCacheEntries)
	}
	if ordLen > maxEventCacheEntries {
		t.Fatalf("R4: eventCacheOrd len=%d, want <= %d (bounded)", ordLen, maxEventCacheEntries)
	}

	// R4 rework: assert FIFO eviction — the oldest entries (tickets 1..10)
	// must have been evicted, and the newest 16 (tickets 11..26) retained.
	b.mu.Lock()
	for i := int64(1); i <= total-int64(maxEventCacheEntries); i++ {
		key := eventCacheKey{ticket: i, magic: 999}
		if _, exists := b.eventCache[key]; exists {
			b.mu.Unlock()
			t.Fatalf("R4: ticket %d should have been evicted (FIFO), but is still in cache", i)
		}
	}
	for i := total - int64(maxEventCacheEntries) + 1; i <= total; i++ {
		key := eventCacheKey{ticket: i, magic: 999}
		if _, exists := b.eventCache[key]; !exists {
			b.mu.Unlock()
			t.Fatalf("R4: ticket %d should have been retained (FIFO), but is missing from cache", i)
		}
	}
	b.mu.Unlock()
}

// ── R7b: close_all only counts barrierConfirmed as closed ──

// R7b: A deterministic rejection in close_all must NOT be counted as "closed".
// Uses a zaptest observer to capture the "dispatchCloseAll complete" log and
// verify the `closed` field only includes confirmed closes.
// Setup: kill switch engaged → both closes are deterministically rejected.
// R7b fix: closed=0 (only confirmed). Old code: closed=2 (rejected counted).
func TestLIVE_ORDER_REENTRY_1_R7b_CloseAllDoesNotCountRejectedAsClosed(t *testing.T) {
	cfg := testLiveCfg()
	expectedMagic := strategyMagic(cfg.ScheduleID)
	exec := &prodMockExecutor{
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{
				{Ticket: 100, Canonical: "EURUSD", Magic: expectedMagic, Volume: decimal.NewFromFloat(0.1)},
				{Ticket: 200, Canonical: "EURUSD", Magic: expectedMagic, Volume: decimal.NewFromFloat(0.1)},
			}, nil
		},
	}
	srv, svc, _ := testCoordinatorSetup(exec)
	// Kill switch engaged from the start → both CloseOrder calls rejected
	// as deterministic_rejected (pre-broker).
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})

	sess := testActiveSess()

	// Use a zaptest observer to capture the completion log.
	core, recorded := observer.New(zap.InfoLevel)
	srv.log = zap.New(core)

	srv.dispatchCloseAll(context.Background(), cfg, sess)

	// Find the "dispatchCloseAll complete" log entry.
	for _, entry := range recorded.All() {
		if entry.Message == "LiveStrategyRunner: dispatchCloseAll complete" {
			closedField := entry.ContextMap()["closed"]
			// zap.Int stores as int64 in the observer's ContextMap.
			closed, ok := toInt(closedField)
			if !ok {
				t.Fatalf("R7b: 'closed' field not int, got %T", closedField)
			}
			// Both closes were rejected (kill switch) → closed must be 0.
			// R7b: only barrierConfirmed counts. Old code counted
			// deterministicRejected too → would be 2.
			if closed != 0 {
				t.Fatalf("R7b: closed=%d, want 0 (both rejected — only confirmed counts)", closed)
			}
			return
		}
	}
	t.Fatal("R7b: dispatchCloseAll complete log not found")
}

// ── R5-⑤: verifyTicketModified distinguishes unspecified from explicit zero ──

// R5-⑤-A: Explicit SL="0" (clearing stop loss) — broker returns SL=0 → confirmed.
