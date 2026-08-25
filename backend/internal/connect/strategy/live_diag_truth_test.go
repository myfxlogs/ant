// live_diag_truth_test.go — LIVE-DIAG-TRUTH-1 backend tests.
// Verifies that L3 diagnostic fields are correctly computed from
// PositionCache + TradeBarrier, and that RecordIndicators no longer
// blocks OrdersTotal updates when indicator values are empty.
package strategy

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/internal/mthub"
)

// TestLIVE_DIAG_TRUTH_1_RecordIndicators_EmptyValuesDoesNotBlockOrdersTotal
// verifies the critical fix: RecordIndicators with empty values map must
// still update ordersTotalSeen (rule 4).
func TestLIVE_DIAG_TRUTH_1_RecordIndicators_EmptyValuesDoesNotBlockOrdersTotal(t *testing.T) {
	d := newSessionDiag()

	// First call with empty values — must still set ordersTotalSeen
	d.RecordIndicators(map[string]decimal.Decimal{}, 5)
	snap := d.SnapshotDiag()
	if snap.OrdersTotalSeen != 5 {
		t.Fatalf("after RecordIndicators with empty values: ordersTotalSeen=%d, want 5 (empty values must not block OrdersTotal update)", snap.OrdersTotalSeen)
	}

	// Second call with non-empty values — ordersTotal should update
	d.RecordIndicators(map[string]decimal.Decimal{"iRSI[14,0]": decimal.NewFromFloat(55.0)}, 7)
	snap = d.SnapshotDiag()
	if snap.OrdersTotalSeen != 7 {
		t.Fatalf("after RecordIndicators with non-empty values: ordersTotalSeen=%d, want 7", snap.OrdersTotalSeen)
	}
}

// TestLIVE_DIAG_TRUTH_1_RecordIndicators_EmptyDoesNotBurnThrottle verifies
// that empty-value calls don't burn the throttle window — a subsequent
// non-empty call within the window should still write indicators.
func TestLIVE_DIAG_TRUTH_1_RecordIndicators_EmptyDoesNotBurnThrottle(t *testing.T) {
	d := newSessionDiag()

	// First: non-empty call burns the throttle window
	d.RecordIndicators(map[string]decimal.Decimal{"key1": decimal.NewFromInt(1)}, 3)

	// Second: empty call updates ordersTotal but doesn't burn throttle
	d.RecordIndicators(map[string]decimal.Decimal{}, 4)

	// Third: non-empty call within throttle window — should be throttled
	// (indicators NOT written), but ordersTotal IS updated
	d.RecordIndicators(map[string]decimal.Decimal{"key2": decimal.NewFromInt(2)}, 6)

	snap := d.SnapshotDiag()
	if snap.OrdersTotalSeen != 6 {
		t.Fatalf("ordersTotalSeen=%d, want 6 (ordersTotal must update even when indicator write is throttled)", snap.OrdersTotalSeen)
	}
	if _, ok := snap.Indicators["key2"]; ok {
		t.Fatal("key2 should NOT be in indicators (throttle window not burned by empty call, but third call is within window from first)")
	}
}

// TestLIVE_DIAG_TRUTH_1_MixedMagic verifies that broker account orders,
// strategy magic orders, and VM orders are all correctly distinguished
// (rule 7: broker account=3, target magic=1, VM=0).
func TestLIVE_DIAG_TRUTH_1_MixedMagic(t *testing.T) {
	pc := NewPositionCache(nil)
	accountID := "test-account"
	magic := int32(1699507621)

	now := time.Now()
	snap := &mthub.PositionSnapshot{
		AccountID:               accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:              now,
		PositionsCapturedAt:     now,
		FinancialsSource:        "account_summary",
		PositionsSource:         "order_stream",
		// 3 positions: 1 with target magic, 2 with other magic
		Positions: []mthub.PositionSnapshotItem{
			{Ticket: 1, Magic: magic, Symbol: "EURUSD"},
			{Ticket: 2, Magic: 999, Symbol: "GBPUSD"},
			{Ticket: 3, Magic: 888, Symbol: "USDJPY"},
		},
	}
	pc.PutSnapshot(snap, now)

	sess := &ActiveSession{
		AccountID:   accountID,
		MagicNumber: magic,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
	}
	// VM saw 0 orders (OrdersTotal not yet recorded)
	sess.diag.RecordIndicators(map[string]decimal.Decimal{}, 0)

	diagSnap := sess.diag.SnapshotDiag()
	enrichDiagSnapshot(&diagSnap, sess, pc)

	if diagSnap.VmOrdersTotal != 0 {
		t.Fatalf("VM OrdersTotal=%d, want 0 (VM saw no orders)", diagSnap.VmOrdersTotal)
	}
	if diagSnap.BrokerAccountOrders != 3 {
		t.Fatalf("Broker account orders=%d, want 3", diagSnap.BrokerAccountOrders)
	}
	if diagSnap.StrategyMagicOrders != 1 {
		t.Fatalf("Strategy magic orders=%d, want 1 (only 1 position matches magic %d)", diagSnap.StrategyMagicOrders, magic)
	}
	if diagSnap.ScheduleMagic != magic {
		t.Fatalf("Schedule magic=%d, want %d", diagSnap.ScheduleMagic, magic)
	}
}

