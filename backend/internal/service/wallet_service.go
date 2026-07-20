package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/repository"
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

// Repo returns the underlying WalletRepository for transactional access by other services.
func (s *WalletService) Repo() *repository.WalletRepository { return s.repo }

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
// idemKey is a unique key for idempotency (R7); empty = auto-generate.
func (s *WalletService) AdjustBalance(ctx context.Context, userID uuid.UUID, amount, txType, description string, operatorID *uuid.UUID, idemKey string) (*model.Wallet, error) {
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

	updated, err := s.repo.AdjustBalanceTx(ctx, tx, w.ID, userID, amount, txType, description, operatorID, idemKey)
	if err != nil {
		if errors.Is(err, model.ErrIdempotentReplay) {
			return w, nil
		}
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

// FreezeForWithdrawal moves amount from balance to frozen_balance (R9).
// idemKey must be unique per withdrawal (e.g. "withdrawal-{withdrawalID}").
func (s *WalletService) FreezeForWithdrawal(ctx context.Context, userID uuid.UUID, amount, idemKey string) (*model.Wallet, error) {
	return s.runFreezeTx(ctx, userID, amount, idemKey, true, "withdrawal_freeze", "Freeze for withdrawal")
}

// CompleteWithdrawal deducts frozen amount after successful broadcast (R9).
func (s *WalletService) CompleteWithdrawal(ctx context.Context, userID uuid.UUID, amount, idemKey string) (*model.Wallet, error) {
	return s.runFreezeTx(ctx, userID, amount, idemKey, false, "withdrawal_complete", "Withdrawal completed")
}

// CancelWithdrawal returns frozen amount to balance (R9).
func (s *WalletService) CancelWithdrawal(ctx context.Context, userID uuid.UUID, amount, idemKey string) (*model.Wallet, error) {
	return s.runFreezeTx(ctx, userID, amount, idemKey, false, "withdrawal_cancel", "Withdrawal cancelled")
}

func (s *WalletService) runFreezeTx(ctx context.Context, userID uuid.UUID, amount, idemKey string, freeze bool, txType, desc string) (*model.Wallet, error) {
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

	var updated *model.Wallet
	if freeze {
		updated, err = s.repo.FreezeForWithdrawal(ctx, tx, w.ID, userID, amount, idemKey)
	} else if txType == "withdrawal_complete" {
		updated, err = s.repo.CompleteWithdrawal(ctx, tx, w.ID, userID, amount, idemKey)
	} else {
		updated, err = s.repo.CancelWithdrawal(ctx, tx, w.ID, userID, amount, idemKey)
	}
	if err != nil {
		if errors.Is(err, model.ErrIdempotentReplay) {
			return w, nil
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return updated, nil
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
