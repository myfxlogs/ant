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

// TestLIVE_DIAG_TRUTH_1_DataAvailableWithCache verifies that DataAvailable
// is true when posCache is provided.
func TestLIVE_DIAG_TRUTH_1_DataAvailableWithCache(t *testing.T) {
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

	if !diagSnap.DataAvailable {
		t.Fatal("data_available should be true when posCache is provided")
	}
}

// TestLIVE_DIAG_TRUTH_1_LogOrderLifecycleWiring verifies that
// logOrderLifecycle actually calls RecordLifecycle on sessionDiag.
// This is the critical wiring test: if the RecordLifecycle call inside
// logOrderLifecycle is removed, this test must RED.

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
		AccountID:               accountID,
		FinancialsAuthoritative: true,
		PositionsAuthoritative:  true,
		CapturedAt:              staleTime,
		PositionsCapturedAt:     staleTime,
		FinancialsSource:        "account_summary",
		PositionsSource:         "order_stream",
		Positions:               []mthub.PositionSnapshotItem{{Magic: magic}},
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
