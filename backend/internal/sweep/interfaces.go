package sweep

import (
	"context"
	"time"

	"github.com/google/uuid"

	"alphaforge/internal/model"
)

// TronClientIface is the subset of TronClient methods used by the sweep pipeline.
// Extracted for testability — production code uses *TronClient which satisfies this.
type TronClientIface interface {
	GetTransactionInfo(ctx context.Context, txid string) (confirmed, success bool, energyUsed int64, err error)
	BroadcastSignedTx(ctx context.Context, signedTxData []byte) (string, error)
	WaitForConfirmation(ctx context.Context, txid string, pollInterval time.Duration) (bool, int64, error)
}

// SweepLogRepoIface is the subset of SweepLogRepository methods used by broadcaster and state machine.
type SweepLogRepoIface interface {
	ListBatchLegs(ctx context.Context, batchID uuid.UUID) ([]model.SweepLog, error)
	ListSweepingWithTxHash(ctx context.Context) ([]model.SweepLog, error)
	UpdateToSweeping(ctx context.Context, id uuid.UUID, txHash string, energyUsed int64) error
	UpdateToDone(ctx context.Context, id uuid.UUID) error
	UpdateTxHash(ctx context.Context, id uuid.UUID, txHash string) error
	UpdateToFailed(ctx context.Context, id uuid.UUID, errMsg string) error
	UpdateToManualReview(ctx context.Context, id uuid.UUID, reason string) error
	GetLatestDoneTransferLeg(ctx context.Context, addrID uuid.UUID) (*model.SweepLog, error)
	MarkStuckSweepingAsFailed(ctx context.Context, maxAge time.Duration) (int64, error)
}

// AddrRepoIface is the subset of DepositAddressRepository methods used by sweep.
type AddrRepoIface interface {
	MarkReceivedUSDT(ctx context.Context, id uuid.UUID) error
}

// AdminRepoIface is the subset of AdminRepository methods used by sweep.
type AdminRepoIface interface {
	GetConfig(ctx context.Context, key string) (*model.SystemConfig, error)
}

// TronGridIface is the subset of TronGridClient methods used by sweep.
type TronGridIface interface {
	HasOutgoingTRC20Transfer(ctx context.Context, from, to, contract string) (bool, error)
}
