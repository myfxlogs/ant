package sweep

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"alphaforge/internal/model"
)

// ─── Tests: ReconfirmSweeping ───────────────────────────────────────────────

// Test: SWEEPING leg confirmed on chain → DONE.
func TestReconfirmSweeping_ConfirmedSuccess_ToDone(t *testing.T) {
	addrID := uuid.New()
	leg := model.SweepLog{
		ID:               uuid.New(),
		BatchID:          uuid.New(),
		DepositAddressID: addrID,
		LegType:          "delegate",
		LegSeq:           0,
		TxHash:           "tx-hash-1",
		Status:           "SWEEPING",
	}

	repo := newFakeSweepLogRepo()
	repo.listSweepingResult = []model.SweepLog{leg}

	tron := &fakeTronClient{
		getTxInfoResult: map[string]*txInfoResult{
			"tx-hash-1": {confirmed: true, success: true, energyUsed: 50000},
		},
	}

	s := NewStateMachine(tron, repo, &fakeTronGrid{}, nil, &fakeAddrRepo{}, testLogger())
	err := s.ReconfirmSweeping(context.Background())

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(repo.doneCalls) != 1 {
		t.Errorf("should mark DONE, got %d calls", len(repo.doneCalls))
	}
}

// Test: SWEEPING leg confirmed FAILED on chain → MANUAL_REVIEW.
func TestReconfirmSweeping_ConfirmedFailed_ToManualReview(t *testing.T) {
	leg := model.SweepLog{
		ID:               uuid.New(),
		BatchID:          uuid.New(),
		DepositAddressID: uuid.New(),
		LegType:          "transfer",
		LegSeq:           1,
		TxHash:           "tx-failed",
		Status:           "SWEEPING",
	}

	repo := newFakeSweepLogRepo()
	repo.listSweepingResult = []model.SweepLog{leg}

	tron := &fakeTronClient{
		getTxInfoResult: map[string]*txInfoResult{
			"tx-failed": {confirmed: true, success: false, energyUsed: 90000},
		},
	}

	s := NewStateMachine(tron, repo, &fakeTronGrid{}, nil, &fakeAddrRepo{}, testLogger())
	err := s.ReconfirmSweeping(context.Background())

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(repo.manualReviewCalls) != 1 {
		t.Errorf("should mark MANUAL_REVIEW, got %d calls", len(repo.manualReviewCalls))
	}
}

// Test: SWEEPING leg not yet confirmed → stays SWEEPING (no state change).
func TestReconfirmSweeping_NotConfirmed_NoChange(t *testing.T) {
	leg := model.SweepLog{
		ID:               uuid.New(),
		BatchID:          uuid.New(),
		DepositAddressID: uuid.New(),
		LegType:          "delegate",
		LegSeq:           0,
		TxHash:           "tx-pending",
		Status:           "SWEEPING",
	}

	repo := newFakeSweepLogRepo()
	repo.listSweepingResult = []model.SweepLog{leg}

	tron := &fakeTronClient{
		getTxInfoResult: map[string]*txInfoResult{
			"tx-pending": {confirmed: false},
		},
	}

	s := NewStateMachine(tron, repo, &fakeTronGrid{}, nil, &fakeAddrRepo{}, testLogger())
	err := s.ReconfirmSweeping(context.Background())

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(repo.doneCalls) != 0 || len(repo.manualReviewCalls) != 0 {
		t.Errorf("should not change state for unconfirmed tx, done=%d manualReview=%d", len(repo.doneCalls), len(repo.manualReviewCalls))
	}
}

// Test D16: FAILED+tx_hash leg confirmed on chain → DONE (safety net for expired bundles).
func TestReconfirmSweeping_FailedLegConfirmed_ToDone(t *testing.T) {
	leg := model.SweepLog{
		ID:               uuid.New(),
		BatchID:          uuid.New(),
		DepositAddressID: uuid.New(),
		LegType:          "transfer",
		LegSeq:           1,
		TxHash:           "tx-failed-but-confirmed",
		Status:           "FAILED",
	}

	repo := newFakeSweepLogRepo()
	repo.listSweepingResult = []model.SweepLog{leg}

	tron := &fakeTronClient{
		getTxInfoResult: map[string]*txInfoResult{
			"tx-failed-but-confirmed": {confirmed: true, success: true, energyUsed: 30000},
		},
	}

	s := NewStateMachine(tron, repo, &fakeTronGrid{}, nil, &fakeAddrRepo{}, testLogger())
	err := s.ReconfirmSweeping(context.Background())

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(repo.doneCalls) != 1 {
		t.Errorf("D16: FAILED leg confirmed on chain should transition to DONE, got %d done calls", len(repo.doneCalls))
	}
}

