// Package audit implements runtime integrity checks for ADR-0026.
// xpub_audit.go implements the 24h address audit cron (ADR-0026 §12.1, R5 runtime).
//
// It periodically re-derives all DB deposit addresses from the configured xpub
// and compares them against the stored addresses. Any mismatch indicates xpub
// substitution or database tampering → immediate alert + block new address assignment.
package audit

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"go.uber.org/zap"

	"alphaforge/internal/hdwallet"
	"alphaforge/internal/repository"
)

// XpubAuditor periodically verifies that all DB deposit addresses match
// the xpub-derived addresses. Runs every 24h (ADR-0026 §12.1).
type XpubAuditor struct {
	addrRepo *repository.DepositAddressRepository
	adminRepo *repository.AdminRepository
	log       *zap.Logger

	xpubKey *hdkeychain.ExtendedKey // parsed once, reused for all derivations
	// compromised is set to true if audit detects a mismatch.
	// When true, address assignment must be blocked.
	compromised atomic.Bool
}

// NewXpubAuditor creates the auditor. xpubStr is the account-level xpub.
func NewXpubAuditor(
	addrRepo *repository.DepositAddressRepository,
	adminRepo *repository.AdminRepository,
	xpubStr string,
	log *zap.Logger,
) (*XpubAuditor, error) {
	ext, err := hdwallet.ParseXpub(xpubStr)
	if err != nil {
		return nil, fmt.Errorf("xpub audit: parse xpub: %w", err)
	}
	return &XpubAuditor{
		addrRepo:  addrRepo,
		adminRepo: adminRepo,
		xpubKey:   ext,
		log:       log,
	}, nil
}

// Run starts the audit loop. Blocks until ctx is cancelled.
// Runs immediately at startup, then every 24h.
func (a *XpubAuditor) Run(ctx context.Context) error {
	a.log.Info("xpub auditor: started")

	if err := a.auditOnce(ctx); err != nil {
		a.log.Error("xpub auditor: initial audit failed", zap.Error(err))
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.log.Info("xpub auditor: stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := a.auditOnce(ctx); err != nil {
				a.log.Error("xpub auditor: audit failed", zap.Error(err))
			}
		}
	}
}

// auditOnce performs a single audit pass: re-derives all DB addresses from xpub
// and compares. Sets compromised flag on mismatch.
func (a *XpubAuditor) auditOnce(ctx context.Context) error {
	addrs, err := a.addrRepo.ListAllAddressesWithIndex(ctx)
	if err != nil {
		return fmt.Errorf("xpub audit: list addresses: %w", err)
	}

	var mismatches int
	for _, dbAddr := range addrs {
		derived, err := hdwallet.DeriveAddressFromExtKey(a.xpubKey, dbAddr.DerivationIndex)
		if err != nil {
			return fmt.Errorf("xpub audit: derive index %d: %w", dbAddr.DerivationIndex, err)
		}
		if derived != dbAddr.Address {
			mismatches++
			a.log.Error("xpub audit: ADDRESS MISMATCH — possible xpub substitution or DB tampering",
				zap.Uint32("derivation_index", dbAddr.DerivationIndex),
				zap.String("db_address", dbAddr.Address),
				zap.String("derived_address", derived))
		}
	}

	if mismatches > 0 {
		a.compromised.Store(true)
		return fmt.Errorf("xpub audit: %d/%d addresses mismatch — xpub may have been substituted", mismatches, len(addrs))
	}

	a.compromised.Store(false)
	a.log.Info("xpub audit: passed", zap.Int("addresses_verified", len(addrs)))
	return nil
}

// IsCompromised returns true if the last audit detected a mismatch.
// When true, new address assignment must be blocked (ADR-0026 §12.1).
func (a *XpubAuditor) IsCompromised() bool {
	return a.compromised.Load()
}
