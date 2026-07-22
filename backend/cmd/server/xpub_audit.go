package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/audit"
	"alphaforge/internal/config"
	"alphaforge/internal/notifier"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
)

// resolveDepositXpub reads deposit_xpub from system_config DB table.
// Falls back to env var (DEPOSIT_XPUB) if DB value is empty.
// DB is the canonical source for xpub — survives server migration (PG volume),
// editable via admin SystemConfig UI.
//
// Security: deposit_xpub_fingerprint is NOT read from DB. It stays as env var
// only (DEPOSIT_XPUB_FINGERPRINT), serving as an integrity anchor. Even if an
// attacker with root access modifies the xpub in DB, they cannot forge the
// fingerprint without also modifying the host's .env/docker-compose.yml —
// providing defense in depth against xpub substitution attacks (ADR-0026 R5).
func resolveDepositXpub(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, log *zap.Logger) {
	adminRepo := repository.NewAdminRepository(pool)

	if dbCfg, err := adminRepo.GetConfig(ctx, "deposit_xpub"); err == nil && dbCfg != nil && dbCfg.Value != "" {
		cfg.DepositXpub = dbCfg.Value
		log.Info("startup: deposit xpub loaded from system_config")
	} else if cfg.DepositXpub != "" {
		log.Info("startup: deposit xpub loaded from env (DEPOSIT_XPUB)")
	} else {
		log.Warn("startup: deposit xpub not configured — deposit address derivation disabled")
	}

	// Fingerprint: env var only. NOT read from DB (integrity anchor).
	if cfg.DepositXpubFingerprint == "" {
		log.Warn("startup: deposit_xpub_fingerprint not set in env — xpub integrity check will be skipped (set DEPOSIT_XPUB_FINGERPRINT for production)")
	}
}

// startBackgroundServices launches non-RPC background goroutines:
//   - hard delete cleanup (daily)
//   - ledger shipper (LISTEN/NOTIFY outbox forwarding)
//   - xpub runtime integrity audit (ADR-0026 §12.1, R5)
func startBackgroundServices(
	ctx context.Context,
	pool *pgxpool.Pool,
	emailNotifier *notifier.EmailNotifier,
	addrRepo *repository.DepositAddressRepository,
	adminRepo *repository.AdminRepository,
	depositSvc *service.DepositService,
	cfg *config.Config,
	log *zap.Logger,
) {
	go startHardDeleteCleanup(ctx, pool, log)
	ledgerShipper := service.NewLedgerShipper(pool, emailNotifier, log)
	go ledgerShipper.Run(ctx)

	if cfg.DepositXpub == "" {
		return
	}
	auditor, err := audit.NewXpubAuditor(addrRepo, adminRepo, depositSvc, log)
	if err != nil {
		log.Fatal("failed to create xpub auditor", zap.Error(err))
	}
	depositSvc.SetCompromisedChecker(auditor)
	go auditor.Run(ctx)
}
