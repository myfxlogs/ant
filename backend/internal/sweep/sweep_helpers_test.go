package sweep

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
)

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
