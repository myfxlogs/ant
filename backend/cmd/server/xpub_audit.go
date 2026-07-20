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
	auditor, err := audit.NewXpubAuditor(addrRepo, adminRepo, cfg.DepositXpub, log)
	if err != nil {
		log.Fatal("failed to create xpub auditor", zap.Error(err))
	}
	depositSvc.SetCompromisedChecker(auditor)
	go auditor.Run(ctx)
}