// Test: successful transfer leg in ReconfirmSweeping → MarkReceivedUSDT called.
func TestReconfirmSweeping_TransferSuccess_MarkReceivedUSDT(t *testing.T) {
	addrID := uuid.New()
	leg := model.SweepLog{
		ID:               uuid.New(),
		BatchID:          uuid.New(),
		DepositAddressID: addrID,
		LegType:          "transfer",
		LegSeq:           1,
		TxHash:           "transfer-tx",
		Status:           "SWEEPING",
	}

	repo := newFakeSweepLogRepo()
	repo.listSweepingResult = []model.SweepLog{leg}

	tron := &fakeTronClient{
		getTxInfoResult: map[string]*txInfoResult{
			"transfer-tx": {confirmed: true, success: true, energyUsed: 30000},
		},
	}

	addrRepo := &fakeAddrRepo{}
	s := NewStateMachine(tron, repo, &fakeTronGrid{}, nil, addrRepo, testLogger())
	_ = s.ReconfirmSweeping(context.Background())

	if len(addrRepo.markReceivedCalls) != 1 {
		t.Errorf("should call MarkReceivedUSDT for confirmed transfer, got %d calls", len(addrRepo.markReceivedCalls))
	}
}

// ─── Tests: CheckDoubleSpend ────────────────────────────────────────────────

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
func TestThreeLegStateTracking_SequentialTransition(t *testing.T) {
	batchID := uuid.New()
	addrID := uuid.New()

	legs := makeLegs(batchID, addrID, "PENDING", "PENDING", "PENDING")
	repo := newFakeSweepLogRepo()
	repo.legs[batchID] = legs

	tron := &fakeTronClient{
		broadcastResult: map[string]*broadcastResult{
			"signed-tx-0": {txid: "delegate-tx"},
			"signed-tx-1": {txid: "transfer-tx"},
			"signed-tx-2": {txid: "undelegate-tx"},
		},
		waitConfirmResult: map[string]*confirmResult{
			"delegate-tx":   {success: true, energyUsed: 50000},
			"transfer-tx":   {success: true, energyUsed: 30000},
			"undelegate-tx": {success: true, energyUsed: 10000},
		},
	}

	addrRepo := &fakeAddrRepo{}
	b := NewBroadcaster(tron, repo, addrRepo, nil, testLogger())
	err := b.BroadcastBundle(context.Background(), makeSignedBundle(batchID.String(), "tx0", "tx1", "tx2"))

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// All 3 legs should be broadcast.
	if tron.broadcastCalls != 3 {
		t.Errorf("should broadcast all 3 legs, got %d calls", tron.broadcastCalls)
	}

	// All 3 legs should be marked DONE.
	if len(repo.doneCalls) != 3 {
		t.Errorf("should mark all 3 legs DONE, got %d calls", len(repo.doneCalls))
	}

	// 3 sweeping transitions (PENDING → SWEEPING).
	if len(repo.sweepingCalls) != 3 {
		t.Errorf("should have 3 SWEEPING transitions, got %d calls", len(repo.sweepingCalls))
	}

	// Only transfer leg (leg 1) should trigger MarkReceivedUSDT.
	if len(addrRepo.markReceivedCalls) != 1 {
		t.Errorf("should call MarkReceivedUSDT once (transfer leg only), got %d calls", len(addrRepo.markReceivedCalls))
	}
}

// Test: partial batch failure — leg 1 (transfer) fails, leg 2 (undelegate) stays PENDING.
func TestThreeLegStateTracking_PartialFailure_StopsAtFailedLeg(t *testing.T) {
	batchID := uuid.New()
	addrID := uuid.New()

	legs := makeLegs(batchID, addrID, "PENDING", "PENDING", "PENDING")
	repo := newFakeSweepLogRepo()
	repo.legs[batchID] = legs

	tron := &fakeTronClient{
		broadcastResult: map[string]*broadcastResult{
			"signed-tx-0": {txid: "delegate-tx"},
			"signed-tx-1": {txid: "", err: fmt.Errorf("broadcast rejected")},
		},
		waitConfirmResult: map[string]*confirmResult{
			"delegate-tx": {success: true, energyUsed: 50000},
		},
	}

	b := NewBroadcaster(tron, repo, &fakeAddrRepo{}, nil, testLogger())
	err := b.BroadcastBundle(context.Background(), makeSignedBundle(batchID.String(), "tx0", "tx1", "tx2"))

	if err == nil {
		t.Fatal("expected error for failed transfer leg")
	}

	// Only leg 0 should be DONE.
	if len(repo.doneCalls) != 1 {
		t.Errorf("should mark only leg 0 DONE, got %d calls", len(repo.doneCalls))
	}

	// Leg 1 should be FAILED.
	if len(repo.failedCalls) != 1 {
		t.Errorf("should mark leg 1 FAILED, got %d calls", len(repo.failedCalls))
	}

	// Leg 2 should NOT be broadcast (bundle stops at failed leg).
	if tron.broadcastCalls != 2 {
		t.Errorf("should broadcast only legs 0 and 1, got %d calls", tron.broadcastCalls)
	}
}
