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
			Kind:         antv1.TxKind_TX_KIND_DELEGATE,
			SignedTxData: []byte(fmt.Sprintf("signed-tx-%d", i)),
			TxHash:       h,
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
