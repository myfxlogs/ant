package sweep

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

// Test C1: Simulate DB crash after broadcast → on recovery, chain check finds
// tx already confirmed → marks DONE, does NOT re-broadcast.
func TestBroadcastBundle_DBCrashAfterBroadcast_ChainCheckNoReBroadcast(t *testing.T) {
	batchID := uuid.New()
	addrID := uuid.New()
	txHash := "abc123"

	// Leg was SWEEPING (broadcast succeeded, DB crash before confirmation).
	legs := makeLegs(batchID, addrID, "SWEEPING")
	legs[0].TxHash = txHash

	repo := newFakeSweepLogRepo()
	repo.legs[batchID] = legs

	tron := &fakeTronClient{
		getTxInfoResult: map[string]*txInfoResult{
			txHash: {confirmed: true, success: true, energyUsed: 50000},
		},
	}

	b := NewBroadcaster(tron, repo, &fakeAddrRepo{}, nil, testLogger())
	err := b.BroadcastBundle(context.Background(), makeSignedBundle(batchID.String(), txHash))

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if tron.broadcastCalls != 0 {
		t.Errorf("should NOT re-broadcast when chain confirms DONE, but broadcast was called %d times", tron.broadcastCalls)
	}
	if len(repo.doneCalls) != 1 {
		t.Errorf("should mark leg DONE, got %d done calls", len(repo.doneCalls))
	}
}

// Test C2: Crash recovery — broadcast interrupted mid-bundle → resume reads
// back legs, skips DONE, continues from next unconfirmed leg.

// Test C2: Crash recovery — broadcast interrupted mid-bundle → resume reads
// back legs, skips DONE, continues from next unconfirmed leg.
func TestBroadcastBundle_CrashRecovery_SkipDoneContinueNext(t *testing.T) {
	batchID := uuid.New()
	addrID := uuid.New()

	// Leg 0 (delegate) already DONE, leg 1 (transfer) PENDING, leg 2 (undelegate) PENDING.
	legs := makeLegs(batchID, addrID, "DONE", "PENDING", "PENDING")

	repo := newFakeSweepLogRepo()
	repo.legs[batchID] = legs

	tron := &fakeTronClient{
		broadcastResult: map[string]*broadcastResult{
			"signed-tx-1": {txid: "tx1"},
			"signed-tx-2": {txid: "tx2"},
		},
		waitConfirmResult: map[string]*confirmResult{
			"tx1": {success: true, energyUsed: 30000},
			"tx2": {success: true, energyUsed: 10000},
		},
	}

	b := NewBroadcaster(tron, repo, &fakeAddrRepo{}, nil, testLogger())
	err := b.BroadcastBundle(context.Background(), makeSignedBundle(batchID.String(), "tx0", "tx1", "tx2"))

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	// Should only broadcast legs 1 and 2 (leg 0 is DONE, skipped).
	if tron.broadcastCalls != 2 {
		t.Errorf("should broadcast 2 legs (skip DONE), got %d broadcast calls", tron.broadcastCalls)
	}
	// Leg 1 is transfer → should mark received USDT.
	addrRepo := &fakeAddrRepo{}
	b.addrRepo = addrRepo
	_ = b.BroadcastBundle(context.Background(), makeSignedBundle(batchID.String(), "tx0", "tx1", "tx2"))
	if len(addrRepo.markReceivedCalls) != 1 {
		t.Errorf("should call MarkReceivedUSDT once for transfer leg, got %d calls", len(addrRepo.markReceivedCalls))
	}
}

// Test C3: MANUAL_REVIEW leg → bundle halts, returns ErrManualReview, does NOT re-broadcast.

// Test C3: MANUAL_REVIEW leg → bundle halts, returns ErrManualReview, does NOT re-broadcast.
func TestBroadcastBundle_ManualReviewLeg_HaltsNoReBroadcast(t *testing.T) {
	batchID := uuid.New()
	addrID := uuid.New()

	legs := makeLegs(batchID, addrID, "DONE", "MANUAL_REVIEW", "PENDING")
	legs[1].TxHash = "old-tx"

	repo := newFakeSweepLogRepo()
	repo.legs[batchID] = legs

	tron := &fakeTronClient{}

	b := NewBroadcaster(tron, repo, &fakeAddrRepo{}, nil, testLogger())
	err := b.BroadcastBundle(context.Background(), makeSignedBundle(batchID.String(), "tx0", "tx1", "tx2"))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrManualReview) {
		t.Errorf("expected ErrManualReview, got: %v", err)
	}
	if tron.broadcastCalls != 0 {
		t.Errorf("should NOT broadcast any tx when MANUAL_REVIEW leg found, got %d calls", tron.broadcastCalls)
	}
}

