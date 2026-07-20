package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/model"
)

// BeginWithdrawal starts the withdrawal flow by creating a challenge.
// challenge = sha256(amount|dest|nonce|user_id) — this is what the user's passkey signs.
// The withdrawal request is stored as PENDING (funds not yet frozen — freeze happens in FinishWithdrawal).
func (s *WebAuthnService) BeginWithdrawal(ctx context.Context, userID uuid.UUID, amount, destAddress string) ([]byte, uint64, string, error) {
	amt, err := decimal.NewFromString(amount)
	if err != nil {
		return nil, 0, "", fmt.Errorf("webauthn service: begin withdrawal: invalid amount: %w", err)
	}
	if amt.LessThanOrEqual(decimal.Zero) {
		return nil, 0, "", fmt.Errorf("webauthn service: begin withdrawal: amount must be positive")
	}

	whitelist, err := s.withdrawRepo.GetActiveWhitelistAddresses(ctx, userID)
	if err != nil {
		return nil, 0, "", fmt.Errorf("webauthn service: begin withdrawal: whitelist: %w", err)
	}
	if len(whitelist) == 0 {
		return nil, 0, "", fmt.Errorf("webauthn service: begin withdrawal: no active whitelist addresses")
	}
	found := false
	for _, addr := range whitelist {
		if addr == destAddress {
			found = true
			break
		}
	}
	if !found {
		return nil, 0, "", fmt.Errorf("webauthn service: begin withdrawal: destination not in whitelist")
	}

	wallet, err := s.walletSvc.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return nil, 0, "", fmt.Errorf("webauthn service: begin withdrawal: get wallet: %w", err)
	}
	balance, err := decimal.NewFromString(wallet.Balance)
	if err != nil {
		return nil, 0, "", fmt.Errorf("webauthn service: begin withdrawal: parse balance: %w", err)
	}
	if balance.LessThan(amt) {
		return nil, 0, "", ErrInsufficientWithdrawalBalance
	}

	nonce := time.Now().UnixNano()
	withdrawalID := uuid.New()
	idemKey := "withdrawal-" + withdrawalID.String()

	challenge := buildWithdrawalChallenge(amount, destAddress, nonce, userID.String())

	err = s.withdrawRepo.CreateWithdrawal(ctx, &model.WithdrawalRequest{
		ID:          withdrawalID,
		UserID:      userID,
		Amount:      amount,
		DestAddress: destAddress,
		Nonce:       nonce,
		IdemKey:     idemKey,
	})
	if err != nil {
		return nil, 0, "", fmt.Errorf("webauthn service: begin withdrawal: create request: %w", err)
	}

	return challenge, uint64(nonce), withdrawalID.String(), nil
}

// FinishWithdrawal completes the withdrawal flow:
// 1. Store the WebAuthn assertion (transitions to SIGNED_WAITING_BUNDLE).
// 2. Freeze the withdrawal amount (R9: balance -= X, frozen += X).
// If freeze fails, rollback to FAILED — no withdrawal without frozen funds.
//
// The online server does NOT verify the WebAuthn assertion — that's coldsign's job (R11).
func (s *WebAuthnService) FinishWithdrawal(ctx context.Context, userID uuid.UUID, withdrawalID string, assertion []byte, credentialID string) (*model.WithdrawalRequest, error) {
	wid, err := uuid.Parse(withdrawalID)
	if err != nil {
		return nil, fmt.Errorf("webauthn service: finish withdrawal: parse ID: %w", err)
	}

	wr, err := s.withdrawRepo.GetWithdrawal(ctx, wid)
	if err != nil {
		return nil, fmt.Errorf("webauthn service: finish withdrawal: get request: %w", err)
	}
	if wr == nil {
		return nil, ErrWithdrawalNotFound
	}
	if wr.UserID != userID {
		return nil, ErrWithdrawalNotOwner
	}
	if wr.Status != "PENDING" {
		return nil, fmt.Errorf("webauthn service: finish withdrawal: status is %s, expected PENDING", wr.Status)
	}

	// Store assertion first (transitions to SIGNED_WAITING_BUNDLE).
	// If this fails, status stays PENDING — user can retry FinishWithdrawal.
	if err := s.withdrawRepo.StoreWithdrawalAssertion(ctx, wid, assertion, credentialID); err != nil {
		return nil, fmt.Errorf("webauthn service: finish withdrawal: store assertion: %w", err)
	}

	// Freeze funds after assertion is stored.
	// If freeze fails for ANY reason, rollback to FAILED — no withdrawal without frozen funds.
	_, err = s.walletSvc.FreezeForWithdrawal(ctx, userID, wr.Amount, wr.IdemKey)
	if err != nil {
		// Rollback: mark withdrawal as FAILED since freeze failed.
		if rbErr := s.withdrawRepo.UpdateWithdrawalStatus(ctx, wid, "FAILED", nil); rbErr != nil {
			s.log.Error("finish withdrawal: freeze failed AND rollback failed",
				zap.String("withdrawal_id", withdrawalID),
				zap.Error(rbErr))
		}
		if errors.Is(err, model.ErrInsufficientBalance) {
			return nil, ErrInsufficientWithdrawalBalance
		}
		return nil, fmt.Errorf("webauthn service: finish withdrawal: freeze: %w", err)
	}

	wr.Assertion = assertion
	wr.CredentialID = credentialID

	s.log.Info("withdrawal initiated, awaiting cold sign",
		zap.String("withdrawal_id", withdrawalID),
		zap.String("user_id", userID.String()),
		zap.String("amount", wr.Amount),
		zap.String("dest", wr.DestAddress),
		zap.String("credential_id", credentialID))

	return wr, nil
}

