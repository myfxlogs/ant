package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/repository"
)

// DepositService manages USDT deposit requests.
// Users create requests; admins approve (credits wallet) or reject.
type DepositService struct {
	depositRepo *repository.DepositRepository
	walletSvc   *WalletService
	adminRepo   *repository.AdminRepository
	pg          *pgxpool.Pool
	log         *zap.Logger
}

func NewDepositService(
	depositRepo *repository.DepositRepository,
	walletSvc *WalletService,
	adminRepo *repository.AdminRepository,
	pg *pgxpool.Pool,
	log *zap.Logger,
) *DepositService {
	return &DepositService{
		depositRepo: depositRepo,
		walletSvc:   walletSvc,
		adminRepo:   adminRepo,
		pg:          pg,
		log:         log,
	}
}

// CreateDeposit creates a new PENDING deposit request for the user.
func (s *DepositService) CreateDeposit(ctx context.Context, userID uuid.UUID, amount, txHash string) (*model.DepositRequest, error) {
	amt, err := decimal.NewFromString(amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}
	if amt.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("amount must be positive")
	}

	// Get exchange rate from system_config.
	rateStr, err := s.getConfigValue(ctx, "usdt_exchange_rate")
	if err != nil || rateStr == "" {
		rateStr = "1"
	}
	rate, err := decimal.NewFromString(rateStr)
	if err != nil || rate.LessThanOrEqual(decimal.Zero) {
		rate = decimal.NewFromInt(1)
	}
	amountUSD := amt.Mul(rate)

	req := &model.DepositRequest{
		ID:        uuid.New(),
		UserID:    userID,
		Amount:    amount,
		AmountUSD: amountUSD.String(),
		Status:    "PENDING",
	}
	if txHash != "" {
		req.TxHash = &txHash
	}

	if err := s.depositRepo.Create(ctx, req); err != nil {
		return nil, fmt.Errorf("create deposit: %w", err)
	}
	s.log.Info("deposit request created",
		zap.String("user_id", userID.String()),
		zap.String("amount", amount),
		zap.String("amount_usd", amountUSD.String()))
	return req, nil
}

// ListMyDeposits returns paginated deposit history for a user.
func (s *DepositService) ListMyDeposits(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]model.DepositRequest, int64, error) {
	return s.depositRepo.ListByUser(ctx, userID, page, pageSize)
}

// GetDepositInfo returns the current USDT receiving address, network, and exchange rate.
func (s *DepositService) GetDepositInfo(ctx context.Context) (addr, network, rate string, err error) {
	addr, err = s.getConfigValue(ctx, "usdt_receiving_address")
	if err != nil {
		return "", "", "", fmt.Errorf("get receiving address: %w", err)
	}
	network, err = s.getConfigValue(ctx, "usdt_network")
	if err != nil || network == "" {
		network = "TRC20"
	}
	rate, err = s.getConfigValue(ctx, "usdt_exchange_rate")
	if err != nil || rate == "" {
		rate = "1"
	}
	return addr, network, rate, nil
}

// ListDeposits returns paginated deposit requests for admin (optionally filtered by status).
func (s *DepositService) ListDeposits(ctx context.Context, page, pageSize int, status string) ([]model.DepositRequest, int64, error) {
	return s.depositRepo.ListAll(ctx, page, pageSize, status)
}

// ApproveDeposit marks a deposit as APPROVED and credits the user's wallet.
// Uses a transaction to ensure atomicity: deposit status update + wallet credit.
func (s *DepositService) ApproveDeposit(ctx context.Context, depositID, reviewerID uuid.UUID, reviewNote string) (*model.DepositRequest, error) {
	// Fetch the deposit first to get user_id and amount_usd.
	deposit, err := s.depositRepo.GetByID(ctx, depositID)
	if err != nil {
		return nil, err
	}
	if deposit.Status != "PENDING" {
		return nil, repository.ErrDepositAlreadyProcessed
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Credit wallet within the transaction.
	walletRepo := s.walletSvc.Repo()
	wallet, err := walletRepo.GetByUserIDTx(ctx, tx, deposit.UserID)
	if err != nil {
		return nil, fmt.Errorf("get wallet: %w", err)
	}
	if wallet == nil {
		return nil, ErrWalletNotFound
	}

	updated, err := walletRepo.AdjustBalanceTx(ctx, tx, wallet.ID, deposit.UserID,
		deposit.AmountUSD, "deposit",
		fmt.Sprintf("USDT deposit approved (deposit_id=%s)", depositID),
		&reviewerID)
	if err != nil {
		return nil, fmt.Errorf("credit wallet: %w", err)
	}

	// Link the wallet transaction ID.
	var walletTxID *uuid.UUID
	if updated.LastTransactionID != nil {
		walletTxID = updated.LastTransactionID
	}

	// Update deposit status.
	if err := s.depositRepo.UpdateStatusTx(ctx, tx, depositID, "APPROVED", reviewerID, reviewNote, walletTxID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	s.log.Info("deposit approved",
		zap.String("deposit_id", depositID.String()),
		zap.String("user_id", deposit.UserID.String()),
		zap.String("amount_usd", deposit.AmountUSD),
		zap.String("reviewer", reviewerID.String()))

	// Re-fetch to return updated state.
	return s.depositRepo.GetByID(ctx, depositID)
}

// RejectDeposit marks a deposit as REJECTED. No wallet credit occurs.
func (s *DepositService) RejectDeposit(ctx context.Context, depositID, reviewerID uuid.UUID, reviewNote string) (*model.DepositRequest, error) {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.depositRepo.UpdateStatusTx(ctx, tx, depositID, "REJECTED", reviewerID, reviewNote, nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	s.log.Info("deposit rejected",
		zap.String("deposit_id", depositID.String()),
		zap.String("reviewer", reviewerID.String()))

	return s.depositRepo.GetByID(ctx, depositID)
}

func (s *DepositService) getConfigValue(ctx context.Context, key string) (string, error) {
	cfg, err := s.adminRepo.GetConfig(ctx, key)
	if err != nil {
		if errors.Is(err, repository.ErrConfigNotFound) {
			return "", nil
		}
		return "", err
	}
	return cfg.Value, nil
}