// Test D10/D11: FAILED+tx_hash leg → chain check finds it confirmed FAILED →
// transitions to MANUAL_REVIEW, returns ErrManualReview.

// Test D10/D11: FAILED+tx_hash leg → chain check finds it confirmed FAILED →
// transitions to MANUAL_REVIEW, returns ErrManualReview.
func TestBroadcastBundle_FailedLegChainConfirmedFailed_ToManualReview(t *testing.T) {
	batchID := uuid.New()
	addrID := uuid.New()
	txHash := "failed-tx"

	legs := makeLegs(batchID, addrID, "DONE", "FAILED", "PENDING")
	legs[1].TxHash = txHash

	repo := newFakeSweepLogRepo()
	repo.legs[batchID] = legs

	tron := &fakeTronClient{
		getTxInfoResult: map[string]*txInfoResult{
			txHash: {confirmed: true, success: false, energyUsed: 80000},
		},
	}

	b := NewBroadcaster(tron, repo, &fakeAddrRepo{}, nil, testLogger())
	err := b.BroadcastBundle(context.Background(), makeSignedBundle(batchID.String(), "tx0", "tx1", "tx2"))

	if !errors.Is(err, ErrManualReview) {
		t.Errorf("expected ErrManualReview, got: %v", err)
	}
	if len(repo.manualReviewCalls) != 1 {
		t.Errorf("should transition FAILED leg to MANUAL_REVIEW, got %d calls", len(repo.manualReviewCalls))
	}
	if tron.broadcastCalls != 0 {
		t.Errorf("should NOT re-broadcast chain-confirmed FAILED tx, got %d calls", tron.broadcastCalls)
	}
}

// Test D17: Chain-confirmed DONE transfer leg → MarkReceivedUSDT called.

// Test D17: Chain-confirmed DONE transfer leg → MarkReceivedUSDT called.
func TestBroadcastBundle_ChainConfirmedTransfer_MarkReceivedUSDT(t *testing.T) {
	batchID := uuid.New()
	addrID := uuid.New()
	txHash := "transfer-tx"

	legs := makeLegs(batchID, addrID, "DONE", "SWEEPING", "PENDING")
	legs[1].TxHash = txHash
	legs[1].LegType = "transfer"

	repo := newFakeSweepLogRepo()
	repo.legs[batchID] = legs

	tron := &fakeTronClient{
		getTxInfoResult: map[string]*txInfoResult{
			txHash: {confirmed: true, success: true, energyUsed: 30000},
		},
		broadcastResult: map[string]*broadcastResult{
			"signed-tx-2": {txid: "tx2"},
		},
		waitConfirmResult: map[string]*confirmResult{
			"tx2": {success: true, energyUsed: 10000},
		},
	}

	addrRepo := &fakeAddrRepo{}
	b := NewBroadcaster(tron, repo, addrRepo, nil, testLogger())
	err := b.BroadcastBundle(context.Background(), makeSignedBundle(batchID.String(), "tx0", "tx1", "tx2"))

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(addrRepo.markReceivedCalls) != 1 {
		t.Errorf("D17: should call MarkReceivedUSDT for chain-confirmed transfer, got %d calls", len(addrRepo.markReceivedCalls))
	}
}

// Test: FAILED+tx_hash leg, not found on chain → safe to re-broadcast.

