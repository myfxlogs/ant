package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/hdwallet"
	"alphaforge/internal/model"
	"alphaforge/internal/repository"
)

// CompromisedChecker returns true if xpub integrity is compromised (ADR-0026 §12.1).
// Implemented by audit.XpubAuditor.
type CompromisedChecker interface {
	IsCompromised() bool
}

// DepositService manages USDT deposits via on-demand HD wallet address derivation.
// The online server holds ONLY the account-level xpub — no private keys (ADR-0026 R1).
type DepositService struct {
	addrRepo   *repository.DepositAddressRepository
	depRepo    *repository.DepositRepository
	walletRepo *repository.WalletRepository
	pg         *pgxpool.Pool
	log        *zap.Logger
	xpub       string
	xpubKey    *hdkeychain.ExtendedKey // pre-parsed for hot-path derivation

	// compromisedChecker blocks new address derivation if xpub audit fails (ADR-0026 §12.1).
	compromisedChecker CompromisedChecker

	// OnAddressClaimed is called when a new address is assigned to a user.
	// Used by chain monitor to register the address for on-chain scanning.
	OnAddressClaimed func(addr string, userID uuid.UUID, addrID uuid.UUID)
}

// NewDepositService creates a deposit service with watch-only xpub derivation.
// xpub is the account-level extended public key (m/44'/195'/0'/0).
func NewDepositService(
	addrRepo *repository.DepositAddressRepository,
	depRepo *repository.DepositRepository,
	walletRepo *repository.WalletRepository,
	pg *pgxpool.Pool,
	xpub string,
	log *zap.Logger,
) *DepositService {
	var parsedKey *hdkeychain.ExtendedKey
	if xpub != "" {
		if ext, err := hdwallet.ParseXpub(xpub); err == nil {
			parsedKey = ext
		} else {
			log.Warn("failed to parse deposit xpub at init", zap.Error(err))
		}
	}
	return &DepositService{
		addrRepo:   addrRepo,
		depRepo:    depRepo,
		walletRepo: walletRepo,
		pg:         pg,
		log:        log,
		xpub:       xpub,
		xpubKey:    parsedKey,
	}
}

// Xpub returns the account-level xpub string. Used by ImportDepositAddresses
// for one-time cross-check of hdgen output against online derivation (ADR-0026 §7 step 5).
func (s *DepositService) Xpub() string {
	return s.xpub
}

// XpubKey returns the pre-parsed account-level extended public key.
// Returns nil if xpub was not set or failed to parse at init.
func (s *DepositService) XpubKey() *hdkeychain.ExtendedKey {
	return s.xpubKey
}

// SetCompromisedChecker wires the xpub auditor to block address derivation on mismatch.
func (s *DepositService) SetCompromisedChecker(c CompromisedChecker) {
	s.compromisedChecker = c
}

// GetOrDeriveAddress returns the user's existing deposit address, or derives
// a new one on-demand from the account xpub using a PG SEQUENCE for index
// allocation (ADR-0026 Q1). Idempotent — concurrent calls for the same user
// return the same address.
func (s *DepositService) GetOrDeriveAddress(ctx context.Context, userID uuid.UUID) (addr, network string, err error) {
	existing, err := s.addrRepo.GetByUserID(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("deposit service: get existing: %w", err)
	}
	if existing != nil {
		return existing.Address, existing.Network, nil
	}

	// ADR-0026 §12.1: block new address derivation if xpub audit detected mismatch.
	if s.compromisedChecker != nil && s.compromisedChecker.IsCompromised() {
		return "", "", fmt.Errorf("deposit service: xpub integrity compromised — address derivation blocked")
	}

	idx, err := s.addrRepo.NextDerivationIndex(ctx)
	if err != nil {
		return "", "", fmt.Errorf("deposit service: next index: %w", err)
	}

	if s.xpubKey != nil {
		addr, err = hdwallet.DeriveAddressFromExtKey(s.xpubKey, uint32(idx))
	} else {
		addr, err = hdwallet.DeriveAddressFromXpub(s.xpub, uint32(idx))
	}
	if err != nil {
		return "", "", fmt.Errorf("deposit service: derive address at index %d: %w", idx, err)
	}

	record, err := s.addrRepo.InsertDepositAddress(ctx, userID, addr, "TRC20", idx)
	if err != nil {
		return "", "", fmt.Errorf("deposit service: insert address: %w", err)
	}

	if s.OnAddressClaimed != nil {
		s.OnAddressClaimed(record.Address, userID, record.ID)
	}

	s.log.Info("derived new deposit address",
		zap.String("user_id", userID.String()),
		zap.Int("derivation_index", idx),
		zap.String("address", addr),
	)

	return record.Address, record.Network, nil
}

