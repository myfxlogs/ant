package sweep

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"alphaforge/internal/model"
)

// Test: DONE transfer leg exists → no double-spend (expected outgoing transfer).
func TestCheckDoubleSpend_DoneTransferLegExists_NoDoubleSpend(t *testing.T) {
	addrID := uuid.New()
	now := time.Now()

	repo := newFakeSweepLogRepo()
	repo.doneTransferLeg = &model.SweepLog{
		DepositAddressID: addrID,
		LegType:          "transfer",
		Status:           "DONE",
		CompletedAt:      &now,
	}

	adminRepo := newFakeAdminRepo()
	adminRepo.configs["cold_wallet_address"] = &model.SystemConfig{Value: "cold-addr"}
	adminRepo.configs["usdt_contract_address"] = &model.SystemConfig{Value: "usdt-contract"}

	s := NewStateMachine(&fakeTronClient{}, repo, &fakeTronGrid{hasOutgoing: true}, adminRepo, &fakeAddrRepo{}, testLogger())
	isDouble, err := s.CheckDoubleSpend(context.Background(), addrID, "from-addr")

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if isDouble {
		t.Error("should NOT flag double-spend when DONE transfer leg exists (expected outgoing)")
	}
}

// Test: no DONE transfer leg + outgoing transfer detected → double-spend.

// Test: no DONE transfer leg + outgoing transfer detected → double-spend.
func TestCheckDoubleSpend_NoDoneLeg_OutgoingDetected_DoubleSpend(t *testing.T) {
	addrID := uuid.New()

	repo := newFakeSweepLogRepo()
	repo.doneTransferLeg = nil

	adminRepo := newFakeAdminRepo()
	adminRepo.configs["cold_wallet_address"] = &model.SystemConfig{Value: "cold-addr"}
	adminRepo.configs["usdt_contract_address"] = &model.SystemConfig{Value: "usdt-contract"}

	s := NewStateMachine(&fakeTronClient{}, repo, &fakeTronGrid{hasOutgoing: true}, adminRepo, &fakeAddrRepo{}, testLogger())
	isDouble, err := s.CheckDoubleSpend(context.Background(), addrID, "from-addr")

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !isDouble {
		t.Error("should flag double-spend when outgoing transfer detected and no DONE leg")
	}
}

// Test: no DONE transfer leg + no outgoing transfer → safe to sweep.

// Test: no DONE transfer leg + no outgoing transfer → safe to sweep.
func TestCheckDoubleSpend_NoDoneLeg_NoOutgoing_Safe(t *testing.T) {
	addrID := uuid.New()

	repo := newFakeSweepLogRepo()
	repo.doneTransferLeg = nil

	adminRepo := newFakeAdminRepo()
	adminRepo.configs["cold_wallet_address"] = &model.SystemConfig{Value: "cold-addr"}
	adminRepo.configs["usdt_contract_address"] = &model.SystemConfig{Value: "usdt-contract"}

	s := NewStateMachine(&fakeTronClient{}, repo, &fakeTronGrid{hasOutgoing: false}, adminRepo, &fakeAddrRepo{}, testLogger())
	isDouble, err := s.CheckDoubleSpend(context.Background(), addrID, "from-addr")

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if isDouble {
		t.Error("should NOT flag double-spend when no outgoing transfer")
	}
}

// Test F5: admin override (sweep_skip_doublecheck=true) → skip check.

// Test F5: admin override (sweep_skip_doublecheck=true) → skip check.
func TestCheckDoubleSpend_AdminOverride_SkipsCheck(t *testing.T) {
	addrID := uuid.New()

	repo := newFakeSweepLogRepo()
	repo.doneTransferLeg = nil

	adminRepo := newFakeAdminRepo()
	adminRepo.configs["sweep_skip_doublecheck"] = &model.SystemConfig{Value: "true"}

	s := NewStateMachine(&fakeTronClient{}, repo, &fakeTronGrid{hasOutgoing: true}, adminRepo, &fakeAddrRepo{}, testLogger())
	isDouble, err := s.CheckDoubleSpend(context.Background(), addrID, "from-addr")

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if isDouble {
		t.Error("F5: admin override should skip double-spend check")
	}
}

// Test: cold_wallet_address not configured → error.

// Test: cold_wallet_address not configured → error.
func TestCheckDoubleSpend_ColdWalletNotConfigured_Error(t *testing.T) {
	addrID := uuid.New()

	repo := newFakeSweepLogRepo()
	adminRepo := newFakeAdminRepo() // empty configs

	s := NewStateMachine(&fakeTronClient{}, repo, &fakeTronGrid{}, adminRepo, &fakeAddrRepo{}, testLogger())
	_, err := s.CheckDoubleSpend(context.Background(), addrID, "from-addr")

	if err == nil {
		t.Error("expected error when cold_wallet_address not configured")
	}
}

// ─── Tests: 3-leg state tracking ────────────────────────────────────────────

// Test: each leg in a 3-leg bundle is independently tracked.
// Verifies that legs transition through PENDING → SWEEPING → DONE in order.
