// mutation_coordinator_test.go — Production-wiring adversarial tests for
// LIVE-ORDER-REENTRY-1 (B8). These tests exercise the REAL production call
// chain: dispatchLiveSignal → submitOrder → coordinateMutation → broker RPC
// → PositionSnapshotBroker subscription → confirmation.
//
// All tests use channel-based synchronization — NO time.Sleep for concurrency.
// Cutting the production wiring (submitOrder→coordinateMutation or
// OnOrderUpdate→barrier) must make these tests RED.

package strategy

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mdgateway/adapter/mt4"
	"alphaforge/internal/mdgateway/adapter/mt5"
	"alphaforge/internal/mthub"
	mt4pb "alphaforge/mt4"
	mt5pb "alphaforge/mt5"
)

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

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingOpen verifies
// that MT4's PendingOpen action label ("pending_open") flows through the
// PositionSnapshotBroker and confirms an "open" barrier mutation.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingOpen(t *testing.T) {
	label := mt4.Mt4UpdateActionLabel(mt4pb.UpdateAction_UpdateAction_PendingOpen)
	if label != "pending_open" {
		t.Fatalf("MT4 PendingOpen label = %q, want %q", label, "pending_open")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	// Simulate the full pipeline: adapter label → PositionSnapshot → broker → barrier.
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("MT4 pending_open → open barrier: state=%s, want confirmed", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingClose verifies
// that MT4's PendingClose action label ("pending_close") confirms a "close" barrier.

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingClose verifies
// that MT4's PendingClose action label ("pending_close") confirms a "close" barrier.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingClose(t *testing.T) {
	label := mt4.Mt4UpdateActionLabel(mt4pb.UpdateAction_UpdateAction_PendingClose)
	if label != "pending_close" {
		t.Fatalf("MT4 PendingClose label = %q, want %q", label, "pending_close")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "close")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("MT4 pending_close → close barrier: state=%s, want confirmed", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingModify verifies
// that MT4's PendingModify action label ("pending_modify") confirms a "modify" barrier.

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingModify verifies
// that MT4's PendingModify action label ("pending_modify") confirms a "modify" barrier.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT4_PendingModify(t *testing.T) {
	label := mt4.Mt4UpdateActionLabel(mt4pb.UpdateAction_UpdateAction_PendingModify)
	if label != "pending_modify" {
		t.Fatalf("MT4 PendingModify label = %q, want %q", label, "pending_modify")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "modify")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("MT4 pending_modify → modify barrier: state=%s, want confirmed", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingOpen verifies
// that MT5's PendingOpen type label ("pending_open") confirms an "open" barrier.

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingOpen verifies
// that MT5's PendingOpen type label ("pending_open") confirms an "open" barrier.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingOpen(t *testing.T) {
	label := mt5.Mt5UpdateTypeLabel(mt5pb.UpdateType_UpdateType_PendingOpen)
	if label != "pending_open" {
		t.Fatalf("MT5 PendingOpen label = %q, want %q", label, "pending_open")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("MT5 pending_open → open barrier: state=%s, want confirmed", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingClose verifies
// that MT5's PendingClose type label ("pending_close") confirms a "close" barrier.

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingClose verifies
// that MT5's PendingClose type label ("pending_close") confirms a "close" barrier.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingClose(t *testing.T) {
	label := mt5.Mt5UpdateTypeLabel(mt5pb.UpdateType_UpdateType_PendingClose)
	if label != "pending_close" {
		t.Fatalf("MT5 PendingClose label = %q, want %q", label, "pending_close")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "close")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("MT5 pending_close → close barrier: state=%s, want confirmed", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingModify verifies
// that MT5's PendingModify type label ("modify") confirms a "modify" barrier.
// Note: MT5 maps both MarketModify and PendingModify to "modify".

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingModify verifies
// that MT5's PendingModify type label ("modify") confirms a "modify" barrier.
// Note: MT5 maps both MarketModify and PendingModify to "modify".
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_MT5_PendingModify(t *testing.T) {
	label := mt5.Mt5UpdateTypeLabel(mt5pb.UpdateType_UpdateType_PendingModify)
	if label != "modify" {
		t.Fatalf("MT5 PendingModify label = %q, want %q", label, "modify")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "modify")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierConfirmed {
		t.Fatalf("MT5 modify → modify barrier: state=%s, want confirmed", state)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_FullBrokerPath verifies
// the complete pipeline: adapter label → publishOrderUpdate (real broker) →
// confirmation listener → barrier. This is a true integration test that
// exercises the REAL PositionSnapshotBroker subscription wiring.

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_FullBrokerPath verifies
// the complete pipeline: adapter label → publishOrderUpdate (real broker) →
// confirmation listener → barrier. This is a true integration test that
// exercises the REAL PositionSnapshotBroker subscription wiring.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_FullBrokerPath(t *testing.T) {
	exec := &prodMockExecutor{
		placeFn: func(ctx context.Context, req *mthub.OrderRequest) (*mthub.OrderRecord, error) {
			return &mthub.OrderRecord{Ticket: 42, State: mthub.OrderStateOpen}, nil
		},
		fetchFn: func(ctx context.Context) ([]*mthub.OrderRecord, error) {
			return []*mthub.OrderRecord{{Ticket: 42, Canonical: "EURUSD"}}, nil
		},
	}
	srv, _, broker := testCoordinatorSetup(exec)
	cfg := testLiveCfg()
	sess := testActiveSess()

	// Use the real MT4 PendingOpen label — this is what the adapter would emit.
	label := mt4.Mt4UpdateActionLabel(mt4pb.UpdateAction_UpdateAction_PendingOpen)

	// Publish the confirmation through the REAL broker, simulating the
	// adapter → pipeline_callbacks.publishPositionSnapshot → broker path.
	// Use a goroutine to publish after the barrier has acquired.
	go func() {
		// WaitState here is defensive synchronization: it ensures the goroutine
		// publishes after the barrier enters submitting. In the current
		// synchronous dispatchLiveSignal model, time.Sleep(0) would also work
		// because the main goroutine blocks inside dispatchLiveSignal. However,
		// WaitState is correct regardless of dispatch model (sync or async),
		// making the test robust to future refactors. The adversarial proof for
		// WaitState necessity in a truly-async scenario is in
		// TestLIVE_ORDER_REENTRY_1_R4_Recovery_CloseConfirmed (recovery
		// goroutine is async — WaitState → time.Sleep(0) mutation goes RED).
		// (R4 S3: no time.Sleep — deterministic sync via cond.Wait.)
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
		sess.barrier.WaitState(waitCtx, barrierSubmitting)
		waitCancel()
		publishOrderUpdate(broker, cfg.AccountID, 42, strategyMagic(cfg.ScheduleID), label)
	}()

	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, sess)

	if state := sess.barrier.State(); state != barrierIdle {
		t.Fatalf("FullBrokerPath: barrier state=%s, want idle (confirmed+released)", state)
	}
	if got := exec.placeCount.Load(); got != 1 {
		t.Fatalf("FullBrokerPath: PlaceOrder called %d times, want 1", got)
	}
}

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_IncompatibleRejects verifies
// that an incompatible adapter label does NOT confirm the barrier. E.g. a
// "modify" event cannot confirm an "open" action.

// TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_IncompatibleRejects verifies
// that an incompatible adapter label does NOT confirm the barrier. E.g. a
// "modify" event cannot confirm an "open" action.
func TestLIVE_ORDER_REENTRY_1_R4_AdapterLabelPipeline_IncompatibleRejects(t *testing.T) {
	// MT4 PositionModify label is "modify" — should NOT confirm an "open" barrier.
	label := mt4.Mt4UpdateActionLabel(mt4pb.UpdateAction_UpdateAction_PositionModify)
	if label != "modify" {
		t.Fatalf("MT4 PositionModify label = %q, want %q", label, "modify")
	}
	b := NewTradeBarrier(zap.NewNop())
	b.Acquire("client-1", 12345, "open")
	b.NotifyBrokerAccepted(42)
	b.NotifyConfirmationEvent(42, 12345, label)
	if state := b.State(); state != barrierAcceptedUnconfirmed {
		t.Fatalf("Incompatible: barrier state=%s, want accepted_unconfirmed (not confirmed by modify event for open action)", state)
	}
}

// ============================================================================
// ④-②: Integration tests — outcomeUnknown reconciliation recovery
// ============================================================================

// TestLIVE_ORDER_REENTRY_1_R4_Recovery_CloseConfirmed verifies that after
// outcomeUnknown for a close mutation, the background recovery goroutine
// reconciles via OpenedOrders (ticket absent = close succeeded) and releases
// the barrier + clears the circuit breaker.
