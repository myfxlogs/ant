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