// Test: FAILED+tx_hash leg, not found on chain → safe to re-broadcast.
func TestBroadcastBundle_FailedLegNotOnChain_ReBroadcast(t *testing.T) {
	batchID := uuid.New()
	addrID := uuid.New()
	oldTxHash := "expired-tx"

	legs := makeLegs(batchID, addrID, "DONE", "FAILED", "PENDING")
	legs[1].TxHash = oldTxHash

	repo := newFakeSweepLogRepo()
	repo.legs[batchID] = legs

	tron := &fakeTronClient{
		getTxInfoResult: map[string]*txInfoResult{
			oldTxHash: {confirmed: false},
		},
		broadcastResult: map[string]*broadcastResult{
			"signed-tx-1": {txid: "new-tx-1"},
			"signed-tx-2": {txid: "new-tx-2"},
		},
		waitConfirmResult: map[string]*confirmResult{
			"new-tx-1": {success: true, energyUsed: 30000},
			"new-tx-2": {success: true, energyUsed: 10000},
		},
	}

	b := NewBroadcaster(tron, repo, &fakeAddrRepo{}, nil, testLogger())
	err := b.BroadcastBundle(context.Background(), makeSignedBundle(batchID.String(), "tx0", "tx1", "tx2"))

	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if tron.broadcastCalls != 2 {
		t.Errorf("should re-broadcast FAILED leg not on chain + PENDING leg 2, got %d calls", tron.broadcastCalls)
	}
}

// Test: broadcast fails → leg marked FAILED, bundle stops.

// Test: broadcast fails → leg marked FAILED, bundle stops.
func TestBroadcastBundle_BroadcastFails_LegFailed(t *testing.T) {
	batchID := uuid.New()
	addrID := uuid.New()

	legs := makeLegs(batchID, addrID, "PENDING", "PENDING", "PENDING")

	repo := newFakeSweepLogRepo()
	repo.legs[batchID] = legs

	tron := &fakeTronClient{
		broadcastResult: map[string]*broadcastResult{
			"signed-tx-0": {txid: "", err: fmt.Errorf("network error")},
		},
	}

	b := NewBroadcaster(tron, repo, &fakeAddrRepo{}, nil, testLogger())
	err := b.BroadcastBundle(context.Background(), makeSignedBundle(batchID.String(), "tx0", "tx1", "tx2"))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(repo.failedCalls) != 1 {
		t.Errorf("should mark failed leg, got %d failed calls", len(repo.failedCalls))
	}
}

// Test: on-chain execution fails (WaitForConfirmation returns success=false) →
// leg transitions to MANUAL_REVIEW, not FAILED.

// Test: on-chain execution fails (WaitForConfirmation returns success=false) →
// leg transitions to MANUAL_REVIEW, not FAILED.
func TestBroadcastBundle_OnChainExecutionFailed_ToManualReview(t *testing.T) {
	batchID := uuid.New()
	addrID := uuid.New()

	legs := makeLegs(batchID, addrID, "PENDING", "PENDING", "PENDING")

	repo := newFakeSweepLogRepo()
	repo.legs[batchID] = legs

	tron := &fakeTronClient{
		broadcastResult: map[string]*broadcastResult{
			"signed-tx-0": {txid: "tx0"},
		},
		waitConfirmResult: map[string]*confirmResult{
			"tx0": {success: false, energyUsed: 100000},
		},
	}

	b := NewBroadcaster(tron, repo, &fakeAddrRepo{}, nil, testLogger())
	err := b.BroadcastBundle(context.Background(), makeSignedBundle(batchID.String(), "tx0", "tx1", "tx2"))

	if !errors.Is(err, ErrManualReview) {
		t.Errorf("expected ErrManualReview, got: %v", err)
	}
	if len(repo.manualReviewCalls) != 1 {
		t.Errorf("should transition to MANUAL_REVIEW, got %d calls", len(repo.manualReviewCalls))
	}
}

// Test: tx count mismatch → error.

// Test: tx count mismatch → error.
func TestBroadcastBundle_TxCountMismatch(t *testing.T) {
	batchID := uuid.New()
	addrID := uuid.New()

	legs := makeLegs(batchID, addrID, "PENDING", "PENDING", "PENDING")
	repo := newFakeSweepLogRepo()
	repo.legs[batchID] = legs

	b := NewBroadcaster(&fakeTronClient{}, repo, &fakeAddrRepo{}, nil, testLogger())
	err := b.BroadcastBundle(context.Background(), makeSignedBundle(batchID.String(), "tx0"))

	if err == nil {
		t.Fatal("expected error for tx count mismatch")
	}
}

// ─── Tests: ReconfirmSweeping ───────────────────────────────────────────────

// Test: SWEEPING leg confirmed on chain → DONE.
