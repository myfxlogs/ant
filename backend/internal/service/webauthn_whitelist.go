package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
)

// AddWhitelistAddress adds a withdrawal destination with 2FA + 24h cooldown (R12).
// The entry is created as PENDING_CONFIRMATION. After 2FA email confirmation and
// cooldown expiry, ActivatePendingWhitelist transitions it to ACTIVE.
func (s *WebAuthnService) AddWhitelistAddress(ctx context.Context, userID uuid.UUID, userEmail, address, label string) error {
	entry := &model.WithdrawalWhitelistEntry{
		ID:      uuid.New(),
		UserID:  userID,
		Address: address,
		Label:   label,
	}
	if err := s.withdrawRepo.CreateWhitelistEntry(ctx, entry); err != nil {
		return fmt.Errorf("webauthn service: add whitelist: %w", err)
	}

	idemKey := "whitelist-add-" + entry.ID.String()
	if err := s.withdrawRepo.CreateCredentialChangeLog(ctx, &model.CredentialChangeLog{
		UserID:     userID,
		ChangeType: "WHITELIST_ADD",
		TargetID:   address,
		IdemKey:    idemKey,
	}); err != nil {
		return fmt.Errorf("webauthn service: add whitelist: change log: %w", err)
	}

	if err := s.walletRepo.WriteCredentialChangeLedger(ctx, userID, "CREDENTIAL_CHANGE",
		"WHITELIST_ADD:"+address, idemKey); err != nil {
		return fmt.Errorf("webauthn service: add whitelist: hash chain: %w", err)
	}

	if s.emailNotifier != nil && userEmail != "" {
		subject := "Withdrawal address added — AlphaForge"
		body := fmt.Sprintf(
			"A new withdrawal address has been added to your account:\n\n"+
				"Address: %s\nLabel: %s\n\n"+
				"This address will be activated after a 24-hour cooldown period.\n"+
				"If you did not add this address, please contact support immediately.",
			address, label,
		)
		if err := s.emailNotifier.SendTo(userEmail, subject, body); err != nil {
			s.log.Warn("whitelist 2FA email send failed",
				zap.String("user_id", userID.String()),
				zap.String("address", address),
				zap.Error(err))
		}
	}

	s.log.Info("whitelist address added (pending 2FA + 24h cooldown)",
		zap.String("user_id", userID.String()),
		zap.String("address", address),
		zap.String("entry_id", entry.ID.String()))

	return nil
}

// ActivatePendingWhitelist transitions a whitelist entry to ACTIVE after cooldown.
// Called by the user after 24h, or by a background sweeper. The repo enforces
// cooldown_until <= NOW() and status = PENDING_CONFIRMATION.
func (s *WebAuthnService) ActivatePendingWhitelist(ctx context.Context, userID, entryID uuid.UUID) error {
	if err := s.withdrawRepo.ActivateWhitelistEntry(ctx, entryID); err != nil {
		return fmt.Errorf("webauthn service: activate whitelist: %w", err)
	}

	idemKey := "whitelist-activate-" + entryID.String()
	if err := s.walletRepo.WriteCredentialChangeLedger(ctx, userID, "CREDENTIAL_CHANGE",
		"WHITELIST_ACTIVATE:"+entryID.String(), idemKey); err != nil {
		return fmt.Errorf("webauthn service: activate whitelist: hash chain: %w", err)
	}

	if err := s.withdrawRepo.ConfirmCredentialChangeLog(ctx, "whitelist-add-"+entryID.String()); err != nil {
		s.log.Warn("whitelist activate: confirm change log failed",
			zap.String("entry_id", entryID.String()),
			zap.Error(err))
	}

	s.log.Info("whitelist address activated",
		zap.String("user_id", userID.String()),
		zap.String("entry_id", entryID.String()))

	return nil
}

// ListWhitelistAddresses returns all whitelist entries for a user.
func (s *WebAuthnService) ListWhitelistAddresses(ctx context.Context, userID uuid.UUID) ([]model.WithdrawalWhitelistEntry, error) {
	return s.withdrawRepo.ListWhitelistByUser(ctx, userID)
}

// RemoveWhitelistAddress removes a whitelist entry (R12).
func (s *WebAuthnService) RemoveWhitelistAddress(ctx context.Context, userID, entryID uuid.UUID) error {
	if err := s.withdrawRepo.RemoveWhitelistEntry(ctx, entryID, userID); err != nil {
		return fmt.Errorf("webauthn service: remove whitelist: %w", err)
	}

	idemKey := "whitelist-remove-" + entryID.String()
	if err := s.withdrawRepo.CreateCredentialChangeLog(ctx, &model.CredentialChangeLog{
		UserID:     userID,
		ChangeType: "WHITELIST_REMOVE",
		TargetID:   entryID.String(),
		IdemKey:    idemKey,
	}); err != nil {
		return fmt.Errorf("webauthn service: remove whitelist: change log: %w", err)
	}

	if err := s.walletRepo.WriteCredentialChangeLedger(ctx, userID, "CREDENTIAL_CHANGE",
		"WHITELIST_REMOVE:"+entryID.String(), idemKey); err != nil {
		return fmt.Errorf("webauthn service: remove whitelist: hash chain: %w", err)
	}

	s.log.Info("whitelist address removed",
		zap.String("user_id", userID.String()),
		zap.String("entry_id", entryID.String()))

	return nil
}

// SweepPendingWhitelist activates all whitelist entries whose cooldown has expired.
// Intended to be called periodically (e.g. every minute) by a background goroutine.
func (s *WebAuthnService) SweepPendingWhitelist(ctx context.Context) error {
	entries, err := s.withdrawRepo.ListPendingActivations(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("webauthn service: sweep pending whitelist: %w", err)
	}
	for _, e := range entries {
		if err := s.ActivatePendingWhitelist(ctx, e.UserID, e.ID); err != nil {
			s.log.Warn("sweep: activate whitelist failed",
				zap.String("entry_id", e.ID.String()),
				zap.Error(err))
		}
	}
	return nil
}

// ExportWhitelist returns a serialized ColdsignWhitelist proto for coldsign USB sync (R12).
func (s *WebAuthnService) ExportWhitelist(ctx context.Context) ([]byte, error) {
	entries, err := s.withdrawRepo.ListAllActiveWhitelist(ctx)
	if err != nil {
		return nil, fmt.Errorf("webauthn service: export whitelist: %w", err)
	}

	wl := &antv1.ColdsignWhitelist{Users: make(map[string]*antv1.ColdsignWhitelistEntry)}
	for _, e := range entries {
		uid := e.UserID.String()
		if wl.Users[uid] == nil {
			wl.Users[uid] = &antv1.ColdsignWhitelistEntry{}
		}
		wl.Users[uid].Addresses = append(wl.Users[uid].Addresses, e.Address)
	}

	data, err := proto.Marshal(wl)
	if err != nil {
		return nil, fmt.Errorf("webauthn service: export whitelist: marshal: %w", err)
	}
	return data, nil
}
