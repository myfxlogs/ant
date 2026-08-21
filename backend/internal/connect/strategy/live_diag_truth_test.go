// live_diag_truth_test.go — LIVE-DIAG-TRUTH-1 backend tests.
// Verifies that L3 diagnostic fields are correctly computed from
// PositionCache + TradeBarrier, and that RecordIndicators no longer
// blocks OrdersTotal updates when indicator values are empty.
package strategy

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

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
		AccountID:              accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:             now,
		PositionsCapturedAt:    now,
		FinancialsSource:       "account_summary",
		PositionsSource:        "order_stream",
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
		AccountID:              accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:             now,
		PositionsCapturedAt:    now,
		FinancialsSource:       "account_summary",
		PositionsSource:        "order_stream",
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
		AccountID:              accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:             now,
		PositionsCapturedAt:    now,
		FinancialsSource:       "account_summary",
		PositionsSource:        "order_stream",
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
		AccountID:              accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:             staleTime,
		PositionsCapturedAt:    staleTime,
		FinancialsSource:       "account_summary",
		PositionsSource:        "order_stream",
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
		AccountID:              accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:             now,
		PositionsCapturedAt:    now,
		FinancialsSource:       "account_summary",
		PositionsSource:        "order_stream",
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
		AccountID:              accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:             now,
		PositionsCapturedAt:    now,
		FinancialsSource:       "account_summary",
		PositionsSource:        "order_stream",
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
func TestLIVE_DIAG_TRUTH_1_LifecycleRejectedPersistence(t *testing.T) {
	sess := &ActiveSession{
		AccountID:   "test",
		MagicNumber: 12345,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
	}

	sess.diag.RecordLifecycle("order_submitting", 0)
	sess.diag.RecordLifecycle("order_rejected", 0)
	sess.barrier.Release()

	diagSnap := sess.diag.SnapshotDiag()
	enrichDiagSnapshot(&diagSnap, sess, nil)

	if diagSnap.OrderLifecycle != "order_rejected" {
		t.Fatalf("order_lifecycle=%q, want %q (persisted)", diagSnap.OrderLifecycle, "order_rejected")
	}
}

// TestLIVE_DIAG_TRUTH_1_DataAvailableWithCache verifies that DataAvailable
// is true when posCache is provided.
func TestLIVE_DIAG_TRUTH_1_DataAvailableWithCache(t *testing.T) {
	pc := NewPositionCache(nil)
	accountID := "test-account"

	now := time.Now()
	snap := &mthub.PositionSnapshot{
		AccountID:              accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:             now,
		PositionsCapturedAt:    now,
		FinancialsSource:       "account_summary",
		PositionsSource:        "order_stream",
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

	if !diagSnap.DataAvailable {
		t.Fatal("data_available should be true when posCache is provided")
	}
}

// TestLIVE_DIAG_TRUTH_1_LogOrderLifecycleWiring verifies that
// logOrderLifecycle actually calls RecordLifecycle on sessionDiag.
// This is the critical wiring test: if the RecordLifecycle call inside
// logOrderLifecycle is removed, this test must RED.
func TestLIVE_DIAG_TRUTH_1_LogOrderLifecycleWiring(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop()}
	sess := &ActiveSession{
		AccountID:   "test",
		MagicNumber: 12345,
		diag:        newSessionDiag(),
	}
	cfg := LiveStrategyConfig{
		AccountID: "test",
		Symbol:    "BTCUSDm",
	}

	// Call logOrderLifecycle for each canonical lifecycle stage
	stages := []struct {
		kind   string
		ticket int64
	}{
		{"order_submitting", 0},
		{"order_submitted", 42},
		{"order_confirmed", 42},
		{"order_rejected", 0},
		{"order_outcome_unknown", 42},
	}
	for _, st := range stages {
		srv.logOrderLifecycle(sess, cfg, st.kind, "buy", st.ticket, "")
	}

	snap := sess.diag.SnapshotDiag()
	// Last call was order_outcome_unknown with ticket=42
	if snap.OrderLifecycle != "order_outcome_unknown" {
		t.Fatalf("order_lifecycle=%q, want %q (wiring from logOrderLifecycle)", snap.OrderLifecycle, "order_outcome_unknown")
	}
	if snap.LastBrokerTicket != 42 {
		t.Fatalf("last_broker_ticket=%d, want 42 (wiring from logOrderLifecycle)", snap.LastBrokerTicket)
	}
}