// ListMyDeposits returns confirmed deposits for a user.
func (s *DepositService) ListMyDeposits(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]model.Deposit, int64, error) {
	return s.depRepo.ListByUser(ctx, userID, page, pageSize)
}

// ListManualReviewDeposits returns deposits requiring manual review.
func (s *DepositService) ListManualReviewDeposits(ctx context.Context, page, pageSize int) ([]model.Deposit, int64, error) {
	return s.depRepo.ListManualReview(ctx, page, pageSize)
}

// ListDepositAddresses returns all deposit addresses with pagination.
func (s *DepositService) ListDepositAddresses(ctx context.Context, status string, page, pageSize int) ([]model.DepositAddress, int64, int, error) {
	addrs, total, err := s.addrRepo.ListAllAddresses(ctx, status, page, pageSize)
	if err != nil {
		return nil, 0, 0, err
	}
	return addrs, total, 0, nil
}

// ConfirmDeposit records a confirmed on-chain deposit and credits the user's wallet
// atomically in a single DB transaction. The tx_hash serves as idem_key for the
// wallet credit (R7 idempotency): if the chain monitor re-processes the same event,
// the deposit INSERT is skipped (ON CONFLICT DO NOTHING) and the wallet credit
// returns ErrIdempotentReplay — both safe no-ops.
func (s *DepositService) ConfirmDeposit(ctx context.Context, userID, addrID uuid.UUID, txHash, amount string, blockNumber int64, confirmations int) error {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("deposit service: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now()
	dep := &model.Deposit{
		ID:               uuid.New(),
		UserID:           userID,
		DepositAddressID: addrID,
		TxHash:           txHash,
		Amount:           amount,
		BlockNumber:      blockNumber,
		Confirmations:    confirmations,
		Status:           "CONFIRMED",
		ConfirmedAt:      &now,
	}
	if err := s.depRepo.Create(ctx, dep); err != nil {
		return fmt.Errorf("deposit service: create deposit: %w", err)
	}

	wallet, err := s.walletRepo.GetByUserIDTx(ctx, tx, userID)
	if err != nil {
		return fmt.Errorf("deposit service: get wallet: %w", err)
	}
	if wallet == nil {
		wallet, err = s.walletRepo.CreateWalletTx(ctx, tx, userID)
		if err != nil {
			return fmt.Errorf("deposit service: create wallet: %w", err)
		}
	}

	_, err = s.walletRepo.AdjustBalanceTx(ctx, tx, wallet.ID, userID, amount, "deposit",
		fmt.Sprintf("On-chain USDT deposit: %s", txHash), nil, "deposit-"+txHash)
	if err != nil {
		if errors.Is(err, model.ErrIdempotentReplay) {
			_ = tx.Rollback(ctx)
			return nil
		}
		return fmt.Errorf("deposit service: credit wallet: %w", err)
	}

	return tx.Commit(ctx)
}

// MarkAddressReceived updates the has_received_usdt flag for an address.
func (s *DepositService) MarkAddressReceived(ctx context.Context, addrID uuid.UUID) error {
	return s.addrRepo.MarkReceivedUSDT(ctx, addrID)
}
