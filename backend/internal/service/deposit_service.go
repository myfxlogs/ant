package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
	"alphaforge/internal/repository"
)

// DepositService manages HD wallet deposits.
// Users claim a personal TRC20 address; deposits are auto-confirmed by chain monitor.
type DepositService struct {
	addrRepo    *repository.DepositAddressRepository
	depositRepo *repository.DepositRepository
	walletSvc   *WalletService
	pg          *pgxpool.Pool
	log         *zap.Logger

	// OnAddressClaimed is called after a successful address claim.
	// Used by chain monitor to register the new address immediately.
	OnAddressClaimed func(addr string, userID uuid.UUID, addrID uuid.UUID)
}

func NewDepositService(
	addrRepo *repository.DepositAddressRepository,
	depositRepo *repository.DepositRepository,
	walletSvc *WalletService,
	pg *pgxpool.Pool,
	log *zap.Logger,
) *DepositService {
	return &DepositService{
		addrRepo:    addrRepo,
		depositRepo: depositRepo,
		walletSvc:   walletSvc,
		pg:          pg,
		log:         log,
	}
}

// GetOrClaimAddress returns the user's assigned deposit address, claiming one from the pool if needed.
func (s *DepositService) GetOrClaimAddress(ctx context.Context, userID uuid.UUID) (addr, network string, err error) {
	a, err := s.addrRepo.ClaimAddress(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrAddressPoolEmpty) {
			return "", "", fmt.Errorf("deposit service: address pool exhausted, please contact support")
		}
		return "", "", fmt.Errorf("deposit service: claim address: %w", err)
	}

	available, err := s.addrRepo.CountAvailable(ctx)
	if err == nil && available < 100 {
		s.log.Warn("address pool running low", zap.Int("available", available))
	}

	if s.OnAddressClaimed != nil {
		s.OnAddressClaimed(a.Address, userID, a.ID)
	}

	return a.Address, a.Network, nil
}

// ListMyDeposits returns paginated deposit history for a user.
func (s *DepositService) ListMyDeposits(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]model.Deposit, int64, error) {
	return s.depositRepo.ListByUser(ctx, userID, page, pageSize)
}

// ListManualReviewDeposits returns deposits requiring manual review (admin).
func (s *DepositService) ListManualReviewDeposits(ctx context.Context, page, pageSize int) ([]model.Deposit, int64, error) {
	return s.depositRepo.ListManualReview(ctx, page, pageSize)
}

// ListDepositAddresses returns paginated deposit addresses with pool stats (admin).
func (s *DepositService) ListDepositAddresses(ctx context.Context, status string, page, pageSize int) ([]model.DepositAddress, int64, int, error) {
	addrs, total, err := s.addrRepo.ListAllAddresses(ctx, status, page, pageSize)
	if err != nil {
		return nil, 0, 0, err
	}
	available, err := s.addrRepo.CountAvailable(ctx)
	if err != nil {
		s.log.Warn("deposit service: count available", zap.Error(err))
		available = 0
	}
	return addrs, total, available, nil
}

// ImportDepositAddresses deserializes an AddressBatch proto and imports addresses into the pool.
func (s *DepositService) ImportDepositAddresses(ctx context.Context, batchData []byte) (int, int, error) {
	var batch antv1.AddressBatch
	if err := proto.Unmarshal(batchData, &batch); err != nil {
		return 0, 0, fmt.Errorf("deposit service: unmarshal batch: %w", err)
	}
	if len(batch.Entries) == 0 {
		return 0, 0, errors.New("deposit service: empty batch")
	}
	if len(batch.Entries) > 5000 {
		return 0, 0, fmt.Errorf("deposit service: batch too large: %d entries (max 5000), split into multiple files", len(batch.Entries))
	}
	addrs := make([]model.DepositAddress, 0, len(batch.Entries))
	for _, e := range batch.Entries {
		network := e.Network
		if network == "" {
			network = "TRC20"
		}
		addrs = append(addrs, model.DepositAddress{
			Address:          e.Address,
			DerivationIndex:  int(e.DerivationIndex),
			EncryptedPrivkey: e.EncryptedPrivkey,
			Network:          network,
			Status:           "AVAILABLE",
		})
	}
	return s.addrRepo.ImportBatchWithStats(ctx, addrs)
}

// ConfirmDeposit creates a deposit record and credits the user's wallet atomically.
// Called by the chain monitor after on-chain verification.
func (s *DepositService) ConfirmDeposit(ctx context.Context, userID uuid.UUID, addrID uuid.UUID, txHash string, amount string, blockNumber int64, confirmations int) error {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("deposit service: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	depositID := uuid.New()
	tag, err := tx.Exec(ctx, `
		INSERT INTO deposits (id, user_id, deposit_address_id, tx_hash, amount, block_number, confirmations, status, confirmed_at)
		VALUES ($1, $2, $3, $4, $5::numeric, $6, $7, 'CONFIRMED', NOW())
		ON CONFLICT (tx_hash) DO NOTHING
	`, depositID, userID, addrID, txHash, amount, blockNumber, confirmations)
	if err != nil {
		return fmt.Errorf("deposit service: insert deposit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Deposit already exists (concurrent call), skip wallet credit.
		return nil
	}

	walletRepo := s.walletSvc.Repo()
	wallet, err := walletRepo.GetByUserIDTx(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("deposit service: get wallet: %w", err)
	}
	if wallet == nil {
		return ErrWalletNotFound
	}

	_, err = walletRepo.AdjustBalanceTx(ctx, tx, wallet.ID, userID,
		amount, "deposit",
		fmt.Sprintf("USDT deposit confirmed (tx=%s)", txHash),
		nil)
	if err != nil {
		return fmt.Errorf("deposit service: credit wallet: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("deposit service: commit: %w", err)
	}

	s.log.Info("deposit confirmed",
		zap.String("user_id", userID.String()),
		zap.String("tx_hash", txHash),
		zap.String("amount", amount),
		zap.Int64("block", blockNumber))

	return nil
}