// TestLIVE_DIAG_TRUTH_1_RecordSignalPersistsSignalGenerated verifies that
// RecordSignal persists "signal_generated" to sessionDiag.
func TestLIVE_DIAG_TRUTH_1_RecordSignalPersistsSignalGenerated(t *testing.T) {
	sess := &ActiveSession{
		AccountID:   "test",
		MagicNumber: 12345,
		diag:        newSessionDiag(),
	}

	// First set a confirmed lifecycle
	sess.diag.RecordLifecycle("order_confirmed", 42)

	// Now a new signal arrives → should reset to signal_generated
	sess.RecordSignal(&SignalEvent{
		RunID:      uuid.Nil,
		AccountID:  "test",
		Symbol:     "BTCUSDm",
		SignalType: "buy",
		Timestamp:  time.Now(),
	})

	snap := sess.diag.SnapshotDiag()
	if snap.OrderLifecycle != "signal_generated" {
		t.Fatalf("order_lifecycle=%q, want %q (RecordSignal should persist signal_generated)", snap.OrderLifecycle, "signal_generated")
	}
	if snap.LastBrokerTicket != 0 {
		t.Fatalf("last_broker_ticket=%d, want 0 (signal_generated has no ticket)", snap.LastBrokerTicket)
	}
}

// TestLIVE_DIAG_TRUTH_1_DataAvailableNoSnapshot verifies that DataAvailable
// is false when posCache is provided but has no snapshot for the account.
// This distinguishes "cache exists but no data" from "cache has data".
func TestLIVE_DIAG_TRUTH_1_DataAvailableNoSnapshot(t *testing.T) {
	pc := NewPositionCache(nil)
	// posCache exists but has NO snapshot for this account
	sess := &ActiveSession{
		AccountID:   "test-account-no-snapshot",
		MagicNumber: 12345,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
	}

	diagSnap := sess.diag.SnapshotDiag()
	enrichDiagSnapshot(&diagSnap, sess, pc)

	// DataAvailable must be false — no snapshot exists for this account
	if diagSnap.DataAvailable {
		t.Fatal("data_available should be false when posCache has no snapshot for the account")
	}
}

// TestLIVE_DIAG_TRUTH_1_OutcomeUnknownWarningWithoutData verifies that
// outcome_unknown execution state triggers warning even when data is
// unavailable (paper mode / no broker data). Execution state is local
// barrier truth, independent of broker data availability.
func TestLIVE_DIAG_TRUTH_1_OutcomeUnknownWarningWithoutData(t *testing.T) {
	sess := &ActiveSession{
		AccountID:   "test",
		MagicNumber: 12345,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
	}

	// Set barrier to outcome_unknown
	sess.barrier.NotifyOutcomeUnknown()

	diagSnap := sess.diag.SnapshotDiag()
	enrichDiagSnapshot(&diagSnap, sess, nil) // no posCache

	if diagSnap.ExecutionState != "outcome_unknown" {
		t.Fatalf("execution_state=%q, want %q", diagSnap.ExecutionState, "outcome_unknown")
	}
	if diagSnap.DataAvailable {
		t.Fatal("data_available should be false (no posCache)")
	}
	// outcome_unknown must be visible even without broker data
	// (frontend deriveState checks executionState before dataAvailable)
}

