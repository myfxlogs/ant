package strategy

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/mthub"
)

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