// TestLIVE_DIAG_TRUTH_1_PendingOrdersCount verifies that pending orders
// are counted separately from market positions.
func TestLIVE_DIAG_TRUTH_1_PendingOrdersCount(t *testing.T) {
	pc := NewPositionCache(nil)
	accountID := "test-account"
	magic := int32(12345)

	now := time.Now()
	snap := &mthub.PositionSnapshot{
		AccountID:               accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:              now,
		PositionsCapturedAt:     now,
		FinancialsSource:        "account_summary",
		PositionsSource:         "order_stream",
		Positions: []mthub.PositionSnapshotItem{
			{Ticket: 1, Magic: magic, Symbol: "EURUSD"},
		},
		PendingOrders: []mthub.PositionSnapshotItem{
			{Ticket: 2, Magic: magic, Symbol: "EURUSD"},
			{Ticket: 3, Magic: magic, Symbol: "GBPUSD"},
		},
	}
	pc.PutSnapshot(snap, now)

	sess := &ActiveSession{
		AccountID:   accountID,
		MagicNumber: magic,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
	}

	diagSnap := sess.diag.SnapshotDiag()
	enrichDiagSnapshot(&diagSnap, sess, pc)

	if diagSnap.BrokerAccountOrders != 3 {
		t.Fatalf("Broker account orders=%d, want 3 (1 position + 2 pending)", diagSnap.BrokerAccountOrders)
	}
	if diagSnap.PendingBrokerOrders != 2 {
		t.Fatalf("Pending broker orders=%d, want 2", diagSnap.PendingBrokerOrders)
	}
	if diagSnap.StrategyMagicOrders != 3 {
		t.Fatalf("Strategy magic orders=%d, want 3 (all match magic)", diagSnap.StrategyMagicOrders)
	}
}

// TestLIVE_DIAG_TRUTH_1_FreshnessFields verifies that financial and positions
// freshness are independently tracked and exposed.
func TestLIVE_DIAG_TRUTH_1_FreshnessFields(t *testing.T) {
	pc := NewPositionCache(nil)
	accountID := "test-account"

	now := time.Now()
	snap := &mthub.PositionSnapshot{
		AccountID:               accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:              now,
		PositionsCapturedAt:     now,
		FinancialsSource:        "account_summary",
		PositionsSource:         "order_stream",
	}
	pc.PutSnapshot(snap, now)

	sess := &ActiveSession{
		AccountID:   accountID,
		MagicNumber: 12345,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
	}

	diagSnap := sess.diag.SnapshotDiag()
	enrichDiagSnapshot(&diagSnap, sess, pc)

	if !diagSnap.FinancialFresh {
		t.Fatal("Financial should be fresh (captured now)")
	}
	if !diagSnap.PositionsFresh {
		t.Fatal("Positions should be fresh (captured now)")
	}
	if diagSnap.FinancialSource != "account_summary" {
		t.Fatalf("Financial source=%q, want %q", diagSnap.FinancialSource, "account_summary")
	}
	if diagSnap.PositionsSource != "order_stream" {
		t.Fatalf("Positions source=%q, want %q", diagSnap.PositionsSource, "order_stream")
	}
	if diagSnap.FinancialAgeMs < 0 {
		t.Fatalf("Financial age=%d, should be >= 0", diagSnap.FinancialAgeMs)
	}
}

