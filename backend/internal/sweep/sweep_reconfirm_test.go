package sweep

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"alphaforge/internal/model"
)

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
