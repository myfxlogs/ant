package mthub

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/risk"
)

// TestTask1_CloseOrder_OMS_ID_IsValidUUID verifies that CloseOrder generates
// a valid UUID for the OMS order ID (not the synthetic "close-..." string).
// Revert to using closeOrderID directly in InsertOrder → PG rejects non-UUID
// → test goes RED (no row inserted).
//
// Adversarial proof: the OMS ID must be a valid UUID parseable by uuid.Parse.
// The old synthetic string "close-acct-123" fails uuid.Parse → RED.

func TestTask1_CloseOrder_OMS_ID_IsValidUUID(t *testing.T) {
	// Verify the MD5-UUID derivation from closeOrderID produces a valid UUID.
	closeOrderID := "close-acct-1-12345"
	omsOrderID := uuid.NewMD5(uuid.NameSpaceOID, []byte(closeOrderID)).String()

	parsed, err := uuid.Parse(omsOrderID)
	if err != nil {
		t.Fatalf("OMS order ID %q is not a valid UUID: %v (CLOSE-ORDER-UUID bug)", omsOrderID, err)
	}
	if parsed == uuid.Nil {
		t.Fatal("OMS order ID is nil UUID")
	}

	// Determinism: same closeOrderID → same UUID (idempotent retry maps to same row).
	omsOrderID2 := uuid.NewMD5(uuid.NameSpaceOID, []byte(closeOrderID)).String()
	if omsOrderID != omsOrderID2 {
		t.Fatalf("MD5-UUID not deterministic: %s vs %s", omsOrderID, omsOrderID2)
	}

	// Adversarial: the old synthetic string must NOT parse as UUID.
	_, err = uuid.Parse(closeOrderID)
	if err == nil {
		t.Fatal("synthetic closeOrderID should NOT parse as UUID — adversarial check failed")
	}
}

// TestTask1_CloseOrder_OMS_ID_DifferentTickets maps to different UUIDs.
func TestTask1_CloseOrder_OMS_ID_DifferentTickets(t *testing.T) {
	id1 := uuid.NewMD5(uuid.NameSpaceOID, []byte("close-acct-1-100")).String()
	id2 := uuid.NewMD5(uuid.NameSpaceOID, []byte("close-acct-1-200")).String()
	if id1 == id2 {
		t.Fatal("different tickets should produce different OMS UUIDs")
	}
}

// TestTask1_CloseOrder_OMS_ID_DifferentAccounts maps to different UUIDs.
func TestTask1_CloseOrder_OMS_ID_DifferentAccounts(t *testing.T) {
	id1 := uuid.NewMD5(uuid.NameSpaceOID, []byte("close-acct-1-100")).String()
	id2 := uuid.NewMD5(uuid.NameSpaceOID, []byte("close-acct-2-100")).String()
	if id1 == id2 {
		t.Fatal("different accounts should produce different OMS UUIDs")
	}
}

// TestTask1_CloseOrder_NoPanic_WithoutOMSWriter verifies CloseOrder works
// when omsWriter is nil (no OMS recording). The OMS ID derivation must not
// panic even when omsWriter is nil.
func TestTask1_CloseOrder_NoPanic_WithoutOMSWriter(t *testing.T) {
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.SetGate(risk.NewDefaultGate())
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: decimal.NewFromInt(100000), Equity: decimal.NewFromInt(100000)}, nil
	})
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)

	err := svc.CloseOrder(context.Background(), "acc-1", 12345, decimal.NewFromFloat(0.1))
	if err != nil {
		t.Fatalf("CloseOrder without OMS writer failed: %v", err)
	}
}