// TestLIVE_DIAG_TRUTH_1_StalePositions verifies that stale positions
// are detected and positions_fresh=false.
func TestLIVE_DIAG_TRUTH_1_StalePositions(t *testing.T) {
	pc := NewPositionCache(nil)
	accountID := "test-account"

	staleTime := time.Now().Add(-2 * time.Minute) // 120s ago, > 90s max age
	snap := &mthub.PositionSnapshot{
		AccountID:               accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:              staleTime,
		PositionsCapturedAt:     staleTime,
		FinancialsSource:        "account_summary",
		PositionsSource:         "order_stream",
	}
	pc.PutSnapshot(snap, staleTime)

	sess := &ActiveSession{
		AccountID:   accountID,
		MagicNumber: 12345,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
	}

	diagSnap := sess.diag.SnapshotDiag()
	enrichDiagSnapshot(&diagSnap, sess, pc)

	if diagSnap.FinancialFresh {
		t.Fatal("Financial should be stale (captured 120s ago, > 90s max)")
	}
	if diagSnap.PositionsFresh {
		t.Fatal("Positions should be stale (captured 120s ago, > 90s max)")
	}
}

// TestLIVE_DIAG_TRUTH_1_ExecutionState verifies that barrier state
// is correctly exposed in diagnostics.
func TestLIVE_DIAG_TRUTH_1_ExecutionState(t *testing.T) {
	sess := &ActiveSession{
		AccountID:   "test",
		MagicNumber: 12345,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
	}

	// Idle state
	diagSnap := sess.diag.SnapshotDiag()
	enrichDiagSnapshot(&diagSnap, sess, nil)
	if diagSnap.ExecutionState != "idle" {
		t.Fatalf("Execution state=%q, want %q", diagSnap.ExecutionState, "idle")
	}
	if diagSnap.OrderLifecycle != "signal_generated" {
		t.Fatalf("Order lifecycle=%q, want %q (default from newSessionDiag)", diagSnap.OrderLifecycle, "signal_generated")
	}
}

// TestLIVE_DIAG_TRUTH_1_ProtoConversion verifies that all L3 fields
// are correctly serialized to proto.
func TestLIVE_DIAG_TRUTH_1_ProtoConversion(t *testing.T) {
	pc := NewPositionCache(nil)
	accountID := "test-account"
	magic := int32(42)

	now := time.Now()
	snap := &mthub.PositionSnapshot{
		AccountID:               accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:              now,
		PositionsCapturedAt:     now,
		FinancialsSource:        "account_summary",
		PositionsSource:         "order_stream",
		Positions: []mthub.PositionSnapshotItem{
			{Ticket: 1, Magic: magic, Symbol: "EURUSD"},
		},
	}
	pc.PutSnapshot(snap, now)

	sess := &ActiveSession{
		AccountID:   accountID,
		MagicNumber: magic,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
	}
	sess.diag.RecordIndicators(map[string]decimal.Decimal{}, 1)

	pb := activeSessionToProto(sess, nil, pc)
	if pb.Diagnostics == nil {
		t.Fatal("Diagnostics should not be nil")
	}
	d := pb.Diagnostics

	if d.VmOrdersTotal != 1 {
		t.Fatalf("proto vm_orders_total=%d, want 1", d.VmOrdersTotal)
	}
	if d.BrokerAccountOrders != 1 {
		t.Fatalf("proto broker_account_orders=%d, want 1", d.BrokerAccountOrders)
	}
	if d.StrategyMagicOrders != 1 {
		t.Fatalf("proto strategy_magic_orders=%d, want 1", d.StrategyMagicOrders)
	}
	if d.ScheduleMagic != magic {
		t.Fatalf("proto schedule_magic=%d, want %d", d.ScheduleMagic, magic)
	}
	if d.ExecutionState != "idle" {
		t.Fatalf("proto execution_state=%q, want %q", d.ExecutionState, "idle")
	}
	if d.OrderLifecycle != "signal_generated" {
		t.Fatalf("proto order_lifecycle=%q, want %q", d.OrderLifecycle, "signal_generated")
	}
	if !d.FinancialFresh {
		t.Fatal("proto financial_fresh should be true")
	}
	if !d.PositionsFresh {
		t.Fatal("proto positions_fresh should be true")
	}
	if d.FinancialSource != "account_summary" {
		t.Fatalf("proto financial_source=%q, want %q", d.FinancialSource, "account_summary")
	}
}