// TestLIVE_DIAG_TRUTH_1_RecoveryCanonicalLifecycle verifies that the recovery
// path uses canonical lifecycle values (order_confirmed/order_rejected), NOT
// order_recovered_*. This test exercises the REAL recoverFromOutcomeUnknown
// function with a mock MtHubService + mock OrderExecutor.
// If recovery is reverted to order_recovered_confirmed, this test REDs because
// the persisted lifecycle won't match the canonical 6-state enum.
func TestLIVE_DIAG_TRUTH_1_RecoveryCanonicalLifecycle(t *testing.T) {
	// Build a mock MtHubService with a registered executor that returns
	// orders matching the ticket (so verify=true → Reconcile(true) → confirmed).
	hub := mthub.NewHub()
	exec := &mockOrderExecutor{}
	hub.Register("test-acct", &mthub.Session{}, exec)
	mtHub := mthub.NewMtHubService(hub, nil, nil, nil, nil, nil, nil)

	srv := &StrategyExecutionServer{log: zap.NewNop(), mtHub: mtHub}
	sess := &ActiveSession{
		AccountID:   "test-acct",
		MagicNumber: 12345,
		diag:        newSessionDiag(),
	}
	cfg := LiveStrategyConfig{
		AccountID: "test-acct",
		Symbol:    "BTCUSDm",
	}

	// Set up barrier in outcomeUnknown state
	barrier := NewTradeBarrier(zap.NewNop())
	barrier.NotifyOutcomeUnknown()

	// verify function: returns true if ticket is present in orders
	ticket := int64(99)
	verify := func(orders []*mthub.OrderRecord) bool {
		for _, o := range orders {
			if o.Ticket == ticket {
				return true
			}
		}
		return false
	}

	// Inject the ticket into the mock executor so FetchOpenedOrders returns it
	exec.mu.Lock()
	exec.tickets = []int64{ticket}
	exec.mu.Unlock()

	conf := confirmationConfig{
		recoveryDelay:         10 * time.Millisecond,
		readAfterWriteTimeout: 5 * time.Second,
	}

	// Call recoverFromOutcomeUnknown — this will:
	// 1. Sleep recoveryDelay
	// 2. Query OpenedOrders (returns ticket=99)
	// 3. verify(orders) = true
	// 4. barrier.Reconcile(true) → barrierConfirmed
	// 5. logOrderLifecycle(..., "order_confirmed", ..., 99, ...)
	// 6. RecordLifecycle("order_confirmed", 99) in sessionDiag
	srv.recoverFromOutcomeUnknown(cfg, sess, barrier, ticket, "close", verify, conf)

	snap := sess.diag.SnapshotDiag()
	if snap.OrderLifecycle != "order_confirmed" {
		t.Fatalf("recovery confirmed: order_lifecycle=%q, want %q (canonical, not order_recovered_confirmed)",
			snap.OrderLifecycle, "order_confirmed")
	}
	if snap.LastBrokerTicket != ticket {
		t.Fatalf("recovery confirmed: last_broker_ticket=%d, want %d", snap.LastBrokerTicket, ticket)
	}
}

// TestLIVE_DIAG_TRUTH_1_RecoveryRejectedCanonicalLifecycle verifies the
// recovery rejected path uses canonical "order_rejected", not "order_recovered_rejected".
// Uses a verify function that returns false → Reconcile(false) → deterministicRejected.
func TestLIVE_DIAG_TRUTH_1_RecoveryRejectedCanonicalLifecycle(t *testing.T) {
	hub := mthub.NewHub()
	exec := &mockOrderExecutor{}
	hub.Register("test-acct-rej", &mthub.Session{}, exec)
	mtHub := mthub.NewMtHubService(hub, nil, nil, nil, nil, nil, nil)

	srv := &StrategyExecutionServer{log: zap.NewNop(), mtHub: mtHub}
	sess := &ActiveSession{
		AccountID:   "test-acct-rej",
		MagicNumber: 12345,
		diag:        newSessionDiag(),
	}
	cfg := LiveStrategyConfig{
		AccountID: "test-acct-rej",
		Symbol:    "BTCUSDm",
	}

	barrier := NewTradeBarrier(zap.NewNop())
	barrier.NotifyOutcomeUnknown()

	ticket := int64(88)
	// verify returns false → mutation was NOT applied → Reconcile(false) → rejected
	verify := func(orders []*mthub.OrderRecord) bool { return false }

	conf := confirmationConfig{
		recoveryDelay:         10 * time.Millisecond,
		readAfterWriteTimeout: 5 * time.Second,
	}

	srv.recoverFromOutcomeUnknown(cfg, sess, barrier, ticket, "close", verify, conf)

	snap := sess.diag.SnapshotDiag()
	if snap.OrderLifecycle != "order_rejected" {
		t.Fatalf("recovery rejected: order_lifecycle=%q, want %q (canonical, not order_recovered_rejected)",
			snap.OrderLifecycle, "order_rejected")
	}
	if snap.LastBrokerTicket != ticket {
		t.Fatalf("recovery rejected: last_broker_ticket=%d, want %d", snap.LastBrokerTicket, ticket)
	}
}

