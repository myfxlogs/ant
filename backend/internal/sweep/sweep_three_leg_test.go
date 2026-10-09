package sweep

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

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