// CancelWithdrawal cancels a pending withdrawal and unfreezes funds.
func (s *WebAuthnService) CancelWithdrawal(ctx context.Context, userID uuid.UUID, withdrawalID string) error {
	wid, err := uuid.Parse(withdrawalID)
	if err != nil {
		return fmt.Errorf("webauthn service: cancel withdrawal: parse ID: %w", err)
	}

	wr, err := s.withdrawRepo.GetWithdrawal(ctx, wid)
	if err != nil {
		return fmt.Errorf("webauthn service: cancel withdrawal: get: %w", err)
	}
	if wr == nil {
		return ErrWithdrawalNotFound
	}
	if wr.UserID != userID {
		return ErrWithdrawalNotOwner
	}
	if wr.Status != "PENDING" && wr.Status != "SIGNED" && wr.Status != "SIGNED_WAITING_BUNDLE" {
		return fmt.Errorf("webauthn service: cancel withdrawal: cannot cancel status %s", wr.Status)
	}

	// Cancel the sweep bundle if one was already created (prevents coldsign from signing).
	if wr.BundleID != nil {
		if err := s.withdrawRepo.CancelWithdrawalBundle(ctx, *wr.BundleID); err != nil {
			s.log.Warn("cancel withdrawal: cancel bundle failed",
				zap.String("withdrawal_id", withdrawalID),
				zap.String("bundle_id", wr.BundleID.String()),
				zap.Error(err))
		}
	}

	_, err = s.walletSvc.CancelWithdrawal(ctx, userID, wr.Amount, wr.IdemKey)
	if err != nil {
		return fmt.Errorf("webauthn service: cancel withdrawal: unfreeze: %w", err)
	}

	if err := s.withdrawRepo.UpdateWithdrawalStatus(ctx, wid, "CANCELLED", nil); err != nil {
		return fmt.Errorf("webauthn service: cancel withdrawal: update status: %w", err)
	}

	s.log.Info("withdrawal cancelled",
		zap.String("withdrawal_id", withdrawalID),
		zap.String("user_id", userID.String()))

	return nil
}

// CompleteWithdrawal marks a withdrawal as DONE after successful broadcast.
func (s *WebAuthnService) CompleteWithdrawal(ctx context.Context, withdrawalID uuid.UUID, txHash string) error {
	wr, err := s.withdrawRepo.GetWithdrawal(ctx, withdrawalID)
	if err != nil {
		return fmt.Errorf("webauthn service: complete withdrawal: get: %w", err)
	}
	if wr == nil {
		return ErrWithdrawalNotFound
	}

	_, err = s.walletSvc.CompleteWithdrawal(ctx, wr.UserID, wr.Amount, wr.IdemKey)
	if err != nil {
		return fmt.Errorf("webauthn service: complete withdrawal: deduct frozen: %w", err)
	}

	if err := s.withdrawRepo.UpdateWithdrawalStatus(ctx, withdrawalID, "DONE", &txHash); err != nil {
		return fmt.Errorf("webauthn service: complete withdrawal: update status: %w", err)
	}

	s.log.Info("withdrawal completed",
		zap.String("withdrawal_id", withdrawalID.String()),
		zap.String("tx_hash", txHash))

	return nil
}

// FailWithdrawal marks a withdrawal as FAILED and unfreezes funds.
func (s *WebAuthnService) FailWithdrawal(ctx context.Context, withdrawalID uuid.UUID) error {
	wr, err := s.withdrawRepo.GetWithdrawal(ctx, withdrawalID)
	if err != nil {
		return fmt.Errorf("webauthn service: fail withdrawal: get: %w", err)
	}
	if wr == nil {
		return ErrWithdrawalNotFound
	}

	_, err = s.walletSvc.CancelWithdrawal(ctx, wr.UserID, wr.Amount, wr.IdemKey)
	if err != nil {
		return fmt.Errorf("webauthn service: fail withdrawal: unfreeze: %w", err)
	}

	if err := s.withdrawRepo.UpdateWithdrawalStatus(ctx, withdrawalID, "FAILED", nil); err != nil {
		return fmt.Errorf("webauthn service: fail withdrawal: update status: %w", err)
	}

	s.log.Info("withdrawal failed, funds unfrozen",
		zap.String("withdrawal_id", withdrawalID.String()))

	return nil
}

// ListWithdrawals returns paginated withdrawal history for a user.
func (s *WebAuthnService) ListWithdrawals(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]model.WithdrawalRequest, int64, error) {
	return s.withdrawRepo.ListWithdrawalsByUser(ctx, userID, page, pageSize)
}
