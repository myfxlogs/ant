//go:build integration

package mthub

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TestOmsWriterInsertAndTransition tests the full OMS state machine with real PG.
func TestOmsWriterInsertAndTransition(t *testing.T) {
	t.Parallel()
	pg := getTestPG(t)

	w := NewOmsWriter(pg, nil)
	ctx := context.Background()
	orderID := uuid.New().String()
	accountID := "06daab3c-5d87-41fd-bd8b-f31ba73c16c1" // real MT account in dev DB

	// Insert order.
	err := w.InsertOrder(ctx, orderID, accountID, "MT5", "EURUSD",
		0, // OrderMarket
		decimal.NewFromInt(1),
		decimal.NewFromFloat(1.085),
		decimal.Zero,
		decimal.Zero,
	)
	if err != nil {
		t.Fatalf("InsertOrder failed: %v", err)
	}

	// Idempotent re-insert (ON CONFLICT DO NOTHING).
	err = w.InsertOrder(ctx, orderID, accountID, "MT5", "EURUSD", 0,
		decimal.NewFromInt(1), decimal.NewFromFloat(1.085),
		decimal.Zero, decimal.Zero)
	if err != nil {
		t.Fatalf("InsertOrder (idempotent) failed: %v", err)
	}

	// Transition: NEW → VALIDATED.
	err = w.Transition(ctx, orderID, accountID, OMSStateNew, OMSStateValidated)
	if err != nil {
		t.Fatalf("Transition NEW→VALIDATED failed: %v", err)
	}

	// Transition: VALIDATED → RISK_APPROVED.
	err = w.Transition(ctx, orderID, accountID, OMSStateValidated, OMSStateRiskApproved)
	if err != nil {
		t.Fatalf("Transition VALIDATED→RISK_APPROVED failed: %v", err)
	}

	// Transition: RISK_APPROVED → SUBMITTED.
	err = w.Transition(ctx, orderID, accountID, OMSStateRiskApproved, OMSStateSubmitted)
	if err != nil {
		t.Fatalf("Transition RISK_APPROVED→SUBMITTED failed: %v", err)
	}

	// Transition: SUBMITTED → WORKING.
	err = w.Transition(ctx, orderID, accountID, OMSStateSubmitted, OMSStateWorking)
	if err != nil {
		t.Fatalf("Transition SUBMITTED→WORKING failed: %v", err)
	}

	// Transition: WORKING → FILLED.
	err = w.Transition(ctx, orderID, accountID, OMSStateWorking, OMSStateFilled)
	if err != nil {
		t.Fatalf("Transition WORKING→FILLED failed: %v", err)
	}

	// Cleanup: delete test order.
	_, _ = pg.Exec(ctx, "DELETE FROM orders WHERE id = $1", orderID)
}

func TestOmsWriterInsertOrder_EmptyPlatform(t *testing.T) {
	t.Parallel()
	pg := getTestPG(t)
	w := NewOmsWriter(pg, nil)
	ctx := context.Background()
	orderID := uuid.New().String()
	accountID := "06daab3c-5d87-41fd-bd8b-f31ba73c16c1" // real MT account in dev DB

	// Platform empty → defaults to "MT5".
	err := w.InsertOrder(ctx, orderID, accountID, "", "EURUSD", 0,
		decimal.NewFromInt(1), decimal.Zero, decimal.Zero, decimal.Zero)
	if err != nil {
		t.Fatalf("InsertOrder with empty platform failed: %v", err)
	}
	_, _ = pg.Exec(ctx, "DELETE FROM orders WHERE id = $1", orderID)
}

func TestOmsWriterTransition_InvalidState(t *testing.T) {
	t.Parallel()
	pg := getTestPG(t)
	w := NewOmsWriter(pg, nil)
	ctx := context.Background()
	orderID := uuid.New().String()
	accountID := "06daab3c-5d87-41fd-bd8b-f31ba73c16c1" // real MT account in dev DB

	// Insert first.
	_ = w.InsertOrder(ctx, orderID, accountID, "MT5", "EURUSD", 0,
		decimal.NewFromInt(1), decimal.NewFromFloat(1.085),
		decimal.Zero, decimal.Zero)

	// Invalid transition: NEW → FILLED (skipping intermediate states).
	err := w.Transition(ctx, orderID, accountID, OMSStateNew, OMSStateFilled)
	if err == nil {
		t.Fatal("expected error for invalid transition NEW→FILLED")
	}
	_, _ = pg.Exec(ctx, "DELETE FROM orders WHERE id = $1", orderID)
}

func TestOmsWriterTransition_WithTradeEventStore(t *testing.T) {
	t.Parallel()
	pg := getTestPG(t)
	store := NewTradeEventStore(nil) // nil NATS — Publish will be no-op but path is covered
	w := NewOmsWriter(pg, store)
	ctx := context.Background()
	orderID := uuid.New().String()
	accountID := "06daab3c-5d87-41fd-bd8b-f31ba73c16c1"

	_ = w.InsertOrder(ctx, orderID, accountID, "MT5", "EURUSD", 0,
		decimal.NewFromInt(1), decimal.NewFromFloat(1.085),
		decimal.Zero, decimal.Zero)

	err := w.Transition(ctx, orderID, accountID, OMSStateNew, OMSStateValidated)
	if err != nil {
		t.Fatalf("Transition with event store failed: %v", err)
	}

	// Also test with OrderEventBroker wired.
	w.SetOrderEventBroker(NewOrderEventBroker())
	err = w.Transition(ctx, orderID, accountID, OMSStateValidated, OMSStateRiskApproved)
	if err != nil {
		t.Fatalf("Transition with order event broker failed: %v", err)
	}

	_, _ = pg.Exec(ctx, "DELETE FROM orders WHERE id = $1", orderID)
}

func TestOmsWriterTransition_ConcurrentConflict(t *testing.T) {
	t.Parallel()
	pg := getTestPG(t)
	w := NewOmsWriter(pg, nil)
	ctx := context.Background()
	orderID := uuid.New().String()
	accountID := "06daab3c-5d87-41fd-bd8b-f31ba73c16c1" // real MT account in dev DB

	_ = w.InsertOrder(ctx, orderID, accountID, "MT5", "EURUSD", 0,
		decimal.NewFromInt(1), decimal.NewFromFloat(1.085),
		decimal.Zero, decimal.Zero)

	// First transition succeeds.
	_ = w.Transition(ctx, orderID, accountID, OMSStateNew, OMSStateValidated)

	// Second transition with WRONG current state (already VALIDATED, not NEW).
	err := w.Transition(ctx, orderID, accountID, OMSStateNew, OMSStateRiskApproved)
	if err == nil {
		t.Fatal("expected concurrent conflict error")
	}
	_, _ = pg.Exec(ctx, "DELETE FROM orders WHERE id = $1", orderID)
}
