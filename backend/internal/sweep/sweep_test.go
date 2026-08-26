package sweep

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
)

// ─── Fake TronClient ────────────────────────────────────────────────────────

type fakeTronClient struct {
	// GetTransactionInfo
	getTxInfoResult map[string]*txInfoResult
	getTxInfoCalls  int
	// BroadcastSignedTx
	broadcastResult map[string]*broadcastResult
	broadcastCalls  int
	// WaitForConfirmation
	waitConfirmResult map[string]*confirmResult
}

type txInfoResult struct {
	confirmed  bool
	success    bool
	energyUsed int64
	err        error
}

type broadcastResult struct {
	txid string
	err  error
}

type confirmResult struct {
	success    bool
	energyUsed int64
	err        error
}

func (f *fakeTronClient) GetTransactionInfo(ctx context.Context, txid string) (bool, bool, int64, error) {
	f.getTxInfoCalls++
	r, ok := f.getTxInfoResult[txid]
	if !ok {
		return false, false, 0, nil
	}
	return r.confirmed, r.success, r.energyUsed, r.err
}

func (f *fakeTronClient) BroadcastSignedTx(ctx context.Context, signedTxData []byte) (string, error) {
	f.broadcastCalls++
	key := string(signedTxData)
	r, ok := f.broadcastResult[key]
	if !ok {
		return "", fmt.Errorf("unexpected broadcast")
	}
	return r.txid, r.err
}

func (f *fakeTronClient) WaitForConfirmation(ctx context.Context, txid string, pollInterval time.Duration) (bool, int64, error) {
	r, ok := f.waitConfirmResult[txid]
	if !ok {
		return false, 0, fmt.Errorf("unexpected WaitForConfirmation for txid %s", txid)
	}
	return r.success, r.energyUsed, r.err
}

// ─── Fake SweepLogRepo ──────────────────────────────────────────────────────

type fakeSweepLogRepo struct {
	legs map[uuid.UUID][]model.SweepLog // batchID → legs

	// Recorded state transitions
	doneCalls          []uuid.UUID
	sweepingCalls      []sweepingCall
	txHashCalls        []txHashCall
	failedCalls        []failedCall
	manualReviewCalls  []manualReviewCall
	stuckCalls         int
	listSweepingResult []model.SweepLog
	doneTransferLeg    *model.SweepLog
}

type sweepingCall struct {
	id         uuid.UUID
	txHash     string
	energyUsed int64
}

type txHashCall struct {
	id     uuid.UUID
	txHash string
}

type failedCall struct {
	id     uuid.UUID
	errMsg string
}

type manualReviewCall struct {
	id     uuid.UUID
	reason string
}

func newFakeSweepLogRepo() *fakeSweepLogRepo {
	return &fakeSweepLogRepo{legs: make(map[uuid.UUID][]model.SweepLog)}
}

func (r *fakeSweepLogRepo) ListBatchLegs(ctx context.Context, batchID uuid.UUID) ([]model.SweepLog, error) {
	return r.legs[batchID], nil
}

func (r *fakeSweepLogRepo) ListSweepingWithTxHash(ctx context.Context) ([]model.SweepLog, error) {
	return r.listSweepingResult, nil
}

func (r *fakeSweepLogRepo) UpdateToSweeping(ctx context.Context, id uuid.UUID, txHash string, energyUsed int64) error {
	r.sweepingCalls = append(r.sweepingCalls, sweepingCall{id, txHash, energyUsed})
	return nil
}

func (r *fakeSweepLogRepo) UpdateToDone(ctx context.Context, id uuid.UUID) error {
	r.doneCalls = append(r.doneCalls, id)
	return nil
}

func (r *fakeSweepLogRepo) UpdateTxHash(ctx context.Context, id uuid.UUID, txHash string) error {
	r.txHashCalls = append(r.txHashCalls, txHashCall{id, txHash})
	return nil
}

func (r *fakeSweepLogRepo) UpdateToFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	r.failedCalls = append(r.failedCalls, failedCall{id, errMsg})
	return nil
}

func (r *fakeSweepLogRepo) UpdateToManualReview(ctx context.Context, id uuid.UUID, reason string) error {
	r.manualReviewCalls = append(r.manualReviewCalls, manualReviewCall{id, reason})
	return nil
}

func (r *fakeSweepLogRepo) GetLatestDoneTransferLeg(ctx context.Context, addrID uuid.UUID) (*model.SweepLog, error) {
	return r.doneTransferLeg, nil
}

func (r *fakeSweepLogRepo) MarkStuckSweepingAsFailed(ctx context.Context, maxAge time.Duration) (int64, error) {
	r.stuckCalls++
	return 0, nil
}

// ─── Fake AddrRepo ──────────────────────────────────────────────────────────

type fakeAddrRepo struct {
	markReceivedCalls []uuid.UUID
}

func (r *fakeAddrRepo) MarkReceivedUSDT(ctx context.Context, id uuid.UUID) error {
	r.markReceivedCalls = append(r.markReceivedCalls, id)
	return nil
}

// ─── Fake AdminRepo ─────────────────────────────────────────────────────────

type fakeAdminRepo struct {
	configs map[string]*model.SystemConfig
}

func newFakeAdminRepo() *fakeAdminRepo {
	return &fakeAdminRepo{configs: make(map[string]*model.SystemConfig)}
}

func (r *fakeAdminRepo) GetConfig(ctx context.Context, key string) (*model.SystemConfig, error) {
	if v, ok := r.configs[key]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("config not found: %s", key)
}

// ─── Fake TronGrid ──────────────────────────────────────────────────────────

type fakeTronGrid struct {
	hasOutgoing bool
	hasOutErr   error
}

func (f *fakeTronGrid) HasOutgoingTRC20Transfer(ctx context.Context, from, to, contract string) (bool, error) {
	return f.hasOutgoing, f.hasOutErr
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func testLogger() *zap.Logger {
	return zap.NewNop()
}

func makeLegs(batchID, addrID uuid.UUID, statuses ...string) []model.SweepLog {
	legTypes := []string{"delegate", "transfer", "undelegate"}
	legs := make([]model.SweepLog, len(statuses))
	for i, st := range statuses {
		legs[i] = model.SweepLog{
			ID:               uuid.New(),
			BatchID:          batchID,
			DepositAddressID: addrID,
			LegType:          legTypes[i%3],
			LegSeq:           i,
			Status:           st,
		}
	}
	return legs
}

func makeSignedBundle(batchID string, txHashes ...string) *antv1.SignedSweepBundle {
	txs := make([]*antv1.SignedTx, len(txHashes))
	for i, h := range txHashes {
		txs[i] = &antv1.SignedTx{
			Kind:        antv1.TxKind_TX_KIND_DELEGATE,
			SignedTxData: []byte(fmt.Sprintf("signed-tx-%d", i)),
			TxHash:      h,
		}
	}
	return &antv1.SignedSweepBundle{BundleId: batchID, Txs: txs}
}

// ─── Tests: BroadcastBundle ─────────────────────────────────────────────────

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
