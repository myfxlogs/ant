package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/config"
	"alphaforge/internal/connect/user"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
	"alphaforge/internal/sweep"

	connectrpc "connectrpc.com/connect"
)

// wireWebAuthn sets up the WebAuthn withdrawal authorization service and handler (ADR-0026 Phase E).
func wireWebAuthn(
	mux *http.ServeMux,
	pool *pgxpool.Pool,
	log *zap.Logger,
	cfg *config.Config,
	walletSvc *service.WalletService,
	walletRepo *repository.WalletRepository,
	emailNotifier service.Notifier,
	platformSvc *service.PlatformService,
	sweepBundleRepo *sweep.BundleRepository,
	sweepTronClient *sweep.TronClient,
	adminRepo *repository.AdminRepository,
	otelInterceptor, authInterceptor connectrpc.Interceptor,
) {
	credRepo := repository.NewWebAuthnRepository(pool)
	withdrawRepo := repository.NewWithdrawalRepository(pool)
	webauthnSvc, err := service.NewWebAuthnService(cfg.WebAuthnRPID, cfg.WebAuthnRPOrigin,
		credRepo, withdrawRepo, walletSvc, walletRepo, emailNotifier, log)
	if err != nil {
		log.Warn("webauthn service: failed to initialize (withdrawals disabled)", zap.Error(err))
		return
	}
	webauthnServer := user.NewWebAuthnServer(webauthnSvc, platformSvc, log)
	mux.Handle(antv1c.NewWebAuthnServiceHandler(webauthnServer, withSency(otelInterceptor, authInterceptor)))

	// Wire WithdrawalBuilder if sweep infrastructure is available.
	if sweepBundleRepo != nil && sweepTronClient != nil {
		wb := service.NewWithdrawalBuilder(withdrawRepo, sweepBundleRepo, adminRepo, sweepTronClient, log)
		go runWithdrawalBuilder(context.Background(), wb, log)
	} else {
		log.Warn("withdrawal builder: sweep infrastructure unavailable (withdrawal bundle building disabled)")
	}

	// Start whitelist cooldown activation sweeper (every 1 min).
	go runWhitelistSweeper(context.Background(), webauthnSvc, log)
}

func runWithdrawalBuilder(ctx context.Context, wb *service.WithdrawalBuilder, log *zap.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := wb.BuildPendingWithdrawals(ctx); err != nil {
				log.Warn("withdrawal builder: periodic build failed", zap.Error(err))
			}
		}
	}
}

func runWhitelistSweeper(ctx context.Context, svc *service.WebAuthnService, log *zap.Logger) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.SweepPendingWhitelist(ctx); err != nil {
				log.Warn("whitelist sweeper: periodic sweep failed", zap.Error(err))
			}
		}
	}
}
