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

// XpubProvider returns the current account-level xpub key.
// Implemented by *service.DepositService — allows auditor to see hot-reloaded xpub.
type XpubProvider interface {
	XpubKey() *hdkeychain.ExtendedKey
}

// XpubAuditor periodically verifies that all DB deposit addresses match
// the xpub-derived addresses. Runs every 24h (ADR-0026 §12.1).
type XpubAuditor struct {
	addrRepo    *repository.DepositAddressRepository
	adminRepo   *repository.AdminRepository
	xpubProvider XpubProvider
	log         *zap.Logger

	// compromised is set to true if audit detects a mismatch.
	// When true, address assignment must be blocked.
	compromised atomic.Bool
}

// NewXpubAuditor creates the auditor. xpubProvider must return the current xpub key
// (dynamically — reflects hot-reloads via DepositService.UpdateXpub).
func NewXpubAuditor(
	addrRepo *repository.DepositAddressRepository,
	adminRepo *repository.AdminRepository,
	xpubProvider XpubProvider,
	log *zap.Logger,
) (*XpubAuditor, error) {
	if xpubProvider == nil {
		return nil, fmt.Errorf("xpub audit: xpubProvider is nil")
	}
	if xpubProvider.XpubKey() == nil {
		return nil, fmt.Errorf("xpub audit: xpub not configured")
	}
	return &XpubAuditor{
		addrRepo:     addrRepo,
		adminRepo:    adminRepo,
		xpubProvider: xpubProvider,
		log:          log,
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
	xpubKey := a.xpubProvider.XpubKey()
	if xpubKey == nil {
		return fmt.Errorf("xpub audit: xpub not configured (provider returned nil)")
	}

	addrs, err := a.addrRepo.ListAllAddressesWithIndex(ctx)
	if err != nil {
		return fmt.Errorf("xpub audit: list addresses: %w", err)
	}

	var mismatches int
	for _, dbAddr := range addrs {
		derived, err := hdwallet.DeriveAddressFromExtKey(xpubKey, dbAddr.DerivationIndex)
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