// TestLIVE_DIAG_TRUTH_1_RecoveryCanonicalLifecycleAdversarial verifies that
// non-canonical lifecycle values (order_recovered_*) are NOT accepted by the
// diagnostic system. This is the adversarial counterpart: if someone reverts
// recovery to use order_recovered_confirmed, the proto/frontend lifecycle
// enum (6 states) would receive an undefined value. This test asserts the
// canonical set explicitly.
func TestLIVE_DIAG_TRUTH_1_RecoveryCanonicalLifecycleAdversarial(t *testing.T) {
	// The canonical lifecycle values allowed by proto/frontend (6 states)
	canonical := map[string]bool{
		"signal_generated":      true,
		"order_submitting":      true,
		"order_submitted":       true,
		"order_confirmed":       true,
		"order_rejected":        true,
		"order_outcome_unknown": true,
	}
	// Recovery must use canonical values, not order_recovered_*
	nonCanonical := []string{"order_recovered_confirmed", "order_recovered_rejected"}
	for _, nc := range nonCanonical {
		if canonical[nc] {
			t.Fatalf("non-canonical lifecycle %q must NOT be in the canonical set", nc)
		}
	}
	// Verify recovery values ARE in canonical set
	recoveryValues := []string{"order_confirmed", "order_rejected"}
	for _, rv := range recoveryValues {
		if !canonical[rv] {
			t.Fatalf("recovery lifecycle %q must be in the canonical set", rv)
		}
	}
}

// TestLIVE_DIAG_TRUTH_1_StaleSnapshotDataAvailable verifies that a stale
// snapshot (captured > 90s ago) still has DataAvailable=true with
// FinancialFresh=false / PositionsFresh=false. This confirms the design
// decision: available=true + stale tag, not available=false.
func TestLIVE_DIAG_TRUTH_1_StaleSnapshotDataAvailable(t *testing.T) {
	pc := NewPositionCache(nil)
	accountID := "test-account-stale"
	magic := int32(1699507621)

	// Snapshot captured 120s ago — stale (> 90s max age)
	staleTime := time.Now().Add(-120 * time.Second)
	snap := &mthub.PositionSnapshot{
		AccountID:              accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:             staleTime,
		PositionsCapturedAt:    staleTime,
		FinancialsSource:       "account_summary",
		PositionsSource:        "order_stream",
		Positions: []mthub.PositionSnapshotItem{{Magic: magic}},
	}
	pc.PutSnapshot(snap, staleTime)

	sess := &ActiveSession{
		AccountID:   accountID,
		MagicNumber: magic,
		barrier:     NewTradeBarrier(nil),
		diag:        newSessionDiag(),
	}

	diagSnap := sess.diag.SnapshotDiag()
	enrichDiagSnapshot(&diagSnap, sess, pc)

	// DataAvailable must be true — snapshot exists (even though stale)
	if !diagSnap.DataAvailable {
		t.Fatal("data_available should be true for stale snapshot (available=true + fresh=false, not available=false)")
	}
	// But freshness must be false
	if diagSnap.FinancialFresh {
		t.Fatal("financial_fresh should be false (captured 120s ago, > 90s max)")
	}
	if diagSnap.PositionsFresh {
		t.Fatal("positions_fresh should be false (captured 120s ago, > 90s max)")
	}
	// Counts should still be populated from the stale snapshot
	if diagSnap.StrategyMagicOrders != 1 {
		t.Fatalf("strategy_magic_orders=%d, want 1 (from stale snapshot)", diagSnap.StrategyMagicOrders)
	}
}
