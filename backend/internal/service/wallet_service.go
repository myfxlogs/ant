package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"anttrader/internal/model"
	"anttrader/internal/repository"
)

// WalletService is the service layer for user wallet operations.
// Sits between the ConnectRPC handler and the repository.
type WalletService struct {
	repo *repository.WalletRepository
	pg   *pgxpool.Pool
	log  *zap.Logger
}

func NewWalletService(repo *repository.WalletRepository, pg *pgxpool.Pool, log *zap.Logger) *WalletService {
	return &WalletService{repo: repo, pg: pg, log: log}
}

// GetOrCreateWallet returns the wallet for a user, auto-creating one if missing (legacy users).
func (s *WalletService) GetOrCreateWallet(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
	w, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if w != nil {
		return w, nil
	}
	return s.repo.CreateWallet(ctx, userID)
}

// CreateWallet creates a new wallet for a user (called during registration).
func (s *WalletService) CreateWallet(ctx context.Context, userID uuid.UUID) (*model.Wallet, error) {
	return s.repo.CreateWallet(ctx, userID)
}

// AdjustBalance credits or debits a user's wallet and records the transaction.
// amount is a numeric string; positive = credit, negative = debit.
func (s *WalletService) AdjustBalance(ctx context.Context, userID uuid.UUID, amount, txType, description string, operatorID *uuid.UUID) (*model.Wallet, error) {
	w, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrWalletNotFound
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	updated, err := s.repo.AdjustBalanceTx(ctx, tx, w.ID, userID, amount, txType, description, operatorID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.log.Info("WalletService: balance adjusted",
		zap.String("userID", userID.String()),
		zap.String("amount", amount),
		zap.String("txType", txType))
	return updated, nil
}

// ListTransactions returns a paginated list of wallet transactions.
func (s *WalletService) ListTransactions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]model.WalletTransaction, int64, error) {
	return s.repo.ListTransactions(ctx, userID, page, pageSize)
}

// ErrWalletNotFound is returned when a wallet doesn't exist for a user.
var ErrWalletNotFound = &walletNotFoundError{}

type walletNotFoundError struct{}

func (e *walletNotFoundError) Error() string { return "wallet not found" }

// InsufficientBalanceError is returned when a wallet has insufficient balance.
type InsufficientBalanceError struct {
	Balance string
	Cost    string
}

func (e *InsufficientBalanceError) Error() string {
	return fmt.Sprintf("insufficient balance: have %s, need %s", e.Balance, e.Cost)
}