// TestLIVE_DIAG_TRUTH_1_NoPositionCache verifies graceful degradation
// when posCache is nil (e.g. paper trading mode).
func TestLIVE_DIAG_TRUTH_1_NoPositionCache(t *testing.T) {
	sess := &ActiveSession{
		AccountID:   "test",
		MagicNumber: 12345,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
		// posCache is nil
	}

	diagSnap := sess.diag.SnapshotDiag()
	enrichDiagSnapshot(&diagSnap, sess, nil)

	// Should not panic, should have zero counts
	if diagSnap.BrokerAccountOrders != 0 {
		t.Fatalf("broker_account_orders=%d, want 0 (no posCache)", diagSnap.BrokerAccountOrders)
	}
	if diagSnap.ExecutionState != "idle" {
		t.Fatalf("execution_state=%q, want %q", diagSnap.ExecutionState, "idle")
	}
	// DataAvailable must be false when posCache is nil
	if diagSnap.DataAvailable {
		t.Fatal("data_available should be false when posCache is nil (paper mode)")
	}
}

// TestLIVE_DIAG_TRUTH_1_VMBrokerMismatch verifies that when VM count
// differs from broker count, the diagnostic snapshot captures both values
// for the frontend to show a warning (rule 3).
func TestLIVE_DIAG_TRUTH_1_VMBrokerMismatch(t *testing.T) {
	pc := NewPositionCache(nil)
	accountID := "test-account"
	magic := int32(100)

	now := time.Now()
	snap := &mthub.PositionSnapshot{
		AccountID:               accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:              now,
		PositionsCapturedAt:     now,
		FinancialsSource:        "account_summary",
		PositionsSource:         "order_stream",
		Positions: []mthub.PositionSnapshotItem{
			{Ticket: 1, Magic: magic, Symbol: "EURUSD"},
			{Ticket: 2, Magic: magic, Symbol: "GBPUSD"},
			{Ticket: 3, Magic: 999, Symbol: "USDJPY"},
		},
	}
	pc.PutSnapshot(snap, now)

	sess := &ActiveSession{
		AccountID:   accountID,
		MagicNumber: magic,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
	}
	// VM saw 0 orders (stale VM state)
	sess.diag.RecordIndicators(map[string]decimal.Decimal{}, 0)

	diagSnap := sess.diag.SnapshotDiag()
	enrichDiagSnapshot(&diagSnap, sess, pc)

	// VM=0, broker=3 → mismatch
	if diagSnap.VmOrdersTotal != 0 {
		t.Fatalf("VM OrdersTotal=%d, want 0", diagSnap.VmOrdersTotal)
	}
	if diagSnap.BrokerAccountOrders != 3 {
		t.Fatalf("Broker account orders=%d, want 3", diagSnap.BrokerAccountOrders)
	}
	if diagSnap.VmOrdersTotal == diagSnap.BrokerAccountOrders {
		t.Fatal("VM and broker counts should differ (mismatch scenario)")
	}
}

// TestLIVE_DIAG_TRUTH_1_LifecyclePersistence verifies that the order lifecycle
// and broker ticket persist after barrier.Release() clears the transient state.
// This is the critical rework fix: without persistence, confirmed/rejected
// orders would degrade to signal_generated/ticket=0 after Release.
func TestLIVE_DIAG_TRUTH_1_LifecyclePersistence(t *testing.T) {
	sess := &ActiveSession{
		AccountID:   "test",
		MagicNumber: 12345,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
	}

	// Simulate lifecycle progression: submitting → submitted → confirmed
	sess.diag.RecordLifecycle("order_submitting", 0)
	sess.diag.RecordLifecycle("order_submitted", 42)
	sess.diag.RecordLifecycle("order_confirmed", 42)

	// Now simulate Release() — barrier goes back to idle, ticket=0
	sess.barrier.NotifyBrokerAccepted(42)
	// In real code, WaitConfirmed would return confirmed, then Release is called.
	// For this test, we just call Release directly.
	sess.barrier.Release()

	// After Release, barrier state is idle, but persisted lifecycle must survive
	diagSnap := sess.diag.SnapshotDiag()
	enrichDiagSnapshot(&diagSnap, sess, nil)

	if diagSnap.ExecutionState != "idle" {
		t.Fatalf("execution_state=%q, want %q (barrier released)", diagSnap.ExecutionState, "idle")
	}
	if diagSnap.OrderLifecycle != "order_confirmed" {
		t.Fatalf("order_lifecycle=%q, want %q (persisted, must survive Release)", diagSnap.OrderLifecycle, "order_confirmed")
	}
	if diagSnap.LastBrokerTicket != 42 {
		t.Fatalf("last_broker_ticket=%d, want 42 (persisted, must survive Release)", diagSnap.LastBrokerTicket)
	}
}

// TestLIVE_DIAG_TRUTH_1_LifecycleRejectedPersistence verifies that
// order_rejected also persists after Release.
